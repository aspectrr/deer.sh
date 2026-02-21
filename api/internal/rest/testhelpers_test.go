package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	"github.com/aspectrr/fluid.sh/api/internal/config"
	"github.com/aspectrr/fluid.sh/api/internal/orchestrator"
	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/store"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
)

// ---------------------------------------------------------------------------
// mockStore implements store.Store with function-field delegation
// ---------------------------------------------------------------------------

type mockStore struct {
	// User
	CreateUserFn     func(ctx context.Context, u *store.User) error
	GetUserFn        func(ctx context.Context, id string) (*store.User, error)
	GetUserByEmailFn func(ctx context.Context, email string) (*store.User, error)
	UpdateUserFn     func(ctx context.Context, u *store.User) error

	// OAuth
	CreateOAuthAccountFn     func(ctx context.Context, oa *store.OAuthAccount) error
	GetOAuthAccountFn        func(ctx context.Context, provider, providerID string) (*store.OAuthAccount, error)
	GetOAuthAccountsByUserFn func(ctx context.Context, userID string) ([]*store.OAuthAccount, error)

	// Session
	CreateSessionFn         func(ctx context.Context, s *store.Session) error
	GetSessionFn            func(ctx context.Context, id string) (*store.Session, error)
	DeleteSessionFn         func(ctx context.Context, id string) error
	DeleteExpiredSessionsFn func(ctx context.Context) error

	// Organization
	CreateOrganizationFn      func(ctx context.Context, org *store.Organization) error
	GetOrganizationFn         func(ctx context.Context, id string) (*store.Organization, error)
	GetOrganizationBySlugFn   func(ctx context.Context, slug string) (*store.Organization, error)
	ListOrganizationsByUserFn func(ctx context.Context, userID string) ([]*store.Organization, error)
	UpdateOrganizationFn      func(ctx context.Context, org *store.Organization) error
	DeleteOrganizationFn      func(ctx context.Context, id string) error

	// OrgMember
	CreateOrgMemberFn  func(ctx context.Context, m *store.OrgMember) error
	GetOrgMemberFn     func(ctx context.Context, orgID, userID string) (*store.OrgMember, error)
	GetOrgMemberByIDFn func(ctx context.Context, orgID, memberID string) (*store.OrgMember, error)
	ListOrgMembersFn   func(ctx context.Context, orgID string) ([]*store.OrgMember, error)
	DeleteOrgMemberFn  func(ctx context.Context, orgID, id string) error

	// Subscription
	CreateSubscriptionFn   func(ctx context.Context, sub *store.Subscription) error
	GetSubscriptionByOrgFn func(ctx context.Context, orgID string) (*store.Subscription, error)
	UpdateSubscriptionFn   func(ctx context.Context, sub *store.Subscription) error

	// Usage
	CreateUsageRecordFn func(ctx context.Context, rec *store.UsageRecord) error
	ListUsageRecordsFn  func(ctx context.Context, orgID string, from, to time.Time) ([]*store.UsageRecord, error)

	// Host
	CreateHostFn          func(ctx context.Context, host *store.Host) error
	GetHostFn             func(ctx context.Context, hostID string) (*store.Host, error)
	ListHostsFn           func(ctx context.Context) ([]store.Host, error)
	UpdateHostFn          func(ctx context.Context, host *store.Host) error
	UpdateHostHeartbeatFn func(ctx context.Context, hostID string, availCPUs int32, availMemMB int64, availDiskMB int64) error

	// Sandbox
	CreateSandboxFn        func(ctx context.Context, sandbox *store.Sandbox) error
	GetSandboxFn           func(ctx context.Context, sandboxID string) (*store.Sandbox, error)
	ListSandboxesFn        func(ctx context.Context) ([]store.Sandbox, error)
	ListSandboxesByOrgFn   func(ctx context.Context, orgID string) ([]store.Sandbox, error)
	UpdateSandboxFn        func(ctx context.Context, sandbox *store.Sandbox) error
	DeleteSandboxFn        func(ctx context.Context, sandboxID string) error
	GetSandboxesByHostIDFn func(ctx context.Context, hostID string) ([]store.Sandbox, error)
	ListExpiredSandboxesFn func(ctx context.Context, defaultTTL time.Duration) ([]store.Sandbox, error)

	// Command
	CreateCommandFn       func(ctx context.Context, cmd *store.Command) error
	ListSandboxCommandsFn func(ctx context.Context, sandboxID string) ([]store.Command, error)

	// SourceHost
	CreateSourceHostFn     func(ctx context.Context, sh *store.SourceHost) error
	GetSourceHostFn        func(ctx context.Context, id string) (*store.SourceHost, error)
	ListSourceHostsByOrgFn func(ctx context.Context, orgID string) ([]*store.SourceHost, error)
	DeleteSourceHostFn     func(ctx context.Context, id string) error

	// HostToken
	CreateHostTokenFn     func(ctx context.Context, token *store.HostToken) error
	GetHostTokenByHashFn  func(ctx context.Context, hash string) (*store.HostToken, error)
	ListHostTokensByOrgFn func(ctx context.Context, orgID string) ([]store.HostToken, error)
	DeleteHostTokenFn     func(ctx context.Context, orgID, id string) error

	// Agent Conversations, Messages, Playbooks, Tasks - commented out
	// (types are commented out in store.go)

	// Billing helpers
	GetOrganizationByStripeCustomerIDFn func(ctx context.Context, customerID string) (*store.Organization, error)
	GetModelMeterFn                     func(ctx context.Context, modelID string) (*store.ModelMeter, error)
	CreateModelMeterFn                  func(ctx context.Context, m *store.ModelMeter) error
	GetOrgModelSubscriptionFn           func(ctx context.Context, orgID, modelID string) (*store.OrgModelSubscription, error)
	CreateOrgModelSubscriptionFn        func(ctx context.Context, s *store.OrgModelSubscription) error
	SumTokenUsageFn                     func(ctx context.Context, orgID string, from, to time.Time) (float64, error)
	ListActiveSubscriptionsFn           func(ctx context.Context) ([]*store.Subscription, error)

	// WithTx
	WithTxFn func(ctx context.Context, fn func(tx store.DataStore) error) error
}

