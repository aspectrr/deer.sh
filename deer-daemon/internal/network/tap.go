// Package network manages TAP devices and bridge networking for microVM sandboxes.
package network

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
)

var (
	runCmdFunc       = runCmd
	runOutputCmdFunc = runOutputCmd
	runtimeGOOS      = runtime.GOOS
)

// CreateTAP creates a TAP device and attaches it to a bridge.
// On Darwin, TAP devices are created dynamically and the requested name is ignored.
func CreateTAP(ctx context.Context, tapName, bridge string, logger *slog.Logger) (string, error) {
	actualTap, err := createTAPForOS(ctx, tapName, bridge)
	if err != nil {
		return "", err
	}

	if logger != nil {
		logger.Info("TAP created", "tap", actualTap, "bridge", bridge)
	}
	return actualTap, nil
}

// DestroyTAP removes a TAP device.
func DestroyTAP(ctx context.Context, tapName string) error {
	switch runtimeGOOS {
	case "darwin":
		return runCmdFunc(ctx, "ifconfig", tapName, "destroy")
	default:
		return runCmdFunc(ctx, "ip", "link", "delete", tapName)
	}
}

// TAPName generates a TAP device name from a sandbox ID.
// Linux interface names are limited to 15 characters, so we keep the suffix
// within 12 characters after the "fl-" prefix. We prefer the trailing portion
// of the ID because sandbox IDs usually share a common prefix and vary at the end.
func TAPName(sandboxID string) string {
	id := strings.TrimPrefix(sandboxID, "SBX-")
	id = strings.TrimPrefix(id, "sbx-")
	if len(id) > 12 {
		id = id[len(id)-12:]
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

func runOutputCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func createTAPForOS(ctx context.Context, tapName, bridge string) (string, error) {
	switch runtimeGOOS {
	case "darwin":
		return createTAPDarwin(ctx, bridge)
	default:
		return createTAPLinux(ctx, tapName, bridge)
	}
}

func createTAPLinux(ctx context.Context, tapName, bridge string) (string, error) {
	if err := runCmdFunc(ctx, "ip", "tuntap", "add", "dev", tapName, "mode", "tap"); err != nil {
		return "", fmt.Errorf("create tap %s: %w", tapName, err)
	}
	if err := runCmdFunc(ctx, "ip", "link", "set", tapName, "master", bridge); err != nil {
		_ = DestroyTAP(ctx, tapName)
		return "", fmt.Errorf("attach tap %s to bridge %s: %w", tapName, bridge, err)
	}
	if err := runCmdFunc(ctx, "ip", "link", "set", tapName, "promisc", "on"); err != nil {
		_ = DestroyTAP(ctx, tapName)
		return "", fmt.Errorf("enable promisc on tap %s: %w", tapName, err)
	}
	if err := runCmdFunc(ctx, "ip", "link", "set", tapName, "up"); err != nil {
		_ = DestroyTAP(ctx, tapName)
		return "", fmt.Errorf("bring up tap %s: %w", tapName, err)
	}
	return tapName, nil
}

func createTAPDarwin(ctx context.Context, bridge string) (string, error) {
	output, err := runOutputCmdFunc(ctx, "ifconfig", "tap", "create")
	if err != nil {
		return "", fmt.Errorf("create darwin tap: %w", err)
	}
	tapName := parseTapCreateOutput(output)
	if tapName == "" {
		return "", fmt.Errorf("create darwin tap: unexpected output %q", strings.TrimSpace(output))
	}
	if err := runCmdFunc(ctx, "ifconfig", bridge, "addm", tapName); err != nil {
		_ = DestroyTAP(ctx, tapName)
		return "", fmt.Errorf("attach tap %s to bridge %s: %w", tapName, bridge, err)
	}
	if err := runCmdFunc(ctx, "ifconfig", tapName, "up"); err != nil {
		_ = DestroyTAP(ctx, tapName)
		return "", fmt.Errorf("bring up tap %s: %w", tapName, err)
	}
	return tapName, nil
}

func parseTapCreateOutput(output string) string {
	for _, field := range strings.Fields(strings.TrimSpace(output)) {
		if strings.HasPrefix(field, "tap") {
			return field
		}
	}
	return ""
}
