// Package janitor provides background cleanup of expired sandboxes on the host.
package janitor

import (
	"context"
	"log/slog"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/state"
)

// DestroyFunc is called to destroy an expired sandbox.
type DestroyFunc func(ctx context.Context, sandboxID string) error

// Janitor periodically cleans up expired sandboxes.
type Janitor struct {
	store      *state.Store
	destroyFn  DestroyFunc
	logger     *slog.Logger
	defaultTTL time.Duration
}

// New creates a new Janitor service.
func New(st *state.Store, destroyFn DestroyFunc, defaultTTL time.Duration, logger *slog.Logger) *Janitor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Janitor{
		store:      st,
		destroyFn:  destroyFn,
		logger:     logger.With("component", "janitor"),
		defaultTTL: defaultTTL,
	}
}

// Start runs the cleanup loop. It blocks until the context is cancelled.
func (j *Janitor) Start(ctx context.Context, interval time.Duration) {
	j.logger.Info("starting janitor",
		"interval", interval,
		"default_ttl", j.defaultTTL,
	)

	// Run once immediately
	j.cleanup(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.logger.Info("janitor stopped")
			return
		case <-ticker.C:
			j.cleanup(ctx)
		}
	}
}

// cleanup finds and destroys all expired sandboxes.
func (j *Janitor) cleanup(ctx context.Context) {
	expired, err := j.store.ListExpiredSandboxes(ctx, j.defaultTTL)
	if err != nil {
		j.logger.Error("failed to list expired sandboxes", "error", err)
		return
	}

	if len(expired) == 0 {
		return
	}

	j.logger.Info("found expired sandboxes", "count", len(expired))

	for _, sb := range expired {
		j.logger.Info("destroying expired sandbox",
			"id", sb.ID,
			"name", sb.Name,
			"ttl_seconds", sb.TTLSeconds,
			"created_at", sb.CreatedAt,
			"age", time.Since(sb.CreatedAt),
		)

		if err := j.destroyFn(ctx, sb.ID); err != nil {
			j.logger.Error("failed to destroy expired sandbox",
				"id", sb.ID,
				"error", err,
			)
		} else {
			j.logger.Info("destroyed expired sandbox", "id", sb.ID)
		}
	}
}
