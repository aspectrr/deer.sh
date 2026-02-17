package sshconfig

import (
	"testing"
)

func TestProbeResult_DefaultValues(t *testing.T) {
	result := ProbeResult{
		Host: SSHHost{Name: "test", HostName: "10.0.0.1", Port: 22},
	}

	if result.Reachable {
		t.Error("expected Reachable=false by default")
	}
	if result.HasLibvirt {
		t.Error("expected HasLibvirt=false by default")
	}
	if result.HasProxmox {
		t.Error("expected HasProxmox=false by default")
	}
	if len(result.VMs) != 0 {
		t.Error("expected no VMs by default")
	}
}

// Note: integration tests for Probe() and ProbeAll() require actual SSH access
// and are not included here. These would be tested via manual or end-to-end tests.
