// Package microvm implements the SandboxProvider interface for QEMU microVMs.
// It wraps the existing microvm, network, image, and sourcevm packages.
package microvm

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/id"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/image"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/microvm"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/network"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/sourcevm"
)

// Provider implements provider.SandboxProvider for QEMU microVMs.
type Provider struct {
	vmMgr    *microvm.Manager
	netMgr   *network.NetworkManager
	imgStore *image.Store
	srcVMMgr *sourcevm.Manager
	logger   *slog.Logger
}

// New creates a new microVM provider.
func New(
	vmMgr *microvm.Manager,
	netMgr *network.NetworkManager,
	imgStore *image.Store,
	srcVMMgr *sourcevm.Manager,
	logger *slog.Logger,
) *Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{
		vmMgr:    vmMgr,
		netMgr:   netMgr,
		imgStore: imgStore,
		srcVMMgr: srcVMMgr,
		logger:   logger.With("provider", "microvm"),
	}
}

func (p *Provider) CreateSandbox(ctx context.Context, req provider.CreateRequest) (*provider.SandboxResult, error) {
	if p.vmMgr == nil {
		return nil, fmt.Errorf("microVM manager not available")
	}

	// Resolve bridge
	bridge, err := p.netMgr.ResolveBridge(ctx, req.SourceVM, req.Network)
	if err != nil {
		return nil, fmt.Errorf("resolve bridge: %w", err)
	}

	// Get base image path
	imagePath, err := p.imgStore.GetImagePath(req.BaseImage)
	if err != nil {
		return nil, fmt.Errorf("get base image: %w", err)
	}

	// Get kernel path
	kernelPath, err := p.imgStore.GetKernelPath(req.BaseImage)
	if err != nil {
		return nil, fmt.Errorf("get kernel: %w", err)
	}

	// Create overlay disk
	overlayPath, err := microvm.CreateOverlay(ctx, imagePath, p.vmMgr.WorkDir(), req.SandboxID)
	if err != nil {
		return nil, fmt.Errorf("create overlay: %w", err)
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
		SandboxID:   req.SandboxID,
		Name:        req.Name,
		OverlayPath: overlayPath,
		KernelPath:  kernelPath,
		TAPDevice:   tapName,
		MACAddress:  mac,
		Bridge:      bridge,
		VCPUs:       vcpus,
		MemoryMB:    memMB,
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

	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	start := time.Now()
	stdout, stderr, exitCode, err := runSSHCommand(ctx, ip, command, timeout)
	if err != nil {
		return nil, fmt.Errorf("run command: %w", err)
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

// runSSHCommand executes a command on a sandbox via SSH.
func runSSHCommand(ctx context.Context, ip, command string, timeout time.Duration) (stdout, stderr string, exitCode int, err error) {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("sandbox@%s", ip),
		command,
	}

	cmd := exec.CommandContext(cmdCtx, "ssh", sshArgs...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return stdoutBuf.String(), stderrBuf.String(), exitErr.ExitCode(), nil
		}
		return "", "", -1, err
	}

	return stdoutBuf.String(), stderrBuf.String(), 0, nil
}
