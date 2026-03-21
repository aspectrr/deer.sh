// Package microvm implements the SandboxProvider interface for QEMU microVMs.
// It wraps the existing microvm, network, image, and sourcevm packages.
package microvm

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/id"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/image"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/microvm"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/network"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/sourcevm"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/sshkeys"
)

// ReadinessWaiter can wait for a sandbox to signal readiness via phone_home.
type ReadinessWaiter interface {
	Register(sandboxID string)
	Unregister(sandboxID string)
	WaitReady(sandboxID string, timeout time.Duration) error
}

// Provider implements provider.SandboxProvider for QEMU microVMs.
type Provider struct {
	vmMgr      *microvm.Manager
	netMgr     *network.NetworkManager
	imgStore   *image.Store
	srcVMMgr   *sourcevm.Manager
	keyMgr     sshkeys.KeyProvider
	kernelPath string
	initrdPath string
	rootDevice string
	accel      string
	caPubKey   string
	bridgeIP   string
	readiness  ReadinessWaiter
	logger     *slog.Logger
}

// New creates a new microVM provider.
func New(
	vmMgr *microvm.Manager,
	netMgr *network.NetworkManager,
	imgStore *image.Store,
	srcVMMgr *sourcevm.Manager,
	keyMgr sshkeys.KeyProvider,
	kernelPath string,
	initrdPath string,
	rootDevice string,
	accel string,
	caPubKey string,
	bridgeIP string,
	readiness ReadinessWaiter,
	logger *slog.Logger,
) *Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{
		vmMgr:      vmMgr,
		netMgr:     netMgr,
		imgStore:   imgStore,
		srcVMMgr:   srcVMMgr,
		keyMgr:     keyMgr,
		kernelPath: kernelPath,
		initrdPath: initrdPath,
		rootDevice: rootDevice,
		accel:      accel,
		caPubKey:   caPubKey,
		bridgeIP:   bridgeIP,
		readiness:  readiness,
		logger:     logger.With("provider", "microvm"),
	}
}

func (p *Provider) CreateSandbox(ctx context.Context, req provider.CreateRequest) (*provider.SandboxResult, error) {
	if p.vmMgr == nil {
		return nil, fmt.Errorf("microVM manager not available")
	}

	// Resolve bridge
	bridge, err := p.netMgr.ResolveBridge(ctx, req.Network)
	if err != nil {
		return nil, fmt.Errorf("resolve bridge: %w", err)
	}

	// Get base image path
	imagePath, err := p.imgStore.GetImagePath(req.BaseImage)
	if err != nil {
		return nil, fmt.Errorf("get base image: %w", err)
	}

	// Use configured kernel path
	kernelPath := p.kernelPath
	if kernelPath == "" {
		return nil, fmt.Errorf("kernel path not configured")
	}

	// Validate initrd exists when configured. Distribution kernels typically
	// need an initramfs to load virtio_blk/ext4 modules - booting without one
	// causes a kernel panic. Set initrd_path: "" in config if not needed.
	initrdPath := p.initrdPath
	if initrdPath != "" {
		if _, err := os.Stat(initrdPath); err != nil {
			return nil, fmt.Errorf("initrd not found at %s (set initrd_path: \"\" in config if not needed): %w", initrdPath, err)
		}
	}

	// Create overlay disk
	overlayPath, err := microvm.CreateOverlay(ctx, imagePath, p.vmMgr.WorkDir(), req.SandboxID)
	if err != nil {
		return nil, fmt.Errorf("create overlay: %w", err)
	}

	// Generate cloud-init NoCloud ISO with catch-all DHCP config so the
	// sandbox gets an IP regardless of the source VM's interface naming.
	cloudInitISO, err := microvm.GenerateCloudInitISO(p.vmMgr.WorkDir(), req.SandboxID, p.caPubKey, p.bridgeIP)
	if err != nil {
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("generate cloud-init ISO: %w", err)
	}

	// Generate MAC address and TAP device
	mac := microvm.GenerateMACAddress()
	tapName := network.TAPName(req.SandboxID)

	// Create TAP device
	if err := network.CreateTAP(ctx, tapName, bridge, p.logger); err != nil {
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("create TAP: %w", err)
	}

	// Apply defaults
	vcpus := req.VCPUs
	if vcpus == 0 {
		vcpus = 2
	}
	memMB := req.MemoryMB
	if memMB == 0 {
		memMB = 2048
	}

	// Launch microVM
	info, err := p.vmMgr.Launch(ctx, microvm.LaunchConfig{
		SandboxID:    req.SandboxID,
		Name:         req.Name,
		OverlayPath:  overlayPath,
		KernelPath:   kernelPath,
		InitrdPath:   initrdPath,
		RootDevice:   p.rootDevice,
		TAPDevice:    tapName,
		MACAddress:   mac,
		Bridge:       bridge,
		VCPUs:        vcpus,
		MemoryMB:     memMB,
		Accel:        p.accel,
		CloudInitISO: cloudInitISO,
	})
	if err != nil {
		_ = network.DestroyTAP(ctx, tapName)
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("launch microVM: %w", err)
	}

	// Discover IP
	ip, err := p.netMgr.DiscoverIP(ctx, mac, bridge, 2*time.Minute)
	if err != nil {
		p.logger.Warn("IP discovery failed", "sandbox_id", req.SandboxID, "error", err)
	}

	return &provider.SandboxResult{
		SandboxID:  req.SandboxID,
		Name:       req.Name,
		State:      "RUNNING",
		IPAddress:  ip,
		MACAddress: mac,
		Bridge:     bridge,
		PID:        info.PID,
	}, nil
}

