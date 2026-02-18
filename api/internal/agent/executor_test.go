package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/api/internal/config"
	"github.com/aspectrr/fluid.sh/api/internal/orchestrator"
	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// ---------------------------------------------------------------------------
// mockSender implements orchestrator.HostSender
// ---------------------------------------------------------------------------

type mockSender struct{}

func (m *mockSender) SendAndWait(_ string, _ *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
	return nil, fmt.Errorf("mockSender not configured")
}

// ---------------------------------------------------------------------------
// mockDataStore implements store.Store (superset of DataStore) so it can be
// used for both the agent Client (DataStore) and the orchestrator (Store).
// Unconfigured methods return zero values instead of panicking.
// ---------------------------------------------------------------------------

type mockDataStore struct {
	// Sandbox
	GetSandboxFn           func(ctx context.Context, sandboxID string) (*store.Sandbox, error)
	ListSandboxesByOrgFn   func(ctx context.Context, orgID string) ([]store.Sandbox, error)
	GetSandboxesByHostIDFn func(ctx context.Context, hostID string) ([]store.Sandbox, error)

	// Playbooks
	CreatePlaybookFn     func(ctx context.Context, pb *store.Playbook) error
	GetPlaybookFn        func(ctx context.Context, id string) (*store.Playbook, error)
	ListPlaybooksByOrgFn func(ctx context.Context, orgID string) ([]*store.Playbook, error)
	DeletePlaybookFn     func(ctx context.Context, id string) error
	UpdatePlaybookFn     func(ctx context.Context, pb *store.Playbook) error

	// Playbook Tasks
	CreatePlaybookTaskFn   func(ctx context.Context, task *store.PlaybookTask) error
	GetPlaybookTaskFn      func(ctx context.Context, id string) (*store.PlaybookTask, error)
	ListPlaybookTasksFn    func(ctx context.Context, playbookID string) ([]*store.PlaybookTask, error)
	UpdatePlaybookTaskFn   func(ctx context.Context, task *store.PlaybookTask) error
	DeletePlaybookTaskFn   func(ctx context.Context, id string) error
	ReorderPlaybookTasksFn func(ctx context.Context, playbookID string, taskIDs []string) error
}

// Store lifecycle methods
func (m *mockDataStore) Config() store.Config       { return store.Config{} }
func (m *mockDataStore) Ping(context.Context) error { return nil }
func (m *mockDataStore) Close() error               { return nil }
func (m *mockDataStore) WithTx(_ context.Context, fn func(tx store.DataStore) error) error {
	return fn(m)
}

// User
func (m *mockDataStore) CreateUser(context.Context, *store.User) error               { return nil }
func (m *mockDataStore) GetUser(context.Context, string) (*store.User, error)        { return nil, nil }
func (m *mockDataStore) GetUserByEmail(context.Context, string) (*store.User, error) { return nil, nil }
func (m *mockDataStore) UpdateUser(context.Context, *store.User) error               { return nil }

// OAuth
func (m *mockDataStore) CreateOAuthAccount(context.Context, *store.OAuthAccount) error { return nil }
func (m *mockDataStore) GetOAuthAccount(context.Context, string, string) (*store.OAuthAccount, error) {
	return nil, nil
}
func (m *mockDataStore) GetOAuthAccountsByUser(context.Context, string) ([]*store.OAuthAccount, error) {
	return nil, nil
}

// Session
func (m *mockDataStore) CreateSession(context.Context, *store.Session) error        { return nil }
func (m *mockDataStore) GetSession(context.Context, string) (*store.Session, error) { return nil, nil }
func (m *mockDataStore) DeleteSession(context.Context, string) error                { return nil }
func (m *mockDataStore) DeleteExpiredSessions(context.Context) error                { return nil }

// Organization
func (m *mockDataStore) CreateOrganization(context.Context, *store.Organization) error { return nil }
func (m *mockDataStore) GetOrganization(context.Context, string) (*store.Organization, error) {
	return nil, nil
}
func (m *mockDataStore) GetOrganizationBySlug(context.Context, string) (*store.Organization, error) {
	return nil, nil
}
func (m *mockDataStore) ListOrganizationsByUser(context.Context, string) ([]*store.Organization, error) {
	return nil, nil
}
func (m *mockDataStore) UpdateOrganization(context.Context, *store.Organization) error { return nil }
func (m *mockDataStore) DeleteOrganization(context.Context, string) error              { return nil }

// OrgMember
func (m *mockDataStore) CreateOrgMember(context.Context, *store.OrgMember) error { return nil }
func (m *mockDataStore) GetOrgMember(context.Context, string, string) (*store.OrgMember, error) {
	return nil, nil
}
func (m *mockDataStore) ListOrgMembers(context.Context, string) ([]*store.OrgMember, error) {
	return nil, nil
}
func (m *mockDataStore) DeleteOrgMember(context.Context, string, string) error { return nil }