func (m *mockStore) call(name string) { panic(fmt.Sprintf("mockStore.%s not configured", name)) }

// Store interface
func (m *mockStore) Config() store.Config       { return store.Config{} }
func (m *mockStore) Ping(context.Context) error { return nil }
func (m *mockStore) Close() error               { return nil }

func (m *mockStore) WithTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	if m.WithTxFn != nil {
		return m.WithTxFn(ctx, fn)
	}
	return fn(m)
}

// User
func (m *mockStore) CreateUser(ctx context.Context, u *store.User) error {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, u)
	}
	m.call("CreateUser")
	return nil
}
func (m *mockStore) GetUser(ctx context.Context, id string) (*store.User, error) {
	if m.GetUserFn != nil {
		return m.GetUserFn(ctx, id)
	}
	m.call("GetUser")
	return nil, nil
}
func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(ctx, email)
	}
	m.call("GetUserByEmail")
	return nil, nil
}
func (m *mockStore) UpdateUser(ctx context.Context, u *store.User) error {
	if m.UpdateUserFn != nil {
		return m.UpdateUserFn(ctx, u)
	}
	m.call("UpdateUser")
	return nil
}

// OAuth
func (m *mockStore) CreateOAuthAccount(ctx context.Context, oa *store.OAuthAccount) error {
	if m.CreateOAuthAccountFn != nil {
		return m.CreateOAuthAccountFn(ctx, oa)
	}
	m.call("CreateOAuthAccount")
	return nil
}
func (m *mockStore) GetOAuthAccount(ctx context.Context, provider, providerID string) (*store.OAuthAccount, error) {
	if m.GetOAuthAccountFn != nil {
		return m.GetOAuthAccountFn(ctx, provider, providerID)
	}
	m.call("GetOAuthAccount")
	return nil, nil
}
func (m *mockStore) GetOAuthAccountsByUser(ctx context.Context, userID string) ([]*store.OAuthAccount, error) {
	if m.GetOAuthAccountsByUserFn != nil {
		return m.GetOAuthAccountsByUserFn(ctx, userID)
	}
	m.call("GetOAuthAccountsByUser")
	return nil, nil
}

