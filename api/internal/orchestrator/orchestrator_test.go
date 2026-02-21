package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// ---------------------------------------------------------------------------
// mockStore implements store.Store with function-field delegation
// ---------------------------------------------------------------------------

type mockStore struct {
	CreateUserFn     func(ctx context.Context, u *store.User) error
	GetUserFn        func(ctx context.Context, id string) (*store.User, error)
	GetUserByEmailFn func(ctx context.Context, email string) (*store.User, error)
	UpdateUserFn     func(ctx context.Context, u *store.User) error

	CreateOAuthAccountFn     func(ctx context.Context, oa *store.OAuthAccount) error
	GetOAuthAccountFn        func(ctx context.Context, provider, providerID string) (*store.OAuthAccount, error)
	GetOAuthAccountsByUserFn func(ctx context.Context, userID string) ([]*store.OAuthAccount, error)

	CreateSessionFn         func(ctx context.Context, s *store.Session) error
	GetSessionFn            func(ctx context.Context, id string) (*store.Session, error)
	DeleteSessionFn         func(ctx context.Context, id string) error
	DeleteExpiredSessionsFn func(ctx context.Context) error

	CreateOrganizationFn      func(ctx context.Context, org *store.Organization) error
	GetOrganizationFn         func(ctx context.Context, id string) (*store.Organization, error)
	GetOrganizationBySlugFn   func(ctx context.Context, slug string) (*store.Organization, error)
	ListOrganizationsByUserFn func(ctx context.Context, userID string) ([]*store.Organization, error)
	UpdateOrganizationFn      func(ctx context.Context, org *store.Organization) error
	DeleteOrganizationFn      func(ctx context.Context, id string) error

	CreateOrgMemberFn func(ctx context.Context, m *store.OrgMember) error
	GetOrgMemberFn    func(ctx context.Context, orgID, userID string) (*store.OrgMember, error)
	ListOrgMembersFn  func(ctx context.Context, orgID string) ([]*store.OrgMember, error)
	DeleteOrgMemberFn func(ctx context.Context, orgID, id string) error

	CreateSubscriptionFn   func(ctx context.Context, sub *store.Subscription) error
	GetSubscriptionByOrgFn func(ctx context.Context, orgID string) (*store.Subscription, error)
	UpdateSubscriptionFn   func(ctx context.Context, sub *store.Subscription) error

	CreateUsageRecordFn func(ctx context.Context, rec *store.UsageRecord) error
	ListUsageRecordsFn  func(ctx context.Context, orgID string, from, to time.Time) ([]*store.UsageRecord, error)

	CreateHostFn          func(ctx context.Context, host *store.Host) error
	GetHostFn             func(ctx context.Context, hostID string) (*store.Host, error)
	ListHostsFn           func(ctx context.Context) ([]store.Host, error)
	UpdateHostFn          func(ctx context.Context, host *store.Host) error
	UpdateHostHeartbeatFn func(ctx context.Context, hostID string, availCPUs int32, availMemMB int64, availDiskMB int64) error

	CreateSandboxFn           func(ctx context.Context, sandbox *store.Sandbox) error
	GetSandboxFn              func(ctx context.Context, sandboxID string) (*store.Sandbox, error)
	GetSandboxByOrgFn         func(ctx context.Context, orgID, sandboxID string) (*store.Sandbox, error)
	ListSandboxesFn           func(ctx context.Context) ([]store.Sandbox, error)
	ListSandboxesByOrgFn      func(ctx context.Context, orgID string) ([]store.Sandbox, error)
	UpdateSandboxFn           func(ctx context.Context, sandbox *store.Sandbox) error
	DeleteSandboxFn           func(ctx context.Context, sandboxID string) error
	GetSandboxesByHostIDFn    func(ctx context.Context, hostID string) ([]store.Sandbox, error)
	CountSandboxesByHostIDsFn func(ctx context.Context, hostIDs []string) (map[string]int, error)
	ListExpiredSandboxesFn    func(ctx context.Context, defaultTTL time.Duration) ([]store.Sandbox, error)

	CreateCommandFn       func(ctx context.Context, cmd *store.Command) error
	ListSandboxCommandsFn func(ctx context.Context, sandboxID string) ([]store.Command, error)

	CreateSourceHostFn     func(ctx context.Context, sh *store.SourceHost) error
	GetSourceHostFn        func(ctx context.Context, id string) (*store.SourceHost, error)
	ListSourceHostsByOrgFn func(ctx context.Context, orgID string) ([]*store.SourceHost, error)
	DeleteSourceHostFn     func(ctx context.Context, id string) error

	CreateHostTokenFn     func(ctx context.Context, token *store.HostToken) error
	GetHostTokenByHashFn  func(ctx context.Context, hash string) (*store.HostToken, error)
	ListHostTokensByOrgFn func(ctx context.Context, orgID string) ([]store.HostToken, error)
	DeleteHostTokenFn     func(ctx context.Context, orgID, id string) error

	// Agent/playbook fields removed - interface methods commented out in store.go

	GetOrganizationByStripeCustomerIDFn func(ctx context.Context, customerID string) (*store.Organization, error)
	GetModelMeterFn                     func(ctx context.Context, modelID string) (*store.ModelMeter, error)
	CreateModelMeterFn                  func(ctx context.Context, m *store.ModelMeter) error
	GetOrgModelSubscriptionFn           func(ctx context.Context, orgID, modelID string) (*store.OrgModelSubscription, error)
	CreateOrgModelSubscriptionFn        func(ctx context.Context, s *store.OrgModelSubscription) error
	SumTokenUsageFn                     func(ctx context.Context, orgID string, from, to time.Time) (float64, error)
	ListActiveSubscriptionsFn           func(ctx context.Context) ([]*store.Subscription, error)

	WithTxFn func(ctx context.Context, fn func(tx store.DataStore) error) error
}

