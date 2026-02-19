package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	billingportal "github.com/stripe/stripe-go/v82/billingportal/session"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// --- Get Billing ---

type billingResponse struct {
	Plan     string        `json:"plan"`
	Status   string        `json:"status"`
	FreeTier *freeTierInfo `json:"free_tier,omitempty"`
	Usage    *usageSummary `json:"usage,omitempty"`
}

type freeTierInfo struct {
	MaxConcurrentSandboxes int `json:"max_concurrent_sandboxes"`
	MaxSourceVMs           int `json:"max_source_vms"`
	MaxAgentHosts          int `json:"max_agent_hosts"`
}

type usageSummary struct {
	SandboxHours float64 `json:"sandbox_hours"`
	SourceVMs    float64 `json:"source_vms"`
	AgentHosts   float64 `json:"agent_hosts"`
	TokensUsed   float64 `json:"tokens_used"`
}

func (s *Server) handleGetBilling(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	_, err = s.store.GetOrgMember(r.Context(), org.ID, user.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("not a member of this organization"))
		return
	}

	sub, err := s.store.GetSubscriptionByOrg(r.Context(), org.ID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get subscription"))
		return
	}

	resp := billingResponse{
		Plan:   string(store.PlanFree),
		Status: string(store.SubStatusActive),
		FreeTier: &freeTierInfo{
			MaxConcurrentSandboxes: s.cfg.Billing.FreeTier.MaxConcurrentSandboxes,
			MaxSourceVMs:           s.cfg.Billing.FreeTier.MaxSourceVMs,
			MaxAgentHosts:          s.cfg.Billing.FreeTier.MaxAgentHosts,
		},
	}

	if sub != nil {
		resp.Plan = string(sub.Plan)
		resp.Status = string(sub.Status)
		if sub.Plan == store.PlanUsageBased {
			resp.FreeTier = nil
		}
	}

	// Get current month usage
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	records, err := s.store.ListUsageRecords(r.Context(), org.ID, startOfMonth, now)
	if err == nil && len(records) > 0 {
		summary := &usageSummary{}
		for _, rec := range records {
			switch rec.ResourceType {
			case "sandbox_hour":
				summary.SandboxHours += rec.Quantity
			case "source_vm":
				summary.SourceVMs += rec.Quantity
			case "agent_host":
				summary.AgentHosts += rec.Quantity
			case "llm_token":
				summary.TokensUsed += rec.Quantity
			}
		}
		resp.Usage = summary
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, resp)
}

// --- Subscribe ---

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	if org.OwnerID != user.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("only the owner can manage billing"))
		return
	}

	if s.cfg.Billing.StripeSecretKey == "" || s.cfg.Billing.StripePriceID == "" {
		_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{
			"message": "Stripe integration pending configuration",
			"status":  "not_configured",
		})
		return
	}

	// Create or reuse Stripe customer
	customerID := org.StripeCustomerID
	if customerID == "" {
		cust, err := customer.New(&stripe.CustomerParams{
			Email: stripe.String(user.Email),
			Name:  stripe.String(org.Name),
			Params: stripe.Params{
				Metadata: map[string]string{
					"org_id": org.ID,
				},
			},
		})
		if err != nil {
			serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create Stripe customer"))
			return
		}
		customerID = cust.ID
		org.StripeCustomerID = customerID
		_ = s.store.UpdateOrganization(r.Context(), org)
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(s.cfg.Billing.StripePriceID), // usage-based price ID
				Quantity: nil,                                        // usage-based - no fixed quantity
			},
		},
		SuccessURL: stripe.String(s.cfg.Frontend.URL + "/billing?success=true"),
		CancelURL:  stripe.String(s.cfg.Frontend.URL + "/billing?canceled=true"),
	}

	sess, err := session.New(params)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create checkout session"))
		return
	}

	s.telemetry.Track(user.ID, "billing_subscribed", map[string]any{"org_id": org.ID})

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{
		"checkout_url": sess.URL,
	})
}