// ProgressFunc is called to report sandbox creation progress.
type ProgressFunc func(step string, stepNum, total int)

// CreateSandboxWithProgress creates a sandbox while reporting granular progress.
func (p *Provider) CreateSandboxWithProgress(ctx context.Context, req provider.CreateRequest, progress ProgressFunc) (*provider.SandboxResult, error) {
	if p.vmMgr == nil {
		return nil, fmt.Errorf("microVM manager not available")
	}
	if p.readiness != nil {
		p.readiness.Register(req.SandboxID)
		defer p.readiness.Unregister(req.SandboxID)
	}

	const totalSteps = 7

	// Step 1: Resolve bridge
	progress("Resolving network bridge", 1, totalSteps)
	bridge, err := p.netMgr.ResolveBridge(ctx, req.Network)
	if err != nil {
		return nil, fmt.Errorf("resolve bridge: %w", err)
	}

	imagePath, err := p.imgStore.GetImagePath(req.BaseImage)
	if err != nil {
		return nil, fmt.Errorf("get base image: %w", err)
	}

	kernelPath := p.kernelPath
	if kernelPath == "" {
		return nil, fmt.Errorf("kernel path not configured")
	}
	initrdPath := p.initrdPath
	if initrdPath != "" {
		if _, err := os.Stat(initrdPath); err != nil {
			return nil, fmt.Errorf("initrd not found at %s: %w", initrdPath, err)
		}
	}

	// Step 2: Create overlay disk
	progress("Creating overlay disk", 2, totalSteps)
	overlayPath, err := microvm.CreateOverlay(ctx, imagePath, p.vmMgr.WorkDir(), req.SandboxID)
	if err != nil {
		return nil, fmt.Errorf("create overlay: %w", err)
	}

	// Step 3: Generate cloud-init
	progress("Generating cloud-init", 3, totalSteps)
	cloudInitISO, err := microvm.GenerateCloudInitISO(p.vmMgr.WorkDir(), req.SandboxID, p.caPubKey, p.bridgeIP)
	if err != nil {
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("generate cloud-init ISO: %w", err)
	}

	// Step 4: Set up network (TAP)
	progress("Setting up network (TAP)", 4, totalSteps)
	mac := microvm.GenerateMACAddress()
	tapName := network.TAPName(req.SandboxID)
	if err := network.CreateTAP(ctx, tapName, bridge, p.logger); err != nil {
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("create TAP: %w", err)
	}

	vcpus := req.VCPUs
	if vcpus == 0 {
		vcpus = 2
	}
	memMB := req.MemoryMB
	if memMB == 0 {
		memMB = 2048
	}

	// Step 5: Boot microVM
	progress("Booting microVM", 5, totalSteps)
	info, err := p.vmMgr.Launch(ctx, microvm.LaunchConfig{
		SandboxID:    req.SandboxID,
		Name:         req.Name,
		OverlayPath:  overlayPath,
		KernelPath:   kernelPath,
		InitrdPath:   initrdPath,
		RootDevice:   p.rootDevice,
		TAPDevice:    tapName,
		MACAddress:   mac,
		Bridge:       bridge,
		VCPUs:        vcpus,
		MemoryMB:     memMB,
		Accel:        p.accel,
		CloudInitISO: cloudInitISO,
	})
	if err != nil {
		_ = network.DestroyTAP(ctx, tapName)
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("launch microVM: %w", err)
	}

	// Step 6: Discover IP
	progress("Discovering IP address", 6, totalSteps)
	ip, err := p.netMgr.DiscoverIP(ctx, mac, bridge, 2*time.Minute)
	if err != nil {
		p.logger.Warn("IP discovery failed", "sandbox_id", req.SandboxID, "error", err)
	}

	// Step 7: Wait for cloud-init ready
	progress("Waiting for cloud-init ready", 7, totalSteps)
	if p.readiness != nil {
		if waitErr := p.readiness.WaitReady(req.SandboxID, 2*time.Minute); waitErr != nil {
			p.logger.Warn("readiness wait failed, sandbox may not be fully ready", "sandbox_id", req.SandboxID, "error", waitErr)
		}
	}

	return &provider.SandboxResult{
		SandboxID:  req.SandboxID,
		Name:       req.Name,
		State:      "RUNNING",
		IPAddress:  ip,
		MACAddress: mac,
		Bridge:     bridge,
		PID:        info.PID,
	}, nil
}

