package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/sandbox"
	"github.com/aspectrr/fluid.sh/fluid/internal/store"
)

// --- helpers ---

func newRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

func parseJSON(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	text, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected TextContent")
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text.Text), &m))
	return m
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testConfig() *config.Config {
	return &config.Config{
		SSH: config.SSHConfig{
			DefaultUser: "sandbox",
		},
		Ansible: config.AnsibleConfig{
			PlaybooksDir: "/tmp/playbooks",
		},
	}
}

// --- mock store (for playbooks) ---

type mockStore struct {
	sandboxes       map[string]*store.Sandbox
	listSandboxesFn func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error)
}

func newMockStore() *mockStore {
	return &mockStore{
		sandboxes: make(map[string]*store.Sandbox),
	}
}

func (m *mockStore) Config() store.Config           { return store.Config{} }
func (m *mockStore) Ping(ctx context.Context) error { return nil }
func (m *mockStore) Close() error                   { return nil }
func (m *mockStore) WithTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	return fn(m)
}

func (m *mockStore) CreateSandbox(ctx context.Context, sb *store.Sandbox) error {
	m.sandboxes[sb.ID] = sb
	return nil
}

func (m *mockStore) GetSandbox(ctx context.Context, id string) (*store.Sandbox, error) {
	sb, ok := m.sandboxes[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return sb, nil
}

func (m *mockStore) GetSandboxByVMName(ctx context.Context, vmName string) (*store.Sandbox, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListSandboxes(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
	if m.listSandboxesFn != nil {
		return m.listSandboxesFn(ctx, filter, opt)
	}
	result := make([]*store.Sandbox, 0, len(m.sandboxes))
	for _, sb := range m.sandboxes {
		result = append(result, sb)
	}
	return result, nil
}

func (m *mockStore) UpdateSandbox(ctx context.Context, sb *store.Sandbox) error { return nil }
func (m *mockStore) UpdateSandboxState(ctx context.Context, id string, newState store.SandboxState, ipAddr *string) error {
	return nil
}
func (m *mockStore) DeleteSandbox(ctx context.Context, id string) error { return nil }
func (m *mockStore) CreateSnapshot(ctx context.Context, sn *store.Snapshot) error {
	return nil
}

func (m *mockStore) GetSnapshot(ctx context.Context, id string) (*store.Snapshot, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetSnapshotByName(ctx context.Context, sandboxID, name string) (*store.Snapshot, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListSnapshots(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Snapshot, error) {
	return nil, nil
}

func (m *mockStore) SaveCommand(ctx context.Context, cmd *store.Command) error { return nil }
func (m *mockStore) GetCommand(ctx context.Context, id string) (*store.Command, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListCommands(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Command, error) {
	return nil, nil
}
func (m *mockStore) SaveDiff(ctx context.Context, d *store.Diff) error { return nil }
func (m *mockStore) GetDiff(ctx context.Context, id string) (*store.Diff, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetDiffBySnapshots(ctx context.Context, sandboxID, fromSnapshot, toSnapshot string) (*store.Diff, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) CreateChangeSet(ctx context.Context, cs *store.ChangeSet) error { return nil }
func (m *mockStore) GetChangeSet(ctx context.Context, id string) (*store.ChangeSet, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetChangeSetByJob(ctx context.Context, jobID string) (*store.ChangeSet, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) CreatePublication(ctx context.Context, p *store.Publication) error { return nil }
func (m *mockStore) UpdatePublicationStatus(ctx context.Context, id string, status store.PublicationStatus, commitSHA, prURL, errMsg *string) error {
	return nil
}

func (m *mockStore) GetPublication(ctx context.Context, id string) (*store.Publication, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) CreatePlaybook(ctx context.Context, pb *store.Playbook) error { return nil }
func (m *mockStore) GetPlaybook(ctx context.Context, id string) (*store.Playbook, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetPlaybookByName(ctx context.Context, name string) (*store.Playbook, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListPlaybooks(ctx context.Context, opt *store.ListOptions) ([]*store.Playbook, error) {
	return nil, nil
}
func (m *mockStore) UpdatePlaybook(ctx context.Context, pb *store.Playbook) error { return nil }
func (m *mockStore) DeletePlaybook(ctx context.Context, id string) error          { return nil }
func (m *mockStore) CreatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	return nil
}

func (m *mockStore) GetPlaybookTask(ctx context.Context, id string) (*store.PlaybookTask, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListPlaybookTasks(ctx context.Context, playbookID string, opt *store.ListOptions) ([]*store.PlaybookTask, error) {
	return nil, nil
}

func (m *mockStore) UpdatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	return nil
}
func (m *mockStore) DeletePlaybookTask(ctx context.Context, id string) error { return nil }
func (m *mockStore) ReorderPlaybookTasks(ctx context.Context, playbookID string, taskIDs []string) error {
	return nil
}

func (m *mockStore) GetNextTaskPosition(ctx context.Context, playbookID string) (int, error) {
	return 0, nil
}

func (m *mockStore) GetSourceVM(ctx context.Context, name string) (*store.SourceVM, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) UpsertSourceVM(ctx context.Context, svm *store.SourceVM) error { return nil }
func (m *mockStore) ListSourceVMs(ctx context.Context) ([]*store.SourceVM, error)  { return nil, nil }

// --- mock sandbox.Service ---

type mockSandboxService struct {
	listSandboxesFn    func(ctx context.Context) ([]*sandbox.SandboxInfo, error)
	createSandboxFn    func(ctx context.Context, req sandbox.CreateRequest) (*sandbox.SandboxInfo, error)
	getSandboxFn       func(ctx context.Context, id string) (*sandbox.SandboxInfo, error)
	destroySandboxFn   func(ctx context.Context, id string) error
	startSandboxFn     func(ctx context.Context, id string) (*sandbox.SandboxInfo, error)
	stopSandboxFn      func(ctx context.Context, id string, force bool) error
	runCommandFn       func(ctx context.Context, sandboxID, command string, timeoutSec int, env map[string]string) (*sandbox.CommandResult, error)
	createSnapshotFn   func(ctx context.Context, sandboxID, name string) (*sandbox.SnapshotInfo, error)
	listVMsFn          func(ctx context.Context) ([]*sandbox.VMInfo, error)
	runSourceCommandFn func(ctx context.Context, vmName, command string, timeoutSec int) (*sandbox.SourceCommandResult, error)
	readSourceFileFn   func(ctx context.Context, vmName, path string) (string, error)
}

func (m *mockSandboxService) CreateSandbox(ctx context.Context, req sandbox.CreateRequest) (*sandbox.SandboxInfo, error) {
	if m.createSandboxFn != nil {
		return m.createSandboxFn(ctx, req)
	}
	return &sandbox.SandboxInfo{ID: "SBX-new", Name: "sbx-new", State: "RUNNING"}, nil
}

func (m *mockSandboxService) GetSandbox(ctx context.Context, id string) (*sandbox.SandboxInfo, error) {
	if m.getSandboxFn != nil {
		return m.getSandboxFn(ctx, id)
	}
	return nil, fmt.Errorf("sandbox not found: %s", id)
}

func (m *mockSandboxService) ListSandboxes(ctx context.Context) ([]*sandbox.SandboxInfo, error) {
	if m.listSandboxesFn != nil {
		return m.listSandboxesFn(ctx)
	}
	return nil, nil
}

func (m *mockSandboxService) DestroySandbox(ctx context.Context, id string) error {
	if m.destroySandboxFn != nil {
		return m.destroySandboxFn(ctx, id)
	}
	return nil
}

func (m *mockSandboxService) StartSandbox(ctx context.Context, id string) (*sandbox.SandboxInfo, error) {
	if m.startSandboxFn != nil {
		return m.startSandboxFn(ctx, id)
	}
	return &sandbox.SandboxInfo{ID: id, State: "RUNNING"}, nil
}

func (m *mockSandboxService) StopSandbox(ctx context.Context, id string, force bool) error {
	if m.stopSandboxFn != nil {
		return m.stopSandboxFn(ctx, id, force)
	}
	return nil
}

func (m *mockSandboxService) RunCommand(ctx context.Context, sandboxID, command string, timeoutSec int, env map[string]string) (*sandbox.CommandResult, error) {
	if m.runCommandFn != nil {
		return m.runCommandFn(ctx, sandboxID, command, timeoutSec, env)
	}
	return &sandbox.CommandResult{SandboxID: sandboxID, ExitCode: 0}, nil
}

func (m *mockSandboxService) CreateSnapshot(ctx context.Context, sandboxID, name string) (*sandbox.SnapshotInfo, error) {
	if m.createSnapshotFn != nil {
		return m.createSnapshotFn(ctx, sandboxID, name)
	}
	return &sandbox.SnapshotInfo{SnapshotID: "SNAP-1", SnapshotName: name, SandboxID: sandboxID}, nil
}

func (m *mockSandboxService) ListVMs(ctx context.Context) ([]*sandbox.VMInfo, error) {
	if m.listVMsFn != nil {
		return m.listVMsFn(ctx)
	}
	return nil, nil
}

func (m *mockSandboxService) ValidateSourceVM(ctx context.Context, vmName string) (*sandbox.ValidationInfo, error) {
	return &sandbox.ValidationInfo{VMName: vmName, Valid: true}, nil
}

func (m *mockSandboxService) PrepareSourceVM(ctx context.Context, vmName, sshUser, keyPath string) (*sandbox.PrepareInfo, error) {
	return &sandbox.PrepareInfo{SourceVM: vmName, Prepared: true}, nil
}

func (m *mockSandboxService) RunSourceCommand(ctx context.Context, vmName, command string, timeoutSec int) (*sandbox.SourceCommandResult, error) {
	if m.runSourceCommandFn != nil {
		return m.runSourceCommandFn(ctx, vmName, command, timeoutSec)
	}
	return &sandbox.SourceCommandResult{SourceVM: vmName, ExitCode: 0}, nil
}

func (m *mockSandboxService) ReadSourceFile(ctx context.Context, vmName, path string) (string, error) {
	if m.readSourceFileFn != nil {
		return m.readSourceFileFn(ctx, vmName, path)
	}
	return "", nil
}

func (m *mockSandboxService) GetHostInfo(ctx context.Context) (*sandbox.HostInfo, error) {
	return &sandbox.HostInfo{}, nil
}

func (m *mockSandboxService) Health(ctx context.Context) error { return nil }
func (m *mockSandboxService) Close() error                     { return nil }

// --- test server helpers ---

func testServer() *Server {
	st := newMockStore()
	cfg := testConfig()
	return &Server{
		cfg:     cfg,
		store:   st,
		service: &mockSandboxService{},
		logger:  noopLogger(),
	}
}

func testServerWithService(svc sandbox.Service) *Server {
	st := newMockStore()
	cfg := testConfig()
	return &Server{
		cfg:     cfg,
		store:   st,
		service: svc,
		logger:  noopLogger(),
	}
}

// --- mock telemetry ---

type mockTelemetry struct {
	mu     sync.Mutex
	events []telemetryEvent
}

type telemetryEvent struct {
	name       string
	properties map[string]any
}

func (m *mockTelemetry) Track(event string, properties map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, telemetryEvent{name: event, properties: properties})
}

func (m *mockTelemetry) Close() {}

func (m *mockTelemetry) getEvents() []telemetryEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]telemetryEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

// --- trackToolCall tests ---

func TestTrackToolCall(t *testing.T) {
	mt := &mockTelemetry{}
	srv := &Server{
		telemetry: mt,
		logger:    noopLogger(),
	}

	srv.trackToolCall("list_sandboxes")

	events := mt.getEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "mcp_tool_call", events[0].name)
	assert.Equal(t, "list_sandboxes", events[0].properties["tool_name"])
}

func TestTrackToolCall_NilTelemetry(t *testing.T) {
	srv := &Server{
		telemetry: nil,
		logger:    noopLogger(),
	}

	// Should not panic with nil telemetry
	srv.trackToolCall("list_sandboxes")
}

func TestTrackToolCall_HandlerIntegration(t *testing.T) {
	mt := &mockTelemetry{}
	svc := &mockSandboxService{}
	srv := &Server{
		cfg:       testConfig(),
		store:     newMockStore(),
		service:   svc,
		telemetry: mt,
		logger:    noopLogger(),
	}
	ctx := context.Background()

	_, _ = srv.handleListSandboxes(ctx, newRequest("list_sandboxes", nil))

	events := mt.getEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "mcp_tool_call", events[0].name)
	assert.Equal(t, "list_sandboxes", events[0].properties["tool_name"])
}

// --- shellEscape tests ---

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "'simple'"},
		{"with spaces", "'with spaces'"},
		{"with'quote", "'with'\\''quote'"},
		{"/etc/passwd", "'/etc/passwd'"},
		{"'; rm -rf /; echo '", "''\\''; rm -rf /; echo '\\'''"},
		{"$HOME", "'$HOME'"},
		{"`id`", "'`id`'"},
		{"file$(whoami)", "'file$(whoami)'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := shellEscape(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShellEscape_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"null byte", "hello\x00world"},
		{"control char", "hello\x07world"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := shellEscape(tt.input)
			assert.Error(t, err)
		})
	}
}

// --- jsonResult tests ---

func TestJsonResult(t *testing.T) {
	result, err := jsonResult(map[string]any{"key": "value"})
	require.NoError(t, err)
	require.NotNil(t, result)

	m := parseJSON(t, result)
	assert.Equal(t, "value", m["key"])
}

func TestJsonResult_Error(t *testing.T) {
	// A channel is not JSON-serializable
	_, err := jsonResult(make(chan int))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal result")
}

// --- errorResult tests ---

func TestErrorResult(t *testing.T) {
	result, err := errorResult(map[string]any{"sandbox_id": "SBX-1", "error": "something failed"})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "expected IsError to be true")

	m := parseJSON(t, result)
	assert.Equal(t, "SBX-1", m["sandbox_id"])
	assert.Equal(t, "something failed", m["error"])
}

