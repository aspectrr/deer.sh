package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	API          APIConfig
	GRPC         GRPCConfig
	Database     DatabaseConfig
	Auth         AuthConfig
	Frontend     FrontendConfig
	Billing      BillingConfig
	Agent        AgentConfig
	Orchestrator OrchestratorConfig
	Logging      LoggingConfig
}

type GRPCConfig struct {
	Address     string
	TLSCertFile string
	TLSKeyFile  string
}

type OrchestratorConfig struct {
	HeartbeatTimeout time.Duration
	DefaultTTL       time.Duration
}

type APIConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	EnableDocs      bool
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	AutoMigrate     bool
}

type AuthConfig struct {
	SessionTTL    time.Duration
	SecureCookies bool
	GitHub        OAuthProviderConfig
	Google        OAuthProviderConfig
}

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type FrontendConfig struct {
	URL string
}

type BillingConfig struct {
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string
	StripePriceID        string
	Prices               PriceConfig
	FreeTier             FreeTierConfig
}

type PriceConfig struct {
	SandboxHourCents int
	SourceVMMonthly  int
	AgentHostMonthly int
}

type FreeTierConfig struct {
	MaxConcurrentSandboxes int
	MaxSourceVMs           int
	MaxAgentHosts          int
}

type AgentConfig struct {
	OpenRouterAPIKey    string
	OpenRouterBaseURL   string
	DefaultModel        string
	MaxTokensPerRequest int
	FreeTokensPerMonth  int
}

type LoggingConfig struct {
	Level  string
	Format string
}

func Load() *Config {
	return &Config{
		API: APIConfig{
			Addr:            envOr("API_ADDR", ":8081"),
			ReadTimeout:     envDuration("API_READ_TIMEOUT", 60*time.Second),
			WriteTimeout:    envDuration("API_WRITE_TIMEOUT", 120*time.Second),
			IdleTimeout:     envDuration("API_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout: envDuration("API_SHUTDOWN_TIMEOUT", 20*time.Second),
			EnableDocs:      envBool("API_ENABLE_DOCS", true),
		},
		Database: DatabaseConfig{
			URL:             os.Getenv("DATABASE_URL"),
			MaxOpenConns:    envInt("DATABASE_MAX_OPEN_CONNS", 16),
			MaxIdleConns:    envInt("DATABASE_MAX_IDLE_CONNS", 8),
			ConnMaxLifetime: envDuration("DATABASE_CONN_MAX_LIFETIME", time.Hour),
			AutoMigrate:     envBool("DATABASE_AUTO_MIGRATE", false),
		},
		Auth: AuthConfig{
			SessionTTL:    envDuration("AUTH_SESSION_TTL", 720*time.Hour),
			SecureCookies: envBool("AUTH_SECURE_COOKIES", true),
			GitHub: OAuthProviderConfig{
				ClientID:     os.Getenv("AUTH_GITHUB_CLIENT_ID"),
				ClientSecret: os.Getenv("AUTH_GITHUB_CLIENT_SECRET"),
				RedirectURL:  envOr("AUTH_GITHUB_REDIRECT_URL", "http://localhost:5173/v1/auth/github/callback"),
			},
			Google: OAuthProviderConfig{
				ClientID:     os.Getenv("AUTH_GOOGLE_CLIENT_ID"),
				ClientSecret: os.Getenv("AUTH_GOOGLE_CLIENT_SECRET"),
				RedirectURL:  envOr("AUTH_GOOGLE_REDIRECT_URL", "http://localhost:5173/v1/auth/google/callback"),
			},
		},
		GRPC: GRPCConfig{
			Address:     envOr("GRPC_ADDR", ":9090"),
			TLSCertFile: os.Getenv("GRPC_TLS_CERT_FILE"),
			TLSKeyFile:  os.Getenv("GRPC_TLS_KEY_FILE"),
		},
		Orchestrator: OrchestratorConfig{
			HeartbeatTimeout: envDuration("ORCHESTRATOR_HEARTBEAT_TIMEOUT", 90*time.Second),
			DefaultTTL:       envDuration("ORCHESTRATOR_DEFAULT_TTL", 24*time.Hour),
		},
		Frontend: FrontendConfig{
			URL: envOr("FRONTEND_URL", "http://localhost:5173"),
		},
		Billing: BillingConfig{
			StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
			StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
			StripePublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
			StripePriceID:        os.Getenv("STRIPE_PRICE_ID"),
			Prices: PriceConfig{
				SandboxHourCents: envInt("BILLING_SANDBOX_HOUR_CENTS", 5),
				SourceVMMonthly:  envInt("BILLING_SOURCE_VM_MONTHLY_CENTS", 500),
				AgentHostMonthly: envInt("BILLING_AGENT_HOST_MONTHLY_CENTS", 1000),
			},
			FreeTier: FreeTierConfig{
				MaxConcurrentSandboxes: envInt("BILLING_FREE_TIER_MAX_SANDBOXES", 1),
				MaxSourceVMs:           envInt("BILLING_FREE_TIER_MAX_SOURCE_VMS", 3),
				MaxAgentHosts:          envInt("BILLING_FREE_TIER_MAX_AGENT_HOSTS", 1),
			},
		},
		Agent: AgentConfig{
			OpenRouterAPIKey:    os.Getenv("OPENROUTER_API_KEY"),
			OpenRouterBaseURL:   envOr("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
			DefaultModel:        envOr("AGENT_DEFAULT_MODEL", "anthropic/claude-sonnet-4"),
			MaxTokensPerRequest: envInt("AGENT_MAX_TOKENS_PER_REQUEST", 8192),
			FreeTokensPerMonth:  envInt("AGENT_FREE_TOKENS_PER_MONTH", 100000),
		},
		Logging: LoggingConfig{
			Level:  envOr("LOG_LEVEL", "info"),
			Format: envOr("LOG_FORMAT", "text"),
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			slog.Warn("invalid integer for env var, using default", "key", key, "value", v, "default", fallback)
			return fallback
		}
		return n
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			slog.Warn("invalid boolean for env var, using default", "key", key, "value", v, "default", fallback)
			return fallback
		}
		return b
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			slog.Warn("invalid duration for env var, using default", "key", key, "value", v, "default", fallback)
			return fallback
		}
		return d
	}
	return fallback
}