func (m *mockStore) p(name string) { panic(fmt.Sprintf("mockStore.%s not configured", name)) }

func (m *mockStore) Config() store.Config       { return store.Config{} }
func (m *mockStore) Ping(context.Context) error { return nil }
func (m *mockStore) Close() error               { return nil }

func (m *mockStore) WithTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	if m.WithTxFn != nil {
		return m.WithTxFn(ctx, fn)
	}
	return fn(m)
}

func (m *mockStore) CreateUser(ctx context.Context, u *store.User) error {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, u)
	}
	m.p("CreateUser")
	return nil
}
func (m *mockStore) GetUser(ctx context.Context, id string) (*store.User, error) {
	if m.GetUserFn != nil {
		return m.GetUserFn(ctx, id)
	}
	m.p("GetUser")
	return nil, nil
}
func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(ctx, email)
	}
	m.p("GetUserByEmail")
	return nil, nil
}
func (m *mockStore) UpdateUser(ctx context.Context, u *store.User) error {
	if m.UpdateUserFn != nil {
		return m.UpdateUserFn(ctx, u)
	}
	m.p("UpdateUser")
	return nil
}

func (m *mockStore) CreateOAuthAccount(ctx context.Context, oa *store.OAuthAccount) error {
	if m.CreateOAuthAccountFn != nil {
		return m.CreateOAuthAccountFn(ctx, oa)
	}
	m.p("CreateOAuthAccount")
	return nil
}
func (m *mockStore) GetOAuthAccount(ctx context.Context, provider, providerID string) (*store.OAuthAccount, error) {
	if m.GetOAuthAccountFn != nil {
		return m.GetOAuthAccountFn(ctx, provider, providerID)
	}
	m.p("GetOAuthAccount")
	return nil, nil
}
func (m *mockStore) GetOAuthAccountsByUser(ctx context.Context, userID string) ([]*store.OAuthAccount, error) {
	if m.GetOAuthAccountsByUserFn != nil {
		return m.GetOAuthAccountsByUserFn(ctx, userID)
	}
	m.p("GetOAuthAccountsByUser")
	return nil, nil
}

func (m *mockStore) CreateSession(ctx context.Context, s *store.Session) error {
	if m.CreateSessionFn != nil {
		return m.CreateSessionFn(ctx, s)
	}
	m.p("CreateSession")
	return nil
}
func (m *mockStore) GetSession(ctx context.Context, id string) (*store.Session, error) {
	if m.GetSessionFn != nil {
		return m.GetSessionFn(ctx, id)
	}
	m.p("GetSession")
	return nil, nil
}
func (m *mockStore) DeleteSession(ctx context.Context, id string) error {
	if m.DeleteSessionFn != nil {
		return m.DeleteSessionFn(ctx, id)
	}
	m.p("DeleteSession")
	return nil
}
func (m *mockStore) DeleteExpiredSessions(ctx context.Context) error {
	if m.DeleteExpiredSessionsFn != nil {
		return m.DeleteExpiredSessionsFn(ctx)
	}
	m.p("DeleteExpiredSessions")
	return nil
}

func (m *mockStore) CreateOrganization(ctx context.Context, org *store.Organization) error {
	if m.CreateOrganizationFn != nil {
		return m.CreateOrganizationFn(ctx, org)
	}
	m.p("CreateOrganization")
	return nil
}
func (m *mockStore) GetOrganization(ctx context.Context, id string) (*store.Organization, error) {
	if m.GetOrganizationFn != nil {
		return m.GetOrganizationFn(ctx, id)
	}
	m.p("GetOrganization")
	return nil, nil
}
func (m *mockStore) GetOrganizationBySlug(ctx context.Context, slug string) (*store.Organization, error) {
	if m.GetOrganizationBySlugFn != nil {
		return m.GetOrganizationBySlugFn(ctx, slug)
	}
	m.p("GetOrganizationBySlug")
	return nil, nil
}
func (m *mockStore) ListOrganizationsByUser(ctx context.Context, userID string) ([]*store.Organization, error) {
	if m.ListOrganizationsByUserFn != nil {
		return m.ListOrganizationsByUserFn(ctx, userID)
	}
	m.p("ListOrganizationsByUser")
	return nil, nil
}
func (m *mockStore) UpdateOrganization(ctx context.Context, org *store.Organization) error {
	if m.UpdateOrganizationFn != nil {
		return m.UpdateOrganizationFn(ctx, org)
	}
	m.p("UpdateOrganization")
	return nil
}
func (m *mockStore) DeleteOrganization(ctx context.Context, id string) error {
	if m.DeleteOrganizationFn != nil {
		return m.DeleteOrganizationFn(ctx, id)
	}
	m.p("DeleteOrganization")
	return nil
}

func (m *mockStore) CreateOrgMember(ctx context.Context, mem *store.OrgMember) error {
	if m.CreateOrgMemberFn != nil {
		return m.CreateOrgMemberFn(ctx, mem)
	}
	m.p("CreateOrgMember")
	return nil
}
func (m *mockStore) GetOrgMember(ctx context.Context, orgID, userID string) (*store.OrgMember, error) {
	if m.GetOrgMemberFn != nil {
		return m.GetOrgMemberFn(ctx, orgID, userID)
	}
	m.p("GetOrgMember")
	return nil, nil
}
func (m *mockStore) GetOrgMemberByID(ctx context.Context, orgID, memberID string) (*store.OrgMember, error) {
	m.p("GetOrgMemberByID")
	return nil, nil
}
func (m *mockStore) ListOrgMembers(ctx context.Context, orgID string) ([]*store.OrgMember, error) {
	if m.ListOrgMembersFn != nil {
		return m.ListOrgMembersFn(ctx, orgID)
	}
	m.p("ListOrgMembers")
	return nil, nil
}
func (m *mockStore) DeleteOrgMember(ctx context.Context, orgID, id string) error {
	if m.DeleteOrgMemberFn != nil {
		return m.DeleteOrgMemberFn(ctx, orgID, id)
	}
	m.p("DeleteOrgMember")
	return nil
}

