package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aspectrr/fluid.sh/fluid/internal/workflow"
)

// ContainerRunner handles Podman container creation and execution.
type ContainerRunner struct {
	// podmanPath is the path to the podman binary.
	podmanPath string
}

// ContainerRunnerConfig configures the container runner.
type ContainerRunnerConfig struct {
	// PodmanPath is the path to the podman binary.
	// If empty, "podman" is looked up in PATH.
	PodmanPath string
}

// NewContainerRunner creates a new ContainerRunner with the given configuration.
func NewContainerRunner(cfg ContainerRunnerConfig) *ContainerRunner {
	podmanPath := cfg.PodmanPath
	if podmanPath == "" {
		podmanPath = "podman"
	}
	return &ContainerRunner{
		podmanPath: podmanPath,
	}
}

// ContainerResult contains the result of creating a container.
type ContainerResult struct {
	// ContainerID is the full container ID.
	ContainerID string

	// ShortID is the short (12 character) container ID.
	ShortID string

	// ContainerName is the deterministic container name.
	ContainerName string

	// Cleanup is a function to stop and remove the container.
	Cleanup workflow.CleanupFunc
}

// ResourceLimits specifies resource constraints for the container.
type ResourceLimits struct {
	// CPUQuota is the CPU quota in microseconds per period (100000 = 1 CPU).
	// 0 means no limit.
	CPUQuota int64

	// MemoryLimit is the memory limit in bytes.
	// 0 means no limit.
	MemoryLimit int64

	// MemorySwap is the memory+swap limit in bytes.
	// -1 means unlimited swap, 0 means same as MemoryLimit.
	MemorySwap int64

	// PidsLimit is the maximum number of PIDs.
	// 0 means no limit.
	PidsLimit int64
}

// DefaultResourceLimits returns sensible default resource limits.
func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		CPUQuota:    200000,                 // 2 CPUs
		MemoryLimit: 2 * 1024 * 1024 * 1024, // 2 GB
		MemorySwap:  -1,                     // Unlimited swap
		PidsLimit:   1024,                   // 1024 processes
	}
}

// RunContainer creates and starts a container from the given image.
// The container uses a deterministic name based on the VM name.
func (r *ContainerRunner) RunContainer(ctx context.Context, imageTag string, vmName string, limits ResourceLimits) (*ContainerResult, error) {
	// Generate deterministic container name
	containerName := generateContainerName(vmName)

	// Check if container already exists
	exists, err := r.ContainerExists(ctx, containerName)
	if err != nil {
		return nil, workflow.NewWorkflowError(
			workflow.StageRunContainer,
			workflow.ErrContainerCreateFailed,
			fmt.Sprintf("failed to check container existence: %v", err),
		)
	}

	// If container exists, remove it for idempotency
	if exists {
		if err := r.RemoveContainer(ctx, containerName, true); err != nil {
			return nil, workflow.NewWorkflowError(
				workflow.StageRunContainer,
				workflow.ErrContainerCreateFailed,
				fmt.Sprintf("failed to remove existing container: %v", err),
			)
		}
	}

	// Build container create arguments
	args := []string{
		"run",
		"--detach",
		"--name", containerName,
		"--hostname", vmName,
		"--env", "container=podman",
		"--tty",
		"--interactive",
	}

	// Apply resource limits
	if limits.CPUQuota > 0 {
		args = append(args, "--cpu-quota", fmt.Sprintf("%d", limits.CPUQuota))
	}
	if limits.MemoryLimit > 0 {
		args = append(args, "--memory", fmt.Sprintf("%d", limits.MemoryLimit))
	}
	if limits.MemorySwap != 0 {
		args = append(args, "--memory-swap", fmt.Sprintf("%d", limits.MemorySwap))
	}
	if limits.PidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", limits.PidsLimit))
	}

	// Add the image and default command
	args = append(args, imageTag, "/bin/sh")

	// Create and start the container
	cmd := exec.CommandContext(ctx, r.podmanPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, workflow.NewWorkflowError(
			workflow.StageRunContainer,
			workflow.ErrContainerCreateFailed,
			fmt.Sprintf("podman run failed: %v: %s", err, stderr.String()),
		)
	}

	// Get the container ID
	containerID := strings.TrimSpace(stdout.String())
	if containerID == "" {
		// If we didn't get an ID from stdout, inspect the container
		containerID, err = r.getContainerID(ctx, containerName)
		if err != nil {
			return nil, workflow.NewWorkflowError(
				workflow.StageRunContainer,
				workflow.ErrContainerCreateFailed,
				fmt.Sprintf("failed to get container ID: %v", err),
			)
		}
	}

	// Generate short ID (first 12 characters)
	shortID := containerID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}

	return &ContainerResult{
		ContainerID:   containerID,
		ShortID:       shortID,
		ContainerName: containerName,
		Cleanup: func() error {
			return r.RemoveContainer(context.Background(), containerName, true)
		},
	}, nil
}

// generateContainerName creates a deterministic container name from the VM name.
func generateContainerName(vmName string) string {
	// Use a prefix to identify VM clone containers
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	return fmt.Sprintf("vmclone-%s-%s", vmName, timestamp)
}

