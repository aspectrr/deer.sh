package billing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	stripeBillingMeter "github.com/stripe/stripe-go/v82/billing/meter"
	stripeMeterEvent "github.com/stripe/stripe-go/v82/billing/meterevent"
	stripePrice "github.com/stripe/stripe-go/v82/price"
	stripeProduct "github.com/stripe/stripe-go/v82/product"
	stripeSubItem "github.com/stripe/stripe-go/v82/subscriptionitem"

	"github.com/aspectrr/fluid.sh/api/internal/agent"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// MeterManager handles Stripe meter/price creation and usage reporting for LLM models.
type MeterManager struct {
	store      store.DataStore
	modelCache *agent.ModelCache
	stripeKey  string
	markup     float64
	freeTokens int
	logger     *slog.Logger
	meterMu    sync.Map // map[string]*sync.Mutex - per-model lock for EnsureModelMeter
	orgMu      sync.Map // map[string]*sync.Mutex - per-org lock for free tier calculation
	subItemMu  sync.Map // map[string]*sync.Mutex - per-org:model lock for subscription items
}

// NewMeterManager creates a new MeterManager.
func NewMeterManager(
	st store.DataStore,
	mc *agent.ModelCache,
	stripeKey string,
	markup float64,
	freeTokens int,
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
		store:      st,
		modelCache: mc,
		stripeKey:  stripeKey,
		markup:     markup,
		freeTokens: freeTokens,
		logger:     logger.With("component", "billing"),
	}
}

