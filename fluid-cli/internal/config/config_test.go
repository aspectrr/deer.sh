package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 2, cfg.VM.DefaultVCPUs)
	assert.Equal(t, 4096, cfg.VM.DefaultMemoryMB)
	assert.Equal(t, "qemu:///system", cfg.Libvirt.URI)
	assert.Equal(t, "info", cfg.Logging.Level)
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoad_ValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yaml := `
api:
  addr: ":9090"
  read_timeout: 30s

vm:
  default_vcpus: 4
  default_memory_mb: 4096
  command_timeout: 5m

logging:
  level: "debug"
  format: "json"
`
	err := os.WriteFile(configPath, []byte(yaml), 0o644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, 4, cfg.VM.DefaultVCPUs)
	assert.Equal(t, 4096, cfg.VM.DefaultMemoryMB)
	assert.Equal(t, 5*time.Minute, cfg.VM.CommandTimeout)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestLoad_PartialYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Only override some values - defaults should fill the rest
	yaml := `
api:
  addr: ":3000"
logging:
  level: "warn"
`
	err := os.WriteFile(configPath, []byte(yaml), 0o644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Overridden values
	assert.Equal(t, "warn", cfg.Logging.Level)

	// Default values preserved
	assert.Equal(t, 2, cfg.VM.DefaultVCPUs)
	assert.Equal(t, "qemu:///system", cfg.Libvirt.URI)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0o644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
}

func TestLoadWithEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yaml := `
logging:
  level: "info"
`
	err := os.WriteFile(configPath, []byte(yaml), 0o600)
	require.NoError(t, err)

	// Set env vars to override (only LOG_LEVEL is supported)
	t.Setenv("LOG_LEVEL", "debug")

	cfg, warnings, err := LoadWithEnvOverride(configPath)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	// Env vars should override YAML
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestApplyEnvOverrides_AllFields(t *testing.T) {
	cfg := DefaultConfig()

	// Only these env vars are currently supported by applyEnvOverrides
	t.Setenv("ENABLE_ANONYMOUS_USAGE", "false")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("OPENROUTER_API_KEY", "test-api-key")

	applyEnvOverrides(cfg)

	assert.Equal(t, false, cfg.Telemetry.EnableAnonymousUsage)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "test-api-key", cfg.AIAgent.APIKey)
}

func TestDefaultConfig_ProxmoxDefaults(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "libvirt", cfg.Provider)
	assert.Equal(t, true, cfg.Proxmox.VerifySSL)
	assert.Equal(t, "full", cfg.Proxmox.CloneMode)
	assert.Equal(t, 9000, cfg.Proxmox.VMIDStart)
	assert.Equal(t, 9999, cfg.Proxmox.VMIDEnd)
	assert.Empty(t, cfg.Proxmox.Host)
	assert.Empty(t, cfg.Proxmox.TokenID)
	assert.Empty(t, cfg.Proxmox.Secret)
	assert.Empty(t, cfg.Proxmox.Node)
}

func TestLoad_ProxmoxConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
provider: proxmox
proxmox:
  host: "https://pve.example.com:8006"
  token_id: "root@pam!fluid"
  secret: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  node: "pve1"
  verify_ssl: false
  storage: "local-lvm"
  bridge: "vmbr0"
  clone_mode: "linked"
  vmid_start: 5000
  vmid_end: 5999
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "proxmox", cfg.Provider)
	assert.Equal(t, "https://pve.example.com:8006", cfg.Proxmox.Host)
	assert.Equal(t, "root@pam!fluid", cfg.Proxmox.TokenID)
	assert.Equal(t, "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", cfg.Proxmox.Secret)
	assert.Equal(t, "pve1", cfg.Proxmox.Node)
	assert.Equal(t, false, cfg.Proxmox.VerifySSL)
	assert.Equal(t, "local-lvm", cfg.Proxmox.Storage)
	assert.Equal(t, "vmbr0", cfg.Proxmox.Bridge)
	assert.Equal(t, "linked", cfg.Proxmox.CloneMode)
	assert.Equal(t, 5000, cfg.Proxmox.VMIDStart)
	assert.Equal(t, 5999, cfg.Proxmox.VMIDEnd)
}