func TestErrorResult_MarshalError(t *testing.T) {
	_, err := errorResult(make(chan int))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal error result")
}

// --- handleListSandboxes tests ---

func TestHandleListSandboxes_Empty(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	result, err := srv.handleListSandboxes(ctx, newRequest("list_sandboxes", nil))
	require.NoError(t, err)

	m := parseJSON(t, result)
	assert.Equal(t, float64(0), m["count"])
	sandboxes, ok := m["sandboxes"].([]any)
	require.True(t, ok)
	assert.Empty(t, sandboxes)
}

func TestHandleListSandboxes_WithSandboxes(t *testing.T) {
	now := time.Now()
	svc := &mockSandboxService{
		listSandboxesFn: func(ctx context.Context) ([]*sandbox.SandboxInfo, error) {
			return []*sandbox.SandboxInfo{
				{
					ID:        "SBX-1",
					Name:      "sbx-test",
					State:     "RUNNING",
					BaseImage: "ubuntu-base",
					IPAddress: "192.168.1.10",
					CreatedAt: now,
				},
			}, nil
		},
	}
	srv := testServerWithService(svc)
	ctx := context.Background()

	result, err := srv.handleListSandboxes(ctx, newRequest("list_sandboxes", nil))
	require.NoError(t, err)

	m := parseJSON(t, result)
	assert.Equal(t, float64(1), m["count"])
	sandboxes := m["sandboxes"].([]any)
	sb := sandboxes[0].(map[string]any)
	assert.Equal(t, "SBX-1", sb["id"])
	assert.Equal(t, "sbx-test", sb["name"])
	assert.Equal(t, "RUNNING", sb["state"])
	assert.Equal(t, "192.168.1.10", sb["ip"])
}

