package tui

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspectrr/deer.sh/deer-cli/internal/sandbox"
)

func TestResolveAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty input defaults to localhost:9091",
			input:    "",
			expected: "localhost:9091",
		},
		{
			name:     "address with port passes through",
			input:    "example.com:9091",
			expected: "example.com:9091",
		},
		{
			name:     "address without port gets default",
			input:    "example.com",
			expected: "example.com:9091",
		},
		{
			name:     "IPv4 address without port",
			input:    "192.168.1.1",
			expected: "192.168.1.1:9091",
		},
		{
			name:     "IPv4 address with port",
			input:    "192.168.1.1:8080",
			expected: "192.168.1.1:8080",
		},
		{
			name:     "localhost with port",
			input:    "localhost:5000",
			expected: "localhost:5000",
		},
		{
			name:     "whitespace trimmed",
			input:    "  example.com  ",
			expected: "example.com:9091",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := ConnectModel{
				step:    StepAddress,
				focused: fieldAddress,
			}
			m.inputs = [fieldInsecure]textinput.Model{textinput.New(), textinput.New()}
			m.inputs[fieldAddress].SetValue(tt.input)

			addr, err := m.resolveAddress()
			if err != nil {
				t.Errorf("resolveAddress() unexpected error: %v", err)
			}
			if addr != tt.expected {
				t.Errorf("resolveAddress() = %q, want %q", addr, tt.expected)
			}
		})
	}
}

