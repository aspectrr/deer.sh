// Package microvm implements the SandboxProvider interface for QEMU microVMs.
// It wraps the existing microvm, network, image, and sourcevm packages.
package microvm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/id"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/image"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/microvm"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/network"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/provider"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sourcevm"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sshkeys"
)

// ReadinessWaiter can wait for a sandbox to signal readiness via phone_home.
type ReadinessWaiter interface {
	Register(sandboxID string)
	Unregister(sandboxID string)
	WaitReady(sandboxID string, timeout time.Duration) error
	ReadyIP(sandboxID string) string
}

// Provider implements provider.SandboxProvider for QEMU microVMs.
type Provider struct {
	vmMgr             *microvm.Manager
	netMgr            *network.NetworkManager
	imgStore          *image.Store
	srcVMMgr          *sourcevm.Manager
	keyMgr            sshkeys.KeyProvider
	kernelPath        string
	initrdPath        string
	rootDevice        string
	accel             string
	ipTimeout         time.Duration
	readyTimeout      time.Duration
	caPubKey          string
	bridgeIP          string
	readiness         ReadinessWaiter
	redpandaCacheURL  string // local Redpanda tarball for faster boot
	disableCloudInit  bool   // skip cloud-init for pre-baked images
	socketVMNetClient string // macOS: path to socket_vmnet_client binary
	socketVMNetPath   string // macOS: Unix socket path for socket_vmnet daemon
	logger            *slog.Logger
}

const (
	defaultSandboxIPDiscoveryTimeout = 5 * time.Minute
	defaultSandboxReadinessTimeout   = 15 * time.Minute
)

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
	ipDiscoveryTimeout time.Duration,
	readinessTimeout time.Duration,
	caPubKey string,
	bridgeIP string,
	readiness ReadinessWaiter,
	redpandaCacheURL string,
	disableCloudInit bool,
	socketVMNetClient string,
	socketVMNetPath string,
	logger *slog.Logger,
) *Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{
		vmMgr:             vmMgr,
		netMgr:            netMgr,
		imgStore:          imgStore,
		srcVMMgr:          srcVMMgr,
		keyMgr:            keyMgr,
		kernelPath:        kernelPath,
		initrdPath:        initrdPath,
		rootDevice:        rootDevice,
		accel:             accel,
		ipTimeout:         ipDiscoveryTimeout,
		readyTimeout:      readinessTimeout,
		caPubKey:          caPubKey,
		bridgeIP:          bridgeIP,
		readiness:         readiness,
		redpandaCacheURL:  redpandaCacheURL,
		disableCloudInit:  disableCloudInit,
		socketVMNetClient: socketVMNetClient,
		socketVMNetPath:   socketVMNetPath,
		logger:            logger.With("provider", "microvm"),
	}
}

