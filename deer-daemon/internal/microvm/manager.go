package microvm

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// SandboxState represents the lifecycle state of a microVM sandbox.
type SandboxState string

const (
	StateCreating SandboxState = "CREATING"
	StateRunning  SandboxState = "RUNNING"
	StateStopped  SandboxState = "STOPPED"
	StateError    SandboxState = "ERROR"
)

// SandboxInfo holds runtime information about a managed microVM.
type SandboxInfo struct {
	ID         string
	Name       string
	PID        int
	State      SandboxState
	OverlayDir string
	TAPDevice  string
	MACAddress string
	Bridge     string
	VCPUs      int
	MemoryMB   int
	IPAddress  string
}

// Manager manages QEMU microVM processes.
type Manager struct {
	mu      sync.RWMutex
	vms     map[string]*SandboxInfo // sandbox_id -> info
	qmpStop map[string]context.CancelFunc
	qemuBin string
	workDir string
	logger  *slog.Logger
}

// NewManager creates a new microVM manager.
func NewManager(qemuBin, workDir string, logger *slog.Logger) (*Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Resolve qemu binary
	bin, err := exec.LookPath(qemuBin)
	if err != nil {
		return nil, fmt.Errorf("qemu binary not found: %w", err)
	}

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}

	m := &Manager{
		vms:     make(map[string]*SandboxInfo),
		qmpStop: make(map[string]context.CancelFunc),
		qemuBin: bin,
		workDir: workDir,
		logger:  logger.With("component", "microvm"),
	}

	return m, nil
}

// WorkDir returns the working directory for sandbox data.
func (m *Manager) SetIP(sandboxID, ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if info, ok := m.vms[sandboxID]; ok {
		info.IPAddress = ip
	}
}

func (m *Manager) WorkDir() string {
	return m.workDir
}

// RecoverState scans the work directory for PID files and rebuilds in-memory state.
// Called on daemon restart to reconnect with running QEMU processes.
func (m *Manager) RecoverState(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := os.ReadDir(m.workDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read work dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sandboxID := entry.Name()
		pidFile := filepath.Join(m.workDir, sandboxID, "qemu.pid")

		pidBytes, err := os.ReadFile(pidFile)
		if err != nil {
			m.logger.Warn("no PID file, skipping", "sandbox_id", sandboxID)
			continue
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
		if err != nil {
			m.logger.Warn("invalid PID file", "sandbox_id", sandboxID, "error", err)
			continue
		}

		// Check if process is still alive
		proc, err := os.FindProcess(pid)
		if err != nil {
			m.logger.Warn("process not found", "sandbox_id", sandboxID, "pid", pid)
			continue
		}

		// Send signal 0 to check if process exists
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			m.logger.Info("process dead, cleaning up", "sandbox_id", sandboxID, "pid", pid)
			continue
		}

		// Read metadata
		meta, err := readMetadata(m.workDir, sandboxID)
		if err != nil {
			m.logger.Warn("failed to read metadata, sandbox state may be incomplete",
				"sandbox_id", sandboxID, "error", err)
		}

		info := &SandboxInfo{
			ID:         sandboxID,
			Name:       meta.Name,
			PID:        pid,
			State:      StateRunning,
			OverlayDir: filepath.Join(m.workDir, sandboxID),
			TAPDevice:  meta.TAPDevice,
			MACAddress: meta.MACAddress,
			Bridge:     meta.Bridge,
			VCPUs:      meta.VCPUs,
			MemoryMB:   meta.MemoryMB,
			IPAddress:  meta.IPAddress,
		}
		m.vms[sandboxID] = info
		m.logger.Info("recovered sandbox", "sandbox_id", sandboxID, "pid", pid)
	}

	return nil
}