// --- Billing Portal ---

func (s *Server) handleBillingPortal(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	if org.OwnerID != user.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("only the owner can manage billing"))
		return
	}

	if s.cfg.Billing.StripeSecretKey == "" || org.StripeCustomerID == "" {
		_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{
			"message": "Stripe integration pending configuration",
			"status":  "not_configured",
		})
		return
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(org.StripeCustomerID),
		ReturnURL: stripe.String(s.cfg.Frontend.URL + "/billing"),
	}

	sess, err := billingportal.New(params)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create billing portal session"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{
		"portal_url": sess.URL,
	})
}

// --- Usage ---

func (s *Server) handleGetUsage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return
	}

	_, err = s.store.GetOrgMember(r.Context(), org.ID, user.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("not a member of this organization"))
		return
	}

	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	records, err := s.store.ListUsageRecords(r.Context(), org.ID, startOfMonth, now)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get usage records"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"records": records,
		"total":   len(records),
	})
}

// --- Calculator (public) ---

type calculatorRequest struct {
	ConcurrentSandboxes int    `json:"concurrent_sandboxes"`
	SourceVMs           int    `json:"source_vms"`
	AgentHosts          int    `json:"agent_hosts"`
	EstimatedTokens     int    `json:"estimated_tokens"`
	Model               string `json:"model"`
}

type calculatorResponse struct {
	SandboxCost    float64         `json:"sandbox_cost"`
	SourceVMCost   float64         `json:"source_vm_cost"`
	AgentHostCost  float64         `json:"agent_host_cost"`
	TokenCost      float64         `json:"token_cost"`
	TokenBreakdown *tokenBreakdown `json:"token_breakdown,omitempty"`
	TotalMonthly   float64         `json:"total_monthly"`
	Currency       string          `json:"currency"`
}

type tokenBreakdown struct {
	EstimatedTokens int     `json:"estimated_tokens"`
	FreeTokens      int     `json:"free_tokens"`
	BillableTokens  int     `json:"billable_tokens"`
	CostPerToken    float64 `json:"cost_per_token"`
	Markup          float64 `json:"markup_percent"`
}

func (s *Server) handleCalculator(w http.ResponseWriter, r *http.Request) {
	var req calculatorRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	prices := s.cfg.Billing.Prices
	freeTier := s.cfg.Billing.FreeTier

	// Apply free tier deductions (min 0 billable)
	billableSandboxes := req.ConcurrentSandboxes - freeTier.MaxConcurrentSandboxes
	if billableSandboxes < 0 {
		billableSandboxes = 0
	}
	billableSourceVMs := req.SourceVMs - freeTier.MaxSourceVMs
	if billableSourceVMs < 0 {
		billableSourceVMs = 0
	}
	billableAgentHosts := req.AgentHosts - freeTier.MaxAgentHosts
	if billableAgentHosts < 0 {
		billableAgentHosts = 0
	}

	sandboxCost := float64(billableSandboxes) * float64(prices.SandboxMonthlyCents) / 100.0
	sourceVMCost := float64(billableSourceVMs) * float64(prices.SourceVMMonthly) / 100.0
	agentHostCost := float64(billableAgentHosts) * float64(prices.AgentHostMonthly) / 100.0

	// Token cost calculation
	var tokenCost float64
	var tb *tokenBreakdown
	freeTokens := s.cfg.Agent.FreeTokensPerMonth

	if req.EstimatedTokens > 0 {
		billable := req.EstimatedTokens - freeTokens
		if billable < 0 {
			billable = 0
		}

		// Default cost per 1k tokens (use model pricing if specified)
		costPer1K := 0.003 // default: Claude Sonnet input rate
		if req.Model != "" {
			for _, m := range modelPricing {
				if m.id == req.Model {
					costPer1K = (m.inputCostPer1K + m.outputCostPer1K) / 2
					break
				}
			}
		}

		markup := s.cfg.Billing.BillingMarkup
		if markup == 0 {
			markup = 1.05
		}
		tokenCost = float64(billable) * costPer1K / 1000.0 * markup
		tb = &tokenBreakdown{
			EstimatedTokens: req.EstimatedTokens,
			FreeTokens:      freeTokens,
			BillableTokens:  billable,
			CostPerToken:    costPer1K / 1000.0,
			Markup:          5.0,
		}
	}

	total := sandboxCost + sourceVMCost + agentHostCost + tokenCost

	_ = serverJSON.RespondJSON(w, http.StatusOK, calculatorResponse{
		SandboxCost:    math.Round(sandboxCost*100) / 100,
		SourceVMCost:   math.Round(sourceVMCost*100) / 100,
		AgentHostCost:  math.Round(agentHostCost*100) / 100,
		TokenCost:      math.Round(tokenCost*100) / 100,
		TokenBreakdown: tb,
		TotalMonthly:   math.Round(total*100) / 100,
		Currency:       "USD",
	})
}

