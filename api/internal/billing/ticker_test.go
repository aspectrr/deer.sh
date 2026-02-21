package billing

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/config"
	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/store"
	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
)

// ---------------------------------------------------------------------------
// tickerMockStore - implements store.DataStore with tracking and overrides
// ---------------------------------------------------------------------------

type tickerMockStore struct {
	mu sync.Mutex

	// Override functions for specific methods
	getOrgFn            func(ctx context.Context, id string) (*store.Organization, error)
	createUsageRecordFn func(ctx context.Context, rec *store.UsageRecord) error

	// Tracking
	usageRecords []*store.UsageRecord
	methodCalls  map[string]int
}

func newTickerMockStore() *tickerMockStore {
	return &tickerMockStore{
		methodCalls: make(map[string]int),
	}
}

func (m *tickerMockStore) track(method string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.methodCalls[method]++
}

func (m *tickerMockStore) callCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.methodCalls[method]
}

// --- Overrideable methods used by reportForOrg ---

func (m *tickerMockStore) GetOrganization(ctx context.Context, id string) (*store.Organization, error) {
	m.track("GetOrganization")
	if m.getOrgFn != nil {
		return m.getOrgFn(ctx, id)
	}
	return &store.Organization{ID: id}, nil
}

func (m *tickerMockStore) ListSandboxesByOrg(context.Context, string) ([]store.Sandbox, error) {
	return nil, nil
}

func (m *tickerMockStore) ListSourceHostsByOrg(context.Context, string) ([]*store.SourceHost, error) {
	return nil, nil
}

func (m *tickerMockStore) CreateUsageRecord(ctx context.Context, rec *store.UsageRecord) error {
	m.track("CreateUsageRecord")
	m.mu.Lock()
	m.usageRecords = append(m.usageRecords, rec)
	m.mu.Unlock()
	if m.createUsageRecordFn != nil {
		return m.createUsageRecordFn(ctx, rec)
	}
	return nil
}

// --- Stub implementations for the rest of store.DataStore ---

func (m *tickerMockStore) CreateUser(context.Context, *store.User) error        { return nil }
func (m *tickerMockStore) GetUser(context.Context, string) (*store.User, error) { return nil, nil }
func (m *tickerMockStore) GetUserByEmail(context.Context, string) (*store.User, error) {
	return nil, nil
}
func (m *tickerMockStore) UpdateUser(context.Context, *store.User) error { return nil }

func (m *tickerMockStore) CreateOAuthAccount(context.Context, *store.OAuthAccount) error { return nil }
func (m *tickerMockStore) GetOAuthAccount(context.Context, string, string) (*store.OAuthAccount, error) {
	return nil, nil
}
func (m *tickerMockStore) GetOAuthAccountsByUser(context.Context, string) ([]*store.OAuthAccount, error) {
	return nil, nil
}

func (m *tickerMockStore) CreateSession(context.Context, *store.Session) error { return nil }
func (m *tickerMockStore) GetSession(context.Context, string) (*store.Session, error) {
	return nil, nil
}
func (m *tickerMockStore) DeleteSession(context.Context, string) error { return nil }
func (m *tickerMockStore) DeleteExpiredSessions(context.Context) error { return nil }

func (m *tickerMockStore) CreateOrganization(context.Context, *store.Organization) error { return nil }
func (m *tickerMockStore) GetOrganizationBySlug(context.Context, string) (*store.Organization, error) {
	return nil, nil
}
func (m *tickerMockStore) ListOrganizationsByUser(context.Context, string) ([]*store.Organization, error) {
	return nil, nil
}
func (m *tickerMockStore) UpdateOrganization(context.Context, *store.Organization) error { return nil }
func (m *tickerMockStore) DeleteOrganization(context.Context, string) error              { return nil }

func (m *tickerMockStore) CreateOrgMember(context.Context, *store.OrgMember) error { return nil }
func (m *tickerMockStore) GetOrgMember(context.Context, string, string) (*store.OrgMember, error) {
	return nil, nil
}
func (m *tickerMockStore) GetOrgMemberByID(context.Context, string, string) (*store.OrgMember, error) {
	return nil, nil
}
func (m *tickerMockStore) ListOrgMembers(context.Context, string) ([]*store.OrgMember, error) {
	return nil, nil
}
func (m *tickerMockStore) DeleteOrgMember(context.Context, string, string) error { return nil }