// Subscription
func (m *mockDataStore) CreateSubscription(context.Context, *store.Subscription) error { return nil }
func (m *mockDataStore) GetSubscriptionByOrg(context.Context, string) (*store.Subscription, error) {
	return nil, nil
}
func (m *mockDataStore) UpdateSubscription(context.Context, *store.Subscription) error { return nil }

// Usage
func (m *mockDataStore) CreateUsageRecord(context.Context, *store.UsageRecord) error { return nil }
func (m *mockDataStore) ListUsageRecords(context.Context, string, time.Time, time.Time) ([]*store.UsageRecord, error) {
	return nil, nil
}

// Host
func (m *mockDataStore) CreateHost(context.Context, *store.Host) error        { return nil }
func (m *mockDataStore) GetHost(context.Context, string) (*store.Host, error) { return nil, nil }
func (m *mockDataStore) ListHosts(context.Context) ([]store.Host, error)      { return nil, nil }
func (m *mockDataStore) UpdateHost(context.Context, *store.Host) error        { return nil }
func (m *mockDataStore) UpdateHostHeartbeat(context.Context, string, int32, int64, int64) error {
	return nil
}

// Sandbox
func (m *mockDataStore) CreateSandbox(context.Context, *store.Sandbox) error { return nil }
func (m *mockDataStore) GetSandbox(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
	if m.GetSandboxFn != nil {
		return m.GetSandboxFn(ctx, sandboxID)
	}
	return nil, nil
}
func (m *mockDataStore) ListSandboxes(context.Context) ([]store.Sandbox, error) { return nil, nil }
func (m *mockDataStore) ListSandboxesByOrg(ctx context.Context, orgID string) ([]store.Sandbox, error) {
	if m.ListSandboxesByOrgFn != nil {
		return m.ListSandboxesByOrgFn(ctx, orgID)
	}
	return nil, nil
}
func (m *mockDataStore) UpdateSandbox(context.Context, *store.Sandbox) error { return nil }
func (m *mockDataStore) DeleteSandbox(context.Context, string) error         { return nil }
func (m *mockDataStore) GetSandboxesByHostID(ctx context.Context, hostID string) ([]store.Sandbox, error) {
	if m.GetSandboxesByHostIDFn != nil {
		return m.GetSandboxesByHostIDFn(ctx, hostID)
	}
	return nil, nil
}
func (m *mockDataStore) ListExpiredSandboxes(context.Context, time.Duration) ([]store.Sandbox, error) {
	return nil, nil
}

// Command
func (m *mockDataStore) CreateCommand(context.Context, *store.Command) error { return nil }
func (m *mockDataStore) ListSandboxCommands(context.Context, string) ([]store.Command, error) {
	return nil, nil
}

// SourceHost
func (m *mockDataStore) CreateSourceHost(context.Context, *store.SourceHost) error { return nil }
func (m *mockDataStore) GetSourceHost(context.Context, string) (*store.SourceHost, error) {
	return nil, nil
}
func (m *mockDataStore) ListSourceHostsByOrg(context.Context, string) ([]*store.SourceHost, error) {
	return nil, nil
}
func (m *mockDataStore) DeleteSourceHost(context.Context, string) error { return nil }

// HostToken
func (m *mockDataStore) CreateHostToken(context.Context, *store.HostToken) error { return nil }
func (m *mockDataStore) GetHostTokenByHash(context.Context, string) (*store.HostToken, error) {
	return nil, nil
}
func (m *mockDataStore) ListHostTokensByOrg(context.Context, string) ([]store.HostToken, error) {
	return nil, nil
}
func (m *mockDataStore) DeleteHostToken(context.Context, string, string) error { return nil }

// Agent Conversations
func (m *mockDataStore) CreateAgentConversation(context.Context, *store.AgentConversation) error {
	return nil
}
func (m *mockDataStore) GetAgentConversation(context.Context, string) (*store.AgentConversation, error) {
	return nil, nil
}
func (m *mockDataStore) ListAgentConversationsByOrg(context.Context, string) ([]*store.AgentConversation, error) {
	return nil, nil
}
func (m *mockDataStore) DeleteAgentConversation(context.Context, string) error { return nil }

// Agent Messages
func (m *mockDataStore) CreateAgentMessage(context.Context, *store.AgentMessage) error { return nil }
func (m *mockDataStore) ListAgentMessages(context.Context, string) ([]*store.AgentMessage, error) {
	return nil, nil
}

