package telemetry

import (
	"runtime"
	"testing"
)

func TestNewNoopService(t *testing.T) {
	svc := NewNoopService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	// NoopService should accept calls without panicking
	svc.Track("test_event", nil)
	svc.Track("test_event", map[string]any{"key": "value"})
	svc.Close()
}

func TestNewServiceDisabled(t *testing.T) {
	cfg := Config{EnableAnonymousUsage: false}
	svc, err := NewService(cfg, "host-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := svc.(*NoopService); !ok {
		t.Errorf("expected *NoopService, got %T", svc)
	}

	svc.Track("test_event", nil)
	svc.Close()
}

func TestNewServiceNoAPIKey(t *testing.T) {
	cfg := Config{EnableAnonymousUsage: true}
	// posthogAPIKey is empty by default
	svc, err := NewService(cfg, "host-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := svc.(*NoopService); !ok {
		t.Errorf("expected *NoopService when API key is empty, got %T", svc)
	}
}

func TestNoopServiceMethods(t *testing.T) {
	svc := &NoopService{}

	svc.Track("event", nil)
	svc.Track("event", map[string]any{
		"string": "value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
	})
	svc.Close()
}

func TestBuildTrackProperties_NilInput(t *testing.T) {
	props := buildTrackProperties(nil)
	if props == nil {
		t.Fatal("expected non-nil properties")
	}
	if ip, ok := props["$ip"]; !ok || ip != "0.0.0.0" {
		t.Errorf("expected $ip=0.0.0.0, got %v", ip)
	}
	if props["os"] != runtime.GOOS {
		t.Errorf("expected os=%s, got %v", runtime.GOOS, props["os"])
	}
	if props["arch"] != runtime.GOARCH {
		t.Errorf("expected arch=%s, got %v", runtime.GOARCH, props["arch"])
	}
	if props["go_version"] != runtime.Version() {
		t.Errorf("expected go_version=%s, got %v", runtime.Version(), props["go_version"])
	}
}

func TestBuildTrackProperties_PreservesExisting(t *testing.T) {
	input := map[string]any{
		"custom_key": "custom_value",
		"count":      42,
	}
	props := buildTrackProperties(input)

	if props["custom_key"] != "custom_value" {
		t.Errorf("expected custom_key preserved, got %v", props["custom_key"])
	}
	if props["count"] != 42 {
		t.Errorf("expected count preserved, got %v", props["count"])
	}
	if props["$ip"] != "0.0.0.0" {
		t.Errorf("expected $ip=0.0.0.0, got %v", props["$ip"])
	}
}
