// Package network manages TAP devices and bridge networking for microVM sandboxes.
package network

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// CreateTAP creates a TAP device and attaches it to a bridge.
// TAP names use format "fluid-<shortID>" where shortID is the first 6 chars of sandbox ID.
func CreateTAP(ctx context.Context, tapName, bridge string, logger *slog.Logger) error {
	// 1. Create TAP device
	if err := runCmd(ctx, "ip", "tuntap", "add", "dev", tapName, "mode", "tap"); err != nil {
		return fmt.Errorf("create tap %s: %w", tapName, err)
	}

	// 2. Attach to bridge
	if err := runCmd(ctx, "ip", "link", "set", tapName, "master", bridge); err != nil {
		// Cleanup on failure
		_ = DestroyTAP(ctx, tapName)
		return fmt.Errorf("attach tap %s to bridge %s: %w", tapName, bridge, err)
	}

	// 3. Bring up
	if err := runCmd(ctx, "ip", "link", "set", tapName, "up"); err != nil {
		_ = DestroyTAP(ctx, tapName)
		return fmt.Errorf("bring up tap %s: %w", tapName, err)
	}

	if logger != nil {
		logger.Info("TAP created", "tap", tapName, "bridge", bridge)
	}
	return nil
}

// DestroyTAP removes a TAP device.
func DestroyTAP(ctx context.Context, tapName string) error {
	return runCmd(ctx, "ip", "link", "delete", tapName)
}

// TAPName generates a TAP device name from a sandbox ID.
// Uses the first 9 characters of the sandbox ID (after any prefix).
// Stays within Linux 15-char interface name limit: "fl-" + 9 = 12.
func TAPName(sandboxID string) string {
	id := strings.TrimPrefix(sandboxID, "SBX-")
	if len(id) > 9 {
		id = id[:9]
	}
	return "fl-" + strings.ToLower(id)
}

func runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
