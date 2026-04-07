package microvm

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	imagestore "github.com/aspectrr/deer.sh/deer-daemon/internal/image"
	microvminternal "github.com/aspectrr/deer.sh/deer-daemon/internal/microvm"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/network"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/provider"
)

type stubReadinessWaiter struct {
	registered   []string
	unregistered []string
	readyIP      string
	waitFn       func(string, time.Duration) error
}

func (s *stubReadinessWaiter) Register(sandboxID string) {
	s.registered = append(s.registered, sandboxID)
}

func (s *stubReadinessWaiter) Unregister(sandboxID string) {
	s.unregistered = append(s.unregistered, sandboxID)
}

func (s *stubReadinessWaiter) WaitReady(sandboxID string, timeout time.Duration) error {
	if s.waitFn != nil {
		return s.waitFn(sandboxID, timeout)
	}
	return nil
}

func (s *stubReadinessWaiter) ReadyIP(string) string {
	return s.readyIP
}

func TestCreateSandboxWithProgress_RegistersReadinessBeforeFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	workDir := t.TempDir()
	imageDir := filepath.Join(workDir, "images")

	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "ubuntu.qcow2"), []byte("qcow2"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}

	vmMgr, err := microvminternal.NewManager("true", filepath.Join(workDir, "sandboxes"), logger)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	imgStore, err := imagestore.NewStore(imageDir, logger)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	readiness := &stubReadinessWaiter{}
	var progressSteps []string
	p := &Provider{
		vmMgr:     vmMgr,
		netMgr:    network.NewNetworkManager("br-test0", nil, "", logger),
		imgStore:  imgStore,
		readiness: readiness,
		logger:    logger,
	}

	_, err = p.CreateSandboxWithProgress(context.Background(), provider.CreateRequest{
		SandboxID: "SBX-123",
		BaseImage: "ubuntu",
	}, func(step string, _ int, _ int) {
		progressSteps = append(progressSteps, step)
	})
	if err == nil {
		t.Fatal("expected CreateSandboxWithProgress to fail without a configured kernel path")
	}
	if len(progressSteps) == 0 {
		t.Fatal("expected at least one progress step before failure")
	}
	if progressSteps[0] != "Resolving network bridge" {
		t.Fatalf("first progress step = %q, want %q", progressSteps[0], "Resolving network bridge")
	}

	if len(readiness.registered) != 1 || readiness.registered[0] != "SBX-123" {
		t.Fatalf("registered = %v, want [SBX-123]", readiness.registered)
	}
	if len(readiness.unregistered) != 1 || readiness.unregistered[0] != "SBX-123" {
		t.Fatalf("unregistered = %v, want [SBX-123]", readiness.unregistered)
	}
}

func TestPhoneHomeURLRequiresReadiness(t *testing.T) {
	p := &Provider{
		bridgeIP:  "192.168.122.1",
		readiness: nil,
	}
	if got := p.phoneHomeURL("sbx-123"); got != "" {
		t.Fatalf("phoneHomeURL without readiness = %q, want empty", got)
	}
}

func TestPhoneHomeURLUsesBridgeWhenReadinessPresent(t *testing.T) {
	p := &Provider{
		bridgeIP:  "192.168.122.1",
		readiness: &stubReadinessWaiter{},
	}
	if got := p.phoneHomeURL("sbx-123"); got != "http://192.168.122.1:9092/ready/sbx-123" {
		t.Fatalf("phoneHomeURL with readiness = %q", got)
	}
}