func (m *tickerMockStore) CreateSubscription(context.Context, *store.Subscription) error { return nil }
func (m *tickerMockStore) GetSubscriptionByOrg(context.Context, string) (*store.Subscription, error) {
	return nil, nil
}
func (m *tickerMockStore) UpdateSubscription(context.Context, *store.Subscription) error { return nil }

func (m *tickerMockStore) ListUsageRecords(context.Context, string, time.Time, time.Time) ([]*store.UsageRecord, error) {
	return nil, nil
}

func (m *tickerMockStore) CreateHost(context.Context, *store.Host) error        { return nil }
func (m *tickerMockStore) GetHost(context.Context, string) (*store.Host, error) { return nil, nil }
func (m *tickerMockStore) ListHosts(context.Context) ([]store.Host, error)      { return nil, nil }
func (m *tickerMockStore) ListHostsByOrg(context.Context, string) ([]store.Host, error) {
	return nil, nil
}
func (m *tickerMockStore) UpdateHost(context.Context, *store.Host) error { return nil }
func (m *tickerMockStore) UpdateHostHeartbeat(context.Context, string, int32, int64, int64) error {
	return nil
}

func (m *tickerMockStore) CreateSandbox(context.Context, *store.Sandbox) error { return nil }
func (m *tickerMockStore) GetSandbox(context.Context, string) (*store.Sandbox, error) {
	return nil, nil
}
func (m *tickerMockStore) GetSandboxByOrg(context.Context, string, string) (*store.Sandbox, error) {
	return nil, nil
}
func (m *tickerMockStore) ListSandboxes(context.Context) ([]store.Sandbox, error) { return nil, nil }
func (m *tickerMockStore) UpdateSandbox(context.Context, *store.Sandbox) error    { return nil }
func (m *tickerMockStore) DeleteSandbox(context.Context, string) error            { return nil }
func (m *tickerMockStore) GetSandboxesByHostID(context.Context, string) ([]store.Sandbox, error) {
	return nil, nil
}
func (m *tickerMockStore) CountSandboxesByHostIDs(context.Context, []string) (map[string]int, error) {
	return map[string]int{}, nil
}
func (m *tickerMockStore) ListExpiredSandboxes(context.Context, time.Duration) ([]store.Sandbox, error) {
	return nil, nil
}

func (m *tickerMockStore) CreateCommand(context.Context, *store.Command) error { return nil }
func (m *tickerMockStore) ListSandboxCommands(context.Context, string) ([]store.Command, error) {
	return nil, nil
}

func (m *tickerMockStore) CreateSourceHost(context.Context, *store.SourceHost) error { return nil }
func (m *tickerMockStore) GetSourceHost(context.Context, string) (*store.SourceHost, error) {
	return nil, nil
}
func (m *tickerMockStore) DeleteSourceHost(context.Context, string) error { return nil }

func (m *tickerMockStore) CreateHostToken(context.Context, *store.HostToken) error { return nil }
func (m *tickerMockStore) GetHostTokenByHash(context.Context, string) (*store.HostToken, error) {
	return nil, nil
}
func (m *tickerMockStore) ListHostTokensByOrg(context.Context, string) ([]store.HostToken, error) {
	return nil, nil
}
func (m *tickerMockStore) DeleteHostToken(context.Context, string, string) error { return nil }

// Agent/playbook mock methods removed - interface methods commented out in store.go

func (m *tickerMockStore) GetOrganizationByStripeCustomerID(context.Context, string) (*store.Organization, error) {
	return nil, nil
}
func (m *tickerMockStore) GetModelMeter(context.Context, string) (*store.ModelMeter, error) {
	return nil, store.ErrNotFound
}
func (m *tickerMockStore) CreateModelMeter(context.Context, *store.ModelMeter) error { return nil }
func (m *tickerMockStore) GetOrgModelSubscription(context.Context, string, string) (*store.OrgModelSubscription, error) {
	return nil, store.ErrNotFound
}
func (m *tickerMockStore) CreateOrgModelSubscription(context.Context, *store.OrgModelSubscription) error {
	return nil
}
func (m *tickerMockStore) SumTokenUsage(context.Context, string, time.Time, time.Time) (float64, error) {
	return 0, nil
}
func (m *tickerMockStore) ListActiveSubscriptions(context.Context) ([]*store.Subscription, error) {
	return nil, nil
}
func (m *tickerMockStore) GetSubscriptionByStripeID(context.Context, string) (*store.Subscription, error) {
	return nil, nil
}
func (m *tickerMockStore) AcquireAdvisoryLock(context.Context, int64) error { return nil }
func (m *tickerMockStore) ReleaseAdvisoryLock(context.Context, int64) error { return nil }

