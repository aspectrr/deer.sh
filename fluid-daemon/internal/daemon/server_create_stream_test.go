package daemon

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/config"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/snapshotpull"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/state"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/telemetry"
	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
	"google.golang.org/grpc/metadata"
)

type fakeCreateSandboxProvider struct {
	createWithProgressFn func(context.Context, provider.CreateRequest, func(string, int, int)) (*provider.SandboxResult, error)
}

func (f *fakeCreateSandboxProvider) CreateSandbox(_ context.Context, _ provider.CreateRequest) (*provider.SandboxResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) DestroySandbox(context.Context, string) error {
	return nil
}

func (f *fakeCreateSandboxProvider) StartSandbox(context.Context, string) (*provider.SandboxResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) StopSandbox(context.Context, string, bool) error {
	return nil
}

func (f *fakeCreateSandboxProvider) GetSandboxIP(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) CreateSnapshot(context.Context, string, string) (*provider.SnapshotResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) RunCommand(context.Context, string, string, time.Duration) (*provider.CommandResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) ListTemplates(context.Context) ([]string, error) {
	return nil, nil
}

func (f *fakeCreateSandboxProvider) ListSourceVMs(context.Context) ([]provider.SourceVMInfo, error) {
	return nil, nil
}

func (f *fakeCreateSandboxProvider) ValidateSourceVM(context.Context, string) (*provider.ValidationResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) PrepareSourceVM(context.Context, string, string, string) (*provider.PrepareResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) RunSourceCommand(context.Context, string, string, time.Duration) (*provider.CommandResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) ReadSourceFile(context.Context, string, string) (string, error) {
	return "", errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) Capabilities(context.Context) (*provider.HostCapabilities, error) {
	return &provider.HostCapabilities{}, nil
}

func (f *fakeCreateSandboxProvider) ActiveSandboxCount() int {
	return 0
}

func (f *fakeCreateSandboxProvider) RecoverState(context.Context) error {
	return nil
}

func (f *fakeCreateSandboxProvider) CreateSandboxWithProgress(ctx context.Context, req provider.CreateRequest, progress func(string, int, int)) (*provider.SandboxResult, error) {
	if f.createWithProgressFn != nil {
		return f.createWithProgressFn(ctx, req, progress)
	}
	return nil, errors.New("not implemented")
}

type fakeCreateSandboxPuller struct {
	result *snapshotpull.PullResult
	err    error
}

