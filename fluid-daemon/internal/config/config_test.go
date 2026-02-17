package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// ControlPlane defaults
	if cfg.ControlPlane.Address != "localhost:9090" {
		t.Errorf("ControlPlane.Address = %q, want %q", cfg.ControlPlane.Address, "localhost:9090")
	}
	if !cfg.ControlPlane.Insecure {
		t.Error("ControlPlane.Insecure = false, want true")
	}

	// MicroVM defaults
	if cfg.MicroVM.QEMUBinary != "qemu-system-x86_64" {
		t.Errorf("MicroVM.QEMUBinary = %q, want %q", cfg.MicroVM.QEMUBinary, "qemu-system-x86_64")
	}
	if cfg.MicroVM.WorkDir != "/var/lib/fluid/sandboxes" {
		t.Errorf("MicroVM.WorkDir = %q, want %q", cfg.MicroVM.WorkDir, "/var/lib/fluid/sandboxes")
	}
	if cfg.MicroVM.DefaultVCPUs != 2 {
		t.Errorf("MicroVM.DefaultVCPUs = %d, want %d", cfg.MicroVM.DefaultVCPUs, 2)
	}
	if cfg.MicroVM.DefaultMemoryMB != 2048 {
		t.Errorf("MicroVM.DefaultMemoryMB = %d, want %d", cfg.MicroVM.DefaultMemoryMB, 2048)
	}
	if cfg.MicroVM.CommandTimeout != 5*time.Minute {
		t.Errorf("MicroVM.CommandTimeout = %v, want %v", cfg.MicroVM.CommandTimeout, 5*time.Minute)
	}
	if cfg.MicroVM.IPDiscoveryTimeout != 2*time.Minute {
		t.Errorf("MicroVM.IPDiscoveryTimeout = %v, want %v", cfg.MicroVM.IPDiscoveryTimeout, 2*time.Minute)
	}

	// Network defaults
	if cfg.Network.DefaultBridge != "virbr0" {
		t.Errorf("Network.DefaultBridge = %q, want %q", cfg.Network.DefaultBridge, "virbr0")
	}
	if cfg.Network.DHCPMode != "arp" {
		t.Errorf("Network.DHCPMode = %q, want %q", cfg.Network.DHCPMode, "arp")
	}
	if v, ok := cfg.Network.BridgeMap["default"]; !ok || v != "virbr0" {
		t.Errorf("Network.BridgeMap[\"default\"] = %q (ok=%v), want %q", v, ok, "virbr0")
	}

	// Image defaults
	if cfg.Image.BaseDir != "/var/lib/fluid/images" {
		t.Errorf("Image.BaseDir = %q, want %q", cfg.Image.BaseDir, "/var/lib/fluid/images")
	}

	// SSH defaults
	home, _ := os.UserHomeDir()
	fluidDir := filepath.Join(home, ".fluid")
	if cfg.SSH.CAKeyPath != filepath.Join(fluidDir, "ssh_ca") {
		t.Errorf("SSH.CAKeyPath = %q, want %q", cfg.SSH.CAKeyPath, filepath.Join(fluidDir, "ssh_ca"))
	}
	if cfg.SSH.CAPubKeyPath != filepath.Join(fluidDir, "ssh_ca.pub") {
		t.Errorf("SSH.CAPubKeyPath = %q, want %q", cfg.SSH.CAPubKeyPath, filepath.Join(fluidDir, "ssh_ca.pub"))
	}
	if cfg.SSH.KeyDir != filepath.Join(fluidDir, "keys") {
		t.Errorf("SSH.KeyDir = %q, want %q", cfg.SSH.KeyDir, filepath.Join(fluidDir, "keys"))
	}
	if cfg.SSH.CertTTL != 30*time.Minute {
		t.Errorf("SSH.CertTTL = %v, want %v", cfg.SSH.CertTTL, 30*time.Minute)
	}
	if cfg.SSH.DefaultUser != "sandbox" {
		t.Errorf("SSH.DefaultUser = %q, want %q", cfg.SSH.DefaultUser, "sandbox")
	}

	// Libvirt defaults
	if cfg.Libvirt.URI != "qemu:///system" {
		t.Errorf("Libvirt.URI = %q, want %q", cfg.Libvirt.URI, "qemu:///system")
	}
	if cfg.Libvirt.Network != "default" {
		t.Errorf("Libvirt.Network = %q, want %q", cfg.Libvirt.Network, "default")
	}

	// State defaults
	if cfg.State.DBPath != filepath.Join(fluidDir, "sandbox-host.db") {
		t.Errorf("State.DBPath = %q, want %q", cfg.State.DBPath, filepath.Join(fluidDir, "sandbox-host.db"))
	}

	// Janitor defaults
	if cfg.Janitor.Interval != 1*time.Minute {
		t.Errorf("Janitor.Interval = %v, want %v", cfg.Janitor.Interval, 1*time.Minute)
	}
	if cfg.Janitor.DefaultTTL != 24*time.Hour {
		t.Errorf("Janitor.DefaultTTL = %v, want %v", cfg.Janitor.DefaultTTL, 24*time.Hour)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%q) returned error: %v", path, err)
	}

	// Should return defaults
	defaults := DefaultConfig()
	if cfg.MicroVM.DefaultVCPUs != defaults.MicroVM.DefaultVCPUs {
		t.Errorf("DefaultVCPUs = %d, want default %d", cfg.MicroVM.DefaultVCPUs, defaults.MicroVM.DefaultVCPUs)
	}
	if cfg.Network.DefaultBridge != defaults.Network.DefaultBridge {
		t.Errorf("DefaultBridge = %q, want default %q", cfg.Network.DefaultBridge, defaults.Network.DefaultBridge)
	}
	if cfg.ControlPlane.Address != defaults.ControlPlane.Address {
		t.Errorf("ControlPlane.Address = %q, want default %q", cfg.ControlPlane.Address, defaults.ControlPlane.Address)
	}
}

