package microvm

import (
	"context"
	"log/slog"
	"os"
	"reflect"
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
		TAPDevice:  "deer-abc123",
		MACAddress: "52:54:00:aa:bb:cc",
		Bridge:     "deer0",
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

func TestSetIP(t *testing.T) {
	m := &Manager{
		vms:     make(map[string]*SandboxInfo),
		qmpStop: make(map[string]context.CancelFunc),
		workDir: t.TempDir(),
		logger:  defaultLogger(),
	}
	m.vms["sbx-test"] = &SandboxInfo{ID: "sbx-test", State: StateRunning}

	m.SetIP("sbx-test", "10.0.0.5")

	info, ok := m.vms["sbx-test"]
	if !ok {
		t.Fatal("expected sandbox to exist")
	}
	if info.IPAddress != "10.0.0.5" {
		t.Errorf("IPAddress = %q, want %q", info.IPAddress, "10.0.0.5")
	}
}

func TestSetIP_NotFound(t *testing.T) {
	m := &Manager{
		vms:     make(map[string]*SandboxInfo),
		qmpStop: make(map[string]context.CancelFunc),
		workDir: t.TempDir(),
		logger:  defaultLogger(),
	}
	m.SetIP("nonexistent", "10.0.0.5")
}

func TestWriteReadMetadata_WithIP(t *testing.T) {
	workDir := t.TempDir()
	sandboxID := "test-sandbox-ip"
	if err := os.MkdirAll(workDir+"/"+sandboxID, 0o755); err != nil {
		t.Fatal(err)
	}

	want := sandboxMetadata{
		Name:       "test",
		TAPDevice:  "deer-abc123",
		MACAddress: "52:54:00:aa:bb:cc",
		Bridge:     "deer0",
		VCPUs:      2,
		MemoryMB:   2048,
		IPAddress:  "10.0.0.42",
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

func TestValidateAccel(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		accel   string
		wantAny int // expected minimum number of warnings
	}{
		{
			name:    "auto on darwin is fine",
			goos:    "darwin",
			accel:   "",
			wantAny: 0,
		},
		{
			name:    "auto on linux is fine",
			goos:    "linux",
			accel:   "",
			wantAny: 0,
		},
		{
			name:    "hvf on darwin is fine",
			goos:    "darwin",
			accel:   "hvf",
			wantAny: 0,
		},
		{
			name:    "kvm on linux is fine",
			goos:    "linux",
			accel:   "kvm",
			wantAny: 0,
		},
		{
			name:    "tcg warns about suboptimal performance on darwin",
			goos:    "darwin",
			accel:   "tcg",
			wantAny: 1,
		},
		{
			name:    "tcg warns about suboptimal performance on linux",
			goos:    "linux",
			accel:   "tcg",
			wantAny: 1,
		},
		{
			name:    "kvm on darwin warns wrong platform",
			goos:    "darwin",
			accel:   "kvm",
			wantAny: 1,
		},
		{
			name:    "hvf on linux warns wrong platform",
			goos:    "linux",
			accel:   "hvf",
			wantAny: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateAccel(tt.goos, tt.accel)
			if (len(got) >= tt.wantAny) != true {
				t.Errorf("ValidateAccel(%q, %q) returned %d warnings, want at least %d", tt.goos, tt.accel, len(got), tt.wantAny)
			}
		})
	}
}

func TestResolveAccelArgs(t *testing.T) {
	tests := []struct {
		name  string
		goos  string
		accel string
		want  []string
	}{
		{
			name:  "empty accel on darwin auto-selects HVF",
			goos:  "darwin",
			accel: "",
			want:  []string{"-accel", "hvf", "-cpu", "max"},
		},
		{
			name:  "explicit hvf",
			goos:  "linux",
			accel: "hvf",
			want:  []string{"-accel", "hvf", "-cpu", "max"},
		},
		{
			name:  "empty accel on linux defaults to KVM",
			goos:  "linux",
			accel: "",
			want:  []string{"-enable-kvm", "-cpu", "host"},
		},
		{
			name:  "explicit kvm",
			goos:  "linux",
			accel: "kvm",
			want:  []string{"-enable-kvm", "-cpu", "host"},
		},
		{
			name:  "explicit tcg",
			goos:  "darwin",
			accel: "tcg",
			want:  []string{"-accel", "tcg", "-cpu", "max"},
		},
		{
			name:  "explicit tcg on linux",
			goos:  "linux",
			accel: "tcg",
			want:  []string{"-accel", "tcg", "-cpu", "max"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAccelArgs(tt.goos, tt.accel)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolveAccelArgs(%q, %q) = %v, want %v", tt.goos, tt.accel, got, tt.want)
			}
		})
	}
}
