// Package microvm manages QEMU microVM lifecycle - process launch, tracking, cleanup.
package microvm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CreateOverlay creates a QCOW2 overlay disk backed by a base image.
// The overlay is created at workDir/<sandboxID>/disk.qcow2.
func CreateOverlay(ctx context.Context, baseImagePath, workDir, sandboxID string) (string, error) {
	sandboxDir := filepath.Join(workDir, sandboxID)
	if err := os.MkdirAll(sandboxDir, 0o755); err != nil {
		return "", fmt.Errorf("create sandbox dir: %w", err)
	}

	overlayPath := filepath.Join(sandboxDir, "disk.qcow2")

	cmd := exec.CommandContext(ctx, "qemu-img", "create",
		"-f", "qcow2",
		"-b", baseImagePath,
		"-F", "qcow2",
		overlayPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("qemu-img create overlay: %w: %s", err, string(output))
	}

	return overlayPath, nil
}

// RemoveOverlay removes the sandbox directory and all its contents (overlay, PID file, etc).
func RemoveOverlay(workDir, sandboxID string) error {
	sandboxDir := filepath.Join(workDir, sandboxID)
	return os.RemoveAll(sandboxDir)
}