// Session
func (m *mockStore) CreateSession(ctx context.Context, s *store.Session) error {
	if m.CreateSessionFn != nil {
		return m.CreateSessionFn(ctx, s)
	}
	m.call("CreateSession")
	return nil
}
func (m *mockStore) GetSession(ctx context.Context, id string) (*store.Session, error) {
	if m.GetSessionFn != nil {
		return m.GetSessionFn(ctx, id)
	}
	m.call("GetSession")
	return nil, nil
}
func (m *mockStore) DeleteSession(ctx context.Context, id string) error {
	if m.DeleteSessionFn != nil {
		return m.DeleteSessionFn(ctx, id)
	}
	m.call("DeleteSession")
	return nil
}
func (m *mockStore) DeleteExpiredSessions(ctx context.Context) error {
	if m.DeleteExpiredSessionsFn != nil {
		return m.DeleteExpiredSessionsFn(ctx)
	}
	m.call("DeleteExpiredSessions")
	return nil
}

// Organization
func (m *mockStore) CreateOrganization(ctx context.Context, org *store.Organization) error {
	if m.CreateOrganizationFn != nil {
		return m.CreateOrganizationFn(ctx, org)
	}
	m.call("CreateOrganization")
	return nil
}
func (m *mockStore) GetOrganization(ctx context.Context, id string) (*store.Organization, error) {
	if m.GetOrganizationFn != nil {
		return m.GetOrganizationFn(ctx, id)
	}
	m.call("GetOrganization")
	return nil, nil
}
func (m *mockStore) GetOrganizationBySlug(ctx context.Context, slug string) (*store.Organization, error) {
	if m.GetOrganizationBySlugFn != nil {
		return m.GetOrganizationBySlugFn(ctx, slug)
	}
	m.call("GetOrganizationBySlug")
	return nil, nil
}
func (m *mockStore) ListOrganizationsByUser(ctx context.Context, userID string) ([]*store.Organization, error) {
	if m.ListOrganizationsByUserFn != nil {
		return m.ListOrganizationsByUserFn(ctx, userID)
	}
	m.call("ListOrganizationsByUser")
	return nil, nil
}
func (m *mockStore) UpdateOrganization(ctx context.Context, org *store.Organization) error {
	if m.UpdateOrganizationFn != nil {
		return m.UpdateOrganizationFn(ctx, org)
	}
	m.call("UpdateOrganization")
	return nil
}
func (m *mockStore) DeleteOrganization(ctx context.Context, id string) error {
	if m.DeleteOrganizationFn != nil {
		return m.DeleteOrganizationFn(ctx, id)
	}
	m.call("DeleteOrganization")
	return nil
}

// OrgMember
func (m *mockStore) CreateOrgMember(ctx context.Context, mem *store.OrgMember) error {
	if m.CreateOrgMemberFn != nil {
		return m.CreateOrgMemberFn(ctx, mem)
	}
	m.call("CreateOrgMember")
	return nil
}
func (m *mockStore) GetOrgMember(ctx context.Context, orgID, userID string) (*store.OrgMember, error) {
	if m.GetOrgMemberFn != nil {
		return m.GetOrgMemberFn(ctx, orgID, userID)
	}
	m.call("GetOrgMember")
	return nil, nil
}
func (m *mockStore) GetOrgMemberByID(ctx context.Context, orgID, memberID string) (*store.OrgMember, error) {
	if m.GetOrgMemberByIDFn != nil {
		return m.GetOrgMemberByIDFn(ctx, orgID, memberID)
	}
	m.call("GetOrgMemberByID")
	return nil, nil
}
func (m *mockStore) ListOrgMembers(ctx context.Context, orgID string) ([]*store.OrgMember, error) {
	if m.ListOrgMembersFn != nil {
		return m.ListOrgMembersFn(ctx, orgID)
	}
	m.call("ListOrgMembers")
	return nil, nil
}
func (m *mockStore) DeleteOrgMember(ctx context.Context, orgID, id string) error {
	if m.DeleteOrgMemberFn != nil {
		return m.DeleteOrgMemberFn(ctx, orgID, id)
	}
	m.call("DeleteOrgMember")
	return nil
}

