package sandbox

import "context"

// Service abstracts sandbox operations for the CLI, MCP, and TUI layers.
// Implementations may call a remote daemon via gRPC or operate locally.
type Service interface {
	// Sandbox lifecycle
	CreateSandbox(ctx context.Context, req CreateRequest) (*SandboxInfo, error)
	GetSandbox(ctx context.Context, id string) (*SandboxInfo, error)
	ListSandboxes(ctx context.Context) ([]*SandboxInfo, error)
	DestroySandbox(ctx context.Context, id string) error
	StartSandbox(ctx context.Context, id string) (*SandboxInfo, error)
	StopSandbox(ctx context.Context, id string, force bool) error

	// Command execution
	RunCommand(ctx context.Context, sandboxID, command string, timeoutSec int, env map[string]string) (*CommandResult, error)

	// Snapshots
	CreateSnapshot(ctx context.Context, sandboxID, name string) (*SnapshotInfo, error)

	// Source VM operations
	ListVMs(ctx context.Context) ([]*VMInfo, error)
	ValidateSourceVM(ctx context.Context, vmName string) (*ValidationInfo, error)
	PrepareSourceVM(ctx context.Context, vmName, sshUser, keyPath string) (*PrepareInfo, error)
	RunSourceCommand(ctx context.Context, vmName, command string, timeoutSec int) (*SourceCommandResult, error)
	ReadSourceFile(ctx context.Context, vmName, path string) (string, error)

	// Host info
	GetHostInfo(ctx context.Context) (*HostInfo, error)
	Health(ctx context.Context) error

	// Close releases resources (e.g. gRPC connection).
	Close() error
}