// getContainerID retrieves the container ID for a given name.
func (r *ContainerRunner) getContainerID(ctx context.Context, containerName string) (string, error) {
	args := []string{
		"inspect",
		"--format", "{{.Id}}",
		containerName,
	}

	cmd := exec.CommandContext(ctx, r.podmanPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("inspect failed: %w: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// ContainerExists checks if a container with the given name exists.
func (r *ContainerRunner) ContainerExists(ctx context.Context, containerName string) (bool, error) {
	args := []string{
		"container",
		"exists",
		containerName,
	}

	cmd := exec.CommandContext(ctx, r.podmanPath, args...)
	err := cmd.Run()

	if err == nil {
		return true, nil
	}

	// Exit code 1 means container doesn't exist
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
	}

	return false, fmt.Errorf("failed to check container existence: %w", err)
}

// RemoveContainer stops and removes a container.
func (r *ContainerRunner) RemoveContainer(ctx context.Context, containerRef string, force bool) error {
	args := []string{"rm"}

	if force {
		args = append(args, "--force")
	}

	args = append(args, containerRef)

	cmd := exec.CommandContext(ctx, r.podmanPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container removal failed: %w: %s", err, stderr.String())
	}

	return nil
}

// StopContainer stops a running container.
func (r *ContainerRunner) StopContainer(ctx context.Context, containerRef string, timeout int) error {
	args := []string{
		"stop",
		"--time", fmt.Sprintf("%d", timeout),
		containerRef,
	}

	cmd := exec.CommandContext(ctx, r.podmanPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container stop failed: %w: %s", err, stderr.String())
	}

	return nil
}

// StartContainer starts a stopped container.
func (r *ContainerRunner) StartContainer(ctx context.Context, containerRef string) error {
	args := []string{"start", containerRef}

	cmd := exec.CommandContext(ctx, r.podmanPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container start failed: %w: %s", err, stderr.String())
	}

	return nil
}

// ContainerInfo contains detailed information about a container.
type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	Status  string
	Running bool
	Created time.Time
}

// InspectContainer retrieves detailed information about a container.
func (r *ContainerRunner) InspectContainer(ctx context.Context, containerRef string) (*ContainerInfo, error) {
	args := []string{
		"inspect",
		"--format", "json",
		containerRef,
	}

	cmd := exec.CommandContext(ctx, r.podmanPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("inspect failed: %w: %s", err, stderr.String())
	}

	// Parse JSON output
	var containers []struct {
		ID      string `json:"Id"`
		Name    string `json:"Name"`
		Image   string `json:"Image"`
		Created string `json:"Created"`
		State   struct {
			Status  string `json:"Status"`
			Running bool   `json:"Running"`
		} `json:"State"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &containers); err != nil {
		return nil, fmt.Errorf("failed to parse inspect output: %w", err)
	}

	if len(containers) == 0 {
		return nil, fmt.Errorf("no container found")
	}

	c := containers[0]
	created, _ := time.Parse(time.RFC3339Nano, c.Created)

	return &ContainerInfo{
		ID:      c.ID,
		Name:    c.Name,
		Image:   c.Image,
		Status:  c.State.Status,
		Running: c.State.Running,
		Created: created,
	}, nil
}

// ExecInContainer executes a command inside a running container.
func (r *ContainerRunner) ExecInContainer(ctx context.Context, containerRef string, command []string) (string, string, int, error) {
	args := []string{"exec", containerRef}
	args = append(args, command...)

	cmd := exec.CommandContext(ctx, r.podmanPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", "", -1, fmt.Errorf("exec failed: %w", err)
		}
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

// ListContainers lists containers matching a filter pattern.
func (r *ContainerRunner) ListContainers(ctx context.Context, all bool, filter string) ([]ContainerInfo, error) {
	args := []string{
		"ps",
		"--format", "json",
	}

	if all {
		args = append(args, "--all")
	}

	if filter != "" {
		args = append(args, "--filter", filter)
	}

	cmd := exec.CommandContext(ctx, r.podmanPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("list containers failed: %w: %s", err, stderr.String())
	}

	// Parse JSON output
	var containers []struct {
		ID      string   `json:"Id"`
		Names   []string `json:"Names"`
		Image   string   `json:"Image"`
		Created int64    `json:"Created"`
		State   string   `json:"State"`
	}

	output := stdout.Bytes()
	if len(output) == 0 {
		return []ContainerInfo{}, nil
	}

	if err := json.Unmarshal(output, &containers); err != nil {
		return nil, fmt.Errorf("failed to parse list output: %w", err)
	}

	result := make([]ContainerInfo, len(containers))
	for i, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		result[i] = ContainerInfo{
			ID:      c.ID,
			Name:    name,
			Image:   c.Image,
			Status:  c.State,
			Running: c.State == "running",
			Created: time.Unix(c.Created, 0),
		}
	}

	return result, nil
}

// WaitForContainer waits for a container to be in a ready state.
func (r *ContainerRunner) WaitForContainer(ctx context.Context, containerRef string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		info, err := r.InspectContainer(ctx, containerRef)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if info.Running {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for container to be ready")
}