func (m *mockStore) CreateSubscription(ctx context.Context, sub *store.Subscription) error {
	if m.CreateSubscriptionFn != nil {
		return m.CreateSubscriptionFn(ctx, sub)
	}
	m.p("CreateSubscription")
	return nil
}
func (m *mockStore) GetSubscriptionByOrg(ctx context.Context, orgID string) (*store.Subscription, error) {
	if m.GetSubscriptionByOrgFn != nil {
		return m.GetSubscriptionByOrgFn(ctx, orgID)
	}
	m.p("GetSubscriptionByOrg")
	return nil, nil
}
func (m *mockStore) UpdateSubscription(ctx context.Context, sub *store.Subscription) error {
	if m.UpdateSubscriptionFn != nil {
		return m.UpdateSubscriptionFn(ctx, sub)
	}
	m.p("UpdateSubscription")
	return nil
}

func (m *mockStore) CreateUsageRecord(ctx context.Context, rec *store.UsageRecord) error {
	if m.CreateUsageRecordFn != nil {
		return m.CreateUsageRecordFn(ctx, rec)
	}
	m.p("CreateUsageRecord")
	return nil
}
func (m *mockStore) ListUsageRecords(ctx context.Context, orgID string, from, to time.Time) ([]*store.UsageRecord, error) {
	if m.ListUsageRecordsFn != nil {
		return m.ListUsageRecordsFn(ctx, orgID, from, to)
	}
	m.p("ListUsageRecords")
	return nil, nil
}

func (m *mockStore) CreateHost(ctx context.Context, host *store.Host) error {
	if m.CreateHostFn != nil {
		return m.CreateHostFn(ctx, host)
	}
	m.p("CreateHost")
	return nil
}
func (m *mockStore) GetHost(ctx context.Context, hostID string) (*store.Host, error) {
	if m.GetHostFn != nil {
		return m.GetHostFn(ctx, hostID)
	}
	m.p("GetHost")
	return nil, nil
}
func (m *mockStore) ListHosts(ctx context.Context) ([]store.Host, error) {
	if m.ListHostsFn != nil {
		return m.ListHostsFn(ctx)
	}
	m.p("ListHosts")
	return nil, nil
}
func (m *mockStore) ListHostsByOrg(_ context.Context, _ string) ([]store.Host, error) {
	return nil, nil
}
func (m *mockStore) UpdateHost(ctx context.Context, host *store.Host) error {
	if m.UpdateHostFn != nil {
		return m.UpdateHostFn(ctx, host)
	}
	m.p("UpdateHost")
	return nil
}
func (m *mockStore) UpdateHostHeartbeat(ctx context.Context, hostID string, availCPUs int32, availMemMB int64, availDiskMB int64) error {
	if m.UpdateHostHeartbeatFn != nil {
		return m.UpdateHostHeartbeatFn(ctx, hostID, availCPUs, availMemMB, availDiskMB)
	}
	m.p("UpdateHostHeartbeat")
	return nil
}

func (m *mockStore) CreateSandbox(ctx context.Context, sandbox *store.Sandbox) error {
	if m.CreateSandboxFn != nil {
		return m.CreateSandboxFn(ctx, sandbox)
	}
	m.p("CreateSandbox")
	return nil
}
func (m *mockStore) GetSandbox(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
	if m.GetSandboxFn != nil {
		return m.GetSandboxFn(ctx, sandboxID)
	}
	m.p("GetSandbox")
	return nil, nil
}
func (m *mockStore) GetSandboxByOrg(ctx context.Context, orgID, sandboxID string) (*store.Sandbox, error) {
	if m.GetSandboxByOrgFn != nil {
		return m.GetSandboxByOrgFn(ctx, orgID, sandboxID)
	}
	// Fall back to GetSandboxFn for backward compat in tests.
	if m.GetSandboxFn != nil {
		return m.GetSandboxFn(ctx, sandboxID)
	}
	m.p("GetSandboxByOrg")
	return nil, nil
}
func (m *mockStore) ListSandboxes(ctx context.Context) ([]store.Sandbox, error) {
	if m.ListSandboxesFn != nil {
		return m.ListSandboxesFn(ctx)
	}
	m.p("ListSandboxes")
	return nil, nil
}
func (m *mockStore) ListSandboxesByOrg(ctx context.Context, orgID string) ([]store.Sandbox, error) {
	if m.ListSandboxesByOrgFn != nil {
		return m.ListSandboxesByOrgFn(ctx, orgID)
	}
	m.p("ListSandboxesByOrg")
	return nil, nil
}
func (m *mockStore) UpdateSandbox(ctx context.Context, sandbox *store.Sandbox) error {
	if m.UpdateSandboxFn != nil {
		return m.UpdateSandboxFn(ctx, sandbox)
	}
	m.p("UpdateSandbox")
	return nil
}
func (m *mockStore) DeleteSandbox(ctx context.Context, sandboxID string) error {
	if m.DeleteSandboxFn != nil {
		return m.DeleteSandboxFn(ctx, sandboxID)
	}
	m.p("DeleteSandbox")
	return nil
}
func (m *mockStore) GetSandboxesByHostID(ctx context.Context, hostID string) ([]store.Sandbox, error) {
	if m.GetSandboxesByHostIDFn != nil {
		return m.GetSandboxesByHostIDFn(ctx, hostID)
	}
	m.p("GetSandboxesByHostID")
	return nil, nil
}
func (m *mockStore) CountSandboxesByHostIDs(ctx context.Context, hostIDs []string) (map[string]int, error) {
	if m.CountSandboxesByHostIDsFn != nil {
		return m.CountSandboxesByHostIDsFn(ctx, hostIDs)
	}
	return map[string]int{}, nil
}
func (m *mockStore) ListExpiredSandboxes(ctx context.Context, defaultTTL time.Duration) ([]store.Sandbox, error) {
	if m.ListExpiredSandboxesFn != nil {
		return m.ListExpiredSandboxesFn(ctx, defaultTTL)
	}
	m.p("ListExpiredSandboxes")
	return nil, nil
}

