package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aspectrr/fluid.sh/api/internal/crypto"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

var (
	_ store.Store     = (*postgresStore)(nil)
	_ store.DataStore = (*postgresStore)(nil)
)

type postgresStore struct {
	db            *gorm.DB
	conf          store.Config
	encryptionKey []byte
}

// GORM models

type UserModel struct {
	ID            string     `gorm:"column:id;primaryKey"`
	Email         string     `gorm:"column:email;uniqueIndex:idx_users_email_not_deleted,where:deleted_at IS NULL"`
	DisplayName   string     `gorm:"column:display_name"`
	AvatarURL     string     `gorm:"column:avatar_url"`
	PasswordHash  string     `gorm:"column:password_hash"`
	EmailVerified bool       `gorm:"column:email_verified;default:false"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at;index"`
}

func (UserModel) TableName() string { return "users" }

type OAuthAccountModel struct {
	ID           string    `gorm:"column:id;primaryKey"`
	UserID       string    `gorm:"column:user_id;index"`
	Provider     string    `gorm:"column:provider"`
	ProviderID   string    `gorm:"column:provider_id;uniqueIndex:idx_oauth_provider_id,composite:provider"`
	Email        string    `gorm:"column:email"`
	AccessToken  string    `gorm:"column:access_token"`
	RefreshToken string    `gorm:"column:refresh_token"`
	TokenExpiry  time.Time `gorm:"column:token_expiry"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (OAuthAccountModel) TableName() string { return "oauth_accounts" }

type SessionModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	UserID    string    `gorm:"column:user_id;index"`
	IPAddress string    `gorm:"column:ip_address"`
	UserAgent string    `gorm:"column:user_agent"`
	ExpiresAt time.Time `gorm:"column:expires_at;index"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (SessionModel) TableName() string { return "sessions" }

type OrganizationModel struct {
	ID               string     `gorm:"column:id;primaryKey"`
	Name             string     `gorm:"column:name"`
	Slug             string     `gorm:"column:slug;uniqueIndex:idx_orgs_slug_not_deleted,where:deleted_at IS NULL"`
	OwnerID          string     `gorm:"column:owner_id;index"`
	StripeCustomerID string     `gorm:"column:stripe_customer_id"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
	DeletedAt        *time.Time `gorm:"column:deleted_at;index"`
}

func (OrganizationModel) TableName() string { return "organizations" }

type OrgMemberModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	OrgID     string    `gorm:"column:org_id;uniqueIndex:idx_org_members_unique,composite:org_user"`
	UserID    string    `gorm:"column:user_id;uniqueIndex:idx_org_members_unique,composite:org_user"`
	Role      string    `gorm:"column:role"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (OrgMemberModel) TableName() string { return "org_members" }

type SubscriptionModel struct {
	ID                   string    `gorm:"column:id;primaryKey"`
	OrgID                string    `gorm:"column:org_id;index"`
	Plan                 string    `gorm:"column:plan"`
	StripeSubscriptionID string    `gorm:"column:stripe_subscription_id;uniqueIndex"`
	StripePriceID        string    `gorm:"column:stripe_price_id"`
	Status               string    `gorm:"column:status"`
	CurrentPeriodStart   time.Time `gorm:"column:current_period_start"`
	CurrentPeriodEnd     time.Time `gorm:"column:current_period_end"`
	CreatedAt            time.Time `gorm:"column:created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
}

func (SubscriptionModel) TableName() string { return "subscriptions" }

type UsageRecordModel struct {
	ID           string    `gorm:"column:id;primaryKey"`
	OrgID        string    `gorm:"column:org_id;index:idx_usage_org_recorded,priority:1"`
	ResourceType string    `gorm:"column:resource_type"`
	Quantity     float64   `gorm:"column:quantity"`
	RecordedAt   time.Time `gorm:"column:recorded_at;index:idx_usage_org_recorded,priority:2"`
	MetadataJSON string    `gorm:"column:metadata_json"`
}

func (UsageRecordModel) TableName() string { return "usage_records" }

// --- Infrastructure GORM models ---

type HostModel struct {
	ID                string              `gorm:"column:id;primaryKey"`
	OrgID             string              `gorm:"column:org_id;not null;index"`
	Hostname          string              `gorm:"column:hostname;not null"`
	Version           string              `gorm:"column:version"`
	TotalCPUs         int32               `gorm:"column:total_cpus;not null;default:0"`
	TotalMemoryMB     int64               `gorm:"column:total_memory_mb;not null;default:0"`
	TotalDiskMB       int64               `gorm:"column:total_disk_mb;not null;default:0"`
	AvailableCPUs     int32               `gorm:"column:available_cpus;not null;default:0"`
	AvailableMemoryMB int64               `gorm:"column:available_memory_mb;not null;default:0"`
	AvailableDiskMB   int64               `gorm:"column:available_disk_mb;not null;default:0"`
	BaseImages        store.StringSlice   `gorm:"column:base_images;type:jsonb;default:'[]'"`
	SourceVMs         store.SourceVMSlice `gorm:"column:source_vms;type:jsonb;default:'[]'"`
	Bridges           store.BridgeSlice   `gorm:"column:bridges;type:jsonb;default:'[]'"`
	Status            string              `gorm:"column:status;not null;default:'OFFLINE'"`
	LastHeartbeat     time.Time           `gorm:"column:last_heartbeat;not null"`
	CreatedAt         time.Time           `gorm:"column:created_at"`
	UpdatedAt         time.Time           `gorm:"column:updated_at"`
}

func (HostModel) TableName() string { return "hosts" }

type SandboxModel struct {
	ID         string     `gorm:"column:id;primaryKey"`
	OrgID      string     `gorm:"column:org_id;not null;index"`
	HostID     string     `gorm:"column:host_id;not null;index"`
	Name       string     `gorm:"column:name;not null"`
	AgentID    string     `gorm:"column:agent_id"`
	BaseImage  string     `gorm:"column:base_image"`
	Bridge     string     `gorm:"column:bridge"`
	TAPDevice  string     `gorm:"column:tap_device"`
	MACAddress string     `gorm:"column:mac_address"`
	IPAddress  string     `gorm:"column:ip_address"`
	State      string     `gorm:"column:state;not null;default:'CREATING'"`
	VCPUs      int32      `gorm:"column:vcpus;not null;default:1"`
	MemoryMB   int32      `gorm:"column:memory_mb;not null;default:512"`
	TTLSeconds int32      `gorm:"column:ttl_seconds;not null;default:0"`
	SourceVM   string     `gorm:"column:source_vm"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at"`
	DeletedAt  *time.Time `gorm:"column:deleted_at;index"`
}

func (SandboxModel) TableName() string { return "sandboxes" }

type CommandModel struct {
	ID         string    `gorm:"column:id;primaryKey"`
	SandboxID  string    `gorm:"column:sandbox_id;not null;index:idx_commands_sandbox_started,priority:1"`
	Command    string    `gorm:"column:command;not null"`
	Stdout     string    `gorm:"column:stdout"`
	Stderr     string    `gorm:"column:stderr"`
	ExitCode   int32     `gorm:"column:exit_code;not null;default:0"`
	DurationMS int64     `gorm:"column:duration_ms;not null;default:0"`
	StartedAt  time.Time `gorm:"column:started_at;not null;index:idx_commands_sandbox_started,priority:2"`
	EndedAt    time.Time `gorm:"column:ended_at"`
}

func (CommandModel) TableName() string { return "commands" }

type HostTokenModel struct {
	ID        string     `gorm:"column:id;primaryKey"`
	OrgID     string     `gorm:"column:org_id;not null;index"`
	Name      string     `gorm:"column:name;not null"`
	TokenHash string     `gorm:"column:token_hash;not null;uniqueIndex"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func (HostTokenModel) TableName() string { return "host_tokens" }

/*
type AgentConversationModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	OrgID     string    `gorm:"column:org_id;not null;index"`
	UserID    string    `gorm:"column:user_id;not null;index"`
	Title     string    `gorm:"column:title;not null"`
	Model     string    `gorm:"column:model"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (AgentConversationModel) TableName() string { return "agent_conversations" }

type AgentMessageModel struct {
	ID             string    `gorm:"column:id;primaryKey"`
	ConversationID string    `gorm:"column:conversation_id;not null;index"`
	Role           string    `gorm:"column:role;not null"`
	Content        string    `gorm:"column:content"`
	ToolCalls      string    `gorm:"column:tool_calls"`
	ToolCallID     string    `gorm:"column:tool_call_id"`
	TokensInput    int       `gorm:"column:tokens_input;default:0"`
	TokensOutput   int       `gorm:"column:tokens_output;default:0"`
	Model          string    `gorm:"column:model"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (AgentMessageModel) TableName() string { return "agent_messages" }

type PlaybookModel struct {
	ID          string    `gorm:"column:id;primaryKey"`
	OrgID       string    `gorm:"column:org_id;not null;index"`
	Name        string    `gorm:"column:name;not null"`
	Description string    `gorm:"column:description"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (PlaybookModel) TableName() string { return "playbooks" }

type PlaybookTaskModel struct {
	ID         string    `gorm:"column:id;primaryKey"`
	PlaybookID string    `gorm:"column:playbook_id;not null;index"`
	SortOrder  int       `gorm:"column:sort_order;not null;default:0"`
	Name       string    `gorm:"column:name;not null"`
	Module     string    `gorm:"column:module;not null"`
	Params     string    `gorm:"column:params;type:jsonb;default:'{}'"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (PlaybookTaskModel) TableName() string { return "playbook_tasks" }
*/

type SourceHostModel struct {
	ID               string            `gorm:"column:id;primaryKey"`
	OrgID            string            `gorm:"column:org_id;not null;index"`
	Name             string            `gorm:"column:name;not null"`
	Hostname         string            `gorm:"column:hostname;not null"`
	Type             string            `gorm:"column:type;not null;default:'libvirt'"`
	SSHUser          string            `gorm:"column:ssh_user"`
	SSHPort          int               `gorm:"column:ssh_port;not null;default:22"`
	SSHIdentityFile  string            `gorm:"column:ssh_identity_file"`
	ProxmoxHost      string            `gorm:"column:proxmox_host"`
	ProxmoxTokenID   string            `gorm:"column:proxmox_token_id"`
	ProxmoxSecret    string            `gorm:"column:proxmox_secret"`
	ProxmoxNode      string            `gorm:"column:proxmox_node"`
	ProxmoxVerifySSL bool              `gorm:"column:proxmox_verify_ssl;default:false"`
	VMs              store.StringSlice `gorm:"column:vms;type:jsonb;default:'[]'"`
	CreatedAt        time.Time         `gorm:"column:created_at"`
	UpdatedAt        time.Time         `gorm:"column:updated_at"`
}

func (SourceHostModel) TableName() string { return "source_hosts" }

type ModelMeterModel struct {
	ID                  string    `gorm:"column:id;primaryKey"`
	ModelID             string    `gorm:"column:model_id;uniqueIndex"`
	StripeProductID     string    `gorm:"column:stripe_product_id"`
	StripeInputMeterID  string    `gorm:"column:stripe_input_meter_id"`
	StripeOutputMeterID string    `gorm:"column:stripe_output_meter_id"`
	StripeInputPriceID  string    `gorm:"column:stripe_input_price_id"`
	StripeOutputPriceID string    `gorm:"column:stripe_output_price_id"`
	InputEventName      string    `gorm:"column:input_event_name"`
	OutputEventName     string    `gorm:"column:output_event_name"`
	InputCostPerToken   float64   `gorm:"column:input_cost_per_token"`
	OutputCostPerToken  float64   `gorm:"column:output_cost_per_token"`
	CreatedAt           time.Time `gorm:"column:created_at"`
}

func (ModelMeterModel) TableName() string { return "model_meters" }

type OrgModelSubscriptionModel struct {
	ID                    string    `gorm:"column:id;primaryKey"`
	OrgID                 string    `gorm:"column:org_id;uniqueIndex:idx_org_model_sub,composite:org_model"`
	ModelID               string    `gorm:"column:model_id;uniqueIndex:idx_org_model_sub,composite:org_model"`
	StripeInputSubItemID  string    `gorm:"column:stripe_input_sub_item_id"`
	StripeOutputSubItemID string    `gorm:"column:stripe_output_sub_item_id"`
	CreatedAt             time.Time `gorm:"column:created_at"`
}

func (OrgModelSubscriptionModel) TableName() string { return "org_model_subscriptions" }

// New creates a Store backed by Postgres + GORM.
func New(ctx context.Context, cfg store.Config) (store.Store, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("postgres: missing DatabaseURL")
	}

	db, err := gorm.Open(
		postgres.Open(cfg.DatabaseURL),
		&gorm.Config{
			NowFunc: func() time.Time { return time.Now().UTC() },
			Logger:  logger.Default.LogMode(logger.Silent),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("postgres: sql.DB handle: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	var encKey []byte
	if cfg.EncryptionKey != "" {
		encKey = crypto.DeriveKey(cfg.EncryptionKey)
	}
	pg := &postgresStore{db: db.WithContext(ctx), conf: cfg, encryptionKey: encKey}

	if cfg.AutoMigrate {
		if err := pg.autoMigrate(ctx); err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
	}

	if err := pg.Ping(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return pg, nil
}

func (s *postgresStore) autoMigrate(_ context.Context) error {
	return s.db.AutoMigrate(
		&UserModel{},
		&OAuthAccountModel{},
		&SessionModel{},
		&OrganizationModel{},
		&OrgMemberModel{},
		&SubscriptionModel{},
		&UsageRecordModel{},
		&HostModel{},
		&SandboxModel{},
		&CommandModel{},
		&HostTokenModel{},
		// &AgentConversationModel{},
		// &AgentMessageModel{},
		// &PlaybookModel{},
		// &PlaybookTaskModel{},
		&SourceHostModel{},
		&ModelMeterModel{},
		&OrgModelSubscriptionModel{},
	)
}

func (s *postgresStore) Config() store.Config { return s.conf }

func (s *postgresStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *postgresStore) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (s *postgresStore) WithTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&postgresStore{db: tx, conf: s.conf, encryptionKey: s.encryptionKey})
	})
}

// --- Model converters ---

func userToModel(u *store.User) *UserModel {
	return &UserModel{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		AvatarURL:     u.AvatarURL,
		PasswordHash:  u.PasswordHash,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		DeletedAt:     u.DeletedAt,
	}
}

func userFromModel(m *UserModel) *store.User {
	return &store.User{
		ID:            m.ID,
		Email:         m.Email,
		DisplayName:   m.DisplayName,
		AvatarURL:     m.AvatarURL,
		PasswordHash:  m.PasswordHash,
		EmailVerified: m.EmailVerified,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DeletedAt:     m.DeletedAt,
	}
}

func (s *postgresStore) oauthToModel(oa *store.OAuthAccount) *OAuthAccountModel {
	m := &OAuthAccountModel{
		ID:           oa.ID,
		UserID:       oa.UserID,
		Provider:     oa.Provider,
		ProviderID:   oa.ProviderID,
		Email:        oa.Email,
		AccessToken:  oa.AccessToken,
		RefreshToken: oa.RefreshToken,
		TokenExpiry:  oa.TokenExpiry,
		CreatedAt:    oa.CreatedAt,
	}
	if len(s.encryptionKey) > 0 {
		if enc, err := crypto.Encrypt(s.encryptionKey, oa.AccessToken); err == nil {
			m.AccessToken = enc
		} else {
			slog.Warn("failed to encrypt access token", "error", err)
		}
		if enc, err := crypto.Encrypt(s.encryptionKey, oa.RefreshToken); err == nil {
			m.RefreshToken = enc
		} else {
			slog.Warn("failed to encrypt refresh token", "error", err)
		}
	}
	return m
}

func (s *postgresStore) oauthFromModel(m *OAuthAccountModel) *store.OAuthAccount {
	oa := &store.OAuthAccount{
		ID:           m.ID,
		UserID:       m.UserID,
		Provider:     m.Provider,
		ProviderID:   m.ProviderID,
		Email:        m.Email,
		AccessToken:  m.AccessToken,
		RefreshToken: m.RefreshToken,
		TokenExpiry:  m.TokenExpiry,
		CreatedAt:    m.CreatedAt,
	}
	if len(s.encryptionKey) > 0 {
		if dec, err := crypto.Decrypt(s.encryptionKey, m.AccessToken); err == nil {
			oa.AccessToken = dec
		}
		if dec, err := crypto.Decrypt(s.encryptionKey, m.RefreshToken); err == nil {
			oa.RefreshToken = dec
		}
	}
	return oa
}

func sessionToModel(s *store.Session) *SessionModel {
	return &SessionModel{
		ID:        s.ID,
		UserID:    s.UserID,
		IPAddress: s.IPAddress,
		UserAgent: s.UserAgent,
		ExpiresAt: s.ExpiresAt,
		CreatedAt: s.CreatedAt,
	}
}

func sessionFromModel(m *SessionModel) *store.Session {
	return &store.Session{
		ID:        m.ID,
		UserID:    m.UserID,
		IPAddress: m.IPAddress,
		UserAgent: m.UserAgent,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}
}

func orgToModel(o *store.Organization) *OrganizationModel {
	return &OrganizationModel{
		ID:               o.ID,
		Name:             o.Name,
		Slug:             o.Slug,
		OwnerID:          o.OwnerID,
		StripeCustomerID: o.StripeCustomerID,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
		DeletedAt:        o.DeletedAt,
	}
}

func orgFromModel(m *OrganizationModel) *store.Organization {
	return &store.Organization{
		ID:               m.ID,
		Name:             m.Name,
		Slug:             m.Slug,
		OwnerID:          m.OwnerID,
		StripeCustomerID: m.StripeCustomerID,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		DeletedAt:        m.DeletedAt,
	}
}

func memberToModel(m *store.OrgMember) *OrgMemberModel {
	return &OrgMemberModel{
		ID:        m.ID,
		OrgID:     m.OrgID,
		UserID:    m.UserID,
		Role:      string(m.Role),
		CreatedAt: m.CreatedAt,
	}
}

func memberFromModel(m *OrgMemberModel) *store.OrgMember {
	return &store.OrgMember{
		ID:        m.ID,
		OrgID:     m.OrgID,
		UserID:    m.UserID,
		Role:      store.OrgRole(m.Role),
		CreatedAt: m.CreatedAt,
	}
}

func subToModel(s *store.Subscription) *SubscriptionModel {
	return &SubscriptionModel{
		ID:                   s.ID,
		OrgID:                s.OrgID,
		Plan:                 string(s.Plan),
		StripeSubscriptionID: s.StripeSubscriptionID,
		StripePriceID:        s.StripePriceID,
		Status:               string(s.Status),
		CurrentPeriodStart:   s.CurrentPeriodStart,
		CurrentPeriodEnd:     s.CurrentPeriodEnd,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}

func subFromModel(m *SubscriptionModel) *store.Subscription {
	return &store.Subscription{
		ID:                   m.ID,
		OrgID:                m.OrgID,
		Plan:                 store.SubscriptionPlan(m.Plan),
		StripeSubscriptionID: m.StripeSubscriptionID,
		StripePriceID:        m.StripePriceID,
		Status:               store.SubscriptionStatus(m.Status),
		CurrentPeriodStart:   m.CurrentPeriodStart,
		CurrentPeriodEnd:     m.CurrentPeriodEnd,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
}

func usageToModel(r *store.UsageRecord) *UsageRecordModel {
	return &UsageRecordModel{
		ID:           r.ID,
		OrgID:        r.OrgID,
		ResourceType: r.ResourceType,
		Quantity:     r.Quantity,
		RecordedAt:   r.RecordedAt,
		MetadataJSON: r.MetadataJSON,
	}
}

func usageFromModel(m *UsageRecordModel) *store.UsageRecord {
	return &store.UsageRecord{
		ID:           m.ID,
		OrgID:        m.OrgID,
		ResourceType: m.ResourceType,
		Quantity:     m.Quantity,
		RecordedAt:   m.RecordedAt,
		MetadataJSON: m.MetadataJSON,
	}
}

// mapDBError converts GORM/Postgres errors to sentinel errors.
func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return store.ErrNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return store.ErrAlreadyExists
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return store.ErrAlreadyExists
		case "23503":
			return store.ErrInvalid
		}
	}
	return err
}

// --- User CRUD ---

func (s *postgresStore) CreateUser(ctx context.Context, u *store.User) error {
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(userToModel(u)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetUser(ctx context.Context, id string) (*store.User, error) {
	var model UserModel
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return userFromModel(&model), nil
}

func (s *postgresStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	var model UserModel
	if err := s.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return userFromModel(&model), nil
}

func (s *postgresStore) UpdateUser(ctx context.Context, u *store.User) error {
	u.UpdatedAt = time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&UserModel{}).Where("id = ? AND deleted_at IS NULL", u.ID).
		Updates(map[string]any{
			"email":          u.Email,
			"display_name":   u.DisplayName,
			"avatar_url":     u.AvatarURL,
			"password_hash":  u.PasswordHash,
			"email_verified": u.EmailVerified,
			"updated_at":     u.UpdatedAt,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

// --- OAuth CRUD ---

func (s *postgresStore) CreateOAuthAccount(ctx context.Context, oa *store.OAuthAccount) error {
	oa.CreatedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Create(s.oauthToModel(oa)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetOAuthAccount(ctx context.Context, provider, providerID string) (*store.OAuthAccount, error) {
	var model OAuthAccountModel
	if err := s.db.WithContext(ctx).Where("provider = ? AND provider_id = ?", provider, providerID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return s.oauthFromModel(&model), nil
}

func (s *postgresStore) GetOAuthAccountsByUser(ctx context.Context, userID string) ([]*store.OAuthAccount, error) {
	var models []OAuthAccountModel
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.OAuthAccount, 0, len(models))
	for i := range models {
		out = append(out, s.oauthFromModel(&models[i]))
	}
	return out, nil
}

// --- Session CRUD ---

func (s *postgresStore) CreateSession(ctx context.Context, sess *store.Session) error {
	sess.CreatedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Create(sessionToModel(sess)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetSession(ctx context.Context, id string) (*store.Session, error) {
	var model SessionModel
	if err := s.db.WithContext(ctx).Where("id = ? AND expires_at > ?", id, time.Now().UTC()).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return sessionFromModel(&model), nil
}

func (s *postgresStore) DeleteSession(ctx context.Context, id string) error {
	res := s.db.WithContext(ctx).Where("id = ?", id).Delete(&SessionModel{})
	if res.Error != nil {
		return mapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) DeleteExpiredSessions(ctx context.Context) error {
	return s.db.WithContext(ctx).Where("expires_at <= ?", time.Now().UTC()).Delete(&SessionModel{}).Error
}

// --- Organization CRUD ---

func (s *postgresStore) CreateOrganization(ctx context.Context, org *store.Organization) error {
	now := time.Now().UTC()
	org.CreatedAt = now
	org.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(orgToModel(org)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetOrganization(ctx context.Context, id string) (*store.Organization, error) {
	var model OrganizationModel
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return orgFromModel(&model), nil
}

func (s *postgresStore) GetOrganizationBySlug(ctx context.Context, slug string) (*store.Organization, error) {
	var model OrganizationModel
	if err := s.db.WithContext(ctx).Where("slug = ? AND deleted_at IS NULL", slug).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return orgFromModel(&model), nil
}

func (s *postgresStore) ListOrganizationsByUser(ctx context.Context, userID string) ([]*store.Organization, error) {
	var models []OrganizationModel
	if err := s.db.WithContext(ctx).
		Joins("JOIN org_members ON org_members.org_id = organizations.id").
		Where("org_members.user_id = ? AND organizations.deleted_at IS NULL", userID).
		Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.Organization, 0, len(models))
	for i := range models {
		out = append(out, orgFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) UpdateOrganization(ctx context.Context, org *store.Organization) error {
	org.UpdatedAt = time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&OrganizationModel{}).Where("id = ? AND deleted_at IS NULL", org.ID).
		Updates(map[string]any{
			"name":               org.Name,
			"slug":               org.Slug,
			"stripe_customer_id": org.StripeCustomerID,
			"updated_at":         org.UpdatedAt,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) DeleteOrganization(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&OrganizationModel{}).Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"deleted_at": &now,
			"updated_at": now,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

// --- OrgMember CRUD ---

func (s *postgresStore) CreateOrgMember(ctx context.Context, m *store.OrgMember) error {
	m.CreatedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Create(memberToModel(m)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetOrgMember(ctx context.Context, orgID, userID string) (*store.OrgMember, error) {
	var model OrgMemberModel
	if err := s.db.WithContext(ctx).Where("org_id = ? AND user_id = ?", orgID, userID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return memberFromModel(&model), nil
}

func (s *postgresStore) GetOrgMemberByID(ctx context.Context, orgID, memberID string) (*store.OrgMember, error) {
	var model OrgMemberModel
	if err := s.db.WithContext(ctx).Where("id = ? AND org_id = ?", memberID, orgID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return memberFromModel(&model), nil
}

func (s *postgresStore) ListOrgMembers(ctx context.Context, orgID string) ([]*store.OrgMember, error) {
	var models []OrgMemberModel
	if err := s.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.OrgMember, 0, len(models))
	for i := range models {
		out = append(out, memberFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) DeleteOrgMember(ctx context.Context, orgID, id string) error {
	res := s.db.WithContext(ctx).Where("id = ? AND org_id = ?", id, orgID).Delete(&OrgMemberModel{})
	if res.Error != nil {
		return mapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

// --- Subscription CRUD ---

func (s *postgresStore) CreateSubscription(ctx context.Context, sub *store.Subscription) error {
	now := time.Now().UTC()
	sub.CreatedAt = now
	sub.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(subToModel(sub)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetSubscriptionByOrg(ctx context.Context, orgID string) (*store.Subscription, error) {
	var model SubscriptionModel
	if err := s.db.WithContext(ctx).Where("org_id = ?", orgID).Order("created_at DESC").First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return subFromModel(&model), nil
}

func (s *postgresStore) UpdateSubscription(ctx context.Context, sub *store.Subscription) error {
	sub.UpdatedAt = time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&SubscriptionModel{}).Where("id = ?", sub.ID).
		Updates(map[string]any{
			"plan":                   string(sub.Plan),
			"stripe_subscription_id": sub.StripeSubscriptionID,
			"stripe_price_id":        sub.StripePriceID,
			"status":                 string(sub.Status),
			"current_period_start":   sub.CurrentPeriodStart,
			"current_period_end":     sub.CurrentPeriodEnd,
			"updated_at":             sub.UpdatedAt,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*store.Subscription, error) {
	var model SubscriptionModel
	if err := s.db.WithContext(ctx).Where("stripe_subscription_id = ?", stripeSubID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return subFromModel(&model), nil
}

// --- UsageRecord CRUD ---

func (s *postgresStore) CreateUsageRecord(ctx context.Context, rec *store.UsageRecord) error {
	rec.RecordedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Create(usageToModel(rec)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) ListUsageRecords(ctx context.Context, orgID string, from, to time.Time) ([]*store.UsageRecord, error) {
	var models []UsageRecordModel
	if err := s.db.WithContext(ctx).
		Where("org_id = ? AND recorded_at >= ? AND recorded_at <= ?", orgID, from, to).
		Order("recorded_at DESC").
		Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.UsageRecord, 0, len(models))
	for i := range models {
		out = append(out, usageFromModel(&models[i]))
	}
	return out, nil
}

// --- Infrastructure model converters ---

func hostToModel(h *store.Host) *HostModel {
	return &HostModel{
		ID:                h.ID,
		OrgID:             h.OrgID,
		Hostname:          h.Hostname,
		Version:           h.Version,
		TotalCPUs:         h.TotalCPUs,
		TotalMemoryMB:     h.TotalMemoryMB,
		TotalDiskMB:       h.TotalDiskMB,
		AvailableCPUs:     h.AvailableCPUs,
		AvailableMemoryMB: h.AvailableMemoryMB,
		AvailableDiskMB:   h.AvailableDiskMB,
		BaseImages:        h.BaseImages,
		SourceVMs:         h.SourceVMs,
		Bridges:           h.Bridges,
		Status:            string(h.Status),
		LastHeartbeat:     h.LastHeartbeat,
		CreatedAt:         h.CreatedAt,
		UpdatedAt:         h.UpdatedAt,
	}
}

func hostFromModel(m *HostModel) *store.Host {
	return &store.Host{
		ID:                m.ID,
		OrgID:             m.OrgID,
		Hostname:          m.Hostname,
		Version:           m.Version,
		TotalCPUs:         m.TotalCPUs,
		TotalMemoryMB:     m.TotalMemoryMB,
		TotalDiskMB:       m.TotalDiskMB,
		AvailableCPUs:     m.AvailableCPUs,
		AvailableMemoryMB: m.AvailableMemoryMB,
		AvailableDiskMB:   m.AvailableDiskMB,
		BaseImages:        m.BaseImages,
		SourceVMs:         m.SourceVMs,
		Bridges:           m.Bridges,
		Status:            store.HostStatus(m.Status),
		LastHeartbeat:     m.LastHeartbeat,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func sandboxToModel(s *store.Sandbox) *SandboxModel {
	return &SandboxModel{
		ID:         s.ID,
		OrgID:      s.OrgID,
		HostID:     s.HostID,
		Name:       s.Name,
		AgentID:    s.AgentID,
		BaseImage:  s.BaseImage,
		Bridge:     s.Bridge,
		TAPDevice:  s.TAPDevice,
		MACAddress: s.MACAddress,
		IPAddress:  s.IPAddress,
		State:      string(s.State),
		VCPUs:      s.VCPUs,
		MemoryMB:   s.MemoryMB,
		TTLSeconds: s.TTLSeconds,
		SourceVM:   s.SourceVM,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
		DeletedAt:  s.DeletedAt,
	}
}

func sandboxFromModel(m *SandboxModel) *store.Sandbox {
	return &store.Sandbox{
		ID:         m.ID,
		OrgID:      m.OrgID,
		HostID:     m.HostID,
		Name:       m.Name,
		AgentID:    m.AgentID,
		BaseImage:  m.BaseImage,
		Bridge:     m.Bridge,
		TAPDevice:  m.TAPDevice,
		MACAddress: m.MACAddress,
		IPAddress:  m.IPAddress,
		State:      store.SandboxState(m.State),
		VCPUs:      m.VCPUs,
		MemoryMB:   m.MemoryMB,
		TTLSeconds: m.TTLSeconds,
		SourceVM:   m.SourceVM,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		DeletedAt:  m.DeletedAt,
	}
}

func commandToModel(c *store.Command) *CommandModel {
	return &CommandModel{
		ID:         c.ID,
		SandboxID:  c.SandboxID,
		Command:    c.Command,
		Stdout:     c.Stdout,
		Stderr:     c.Stderr,
		ExitCode:   c.ExitCode,
		DurationMS: c.DurationMS,
		StartedAt:  c.StartedAt,
		EndedAt:    c.EndedAt,
	}
}

func commandFromModel(m *CommandModel) *store.Command {
	return &store.Command{
		ID:         m.ID,
		SandboxID:  m.SandboxID,
		Command:    m.Command,
		Stdout:     m.Stdout,
		Stderr:     m.Stderr,
		ExitCode:   m.ExitCode,
		DurationMS: m.DurationMS,
		StartedAt:  m.StartedAt,
		EndedAt:    m.EndedAt,
	}
}

func hostTokenToModel(t *store.HostToken) *HostTokenModel {
	return &HostTokenModel{
		ID:        t.ID,
		OrgID:     t.OrgID,
		Name:      t.Name,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
	}
}

func hostTokenFromModel(m *HostTokenModel) *store.HostToken {
	return &store.HostToken{
		ID:        m.ID,
		OrgID:     m.OrgID,
		Name:      m.Name,
		TokenHash: m.TokenHash,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}
}

// --- Host CRUD ---

func (s *postgresStore) CreateHost(ctx context.Context, host *store.Host) error {
	now := time.Now().UTC()
	host.CreatedAt = now
	host.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(hostToModel(host)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetHost(ctx context.Context, hostID string) (*store.Host, error) {
	var model HostModel
	if err := s.db.WithContext(ctx).Where("id = ?", hostID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return hostFromModel(&model), nil
}

func (s *postgresStore) ListHosts(ctx context.Context) ([]store.Host, error) {
	var models []HostModel
	if err := s.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]store.Host, 0, len(models))
	for i := range models {
		out = append(out, *hostFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) ListHostsByOrg(ctx context.Context, orgID string) ([]store.Host, error) {
	var models []HostModel
	if err := s.db.WithContext(ctx).Where("org_id = ?", orgID).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]store.Host, 0, len(models))
	for i := range models {
		out = append(out, *hostFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) UpdateHost(ctx context.Context, host *store.Host) error {
	host.UpdatedAt = time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&HostModel{}).Where("id = ?", host.ID).
		Updates(map[string]any{
			"org_id":              host.OrgID,
			"hostname":            host.Hostname,
			"version":             host.Version,
			"total_cpus":          host.TotalCPUs,
			"total_memory_mb":     host.TotalMemoryMB,
			"total_disk_mb":       host.TotalDiskMB,
			"available_cpus":      host.AvailableCPUs,
			"available_memory_mb": host.AvailableMemoryMB,
			"available_disk_mb":   host.AvailableDiskMB,
			"base_images":         host.BaseImages,
			"source_vms":          host.SourceVMs,
			"bridges":             host.Bridges,
			"status":              string(host.Status),
			"last_heartbeat":      host.LastHeartbeat,
			"updated_at":          host.UpdatedAt,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) UpdateHostHeartbeat(ctx context.Context, hostID string, availCPUs int32, availMemMB int64, availDiskMB int64) error {
	res := s.db.WithContext(ctx).
		Model(&HostModel{}).
		Where("id = ?", hostID).
		Updates(map[string]any{
			"available_cpus":      availCPUs,
			"available_memory_mb": availMemMB,
			"available_disk_mb":   availDiskMB,
			"status":              string(store.HostStatusOnline),
			"last_heartbeat":      time.Now().UTC(),
			"updated_at":          time.Now().UTC(),
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

// --- Sandbox CRUD ---

func (s *postgresStore) CreateSandbox(ctx context.Context, sandbox *store.Sandbox) error {
	now := time.Now().UTC()
	sandbox.CreatedAt = now
	sandbox.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(sandboxToModel(sandbox)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetSandbox(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
	var model SandboxModel
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", sandboxID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return sandboxFromModel(&model), nil
}

func (s *postgresStore) GetSandboxByOrg(ctx context.Context, orgID, sandboxID string) (*store.Sandbox, error) {
	var model SandboxModel
	if err := s.db.WithContext(ctx).Where("id = ? AND org_id = ? AND deleted_at IS NULL", sandboxID, orgID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return sandboxFromModel(&model), nil
}

func (s *postgresStore) ListSandboxes(ctx context.Context) ([]store.Sandbox, error) {
	var models []SandboxModel
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]store.Sandbox, 0, len(models))
	for i := range models {
		out = append(out, *sandboxFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) ListSandboxesByOrg(ctx context.Context, orgID string) ([]store.Sandbox, error) {
	var models []SandboxModel
	if err := s.db.WithContext(ctx).Where("org_id = ? AND deleted_at IS NULL", orgID).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]store.Sandbox, 0, len(models))
	for i := range models {
		out = append(out, *sandboxFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) UpdateSandbox(ctx context.Context, sandbox *store.Sandbox) error {
	sandbox.UpdatedAt = time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&SandboxModel{}).Where("id = ? AND deleted_at IS NULL", sandbox.ID).
		Updates(map[string]any{
			"state":       string(sandbox.State),
			"ip_address":  sandbox.IPAddress,
			"tap_device":  sandbox.TAPDevice,
			"mac_address": sandbox.MACAddress,
			"bridge":      sandbox.Bridge,
			"updated_at":  sandbox.UpdatedAt,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) DeleteSandbox(ctx context.Context, sandboxID string) error {
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).
		Model(&SandboxModel{}).
		Where("id = ? AND deleted_at IS NULL", sandboxID).
		Updates(map[string]any{
			"deleted_at": &now,
			"state":      string(store.SandboxStateDestroyed),
			"updated_at": now,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) GetSandboxesByHostID(ctx context.Context, hostID string) ([]store.Sandbox, error) {
	var models []SandboxModel
	if err := s.db.WithContext(ctx).Where("host_id = ? AND deleted_at IS NULL", hostID).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]store.Sandbox, 0, len(models))
	for i := range models {
		out = append(out, *sandboxFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) CountSandboxesByHostIDs(ctx context.Context, hostIDs []string) (map[string]int, error) {
	if len(hostIDs) == 0 {
		return map[string]int{}, nil
	}
	type row struct {
		HostID string
		Count  int
	}
	var rows []row
	err := s.db.WithContext(ctx).
		Model(&SandboxModel{}).
		Select("host_id, COUNT(*) as count").
		Where("host_id IN ? AND deleted_at IS NULL", hostIDs).
		Group("host_id").
		Find(&rows).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	result := make(map[string]int, len(rows))
	for _, r := range rows {
		result[r.HostID] = r.Count
	}
	return result, nil
}

func (s *postgresStore) ListExpiredSandboxes(ctx context.Context, defaultTTL time.Duration) ([]store.Sandbox, error) {
	now := time.Now().UTC()
	query := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("state IN ?", []string{string(store.SandboxStateRunning), string(store.SandboxStateStopped)})

	if defaultTTL > 0 {
		defaultTTLSeconds := int32(defaultTTL.Seconds())
		query = query.Where(
			"(ttl_seconds > 0 AND created_at + (ttl_seconds || ' seconds')::interval < ?) "+
				"OR (ttl_seconds = 0 AND created_at + (? || ' seconds')::interval < ?)",
			now, defaultTTLSeconds, now,
		)
	} else {
		query = query.Where(
			"ttl_seconds > 0 AND created_at + (ttl_seconds || ' seconds')::interval < ?",
			now,
		)
	}

	var models []SandboxModel
	if err := query.Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]store.Sandbox, 0, len(models))
	for i := range models {
		out = append(out, *sandboxFromModel(&models[i]))
	}
	return out, nil
}

// --- Command CRUD ---

func (s *postgresStore) CreateCommand(ctx context.Context, cmd *store.Command) error {
	if err := s.db.WithContext(ctx).Create(commandToModel(cmd)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) ListSandboxCommands(ctx context.Context, sandboxID string) ([]store.Command, error) {
	var models []CommandModel
	if err := s.db.WithContext(ctx).
		Where("sandbox_id = ?", sandboxID).
		Order("started_at ASC").
		Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]store.Command, 0, len(models))
	for i := range models {
		out = append(out, *commandFromModel(&models[i]))
	}
	return out, nil
}

// --- SourceHost CRUD ---

func (s *postgresStore) CreateSourceHost(ctx context.Context, sh *store.SourceHost) error {
	sh.CreatedAt = time.Now().UTC()
	sh.UpdatedAt = sh.CreatedAt
	if err := s.db.WithContext(ctx).Create(s.sourceHostToModel(sh)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetSourceHost(ctx context.Context, id string) (*store.SourceHost, error) {
	var model SourceHostModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return s.sourceHostFromModel(&model), nil
}

func (s *postgresStore) ListSourceHostsByOrg(ctx context.Context, orgID string) ([]*store.SourceHost, error) {
	var models []SourceHostModel
	if err := s.db.WithContext(ctx).Where("org_id = ?", orgID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.SourceHost, 0, len(models))
	for i := range models {
		out = append(out, s.sourceHostFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) DeleteSourceHost(ctx context.Context, id string) error {
	res := s.db.WithContext(ctx).Where("id = ?", id).Delete(&SourceHostModel{})
	if res.Error != nil {
		return mapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) sourceHostToModel(sh *store.SourceHost) *SourceHostModel {
	m := &SourceHostModel{
		ID:               sh.ID,
		OrgID:            sh.OrgID,
		Name:             sh.Name,
		Hostname:         sh.Hostname,
		Type:             sh.Type,
		SSHUser:          sh.SSHUser,
		SSHPort:          sh.SSHPort,
		SSHIdentityFile:  sh.SSHIdentityFile,
		ProxmoxHost:      sh.ProxmoxHost,
		ProxmoxTokenID:   sh.ProxmoxTokenID,
		ProxmoxSecret:    sh.ProxmoxSecret,
		ProxmoxNode:      sh.ProxmoxNode,
		ProxmoxVerifySSL: sh.ProxmoxVerifySSL,
		VMs:              sh.VMs,
		CreatedAt:        sh.CreatedAt,
		UpdatedAt:        sh.UpdatedAt,
	}
	if len(s.encryptionKey) > 0 {
		if enc, err := crypto.Encrypt(s.encryptionKey, sh.ProxmoxTokenID); err == nil {
			m.ProxmoxTokenID = enc
		}
		if enc, err := crypto.Encrypt(s.encryptionKey, sh.ProxmoxSecret); err == nil {
			m.ProxmoxSecret = enc
		}
	}
	return m
}

func (s *postgresStore) sourceHostFromModel(m *SourceHostModel) *store.SourceHost {
	sh := &store.SourceHost{
		ID:               m.ID,
		OrgID:            m.OrgID,
		Name:             m.Name,
		Hostname:         m.Hostname,
		Type:             m.Type,
		SSHUser:          m.SSHUser,
		SSHPort:          m.SSHPort,
		SSHIdentityFile:  m.SSHIdentityFile,
		ProxmoxHost:      m.ProxmoxHost,
		ProxmoxTokenID:   m.ProxmoxTokenID,
		ProxmoxSecret:    m.ProxmoxSecret,
		ProxmoxNode:      m.ProxmoxNode,
		ProxmoxVerifySSL: m.ProxmoxVerifySSL,
		VMs:              m.VMs,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
	if len(s.encryptionKey) > 0 {
		if dec, err := crypto.Decrypt(s.encryptionKey, m.ProxmoxTokenID); err == nil {
			sh.ProxmoxTokenID = dec
		}
		if dec, err := crypto.Decrypt(s.encryptionKey, m.ProxmoxSecret); err == nil {
			sh.ProxmoxSecret = dec
		}
	}
	return sh
}

// --- HostToken CRUD ---

func (s *postgresStore) CreateHostToken(ctx context.Context, token *store.HostToken) error {
	token.CreatedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Create(hostTokenToModel(token)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetHostTokenByHash(ctx context.Context, hash string) (*store.HostToken, error) {
	var model HostTokenModel
	if err := s.db.WithContext(ctx).Where("token_hash = ? AND (expires_at IS NULL OR expires_at > ?)", hash, time.Now()).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return hostTokenFromModel(&model), nil
}

func (s *postgresStore) ListHostTokensByOrg(ctx context.Context, orgID string) ([]store.HostToken, error) {
	var models []HostTokenModel
	if err := s.db.WithContext(ctx).Where("org_id = ?", orgID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]store.HostToken, 0, len(models))
	for i := range models {
		out = append(out, *hostTokenFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) DeleteHostToken(ctx context.Context, orgID, id string) error {
	res := s.db.WithContext(ctx).Where("id = ? AND org_id = ?", id, orgID).Delete(&HostTokenModel{})
	if res.Error != nil {
		return mapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

/*
// --- Agent Conversation converters ---

func convToModel(c *store.AgentConversation) *AgentConversationModel {
	return &AgentConversationModel{
		ID:        c.ID,
		OrgID:     c.OrgID,
		UserID:    c.UserID,
		Title:     c.Title,
		Model:     c.Model,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func convFromModel(m *AgentConversationModel) *store.AgentConversation {
	return &store.AgentConversation{
		ID:        m.ID,
		OrgID:     m.OrgID,
		UserID:    m.UserID,
		Title:     m.Title,
		Model:     m.Model,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func msgToModel(m *store.AgentMessage) *AgentMessageModel {
	return &AgentMessageModel{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           string(m.Role),
		Content:        m.Content,
		ToolCalls:      m.ToolCalls,
		ToolCallID:     m.ToolCallID,
		TokensInput:    m.TokensInput,
		TokensOutput:   m.TokensOutput,
		Model:          m.Model,
		CreatedAt:      m.CreatedAt,
	}
}

func msgFromModel(m *AgentMessageModel) *store.AgentMessage {
	return &store.AgentMessage{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           store.MessageRole(m.Role),
		Content:        m.Content,
		ToolCalls:      m.ToolCalls,
		ToolCallID:     m.ToolCallID,
		TokensInput:    m.TokensInput,
		TokensOutput:   m.TokensOutput,
		Model:          m.Model,
		CreatedAt:      m.CreatedAt,
	}
}

func pbToModel(p *store.Playbook) *PlaybookModel {
	return &PlaybookModel{
		ID:          p.ID,
		OrgID:       p.OrgID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func pbFromModel(m *PlaybookModel) *store.Playbook {
	return &store.Playbook{
		ID:          m.ID,
		OrgID:       m.OrgID,
		Name:        m.Name,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func taskToModel(t *store.PlaybookTask) *PlaybookTaskModel {
	return &PlaybookTaskModel{
		ID:         t.ID,
		PlaybookID: t.PlaybookID,
		SortOrder:  t.SortOrder,
		Name:       t.Name,
		Module:     t.Module,
		Params:     t.Params,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func taskFromModel(m *PlaybookTaskModel) *store.PlaybookTask {
	return &store.PlaybookTask{
		ID:         m.ID,
		PlaybookID: m.PlaybookID,
		SortOrder:  m.SortOrder,
		Name:       m.Name,
		Module:     m.Module,
		Params:     m.Params,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
*/

/*
// --- Agent Conversation CRUD ---

func (s *postgresStore) CreateAgentConversation(ctx context.Context, conv *store.AgentConversation) error {
	now := time.Now().UTC()
	conv.CreatedAt = now
	conv.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(convToModel(conv)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetAgentConversation(ctx context.Context, id string) (*store.AgentConversation, error) {
	var model AgentConversationModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return convFromModel(&model), nil
}

func (s *postgresStore) ListAgentConversationsByOrg(ctx context.Context, orgID string) ([]*store.AgentConversation, error) {
	var models []AgentConversationModel
	if err := s.db.WithContext(ctx).Where("org_id = ?", orgID).Order("updated_at DESC").Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.AgentConversation, 0, len(models))
	for i := range models {
		out = append(out, convFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) DeleteAgentConversation(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("conversation_id = ?", id).Delete(&AgentMessageModel{}).Error; err != nil {
			return mapDBError(err)
		}
		res := tx.Where("id = ?", id).Delete(&AgentConversationModel{})
		if res.Error != nil {
			return mapDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return store.ErrNotFound
		}
		return nil
	})
}

// --- Agent Message CRUD ---

func (s *postgresStore) CreateAgentMessage(ctx context.Context, msg *store.AgentMessage) error {
	msg.CreatedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Create(msgToModel(msg)).Error; err != nil {
		return mapDBError(err)
	}
	// Update conversation updated_at
	if err := s.db.WithContext(ctx).Model(&AgentConversationModel{}).
		Where("id = ?", msg.ConversationID).
		Update("updated_at", time.Now().UTC()).Error; err != nil {
		slog.Warn("failed to update conversation updated_at",
			"conversation_id", msg.ConversationID, "error", err)
	}
	return nil
}

func (s *postgresStore) ListAgentMessages(ctx context.Context, conversationID string) ([]*store.AgentMessage, error) {
	var models []AgentMessageModel
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.AgentMessage, 0, len(models))
	for i := range models {
		out = append(out, msgFromModel(&models[i]))
	}
	return out, nil
}

// --- Playbook CRUD ---

func (s *postgresStore) CreatePlaybook(ctx context.Context, pb *store.Playbook) error {
	now := time.Now().UTC()
	pb.CreatedAt = now
	pb.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(pbToModel(pb)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetPlaybook(ctx context.Context, id string) (*store.Playbook, error) {
	var model PlaybookModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return pbFromModel(&model), nil
}

func (s *postgresStore) ListPlaybooksByOrg(ctx context.Context, orgID string) ([]*store.Playbook, error) {
	var models []PlaybookModel
	if err := s.db.WithContext(ctx).Where("org_id = ?", orgID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.Playbook, 0, len(models))
	for i := range models {
		out = append(out, pbFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) UpdatePlaybook(ctx context.Context, pb *store.Playbook) error {
	pb.UpdatedAt = time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&PlaybookModel{}).Where("id = ?", pb.ID).
		Updates(map[string]any{
			"name":        pb.Name,
			"description": pb.Description,
			"updated_at":  pb.UpdatedAt,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) DeletePlaybook(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("playbook_id = ?", id).Delete(&PlaybookTaskModel{}).Error; err != nil {
			return mapDBError(err)
		}
		res := tx.Where("id = ?", id).Delete(&PlaybookModel{})
		if res.Error != nil {
			return mapDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return store.ErrNotFound
		}
		return nil
	})
}

// --- Playbook Task CRUD ---

func (s *postgresStore) CreatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	now := time.Now().UTC()
	task.CreatedAt = now
	task.UpdatedAt = now

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxOrder *int
		if err := tx.Model(&PlaybookTaskModel{}).
			Where("playbook_id = ?", task.PlaybookID).
			Select("MAX(sort_order)").
			Scan(&maxOrder).Error; err != nil {
			return mapDBError(err)
		}
		if maxOrder != nil {
			task.SortOrder = *maxOrder + 1
		} else {
			task.SortOrder = 0
		}
		if err := tx.Create(taskToModel(task)).Error; err != nil {
			return mapDBError(err)
		}
		return nil
	})
}

func (s *postgresStore) GetPlaybookTask(ctx context.Context, id string) (*store.PlaybookTask, error) {
	var model PlaybookTaskModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return taskFromModel(&model), nil
}

func (s *postgresStore) ListPlaybookTasks(ctx context.Context, playbookID string) ([]*store.PlaybookTask, error) {
	var models []PlaybookTaskModel
	if err := s.db.WithContext(ctx).Where("playbook_id = ?", playbookID).Order("sort_order ASC").Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.PlaybookTask, 0, len(models))
	for i := range models {
		out = append(out, taskFromModel(&models[i]))
	}
	return out, nil
}

func (s *postgresStore) UpdatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	task.UpdatedAt = time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&PlaybookTaskModel{}).Where("id = ?", task.ID).
		Updates(map[string]any{
			"name":       task.Name,
			"module":     task.Module,
			"params":     task.Params,
			"sort_order": task.SortOrder,
			"updated_at": task.UpdatedAt,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) DeletePlaybookTask(ctx context.Context, id string) error {
	res := s.db.WithContext(ctx).Where("id = ?", id).Delete(&PlaybookTaskModel{})
	if res.Error != nil {
		return mapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresStore) ReorderPlaybookTasks(ctx context.Context, playbookID string, taskIDs []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range taskIDs {
			res := tx.Model(&PlaybookTaskModel{}).Where("id = ? AND playbook_id = ?", id, playbookID).
				Update("sort_order", i)
			if res.Error != nil {
				return mapDBError(res.Error)
			}
		}
		return nil
	})
}
*/

// --- Billing helpers ---

func (s *postgresStore) GetOrganizationByStripeCustomerID(ctx context.Context, customerID string) (*store.Organization, error) {
	var model OrganizationModel
	if err := s.db.WithContext(ctx).Where("stripe_customer_id = ? AND deleted_at IS NULL", customerID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return orgFromModel(&model), nil
}

// --- Model Meter / Billing ---

func modelMeterToModel(m *store.ModelMeter) *ModelMeterModel {
	return &ModelMeterModel{
		ID:                  m.ID,
		ModelID:             m.ModelID,
		StripeProductID:     m.StripeProductID,
		StripeInputMeterID:  m.StripeInputMeterID,
		StripeOutputMeterID: m.StripeOutputMeterID,
		StripeInputPriceID:  m.StripeInputPriceID,
		StripeOutputPriceID: m.StripeOutputPriceID,
		InputEventName:      m.InputEventName,
		OutputEventName:     m.OutputEventName,
		InputCostPerToken:   m.InputCostPerToken,
		OutputCostPerToken:  m.OutputCostPerToken,
		CreatedAt:           m.CreatedAt,
	}
}

func modelMeterFromModel(m *ModelMeterModel) *store.ModelMeter {
	return &store.ModelMeter{
		ID:                  m.ID,
		ModelID:             m.ModelID,
		StripeProductID:     m.StripeProductID,
		StripeInputMeterID:  m.StripeInputMeterID,
		StripeOutputMeterID: m.StripeOutputMeterID,
		StripeInputPriceID:  m.StripeInputPriceID,
		StripeOutputPriceID: m.StripeOutputPriceID,
		InputEventName:      m.InputEventName,
		OutputEventName:     m.OutputEventName,
		InputCostPerToken:   m.InputCostPerToken,
		OutputCostPerToken:  m.OutputCostPerToken,
		CreatedAt:           m.CreatedAt,
	}
}

func orgModelSubToModel(s *store.OrgModelSubscription) *OrgModelSubscriptionModel {
	return &OrgModelSubscriptionModel{
		ID:                    s.ID,
		OrgID:                 s.OrgID,
		ModelID:               s.ModelID,
		StripeInputSubItemID:  s.StripeInputSubItemID,
		StripeOutputSubItemID: s.StripeOutputSubItemID,
		CreatedAt:             s.CreatedAt,
	}
}

func orgModelSubFromModel(m *OrgModelSubscriptionModel) *store.OrgModelSubscription {
	return &store.OrgModelSubscription{
		ID:                    m.ID,
		OrgID:                 m.OrgID,
		ModelID:               m.ModelID,
		StripeInputSubItemID:  m.StripeInputSubItemID,
		StripeOutputSubItemID: m.StripeOutputSubItemID,
		CreatedAt:             m.CreatedAt,
	}
}

func (s *postgresStore) GetModelMeter(ctx context.Context, modelID string) (*store.ModelMeter, error) {
	var model ModelMeterModel
	if err := s.db.WithContext(ctx).Where("model_id = ?", modelID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return modelMeterFromModel(&model), nil
}

func (s *postgresStore) CreateModelMeter(ctx context.Context, m *store.ModelMeter) error {
	m.CreatedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Create(modelMeterToModel(m)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) GetOrgModelSubscription(ctx context.Context, orgID, modelID string) (*store.OrgModelSubscription, error) {
	var model OrgModelSubscriptionModel
	if err := s.db.WithContext(ctx).Where("org_id = ? AND model_id = ?", orgID, modelID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return orgModelSubFromModel(&model), nil
}

func (s *postgresStore) CreateOrgModelSubscription(ctx context.Context, sub *store.OrgModelSubscription) error {
	sub.CreatedAt = time.Now().UTC()
	if err := s.db.WithContext(ctx).Create(orgModelSubToModel(sub)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *postgresStore) SumTokenUsage(ctx context.Context, orgID string, from, to time.Time) (float64, error) {
	var total float64
	err := s.db.WithContext(ctx).
		Model(&UsageRecordModel{}).
		Where("org_id = ? AND recorded_at >= ? AND recorded_at <= ? AND resource_type = ?", orgID, from, to, "llm_token").
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, mapDBError(err)
	}
	return total, nil
}

func (s *postgresStore) ListActiveSubscriptions(ctx context.Context) ([]*store.Subscription, error) {
	var models []SubscriptionModel
	if err := s.db.WithContext(ctx).Where("status = ?", string(store.SubStatusActive)).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.Subscription, 0, len(models))
	for i := range models {
		out = append(out, subFromModel(&models[i]))
	}
	return out, nil
}

// --- Advisory Locks ---

func (s *postgresStore) AcquireAdvisoryLock(ctx context.Context, key int64) error {
	return s.db.WithContext(ctx).Exec("SELECT pg_advisory_lock(?)", key).Error
}

func (s *postgresStore) ReleaseAdvisoryLock(ctx context.Context, key int64) error {
	return s.db.WithContext(ctx).Exec("SELECT pg_advisory_unlock(?)", key).Error
}