// Subscription
func (m *mockStore) CreateSubscription(ctx context.Context, sub *store.Subscription) error {
	if m.CreateSubscriptionFn != nil {
		return m.CreateSubscriptionFn(ctx, sub)
	}
	m.call("CreateSubscription")
	return nil
}
func (m *mockStore) GetSubscriptionByOrg(ctx context.Context, orgID string) (*store.Subscription, error) {
	if m.GetSubscriptionByOrgFn != nil {
		return m.GetSubscriptionByOrgFn(ctx, orgID)
	}
	m.call("GetSubscriptionByOrg")
	return nil, nil
}
func (m *mockStore) UpdateSubscription(ctx context.Context, sub *store.Subscription) error {
	if m.UpdateSubscriptionFn != nil {
		return m.UpdateSubscriptionFn(ctx, sub)
	}
	m.call("UpdateSubscription")
	return nil
}

// Usage
func (m *mockStore) CreateUsageRecord(ctx context.Context, rec *store.UsageRecord) error {
	if m.CreateUsageRecordFn != nil {
		return m.CreateUsageRecordFn(ctx, rec)
	}
	m.call("CreateUsageRecord")
	return nil
}
func (m *mockStore) ListUsageRecords(ctx context.Context, orgID string, from, to time.Time) ([]*store.UsageRecord, error) {
	if m.ListUsageRecordsFn != nil {
		return m.ListUsageRecordsFn(ctx, orgID, from, to)
	}
	m.call("ListUsageRecords")
	return nil, nil
}

// Host
func (m *mockStore) CreateHost(ctx context.Context, host *store.Host) error {
	if m.CreateHostFn != nil {
		return m.CreateHostFn(ctx, host)
	}
	m.call("CreateHost")
	return nil
}
func (m *mockStore) GetHost(ctx context.Context, hostID string) (*store.Host, error) {
	if m.GetHostFn != nil {
		return m.GetHostFn(ctx, hostID)
	}
	m.call("GetHost")
	return nil, nil
}
func (m *mockStore) ListHosts(ctx context.Context) ([]store.Host, error) {
	if m.ListHostsFn != nil {
		return m.ListHostsFn(ctx)
	}
	m.call("ListHosts")
	return nil, nil
}
func (m *mockStore) ListHostsByOrg(_ context.Context, _ string) ([]store.Host, error) {
	return nil, nil
}
func (m *mockStore) UpdateHost(ctx context.Context, host *store.Host) error {
	if m.UpdateHostFn != nil {
		return m.UpdateHostFn(ctx, host)
	}
	m.call("UpdateHost")
	return nil
}
func (m *mockStore) UpdateHostHeartbeat(ctx context.Context, hostID string, availCPUs int32, availMemMB int64, availDiskMB int64) error {
	if m.UpdateHostHeartbeatFn != nil {
		return m.UpdateHostHeartbeatFn(ctx, hostID, availCPUs, availMemMB, availDiskMB)
	}
	m.call("UpdateHostHeartbeat")
	return nil
}

// Sandbox
func (m *mockStore) CreateSandbox(ctx context.Context, sandbox *store.Sandbox) error {
	if m.CreateSandboxFn != nil {
		return m.CreateSandboxFn(ctx, sandbox)
	}
	m.call("CreateSandbox")
	return nil
}
func (m *mockStore) GetSandbox(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
	if m.GetSandboxFn != nil {
		return m.GetSandboxFn(ctx, sandboxID)
	}
	m.call("GetSandbox")
	return nil, nil
}
func (m *mockStore) GetSandboxByOrg(ctx context.Context, orgID, sandboxID string) (*store.Sandbox, error) {
	if m.GetSandboxFn != nil {
		sb, err := m.GetSandboxFn(ctx, sandboxID)
		if err != nil {
			return nil, err
		}
		if sb.OrgID != orgID {
			return nil, store.ErrNotFound
		}
		return sb, nil
	}
	m.call("GetSandboxByOrg")
	return nil, nil
}
func (m *mockStore) ListSandboxes(ctx context.Context) ([]store.Sandbox, error) {
	if m.ListSandboxesFn != nil {
		return m.ListSandboxesFn(ctx)
	}
	m.call("ListSandboxes")
	return nil, nil
}
func (m *mockStore) ListSandboxesByOrg(ctx context.Context, orgID string) ([]store.Sandbox, error) {
	if m.ListSandboxesByOrgFn != nil {
		return m.ListSandboxesByOrgFn(ctx, orgID)
	}
	m.call("ListSandboxesByOrg")
	return nil, nil
}
func (m *mockStore) UpdateSandbox(ctx context.Context, sandbox *store.Sandbox) error {
	if m.UpdateSandboxFn != nil {
		return m.UpdateSandboxFn(ctx, sandbox)
	}
	m.call("UpdateSandbox")
	return nil
}
func (m *mockStore) DeleteSandbox(ctx context.Context, sandboxID string) error {
	if m.DeleteSandboxFn != nil {
		return m.DeleteSandboxFn(ctx, sandboxID)
	}
	m.call("DeleteSandbox")
	return nil
}
func (m *mockStore) GetSandboxesByHostID(ctx context.Context, hostID string) ([]store.Sandbox, error) {
	if m.GetSandboxesByHostIDFn != nil {
		return m.GetSandboxesByHostIDFn(ctx, hostID)
	}
	m.call("GetSandboxesByHostID")
	return nil, nil
}
func (m *mockStore) CountSandboxesByHostIDs(_ context.Context, _ []string) (map[string]int, error) {
	return map[string]int{}, nil
}
func (m *mockStore) ListExpiredSandboxes(ctx context.Context, defaultTTL time.Duration) ([]store.Sandbox, error) {
	if m.ListExpiredSandboxesFn != nil {
		return m.ListExpiredSandboxesFn(ctx, defaultTTL)
	}
	m.call("ListExpiredSandboxes")
	return nil, nil
}