func (m *mockStore) CreateCommand(ctx context.Context, cmd *store.Command) error {
	if m.CreateCommandFn != nil {
		return m.CreateCommandFn(ctx, cmd)
	}
	m.p("CreateCommand")
	return nil
}
func (m *mockStore) ListSandboxCommands(ctx context.Context, sandboxID string) ([]store.Command, error) {
	if m.ListSandboxCommandsFn != nil {
		return m.ListSandboxCommandsFn(ctx, sandboxID)
	}
	m.p("ListSandboxCommands")
	return nil, nil
}

func (m *mockStore) CreateSourceHost(ctx context.Context, sh *store.SourceHost) error {
	if m.CreateSourceHostFn != nil {
		return m.CreateSourceHostFn(ctx, sh)
	}
	m.p("CreateSourceHost")
	return nil
}
func (m *mockStore) GetSourceHost(ctx context.Context, id string) (*store.SourceHost, error) {
	if m.GetSourceHostFn != nil {
		return m.GetSourceHostFn(ctx, id)
	}
	m.p("GetSourceHost")
	return nil, nil
}
func (m *mockStore) ListSourceHostsByOrg(ctx context.Context, orgID string) ([]*store.SourceHost, error) {
	if m.ListSourceHostsByOrgFn != nil {
		return m.ListSourceHostsByOrgFn(ctx, orgID)
	}
	m.p("ListSourceHostsByOrg")
	return nil, nil
}
func (m *mockStore) DeleteSourceHost(ctx context.Context, id string) error {
	if m.DeleteSourceHostFn != nil {
		return m.DeleteSourceHostFn(ctx, id)
	}
	m.p("DeleteSourceHost")
	return nil
}

func (m *mockStore) CreateHostToken(ctx context.Context, token *store.HostToken) error {
	if m.CreateHostTokenFn != nil {
		return m.CreateHostTokenFn(ctx, token)
	}
	m.p("CreateHostToken")
	return nil
}
func (m *mockStore) GetHostTokenByHash(ctx context.Context, hash string) (*store.HostToken, error) {
	if m.GetHostTokenByHashFn != nil {
		return m.GetHostTokenByHashFn(ctx, hash)
	}
	m.p("GetHostTokenByHash")
	return nil, nil
}
func (m *mockStore) ListHostTokensByOrg(ctx context.Context, orgID string) ([]store.HostToken, error) {
	if m.ListHostTokensByOrgFn != nil {
		return m.ListHostTokensByOrgFn(ctx, orgID)
	}
	m.p("ListHostTokensByOrg")
	return nil, nil
}
func (m *mockStore) DeleteHostToken(ctx context.Context, orgID, id string) error {
	if m.DeleteHostTokenFn != nil {
		return m.DeleteHostTokenFn(ctx, orgID, id)
	}
	m.p("DeleteHostToken")
	return nil
}

// Agent/playbook mock methods removed - interface methods commented out in store.go