func TestLoad_ProxmoxPartialConfig_AppliesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
provider: proxmox
proxmox:
  host: "https://pve.example.com:8006"
  token_id: "root@pam!fluid"
  secret: "my-secret"
  node: "pve1"
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Specified fields
	assert.Equal(t, "proxmox", cfg.Provider)
	assert.Equal(t, "https://pve.example.com:8006", cfg.Proxmox.Host)

	// Defaults applied
	assert.Equal(t, "full", cfg.Proxmox.CloneMode)
	assert.Equal(t, 9000, cfg.Proxmox.VMIDStart)
	assert.Equal(t, 9999, cfg.Proxmox.VMIDEnd)
}

func TestApplyEnvOverrides_Proxmox(t *testing.T) {
	cfg := DefaultConfig()

	t.Setenv("PROXMOX_HOST", "https://pve2.example.com:8006")
	t.Setenv("PROXMOX_TOKEN_ID", "admin@pam!ci")
	t.Setenv("PROXMOX_SECRET", "env-secret-value")
	t.Setenv("PROXMOX_NODE", "node2")

	applyEnvOverrides(cfg)

	assert.Equal(t, "proxmox", cfg.Provider) // Auto-set when PROXMOX_HOST is set
	assert.Equal(t, "https://pve2.example.com:8006", cfg.Proxmox.Host)
	assert.Equal(t, "admin@pam!ci", cfg.Proxmox.TokenID)
	assert.Equal(t, "env-secret-value", cfg.Proxmox.Secret)
	assert.Equal(t, "node2", cfg.Proxmox.Node)
}

func TestApplyEnvOverrides_ProxmoxHostSetsProvider(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "libvirt", cfg.Provider)

	t.Setenv("PROXMOX_HOST", "https://pve.example.com:8006")
	applyEnvOverrides(cfg)

	// Setting PROXMOX_HOST should auto-switch provider to proxmox
	assert.Equal(t, "proxmox", cfg.Provider)
}

func TestApplyEnvOverrides_ProxmoxPartial(t *testing.T) {
	cfg := DefaultConfig()

	// Only set some env vars - others should remain at defaults
	t.Setenv("PROXMOX_NODE", "node3")
	applyEnvOverrides(cfg)

	assert.Equal(t, "node3", cfg.Proxmox.Node)
	assert.Equal(t, "libvirt", cfg.Provider) // Not changed without PROXMOX_HOST
	assert.Empty(t, cfg.Proxmox.Host)
}

func TestLoad_ProxmoxEnvOverridesYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
provider: proxmox
proxmox:
  host: "https://yaml-host:8006"
  token_id: "yaml-token"
  secret: "yaml-secret"
  node: "yaml-node"
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0o600)
	require.NoError(t, err)

	// Env vars override YAML values
	t.Setenv("PROXMOX_HOST", "https://env-host:8006")
	t.Setenv("PROXMOX_SECRET", "env-secret")

	cfg, _, err := LoadWithEnvOverride(configPath)
	require.NoError(t, err)

	// Env vars take precedence
	assert.Equal(t, "https://env-host:8006", cfg.Proxmox.Host)
	assert.Equal(t, "env-secret", cfg.Proxmox.Secret)
	// YAML values preserved when no env override
	assert.Equal(t, "yaml-token", cfg.Proxmox.TokenID)
	assert.Equal(t, "yaml-node", cfg.Proxmox.Node)
}