type modelPrice struct {
	id              string
	inputCostPer1K  float64
	outputCostPer1K float64
}

var modelPricing = []modelPrice{
	{"anthropic/claude-sonnet-4", 0.003, 0.015},
	{"anthropic/claude-haiku-4", 0.0008, 0.004},
	{"openai/gpt-4o", 0.0025, 0.01},
	{"openai/gpt-4o-mini", 0.00015, 0.0006},
	{"google/gemini-2.5-pro", 0.00125, 0.01},
}

// --- Stripe Webhook ---

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Billing.StripeSecretKey == "" {
		_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "not_configured"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("failed to read body"))
		return
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), s.cfg.Billing.StripeWebhookSecret)
	if err != nil {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("webhook signature verification failed"))
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			break
		}
		if sess.Customer != nil {
			if sess.Subscription == nil {
				break
			}
			org, err := s.store.GetOrganizationByStripeCustomerID(r.Context(), sess.Customer.ID)
			if err != nil {
				break
			}
			// Idempotency: check if subscription already exists
			if _, err := s.store.GetSubscriptionByStripeID(r.Context(), sess.Subscription.ID); err == nil {
				break // already processed
			}
			sub := &store.Subscription{
				ID:                   uuid.New().String(),
				OrgID:                org.ID,
				Plan:                 store.PlanUsageBased,
				StripeSubscriptionID: sess.Subscription.ID,
				Status:               store.SubStatusActive,
				CurrentPeriodStart:   time.Now().UTC(),
				CurrentPeriodEnd:     time.Now().UTC().AddDate(0, 1, 0),
			}
			if err := s.store.CreateSubscription(r.Context(), sub); err != nil {
				serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create subscription"))
				return
			}
		}

	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			break
		}
		var existing *store.Subscription
		if sub.Customer != nil {
			org, err := s.store.GetOrganizationByStripeCustomerID(r.Context(), sub.Customer.ID)
			if err != nil {
				break
			}
			existing, err = s.store.GetSubscriptionByOrg(r.Context(), org.ID)
			if err != nil {
				break
			}
		} else {
			break
		}
		newStatus := store.SubscriptionStatus(sub.Status)
		switch newStatus {
		case store.SubStatusActive, store.SubStatusPastDue, store.SubStatusCancelled:
			existing.Status = newStatus
		default:
			existing.Status = store.SubStatusPastDue
		}
		_ = s.store.UpdateSubscription(r.Context(), existing)

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			break
		}
		if sub.Customer != nil {
			org, err := s.store.GetOrganizationByStripeCustomerID(r.Context(), sub.Customer.ID)
			if err != nil {
				break
			}
			existing, err := s.store.GetSubscriptionByOrg(r.Context(), org.ID)
			if err != nil {
				break
			}
			existing.Status = store.SubStatusCancelled
			_ = s.store.UpdateSubscription(r.Context(), existing)
		}
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "received"})
}
