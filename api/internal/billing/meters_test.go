package billing

import (
	"testing"
)

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
// Token billing tests - commented out, not yet ready for integration.
// ---------------------------------------------------------------------------
/*
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
	ms := &mockDataStore{
		org: &store.Organization{ID: "org-1", StripeCustomerID: "cus_123"},
		sub: &store.Subscription{
			Status:               store.SubStatusActive,
			StripeSubscriptionID: "sub_123",
		},
		sumUsage: 200,
	}
	mm := newTestMeterManager(ms, 10000)

	err := mm.ReportUsage(context.Background(), "org-1", "anthropic/claude-sonnet-4", 100, 50)
	if err != nil {
		t.Fatalf("expected nil error when all usage is within free tier, got: %v", err)
	}
}

func TestReportUsage_AllBillable(t *testing.T) {
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
	if err == nil {
		t.Fatal("expected non-nil error from Stripe API path, got nil")
	}
}

func TestReportUsage_ExactlyAtFreeTierBoundary(t *testing.T) {
	ms := &mockDataStore{
		org: &store.Organization{ID: "org-1", StripeCustomerID: "cus_123"},
		sub: &store.Subscription{
			Status:               store.SubStatusActive,
			StripeSubscriptionID: "sub_123",
		},
		sumUsage: 1000,
	}
	mm := newTestMeterManager(ms, 1000)

	err := mm.ReportUsage(context.Background(), "org-1", "anthropic/claude-sonnet-4", 120, 80)
	if err != nil {
		t.Fatalf("expected nil error at exact free tier boundary, got: %v", err)
	}
}

func TestReportUsage_OneTokenOverFreeTier(t *testing.T) {
	ms := &mockDataStore{
		org: &store.Organization{ID: "org-1", StripeCustomerID: "cus_123"},
		sub: &store.Subscription{
			Status:               store.SubStatusActive,
			StripeSubscriptionID: "sub_123",
		},
		sumUsage: 1001,
	}
	mm := newTestMeterManager(ms, 1000)

	err := mm.ReportUsage(context.Background(), "org-1", "anthropic/claude-sonnet-4", 120, 80)
	if err == nil {
		t.Fatal("expected non-nil error from Stripe API path for 1 token over free tier, got nil")
	}
}
*/