// Playbooks
func (m *mockDataStore) CreatePlaybook(ctx context.Context, pb *store.Playbook) error {
	if m.CreatePlaybookFn != nil {
		return m.CreatePlaybookFn(ctx, pb)
	}
	return nil
}
func (m *mockDataStore) GetPlaybook(ctx context.Context, id string) (*store.Playbook, error) {
	if m.GetPlaybookFn != nil {
		return m.GetPlaybookFn(ctx, id)
	}
	return nil, nil
}
func (m *mockDataStore) ListPlaybooksByOrg(ctx context.Context, orgID string) ([]*store.Playbook, error) {
	if m.ListPlaybooksByOrgFn != nil {
		return m.ListPlaybooksByOrgFn(ctx, orgID)
	}
	return nil, nil
}
func (m *mockDataStore) UpdatePlaybook(ctx context.Context, pb *store.Playbook) error {
	if m.UpdatePlaybookFn != nil {
		return m.UpdatePlaybookFn(ctx, pb)
	}
	return nil
}
func (m *mockDataStore) DeletePlaybook(ctx context.Context, id string) error {
	if m.DeletePlaybookFn != nil {
		return m.DeletePlaybookFn(ctx, id)
	}
	return nil
}

// Playbook Tasks
func (m *mockDataStore) CreatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	if m.CreatePlaybookTaskFn != nil {
		return m.CreatePlaybookTaskFn(ctx, task)
	}
	return nil
}
func (m *mockDataStore) GetPlaybookTask(ctx context.Context, id string) (*store.PlaybookTask, error) {
	if m.GetPlaybookTaskFn != nil {
		return m.GetPlaybookTaskFn(ctx, id)
	}
	return nil, nil
}
func (m *mockDataStore) ListPlaybookTasks(ctx context.Context, playbookID string) ([]*store.PlaybookTask, error) {
	if m.ListPlaybookTasksFn != nil {
		return m.ListPlaybookTasksFn(ctx, playbookID)
	}
	return nil, nil
}
func (m *mockDataStore) UpdatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	if m.UpdatePlaybookTaskFn != nil {
		return m.UpdatePlaybookTaskFn(ctx, task)
	}
	return nil
}
func (m *mockDataStore) DeletePlaybookTask(ctx context.Context, id string) error {
	if m.DeletePlaybookTaskFn != nil {
		return m.DeletePlaybookTaskFn(ctx, id)
	}
	return nil
}
func (m *mockDataStore) ReorderPlaybookTasks(ctx context.Context, playbookID string, taskIDs []string) error {
	if m.ReorderPlaybookTasksFn != nil {
		return m.ReorderPlaybookTasksFn(ctx, playbookID, taskIDs)
	}
	return nil
}