// LaunchConfig contains parameters for launching a microVM.
type LaunchConfig struct {
	SandboxID    string
	Name         string
	OverlayPath  string
	KernelPath   string
	TAPDevice    string
	MACAddress   string
	Bridge       string
	VCPUs        int
	MemoryMB     int
	InitrdPath   string // optional initramfs image
	RootDevice   string // kernel root= device, defaults to /dev/vda
	CloudInitISO string // optional
	Accel        string // "kvm" (default), "hvf", or "tcg"
	// SocketVMNetClient is the path to socket_vmnet_client binary (macOS only).
	// When set, networking uses socket_vmnet instead of TAP devices.
	SocketVMNetClient string
	SocketVMNetPath   string
}

// Launch starts a QEMU microVM process with the given configuration.
func (m *Manager) Launch(ctx context.Context, cfg LaunchConfig) (*SandboxInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.vms[cfg.SandboxID]; exists {
		return nil, fmt.Errorf("sandbox %s already exists", cfg.SandboxID)
	}

	if !filepath.IsAbs(cfg.KernelPath) {
		return nil, fmt.Errorf("kernel path must be absolute: %s", cfg.KernelPath)
	}
	if !filepath.IsAbs(cfg.OverlayPath) {
		return nil, fmt.Errorf("overlay path must be absolute: %s", cfg.OverlayPath)
	}

	sandboxDir := filepath.Join(m.workDir, cfg.SandboxID)
	if err := os.MkdirAll(sandboxDir, 0o755); err != nil {
		return nil, fmt.Errorf("create sandbox dir: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(sandboxDir)
		}
	}()

	pidFile := filepath.Join(sandboxDir, "qemu.pid")
	qmpSocket := filepath.Join(sandboxDir, "qmp.sock")
	qemuStderrPath := filepath.Join(sandboxDir, "qemu-stderr.log")
	qemuEventsPath := filepath.Join(sandboxDir, "qemu-events.log")
	stderrFile, err := os.OpenFile(qemuStderrPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open qemu stderr log: %w", err)
	}
	defer func() { _ = stderrFile.Close() }()

	rootDev := cfg.RootDevice
	platform := qemuPlatformOptions(m.qemuBin)
	if rootDev == "" {
		rootDev = platform.defaultRootDevice
	}

	// Build QEMU command args
	accelArgs := resolveAccelArgs(runtime.GOOS, cfg.Accel)
	args := append([]string{"-M", platform.machineType}, accelArgs...)
	args = append(args,
		"-m", strconv.Itoa(cfg.MemoryMB),
		"-smp", strconv.Itoa(cfg.VCPUs),
		"-kernel", cfg.KernelPath,
	)
	if cfg.InitrdPath != "" {
		args = append(args, "-initrd", cfg.InitrdPath)
	}
	kernelArgs := fmt.Sprintf("console=%s root=%s rw rootwait", platform.consoleDevice, rootDev)
	if extraKernelArgs := strings.TrimSpace(os.Getenv("FLUID_QEMU_KERNEL_APPEND")); extraKernelArgs != "" {
		kernelArgs = kernelArgs + " " + extraKernelArgs
	} else {
		kernelArgs = kernelArgs + " quiet"
	}
	var netdevArg string
	if cfg.SocketVMNetClient != "" {
		netdevArg = "socket,id=net0,fd=3"
	} else {
		netdevArg = fmt.Sprintf("tap,id=net0,ifname=%s,script=no,downscript=no", cfg.TAPDevice)
	}
	args = append(args,
		"-append", kernelArgs,
		"-drive", fmt.Sprintf("id=root,file=%s,format=qcow2,if=none", cfg.OverlayPath),
		"-device", fmt.Sprintf("%s,drive=root", platform.blockDevice),
		"-netdev", netdevArg,
		"-device", fmt.Sprintf("%s,netdev=net0,mac=%s", platform.netDevice, cfg.MACAddress),
		"-serial", fmt.Sprintf("file:%s", filepath.Join(sandboxDir, "serial.log")),
		"-qmp", fmt.Sprintf("unix:%s,server=on,wait=off", qmpSocket),
		"-nographic", "-nodefaults",
		"-daemonize",
		"-pidfile", pidFile,
	)

	// Add cloud-init ISO if provided
	if cfg.CloudInitISO != "" {
		args = append(args, "-drive", fmt.Sprintf("id=cidata,file=%s,format=raw,readonly=on,if=none", cfg.CloudInitISO))
		if platform.cloudInitCtl != "" {
			args = append(args, "-device", platform.cloudInitCtl)
		}
		args = append(args, "-device", platform.cloudInitDevice)
	}

	m.logger.Info("launching microVM",
		"sandbox_id", cfg.SandboxID,
		"kernel", cfg.KernelPath,
		"overlay", cfg.OverlayPath,
		"tap", cfg.TAPDevice,
		"mac", cfg.MACAddress,
		"vcpus", cfg.VCPUs,
		"memory_mb", cfg.MemoryMB,
	)

	var cmd *exec.Cmd
	if cfg.SocketVMNetClient != "" {
		// socket_vmnet_client <socket_path> <qemu_binary> [qemu_args...]
		// It opens the vmnet socket, passes fd=3 to QEMU, then execs QEMU.
		cmdArgs := append([]string{cfg.SocketVMNetPath, m.qemuBin}, args...)
		cmd = exec.CommandContext(ctx, cfg.SocketVMNetClient, cmdArgs...)
	} else {
		cmd = exec.CommandContext(ctx, m.qemuBin, args...)
	}
	var launchOutput bytes.Buffer
	logWriter := io.MultiWriter(stderrFile, &launchOutput)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("qemu launch failed: %w: %s", err, strings.TrimSpace(launchOutput.String()))
	}

	// Read PID from pidfile (QEMU writes it after daemonizing)
	var pid int
	for i := 0; i < 10; i++ {
		pidBytes, err := os.ReadFile(pidFile)
		if err == nil {
			pid, err = strconv.Atoi(strings.TrimSpace(string(pidBytes)))
			if err == nil {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	if pid == 0 {
		return nil, fmt.Errorf("failed to read QEMU PID from %s", pidFile)
	}

	info := &SandboxInfo{
		ID:         cfg.SandboxID,
		Name:       cfg.Name,
		PID:        pid,
		State:      StateRunning,
		OverlayDir: sandboxDir,
		TAPDevice:  cfg.TAPDevice,
		MACAddress: cfg.MACAddress,
		Bridge:     cfg.Bridge,
		VCPUs:      cfg.VCPUs,
		MemoryMB:   cfg.MemoryMB,
	}

	// Persist metadata for recovery (log but don't fail - VM is already running)
	if err := writeMetadata(m.workDir, cfg.SandboxID, sandboxMetadata{
		Name:       cfg.Name,
		TAPDevice:  cfg.TAPDevice,
		MACAddress: cfg.MACAddress,
		Bridge:     cfg.Bridge,
		VCPUs:      cfg.VCPUs,
		MemoryMB:   cfg.MemoryMB,
	}); err != nil {
		m.logger.Warn("failed to write metadata", "sandbox_id", cfg.SandboxID, "error", err)
	}

	m.vms[cfg.SandboxID] = info
	watchCtx, cancel := context.WithCancel(context.Background())
	m.qmpStop[cfg.SandboxID] = cancel
	success = true
	m.logger.Info("microVM launched", "sandbox_id", cfg.SandboxID, "pid", pid)
	go m.watchQMPEvents(watchCtx, cfg.SandboxID, qmpSocket, qemuEventsPath)

	return info, nil
}

// Stop sends SIGTERM to the QEMU process for graceful shutdown.
func (m *Manager) Stop(ctx context.Context, sandboxID string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.vms[sandboxID]
	if !ok {
		return fmt.Errorf("sandbox %s not found", sandboxID)
	}
	if cancel, ok := m.qmpStop[sandboxID]; ok {
		cancel()
		delete(m.qmpStop, sandboxID)
	}

	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return fmt.Errorf("find process %d: %w", info.PID, err)
	}

	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}

	if err := proc.Signal(sig); err != nil {
		return fmt.Errorf("signal process %d: %w", info.PID, err)
	}

	// Wait briefly for the process to exit.
	done := make(chan struct{})
	go func() {
		_, _ = proc.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}

	info.State = StateStopped
	m.logger.Info("microVM stopped", "sandbox_id", sandboxID, "pid", info.PID, "force", force)
	return nil
}

