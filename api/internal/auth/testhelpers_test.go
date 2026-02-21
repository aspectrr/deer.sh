package auth

import (
	"context"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// mockStore implements store.Store for testing. Only the methods used in tests
// have real implementations; everything else panics so unexpected calls surface
// immediately.
type mockStore struct {
	// Settable hooks for the methods under test.
	getSessionFn         func(ctx context.Context, id string) (*store.Session, error)
	getUserFn            func(ctx context.Context, id string) (*store.User, error)
	getHostTokenByHashFn func(ctx context.Context, hash string) (*store.HostToken, error)
}

// ---- store.Store lifecycle methods ----

func (m *mockStore) Config() store.Config       { return store.Config{} }
func (m *mockStore) Ping(context.Context) error { return nil }
func (m *mockStore) WithTx(_ context.Context, _ func(store.DataStore) error) error {
	panic("mockStore: WithTx not implemented")
}
func (m *mockStore) Close() error { return nil }

// ---- User ----

func (m *mockStore) CreateUser(context.Context, *store.User) error {
	panic("mockStore: CreateUser not implemented")
}
func (m *mockStore) GetUser(ctx context.Context, id string) (*store.User, error) {
	if m.getUserFn != nil {
		return m.getUserFn(ctx, id)
	}
	panic("mockStore: GetUser not implemented")
}
func (m *mockStore) GetUserByEmail(context.Context, string) (*store.User, error) {
	panic("mockStore: GetUserByEmail not implemented")
}
func (m *mockStore) UpdateUser(context.Context, *store.User) error {
	panic("mockStore: UpdateUser not implemented")
}

// ---- OAuth ----

func (m *mockStore) CreateOAuthAccount(context.Context, *store.OAuthAccount) error {
	panic("mockStore: CreateOAuthAccount not implemented")
}
func (m *mockStore) GetOAuthAccount(context.Context, string, string) (*store.OAuthAccount, error) {
	panic("mockStore: GetOAuthAccount not implemented")
}
func (m *mockStore) GetOAuthAccountsByUser(context.Context, string) ([]*store.OAuthAccount, error) {
	panic("mockStore: GetOAuthAccountsByUser not implemented")
}

// ---- Session ----

func (m *mockStore) CreateSession(context.Context, *store.Session) error {
	panic("mockStore: CreateSession not implemented")
}
func (m *mockStore) GetSession(ctx context.Context, id string) (*store.Session, error) {
	if m.getSessionFn != nil {
		return m.getSessionFn(ctx, id)
	}
	panic("mockStore: GetSession not implemented")
}
func (m *mockStore) DeleteSession(context.Context, string) error {
	panic("mockStore: DeleteSession not implemented")
}
func (m *mockStore) DeleteExpiredSessions(context.Context) error {
	panic("mockStore: DeleteExpiredSessions not implemented")
}

// ---- Organization ----

func (m *mockStore) CreateOrganization(context.Context, *store.Organization) error {
	panic("mockStore: CreateOrganization not implemented")
}
func (m *mockStore) GetOrganization(context.Context, string) (*store.Organization, error) {
	panic("mockStore: GetOrganization not implemented")
}
func (m *mockStore) GetOrganizationBySlug(context.Context, string) (*store.Organization, error) {
	panic("mockStore: GetOrganizationBySlug not implemented")
}
func (m *mockStore) ListOrganizationsByUser(context.Context, string) ([]*store.Organization, error) {
	panic("mockStore: ListOrganizationsByUser not implemented")
}
func (m *mockStore) UpdateOrganization(context.Context, *store.Organization) error {
	panic("mockStore: UpdateOrganization not implemented")
}
func (m *mockStore) DeleteOrganization(context.Context, string) error {
	panic("mockStore: DeleteOrganization not implemented")
}

// ---- OrgMember ----

func (m *mockStore) CreateOrgMember(context.Context, *store.OrgMember) error {
	panic("mockStore: CreateOrgMember not implemented")
}
func (m *mockStore) GetOrgMember(context.Context, string, string) (*store.OrgMember, error) {
	panic("mockStore: GetOrgMember not implemented")
}
func (m *mockStore) GetOrgMemberByID(context.Context, string, string) (*store.OrgMember, error) {
	panic("mockStore: GetOrgMemberByID not implemented")
}
func (m *mockStore) ListOrgMembers(context.Context, string) ([]*store.OrgMember, error) {
	panic("mockStore: ListOrgMembers not implemented")
}
func (m *mockStore) DeleteOrgMember(context.Context, string, string) error {
	panic("mockStore: DeleteOrgMember not implemented")
}

// ---- Subscription ----

func (m *mockStore) CreateSubscription(context.Context, *store.Subscription) error {
	panic("mockStore: CreateSubscription not implemented")
}
func (m *mockStore) GetSubscriptionByOrg(context.Context, string) (*store.Subscription, error) {
	panic("mockStore: GetSubscriptionByOrg not implemented")
}
func (m *mockStore) UpdateSubscription(context.Context, *store.Subscription) error {
	panic("mockStore: UpdateSubscription not implemented")
}

// ---- Usage ----

func (m *mockStore) CreateUsageRecord(context.Context, *store.UsageRecord) error {
	panic("mockStore: CreateUsageRecord not implemented")
}
func (m *mockStore) ListUsageRecords(context.Context, string, time.Time, time.Time) ([]*store.UsageRecord, error) {
	panic("mockStore: ListUsageRecords not implemented")
}

// ---- Host ----

func (m *mockStore) CreateHost(context.Context, *store.Host) error {
	panic("mockStore: CreateHost not implemented")
}
func (m *mockStore) GetHost(context.Context, string) (*store.Host, error) {
	panic("mockStore: GetHost not implemented")
}
func (m *mockStore) ListHosts(context.Context) ([]store.Host, error) {
	panic("mockStore: ListHosts not implemented")
}
func (m *mockStore) ListHostsByOrg(context.Context, string) ([]store.Host, error) {
	panic("mockStore: ListHostsByOrg not implemented")
}
func (m *mockStore) UpdateHost(context.Context, *store.Host) error {
	panic("mockStore: UpdateHost not implemented")
}
func (m *mockStore) UpdateHostHeartbeat(context.Context, string, int32, int64, int64) error {
	panic("mockStore: UpdateHostHeartbeat not implemented")
}

// ---- Sandbox ----

func (m *mockStore) CreateSandbox(context.Context, *store.Sandbox) error {
	panic("mockStore: CreateSandbox not implemented")
}
func (m *mockStore) GetSandbox(context.Context, string) (*store.Sandbox, error) {
	panic("mockStore: GetSandbox not implemented")
}
func (m *mockStore) GetSandboxByOrg(context.Context, string, string) (*store.Sandbox, error) {
	panic("mockStore: GetSandboxByOrg not implemented")
}
func (m *mockStore) ListSandboxes(context.Context) ([]store.Sandbox, error) {
	panic("mockStore: ListSandboxes not implemented")
}
func (m *mockStore) ListSandboxesByOrg(context.Context, string) ([]store.Sandbox, error) {
	panic("mockStore: ListSandboxesByOrg not implemented")
}
func (m *mockStore) UpdateSandbox(context.Context, *store.Sandbox) error {
	panic("mockStore: UpdateSandbox not implemented")
}
func (m *mockStore) DeleteSandbox(context.Context, string) error {
	panic("mockStore: DeleteSandbox not implemented")
}
func (m *mockStore) GetSandboxesByHostID(context.Context, string) ([]store.Sandbox, error) {
	panic("mockStore: GetSandboxesByHostID not implemented")
}
func (m *mockStore) CountSandboxesByHostIDs(context.Context, []string) (map[string]int, error) {
	panic("mockStore: CountSandboxesByHostIDs not implemented")
}
func (m *mockStore) ListExpiredSandboxes(context.Context, time.Duration) ([]store.Sandbox, error) {
	panic("mockStore: ListExpiredSandboxes not implemented")
}

// ---- Command ----

func (m *mockStore) CreateCommand(context.Context, *store.Command) error {
	panic("mockStore: CreateCommand not implemented")
}
func (m *mockStore) ListSandboxCommands(context.Context, string) ([]store.Command, error) {
	panic("mockStore: ListSandboxCommands not implemented")
}

// ---- SourceHost ----

func (m *mockStore) CreateSourceHost(context.Context, *store.SourceHost) error {
	panic("mockStore: CreateSourceHost not implemented")
}
func (m *mockStore) GetSourceHost(context.Context, string) (*store.SourceHost, error) {
	panic("mockStore: GetSourceHost not implemented")
}
func (m *mockStore) ListSourceHostsByOrg(context.Context, string) ([]*store.SourceHost, error) {
	panic("mockStore: ListSourceHostsByOrg not implemented")
}
func (m *mockStore) DeleteSourceHost(context.Context, string) error {
	panic("mockStore: DeleteSourceHost not implemented")
}

// ---- HostToken ----

func (m *mockStore) CreateHostToken(context.Context, *store.HostToken) error {
	panic("mockStore: CreateHostToken not implemented")
}
func (m *mockStore) GetHostTokenByHash(ctx context.Context, hash string) (*store.HostToken, error) {
	if m.getHostTokenByHashFn != nil {
		return m.getHostTokenByHashFn(ctx, hash)
	}
	panic("mockStore: GetHostTokenByHash not implemented")
}
func (m *mockStore) ListHostTokensByOrg(context.Context, string) ([]store.HostToken, error) {
	panic("mockStore: ListHostTokensByOrg not implemented")
}
func (m *mockStore) DeleteHostToken(context.Context, string, string) error {
	panic("mockStore: DeleteHostToken not implemented")
}

// // ---- Agent Conversations ----

// func (m *mockStore) CreateAgentConversation(context.Context, *store.AgentConversation) error {
// 	panic("mockStore: CreateAgentConversation not implemented")
// }
// func (m *mockStore) GetAgentConversation(context.Context, string) (*store.AgentConversation, error) {
// 	panic("mockStore: GetAgentConversation not implemented")
// }
// func (m *mockStore) ListAgentConversationsByOrg(context.Context, string) ([]*store.AgentConversation, error) {
// 	panic("mockStore: ListAgentConversationsByOrg not implemented")
// }
// func (m *mockStore) DeleteAgentConversation(context.Context, string) error {
// 	panic("mockStore: DeleteAgentConversation not implemented")
// }

// // ---- Agent Messages ----

// func (m *mockStore) CreateAgentMessage(context.Context, *store.AgentMessage) error {
// 	panic("mockStore: CreateAgentMessage not implemented")
// }
// func (m *mockStore) ListAgentMessages(context.Context, string) ([]*store.AgentMessage, error) {
// 	panic("mockStore: ListAgentMessages not implemented")
// }

// // ---- Playbooks ----

// func (m *mockStore) CreatePlaybook(context.Context, *store.Playbook) error {
// 	panic("mockStore: CreatePlaybook not implemented")
// }
// func (m *mockStore) GetPlaybook(context.Context, string) (*store.Playbook, error) {
// 	panic("mockStore: GetPlaybook not implemented")
// }
// func (m *mockStore) ListPlaybooksByOrg(context.Context, string) ([]*store.Playbook, error) {
// 	panic("mockStore: ListPlaybooksByOrg not implemented")
// }
// func (m *mockStore) UpdatePlaybook(context.Context, *store.Playbook) error {
// 	panic("mockStore: UpdatePlaybook not implemented")
// }
// func (m *mockStore) DeletePlaybook(context.Context, string) error {
// 	panic("mockStore: DeletePlaybook not implemented")
// }

// // ---- Playbook Tasks ----

// func (m *mockStore) CreatePlaybookTask(context.Context, *store.PlaybookTask) error {
// 	panic("mockStore: CreatePlaybookTask not implemented")
// }
// func (m *mockStore) GetPlaybookTask(context.Context, string) (*store.PlaybookTask, error) {
// 	panic("mockStore: GetPlaybookTask not implemented")
// }
// func (m *mockStore) ListPlaybookTasks(context.Context, string) ([]*store.PlaybookTask, error) {
// 	panic("mockStore: ListPlaybookTasks not implemented")
// }
// func (m *mockStore) UpdatePlaybookTask(context.Context, *store.PlaybookTask) error {
// 	panic("mockStore: UpdatePlaybookTask not implemented")
// }
// func (m *mockStore) DeletePlaybookTask(context.Context, string) error {
// 	panic("mockStore: DeletePlaybookTask not implemented")
// }
// func (m *mockStore) ReorderPlaybookTasks(context.Context, string, []string) error {
// 	panic("mockStore: ReorderPlaybookTasks not implemented")
// }

// ---- Billing helpers ----

func (m *mockStore) GetOrganizationByStripeCustomerID(context.Context, string) (*store.Organization, error) {
	panic("mockStore: GetOrganizationByStripeCustomerID not implemented")
}
func (m *mockStore) GetModelMeter(context.Context, string) (*store.ModelMeter, error) {
	panic("mockStore: GetModelMeter not implemented")
}
func (m *mockStore) CreateModelMeter(context.Context, *store.ModelMeter) error {
	panic("mockStore: CreateModelMeter not implemented")
}
func (m *mockStore) GetOrgModelSubscription(context.Context, string, string) (*store.OrgModelSubscription, error) {
	panic("mockStore: GetOrgModelSubscription not implemented")
}
func (m *mockStore) CreateOrgModelSubscription(context.Context, *store.OrgModelSubscription) error {
	panic("mockStore: CreateOrgModelSubscription not implemented")
}
func (m *mockStore) SumTokenUsage(context.Context, string, time.Time, time.Time) (float64, error) {
	panic("mockStore: SumTokenUsage not implemented")
}
func (m *mockStore) ListActiveSubscriptions(context.Context) ([]*store.Subscription, error) {
	panic("mockStore: ListActiveSubscriptions not implemented")
}
func (m *mockStore) GetSubscriptionByStripeID(context.Context, string) (*store.Subscription, error) {
	panic("mockStore: GetSubscriptionByStripeID not implemented")
}
func (m *mockStore) AcquireAdvisoryLock(context.Context, int64) error {
	panic("mockStore: AcquireAdvisoryLock not implemented")
}
func (m *mockStore) ReleaseAdvisoryLock(context.Context, int64) error {
	panic("mockStore: ReleaseAdvisoryLock not implemented")
}
