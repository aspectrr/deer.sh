// Package sshconfig parses SSH config files and probes remote hosts
// to detect hypervisor capabilities.
package sshconfig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// SSHHost represents a parsed host entry from an SSH config file.
type SSHHost struct {
	Name         string // Host alias
	HostName     string // actual hostname/IP
	User         string
	Port         int
	IdentityFile string
}

// Parse reads SSH config content from a reader and returns structured host entries.
// Wildcard entries (Host *) are skipped.
func Parse(r io.Reader) ([]SSHHost, error) {
	scanner := bufio.NewScanner(r)
	var hosts []SSHHost
	var current *SSHHost

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first whitespace or =
		key, value := splitDirective(line)
		if key == "" {
			continue
		}

		switch strings.ToLower(key) {
		case "host":
			// Save previous host if any
			if current != nil {
				hosts = append(hosts, *current)
			}

			// Skip wildcard entries
			if strings.Contains(value, "*") || strings.Contains(value, "?") {
				current = nil
				continue
			}

			current = &SSHHost{
				Name: value,
				Port: 22,
			}

		case "hostname":
			if current != nil {
				current.HostName = value
			}

		case "user":
			if current != nil {
				current.User = value
			}

		case "port":
			if current != nil {
				if p, err := strconv.Atoi(value); err == nil {
					current.Port = p
				}
			}

		case "identityfile":
			if current != nil {
				current.IdentityFile = expandTilde(value)
			}
		}
	}

	// Save last host
	if current != nil {
		hosts = append(hosts, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan ssh config: %w", err)
	}

	// Fill in defaults: if HostName is empty, use Name
	for i := range hosts {
		if hosts[i].HostName == "" {
			hosts[i].HostName = hosts[i].Name
		}
	}

	return hosts, nil
}

// ParseFile reads an SSH config file from the given path.
func ParseFile(path string) ([]SSHHost, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open ssh config %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Parse(f)
}

// splitDirective splits an SSH config line into key and value.
// Handles both "Key Value" and "Key=Value" formats.
func splitDirective(line string) (string, string) {
	// Try = separator first
	if idx := strings.IndexByte(line, '='); idx >= 0 {
		return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:])
	}

	// Split on whitespace
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(line, "\t", 2)
	}
	if len(parts) < 2 {
		return parts[0], ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

// expandTilde replaces a leading ~ with the user's home directory.
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}
