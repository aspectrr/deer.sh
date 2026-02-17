package lxc

import (
	"testing"
	"time"
)

func TestConfig_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "missing host",
			cfg:     Config{TokenID: "user@pam!tok", Secret: "s", Node: "pve"},
			wantErr: "lxc host is required",
		},
		{
			name:    "missing token_id",
			cfg:     Config{Host: "https://pve:8006", Secret: "s", Node: "pve"},
			wantErr: "lxc token_id is required",
		},
		{
			name:    "missing secret",
			cfg:     Config{Host: "https://pve:8006", TokenID: "user@pam!tok", Node: "pve"},
			wantErr: "lxc secret is required",
		},
		{
			name:    "missing node",
			cfg:     Config{Host: "https://pve:8006", TokenID: "user@pam!tok", Secret: "s"},
			wantErr: "lxc node is required",
		},
		{
			name: "vmid_end less than vmid_start",
			cfg: Config{
				Host: "https://pve:8006", TokenID: "user@pam!tok", Secret: "s", Node: "pve",
				VMIDStart: 100, VMIDEnd: 50,
			},
			wantErr: "vmid_end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_Defaults(t *testing.T) {
	cfg := Config{
		Host:    "https://pve:8006",
		TokenID: "user@pam!tok",
		Secret:  "secret123",
		Node:    "pve",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	if cfg.VMIDStart != 9000 {
		t.Errorf("VMIDStart = %d, want 9000", cfg.VMIDStart)
	}
	if cfg.VMIDEnd != 9999 {
		t.Errorf("VMIDEnd = %d, want 9999", cfg.VMIDEnd)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want 5m", cfg.Timeout)
	}
	if cfg.Bridge != "vmbr0" {
		t.Errorf("Bridge = %q, want vmbr0", cfg.Bridge)
	}
}

func TestConfig_Validate_PreservesExplicitValues(t *testing.T) {
	cfg := Config{
		Host:      "https://pve:8006",
		TokenID:   "user@pam!tok",
		Secret:    "secret123",
		Node:      "pve",
		VMIDStart: 5000,
		VMIDEnd:   5999,
		Timeout:   30 * time.Second,
		Bridge:    "vmbr1",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	if cfg.VMIDStart != 5000 {
		t.Errorf("VMIDStart = %d, want 5000", cfg.VMIDStart)
	}
	if cfg.VMIDEnd != 5999 {
		t.Errorf("VMIDEnd = %d, want 5999", cfg.VMIDEnd)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
	if cfg.Bridge != "vmbr1" {
		t.Errorf("Bridge = %q, want vmbr1", cfg.Bridge)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