func TestSave_PreservesProxmoxConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Provider = "proxmox"
	cfg.Proxmox.Host = "https://pve.example.com:8006"
	cfg.Proxmox.TokenID = "root@pam!fluid"
	cfg.Proxmox.Secret = "test-secret"
	cfg.Proxmox.Node = "pve1"
	cfg.Proxmox.Storage = "ceph"
	cfg.Proxmox.Bridge = "vmbr1"

	err := cfg.Save(configPath)
	require.NoError(t, err)

	// Reload and verify
	loaded, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "proxmox", loaded.Provider)
	assert.Equal(t, "https://pve.example.com:8006", loaded.Proxmox.Host)
	assert.Equal(t, "root@pam!fluid", loaded.Proxmox.TokenID)
	assert.Equal(t, "test-secret", loaded.Proxmox.Secret)
	assert.Equal(t, "pve1", loaded.Proxmox.Node)
	assert.Equal(t, "ceph", loaded.Proxmox.Storage)
	assert.Equal(t, "vmbr1", loaded.Proxmox.Bridge)
}

func TestApplyDefaults_ProviderDefault(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)
	assert.Equal(t, "libvirt", cfg.Provider)
}

func TestApplyDefaults_ProxmoxDefaults(t *testing.T) {
	cfg := &Config{Provider: "proxmox"}
	applyDefaults(cfg)
	assert.Equal(t, "full", cfg.Proxmox.CloneMode)
	assert.Equal(t, 9000, cfg.Proxmox.VMIDStart)
	assert.Equal(t, 9999, cfg.Proxmox.VMIDEnd)
}

func TestCheckFilePermissions_SecureFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(path, []byte("test"), 0o600)
	require.NoError(t, err)

	warnings := CheckFilePermissions(path)
	assert.Empty(t, warnings)
}

func TestCheckFilePermissions_InsecureFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(path, []byte("test"), 0o644)
	require.NoError(t, err)

	warnings := CheckFilePermissions(path)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "insecure permissions")
	assert.Contains(t, warnings[0], "chmod 600")
}

func TestCheckFilePermissions_NonexistentFile(t *testing.T) {
	warnings := CheckFilePermissions("/nonexistent/path/config.yaml")
	assert.Empty(t, warnings)
}

func TestSave_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	err := cfg.Save(configPath)
	require.NoError(t, err)

	info, err := os.Stat(configPath)
	require.NoError(t, err)

	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"60", 60 * time.Second},
		{"300", 5 * time.Minute},
		{"5m", 5 * time.Minute},
		{"1h", time.Hour},
		{"30s", 30 * time.Second},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDuration(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasSecrets(t *testing.T) {
	tests := []struct {
		name   string
		cfg    Config
		expect bool
	}{
		{"empty config", Config{}, false},
		{"proxmox secret", Config{Proxmox: ProxmoxConfig{Secret: "s"}}, true},
		{"proxmox token_id", Config{Proxmox: ProxmoxConfig{TokenID: "t"}}, true},
		{"ai api key", Config{AIAgent: AIAgentConfig{APIKey: "k"}}, true},
		{"all secrets", Config{
			Proxmox: ProxmoxConfig{Secret: "s", TokenID: "t"},
			AIAgent: AIAgentConfig{APIKey: "k"},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.cfg.HasSecrets())
		})
	}
}

func TestLoadWithEnvOverride_PermissionWarnings(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write config with insecure permissions (0644) but no secrets
	err := os.WriteFile(configPath, []byte("logging:\n  level: info\n"), 0o644)
	require.NoError(t, err)

	_, warnings, err := LoadWithEnvOverride(configPath)
	require.NoError(t, err)
	// Should have permission warning but NOT the secrets warning
	assert.NotEmpty(t, warnings)
	assert.Contains(t, warnings[0], "insecure permissions")
	// Only 1 warning - no secrets present
	assert.Len(t, warnings, 1)
}

func TestLoadWithEnvOverride_SecureFileNoWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte("logging:\n  level: info\n"), 0o600)
	require.NoError(t, err)

	_, warnings, err := LoadWithEnvOverride(configPath)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestLoadWithEnvOverride_InsecureWithSecrets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config with secrets and insecure permissions
	yamlContent := `
proxmox:
  secret: "my-secret-token"
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	_, warnings, err := LoadWithEnvOverride(configPath)
	require.NoError(t, err)
	// Should have both: permission warning + secrets warning
	assert.Len(t, warnings, 2)
	assert.Contains(t, warnings[0], "insecure permissions")
	assert.Contains(t, warnings[1], "contains secrets")
}
