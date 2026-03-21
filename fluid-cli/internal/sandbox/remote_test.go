package sandbox

import (
	"context"
	"io"
	"testing"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// mockDaemonClient implements fluidv1.DaemonServiceClient for testing.
type mockDaemonClient struct {
	vms               []*fluidv1.SourceVMListEntry
	createSandboxResp *fluidv1.SandboxCreated
	createSandboxErr  error
	createStream      grpc.ServerStreamingClient[fluidv1.SandboxProgress]
	createStreamErr   error
}

func (m *mockDaemonClient) ListSourceVMs(_ context.Context, _ *fluidv1.ListSourceVMsCommand, _ ...grpc.CallOption) (*fluidv1.SourceVMsList, error) {
	return &fluidv1.SourceVMsList{Vms: m.vms}, nil
}

// Stubs for the rest of the interface.

func (m *mockDaemonClient) CreateSandbox(context.Context, *fluidv1.CreateSandboxCommand, ...grpc.CallOption) (*fluidv1.SandboxCreated, error) {
	if m.createSandboxErr != nil {
		return nil, m.createSandboxErr
	}
	if m.createSandboxResp != nil {
		return m.createSandboxResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) GetSandbox(context.Context, *fluidv1.GetSandboxRequest, ...grpc.CallOption) (*fluidv1.SandboxInfo, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) ListSandboxes(context.Context, *fluidv1.ListSandboxesRequest, ...grpc.CallOption) (*fluidv1.ListSandboxesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) DestroySandbox(context.Context, *fluidv1.DestroySandboxCommand, ...grpc.CallOption) (*fluidv1.SandboxDestroyed, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) StartSandbox(context.Context, *fluidv1.StartSandboxCommand, ...grpc.CallOption) (*fluidv1.SandboxStarted, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) StopSandbox(context.Context, *fluidv1.StopSandboxCommand, ...grpc.CallOption) (*fluidv1.SandboxStopped, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) RunCommand(context.Context, *fluidv1.RunCommandCommand, ...grpc.CallOption) (*fluidv1.CommandResult, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) CreateSnapshot(context.Context, *fluidv1.SnapshotCommand, ...grpc.CallOption) (*fluidv1.SnapshotCreated, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) ValidateSourceVM(context.Context, *fluidv1.ValidateSourceVMCommand, ...grpc.CallOption) (*fluidv1.SourceVMValidation, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) PrepareSourceVM(context.Context, *fluidv1.PrepareSourceVMCommand, ...grpc.CallOption) (*fluidv1.SourceVMPrepared, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) RunSourceCommand(context.Context, *fluidv1.RunSourceCommandCommand, ...grpc.CallOption) (*fluidv1.SourceCommandResult, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) ReadSourceFile(context.Context, *fluidv1.ReadSourceFileCommand, ...grpc.CallOption) (*fluidv1.SourceFileResult, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) GetHostInfo(context.Context, *fluidv1.GetHostInfoRequest, ...grpc.CallOption) (*fluidv1.HostInfoResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) Health(context.Context, *fluidv1.HealthRequest, ...grpc.CallOption) (*fluidv1.HealthResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) DiscoverHosts(context.Context, *fluidv1.DiscoverHostsCommand, ...grpc.CallOption) (*fluidv1.DiscoverHostsResult, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) DoctorCheck(context.Context, *fluidv1.DoctorCheckRequest, ...grpc.CallOption) (*fluidv1.DoctorCheckResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDaemonClient) CreateSandboxStream(_ context.Context, _ *fluidv1.CreateSandboxCommand, _ ...grpc.CallOption) (grpc.ServerStreamingClient[fluidv1.SandboxProgress], error) {
	if m.createStreamErr != nil {
		return nil, m.createStreamErr
	}
	if m.createStream != nil {
		return m.createStream, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

type fakeSandboxProgressStream struct {
	msgs []*fluidv1.SandboxProgress
	idx  int
}

func (f *fakeSandboxProgressStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeSandboxProgressStream) Trailer() metadata.MD         { return nil }
func (f *fakeSandboxProgressStream) CloseSend() error             { return nil }
func (f *fakeSandboxProgressStream) Context() context.Context     { return context.Background() }
func (f *fakeSandboxProgressStream) SendMsg(any) error            { return nil }
func (f *fakeSandboxProgressStream) RecvMsg(any) error            { return nil }

func (f *fakeSandboxProgressStream) Recv() (*fluidv1.SandboxProgress, error) {
	if f.idx >= len(f.msgs) {
		return nil, io.EOF
	}
	msg := f.msgs[f.idx]
	f.idx++
	return msg, nil
}

func (m *mockDaemonClient) ScanSourceHostKeys(_ context.Context, _ *fluidv1.ScanSourceHostKeysRequest, _ ...grpc.CallOption) (*fluidv1.ScanSourceHostKeysResponse, error) {
	return &fluidv1.ScanSourceHostKeysResponse{
		Results: []*fluidv1.ScanSourceHostKeysResult{
			{Address: "10.0.0.1", Success: true},
		},
	}, nil
}

func TestListVMs_DelegatesToDaemon(t *testing.T) {
	mock := &mockDaemonClient{
		vms: []*fluidv1.SourceVMListEntry{
			{Name: "vm-a", State: "running", Host: "10.0.0.1"},
			{Name: "vm-b", State: "stopped", Host: "10.0.0.2"},
		},
	}
	svc := &RemoteService{client: mock}

	vms, err := svc.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("ListVMs: %v", err)
	}
	if len(vms) != 2 {
		t.Fatalf("got %d VMs, want 2", len(vms))
	}
	if vms[0].Name != "vm-a" {
		t.Errorf("got VM[0] %q, want vm-a", vms[0].Name)
	}
	if vms[1].Name != "vm-b" {
		t.Errorf("got VM[1] %q, want vm-b", vms[1].Name)
	}
}

func TestScanSourceHostKeys_DelegatesToDaemon(t *testing.T) {
	mock := &mockDaemonClient{}
	svc := &RemoteService{client: mock}

	results, err := svc.ScanSourceHostKeys(context.Background())
	if err != nil {
		t.Fatalf("ScanSourceHostKeys: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Address != "10.0.0.1" {
		t.Errorf("got address %q, want 10.0.0.1", results[0].Address)
	}
	if !results[0].Success {
		t.Error("expected success=true")
	}
}

func TestCreateSandboxStream_DelegatesProgressToCallback(t *testing.T) {
	mock := &mockDaemonClient{
		createStream: &fakeSandboxProgressStream{
			msgs: []*fluidv1.SandboxProgress{
				{Step: "Using provided source host", StepNum: 1, TotalSteps: 9},
				{Step: "Pulling fresh snapshot", StepNum: 2, TotalSteps: 9},
				{
					Done: true,
					Result: &fluidv1.SandboxCreated{
						SandboxId: "sbx-123",
						Name:      "sandbox",
						State:     "RUNNING",
						IpAddress: "10.0.0.2",
					},
				},
			},
		},
	}
	svc := &RemoteService{client: mock}
	var progressSteps []string

	info, err := svc.CreateSandboxStream(context.Background(), CreateRequest{
		SourceVM: "vm-1",
	}, func(step string, stepNum, total int) {
		progressSteps = append(progressSteps, step)
		if total != 9 {
			t.Fatalf("progress total = %d, want 9", total)
		}
		if stepNum != len(progressSteps) {
			t.Fatalf("progress step number = %d, want %d", stepNum, len(progressSteps))
		}
	})
	if err != nil {
		t.Fatalf("CreateSandboxStream: %v", err)
	}
	if info.ID != "sbx-123" {
		t.Fatalf("sandbox id = %q, want %q", info.ID, "sbx-123")
	}
	if len(progressSteps) != 2 {
		t.Fatalf("progress step count = %d, want 2", len(progressSteps))
	}
	if progressSteps[0] != "Using provided source host" || progressSteps[1] != "Pulling fresh snapshot" {
		t.Fatalf("progress steps = %v, want provided source host then snapshot pull", progressSteps)
	}
}

func TestCreateSandboxStream_FallsBackToUnaryWithSyntheticProgress(t *testing.T) {
	mock := &mockDaemonClient{
		createStreamErr: status.Error(codes.Unimplemented, "not implemented"),
		createSandboxResp: &fluidv1.SandboxCreated{
			SandboxId: "sbx-legacy",
			Name:      "sandbox",
			State:     "RUNNING",
			IpAddress: "10.0.0.9",
		},
	}
	svc := &RemoteService{client: mock}
	var progress [][]any

	info, err := svc.CreateSandboxStream(context.Background(), CreateRequest{
		SourceVM: "vm-1",
	}, func(step string, stepNum, total int) {
		progress = append(progress, []any{step, stepNum, total})
	})
	if err != nil {
		t.Fatalf("CreateSandboxStream: %v", err)
	}
	if info.ID != "sbx-legacy" {
		t.Fatalf("sandbox id = %q, want %q", info.ID, "sbx-legacy")
	}
	if len(progress) != 1 {
		t.Fatalf("progress message count = %d, want 1", len(progress))
	}
	if progress[0][0] != "Creating sandbox" || progress[0][1] != 1 || progress[0][2] != 9 {
		t.Fatalf("synthetic progress = %v, want [Creating sandbox 1 9]", progress[0])
	}
}
