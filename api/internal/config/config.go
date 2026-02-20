package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	API      APIConfig
	GRPC     GRPCConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Frontend FrontendConfig
	Billing  BillingConfig
	// Agent AgentConfig - commented out, not yet ready for integration
	Orchestrator  OrchestratorConfig
	Logging       LoggingConfig
	PostHog       PostHogConfig
	EncryptionKey string
}

type PostHogConfig struct {
	APIKey   string
	Endpoint string
}

type GRPCConfig struct {
	Address       string
	TLSCertFile   string
	TLSKeyFile    string
	AllowInsecure bool
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
	TrustedProxies  []string
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
	BillingMarkup        float64
}

type PriceConfig struct {
	SandboxMonthlyCents int
	SourceVMMonthly     int
	AgentHostMonthly    int
}

type FreeTierConfig struct {
	MaxConcurrentSandboxes int
	MaxSourceVMs           int
	MaxAgentHosts          int
}

// AgentConfig - commented out, not yet ready for integration.
/*
type AgentConfig struct {
	OpenRouterAPIKey    string
	OpenRouterBaseURL   string
	DefaultModel        string
	MaxTokensPerRequest int
	FreeTokensPerMonth  int
}
*/

type LoggingConfig struct {
	Level  string
	Format string
}

// Validate checks that required configuration fields are set and valid.
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.Frontend.URL == "*" {
		return fmt.Errorf("FRONTEND_URL must not be '*'")
	}
	if u, err := url.Parse(c.Frontend.URL); err != nil || u.Scheme == "" {
		return fmt.Errorf("FRONTEND_URL must be a valid URL")
	}
	if c.GRPC.TLSCertFile == "" && c.GRPC.TLSKeyFile == "" && !c.GRPC.AllowInsecure {
		return fmt.Errorf("gRPC TLS not configured; set GRPC_TLS_CERT_FILE/GRPC_TLS_KEY_FILE or GRPC_ALLOW_INSECURE=true")
	}
	if c.EncryptionKey == "" {
		slog.Warn("ENCRYPTION_KEY not set: OAuth tokens and Proxmox secrets will be stored in plaintext")
	}
	return nil
}

func Load() *Config {
	return &Config{
		API: APIConfig{
			Addr:            envOr("API_ADDR", ":8080"),
			ReadTimeout:     envDuration("API_READ_TIMEOUT", 60*time.Second),
			WriteTimeout:    envDuration("API_WRITE_TIMEOUT", 120*time.Second),
			IdleTimeout:     envDuration("API_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout: envDuration("API_SHUTDOWN_TIMEOUT", 20*time.Second),
			EnableDocs:      envBool("API_ENABLE_DOCS", false),
			TrustedProxies:  envStringSlice("TRUSTED_PROXIES"),
		},
		Database: DatabaseConfig{
			URL:             os.Getenv("DATABASE_URL"),
			MaxOpenConns:    envInt("DATABASE_MAX_OPEN_CONNS", 16),
			MaxIdleConns:    envInt("DATABASE_MAX_IDLE_CONNS", 8),
			ConnMaxLifetime: envDuration("DATABASE_CONN_MAX_LIFETIME", time.Hour),
			AutoMigrate:     envBool("DATABASE_AUTO_MIGRATE", false),
		},
		Auth: AuthConfig{
			SessionTTL:    envDuration("AUTH_SESSION_TTL", 168*time.Hour),
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
			Address:       envOr("GRPC_ADDR", ":9090"),
			TLSCertFile:   os.Getenv("GRPC_TLS_CERT_FILE"),
			TLSKeyFile:    os.Getenv("GRPC_TLS_KEY_FILE"),
			AllowInsecure: envBool("GRPC_ALLOW_INSECURE", false),
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
				SandboxMonthlyCents: envInt("BILLING_SANDBOX_MONTHLY_CENTS", 5000),
				SourceVMMonthly:     envInt("BILLING_SOURCE_VM_MONTHLY_CENTS", 500),
				AgentHostMonthly:    envInt("BILLING_AGENT_HOST_MONTHLY_CENTS", 1000),
			},
			FreeTier: FreeTierConfig{
				MaxConcurrentSandboxes: envInt("BILLING_FREE_TIER_MAX_SANDBOXES", 1),
				MaxSourceVMs:           envInt("BILLING_FREE_TIER_MAX_SOURCE_VMS", 3),
				MaxAgentHosts:          envInt("BILLING_FREE_TIER_MAX_AGENT_HOSTS", 1),
			},
			BillingMarkup: envFloat("BILLING_MARKUP", 1.05),
		},
		// Agent config - commented out, not yet ready for integration.
		// Agent: AgentConfig{
		// 	OpenRouterAPIKey:    os.Getenv("OPENROUTER_API_KEY"),
		// 	OpenRouterBaseURL:   envOr("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
		// 	DefaultModel:        envOr("AGENT_DEFAULT_MODEL", "anthropic/claude-sonnet-4"),
		// 	MaxTokensPerRequest: envInt("AGENT_MAX_TOKENS_PER_REQUEST", 8192),
		// 	FreeTokensPerMonth:  envInt("AGENT_FREE_TOKENS_PER_MONTH", 100000),
		// },
		Logging: LoggingConfig{
			Level:  envOr("LOG_LEVEL", "info"),
			Format: envOr("LOG_FORMAT", "text"),
		},
		PostHog: PostHogConfig{
			APIKey:   os.Getenv("POSTHOG_API_KEY"),
			Endpoint: envOr("POSTHOG_ENDPOINT", "https://nautilus.fluid.sh"),
		},
		EncryptionKey: os.Getenv("ENCRYPTION_KEY"),
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

func envStringSlice(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			slog.Warn("invalid float for env var, using default", "key", key, "value", v, "default", fallback)
			return fallback
		}
		return f
	}
	return fallback
}
