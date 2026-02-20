package doctor

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/aspectrr/fluid.sh/fluid/internal/hostexec"
)

// qemuBinaryName returns the correct QEMU binary name for the current architecture.
func qemuBinaryName() string {
	switch runtime.GOARCH {
	case "arm64":
		return "qemu-system-aarch64"
	default:
		return "qemu-system-x86_64"
	}
}

type check struct {
	name string
	fn   func(ctx context.Context, run hostexec.RunFunc) CheckResult
}

func allChecks() []check {
	return []check{
		{"daemon-binary", checkDaemonBinary},
		{"grpc-port", checkGRPCPort},
		{"systemd-active", checkSystemdActive},
		{"systemd-enabled", checkSystemdEnabled},
		{"libvirt-running", checkLibvirtRunning},
		{"kvm-available", checkKVMAvailable},
		{"qemu-binary", checkQEMUBinary},
		{"storage-dirs", checkStorageDirs},
		{"daemon-config", checkDaemonConfig},
	}
}

func checkDaemonBinary(ctx context.Context, run hostexec.RunFunc) CheckResult {
	_, _, code, _ := run(ctx, "which fluid-daemon")
	if code == 0 {
		return CheckResult{
			Name:     "daemon-binary",
			Category: "binary",
			Passed:   true,
			Message:  "fluid-daemon binary found",
		}
	}
	return CheckResult{
		Name:     "daemon-binary",
		Category: "binary",
		Passed:   false,
		Message:  "fluid-daemon binary not found",
		FixCmd:   "curl -fsSL https://get.fluid.sh/daemon | bash",
	}
}

func checkGRPCPort(ctx context.Context, run hostexec.RunFunc) CheckResult {
	stdout, _, code, _ := run(ctx, "ss -tlnp 2>/dev/null | grep :9091 || netstat -tlnp 2>/dev/null | grep :9091")
	if code == 0 && strings.TrimSpace(stdout) != "" {
		return CheckResult{
			Name:     "grpc-port",
			Category: "connectivity",
			Passed:   true,
			Message:  "gRPC port :9091 listening",
		}
	}
	return CheckResult{
		Name:     "grpc-port",
		Category: "connectivity",
		Passed:   false,
		Message:  "gRPC port :9091 not listening",
		FixCmd:   "sudo systemctl start fluid-daemon",
	}
}

func checkSystemdActive(ctx context.Context, run hostexec.RunFunc) CheckResult {
	stdout, _, _, _ := run(ctx, "systemctl is-active fluid-daemon 2>/dev/null")
	if strings.TrimSpace(stdout) == "active" {
		return CheckResult{
			Name:     "systemd-active",
			Category: "service",
			Passed:   true,
			Message:  "fluid-daemon service active",
		}
	}
	return CheckResult{
		Name:     "systemd-active",
		Category: "service",
		Passed:   false,
		Message:  "fluid-daemon service not active",
		FixCmd:   "sudo systemctl start fluid-daemon",
	}
}

func checkSystemdEnabled(ctx context.Context, run hostexec.RunFunc) CheckResult {
	stdout, _, _, _ := run(ctx, "systemctl is-enabled fluid-daemon 2>/dev/null")
	if strings.TrimSpace(stdout) == "enabled" {
		return CheckResult{
			Name:     "systemd-enabled",
			Category: "service",
			Passed:   true,
			Message:  "fluid-daemon service enabled at boot",
		}
	}
	return CheckResult{
		Name:     "systemd-enabled",
		Category: "service",
		Passed:   false,
		Message:  "fluid-daemon service not enabled at boot",
		FixCmd:   "sudo systemctl enable fluid-daemon",
	}
}

func checkLibvirtRunning(ctx context.Context, run hostexec.RunFunc) CheckResult {
	stdout, _, _, _ := run(ctx, "systemctl is-active libvirtd 2>/dev/null")
	if strings.TrimSpace(stdout) == "active" {
		return CheckResult{
			Name:     "libvirt-running",
			Category: "prerequisites",
			Passed:   true,
			Message:  "libvirt running",
		}
	}
	return CheckResult{
		Name:     "libvirt-running",
		Category: "prerequisites",
		Passed:   false,
		Message:  "libvirt not running",
		FixCmd:   "sudo apt install -y libvirt-daemon-system && sudo systemctl start libvirtd",
	}
}

func checkKVMAvailable(ctx context.Context, run hostexec.RunFunc) CheckResult {
	_, _, code, _ := run(ctx, "test -e /dev/kvm")
	if code == 0 {
		return CheckResult{
			Name:     "kvm-available",
			Category: "prerequisites",
			Passed:   true,
			Message:  "KVM available (/dev/kvm)",
		}
	}
	return CheckResult{
		Name:     "kvm-available",
		Category: "prerequisites",
		Passed:   false,
		Message:  "KVM not available (/dev/kvm missing)",
		FixCmd:   "sudo modprobe kvm && sudo modprobe kvm_intel || sudo modprobe kvm_amd",
	}
}

func checkQEMUBinary(ctx context.Context, run hostexec.RunFunc) CheckResult {
	binary := qemuBinaryName()
	_, _, code, _ := run(ctx, fmt.Sprintf("which %s", binary))
	if code == 0 {
		return CheckResult{
			Name:     "qemu-binary",
			Category: "binary",
			Passed:   true,
			Message:  "QEMU binary found",
		}
	}
	return CheckResult{
		Name:     "qemu-binary",
		Category: "binary",
		Passed:   false,
		Message:  fmt.Sprintf("%s not found", binary),
		FixCmd:   "sudo apt install -y qemu-system-x86",
	}
}

func checkStorageDirs(ctx context.Context, run hostexec.RunFunc) CheckResult {
	_, _, code, _ := run(ctx, "test -d /var/lib/fluid-daemon/images && test -d /var/lib/fluid-daemon/overlays")
	if code == 0 {
		return CheckResult{
			Name:     "storage-dirs",
			Category: "storage",
			Passed:   true,
			Message:  "storage directories exist",
		}
	}
	return CheckResult{
		Name:     "storage-dirs",
		Category: "storage",
		Passed:   false,
		Message:  "storage directories missing (/var/lib/fluid-daemon/{images,overlays})",
		FixCmd:   "sudo mkdir -p /var/lib/fluid-daemon/images /var/lib/fluid-daemon/overlays",
	}
}

func checkDaemonConfig(ctx context.Context, run hostexec.RunFunc) CheckResult {
	_, _, code, _ := run(ctx, "test -f /etc/fluid-daemon/daemon.yaml || test -f /etc/fluid/daemon.yaml || test -f ~/.config/fluid/daemon.yaml")
	if code == 0 {
		return CheckResult{
			Name:     "daemon-config",
			Category: "config",
			Passed:   true,
			Message:  "daemon config found",
		}
	}
	return CheckResult{
		Name:     "daemon-config",
		Category: "config",
		Passed:   false,
		Message:  "daemon config not found",
		FixCmd:   "Run the guided setup in onboarding or create /etc/fluid-daemon/daemon.yaml",
	}
}
