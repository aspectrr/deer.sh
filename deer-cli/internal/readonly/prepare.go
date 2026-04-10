package readonly

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
)

// SSHRunFunc executes a command on a remote host via SSH.
// Returns stdout, stderr, exit code, and error.
type SSHRunFunc func(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error)

// PrepareStep identifies a step in the source VM preparation flow.
type PrepareStep int

const (
	StepInstallShell     PrepareStep = iota // Install restricted shell script
	StepCreateUser                          // Create deer-readonly user
	StepInstallCAKey                        // Copy CA pub key
	StepConfigureSSHD                       // Configure sshd to trust CA key
	StepCreatePrincipals                    // Set up authorized principals
	StepRestartSSHD                         // Restart sshd
)

// PrepareProgress reports progress during source VM preparation.
type PrepareProgress struct {
	Step     PrepareStep
	StepName string
	Total    int  // always 6
	Done     bool // false=starting, true=completed
}

// ProgressFunc is called before and after each preparation step.
// If nil, no progress is reported.
type ProgressFunc func(PrepareProgress)

// PrepareResult contains the outcome of preparing a golden VM for read-only access.
type PrepareResult struct {
	UserCreated       bool
	ShellInstalled    bool
	CAKeyInstalled    bool
	SSHDConfigured    bool
	PrincipalsCreated bool
	SSHDRestarted     bool
}

// PrepareWithKeyResult contains the outcome of key-based preparation.
type PrepareWithKeyResult struct {
	ShellInstalled bool
	UserCreated    bool
	KeyDeployed    bool
}