func (p *Provider) CreateSandbox(ctx context.Context, req provider.CreateRequest) (*provider.SandboxResult, error) {
	if p.vmMgr == nil {
		return nil, fmt.Errorf("microVM manager not available")
	}
	req, clamped := provider.NormalizeCreateRequestResources(req, provider.DefaultSandboxVCPUs, provider.DefaultSandboxMemMB)
	if clamped {
		p.logger.Info("clamped kafka-backed sandbox resources",
			"sandbox_id", req.SandboxID,
			"effective_vcpus", req.VCPUs,
			"effective_memory_mb", req.MemoryMB,
		)
	}
	if p.readiness != nil {
		p.readiness.Register(req.SandboxID)
		defer p.readiness.Unregister(req.SandboxID)
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
	overlayPath, err := microvm.CreateOverlay(ctx, imagePath, p.vmMgr.WorkDir(), req.SandboxID, req.DiskSizeGB())
	if err != nil {
		return nil, fmt.Errorf("create overlay: %w", err)
	}

	// Generate cloud-init NoCloud ISO with catch-all DHCP config so the
	// sandbox gets an IP regardless of the source VM's interface naming.
	cloudInitISO, err := microvm.GenerateCloudInitISO(p.vmMgr.WorkDir(), req.SandboxID, microvm.CloudInitOptions{
		CAPubKey:            p.caPubKey,
		PhoneHomeURL:        p.phoneHomeURL(req.SandboxID),
		KafkaBroker:         kafkaBrokerOptions(req),
		ElasticsearchBroker: elasticsearchBrokerOptions(req),
		RedpandaCacheURL:    p.redpandaCacheURL,
		Disable:             p.disableCloudInit,
	})
	if err != nil {
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("generate cloud-init ISO: %w", err)
	}

	// Generate MAC address; create TAP device unless using socket_vmnet
	mac := microvm.GenerateMACAddress()
	tapName := ""
	if p.socketVMNetClient == "" {
		tapName = network.TAPName(req.SandboxID)
		tapName, err = network.CreateTAP(ctx, tapName, bridge, p.logger)
		if err != nil {
			_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
			return nil, fmt.Errorf("create TAP: %w", err)
		}
	}

	// Launch microVM
	info, err := p.vmMgr.Launch(ctx, microvm.LaunchConfig{
		SandboxID:         req.SandboxID,
		Name:              req.Name,
		OverlayPath:       overlayPath,
		KernelPath:        kernelPath,
		InitrdPath:        initrdPath,
		RootDevice:        p.rootDevice,
		TAPDevice:         tapName,
		MACAddress:        mac,
		Bridge:            bridge,
		VCPUs:             req.VCPUs,
		MemoryMB:          req.MemoryMB,
		Accel:             p.accel,
		CloudInitISO:      cloudInitISO,
		SocketVMNetClient: p.socketVMNetClient,
		SocketVMNetPath:   p.socketVMNetPath,
	})
	if err != nil {
		if tapName != "" {
			_ = network.DestroyTAP(ctx, tapName)
		}
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("launch microVM: %w", err)
	}

	return p.completeCreate(ctx, req, info, mac, bridge, tapName)
}

// ProgressFunc is called to report sandbox creation progress.
type ProgressFunc func(step string, stepNum, total int)

// CreateSandboxWithProgress creates a sandbox while reporting granular progress.
func (p *Provider) CreateSandboxWithProgress(ctx context.Context, req provider.CreateRequest, progress ProgressFunc) (*provider.SandboxResult, error) {
	if p.vmMgr == nil {
		return nil, fmt.Errorf("microVM manager not available")
	}
	req, clamped := provider.NormalizeCreateRequestResources(req, provider.DefaultSandboxVCPUs, provider.DefaultSandboxMemMB)
	if clamped {
		p.logger.Info("clamped kafka-backed sandbox resources",
			"sandbox_id", req.SandboxID,
			"effective_vcpus", req.VCPUs,
			"effective_memory_mb", req.MemoryMB,
		)
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
	overlayPath, err := microvm.CreateOverlay(ctx, imagePath, p.vmMgr.WorkDir(), req.SandboxID, req.DiskSizeGB())
	if err != nil {
		return nil, fmt.Errorf("create overlay: %w", err)
	}

	// Step 3: Generate cloud-init
	progress("Generating cloud-init", 3, totalSteps)
	cloudInitISO, err := microvm.GenerateCloudInitISO(p.vmMgr.WorkDir(), req.SandboxID, microvm.CloudInitOptions{
		CAPubKey:            p.caPubKey,
		PhoneHomeURL:        p.phoneHomeURL(req.SandboxID),
		KafkaBroker:         kafkaBrokerOptions(req),
		ElasticsearchBroker: elasticsearchBrokerOptions(req),
		RedpandaCacheURL:    p.redpandaCacheURL,
		Disable:             p.disableCloudInit,
	})
	if err != nil {
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("generate cloud-init ISO: %w", err)
	}

	// Step 4: Set up network (TAP or socket_vmnet)
	progress("Setting up network", 4, totalSteps)
	mac := microvm.GenerateMACAddress()
	tapName := ""
	if p.socketVMNetClient == "" {
		tapName = network.TAPName(req.SandboxID)
		tapName, err = network.CreateTAP(ctx, tapName, bridge, p.logger)
		if err != nil {
			_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
			return nil, fmt.Errorf("create TAP: %w", err)
		}
	}

	// Step 5: Boot microVM
	progress("Booting microVM", 5, totalSteps)
	info, err := p.vmMgr.Launch(ctx, microvm.LaunchConfig{
		SandboxID:         req.SandboxID,
		Name:              req.Name,
		OverlayPath:       overlayPath,
		KernelPath:        kernelPath,
		InitrdPath:        initrdPath,
		RootDevice:        p.rootDevice,
		TAPDevice:         tapName,
		MACAddress:        mac,
		Bridge:            bridge,
		VCPUs:             req.VCPUs,
		MemoryMB:          req.MemoryMB,
		Accel:             p.accel,
		CloudInitISO:      cloudInitISO,
		SocketVMNetClient: p.socketVMNetClient,
		SocketVMNetPath:   p.socketVMNetPath,
	})
	if err != nil {
		if tapName != "" {
			_ = network.DestroyTAP(ctx, tapName)
		}
		_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), req.SandboxID)
		return nil, fmt.Errorf("launch microVM: %w", err)
	}

	// Step 6: Discover IP
	progress("Discovering IP address", 6, totalSteps)
	progress("Waiting for cloud-init ready", 7, totalSteps)
	return p.completeCreate(ctx, req, info, mac, bridge, tapName)
}

