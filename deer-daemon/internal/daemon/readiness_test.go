package daemon

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestReadinessServerSignalsAndTracksReady(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rs := NewReadinessServer("127.0.0.1:39092", logger)
	ln, err := net.Listen("tcp", "127.0.0.1:39092")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- rs.Serve(ln)
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = rs.Shutdown(ctx)
		<-done
	})

	rs.Register("sbx-123")
	if rs.WasReady("sbx-123") {
		t.Fatal("WasReady before callback = true, want false")
	}

	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:39092/ready/sbx-123", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST readiness: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if err := rs.WaitReady("sbx-123", 2*time.Second); err != nil {
		t.Fatalf("WaitReady: %v", err)
	}
	if !rs.WasReady("sbx-123") {
		t.Fatal("WasReady after callback = false, want true")
	}
	if got := rs.ReadyIP("sbx-123"); got != "127.0.0.1" {
		t.Fatalf("ReadyIP = %q, want 127.0.0.1", got)
	}
}

func TestReadinessServerNestedRegisterKeepsWaiterUntilFinalUnregister(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rs := NewReadinessServer("127.0.0.1:39093", logger)
	ln, err := net.Listen("tcp", "127.0.0.1:39093")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- rs.Serve(ln)
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = rs.Shutdown(ctx)
		<-done
	})

	rs.Register("sbx-nested")
	rs.Register("sbx-nested")
	rs.Unregister("sbx-nested")

	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:39093/ready/sbx-nested", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST readiness: %v", err)
	}
	_ = resp.Body.Close()

	if err := rs.WaitReady("sbx-nested", 2*time.Second); err != nil {
		t.Fatalf("WaitReady after nested unregister: %v", err)
	}
	if !rs.WasReady("sbx-nested") {
		t.Fatal("WasReady after nested unregister = false, want true")
	}
	if got := rs.ReadyIP("sbx-nested"); got != "127.0.0.1" {
		t.Fatalf("ReadyIP = %q, want 127.0.0.1", got)
	}

	rs.Unregister("sbx-nested")
	if rs.WasReady("sbx-nested") {
		t.Fatal("WasReady after final unregister = true, want false")
	}
	if got := rs.ReadyIP("sbx-nested"); got != "" {
		t.Fatalf("ReadyIP after final unregister = %q, want empty", got)
	}
}

func TestReadinessServer_WaitReadyTimeout(t *testing.T) {
	rs := NewReadinessServer("127.0.0.1:39094", slog.Default())
	rs.Register("sbx-timeout")

	err := rs.WaitReady("sbx-timeout", 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("error = %q, want timeout", err.Error())
	}
}

func TestReadinessServer_UnregisteredReady(t *testing.T) {
	rs := NewReadinessServer("127.0.0.1:39095", slog.Default())

	ln, err := net.Listen("tcp", "127.0.0.1:39095")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = rs.Serve(ln) }()
	defer func() { _ = rs.Shutdown(context.Background()) }()

	resp, err := http.Post("http://127.0.0.1:39095/ready/sbx-unknown", "", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unregistered ID, got %d", resp.StatusCode)
	}
	if rs.WasReady("sbx-unknown") {
		t.Fatal("unregistered ID should not be marked ready")
	}
}

func TestReadinessServer_WrongMethod(t *testing.T) {
	rs := NewReadinessServer("127.0.0.1:39096", slog.Default())

	ln, err := net.Listen("tcp", "127.0.0.1:39096")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = rs.Serve(ln) }()
	defer func() { _ = rs.Shutdown(context.Background()) }()

	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:39096/ready/sbx-1", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}
