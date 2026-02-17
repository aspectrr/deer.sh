package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aspectrr/fluid.sh/fluid/internal/store"
)

// Ensure interface compliance.
var (
	_ store.Store     = (*sqliteStore)(nil)
	_ store.DataStore = (*sqliteStore)(nil)
)

type sqliteStore struct {
	db   *gorm.DB
	conf store.Config
}

// New creates a Store backed by SQLite + GORM.
// If cfg.DatabaseURL is empty, uses ~/.config/fluid/state.db
func New(ctx context.Context, cfg store.Config) (store.Store, error) {
	dbPath := cfg.DatabaseURL
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("sqlite: get home dir: %w", err)
		}
		dbPath = filepath.Join(home, ".fluid", "state.db")
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("sqlite: create config dir: %w", err)
	}

	db, err := gorm.Open(
		sqlite.Open(dbPath),
		&gorm.Config{
			NowFunc: func() time.Time { return time.Now().UTC() },
			Logger:  logger.Default.LogMode(logger.Silent),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("sqlite: sql.DB handle: %w", err)
	}

	// SQLite connection pool settings
	sqlDB.SetMaxOpenConns(1) // SQLite doesn't handle concurrent writes well
	sqlDB.SetMaxIdleConns(1)

	s := &sqliteStore{
		db:   db.WithContext(ctx),
		conf: cfg,
	}

	if cfg.AutoMigrate && !cfg.ReadOnly {
		if err := s.autoMigrate(ctx); err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
	}

	if err := s.Ping(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return s, nil
}

// NewWithDB wraps an existing *gorm.DB (useful for tests).
func NewWithDB(db *gorm.DB, cfg store.Config) store.Store {
	return &sqliteStore{db: db, conf: cfg}
}

func (s *sqliteStore) Config() store.Config {
	return s.conf
}

func (s *sqliteStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *sqliteStore) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (s *sqliteStore) WithTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&sqliteStore{db: tx, conf: s.conf})
	})
}

// --- Sandbox ---

func (s *sqliteStore) CreateSandbox(ctx context.Context, sb *store.Sandbox) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: CreateSandbox: %w", store.ErrInvalid)
	}
	if sb == nil || sb.ID == "" || sb.JobID == "" || sb.AgentID == "" || sb.SandboxName == "" ||
		sb.BaseImage == "" || sb.Network == "" || sb.State == "" {
		return fmt.Errorf("sqlite: CreateSandbox: %w", store.ErrInvalid)
	}

	now := time.Now().UTC()
	sb.CreatedAt = now
	sb.UpdatedAt = now

	if err := s.db.WithContext(ctx).Create(sandboxToModel(sb)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *sqliteStore) GetSandbox(ctx context.Context, id string) (*store.Sandbox, error) {
	var model SandboxModel
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return sandboxFromModel(&model), nil
}

func (s *sqliteStore) GetSandboxByVMName(ctx context.Context, vmName string) (*store.Sandbox, error) {
	var model SandboxModel
	if err := s.db.WithContext(ctx).
		Where("sandbox_name = ? AND deleted_at IS NULL", vmName).
		First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return sandboxFromModel(&model), nil
}

func (s *sqliteStore) ListSandboxes(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
	tx := s.db.WithContext(ctx).Model(&SandboxModel{}).Where("deleted_at IS NULL")
	if filter.AgentID != nil {
		tx = tx.Where("agent_id = ?", *filter.AgentID)
	}
	if filter.JobID != nil {
		tx = tx.Where("job_id = ?", *filter.JobID)
	}
	if filter.BaseImage != nil {
		tx = tx.Where("base_image = ?", *filter.BaseImage)
	}
	if filter.State != nil {
		tx = tx.Where("state = ?", string(*filter.State))
	}
	if filter.VMName != nil {
		tx = tx.Where("sandbox_name = ?", *filter.VMName)
	}

	tx = applyListOptions(tx, opt, map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"vm_name":    "sandbox_name",
	})

	var models []SandboxModel
	if err := tx.Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}

	out := make([]*store.Sandbox, 0, len(models))
	for i := range models {
		out = append(out, sandboxFromModel(&models[i]))
	}
	return out, nil
}

