package sshconfig

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ProbeResult describes the outcome of probing a single host.
type ProbeResult struct {
	Host       SSHHost
	Reachable  bool
	HasLibvirt bool
	HasProxmox bool
	VMs        []string
	Error      string
}

// ProbeAll probes multiple hosts concurrently and returns results.
func ProbeAll(ctx context.Context, hosts []SSHHost) []ProbeResult {
	results := make([]ProbeResult, len(hosts))
	var wg sync.WaitGroup

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h SSHHost) {
			defer wg.Done()
			results[idx] = Probe(ctx, h)
		}(i, host)
	}

	wg.Wait()
	return results
}

// Probe connects to a single host via SSH and detects hypervisor capabilities.
// Uses a 10-second timeout per host.
func Probe(ctx context.Context, host SSHHost) ProbeResult {
	result := ProbeResult{Host: host}

	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Test connectivity by running a simple command
	out, err := sshRun(probeCtx, host, "echo ok")
	if err != nil {
		result.Error = fmt.Sprintf("unreachable: %v", err)
		return result
	}
	if !strings.Contains(out, "ok") {
		result.Error = "unexpected ssh response"
		return result
	}
	result.Reachable = true

	// Detect libvirt
	_, err = sshRun(probeCtx, host, "which virsh")
	if err == nil {
		result.HasLibvirt = true

		// List VMs
		vmOut, err := sshRun(probeCtx, host, "virsh list --all --name")
		if err == nil {
			for _, line := range strings.Split(vmOut, "\n") {
				name := strings.TrimSpace(line)
				if name != "" {
					result.VMs = append(result.VMs, name)
				}
			}
		}
	}

	// Detect Proxmox
	_, err = sshRun(probeCtx, host, "which pveversion 2>/dev/null || test -x /usr/bin/pvesh")
	if err == nil {
		result.HasProxmox = true
	}

	return result
}

// sshRun executes a command on a remote host via SSH.
func sshRun(ctx context.Context, host SSHHost, command string) (string, error) {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-p", fmt.Sprintf("%d", host.Port),
	}
	if host.IdentityFile != "" {
		args = append(args, "-i", host.IdentityFile)
	}

	target := host.HostName
	if host.User != "" {
		target = host.User + "@" + host.HostName
	}
	args = append(args, target, command)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
