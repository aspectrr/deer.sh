package daemon

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/config"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/provider"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/snapshotpull"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/state"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/telemetry"
	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"
	"google.golang.org/grpc/metadata"
)

type fakeCreateSandboxProvider struct {
	createFn             func(context.Context, provider.CreateRequest) (*provider.SandboxResult, error)
	createWithProgressFn func(context.Context, provider.CreateRequest, func(string, int, int)) (*provider.SandboxResult, error)
	destroyFn            func(context.Context, string) error
	destroyed            []string
}

func (f *fakeCreateSandboxProvider) CreateSandbox(ctx context.Context, req provider.CreateRequest) (*provider.SandboxResult, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (f *fakeCreateSandboxProvider) DestroySandbox(ctx context.Context, sandboxID string) error {
	f.destroyed = append(f.destroyed, sandboxID)
	if f.destroyFn != nil {
		return f.destroyFn(ctx, sandboxID)
	}
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
	msgs []*deerv1.SandboxProgress
}

func (f *fakeCreateSandboxStream) Send(msg *deerv1.SandboxProgress) error {
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
		vmHostCache: make(map[string]*deerv1.SourceHostConnection),
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

	err := server.CreateSandboxStream(&deerv1.CreateSandboxCommand{
		SourceVm: "vm-1",
		SourceHostConnection: &deerv1.SourceHostConnection{
			Type:    "libvirt",
			SshHost: "host1",
			SshPort: 22,
			SshUser: "deer-daemon",
		},
		SnapshotMode: deerv1.SnapshotMode_SNAPSHOT_MODE_FRESH,
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

func TestCreateSandboxStream_PassesGenericKafkaDataSourcesToProvider(t *testing.T) {
	prov := &fakeCreateSandboxProvider{
		createWithProgressFn: func(_ context.Context, req provider.CreateRequest, progress func(string, int, int)) (*provider.SandboxResult, error) {
			if len(req.DataSources) != 1 {
				t.Fatalf("DataSources len = %d, want 1", len(req.DataSources))
			}
			ds := req.DataSources[0]
			if ds.Type != provider.DataSourceTypeKafka {
				t.Fatalf("data source type = %q, want kafka", ds.Type)
			}
			if ds.ConfigRef != "cfg-1" {
				t.Fatalf("config ref = %q, want cfg-1", ds.ConfigRef)
			}
			if ds.Kafka == nil {
				t.Fatal("expected kafka attachment details")
			}
			if got := ds.Kafka.Topics; len(got) != 1 || got[0] != "logs" {
				t.Fatalf("kafka topics = %v, want [logs]", got)
			}
			if got := ds.Kafka.ReplayWindow; got != 2*time.Minute {
				t.Fatalf("kafka replay window = %v, want 2m", got)
			}
			if req.KafkaBroker == nil || req.KafkaBroker.Port != 9092 {
				t.Fatalf("expected sandbox-local kafka broker provisioning, got %+v", req.KafkaBroker)
			}
			progress("Booting microVM", 1, 1)
			return &provider.SandboxResult{
				SandboxID:  "sbx-ds",
				Name:       "sandbox",
				State:      "RUNNING",
				IPAddress:  "10.0.0.3",
				MACAddress: "52:54:00:12:34:57",
				Bridge:     "br0",
				PID:        2345,
			}, nil
		},
	}
	server := newTestCreateSandboxServer(t, prov, nil, &config.Config{})
	stream := &fakeCreateSandboxStream{}

	err := server.CreateSandboxStream(&deerv1.CreateSandboxCommand{
		SandboxId: "sbx-ds",
		Name:      "sandbox",
		BaseImage: "ubuntu-22.04",
		DataSources: []*deerv1.DataSourceAttachment{
			{
				Type:      deerv1.DataSourceType_DATA_SOURCE_TYPE_KAFKA,
				ConfigRef: "cfg-1",
				Config: &deerv1.DataSourceAttachment_Kafka{
					Kafka: &deerv1.KafkaDataSourceAttachment{
						CaptureConfig: &deerv1.KafkaCaptureConfigBinding{
							Id:                  "cfg-1",
							SourceVm:            "vm-1",
							BootstrapServers:    []string{"kafka-1:9092"},
							Topics:              []string{"logs", "metrics"},
							Codec:               "json",
							MaxBufferAgeSeconds: 600,
							MaxBufferBytes:      4096,
							Enabled:             true,
						},
						Topics:              []string{"logs"},
						ReplayWindowSeconds: 120,
					},
				},
			},
		},
	}, stream)
	if err != nil {
		t.Fatalf("CreateSandboxStream: %v", err)
	}
	if len(stream.msgs) == 0 || !stream.msgs[len(stream.msgs)-1].GetDone() {
		t.Fatalf("expected terminal progress message, got %+v", stream.msgs)
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

	err := server.CreateSandboxStream(&deerv1.CreateSandboxCommand{
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

func TestCreateSandbox_RollsBackOnKafkaAttachFailure(t *testing.T) {
	prov := &fakeCreateSandboxProvider{
		createFn: func(_ context.Context, _ provider.CreateRequest) (*provider.SandboxResult, error) {
			return &provider.SandboxResult{
				SandboxID:  "sbx-kafka",
				Name:       "sandbox",
				State:      "RUNNING",
				IPAddress:  "10.0.0.2",
				MACAddress: "52:54:00:12:34:56",
				Bridge:     "br0",
				PID:        1234,
			}, nil
		},
	}
	server := newTestCreateSandboxServer(t, prov, nil, &config.Config{})
	server.attachKafkaDataSourcesFn = func(context.Context, string, string, []*deerv1.DataSourceAttachment, []*deerv1.KafkaCaptureConfigBinding) ([]*deerv1.SandboxKafkaStubInfo, error) {
		return nil, errors.New("forced kafka attach failure")
	}

	_, err := server.CreateSandbox(context.Background(), &deerv1.CreateSandboxCommand{
		SandboxId: "sbx-kafka",
		Name:      "sandbox",
		BaseImage: "ubuntu-22.04",
		DataSources: []*deerv1.DataSourceAttachment{
			{
				Type:      deerv1.DataSourceType_DATA_SOURCE_TYPE_KAFKA,
				ConfigRef: "cfg-1",
				Config: &deerv1.DataSourceAttachment_Kafka{
					Kafka: &deerv1.KafkaDataSourceAttachment{
						CaptureConfig: &deerv1.KafkaCaptureConfigBinding{Id: "cfg-1", SourceVm: "vm-1", Topics: []string{"logs"}},
					},
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "forced kafka attach failure") {
		t.Fatalf("CreateSandbox error = %v, want attach failure", err)
	}
	if len(prov.destroyed) != 1 || prov.destroyed[0] != "sbx-kafka" {
		t.Fatalf("destroyed = %v, want [sbx-kafka]", prov.destroyed)
	}
	sandboxes, listErr := server.store.ListSandboxes(context.Background())
	if listErr != nil {
		t.Fatalf("ListSandboxes: %v", listErr)
	}
	if len(sandboxes) != 0 {
		t.Fatalf("expected sandbox rollback to remove active state, got %d sandboxes", len(sandboxes))
	}
}

func TestCreateSandboxStream_RollsBackOnKafkaAttachFailure(t *testing.T) {
	prov := &fakeCreateSandboxProvider{
		createWithProgressFn: func(_ context.Context, _ provider.CreateRequest, progress func(string, int, int)) (*provider.SandboxResult, error) {
			progress("Booting microVM", 1, 1)
			return &provider.SandboxResult{
				SandboxID:  "sbx-stream-kafka",
				Name:       "sandbox",
				State:      "RUNNING",
				IPAddress:  "10.0.0.3",
				MACAddress: "52:54:00:12:34:57",
				Bridge:     "br0",
				PID:        2345,
			}, nil
		},
	}
	server := newTestCreateSandboxServer(t, prov, nil, &config.Config{})
	server.attachKafkaDataSourcesFn = func(context.Context, string, string, []*deerv1.DataSourceAttachment, []*deerv1.KafkaCaptureConfigBinding) ([]*deerv1.SandboxKafkaStubInfo, error) {
		return nil, errors.New("forced kafka attach failure")
	}
	stream := &fakeCreateSandboxStream{}

	err := server.CreateSandboxStream(&deerv1.CreateSandboxCommand{
		SandboxId: "sbx-stream-kafka",
		Name:      "sandbox",
		BaseImage: "ubuntu-22.04",
		DataSources: []*deerv1.DataSourceAttachment{
			{
				Type:      deerv1.DataSourceType_DATA_SOURCE_TYPE_KAFKA,
				ConfigRef: "cfg-1",
				Config: &deerv1.DataSourceAttachment_Kafka{
					Kafka: &deerv1.KafkaDataSourceAttachment{
						CaptureConfig: &deerv1.KafkaCaptureConfigBinding{Id: "cfg-1", SourceVm: "vm-1", Topics: []string{"logs"}},
					},
				},
			},
		},
	}, stream)
	if err == nil || !strings.Contains(err.Error(), "forced kafka attach failure") {
		t.Fatalf("CreateSandboxStream error = %v, want attach failure", err)
	}
	if len(prov.destroyed) != 1 || prov.destroyed[0] != "sbx-stream-kafka" {
		t.Fatalf("destroyed = %v, want [sbx-stream-kafka]", prov.destroyed)
	}
	if len(stream.msgs) == 0 || !stream.msgs[len(stream.msgs)-1].GetDone() || !strings.Contains(stream.msgs[len(stream.msgs)-1].GetError(), "forced kafka attach failure") {
		t.Fatalf("expected terminal error progress message, got %+v", stream.msgs)
	}
}

func TestCreateSandbox_ClampsKafkaBackedResources(t *testing.T) {
	prov := &fakeCreateSandboxProvider{
		createFn: func(_ context.Context, req provider.CreateRequest) (*provider.SandboxResult, error) {
			if req.VCPUs != provider.KafkaBrokerMinVCPUs {
				t.Fatalf("VCPUs = %d, want %d", req.VCPUs, provider.KafkaBrokerMinVCPUs)
			}
			if req.MemoryMB != provider.KafkaBrokerMinMemoryMB {
				t.Fatalf("MemoryMB = %d, want %d", req.MemoryMB, provider.KafkaBrokerMinMemoryMB)
			}
			return &provider.SandboxResult{
				SandboxID:  "sbx-kafka-floor",
				Name:       "sandbox",
				State:      "RUNNING",
				IPAddress:  "10.0.0.4",
				MACAddress: "52:54:00:12:34:58",
				Bridge:     "br0",
				PID:        3456,
			}, nil
		},
	}
	server := newTestCreateSandboxServer(t, prov, nil, &config.Config{})

	_, err := server.CreateSandbox(context.Background(), &deerv1.CreateSandboxCommand{
		SandboxId: "sbx-kafka-floor",
		Name:      "sandbox",
		BaseImage: "ubuntu-22.04",
		Vcpus:     1,
		MemoryMb:  512,
		DataSources: []*deerv1.DataSourceAttachment{
			{
				Type:      deerv1.DataSourceType_DATA_SOURCE_TYPE_KAFKA,
				ConfigRef: "cfg-1",
				Config: &deerv1.DataSourceAttachment_Kafka{
					Kafka: &deerv1.KafkaDataSourceAttachment{
						CaptureConfig: &deerv1.KafkaCaptureConfigBinding{Id: "cfg-1"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}

	sb, err := server.store.GetSandbox(context.Background(), "sbx-kafka-floor")
	if err != nil {
		t.Fatalf("GetSandbox: %v", err)
	}
	if sb.VCPUs != provider.KafkaBrokerMinVCPUs {
		t.Fatalf("stored VCPUs = %d, want %d", sb.VCPUs, provider.KafkaBrokerMinVCPUs)
	}
	if sb.MemoryMB != provider.KafkaBrokerMinMemoryMB {
		t.Fatalf("stored MemoryMB = %d, want %d", sb.MemoryMB, provider.KafkaBrokerMinMemoryMB)
	}
}

func TestCreateSandboxStream_ClampsKafkaBackedResources(t *testing.T) {
	prov := &fakeCreateSandboxProvider{
		createWithProgressFn: func(_ context.Context, req provider.CreateRequest, progress func(string, int, int)) (*provider.SandboxResult, error) {
			if req.VCPUs != provider.KafkaBrokerMinVCPUs {
				t.Fatalf("VCPUs = %d, want %d", req.VCPUs, provider.KafkaBrokerMinVCPUs)
			}
			if req.MemoryMB != provider.KafkaBrokerMinMemoryMB {
				t.Fatalf("MemoryMB = %d, want %d", req.MemoryMB, provider.KafkaBrokerMinMemoryMB)
			}
			progress("Booting microVM", 1, 1)
			return &provider.SandboxResult{
				SandboxID:  "sbx-stream-floor",
				Name:       "sandbox",
				State:      "RUNNING",
				IPAddress:  "10.0.0.5",
				MACAddress: "52:54:00:12:34:59",
				Bridge:     "br0",
				PID:        4567,
			}, nil
		},
	}
	server := newTestCreateSandboxServer(t, prov, nil, &config.Config{})
	stream := &fakeCreateSandboxStream{}

	err := server.CreateSandboxStream(&deerv1.CreateSandboxCommand{
		SandboxId: "sbx-stream-floor",
		Name:      "sandbox",
		BaseImage: "ubuntu-22.04",
		Vcpus:     1,
		MemoryMb:  512,
		DataSources: []*deerv1.DataSourceAttachment{
			{
				Type:      deerv1.DataSourceType_DATA_SOURCE_TYPE_KAFKA,
				ConfigRef: "cfg-1",
				Config: &deerv1.DataSourceAttachment_Kafka{
					Kafka: &deerv1.KafkaDataSourceAttachment{
						CaptureConfig: &deerv1.KafkaCaptureConfigBinding{Id: "cfg-1"},
					},
				},
			},
		},
	}, stream)
	if err != nil {
		t.Fatalf("CreateSandboxStream: %v", err)
	}

	sb, err := server.store.GetSandbox(context.Background(), "sbx-stream-floor")
	if err != nil {
		t.Fatalf("GetSandbox: %v", err)
	}
	if sb.VCPUs != provider.KafkaBrokerMinVCPUs {
		t.Fatalf("stored VCPUs = %d, want %d", sb.VCPUs, provider.KafkaBrokerMinVCPUs)
	}
	if sb.MemoryMB != provider.KafkaBrokerMinMemoryMB {
		t.Fatalf("stored MemoryMB = %d, want %d", sb.MemoryMB, provider.KafkaBrokerMinMemoryMB)
	}
}
