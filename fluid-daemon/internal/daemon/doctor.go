package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
)

func (s *Server) DoctorCheck(ctx context.Context, _ *fluidv1.DoctorCheckRequest) (*fluidv1.DoctorCheckResponse, error) {
	var results []*fluidv1.DoctorCheckResult
	results = append(results, s.checkQEMUBinary())
	results = append(results, s.checkKVMAvailable())
	results = append(results, s.checkKernelPath())
	results = append(results, s.checkInitrdPath())
	results = append(results, s.checkStorageDirs()...)
	results = append(results, s.checkNetworkBridge())
	results = append(results, s.checkSourceHosts(ctx)...)
	return &fluidv1.DoctorCheckResponse{Results: results}, nil
}

func (s *Server) checkQEMUBinary() *fluidv1.DoctorCheckResult {
	binary := s.cfg.MicroVM.QEMUBinary
	path, err := exec.LookPath(binary)
	if err != nil {
		return &fluidv1.DoctorCheckResult{
			Name:     "qemu-binary",
			Category: "binary",
			Passed:   false,
			Message:  fmt.Sprintf("QEMU binary %q not found in PATH", binary),
			FixCmd:   "sudo apt install -y qemu-system-x86",
		}
	}
	return &fluidv1.DoctorCheckResult{
		Name:     "qemu-binary",
		Category: "binary",
		Passed:   true,
		Message:  fmt.Sprintf("QEMU binary found at %s", path),
	}
}

func (s *Server) checkKVMAvailable() *fluidv1.DoctorCheckResult {
	if _, err := os.Stat("/dev/kvm"); err != nil {
		return &fluidv1.DoctorCheckResult{
			Name:     "kvm-available",
			Category: "prerequisites",
			Passed:   false,
			Message:  "KVM not available (/dev/kvm missing)",
			FixCmd:   "sudo modprobe kvm && sudo modprobe kvm_intel || sudo modprobe kvm_amd. Or set accel: tcg in daemon config to use software emulation",
		}
	}
	return &fluidv1.DoctorCheckResult{
		Name:     "kvm-available",
		Category: "prerequisites",
		Passed:   true,
		Message:  "KVM available (/dev/kvm)",
	}
}

func (s *Server) checkKernelPath() *fluidv1.DoctorCheckResult {
	kp := s.cfg.MicroVM.KernelPath
	if _, err := os.Stat(kp); err != nil {
		return &fluidv1.DoctorCheckResult{
			Name:     "kernel-path",
			Category: "prerequisites",
			Passed:   false,
			Message:  fmt.Sprintf("kernel not found at %s", kp),
			FixCmd:   fmt.Sprintf("download a vmlinuz to %s", kp),
		}
	}
	return &fluidv1.DoctorCheckResult{
		Name:     "kernel-path",
		Category: "prerequisites",
		Passed:   true,
		Message:  fmt.Sprintf("kernel found at %s", kp),
	}
}

func (s *Server) checkInitrdPath() *fluidv1.DoctorCheckResult {
	ip := s.cfg.MicroVM.InitrdPath
	if ip == "" {
		return &fluidv1.DoctorCheckResult{
			Name:     "initrd-path",
			Category: "prerequisites",
			Passed:   true,
			Message:  "initrd not configured (direct kernel boot without initramfs)",
		}
	}
	if _, err := os.Stat(ip); err != nil {
		return &fluidv1.DoctorCheckResult{
			Name:     "initrd-path",
			Category: "prerequisites",
			Passed:   false,
			Message:  fmt.Sprintf("initrd not found at %s", ip),
			FixCmd:   fmt.Sprintf("sudo cp /boot/initrd.img-$(uname -r) %s", ip),
		}
	}
	return &fluidv1.DoctorCheckResult{
		Name:     "initrd-path",
		Category: "prerequisites",
		Passed:   true,
		Message:  fmt.Sprintf("initrd found at %s", ip),
	}
}

func (s *Server) checkStorageDirs() []*fluidv1.DoctorCheckResult {
	dirs := map[string]string{
		"image-dir": s.cfg.Image.BaseDir,
		"work-dir":  s.cfg.MicroVM.WorkDir,
	}
	var results []*fluidv1.DoctorCheckResult
	for name, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			results = append(results, &fluidv1.DoctorCheckResult{
				Name:     "storage-dirs",
				Category: "storage",
				Passed:   false,
				Message:  fmt.Sprintf("%s missing: %s", name, dir),
				FixCmd:   fmt.Sprintf("sudo mkdir -p %s", dir),
			})
		} else {
			results = append(results, &fluidv1.DoctorCheckResult{
				Name:     "storage-dirs",
				Category: "storage",
				Passed:   true,
				Message:  fmt.Sprintf("%s exists: %s", name, dir),
			})
		}
	}
	return results
}

func (s *Server) checkNetworkBridge() *fluidv1.DoctorCheckResult {
	bridge := s.cfg.Network.DefaultBridge
	if _, err := net.InterfaceByName(bridge); err != nil {
		return &fluidv1.DoctorCheckResult{
			Name:     "network-bridge",
			Category: "network",
			Passed:   false,
			Message:  fmt.Sprintf("bridge %q not found", bridge),
			FixCmd:   fmt.Sprintf("sudo ip link add %s type bridge && sudo ip link set %s up", bridge, bridge),
		}
	}
	return &fluidv1.DoctorCheckResult{
		Name:     "network-bridge",
		Category: "network",
		Passed:   true,
		Message:  fmt.Sprintf("bridge %q found", bridge),
	}
}

// checkSourceHosts verifies SSH + libvirt connectivity to each configured source host.
func (s *Server) checkSourceHosts(ctx context.Context) []*fluidv1.DoctorCheckResult {
	if len(s.cfg.SourceHosts) == 0 {
		return nil
	}

	var results []*fluidv1.DoctorCheckResult
	for _, conn := range s.sourceHostConns() {
		host := conn.SshHost
		user := conn.SshUser
		name := fmt.Sprintf("source-host-%s", host)

		mgr, err := s.adhocSourceVMManager(conn)
		if err != nil {
			results = append(results, &fluidv1.DoctorCheckResult{
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
			results = append(results, &fluidv1.DoctorCheckResult{
				Name:     name,
				Category: "source-hosts",
				Passed:   false,
				Message:  fmt.Sprintf("cannot reach %s as %s: %v", host, user, err),
				FixCmd:   fmt.Sprintf("Run 'fluid connect' and press Enter to setup source hosts, or manually: ssh-keyscan -H %s >> ~fluid-daemon/.ssh/known_hosts", host),
			})
		} else {
			results = append(results, &fluidv1.DoctorCheckResult{
				Name:     name,
				Category: "source-hosts",
				Passed:   true,
				Message:  fmt.Sprintf("SSH + libvirt OK for %s@%s", user, host),
			})
		}
	}
	return results
}