func (m *mockStore) GetOrganizationByStripeCustomerID(ctx context.Context, customerID string) (*store.Organization, error) {
	if m.GetOrganizationByStripeCustomerIDFn != nil {
		return m.GetOrganizationByStripeCustomerIDFn(ctx, customerID)
	}
	m.p("GetOrganizationByStripeCustomerID")
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
// mockSender implements HostSender
// ---------------------------------------------------------------------------

type mockSender struct {
	SendAndWaitFn func(ctx context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error)
}

func (m *mockSender) SendAndWait(ctx context.Context, hostID string, msg *fluidv1.ControlMessage, timeout time.Duration) (*fluidv1.HostMessage, error) {
	if m.SendAndWaitFn != nil {
		return m.SendAndWaitFn(ctx, hostID, msg, timeout)
	}
	return nil, fmt.Errorf("mockSender.SendAndWait not configured")
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestOrchestrator(ms *mockStore, sender *mockSender) *Orchestrator {
	reg := registry.New()
	return New(reg, ms, sender, nil, 24*time.Hour, 90*time.Second)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestListSandboxesByOrg_Success(t *testing.T) {
	ms := &mockStore{
		ListSandboxesByOrgFn: func(_ context.Context, orgID string) ([]store.Sandbox, error) {
			if orgID != "org-1" {
				t.Errorf("orgID = %q, want %q", orgID, "org-1")
			}
			return []store.Sandbox{
				{ID: "sbx-1", OrgID: "org-1", Name: "sandbox-1", State: store.SandboxStateRunning},
				{ID: "sbx-2", OrgID: "org-1", Name: "sandbox-2", State: store.SandboxStateStopped},
			}, nil
		},
	}

	orch := newTestOrchestrator(ms, &mockSender{})
	result, err := orch.ListSandboxesByOrg(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("ListSandboxesByOrg: unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListSandboxesByOrg: got %d sandboxes, want 2", len(result))
	}
	if result[0].ID != "sbx-1" {
		t.Errorf("result[0].ID = %q, want %q", result[0].ID, "sbx-1")
	}
	if result[1].ID != "sbx-2" {
		t.Errorf("result[1].ID = %q, want %q", result[1].ID, "sbx-2")
	}
}

func TestListSandboxesByOrg_Empty(t *testing.T) {
	ms := &mockStore{
		ListSandboxesByOrgFn: func(_ context.Context, _ string) ([]store.Sandbox, error) {
			return []store.Sandbox{}, nil
		},
	}

	orch := newTestOrchestrator(ms, &mockSender{})
	result, err := orch.ListSandboxesByOrg(context.Background(), "org-empty")
	if err != nil {
		t.Fatalf("ListSandboxesByOrg: unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("ListSandboxesByOrg: got %d sandboxes, want 0", len(result))
	}
}

func TestListSandboxesByOrg_StoreError(t *testing.T) {
	ms := &mockStore{
		ListSandboxesByOrgFn: func(_ context.Context, _ string) ([]store.Sandbox, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	orch := newTestOrchestrator(ms, &mockSender{})
	_, err := orch.ListSandboxesByOrg(context.Background(), "org-1")
	if err == nil {
		t.Fatal("ListSandboxesByOrg: expected error from store")
	}
}

func TestGetSandbox_Success(t *testing.T) {
	expected := &store.Sandbox{
		ID:    "sbx-1",
		OrgID: "org-1",
		Name:  "my-sandbox",
		State: store.SandboxStateRunning,
	}
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			if id == "sbx-1" {
				return expected, nil
			}
			return nil, store.ErrNotFound
		},
	}

	orch := newTestOrchestrator(ms, &mockSender{})
	result, err := orch.GetSandbox(context.Background(), "org-1", "sbx-1")
	if err != nil {
		t.Fatalf("GetSandbox: unexpected error: %v", err)
	}
	if result.ID != expected.ID {
		t.Errorf("ID = %q, want %q", result.ID, expected.ID)
	}
	if result.Name != expected.Name {
		t.Errorf("Name = %q, want %q", result.Name, expected.Name)
	}
}

func TestGetSandbox_NotFound(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, _ string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		},
	}

	orch := newTestOrchestrator(ms, &mockSender{})
	_, err := orch.GetSandbox(context.Background(), "org-1", "nonexistent")
	if err == nil {
		t.Fatal("GetSandbox: expected error for nonexistent sandbox")
	}
}

func TestListCommands_Success(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, OrgID: "org-1"}, nil
		},
		ListSandboxCommandsFn: func(_ context.Context, sandboxID string) ([]store.Command, error) {
			if sandboxID != "sbx-1" {
				t.Errorf("sandboxID = %q, want %q", sandboxID, "sbx-1")
			}
			return []store.Command{
				{ID: "cmd-1", SandboxID: "sbx-1", Command: "ls -la", ExitCode: 0},
				{ID: "cmd-2", SandboxID: "sbx-1", Command: "pwd", ExitCode: 0},
			}, nil
		},
	}

	orch := newTestOrchestrator(ms, &mockSender{})
	result, err := orch.ListCommands(context.Background(), "org-1", "sbx-1")
	if err != nil {
		t.Fatalf("ListCommands: unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListCommands: got %d commands, want 2", len(result))
	}
	if result[0].Command != "ls -la" {
		t.Errorf("result[0].Command = %q, want %q", result[0].Command, "ls -la")
	}
}

func TestListCommands_StoreError(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, OrgID: "org-1"}, nil
		},
		ListSandboxCommandsFn: func(_ context.Context, _ string) ([]store.Command, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	orch := newTestOrchestrator(ms, &mockSender{})
	_, err := orch.ListCommands(context.Background(), "org-1", "sbx-1")
	if err == nil {
		t.Fatal("ListCommands: expected error from store")
	}
}

func TestListHosts_Success(t *testing.T) {
	reg := registry.New()
	_ = reg.Register("host-1", "org-1", "production-1", &mockStream{})
	reg.SetRegistration("host-1", &fluidv1.HostRegistration{
		AvailableCpus:     16,
		AvailableMemoryMb: 32768,
		AvailableDiskMb:   512000,
		BaseImages:        []string{"ubuntu-22.04"},
	})

	ms := &mockStore{
		CountSandboxesByHostIDsFn: func(_ context.Context, hostIDs []string) (map[string]int, error) {
			counts := map[string]int{}
			for _, id := range hostIDs {
				if id == "host-1" {
					counts[id] = 2
				}
			}
			return counts, nil
		},
	}

	sender := &mockSender{}
	orch := New(reg, ms, sender, nil, 24*time.Hour, 90*time.Second)

	result, err := orch.ListHosts(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("ListHosts: unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("ListHosts: got %d hosts, want 1", len(result))
	}

	h := result[0]
	if h.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", h.HostID, "host-1")
	}
	if h.Hostname != "production-1" {
		t.Errorf("Hostname = %q, want %q", h.Hostname, "production-1")
	}
	if h.Status != "ONLINE" {
		t.Errorf("Status = %q, want %q", h.Status, "ONLINE")
	}
	if h.AvailableCPUs != 16 {
		t.Errorf("AvailableCPUs = %d, want 16", h.AvailableCPUs)
	}
	if h.AvailableMemMB != 32768 {
		t.Errorf("AvailableMemMB = %d, want 32768", h.AvailableMemMB)
	}
	if h.ActiveSandboxes != 2 {
		t.Errorf("ActiveSandboxes = %d, want 2", h.ActiveSandboxes)
	}
	if len(h.BaseImages) != 1 || h.BaseImages[0] != "ubuntu-22.04" {
		t.Errorf("BaseImages = %v, want [ubuntu-22.04]", h.BaseImages)
	}
}

