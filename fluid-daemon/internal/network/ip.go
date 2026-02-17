package network

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
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

	// Try common lease file locations
	leaseFiles := []string{
		"/var/lib/libvirt/dnsmasq/default.leases",
		"/var/lib/libvirt/dnsmasq/virbr0.leases",
		fmt.Sprintf("/var/lib/libvirt/dnsmasq/%s.leases", bridge),
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
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

		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("IP discovery timed out for MAC %s (libvirt mode)", macAddress)
}

// discoverIPARP polls the ARP table to find IP for a MAC.
func discoverIPARP(ctx context.Context, macAddress, bridge string, timeout time.Duration, logger *slog.Logger) (string, error) {
	mac := strings.ToLower(macAddress)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Try ip neigh first
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
						logger.Info("discovered IP via ARP", "mac", macAddress, "ip", ip)
						return ip, nil
					}
				}
			}
		}

		// Fallback: arp -an
		cmd = exec.CommandContext(ctx, "arp", "-an")
		output, err = cmd.Output()
		if err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				if strings.Contains(strings.ToLower(line), mac) {
					// Format: ? (IP) at MAC [ether] on interface
					start := strings.Index(line, "(")
					end := strings.Index(line, ")")
					if start >= 0 && end > start {
						ip := line[start+1 : end]
						logger.Info("discovered IP via arp", "mac", macAddress, "ip", ip)
						return ip, nil
					}
				}
			}
		}

		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("IP discovery timed out for MAC %s (arp mode)", macAddress)
}

// discoverIPDnsmasq reads local dnsmasq lease file for IP discovery.
func discoverIPDnsmasq(ctx context.Context, macAddress, bridge string, timeout time.Duration, logger *slog.Logger) (string, error) {
	mac := strings.ToLower(macAddress)
	deadline := time.Now().Add(timeout)

	leaseFile := fmt.Sprintf("/var/lib/fluid/dnsmasq/%s.leases", bridge)

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

		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("IP discovery timed out for MAC %s (dnsmasq mode)", macAddress)
}
