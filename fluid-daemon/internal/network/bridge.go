package network

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// NetworkManager handles bridge resolution and TAP management.
type NetworkManager struct {
	defaultBridge string
	bridgeMap     map[string]string // libvirt network name -> local bridge name
	dhcpMode      string
	logger        *slog.Logger
}

// NewNetworkManager creates a network manager with the given configuration.
func NewNetworkManager(defaultBridge string, bridgeMap map[string]string, dhcpMode string, logger *slog.Logger) *NetworkManager {
	if logger == nil {
		logger = slog.Default()
	}
	if bridgeMap == nil {
		bridgeMap = make(map[string]string)
	}
	return &NetworkManager{
		defaultBridge: defaultBridge,
		bridgeMap:     bridgeMap,
		dhcpMode:      dhcpMode,
		logger:        logger.With("component", "network"),
	}
}

// ResolveBridge determines which bridge to attach a sandbox's TAP to.
// Priority: explicit request > source VM's network > default bridge.
func (n *NetworkManager) ResolveBridge(ctx context.Context, sourceVMName, requestedNetwork string) (string, error) {
	// 1. If explicit network requested, look up in bridge_map
	if requestedNetwork != "" {
		if bridge, ok := n.bridgeMap[requestedNetwork]; ok {
			n.logger.Info("resolved bridge from requested network", "network", requestedNetwork, "bridge", bridge)
			return bridge, nil
		}
		// If the requested network looks like a bridge name (not a libvirt network), use it directly
		if strings.HasPrefix(requestedNetwork, "br") || strings.HasPrefix(requestedNetwork, "virbr") {
			return requestedNetwork, nil
		}
		return "", fmt.Errorf("unknown network %q: not found in bridge_map", requestedNetwork)
	}

	// 2. If source VM specified, query libvirt for its network
	if sourceVMName != "" {
		bridge, err := n.resolveFromSourceVM(ctx, sourceVMName)
		if err == nil && bridge != "" {
			n.logger.Info("resolved bridge from source VM", "source_vm", sourceVMName, "bridge", bridge)
			return bridge, nil
		}
		if err != nil {
			n.logger.Warn("failed to resolve bridge from source VM, using default",
				"source_vm", sourceVMName, "error", err)
		}
	}

	// 3. Fall back to default bridge
	n.logger.Info("using default bridge", "bridge", n.defaultBridge)
	return n.defaultBridge, nil
}

// resolveFromSourceVM queries virsh to determine which bridge a source VM is connected to.
func (n *NetworkManager) resolveFromSourceVM(ctx context.Context, sourceVMName string) (string, error) {
	// virsh domiflist <source-vm> returns network/bridge info
	cmd := exec.CommandContext(ctx, "virsh", "domiflist", sourceVMName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("virsh domiflist: %w", err)
	}

	// Parse output to find network name or bridge
	// Format: Interface  Type  Source  Model  MAC
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "Interface") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		ifType := fields[1]
		source := fields[2]

		// If type is "bridge", source is the bridge name directly
		if ifType == "bridge" {
			return source, nil
		}

		// If type is "network", source is a libvirt network name
		if ifType == "network" {
			// Check bridge_map first
			if bridge, ok := n.bridgeMap[source]; ok {
				return bridge, nil
			}

			// Resolve via virsh net-info
			return n.resolveNetworkToBridge(ctx, source)
		}
	}

	return "", fmt.Errorf("no network interface found for VM %s", sourceVMName)
}

// resolveNetworkToBridge uses virsh net-info to find the bridge for a libvirt network.
func (n *NetworkManager) resolveNetworkToBridge(ctx context.Context, networkName string) (string, error) {
	cmd := exec.CommandContext(ctx, "virsh", "net-info", networkName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("virsh net-info %s: %w", networkName, err)
	}

	// Parse "Bridge: virbr0" from output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Bridge:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("no bridge found for network %s", networkName)
}

// DHCPMode returns the configured DHCP mode.
func (n *NetworkManager) DHCPMode() string {
	return n.dhcpMode
}
