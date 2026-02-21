package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for store implementations.
var (
	ErrNotFound      = errors.New("store: not found")
	ErrAlreadyExists = errors.New("store: already exists")
	ErrConflict      = errors.New("store: conflict")
	ErrInvalid       = errors.New("store: invalid data")
)

type Config struct {
	DatabaseURL     string        `json:"database_url"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	AutoMigrate     bool          `json:"auto_migrate"`
	EncryptionKey   string        `json:"-"`
}

type ListOptions struct {
	Limit   int
	Offset  int
	OrderBy string
	Asc     bool
}

// OrgRole enumerates roles within an organization.
type OrgRole string

const (
	OrgRoleOwner  OrgRole = "owner"
	OrgRoleAdmin  OrgRole = "admin"
	OrgRoleMember OrgRole = "member"
)

// SubscriptionPlan enumerates billing plans.
type SubscriptionPlan string

const (
	PlanFree       SubscriptionPlan = "free"
	PlanUsageBased SubscriptionPlan = "usage_based"
)

// SubscriptionStatus enumerates subscription statuses.
type SubscriptionStatus string

const (
	SubStatusActive    SubscriptionStatus = "active"
	SubStatusPastDue   SubscriptionStatus = "past_due"
	SubStatusCancelled SubscriptionStatus = "cancelled"
)

// User represents a registered user.
type User struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	DisplayName   string     `json:"display_name"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	PasswordHash  string     `json:"-"`
	EmailVerified bool       `json:"email_verified"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// OAuthAccount links an external OAuth provider identity to a user.
type OAuthAccount struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Provider     string    `json:"provider"`
	ProviderID   string    `json:"provider_id"`
	Email        string    `json:"email"`
	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	TokenExpiry  time.Time `json:"token_expiry"`
	CreatedAt    time.Time `json:"created_at"`
}

// Session represents an active user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Organization represents a team/org that owns resources.
type Organization struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Slug             string     `json:"slug"`
	OwnerID          string     `json:"owner_id"`
	StripeCustomerID string     `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

// OrgMember represents a user's membership in an organization.
type OrgMember struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	Role      OrgRole   `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Subscription tracks an org's billing plan.
type Subscription struct {
	ID                   string             `json:"id"`
	OrgID                string             `json:"org_id"`
	Plan                 SubscriptionPlan   `json:"plan"`
	StripeSubscriptionID string             `json:"stripe_subscription_id,omitempty"`
	StripePriceID        string             `json:"stripe_price_id,omitempty"`
	Status               SubscriptionStatus `json:"status"`
	CurrentPeriodStart   time.Time          `json:"current_period_start"`
	CurrentPeriodEnd     time.Time          `json:"current_period_end"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// UsageRecord tracks resource consumption.
type UsageRecord struct {
	ID           string    `json:"id"`
	OrgID        string    `json:"org_id"`
	ResourceType string    `json:"resource_type"`
	Quantity     float64   `json:"quantity"`
	RecordedAt   time.Time `json:"recorded_at"`
	MetadataJSON string    `json:"metadata_json,omitempty"`
}

// ---------------------------------------------------------------------------
// Infrastructure types (from control-plane)
// ---------------------------------------------------------------------------

// HostStatus represents the connectivity status of a sandbox host.
type HostStatus string

const (
	HostStatusOnline  HostStatus = "ONLINE"
	HostStatusOffline HostStatus = "OFFLINE"
)

// SandboxState represents the lifecycle state of a sandbox.
type SandboxState string

const (
	SandboxStateCreating  SandboxState = "CREATING"
	SandboxStateRunning   SandboxState = "RUNNING"
	SandboxStateStopped   SandboxState = "STOPPED"
	SandboxStateDestroyed SandboxState = "DESTROYED"
	SandboxStateError     SandboxState = "ERROR"
)

// StringSlice is a JSON-serialized []string for use as a GORM column type.
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal StringSlice: %w", err)
	}
	return string(b), nil
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for StringSlice: %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// SourceVMJSON represents a source VM entry stored as JSON in the host record.
type SourceVMJSON struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	IPAddress string `json:"ip_address"`
	Prepared  bool   `json:"prepared"`
}

// SourceVMSlice is a JSON-serialized []SourceVMJSON for use as a GORM column type.
type SourceVMSlice []SourceVMJSON

func (s SourceVMSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal SourceVMSlice: %w", err)
	}
	return string(b), nil
}