func TestListHosts_NoHosts(t *testing.T) {
	reg := registry.New()
	ms := &mockStore{}
	sender := &mockSender{}
	orch := New(reg, ms, sender, nil, 24*time.Hour, 90*time.Second)

	result, err := orch.ListHosts(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("ListHosts: unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("ListHosts: got %d hosts, want 0", len(result))
	}
}

func TestListHosts_FiltersByOrg(t *testing.T) {
	reg := registry.New()
	_ = reg.Register("host-1", "org-1", "h1", &mockStream{})
	reg.SetRegistration("host-1", &fluidv1.HostRegistration{})
	_ = reg.Register("host-2", "org-2", "h2", &mockStream{})
	reg.SetRegistration("host-2", &fluidv1.HostRegistration{})

	ms := &mockStore{}

	sender := &mockSender{}
	orch := New(reg, ms, sender, nil, 24*time.Hour, 90*time.Second)

	result, err := orch.ListHosts(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("ListHosts: unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("ListHosts: got %d hosts, want 1", len(result))
	}
	if result[0].HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", result[0].HostID, "host-1")
	}
}

// ---------------------------------------------------------------------------
// Write operation tests
// ---------------------------------------------------------------------------

func TestCreateSandbox_Success(t *testing.T) {
	reg := newRegistryWithHost(t, "host-1", "org-1", &fluidv1.HostRegistration{
		AvailableCpus:     16,
		AvailableMemoryMb: 32768,
		BaseImages:        []string{"ubuntu-22.04"},
	})

	var storedSandbox *store.Sandbox
	ms := &mockStore{
		CreateSandboxFn: func(_ context.Context, s *store.Sandbox) error {
			storedSandbox = s
			return nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			if hostID != "host-1" {
				t.Errorf("hostID = %q, want %q", hostID, "host-1")
			}
			return &fluidv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &fluidv1.HostMessage_SandboxCreated{
					SandboxCreated: &fluidv1.SandboxCreated{
						SandboxId:  msg.GetCreateSandbox().GetSandboxId(),
						Name:       "my-sandbox",
						State:      "RUNNING",
						IpAddress:  "10.0.0.5",
						Bridge:     "br0",
						MacAddress: "aa:bb:cc:dd:ee:ff",
					},
				},
			}, nil
		},
	}

	orch := New(reg, ms, sender, nil, 24*time.Hour, 90*time.Second)
	result, err := orch.CreateSandbox(context.Background(), CreateSandboxRequest{
		OrgID:    "org-1",
		SourceVM: "ubuntu-22.04",
		Name:     "my-sandbox",
		VCPUs:    4,
		MemoryMB: 4096,
	})
	if err != nil {
		t.Fatalf("CreateSandbox: unexpected error: %v", err)
	}
	if result.OrgID != "org-1" {
		t.Errorf("OrgID = %q, want %q", result.OrgID, "org-1")
	}
	if result.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", result.HostID, "host-1")
	}
	if result.Name != "my-sandbox" {
		t.Errorf("Name = %q, want %q", result.Name, "my-sandbox")
	}
	if result.State != store.SandboxStateRunning {
		t.Errorf("State = %q, want %q", result.State, store.SandboxStateRunning)
	}
	if result.IPAddress != "10.0.0.5" {
		t.Errorf("IPAddress = %q, want %q", result.IPAddress, "10.0.0.5")
	}
	if result.Bridge != "br0" {
		t.Errorf("Bridge = %q, want %q", result.Bridge, "br0")
	}
	if result.MACAddress != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MACAddress = %q, want %q", result.MACAddress, "aa:bb:cc:dd:ee:ff")
	}
	if result.VCPUs != 4 {
		t.Errorf("VCPUs = %d, want 4", result.VCPUs)
	}
	if result.MemoryMB != 4096 {
		t.Errorf("MemoryMB = %d, want 4096", result.MemoryMB)
	}
	if storedSandbox == nil {
		t.Fatal("CreateSandboxFn was not called")
	}
	if storedSandbox.ID != result.ID {
		t.Errorf("stored ID = %q, want %q", storedSandbox.ID, result.ID)
	}
}

func TestCreateSandbox_NoHost(t *testing.T) {
	ms := &mockStore{}
	sender := &mockSender{}
	orch := newTestOrchestrator(ms, sender)

	_, err := orch.CreateSandbox(context.Background(), CreateSandboxRequest{
		OrgID:    "org-1",
		SourceVM: "ubuntu-22.04",
	})
	if err == nil {
		t.Fatal("CreateSandbox: expected error when no hosts available")
	}
	if !strings.Contains(err.Error(), "no connected hosts") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "no connected hosts")
	}
}

func TestCreateSandbox_SenderError(t *testing.T) {
	reg := newRegistryWithHost(t, "host-1", "org-1", &fluidv1.HostRegistration{
		AvailableCpus:     16,
		AvailableMemoryMb: 32768,
		BaseImages:        []string{"ubuntu-22.04"},
	})

	ms := &mockStore{}
	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, _ string, _ *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			return nil, fmt.Errorf("connection lost")
		},
	}

	orch := New(reg, ms, sender, nil, 24*time.Hour, 90*time.Second)
	_, err := orch.CreateSandbox(context.Background(), CreateSandboxRequest{
		OrgID:    "org-1",
		SourceVM: "ubuntu-22.04",
	})
	if err == nil {
		t.Fatal("CreateSandbox: expected error from sender")
	}
	if !strings.Contains(err.Error(), "connection lost") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "connection lost")
	}
}

func TestCreateSandbox_HostError(t *testing.T) {
	reg := newRegistryWithHost(t, "host-1", "org-1", &fluidv1.HostRegistration{
		AvailableCpus:     16,
		AvailableMemoryMb: 32768,
		BaseImages:        []string{"ubuntu-22.04"},
	})

	ms := &mockStore{}
	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, _ string, msg *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			return &fluidv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &fluidv1.HostMessage_ErrorReport{
					ErrorReport: &fluidv1.ErrorReport{Error: "disk full"},
				},
			}, nil
		},
	}

	orch := New(reg, ms, sender, nil, 24*time.Hour, 90*time.Second)
	_, err := orch.CreateSandbox(context.Background(), CreateSandboxRequest{
		OrgID:    "org-1",
		SourceVM: "ubuntu-22.04",
	})
	if err == nil {
		t.Fatal("CreateSandbox: expected error from host error report")
	}
	if !strings.Contains(err.Error(), "host error:") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "host error:")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "disk full")
	}
}

