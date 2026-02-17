package telemetry

import (
	"runtime"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"
)

// posthogAPIKey is the PostHog API key. By default uses dev key.
// Override at build time with: -ldflags "-X github.com/aspectrr/fluid.sh/fluid/internal/telemetry.posthogAPIKey=YOUR_KEY"
var posthogAPIKey = "phc_nZdxqaqWmZhHpWPIsUFqmwtr9WfyYaae0IOdRmh8YGT"

// Service defines the interface for telemetry operations.
type Service interface {
	Track(event string, properties map[string]any)
	Close()
}

// NoopService is a telemetry service that does nothing.
// Use this when telemetry is disabled or initialization fails.
type NoopService struct{}

func (s *NoopService) Track(event string, properties map[string]any) {}
func (s *NoopService) Close()                                        {}

// NewNoopService returns a telemetry service that does nothing.
// Use this as a fallback when telemetry initialization fails
// or when you explicitly want to disable telemetry.
func NewNoopService() Service {
	return &NoopService{}
}

type posthogService struct {
	client     posthog.Client
	distinctID string
}

// NewService creates a new telemetry service based on configuration.
func NewService(cfg config.TelemetryConfig) (Service, error) {
	if !cfg.EnableAnonymousUsage {
		return &NoopService{}, nil
	}

	client, err := posthog.NewWithConfig(posthogAPIKey, posthog.Config{Endpoint: "https://nautilus.fluid.sh"})
	if err != nil {
		return nil, err
	}

	// Generate a unique ID for this session.
	// In a real application, you might want to persist this ID.
	distinctID := uuid.New().String()

	return &posthogService{
		client:     client,
		distinctID: distinctID,
	}, nil
}

func (s *posthogService) Track(event string, properties map[string]any) {
	if properties == nil {
		properties = make(map[string]any)
	}

	// Add common properties
	properties["os"] = runtime.GOOS
	properties["arch"] = runtime.GOARCH
	properties["go_version"] = runtime.Version()

	_ = s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinctID,
		Event:      event,
		Properties: properties,
	})
}

func (s *posthogService) Close() {
	_ = s.client.Close()
}
