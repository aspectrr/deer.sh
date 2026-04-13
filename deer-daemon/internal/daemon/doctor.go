package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"time"

	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"
)

func (s *Server) DoctorCheck(ctx context.Context, _ *deerv1.DoctorCheckRequest) (*deerv1.DoctorCheckResponse, error) {
	var results []*deerv1.DoctorCheckResult
	results = append(results, s.checkQEMUBinary())
	results = append(results, s.checkKVMAvailable())
	results = append(results, s.checkKernelPath())
	results = append(results, s.checkInitrdPath())
	results = append(results, s.checkStorageDirs()...)
	results = append(results, s.checkNetworkBridge())
	results = append(results, s.checkSourceHosts(ctx)...)
	return &deerv1.DoctorCheckResponse{Results: results}, nil
}

func (s *Server) checkQEMUBinary() *deerv1.DoctorCheckResult {
	binary := s.cfg.MicroVM.QEMUBinary
	path, err := exec.LookPath(binary)
	if err != nil {
		return &deerv1.DoctorCheckResult{
			Name:     "qemu-binary",
			Category: "binary",
			Passed:   false,
			Message:  fmt.Sprintf("QEMU binary %q not found in PATH", binary),
			FixCmd:   "sudo apt install -y qemu-system-x86",
		}
	}
	return &deerv1.DoctorCheckResult{
		Name:     "qemu-binary",
		Category: "binary",
		Passed:   true,
		Message:  fmt.Sprintf("QEMU binary found at %s", path),
	}
}

func (s *Server) checkKVMAvailable() *deerv1.DoctorCheckResult {
	if runtime.GOOS == "darwin" {
		return s.checkHVFAvailable()
	}
	if _, err := os.Stat("/dev/kvm"); err != nil {
		return &deerv1.DoctorCheckResult{
			Name:     "kvm-available",
			Category: "prerequisites",
			Passed:   false,
			Message:  "KVM not available (/dev/kvm missing)",
			FixCmd:   "sudo modprobe kvm && (sudo modprobe kvm_intel || sudo modprobe kvm_amd). Or set accel: tcg in daemon config to use software emulation",
		}
	}
	return &deerv1.DoctorCheckResult{
		Name:     "kvm-available",
		Category: "prerequisites",
		Passed:   true,
		Message:  "KVM available (/dev/kvm)",
	}
}

func (s *Server) checkHVFAvailable() *deerv1.DoctorCheckResult {
	// HVF (Hypervisor.framework) is always present on Apple Silicon.
	// Verify by checking QEMU binary can be found - HVF itself needs no device node.
	binary := s.cfg.MicroVM.QEMUBinary
	if _, err := exec.LookPath(binary); err != nil {
		return &deerv1.DoctorCheckResult{
			Name:     "hvf-available",
			Category: "prerequisites",
			Passed:   false,
			Message:  fmt.Sprintf("QEMU binary %q not found (required for HVF)", binary),
			FixCmd:   "brew install qemu",
		}
	}
	result := &deerv1.DoctorCheckResult{
		Name:     "hvf-available",
		Category: "prerequisites",
		Passed:   true,
		Message:  "HVF accelerator available (macOS Hypervisor.framework)",
	}
	// If socket_vmnet is configured, verify the socket path is accessible
	if s.cfg.MicroVM.SocketVMNetClient != "" {
		if _, err := os.Stat(s.cfg.MicroVM.SocketVMNetPath); err != nil {
			return &deerv1.DoctorCheckResult{
				Name:     "hvf-available",
				Category: "prerequisites",
				Passed:   false,
				Message:  fmt.Sprintf("socket_vmnet socket not found at %s", s.cfg.MicroVM.SocketVMNetPath),
				FixCmd:   "brew install socket_vmnet && sudo brew services start socket_vmnet",
			}
		}
		result.Message += fmt.Sprintf(", socket_vmnet socket found at %s", s.cfg.MicroVM.SocketVMNetPath)
	}
	return result
}

func (s *Server) checkKernelPath() *deerv1.DoctorCheckResult {
	kp := s.cfg.MicroVM.KernelPath
	if _, err := os.Stat(kp); err != nil {
		return &deerv1.DoctorCheckResult{
			Name:     "kernel-path",
			Category: "prerequisites",
			Passed:   false,
			Message:  fmt.Sprintf("kernel not found at %s", kp),
			FixCmd:   fmt.Sprintf("download a vmlinuz to %s", kp),
		}
	}
	return &deerv1.DoctorCheckResult{
		Name:     "kernel-path",
		Category: "prerequisites",
		Passed:   true,
		Message:  fmt.Sprintf("kernel found at %s", kp),
	}
}