func (s *sqliteStore) UpdateSandbox(ctx context.Context, sb *store.Sandbox) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: UpdateSandbox: %w", store.ErrInvalid)
	}
	if sb == nil || sb.ID == "" {
		return fmt.Errorf("sqlite: UpdateSandbox: %w", store.ErrInvalid)
	}
	sb.UpdatedAt = time.Now().UTC()
	model := sandboxToModel(sb)

	res := s.db.WithContext(ctx).
		Model(&SandboxModel{}).
		Where("id = ? AND deleted_at IS NULL", sb.ID).
		Updates(map[string]any{
			"job_id":       model.JobID,
			"agent_id":     model.AgentID,
			"sandbox_name": model.SandboxName,
			"base_image":   model.BaseImage,
			"network":      model.Network,
			"ip":           model.IPAddress,
			"state":        model.State,
			"ttl_seconds":  model.TTLSeconds,
			"vcpus":        model.VCPUs,
			"memory_mb":    model.MemoryMB,
			"storage_mb":   model.StorageMB,
			"updated_at":   model.UpdatedAt,
		})

	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *sqliteStore) UpdateSandboxState(ctx context.Context, id string, newState store.SandboxState, ipAddr *string) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: UpdateSandboxState: %w", store.ErrInvalid)
	}
	if id == "" {
		return fmt.Errorf("sqlite: UpdateSandboxState: %w", store.ErrInvalid)
	}

	res := s.db.WithContext(ctx).Model(&SandboxModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"state":      string(newState),
			"ip":         copyString(ipAddr),
			"updated_at": time.Now().UTC(),
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *sqliteStore) DeleteSandbox(ctx context.Context, id string) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: DeleteSandbox: %w", store.ErrInvalid)
	}
	if id == "" {
		return fmt.Errorf("sqlite: DeleteSandbox: %w", store.ErrInvalid)
	}
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&SandboxModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"state":      string(store.SandboxStateDestroyed),
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

// --- Snapshot ---

func (s *sqliteStore) CreateSnapshot(ctx context.Context, sn *store.Snapshot) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: CreateSnapshot: %w", store.ErrInvalid)
	}
	if sn == nil || sn.ID == "" || sn.SandboxID == "" || sn.Name == "" || sn.Ref == "" || sn.Kind == "" {
		return fmt.Errorf("sqlite: CreateSnapshot: %w", store.ErrInvalid)
	}
	if sn.CreatedAt.IsZero() {
		sn.CreatedAt = time.Now().UTC()
	}
	if err := s.db.WithContext(ctx).Create(snapshotToModel(sn)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *sqliteStore) GetSnapshot(ctx context.Context, id string) (*store.Snapshot, error) {
	var model SnapshotModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return snapshotFromModel(&model), nil
}

func (s *sqliteStore) GetSnapshotByName(ctx context.Context, sandboxID, name string) (*store.Snapshot, error) {
	var model SnapshotModel
	if err := s.db.WithContext(ctx).
		Where("sandbox_id = ? AND name = ?", sandboxID, name).
		First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return snapshotFromModel(&model), nil
}

func (s *sqliteStore) ListSnapshots(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Snapshot, error) {
	tx := s.db.WithContext(ctx).Model(&SnapshotModel{}).Where("sandbox_id = ?", sandboxID)
	tx = applyListOptions(tx, opt, map[string]string{
		"created_at": "created_at",
		"name":       "name",
	})

	var models []SnapshotModel
	if err := tx.Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.Snapshot, 0, len(models))
	for i := range models {
		out = append(out, snapshotFromModel(&models[i]))
	}
	return out, nil
}

// --- Command ---

func (s *sqliteStore) SaveCommand(ctx context.Context, cmd *store.Command) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: SaveCommand: %w", store.ErrInvalid)
	}
	if cmd == nil || cmd.ID == "" || cmd.SandboxID == "" || cmd.Command == "" {
		return fmt.Errorf("sqlite: SaveCommand: %w", store.ErrInvalid)
	}
	if cmd.StartedAt.IsZero() {
		cmd.StartedAt = time.Now().UTC()
	}
	if cmd.EndedAt.IsZero() {
		cmd.EndedAt = time.Now().UTC()
	}

	if err := s.db.WithContext(ctx).Create(commandToModel(cmd)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *sqliteStore) GetCommand(ctx context.Context, id string) (*store.Command, error) {
	var model CommandModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return commandFromModel(&model), nil
}

func (s *sqliteStore) ListCommands(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Command, error) {
	tx := s.db.WithContext(ctx).Model(&CommandModel{}).Where("sandbox_id = ?", sandboxID)

	// Command model uses started_at instead of created_at
	if opt == nil || opt.OrderBy == "" {
		tx = tx.Order("started_at DESC")
	} else {
		tx = applyListOptions(tx, opt, map[string]string{
			"started_at": "started_at",
			"ended_at":   "ended_at",
		})
	}

	if opt != nil && opt.Limit > 0 {
		tx = tx.Limit(opt.Limit)
		if opt.Offset > 0 {
			tx = tx.Offset(opt.Offset)
		}
	}

	var models []CommandModel
	if err := tx.Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.Command, 0, len(models))
	for i := range models {
		out = append(out, commandFromModel(&models[i]))
	}
	return out, nil
}

// --- Diff ---

func (s *sqliteStore) SaveDiff(ctx context.Context, d *store.Diff) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: SaveDiff: %w", store.ErrInvalid)
	}
	if d == nil || d.ID == "" || d.SandboxID == "" || d.FromSnapshot == "" || d.ToSnapshot == "" {
		return fmt.Errorf("sqlite: SaveDiff: %w", store.ErrInvalid)
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	model, err := diffToModel(d)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(model).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *sqliteStore) GetDiff(ctx context.Context, id string) (*store.Diff, error) {
	var model DiffModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return diffFromModel(&model)
}

