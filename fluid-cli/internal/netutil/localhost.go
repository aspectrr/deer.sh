package netutil

import (
	"net"
	"strings"
)

// IsLocalHost reports whether the host is a loopback address or empty.
func IsLocalHost(host string) bool {
	if host == "" || host == "localhost" {
		return true
	}

	// Handle bracketed IPv6 addresses (e.g., "[::1]")
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")

	// Check for loopback IP addresses
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}

	return false
}
