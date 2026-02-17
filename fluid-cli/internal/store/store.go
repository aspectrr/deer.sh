package store

import (
	"context"
	"errors"
	"time"
)

// Domain model and persistence contracts for the VM sandbox system.
// This package declares the data structures persisted in the DB and the
// storage interfaces that concrete implementations (SQLite/Postgres) must provide.

// Config describes database-related configuration for a Store implementation.
type Config struct {
	// DatabaseURL is the DSN/URL used to connect to the database.
	// Examples:
	// - Postgres: postgres://user:pass@host:5432/dbname?sslmode=disable
	DatabaseURL string `json:"database_url"`

	// MaxOpenConns sets the maximum number of open connections to the database.
	MaxOpenConns int `json:"max_open_conns"`

	// MaxIdleConns sets the maximum number of connections in the idle connection pool.
	MaxIdleConns int `json:"max_idle_conns"`

	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`

	// AutoMigrate, when true, allows the store to create/update schema automatically.
	AutoMigrate bool `json:"auto_migrate"`

	// ReadOnly, when true, disallows mutating operations.
	ReadOnly bool `json:"read_only"`
}

// ListOptions supports pagination and ordering for list operations.
type ListOptions struct {
	Limit   int    // Max records to return (0 = default/backend-defined)
	Offset  int    // Records to skip
	OrderBy string // Column to order by (implementation should whitelist)
	Asc     bool   // Ascending if true, descending if false
}

// Common sentinel errors for store implementations.
var (
	ErrNotFound      = errors.New("store: not found")
	ErrAlreadyExists = errors.New("store: already exists")
	ErrConflict      = errors.New("store: conflict")
	ErrInvalid       = errors.New("store: invalid data")
)

// SandboxState enumerates lifecycle states for a sandbox VM.
type SandboxState string

const (
	SandboxStateCreated   SandboxState = "CREATED"
	SandboxStateStarting  SandboxState = "STARTING"
	SandboxStateRunning   SandboxState = "RUNNING"
	SandboxStateStopped   SandboxState = "STOPPED"
	SandboxStateDestroyed SandboxState = "DESTROYED"
	SandboxStateError     SandboxState = "ERROR"
)

// SnapshotKind describes how a snapshot is taken/stored.
type SnapshotKind string

const (
	// SnapshotKindInternal refers to libvirt/qemu internal snapshot (domain-managed).
	SnapshotKindInternal SnapshotKind = "INTERNAL"
	// SnapshotKindExternal refers to external snapshot (file/overlay).
	SnapshotKindExternal SnapshotKind = "EXTERNAL"
)

// PublicationStatus tracks GitOps publishing lifecycle.
type PublicationStatus string

const (
	PublicationStatusPending   PublicationStatus = "PENDING"
	PublicationStatusCommitted PublicationStatus = "COMMITTED"
	PublicationStatusPRCreated PublicationStatus = "PR_CREATED"
	PublicationStatusMerged    PublicationStatus = "MERGED"
	PublicationStatusFailed    PublicationStatus = "FAILED"
)

// Sandbox represents a disposable VM environment cloned from a golden image.
type Sandbox struct {
	ID          string       `json:"id" db:"id"`                     // e.g., "SBX-0001"
	JobID       string       `json:"job_id" db:"job_id"`             // correlation id for the end-to-end change set
	AgentID     string       `json:"agent_id" db:"agent_id"`         // requesting agent identity
	SandboxName string       `json:"sandbox_name" db:"sandbox_name"` // libvirt domain name
	BaseImage   string       `json:"base_image" db:"base_image"`     // base qcow2 filename
	Network     string       `json:"network" db:"network"`           // libvirt network name
	IPAddress   *string      `json:"ip_address,omitempty" db:"ip"`   // discovered IP (if any)
	State       SandboxState `json:"state" db:"state"`
	TTLSeconds  *int         `json:"ttl_seconds,omitempty" db:"ttl_seconds"` // optional TTL for auto GC

	// Resource allocations for this sandbox
	VCPUs     int   `json:"vcpus" db:"vcpus"`           // Number of vCPUs allocated
	MemoryMB  int   `json:"memory_mb" db:"memory_mb"`   // RAM allocated in MB
	StorageMB int64 `json:"storage_mb" db:"storage_mb"` // Storage used/allocated in MB (qcow2 overlay)

	// Multi-host support: identifies which libvirt host this sandbox runs on
	HostName    *string `json:"host_name,omitempty" db:"host_name"`       // display name of the host (e.g., "kvm-01")
	HostAddress *string `json:"host_address,omitempty" db:"host_address"` // IP or hostname of the libvirt host

	// Metadata
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// SandboxFilter enables scoped queries for sandboxes.
type SandboxFilter struct {
	AgentID   *string
	JobID     *string
	BaseImage *string
	State     *SandboxState
	VMName    *string
}

// HostResources tracks resource inventory and allocations for a libvirt host.
type HostResources struct {
	ID                string    `json:"id" db:"id"`                                   // Host identifier
	Name              string    `json:"name" db:"name"`                               // Display name (e.g., "host-1")
	Address           string    `json:"address" db:"address"`                         // IP or hostname
	TotalCPUs         int       `json:"total_cpus" db:"total_cpus"`                   // Total physical CPUs
	TotalMemoryMB     int64     `json:"total_memory_mb" db:"total_memory_mb"`         // Total RAM in MB
	TotalStorageMB    int64     `json:"total_storage_mb" db:"total_storage_mb"`       // Total available storage in MB
	ReservedCPUs      int       `json:"reserved_cpus" db:"reserved_cpus"`             // CPUs allocated to sandboxes
	ReservedMemoryMB  int64     `json:"reserved_memory_mb" db:"reserved_memory_mb"`   // RAM allocated to sandboxes
	ReservedStorageMB int64     `json:"reserved_storage_mb" db:"reserved_storage_mb"` // Storage allocated to sandboxes
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// Snapshot represents a VM snapshot reference.
type Snapshot struct {
	ID        string       `json:"id" db:"id"`
	SandboxID string       `json:"sandbox_id" db:"sandbox_id"`
	Name      string       `json:"name" db:"name"` // logical name (unique per sandbox)
	Kind      SnapshotKind `json:"kind" db:"kind"`
	// Ref is a backend-specific reference: for internal snapshots this could be a UUID or name,
	// for external snapshots it could be a file path to the overlay qcow2.
	Ref       string    `json:"ref" db:"ref"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	MetaJSON  *string   `json:"meta_json,omitempty" db:"meta_json"` // optional JSON metadata
}