// ValidateAccel checks the configured accelerator against the runtime platform
// and returns warnings if the choice is suboptimal.
func ValidateAccel(goos, accel string) []string {
	var warnings []string

	switch {
	case accel == "tcg":
		optimal := optimalAccel(goos)
		warnings = append(warnings, fmt.Sprintf("accel is set to 'tcg' (software emulation); '%s' would provide significantly better performance on %s", optimal, goos))
	case accel == "hvf" && goos != "darwin":
		warnings = append(warnings, "accel is set to 'hvf' which is only available on macOS; consider using 'kvm' on Linux for hardware acceleration")
	case accel == "kvm" && goos == "darwin":
		warnings = append(warnings, "accel is set to 'kvm' which is not available on macOS; consider using 'hvf' for hardware acceleration or remove the setting to auto-detect")
	case accel == "" && goos == "linux":
		if _, err := os.Stat("/dev/kvm"); err != nil {
			warnings = append(warnings, "KVM device /dev/kvm not found; virtualization will fall back to software emulation (tcg). Install KVM kernel modules for hardware acceleration")
		}
	case accel == "" && goos == "darwin":
		// HVF is always available on Apple Silicon and Intel Macs with Hypervisor.framework
	}

	return warnings
}

// optimalAccel returns the best accelerator for the given OS.
func optimalAccel(goos string) string {
	if goos == "darwin" {
		return "hvf"
	}
	return "kvm"
}