func (p *Provider) DestroySandbox(ctx context.Context, sandboxID string) error {
	if p.vmMgr != nil {
		info, err := p.vmMgr.Get(sandboxID)
		if err == nil {
			_ = network.DestroyTAP(ctx, info.TAPDevice)
		}
		if err := p.vmMgr.Destroy(ctx, sandboxID); err != nil {
			p.logger.Error("destroy microVM failed", "sandbox_id", sandboxID, "error", err)
		}
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), sandboxID)
	}
	return nil
}

func (p *Provider) StartSandbox(ctx context.Context, sandboxID string) (*provider.SandboxResult, error) {
	if p.vmMgr == nil {
		return nil, fmt.Errorf("microVM manager not available")
	}

	info, err := p.vmMgr.Get(sandboxID)
	if err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}

	ip := ""
	if p.netMgr != nil {
		ip, _ = p.netMgr.DiscoverIP(ctx, info.MACAddress, info.Bridge, 30*time.Second)
	}

	return &provider.SandboxResult{
		SandboxID: sandboxID,
		State:     "RUNNING",
		IPAddress: ip,
	}, nil
}

func (p *Provider) StopSandbox(ctx context.Context, sandboxID string, force bool) error {
	if p.vmMgr == nil {
		return fmt.Errorf("microVM manager not available")
	}
	return p.vmMgr.Stop(ctx, sandboxID, force)
}

func (p *Provider) GetSandboxIP(ctx context.Context, sandboxID string) (string, error) {
	if p.vmMgr == nil {
		return "", fmt.Errorf("microVM manager not available")
	}

	info, err := p.vmMgr.Get(sandboxID)
	if err != nil {
		return "", fmt.Errorf("get sandbox: %w", err)
	}

	if p.netMgr == nil {
		return "", fmt.Errorf("network manager not available")
	}

	return p.netMgr.DiscoverIP(ctx, info.MACAddress, info.Bridge, 30*time.Second)
}

