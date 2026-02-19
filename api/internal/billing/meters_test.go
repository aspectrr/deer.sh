package billing

import (
	"context"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/agent"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// ---------------------------------------------------------------------------
// mockDataStore - minimal store.DataStore implementation for billing tests.
// Only the methods called by ReportUsage are implemented with real logic;
// everything else panics so we notice immediately if an unexpected method
// is called.
// ---------------------------------------------------------------------------

type mockDataStore struct {
	org         *store.Organization
	sub         *store.Subscription
	subErr      error
	sumUsage    float64
	sumUsageErr error
	modelMeter  *store.ModelMeter
	orgModelSub *store.OrgModelSubscription
}

// Methods used by ReportUsage ---

func (m *mockDataStore) GetOrganization(_ context.Context, _ string) (*store.Organization, error) {
	return m.org, nil
}

func (m *mockDataStore) GetSubscriptionByOrg(_ context.Context, _ string) (*store.Subscription, error) {
	if m.subErr != nil {
		return nil, m.subErr
	}
	return m.sub, nil
}

func (m *mockDataStore) SumTokenUsage(_ context.Context, _ string, _, _ time.Time) (float64, error) {
	if m.sumUsageErr != nil {
		return 0, m.sumUsageErr
	}
	return m.sumUsage, nil
}

// All remaining DataStore interface methods - unimplemented stubs ---

func (m *mockDataStore) CreateUser(context.Context, *store.User) error { panic("unimplemented") }
func (m *mockDataStore) GetUser(context.Context, string) (*store.User, error) {
	panic("unimplemented")
}
func (m *mockDataStore) GetUserByEmail(context.Context, string) (*store.User, error) {
	panic("unimplemented")
}
func (m *mockDataStore) UpdateUser(context.Context, *store.User) error { panic("unimplemented") }

func (m *mockDataStore) CreateOAuthAccount(context.Context, *store.OAuthAccount) error {
	panic("unimplemented")
}
func (m *mockDataStore) GetOAuthAccount(context.Context, string, string) (*store.OAuthAccount, error) {
	panic("unimplemented")
}
func (m *mockDataStore) GetOAuthAccountsByUser(context.Context, string) ([]*store.OAuthAccount, error) {
	panic("unimplemented")
}

func (m *mockDataStore) CreateSession(context.Context, *store.Session) error { panic("unimplemented") }
func (m *mockDataStore) GetSession(context.Context, string) (*store.Session, error) {
	panic("unimplemented")
}
func (m *mockDataStore) DeleteSession(context.Context, string) error { panic("unimplemented") }
func (m *mockDataStore) DeleteExpiredSessions(context.Context) error { panic("unimplemented") }

func (m *mockDataStore) CreateOrganization(context.Context, *store.Organization) error {
	panic("unimplemented")
}
func (m *mockDataStore) GetOrganizationBySlug(context.Context, string) (*store.Organization, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListOrganizationsByUser(context.Context, string) ([]*store.Organization, error) {
	panic("unimplemented")
}
func (m *mockDataStore) UpdateOrganization(context.Context, *store.Organization) error {
	panic("unimplemented")
}
func (m *mockDataStore) DeleteOrganization(context.Context, string) error { panic("unimplemented") }

func (m *mockDataStore) CreateOrgMember(context.Context, *store.OrgMember) error {
	panic("unimplemented")
}
func (m *mockDataStore) GetOrgMember(context.Context, string, string) (*store.OrgMember, error) {
	panic("unimplemented")
}
func (m *mockDataStore) GetOrgMemberByID(context.Context, string, string) (*store.OrgMember, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListOrgMembers(context.Context, string) ([]*store.OrgMember, error) {
	panic("unimplemented")
}
func (m *mockDataStore) DeleteOrgMember(context.Context, string, string) error {
	panic("unimplemented")
}

func (m *mockDataStore) CreateSubscription(context.Context, *store.Subscription) error {
	panic("unimplemented")
}
func (m *mockDataStore) UpdateSubscription(context.Context, *store.Subscription) error {
	panic("unimplemented")
}

func (m *mockDataStore) CreateUsageRecord(context.Context, *store.UsageRecord) error {
	panic("unimplemented")
}
func (m *mockDataStore) ListUsageRecords(context.Context, string, time.Time, time.Time) ([]*store.UsageRecord, error) {
	panic("unimplemented")
}

func (m *mockDataStore) CreateHost(context.Context, *store.Host) error        { panic("unimplemented") }
func (m *mockDataStore) GetHost(context.Context, string) (*store.Host, error) { panic("unimplemented") }
func (m *mockDataStore) ListHosts(context.Context) ([]store.Host, error)      { panic("unimplemented") }
func (m *mockDataStore) UpdateHost(context.Context, *store.Host) error        { panic("unimplemented") }
func (m *mockDataStore) UpdateHostHeartbeat(context.Context, string, int32, int64, int64) error {
	panic("unimplemented")
}

func (m *mockDataStore) CreateSandbox(context.Context, *store.Sandbox) error { panic("unimplemented") }
func (m *mockDataStore) GetSandbox(context.Context, string) (*store.Sandbox, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListSandboxes(context.Context) ([]store.Sandbox, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListSandboxesByOrg(context.Context, string) ([]store.Sandbox, error) {
	panic("unimplemented")
}
func (m *mockDataStore) UpdateSandbox(context.Context, *store.Sandbox) error { panic("unimplemented") }
func (m *mockDataStore) DeleteSandbox(context.Context, string) error         { panic("unimplemented") }
func (m *mockDataStore) GetSandboxesByHostID(context.Context, string) ([]store.Sandbox, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListExpiredSandboxes(context.Context, time.Duration) ([]store.Sandbox, error) {
	panic("unimplemented")
}

func (m *mockDataStore) CreateCommand(context.Context, *store.Command) error { panic("unimplemented") }
func (m *mockDataStore) ListSandboxCommands(context.Context, string) ([]store.Command, error) {
	panic("unimplemented")
}

func (m *mockDataStore) CreateSourceHost(context.Context, *store.SourceHost) error {
	panic("unimplemented")
}
func (m *mockDataStore) GetSourceHost(context.Context, string) (*store.SourceHost, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListSourceHostsByOrg(context.Context, string) ([]*store.SourceHost, error) {
	panic("unimplemented")
}
func (m *mockDataStore) DeleteSourceHost(context.Context, string) error { panic("unimplemented") }

func (m *mockDataStore) CreateHostToken(context.Context, *store.HostToken) error {
	panic("unimplemented")
}
func (m *mockDataStore) GetHostTokenByHash(context.Context, string) (*store.HostToken, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListHostTokensByOrg(context.Context, string) ([]store.HostToken, error) {
	panic("unimplemented")
}
func (m *mockDataStore) DeleteHostToken(context.Context, string, string) error {
	panic("unimplemented")
}

func (m *mockDataStore) CreateAgentConversation(context.Context, *store.AgentConversation) error {
	panic("unimplemented")
}
func (m *mockDataStore) GetAgentConversation(context.Context, string) (*store.AgentConversation, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListAgentConversationsByOrg(context.Context, string) ([]*store.AgentConversation, error) {
	panic("unimplemented")
}
func (m *mockDataStore) DeleteAgentConversation(context.Context, string) error {
	panic("unimplemented")
}

func (m *mockDataStore) CreateAgentMessage(context.Context, *store.AgentMessage) error {
	panic("unimplemented")
}
func (m *mockDataStore) ListAgentMessages(context.Context, string) ([]*store.AgentMessage, error) {
	panic("unimplemented")
}

func (m *mockDataStore) CreatePlaybook(context.Context, *store.Playbook) error {
	panic("unimplemented")
}
func (m *mockDataStore) GetPlaybook(context.Context, string) (*store.Playbook, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListPlaybooksByOrg(context.Context, string) ([]*store.Playbook, error) {
	panic("unimplemented")
}
func (m *mockDataStore) UpdatePlaybook(context.Context, *store.Playbook) error {
	panic("unimplemented")
}
func (m *mockDataStore) DeletePlaybook(context.Context, string) error { panic("unimplemented") }

func (m *mockDataStore) CreatePlaybookTask(context.Context, *store.PlaybookTask) error {
	panic("unimplemented")
}
func (m *mockDataStore) GetPlaybookTask(context.Context, string) (*store.PlaybookTask, error) {
	panic("unimplemented")
}
func (m *mockDataStore) ListPlaybookTasks(context.Context, string) ([]*store.PlaybookTask, error) {
	panic("unimplemented")
}
func (m *mockDataStore) UpdatePlaybookTask(context.Context, *store.PlaybookTask) error {
	panic("unimplemented")
}
func (m *mockDataStore) DeletePlaybookTask(context.Context, string) error { panic("unimplemented") }
func (m *mockDataStore) ReorderPlaybookTasks(context.Context, string, []string) error {
	panic("unimplemented")
}

func (m *mockDataStore) GetOrganizationByStripeCustomerID(context.Context, string) (*store.Organization, error) {
	panic("unimplemented")
}
func (m *mockDataStore) GetModelMeter(_ context.Context, _ string) (*store.ModelMeter, error) {
	if m.modelMeter != nil {
		return m.modelMeter, nil
	}
	return nil, store.ErrNotFound
}
func (m *mockDataStore) CreateModelMeter(_ context.Context, _ *store.ModelMeter) error {
	return nil
}
func (m *mockDataStore) GetOrgModelSubscription(_ context.Context, _, _ string) (*store.OrgModelSubscription, error) {
	if m.orgModelSub != nil {
		return m.orgModelSub, nil
	}
	return nil, store.ErrNotFound
}
func (m *mockDataStore) CreateOrgModelSubscription(_ context.Context, _ *store.OrgModelSubscription) error {
	return nil
}
func (m *mockDataStore) ListActiveSubscriptions(context.Context) ([]*store.Subscription, error) {
	panic("unimplemented")
}

// ---------------------------------------------------------------------------
// Tests for sanitizeEventName
// ---------------------------------------------------------------------------

func TestSanitizeEventName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "slash separated model ID",
			input: "anthropic/claude-sonnet-4",
			want:  "anthropic_claude_sonnet_4",
		},
		{
			name:  "already clean",
			input: "gpt4o",
			want:  "gpt4o",
		},
		{
			name:  "multiple special chars in a row",
			input: "vendor//model--v2",
			want:  "vendor_model_v2",
		},
		{
			name:  "uppercase converted to lowercase",
			input: "OpenAI/GPT-4",
			want:  "openai_gpt_4",
		},
		{
			name:  "dots and colons",
			input: "meta.llama:3.1",
			want:  "meta_llama_3_1",
		},
		{
			name:  "leading separators trimmed",
			input: "/leading-model",
			want:  "leading_model",
		},
		{
			name:  "trailing separators trimmed",
			input: "trailing-model/",
			want:  "trailing_model",
		},
		{
			name:  "leading and trailing separators",
			input: "--model--",
			want:  "model",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeEventName(tc.input)
			if got != tc.want {
				t.Errorf("sanitizeEventName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for ReportUsage
// ---------------------------------------------------------------------------

func newTestMeterManager(ms *mockDataStore, freeTokens int) *MeterManager {
	mc := agent.NewModelCache(time.Hour)
	// Pass empty stripeKey so we never touch real Stripe.
	return NewMeterManager(ms, mc, "", 1.5, freeTokens, nil)
}

func TestReportUsage_ZeroTokens(t *testing.T) {
	ms := &mockDataStore{
		org: &store.Organization{ID: "org-1", StripeCustomerID: "cus_123"},
	}
	mm := newTestMeterManager(ms, 1000)

	err := mm.ReportUsage(context.Background(), "org-1", "anthropic/claude-sonnet-4", 0, 0)
	if err != nil {
		t.Fatalf("expected nil error for zero tokens, got: %v", err)
	}
}

func TestReportUsage_NoStripeCustomer(t *testing.T) {
	ms := &mockDataStore{
		org: &store.Organization{ID: "org-1", StripeCustomerID: ""},
	}
	mm := newTestMeterManager(ms, 1000)

	err := mm.ReportUsage(context.Background(), "org-1", "anthropic/claude-sonnet-4", 100, 50)
	if err != nil {
		t.Fatalf("expected nil error when org has no Stripe customer, got: %v", err)
	}
}

func TestReportUsage_NoActiveSub(t *testing.T) {
	ms := &mockDataStore{
		org:    &store.Organization{ID: "org-1", StripeCustomerID: "cus_123"},
		subErr: store.ErrNotFound,
	}
	mm := newTestMeterManager(ms, 1000)

	err := mm.ReportUsage(context.Background(), "org-1", "anthropic/claude-sonnet-4", 100, 50)
	if err != nil {
		t.Fatalf("expected nil error when subscription not found, got: %v", err)
	}
}

func TestReportUsage_AllFree(t *testing.T) {
	// Free tier is 10000 tokens. Cumulative usage (including this chat) is 200.
	// prevTotal = 200 - 150 = 50, prevTotal + thisChat = 200 <= 10000, so all free.
	ms := &mockDataStore{
		org: &store.Organization{ID: "org-1", StripeCustomerID: "cus_123"},
		sub: &store.Subscription{
			Status:               store.SubStatusActive,
			StripeSubscriptionID: "sub_123",
		},
		sumUsage: 200, // cumulative including this chat
	}
	mm := newTestMeterManager(ms, 10000)

	err := mm.ReportUsage(context.Background(), "org-1", "anthropic/claude-sonnet-4", 100, 50)
	if err != nil {
		t.Fatalf("expected nil error when all usage is within free tier, got: %v", err)
	}
}

func TestReportUsage_AllBillable(t *testing.T) {
	// Free tier is 1000. Cumulative is 5000, thisChat = 200.
	// prevTotal = 5000 - 200 = 4800, which >= 1000, so all tokens are billable.
	// The flow proceeds past free tier calculation into EnsureModelMeter.
	// Since GetModelMeter returns ErrNotFound and there is no Stripe key,
	// the Stripe product creation will fail. We verify the error comes from
	// the Stripe path (not from a premature free-tier return).
	ms := &mockDataStore{
		org: &store.Organization{ID: "org-1", StripeCustomerID: "cus_123"},
		sub: &store.Subscription{
			Status:               store.SubStatusActive,
			StripeSubscriptionID: "sub_123",
		},
		sumUsage: 5000,
	}
	mm := newTestMeterManager(ms, 1000)

	err := mm.ReportUsage(context.Background(), "org-1", "anthropic/claude-sonnet-4", 120, 80)
	// We expect a non-nil error from the Stripe API call (no valid key configured).
	// This confirms the free tier logic correctly identified all tokens as billable
	// and did NOT return nil early.
	if err == nil {
		t.Fatal("expected non-nil error from Stripe API path, got nil")
	}
}
