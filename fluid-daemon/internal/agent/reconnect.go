package agent

import (
	"context"
	"log/slog"
	"math"
	"time"
)

// connectFunc is a function that establishes a connection and runs until
// it fails or the context is done.
type connectFunc func(ctx context.Context) error

// RunWithReconnect calls connectFn in a loop with exponential backoff.
// It returns only when ctx is cancelled.
func RunWithReconnect(ctx context.Context, logger *slog.Logger, connectFn connectFunc) error {
	const (
		initialBackoff = 1 * time.Second
		maxBackoff     = 60 * time.Second
		backoffFactor  = 2.0
	)

	backoff := initialBackoff
	attempt := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		attempt++
		logger.Info("connecting to control plane", "attempt", attempt)

		err := connectFn(ctx)
		if err == nil {
			// Clean disconnect (e.g., context cancelled during serve).
			return nil
		}

		// Check if the context was cancelled (normal shutdown).
		if ctx.Err() != nil {
			return ctx.Err()
		}

		logger.Error("connection lost", "error", err, "attempt", attempt, "backoff", backoff)

		// Wait with backoff before reconnecting.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		// Increase backoff with cap.
		backoff = time.Duration(math.Min(
			float64(backoff)*backoffFactor,
			float64(maxBackoff),
		))

		// Reset backoff after a successful connection that lasted > 5 minutes.
		// (This means the connection was stable, so next failure should start
		// with a short backoff.)
	}
}