func (s *Server) checkInitrdPath() *deerv1.DoctorCheckResult {
	ip := s.cfg.MicroVM.InitrdPath
	if ip == "" {
		return &deerv1.DoctorCheckResult{
			Name:     "initrd-path",
			Category: "prerequisites",
			Passed:   true,
			Message:  "initrd not configured (direct kernel boot without initramfs)",
		}
	}
	if _, err := os.Stat(ip); err != nil {
		return &deerv1.DoctorCheckResult{
			Name:     "initrd-path",
			Category: "prerequisites",
			Passed:   false,
			Message:  fmt.Sprintf("initrd not found at %s", ip),
			FixCmd:   fmt.Sprintf("sudo cp /boot/initrd.img-$(uname -r) %s", ip),
		}
	}
	return &deerv1.DoctorCheckResult{
		Name:     "initrd-path",
		Category: "prerequisites",
		Passed:   true,
		Message:  fmt.Sprintf("initrd found at %s", ip),
	}
}

func (s *Server) checkStorageDirs() []*deerv1.DoctorCheckResult {
	dirs := map[string]string{
		"image-dir": s.cfg.Image.BaseDir,
		"work-dir":  s.cfg.MicroVM.WorkDir,
	}
	var results []*deerv1.DoctorCheckResult
	for name, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			results = append(results, &deerv1.DoctorCheckResult{
				Name:     "storage-dirs",
				Category: "storage",
				Passed:   false,
				Message:  fmt.Sprintf("%s missing: %s", name, dir),
				FixCmd:   fmt.Sprintf("sudo mkdir -p %s", dir),
			})
		} else {
			results = append(results, &deerv1.DoctorCheckResult{
				Name:     "storage-dirs",
				Category: "storage",
				Passed:   true,
				Message:  fmt.Sprintf("%s exists: %s", name, dir),
			})
		}
	}
	return results
}

func (s *Server) checkNetworkBridge() *deerv1.DoctorCheckResult {
	// When socket_vmnet is configured (macOS), QEMU connects through a
	// unix socket to the vmnet framework — no Linux bridge interface exists.
	if s.cfg.MicroVM.SocketVMNetClient != "" {
		return &deerv1.DoctorCheckResult{
			Name:     "network-bridge",
			Category: "network",
			Passed:   true,
			Message:  fmt.Sprintf("using socket_vmnet for networking (bridge %q configured but not required)", s.cfg.Network.DefaultBridge),
		}
	}
	bridge := s.cfg.Network.DefaultBridge
	if _, err := net.InterfaceByName(bridge); err != nil {
		return &deerv1.DoctorCheckResult{
			Name:     "network-bridge",
			Category: "network",
			Passed:   false,
			Message:  fmt.Sprintf("bridge %q not found", bridge),
			FixCmd:   fmt.Sprintf("sudo ip link add %s type bridge && sudo ip link set %s up", bridge, bridge),
		}
	}
	return &deerv1.DoctorCheckResult{
		Name:     "network-bridge",
		Category: "network",
		Passed:   true,
		Message:  fmt.Sprintf("bridge %q found", bridge),
	}
}

// checkSourceHosts verifies SSH + libvirt connectivity to each configured source host.
func (s *Server) checkSourceHosts(ctx context.Context) []*deerv1.DoctorCheckResult {
	if len(s.cfg.SourceHosts) == 0 {
		return nil
	}

	var results []*deerv1.DoctorCheckResult
	for _, conn := range s.sourceHostConns() {
		host := conn.SshHost
		user := conn.SshUser
		name := fmt.Sprintf("source-host-%s", host)

		mgr, err := s.adhocSourceVMManager(conn)
		if err != nil {
			results = append(results, &deerv1.DoctorCheckResult{
				Name:     name,
				Category: "source-hosts",
				Passed:   false,
				Message:  fmt.Sprintf("cannot create manager for %s: %v", host, err),
			})
			continue
		}

		checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		_, err = mgr.ListVMs(checkCtx)
		cancel()

		if err != nil {
			results = append(results, &deerv1.DoctorCheckResult{
				Name:     name,
				Category: "source-hosts",
				Passed:   false,
				Message:  fmt.Sprintf("cannot reach %s as %s: %v", host, user, err),
				FixCmd:   fmt.Sprintf("Run 'deer connect' and press Enter to setup source hosts, or manually: ssh-keyscan -H %s >> ~deer-daemon/.ssh/known_hosts", host),
			})
		} else {
			results = append(results, &deerv1.DoctorCheckResult{
				Name:     name,
				Category: "source-hosts",
				Passed:   true,
				Message:  fmt.Sprintf("SSH + libvirt OK for %s@%s", user, host),
			})
		}
	}
	return results
}
