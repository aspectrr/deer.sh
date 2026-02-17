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
	StepCreateUser                          // Create fluid-readonly user
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

// Prepare configures a golden VM for read-only access via the fluid-readonly user.
// All steps are idempotent. The sshRun function is used to execute commands on the VM.
//
// Steps:
//  1. Create fluid-readonly user with restricted shell
//  2. Install restricted shell script
//  3. Copy CA pub key for certificate verification
//  4. Configure sshd to trust the CA key
//  5. Set up authorized principals for fluid-readonly
//  6. Restart sshd
func Prepare(ctx context.Context, sshRun SSHRunFunc, caPubKey string, onProgress ProgressFunc, logger *slog.Logger) (*PrepareResult, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(caPubKey) == "" {
		return nil, fmt.Errorf("CA public key is required")
	}

	result := &PrepareResult{}

	report := func(step PrepareStep, name string, done bool) {
		if onProgress != nil {
			onProgress(PrepareProgress{Step: step, StepName: name, Total: 6, Done: done})
		}
	}

	// Wrap sshRun to elevate all commands with sudo via base64 transport.
	//
	// Security context: Prepare runs during one-time source VM setup by a
	// trusted operator (not by AI agents). The SSH session is authenticated
	// with the operator's own credentials, not the fluid-readonly user.
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
	// go through RunWithCert which connects as the fluid-readonly user
	// directly - no sudo, no base64, no privilege escalation.
	origRun := sshRun
	sshRun = func(ctx context.Context, command string) (string, string, int, error) {
		encoded := base64.StdEncoding.EncodeToString([]byte(command))
		return origRun(ctx, fmt.Sprintf("echo %s | base64 -d | sudo bash", encoded))
	}

	// 1. Install restricted shell script at /usr/local/bin/fluid-readonly-shell
	report(StepInstallShell, "Installing restricted shell", false)
	logger.Info("installing restricted shell script")
	shellCmd := fmt.Sprintf("cat > /usr/local/bin/fluid-readonly-shell << 'FLUID_SHELL_EOF'\n%sFLUID_SHELL_EOF\nchmod 755 /usr/local/bin/fluid-readonly-shell", RestrictedShellScript)
	stdout, stderr, code, err := sshRun(ctx, shellCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("install restricted shell: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	result.ShellInstalled = true
	logger.Info("restricted shell installed")
	report(StepInstallShell, "Installing restricted shell", true)

	// 2. Create fluid-readonly user (idempotent - ignore if exists)
	report(StepCreateUser, "Creating fluid-readonly user", false)
	logger.Info("creating fluid-readonly user")
	userCmd := `mkdir -p /var/empty && id fluid-readonly >/dev/null 2>&1 || useradd -r -s /usr/local/bin/fluid-readonly-shell -d /var/empty -M fluid-readonly`
	stdout, stderr, code, err = sshRun(ctx, userCmd)
	if err != nil || code != 0 {
		return result, fmt.Errorf("create fluid-readonly user: exit=%d stdout=%q stderr=%q err=%v", code, stdout, stderr, err)
	}
	// Ensure the shell and home directory are correct even if user already existed
	modOut, modErr, modCode, modRunErr := sshRun(ctx, "usermod -s /usr/local/bin/fluid-readonly-shell -d /var/empty fluid-readonly")
	if modRunErr != nil || modCode != 0 {
		logger.Warn("usermod fixup failed (non-fatal)", "exit", modCode, "stdout", modOut, "stderr", modErr, "error", modRunErr)
	} else {
		logger.Info("usermod fixup applied (shell and home directory)")
	}
	result.UserCreated = true
	report(StepCreateUser, "Creating fluid-readonly user", true)

	// 3. Copy CA pub key to /etc/ssh/fluid_ca.pub
	report(StepInstallCAKey, "Installing CA key", false)
	logger.Info("installing CA public key")
	caCmd := fmt.Sprintf("cat > /etc/ssh/fluid_ca.pub << 'FLUID_CA_EOF'\n%s\nFLUID_CA_EOF\nchmod 644 /etc/ssh/fluid_ca.pub", strings.TrimSpace(caPubKey))
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
		`grep -q 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' /etc/ssh/sshd_config || echo 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' >> /etc/ssh/sshd_config`,
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

	// 5. Create authorized_principals directory and file for fluid-readonly
	report(StepCreatePrincipals, "Creating authorized principals", false)
	logger.Info("creating authorized principals")
	principalsCmds := []string{
		"mkdir -p /etc/ssh/authorized_principals",
		"echo 'fluid-readonly' > /etc/ssh/authorized_principals/fluid-readonly",
		"chmod 644 /etc/ssh/authorized_principals/fluid-readonly",
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