// Command
func (m *mockStore) CreateCommand(ctx context.Context, cmd *store.Command) error {
	if m.CreateCommandFn != nil {
		return m.CreateCommandFn(ctx, cmd)
	}
	m.call("CreateCommand")
	return nil
}
func (m *mockStore) ListSandboxCommands(ctx context.Context, sandboxID string) ([]store.Command, error) {
	if m.ListSandboxCommandsFn != nil {
		return m.ListSandboxCommandsFn(ctx, sandboxID)
	}
	m.call("ListSandboxCommands")
	return nil, nil
}

// SourceHost
func (m *mockStore) CreateSourceHost(ctx context.Context, sh *store.SourceHost) error {
	if m.CreateSourceHostFn != nil {
		return m.CreateSourceHostFn(ctx, sh)
	}
	m.call("CreateSourceHost")
	return nil
}
func (m *mockStore) GetSourceHost(ctx context.Context, id string) (*store.SourceHost, error) {
	if m.GetSourceHostFn != nil {
		return m.GetSourceHostFn(ctx, id)
	}
	m.call("GetSourceHost")
	return nil, nil
}
func (m *mockStore) ListSourceHostsByOrg(ctx context.Context, orgID string) ([]*store.SourceHost, error) {
	if m.ListSourceHostsByOrgFn != nil {
		return m.ListSourceHostsByOrgFn(ctx, orgID)
	}
	m.call("ListSourceHostsByOrg")
	return nil, nil
}
func (m *mockStore) DeleteSourceHost(ctx context.Context, id string) error {
	if m.DeleteSourceHostFn != nil {
		return m.DeleteSourceHostFn(ctx, id)
	}
	m.call("DeleteSourceHost")
	return nil
}

// HostToken
func (m *mockStore) CreateHostToken(ctx context.Context, token *store.HostToken) error {
	if m.CreateHostTokenFn != nil {
		return m.CreateHostTokenFn(ctx, token)
	}
	m.call("CreateHostToken")
	return nil
}
func (m *mockStore) GetHostTokenByHash(ctx context.Context, hash string) (*store.HostToken, error) {
	if m.GetHostTokenByHashFn != nil {
		return m.GetHostTokenByHashFn(ctx, hash)
	}
	m.call("GetHostTokenByHash")
	return nil, nil
}
func (m *mockStore) ListHostTokensByOrg(ctx context.Context, orgID string) ([]store.HostToken, error) {
	if m.ListHostTokensByOrgFn != nil {
		return m.ListHostTokensByOrgFn(ctx, orgID)
	}
	m.call("ListHostTokensByOrg")
	return nil, nil
}
func (m *mockStore) DeleteHostToken(ctx context.Context, orgID, id string) error {
	if m.DeleteHostTokenFn != nil {
		return m.DeleteHostTokenFn(ctx, orgID, id)
	}
	m.call("DeleteHostToken")
	return nil
}

