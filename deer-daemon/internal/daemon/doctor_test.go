package daemon

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/config"
	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"
)

func newTestServer(cfg *config.Config) *Server {
	return &Server{cfg: cfg}
}

func TestDoctorCheck_KVMAvailable(t *testing.T) {
	s := newTestServer(&config.Config{})
	result := s.checkKVMAvailable()

	// On macOS the check delegates to HVF; on Linux it checks /dev/kvm.
	// Either way, category must be "prerequisites" and name must match platform.
	wantName := "kvm-available"
	if runtime.GOOS == "darwin" {
		wantName = "hvf-available"
	}
	if result.Name != wantName {
		t.Errorf("name = %q, want %q", result.Name, wantName)
	}
	if result.Category != "prerequisites" {
		t.Errorf("category = %q, want %q", result.Category, "prerequisites")
	}
}

func TestDoctorCheck_KernelPath(t *testing.T) {
	tmp := t.TempDir()
	kernelPath := filepath.Join(tmp, "vmlinuz")

	// Missing kernel
	s := newTestServer(&config.Config{
		MicroVM: config.MicroVMConfig{KernelPath: kernelPath},
	})
	result := s.checkKernelPath()
	if result.Passed {
		t.Error("expected kernel-path to fail when file missing")
	}

	// Create kernel file
	if err := os.WriteFile(kernelPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	result = s.checkKernelPath()
	if !result.Passed {
		t.Error("expected kernel-path to pass when file exists")
	}
}

func TestDoctorCheck_InitrdPath(t *testing.T) {
	tmp := t.TempDir()
	initrdPath := filepath.Join(tmp, "initrd.img")

	// Configured but missing
	s := newTestServer(&config.Config{
		MicroVM: config.MicroVMConfig{InitrdPath: initrdPath},
	})
	result := s.checkInitrdPath()
	if result.Passed {
		t.Error("expected initrd-path to fail when file missing")
	}
	if result.Name != "initrd-path" {
		t.Errorf("name = %q, want %q", result.Name, "initrd-path")
	}
	if result.FixCmd == "" {
		t.Error("expected fix command when initrd missing")
	}

	// Configured and exists
	if err := os.WriteFile(initrdPath, []byte("fake-initrd"), 0o644); err != nil {
		t.Fatal(err)
	}
	result = s.checkInitrdPath()
	if !result.Passed {
		t.Error("expected initrd-path to pass when file exists")
	}

	// Not configured (empty path)
	s = newTestServer(&config.Config{
		MicroVM: config.MicroVMConfig{InitrdPath: ""},
	})
	result = s.checkInitrdPath()
	if !result.Passed {
		t.Error("expected initrd-path to pass when not configured")
	}
	if result.Message != "initrd not configured (direct kernel boot without initramfs)" {
		t.Errorf("unexpected message for unconfigured initrd: %s", result.Message)
	}
}

func TestDoctorCheck_StorageDirs(t *testing.T) {
	tmp := t.TempDir()
	imageDir := filepath.Join(tmp, "images")
	workDir := filepath.Join(tmp, "overlays")

	s := newTestServer(&config.Config{
		Image:   config.ImageConfig{BaseDir: imageDir},
		MicroVM: config.MicroVMConfig{WorkDir: workDir},
	})

	// Both missing
	results := s.checkStorageDirs()
	for _, r := range results {
		if r.Passed {
			t.Errorf("expected %s to fail when dir missing", r.Message)
		}
	}

	// Create dirs
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	results = s.checkStorageDirs()
	for _, r := range results {
		if !r.Passed {
			t.Errorf("expected %s to pass when dir exists", r.Message)
		}
	}
}

func TestDoctorCheck_NetworkBridge(t *testing.T) {
	s := newTestServer(&config.Config{
		Network: config.NetworkConfig{DefaultBridge: "nonexistent-bridge-xyz"},
	})
	result := s.checkNetworkBridge()
	if result.Passed {
		t.Error("expected network-bridge to fail for nonexistent bridge")
	}
	if result.Name != "network-bridge" {
		t.Errorf("name = %q, want %q", result.Name, "network-bridge")
	}
}

func TestScanSourceHostKeys_NoSourceHosts(t *testing.T) {
	s := newTestServer(&config.Config{})
	resp, err := s.ScanSourceHostKeys(context.Background(), &deerv1.ScanSourceHostKeysRequest{})
	if err != nil {
		t.Fatalf("ScanSourceHostKeys() error: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

func TestDoctorCheck_FullRPC(t *testing.T) {
	tmp := t.TempDir()
	imageDir := filepath.Join(tmp, "images")
	workDir := filepath.Join(tmp, "overlays")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	s := newTestServer(&config.Config{
		MicroVM: config.MicroVMConfig{
			QEMUBinary: "nonexistent-qemu-binary-xyz",
			KernelPath: filepath.Join(tmp, "vmlinuz"),
			WorkDir:    workDir,
		},
		Image:   config.ImageConfig{BaseDir: imageDir},
		Network: config.NetworkConfig{DefaultBridge: "nonexistent-bridge-xyz"},
	})

	resp, err := s.DoctorCheck(context.Background(), &deerv1.DoctorCheckRequest{})
	if err != nil {
		t.Fatalf("DoctorCheck() error: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected at least one result")
	}

	// Verify all results have names and categories
	for _, r := range resp.Results {
		if r.Name == "" {
			t.Error("result has empty name")
		}
		if r.Category == "" {
			t.Error("result has empty category")
		}
	}
}
