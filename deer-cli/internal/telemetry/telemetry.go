// Package telemetry provides anonymous usage telemetry via PostHog.
//
// Anonymity design:
//   - A persistent UUID is stored in ~/.config/deer/telemetry_id for cross-session correlation
//   - $ip is explicitly set to "0.0.0.0" to prevent PostHog from capturing client IP
//   - Only non-PII properties are tracked: OS, architecture, Go version
package telemetry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
	"github.com/aspectrr/deer.sh/deer-cli/internal/paths"

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"
)

// posthogAPIKey is the PostHog API key. Empty by default - must be injected at build time.
// Override at build time with: -ldflags "-X github.com/aspectrr/deer.sh/deer-cli/internal/telemetry.posthogAPIKey=YOUR_KEY"
var posthogAPIKey = ""

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
// When enabled, telemetry is fully anonymous: a persistent UUID stored in
// ~/.config/deer/telemetry_id, $ip set to 0.0.0.0, and only OS/arch/go_version tracked.
func NewService(cfg config.TelemetryConfig) (Service, error) {
	if !cfg.EnableAnonymousUsage || posthogAPIKey == "" {
		return &NoopService{}, nil
	}

	client, err := posthog.NewWithConfig(posthogAPIKey, posthog.Config{Endpoint: "https://nautilus.deer.sh"})
	if err != nil {
		return nil, err
	}

	distinctID := getOrCreateDistinctID()

	return &posthogService{
		client:     client,
		distinctID: distinctID,
	}, nil
}

// getOrCreateDistinctID reads a persistent telemetry ID from the config directory.
// If the file does not exist, it generates a new UUID and writes it.
func getOrCreateDistinctID() string {
	dir, err := paths.ConfigDir()
	if err != nil {
		return uuid.New().String()
	}
	return getOrCreateDistinctIDInDir(dir)
}

// getOrCreateDistinctIDInDir reads or creates a telemetry ID in the given directory.
// Extracted for testability.
func getOrCreateDistinctIDInDir(dir string) string {
	idPath := filepath.Join(dir, "telemetry_id")

	data, err := os.ReadFile(idPath)
	if err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id
		}
	}

	id := uuid.New().String()
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(idPath, []byte(id), 0o600)
	return id
}

// buildTrackProperties adds common anonymous properties to a track event.
// Sets $ip to 0.0.0.0 so PostHog does not capture the client IP server-side.
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
