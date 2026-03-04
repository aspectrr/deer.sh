package hostexec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// RunFunc executes a command on a host and returns stdout, stderr, exit code, and error.
type RunFunc func(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error)

// NewLocal returns a RunFunc that executes commands locally via bash.
// Security: the command string is passed directly to "bash -c". Callers
// must only pass trusted or pre-validated input to avoid shell injection.
func NewLocal() RunFunc {
	return func(ctx context.Context, command string) (string, string, int, error) {
		cmd := exec.CommandContext(ctx, "bash", "-c", command)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return stdout.String(), stderr.String(), 1, err
			}
		}
		return stdout.String(), stderr.String(), exitCode, nil
	}
}

// NewSSH returns a RunFunc that executes commands on a remote host via SSH.
func NewSSH(addr, user string, port int) RunFunc {
	return func(ctx context.Context, command string) (string, string, int, error) {
		args := []string{
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "ConnectTimeout=15",
			"-o", "BatchMode=yes",
		}
		if port != 0 && port != 22 {
			args = append(args, "-p", fmt.Sprintf("%d", port))
		}
		args = append(args, fmt.Sprintf("%s@%s", user, addr), "--", command)

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
				return stdout.String(), stderr.String(), 1, err
			}
		}
		return stdout.String(), stderr.String(), exitCode, nil
	}
}

// SSHOption modifies the SSH argument list before execution.
type SSHOption func(args []string) []string

// WithRelaxedHostKeys sets StrictHostKeyChecking=no and UserKnownHostsFile=/dev/null.
// Use for ephemeral VMs whose IPs and host keys change frequently.
func WithRelaxedHostKeys() SSHOption {
	return func(args []string) []string {
		var result []string
		replaced := false
		for i := 0; i < len(args); i++ {
			if args[i] == "-o" && i+1 < len(args) && strings.HasPrefix(args[i+1], "StrictHostKeyChecking=") {
				result = append(result, "-o", "StrictHostKeyChecking=no")
				i++ // skip the original value
				replaced = true
				continue
			}
			result = append(result, args[i])
		}
		if !replaced {
			result = append(result, "-o", "StrictHostKeyChecking=no")
		}
		return append(result, "-o", "UserKnownHostsFile=/dev/null")
	}
}

// NewSSHWithJump returns a RunFunc that executes commands on a remote host via SSH
// with a proxy jump through an intermediate host.
// jumpHost format: "user@host" or "user@host:port"
func NewSSHWithJump(addr, user string, port int, jumpHost string, opts ...SSHOption) RunFunc {
	return func(ctx context.Context, command string) (string, string, int, error) {
		args := []string{
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "ConnectTimeout=15",
			"-o", "BatchMode=yes",
			"-J", jumpHost,
		}
		if port != 0 && port != 22 {
			args = append(args, "-p", fmt.Sprintf("%d", port))
		}
		for _, opt := range opts {
			args = opt(args)
		}

		args = append(args, fmt.Sprintf("%s@%s", user, addr), "--", command)

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
				return stdout.String(), stderr.String(), 1, err
			}
		}
		return stdout.String(), stderr.String(), exitCode, nil
	}
}

// NewSSHWithKey returns a RunFunc that executes commands via SSH using a specific private key.
func NewSSHWithKey(addr, user string, port int, keyPath string) RunFunc {
	return func(ctx context.Context, command string) (string, string, int, error) {
		args := []string{
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "ConnectTimeout=15",
			"-o", "BatchMode=yes",
			"-o", "IdentitiesOnly=yes",
			"-i", keyPath,
		}
		if port != 0 && port != 22 {
			args = append(args, "-p", fmt.Sprintf("%d", port))
		}
		args = append(args, fmt.Sprintf("%s@%s", user, addr), "--", command)

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
				return stdout.String(), stderr.String(), 1, err
			}
		}
		return stdout.String(), stderr.String(), exitCode, nil
	}
}

// NewReadOnlySSH returns a RunFunc that connects as fluid-readonly using a specific key.
func NewReadOnlySSH(addr string, port int, keyPath string) RunFunc {
	return NewSSHWithKey(addr, "fluid-readonly", port, keyPath)
}

// NewSSHAlias returns a RunFunc that executes commands via SSH using the original
// host alias. This lets SSH apply the user's full ~/.ssh/config (ProxyJump,
// IdentityFile, etc.) rather than bypassing it with a raw resolved hostname.
// Extra SSH flags can be prepended via extraArgs (e.g. -l user, -i key).
func NewSSHAlias(hostAlias string, extraArgs ...string) RunFunc {
	return func(ctx context.Context, command string) (string, string, int, error) {
		args := []string{
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "ConnectTimeout=15",
			"-o", "BatchMode=yes",
		}
		args = append(args, extraArgs...)
		args = append(args, hostAlias, "--", command)

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
				return stdout.String(), stderr.String(), 1, err
			}
		}
		return stdout.String(), stderr.String(), exitCode, nil
	}
}

// NewReadOnlySSHAlias returns a RunFunc that connects as fluid-readonly using
// the host alias (preserving ~/.ssh/config) plus the fluid key.
func NewReadOnlySSHAlias(hostAlias, keyPath string) RunFunc {
	return NewSSHAlias(hostAlias,
		"-l", "fluid-readonly",
		"-o", "IdentitiesOnly=yes",
		"-i", keyPath,
	)
}

// OutputCallback receives a chunk of streaming command output.
type OutputCallback func(chunk string, isStderr bool)

// RunStreamingSSHAlias runs a command via SSH using the host alias, streaming
// stdout/stderr line-by-line through the callback as they arrive.
// Returns the full buffered stdout, stderr, exit code, and error.
func RunStreamingSSHAlias(ctx context.Context, hostAlias string, extraArgs []string, command string, onOutput OutputCallback) (stdout, stderr string, exitCode int, err error) {
	args := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=15",
		"-o", "BatchMode=yes",
	}
	args = append(args, extraArgs...)
	args = append(args, hostAlias, "--", command)

	cmd := exec.CommandContext(ctx, "ssh", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", 1, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", "", 1, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", "", 1, fmt.Errorf("start: %w", err)
	}

	var stdoutBuf, stderrBuf strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			stdoutBuf.WriteString(line)
			if onOutput != nil {
				onOutput(line, false)
			}
		}
		if err := scanner.Err(); err != nil {
			if onOutput != nil {
				onOutput(fmt.Sprintf("[fluid] stdout scanner error: %v\n", err), true)
			}
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			stderrBuf.WriteString(line)
			if onOutput != nil {
				onOutput(line, true)
			}
		}
		if err := scanner.Err(); err != nil {
			if onOutput != nil {
				onOutput(fmt.Sprintf("[fluid] stderr scanner error: %v\n", err), true)
			}
		}
	}()

	wg.Wait()
	waitErr := cmd.Wait()

	exitCode = 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return stdoutBuf.String(), stderrBuf.String(), 1, waitErr
		}
	}
	return stdoutBuf.String(), stderrBuf.String(), exitCode, nil
}

// WithSudo wraps a RunFunc to execute commands with sudo via base64 encoding.
// This avoids shell quoting issues with complex commands.
// Security: callers must ensure commands have been validated (e.g. via
// readonly.ValidateCommand) before reaching this path.
func WithSudo(run RunFunc) RunFunc {
	return func(ctx context.Context, command string) (string, string, int, error) {
		encoded := base64.StdEncoding.EncodeToString([]byte(command))
		return run(ctx, fmt.Sprintf("echo %s | base64 -d | sudo bash", encoded))
	}
}
