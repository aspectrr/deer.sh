package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

func TestHandleGetBilling(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	// No subscription - free plan
	ms.GetSubscriptionByOrgFn = func(_ context.Context, orgID string) (*store.Subscription, error) {
		return nil, store.ErrNotFound
	}
	ms.ListUsageRecordsFn = func(_ context.Context, orgID string, from, to time.Time) ([]*store.UsageRecord, error) {
		return nil, nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/billing", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp billingResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if resp.Plan != string(store.PlanFree) {
		t.Fatalf("expected plan 'free', got %q", resp.Plan)
	}
	if resp.Status != string(store.SubStatusActive) {
		t.Fatalf("expected status 'active', got %q", resp.Status)
	}
	if resp.FreeTier == nil {
		t.Fatal("expected free_tier to be present")
	}
	cfg := testConfig()
	if resp.FreeTier.MaxConcurrentSandboxes != cfg.Billing.FreeTier.MaxConcurrentSandboxes {
		t.Fatalf("expected max_concurrent_sandboxes=%d, got %d", cfg.Billing.FreeTier.MaxConcurrentSandboxes, resp.FreeTier.MaxConcurrentSandboxes)
	}
	if resp.FreeTier.MaxSourceVMs != cfg.Billing.FreeTier.MaxSourceVMs {
		t.Fatalf("expected max_source_vms=%d, got %d", cfg.Billing.FreeTier.MaxSourceVMs, resp.FreeTier.MaxSourceVMs)
	}
	if resp.FreeTier.MaxAgentHosts != cfg.Billing.FreeTier.MaxAgentHosts {
		t.Fatalf("expected max_agent_hosts=%d, got %d", cfg.Billing.FreeTier.MaxAgentHosts, resp.FreeTier.MaxAgentHosts)
	}
}

func TestHandleGetBillingWithUsage(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	ms.GetSubscriptionByOrgFn = func(_ context.Context, orgID string) (*store.Subscription, error) {
		return nil, store.ErrNotFound
	}
	ms.ListUsageRecordsFn = func(_ context.Context, orgID string, from, to time.Time) ([]*store.UsageRecord, error) {
		return []*store.UsageRecord{
			{ID: "u1", OrgID: orgID, ResourceType: "max_concurrent_sandboxes", Quantity: 10.5},
			{ID: "u2", OrgID: orgID, ResourceType: "source_vm", Quantity: 2},
			{ID: "u3", OrgID: orgID, ResourceType: "agent_host", Quantity: 1},
			{ID: "u4", OrgID: orgID, ResourceType: "llm_token", Quantity: 5000},
		}, nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/billing", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp billingResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if resp.Usage == nil {
		t.Fatal("expected usage to be present")
	}
	if resp.Usage.MaxConcurrentSandboxes != 10.5 {
		t.Fatalf("expected max_concurrent_sandboxes=10.5, got %v", resp.Usage.MaxConcurrentSandboxes)
	}
	if resp.Usage.SourceVMs != 2 {
		t.Fatalf("expected source_vms=2, got %v", resp.Usage.SourceVMs)
	}
	if resp.Usage.AgentHosts != 1 {
		t.Fatalf("expected agent_hosts=1, got %v", resp.Usage.AgentHosts)
	}
	if resp.Usage.TokensUsed != 5000 {
		t.Fatalf("expected tokens_used=5000, got %v", resp.Usage.TokensUsed)
	}
}

func TestHandleCalculator(t *testing.T) {
	ms := &mockStore{}
	cfg := testConfig()
	s := newTestServer(ms, cfg)

	body := bytes.NewBufferString(`{
		"concurrent_sandboxes": 2,
		"source_vms": 5,
		"agent_hosts": 2
	}`)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/billing/calculator", body)
	req.Header.Set("Content-Type", "application/json")
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp calculatorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// SandboxCost: (2 - 1 free) * 5000 cents / 100 = $50.00
	expectedSandboxCost := 50.0
	if resp.SandboxCost != expectedSandboxCost {
		t.Fatalf("expected sandbox_cost=%v, got %v", expectedSandboxCost, resp.SandboxCost)
	}

	// SourceVMCost: (5 - 3 free) * 500 cents / 100 = $10.00
	expectedSourceVMCost := 10.0
	if resp.SourceVMCost != expectedSourceVMCost {
		t.Fatalf("expected source_vm_cost=%v, got %v", expectedSourceVMCost, resp.SourceVMCost)
	}

	// AgentHostCost: (2 - 1 free) * 1000 cents / 100 = $10.00
	expectedAgentHostCost := 10.0
	if resp.AgentHostCost != expectedAgentHostCost {
		t.Fatalf("expected agent_host_cost=%v, got %v", expectedAgentHostCost, resp.AgentHostCost)
	}

	// Total: 50 + 10 + 10 = 70.00
	expectedTotal := 70.0
	if resp.TotalMonthly != expectedTotal {
		t.Fatalf("expected total_monthly=%v, got %v", expectedTotal, resp.TotalMonthly)
	}

	if resp.Currency != "USD" {
		t.Fatalf("expected currency=USD, got %q", resp.Currency)
	}
}

// TestHandleCalculatorWithTokens - commented out, token billing not yet ready for integration.
/*
func TestHandleCalculatorWithTokens(t *testing.T) {
	...
}
*/
