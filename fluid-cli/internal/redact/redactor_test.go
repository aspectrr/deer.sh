package redact

import (
	"encoding/base64"
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

func TestIPv4Restoration(t *testing.T) {
	r := New()
	input := "Connect to 10.0.0.55 for the service"
	redacted := r.Redact(input)
	restored := r.Restore(redacted)

	if restored != input {
		t.Errorf("round-trip failed:\n  input:    %s\n  restored: %s", input, restored)
	}
}

func TestIPv4SkipsVersionNumbers(t *testing.T) {
	r := New()
	// All octets <= 3 - treated as version-like, should not be redacted.
	input := "version 2.0.0.1 is stable"
	redacted := r.Redact(input)

	if redacted != input {
		t.Errorf("version-like string should not be redacted, got: %s", redacted)
	}
}

func TestIPv4RedactsPublicDNS(t *testing.T) {
	r := New()
	// 8.8.8.8 has octets > 3, so it should be redacted.
	input := "dns 8.8.8.8 configured"
	redacted := r.Redact(input)
	if strings.Contains(redacted, "8.8.8.8") {
		t.Errorf("public DNS IP 8.8.8.8 should be redacted, got: %s", redacted)
	}
}

func TestIPv4ValidatesOctets(t *testing.T) {
	r := New()
	input := "address 999.999.999.999 is invalid"
	redacted := r.Redact(input)

	if redacted != input {
		t.Errorf("invalid IP should not be redacted, got: %s", redacted)
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

func TestIPv6Compressed(t *testing.T) {
	r := New()
	input := "address fe80::1 is link-local"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "fe80::1") {
		t.Errorf("compressed IPv6 should be redacted, got: %s", redacted)
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

func TestBearerTokenRedaction(t *testing.T) {
	r := New()
	input := "Authorization: Bearer eyTHISISAFAKEJWTFORTESTINGONLY1234"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "eyTHISISAFAKEJWTFORTESTINGONLY1234") {
		t.Errorf("Bearer token should be redacted, got: %s", redacted)
	}
}

func TestAWSKeyRedaction(t *testing.T) {
	r := New()
	input := "aws_access_key_id = AKIAIOSFODNN7EXAMPLE"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("AWS key should be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_KEY_") {
		t.Errorf("expected REDACTED_KEY token, got: %s", redacted)
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
	if !strings.Contains(redacted, "config after") {
		t.Errorf("non-sensitive text should be preserved, got: %s", redacted)
	}
}

func TestBase64PEMKeyRedaction(t *testing.T) {
	r := New()
	pem := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEATHISISFAKEKEYTESTING\n-----END RSA PRIVATE KEY-----"
	encoded := base64.StdEncoding.EncodeToString([]byte(pem))
	redacted := r.Redact(encoded)

	if strings.Contains(redacted, "LS0tLS1CRUdJTi") {
		t.Errorf("base64 PEM key should be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_KEY_") {
		t.Errorf("expected REDACTED_KEY token, got: %s", redacted)
	}
}

func TestBase64NonKeyNotRedacted(t *testing.T) {
	r := New()
	// Base64 of "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWg\n-----END CERTIFICATE-----"
	pem := "-----BEGIN CERTIFICATE-----\nMIIDXTCCAkWg\n-----END CERTIFICATE-----"
	encoded := base64.StdEncoding.EncodeToString([]byte(pem))
	redacted := r.Redact(encoded)

	// Certificate should not be redacted - only private keys
	if redacted != encoded {
		t.Errorf("base64 certificate should not be redacted, got: %s", redacted)
	}
}

func TestK8sSecretYAMLRedaction(t *testing.T) {
	r := New()
	input := "  tls.key: " + strings.Repeat("ABCDEFGHIJKLMNOP", 5)
	redacted := r.Redact(input)

	if !strings.Contains(redacted, "[REDACTED_KEY_") {
		t.Errorf("K8s tls.key field should be redacted, got: %s", redacted)
	}
}

func TestK8sSecretJSONRedaction(t *testing.T) {
	r := New()
	input := `"private_key": "` + strings.Repeat("ABCDEFGHIJKLMNOP", 5) + `"`
	redacted := r.Redact(input)

	if !strings.Contains(redacted, "[REDACTED_KEY_") {
		t.Errorf("K8s private_key field should be redacted, got: %s", redacted)
	}
}

func TestK8sTLSCertNotRedacted(t *testing.T) {
	r := New()
	// tls.crt is a public certificate, not secret material - should not be redacted
	input := "  tls.crt: " + strings.Repeat("ABCDEFGHIJKLMNOP", 5)
	redacted := r.Redact(input)

	if redacted != input {
		t.Errorf("tls.crt (public cert) should not be redacted, got: %s", redacted)
	}
}

func TestK8sPublicKeyNotRedacted(t *testing.T) {
	r := New()
	// "public_key" is not in the known secret field names
	input := `public_key: ` + strings.Repeat("ABCDEFGHIJKLMNOP", 5)
	redacted := r.Redact(input)

	if redacted != input {
		t.Errorf("public_key field should not be redacted, got: %s", redacted)
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

func TestRedisConnectionString(t *testing.T) {
	r := New()
	input := "REDIS=redis://user:pass@cache.internal:6379"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "redis://") {
		t.Errorf("redis connection string should be redacted, got: %s", redacted)
	}
}

func TestConsistency(t *testing.T) {
	r := New()
	text1 := "connect to 192.168.1.100"
	text2 := "also use 192.168.1.100 here"

	redacted1 := r.Redact(text1)
	redacted2 := r.Redact(text2)

	// Extract the token used in the first redaction.
	token := strings.TrimPrefix(redacted1, "connect to ")
	token2 := strings.TrimPrefix(redacted2, "also use ")
	token2 = strings.TrimSuffix(token2, " here")

	if token != token2 {
		t.Errorf("same value should map to same token: %q vs %q", token, token2)
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

func TestConfigValues(t *testing.T) {
	r := New(WithConfigValues(
		[]string{"prod-db.internal.corp"},
		[]string{"172.16.0.50"},
		[]string{"/home/deploy/.ssh/id_rsa"},
	))
	input := "connecting to prod-db.internal.corp at 172.16.0.50 using /home/deploy/.ssh/id_rsa"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "prod-db.internal.corp") {
		t.Errorf("config hostname should be redacted, got: %s", redacted)
	}
	if strings.Contains(redacted, "/home/deploy/.ssh/id_rsa") {
		t.Errorf("config key path should be redacted, got: %s", redacted)
	}

	// Verify categories.
	if !strings.Contains(redacted, "[REDACTED_HOST_") {
		t.Errorf("hostname should use HOST category, got: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_PATH_") {
		t.Errorf("key path should use PATH category, got: %s", redacted)
	}
}

func TestOverlappingPatternsLongestWins(t *testing.T) {
	// A connection string contains an IP. The connection string is longer
	// and should win as a single redaction.
	r := New()
	input := "url: postgres://admin:pass@192.168.1.50:5432/db"
	redacted := r.Redact(input)

	// The entire connection string should be one token.
	if strings.Contains(redacted, "postgres://") {
		t.Errorf("connection string should be fully redacted, got: %s", redacted)
	}
	// Should not have separate IP redaction alongside connection string.
	// Count REDACTED tokens.
	count := strings.Count(redacted, "[REDACTED_")
	if count != 1 {
		t.Errorf("expected 1 redaction token (longest wins), got %d in: %s", count, redacted)
	}
}

func TestRoundTrip(t *testing.T) {
	r := New(WithConfigValues(
		[]string{"myhost.example.com"},
		[]string{"10.20.30.40"},
		[]string{"/etc/ssl/private/key.pem"},
	))
	input := `Server myhost.example.com (10.20.30.40) uses key /etc/ssl/private/key.pem
AWS key: AKIAIOSFODNN7EXAMPLE
DB: postgres://root:hunter2@myhost.example.com:5432/app
API: sk-1234567890abcdefghijklmnop`

	redacted := r.Redact(input)

	// Verify nothing sensitive remains.
	for _, sensitive := range []string{
		"myhost.example.com",
		"10.20.30.40",
		"/etc/ssl/private/key.pem",
		"AKIAIOSFODNN7EXAMPLE",
		"sk-1234567890abcdefghijklmnop",
	} {
		if strings.Contains(redacted, sensitive) {
			t.Errorf("sensitive value %q should be redacted in: %s", sensitive, redacted)
		}
	}

	// Round-trip restore.
	restored := r.Restore(redacted)
	if restored != input {
		t.Errorf("round-trip failed:\n  input:\n%s\n  restored:\n%s", input, restored)
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

func TestNoSensitiveData(t *testing.T) {
	r := New()
	input := "this is a normal message with no sensitive data"
	redacted := r.Redact(input)

	if redacted != input {
		t.Errorf("text without sensitive data should be unchanged, got: %s", redacted)
	}
}

func TestEmptyInput(t *testing.T) {
	r := New()
	redacted := r.Redact("")
	if redacted != "" {
		t.Errorf("empty input should produce empty output, got: %q", redacted)
	}

	restored := r.Restore("")
	if restored != "" {
		t.Errorf("empty restore should produce empty output, got: %q", restored)
	}
}

func TestCustomPatterns(t *testing.T) {
	r := New(WithCustomPatterns([]string{`\bTOKEN_[A-Z0-9]{10,}\b`}))
	input := "auth with TOKEN_ABCDEF1234567890"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "TOKEN_ABCDEF1234567890") {
		t.Errorf("custom pattern should be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, "[REDACTED_SECRET_") {
		t.Errorf("custom pattern should use SECRET category, got: %s", redacted)
	}
}

func TestRedactMap(t *testing.T) {
	r := New()
	m := map[string]any{
		"command":    "ssh root@192.168.1.100",
		"sandbox_id": "sbx-123",
		"nested": map[string]any{
			"ip":  "10.0.0.55",
			"key": "sk-abcdefghijklmnopqrstuvwxyz",
		},
		"count": 42,
	}

	result := r.RedactMap(m)

	// IP in command should be redacted
	if s, ok := result["command"].(string); ok {
		if strings.Contains(s, "192.168.1.100") {
			t.Errorf("IP in command should be redacted, got: %s", s)
		}
	}

	// sandbox_id should be unchanged (no sensitive pattern)
	if result["sandbox_id"] != "sbx-123" {
		t.Errorf("sandbox_id should be unchanged, got: %v", result["sandbox_id"])
	}

	// Nested map values should be redacted
	nested, ok := result["nested"].(map[string]any)
	if !ok {
		t.Fatal("nested should be a map")
	}
	if s, ok := nested["ip"].(string); ok {
		if strings.Contains(s, "10.0.0.55") {
			t.Errorf("nested IP should be redacted, got: %s", s)
		}
	}
	if s, ok := nested["key"].(string); ok {
		if strings.Contains(s, "sk-abcdefghijklmnopqrstuvwxyz") {
			t.Errorf("nested API key should be redacted, got: %s", s)
		}
	}

	// Non-string values should pass through
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

	// String with IP
	s := r.RedactAny("connect to 192.168.1.100")
	if str, ok := s.(string); ok {
		if strings.Contains(str, "192.168.1.100") {
			t.Errorf("string IP should be redacted, got: %s", str)
		}
	}

	// Non-string passthrough
	if r.RedactAny(42) != 42 {
		t.Error("int should pass through")
	}
	if r.RedactAny(true) != true {
		t.Error("bool should pass through")
	}
	if r.RedactAny(nil) != nil {
		t.Error("nil should pass through")
	}

	// Slice
	slice := r.RedactAny([]any{"10.0.0.55", 123, "hello"})
	if arr, ok := slice.([]any); ok {
		if s, ok := arr[0].(string); ok && strings.Contains(s, "10.0.0.55") {
			t.Errorf("IP in slice should be redacted, got: %s", s)
		}
		if arr[1] != 123 {
			t.Error("int in slice should pass through")
		}
	}
}

func TestMultipleIPsSameText(t *testing.T) {
	r := New()
	input := "from 192.168.1.10 to 10.0.0.20"
	redacted := r.Redact(input)

	if strings.Contains(redacted, "192.168.1.10") || strings.Contains(redacted, "10.0.0.20") {
		t.Errorf("both IPs should be redacted, got: %s", redacted)
	}

	// They should get different tokens.
	if !strings.Contains(redacted, "[REDACTED_IP_1]") || !strings.Contains(redacted, "[REDACTED_IP_2]") {
		t.Errorf("different IPs should get different tokens, got: %s", redacted)
	}
}
