package sandbox

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// RemoteService implements Service by calling the deer-daemon via gRPC.
// Source VM resolution (which hypervisor host owns which VM) is handled
// by the daemon via its configured source_hosts. The CLI just sends VM names.
type RemoteService struct {
	conn   *grpc.ClientConn
	client deerv1.DaemonServiceClient
}

// NewRemoteService dials the daemon gRPC endpoint and returns a Service.
// It uses TLS configuration from the ControlPlaneConfig:
//   - If DaemonCAFile is set, use it to verify the daemon's TLS cert
//   - If DaemonInsecure is false and no CA file, use the system cert pool
//   - Only use insecure credentials when DaemonInsecure is explicitly true
func NewRemoteService(addr string, cpCfg config.ControlPlaneConfig) (*RemoteService, error) {
	var creds credentials.TransportCredentials

	switch {
	case cpCfg.DaemonCAFile != "":
		// Use the specified CA certificate
		caCert, err := os.ReadFile(cpCfg.DaemonCAFile)
		if err != nil {
			return nil, fmt.Errorf("read daemon CA file %s: %w", cpCfg.DaemonCAFile, err)
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse daemon CA certificate from %s", cpCfg.DaemonCAFile)
		}
		creds = credentials.NewTLS(&tls.Config{
			RootCAs:    certPool,
			MinVersion: tls.VersionTLS12,
		})

	case cpCfg.DaemonInsecure:
		// Explicitly insecure - no TLS
		creds = insecure.NewCredentials()

	default:
		// Use system cert pool
		creds = credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})
	}

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("dial daemon at %s: %w", addr, err)
	}
	return &RemoteService{
		conn:   conn,
		client: deerv1.NewDaemonServiceClient(conn),
	}, nil
}

func (r *RemoteService) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

func (r *RemoteService) CreateSandbox(ctx context.Context, req CreateRequest) (*SandboxInfo, error) {
	resp, err := r.client.CreateSandbox(ctx, &deerv1.CreateSandboxCommand{
		BaseImage:                 req.SourceVM,
		SourceVm:                  req.SourceVM,
		Name:                      req.Name,
		Vcpus:                     int32(req.VCPUs),
		MemoryMb:                  int32(req.MemoryMB),
		TtlSeconds:                int32(req.TTLSeconds),
		AgentId:                   req.AgentID,
		Network:                   req.Network,
		Live:                      req.Live,
		SimpleKafkaBroker:         req.SimpleKafkaBroker,
		SimpleElasticsearchBroker: req.SimpleElasticsearchBroker,
	})
	if err != nil {
		return nil, err
	}
	return &SandboxInfo{
		ID:        resp.GetSandboxId(),
		Name:      resp.GetName(),
		State:     resp.GetState(),
		IPAddress: resp.GetIpAddress(),
	}, nil
}

func (r *RemoteService) CreateSandboxStream(ctx context.Context, req CreateRequest, onProgress func(step string, stepNum, total int)) (*SandboxInfo, error) {
	stream, err := r.client.CreateSandboxStream(ctx, &deerv1.CreateSandboxCommand{
		BaseImage:                 req.SourceVM,
		SourceVm:                  req.SourceVM,
		Name:                      req.Name,
		Vcpus:                     int32(req.VCPUs),
		MemoryMb:                  int32(req.MemoryMB),
		TtlSeconds:                int32(req.TTLSeconds),
		AgentId:                   req.AgentID,
		Network:                   req.Network,
		Live:                      req.Live,
		SimpleKafkaBroker:         req.SimpleKafkaBroker,
		SimpleElasticsearchBroker: req.SimpleElasticsearchBroker,
	})
	if err != nil {
		// Fall back to unary if streaming is unimplemented (older daemon)
		if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
			if onProgress != nil {
				onProgress("Creating sandbox", 1, 9)
			}
			return r.CreateSandbox(ctx, req)
		}
		return nil, err
	}

	for {
		progress, err := stream.Recv()
		if err != nil {
			return nil, err
		}

		if progress.GetError() != "" {
			return nil, fmt.Errorf("sandbox creation failed: %s", progress.GetError())
		}

		if progress.GetDone() {
			result := progress.GetResult()
			return &SandboxInfo{
				ID:        result.GetSandboxId(),
				Name:      result.GetName(),
				State:     result.GetState(),
				IPAddress: result.GetIpAddress(),
			}, nil
		}

		if onProgress != nil {
			onProgress(progress.GetStep(), int(progress.GetStepNum()), int(progress.GetTotalSteps()))
		}
	}
}

func (r *RemoteService) GetSandbox(ctx context.Context, id string) (*SandboxInfo, error) {
	resp, err := r.client.GetSandbox(ctx, &deerv1.GetSandboxRequest{SandboxId: id})
	if err != nil {
		return nil, err
	}
	return protoToSandboxInfo(resp), nil
}

