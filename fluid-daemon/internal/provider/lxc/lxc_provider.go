// Package lxc implements the SandboxProvider interface for Proxmox LXC containers.
// It uses the Proxmox REST API to clone container templates, manage lifecycle,
// and execute commands via pct exec.
package lxc

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/id"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider"
)

// Provider implements provider.SandboxProvider for Proxmox LXC containers.
type Provider struct {
	client   *Client
	cfg      Config
	resolver *CTResolver
	logger   *slog.Logger

	// Protects VMID allocation and sandbox tracking.
	mu sync.Mutex
	// sandboxID -> vmid mapping for active sandboxes.
	sandboxes map[string]int
}

// New creates a new LXC provider.
func New(cfg Config, logger *slog.Logger) (*Provider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid lxc config: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}

	client := NewClient(cfg, logger)
	return &Provider{
		client:    client,
		cfg:       cfg,
		resolver:  NewCTResolver(client),
		logger:    logger.With("provider", "lxc"),
		sandboxes: make(map[string]int),
	}, nil
}

func (p *Provider) CreateSandbox(ctx context.Context, req provider.CreateRequest) (*provider.SandboxResult, error) {
	// Resolve source CT template VMID
	sourceVMID, err := p.resolver.ResolveVMID(ctx, req.SourceVM)
	if err != nil {
		return nil, fmt.Errorf("resolve source CT %q: %w", req.SourceVM, err)
	}

	// Allocate next VMID
	p.mu.Lock()
	newVMID, err := p.client.NextVMID(ctx, p.cfg.VMIDStart, p.cfg.VMIDEnd)
	if err != nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("allocate VMID: %w", err)
	}

	hostname := req.Name
	if hostname == "" {
		hostname = fmt.Sprintf("sbx-%s", req.SandboxID[:8])
	}

	p.logger.Info("cloning CT",
		"source_vmid", sourceVMID,
		"new_vmid", newVMID,
		"hostname", hostname,
	)

	// Clone the template
	upid, err := p.client.CloneCT(ctx, sourceVMID, newVMID, hostname, true)
	p.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("clone CT: %w", err)
	}

	if err := p.client.WaitForTask(ctx, upid); err != nil {
		return nil, fmt.Errorf("wait for clone: %w", err)
	}

	// Configure the clone
	params := url.Values{}
	if req.VCPUs > 0 {
		params.Set("cores", fmt.Sprintf("%d", req.VCPUs))
	}
	if req.MemoryMB > 0 {
		params.Set("memory", fmt.Sprintf("%d", req.MemoryMB))
	}

	bridge := p.cfg.Bridge
	if req.Network != "" {
		bridge = req.Network
	}
	params.Set("net0", fmt.Sprintf("name=eth0,bridge=%s,ip=dhcp", bridge))

	if len(params) > 0 {
		if err := p.client.SetCTConfig(ctx, newVMID, params); err != nil {
			// Cleanup on config failure
			_ = p.cleanupCT(ctx, newVMID)
			return nil, fmt.Errorf("configure clone: %w", err)
		}
	}

	// Inject SSH public key if provided
	if req.SSHPublicKey != "" {
		sshParams := url.Values{
			"ssh-public-keys": {url.QueryEscape(strings.TrimSpace(req.SSHPublicKey))},
		}
		if err := p.client.SetCTConfig(ctx, newVMID, sshParams); err != nil {
			p.logger.Warn("failed to inject SSH key", "error", err)
		}
	}

	// Start container
	startUPID, err := p.client.StartCT(ctx, newVMID)
	if err != nil {
		_ = p.cleanupCT(ctx, newVMID)
		return nil, fmt.Errorf("start CT: %w", err)
	}

	if err := p.client.WaitForTask(ctx, startUPID); err != nil {
		_ = p.cleanupCT(ctx, newVMID)
		return nil, fmt.Errorf("wait for start: %w", err)
	}

	// Discover IP by polling interfaces
	ip, err := p.discoverIP(ctx, newVMID, 2*time.Minute)
	if err != nil {
		p.logger.Warn("IP discovery failed", "sandbox_id", req.SandboxID, "error", err)
	}

	// Track sandbox
	p.mu.Lock()
	p.sandboxes[req.SandboxID] = newVMID
	p.mu.Unlock()

	// Refresh resolver cache
	_ = p.resolver.Refresh(ctx)

	return &provider.SandboxResult{
		SandboxID: req.SandboxID,
		Name:      hostname,
		State:     "RUNNING",
		IPAddress: ip,
		Bridge:    bridge,
	}, nil
}