func (p *Provider) DestroySandbox(ctx context.Context, sandboxID string) error {
	if p.vmMgr == nil {
		return nil
	}
	info, err := p.vmMgr.Get(sandboxID)
	if err == nil && info.TAPDevice != "" {
		_ = network.DestroyTAP(ctx, info.TAPDevice)
	}
	var destroyErr error
	if err := p.vmMgr.Destroy(ctx, sandboxID); err != nil {
		p.logger.Error("destroy microVM failed", "sandbox_id", sandboxID, "error", err)
		destroyErr = err
	}
	_ = microvm.RemoveOverlay(p.vmMgr.WorkDir(), sandboxID)
	return destroyErr
}

func (p *Provider) StartSandbox(ctx context.Context, sandboxID string) (*provider.SandboxResult, error) {
	if p.vmMgr == nil {
		return nil, fmt.Errorf("microVM manager not available")
	}

	info, err := p.vmMgr.Get(sandboxID)
	if err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}

	ip := info.IPAddress
	if ip == "" && p.netMgr != nil {
		ip, _ = p.netMgr.DiscoverIP(ctx, info.MACAddress, info.Bridge, p.resolvedIPDiscoveryTimeout())
		if ip != "" {
			p.vmMgr.SetIP(sandboxID, ip)
		}
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

	if info.IPAddress != "" {
		return info.IPAddress, nil
	}

	if p.netMgr == nil {
		return "", fmt.Errorf("network manager not available")
	}

	ip, err := p.netMgr.DiscoverIP(ctx, info.MACAddress, info.Bridge, p.resolvedIPDiscoveryTimeout())
	if err != nil {
		return "", err
	}
	if ip != "" {
		p.vmMgr.SetIP(sandboxID, ip)
	}
	return ip, nil
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

	ip := info.IPAddress
	if ip == "" && p.netMgr != nil {
		var discoverErr error
		ip, discoverErr = p.netMgr.DiscoverIP(ctx, info.MACAddress, info.Bridge, p.resolvedIPDiscoveryTimeout())
		if discoverErr != nil {
			p.logger.Warn("IP discovery failed in RunCommand", "sandbox_id", sandboxID, "error", discoverErr)
		}
		if ip != "" {
			p.vmMgr.SetIP(sandboxID, ip)
		}
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
			strings.Contains(errMsg, "Permission denied") ||
			strings.Contains(errMsg, "Received disconnect") ||
			strings.Contains(errMsg, "Too many authentication failures")

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
	// For the microVM provider, source VMs are base QCOW2 images in the image store.
	// Return those rather than querying libvirt (which is unused in this provider).
	imgs, err := p.imgStore.List()
	if err != nil {
		return nil, err
	}
	result := make([]provider.SourceVMInfo, 0, len(imgs))
	for _, img := range imgs {
		result = append(result, provider.SourceVMInfo{
			Name:  img.Name,
			State: "available",
		})
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

func readinessURL(bridgeIP, sandboxID string) string {
	if bridgeIP == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:9092/ready/%s", bridgeIP, sandboxID)
}

func (p *Provider) phoneHomeURL(sandboxID string) string {
	if p.readiness == nil {
		return ""
	}
	return readinessURL(p.bridgeIP, sandboxID)
}

func (p *Provider) resolvedIPDiscoveryTimeout() time.Duration {
	if p.ipTimeout > 0 {
		return p.ipTimeout
	}
	return defaultSandboxIPDiscoveryTimeout
}

func (p *Provider) resolvedReadinessTimeout() time.Duration {
	if p.readyTimeout > 0 {
		return p.readyTimeout
	}
	return defaultSandboxReadinessTimeout
}

func (p *Provider) applyReadinessIPFallback(sandboxID, discoveredIP string) string {
	if discoveredIP != "" || p.readiness == nil {
		return discoveredIP
	}
	if readyIP := p.readiness.ReadyIP(sandboxID); readyIP != "" {
		p.logger.Info("using readiness callback IP fallback", "sandbox_id", sandboxID, "ip", readyIP)
		return readyIP
	}
	return discoveredIP
}

func (p *Provider) completeCreate(ctx context.Context, req provider.CreateRequest, info *microvm.SandboxInfo, mac, bridge, tapName string) (*provider.SandboxResult, error) {
	ip := ""
	if p.netMgr != nil {
		discoveredIP, err := p.netMgr.DiscoverIP(ctx, mac, bridge, p.resolvedIPDiscoveryTimeout())
		if err != nil {
			p.logger.Warn("IP discovery failed", "sandbox_id", req.SandboxID, "error", err)
		}
		ip = discoveredIP
	}
	ip = p.applyReadinessIPFallback(req.SandboxID, ip)

	if err := p.waitForReadiness(ctx, req.SandboxID, info.PID); err != nil {
		cleanupErr := p.cleanupFailedCreate(context.Background(), req.SandboxID, tapName)
		if cleanupErr != nil {
			return nil, fmt.Errorf("%w\ncleanup_error: %v\nhost_diagnostics:\n%s", err, cleanupErr, sandboxHostDiagnostics(p.vmMgr.WorkDir(), req.SandboxID, info.PID))
		}
		return nil, fmt.Errorf("%w\nhost_diagnostics:\n%s", err, sandboxHostDiagnostics(p.vmMgr.WorkDir(), req.SandboxID, info.PID))
	}

	ip = p.applyReadinessIPFallback(req.SandboxID, ip)
	if ip != "" && p.vmMgr != nil {
		p.vmMgr.SetIP(req.SandboxID, ip)
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

func (p *Provider) waitForReadiness(ctx context.Context, sandboxID string, pid int) error {
	if p.readiness == nil || p.phoneHomeURL(sandboxID) == "" {
		return nil
	}

	deadline := time.Now().Add(p.resolvedReadinessTimeout())
	for {
		waitSlice := 2 * time.Second
		if remaining := time.Until(deadline); remaining < waitSlice {
			waitSlice = remaining
		}
		if waitSlice <= 0 {
			return fmt.Errorf("phone_home readiness timeout for sandbox %s after %v", sandboxID, p.resolvedReadinessTimeout())
		}

		err := p.readiness.WaitReady(sandboxID, waitSlice)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("wait for phone_home readiness: %w", ctx.Err())
		}
		if !isReadinessTimeoutErr(err) {
			return fmt.Errorf("wait for phone_home readiness: %w", err)
		}

		running, state, runErr := p.sandboxRunning(sandboxID, pid)
		if runErr != nil {
			return fmt.Errorf("sandbox %s failed before phone_home: %w", sandboxID, runErr)
		}
		if !running {
			return fmt.Errorf("sandbox %s exited before phone_home (state=%s)", sandboxID, state)
		}
	}
}

func (p *Provider) sandboxRunning(sandboxID string, _ int) (bool, microvm.SandboxState, error) {
	if p.vmMgr == nil {
		return true, microvm.StateRunning, nil
	}
	info, err := p.vmMgr.Get(sandboxID)
	if err != nil {
		return false, microvm.StateError, err
	}
	return info.State == microvm.StateRunning, info.State, nil
}

func (p *Provider) cleanupFailedCreate(ctx context.Context, sandboxID, tapName string) error {
	var errs []string
	if tapName != "" {
		if err := network.DestroyTAP(ctx, tapName); err != nil {
			errs = append(errs, fmt.Sprintf("destroy TAP %s: %v", tapName, err))
		}
	}
	if p.vmMgr != nil {
		if err := p.vmMgr.Destroy(ctx, sandboxID); err != nil {
			errs = append(errs, fmt.Sprintf("destroy sandbox %s: %v", sandboxID, err))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, "; "))
}

func isReadinessTimeoutErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "readiness timeout")
}

func recentKernelLog() string {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return ""
	}
	cmd := exec.Command("journalctl", "-k", "--no-pager", "-n", "200")
	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func kernelOOMDiagnosticsForPID(pid int, kernelLog string) string {
	if pid <= 0 || strings.TrimSpace(kernelLog) == "" {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(kernelLog), "\n")
	pidStr := strconv.Itoa(pid)
	matchLine := func(line string) bool {
		return strings.Contains(line, "Killed process "+pidStr+" ") ||
			strings.Contains(line, "pid="+pidStr+" ") ||
			strings.Contains(line, "pid="+pidStr+"]") ||
			strings.Contains(line, "["+pidStr+"]")
	}

	seen := make(map[int]struct{})
	matched := make([]string, 0)
	for i, line := range lines {
		if !matchLine(line) {
			continue
		}
		start := i - 4
		if start < 0 {
			start = 0
		}
		end := i + 3
		if end > len(lines) {
			end = len(lines)
		}
		for idx := start; idx < end; idx++ {
			if _, ok := seen[idx]; ok {
				continue
			}
			seen[idx] = struct{}{}
			matched = append(matched, lines[idx])
		}
	}

	return strings.TrimSpace(strings.Join(matched, "\n"))
}

func sandboxHostDiagnostics(workDir, sandboxID string, pid int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "sandbox_dir: %s\n", filepath.Join(workDir, sandboxID))
	if pid == 0 {
		if pidBytes, err := os.ReadFile(filepath.Join(workDir, sandboxID, "qemu.pid")); err == nil {
			if parsed, convErr := strconv.Atoi(strings.TrimSpace(string(pidBytes))); convErr == nil {
				pid = parsed
			}
		}
	}
	if pid != 0 {
		fmt.Fprintf(&b, "qemu_pid: %d\n", pid)
		proc, err := os.FindProcess(pid)
		switch {
		case err != nil:
			fmt.Fprintf(&b, "qemu_alive: false (%v)\n", err)
		case proc.Signal(syscall.Signal(0)) != nil:
			fmt.Fprintf(&b, "qemu_alive: false\n")
		default:
			fmt.Fprintf(&b, "qemu_alive: true\n")
		}
	}
	if oomLines := kernelOOMDiagnosticsForPID(pid, recentKernelLog()); oomLines != "" {
		fmt.Fprintf(&b, "===== kernel_oom.log =====\n%s\n", oomLines)
	}

	for _, name := range []string{"qemu.pid", "metadata.json", "qemu-stderr.log", "qemu-events.log", "serial.log"} {
		path := filepath.Join(workDir, sandboxID, name)
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(&b, "===== %s =====\nmissing: %v\n", name, err)
			continue
		}
		fmt.Fprintf(&b, "===== %s =====\n%s\n", name, strings.TrimSpace(string(data)))
		if name == "metadata.json" {
			var meta struct {
				VCPUs    int `json:"vcpus"`
				MemoryMB int `json:"memory_mb"`
			}
			if err := json.Unmarshal(data, &meta); err == nil {
				fmt.Fprintf(&b, "effective_vcpus: %d\n", meta.VCPUs)
				fmt.Fprintf(&b, "effective_memory_mb: %d\n", meta.MemoryMB)
			}
		}
	}

	return strings.TrimSpace(b.String())
}

func kafkaBrokerOptions(req provider.CreateRequest) microvm.KafkaBrokerOptions {
	if req.KafkaBroker != nil {
		port := req.KafkaBroker.Port
		if port == 0 {
			port = 9092
		}
		return microvm.KafkaBrokerOptions{
			Enabled:          true,
			AdvertiseAddress: req.KafkaBroker.AdvertiseAddress,
			ArchiveURL:       req.KafkaBroker.ArchiveURL,
			Port:             port,
		}
	}
	for _, ds := range req.DataSources {
		if ds.Type == provider.DataSourceTypeKafka {
			return microvm.KafkaBrokerOptions{
				Enabled: true,
				Port:    9092,
			}
		}
	}
	return microvm.KafkaBrokerOptions{}
}

func elasticsearchBrokerOptions(req provider.CreateRequest) microvm.ElasticsearchBrokerOptions {
	if req.ElasticsearchBroker != nil {
		port := req.ElasticsearchBroker.Port
		if port == 0 {
			port = 9200
		}
		return microvm.ElasticsearchBrokerOptions{
			Enabled:    true,
			Port:       port,
			ArchiveURL: req.ElasticsearchBroker.ArchiveURL,
		}
	}
	return microvm.ElasticsearchBrokerOptions{}
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
