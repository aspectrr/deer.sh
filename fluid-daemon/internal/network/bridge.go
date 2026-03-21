package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strings"
)

var validBridge = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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
// Priority: explicit request > default bridge.
func (n *NetworkManager) ResolveBridge(ctx context.Context, requestedNetwork string) (string, error) {
	var bridge string

	// 1. If explicit network requested, look up in bridge_map
	if requestedNetwork != "" {
		if b, ok := n.bridgeMap[requestedNetwork]; ok {
			n.logger.Info("resolved bridge from requested network", "network", requestedNetwork, "bridge", b)
			bridge = b
		} else if strings.HasPrefix(requestedNetwork, "br") || strings.HasPrefix(requestedNetwork, "virbr") {
			bridge = requestedNetwork
		} else {
			return "", fmt.Errorf("unknown network %q: not found in bridge_map", requestedNetwork)
		}
	}

	// 2. Fall back to default bridge
	if bridge == "" {
		bridge = n.defaultBridge
		n.logger.Info("using default bridge", "bridge", bridge)
	}

	if !validBridge.MatchString(bridge) {
		return "", fmt.Errorf("invalid bridge name %q: must match [a-zA-Z0-9_-]+", bridge)
	}

	return bridge, nil
}

// DHCPMode returns the configured DHCP mode.
func (n *NetworkManager) DHCPMode() string {
	return n.dhcpMode
}

// GetBridgeIP returns the first IPv4 address assigned to the named bridge interface.
func GetBridgeIP(bridge string) (string, error) {
	iface, err := net.InterfaceByName(bridge)
	if err != nil {
		return "", fmt.Errorf("interface %s: %w", bridge, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("addrs for %s: %w", bridge, err)
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.To4() != nil {
			return ipNet.IP.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 address on bridge %s", bridge)
}
