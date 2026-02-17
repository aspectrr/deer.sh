package telemetry

import (
	"testing"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
)

func TestNewNoopService(t *testing.T) {
	svc := NewNoopService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	// Verify it implements Service interface
	_ = svc

	// NoopService should accept calls without panicking
	svc.Track("test_event", nil)
	svc.Track("test_event", map[string]any{"key": "value"})
	svc.Close()
}

func TestNewServiceDisabled(t *testing.T) {
	cfg := config.TelemetryConfig{
		EnableAnonymousUsage: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	// Should return a NoopService when disabled
	if _, ok := svc.(*NoopService); !ok {
		t.Errorf("expected *NoopService, got %T", svc)
	}

	// Should work without panicking
	svc.Track("test_event", nil)
	svc.Close()
}

func TestNoopServiceMethods(t *testing.T) {
	svc := &NoopService{}

	// Track should not panic with nil properties
	svc.Track("event", nil)

	// Track should not panic with properties
	svc.Track("event", map[string]any{
		"string": "value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"nested": map[string]any{"inner": "value"},
	})

	// Close should not panic
	svc.Close()
}
