package microvm

import (
	"log/slog"
	"testing"
)

func TestGenerateMACAddress(t *testing.T) {
	mac := GenerateMACAddress()
	if mac == "" {
		t.Error("MAC address should not be empty")
	}
	if len(mac) != 17 { // XX:XX:XX:XX:XX:XX
		t.Errorf("MAC address should be 17 chars, got %d: %s", len(mac), mac)
	}
	if mac[:8] != "52:54:00" {
		t.Errorf("MAC should have QEMU prefix 52:54:00, got %s", mac[:8])
	}

	// Generate two and verify they differ
	mac2 := GenerateMACAddress()
	if mac == mac2 {
		t.Error("two generated MACs should differ (random)")
	}
}

func TestExtractJSONString(t *testing.T) {
	input := `{"name":"test","tap_device":"fluid-abc123","vcpus":2,"memory_mb":2048}`

	tests := []struct {
		key  string
		want string
	}{
		{"name", "test"},
		{"tap_device", "fluid-abc123"},
		{"vcpus", "2"},
		{"memory_mb", "2048"},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		got := extractJSONString(input, tt.key)
		if got != tt.want {
			t.Errorf("extractJSONString(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestRecoverState_EmptyDir(t *testing.T) {
	workDir := t.TempDir()

	// This will fail because qemu binary won't be found on macOS,
	// so we test the recovery logic directly
	m := &Manager{
		vms:     make(map[string]*SandboxInfo),
		workDir: workDir,
		qemuBin: "/bin/true",
		logger:  nil,
	}

	// Set up a nil-safe logger
	if m.logger == nil {
		m.logger = defaultLogger()
	}

	// Empty dir should recover without error
	if err := m.RecoverState(nil); err != nil {
		t.Errorf("RecoverState on empty dir: %v", err)
	}

	if len(m.vms) != 0 {
		t.Errorf("expected 0 VMs, got %d", len(m.vms))
	}
}

func defaultLogger() *slog.Logger {
	return slog.Default()
}
