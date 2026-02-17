package microvm

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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
}

// Manager manages QEMU microVM processes.
type Manager struct {
	mu      sync.RWMutex
	vms     map[string]*SandboxInfo // sandbox_id -> info
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
		qemuBin: bin,
		workDir: workDir,
		logger:  logger.With("component", "microvm"),
	}

	return m, nil
}

// WorkDir returns the working directory for sandbox data.
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
		meta, _ := readMetadata(m.workDir, sandboxID)

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
	CloudInitISO string // optional
}

// Launch starts a QEMU microVM process with the given configuration.
func (m *Manager) Launch(ctx context.Context, cfg LaunchConfig) (*SandboxInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.vms[cfg.SandboxID]; exists {
		return nil, fmt.Errorf("sandbox %s already exists", cfg.SandboxID)
	}

	sandboxDir := filepath.Join(m.workDir, cfg.SandboxID)
	if err := os.MkdirAll(sandboxDir, 0o755); err != nil {
		return nil, fmt.Errorf("create sandbox dir: %w", err)
	}

	pidFile := filepath.Join(sandboxDir, "qemu.pid")

	// Build QEMU command args
	args := []string{
		"-M", "microvm", "-enable-kvm", "-cpu", "host",
		"-m", strconv.Itoa(cfg.MemoryMB),
		"-smp", strconv.Itoa(cfg.VCPUs),
		"-kernel", cfg.KernelPath,
		"-append", "console=ttyS0 root=/dev/vda rw quiet",
		"-drive", fmt.Sprintf("id=root,file=%s,format=qcow2,if=none", cfg.OverlayPath),
		"-device", "virtio-blk-device,drive=root",
		"-netdev", fmt.Sprintf("tap,id=net0,ifname=%s,script=no,downscript=no", cfg.TAPDevice),
		"-device", fmt.Sprintf("virtio-net-device,netdev=net0,mac=%s", cfg.MACAddress),
		"-serial", "stdio",
		"-nographic", "-nodefaults",
		"-daemonize",
		"-pidfile", pidFile,
	}

	// Add cloud-init ISO if provided
	if cfg.CloudInitISO != "" {
		args = append(args,
			"-drive", fmt.Sprintf("id=cidata,file=%s,format=raw,if=none", cfg.CloudInitISO),
			"-device", "virtio-blk-device,drive=cidata",
		)
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

	cmd := exec.CommandContext(ctx, m.qemuBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("qemu launch failed: %w: %s", err, string(output))
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

	// Persist metadata for recovery
	writeMetadata(m.workDir, cfg.SandboxID, sandboxMetadata{
		Name:       cfg.Name,
		TAPDevice:  cfg.TAPDevice,
		MACAddress: cfg.MACAddress,
		Bridge:     cfg.Bridge,
		VCPUs:      cfg.VCPUs,
		MemoryMB:   cfg.MemoryMB,
	})

	m.vms[cfg.SandboxID] = info
	m.logger.Info("microVM launched", "sandbox_id", cfg.SandboxID, "pid", pid)

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

	info.State = StateStopped
	m.logger.Info("microVM stopped", "sandbox_id", sandboxID, "pid", info.PID, "force", force)
	return nil
}

// Destroy stops the QEMU process and removes all associated resources.
func (m *Manager) Destroy(ctx context.Context, sandboxID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.vms[sandboxID]
	if !ok {
		// Even if not tracked, try to clean up disk
		RemoveOverlay(m.workDir, sandboxID)
		return nil
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

// Get returns info about a sandbox.
func (m *Manager) Get(sandboxID string) (*SandboxInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.vms[sandboxID]
	if !ok {
		return nil, fmt.Errorf("sandbox %s not found", sandboxID)
	}

	// Check if process is still alive
	proc, err := os.FindProcess(info.PID)
	if err != nil {
		info.State = StateError
		return info, nil
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		info.State = StateStopped
	}

	return info, nil
}

// List returns all tracked sandboxes.
func (m *Manager) List() []*SandboxInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*SandboxInfo, 0, len(m.vms))
	for _, info := range m.vms {
		result = append(result, info)
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
}

func writeMetadata(workDir, sandboxID string, meta sandboxMetadata) {
	path := filepath.Join(workDir, sandboxID, "metadata.json")
	data := fmt.Sprintf(`{"name":%q,"tap_device":%q,"mac_address":%q,"bridge":%q,"vcpus":%d,"memory_mb":%d}`,
		meta.Name, meta.TAPDevice, meta.MACAddress, meta.Bridge, meta.VCPUs, meta.MemoryMB)
	_ = os.WriteFile(path, []byte(data), 0o644)
}

func readMetadata(workDir, sandboxID string) (sandboxMetadata, error) {
	path := filepath.Join(workDir, sandboxID, "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return sandboxMetadata{}, err
	}

	// Simple JSON parse - avoid importing encoding/json just for this
	var meta sandboxMetadata
	s := string(data)

	meta.Name = extractJSONString(s, "name")
	meta.TAPDevice = extractJSONString(s, "tap_device")
	meta.MACAddress = extractJSONString(s, "mac_address")
	meta.Bridge = extractJSONString(s, "bridge")

	if v := extractJSONString(s, "vcpus"); v != "" {
		meta.VCPUs, _ = strconv.Atoi(v)
	}
	if v := extractJSONString(s, "memory_mb"); v != "" {
		meta.MemoryMB, _ = strconv.Atoi(v)
	}

	return meta, nil
}

func extractJSONString(s, key string) string {
	needle := fmt.Sprintf(`"%s":`, key)
	idx := strings.Index(s, needle)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(needle):]
	rest = strings.TrimSpace(rest)

	if len(rest) == 0 {
		return ""
	}

	if rest[0] == '"' {
		// String value
		end := strings.Index(rest[1:], `"`)
		if end < 0 {
			return ""
		}
		return rest[1 : end+1]
	}

	// Numeric value
	end := strings.IndexAny(rest, ",}")
	if end < 0 {
		return rest
	}
	return rest[:end]
}
