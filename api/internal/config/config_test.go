package config

import (
	"testing"
	"time"
)

func TestEnvOr_WithValue(t *testing.T) {
	t.Setenv("TEST_ENV_OR_KEY", "custom_value")

	got := envOr("TEST_ENV_OR_KEY", "default_value")
	if got != "custom_value" {
		t.Errorf("expected 'custom_value', got %q", got)
	}
}

func TestEnvOr_Fallback(t *testing.T) {
	// TEST_ENV_OR_MISSING is not set
	got := envOr("TEST_ENV_OR_MISSING", "fallback")
	if got != "fallback" {
		t.Errorf("expected 'fallback', got %q", got)
	}
}

func TestEnvOr_EmptyString(t *testing.T) {
	t.Setenv("TEST_ENV_OR_EMPTY", "")

	got := envOr("TEST_ENV_OR_EMPTY", "fallback")
	if got != "fallback" {
		t.Errorf("expected 'fallback' when env is empty string, got %q", got)
	}
}

func TestEnvInt_ValidInt(t *testing.T) {
	t.Setenv("TEST_ENV_INT", "42")

	got := envInt("TEST_ENV_INT", 10)
	if got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestEnvInt_InvalidInt(t *testing.T) {
	t.Setenv("TEST_ENV_INT_BAD", "notanumber")

	got := envInt("TEST_ENV_INT_BAD", 10)
	if got != 10 {
		t.Errorf("expected fallback 10, got %d", got)
	}
}

func TestEnvInt_Missing(t *testing.T) {
	got := envInt("TEST_ENV_INT_MISSING", 99)
	if got != 99 {
		t.Errorf("expected fallback 99, got %d", got)
	}
}

func TestEnvBool_True(t *testing.T) {
	t.Setenv("TEST_ENV_BOOL", "true")

	got := envBool("TEST_ENV_BOOL", false)
	if got != true {
		t.Error("expected true, got false")
	}
}

func TestEnvBool_False(t *testing.T) {
	t.Setenv("TEST_ENV_BOOL_F", "false")

	got := envBool("TEST_ENV_BOOL_F", true)
	if got != false {
		t.Error("expected false, got true")
	}
}

func TestEnvBool_Invalid(t *testing.T) {
	t.Setenv("TEST_ENV_BOOL_BAD", "notabool")

	got := envBool("TEST_ENV_BOOL_BAD", true)
	if got != true {
		t.Error("expected fallback true, got false")
	}
}

func TestEnvBool_Missing(t *testing.T) {
	got := envBool("TEST_ENV_BOOL_MISSING", true)
	if got != true {
		t.Error("expected fallback true, got false")
	}
}

func TestEnvDuration_Valid(t *testing.T) {
	t.Setenv("TEST_ENV_DUR", "5s")

	got := envDuration("TEST_ENV_DUR", time.Minute)
	if got != 5*time.Second {
		t.Errorf("expected 5s, got %v", got)
	}
}

func TestEnvDuration_Complex(t *testing.T) {
	t.Setenv("TEST_ENV_DUR_COMPLEX", "2h30m")

	got := envDuration("TEST_ENV_DUR_COMPLEX", time.Minute)
	expected := 2*time.Hour + 30*time.Minute
	if got != expected {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestEnvDuration_Invalid(t *testing.T) {
	t.Setenv("TEST_ENV_DUR_BAD", "notaduration")

	got := envDuration("TEST_ENV_DUR_BAD", 30*time.Second)
	if got != 30*time.Second {
		t.Errorf("expected fallback 30s, got %v", got)
	}
}

func TestEnvDuration_Missing(t *testing.T) {
	got := envDuration("TEST_ENV_DUR_MISSING", time.Hour)
	if got != time.Hour {
		t.Errorf("expected fallback 1h, got %v", got)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might interfere with defaults
	t.Setenv("API_ADDR", "")
	t.Setenv("GRPC_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")
	t.Setenv("OPENROUTER_BASE_URL", "")
	t.Setenv("FRONTEND_URL", "")
	t.Setenv("DATABASE_AUTO_MIGRATE", "")

	cfg := Load()

	if cfg.API.Addr != ":8080" {
		t.Errorf("expected API.Addr ':8080', got %q", cfg.API.Addr)
	}
	if cfg.API.ReadTimeout != 60*time.Second {
		t.Errorf("expected API.ReadTimeout 60s, got %v", cfg.API.ReadTimeout)
	}
	if cfg.API.WriteTimeout != 120*time.Second {
		t.Errorf("expected API.WriteTimeout 120s, got %v", cfg.API.WriteTimeout)
	}
	if cfg.GRPC.Address != ":9090" {
		t.Errorf("expected GRPC.Address ':9090', got %q", cfg.GRPC.Address)
	}
	if cfg.Database.MaxOpenConns != 16 {
		t.Errorf("expected Database.MaxOpenConns 16, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 8 {
		t.Errorf("expected Database.MaxIdleConns 8, got %d", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.AutoMigrate != false {
		t.Error("expected Database.AutoMigrate false, got true")
	}
	if cfg.Orchestrator.HeartbeatTimeout != 90*time.Second {
		t.Errorf("expected Orchestrator.HeartbeatTimeout 90s, got %v", cfg.Orchestrator.HeartbeatTimeout)
	}
	if cfg.Orchestrator.DefaultTTL != 24*time.Hour {
		t.Errorf("expected Orchestrator.DefaultTTL 24h, got %v", cfg.Orchestrator.DefaultTTL)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected Logging.Level 'info', got %q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("expected Logging.Format 'text', got %q", cfg.Logging.Format)
	}
	if cfg.Frontend.URL != "http://localhost:5173" {
		t.Errorf("expected Frontend.URL 'http://localhost:5173', got %q", cfg.Frontend.URL)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("API_ADDR", ":9999")
	t.Setenv("GRPC_ADDR", ":7070")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "32")
	t.Setenv("DATABASE_AUTO_MIGRATE", "false")
	t.Setenv("API_READ_TIMEOUT", "30s")

	cfg := Load()

	if cfg.API.Addr != ":9999" {
		t.Errorf("expected API.Addr ':9999', got %q", cfg.API.Addr)
	}
	if cfg.GRPC.Address != ":7070" {
		t.Errorf("expected GRPC.Address ':7070', got %q", cfg.GRPC.Address)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("expected Logging.Level 'debug', got %q", cfg.Logging.Level)
	}
	if cfg.Database.MaxOpenConns != 32 {
		t.Errorf("expected Database.MaxOpenConns 32, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.AutoMigrate != false {
		t.Error("expected Database.AutoMigrate false, got true")
	}
	if cfg.API.ReadTimeout != 30*time.Second {
		t.Errorf("expected API.ReadTimeout 30s, got %v", cfg.API.ReadTimeout)
	}
}