// Command captures an executed command inside a sandbox.
type Command struct {
	ID        string             `json:"id" db:"id"`
	SandboxID string             `json:"sandbox_id" db:"sandbox_id"`
	Command   string             `json:"command" db:"command"`
	EnvJSON   *string            `json:"env_json,omitempty" db:"env_json"` // JSON-encoded env map
	Stdout    string             `json:"stdout" db:"stdout"`
	Stderr    string             `json:"stderr" db:"stderr"`
	ExitCode  int                `json:"exit_code" db:"exit_code"`
	StartedAt time.Time          `json:"started_at" db:"started_at"`
	EndedAt   time.Time          `json:"ended_at" db:"ended_at"`
	Metadata  *CommandExecRecord `json:"metadata,omitempty" db:"-"`
}

// CommandExecRecord is a non-persisted helper payload commonly serialized into Metadata fields.
// It can be persisted by serializing to JSON and storing in an auxiliary column if desired.
type CommandExecRecord struct {
	User     string            `json:"user,omitempty"`
	WorkDir  string            `json:"work_dir,omitempty"`
	Timeout  *time.Duration    `json:"timeout,omitempty"`
	Redacted map[string]string `json:"redacted,omitempty"` // placeholders for secrets redaction
}

// Diff represents a computed difference between two snapshots of a sandbox.
type Diff struct {
	ID           string     `json:"id" db:"id"`
	SandboxID    string     `json:"sandbox_id" db:"sandbox_id"`
	FromSnapshot string     `json:"from_snapshot" db:"from_snapshot"`
	ToSnapshot   string     `json:"to_snapshot" db:"to_snapshot"`
	DiffJSON     ChangeDiff `json:"diff_json" db:"diff_json"` // JSON-encoded change diff
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// PackageInfo captures package name and version.
type PackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServiceChange represents a system service change.
type ServiceChange struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled,omitempty"`
	State   string `json:"state,omitempty"` // started|stopped|restarted|reloaded
}