func (s *sqliteStore) GetDiffBySnapshots(ctx context.Context, sandboxID, fromSnapshot, toSnapshot string) (*store.Diff, error) {
	var model DiffModel
	if err := s.db.WithContext(ctx).
		Where("sandbox_id = ? AND from_snapshot = ? AND to_snapshot = ?", sandboxID, fromSnapshot, toSnapshot).
		First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return diffFromModel(&model)
}

// --- ChangeSet ---

func (s *sqliteStore) CreateChangeSet(ctx context.Context, cs *store.ChangeSet) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: CreateChangeSet: %w", store.ErrInvalid)
	}
	if cs == nil || cs.ID == "" || cs.JobID == "" || cs.SandboxID == "" || cs.DiffID == "" ||
		cs.PathAnsible == "" || cs.PathPuppet == "" {
		return fmt.Errorf("sqlite: CreateChangeSet: %w", store.ErrInvalid)
	}
	if cs.CreatedAt.IsZero() {
		cs.CreatedAt = time.Now().UTC()
	}
	if err := s.db.WithContext(ctx).Create(changeSetToModel(cs)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *sqliteStore) GetChangeSet(ctx context.Context, id string) (*store.ChangeSet, error) {
	var model ChangeSetModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return changeSetFromModel(&model), nil
}

func (s *sqliteStore) GetChangeSetByJob(ctx context.Context, jobID string) (*store.ChangeSet, error) {
	var model ChangeSetModel
	if err := s.db.WithContext(ctx).Where("job_id = ?", jobID).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return changeSetFromModel(&model), nil
}

// --- Publication ---

func (s *sqliteStore) CreatePublication(ctx context.Context, p *store.Publication) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: CreatePublication: %w", store.ErrInvalid)
	}
	if p == nil || p.ID == "" || p.JobID == "" || p.RepoURL == "" || p.Branch == "" || p.Status == "" {
		return fmt.Errorf("sqlite: CreatePublication: %w", store.ErrInvalid)
	}
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	if err := s.db.WithContext(ctx).Create(publicationToModel(p)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *sqliteStore) UpdatePublicationStatus(ctx context.Context, id string, status store.PublicationStatus, commitSHA, prURL, errMsg *string) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: UpdatePublicationStatus: %w", store.ErrInvalid)
	}
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&PublicationModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     string(status),
			"commit_sha": copyString(commitSHA),
			"pr_url":     copyString(prURL),
			"error_msg":  copyString(errMsg),
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

func (s *sqliteStore) GetPublication(ctx context.Context, id string) (*store.Publication, error) {
	var model PublicationModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return publicationFromModel(&model), nil
}

// --- Playbook ---

func (s *sqliteStore) CreatePlaybook(ctx context.Context, pb *store.Playbook) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: CreatePlaybook: %w", store.ErrInvalid)
	}
	if pb == nil || pb.ID == "" || pb.Name == "" || pb.Hosts == "" {
		return fmt.Errorf("sqlite: CreatePlaybook: %w", store.ErrInvalid)
	}
	now := time.Now().UTC()
	pb.CreatedAt = now
	pb.UpdatedAt = now

	if err := s.db.WithContext(ctx).Create(playbookToModel(pb)).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *sqliteStore) GetPlaybook(ctx context.Context, id string) (*store.Playbook, error) {
	var model PlaybookModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return playbookFromModel(&model), nil
}

func (s *sqliteStore) GetPlaybookByName(ctx context.Context, name string) (*store.Playbook, error) {
	var model PlaybookModel
	if err := s.db.WithContext(ctx).Where("name = ?", name).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return playbookFromModel(&model), nil
}

func (s *sqliteStore) ListPlaybooks(ctx context.Context, opt *store.ListOptions) ([]*store.Playbook, error) {
	tx := s.db.WithContext(ctx).Model(&PlaybookModel{})
	tx = applyListOptions(tx, opt, map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"name":       "name",
	})

	var models []PlaybookModel
	if err := tx.Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.Playbook, 0, len(models))
	for i := range models {
		out = append(out, playbookFromModel(&models[i]))
	}
	return out, nil
}

func (s *sqliteStore) UpdatePlaybook(ctx context.Context, pb *store.Playbook) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: UpdatePlaybook: %w", store.ErrInvalid)
	}
	if pb == nil || pb.ID == "" {
		return fmt.Errorf("sqlite: UpdatePlaybook: %w", store.ErrInvalid)
	}
	pb.UpdatedAt = time.Now().UTC()
	model := playbookToModel(pb)

	res := s.db.WithContext(ctx).
		Model(&PlaybookModel{}).
		Where("id = ?", pb.ID).
		Updates(map[string]any{
			"name":       model.Name,
			"hosts":      model.Hosts,
			"become":     model.Become,
			"file_path":  model.FilePath,
			"updated_at": model.UpdatedAt,
		})

	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *sqliteStore) DeletePlaybook(ctx context.Context, id string) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: DeletePlaybook: %w", store.ErrInvalid)
	}
	if id == "" {
		return fmt.Errorf("sqlite: DeletePlaybook: %w", store.ErrInvalid)
	}

	// Delete associated tasks first
	if err := s.db.WithContext(ctx).Where("playbook_id = ?", id).Delete(&PlaybookTaskModel{}).Error; err != nil {
		return mapDBError(err)
	}

	res := s.db.WithContext(ctx).Where("id = ?", id).Delete(&PlaybookModel{})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