func (p *Provider) DestroySandbox(ctx context.Context, sandboxID string) error {
	p.mu.Lock()
	vmid, ok := p.sandboxes[sandboxID]
	if ok {
		delete(p.sandboxes, sandboxID)
	}
	p.mu.Unlock()

	if !ok {
		return fmt.Errorf("sandbox %s not tracked", sandboxID)
	}

	return p.cleanupCT(ctx, vmid)
}

func (p *Provider) StartSandbox(ctx context.Context, sandboxID string) (*provider.SandboxResult, error) {
	vmid, err := p.getVMID(sandboxID)
	if err != nil {
		return nil, err
	}

	upid, err := p.client.StartCT(ctx, vmid)
	if err != nil {
		return nil, fmt.Errorf("start CT: %w", err)
	}

	if err := p.client.WaitForTask(ctx, upid); err != nil {
		return nil, fmt.Errorf("wait for start: %w", err)
	}

	ip, _ := p.discoverIP(ctx, vmid, 30*time.Second)

	return &provider.SandboxResult{
		SandboxID: sandboxID,
		State:     "RUNNING",
		IPAddress: ip,
	}, nil
}

func (p *Provider) StopSandbox(ctx context.Context, sandboxID string, force bool) error {
	vmid, err := p.getVMID(sandboxID)
	if err != nil {
		return err
	}

	var upid string
	if force {
		upid, err = p.client.StopCT(ctx, vmid)
	} else {
		upid, err = p.client.ShutdownCT(ctx, vmid)
	}
	if err != nil {
		return fmt.Errorf("stop CT: %w", err)
	}

	return p.client.WaitForTask(ctx, upid)
}

func (p *Provider) GetSandboxIP(ctx context.Context, sandboxID string) (string, error) {
	vmid, err := p.getVMID(sandboxID)
	if err != nil {
		return "", err
	}

	return p.discoverIP(ctx, vmid, 30*time.Second)
}

func (p *Provider) CreateSnapshot(ctx context.Context, sandboxID, name string) (*provider.SnapshotResult, error) {
	vmid, err := p.getVMID(sandboxID)
	if err != nil {
		return nil, err
	}

	upid, err := p.client.CreateSnapshot(ctx, vmid, name)
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}

	if err := p.client.WaitForTask(ctx, upid); err != nil {
		return nil, fmt.Errorf("wait for snapshot: %w", err)
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
	vmid, err := p.getVMID(sandboxID)
	if err != nil {
		return nil, err
	}

	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	start := time.Now()
	stdout, stderr, exitCode, err := p.pctExec(ctx, vmid, command, timeout)
	if err != nil {
		return nil, fmt.Errorf("pct exec: %w", err)
	}

	return &provider.CommandResult{
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		DurationMS: time.Since(start).Milliseconds(),
	}, nil
}

func (p *Provider) ListTemplates(ctx context.Context) ([]string, error) {
	cts, err := p.client.ListCTs(ctx)
	if err != nil {
		return nil, err
	}

	var templates []string
	for _, ct := range cts {
		if ct.Template == 1 {
			templates = append(templates, ct.Name)
		}
	}
	return templates, nil
}

func (p *Provider) ListSourceVMs(ctx context.Context) ([]provider.SourceVMInfo, error) {
	cts, err := p.client.ListCTs(ctx)
	if err != nil {
		return nil, err
	}

	var vms []provider.SourceVMInfo
	for _, ct := range cts {
		// Skip templates and sandbox containers
		if ct.Template == 1 || strings.HasPrefix(ct.Name, "sbx-") {
			continue
		}
		ip := ""
		if ct.Status == "running" {
			ip, _ = p.discoverIP(ctx, ct.VMID, 5*time.Second)
		}
		vms = append(vms, provider.SourceVMInfo{
			Name:      ct.Name,
			State:     ct.Status,
			IPAddress: ip,
		})
	}
	return vms, nil
}