func TestApplyReadinessIPFallback_UsesReadyIPWhenDiscoveryEmpty(t *testing.T) {
	p := &Provider{
		readiness: &stubReadinessWaiter{readyIP: "192.168.122.44"},
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if got := p.applyReadinessIPFallback("sbx-123", ""); got != "192.168.122.44" {
		t.Fatalf("fallback IP = %q, want 192.168.122.44", got)
	}
}

func TestWaitForReadiness_TimeoutFails(t *testing.T) {
	p := &Provider{
		readiness:    &stubReadinessWaiter{waitFn: func(string, time.Duration) error { return fmt.Errorf("readiness timeout for sandbox sbx-123 after 2s") }},
		bridgeIP:     "192.168.122.1",
		readyTimeout: 25 * time.Millisecond,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	err := p.waitForReadiness(context.Background(), "sbx-123", 1234)
	if err == nil || !strings.Contains(err.Error(), "phone_home readiness timeout") {
		t.Fatalf("waitForReadiness error = %v, want phone_home timeout", err)
	}
}

func TestWaitForReadiness_EarlyDeathFails(t *testing.T) {
	p := &Provider{
		vmMgr:        &microvminternal.Manager{},
		readiness:    &stubReadinessWaiter{waitFn: func(string, time.Duration) error { return fmt.Errorf("readiness timeout for sandbox sbx-123 after 2s") }},
		bridgeIP:     "192.168.122.1",
		readyTimeout: time.Second,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	err := p.waitForReadiness(context.Background(), "sbx-123", 1234)
	if err == nil || !strings.Contains(err.Error(), "failed before phone_home") {
		t.Fatalf("waitForReadiness error = %v, want early death failure", err)
	}
}

func TestCompleteCreate_UsesReadyIPFallbackAfterPhoneHome(t *testing.T) {
	p := &Provider{
		readiness: &stubReadinessWaiter{
			readyIP: "192.168.122.44",
		},
		bridgeIP: "192.168.122.1",
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	result, err := p.completeCreate(context.Background(), provider.CreateRequest{
		SandboxID: "sbx-123",
		Name:      "sandbox",
	}, &microvminternal.SandboxInfo{PID: 4321}, "52:54:00:12:34:56", "br0", "tap0")
	if err != nil {
		t.Fatalf("completeCreate: %v", err)
	}
	if result.IPAddress != "192.168.122.44" {
		t.Fatalf("IPAddress = %q, want 192.168.122.44", result.IPAddress)
	}
}

func TestKafkaBrokerOptions_EnabledByGenericDataSource(t *testing.T) {
	opts := kafkaBrokerOptions(provider.CreateRequest{
		DataSources: []provider.DataSourceAttachment{
			{
				Type:      provider.DataSourceTypeKafka,
				ConfigRef: "cfg-1",
				Kafka: &provider.KafkaDataSourceConfig{
					CaptureConfigID: "cfg-1",
				},
			},
		},
	})
	if !opts.Enabled {
		t.Fatal("expected kafka broker options to be enabled for kafka data source")
	}
	if opts.Port != 9092 {
		t.Fatalf("port = %d, want 9092", opts.Port)
	}
	if opts.AdvertiseAddress != "" {
		t.Fatalf("AdvertiseAddress = %q, want empty for runtime guest resolution", opts.AdvertiseAddress)
	}
}

func TestKafkaBrokerOptions_PreservesExplicitAdvertiseAddress(t *testing.T) {
	opts := kafkaBrokerOptions(provider.CreateRequest{
		KafkaBroker: &provider.KafkaBrokerConfig{
			AdvertiseAddress: "192.168.122.44",
			Port:             9092,
		},
	})
	if !opts.Enabled {
		t.Fatal("expected kafka broker options to be enabled")
	}
	if opts.AdvertiseAddress != "192.168.122.44" {
		t.Fatalf("AdvertiseAddress = %q, want 192.168.122.44", opts.AdvertiseAddress)
	}
}

func TestKernelOOMDiagnosticsForPID_MatchesRelevantWindow(t *testing.T) {
	kernelLog := strings.Join([]string{
		"Mar 28 16:19:16 host kernel: qemu-system-aar invoked oom-killer: gfp_mask=0x140cca",
		"Mar 28 16:19:16 host kernel: [  pid  ]   uid  tgid total_vm      rss rss_anon rss_file rss_shmem pgtables_bytes swapents oom_score_adj name",
		"Mar 28 16:19:16 host kernel: [ 416544]     0 416544  1322332   389282   389009      273         0  3731456        0             0 qemu-system-aar",
		"Mar 28 16:19:16 host kernel: oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),task=qemu-system-aar,pid=416544,uid=0",
		"Mar 28 16:19:16 host kernel: Out of memory: Killed process 416544 (qemu-system-aar) total-vm:5289328kB",
		"Mar 28 16:19:20 host kernel: unrelated tail line",
	}, "\n")

	got := kernelOOMDiagnosticsForPID(416544, kernelLog)
	if !strings.Contains(got, "Killed process 416544") {
		t.Fatalf("expected killed-process line in diagnostics, got %q", got)
	}
	if !strings.Contains(got, "pid=416544") {
		t.Fatalf("expected pid match in diagnostics, got %q", got)
	}
	if !strings.Contains(got, "[ 416544]") {
		t.Fatalf("expected process table line in diagnostics, got %q", got)
	}
}

func TestKernelOOMDiagnosticsForPID_IgnoresOtherProcesses(t *testing.T) {
	kernelLog := strings.Join([]string{
		"Mar 28 16:19:16 host kernel: Out of memory: Killed process 111111 (qemu-system-aar) total-vm:5289328kB",
		"Mar 28 16:19:20 host kernel: unrelated tail line",
	}, "\n")

	if got := kernelOOMDiagnosticsForPID(416544, kernelLog); got != "" {
		t.Fatalf("expected no diagnostics for unmatched pid, got %q", got)
	}
}