func TestLoad_ValidYAML(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		check func(t *testing.T, cfg *Config)
	}{
		{
			name: "override control plane address",
			yaml: `control_plane:
  address: "cp.example.com:443"
  insecure: false
`,
			check: func(t *testing.T, cfg *Config) {
				if cfg.ControlPlane.Address != "cp.example.com:443" {
					t.Errorf("ControlPlane.Address = %q, want %q", cfg.ControlPlane.Address, "cp.example.com:443")
				}
				if cfg.ControlPlane.Insecure {
					t.Error("ControlPlane.Insecure = true, want false")
				}
			},
		},
		{
			name: "override microvm settings",
			yaml: `microvm:
  default_vcpus: 4
  default_memory_mb: 4096
  qemu_binary: /usr/local/bin/qemu-system-x86_64
`,
			check: func(t *testing.T, cfg *Config) {
				if cfg.MicroVM.DefaultVCPUs != 4 {
					t.Errorf("DefaultVCPUs = %d, want %d", cfg.MicroVM.DefaultVCPUs, 4)
				}
				if cfg.MicroVM.DefaultMemoryMB != 4096 {
					t.Errorf("DefaultMemoryMB = %d, want %d", cfg.MicroVM.DefaultMemoryMB, 4096)
				}
				if cfg.MicroVM.QEMUBinary != "/usr/local/bin/qemu-system-x86_64" {
					t.Errorf("QEMUBinary = %q, want %q", cfg.MicroVM.QEMUBinary, "/usr/local/bin/qemu-system-x86_64")
				}
			},
		},
		{
			name: "override network bridge",
			yaml: `network:
  default_bridge: br0
  dhcp_mode: dnsmasq
  bridge_map:
    production: br0
`,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Network.DefaultBridge != "br0" {
					t.Errorf("DefaultBridge = %q, want %q", cfg.Network.DefaultBridge, "br0")
				}
				if cfg.Network.DHCPMode != "dnsmasq" {
					t.Errorf("DHCPMode = %q, want %q", cfg.Network.DHCPMode, "dnsmasq")
				}
				if v, ok := cfg.Network.BridgeMap["production"]; !ok || v != "br0" {
					t.Errorf("BridgeMap[\"production\"] = %q (ok=%v), want %q", v, ok, "br0")
				}
			},
		},
		{
			name: "override janitor settings",
			yaml: `janitor:
  interval: 5m
  default_ttl: 48h
`,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Janitor.Interval != 5*time.Minute {
					t.Errorf("Janitor.Interval = %v, want %v", cfg.Janitor.Interval, 5*time.Minute)
				}
				if cfg.Janitor.DefaultTTL != 48*time.Hour {
					t.Errorf("Janitor.DefaultTTL = %v, want %v", cfg.Janitor.DefaultTTL, 48*time.Hour)
				}
			},
		},
		{
			name: "partial override preserves defaults",
			yaml: `microvm:
  default_vcpus: 8
`,
			check: func(t *testing.T, cfg *Config) {
				if cfg.MicroVM.DefaultVCPUs != 8 {
					t.Errorf("DefaultVCPUs = %d, want %d", cfg.MicroVM.DefaultVCPUs, 8)
				}
				// Non-overridden fields keep defaults
				if cfg.Network.DefaultBridge != "virbr0" {
					t.Errorf("DefaultBridge = %q, want default %q", cfg.Network.DefaultBridge, "virbr0")
				}
				if cfg.MicroVM.DefaultMemoryMB != 2048 {
					t.Errorf("DefaultMemoryMB = %d, want default %d", cfg.MicroVM.DefaultMemoryMB, 2048)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")

			if err := os.WriteFile(path, []byte(tt.yaml), 0o644); err != nil {
				t.Fatalf("write test yaml: %v", err)
			}

			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load(%q) returned error: %v", path, err)
			}

			tt.check(t, cfg)
		})
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "malformed yaml",
			content: ":\n\t- :\n\t\t invalid: [",
		},
		{
			name:    "tabs instead of spaces",
			content: "control_plane:\n\taddress: bad",
		},
		{
			name:    "wrong type for integer field",
			content: "microvm:\n  default_vcpus: not_a_number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "bad.yaml")

			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("write test yaml: %v", err)
			}

			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load(%q) expected error for invalid YAML, got nil", path)
			}
		})
	}
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yaml")

	original := DefaultConfig()
	original.HostID = "test-host-001"
	original.ControlPlane.Address = "cp.example.com:9090"
	original.ControlPlane.Insecure = false
	original.MicroVM.DefaultVCPUs = 4
	original.MicroVM.DefaultMemoryMB = 8192
	original.Network.DefaultBridge = "br-custom"
	original.Network.BridgeMap = map[string]string{
		"default": "virbr0",
		"custom":  "br-custom",
	}
	original.SSH.DefaultUser = "admin"
	original.Libvirt.URI = "qemu+ssh://user@host/system"
	original.Janitor.DefaultTTL = 12 * time.Hour

	// Save
	if err := Save(path, &original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file was created (Save creates intermediate dirs)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file does not exist: %v", err)
	}

	// Load back
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify all overridden fields round-trip correctly
	if loaded.HostID != original.HostID {
		t.Errorf("HostID = %q, want %q", loaded.HostID, original.HostID)
	}
	if loaded.ControlPlane.Address != original.ControlPlane.Address {
		t.Errorf("ControlPlane.Address = %q, want %q", loaded.ControlPlane.Address, original.ControlPlane.Address)
	}
	if loaded.ControlPlane.Insecure != original.ControlPlane.Insecure {
		t.Errorf("ControlPlane.Insecure = %v, want %v", loaded.ControlPlane.Insecure, original.ControlPlane.Insecure)
	}
	if loaded.MicroVM.DefaultVCPUs != original.MicroVM.DefaultVCPUs {
		t.Errorf("DefaultVCPUs = %d, want %d", loaded.MicroVM.DefaultVCPUs, original.MicroVM.DefaultVCPUs)
	}
	if loaded.MicroVM.DefaultMemoryMB != original.MicroVM.DefaultMemoryMB {
		t.Errorf("DefaultMemoryMB = %d, want %d", loaded.MicroVM.DefaultMemoryMB, original.MicroVM.DefaultMemoryMB)
	}
	if loaded.Network.DefaultBridge != original.Network.DefaultBridge {
		t.Errorf("DefaultBridge = %q, want %q", loaded.Network.DefaultBridge, original.Network.DefaultBridge)
	}
	if loaded.Network.DHCPMode != original.Network.DHCPMode {
		t.Errorf("DHCPMode = %q, want %q", loaded.Network.DHCPMode, original.Network.DHCPMode)
	}
	if v, ok := loaded.Network.BridgeMap["custom"]; !ok || v != "br-custom" {
		t.Errorf("BridgeMap[\"custom\"] = %q (ok=%v), want %q", v, ok, "br-custom")
	}
	if loaded.SSH.DefaultUser != original.SSH.DefaultUser {
		t.Errorf("SSH.DefaultUser = %q, want %q", loaded.SSH.DefaultUser, original.SSH.DefaultUser)
	}
	if loaded.Libvirt.URI != original.Libvirt.URI {
		t.Errorf("Libvirt.URI = %q, want %q", loaded.Libvirt.URI, original.Libvirt.URI)
	}
	if loaded.Janitor.DefaultTTL != original.Janitor.DefaultTTL {
		t.Errorf("Janitor.DefaultTTL = %v, want %v", loaded.Janitor.DefaultTTL, original.Janitor.DefaultTTL)
	}
	if loaded.MicroVM.CommandTimeout != original.MicroVM.CommandTimeout {
		t.Errorf("CommandTimeout = %v, want %v", loaded.MicroVM.CommandTimeout, original.MicroVM.CommandTimeout)
	}
	if loaded.SSH.CertTTL != original.SSH.CertTTL {
		t.Errorf("SSH.CertTTL = %v, want %v", loaded.SSH.CertTTL, original.SSH.CertTTL)
	}
}
