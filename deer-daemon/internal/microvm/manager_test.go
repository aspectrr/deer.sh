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

func TestQEMUPlatformOptions(t *testing.T) {
	tests := []struct {
		name     string
		qemuBin  string
		wantOpts qemuPlatform
	}{
		{
			name:    "x86",
			qemuBin: "/usr/bin/qemu-system-x86_64",
			wantOpts: qemuPlatform{
				machineType:       "microvm",
				consoleDevice:     "ttyS0",
				blockDevice:       "virtio-blk-device",
				netDevice:         "virtio-net-device",
				defaultRootDevice: "/dev/vda",
				cloudInitCtl:      "",
				cloudInitDevice:   "virtio-blk-device,drive=cidata",
			},
		},
		{
			name:    "arm64",
			qemuBin: "/opt/homebrew/bin/qemu-system-aarch64",
			wantOpts: qemuPlatform{
				machineType:       "virt",
				consoleDevice:     "ttyAMA0",
				blockDevice:       "virtio-blk-device",
				netDevice:         "virtio-net-device",
				defaultRootDevice: "/dev/vda1",
				cloudInitCtl:      "virtio-scsi-device,id=scsi0",
				cloudInitDevice:   "scsi-cd,drive=cidata,bus=scsi0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := qemuPlatformOptions(tt.qemuBin)
			if got != tt.wantOpts {
				t.Fatalf("qemuPlatformOptions(%q) = %+v, want %+v", tt.qemuBin, got, tt.wantOpts)
			}
		})
	}
}
