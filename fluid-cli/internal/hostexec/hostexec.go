package hostexec

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
)

// RunFunc executes a command on a host and returns stdout, stderr, exit code, and error.
type RunFunc func(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error)

// NewLocal returns a RunFunc that executes commands locally via bash.
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

// NewSSHWithJump returns a RunFunc that executes commands on a remote host via SSH
// with a proxy jump through an intermediate host.
// jumpHost format: "user@host" or "user@host:port"
func NewSSHWithJump(addr, user string, port int, jumpHost string) RunFunc {
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

// WithSudo wraps a RunFunc to execute commands with sudo via base64 encoding.
// This avoids shell quoting issues with complex commands.
func WithSudo(run RunFunc) RunFunc {
	return func(ctx context.Context, command string) (string, string, int, error) {
		encoded := base64.StdEncoding.EncodeToString([]byte(command))
		return run(ctx, fmt.Sprintf("echo %s | base64 -d | sudo bash", encoded))
	}
}
