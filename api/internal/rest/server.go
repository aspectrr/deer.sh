package rest

import (
	"fmt"
	"log/slog"
	"net/http"

	scalar "github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	"github.com/aspectrr/fluid.sh/api/internal/config"
	"github.com/aspectrr/fluid.sh/api/internal/orchestrator"
	"github.com/aspectrr/fluid.sh/api/internal/store"
	"github.com/aspectrr/fluid.sh/api/internal/telemetry"
)

type Server struct {
	Router       *chi.Mux
	store        store.Store
	cfg          *config.Config
	orchestrator *orchestrator.Orchestrator
	telemetry    telemetry.Service
	logger       *slog.Logger
	openapiYAML  []byte
}

func NewServer(st store.Store, cfg *config.Config, orch *orchestrator.Orchestrator, tel telemetry.Service, openapiYAML []byte) *Server {
	if tel == nil {
		tel = &telemetry.NoopService{}
	}
	s := &Server{
		store:        st,
		cfg:          cfg,
		orchestrator: orch,
		telemetry:    tel,
		logger:       slog.Default().With("component", "rest"),
		openapiYAML:  openapiYAML,
	}
	// stripe.Key is set once in billing.NewMeterManager to avoid race conditions.

	s.Router = s.routes()
	return s
}

func (s *Server) routes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(s.cfg.Frontend.URL))

	trustedNets := parseCIDRs(s.cfg.API.TrustedProxies, s.logger)

	// Public routes
	r.Get("/v1/health", s.handleHealth)

	r.Get("/v1/docs/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		_, _ = w.Write(s.openapiYAML)
	})

	if s.cfg.API.EnableDocs {
		r.Get("/v1/docs", func(w http.ResponseWriter, r *http.Request) {
			html, err := scalar.ApiReferenceHTML(&scalar.Options{
				SpecURL: "/v1/docs/openapi.yaml",
				CustomOptions: scalar.CustomOptions{
					PageTitle: "Fluid API Reference",
				},
				DarkMode: true,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintln(w, html)
		})
	}

	// Auth routes (public)
	r.Route("/v1/auth", func(r chi.Router) {
		r.With(rateLimitByIP(0.1, 5, trustedNets)).Post("/register", s.handleRegister)
		r.With(rateLimitByIP(0.2, 10, trustedNets)).Post("/login", s.handleLogin)

		// OAuth (rate-limited)
		r.Group(func(r chi.Router) {
			r.Use(rateLimitByIP(0.5, 10, trustedNets))
			r.Get("/github", s.handleGitHubLogin)
			r.Get("/github/callback", s.handleGitHubCallback)
			r.Get("/google", s.handleGoogleLogin)
			r.Get("/google/callback", s.handleGoogleCallback)
		})

		// Protected auth routes
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(s.store, s.cfg.Auth.SecureCookies))
			r.Post("/logout", s.handleLogout)
			r.Get("/me", s.handleMe)
			r.Post("/onboarding", s.handleOnboarding)
		})
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(s.store, s.cfg.Auth.SecureCookies))

		// Organizations
		r.Route("/v1/orgs", func(r chi.Router) {
			r.Post("/", s.handleCreateOrg)
			r.Get("/", s.handleListOrgs)
			r.Route("/{slug}", func(r chi.Router) {
				r.Get("/", s.handleGetOrg)
				r.Patch("/", s.handleUpdateOrg)
				r.Delete("/", s.handleDeleteOrg)

				// Members
				r.Get("/members", s.handleListMembers)
				r.Post("/members", s.handleAddMember)
				r.Delete("/members/{memberID}", s.handleRemoveMember)

				// Billing
				r.Get("/billing", s.handleGetBilling)
				r.Post("/billing/subscribe", s.handleSubscribe)
				r.Post("/billing/portal", s.handleBillingPortal)
				r.Get("/billing/usage", s.handleGetUsage)

				// Sandboxes
				r.Post("/sandboxes", s.handleCreateSandbox)
				r.Get("/sandboxes", s.handleListSandboxes)
				r.Route("/sandboxes/{sandboxID}", func(r chi.Router) {
					r.Get("/", s.handleGetSandbox)
					r.Delete("/", s.handleDestroySandbox)
					r.Post("/run", s.handleRunCommand)
					r.Post("/start", s.handleStartSandbox)
					r.Post("/stop", s.handleStopSandbox)
					r.Get("/ip", s.handleGetSandboxIP)
					r.Post("/snapshot", s.handleCreateSnapshot)
					r.Get("/commands", s.handleListCommands)
				})

				// Hosts + tokens
				r.Get("/hosts", s.handleListHosts)
				r.Get("/hosts/{hostID}", s.handleGetHost)
				r.Post("/hosts/tokens", s.handleCreateHostToken)
				r.Get("/hosts/tokens", s.handleListHostTokens)
				r.Delete("/hosts/tokens/{tokenID}", s.handleDeleteHostToken)

				// Source Hosts
				r.Post("/source-hosts/discover", s.handleDiscoverSourceHosts)
				r.Post("/source-hosts", s.handleConfirmSourceHosts)
				r.Get("/source-hosts", s.handleListSourceHosts)
				r.Delete("/source-hosts/{sourceHostID}", s.handleDeleteSourceHost)

				// Source VMs
				r.Get("/vms", s.handleListVMs)
				r.Post("/sources/{vm}/prepare", s.handlePrepareSourceVM)
				r.Post("/sources/{vm}/run", s.handleRunSourceCommand)
				r.Post("/sources/{vm}/read", s.handleReadSourceFile)

				// Agent - commented out, not yet ready for integration
				// r.Post("/agent/chat", s.handleAgentChat)
				// r.Get("/agent/conversations", s.handleListConversations)
				// r.Get("/agent/conversations/{conversationID}", s.handleGetConversation)
				// r.Get("/agent/conversations/{conversationID}/messages", s.handleListMessages)
				// r.Delete("/agent/conversations/{conversationID}", s.handleDeleteConversation)
				// r.Get("/agent/models", s.handleListModels)

				// Playbooks - commented out, not yet ready for integration
				// r.Post("/playbooks", s.handleCreatePlaybook)
				// r.Get("/playbooks", s.handleListPlaybooks)
				// r.Route("/playbooks/{playbookID}", func(r chi.Router) {
				// 	r.Get("/", s.handleGetPlaybook)
				// 	r.Patch("/", s.handleUpdatePlaybook)
				// 	r.Delete("/", s.handleDeletePlaybook)
				// 	r.Post("/tasks", s.handleCreatePlaybookTask)
				// 	r.Get("/tasks", s.handleListPlaybookTasks)
				// 	r.Put("/tasks/reorder", s.handleReorderPlaybookTasks)
				// 	r.Patch("/tasks/{taskID}", s.handleUpdatePlaybookTask)
				// 	r.Delete("/tasks/{taskID}", s.handleDeletePlaybookTask)
				// })
			})
		})
	})

	// Docs progress (public, ephemeral session codes)
	r.Post("/v1/docs-progress/register", s.handleDocsProgressRegister)
	r.Post("/v1/docs-progress/complete", s.handleDocsProgressComplete)
	r.Get("/v1/docs-progress/progress", s.handleDocsProgressGet)

	// Public billing endpoints
	r.Post("/v1/billing/calculator", s.handleCalculator)
	r.Post("/v1/webhooks/stripe", s.handleStripeWebhook)

	return r
}

func corsMiddleware(frontendURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", frontendURL)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Add("Vary", "Origin")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