func (r *RemoteService) ListSandboxes(ctx context.Context) ([]*SandboxInfo, error) {
	resp, err := r.client.ListSandboxes(ctx, &deerv1.ListSandboxesRequest{})
	if err != nil {
		return nil, err
	}
	result := make([]*SandboxInfo, 0, len(resp.GetSandboxes()))
	for _, sb := range resp.GetSandboxes() {
		result = append(result, protoToSandboxInfo(sb))
	}
	return result, nil
}

func (r *RemoteService) DestroySandbox(ctx context.Context, id string) error {
	_, err := r.client.DestroySandbox(ctx, &deerv1.DestroySandboxCommand{SandboxId: id})
	return err
}

func (r *RemoteService) StartSandbox(ctx context.Context, id string) (*SandboxInfo, error) {
	resp, err := r.client.StartSandbox(ctx, &deerv1.StartSandboxCommand{SandboxId: id})
	if err != nil {
		return nil, err
	}
	return &SandboxInfo{
		ID:        resp.GetSandboxId(),
		State:     resp.GetState(),
		IPAddress: resp.GetIpAddress(),
	}, nil
}

func (r *RemoteService) StopSandbox(ctx context.Context, id string, force bool) error {
	_, err := r.client.StopSandbox(ctx, &deerv1.StopSandboxCommand{SandboxId: id, Force: force})
	return err
}

func (r *RemoteService) RunCommand(ctx context.Context, sandboxID, command string, timeoutSec int, env map[string]string) (*CommandResult, error) {
	resp, err := r.client.RunCommand(ctx, &deerv1.RunCommandCommand{
		SandboxId:      sandboxID,
		Command:        command,
		TimeoutSeconds: int32(timeoutSec),
		Env:            env,
	})
	if err != nil {
		return nil, err
	}
	return &CommandResult{
		SandboxID:  resp.GetSandboxId(),
		Stdout:     resp.GetStdout(),
		Stderr:     resp.GetStderr(),
		ExitCode:   int(resp.GetExitCode()),
		DurationMS: resp.GetDurationMs(),
	}, nil
}

func (r *RemoteService) CreateSnapshot(ctx context.Context, sandboxID, name string) (*SnapshotInfo, error) {
	resp, err := r.client.CreateSnapshot(ctx, &deerv1.SnapshotCommand{
		SandboxId:    sandboxID,
		SnapshotName: name,
	})
	if err != nil {
		return nil, err
	}
	return &SnapshotInfo{
		SnapshotID:   resp.GetSnapshotId(),
		SnapshotName: resp.GetSnapshotName(),
		SandboxID:    resp.GetSandboxId(),
	}, nil
}

func (r *RemoteService) ListVMs(ctx context.Context) ([]*VMInfo, error) {
	resp, err := r.client.ListSourceVMs(ctx, &deerv1.ListSourceVMsCommand{})
	if err != nil {
		return nil, err
	}
	result := make([]*VMInfo, 0, len(resp.GetVms()))
	for _, vm := range resp.GetVms() {
		result = append(result, &VMInfo{
			Name:      vm.GetName(),
			State:     vm.GetState(),
			IPAddress: vm.GetIpAddress(),
			Prepared:  vm.GetPrepared(),
		})
	}
	return result, nil
}

func (r *RemoteService) ValidateSourceVM(ctx context.Context, vmName string) (*ValidationInfo, error) {
	resp, err := r.client.ValidateSourceVM(ctx, &deerv1.ValidateSourceVMCommand{
		SourceVm: vmName,
	})
	if err != nil {
		return nil, err
	}
	return &ValidationInfo{
		VMName:     resp.GetSourceVm(),
		Valid:      resp.GetValid(),
		State:      resp.GetState(),
		MACAddress: resp.GetMacAddress(),
		IPAddress:  resp.GetIpAddress(),
		HasNetwork: resp.GetHasNetwork(),
		Warnings:   resp.GetWarnings(),
		Errors:     resp.GetErrors(),
	}, nil
}

func (r *RemoteService) PrepareSourceVM(ctx context.Context, vmName, sshUser, keyPath string) (*PrepareInfo, error) {
	resp, err := r.client.PrepareSourceVM(ctx, &deerv1.PrepareSourceVMCommand{
		SourceVm:   vmName,
		SshUser:    sshUser,
		SshKeyPath: keyPath,
	})
	if err != nil {
		return nil, err
	}
	return &PrepareInfo{
		SourceVM:          resp.GetSourceVm(),
		IPAddress:         resp.GetIpAddress(),
		Prepared:          resp.GetPrepared(),
		UserCreated:       resp.GetUserCreated(),
		ShellInstalled:    resp.GetShellInstalled(),
		CAKeyInstalled:    resp.GetCaKeyInstalled(),
		SSHDConfigured:    resp.GetSshdConfigured(),
		PrincipalsCreated: resp.GetPrincipalsCreated(),
		SSHDRestarted:     resp.GetSshdRestarted(),
	}, nil
}