// resolveAccelArgs returns the QEMU accelerator flags for the given OS and accel config.
// On darwin, an empty accel defaults to HVF. On linux, an empty accel defaults to KVM.
func resolveAccelArgs(goos, accel string) []string {
	switch {
	case accel == "tcg":
		return []string{"-accel", "tcg", "-cpu", "max"}
	case accel == "hvf" || (accel == "" && goos == "darwin"):
		return []string{"-accel", "hvf", "-cpu", "max"}
	default: // "kvm" or empty on linux
		return []string{"-enable-kvm", "-cpu", "host"}
	}
}

type qemuPlatform struct {
	machineType       string
	consoleDevice     string
	blockDevice       string
	netDevice         string
	defaultRootDevice string
	cloudInitCtl      string
	cloudInitDevice   string
}

func qemuPlatformOptions(qemuBin string) qemuPlatform {
	base := strings.ToLower(filepath.Base(qemuBin))
	switch {
	case strings.Contains(base, "aarch64"), strings.Contains(base, "arm"):
		return qemuPlatform{
			machineType:       "virt",
			consoleDevice:     "ttyAMA0",
			blockDevice:       "virtio-blk-device",
			netDevice:         "virtio-net-device",
			defaultRootDevice: "/dev/vda1",
			cloudInitCtl:      "virtio-scsi-device,id=scsi0",
			cloudInitDevice:   "scsi-cd,drive=cidata,bus=scsi0.0",
		}
	default:
		return qemuPlatform{
			machineType:       "microvm",
			consoleDevice:     "ttyS0",
			blockDevice:       "virtio-blk-device",
			netDevice:         "virtio-net-device",
			defaultRootDevice: "/dev/vda",
			cloudInitDevice:   "virtio-blk-device,drive=cidata",
		}
	}
}

