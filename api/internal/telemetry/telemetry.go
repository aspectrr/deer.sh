package telemetry

import (
	"github.com/posthog/posthog-go"
)

// Service defines the interface for telemetry operations.
type Service interface {
	Track(userID, event string, properties map[string]any)
	GroupIdentify(orgID string, properties map[string]any)
	Close()
}

// NoopService is a telemetry service that does nothing.
type NoopService struct{}

func (s *NoopService) Track(userID, event string, properties map[string]any) {}
func (s *NoopService) GroupIdentify(orgID string, properties map[string]any) {}
func (s *NoopService) Close()                                                {}

type posthogService struct {
	client posthog.Client
}

// New creates a new telemetry service. Returns NoopService if apiKey is empty.
func New(apiKey, endpoint string) Service {
	if apiKey == "" {
		return &NoopService{}
	}

	if endpoint == "" {
		endpoint = "https://nautilus.fluid.sh"
	}

	client, err := posthog.NewWithConfig(apiKey, posthog.Config{Endpoint: endpoint})
	if err != nil {
		return &NoopService{}
	}

	return &posthogService{client: client}
}

func (s *posthogService) Track(userID, event string, properties map[string]any) {
	props := posthog.NewProperties()
	for k, v := range properties {
		props.Set(k, v)
	}

	_ = s.client.Enqueue(posthog.Capture{
		DistinctId: userID,
		Event:      event,
		Properties: props,
	})
}

func (s *posthogService) GroupIdentify(orgID string, properties map[string]any) {
	props := posthog.NewProperties()
	for k, v := range properties {
		props.Set(k, v)
	}

	_ = s.client.Enqueue(posthog.GroupIdentify{
		Type:       "organization",
		Key:        orgID,
		Properties: props,
	})
}

func (s *posthogService) Close() {
	_ = s.client.Close()
}
