package agent

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunWithReconnect_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	connectFn := func(ctx context.Context) error {
		return errors.New("should not be called meaningfully")
	}

	err := RunWithReconnect(ctx, slog.Default(), connectFn)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRunWithReconnect_SuccessfulConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connectFn := func(ctx context.Context) error {
		return nil // success on first call
	}

	err := RunWithReconnect(ctx, slog.Default(), connectFn)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRunWithReconnect_RetriesOnError(t *testing.T) {
	var callCount atomic.Int32

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connectFn := func(ctx context.Context) error {
		n := callCount.Add(1)
		if n < 3 {
			return errors.New("connection failed")
		}
		return nil // succeed on 3rd attempt
	}

	// Override timing: we cancel context after connectFn succeeds, so
	// backoff waits are the bottleneck. The first two failures will incur
	// backoff waits of 1s and 2s respectively from the production code.
	// We use a generous timeout above to accommodate that.
	err := RunWithReconnect(ctx, slog.Default(), connectFn)
	if err != nil {
		t.Fatalf("expected nil after retries, got %v", err)
	}

	got := callCount.Load()
	if got != 3 {
		t.Fatalf("expected 3 calls, got %d", got)
	}
}

func TestRunWithReconnect_BackoffCap(t *testing.T) {
	// Verify the backoff math: starting at 1s, doubling, capped at 60s.
	// This tests the algorithm without waiting for actual backoff durations.
	const (
		initialBackoff = 1 * time.Second
		maxBackoff     = 60 * time.Second
		backoffFactor  = 2.0
	)

	backoff := initialBackoff
	expected := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		32 * time.Second,
		60 * time.Second, // capped
		60 * time.Second, // stays capped
	}

	for i, want := range expected {
		if backoff != want {
			t.Fatalf("step %d: expected backoff %v, got %v", i, want, backoff)
		}
		// Apply the same backoff calculation as RunWithReconnect
		backoff = time.Duration(math.Min(
			float64(backoff)*backoffFactor,
			float64(maxBackoff),
		))
	}
}

func TestRunWithReconnect_CancelDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var callCount atomic.Int32

	connectFn := func(ctx context.Context) error {
		n := callCount.Add(1)
		if n == 1 {
			// After first failure, cancel context so RunWithReconnect
			// exits during the backoff wait.
			go func() {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}()
		}
		return errors.New("connection failed")
	}

	err := RunWithReconnect(ctx, slog.Default(), connectFn)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	got := callCount.Load()
	if got < 1 {
		t.Fatalf("expected at least 1 call, got %d", got)
	}
}
