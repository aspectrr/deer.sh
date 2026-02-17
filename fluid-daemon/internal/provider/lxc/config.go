package lxc

import (
	"fmt"
	"time"
)

// Config holds settings for connecting to Proxmox VE and managing LXC containers.
type Config struct {
	Host      string        `yaml:"host"`       // Base URL, e.g. "https://proxmox:8006"
	TokenID   string        `yaml:"token_id"`   // API token ID, e.g. "user@pam!fluid"
	Secret    string        `yaml:"secret"`     // API token secret
	Node      string        `yaml:"node"`       // Target node name, e.g. "pve"
	Storage   string        `yaml:"storage"`    // Storage for CT disks, e.g. "local-lvm"
	Bridge    string        `yaml:"bridge"`     // Network bridge, e.g. "vmbr0"
	VMIDStart int           `yaml:"vmid_start"` // Start of VMID range for sandboxes
	VMIDEnd   int           `yaml:"vmid_end"`   // End of VMID range for sandboxes
	VerifySSL bool          `yaml:"verify_ssl"` // Verify TLS certificates
	Timeout   time.Duration `yaml:"timeout"`    // HTTP client timeout
}

// Validate checks that required config fields are set and applies defaults.
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("lxc host is required")
	}
	if c.TokenID == "" {
		return fmt.Errorf("lxc token_id is required")
	}
	if c.Secret == "" {
		return fmt.Errorf("lxc secret is required")
	}
	if c.Node == "" {
		return fmt.Errorf("lxc node is required")
	}
	if c.VMIDStart <= 0 {
		c.VMIDStart = 9000
	}
	if c.VMIDEnd <= 0 {
		c.VMIDEnd = 9999
	}
	if c.VMIDEnd <= c.VMIDStart {
		return fmt.Errorf("lxc vmid_end (%d) must be greater than vmid_start (%d)", c.VMIDEnd, c.VMIDStart)
	}
	if c.Timeout == 0 {
		c.Timeout = 5 * time.Minute
	}
	if c.Bridge == "" {
		c.Bridge = "vmbr0"
	}
	return nil
}