// meterLock returns a per-model mutex for EnsureModelMeter, creating one if needed.
func (mm *MeterManager) meterLock(modelID string) *sync.Mutex {
	v, _ := mm.meterMu.LoadOrStore(modelID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// orgLock returns a per-org mutex, creating one if needed.
func (mm *MeterManager) orgLock(orgID string) *sync.Mutex {
	v, _ := mm.orgMu.LoadOrStore(orgID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// subItemLock returns a per-org:model mutex for subscription item creation.
func (mm *MeterManager) subItemLock(orgID, modelID string) *sync.Mutex {
	v, _ := mm.subItemMu.LoadOrStore(orgID+":"+modelID, &sync.Mutex{})
	return v.(*sync.Mutex)
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

// EnsureModelMeter returns an existing ModelMeter or creates Stripe objects and stores a new one.
func (mm *MeterManager) EnsureModelMeter(ctx context.Context, modelID string) (*store.ModelMeter, error) {
	existing, err := mm.store.GetModelMeter(ctx, modelID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("get model meter: %w", err)
	}

	mu := mm.meterLock(modelID)
	mu.Lock()
	defer mu.Unlock()

	// Double-check after acquiring lock
	existing, err = mm.store.GetModelMeter(ctx, modelID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("get model meter: %w", err)
	}

	// Look up model pricing
	var inputCostPerToken, outputCostPerToken float64
	models, fetchErr := mm.modelCache.GetModels(ctx)
	if fetchErr == nil {
		for _, m := range models {
			if m.ID == modelID {
				inputCostPerToken = m.InputCostPer1K / 1000.0
				outputCostPerToken = m.OutputCostPer1K / 1000.0
				break
			}
		}
	}
	if inputCostPerToken == 0 {
		inputCostPerToken = 0.000003  // fallback: $3/1M tokens
		outputCostPerToken = 0.000015 // fallback: $15/1M tokens
	}

	sanitized := sanitizeEventName(modelID)

	// Track created Stripe objects for rollback on partial failure.
	// NOTE: Stripe billing meters cannot be deleted or deactivated via API.
	// If a partial failure occurs after meters are created, orphaned meters
	// may remain in Stripe. These are harmless but should be cleaned up manually.
	var createdProductID string
	var createdPriceIDs []string
	rollback := func() {
		for _, priceID := range createdPriceIDs {
			if _, err := stripePrice.Update(priceID, &stripe.PriceParams{
				Active: stripe.Bool(false),
			}); err != nil {
				mm.logger.Warn("rollback: failed to deactivate price", "price_id", priceID, "error", err)
			}
		}
		if createdProductID != "" {
			if _, err := stripeProduct.Update(createdProductID, &stripe.ProductParams{
				Active: stripe.Bool(false),
			}); err != nil {
				mm.logger.Warn("rollback: failed to deactivate product", "product_id", createdProductID, "error", err)
			}
		}
	}

	// Create Stripe Product
	prod, err := stripeProduct.New(&stripe.ProductParams{
		Name: stripe.String(fmt.Sprintf("LLM Tokens: %s", modelID)),
		Params: stripe.Params{
			Metadata: map[string]string{
				"model_id": modelID,
				"type":     "llm_metered",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create stripe product: %w", err)
	}
	createdProductID = prod.ID

	// Create input meter
	inputEventName := sanitized + "_input"
	inputMeter, err := stripeBillingMeter.New(&stripe.BillingMeterParams{
		EventName:   stripe.String(inputEventName),
		DisplayName: stripe.String(fmt.Sprintf("%s Input Tokens", modelID)),
		DefaultAggregation: &stripe.BillingMeterDefaultAggregationParams{
			Formula: stripe.String(string(stripe.BillingMeterDefaultAggregationFormulaSum)),
		},
		CustomerMapping: &stripe.BillingMeterCustomerMappingParams{
			EventPayloadKey: stripe.String("stripe_customer_id"),
			Type:            stripe.String("by_id"),
		},
	})
	if err != nil {
		rollback()
		return nil, fmt.Errorf("create input meter: %w", err)
	}

	// Create output meter
	outputEventName := sanitized + "_output"
	outputMeter, err := stripeBillingMeter.New(&stripe.BillingMeterParams{
		EventName:   stripe.String(outputEventName),
		DisplayName: stripe.String(fmt.Sprintf("%s Output Tokens", modelID)),
		DefaultAggregation: &stripe.BillingMeterDefaultAggregationParams{
			Formula: stripe.String(string(stripe.BillingMeterDefaultAggregationFormulaSum)),
		},
		CustomerMapping: &stripe.BillingMeterCustomerMappingParams{
			EventPayloadKey: stripe.String("stripe_customer_id"),
			Type:            stripe.String("by_id"),
		},
	})
	if err != nil {
		rollback()
		return nil, fmt.Errorf("create output meter: %w", err)
	}

	// Create input price (per-token, with markup, in cents)
	inputCentsPer := inputCostPerToken * mm.markup * 100.0
	inputPrice, err := stripePrice.New(&stripe.PriceParams{
		Currency:          stripe.String(string(stripe.CurrencyUSD)),
		Product:           stripe.String(prod.ID),
		UnitAmountDecimal: stripe.Float64(inputCentsPer),
		BillingScheme:     stripe.String(string(stripe.PriceBillingSchemePerUnit)),
		Recurring: &stripe.PriceRecurringParams{
			Interval:  stripe.String(string(stripe.PriceRecurringIntervalMonth)),
			UsageType: stripe.String(string(stripe.PriceRecurringUsageTypeMetered)),
			Meter:     stripe.String(inputMeter.ID),
		},
	})
	if err != nil {
		rollback()
		return nil, fmt.Errorf("create input price: %w", err)
	}
	createdPriceIDs = append(createdPriceIDs, inputPrice.ID)

	// Create output price
	outputCentsPer := outputCostPerToken * mm.markup * 100.0
	outputPrice, err := stripePrice.New(&stripe.PriceParams{
		Currency:          stripe.String(string(stripe.CurrencyUSD)),
		Product:           stripe.String(prod.ID),
		UnitAmountDecimal: stripe.Float64(outputCentsPer),
		BillingScheme:     stripe.String(string(stripe.PriceBillingSchemePerUnit)),
		Recurring: &stripe.PriceRecurringParams{
			Interval:  stripe.String(string(stripe.PriceRecurringIntervalMonth)),
			UsageType: stripe.String(string(stripe.PriceRecurringUsageTypeMetered)),
			Meter:     stripe.String(outputMeter.ID),
		},
	})
	if err != nil {
		rollback()
		return nil, fmt.Errorf("create output price: %w", err)
	}
	createdPriceIDs = append(createdPriceIDs, outputPrice.ID)

	meter := &store.ModelMeter{
		ID:                  uuid.New().String(),
		ModelID:             modelID,
		StripeProductID:     prod.ID,
		StripeInputMeterID:  inputMeter.ID,
		StripeOutputMeterID: outputMeter.ID,
		StripeInputPriceID:  inputPrice.ID,
		StripeOutputPriceID: outputPrice.ID,
		InputEventName:      inputEventName,
		OutputEventName:     outputEventName,
		InputCostPerToken:   inputCostPerToken,
		OutputCostPerToken:  outputCostPerToken,
	}

	if err := mm.store.CreateModelMeter(ctx, meter); err != nil {
		rollback()
		return nil, fmt.Errorf("save model meter: %w", err)
	}

	mm.logger.Info("created stripe meters for model",
		"model", modelID,
		"product", prod.ID,
		"input_meter", inputMeter.ID,
		"output_meter", outputMeter.ID,
	)

	return meter, nil
}

// EnsureOrgSubscriptionItems adds subscription items for a model to an org's subscription.
func (mm *MeterManager) EnsureOrgSubscriptionItems(ctx context.Context, orgID, modelID string) error {
	_, err := mm.store.GetOrgModelSubscription(ctx, orgID, modelID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("get org model subscription: %w", err)
	}

	mu := mm.subItemLock(orgID, modelID)
	mu.Lock()
	defer mu.Unlock()

	// Double-check after acquiring lock
	_, err = mm.store.GetOrgModelSubscription(ctx, orgID, modelID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("get org model subscription: %w", err)
	}

	sub, err := mm.store.GetSubscriptionByOrg(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}
	if sub.Status != store.SubStatusActive || sub.StripeSubscriptionID == "" {
		return nil
	}

	meter, err := mm.EnsureModelMeter(ctx, modelID)
	if err != nil {
		return fmt.Errorf("ensure model meter: %w", err)
	}

	inputItem, err := stripeSubItem.New(&stripe.SubscriptionItemParams{
		Subscription: stripe.String(sub.StripeSubscriptionID),
		Price:        stripe.String(meter.StripeInputPriceID),
	})
	if err != nil {
		return fmt.Errorf("add input subscription item: %w", err)
	}

	outputItem, err := stripeSubItem.New(&stripe.SubscriptionItemParams{
		Subscription: stripe.String(sub.StripeSubscriptionID),
		Price:        stripe.String(meter.StripeOutputPriceID),
	})
	if err != nil {
		// Rollback: remove the input subscription item we just created
		if _, delErr := stripeSubItem.Del(inputItem.ID, &stripe.SubscriptionItemParams{}); delErr != nil {
			mm.logger.Warn("rollback: failed to delete input subscription item", "item_id", inputItem.ID, "error", delErr)
		}
		return fmt.Errorf("add output subscription item: %w", err)
	}

	oms := &store.OrgModelSubscription{
		ID:                    uuid.New().String(),
		OrgID:                 orgID,
		ModelID:               modelID,
		StripeInputSubItemID:  inputItem.ID,
		StripeOutputSubItemID: outputItem.ID,
	}

	if err := mm.store.CreateOrgModelSubscription(ctx, oms); err != nil {
		return fmt.Errorf("save org model subscription: %w", err)
	}

	mm.logger.Info("added subscription items for model",
		"org_id", orgID,
		"model", modelID,
		"input_item", inputItem.ID,
		"output_item", outputItem.ID,
	)

	return nil
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
		Identifier: stripe.String(uuid.New().String()),
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

// ReportUsage reports token usage to Stripe billing meters, respecting the free tier.
// The caller MUST persist the usage record before calling ReportUsage, because
// SumTokenUsage is expected to include the current chat's tokens in the cumulative total.
func (mm *MeterManager) ReportUsage(ctx context.Context, orgID, modelID string, inputTokens, outputTokens int) error {
	if inputTokens+outputTokens == 0 {
		return nil
	}

	org, err := mm.store.GetOrganization(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get org: %w", err)
	}
	if org.StripeCustomerID == "" {
		return nil
	}

	sub, err := mm.store.GetSubscriptionByOrg(ctx, orgID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get subscription: %w", err)
	}
	if sub.Status != store.SubStatusActive || sub.StripeSubscriptionID == "" {
		return nil
	}

	// Serialize free tier calculation + Stripe submission per org to prevent
	// concurrent requests from double-counting free tier allowance.
	omu := mm.orgLock(orgID)
	omu.Lock()
	defer omu.Unlock()

	// Calculate free tier
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	cumulative, err := mm.store.SumTokenUsage(ctx, orgID, startOfMonth, now)
	if err != nil {
		return fmt.Errorf("sum token usage: %w", err)
	}

	thisChat := float64(inputTokens + outputTokens)
	prevTotal := cumulative - thisChat // usage before this chat (already recorded by the time we get here)
	if prevTotal < 0 {
		mm.logger.Warn("free tier: cumulative < thisChat, clamping prevTotal to 0",
			"org_id", orgID, "cumulative", cumulative, "this_chat", thisChat)
		prevTotal = 0
	}

	var billableInput, billableOutput int
	freeTokens := float64(mm.freeTokens)

	if prevTotal >= freeTokens {
		billableInput = inputTokens
		billableOutput = outputTokens
	} else if prevTotal+thisChat <= freeTokens {
		return nil
	} else {
		excess := (prevTotal + thisChat) - freeTokens
		totalBillable := int(math.Round(excess))
		ratio := float64(inputTokens) / thisChat
		billableInput = int(math.Round(float64(totalBillable) * ratio))
		billableOutput = totalBillable - billableInput
		if billableOutput < 0 {
			billableOutput = 0
		}
	}

	if billableInput+billableOutput == 0 {
		return nil
	}

	if err := mm.EnsureOrgSubscriptionItems(ctx, orgID, modelID); err != nil {
		mm.logger.Warn("failed to ensure subscription items", "error", err)
	}

	meter, err := mm.EnsureModelMeter(ctx, modelID)
	if err != nil {
		return fmt.Errorf("ensure model meter: %w", err)
	}

	if billableInput > 0 {
		_, err := stripeMeterEvent.New(&stripe.BillingMeterEventParams{
			EventName: stripe.String(meter.InputEventName),
			Payload: map[string]string{
				"stripe_customer_id": org.StripeCustomerID,
				"value":              fmt.Sprintf("%d", billableInput),
			},
			Identifier: stripe.String(uuid.New().String()),
		})
		if err != nil {
			mm.logger.Warn("failed to report input meter event", "error", err, "model", modelID)
		}
	}

	if billableOutput > 0 {
		_, err := stripeMeterEvent.New(&stripe.BillingMeterEventParams{
			EventName: stripe.String(meter.OutputEventName),
			Payload: map[string]string{
				"stripe_customer_id": org.StripeCustomerID,
				"value":              fmt.Sprintf("%d", billableOutput),
			},
			Identifier: stripe.String(uuid.New().String()),
		})
		if err != nil {
			mm.logger.Warn("failed to report output meter event", "error", err, "model", modelID)
		}
	}

	mm.logger.Debug("reported usage to stripe",
		"org_id", orgID,
		"model", modelID,
		"billable_input", billableInput,
		"billable_output", billableOutput,
	)

	return nil
}