func (p *Provider) ValidateSourceVM(ctx context.Context, vmName string) (*provider.ValidationResult, error) {
	result := &provider.ValidationResult{
		VMName: vmName,
	}

	vmid, err := p.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("CT %q not found: %v", vmName, err))
		return result, nil
	}

	status, err := p.client.GetCTStatus(ctx, vmid)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to get CT status: %v", err))
		return result, nil
	}
	result.State = status.Status

	cfg, err := p.client.GetCTConfig(ctx, vmid)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("could not read CT config: %v", err))
	} else {
		if cfg.Net0 == "" {
			result.HasNetwork = false
			result.Warnings = append(result.Warnings, "CT has no network interface (net0)")
		} else {
			result.HasNetwork = true
		}
	}

	if status.Status == "running" {
		ip, err := p.discoverIP(ctx, vmid, 10*time.Second)
		if err == nil {
			result.IPAddress = ip
		} else {
			result.Warnings = append(result.Warnings, "Could not determine IP address")
		}
	}

	if len(result.Errors) == 0 {
		result.Valid = true
	}

	return result, nil
}

func (p *Provider) PrepareSourceVM(ctx context.Context, vmName, sshUser, sshKeyPath string) (*provider.PrepareResult, error) {
	vmid, err := p.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("resolve CT %q: %w", vmName, err)
	}

	ip, _ := p.discoverIP(ctx, vmid, 10*time.Second)

	// Install fluid-readonly user + restricted shell via pct exec
	steps := []struct {
		name  string
		cmd   string
		field func(*provider.PrepareResult)
	}{
		{
			name:  "install restricted shell",
			cmd:   "cat > /usr/local/bin/fluid-readonly-shell << 'EOF'\n#!/bin/bash\nset -euo pipefail\nif [ -n \"${SSH_ORIGINAL_COMMAND:-}\" ]; then CMD=\"$SSH_ORIGINAL_COMMAND\"; elif [ \"${1:-}\" = \"-c\" ] && [ -n \"${2:-}\" ]; then CMD=\"$2\"; else echo 'ERROR: Interactive login not permitted.' >&2; exit 1; fi\nexec /bin/bash -c \"$CMD\"\nEOF\nchmod 755 /usr/local/bin/fluid-readonly-shell",
			field: func(r *provider.PrepareResult) { r.ShellInstalled = true },
		},
		{
			name:  "create fluid-readonly user",
			cmd:   "mkdir -p /var/empty && id fluid-readonly >/dev/null 2>&1 || useradd -r -s /usr/local/bin/fluid-readonly-shell -d /var/empty -M fluid-readonly",
			field: func(r *provider.PrepareResult) { r.UserCreated = true },
		},
	}

	result := &provider.PrepareResult{
		SourceVM:  vmName,
		IPAddress: ip,
	}

	for _, step := range steps {
		_, stderr, exitCode, err := p.pctExec(ctx, vmid, step.cmd, 60*time.Second)
		if err != nil {
			return result, fmt.Errorf("%s: %w", step.name, err)
		}
		if exitCode != 0 {
			return result, fmt.Errorf("%s: exit %d: %s", step.name, exitCode, stderr)
		}
		step.field(result)
	}

	result.Prepared = true
	return result, nil
}

func (p *Provider) RunSourceCommand(ctx context.Context, vmName, command string, timeout time.Duration) (*provider.CommandResult, error) {
	vmid, err := p.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("resolve CT %q: %w", vmName, err)
	}

	if timeout == 0 {
		timeout = 30 * time.Second
	}

	start := time.Now()
	// Execute as fluid-readonly user via pct exec
	wrappedCmd := fmt.Sprintf("su -s /usr/local/bin/fluid-readonly-shell fluid-readonly -c '%s'",
		strings.ReplaceAll(command, "'", "'\"'\"'"))

	stdout, stderr, exitCode, err := p.pctExec(ctx, vmid, wrappedCmd, timeout)
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
	vmid, err := p.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		return "", fmt.Errorf("resolve CT %q: %w", vmName, err)
	}

	// Read file as fluid-readonly user
	cmd := fmt.Sprintf("su - fluid-readonly -c 'cat %s'",
		strings.ReplaceAll(path, "'", "'\"'\"'"))

	stdout, stderr, exitCode, err := p.pctExec(ctx, vmid, cmd, 30*time.Second)
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", fmt.Errorf("read file failed (exit %d): %s", exitCode, stderr)
	}

	return stdout, nil
}