func (p *Provider) CreateSnapshot(_ context.Context, sandboxID, name string) (*provider.SnapshotResult, error) {
	if p.vmMgr == nil {
		return nil, fmt.Errorf("microVM manager not available")
	}

	snapshotID, err := id.Generate("SNP-")
	if err != nil {
		return nil, fmt.Errorf("generate snapshot ID: %w", err)
	}
	return &provider.SnapshotResult{
		SnapshotID:   snapshotID,
		SnapshotName: name,
	}, nil
}

func (p *Provider) RunCommand(ctx context.Context, sandboxID, command string, timeout time.Duration) (*provider.CommandResult, error) {
	if p.vmMgr == nil {
		return nil, fmt.Errorf("microVM manager not available")
	}

	info, err := p.vmMgr.Get(sandboxID)
	if err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}

	ip := ""
	if p.netMgr != nil {
		ip, _ = p.netMgr.DiscoverIP(ctx, info.MACAddress, info.Bridge, 30*time.Second)
	}
	if ip == "" {
		return nil, fmt.Errorf("unable to discover sandbox IP for SSH")
	}

	if p.keyMgr == nil {
		return nil, fmt.Errorf("SSH key manager not available - cannot connect to sandbox")
	}
	creds, err := p.keyMgr.GetCredentials(ctx, sandboxID, "sandbox")
	if err != nil {
		return nil, fmt.Errorf("get sandbox SSH credentials: %w", err)
	}

	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	// Retry loop: sshd may not be ready yet after IP is assigned.
	const maxRetries = 6
	const retryDelay = 5 * time.Second

	start := time.Now()
	var stdout, stderr string
	var exitCode int

	for attempt := 0; attempt <= maxRetries; attempt++ {
		stdout, stderr, exitCode, err = runSSHCommand(ctx, ip, creds, command, timeout)
		if err == nil {
			break
		}

		// Retry on transient errors: sshd not yet listening, or cert auth
		// not yet configured (cloud-init still running).
		errMsg := err.Error()
		isTransient := strings.Contains(errMsg, "Connection refused") ||
			strings.Contains(errMsg, "Connection reset") ||
			strings.Contains(errMsg, "No route to host") ||
			strings.Contains(errMsg, "connection refused") ||
			strings.Contains(errMsg, "connection reset") ||
			strings.Contains(errMsg, "Permission denied")

		if !isTransient || attempt == maxRetries {
			return nil, fmt.Errorf("run command: %w", err)
		}

		p.logger.Info("SSH connection failed, retrying (sshd may still be starting)",
			"sandbox_id", sandboxID,
			"attempt", attempt+1,
			"max_retries", maxRetries,
			"error", err,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryDelay):
		}
	}

	return &provider.CommandResult{
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		DurationMS: time.Since(start).Milliseconds(),
	}, nil
}

func (p *Provider) ListTemplates(_ context.Context) ([]string, error) {
	if p.imgStore == nil {
		return nil, nil
	}
	return p.imgStore.ListNames()
}

func (p *Provider) ListSourceVMs(ctx context.Context) ([]provider.SourceVMInfo, error) {
	if p.srcVMMgr == nil {
		return nil, fmt.Errorf("source VM manager not available")
	}

	vms, err := p.srcVMMgr.ListVMs(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]provider.SourceVMInfo, len(vms))
	for i, vm := range vms {
		result[i] = provider.SourceVMInfo{
			Name:      vm.Name,
			State:     vm.State,
			IPAddress: vm.IPAddress,
			Prepared:  vm.Prepared,
		}
	}
	return result, nil
}

func (p *Provider) ValidateSourceVM(ctx context.Context, vmName string) (*provider.ValidationResult, error) {
	if p.srcVMMgr == nil {
		return nil, fmt.Errorf("source VM manager not available")
	}

	result, err := p.srcVMMgr.ValidateSourceVM(ctx, vmName)
	if err != nil {
		return nil, err
	}

	return &provider.ValidationResult{
		VMName:     result.VMName,
		Valid:      result.Valid,
		State:      result.State,
		MACAddress: result.MACAddress,
		IPAddress:  result.IPAddress,
		HasNetwork: result.HasNetwork,
		Warnings:   result.Warnings,
		Errors:     result.Errors,
	}, nil
}