func (s *SourceVMSlice) Scan(value interface{}) error {
	if value == nil {
		*s = SourceVMSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for SourceVMSlice: %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// BridgeJSON represents a network bridge entry stored as JSON in the host record.
type BridgeJSON struct {
	Name   string `json:"name"`
	Subnet string `json:"subnet"`
}

// BridgeSlice is a JSON-serialized []BridgeJSON for use as a GORM column type.
type BridgeSlice []BridgeJSON

func (s BridgeSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal BridgeSlice: %w", err)
	}
	return string(b), nil
}

func (s *BridgeSlice) Scan(value interface{}) error {
	if value == nil {
		*s = BridgeSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("unsupported type for BridgeSlice: %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// Host represents a sandbox host machine registered with the control plane.
type Host struct {
	ID                string        `json:"id"`
	OrgID             string        `json:"org_id"`
	Hostname          string        `json:"hostname"`
	Version           string        `json:"version"`
	TotalCPUs         int32         `json:"total_cpus"`
	TotalMemoryMB     int64         `json:"total_memory_mb"`
	TotalDiskMB       int64         `json:"total_disk_mb"`
	AvailableCPUs     int32         `json:"available_cpus"`
	AvailableMemoryMB int64         `json:"available_memory_mb"`
	AvailableDiskMB   int64         `json:"available_disk_mb"`
	BaseImages        StringSlice   `json:"base_images"`
	SourceVMs         SourceVMSlice `json:"source_vms"`
	Bridges           BridgeSlice   `json:"bridges"`
	Status            HostStatus    `json:"status"`
	LastHeartbeat     time.Time     `json:"last_heartbeat"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// Sandbox represents a VM sandbox managed by the control plane.
type Sandbox struct {
	ID         string       `json:"id"`
	OrgID      string       `json:"org_id"`
	HostID     string       `json:"host_id"`
	Name       string       `json:"name"`
	AgentID    string       `json:"agent_id"`
	BaseImage  string       `json:"base_image"`
	Bridge     string       `json:"bridge"`
	TAPDevice  string       `json:"tap_device"`
	MACAddress string       `json:"mac_address"`
	IPAddress  string       `json:"ip_address"`
	State      SandboxState `json:"state"`
	VCPUs      int32        `json:"vcpus"`
	MemoryMB   int32        `json:"memory_mb"`
	TTLSeconds int32        `json:"ttl_seconds"`
	SourceVM   string       `json:"source_vm"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
	DeletedAt  *time.Time   `json:"deleted_at,omitempty"`
}

// Command represents a command executed within a sandbox.
type Command struct {
	ID         string    `json:"id"`
	SandboxID  string    `json:"sandbox_id"`
	Command    string    `json:"command"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
	ExitCode   int32     `json:"exit_code"`
	DurationMS int64     `json:"duration_ms"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
}

// Agent conversation and playbook types - commented out, not yet ready for integration.
/*
// AgentConversation represents a chat conversation with the AI agent.
type AgentConversation struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MessageRole enumerates agent message roles.
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

// AgentMessage represents a single message in an agent conversation.
type AgentMessage struct {
	ID             string      `json:"id"`
	ConversationID string      `json:"conversation_id"`
	Role           MessageRole `json:"role"`
	Content        string      `json:"content"`
	ToolCalls      string      `json:"tool_calls,omitempty"`
	ToolCallID     string      `json:"tool_call_id,omitempty"`
	TokensInput    int         `json:"tokens_input,omitempty"`
	TokensOutput   int         `json:"tokens_output,omitempty"`
	Model          string      `json:"model,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

// Playbook represents an Ansible-style playbook.
type Playbook struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PlaybookTask represents a single task within a playbook.
type PlaybookTask struct {
	ID         string    `json:"id"`
	PlaybookID string    `json:"playbook_id"`
	SortOrder  int       `json:"sort_order"`
	Name       string    `json:"name"`
	Module     string    `json:"module"`
	Params     string    `json:"params"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
*/

// ModelMeter tracks Stripe meter/price objects for a specific LLM model.
type ModelMeter struct {
	ID                  string    `json:"id"`
	ModelID             string    `json:"model_id"`
	StripeProductID     string    `json:"stripe_product_id"`
	StripeInputMeterID  string    `json:"stripe_input_meter_id"`
	StripeOutputMeterID string    `json:"stripe_output_meter_id"`
	StripeInputPriceID  string    `json:"stripe_input_price_id"`
	StripeOutputPriceID string    `json:"stripe_output_price_id"`
	InputEventName      string    `json:"input_event_name"`
	OutputEventName     string    `json:"output_event_name"`
	InputCostPerToken   float64   `json:"input_cost_per_token"`
	OutputCostPerToken  float64   `json:"output_cost_per_token"`
	CreatedAt           time.Time `json:"created_at"`
}

// OrgModelSubscription links an org's Stripe subscription items for a specific model.
type OrgModelSubscription struct {
	ID                    string    `json:"id"`
	OrgID                 string    `json:"org_id"`
	ModelID               string    `json:"model_id"`
	StripeInputSubItemID  string    `json:"stripe_input_sub_item_id"`
	StripeOutputSubItemID string    `json:"stripe_output_sub_item_id"`
	CreatedAt             time.Time `json:"created_at"`
}

// SourceHost represents a confirmed source host that can be used for snapshot-based sandboxes.
type SourceHost struct {
	ID               string      `json:"id"`
	OrgID            string      `json:"org_id"`
	Name             string      `json:"name"`
	Hostname         string      `json:"hostname"`
	Type             string      `json:"type"` // "libvirt" or "proxmox"
	SSHUser          string      `json:"ssh_user"`
	SSHPort          int         `json:"ssh_port"`
	SSHIdentityFile  string      `json:"ssh_identity_file"`
	ProxmoxHost      string      `json:"proxmox_host,omitempty"`
	ProxmoxTokenID   string      `json:"proxmox_token_id,omitempty"`
	ProxmoxSecret    string      `json:"-"`
	ProxmoxNode      string      `json:"proxmox_node,omitempty"`
	ProxmoxVerifySSL bool        `json:"proxmox_verify_ssl,omitempty"`
	VMs              StringSlice `json:"vms"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// HostToken is a bearer token that a sandbox host uses to authenticate its
// gRPC connection. Tokens are scoped to an organization.
type HostToken struct {
	ID        string     `json:"id"`
	OrgID     string     `json:"org_id"`
	Name      string     `json:"name"`
	TokenHash string     `json:"-"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// DataStore declares data operations.
type DataStore interface {
	// User
	CreateUser(ctx context.Context, u *User) error
	GetUser(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, u *User) error

	// OAuth
	CreateOAuthAccount(ctx context.Context, oa *OAuthAccount) error
	GetOAuthAccount(ctx context.Context, provider, providerID string) (*OAuthAccount, error)
	GetOAuthAccountsByUser(ctx context.Context, userID string) ([]*OAuthAccount, error)

	// Session
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error

	// Organization
	CreateOrganization(ctx context.Context, org *Organization) error
	GetOrganization(ctx context.Context, id string) (*Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
	ListOrganizationsByUser(ctx context.Context, userID string) ([]*Organization, error)
	UpdateOrganization(ctx context.Context, org *Organization) error
	DeleteOrganization(ctx context.Context, id string) error

	// OrgMember
	CreateOrgMember(ctx context.Context, m *OrgMember) error
	GetOrgMember(ctx context.Context, orgID, userID string) (*OrgMember, error)
	GetOrgMemberByID(ctx context.Context, orgID, memberID string) (*OrgMember, error)
	ListOrgMembers(ctx context.Context, orgID string) ([]*OrgMember, error)
	DeleteOrgMember(ctx context.Context, orgID, id string) error

	// Subscription
	CreateSubscription(ctx context.Context, sub *Subscription) error
	GetSubscriptionByOrg(ctx context.Context, orgID string) (*Subscription, error)
	UpdateSubscription(ctx context.Context, sub *Subscription) error
	GetSubscriptionByStripeID(ctx context.Context, stripeSubID string) (*Subscription, error)

	// Usage
	CreateUsageRecord(ctx context.Context, rec *UsageRecord) error
	ListUsageRecords(ctx context.Context, orgID string, from, to time.Time) ([]*UsageRecord, error)

	// Host
	CreateHost(ctx context.Context, host *Host) error
	GetHost(ctx context.Context, hostID string) (*Host, error)
	ListHosts(ctx context.Context) ([]Host, error)
	ListHostsByOrg(ctx context.Context, orgID string) ([]Host, error)
	UpdateHost(ctx context.Context, host *Host) error
	UpdateHostHeartbeat(ctx context.Context, hostID string, availCPUs int32, availMemMB int64, availDiskMB int64) error

	// Sandbox
	CreateSandbox(ctx context.Context, sandbox *Sandbox) error
	GetSandbox(ctx context.Context, sandboxID string) (*Sandbox, error)
	GetSandboxByOrg(ctx context.Context, orgID, sandboxID string) (*Sandbox, error)
	ListSandboxes(ctx context.Context) ([]Sandbox, error)
	ListSandboxesByOrg(ctx context.Context, orgID string) ([]Sandbox, error)
	UpdateSandbox(ctx context.Context, sandbox *Sandbox) error
	DeleteSandbox(ctx context.Context, sandboxID string) error
	GetSandboxesByHostID(ctx context.Context, hostID string) ([]Sandbox, error)
	CountSandboxesByHostIDs(ctx context.Context, hostIDs []string) (map[string]int, error)
	ListExpiredSandboxes(ctx context.Context, defaultTTL time.Duration) ([]Sandbox, error)

	// Command
	CreateCommand(ctx context.Context, cmd *Command) error
	ListSandboxCommands(ctx context.Context, sandboxID string) ([]Command, error)

	// SourceHost
	CreateSourceHost(ctx context.Context, sh *SourceHost) error
	GetSourceHost(ctx context.Context, id string) (*SourceHost, error)
	ListSourceHostsByOrg(ctx context.Context, orgID string) ([]*SourceHost, error)
	DeleteSourceHost(ctx context.Context, id string) error

	// HostToken
	CreateHostToken(ctx context.Context, token *HostToken) error
	GetHostTokenByHash(ctx context.Context, hash string) (*HostToken, error)
	ListHostTokensByOrg(ctx context.Context, orgID string) ([]HostToken, error)
	DeleteHostToken(ctx context.Context, orgID, id string) error

	// Agent Conversations - commented out, not yet ready for integration
	// CreateAgentConversation(ctx context.Context, conv *AgentConversation) error
	// GetAgentConversation(ctx context.Context, id string) (*AgentConversation, error)
	// ListAgentConversationsByOrg(ctx context.Context, orgID string) ([]*AgentConversation, error)
	// DeleteAgentConversation(ctx context.Context, id string) error

	// Agent Messages - commented out, not yet ready for integration
	// CreateAgentMessage(ctx context.Context, msg *AgentMessage) error
	// ListAgentMessages(ctx context.Context, conversationID string) ([]*AgentMessage, error)

	// Playbooks - commented out, not yet ready for integration
	// CreatePlaybook(ctx context.Context, pb *Playbook) error
	// GetPlaybook(ctx context.Context, id string) (*Playbook, error)
	// ListPlaybooksByOrg(ctx context.Context, orgID string) ([]*Playbook, error)
	// UpdatePlaybook(ctx context.Context, pb *Playbook) error
	// DeletePlaybook(ctx context.Context, id string) error

	// Playbook Tasks - commented out, not yet ready for integration
	// CreatePlaybookTask(ctx context.Context, task *PlaybookTask) error
	// GetPlaybookTask(ctx context.Context, id string) (*PlaybookTask, error)
	// ListPlaybookTasks(ctx context.Context, playbookID string) ([]*PlaybookTask, error)
	// UpdatePlaybookTask(ctx context.Context, task *PlaybookTask) error
	// DeletePlaybookTask(ctx context.Context, id string) error
	// ReorderPlaybookTasks(ctx context.Context, playbookID string, taskIDs []string) error

	// Billing helpers
	GetOrganizationByStripeCustomerID(ctx context.Context, customerID string) (*Organization, error)
	GetModelMeter(ctx context.Context, modelID string) (*ModelMeter, error)
	CreateModelMeter(ctx context.Context, m *ModelMeter) error
	GetOrgModelSubscription(ctx context.Context, orgID, modelID string) (*OrgModelSubscription, error)
	CreateOrgModelSubscription(ctx context.Context, s *OrgModelSubscription) error
	SumTokenUsage(ctx context.Context, orgID string, from, to time.Time) (float64, error)
	ListActiveSubscriptions(ctx context.Context) ([]*Subscription, error)

	// Advisory locks (multi-instance safety)
	AcquireAdvisoryLock(ctx context.Context, key int64) error
	ReleaseAdvisoryLock(ctx context.Context, key int64) error
}

// Store is the root database handle with lifecycle methods.
type Store interface {
	DataStore
	Config() Config
	Ping(ctx context.Context) error
	WithTx(ctx context.Context, fn func(tx DataStore) error) error
	Close() error
}