func TestHandleListSandboxes_StoreError(t *testing.T) {
	svc := &mockSandboxService{
		listSandboxesFn: func(ctx context.Context) ([]*sandbox.SandboxInfo, error) {
			return nil, fmt.Errorf("db connection failed")
		},
	}
	srv := testServerWithService(svc)
	ctx := context.Background()

	result, err := srv.handleListSandboxes(ctx, newRequest("list_sandboxes", nil))
	require.NoError(t, err)
	require.True(t, result.IsError, "expected IsError to be true")
	m := parseJSON(t, result)
	assert.Contains(t, m["error"], "list sandboxes")
}

// --- handleGetSandbox tests ---

func TestHandleGetSandbox_Success(t *testing.T) {
	now := time.Now()
	svc := &mockSandboxService{
		getSandboxFn: func(ctx context.Context, id string) (*sandbox.SandboxInfo, error) {
			return &sandbox.SandboxInfo{
				ID:        "SBX-1",
				Name:      "sbx-test",
				State:     "RUNNING",
				BaseImage: "ubuntu-base",
				AgentID:   "mcp-agent",
				IPAddress: "192.168.1.10",
				CreatedAt: now,
			}, nil
		},
	}
	srv := testServerWithService(svc)
	ctx := context.Background()

	result, err := srv.handleGetSandbox(ctx, newRequest("get_sandbox", map[string]any{
		"sandbox_id": "SBX-1",
	}))
	require.NoError(t, err)

	m := parseJSON(t, result)
	assert.Equal(t, "SBX-1", m["sandbox_id"])
	assert.Equal(t, "sbx-test", m["name"])
	assert.Equal(t, "RUNNING", m["state"])
	assert.Equal(t, "ubuntu-base", m["base_image"])
	assert.Equal(t, "mcp-agent", m["agent_id"])
	assert.Equal(t, "192.168.1.10", m["ip"])
}

