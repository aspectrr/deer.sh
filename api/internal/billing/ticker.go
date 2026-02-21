package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/aspectrr/fluid.sh/api/internal/config"
	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// ResourceTicker periodically reports non-token resource usage to Stripe meters.
// It uses store.DataStore (not store.Store) because it only needs data access
// methods, not lifecycle methods like Ping/Close/WithTx.
type ResourceTicker struct {
	store    store.DataStore
	meter    *MeterManager
	registry *registry.Registry
	cfg      config.BillingConfig
	freeTier config.FreeTierConfig
	logger   *slog.Logger
	interval time.Duration
}

// NewResourceTicker creates a new ticker that reports resource usage every interval.
func NewResourceTicker(
	st store.DataStore,
	mm *MeterManager,
	reg *registry.Registry,
	cfg config.BillingConfig,
	logger *slog.Logger,
) *ResourceTicker {
	if logger == nil {
		logger = slog.Default()
	}
	return &ResourceTicker{
		store:    st,
		meter:    mm,
		registry: reg,
		cfg:      cfg,
		freeTier: cfg.FreeTier,
		logger:   logger.With("component", "billing-ticker"),
		interval: time.Hour,
	}
}

// Start runs the ticker loop until ctx is cancelled.
func (rt *ResourceTicker) Start(ctx context.Context) {
	ticker := time.NewTicker(rt.interval)
	defer ticker.Stop()

	rt.logger.Info("resource billing ticker started", "interval", rt.interval)

	for {
		select {
		case <-ctx.Done():
			rt.logger.Info("resource billing ticker stopped")
			return
		case <-ticker.C:
			rt.tick(ctx)
		}
	}
}

func (rt *ResourceTicker) tick(ctx context.Context) {
	subs, err := rt.store.ListActiveSubscriptions(ctx)
	if err != nil {
		rt.logger.Warn("failed to list active subscriptions", "error", err)
		return
	}

	for _, sub := range subs {
		rt.reportForOrg(ctx, sub.OrgID)
	}
}

func (rt *ResourceTicker) reportForOrg(ctx context.Context, orgID string) {
	org, err := rt.store.GetOrganization(ctx, orgID)
	if err != nil {
		rt.logger.Warn("failed to get org for billing tick", "error", err, "org_id", orgID)
		return
	}
	if org.StripeCustomerID == "" {
		return
	}

	now := time.Now().UTC()

	// Get counts from registry (daemon-reported data via heartbeats)
	runningSandboxes, sourceVMCount, daemonCount := rt.registry.OrgResourceCounts(orgID)

	// Subtract free tier
	billableSandboxes := int64(runningSandboxes - rt.freeTier.MaxConcurrentSandboxes)
	billableSourceVMs := int64(sourceVMCount - rt.freeTier.MaxSourceVMs)
	billableDaemons := int64(daemonCount - rt.freeTier.MaxAgentHosts)

	// Report to Stripe
	if billableSandboxes > 0 {
		rt.meter.ReportResourceUsage(ctx, org.StripeCustomerID, "concurrent_sandboxes", billableSandboxes)
	}
	if billableSourceVMs > 0 {
		rt.meter.ReportResourceUsage(ctx, org.StripeCustomerID, "source_vms", billableSourceVMs)
	}
	if billableDaemons > 0 {
		rt.meter.ReportResourceUsage(ctx, org.StripeCustomerID, "fluid_daemons", billableDaemons)
	}

	// Create local usage records
	if runningSandboxes > 0 {
		if err := rt.store.CreateUsageRecord(ctx, &store.UsageRecord{
			ID:           uuid.New().String(),
			OrgID:        orgID,
			ResourceType: "max_concurrent_sandboxes",
			Quantity:     float64(runningSandboxes),
			RecordedAt:   now,
		}); err != nil {
			rt.logger.Warn("failed to create usage record", "type", "max_concurrent_sandboxes", "org_id", orgID, "error", err)
		}
	}
	if sourceVMCount > 0 {
		if err := rt.store.CreateUsageRecord(ctx, &store.UsageRecord{
			ID:           uuid.New().String(),
			OrgID:        orgID,
			ResourceType: "source_vm",
			Quantity:     float64(sourceVMCount),
			RecordedAt:   now,
		}); err != nil {
			rt.logger.Warn("failed to create usage record", "type", "source_vm", "org_id", orgID, "error", err)
		}
	}
	if daemonCount > 0 {
		if err := rt.store.CreateUsageRecord(ctx, &store.UsageRecord{
			ID:           uuid.New().String(),
			OrgID:        orgID,
			ResourceType: "agent_host",
			Quantity:     float64(daemonCount),
			RecordedAt:   now,
		}); err != nil {
			rt.logger.Warn("failed to create usage record", "type", "agent_host", "org_id", orgID, "error", err)
		}
	}

	rt.logger.Debug("billing tick completed",
		"org_id", orgID,
		"sandboxes", runningSandboxes,
		"source_vms", sourceVMCount,
		"daemons", daemonCount,
	)
}
