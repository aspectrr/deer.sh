package source

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
	"github.com/aspectrr/deer.sh/deer-cli/internal/hostexec"
	"github.com/aspectrr/deer.sh/deer-cli/internal/readonly"
)

// CommandResult holds the result of running a command on a source host.
type CommandResult struct {
	Host     string `json:"host"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// HostInfo describes a configured source host.
type HostInfo struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Prepared bool   `json:"prepared"`
}

// Provider is the interface the agent uses to interact with source hosts.
// *Service satisfies this interface.
type Provider interface {
	RunCommandStreaming(ctx context.Context, hostName, command string, onOutput hostexec.OutputCallback) (*CommandResult, error)
	RunCommandElevated(ctx context.Context, hostName, command string) (*CommandResult, error)
	ReadFile(ctx context.Context, hostName, path string) (string, error)
	ListHosts() []HostInfo
}

// Service provides direct SSH access to source hosts for read-only operations.
type Service struct {
	cfg     *config.Config
	keyPath string
	logger  *slog.Logger
}

// NewService creates a source service that uses direct SSH with the given key.
func NewService(cfg *config.Config, keyPath string, logger *slog.Logger) *Service {
	return &Service{
		cfg:     cfg,
		keyPath: keyPath,
		logger:  logger,
	}
}

// RunCommand executes a read-only command on a source host via direct SSH.
// Uses the host name as an SSH alias so ~/.ssh/config is applied (ProxyJump, etc.).
func (s *Service) RunCommand(ctx context.Context, hostName, command string) (*CommandResult, error) {
	host, err := s.findHost(hostName)
	if err != nil {
		return nil, err
	}
	if !host.Prepared {
		return nil, fmt.Errorf("host %q is not prepared - run: deer source prepare %s", hostName, hostName)
	}

	if err := readonly.ValidateCommandWithExtra(command, s.cfg.ExtraAllowedCommands); err != nil {
		return nil, fmt.Errorf("command not allowed: %w (use request_source_access to ask the human for approval if this command is needed for diagnosis)", err)
	}

	// Use host name as SSH alias to preserve ~/.ssh/config (ProxyJump, etc.)
	run := hostexec.NewReadOnlySSHAlias(hostName, s.keyPath)
	stdout, stderr, exitCode, err := run(ctx, command)
	if err != nil {
		return &CommandResult{
			Host:     hostName,
			ExitCode: exitCode,
			Stdout:   stdout,
			Stderr:   stderr,
		}, fmt.Errorf("ssh command failed: %w", err)
	}

	return &CommandResult{
		Host:     hostName,
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
	}, nil
}

// RunCommandStreaming executes a read-only command with live output streaming.
// Functionally identical to RunCommand but calls onOutput for each line as it arrives.
func (s *Service) RunCommandStreaming(ctx context.Context, hostName, command string, onOutput hostexec.OutputCallback) (*CommandResult, error) {
	host, err := s.findHost(hostName)
	if err != nil {
		return nil, err
	}
	if !host.Prepared {
		return nil, fmt.Errorf("host %q is not prepared - run: deer source prepare %s", hostName, hostName)
	}

	if err := readonly.ValidateCommandWithExtra(command, s.cfg.ExtraAllowedCommands); err != nil {
		return nil, fmt.Errorf("command not allowed: %w (use request_source_access to ask the human for approval if this command is needed for diagnosis)", err)
	}

	extraArgs := []string{
		"-l", "deer-readonly",
		"-o", "IdentitiesOnly=yes",
		"-i", s.keyPath,
	}
	stdout, stderr, exitCode, err := hostexec.RunStreamingSSHAlias(ctx, hostName, extraArgs, command, onOutput)
	if err != nil {
		return &CommandResult{
			Host:     hostName,
			ExitCode: exitCode,
			Stdout:   stdout,
			Stderr:   stderr,
		}, fmt.Errorf("ssh command failed: %w", err)
	}

	return &CommandResult{
		Host:     hostName,
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
	}, nil
}

// shellQuote wraps a string in POSIX single quotes, escaping any embedded
// single quotes with the '\” idiom.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// ReadFile reads a file from a source host via direct SSH.
func (s *Service) ReadFile(ctx context.Context, hostName, path string) (string, error) {
	if !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("path must be absolute")
	}

	result, err := s.RunCommand(ctx, hostName, fmt.Sprintf("cat -- %s", shellQuote(path)))
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("read file failed: exit=%d stderr=%s", result.ExitCode, result.Stderr)
	}
	return result.Stdout, nil
}

// RunCommandElevated executes a command on a source host after human approval.
// It bypasses the read-only command validation - the human has explicitly approved this.
func (s *Service) RunCommandElevated(ctx context.Context, hostName, command string) (*CommandResult, error) {
	host, err := s.findHost(hostName)
	if err != nil {
		return nil, err
	}
	if !host.Prepared {
		return nil, fmt.Errorf("host %q is not prepared - run: deer source prepare %s", hostName, hostName)
	}

	extraArgs := []string{
		"-l", "deer-readonly",
		"-o", "IdentitiesOnly=yes",
		"-i", s.keyPath,
	}
	stdout, stderr, exitCode, err := hostexec.RunStreamingSSHAlias(ctx, hostName, extraArgs, command, nil)
	if err != nil {
		return &CommandResult{
			Host:     hostName,
			ExitCode: exitCode,
			Stdout:   stdout,
			Stderr:   stderr,
		}, fmt.Errorf("ssh command failed: %w", err)
	}

	return &CommandResult{
		Host:     hostName,
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
	}, nil
}

// ListHosts returns all configured source hosts.
func (s *Service) ListHosts() []HostInfo {
	hosts := make([]HostInfo, 0, len(s.cfg.Hosts))
	for _, h := range s.cfg.Hosts {
		hosts = append(hosts, HostInfo{
			Name:     h.Name,
			Address:  h.Address,
			Prepared: h.Prepared,
		})
	}
	return hosts
}

func (s *Service) findHost(name string) (*config.HostConfig, error) {
	for i := range s.cfg.Hosts {
		if s.cfg.Hosts[i].Name == name {
			return &s.cfg.Hosts[i], nil
		}
	}
	return nil, fmt.Errorf("host %q not found in config - run: deer source prepare %s", name, name)
}