// CommandSummary summarizes executed commands affecting the diff.
type CommandSummary struct {
	Cmd      string    `json:"cmd"`
	ExitCode int       `json:"exit_code"`
	At       time.Time `json:"at"`
}

// ChangeDiff is the normalized change representation generated by diffing snapshots.
type ChangeDiff struct {
	FilesModified   []string         `json:"files_modified,omitempty"`
	FilesAdded      []string         `json:"files_added,omitempty"`
	FilesRemoved    []string         `json:"files_removed,omitempty"`
	PackagesAdded   []PackageInfo    `json:"packages_added,omitempty"`
	PackagesRemoved []PackageInfo    `json:"packages_removed,omitempty"`
	ServicesChanged []ServiceChange  `json:"services_changed,omitempty"`
	CommandsRun     []CommandSummary `json:"commands_run,omitempty"`
}

// ChangeSet captures generator outputs (Ansible/Puppet) for a job.
type ChangeSet struct {
	ID          string    `json:"id" db:"id"`
	JobID       string    `json:"job_id" db:"job_id"`
	SandboxID   string    `json:"sandbox_id" db:"sandbox_id"`
	DiffID      string    `json:"diff_id" db:"diff_id"`
	PathAnsible string    `json:"path_ansible" db:"path_ansible"` // e.g., /changes/{job_id}/ansible
	PathPuppet  string    `json:"path_puppet" db:"path_puppet"`   // e.g., /changes/{job_id}/puppet
	MetaJSON    *string   `json:"meta_json,omitempty" db:"meta_json"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Publication records a GitOps publication attempt and status.
type Publication struct {
	ID        string            `json:"id" db:"id"`
	JobID     string            `json:"job_id" db:"job_id"`
	RepoURL   string            `json:"repo_url" db:"repo_url"`
	Branch    string            `json:"branch" db:"branch"`
	CommitSHA *string           `json:"commit_sha,omitempty" db:"commit_sha"`
	PRURL     *string           `json:"pr_url,omitempty" db:"pr_url"`
	Status    PublicationStatus `json:"status" db:"status"`
	ErrorMsg  *string           `json:"error_msg,omitempty" db:"error_msg"`
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" db:"updated_at"`
}

