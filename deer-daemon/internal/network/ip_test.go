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
