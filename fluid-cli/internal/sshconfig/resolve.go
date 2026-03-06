package sshconfig

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// ResolvedHost contains SSH connection details resolved from ~/.ssh/config.
type ResolvedHost struct {
	Hostname     string
	User         string
	Port         int
	IdentityFile string
}

// Resolve uses `ssh -G` to resolve connection details for a host alias.
// This handles all SSH config features (includes, wildcards, ProxyJump, etc.)
// without needing a Go SSH config parser.
func Resolve(hostAlias string) (*ResolvedHost, error) {
	cmd := exec.Command("ssh", "-G", hostAlias)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ssh -G %s: %w (stderr: %s)", hostAlias, err, stderr.String())
	}

	result := &ResolvedHost{
		Port: 22,
	}

	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "hostname":
			result.Hostname = value
		case "user":
			result.User = value
		case "port":
			if p, err := strconv.Atoi(value); err == nil {
				result.Port = p
			}
		case "identityfile":
			// Take the first identity file that doesn't look like a default
			if result.IdentityFile == "" {
				result.IdentityFile = value
			}
		}
	}

	if result.Hostname == "" {
		result.Hostname = hostAlias
	}
	if result.User == "" {
		if u, err := user.Current(); err == nil {
			result.User = u.Username
		} else {
			result.User = "root"
		}
	}

	return result, nil
}

// ListHosts parses ~/.ssh/config and returns Host aliases, filtering out wildcards.
// TODO: this does not follow Include directives in ~/.ssh/config. Resolve() uses
// `ssh -G` which handles includes, so host resolution is unaffected.
func ListHosts() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	f, err := os.Open(filepath.Join(home, ".ssh", "config"))
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	return ListHostsFromReader(f)
}

// ListHostsFromReader scans an SSH config from r and returns Host aliases,
// filtering out wildcard patterns.
func ListHostsFromReader(r io.Reader) []string {
	var hosts []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		if strings.EqualFold(parts[0], "Host") {
			// A Host line can have multiple patterns
			for _, h := range parts[1:] {
				if !strings.Contains(h, "*") && !strings.Contains(h, "?") {
					hosts = append(hosts, h)
				}
			}
		}
	}
	return hosts
}