// --- PlaybookTask ---

func (s *sqliteStore) CreatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: CreatePlaybookTask: %w", store.ErrInvalid)
	}
	if task == nil || task.ID == "" || task.PlaybookID == "" || task.Name == "" || task.Module == "" {
		return fmt.Errorf("sqlite: CreatePlaybookTask: %w", store.ErrInvalid)
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	}

	model, err := playbookTaskToModel(task)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(model).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (s *sqliteStore) GetPlaybookTask(ctx context.Context, id string) (*store.PlaybookTask, error) {
	var model PlaybookTaskModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return playbookTaskFromModel(&model)
}

func (s *sqliteStore) ListPlaybookTasks(ctx context.Context, playbookID string, opt *store.ListOptions) ([]*store.PlaybookTask, error) {
	tx := s.db.WithContext(ctx).Model(&PlaybookTaskModel{}).Where("playbook_id = ?", playbookID)

	// Default ordering by position
	if opt == nil || opt.OrderBy == "" {
		tx = tx.Order("position ASC")
	} else {
		tx = applyListOptions(tx, opt, map[string]string{
			"position":   "position",
			"created_at": "created_at",
			"name":       "name",
		})
	}

	if opt != nil && opt.Limit > 0 {
		tx = tx.Limit(opt.Limit)
		if opt.Offset > 0 {
			tx = tx.Offset(opt.Offset)
		}
	}

	var models []PlaybookTaskModel
	if err := tx.Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.PlaybookTask, 0, len(models))
	for i := range models {
		task, err := playbookTaskFromModel(&models[i])
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, nil
}

func (s *sqliteStore) UpdatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: UpdatePlaybookTask: %w", store.ErrInvalid)
	}
	if task == nil || task.ID == "" {
		return fmt.Errorf("sqlite: UpdatePlaybookTask: %w", store.ErrInvalid)
	}

	model, err := playbookTaskToModel(task)
	if err != nil {
		return err
	}

	res := s.db.WithContext(ctx).
		Model(&PlaybookTaskModel{}).
		Where("id = ?", task.ID).
		Updates(map[string]any{
			"name":     model.Name,
			"module":   model.Module,
			"params":   model.Params,
			"position": model.Position,
		})

	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *sqliteStore) DeletePlaybookTask(ctx context.Context, id string) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: DeletePlaybookTask: %w", store.ErrInvalid)
	}
	if id == "" {
		return fmt.Errorf("sqlite: DeletePlaybookTask: %w", store.ErrInvalid)
	}

	res := s.db.WithContext(ctx).Where("id = ?", id).Delete(&PlaybookTaskModel{})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *sqliteStore) ReorderPlaybookTasks(ctx context.Context, playbookID string, taskIDs []string) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: ReorderPlaybookTasks: %w", store.ErrInvalid)
	}
	if playbookID == "" || len(taskIDs) == 0 {
		return fmt.Errorf("sqlite: ReorderPlaybookTasks: %w", store.ErrInvalid)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, taskID := range taskIDs {
			res := tx.Model(&PlaybookTaskModel{}).
				Where("id = ? AND playbook_id = ?", taskID, playbookID).
				Update("position", i)
			if res.Error != nil {
				return mapDBError(res.Error)
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("task %s not found in playbook %s", taskID, playbookID)
			}
		}
		return nil
	})
}

func (s *sqliteStore) GetNextTaskPosition(ctx context.Context, playbookID string) (int, error) {
	var maxPos *int
	err := s.db.WithContext(ctx).
		Model(&PlaybookTaskModel{}).
		Where("playbook_id = ?", playbookID).
		Select("MAX(position)").
		Scan(&maxPos).Error
	if err != nil {
		return 0, mapDBError(err)
	}
	if maxPos == nil {
		return 0, nil
	}
	return *maxPos + 1, nil
}

// --- SourceVM ---

func (s *sqliteStore) GetSourceVM(ctx context.Context, name string) (*store.SourceVM, error) {
	var model SourceVMModel
	if err := s.db.WithContext(ctx).Where("name = ?", name).First(&model).Error; err != nil {
		return nil, mapDBError(err)
	}
	return sourceVMFromModel(&model), nil
}