func TestResolveAddress_Invalid(t *testing.T) {
	m := ConnectModel{
		step:    StepAddress,
		focused: fieldAddress,
	}
	m.inputs = [fieldInsecure]textinput.Model{textinput.New(), textinput.New()}

	// Test various edge cases that should still work
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string gets default", "", "localhost:9091"},
		{"whitespace only gets default", "   ", "localhost:9091"},
		{"valid host", "myhost", "myhost:9091"},
		{"valid host:port", "myhost:1234", "myhost:1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.inputs[fieldAddress].SetValue(tt.input)
			addr, err := m.resolveAddress()
			if err != nil {
				t.Errorf("resolveAddress() unexpected error for %q: %v", tt.input, err)
			}
			if addr != tt.expected {
				t.Errorf("resolveAddress(%q) = %q, want %q", tt.input, addr, tt.expected)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	tests := []struct {
		name         string
		addressInput string
		nameInput    string
		hostInfo     *sandbox.HostInfo
		insecure     bool
		wantName     string
		wantAddr     string
	}{
		{
			name:         "uses provided name",
			addressInput: "example.com:9091",
			nameInput:    "mydaemon",
			hostInfo:     nil,
			wantName:     "mydaemon",
			wantAddr:     "example.com:9091",
		},
		{
			name:         "uses hostname when name empty",
			addressInput: "example.com:9091",
			nameInput:    "",
			hostInfo:     &sandbox.HostInfo{Hostname: "server1"},
			wantName:     "server1",
			wantAddr:     "example.com:9091",
		},
		{
			name:         "uses default when both empty",
			addressInput: "example.com:9091",
			nameInput:    "",
			hostInfo:     nil,
			wantName:     "default",
			wantAddr:     "example.com:9091",
		},
		{
			name:         "appends port when missing",
			addressInput: "example.com",
			nameInput:    "mydaemon",
			hostInfo:     nil,
			wantName:     "mydaemon",
			wantAddr:     "example.com:9091",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := ConnectModel{
				step:     StepAddress,
				focused:  fieldAddress,
				hostInfo: tt.hostInfo,
				insecure: tt.insecure,
			}
			m.inputs = [fieldInsecure]textinput.Model{textinput.New(), textinput.New()}
			m.inputs[fieldAddress].SetValue(tt.addressInput)
			m.inputs[fieldName].SetValue(tt.nameInput)

			cfg := m.buildConfig()
			if cfg.Name != tt.wantName {
				t.Errorf("buildConfig().Name = %q, want %q", cfg.Name, tt.wantName)
			}
			if cfg.DaemonAddress != tt.wantAddr {
				t.Errorf("buildConfig().DaemonAddress = %q, want %q", cfg.DaemonAddress, tt.wantAddr)
			}
		})
	}
}

func TestUpdate_TabNavigation(t *testing.T) {
	m := NewConnectModel(nil)

	// Initial state: focused on address field
	if m.focused != fieldAddress {
		t.Errorf("initial focused = %v, want %v", m.focused, fieldAddress)
	}

	// Press tab to go to name field
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(ConnectModel)
	if m.focused != fieldName {
		t.Errorf("after tab: focused = %v, want %v", m.focused, fieldName)
	}

	// Press tab again to go to insecure checkbox
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(ConnectModel)
	if m.focused != fieldInsecure {
		t.Errorf("after second tab: focused = %v, want %v", m.focused, fieldInsecure)
	}

	// Press tab again to wrap back to address
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedModel.(ConnectModel)
	if m.focused != fieldAddress {
		t.Errorf("after third tab: focused = %v, want %v", m.focused, fieldAddress)
	}
}

func TestUpdate_ShiftTabNavigation(t *testing.T) {
	m := NewConnectModel(nil)

	// Initial state: address field
	m.focused = fieldAddress

	// Shift+tab should go backwards (address -> insecure)
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updatedModel.(ConnectModel)
	if m.focused != fieldInsecure {
		t.Errorf("after shift+tab: focused = %v, want %v", m.focused, fieldInsecure)
	}
}

func TestUpdate_EscapeCancels(t *testing.T) {
	m := NewConnectModel(nil)

	// Press escape on address step
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_ = updatedModel.(ConnectModel)

	// Should return ConnectCloseMsg
	msg := cmd()
	if _, ok := msg.(ConnectCloseMsg); !ok {
		t.Errorf("expected ConnectCloseMsg, got %T", msg)
	}
}

func TestUpdate_InsecureToggle(t *testing.T) {
	m := NewConnectModel(nil)
	m.focused = fieldInsecure

	if m.insecure {
		t.Error("initial insecure should be false")
	}

	// Press space to toggle insecure on
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updatedModel.(ConnectModel)
	if !m.insecure {
		t.Error("insecure should be true after space")
	}

	// Press space again to toggle off
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updatedModel.(ConnectModel)
	if m.insecure {
		t.Error("insecure should be false after second space")
	}

	// Press 'n' to set to false
	m.insecure = true
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updatedModel.(ConnectModel)
	if m.insecure {
		t.Error("insecure should be false after 'n'")
	}

	// Press 'y' to set to true
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updatedModel.(ConnectModel)
	if !m.insecure {
		t.Error("insecure should be true after 'y'")
	}
}

func TestUpdate_PerHostDeploy(t *testing.T) {
	m := NewConnectModel(nil)
	m.step = StepDone
	m.hostInfo = &sandbox.HostInfo{
		Hostname:          "daemon1",
		SSHIdentityPubKey: "ssh-ed25519 AAAA...",
		SourceHosts: []sandbox.SourceHostInfo{
			{Address: "192.168.1.10", SSHUser: "deer-daemon", SSHPort: 22},
			{Address: "192.168.1.11", SSHUser: "deer-daemon", SSHPort: 22},
			{Address: "192.168.1.12", SSHUser: "deer-daemon", SSHPort: 22},
		},
	}

	// Press enter to start deploy
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedModel.(ConnectModel)

	if m.step != StepDeployKeys {
		t.Fatalf("expected StepDeployKeys, got %v", m.step)
	}
	if len(m.hostDeployStatuses) != 3 {
		t.Fatalf("expected 3 host statuses, got %d", len(m.hostDeployStatuses))
	}
	if m.hostDeployStatuses[0].State != HostDeployDeploying {
		t.Errorf("host 0 should be deploying, got %v", m.hostDeployStatuses[0].State)
	}
	if m.hostDeployStatuses[1].State != HostDeployPending {
		t.Errorf("host 1 should be pending, got %v", m.hostDeployStatuses[1].State)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Simulate host 0 success
	updatedModel, cmd = m.Update(HostKeyDeployedMsg{Host: "host1", Index: 0, Err: nil})
	m = updatedModel.(ConnectModel)
	if m.hostDeployStatuses[0].State != HostDeployDone {
		t.Errorf("host 0 should be done, got %v", m.hostDeployStatuses[0].State)
	}
	if m.hostDeployStatuses[1].State != HostDeployDeploying {
		t.Errorf("host 1 should be deploying, got %v", m.hostDeployStatuses[1].State)
	}
	if cmd == nil {
		t.Fatal("expected cmd for next host")
	}

	// Simulate host 1 failure
	updatedModel, _ = m.Update(HostKeyDeployedMsg{Host: "host2", Index: 1, Err: fmt.Errorf("connection refused")})
	m = updatedModel.(ConnectModel)
	if m.hostDeployStatuses[1].State != HostDeployFailed {
		t.Errorf("host 1 should be failed, got %v", m.hostDeployStatuses[1].State)
	}
	if m.hostDeployStatuses[2].State != HostDeployDeploying {
		t.Errorf("host 2 should be deploying, got %v", m.hostDeployStatuses[2].State)
	}

	// Simulate host 2 success - should scan host keys, then rerun doctor checks
	updatedModel, _ = m.Update(HostKeyDeployedMsg{Host: "host3", Index: 2, Err: nil})
	m = updatedModel.(ConnectModel)
	if !m.keysDeployed {
		t.Error("keysDeployed should be true after all hosts")
	}
	if m.step != StepDoctor {
		t.Errorf("should be on StepDoctor (scanning keys), got %v", m.step)
	}
	if m.deployResults == nil {
		t.Fatal("deployResults should be set")
	}
	if m.deployResults.Deployed != 2 {
		t.Errorf("expected 2 deployed, got %d", m.deployResults.Deployed)
	}
	if len(m.deployResults.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(m.deployResults.Errors))
	}

	// Simulate scan keys complete - should transition to doctor checks
	updatedModel, _ = m.Update(ScanKeysCompleteMsg{})
	m = updatedModel.(ConnectModel)
	if m.step != StepDoctor {
		t.Errorf("should be on StepDoctor (running checks after scan), got %v", m.step)
	}
}
