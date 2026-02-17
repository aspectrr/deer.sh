package janitor

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/state"
)

func newTestStore(t *testing.T) *state.Store {
	t.Helper()
	st, err := state.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func insertExpiredSandbox(t *testing.T, st *state.Store, id string, ttlSeconds int, createdAt time.Time) {
	t.Helper()
	sb := &state.Sandbox{
		ID:         id,
		Name:       "test-" + id,
		AgentID:    "agent-1",
		BaseImage:  "ubuntu",
		State:      "RUNNING",
		TTLSeconds: ttlSeconds,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
	if err := st.CreateSandbox(context.Background(), sb); err != nil {
		t.Fatalf("failed to insert sandbox: %v", err)
	}
}

func TestJanitor_CleanupExpired(t *testing.T) {
	st := newTestStore(t)

	// Insert a sandbox that expired 10 seconds ago (TTL=1s, created 11s ago).
	insertExpiredSandbox(t, st, "SBX-expired", 1, time.Now().UTC().Add(-11*time.Second))

	var mu sync.Mutex
	destroyed := make([]string, 0)

	destroyFn := func(_ context.Context, sandboxID string) error {
		mu.Lock()
		defer mu.Unlock()
		destroyed = append(destroyed, sandboxID)
		return nil
	}

	j := New(st, destroyFn, 5*time.Minute, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())

	// Start janitor in background; it runs cleanup immediately.
	done := make(chan struct{})
	go func() {
		j.Start(ctx, 50*time.Millisecond)
		close(done)
	}()

	// Give it time to run the immediate cleanup.
	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()

	if len(destroyed) == 0 {
		t.Fatal("expected destroyFn to be called for expired sandbox, but it was not")
	}

	found := false
	for _, id := range destroyed {
		if id == "SBX-expired" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected SBX-expired in destroyed list, got %v", destroyed)
	}
}

func TestJanitor_NoExpired(t *testing.T) {
	st := newTestStore(t)

	// Insert a sandbox that is NOT expired (TTL=1h, created just now).
	insertExpiredSandbox(t, st, "SBX-fresh", 3600, time.Now().UTC())

	called := false
	destroyFn := func(_ context.Context, _ string) error {
		called = true
		return nil
	}

	j := New(st, destroyFn, 5*time.Minute, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		j.Start(ctx, 50*time.Millisecond)
		close(done)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	if called {
		t.Error("expected destroyFn NOT to be called when no sandboxes are expired")
	}
}

func TestJanitor_DestroyError(t *testing.T) {
	st := newTestStore(t)

	// Insert two expired sandboxes.
	insertExpiredSandbox(t, st, "SBX-fail", 1, time.Now().UTC().Add(-11*time.Second))
	insertExpiredSandbox(t, st, "SBX-ok", 1, time.Now().UTC().Add(-11*time.Second))

	var mu sync.Mutex
	calls := make([]string, 0)

	destroyFn := func(_ context.Context, sandboxID string) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, sandboxID)
		if sandboxID == "SBX-fail" {
			return errors.New("simulated destroy failure")
		}
		return nil
	}

	j := New(st, destroyFn, 5*time.Minute, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		j.Start(ctx, 50*time.Millisecond)
		close(done)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()

	// Both sandboxes should have been attempted regardless of the error on the first.
	if len(calls) < 2 {
		t.Errorf("expected destroyFn to be called for both sandboxes, got calls: %v", calls)
	}
}
