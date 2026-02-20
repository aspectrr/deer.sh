package microvm

import (
	"context"
	"log/slog"
	"os"
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

func TestWriteReadMetadata(t *testing.T) {
	workDir := t.TempDir()
	sandboxID := "test-sandbox"
	if err := os.MkdirAll(workDir+"/"+sandboxID, 0o755); err != nil {
		t.Fatal(err)
	}

	want := sandboxMetadata{
		Name:       "test",
		TAPDevice:  "fluid-abc123",
		MACAddress: "52:54:00:aa:bb:cc",
		Bridge:     "fluid0",
		VCPUs:      2,
		MemoryMB:   2048,
	}

	if err := writeMetadata(workDir, sandboxID, want); err != nil {
		t.Fatalf("writeMetadata: %v", err)
	}

	got, err := readMetadata(workDir, sandboxID)
	if err != nil {
		t.Fatalf("readMetadata: %v", err)
	}

	if got != want {
		t.Errorf("metadata mismatch:\n got: %+v\nwant: %+v", got, want)
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
	if err := m.RecoverState(context.TODO()); err != nil {
		t.Errorf("RecoverState on empty dir: %v", err)
	}

	if len(m.vms) != 0 {
		t.Errorf("expected 0 VMs, got %d", len(m.vms))
	}
}

func defaultLogger() *slog.Logger {
	return slog.Default()
}
