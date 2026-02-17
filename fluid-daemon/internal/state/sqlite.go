// Package state provides local SQLite state storage for the sandbox host.
package state

import (
	"context"
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Sandbox represents a sandbox in local state.
type Sandbox struct {
	ID         string `gorm:"primaryKey"`
	Name       string `gorm:"index"`
	AgentID    string `gorm:"index"`
	BaseImage  string
	Bridge     string
	TAPDevice  string
	MACAddress string
	IPAddress  string
	State      string `gorm:"index"`
	PID        int
	VCPUs      int
	MemoryMB   int
	TTLSeconds int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time `gorm:"index"`
}

// CachedImage tracks a pulled snapshot image in the local cache.
type CachedImage struct {
	ID         string `gorm:"primaryKey"`
	ImageName  string `gorm:"uniqueIndex"`
	SourceHost string `gorm:"index"`
	VMName     string `gorm:"index"`
	SizeMB     int64
	PulledAt   time.Time
}

// Command represents a command execution record.
type Command struct {
	ID         string `gorm:"primaryKey"`
	SandboxID  string `gorm:"index"`
	Command    string
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMS int64
	StartedAt  time.Time
	EndedAt    time.Time
}

// Store provides local state persistence via SQLite.
type Store struct {
	db *gorm.DB
}

// NewStore creates a new SQLite state store.
func NewStore(dbPath string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		NowFunc: func() time.Time { return time.Now().UTC() },
		Logger:  logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Auto-migrate tables
	if err := db.AutoMigrate(&Sandbox{}, &Command{}, &CachedImage{}); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}

	return &Store{db: db}, nil
}

// DB returns the underlying GORM database handle.
func (s *Store) DB() *gorm.DB { return s.db }

// Close closes the database connection.
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CreateSandbox creates a new sandbox record.
func (s *Store) CreateSandbox(ctx context.Context, sb *Sandbox) error {
	return s.db.WithContext(ctx).Create(sb).Error
}

// GetSandbox retrieves a sandbox by ID.
func (s *Store) GetSandbox(ctx context.Context, id string) (*Sandbox, error) {
	var sb Sandbox
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&sb).Error; err != nil {
		return nil, err
	}
	return &sb, nil
}

// ListSandboxes returns all non-deleted sandboxes.
func (s *Store) ListSandboxes(ctx context.Context) ([]*Sandbox, error) {
	var sandboxes []*Sandbox
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Find(&sandboxes).Error; err != nil {
		return nil, err
	}
	return sandboxes, nil
}

// UpdateSandbox updates a sandbox record.
func (s *Store) UpdateSandbox(ctx context.Context, sb *Sandbox) error {
	return s.db.WithContext(ctx).Save(sb).Error
}

// DeleteSandbox soft-deletes a sandbox.
func (s *Store) DeleteSandbox(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&Sandbox{}).Where("id = ?", id).
		Updates(map[string]any{
			"deleted_at": &now,
			"state":      "DESTROYED",
		}).Error
}

// ListExpiredSandboxes returns sandboxes past their TTL.
func (s *Store) ListExpiredSandboxes(ctx context.Context, defaultTTL time.Duration) ([]*Sandbox, error) {
	var sandboxes []*Sandbox
	now := time.Now().UTC()

	// Find sandboxes where TTL has expired
	err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND state NOT IN (?, ?)", "DESTROYED", "ERROR").
		Find(&sandboxes).Error
	if err != nil {
		return nil, err
	}

	var expired []*Sandbox
	for _, sb := range sandboxes {
		ttl := defaultTTL
		if sb.TTLSeconds > 0 {
			ttl = time.Duration(sb.TTLSeconds) * time.Second
		}
		if ttl > 0 && now.After(sb.CreatedAt.Add(ttl)) {
			expired = append(expired, sb)
		}
	}

	return expired, nil
}

// CreateCommand creates a command execution record.
func (s *Store) CreateCommand(ctx context.Context, cmd *Command) error {
	return s.db.WithContext(ctx).Create(cmd).Error
}

// ListSandboxCommands returns commands for a sandbox.
func (s *Store) ListSandboxCommands(ctx context.Context, sandboxID string) ([]*Command, error) {
	var commands []*Command
	if err := s.db.WithContext(ctx).Where("sandbox_id = ?", sandboxID).Order("started_at DESC").Find(&commands).Error; err != nil {
		return nil, err
	}
	return commands, nil
}