func (p *Provider) PrepareSourceVM(ctx context.Context, vmName, sshUser, sshKeyPath string) (*provider.PrepareResult, error) {
	if p.srcVMMgr == nil {
		return nil, fmt.Errorf("source VM manager not available")
	}

	result, err := p.srcVMMgr.PrepareSourceVM(ctx, vmName, sshUser, sshKeyPath)
	if err != nil {
		return nil, err
	}

	return &provider.PrepareResult{
		SourceVM:          result.SourceVM,
		IPAddress:         result.IPAddress,
		Prepared:          result.Prepared,
		UserCreated:       result.UserCreated,
		ShellInstalled:    result.ShellInstalled,
		CAKeyInstalled:    result.CAKeyInstalled,
		SSHDConfigured:    result.SSHDConfigured,
		PrincipalsCreated: result.PrincipalsCreated,
		SSHDRestarted:     result.SSHDRestarted,
	}, nil
}

func (p *Provider) RunSourceCommand(ctx context.Context, vmName, command string, timeout time.Duration) (*provider.CommandResult, error) {
	if p.srcVMMgr == nil {
		return nil, fmt.Errorf("source VM manager not available")
	}

	start := time.Now()
	stdout, stderr, exitCode, err := p.srcVMMgr.RunSourceCommand(ctx, vmName, command, timeout)
	if err != nil {
		return nil, err
	}

	return &provider.CommandResult{
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		DurationMS: time.Since(start).Milliseconds(),
	}, nil
}

func (p *Provider) ReadSourceFile(ctx context.Context, vmName, path string) (string, error) {
	if p.srcVMMgr == nil {
		return "", fmt.Errorf("source VM manager not available")
	}
	return p.srcVMMgr.ReadSourceFile(ctx, vmName, path)
}

func (p *Provider) Capabilities(_ context.Context) (*provider.HostCapabilities, error) {
	caps := &provider.HostCapabilities{
		TotalCPUs:     runtime.NumCPU(),
		AvailableCPUs: runtime.NumCPU(),
	}

	// Read system memory from /proc/meminfo
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						caps.TotalMemoryMB = int(kb / 1024)
					}
				}
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						caps.AvailableMemMB = int(kb / 1024)
					}
				}
			}
		}
	}

	if p.imgStore != nil {
		names, _ := p.imgStore.ListNames()
		caps.BaseImages = names
	}

	return caps, nil
}

func (p *Provider) ActiveSandboxCount() int {
	if p.vmMgr == nil {
		return 0
	}
	return len(p.vmMgr.List())
}

func (p *Provider) RecoverState(ctx context.Context) error {
	if p.vmMgr == nil {
		return nil
	}
	return p.vmMgr.RecoverState(ctx)
}

// runSSHCommand executes a command on a sandbox via SSH using cert-based auth.
func runSSHCommand(ctx context.Context, ip string, creds *sshkeys.Credentials, command string, timeout time.Duration) (stdout, stderr string, exitCode int, err error) {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sshArgs := []string{
		"-i", creds.PrivateKeyPath,
		"-o", "CertificateFile=" + creds.CertificatePath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("%s@%s", creds.Username, ip),
		command,
	}

	cmd := exec.CommandContext(cmdCtx, "ssh", sshArgs...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 255 {
				stderrStr := stderrBuf.String()
				return "", stderrStr, 255, fmt.Errorf("ssh failed (exit 255): %s", stderrStr)
			}
			return stdoutBuf.String(), stderrBuf.String(), exitErr.ExitCode(), nil
		}
		// Include stderr in the error for connection diagnostics.
		if stderrStr := stderrBuf.String(); stderrStr != "" {
			return "", stderrStr, -1, fmt.Errorf("%w: %s", err, stderrStr)
		}
		return "", "", -1, err
	}

	return stdoutBuf.String(), stderrBuf.String(), 0, nil
}
