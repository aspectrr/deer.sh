// Package provider defines the SandboxProvider interface that abstracts
// sandbox lifecycle management across different backends (microVM, LXC).
package provider

import (
	"context"
	"time"
)

// SandboxProvider abstracts sandbox lifecycle management.
// Implementations handle the details of creating, managing, and executing
// commands in sandboxes - whether they are QEMU microVMs or LXC containers.
type SandboxProvider interface {
	// Sandbox lifecycle
	CreateSandbox(ctx context.Context, req CreateRequest) (*SandboxResult, error)
	DestroySandbox(ctx context.Context, sandboxID string) error
	StartSandbox(ctx context.Context, sandboxID string) (*SandboxResult, error)
	StopSandbox(ctx context.Context, sandboxID string, force bool) error
	GetSandboxIP(ctx context.Context, sandboxID string) (string, error)
	CreateSnapshot(ctx context.Context, sandboxID, name string) (*SnapshotResult, error)

	// Command execution (SSH for microvm, pct exec for lxc)
	RunCommand(ctx context.Context, sandboxID, command string, timeout time.Duration) (*CommandResult, error)

	// Template/image listing for registration
	ListTemplates(ctx context.Context) ([]string, error)

	// Source VM/CT operations
	ListSourceVMs(ctx context.Context) ([]SourceVMInfo, error)
	ValidateSourceVM(ctx context.Context, vmName string) (*ValidationResult, error)
	PrepareSourceVM(ctx context.Context, vmName, sshUser, sshKeyPath string) (*PrepareResult, error)
	RunSourceCommand(ctx context.Context, vmName, command string, timeout time.Duration) (*CommandResult, error)
	ReadSourceFile(ctx context.Context, vmName, path string) (string, error)

	// Registration capabilities
	Capabilities(ctx context.Context) (*HostCapabilities, error)

	// Active sandbox count for heartbeat
	ActiveSandboxCount() int

	// Recovery after restart
	RecoverState(ctx context.Context) error
}

// CreateRequest holds parameters for creating a sandbox.
type CreateRequest struct {
	SandboxID    string
	Name         string
	BaseImage    string // QCOW2 name (microvm) or CT template name (lxc)
	SourceVM     string // for bridge resolution (microvm) or clone source (lxc)
	Network      string // bridge override
	VCPUs        int
	MemoryMB     int
	TTLSeconds   int
	AgentID      string
	SSHPublicKey string
}

// SandboxResult holds the result of a sandbox lifecycle operation.
type SandboxResult struct {
	SandboxID  string
	Name       string
	State      string // RUNNING, STOPPED, etc.
	IPAddress  string
	MACAddress string
	Bridge     string
	PID        int // QEMU PID (microvm) or 0 (lxc)
}

// SnapshotResult holds the result of a snapshot operation.
type SnapshotResult struct {
	SnapshotID   string
	SnapshotName string
}

// CommandResult holds the result of a command execution.
type CommandResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMS int64
}

// PrepareResult holds the outcome of preparing a source VM for read-only access.
type PrepareResult struct {
	SourceVM          string
	IPAddress         string
	Prepared          bool
	UserCreated       bool
	ShellInstalled    bool
	CAKeyInstalled    bool
	SSHDConfigured    bool
	PrincipalsCreated bool
	SSHDRestarted     bool
}

// SourceVMInfo describes a source VM/CT visible to the provider.
type SourceVMInfo struct {
	Name      string
	State     string
	IPAddress string
	Prepared  bool
}

// ValidationResult contains the result of validating a source VM.
type ValidationResult struct {
	VMName     string
	Valid      bool
	State      string
	MACAddress string
	IPAddress  string
	HasNetwork bool
	Warnings   []string
	Errors     []string
}

// HostCapabilities describes the resources and images available on this host.
type HostCapabilities struct {
	TotalCPUs       int
	AvailableCPUs   int
	TotalMemoryMB   int
	AvailableMemMB  int
	TotalDiskMB     int
	AvailableDiskMB int
	BaseImages      []string
	Bridges         []string
}
