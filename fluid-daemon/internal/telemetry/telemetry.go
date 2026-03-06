// Package telemetry provides anonymous usage telemetry via PostHog for the daemon.
//
// Anonymity design:
//   - Uses the daemon's persistent HostID as distinct ID
//   - $ip is explicitly set to "0.0.0.0" to prevent PostHog from capturing client IP
//   - Only non-PII properties are tracked: OS, architecture, Go version
package telemetry

import (
	"runtime"

	"github.com/posthog/posthog-go"
)

// posthogAPIKey is the PostHog API key. Empty by default - must be injected at build time.
// Override at build time with: -ldflags "-X github.com/aspectrr/fluid.sh/fluid-daemon/internal/telemetry.posthogAPIKey=YOUR_KEY"
var posthogAPIKey = ""

// Config controls telemetry behavior.
type Config struct {
	EnableAnonymousUsage bool `yaml:"enable_anonymous_usage"`
}

// Service defines the interface for telemetry operations.
type Service interface {
	Track(event string, properties map[string]any)
	Close()
}

// NoopService is a telemetry service that does nothing.
type NoopService struct{}

func (s *NoopService) Track(event string, properties map[string]any) {}
func (s *NoopService) Close()                                        {}

// NewNoopService returns a telemetry service that does nothing.
func NewNoopService() Service {
	return &NoopService{}
}

type posthogService struct {
	client     posthog.Client
	distinctID string
}

// NewService creates a new telemetry service based on configuration.
// Uses the daemon's persistent hostID as the distinct ID.
func NewService(cfg Config, hostID string) (Service, error) {
	if !cfg.EnableAnonymousUsage || posthogAPIKey == "" {
		return &NoopService{}, nil
	}

	client, err := posthog.NewWithConfig(posthogAPIKey, posthog.Config{Endpoint: "https://nautilus.fluid.sh"})
	if err != nil {
		return nil, err
	}

	return &posthogService{
		client:     client,
		distinctID: hostID,
	}, nil
}

// buildTrackProperties adds common anonymous properties to a track event.
func buildTrackProperties(properties map[string]any) map[string]any {
	if properties == nil {
		properties = make(map[string]any)
	}
	properties["os"] = runtime.GOOS
	properties["arch"] = runtime.GOARCH
	properties["go_version"] = runtime.Version()
	properties["$ip"] = "0.0.0.0"
	return properties
}

func (s *posthogService) Track(event string, properties map[string]any) {
	properties = buildTrackProperties(properties)

	_ = s.client.Enqueue(posthog.Capture{
		DistinctId: s.distinctID,
		Event:      event,
		Properties: properties,
	})
}

func (s *posthogService) Close() {
	_ = s.client.Close()
}