func (s *sqliteStore) UpsertSourceVM(ctx context.Context, svm *store.SourceVM) error {
	if s.conf.ReadOnly {
		return fmt.Errorf("sqlite: UpsertSourceVM: %w", store.ErrInvalid)
	}
	if svm == nil || svm.Name == "" {
		return fmt.Errorf("sqlite: UpsertSourceVM: %w", store.ErrInvalid)
	}

	now := time.Now().UTC()
	svm.UpdatedAt = now

	// Check if exists
	var existing SourceVMModel
	err := s.db.WithContext(ctx).Where("name = ?", svm.Name).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return mapDBError(err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Insert
		if svm.ID == "" {
			svm.ID = fmt.Sprintf("SVM-%d", now.UnixNano())
		}
		svm.CreatedAt = now
		if err := s.db.WithContext(ctx).Create(sourceVMToModel(svm)).Error; err != nil {
			return mapDBError(err)
		}
		return nil
	}

	// Update
	svm.ID = existing.ID
	svm.CreatedAt = existing.CreatedAt
	model := sourceVMToModel(svm)
	res := s.db.WithContext(ctx).
		Model(&SourceVMModel{}).
		Where("id = ?", existing.ID).
		Updates(map[string]any{
			"host_name":      model.HostName,
			"host_address":   model.HostAddress,
			"prepared":       model.Prepared,
			"prepared_at":    model.PreparedAt,
			"prepare_json":   model.PrepareJSON,
			"ca_fingerprint": model.CAFingerprint,
			"updated_at":     model.UpdatedAt,
		})
	if err := mapDBError(res.Error); err != nil {
		return err
	}
	return nil
}

func (s *sqliteStore) ListSourceVMs(ctx context.Context) ([]*store.SourceVM, error) {
	var models []SourceVMModel
	if err := s.db.WithContext(ctx).Order("name ASC").Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	out := make([]*store.SourceVM, 0, len(models))
	for i := range models {
		out = append(out, sourceVMFromModel(&models[i]))
	}
	return out, nil
}

// --- Migration ---

func (s *sqliteStore) autoMigrate(ctx context.Context) error {
	if err := s.db.WithContext(ctx).AutoMigrate(
		&SandboxModel{},
		&SnapshotModel{},
		&CommandModel{},
		&DiffModel{},
		&ChangeSetModel{},
		&PublicationModel{},
		&PlaybookModel{},
		&PlaybookTaskModel{},
		&HostResourcesModel{},
		&SourceVMModel{},
	); err != nil {
		return err
	}

	// Create unique index on sandbox_name for non-deleted rows
	if err := s.db.WithContext(ctx).Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_sandbox_name_active
		ON sandboxes (sandbox_name)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return fmt.Errorf("create partial unique index: %w", err)
	}

	return nil
}

// --- Models ---

type SandboxModel struct {
	ID          string     `gorm:"primaryKey;column:id"`
	JobID       string     `gorm:"column:job_id;not null;index"`
	AgentID     string     `gorm:"column:agent_id;not null;index"`
	SandboxName string     `gorm:"column:sandbox_name;not null"`
	BaseImage   string     `gorm:"column:base_image;not null;index"`
	Network     string     `gorm:"column:network;not null"`
	IPAddress   *string    `gorm:"column:ip"`
	State       string     `gorm:"column:state;not null;index"`
	TTLSeconds  *int       `gorm:"column:ttl_seconds"`
	VCPUs       int        `gorm:"column:vcpus;not null;default:2"`
	MemoryMB    int        `gorm:"column:memory_mb;not null;default:4096"`
	StorageMB   int64      `gorm:"column:storage_mb;not null;default:0"`
	HostName    *string    `gorm:"column:host_name"`
	HostAddress *string    `gorm:"column:host_address"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null"`
	DeletedAt   *time.Time `gorm:"column:deleted_at;index"`
}

func (SandboxModel) TableName() string { return "sandboxes" }

type SnapshotModel struct {
	ID        string    `gorm:"primaryKey;column:id"`
	SandboxID string    `gorm:"column:sandbox_id;not null;index;uniqueIndex:idx_snapshots_sandbox_name"`
	Name      string    `gorm:"column:name;not null;uniqueIndex:idx_snapshots_sandbox_name"`
	Kind      string    `gorm:"column:kind;not null"`
	Ref       string    `gorm:"column:ref;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	MetaJSON  *string   `gorm:"column:meta_json;type:text"`
}

func (SnapshotModel) TableName() string { return "snapshots" }

type CommandModel struct {
	ID        string    `gorm:"primaryKey;column:id"`
	SandboxID string    `gorm:"column:sandbox_id;not null;index"`
	Command   string    `gorm:"column:command;not null"`
	EnvJSON   *string   `gorm:"column:env_json;type:text"`
	Stdout    string    `gorm:"column:stdout;not null"`
	Stderr    string    `gorm:"column:stderr;not null"`
	ExitCode  int       `gorm:"column:exit_code;not null"`
	StartedAt time.Time `gorm:"column:started_at;not null;index"`
	EndedAt   time.Time `gorm:"column:ended_at;not null"`
}

func (CommandModel) TableName() string { return "commands" }

type DiffModel struct {
	ID           string    `gorm:"primaryKey;column:id"`
	SandboxID    string    `gorm:"column:sandbox_id;not null;index;uniqueIndex:idx_diffs_sandbox_snapshots"`
	FromSnapshot string    `gorm:"column:from_snapshot;not null;uniqueIndex:idx_diffs_sandbox_snapshots"`
	ToSnapshot   string    `gorm:"column:to_snapshot;not null;uniqueIndex:idx_diffs_sandbox_snapshots"`
	DiffJSON     string    `gorm:"column:diff_json;type:text;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
}