func TestCreateSandbox_Defaults(t *testing.T) {
	reg := newRegistryWithHost(t, "host-1", "org-1", &fluidv1.HostRegistration{
		AvailableCpus:     16,
		AvailableMemoryMb: 32768,
		BaseImages:        []string{"ubuntu-22.04"},
	})

	var capturedCmd *fluidv1.CreateSandboxCommand
	ms := &mockStore{
		CreateSandboxFn: func(_ context.Context, _ *store.Sandbox) error {
			return nil
		},
	}
	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, _ string, msg *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			capturedCmd = msg.GetCreateSandbox()
			return &fluidv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &fluidv1.HostMessage_SandboxCreated{
					SandboxCreated: &fluidv1.SandboxCreated{
						SandboxId: capturedCmd.GetSandboxId(),
						Name:      capturedCmd.GetName(),
						State:     "RUNNING",
					},
				},
			}, nil
		},
	}

	orch := New(reg, ms, sender, nil, 24*time.Hour, 90*time.Second)
	result, err := orch.CreateSandbox(context.Background(), CreateSandboxRequest{
		OrgID:    "org-1",
		SourceVM: "ubuntu-22.04",
		VCPUs:    0,
		MemoryMB: 0,
	})
	if err != nil {
		t.Fatalf("CreateSandbox: unexpected error: %v", err)
	}
	if capturedCmd == nil {
		t.Fatal("SendAndWait was not called")
	}
	if capturedCmd.GetVcpus() != 2 {
		t.Errorf("default VCPUs = %d, want 2", capturedCmd.GetVcpus())
	}
	if capturedCmd.GetMemoryMb() != 2048 {
		t.Errorf("default MemoryMB = %d, want 2048", capturedCmd.GetMemoryMb())
	}
	if result.VCPUs != 2 {
		t.Errorf("result.VCPUs = %d, want 2", result.VCPUs)
	}
	if result.MemoryMB != 2048 {
		t.Errorf("result.MemoryMB = %d, want 2048", result.MemoryMB)
	}
}

func TestDestroySandbox_Success(t *testing.T) {
	var deletedID string
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, OrgID: "org-1", HostID: "host-1", State: store.SandboxStateRunning}, nil
		},
		DeleteSandboxFn: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			if hostID != "host-1" {
				t.Errorf("hostID = %q, want %q", hostID, "host-1")
			}
			return &fluidv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &fluidv1.HostMessage_SandboxDestroyed{
					SandboxDestroyed: &fluidv1.SandboxDestroyed{
						SandboxId: msg.GetDestroySandbox().GetSandboxId(),
					},
				},
			}, nil
		},
	}

	orch := newTestOrchestrator(ms, sender)
	err := orch.DestroySandbox(context.Background(), "org-1", "sbx-1")
	if err != nil {
		t.Fatalf("DestroySandbox: unexpected error: %v", err)
	}
	if deletedID != "sbx-1" {
		t.Errorf("deleted sandbox ID = %q, want %q", deletedID, "sbx-1")
	}
}

func TestDestroySandbox_NotFound(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, _ string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		},
	}

	orch := newTestOrchestrator(ms, &mockSender{})
	err := orch.DestroySandbox(context.Background(), "org-1", "nonexistent")
	if err == nil {
		t.Fatal("DestroySandbox: expected error for nonexistent sandbox")
	}
}

func TestDestroySandbox_SenderError(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, HostID: "host-1"}, nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, _ string, _ *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			return nil, fmt.Errorf("timeout waiting for response")
		},
	}

	orch := newTestOrchestrator(ms, sender)
	err := orch.DestroySandbox(context.Background(), "org-1", "sbx-1")
	if err == nil {
		t.Fatal("DestroySandbox: expected error from sender")
	}
	if !strings.Contains(err.Error(), "timeout waiting for response") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "timeout waiting for response")
	}
}

func TestRunCommand_Success(t *testing.T) {
	var storedCmd *store.Command
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, HostID: "host-1"}, nil
		},
		CreateCommandFn: func(_ context.Context, cmd *store.Command) error {
			storedCmd = cmd
			return nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			if hostID != "host-1" {
				t.Errorf("hostID = %q, want %q", hostID, "host-1")
			}
			return &fluidv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &fluidv1.HostMessage_CommandResult{
					CommandResult: &fluidv1.CommandResult{
						Stdout:     "hello world\n",
						Stderr:     "",
						ExitCode:   0,
						DurationMs: 42,
					},
				},
			}, nil
		},
	}

	orch := newTestOrchestrator(ms, sender)
	result, err := orch.RunCommand(context.Background(), "org-1", "sbx-1", "echo hello world", 30)
	if err != nil {
		t.Fatalf("RunCommand: unexpected error: %v", err)
	}
	if result.Stdout != "hello world\n" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "hello world\n")
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.DurationMS != 42 {
		t.Errorf("DurationMS = %d, want 42", result.DurationMS)
	}
	if result.Command != "echo hello world" {
		t.Errorf("Command = %q, want %q", result.Command, "echo hello world")
	}
	if result.SandboxID != "sbx-1" {
		t.Errorf("SandboxID = %q, want %q", result.SandboxID, "sbx-1")
	}
	if storedCmd == nil {
		t.Fatal("CreateCommandFn was not called")
	}
	if storedCmd.Stdout != "hello world\n" {
		t.Errorf("stored Stdout = %q, want %q", storedCmd.Stdout, "hello world\n")
	}
}