// SourceVM tracks preparation state for source/golden VMs used for cloning.
type SourceVM struct {
	ID            string     `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`                           // VM name (unique key)
	HostName      *string    `json:"host_name,omitempty" db:"host_name"`       // display name of the host
	HostAddress   *string    `json:"host_address,omitempty" db:"host_address"` // IP or hostname of the host
	Prepared      bool       `json:"prepared" db:"prepared"`
	PreparedAt    *time.Time `json:"prepared_at,omitempty" db:"prepared_at"`
	PrepareJSON   *string    `json:"prepare_json,omitempty" db:"prepare_json"`     // JSON of PrepareResult
	CAFingerprint *string    `json:"ca_fingerprint,omitempty" db:"ca_fingerprint"` // detect CA rotation
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// Playbook represents an Ansible playbook stored in the database.
type Playbook struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`                     // unique playbook name
	Hosts     string    `json:"hosts" db:"hosts"`                   // target hosts pattern (e.g., "all", "webservers")
	Become    bool      `json:"become" db:"become"`                 // whether to use privilege escalation
	FilePath  *string   `json:"file_path,omitempty" db:"file_path"` // rendered YAML file path
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// PlaybookTask represents a single task within an Ansible playbook.
type PlaybookTask struct {
	ID         string         `json:"id" db:"id"`
	PlaybookID string         `json:"playbook_id" db:"playbook_id"`
	Position   int            `json:"position" db:"position"`                  // ordering within playbook
	Name       string         `json:"name" db:"name"`                          // task name/description
	Module     string         `json:"module" db:"module"`                      // ansible module (apt, shell, copy, etc.)
	Params     map[string]any `json:"params" db:"params" swaggertype:"object"` // module-specific parameters
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
}

// DataStore declares data operations. This is transaction-friendly and
// can be implemented by both the root Store and a transactional context.
type DataStore interface {
	// Sandbox
	CreateSandbox(ctx context.Context, sb *Sandbox) error
	GetSandbox(ctx context.Context, id string) (*Sandbox, error)
	GetSandboxByVMName(ctx context.Context, vmName string) (*Sandbox, error)
	ListSandboxes(ctx context.Context, filter SandboxFilter, opt *ListOptions) ([]*Sandbox, error)
	UpdateSandbox(ctx context.Context, sb *Sandbox) error
	UpdateSandboxState(ctx context.Context, id string, newState SandboxState, ipAddr *string) error
	DeleteSandbox(ctx context.Context, id string) error

	// Snapshot
	CreateSnapshot(ctx context.Context, sn *Snapshot) error
	GetSnapshot(ctx context.Context, id string) (*Snapshot, error)
	GetSnapshotByName(ctx context.Context, sandboxID, name string) (*Snapshot, error)
	ListSnapshots(ctx context.Context, sandboxID string, opt *ListOptions) ([]*Snapshot, error)

	// Command
	SaveCommand(ctx context.Context, cmd *Command) error
	GetCommand(ctx context.Context, id string) (*Command, error)
	ListCommands(ctx context.Context, sandboxID string, opt *ListOptions) ([]*Command, error)

	// Diff
	SaveDiff(ctx context.Context, d *Diff) error
	GetDiff(ctx context.Context, id string) (*Diff, error)
	GetDiffBySnapshots(ctx context.Context, sandboxID, fromSnapshot, toSnapshot string) (*Diff, error)

	// ChangeSet
	CreateChangeSet(ctx context.Context, cs *ChangeSet) error
	GetChangeSet(ctx context.Context, id string) (*ChangeSet, error)
	GetChangeSetByJob(ctx context.Context, jobID string) (*ChangeSet, error)

	// Publication
	CreatePublication(ctx context.Context, p *Publication) error
	UpdatePublicationStatus(ctx context.Context, id string, status PublicationStatus, commitSHA, prURL, errMsg *string) error
	GetPublication(ctx context.Context, id string) (*Publication, error)

	// Playbook
	CreatePlaybook(ctx context.Context, pb *Playbook) error
	GetPlaybook(ctx context.Context, id string) (*Playbook, error)
	GetPlaybookByName(ctx context.Context, name string) (*Playbook, error)
	ListPlaybooks(ctx context.Context, opt *ListOptions) ([]*Playbook, error)
	UpdatePlaybook(ctx context.Context, pb *Playbook) error
	DeletePlaybook(ctx context.Context, id string) error

	// SourceVM
	GetSourceVM(ctx context.Context, name string) (*SourceVM, error)
	UpsertSourceVM(ctx context.Context, svm *SourceVM) error
	ListSourceVMs(ctx context.Context) ([]*SourceVM, error)

	// PlaybookTask
	CreatePlaybookTask(ctx context.Context, task *PlaybookTask) error
	GetPlaybookTask(ctx context.Context, id string) (*PlaybookTask, error)
	ListPlaybookTasks(ctx context.Context, playbookID string, opt *ListOptions) ([]*PlaybookTask, error)
	UpdatePlaybookTask(ctx context.Context, task *PlaybookTask) error
	DeletePlaybookTask(ctx context.Context, id string) error
	ReorderPlaybookTasks(ctx context.Context, playbookID string, taskIDs []string) error
	GetNextTaskPosition(ctx context.Context, playbookID string) (int, error)
}

// Store is the root database handle. It can produce transactional views and
// exposes liveness and lifecycle methods in addition to the DataStore.
type Store interface {
	DataStore

	// Config returns the configuration the store was created with.
	Config() Config

	// Ping verifies DB connectivity/health.
	Ping(ctx context.Context) error

	// WithTx runs fn in a transaction. The provided DataStore must be used for
	// all DB calls within fn and is committed if fn returns nil, rolled back otherwise.
	WithTx(ctx context.Context, fn func(tx DataStore) error) error

	// Close releases resources held by the Store.
	Close() error
}
