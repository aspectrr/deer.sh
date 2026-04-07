package sandbox

import (
	"context"
	"errors"
)

const noSandboxMsg = "no sandbox hosts configured, configure a sandbox host to create sandboxes, run commands, and edit files, daemon setup guide: https://deer.sh/docs/daemon"

// NoopService implements the Service interface but returns "not configured" for all operations.
// Used when no sandbox hosts are configured, allowing the CLI to still function
// with read-only source access.
type NoopService struct{}

// NewNoopService returns a sandbox service that gracefully rejects all operations.
func NewNoopService() *NoopService {
	return &NoopService{}
}

func (n *NoopService) CreateSandbox(ctx context.Context, req CreateRequest) (*SandboxInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) CreateSandboxStream(ctx context.Context, req CreateRequest, onProgress func(step string, stepNum, total int)) (*SandboxInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) GetSandbox(ctx context.Context, id string) (*SandboxInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) ListSandboxes(ctx context.Context) ([]*SandboxInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) DestroySandbox(ctx context.Context, id string) error {
	return errors.New(noSandboxMsg)
}

func (n *NoopService) StartSandbox(ctx context.Context, id string) (*SandboxInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) StopSandbox(ctx context.Context, id string, force bool) error {
	return errors.New(noSandboxMsg)
}

func (n *NoopService) RunCommand(ctx context.Context, sandboxID, command string, timeoutSec int, env map[string]string) (*CommandResult, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) CreateSnapshot(ctx context.Context, sandboxID, name string) (*SnapshotInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) ListVMs(ctx context.Context) ([]*VMInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) ValidateSourceVM(ctx context.Context, vmName string) (*ValidationInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) PrepareSourceVM(ctx context.Context, vmName, sshUser, keyPath string) (*PrepareInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) RunSourceCommand(ctx context.Context, vmName, command string, timeoutSec int) (*SourceCommandResult, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) ReadSourceFile(ctx context.Context, vmName, path string) (string, error) {
	return "", errors.New(noSandboxMsg)
}

func (n *NoopService) GetHostInfo(ctx context.Context) (*HostInfo, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) Health(ctx context.Context) error {
	return errors.New(noSandboxMsg)
}

func (n *NoopService) DoctorCheck(ctx context.Context) ([]DoctorCheckResult, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) ScanSourceHostKeys(ctx context.Context) ([]ScanSourceHostKeysResult, error) {
	return nil, errors.New(noSandboxMsg)
}

func (n *NoopService) Close() error {
	return nil
}
