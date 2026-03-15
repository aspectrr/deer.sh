package main

import (
	"testing"

	"github.com/aspectrr/fluid.sh/fluid-cli/internal/config"
)

func TestUpsertSandboxHost(t *testing.T) {
	tests := []struct {
		name      string
		hosts     []config.SandboxHostConfig
		entry     config.SandboxHostConfig
		wantLen   int
		wantIndex int
		wantName  string
		wantAddr  string
	}{
		{
			name:      "append to empty list",
			hosts:     nil,
			entry:     config.SandboxHostConfig{Name: "host1", DaemonAddress: "10.0.0.1:9091"},
			wantLen:   1,
			wantIndex: 0,
			wantName:  "host1",
			wantAddr:  "10.0.0.1:9091",
		},
		{
			name: "append to existing list",
			hosts: []config.SandboxHostConfig{
				{Name: "host1", DaemonAddress: "10.0.0.1:9091"},
			},
			entry:     config.SandboxHostConfig{Name: "host2", DaemonAddress: "10.0.0.2:9091"},
			wantLen:   2,
			wantIndex: 1,
			wantName:  "host2",
			wantAddr:  "10.0.0.2:9091",
		},
		{
			name: "update by name match",
			hosts: []config.SandboxHostConfig{
				{Name: "host1", DaemonAddress: "10.0.0.1:9091"},
			},
			entry:     config.SandboxHostConfig{Name: "host1", DaemonAddress: "10.0.0.99:9091", Insecure: true},
			wantLen:   1,
			wantIndex: 0,
			wantName:  "host1",
			wantAddr:  "10.0.0.99:9091",
		},
		{
			name: "update by address match",
			hosts: []config.SandboxHostConfig{
				{Name: "old-name", DaemonAddress: "10.0.0.1:9091"},
			},
			entry:     config.SandboxHostConfig{Name: "new-name", DaemonAddress: "10.0.0.1:9091"},
			wantLen:   1,
			wantIndex: 0,
			wantName:  "new-name",
			wantAddr:  "10.0.0.1:9091",
		},
		{
			name: "name and address match first hit wins",
			hosts: []config.SandboxHostConfig{
				{Name: "host1", DaemonAddress: "10.0.0.1:9091"},
				{Name: "host2", DaemonAddress: "10.0.0.2:9091"},
			},
			entry:     config.SandboxHostConfig{Name: "host1", DaemonAddress: "10.0.0.1:9091", Insecure: true},
			wantLen:   2,
			wantIndex: 0,
			wantName:  "host1",
			wantAddr:  "10.0.0.1:9091",
		},
		{
			name: "different name different address appends",
			hosts: []config.SandboxHostConfig{
				{Name: "host1", DaemonAddress: "10.0.0.1:9091"},
			},
			entry:     config.SandboxHostConfig{Name: "host2", DaemonAddress: "10.0.0.2:9091"},
			wantLen:   2,
			wantIndex: 1,
			wantName:  "host2",
			wantAddr:  "10.0.0.2:9091",
		},
		{
			name: "dedup conflicting name and address entries",
			hosts: []config.SandboxHostConfig{
				{Name: "A", DaemonAddress: "X"},
				{Name: "B", DaemonAddress: "Y"},
			},
			entry:     config.SandboxHostConfig{Name: "A", DaemonAddress: "Y"},
			wantLen:   1,
			wantIndex: 0,
			wantName:  "A",
			wantAddr:  "Y",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Copy input to avoid mutating test data
			input := make([]config.SandboxHostConfig, len(tc.hosts))
			copy(input, tc.hosts)

			result := config.UpsertSandboxHost(input, tc.entry)
			if len(result) != tc.wantLen {
				t.Fatalf("got len %d, want %d", len(result), tc.wantLen)
			}

			if tc.wantName != "" {
				if result[tc.wantIndex].Name != tc.wantName {
					t.Errorf("result[%d].Name = %q, want %q", tc.wantIndex, result[tc.wantIndex].Name, tc.wantName)
				}
			}
			if tc.wantAddr != "" {
				if result[tc.wantIndex].DaemonAddress != tc.wantAddr {
					t.Errorf("result[%d].DaemonAddress = %q, want %q", tc.wantIndex, result[tc.wantIndex].DaemonAddress, tc.wantAddr)
				}
			}
		})
	}
}