// PrepareWithKey configures a host for read-only access using an SSH public key
// instead of a CA certificate. This is simpler (4 steps vs 6) and works directly
// from the user's laptop without needing a daemon.
//
// Steps:
//  1. Install restricted shell script
//  2. Create deer-readonly user with restricted shell
//  3. Deploy public key to authorized_keys
//  4. Restart sshd
func PrepareWithKey(ctx context.Context, sshRun SSHRunFunc, pubKey string, onProgress ProgressFunc, logger *slog.Logger) (*PrepareWithKeyResult, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(pubKey) == "" {
		return nil, fmt.Errorf("public key is required")
	}

	totalSteps := 4
	result := &PrepareWithKeyResult{}

	report := func(step PrepareStep, name string, done bool) {
		if onProgress != nil {
			onProgress(PrepareProgress{Step: step, StepName: name, Total: totalSteps, Done: done})
		}
	}

	// Wrap sshRun to elevate all commands with sudo via base64 transport.
	origRun := sshRun
	sshRun = func(ctx context.Context, command string) (string, string, int, error) {
		encoded := base64.StdEncoding.EncodeToString([]byte(command))
		return origRun(ctx, fmt.Sprintf("echo %s | base64 -d | sudo bash", encoded))
	}

	// 1. Install restricted shell script
	report(StepInstallShell, "Installing restricted shell", false)
	logger.Info("installing restricted shell script")
	shellCmd := fmt.Sprintf("cat > /usr/local/bin/deer-readonly-shell << 'DEER_SHELL_EOF'\n%sDEER_SHELL_EOF\nchmod 755 /usr/local/bin/deer-readonly-shell", RestrictedShellScript)
	stdout, stderr, code, err := sshRun(ctx, shellCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("install restricted shell: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	result.ShellInstalled = true
	report(StepInstallShell, "Installing restricted shell", true)

	// 2. Create deer-readonly user (idempotent)
	report(StepCreateUser, "Creating deer-readonly user", false)
	logger.Info("creating deer-readonly user")
	userCmd := `id deer-readonly >/dev/null 2>&1 || useradd -r -s /usr/local/bin/deer-readonly-shell -m deer-readonly`
	stdout, stderr, code, err = sshRun(ctx, userCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("create deer-readonly user: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	// Ensure shell is correct even if user already existed
	sshRun(ctx, "usermod -s /usr/local/bin/deer-readonly-shell deer-readonly") //nolint:errcheck
	// systemd-journal grants journal read access; adm omitted as overly broad
	sshRun(ctx, "usermod -a -G systemd-journal deer-readonly 2>/dev/null || true") //nolint:errcheck
	result.UserCreated = true
	report(StepCreateUser, "Creating deer-readonly user", true)

	// 3. Deploy public key to authorized_keys
	report(StepInstallCAKey, "Deploying SSH public key", false)
	logger.Info("deploying SSH public key")
	keyCmd := fmt.Sprintf("mkdir -p /home/deer-readonly/.ssh && chmod 700 /home/deer-readonly/.ssh && cat > /home/deer-readonly/.ssh/authorized_keys << 'DEER_KEY_EOF'\n%s\nDEER_KEY_EOF\nchmod 600 /home/deer-readonly/.ssh/authorized_keys && chown -R deer-readonly:deer-readonly /home/deer-readonly/.ssh", strings.TrimSpace(pubKey))
	stdout, stderr, code, err = sshRun(ctx, keyCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("deploy public key: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	result.KeyDeployed = true
	report(StepInstallCAKey, "Deploying SSH public key", true)

	// 4. Restart sshd
	report(StepRestartSSHD, "Restarting sshd", false)
	logger.Info("restarting sshd")
	restartCmd := `systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || service sshd restart 2>/dev/null || service ssh restart`
	stdout, stderr, code, err = sshRun(ctx, restartCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("restart sshd: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	report(StepRestartSSHD, "Restarting sshd", true)

	return result, nil
}

// SetupSourceHost creates the deer-daemon user (if missing), adds it to the
// libvirt group, and deploys the daemon's SSH identity key. This is the full
// setup needed for the daemon to reach a source host via qemu+ssh.
// All steps are idempotent. Requires sudo on the target host.
func SetupSourceHost(ctx context.Context, sshRun SSHRunFunc, identityPubKey string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	key := strings.TrimSpace(identityPubKey)
	if key == "" {
		return fmt.Errorf("daemon identity pub key is empty")
	}

	// Create deer-daemon user with libvirt access, deploy key, and grant
	// passwordless sudo for qemu-img so the daemon can read QEMU-owned snapshot
	// files (libvirt-qemu:kvm 0600) when pulling disk images.
	cmd := fmt.Sprintf(
		"id deer-daemon >/dev/null 2>&1 || useradd --system --shell /bin/bash -m deer-daemon && "+
			"usermod -aG libvirt deer-daemon 2>/dev/null || true && "+
			"mkdir -p ~deer-daemon/.ssh && chmod 700 ~deer-daemon/.ssh && "+
			"grep -qF '%s' ~deer-daemon/.ssh/authorized_keys 2>/dev/null || echo '%s' >> ~deer-daemon/.ssh/authorized_keys && "+
			"chmod 600 ~deer-daemon/.ssh/authorized_keys && chown -R deer-daemon:deer-daemon ~deer-daemon/.ssh && "+
			"QEMU_IMG=$(command -v qemu-img); "+
			"echo \"deer-daemon ALL=(root) NOPASSWD: ${QEMU_IMG}\" > /etc/sudoers.d/deer-daemon-qemuimg && "+
			"chmod 440 /etc/sudoers.d/deer-daemon-qemuimg",
		key, key,
	)
	encoded := base64.StdEncoding.EncodeToString([]byte(cmd))
	wrapped := fmt.Sprintf("echo %s | base64 -d | sudo bash", encoded)

	stdout, stderr, code, err := sshRun(ctx, wrapped)
	if err != nil {
		return fmt.Errorf("setup source host: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("setup source host: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	logger.Info("source host setup complete: deer-daemon user + key deployed")
	return nil
}

// DeployDaemonKey deploys the daemon's SSH identity pub key to the deer-daemon
// user's authorized_keys on a source host, allowing the daemon to SSH in for
// virsh/rsync operations (via qemu+ssh://deer-daemon@host/system).
// The deer-daemon user must already exist on the target host.
// The command is idempotent.
func DeployDaemonKey(ctx context.Context, sshRun SSHRunFunc, identityPubKey string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	key := strings.TrimSpace(identityPubKey)
	if key == "" {
		return fmt.Errorf("daemon identity pub key is empty")
	}

	cmd := fmt.Sprintf(
		"mkdir -p ~deer-daemon/.ssh && chmod 700 ~deer-daemon/.ssh && "+
			"grep -qF '%s' ~deer-daemon/.ssh/authorized_keys 2>/dev/null || echo '%s' >> ~deer-daemon/.ssh/authorized_keys && "+
			"chmod 600 ~deer-daemon/.ssh/authorized_keys && chown -R deer-daemon:deer-daemon ~deer-daemon/.ssh",
		key, key,
	)
	encoded := base64.StdEncoding.EncodeToString([]byte(cmd))
	wrapped := fmt.Sprintf("echo %s | base64 -d | sudo bash", encoded)

	stdout, stderr, code, err := sshRun(ctx, wrapped)
	if err != nil {
		return fmt.Errorf("deploy daemon key: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("deploy daemon key: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	logger.Info("daemon identity key deployed to deer-daemon user")
	return nil
}

// Prepare configures a golden VM for read-only access via the deer-readonly user.
// All steps are idempotent. The sshRun function is used to execute commands on the VM.
//
// Steps:
//  1. Create deer-readonly user with restricted shell
//  2. Install restricted shell script
//  3. Copy CA pub key for certificate verification
//  4. Configure sshd to trust the CA key
//  5. Set up authorized principals for deer-readonly
//  6. Restart sshd
func Prepare(ctx context.Context, sshRun SSHRunFunc, caPubKey string, onProgress ProgressFunc, logger *slog.Logger) (*PrepareResult, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(caPubKey) == "" {
		return nil, fmt.Errorf("CA public key is required")
	}

	totalSteps := 6

	result := &PrepareResult{}

	report := func(step PrepareStep, name string, done bool) {
		if onProgress != nil {
			onProgress(PrepareProgress{Step: step, StepName: name, Total: totalSteps, Done: done})
		}
	}

	// Wrap sshRun to elevate all commands with sudo via base64 transport.
	//
	// Security context: Prepare runs during one-time source VM setup by a
	// trusted operator (not by AI agents). The SSH session is authenticated
	// with the operator's own credentials, not the deer-readonly user.
	//
	// Why base64: preparation commands contain heredocs, single quotes,
	// double quotes, and newlines (e.g. writing the restricted shell script).
	// Passing these through SSH + sudo introduces nested quoting that is
	// fragile and error-prone. Base64 encoding the command on the Go side
	// and decoding on the VM side avoids all shell interpolation issues.
	//
	// The pattern is: echo <base64> | base64 -d | sudo bash
	//   - echo: emits the opaque base64 blob (no special chars to escape)
	//   - base64 -d: decodes to the original command string
	//   - sudo bash: executes with root privileges
	//
	// This wrapper is NOT used at runtime for agent commands. Agent commands
	// go through RunWithCert which connects as the deer-readonly user
	// directly - no sudo, no base64, no privilege escalation.
	origRun := sshRun
	sshRun = func(ctx context.Context, command string) (string, string, int, error) {
		encoded := base64.StdEncoding.EncodeToString([]byte(command))
		return origRun(ctx, fmt.Sprintf("echo %s | base64 -d | sudo bash", encoded))
	}

	// 1. Install restricted shell script at /usr/local/bin/deer-readonly-shell
	report(StepInstallShell, "Installing restricted shell", false)
	logger.Info("installing restricted shell script")
	shellCmd := fmt.Sprintf("cat > /usr/local/bin/deer-readonly-shell << 'DEER_SHELL_EOF'\n%sDEER_SHELL_EOF\nchmod 755 /usr/local/bin/deer-readonly-shell", RestrictedShellScript)
	stdout, stderr, code, err := sshRun(ctx, shellCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("install restricted shell: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	result.ShellInstalled = true
	logger.Info("restricted shell installed")
	report(StepInstallShell, "Installing restricted shell", true)

	// 2. Create deer-readonly user (idempotent - ignore if exists)
	report(StepCreateUser, "Creating deer-readonly user", false)
	logger.Info("creating deer-readonly user")
	userCmd := `mkdir -p /var/empty && id deer-readonly >/dev/null 2>&1 || useradd -r -s /usr/local/bin/deer-readonly-shell -d /var/empty -M deer-readonly`
	stdout, stderr, code, err = sshRun(ctx, userCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("create deer-readonly user: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	// Ensure the shell and home directory are correct even if user already existed
	modOut, modErr, modCode, modRunErr := sshRun(ctx, "usermod -s /usr/local/bin/deer-readonly-shell -d /var/empty deer-readonly")
	if modRunErr != nil || modCode != 0 {
		logger.Warn("usermod fixup failed (non-fatal)", "exit", modCode, "stdout", modOut, "stderr", modErr, "error", modRunErr)
	} else {
		logger.Info("usermod fixup applied (shell and home directory)")
	}
	// systemd-journal grants journal read access; adm omitted as overly broad
	sshRun(ctx, "usermod -a -G systemd-journal deer-readonly 2>/dev/null || true") //nolint:errcheck
	result.UserCreated = true
	report(StepCreateUser, "Creating deer-readonly user", true)

	// 3. Copy CA pub key to /etc/ssh/deer_ca.pub
	report(StepInstallCAKey, "Installing CA key", false)
	logger.Info("installing CA public key")
	caCmd := fmt.Sprintf("cat > /etc/ssh/deer_ca.pub << 'DEER_CA_EOF'\n%s\nDEER_CA_EOF\nchmod 644 /etc/ssh/deer_ca.pub", strings.TrimSpace(caPubKey))
	stdout, stderr, code, err = sshRun(ctx, caCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("install CA pub key: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	result.CAKeyInstalled = true
	logger.Info("CA public key installed")
	report(StepInstallCAKey, "Installing CA key", true)

	// 4. Configure sshd to trust the CA key (idempotent)
	report(StepConfigureSSHD, "Configuring sshd", false)
	logger.Info("configuring sshd")
	sshdCmds := []string{
		// Add TrustedUserCAKeys if not present
		`grep -q 'TrustedUserCAKeys /etc/ssh/deer_ca.pub' /etc/ssh/sshd_config || echo 'TrustedUserCAKeys /etc/ssh/deer_ca.pub' >> /etc/ssh/sshd_config`,
		// Add AuthorizedPrincipalsFile if not present
		`grep -q 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' /etc/ssh/sshd_config || echo 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' >> /etc/ssh/sshd_config`,
	}
	for _, cmd := range sshdCmds {
		stdout, stderr, code, err = sshRun(ctx, cmd)
		if err != nil || code != 0 {
			return result, fmt.Errorf("configure sshd: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
		}
	}
	result.SSHDConfigured = true
	logger.Info("sshd configured")
	report(StepConfigureSSHD, "Configuring sshd", true)

	// 5. Create authorized_principals directory and file for deer-readonly
	report(StepCreatePrincipals, "Creating authorized principals", false)
	logger.Info("creating authorized principals")
	principalsCmds := []string{
		"mkdir -p /etc/ssh/authorized_principals",
		"echo 'deer-readonly' > /etc/ssh/authorized_principals/deer-readonly",
		"chmod 644 /etc/ssh/authorized_principals/deer-readonly",
	}
	for _, cmd := range principalsCmds {
		stdout, stderr, code, err = sshRun(ctx, cmd)
		if err != nil || code != 0 {
			return result, fmt.Errorf("create principals: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
		}
	}
	result.PrincipalsCreated = true
	logger.Info("authorized principals created")
	report(StepCreatePrincipals, "Creating authorized principals", true)

	// 6. Restart sshd to apply changes
	// Try systemctl first, fall back to service command
	report(StepRestartSSHD, "Restarting sshd", false)
	logger.Info("restarting sshd")
	restartCmd := `systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || service sshd restart 2>/dev/null || service ssh restart`
	stdout, stderr, code, err = sshRun(ctx, restartCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("restart sshd: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	result.SSHDRestarted = true
	logger.Info("sshd restarted")
	report(StepRestartSSHD, "Restarting sshd", true)

	return result, nil
}
