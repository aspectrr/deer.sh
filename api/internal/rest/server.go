package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/aspectrr/fluid.sh/api/docs"

	"github.com/aspectrr/fluid.sh/api/internal/agent"
	"github.com/aspectrr/fluid.sh/api/internal/auth"
	"github.com/aspectrr/fluid.sh/api/internal/config"
	"github.com/aspectrr/fluid.sh/api/internal/orchestrator"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// swaggerError is the error response model for swagger documentation.
//
//nolint:unused // referenced by swag godoc annotations
type swaggerError struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

type Server struct {
	Router       *chi.Mux
	store        store.Store
	cfg          *config.Config
	orchestrator *orchestrator.Orchestrator
	agentClient  *agent.Client
}

func NewServer(st store.Store, cfg *config.Config, orch *orchestrator.Orchestrator, agentClient *agent.Client) *Server {
	s := &Server{
		store:        st,
		cfg:          cfg,
		orchestrator: orch,
		agentClient:  agentClient,
	}
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

	// Public routes
	r.Get("/v1/health", s.handleHealth)
	r.Get("/v1/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/v1/swagger/doc.json"),
	))

	// Auth routes (public)
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/register", s.handleRegister)
		r.Post("/login", s.handleLogin)

		// OAuth
		r.Get("/github", s.handleGitHubLogin)
		r.Get("/github/callback", s.handleGitHubCallback)
		r.Get("/google", s.handleGoogleLogin)
		r.Get("/google/callback", s.handleGoogleCallback)

		// Protected auth routes
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(s.store))
			r.Post("/logout", s.handleLogout)
			r.Get("/me", s.handleMe)
		})
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(s.store))

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

				// Agent
				r.Post("/agent/chat", s.handleAgentChat)
				r.Get("/agent/conversations", s.handleListConversations)
				r.Get("/agent/conversations/{conversationID}", s.handleGetConversation)
				r.Get("/agent/conversations/{conversationID}/messages", s.handleListMessages)
				r.Delete("/agent/conversations/{conversationID}", s.handleDeleteConversation)
				r.Get("/agent/models", s.handleListModels)

				// Playbooks
				r.Post("/playbooks", s.handleCreatePlaybook)
				r.Get("/playbooks", s.handleListPlaybooks)
				r.Route("/playbooks/{playbookID}", func(r chi.Router) {
					r.Get("/", s.handleGetPlaybook)
					r.Patch("/", s.handleUpdatePlaybook)
					r.Delete("/", s.handleDeletePlaybook)
					r.Post("/tasks", s.handleCreatePlaybookTask)
					r.Get("/tasks", s.handleListPlaybookTasks)
					r.Put("/tasks/reorder", s.handleReorderPlaybookTasks)
					r.Patch("/tasks/{taskID}", s.handleUpdatePlaybookTask)
					r.Delete("/tasks/{taskID}", s.handleDeletePlaybookTask)
				})
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

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