func (DiffModel) TableName() string { return "diffs" }

type ChangeSetModel struct {
	ID          string    `gorm:"primaryKey;column:id"`
	JobID       string    `gorm:"column:job_id;not null;uniqueIndex"`
	SandboxID   string    `gorm:"column:sandbox_id;not null;index"`
	DiffID      string    `gorm:"column:diff_id;not null;index"`
	PathAnsible string    `gorm:"column:path_ansible;not null"`
	PathPuppet  string    `gorm:"column:path_puppet;not null"`
	MetaJSON    *string   `gorm:"column:meta_json;type:text"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
}

func (ChangeSetModel) TableName() string { return "changesets" }

type PublicationModel struct {
	ID        string    `gorm:"primaryKey;column:id"`
	JobID     string    `gorm:"column:job_id;not null;index"`
	RepoURL   string    `gorm:"column:repo_url;not null"`
	Branch    string    `gorm:"column:branch;not null"`
	CommitSHA *string   `gorm:"column:commit_sha"`
	PRURL     *string   `gorm:"column:pr_url"`
	Status    string    `gorm:"column:status;not null;index"`
	ErrorMsg  *string   `gorm:"column:error_msg"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (PublicationModel) TableName() string { return "publications" }

type PlaybookModel struct {
	ID        string    `gorm:"primaryKey;column:id"`
	Name      string    `gorm:"column:name;not null;uniqueIndex"`
	Hosts     string    `gorm:"column:hosts;not null"`
	Become    bool      `gorm:"column:become;not null;default:false"`
	FilePath  *string   `gorm:"column:file_path"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (PlaybookModel) TableName() string { return "playbooks" }

type PlaybookTaskModel struct {
	ID         string    `gorm:"primaryKey;column:id"`
	PlaybookID string    `gorm:"column:playbook_id;not null;index"`
	Position   int       `gorm:"column:position;not null;index"`
	Name       string    `gorm:"column:name;not null"`
	Module     string    `gorm:"column:module;not null"`
	Params     string    `gorm:"column:params;type:text;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;not null"`
}

func (PlaybookTaskModel) TableName() string { return "playbook_tasks" }

type HostResourcesModel struct {
	ID                string    `gorm:"primaryKey;column:id"`
	Name              string    `gorm:"column:name;not null;uniqueIndex"`
	Address           string    `gorm:"column:address;not null"`
	TotalCPUs         int       `gorm:"column:total_cpus;not null"`
	TotalMemoryMB     int64     `gorm:"column:total_memory_mb;not null"`
	TotalStorageMB    int64     `gorm:"column:total_storage_mb;not null"`
	ReservedCPUs      int       `gorm:"column:reserved_cpus;not null;default:0"`
	ReservedMemoryMB  int64     `gorm:"column:reserved_memory_mb;not null;default:0"`
	ReservedStorageMB int64     `gorm:"column:reserved_storage_mb;not null;default:0"`
	UpdatedAt         time.Time `gorm:"column:updated_at;not null"`
}

func (HostResourcesModel) TableName() string { return "host_resources" }

type SourceVMModel struct {
	ID            string     `gorm:"primaryKey;column:id"`
	Name          string     `gorm:"column:name;not null;uniqueIndex"`
	HostName      *string    `gorm:"column:host_name"`
	HostAddress   *string    `gorm:"column:host_address"`
	Prepared      bool       `gorm:"column:prepared;not null;default:false"`
	PreparedAt    *time.Time `gorm:"column:prepared_at"`
	PrepareJSON   *string    `gorm:"column:prepare_json;type:text"`
	CAFingerprint *string    `gorm:"column:ca_fingerprint"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;not null"`
}

func (SourceVMModel) TableName() string { return "source_vms" }

// --- Converters ---

func sandboxToModel(sb *store.Sandbox) *SandboxModel {
	return &SandboxModel{
		ID:          sb.ID,
		JobID:       sb.JobID,
		AgentID:     sb.AgentID,
		SandboxName: sb.SandboxName,
		BaseImage:   sb.BaseImage,
		Network:     sb.Network,
		IPAddress:   copyString(sb.IPAddress),
		State:       string(sb.State),
		TTLSeconds:  copyInt(sb.TTLSeconds),
		VCPUs:       sb.VCPUs,
		MemoryMB:    sb.MemoryMB,
		StorageMB:   sb.StorageMB,
		HostName:    copyString(sb.HostName),
		HostAddress: copyString(sb.HostAddress),
		CreatedAt:   sb.CreatedAt,
		UpdatedAt:   sb.UpdatedAt,
		DeletedAt:   copyTime(sb.DeletedAt),
	}
}

func sandboxFromModel(m *SandboxModel) *store.Sandbox {
	return &store.Sandbox{
		ID:          m.ID,
		JobID:       m.JobID,
		AgentID:     m.AgentID,
		SandboxName: m.SandboxName,
		BaseImage:   m.BaseImage,
		Network:     m.Network,
		IPAddress:   copyString(m.IPAddress),
		State:       store.SandboxState(m.State),
		TTLSeconds:  copyInt(m.TTLSeconds),
		VCPUs:       m.VCPUs,
		MemoryMB:    m.MemoryMB,
		StorageMB:   m.StorageMB,
		HostName:    copyString(m.HostName),
		HostAddress: copyString(m.HostAddress),
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   copyTime(m.DeletedAt),
	}
}

func snapshotToModel(sn *store.Snapshot) *SnapshotModel {
	return &SnapshotModel{
		ID:        sn.ID,
		SandboxID: sn.SandboxID,
		Name:      sn.Name,
		Kind:      string(sn.Kind),
		Ref:       sn.Ref,
		CreatedAt: sn.CreatedAt,
		MetaJSON:  copyString(sn.MetaJSON),
	}
}

func snapshotFromModel(m *SnapshotModel) *store.Snapshot {
	return &store.Snapshot{
		ID:        m.ID,
		SandboxID: m.SandboxID,
		Name:      m.Name,
		Kind:      store.SnapshotKind(m.Kind),
		Ref:       m.Ref,
		CreatedAt: m.CreatedAt,
		MetaJSON:  copyString(m.MetaJSON),
	}
}

func commandToModel(cmd *store.Command) *CommandModel {
	return &CommandModel{
		ID:        cmd.ID,
		SandboxID: cmd.SandboxID,
		Command:   cmd.Command,
		EnvJSON:   copyString(cmd.EnvJSON),
		Stdout:    cmd.Stdout,
		Stderr:    cmd.Stderr,
		ExitCode:  cmd.ExitCode,
		StartedAt: cmd.StartedAt,
		EndedAt:   cmd.EndedAt,
	}
}

func commandFromModel(m *CommandModel) *store.Command {
	return &store.Command{
		ID:        m.ID,
		SandboxID: m.SandboxID,
		Command:   m.Command,
		EnvJSON:   copyString(m.EnvJSON),
		Stdout:    m.Stdout,
		Stderr:    m.Stderr,
		ExitCode:  m.ExitCode,
		StartedAt: m.StartedAt,
		EndedAt:   m.EndedAt,
	}
}

func diffToModel(d *store.Diff) (*DiffModel, error) {
	payload, err := json.Marshal(d.DiffJSON)
	if err != nil {
		return nil, fmt.Errorf("sqlite: marshal diff_json: %w", err)
	}
	return &DiffModel{
		ID:           d.ID,
		SandboxID:    d.SandboxID,
		FromSnapshot: d.FromSnapshot,
		ToSnapshot:   d.ToSnapshot,
		DiffJSON:     string(payload),
		CreatedAt:    d.CreatedAt,
	}, nil
}

func diffFromModel(m *DiffModel) (*store.Diff, error) {
	var diff store.Diff
	diff.ID = m.ID
	diff.SandboxID = m.SandboxID
	diff.FromSnapshot = m.FromSnapshot
	diff.ToSnapshot = m.ToSnapshot
	diff.CreatedAt = m.CreatedAt
	if err := json.Unmarshal([]byte(m.DiffJSON), &diff.DiffJSON); err != nil {
		return nil, fmt.Errorf("sqlite: unmarshal diff_json: %w", err)
	}
	return &diff, nil
}

func changeSetToModel(cs *store.ChangeSet) *ChangeSetModel {
	return &ChangeSetModel{
		ID:          cs.ID,
		JobID:       cs.JobID,
		SandboxID:   cs.SandboxID,
		DiffID:      cs.DiffID,
		PathAnsible: cs.PathAnsible,
		PathPuppet:  cs.PathPuppet,
		MetaJSON:    copyString(cs.MetaJSON),
		CreatedAt:   cs.CreatedAt,
	}
}

func changeSetFromModel(m *ChangeSetModel) *store.ChangeSet {
	return &store.ChangeSet{
		ID:          m.ID,
		JobID:       m.JobID,
		SandboxID:   m.SandboxID,
		DiffID:      m.DiffID,
		PathAnsible: m.PathAnsible,
		PathPuppet:  m.PathPuppet,
		MetaJSON:    copyString(m.MetaJSON),
		CreatedAt:   m.CreatedAt,
	}
}

func publicationToModel(p *store.Publication) *PublicationModel {
	return &PublicationModel{
		ID:        p.ID,
		JobID:     p.JobID,
		RepoURL:   p.RepoURL,
		Branch:    p.Branch,
		CommitSHA: copyString(p.CommitSHA),
		PRURL:     copyString(p.PRURL),
		Status:    string(p.Status),
		ErrorMsg:  copyString(p.ErrorMsg),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func publicationFromModel(m *PublicationModel) *store.Publication {
	return &store.Publication{
		ID:        m.ID,
		JobID:     m.JobID,
		RepoURL:   m.RepoURL,
		Branch:    m.Branch,
		CommitSHA: copyString(m.CommitSHA),
		PRURL:     copyString(m.PRURL),
		Status:    store.PublicationStatus(m.Status),
		ErrorMsg:  copyString(m.ErrorMsg),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func playbookToModel(pb *store.Playbook) *PlaybookModel {
	return &PlaybookModel{
		ID:        pb.ID,
		Name:      pb.Name,
		Hosts:     pb.Hosts,
		Become:    pb.Become,
		FilePath:  copyString(pb.FilePath),
		CreatedAt: pb.CreatedAt,
		UpdatedAt: pb.UpdatedAt,
	}
}

func playbookFromModel(m *PlaybookModel) *store.Playbook {
	return &store.Playbook{
		ID:        m.ID,
		Name:      m.Name,
		Hosts:     m.Hosts,
		Become:    m.Become,
		FilePath:  copyString(m.FilePath),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func playbookTaskToModel(task *store.PlaybookTask) (*PlaybookTaskModel, error) {
	params, err := json.Marshal(task.Params)
	if err != nil {
		return nil, fmt.Errorf("sqlite: marshal task params: %w", err)
	}
	return &PlaybookTaskModel{
		ID:         task.ID,
		PlaybookID: task.PlaybookID,
		Position:   task.Position,
		Name:       task.Name,
		Module:     task.Module,
		Params:     string(params),
		CreatedAt:  task.CreatedAt,
	}, nil
}

func playbookTaskFromModel(m *PlaybookTaskModel) (*store.PlaybookTask, error) {
	var params map[string]any
	if len(m.Params) > 0 {
		if err := json.Unmarshal([]byte(m.Params), &params); err != nil {
			return nil, fmt.Errorf("sqlite: unmarshal task params: %w", err)
		}
	}
	return &store.PlaybookTask{
		ID:         m.ID,
		PlaybookID: m.PlaybookID,
		Position:   m.Position,
		Name:       m.Name,
		Module:     m.Module,
		Params:     params,
		CreatedAt:  m.CreatedAt,
	}, nil
}

func sourceVMToModel(svm *store.SourceVM) *SourceVMModel {
	return &SourceVMModel{
		ID:            svm.ID,
		Name:          svm.Name,
		HostName:      copyString(svm.HostName),
		HostAddress:   copyString(svm.HostAddress),
		Prepared:      svm.Prepared,
		PreparedAt:    copyTime(svm.PreparedAt),
		PrepareJSON:   copyString(svm.PrepareJSON),
		CAFingerprint: copyString(svm.CAFingerprint),
		CreatedAt:     svm.CreatedAt,
		UpdatedAt:     svm.UpdatedAt,
	}
}

func sourceVMFromModel(m *SourceVMModel) *store.SourceVM {
	return &store.SourceVM{
		ID:            m.ID,
		Name:          m.Name,
		HostName:      copyString(m.HostName),
		HostAddress:   copyString(m.HostAddress),
		Prepared:      m.Prepared,
		PreparedAt:    copyTime(m.PreparedAt),
		PrepareJSON:   copyString(m.PrepareJSON),
		CAFingerprint: copyString(m.CAFingerprint),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

// --- Helpers ---

func applyListOptions(tx *gorm.DB, opt *store.ListOptions, whitelist map[string]string) *gorm.DB {
	orderApplied := false
	if opt != nil {
		if col, ok := whitelist[opt.OrderBy]; ok {
			dir := "DESC"
			if opt.Asc {
				dir = "ASC"
			}
			tx = tx.Order(fmt.Sprintf("%s %s", col, dir))
			orderApplied = true
		}
		if opt.Limit > 0 {
			tx = tx.Limit(opt.Limit)
			if opt.Offset > 0 {
				tx = tx.Offset(opt.Offset)
			}
		}
	}
	if !orderApplied {
		tx = tx.Order("created_at DESC")
	}
	return tx
}

func copyString(src *string) *string {
	if src == nil {
		return nil
	}
	val := *src
	return &val
}

func copyInt(src *int) *int {
	if src == nil {
		return nil
	}
	val := *src
	return &val
}

func copyTime(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	val := *src
	return &val
}

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
	// SQLite constraint violations
	errStr := err.Error()
	if contains(errStr, "UNIQUE constraint failed") {
		return store.ErrAlreadyExists
	}
	if contains(errStr, "FOREIGN KEY constraint failed") {
		return store.ErrInvalid
	}
	return err
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
