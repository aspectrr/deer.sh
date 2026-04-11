package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DiscoverIP discovers the IP address assigned to a MAC address on a given bridge.
// It uses the configured DHCP mode to determine the discovery strategy.
func (n *NetworkManager) DiscoverIP(ctx context.Context, macAddress, bridge string, timeout time.Duration) (string, error) {
	switch n.dhcpMode {
	case "libvirt":
		return discoverIPLibvirt(ctx, macAddress, bridge, timeout, n.logger)
	case "arp":
		return discoverIPARP(ctx, macAddress, bridge, timeout, n.logger)
	case "dnsmasq":
		return discoverIPDnsmasq(ctx, macAddress, bridge, timeout, n.logger)
	default:
		return discoverIPARP(ctx, macAddress, bridge, timeout, n.logger)
	}
}

// discoverIPLibvirt reads libvirt dnsmasq lease files to find IP for a MAC.
func discoverIPLibvirt(ctx context.Context, macAddress, bridge string, timeout time.Duration, logger *slog.Logger) (string, error) {
	mac := strings.ToLower(macAddress)
	deadline := time.Now().Add(timeout)

	// Sanitize bridge name to prevent path traversal.
	safeBridge := filepath.Base(bridge)

	// Try common lease file locations
	leaseFiles := []string{
		"/var/lib/libvirt/dnsmasq/default.leases",
		"/var/lib/libvirt/dnsmasq/virbr0.leases",
		fmt.Sprintf("/var/lib/libvirt/dnsmasq/%s.leases", safeBridge),
	}
	statusFiles := []string{
		"/var/lib/libvirt/dnsmasq/default.status",
		"/var/lib/libvirt/dnsmasq/virbr0.status",
		fmt.Sprintf("/var/lib/libvirt/dnsmasq/%s.status", safeBridge),
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		for _, statusFile := range statusFiles {
			ip, err := readLibvirtStatusIP(statusFile, mac)
			if err == nil && ip != "" {
				logger.Info("discovered IP via libvirt status", "mac", macAddress, "ip", ip)
				return ip, nil
			}
		}

		for _, leaseFile := range leaseFiles {
			data, err := os.ReadFile(leaseFile)
			if err != nil {
				continue
			}

			// Lease file format: timestamp MAC IP hostname client-id
			for _, line := range strings.Split(string(data), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 3 && strings.EqualFold(fields[1], mac) {
					logger.Info("discovered IP via libvirt lease", "mac", macAddress, "ip", fields[2])
					return fields[2], nil
				}
			}
		}

		if err := contextSleep(ctx, 2*time.Second); err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf("IP discovery timed out for MAC %s (libvirt mode)", macAddress)
}

type libvirtStatusLease struct {
	IPAddress  string `json:"ip-address"`
	MACAddress string `json:"mac-address"`
}

func readLibvirtStatusIP(path, mac string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var leases []libvirtStatusLease
	if err := json.Unmarshal(data, &leases); err != nil {
		return "", err
	}
	for _, lease := range leases {
		if strings.EqualFold(lease.MACAddress, mac) {
			return lease.IPAddress, nil
		}
	}
	return "", nil
}

// discoverIPARP polls the ARP table to find IP for a MAC.
func discoverIPARP(ctx context.Context, macAddress, bridge string, timeout time.Duration, logger *slog.Logger) (string, error) {
	mac := strings.ToLower(macAddress)
	// Normalize MAC to colon-free lowercase for comparison.
	// macOS ARP collapses leading zeros: 52:54:00:4f:fb:3c -> 52:54:0:4f:fb:3c
	macNormalized := strings.ReplaceAll(mac, ":", "")
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Try ip neigh first (Linux)
		cmd := exec.CommandContext(ctx, "ip", "neigh", "show", "dev", bridge)
		output, err := cmd.Output()
		if err == nil {
			// Format: IP lladdr MAC STATE
			for _, line := range strings.Split(string(output), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				fields := strings.Fields(line)
				// Look for the MAC address in the line
				for i, f := range fields {
					if strings.EqualFold(f, mac) && i > 0 {
						ip := fields[0]
						logger.Info("discovered IP via ip neigh", "mac", macAddress, "ip", ip)
						return ip, nil
					}
				}
			}
		}

		// Fallback: arp -an (works on macOS and Linux)
		cmd = exec.CommandContext(ctx, "arp", "-an")
		output, err = cmd.Output()
		if err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				lineLower := strings.ToLower(line)
				if !strings.Contains(lineLower, mac) && !strings.Contains(normalizeARPMac(lineLower), macNormalized) {
					continue
				}
				// Format: ? (IP) at MAC [ether] on interface
				start := strings.Index(line, "(")
				end := strings.Index(line, ")")
				if start >= 0 && end > start {
					ip := line[start+1 : end]
					logger.Info("discovered IP via arp -an", "mac", macAddress, "ip", ip)
					return ip, nil
				}
			}
		}

		if err := contextSleep(ctx, 2*time.Second); err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf("IP discovery timed out for MAC %s (arp mode)", macAddress)
}

// normalizeARPMac extracts the MAC from an arp line and returns it
// colon-free, lowercased, and zero-padded for comparison.
// macOS ARP drops leading zeros in octets: 52:54:00:0c:09 → 52:54:0:c:9
func normalizeARPMac(line string) string {
	// macOS format: ? (192.168.105.61) at 52:54:0:4f:fb:3c on bridge100
	afterAt := ""
	if idx := strings.Index(line, " at "); idx >= 0 {
		afterAt = line[idx+4:]
	}
	if afterAt == "" {
		return ""
	}
	fields := strings.Fields(afterAt)
	if len(fields) == 0 {
		return ""
	}
	rawMAC := fields[0]
	parts := strings.Split(strings.ToLower(rawMAC), ":")
	for i, p := range parts {
		if len(p) == 1 {
			parts[i] = "0" + p
		}
	}
	return strings.Join(parts, "")
}

// discoverIPDnsmasq reads local dnsmasq lease file for IP discovery.
func discoverIPDnsmasq(ctx context.Context, macAddress, bridge string, timeout time.Duration, logger *slog.Logger) (string, error) {
	mac := strings.ToLower(macAddress)
	deadline := time.Now().Add(timeout)

	// Sanitize bridge name to prevent path traversal.
	safeBridge := filepath.Base(bridge)
	leaseFile := fmt.Sprintf("/var/lib/deer/dnsmasq/%s.leases", safeBridge)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		data, err := os.ReadFile(leaseFile)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 3 && strings.EqualFold(fields[1], mac) {
					logger.Info("discovered IP via dnsmasq lease", "mac", macAddress, "ip", fields[2])
					return fields[2], nil
				}
			}
		}

		if err := contextSleep(ctx, 2*time.Second); err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf("IP discovery timed out for MAC %s (dnsmasq mode)", macAddress)
}

func contextSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
