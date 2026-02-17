// Package sourcevm manages source VM operations: listing, validation,
// preparation for read-only access, and running read-only commands.
package sourcevm

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/readonly"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/sshkeys"
)

// VMInfo describes a source VM visible via libvirt.
type VMInfo struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	IPAddress string `json:"ip_address,omitempty"`
	Prepared  bool   `json:"prepared"`
}

// ValidationResult contains the result of validating a source VM.
type ValidationResult struct {
	VMName     string   `json:"vm_name"`
	Valid      bool     `json:"valid"`
	State      string   `json:"state"`
	MACAddress string   `json:"mac_address,omitempty"`
	IPAddress  string   `json:"ip_address,omitempty"`
	HasNetwork bool     `json:"has_network"`
	Warnings   []string `json:"warnings,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

// PrepareResult contains the outcome of preparing a source VM.
type PrepareResult struct {
	SourceVM          string `json:"source_vm"`
	IPAddress         string `json:"ip_address,omitempty"`
	Prepared          bool   `json:"prepared"`
	UserCreated       bool   `json:"user_created"`
	ShellInstalled    bool   `json:"shell_installed"`
	CAKeyInstalled    bool   `json:"ca_key_installed"`
	SSHDConfigured    bool   `json:"sshd_configured"`
	PrincipalsCreated bool   `json:"principals_created"`
	SSHDRestarted     bool   `json:"sshd_restarted"`
}

// Manager handles source VM operations.
type Manager struct {
	libvirtURI string
	network    string
	keyMgr     sshkeys.KeyProvider
	sshUser    string
	proxyJump  string
	logger     *slog.Logger
}

// NewManager creates a source VM manager.
func NewManager(libvirtURI, network string, keyMgr sshkeys.KeyProvider, sshUser, proxyJump string, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	if sshUser == "" {
		sshUser = "sandbox"
	}
	return &Manager{
		libvirtURI: libvirtURI,
		network:    network,
		keyMgr:     keyMgr,
		sshUser:    sshUser,
		proxyJump:  proxyJump,
		logger:     logger.With("component", "sourcevm"),
	}
}

// ListVMs returns available source VMs (non-sandbox VMs visible to libvirt).
func (m *Manager) ListVMs(ctx context.Context) ([]VMInfo, error) {
	// Use virsh to list all VMs
	output, err := m.virsh(ctx, "list", "--all", "--name")
	if err != nil {
		return nil, fmt.Errorf("virsh list: %w", err)
	}

	var vms []VMInfo
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.HasPrefix(name, "sbx-") {
			continue // Skip sandbox VMs
		}

		state, _ := m.getVMState(ctx, name)
		ip := ""
		if state == "running" {
			ip, _ = m.getVMIP(ctx, name)
		}

		vms = append(vms, VMInfo{
			Name:      name,
			State:     state,
			IPAddress: ip,
		})
	}

	return vms, nil
}

// ValidateSourceVM checks if a source VM is ready for read-only access.
func (m *Manager) ValidateSourceVM(ctx context.Context, vmName string) (*ValidationResult, error) {
	result := &ValidationResult{
		VMName: vmName,
	}

	// Check VM exists and get state
	state, err := m.getVMState(ctx, vmName)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("VM not found: %v", err))
		return result, nil
	}
	result.State = state

	if state != "running" {
		result.Errors = append(result.Errors, "VM is not running")
		return result, nil
	}

	// Check network/MAC
	mac, err := m.getVMMAC(ctx, vmName)
	if err == nil && mac != "" {
		result.MACAddress = mac
		result.HasNetwork = true
	} else {
		result.Warnings = append(result.Warnings, "Could not determine MAC address")
	}

	// Check IP
	ip, err := m.getVMIP(ctx, vmName)
	if err == nil && ip != "" {
		result.IPAddress = ip
	} else {
		result.Warnings = append(result.Warnings, "Could not determine IP address")
	}

	// Check if fluid-readonly user exists by trying SSH
	if ip != "" && m.keyMgr != nil {
		creds, err := m.keyMgr.GetSourceVMCredentials(ctx, vmName)
		if err == nil {
			_, _, exitCode, err := m.sshCmd(ctx, ip, "fluid-readonly", creds, "whoami", 10*time.Second)
			if err == nil && exitCode == 0 {
				result.Valid = true
			} else {
				result.Warnings = append(result.Warnings, "SSH as fluid-readonly failed - VM may not be prepared")
			}
		} else {
			result.Warnings = append(result.Warnings, "Could not get SSH credentials")
		}
	}

	if len(result.Errors) == 0 && result.HasNetwork && result.IPAddress != "" {
		result.Valid = true
	}

	return result, nil
}

// PrepareSourceVM installs readonly shell, fluid-readonly user, SSH CA on a source VM.
func (m *Manager) PrepareSourceVM(ctx context.Context, vmName, sshUser, sshKeyPath string) (*PrepareResult, error) {
	if sshUser == "" {
		sshUser = m.sshUser
	}

	ip, err := m.getVMIP(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("get VM IP: %w", err)
	}

	// Build SSH run function for the prepare flow
	sshRun := func(ctx context.Context, command string) (string, string, int, error) {
		return m.sshCmdWithKey(ctx, ip, sshUser, sshKeyPath, command, 60*time.Second)
	}

	// Get CA public key
	var caPubKey string
	if m.keyMgr != nil {
		// The key manager's CA should have the public key
		// For now, we'll read it from the sshca package via the key path config
		// This will be wired properly through the CA instance
		caPubKey = "" // Will be set by caller
	}

	if caPubKey == "" {
		return nil, fmt.Errorf("CA public key is required for source VM preparation")
	}

	result, err := readonly.Prepare(ctx, sshRun, caPubKey, nil, m.logger)
	if err != nil {
		return nil, err
	}

	return &PrepareResult{
		SourceVM:          vmName,
		IPAddress:         ip,
		Prepared:          true,
		UserCreated:       result.UserCreated,
		ShellInstalled:    result.ShellInstalled,
		CAKeyInstalled:    result.CAKeyInstalled,
		SSHDConfigured:    result.SSHDConfigured,
		PrincipalsCreated: result.PrincipalsCreated,
		SSHDRestarted:     result.SSHDRestarted,
	}, nil
}

// PrepareSourceVMWithCA prepares a source VM with an explicit CA public key.
func (m *Manager) PrepareSourceVMWithCA(ctx context.Context, vmName, sshUser, sshKeyPath, caPubKey string) (*PrepareResult, error) {
	if sshUser == "" {
		sshUser = m.sshUser
	}

	ip, err := m.getVMIP(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("get VM IP: %w", err)
	}

	sshRun := func(ctx context.Context, command string) (string, string, int, error) {
		return m.sshCmdWithKey(ctx, ip, sshUser, sshKeyPath, command, 60*time.Second)
	}

	result, err := readonly.Prepare(ctx, sshRun, caPubKey, nil, m.logger)
	if err != nil {
		return nil, err
	}

	return &PrepareResult{
		SourceVM:          vmName,
		IPAddress:         ip,
		Prepared:          true,
		UserCreated:       result.UserCreated,
		ShellInstalled:    result.ShellInstalled,
		CAKeyInstalled:    result.CAKeyInstalled,
		SSHDConfigured:    result.SSHDConfigured,
		PrincipalsCreated: result.PrincipalsCreated,
		SSHDRestarted:     result.SSHDRestarted,
	}, nil
}

// RunSourceCommand executes a read-only command on a source VM.
// Two-layer validation: client-side allowlist + server-side restricted shell.
func (m *Manager) RunSourceCommand(ctx context.Context, vmName, command string, timeout time.Duration) (stdout, stderr string, exitCode int, err error) {
	// Client-side validation
	if err := readonly.ValidateCommand(command); err != nil {
		return "", "", 126, fmt.Errorf("command validation: %w", err)
	}

	ip, err := m.getVMIP(ctx, vmName)
	if err != nil {
		return "", "", -1, fmt.Errorf("get VM IP: %w", err)
	}

	if m.keyMgr == nil {
		return "", "", -1, fmt.Errorf("SSH key manager required for source VM commands")
	}

	creds, err := m.keyMgr.GetSourceVMCredentials(ctx, vmName)
	if err != nil {
		return "", "", -1, fmt.Errorf("get credentials: %w", err)
	}

	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return m.sshCmd(ctx, ip, "fluid-readonly", creds, command, timeout)
}

// ReadSourceFile reads a file from a source VM via base64-encoded transfer.
func (m *Manager) ReadSourceFile(ctx context.Context, vmName, path string) (string, error) {
	// Use base64 encoding for safe binary transfer
	command := fmt.Sprintf("base64 %s", path)

	stdout, stderr, exitCode, err := m.RunSourceCommand(ctx, vmName, command, 30*time.Second)
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", fmt.Errorf("read file failed (exit %d): %s", exitCode, stderr)
	}

	return stdout, nil
}

// --- Internal helpers ---

func (m *Manager) virsh(ctx context.Context, args ...string) (string, error) {
	allArgs := append([]string{"-c", m.libvirtURI}, args...)
	cmd := exec.CommandContext(ctx, "virsh", allArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", stderr.String(), err)
	}
	return stdout.String(), nil
}

func (m *Manager) getVMState(ctx context.Context, vmName string) (string, error) {
	output, err := m.virsh(ctx, "domstate", vmName)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (m *Manager) getVMMAC(ctx context.Context, vmName string) (string, error) {
	output, err := m.virsh(ctx, "domiflist", vmName)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[0] != "Interface" && !strings.HasPrefix(line, "---") {
			return fields[4], nil
		}
	}
	return "", fmt.Errorf("no MAC address found")
}

func (m *Manager) getVMIP(ctx context.Context, vmName string) (string, error) {
	output, err := m.virsh(ctx, "domifaddr", vmName, "--source", "lease")
	if err != nil {
		// Try agent source
		output, err = m.virsh(ctx, "domifaddr", vmName, "--source", "agent")
		if err != nil {
			return "", err
		}
	}

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			ipCIDR := fields[3]
			ip := strings.SplitN(ipCIDR, "/", 2)[0]
			if ip != "" && !strings.HasPrefix(ip, "127.") {
				return ip, nil
			}
		}
	}
	return "", fmt.Errorf("no IP address found")
}

func (m *Manager) sshCmd(ctx context.Context, ip, user string, creds *sshkeys.Credentials, command string, timeout time.Duration) (string, string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{
		"-i", creds.PrivateKeyPath,
		"-o", "CertificateFile=" + creds.CertificatePath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", fmt.Sprintf("ConnectTimeout=%d", int(timeout.Seconds())),
	}

	if m.proxyJump != "" {
		args = append(args, "-J", m.proxyJump)
	}

	args = append(args, fmt.Sprintf("%s@%s", user, ip), command)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return stdout.String(), stderr.String(), -1, err
		}
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

func (m *Manager) sshCmdWithKey(ctx context.Context, ip, user, keyPath, command string, timeout time.Duration) (string, string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", fmt.Sprintf("ConnectTimeout=%d", int(timeout.Seconds())),
	}

	if m.proxyJump != "" {
		args = append(args, "-J", m.proxyJump)
	}

	args = append(args, fmt.Sprintf("%s@%s", user, ip), command)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return stdout.String(), stderr.String(), -1, err
		}
	}

	return stdout.String(), stderr.String(), exitCode, nil
}