func (f *fakeCreateSandboxPuller) Pull(context.Context, snapshotpull.PullRequest, snapshotpull.SnapshotBackend) (*snapshotpull.PullResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

type fakeCreateSandboxStream struct {
	ctx  context.Context
	msgs []*fluidv1.SandboxProgress
}

func (f *fakeCreateSandboxStream) Send(msg *fluidv1.SandboxProgress) error {
	f.msgs = append(f.msgs, msg)
	return nil
}

func (f *fakeCreateSandboxStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeCreateSandboxStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeCreateSandboxStream) SetTrailer(metadata.MD)       {}
func (f *fakeCreateSandboxStream) Context() context.Context {
	if f.ctx != nil {
		return f.ctx
	}
	return context.Background()
}
func (f *fakeCreateSandboxStream) SendMsg(any) error { return nil }
func (f *fakeCreateSandboxStream) RecvMsg(any) error { return nil }

func newTestCreateSandboxServer(t *testing.T, prov provider.SandboxProvider, puller sandboxCreatePuller, cfg *config.Config) *Server {
	t.Helper()

	if cfg == nil {
		cfg = &config.Config{}
	}
	store, err := state.NewStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &Server{
		cfg:         cfg,
		prov:        prov,
		store:       store,
		puller:      puller,
		telemetry:   telemetry.NewNoopService(),
		logger:      logger,
		vmHostCache: make(map[string]*fluidv1.SourceHostConnection),
	}
}

func TestCreateSandboxStream_EmitsEndToEndProgress(t *testing.T) {
	prov := &fakeCreateSandboxProvider{
		createWithProgressFn: func(_ context.Context, req provider.CreateRequest, progress func(string, int, int)) (*provider.SandboxResult, error) {
			if req.BaseImage != "snap-host1-vm-1" {
				t.Fatalf("BaseImage = %q, want pulled image", req.BaseImage)
			}
			steps := []string{
				"Resolving network bridge",
				"Creating overlay disk",
				"Generating cloud-init",
				"Setting up network (TAP)",
				"Booting microVM",
				"Discovering IP address",
				"Waiting for cloud-init ready",
			}
			for i, step := range steps {
				progress(step, i+1, len(steps))
			}
			return &provider.SandboxResult{
				SandboxID:  "sbx-test",
				Name:       "sandbox",
				State:      "RUNNING",
				IPAddress:  "10.0.0.2",
				MACAddress: "52:54:00:12:34:56",
				Bridge:     "br0",
				PID:        1234,
			}, nil
		},
	}
	puller := &fakeCreateSandboxPuller{
		result: &snapshotpull.PullResult{ImageName: "snap-host1-vm-1", Cached: false},
	}
	server := newTestCreateSandboxServer(t, prov, puller, &config.Config{})
	stream := &fakeCreateSandboxStream{}

	err := server.CreateSandboxStream(&fluidv1.CreateSandboxCommand{
		SourceVm: "vm-1",
		SourceHostConnection: &fluidv1.SourceHostConnection{
			Type:    "libvirt",
			SshHost: "host1",
			SshPort: 22,
			SshUser: "fluid-daemon",
		},
		SnapshotMode: fluidv1.SnapshotMode_SNAPSHOT_MODE_FRESH,
	}, stream)
	if err != nil {
		t.Fatalf("CreateSandboxStream: %v", err)
	}

	if len(stream.msgs) != 10 {
		t.Fatalf("message count = %d, want 10", len(stream.msgs))
	}

	expectedSteps := []string{
		"Using provided source host",
		"Pulling fresh snapshot",
		"Resolving network bridge",
		"Creating overlay disk",
		"Generating cloud-init",
		"Setting up network (TAP)",
		"Booting microVM",
		"Discovering IP address",
		"Waiting for cloud-init ready",
	}
	for i, expected := range expectedSteps {
		msg := stream.msgs[i]
		if msg.GetStepNum() != int32(i+1) {
			t.Fatalf("step %d number = %d, want %d", i, msg.GetStepNum(), i+1)
		}
		if msg.GetTotalSteps() != createSandboxStreamTotalSteps {
			t.Fatalf("step %d total = %d, want %d", i, msg.GetTotalSteps(), createSandboxStreamTotalSteps)
		}
		if msg.GetStep() != expected {
			t.Fatalf("step %d label = %q, want %q", i+1, msg.GetStep(), expected)
		}
	}

	final := stream.msgs[len(stream.msgs)-1]
	if !final.GetDone() {
		t.Fatal("expected final message to mark stream done")
	}
	if final.GetResult().GetSandboxId() != "sbx-test" {
		t.Fatalf("final sandbox id = %q, want %q", final.GetResult().GetSandboxId(), "sbx-test")
	}
}

func TestCreateSandboxStream_NoPullPathEmitsNoOpSteps(t *testing.T) {
	prov := &fakeCreateSandboxProvider{
		createWithProgressFn: func(_ context.Context, _ provider.CreateRequest, progress func(string, int, int)) (*provider.SandboxResult, error) {
			steps := []string{
				"Resolving network bridge",
				"Creating overlay disk",
				"Generating cloud-init",
				"Setting up network (TAP)",
				"Booting microVM",
				"Discovering IP address",
				"Waiting for cloud-init ready",
			}
			for i, step := range steps {
				progress(step, i+1, len(steps))
			}
			return &provider.SandboxResult{
				SandboxID: "sbx-direct",
				Name:      "sandbox",
				State:     "RUNNING",
			}, nil
		},
	}
	server := newTestCreateSandboxServer(t, prov, nil, &config.Config{})
	stream := &fakeCreateSandboxStream{}

	err := server.CreateSandboxStream(&fluidv1.CreateSandboxCommand{
		BaseImage: "ubuntu-base",
	}, stream)
	if err != nil {
		t.Fatalf("CreateSandboxStream: %v", err)
	}

	if len(stream.msgs) < 3 {
		t.Fatalf("message count = %d, want at least 3", len(stream.msgs))
	}
	if stream.msgs[0].GetStep() != "No source host resolution needed" {
		t.Fatalf("step 1 label = %q, want %q", stream.msgs[0].GetStep(), "No source host resolution needed")
	}
	if stream.msgs[1].GetStep() != "Using requested base image" {
		t.Fatalf("step 2 label = %q, want %q", stream.msgs[1].GetStep(), "Using requested base image")
	}
	if !stream.msgs[len(stream.msgs)-1].GetDone() {
		t.Fatal("expected final done message")
	}
}