// Agent Conversations, Messages, Playbooks, Tasks mock methods - commented out
/*
func (m *mockStore) CreateAgentConversation(ctx context.Context, conv *store.AgentConversation) error {
	if m.CreateAgentConversationFn != nil {
		return m.CreateAgentConversationFn(ctx, conv)
	}
	m.call("CreateAgentConversation")
	return nil
}
func (m *mockStore) GetAgentConversation(ctx context.Context, id string) (*store.AgentConversation, error) {
	if m.GetAgentConversationFn != nil {
		return m.GetAgentConversationFn(ctx, id)
	}
	m.call("GetAgentConversation")
	return nil, nil
}
func (m *mockStore) ListAgentConversationsByOrg(ctx context.Context, orgID string) ([]*store.AgentConversation, error) {
	if m.ListAgentConversationsByOrgFn != nil {
		return m.ListAgentConversationsByOrgFn(ctx, orgID)
	}
	m.call("ListAgentConversationsByOrg")
	return nil, nil
}
func (m *mockStore) DeleteAgentConversation(ctx context.Context, id string) error {
	if m.DeleteAgentConversationFn != nil {
		return m.DeleteAgentConversationFn(ctx, id)
	}
	m.call("DeleteAgentConversation")
	return nil
}

func (m *mockStore) CreateAgentMessage(ctx context.Context, msg *store.AgentMessage) error {
	if m.CreateAgentMessageFn != nil {
		return m.CreateAgentMessageFn(ctx, msg)
	}
	m.call("CreateAgentMessage")
	return nil
}
func (m *mockStore) ListAgentMessages(ctx context.Context, conversationID string) ([]*store.AgentMessage, error) {
	if m.ListAgentMessagesFn != nil {
		return m.ListAgentMessagesFn(ctx, conversationID)
	}
	m.call("ListAgentMessages")
	return nil, nil
}

func (m *mockStore) CreatePlaybook(ctx context.Context, pb *store.Playbook) error {
	if m.CreatePlaybookFn != nil {
		return m.CreatePlaybookFn(ctx, pb)
	}
	m.call("CreatePlaybook")
	return nil
}
func (m *mockStore) GetPlaybook(ctx context.Context, id string) (*store.Playbook, error) {
	if m.GetPlaybookFn != nil {
		return m.GetPlaybookFn(ctx, id)
	}
	m.call("GetPlaybook")
	return nil, nil
}
func (m *mockStore) ListPlaybooksByOrg(ctx context.Context, orgID string) ([]*store.Playbook, error) {
	if m.ListPlaybooksByOrgFn != nil {
		return m.ListPlaybooksByOrgFn(ctx, orgID)
	}
	m.call("ListPlaybooksByOrg")
	return nil, nil
}
func (m *mockStore) UpdatePlaybook(ctx context.Context, pb *store.Playbook) error {
	if m.UpdatePlaybookFn != nil {
		return m.UpdatePlaybookFn(ctx, pb)
	}
	m.call("UpdatePlaybook")
	return nil
}
func (m *mockStore) DeletePlaybook(ctx context.Context, id string) error {
	if m.DeletePlaybookFn != nil {
		return m.DeletePlaybookFn(ctx, id)
	}
	m.call("DeletePlaybook")
	return nil
}

func (m *mockStore) CreatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	if m.CreatePlaybookTaskFn != nil {
		return m.CreatePlaybookTaskFn(ctx, task)
	}
	m.call("CreatePlaybookTask")
	return nil
}
func (m *mockStore) GetPlaybookTask(ctx context.Context, id string) (*store.PlaybookTask, error) {
	if m.GetPlaybookTaskFn != nil {
		return m.GetPlaybookTaskFn(ctx, id)
	}
	m.call("GetPlaybookTask")
	return nil, nil
}
func (m *mockStore) ListPlaybookTasks(ctx context.Context, playbookID string) ([]*store.PlaybookTask, error) {
	if m.ListPlaybookTasksFn != nil {
		return m.ListPlaybookTasksFn(ctx, playbookID)
	}
	m.call("ListPlaybookTasks")
	return nil, nil
}
func (m *mockStore) UpdatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	if m.UpdatePlaybookTaskFn != nil {
		return m.UpdatePlaybookTaskFn(ctx, task)
	}
	m.call("UpdatePlaybookTask")
	return nil
}
func (m *mockStore) DeletePlaybookTask(ctx context.Context, id string) error {
	if m.DeletePlaybookTaskFn != nil {
		return m.DeletePlaybookTaskFn(ctx, id)
	}
	m.call("DeletePlaybookTask")
	return nil
}
func (m *mockStore) ReorderPlaybookTasks(ctx context.Context, playbookID string, taskIDs []string) error {
	if m.ReorderPlaybookTasksFn != nil {
		return m.ReorderPlaybookTasksFn(ctx, playbookID, taskIDs)
	}
	m.call("ReorderPlaybookTasks")
	return nil
}
*/

