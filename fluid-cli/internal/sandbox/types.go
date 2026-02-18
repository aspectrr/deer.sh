// Package sandbox defines the canonical types and service interface for
// sandbox operations. These types decouple the CLI/MCP/TUI layers from
// the transport (gRPC, local provider, etc.).
package sandbox

import "time"

// SandboxInfo contains details about a sandbox.
type SandboxInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	State     string    `json:"state"`
	IPAddress string    `json:"ip_address,omitempty"`
	BaseImage string    `json:"base_image"`
	AgentID   string    `json:"agent_id"`
	VCPUs     int       `json:"vcpus"`
	MemoryMB  int       `json:"memory_mb"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateRequest holds parameters for creating a sandbox.
type CreateRequest struct {
	SourceVM   string
	Name       string
	AgentID    string
	VCPUs      int
	MemoryMB   int
	TTLSeconds int
	Network    string
	Live       bool
}

// CommandResult holds the result of a command execution.
type CommandResult struct {
	SandboxID  string `json:"sandbox_id"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
}

// SnapshotInfo holds details about a created snapshot.
type SnapshotInfo struct {
	SnapshotID   string `json:"snapshot_id"`
	SnapshotName string `json:"snapshot_name"`
	SandboxID    string `json:"sandbox_id"`
}

// VMInfo describes a source VM available for cloning.
type VMInfo struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	IPAddress string `json:"ip_address,omitempty"`
	Prepared  bool   `json:"prepared"`
}

// ValidationInfo contains source VM validation results.
type ValidationInfo struct {
	VMName     string   `json:"vm_name"`
	Valid      bool     `json:"valid"`
	State      string   `json:"state"`
	MACAddress string   `json:"mac_address,omitempty"`
	IPAddress  string   `json:"ip_address,omitempty"`
	HasNetwork bool     `json:"has_network"`
	Warnings   []string `json:"warnings,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

// PrepareInfo contains the result of preparing a source VM.
type PrepareInfo struct {
	SourceVM          string `json:"source_vm"`
	IPAddress         string `json:"ip_address,omitempty"`
	Prepared          bool   `json:"prepared"`
	UserCreated       bool   `json:"user_created"`
	ShellInstalled    bool   `json:"shell_installed"`
	CAKeyInstalled    bool   `json:"ca_key_installed"`
	SSHDConfigured    bool   `json:"sshd_configured"`
	PrincipalsCreated bool   `json:"principals_created"`
	SSHDRestarted     bool   `json:"sshd_restarted"`
}

// SourceCommandResult holds the output of a source VM command.
type SourceCommandResult struct {
	SourceVM string `json:"source_vm"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// HostInfo contains host resource and capability information.
type HostInfo struct {
	HostID          string   `json:"host_id"`
	Hostname        string   `json:"hostname"`
	Version         string   `json:"version"`
	TotalCPUs       int      `json:"total_cpus"`
	TotalMemoryMB   int64    `json:"total_memory_mb"`
	ActiveSandboxes int      `json:"active_sandboxes"`
	BaseImages      []string `json:"base_images"`
}
