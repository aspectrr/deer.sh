package network

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLibvirtStatusIP(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	statusPath := filepath.Join(dir, "virbr0.status")
	statusJSON := `[
  {
    "ip-address": "192.168.122.198",
    "mac-address": "52:54:00:30:38:94"
  },
  {
    "ip-address": "192.168.122.205",
    "mac-address": "52:54:00:7a:03:1e"
  }
]`
	if err := os.WriteFile(statusPath, []byte(statusJSON), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", statusPath, err)
	}

	ip, err := readLibvirtStatusIP(statusPath, "52:54:00:30:38:94")
	if err != nil {
		t.Fatalf("readLibvirtStatusIP returned error: %v", err)
	}
	if ip != "192.168.122.198" {
		t.Fatalf("readLibvirtStatusIP returned %q, want %q", ip, "192.168.122.198")
	}
}

func TestReadLibvirtStatusIP_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	statusPath := filepath.Join(dir, "virbr0.status")
	if err := os.WriteFile(statusPath, []byte(`[]`), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", statusPath, err)
	}

	ip, err := readLibvirtStatusIP(statusPath, "52:54:00:ff:ff:ff")
	if err != nil {
		t.Fatalf("readLibvirtStatusIP returned error: %v", err)
	}
	if ip != "" {
		t.Fatalf("readLibvirtStatusIP returned %q, want empty string", ip)
	}
}

func TestNormalizeARPMac(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "macOS with leading zeros stripped",
			line:     "? (192.168.105.93) at 52:54:0:2c:c:9 on bridge100 ifscope [bridge]",
			expected: "5254002c0c09",
		},
		{
			name:     "standard format with all digits",
			line:     "? (192.168.122.5) at 52:54:00:aa:bb:cc on virbr0 [ether]",
			expected: "525400aabbcc",
		},
		{
			name:     "single digit octets",
			line:     "? (10.0.0.1) at 1:2:3:4:5:6 on eth0 [ether]",
			expected: "010203040506",
		},
		{
			name:     "no match - no 'at' keyword",
			line:     "incomplete entry",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeARPMac(tt.line)
			if got != tt.expected {
				t.Errorf("normalizeARPMac(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

func TestNormalizeARPMac_MatchesMAC(t *testing.T) {
	macNormalized := "5254002c0c09"

	line := "? (192.168.105.93) at 52:54:0:2c:c:9 on bridge100 ifscope [bridge]"
	got := normalizeARPMac(line)
	if got != macNormalized {
		t.Errorf("normalizeARPMac = %q, want %q", got, macNormalized)
	}
}
