package redact

import (
	"strings"
	"testing"
)

func TestIPv4Redaction(t *testing.T) {
	r := New()
	input := "Server at 192.168.1.100 is running on port 8080"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "192.168.1.100") {
		t.Errorf("IPv4 address should be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_IP_1]") {
		t.Errorf("expected [REDACTED_IP_1] token, got: %s", redacted)
	}
}

func TestIPv6Redaction(t *testing.T) {
	r := New()
	input := "listening on 2001:0db8:85a3:0000:0000:8a2e:0370:7334 interface"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "2001:0db8") {
		t.Errorf("IPv6 address should be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_IP_") {
		t.Errorf("expected REDACTED_IP token, got: %s", redacted)
	}
}

func TestAPIKeyRedaction(t *testing.T) {
	r := New()
	input := "use key sk-abcdefghijklmnopqrstuvwxyz for auth"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "sk-abcdefghijklmnopqrstuvwxyz") {
		t.Errorf("API key should be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_KEY_") {
		t.Errorf("expected REDACTED_KEY token, got: %s", redacted)
	}
}

func TestAWSKeyRedaction(t *testing.T) {
	r := New()
	input := "aws_access_key_id = AKIAIOSFODNN7EXAMPLE"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("AWS key should be redacted, got: %s", redacted)
	}
}

func TestSSHPrivateKeyRedaction(t *testing.T) {
	r := New()
	input := `config before
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEATHISISFAKEKEYTESTING
base64encodedFAKEdata1234567890abcdef
-----END RSA PRIVATE KEY-----
config after`
	redacted := r.Redact(input)

	if strings.Contains(redacted, "BEGIN RSA PRIVATE KEY") {
		t.Errorf("SSH key block should be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_KEY_") {
		t.Errorf("expected REDACTED_KEY token, got: %s", redacted)
	}
	if !strings.Contains(redacted, "config before") {
		t.Errorf("non-sensitive text should be preserved, got: %s", redacted)
	}
}

func TestConnectionStringRedaction(t *testing.T) {
	r := New()
	input := "DATABASE_URL=postgres://admin:secret@db.internal:5432/mydb"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "postgres://admin:secret") {
		t.Errorf("connection string should be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_SECRET_") {
		t.Errorf("expected REDACTED_SECRET token, got: %s", redacted)
	}
}

func TestRoundTrip(t *testing.T) {
	r := New()
	input := `Server (10.20.30.40) with key sk-1234567890abcdefghijklmnop`
	redacted := r.Redact(input)
	restored := r.Restore(redacted)

	if restored != input {
		t.Errorf("round-trip failed:\n  input:    %s\n  restored: %s", input, restored)
	}
}

func TestRedactMap(t *testing.T) {
	r := New()
	m := map[string]any{
		"command":    "ssh root@192.168.1.100",
		"sandbox_id": "sbx-123",
		"nested": map[string]any{
			"ip": "10.0.0.55",
		},
		"count": 42,
	}

	result := r.RedactMap(m)

	if s, ok := result["command"].(string); ok {
		if strings.Contains(s, "192.168.1.100") {
			t.Errorf("IP in command should be redacted, got: %s", s)
		}
	}
	if result["sandbox_id"] != "sbx-123" {
		t.Errorf("sandbox_id should be unchanged, got: %v", result["sandbox_id"])
	}
	nested, ok := result["nested"].(map[string]any)
	if !ok {
		t.Fatal("nested should be a map")
	}
	if s, ok := nested["ip"].(string); ok {
		if strings.Contains(s, "10.0.0.55") {
			t.Errorf("nested IP should be redacted, got: %s", s)
		}
	}
	if result["count"] != 42 {
		t.Errorf("non-string value should be unchanged, got: %v", result["count"])
	}
}

func TestRedactMapNil(t *testing.T) {
	r := New()
	result := r.RedactMap(nil)
	if result != nil {
		t.Errorf("RedactMap(nil) should return nil, got: %v", result)
	}
}

func TestRedactAny(t *testing.T) {
	r := New()

	s := r.RedactAny("connect to 192.168.1.100")
	if str, ok := s.(string); ok {
		if strings.Contains(str, "192.168.1.100") {
			t.Errorf("string IP should be redacted, got: %s", str)
		}
	}

	if r.RedactAny(42) != 42 {
		t.Error("int should pass through")
	}
	if r.RedactAny(true) != true {
		t.Error("bool should pass through")
	}
	if r.RedactAny(nil) != nil {
		t.Error("nil should pass through")
	}
}

func TestAllowlist(t *testing.T) {
	r := New(WithAllowlist([]string{"192.168.1.1"}))
	input := "gateway 192.168.1.1 and server 10.20.30.40"
	redacted := r.Redact(input)

	if !strings.Contains(redacted, "192.168.1.1") {
		t.Errorf("allowlisted IP should not be redacted, got: %s", redacted)
	}
	if strings.Contains(redacted, "10.20.30.40") {
		t.Errorf("non-allowlisted IP should be redacted, got: %s", redacted)
	}
}

func TestStats(t *testing.T) {
	r := New()
	r.Redact("connect to 192.168.1.100 and 10.0.0.55")
	r.Redact("key: sk-abcdefghijklmnopqrstuvwxyz")

	stats := r.Stats()
	if stats.Total != 3 {
		t.Errorf("expected 3 total redactions, got %d", stats.Total)
	}
	if stats.ByCategory["IP"] != 2 {
		t.Errorf("expected 2 IP redactions, got %d", stats.ByCategory["IP"])
	}
	if stats.ByCategory["KEY"] != 1 {
		t.Errorf("expected 1 KEY redaction, got %d", stats.ByCategory["KEY"])
	}
}
