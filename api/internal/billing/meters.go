package billing

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v82"
	stripeMeterEvent "github.com/stripe/stripe-go/v82/billing/meterevent"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// MeterManager handles Stripe meter/price creation and usage reporting.
type MeterManager struct {
	store     store.DataStore
	stripeKey string
	markup    float64
	logger    *slog.Logger
}

// NewMeterManager creates a new MeterManager.
func NewMeterManager(
	st store.DataStore,
	stripeKey string,
	markup float64,
	logger *slog.Logger,
) *MeterManager {
	if logger == nil {
		logger = slog.Default()
	}
	// Set the Stripe key once at initialization rather than per-call
	// to avoid a race condition when multiple goroutines set the global.
	if stripeKey != "" {
		stripe.Key = stripeKey
	}
	return &MeterManager{
		store:     st,
		stripeKey: stripeKey,
		markup:    markup,
		logger:    logger.With("component", "billing"),
	}
}

// Markup returns the configured billing markup multiplier.
func (mm *MeterManager) Markup() float64 { return mm.markup }

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// sanitizeEventName converts a model ID like "anthropic/claude-sonnet-4" to "anthropic_claude_sonnet_4".
func sanitizeEventName(modelID string) string {
	s := strings.ToLower(modelID)
	s = nonAlphaNum.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}

// ReportResourceUsage reports non-token resource usage (sandboxes, source VMs, daemons) to Stripe.
// The caller is responsible for subtracting free tier counts before calling this.
func (mm *MeterManager) ReportResourceUsage(ctx context.Context, stripeCustomerID, eventName string, value int64) {
	if value <= 0 || stripeCustomerID == "" {
		return
	}

	_, err := stripeMeterEvent.New(&stripe.BillingMeterEventParams{
		EventName: stripe.String(eventName),
		Payload: map[string]string{
			"stripe_customer_id": stripeCustomerID,
			"value":              fmt.Sprintf("%d", value),
		},
		Identifier: stripe.String(fmt.Sprintf("%s_%s_%d", stripeCustomerID, eventName, time.Now().UTC().Truncate(time.Hour).Unix())),
	})
	if err != nil {
		mm.logger.Warn("failed to report resource meter event",
			"error", err,
			"event_name", eventName,
			"value", value,
		)
		return
	}

	mm.logger.Debug("reported resource usage to stripe",
		"event_name", eventName,
		"value", value,
		"customer", stripeCustomerID,
	)
}

// LLM token metering - commented out, not yet ready for integration.
/*
import (
	"context"
	"errors"
	"math"
	"sync"

	stripeBillingMeter "github.com/stripe/stripe-go/v82/billing/meter"
	stripePrice "github.com/stripe/stripe-go/v82/price"
	stripeProduct "github.com/stripe/stripe-go/v82/product"
	stripeSubItem "github.com/stripe/stripe-go/v82/subscriptionitem"

	"github.com/aspectrr/fluid.sh/api/internal/agent"
)

// EnsureModelMeter returns an existing ModelMeter or creates Stripe objects and stores a new one.
func (mm *MeterManager) EnsureModelMeter(ctx context.Context, modelID string) (*store.ModelMeter, error) {
	...
}

// EnsureOrgSubscriptionItems adds subscription items for a model to an org's subscription.
func (mm *MeterManager) EnsureOrgSubscriptionItems(ctx context.Context, orgID, modelID string) error {
	...
}

// ReportUsage reports token usage to Stripe billing meters, respecting the free tier.
func (mm *MeterManager) ReportUsage(ctx context.Context, orgID, modelID string, inputTokens, outputTokens int) error {
	...
}
*/