func (p *Provider) Capabilities(ctx context.Context) (*provider.HostCapabilities, error) {
	nodeStatus, err := p.client.GetNodeStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("get node status: %w", err)
	}

	caps := &provider.HostCapabilities{
		TotalCPUs:       nodeStatus.MaxCPU,
		AvailableCPUs:   nodeStatus.MaxCPU,
		TotalMemoryMB:   int(nodeStatus.Memory.Total / (1024 * 1024)),
		AvailableMemMB:  int(nodeStatus.Memory.Free / (1024 * 1024)),
		TotalDiskMB:     int(nodeStatus.RootFS.Total / (1024 * 1024)),
		AvailableDiskMB: int(nodeStatus.RootFS.Available / (1024 * 1024)),
	}

	// Get template names
	templates, _ := p.ListTemplates(ctx)
	caps.BaseImages = templates

	return caps, nil
}

func (p *Provider) ActiveSandboxCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sandboxes)
}

func (p *Provider) RecoverState(ctx context.Context) error {
	cts, err := p.client.ListCTs(ctx)
	if err != nil {
		return fmt.Errorf("list CTs for recovery: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, ct := range cts {
		if strings.HasPrefix(ct.Name, "sbx-") && ct.Template == 0 {
			// Use the CT name as sandbox ID for recovery
			sandboxID := ct.Name
			p.sandboxes[sandboxID] = ct.VMID
			p.logger.Info("recovered sandbox CT", "sandbox_id", sandboxID, "vmid", ct.VMID)
		}
	}

	return nil
}

// --- Internal helpers ---

// getVMID returns the Proxmox VMID for a tracked sandbox.
func (p *Provider) getVMID(sandboxID string) (int, error) {
	p.mu.Lock()
	vmid, ok := p.sandboxes[sandboxID]
	p.mu.Unlock()
	if !ok {
		return 0, fmt.Errorf("sandbox %s not tracked", sandboxID)
	}
	return vmid, nil
}

// cleanupCT stops and deletes a container.
func (p *Provider) cleanupCT(ctx context.Context, vmid int) error {
	// Check status first
	status, err := p.client.GetCTStatus(ctx, vmid)
	if err != nil {
		// CT may already be gone
		p.logger.Warn("get CT status for cleanup failed", "vmid", vmid, "error", err)
		return nil
	}

	if status.Status == "running" {
		stopUPID, err := p.client.StopCT(ctx, vmid)
		if err != nil {
			p.logger.Error("stop CT for cleanup failed", "vmid", vmid, "error", err)
		} else {
			_ = p.client.WaitForTask(ctx, stopUPID)
		}
	}

	delUPID, err := p.client.DeleteCT(ctx, vmid)
	if err != nil {
		return fmt.Errorf("delete CT %d: %w", vmid, err)
	}

	if err := p.client.WaitForTask(ctx, delUPID); err != nil {
		return fmt.Errorf("wait for delete CT %d: %w", vmid, err)
	}

	_ = p.resolver.Refresh(ctx)
	return nil
}

// discoverIP polls the CT interfaces endpoint until an IPv4 address appears.
func (p *Provider) discoverIP(ctx context.Context, vmid int, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		ifaces, err := p.client.GetCTInterfaces(ctx, vmid)
		if err == nil {
			for _, iface := range ifaces {
				if iface.Name == "lo" {
					continue
				}
				if iface.Inet != "" {
					// inet format: "10.0.0.5/24" - strip prefix
					ipStr := strings.SplitN(iface.Inet, "/", 2)[0]
					ip := net.ParseIP(ipStr)
					if ip != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
						return ipStr, nil
					}
				}
			}
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for IP of CT %d", vmid)
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}
	}
}

// pctExec runs a command inside a container via pct exec.
func (p *Provider) pctExec(ctx context.Context, vmid int, command string, timeout time.Duration) (stdout, stderr string, exitCode int, err error) {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "pct", "exec", fmt.Sprintf("%d", vmid), "--", "sh", "-c", command)
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