// Billing helpers
func (m *mockDataStore) GetOrganizationByStripeCustomerID(context.Context, string) (*store.Organization, error) {
	return nil, nil
}
func (m *mockDataStore) GetModelMeter(context.Context, string) (*store.ModelMeter, error) {
	return nil, store.ErrNotFound
}
func (m *mockDataStore) CreateModelMeter(context.Context, *store.ModelMeter) error { return nil }
func (m *mockDataStore) GetOrgModelSubscription(context.Context, string, string) (*store.OrgModelSubscription, error) {
	return nil, store.ErrNotFound
}
func (m *mockDataStore) CreateOrgModelSubscription(context.Context, *store.OrgModelSubscription) error {
	return nil
}
func (m *mockDataStore) SumTokenUsage(context.Context, string, time.Time, time.Time) (float64, error) {
	return 0, nil
}
func (m *mockDataStore) ListActiveSubscriptions(context.Context) ([]*store.Subscription, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestClient(ms *mockDataStore) *Client {
	reg := registry.New()
	orch := orchestrator.New(reg, ms, &mockSender{}, nil, 24*time.Hour, 90*time.Second)
	return NewClient(config.AgentConfig{}, ms, orch, nil)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestExecuteTool_UnknownTool(t *testing.T) {
	ms := &mockDataStore{}
	client := newTestClient(ms)

	result, err := client.ExecuteTool(context.Background(), "org-1", "foo_bar", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "unknown tool: foo_bar") {
		t.Errorf("result = %q, want it to contain %q", result, "unknown tool: foo_bar")
	}
}

func TestExecuteTool_ListSandboxes(t *testing.T) {
	ms := &mockDataStore{
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
	client := newTestClient(ms)

	result, err := client.ExecuteTool(context.Background(), "org-1", "list_sandboxes", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	count, ok := parsed["count"].(float64)
	if !ok {
		t.Fatalf("missing or non-numeric 'count' in result: %v", parsed)
	}
	if int(count) != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

func TestExecuteTool_GetSandbox(t *testing.T) {
	ms := &mockDataStore{
		GetSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			if id == "sbx-1" {
				return &store.Sandbox{
					ID:    "sbx-1",
					OrgID: "org-1",
					Name:  "my-sandbox",
					State: store.SandboxStateRunning,
				}, nil
			}
			return nil, store.ErrNotFound
		},
	}
	client := newTestClient(ms)

	args := json.RawMessage(`{"sandbox_id":"sbx-1"}`)
	result, err := client.ExecuteTool(context.Background(), "org-1", "get_sandbox", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	if parsed["id"] != "sbx-1" {
		t.Errorf("id = %v, want %q", parsed["id"], "sbx-1")
	}
}

func TestExecuteTool_CreatePlaybook(t *testing.T) {
	var captured *store.Playbook
	ms := &mockDataStore{
		CreatePlaybookFn: func(_ context.Context, pb *store.Playbook) error {
			captured = pb
			return nil
		},
	}
	client := newTestClient(ms)

	args := json.RawMessage(`{"name":"test-pb"}`)
	result, err := client.ExecuteTool(context.Background(), "org-1", "create_playbook", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured == nil {
		t.Fatal("CreatePlaybook was not called")
	}
	if captured.Name != "test-pb" {
		t.Errorf("captured name = %q, want %q", captured.Name, "test-pb")
	}
	if captured.OrgID != "org-1" {
		t.Errorf("captured org_id = %q, want %q", captured.OrgID, "org-1")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	if parsed["name"] != "test-pb" {
		t.Errorf("result name = %v, want %q", parsed["name"], "test-pb")
	}
}

func TestExecuteTool_DeletePlaybook(t *testing.T) {
	var deletedID string
	ms := &mockDataStore{
		DeletePlaybookFn: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	client := newTestClient(ms)

	args := json.RawMessage(`{"playbook_id":"pb-1"}`)
	result, err := client.ExecuteTool(context.Background(), "org-1", "delete_playbook", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != "pb-1" {
		t.Errorf("deleted ID = %q, want %q", deletedID, "pb-1")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	if parsed["deleted"] != true {
		t.Errorf("deleted = %v, want true", parsed["deleted"])
	}
}

func TestExecuteTool_ListPlaybooks(t *testing.T) {
	ms := &mockDataStore{
		ListPlaybooksByOrgFn: func(_ context.Context, orgID string) ([]*store.Playbook, error) {
			if orgID != "org-1" {
				t.Errorf("orgID = %q, want %q", orgID, "org-1")
			}
			return []*store.Playbook{
				{ID: "pb-1", OrgID: "org-1", Name: "playbook-1"},
				{ID: "pb-2", OrgID: "org-1", Name: "playbook-2"},
			}, nil
		},
	}
	client := newTestClient(ms)

	result, err := client.ExecuteTool(context.Background(), "org-1", "list_playbooks", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	count, ok := parsed["count"].(float64)
	if !ok {
		t.Fatalf("missing or non-numeric 'count' in result: %v", parsed)
	}
	if int(count) != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

func TestExecuteTool_AddPlaybookTask(t *testing.T) {
	var capturedTask *store.PlaybookTask
	ms := &mockDataStore{
		ListPlaybookTasksFn: func(_ context.Context, _ string) ([]*store.PlaybookTask, error) {
			return nil, nil // empty tasks list
		},
		CreatePlaybookTaskFn: func(_ context.Context, task *store.PlaybookTask) error {
			capturedTask = task
			return nil
		},
	}
	client := newTestClient(ms)

	args := json.RawMessage(`{"playbook_id":"pb-1","name":"Install","module":"apt"}`)
	result, err := client.ExecuteTool(context.Background(), "org-1", "add_playbook_task", args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTask == nil {
		t.Fatal("CreatePlaybookTask was not called")
	}
	if capturedTask.PlaybookID != "pb-1" {
		t.Errorf("playbook_id = %q, want %q", capturedTask.PlaybookID, "pb-1")
	}
	if capturedTask.Name != "Install" {
		t.Errorf("name = %q, want %q", capturedTask.Name, "Install")
	}
	if capturedTask.Module != "apt" {
		t.Errorf("module = %q, want %q", capturedTask.Module, "apt")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	if parsed["module"] != "apt" {
		t.Errorf("result module = %v, want %q", parsed["module"], "apt")
	}
}

func TestExecuteTool_InvalidArgs(t *testing.T) {
	ms := &mockDataStore{}
	client := newTestClient(ms)

	_, err := client.ExecuteTool(context.Background(), "org-1", "list_sandboxes", []byte("{bad json"))
	if err == nil {
		t.Fatal("expected Go error for invalid JSON args")
	}
	if !strings.Contains(err.Error(), "invalid tool arguments") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "invalid tool arguments")
	}
}
