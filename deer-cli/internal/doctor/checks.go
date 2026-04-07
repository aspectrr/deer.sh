package doctor

import (
	"context"
	"strings"

	"github.com/aspectrr/deer.sh/deer-cli/internal/hostexec"
)

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
		{"kernel-tools", checkKernelTools},
		{"storage-dirs", checkStorageDirs},
		{"daemon-config", checkDaemonConfig},
	}
}

func checkDaemonBinary(ctx context.Context, run hostexec.RunFunc) CheckResult {
	_, _, code, _ := run(ctx, "which deer-daemon")
	if code == 0 {
		return CheckResult{
			Name:     "daemon-binary",
			Category: "binary",
			Passed:   true,
			Message:  "deer-daemon binary found",
		}
	}
	return CheckResult{
		Name:     "daemon-binary",
		Category: "binary",
		Passed:   false,
		Message:  "deer-daemon binary not found",
		FixCmd:   "sudo apt install deer-daemon  # or see https://deer.sh/docs/daemon",
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
		FixCmd:   "sudo systemctl start deer-daemon",
	}
}

func checkSystemdActive(ctx context.Context, run hostexec.RunFunc) CheckResult {
	stdout, _, _, _ := run(ctx, "systemctl is-active deer-daemon 2>/dev/null")
	if strings.TrimSpace(stdout) == "active" {
		return CheckResult{
			Name:     "systemd-active",
			Category: "service",
			Passed:   true,
			Message:  "deer-daemon service active",
		}
	}
	return CheckResult{
		Name:     "systemd-active",
		Category: "service",
		Passed:   false,
		Message:  "deer-daemon service not active",
		FixCmd:   "sudo systemctl start deer-daemon",
	}
}

func checkSystemdEnabled(ctx context.Context, run hostexec.RunFunc) CheckResult {
	stdout, _, _, _ := run(ctx, "systemctl is-enabled deer-daemon 2>/dev/null")
	if strings.TrimSpace(stdout) == "enabled" {
		return CheckResult{
			Name:     "systemd-enabled",
			Category: "service",
			Passed:   true,
			Message:  "deer-daemon service enabled at boot",
		}
	}
	return CheckResult{
		Name:     "systemd-enabled",
		Category: "service",
		Passed:   false,
		Message:  "deer-daemon service not enabled at boot",
		FixCmd:   "sudo systemctl enable deer-daemon",
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
		FixCmd:   "sudo modprobe kvm && sudo modprobe kvm_intel || sudo modprobe kvm_amd. If this fails, you may be on a virtualized cloud host (e.g., Hetzner Cloud) that doesn't support nested KVM. Use a dedicated server instead.",
	}
}

func checkQEMUBinary(ctx context.Context, run hostexec.RunFunc) CheckResult {
	_, _, code, _ := run(ctx, "which qemu-system-x86_64 || which qemu-system-aarch64")
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
		Message:  "no qemu-system binary found on daemon host",
		FixCmd:   "sudo apt install -y qemu-system-x86",
	}
}

func checkKernelTools(ctx context.Context, run hostexec.RunFunc) CheckResult {
	_, _, code, _ := run(ctx, "which virt-cat || which qemu-nbd")
	if code == 0 {
		return CheckResult{
			Name:     "kernel-tools",
			Category: "prerequisites",
			Passed:   true,
			Message:  "kernel extraction tools available (virt-cat or qemu-nbd)",
		}
	}
	return CheckResult{
		Name:     "kernel-tools",
		Category: "prerequisites",
		Passed:   false,
		Message:  "kernel extraction tools missing (need virt-cat or qemu-nbd)",
		FixCmd:   "sudo apt install -y libguestfs-tools",
	}
}

func checkStorageDirs(ctx context.Context, run hostexec.RunFunc) CheckResult {
	_, _, code, _ := run(ctx, "test -d /var/lib/deer-daemon/images && test -d /var/lib/deer-daemon/overlays")
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
		Message:  "storage directories missing (/var/lib/deer-daemon/{images,overlays})",
		FixCmd:   "sudo mkdir -p /var/lib/deer-daemon/images /var/lib/deer-daemon/overlays",
	}
}

func checkDaemonConfig(ctx context.Context, run hostexec.RunFunc) CheckResult {
	_, _, code, _ := run(ctx, "test -f /etc/deer-daemon/daemon.yaml || test -f /etc/deer/daemon.yaml || test -f ~/.config/deer/daemon.yaml")
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
		FixCmd:   "Run the guided setup in onboarding or create /etc/deer-daemon/daemon.yaml",
	}
}
