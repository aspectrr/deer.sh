package kafkastub

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/redact"
)

func TestRecordCaptureEvictsByMaxBytes(t *testing.T) {
	t.Parallel()

	mgr, err := NewManager(t.TempDir(), redact.New(), slog.Default())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	mgr.EnsureCaptureConfig(CaptureConfig{
		ID:             "cfg-1",
		SourceVM:       "logstash-1",
		Codec:          "text",
		MaxBufferBytes: 900,
		Enabled:        true,
	})

	ctx := context.Background()
	if _, err := mgr.RecordCapture(ctx, "cfg-1", strings.Repeat("first-payload-", 30), time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordCapture #1: %v", err)
	}
	status, err := mgr.RecordCapture(ctx, "cfg-1", strings.Repeat("second-payload-", 30), time.Unix(2, 0))
	if err != nil {
		t.Fatalf("RecordCapture #2: %v", err)
	}

	if status.SegmentCount != 1 {
		t.Fatalf("expected 1 segment after eviction, got %d", status.SegmentCount)
	}
	if status.BufferedBytes > 900 {
		t.Fatalf("expected buffered bytes <= 900, got %d", status.BufferedBytes)
	}
}

func TestRecordCaptureRejectsUnsupportedCodec(t *testing.T) {
	t.Parallel()

	mgr, err := NewManager(t.TempDir(), redact.New(), slog.Default())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	mgr.EnsureCaptureConfig(CaptureConfig{
		ID:       "cfg-unsupported",
		SourceVM: "logstash-1",
		Codec:    "avro",
		Enabled:  true,
	})

	if _, err := mgr.RecordCapture(context.Background(), "cfg-unsupported", "payload", time.Now()); err == nil {
		t.Fatal("expected unsupported codec error")
	}
}

func TestSandboxStubLifecycle(t *testing.T) {
	t.Parallel()

	mgr, err := NewManager(t.TempDir(), redact.New(), slog.Default())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	stubs, err := mgr.AttachSandbox(context.Background(), "SBX-1", "10.0.0.10", []SandboxAttachment{{
		CaptureConfig: CaptureConfig{
			ID:             "cfg-1",
			SourceVM:       "logstash-1",
			Topics:         []string{"logs"},
			Codec:          "json",
			MaxBufferAge:   10 * time.Minute,
			MaxBufferBytes: 1024,
			Enabled:        true,
		},
	}})
	if err != nil {
		t.Fatalf("AttachSandbox: %v", err)
	}
	if len(stubs) != 1 {
		t.Fatalf("expected 1 stub, got %d", len(stubs))
	}
	if stubs[0].BrokerEndpoint != "10.0.0.10:9092" {
		t.Fatalf("unexpected broker endpoint %q", stubs[0].BrokerEndpoint)
	}

	stopped, err := mgr.StopSandboxStub(context.Background(), "SBX-1", stubs[0].ID)
	if err != nil {
		t.Fatalf("StopSandboxStub: %v", err)
	}
	if stopped.State != StateStopped {
		t.Fatalf("expected stopped state, got %q", stopped.State)
	}

	restarted, err := mgr.RestartSandboxStub(context.Background(), "SBX-1", stubs[0].ID)
	if err != nil {
		t.Fatalf("RestartSandboxStub: %v", err)
	}
	if restarted.State != StateRunning {
		t.Fatalf("expected running state, got %q", restarted.State)
	}
	if restarted.LastReplayCursor != "head" {
		t.Fatalf("expected replay cursor reset to head, got %q", restarted.LastReplayCursor)
	}

	statuses, err := mgr.ListCaptureStatuses(context.Background(), []string{"cfg-1"})
	if err != nil {
		t.Fatalf("ListCaptureStatuses: %v", err)
	}
	if len(statuses) != 1 || statuses[0].AttachedSandboxCount != 1 {
		t.Fatalf("expected one attached sandbox, got %+v", statuses)
	}

	if err := mgr.DetachSandbox(context.Background(), "SBX-1"); err != nil {
		t.Fatalf("DetachSandbox: %v", err)
	}
	listed, err := mgr.ListSandboxStubs(context.Background(), "SBX-1")
	if err != nil {
		t.Fatalf("ListSandboxStubs: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected no stubs after detach, got %d", len(listed))
	}
}

func TestAttachSandbox_AppliesTopicAndReplayOverrides(t *testing.T) {
	t.Parallel()

	mgr, err := NewManager(t.TempDir(), redact.New(), slog.Default())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	stubs, err := mgr.AttachSandbox(context.Background(), "SBX-2", "10.0.0.11", []SandboxAttachment{{
		CaptureConfig: CaptureConfig{
			ID:             "cfg-2",
			SourceVM:       "logstash-2",
			Topics:         []string{"logs", "metrics"},
			Codec:          "json",
			MaxBufferAge:   15 * time.Minute,
			MaxBufferBytes: 4096,
			Enabled:        true,
		},
		Topics:       []string{"logs"},
		ReplayWindow: 2 * time.Minute,
	}})
	if err != nil {
		t.Fatalf("AttachSandbox: %v", err)
	}
	if len(stubs) != 1 {
		t.Fatalf("expected 1 stub, got %d", len(stubs))
	}
	if got := stubs[0].Topics; len(got) != 1 || got[0] != "logs" {
		t.Fatalf("stub topics = %v, want [logs]", got)
	}
	if got := stubs[0].ReplayWindow; got != 2*time.Minute {
		t.Fatalf("replay window = %v, want 2m", got)
	}
}

func TestRecordCapturePersistsRedactedPayload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr, err := NewManager(dir, redact.New(), slog.Default())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	mgr.EnsureCaptureConfig(CaptureConfig{
		ID:       "cfg-redact",
		SourceVM: "logstash-1",
		Codec:    "text",
		Enabled:  true,
	})

	if _, err := mgr.RecordCapture(context.Background(), "cfg-redact", "connect 10.0.0.1", time.Unix(3, 0)); err != nil {
		t.Fatalf("RecordCapture: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "cfg-redact", "*.json"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one persisted segment, got %d", len(files))
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var stored persistedRecord
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if string(stored.ValueBytes()) == "connect 10.0.0.1" {
		t.Fatal("expected persisted payload to be redacted")
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct{ in, want string }{
		{"normal-id", "normal-id"},
		{"has/slash", "has-slash"},
		{"has:colon", "has-colon"},
		{"has_underscore", "has-underscore"},
		{"HAS-CAPS", "has-caps"},
		{"../traversal", "--traversal"},
		{"..", "-"},
		{"a\x00b", "ab"},
		{"mix/../path:with_special\x00chars", "mix---path-with-specialchars"},
	}
	for _, tt := range tests {
		got := sanitizeID(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