// ---------------------------------------------------------------------------
// mockHostStream - minimal HostStream for registry.Register
// ---------------------------------------------------------------------------

type mockHostStream struct{}

func (mockHostStream) Send(_ *fluidv1.ControlMessage) error { return nil }

// ---------------------------------------------------------------------------
// Helper to build a ResourceTicker for tests
// ---------------------------------------------------------------------------

func newTestTicker(st store.DataStore, freeTier config.FreeTierConfig) *ResourceTicker {
	mm := NewMeterManager(st, "", 1.2, nil)
	reg := registry.New()
	cfg := config.BillingConfig{
		FreeTier: freeTier,
	}
	return NewResourceTicker(st, mm, reg, cfg, nil)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestReportForOrg_NoStripeCustomer(t *testing.T) {
	ms := newTickerMockStore()
	ms.getOrgFn = func(_ context.Context, id string) (*store.Organization, error) {
		return &store.Organization{
			ID:               id,
			StripeCustomerID: "", // no Stripe customer
		}, nil
	}

	rt := newTestTicker(ms, config.FreeTierConfig{
		MaxConcurrentSandboxes: 1,
		MaxSourceVMs:           1,
		MaxAgentHosts:          1,
	})

	rt.reportForOrg(context.Background(), "org-no-stripe")

	if ms.callCount("GetOrganization") != 1 {
		t.Errorf("GetOrganization call count = %d, want 1", ms.callCount("GetOrganization"))
	}

	// Should return early before creating any usage records.
	if ms.callCount("CreateUsageRecord") != 0 {
		t.Errorf("CreateUsageRecord should not be called, got %d calls", ms.callCount("CreateUsageRecord"))
	}
}

func TestReportForOrg_FreeTierSubtraction(t *testing.T) {
	ms := newTickerMockStore()
	ms.getOrgFn = func(_ context.Context, id string) (*store.Organization, error) {
		return &store.Organization{
			ID:               id,
			StripeCustomerID: "cus_test123",
		}, nil
	}

	freeTier := config.FreeTierConfig{
		MaxConcurrentSandboxes: 1,
		MaxSourceVMs:           1,
		MaxAgentHosts:          1,
	}

	// Build the ticker with a registry that has 2 connected daemons for this org.
	mm := NewMeterManager(ms, "", 1.2, nil)
	reg := registry.New()
	cfg := config.BillingConfig{FreeTier: freeTier}
	rt := NewResourceTicker(ms, mm, reg, cfg, nil)

	// Register 2 daemons and set heartbeat counts.
	_ = reg.Register("host-1", "org-ft", "daemon-1", mockHostStream{})
	_ = reg.Register("host-2", "org-ft", "daemon-2", mockHostStream{})
	reg.UpdateHeartbeatCounts("host-1", 1, 2) // 1 sandbox, 2 source VMs
	reg.UpdateHeartbeatCounts("host-2", 1, 1) // 1 sandbox, 1 source VM

	rt.reportForOrg(context.Background(), "org-ft")

	// 2 sandboxes, 3 source VMs, 2 daemons -> usage records for all three.
	// Local usage records record the raw counts (not free-tier-subtracted).
	if ms.callCount("CreateUsageRecord") != 3 {
		t.Fatalf("CreateUsageRecord call count = %d, want 3", ms.callCount("CreateUsageRecord"))
	}

	// Verify usage record contents.
	recordsByType := make(map[string]*store.UsageRecord)
	for _, rec := range ms.usageRecords {
		recordsByType[rec.ResourceType] = rec
	}

	// max_concurrent_sandboxes: raw count = 2 (1+1 from heartbeats)
	if rec, ok := recordsByType["max_concurrent_sandboxes"]; !ok {
		t.Error("missing max_concurrent_sandboxes usage record")
	} else if rec.Quantity != 2 {
		t.Errorf("max_concurrent_sandboxes quantity = %v, want 2", rec.Quantity)
	}

	// source_vm: raw count = 3 (2+1 from heartbeats)
	if rec, ok := recordsByType["source_vm"]; !ok {
		t.Error("missing source_vm usage record")
	} else if rec.Quantity != 3 {
		t.Errorf("source_vm quantity = %v, want 3", rec.Quantity)
	}

	// agent_host: raw count = 2 daemons
	if rec, ok := recordsByType["agent_host"]; !ok {
		t.Error("missing agent_host usage record")
	} else if rec.Quantity != 2 {
		t.Errorf("agent_host quantity = %v, want 2", rec.Quantity)
	}

	// Verify orgID on all records.
	for _, rec := range ms.usageRecords {
		if rec.OrgID != "org-ft" {
			t.Errorf("usage record OrgID = %q, want %q", rec.OrgID, "org-ft")
		}
	}
}

func TestReportForOrg_StoreError(t *testing.T) {
	ms := newTickerMockStore()
	ms.getOrgFn = func(_ context.Context, _ string) (*store.Organization, error) {
		return nil, fmt.Errorf("database connection lost")
	}

	rt := newTestTicker(ms, config.FreeTierConfig{
		MaxConcurrentSandboxes: 1,
		MaxSourceVMs:           1,
		MaxAgentHosts:          1,
	})

	// Should not panic.
	rt.reportForOrg(context.Background(), "org-error")

	if ms.callCount("GetOrganization") != 1 {
		t.Errorf("GetOrganization call count = %d, want 1", ms.callCount("GetOrganization"))
	}

	// Should return early without creating usage records.
	if ms.callCount("CreateUsageRecord") != 0 {
		t.Errorf("CreateUsageRecord should not be called after store error, got %d", ms.callCount("CreateUsageRecord"))
	}
}

func TestReportForOrg_RegistryZeroCounts(t *testing.T) {
	ms := newTickerMockStore()
	ms.getOrgFn = func(_ context.Context, id string) (*store.Organization, error) {
		return &store.Organization{
			ID:               id,
			StripeCustomerID: "cus_test123",
		}, nil
	}

	// Registry has a daemon but heartbeat counts are zero.
	mm := NewMeterManager(ms, "", 1.2, nil)
	reg := registry.New()
	cfg := config.BillingConfig{FreeTier: config.FreeTierConfig{
		MaxConcurrentSandboxes: 1,
		MaxSourceVMs:           1,
		MaxAgentHosts:          1,
	}}
	rt := NewResourceTicker(ms, mm, reg, cfg, nil)

	_ = reg.Register("host-1", "org-zero", "daemon-1", mockHostStream{})
	// No UpdateHeartbeatCounts called - counts remain 0.

	rt.reportForOrg(context.Background(), "org-zero")

	// 1 daemon > 0 so agent_host usage record created; sandboxes and source VMs are 0.
	if ms.callCount("CreateUsageRecord") != 1 {
		t.Errorf("CreateUsageRecord call count = %d, want 1 (only agent_host)", ms.callCount("CreateUsageRecord"))
	}
	if len(ms.usageRecords) == 1 && ms.usageRecords[0].ResourceType != "agent_host" {
		t.Errorf("expected agent_host usage record, got %q", ms.usageRecords[0].ResourceType)
	}
}

func TestReportForOrg_NoConnectedDaemons(t *testing.T) {
	ms := newTickerMockStore()
	ms.getOrgFn = func(_ context.Context, id string) (*store.Organization, error) {
		return &store.Organization{
			ID:               id,
			StripeCustomerID: "cus_test123",
		}, nil
	}

	// Empty registry - no daemons connected.
	rt := newTestTicker(ms, config.FreeTierConfig{
		MaxConcurrentSandboxes: 1,
		MaxSourceVMs:           1,
		MaxAgentHosts:          1,
	})

	rt.reportForOrg(context.Background(), "org-empty")

	// All counts are 0, nothing to report.
	if ms.callCount("CreateUsageRecord") != 0 {
		t.Errorf("CreateUsageRecord should not be called with no daemons, got %d", ms.callCount("CreateUsageRecord"))
	}
}
