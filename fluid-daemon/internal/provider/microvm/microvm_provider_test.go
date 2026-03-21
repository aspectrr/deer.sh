package microvm

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	imagestore "github.com/aspectrr/fluid.sh/fluid-daemon/internal/image"
	microvminternal "github.com/aspectrr/fluid.sh/fluid-daemon/internal/microvm"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/network"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider"
)

type stubReadinessWaiter struct {
	registered   []string
	unregistered []string
}

func (s *stubReadinessWaiter) Register(sandboxID string) {
	s.registered = append(s.registered, sandboxID)
}

func (s *stubReadinessWaiter) Unregister(sandboxID string) {
	s.unregistered = append(s.unregistered, sandboxID)
}

func (s *stubReadinessWaiter) WaitReady(string, time.Duration) error {
	return nil
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