func (r *RemoteService) RunSourceCommand(ctx context.Context, vmName, command string, timeoutSec int) (*SourceCommandResult, error) {
	resp, err := r.client.RunSourceCommand(ctx, &deerv1.RunSourceCommandCommand{
		SourceVm:       vmName,
		Command:        command,
		TimeoutSeconds: int32(timeoutSec),
	})
	if err != nil {
		return nil, err
	}
	return &SourceCommandResult{
		SourceVM: resp.GetSourceVm(),
		ExitCode: int(resp.GetExitCode()),
		Stdout:   resp.GetStdout(),
		Stderr:   resp.GetStderr(),
	}, nil
}

func (r *RemoteService) ReadSourceFile(ctx context.Context, vmName, path string) (string, error) {
	resp, err := r.client.ReadSourceFile(ctx, &deerv1.ReadSourceFileCommand{
		SourceVm: vmName,
		Path:     path,
	})
	if err != nil {
		return "", err
	}
	return resp.GetContent(), nil
}

func (r *RemoteService) GetHostInfo(ctx context.Context) (*HostInfo, error) {
	resp, err := r.client.GetHostInfo(ctx, &deerv1.GetHostInfoRequest{})
	if err != nil {
		return nil, err
	}
	var sourceHosts []SourceHostInfo
	for _, sh := range resp.GetSourceHosts() {
		sourceHosts = append(sourceHosts, SourceHostInfo{
			Address: sh.GetAddress(),
			SSHUser: sh.GetSshUser(),
			SSHPort: int(sh.GetSshPort()),
		})
	}

	return &HostInfo{
		HostID:            resp.GetHostId(),
		Hostname:          resp.GetHostname(),
		Version:           resp.GetVersion(),
		TotalCPUs:         int(resp.GetTotalCpus()),
		TotalMemoryMB:     resp.GetTotalMemoryMb(),
		ActiveSandboxes:   int(resp.GetActiveSandboxes()),
		BaseImages:        resp.GetBaseImages(),
		SSHCAPubKey:       resp.GetSshCaPubKey(),
		SSHIdentityPubKey: resp.GetSshIdentityPubKey(),
		SourceHosts:       sourceHosts,
	}, nil
}

func (r *RemoteService) Health(ctx context.Context) error {
	_, err := r.client.Health(ctx, &deerv1.HealthRequest{})
	return err
}

func (r *RemoteService) DoctorCheck(ctx context.Context) ([]DoctorCheckResult, error) {
	resp, err := r.client.DoctorCheck(ctx, &deerv1.DoctorCheckRequest{})
	if err != nil {
		return nil, err
	}
	results := make([]DoctorCheckResult, len(resp.GetResults()))
	for i, res := range resp.GetResults() {
		results[i] = DoctorCheckResult{
			Name:     res.GetName(),
			Category: res.GetCategory(),
			Passed:   res.GetPassed(),
			Message:  res.GetMessage(),
			FixCmd:   res.GetFixCmd(),
		}
	}
	return results, nil
}

func (r *RemoteService) ScanSourceHostKeys(ctx context.Context) ([]ScanSourceHostKeysResult, error) {
	resp, err := r.client.ScanSourceHostKeys(ctx, &deerv1.ScanSourceHostKeysRequest{})
	if err != nil {
		return nil, err
	}
	results := make([]ScanSourceHostKeysResult, len(resp.GetResults()))
	for i, res := range resp.GetResults() {
		results[i] = ScanSourceHostKeysResult{
			Address: res.GetAddress(),
			Success: res.GetSuccess(),
			Error:   res.GetError(),
		}
	}
	return results, nil
}

// protoToSandboxInfo converts a proto SandboxInfo to the canonical type.
func protoToSandboxInfo(pb *deerv1.SandboxInfo) *SandboxInfo {
	var createdAt time.Time
	if pb.GetCreatedAt() != "" {
		createdAt, _ = time.Parse(time.RFC3339, pb.GetCreatedAt())
	}
	return &SandboxInfo{
		ID:        pb.GetSandboxId(),
		Name:      pb.GetName(),
		State:     pb.GetState(),
		IPAddress: pb.GetIpAddress(),
		BaseImage: pb.GetBaseImage(),
		AgentID:   pb.GetAgentId(),
		VCPUs:     int(pb.GetVcpus()),
		MemoryMB:  int(pb.GetMemoryMb()),
		CreatedAt: createdAt,
	}
}