// Billing helpers
func (m *mockStore) GetOrganizationByStripeCustomerID(ctx context.Context, customerID string) (*store.Organization, error) {
	if m.GetOrganizationByStripeCustomerIDFn != nil {
		return m.GetOrganizationByStripeCustomerIDFn(ctx, customerID)
	}
	m.call("GetOrganizationByStripeCustomerID")
	return nil, nil
}
func (m *mockStore) GetModelMeter(ctx context.Context, modelID string) (*store.ModelMeter, error) {
	if m.GetModelMeterFn != nil {
		return m.GetModelMeterFn(ctx, modelID)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) CreateModelMeter(ctx context.Context, mm *store.ModelMeter) error {
	if m.CreateModelMeterFn != nil {
		return m.CreateModelMeterFn(ctx, mm)
	}
	return nil
}
func (m *mockStore) GetOrgModelSubscription(ctx context.Context, orgID, modelID string) (*store.OrgModelSubscription, error) {
	if m.GetOrgModelSubscriptionFn != nil {
		return m.GetOrgModelSubscriptionFn(ctx, orgID, modelID)
	}
	return nil, store.ErrNotFound
}
func (m *mockStore) CreateOrgModelSubscription(ctx context.Context, s *store.OrgModelSubscription) error {
	if m.CreateOrgModelSubscriptionFn != nil {
		return m.CreateOrgModelSubscriptionFn(ctx, s)
	}
	return nil
}
func (m *mockStore) SumTokenUsage(ctx context.Context, orgID string, from, to time.Time) (float64, error) {
	if m.SumTokenUsageFn != nil {
		return m.SumTokenUsageFn(ctx, orgID, from, to)
	}
	return 0, nil
}
func (m *mockStore) ListActiveSubscriptions(ctx context.Context) ([]*store.Subscription, error) {
	if m.ListActiveSubscriptionsFn != nil {
		return m.ListActiveSubscriptionsFn(ctx)
	}
	return nil, nil
}
func (m *mockStore) GetSubscriptionByStripeID(_ context.Context, _ string) (*store.Subscription, error) {
	return nil, nil
}
func (m *mockStore) AcquireAdvisoryLock(_ context.Context, _ int64) error { return nil }
func (m *mockStore) ReleaseAdvisoryLock(_ context.Context, _ int64) error { return nil }

// ---------------------------------------------------------------------------
// mockHostSender implements orchestrator.HostSender
// ---------------------------------------------------------------------------

type mockHostSender struct {
	SendAndWaitFn func(ctx context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error)
}

func (m *mockHostSender) SendAndWait(ctx context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
	if m.SendAndWaitFn != nil {
		return m.SendAndWaitFn(ctx, hostID, msg, timeout)
	}
	return nil, fmt.Errorf("mockHostSender.SendAndWait not configured")
}

// ---------------------------------------------------------------------------
// testConfig returns a Config with safe defaults for testing
// ---------------------------------------------------------------------------

func testConfig() *config.Config {
	return &config.Config{
		API: config.APIConfig{
			Addr:            ":0",
			ReadTimeout:     60 * time.Second,
			WriteTimeout:    120 * time.Second,
			IdleTimeout:     120 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
		Frontend: config.FrontendConfig{
			URL: "http://localhost:5173",
		},
		Auth: config.AuthConfig{
			SessionTTL: 720 * time.Hour,
		},
		Billing: config.BillingConfig{
			Prices: config.PriceConfig{
				SandboxMonthlyCents: 5000,
				SourceVMMonthly:     500,
				AgentHostMonthly:    1000,
			},
			FreeTier: config.FreeTierConfig{
				MaxConcurrentSandboxes: 1,
				MaxSourceVMs:           3,
				MaxAgentHosts:          1,
			},
		},
		// Agent config - commented out, not yet ready for integration.
		// Agent: config.AgentConfig{
		// 	DefaultModel:        "anthropic/claude-sonnet-4",
		// 	MaxTokensPerRequest: 8192,
		// 	FreeTokensPerMonth:  100000,
		// },
		Orchestrator: config.OrchestratorConfig{
			HeartbeatTimeout: 90 * time.Second,
			DefaultTTL:       24 * time.Hour,
		},
	}
}

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

var (
	testUser = &store.User{
		ID:          "USR-test1234",
		Email:       "test@example.com",
		DisplayName: "Test User",
	}

	testOrg = &store.Organization{
		ID:      "ORG-test1234",
		Name:    "Test Org",
		Slug:    "test-org",
		OwnerID: testUser.ID,
	}

	testMember = &store.OrgMember{
		ID:     "MBR-test1234",
		OrgID:  testOrg.ID,
		UserID: testUser.ID,
		Role:   store.OrgRoleOwner,
	}

	testSessionToken = "test-session-token-value"
)

// ---------------------------------------------------------------------------
// newTestServer creates a Server wired up with the given mock store
// ---------------------------------------------------------------------------

func newTestServer(ms *mockStore, cfg *config.Config) *Server {
	if cfg == nil {
		cfg = testConfig()
	}
	reg := registry.New()
	sender := &mockHostSender{}
	orch := orchestrator.New(reg, ms, sender, nil, cfg.Orchestrator.DefaultTTL, cfg.Orchestrator.HeartbeatTimeout)
	return NewServer(ms, cfg, orch, nil, nil)
}

// newTestServerWithSender creates a Server with a custom HostSender
func newTestServerWithSender(ms *mockStore, sender *mockHostSender, cfg *config.Config) *Server {
	if cfg == nil {
		cfg = testConfig()
	}
	reg := registry.New()
	orch := orchestrator.New(reg, ms, sender, nil, cfg.Orchestrator.DefaultTTL, cfg.Orchestrator.HeartbeatTimeout)
	return NewServer(ms, cfg, orch, nil, nil)
}

// ---------------------------------------------------------------------------
// authenticatedRequest creates an http.Request with a valid session cookie
// and configures the mock store to authenticate the test user.
// The auth middleware passes cookie.Value directly to GetSession.
// ---------------------------------------------------------------------------

func authenticatedRequest(ms *mockStore, method, path string, body *http.Request) *http.Request {
	var req *http.Request
	if body != nil {
		req = body
		req.Method = method
		req.RequestURI = path
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	req.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: testSessionToken,
	})

	if ms.GetSessionFn == nil {
		hashedToken := auth.HashSessionToken(testSessionToken)
		ms.GetSessionFn = func(_ context.Context, id string) (*store.Session, error) {
			if id == hashedToken {
				return &store.Session{
					ID:        hashedToken,
					UserID:    testUser.ID,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}, nil
			}
			return nil, store.ErrNotFound
		}
	}

	if ms.GetUserFn == nil {
		ms.GetUserFn = func(_ context.Context, id string) (*store.User, error) {
			if id == testUser.ID {
				return testUser, nil
			}
			return nil, store.ErrNotFound
		}
	}

	return req
}

// setupOrgMembership configures the mock store to resolve org by slug and membership
func setupOrgMembership(ms *mockStore) {
	if ms.GetOrganizationBySlugFn == nil {
		ms.GetOrganizationBySlugFn = func(_ context.Context, slug string) (*store.Organization, error) {
			if slug == testOrg.Slug {
				return testOrg, nil
			}
			return nil, store.ErrNotFound
		}
	}

	if ms.GetOrgMemberFn == nil {
		ms.GetOrgMemberFn = func(_ context.Context, orgID, userID string) (*store.OrgMember, error) {
			if orgID == testOrg.ID && userID == testUser.ID {
				return testMember, nil
			}
			return nil, store.ErrNotFound
		}
	}
}

// parseJSONResponse reads body into a map
func parseJSONResponse(rr *httptest.ResponseRecorder) map[string]any {
	var result map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &result)
	return result
}
