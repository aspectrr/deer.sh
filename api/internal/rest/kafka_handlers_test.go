package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

func TestHandleCreateKafkaCaptureConfig(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	var created *store.KafkaCaptureConfig
	ms.CreateKafkaCaptureConfigFn = func(_ context.Context, cfg *store.KafkaCaptureConfig) error {
		created = cfg
		return nil
	}

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("POST", "/v1/orgs/test-org/kafka-capture-configs", strings.NewReader(`{
		"source_host_id":"sh-1",
		"source_vm":"logstash-1",
		"name":"Logs",
		"bootstrap_servers":["kafka-1:9092"],
		"topics":["logs"],
		"codec":"json",
		"enabled":true
	}`))
	body.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/kafka-capture-configs", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if created == nil {
		t.Fatal("expected CreateKafkaCaptureConfig to be called")
	}
	if created.SourceHostID != "sh-1" || created.SourceVM != "logstash-1" {
		t.Fatalf("unexpected created config: %+v", created)
	}
}

func TestHandleListSandboxKafkaStubs(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
		return &store.Sandbox{ID: sandboxID, OrgID: testOrg.ID, HostID: "host-1"}, nil
	}
	ms.UpdateSandboxKafkaStubFn = func(_ context.Context, _ *store.SandboxKafkaStub) error {
		return store.ErrNotFound
	}
	ms.CreateSandboxKafkaStubFn = func(_ context.Context, _ *store.SandboxKafkaStub) error {
		return nil
	}

	sender := &mockHostSender{
		SendAndWaitFn: func(_ context.Context, hostID string, msg *fluidv1.ControlMessage, _ time.Duration) (*fluidv1.HostMessage, error) {
			if hostID != "host-1" {
				t.Fatalf("unexpected hostID %q", hostID)
			}
			if msg.GetListSandboxKafkaStubs() == nil {
				t.Fatalf("expected ListSandboxKafkaStubs command, got %#v", msg.Payload)
			}
			return &fluidv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &fluidv1.HostMessage_ListSandboxKafkaStubsResponse{
					ListSandboxKafkaStubsResponse: &fluidv1.ListSandboxKafkaStubsResponse{
						Stubs: []*fluidv1.SandboxKafkaStubInfo{{
							StubId:              "stub-1",
							SandboxId:           "SBX-1",
							CaptureConfigId:     "cfg-1",
							BrokerEndpoint:      "10.0.0.10:9092",
							Topics:              []string{"logs"},
							ReplayWindowSeconds: 300,
							State:               fluidv1.KafkaStubState_KAFKA_STUB_STATE_RUNNING,
						}},
					},
				},
			}, nil
		},
	}

	s := newTestServerWithSender(ms, sender, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/SBX-1/kafka-stubs", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResponse(rr)
	if resp["count"] != float64(1) {
		t.Fatalf("expected count=1, got %v", resp["count"])
	}
}