func TestRunCommand_SenderError(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, HostID: "host-1"}, nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, _ string, _ *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			return nil, fmt.Errorf("host unreachable")
		},
	}

	orch := newTestOrchestrator(ms, sender)
	_, err := orch.RunCommand(context.Background(), "org-1", "sbx-1", "ls", 30)
	if err == nil {
		t.Fatal("RunCommand: expected error from sender")
	}
	if !strings.Contains(err.Error(), "host unreachable") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "host unreachable")
	}
}

func TestStartSandbox_Success(t *testing.T) {
	var updatedSandbox *store.Sandbox
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, HostID: "host-1", State: store.SandboxStateStopped}, nil
		},
		UpdateSandboxFn: func(_ context.Context, s *store.Sandbox) error {
			updatedSandbox = s
			return nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			if hostID != "host-1" {
				t.Errorf("hostID = %q, want %q", hostID, "host-1")
			}
			return &fluidv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &fluidv1.HostMessage_SandboxStarted{
					SandboxStarted: &fluidv1.SandboxStarted{
						SandboxId: msg.GetStartSandbox().GetSandboxId(),
						State:     "RUNNING",
						IpAddress: "10.0.0.10",
					},
				},
			}, nil
		},
	}

	orch := newTestOrchestrator(ms, sender)
	err := orch.StartSandbox(context.Background(), "org-1", "sbx-1")
	if err != nil {
		t.Fatalf("StartSandbox: unexpected error: %v", err)
	}
	if updatedSandbox == nil {
		t.Fatal("UpdateSandboxFn was not called")
	}
	if updatedSandbox.State != store.SandboxStateRunning {
		t.Errorf("State = %q, want %q", updatedSandbox.State, store.SandboxStateRunning)
	}
	if updatedSandbox.IPAddress != "10.0.0.10" {
		t.Errorf("IPAddress = %q, want %q", updatedSandbox.IPAddress, "10.0.0.10")
	}
}

func TestStartSandbox_NotFound(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, _ string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		},
	}

	orch := newTestOrchestrator(ms, &mockSender{})
	err := orch.StartSandbox(context.Background(), "org-1", "nonexistent")
	if err == nil {
		t.Fatal("StartSandbox: expected error for nonexistent sandbox")
	}
}

func TestStopSandbox_Success(t *testing.T) {
	var updatedSandbox *store.Sandbox
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, HostID: "host-1", State: store.SandboxStateRunning}, nil
		},
		UpdateSandboxFn: func(_ context.Context, s *store.Sandbox) error {
			updatedSandbox = s
			return nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			if hostID != "host-1" {
				t.Errorf("hostID = %q, want %q", hostID, "host-1")
			}
			return &fluidv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &fluidv1.HostMessage_SandboxStopped{
					SandboxStopped: &fluidv1.SandboxStopped{
						SandboxId: msg.GetStopSandbox().GetSandboxId(),
						State:     "STOPPED",
					},
				},
			}, nil
		},
	}

	orch := newTestOrchestrator(ms, sender)
	err := orch.StopSandbox(context.Background(), "org-1", "sbx-1")
	if err != nil {
		t.Fatalf("StopSandbox: unexpected error: %v", err)
	}
	if updatedSandbox == nil {
		t.Fatal("UpdateSandboxFn was not called")
	}
	if updatedSandbox.State != store.SandboxStateStopped {
		t.Errorf("State = %q, want %q", updatedSandbox.State, store.SandboxStateStopped)
	}
}

func TestStopSandbox_SenderError(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, HostID: "host-1"}, nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, _ string, _ *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			return nil, fmt.Errorf("stream closed")
		},
	}

	orch := newTestOrchestrator(ms, sender)
	err := orch.StopSandbox(context.Background(), "org-1", "sbx-1")
	if err == nil {
		t.Fatal("StopSandbox: expected error from sender")
	}
	if !strings.Contains(err.Error(), "stream closed") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "stream closed")
	}
}

func TestCreateSnapshot_Success(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, HostID: "host-1", State: store.SandboxStateRunning}, nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			if hostID != "host-1" {
				t.Errorf("hostID = %q, want %q", hostID, "host-1")
			}
			return &fluidv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &fluidv1.HostMessage_SnapshotCreated{
					SnapshotCreated: &fluidv1.SnapshotCreated{
						SnapshotId:   "snap-abc123",
						SnapshotName: "before-deploy",
					},
				},
			}, nil
		},
	}

	orch := newTestOrchestrator(ms, sender)
	result, err := orch.CreateSnapshot(context.Background(), "org-1", "sbx-1", "before-deploy")
	if err != nil {
		t.Fatalf("CreateSnapshot: unexpected error: %v", err)
	}
	if result.SnapshotID != "snap-abc123" {
		t.Errorf("SnapshotID = %q, want %q", result.SnapshotID, "snap-abc123")
	}
	if result.SnapshotName != "before-deploy" {
		t.Errorf("SnapshotName = %q, want %q", result.SnapshotName, "before-deploy")
	}
	if result.SandboxID != "sbx-1" {
		t.Errorf("SandboxID = %q, want %q", result.SandboxID, "sbx-1")
	}
	if result.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestCreateSnapshot_SenderError(t *testing.T) {
	ms := &mockStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id, HostID: "host-1"}, nil
		},
	}

	sender := &mockSender{
		SendAndWaitFn: func(_ context.Context, _ string, _ *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			return nil, fmt.Errorf("snapshot failed")
		},
	}

	orch := newTestOrchestrator(ms, sender)
	_, err := orch.CreateSnapshot(context.Background(), "org-1", "sbx-1", "my-snap")
	if err == nil {
		t.Fatal("CreateSnapshot: expected error from sender")
	}
	if !strings.Contains(err.Error(), "snapshot failed") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "snapshot failed")
	}
}