func TestHandleGetSandbox_MissingID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleGetSandbox(ctx, newRequest("get_sandbox", map[string]any{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox_id is required")
}

func TestHandleGetSandbox_NotFound(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	result, err := srv.handleGetSandbox(ctx, newRequest("get_sandbox", map[string]any{
		"sandbox_id": "SBX-nonexistent",
	}))
	require.NoError(t, err)
	require.True(t, result.IsError, "expected IsError to be true")
	m := parseJSON(t, result)
	assert.Contains(t, m["error"], "get sandbox")
	assert.Equal(t, "SBX-nonexistent", m["sandbox_id"])
}

// --- handleDestroySandbox tests ---

func TestHandleDestroySandbox_MissingID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleDestroySandbox(ctx, newRequest("destroy_sandbox", map[string]any{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox_id is required")
}

// --- handleStartSandbox tests ---

func TestHandleStartSandbox_MissingID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleStartSandbox(ctx, newRequest("start_sandbox", map[string]any{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox_id is required")
}

// --- handleStopSandbox tests ---

func TestHandleStopSandbox_MissingID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleStopSandbox(ctx, newRequest("stop_sandbox", map[string]any{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox_id is required")
}

// --- handleCreateSandbox tests ---

func TestHandleCreateSandbox_MissingSourceVM(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleCreateSandbox(ctx, newRequest("create_sandbox", map[string]any{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source_vm is required")
}

// --- handleRunCommand tests ---

func TestHandleRunCommand_MissingSandboxID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleRunCommand(ctx, newRequest("run_command", map[string]any{
		"command": "ls",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox_id is required")
}

func TestHandleRunCommand_MissingCommand(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleRunCommand(ctx, newRequest("run_command", map[string]any{
		"sandbox_id": "SBX-1",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
}

// --- handleCreateSnapshot tests ---

func TestHandleCreateSnapshot_MissingSandboxID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleCreateSnapshot(ctx, newRequest("create_snapshot", map[string]any{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox_id is required")
}

// --- handleCreatePlaybook tests ---

func TestHandleCreatePlaybook_MissingName(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleCreatePlaybook(ctx, newRequest("create_playbook", map[string]any{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

// --- handleAddPlaybookTask tests ---

func TestHandleAddPlaybookTask_MissingPlaybookID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleAddPlaybookTask(ctx, newRequest("add_playbook_task", map[string]any{
		"name":   "Install nginx",
		"module": "apt",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "playbook_id is required")
}

func TestHandleAddPlaybookTask_MissingName(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleAddPlaybookTask(ctx, newRequest("add_playbook_task", map[string]any{
		"playbook_id": "PB-1",
		"module":      "apt",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestHandleAddPlaybookTask_MissingModule(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleAddPlaybookTask(ctx, newRequest("add_playbook_task", map[string]any{
		"playbook_id": "PB-1",
		"name":        "Install nginx",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module is required")
}

// --- handleEditFile tests ---

func TestHandleEditFile_MissingSandboxID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleEditFile(ctx, newRequest("edit_file", map[string]any{
		"path":    "/etc/config",
		"new_str": "content",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox_id is required")
}

func TestHandleEditFile_MissingPath(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleEditFile(ctx, newRequest("edit_file", map[string]any{
		"sandbox_id": "SBX-1",
		"new_str":    "content",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestHandleEditFile_RelativePath(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleEditFile(ctx, newRequest("edit_file", map[string]any{
		"sandbox_id": "SBX-1",
		"path":       "relative/path",
		"new_str":    "content",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")
}

// --- handleReadFile tests ---

func TestHandleReadFile_MissingSandboxID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleReadFile(ctx, newRequest("read_file", map[string]any{
		"path": "/etc/config",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox_id is required")
}

func TestHandleReadFile_MissingPath(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleReadFile(ctx, newRequest("read_file", map[string]any{
		"sandbox_id": "SBX-1",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestHandleReadFile_RelativePath(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleReadFile(ctx, newRequest("read_file", map[string]any{
		"sandbox_id": "SBX-1",
		"path":       "relative/path",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")
}

// --- handleGetPlaybook tests ---

func TestHandleGetPlaybook_MissingID(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleGetPlaybook(ctx, newRequest("get_playbook", map[string]any{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "playbook_id is required")
}

// --- handleRunSourceCommand tests ---

func TestHandleRunSourceCommand_MissingSourceVM(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleRunSourceCommand(ctx, newRequest("run_source_command", map[string]any{
		"command": "ls",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source_vm is required")
}

func TestHandleRunSourceCommand_MissingCommand(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleRunSourceCommand(ctx, newRequest("run_source_command", map[string]any{
		"source_vm": "ubuntu-base",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command is required")
}

// --- handleReadSourceFile tests ---

func TestHandleReadSourceFile_MissingSourceVM(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleReadSourceFile(ctx, newRequest("read_source_file", map[string]any{
		"path": "/etc/passwd",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source_vm is required")
}

func TestHandleReadSourceFile_MissingPath(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleReadSourceFile(ctx, newRequest("read_source_file", map[string]any{
		"source_vm": "ubuntu-base",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestHandleReadSourceFile_RelativePath(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleReadSourceFile(ctx, newRequest("read_source_file", map[string]any{
		"source_vm": "ubuntu-base",
		"path":      "relative/path",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")
}

// --- handleListPlaybooks tests ---

func TestHandleListPlaybooks_Empty(t *testing.T) {
	st := newMockStore()
	cfg := testConfig()
	srv := NewServer(cfg, st, nil, nil, noopLogger())
	ctx := context.Background()

	result, err := srv.handleListPlaybooks(ctx, newRequest("list_playbooks", nil))
	require.NoError(t, err)

	m := parseJSON(t, result)
	assert.Equal(t, float64(0), m["count"])
}

func TestHandleListPlaybooks_NoPlaybooksDir(t *testing.T) {
	st := newMockStore()
	cfg := testConfig()
	cfg.Ansible.PlaybooksDir = ""
	srv := NewServer(cfg, st, nil, nil, noopLogger())
	ctx := context.Background()

	result, err := srv.handleListPlaybooks(ctx, newRequest("list_playbooks", nil))
	require.NoError(t, err)

	m := parseJSON(t, result)
	assert.Equal(t, float64(0), m["count"])
}

// --- handleListVMs tests ---

func TestHandleListVMs_Empty(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	result, err := srv.handleListVMs(ctx, newRequest("list_vms", nil))
	require.NoError(t, err)

	m := parseJSON(t, result)
	assert.Equal(t, float64(0), m["count"])
}

// --- security tests ---

func TestHandleEditFile_NullByteInPath(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleEditFile(ctx, newRequest("edit_file", map[string]any{
		"sandbox_id": "SBX-1",
		"path":       "/etc/config\x00evil",
		"new_str":    "content",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")
}

func TestHandleReadFile_NullByteInPath(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	_, err := srv.handleReadFile(ctx, newRequest("read_file", map[string]any{
		"sandbox_id": "SBX-1",
		"path":       "/etc/config\x00evil",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")
}

func TestHandleEditFile_FileTooLarge(t *testing.T) {
	srv := testServer()
	ctx := context.Background()

	largeContent := strings.Repeat("x", 11*1024*1024) // 11MB > 10MB limit
	result, err := srv.handleEditFile(ctx, newRequest("edit_file", map[string]any{
		"sandbox_id": "SBX-1",
		"path":       "/etc/config",
		"new_str":    largeContent,
	}))
	require.NoError(t, err) // errorResult returns nil error
	require.True(t, result.IsError)
	m := parseJSON(t, result)
	assert.Contains(t, m["error"], "file too large")
}

func TestHandleRunCommand_IncludesCommandInError(t *testing.T) {
	svc := &mockSandboxService{
		runCommandFn: func(ctx context.Context, sandboxID, command string, timeoutSec int, env map[string]string) (*sandbox.CommandResult, error) {
			return nil, fmt.Errorf("ssh connection failed")
		},
	}
	srv := testServerWithService(svc)
	ctx := context.Background()

	result, err := srv.handleRunCommand(ctx, newRequest("run_command", map[string]any{
		"sandbox_id": "SBX-1",
		"command":    "whoami",
	}))
	require.NoError(t, err)
	require.True(t, result.IsError)
	m := parseJSON(t, result)
	assert.Equal(t, "whoami", m["command"])
}

// --- handleEditFile with mock VM ---

func TestHandleEditFile_OldStrNotFound(t *testing.T) {
	svc := &mockSandboxService{
		runCommandFn: func(_ context.Context, _, command string, _ int, _ map[string]string) (*sandbox.CommandResult, error) {
			// Return base64-encoded "hello world" for any read command
			return &sandbox.CommandResult{ExitCode: 0, Stdout: "aGVsbG8gd29ybGQ="}, nil
		},
	}
	srv := testServerWithService(svc)
	ctx := context.Background()

	result, err := srv.handleEditFile(ctx, newRequest("edit_file", map[string]any{
		"sandbox_id": "SBX-1",
		"path":       "/etc/config",
		"old_str":    "nonexistent",
		"new_str":    "replacement",
	}))
	require.NoError(t, err)
	require.False(t, result.IsError, "expected IsError to be false")

	m := parseJSON(t, result)
	assert.Equal(t, "old_str_not_found", m["action"])
	assert.Equal(t, "SBX-1", m["sandbox_id"])
	assert.Equal(t, "/etc/config", m["path"])
}

func TestHandleEditFile_ReplaceAll(t *testing.T) {
	var writtenContent string
	svc := &mockSandboxService{
		runCommandFn: func(_ context.Context, _, command string, _ int, _ map[string]string) (*sandbox.CommandResult, error) {
			if strings.Contains(command, "base64 -d >") {
				// Write command - capture the base64 content from the heredoc
				const delim = "--FLUID_B64--"
				first := strings.Index(command, delim)
				if first >= 0 {
					rest := command[first+len(delim):]
					second := strings.Index(rest, delim)
					if second > 0 {
						b64 := rest[:second]
						b64 = strings.Trim(b64, "\\n\n'\"")
						decoded, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
						writtenContent = string(decoded)
					}
				}
				return &sandbox.CommandResult{ExitCode: 0}, nil
			}
			// Read command - return base64("aaa bbb aaa")
			return &sandbox.CommandResult{ExitCode: 0, Stdout: "YWFhIGJiYiBhYWE="}, nil
		},
	}
	srv := testServerWithService(svc)
	ctx := context.Background()

	result, err := srv.handleEditFile(ctx, newRequest("edit_file", map[string]any{
		"sandbox_id":  "SBX-1",
		"path":        "/etc/config",
		"old_str":     "aaa",
		"new_str":     "zzz",
		"replace_all": true,
	}))
	require.NoError(t, err)
	require.False(t, result.IsError, "expected IsError to be false")

	m := parseJSON(t, result)
	assert.Equal(t, "edited", m["action"])

	// Verify all occurrences were replaced
	assert.Equal(t, "zzz bbb zzz", writtenContent)
}