// Destroy stops the QEMU process and removes all associated resources.
func (m *Manager) Destroy(ctx context.Context, sandboxID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.vms[sandboxID]
	if !ok {
		// Even if not tracked, try to clean up disk
		_ = RemoveOverlay(m.workDir, sandboxID)
		return nil
	}
	if cancel, ok := m.qmpStop[sandboxID]; ok {
		cancel()
		delete(m.qmpStop, sandboxID)
	}

	// Kill the process
	proc, err := os.FindProcess(info.PID)
	if err == nil {
		_ = proc.Signal(syscall.SIGKILL)
		// Wait briefly for process to exit
		done := make(chan struct{})
		go func() {
			_, _ = proc.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			m.logger.Warn("process did not exit after SIGKILL", "sandbox_id", sandboxID, "pid", info.PID)
		}
	}

	delete(m.vms, sandboxID)
	m.logger.Info("microVM destroyed", "sandbox_id", sandboxID)

	// Overlay and TAP cleanup happens at a higher layer
	return nil
}

// Get returns info about a sandbox. The returned SandboxInfo is a copy
// and is safe to use without holding the manager lock.
func (m *Manager) Get(sandboxID string) (*SandboxInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.vms[sandboxID]
	if !ok {
		return nil, fmt.Errorf("sandbox %s not found", sandboxID)
	}

	// Check if process is still alive
	proc, err := os.FindProcess(info.PID)
	if err != nil {
		info.State = StateError
	} else if err := proc.Signal(syscall.Signal(0)); err != nil {
		info.State = StateStopped
	}

	cp := *info
	return &cp, nil
}

// List returns all tracked sandboxes. Each returned SandboxInfo is a copy
// and is safe to use without holding the manager lock.
func (m *Manager) List() []*SandboxInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*SandboxInfo, 0, len(m.vms))
	for _, info := range m.vms {
		cp := *info
		result = append(result, &cp)
	}
	return result
}

// GenerateMACAddress generates a random MAC address with QEMU/KVM prefix 52:54:00.
func GenerateMACAddress() string {
	buf := make([]byte, 3)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", buf[0], buf[1], buf[2])
}

// sandboxMetadata is persisted to disk for recovery on daemon restart.
type sandboxMetadata struct {
	Name       string `json:"name"`
	TAPDevice  string `json:"tap_device"`
	MACAddress string `json:"mac_address"`
	Bridge     string `json:"bridge"`
	VCPUs      int    `json:"vcpus"`
	MemoryMB   int    `json:"memory_mb"`
	IPAddress  string `json:"ip_address"`
}

func writeMetadata(workDir, sandboxID string, meta sandboxMetadata) error {
	path := filepath.Join(workDir, sandboxID, "metadata.json")
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

func readMetadata(workDir, sandboxID string) (sandboxMetadata, error) {
	path := filepath.Join(workDir, sandboxID, "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return sandboxMetadata{}, err
	}
	var meta sandboxMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return sandboxMetadata{}, err
	}
	return meta, nil
}

func (m *Manager) watchQMPEvents(ctx context.Context, sandboxID, socketPath, logPath string) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		m.logger.Warn("open qmp events log failed", "sandbox_id", sandboxID, "error", err)
		return
	}
	defer func() { _ = logFile.Close() }()

	writeLine := func(format string, args ...any) {
		_, _ = fmt.Fprintf(logFile, format+"\n", args...)
	}

	deadline := time.Now().Add(30 * time.Second)
	var conn net.Conn
	for {
		if ctx.Err() != nil {
			writeLine("qmp watcher canceled before connect")
			return
		}
		conn, err = net.DialTimeout("unix", socketPath, time.Second)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			writeLine("qmp watcher connect timeout: %v", err)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	defer func() { _ = conn.Close() }()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)
	if line, readErr := reader.ReadString('\n'); readErr == nil || len(line) > 0 {
		writeLine("%s", strings.TrimSpace(line))
	} else {
		writeLine("qmp watcher handshake failed: %v", readErr)
		return
	}
	if _, err := io.WriteString(conn, "{\"execute\":\"qmp_capabilities\"}\n"); err != nil {
		writeLine("qmp watcher capabilities failed: %v", err)
		return
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		writeLine("%s", scanner.Text())
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		writeLine("qmp watcher read failed: %v", err)
	}
	if ctx.Err() != nil {
		writeLine("qmp watcher stopped")
	}
}
