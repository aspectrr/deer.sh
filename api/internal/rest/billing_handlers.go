package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"time"

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
	MaxConcurrentSandboxes float64 `json:"max_concurrent_sandboxes"`
	SourceVMs              float64 `json:"source_vms"`
	AgentHosts             float64 `json:"agent_hosts"`
	TokensUsed             float64 `json:"tokens_used"`
}

func (s *Server) handleGetBilling(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
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
			case "max_concurrent_sandboxes":
				summary.MaxConcurrentSandboxes += rec.Quantity
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
	org, _, ok := s.resolveOrgRole(w, r, store.OrgRoleOwner)
	if !ok {
		return
	}

	user := auth.UserFromContext(r.Context())

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
		if err := s.store.UpdateOrganization(r.Context(), org); err != nil {
			serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to save stripe customer"))
			return
		}
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
	org, _, ok := s.resolveOrgRole(w, r, store.OrgRoleOwner)
	if !ok {
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
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
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
	ConcurrentSandboxes int `json:"concurrent_sandboxes"`
	SourceVMs           int `json:"source_vms"`
	AgentHosts          int `json:"agent_hosts"`
}

type calculatorResponse struct {
	SandboxCost   float64 `json:"sandbox_cost"`
	SourceVMCost  float64 `json:"source_vm_cost"`
	AgentHostCost float64 `json:"agent_host_cost"`
	TotalMonthly  float64 `json:"total_monthly"`
	Currency      string  `json:"currency"`
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

	total := sandboxCost + sourceVMCost + agentHostCost

	_ = serverJSON.RespondJSON(w, http.StatusOK, calculatorResponse{
		SandboxCost:   math.Round(sandboxCost*100) / 100,
		SourceVMCost:  math.Round(sourceVMCost*100) / 100,
		AgentHostCost: math.Round(agentHostCost*100) / 100,
		TotalMonthly:  math.Round(total*100) / 100,
		Currency:      "USD",
	})
}

// --- Stripe Webhook ---

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Billing.StripeSecretKey == "" {
		_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "not_configured"})
		return
	}

	if s.cfg.Billing.StripeWebhookSecret == "" {
		_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "webhook_not_configured"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("failed to read body"))
		return
	}

	event, err := webhook.ConstructEventWithOptions(body, r.Header.Get("Stripe-Signature"), s.cfg.Billing.StripeWebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("webhook signature verification failed"))
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			slog.Error("webhook: unmarshal checkout session", "error", err)
			break
		}
		if sess.Customer != nil {
			if sess.Subscription == nil {
				slog.Warn("webhook: checkout session has no subscription")
				break
			}
			org, err := s.store.GetOrganizationByStripeCustomerID(r.Context(), sess.Customer.ID)
			if err != nil {
				slog.Error("webhook: lookup org by stripe customer", "error", err)
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
			slog.Error("webhook: unmarshal subscription update", "error", err)
			break
		}
		var existing *store.Subscription
		if sub.Customer != nil {
			org, err := s.store.GetOrganizationByStripeCustomerID(r.Context(), sub.Customer.ID)
			if err != nil {
				slog.Error("webhook: lookup org for subscription update", "error", err)
				break
			}
			existing, err = s.store.GetSubscriptionByOrg(r.Context(), org.ID)
			if err != nil {
				slog.Error("webhook: get subscription for update", "error", err)
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
		if err := s.store.UpdateSubscription(r.Context(), existing); err != nil {
			slog.Error("webhook: update subscription", "error", err)
			serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to update subscription"))
			return
		}

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			slog.Error("webhook: unmarshal subscription delete", "error", err)
			break
		}
		if sub.Customer != nil {
			org, err := s.store.GetOrganizationByStripeCustomerID(r.Context(), sub.Customer.ID)
			if err != nil {
				slog.Error("webhook: lookup org for subscription delete", "error", err)
				break
			}
			existing, err := s.store.GetSubscriptionByOrg(r.Context(), org.ID)
			if err != nil {
				slog.Error("webhook: get subscription for delete", "error", err)
				break
			}
			existing.Status = store.SubStatusCancelled
			if err := s.store.UpdateSubscription(r.Context(), existing); err != nil {
				slog.Error("webhook: cancel subscription", "error", err)
				serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to cancel subscription"))
				return
			}
		}
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "received"})
}
