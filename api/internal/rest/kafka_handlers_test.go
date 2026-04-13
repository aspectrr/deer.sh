package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"

	"github.com/aspectrr/deer.sh/api/internal/store"
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
		SendAndWaitFn: func(_ context.Context, hostID string, msg *deerv1.ControlMessage, _ time.Duration) (*deerv1.HostMessage, error) {
			if hostID != "host-1" {
				t.Fatalf("unexpected hostID %q", hostID)
			}
			if msg.GetListSandboxKafkaStubs() == nil {
				t.Fatalf("expected ListSandboxKafkaStubs command, got %#v", msg.Payload)
			}
			return &deerv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &deerv1.HostMessage_ListSandboxKafkaStubsResponse{
					ListSandboxKafkaStubsResponse: &deerv1.ListSandboxKafkaStubsResponse{
						Stubs: []*deerv1.SandboxKafkaStubInfo{{
							StubId:              "stub-1",
							SandboxId:           "SBX-1",
							CaptureConfigId:     "cfg-1",
							BrokerEndpoint:      "10.0.0.10:9092",
							Topics:              []string{"logs"},
							ReplayWindowSeconds: 300,
							State:               deerv1.KafkaStubState_KAFKA_STUB_STATE_RUNNING,
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

func TestHandleGetKafkaCaptureConfig_Success(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	existing := &store.KafkaCaptureConfig{
		ID:               "cfg-get-1",
		OrgID:            testOrg.ID,
		SourceHostID:     "sh-1",
		SourceVM:         "logstash-1",
		Name:             "Logs",
		BootstrapServers: store.StringSlice{"kafka-1:9092"},
		Topics:           store.StringSlice{"logs"},
		Codec:            "json",
		Enabled:          true,
	}
	ms.GetKafkaCaptureConfigFn = func(_ context.Context, id string) (*store.KafkaCaptureConfig, error) {
		if id == existing.ID {
			return existing, nil
		}
		return nil, store.ErrNotFound
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/kafka-capture-configs/cfg-get-1", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResponse(rr)
	if resp["id"] != "cfg-get-1" {
		t.Fatalf("expected id=cfg-get-1, got %v", resp["id"])
	}
	if resp["name"] != "Logs" {
		t.Fatalf("expected name=Logs, got %v", resp["name"])
	}
}

func TestHandleGetKafkaCaptureConfig_NotFound(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	ms.GetKafkaCaptureConfigFn = func(_ context.Context, id string) (*store.KafkaCaptureConfig, error) {
		return nil, store.ErrNotFound
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/kafka-capture-configs/cfg-missing", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleGetKafkaCaptureConfig_WrongOrg(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	otherOrgConfig := &store.KafkaCaptureConfig{
		ID:               "cfg-other-org",
		OrgID:            "ORG-other",
		SourceHostID:     "sh-2",
		SourceVM:         "logstash-2",
		Name:             "Other",
		BootstrapServers: store.StringSlice{"kafka-2:9092"},
		Topics:           store.StringSlice{"other-logs"},
		Codec:            "json",
		Enabled:          true,
	}
	ms.GetKafkaCaptureConfigFn = func(_ context.Context, id string) (*store.KafkaCaptureConfig, error) {
		if id == otherOrgConfig.ID {
			return otherOrgConfig, nil
		}
		return nil, store.ErrNotFound
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/kafka-capture-configs/cfg-other-org", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateKafkaCaptureConfig_Success(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	existing := &store.KafkaCaptureConfig{
		ID:               "cfg-upd-1",
		OrgID:            testOrg.ID,
		SourceHostID:     "sh-1",
		SourceVM:         "logstash-1",
		Name:             "Logs",
		BootstrapServers: store.StringSlice{"kafka-1:9092"},
		Topics:           store.StringSlice{"logs"},
		Codec:            "json",
		Enabled:          true,
	}
	ms.GetKafkaCaptureConfigFn = func(_ context.Context, id string) (*store.KafkaCaptureConfig, error) {
		if id == existing.ID {
			return existing, nil
		}
		return nil, store.ErrNotFound
	}

	var updated *store.KafkaCaptureConfig
	ms.UpdateKafkaCaptureConfigFn = func(_ context.Context, cfg *store.KafkaCaptureConfig) error {
		updated = cfg
		return nil
	}

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("PATCH", "/v1/orgs/test-org/kafka-capture-configs/cfg-upd-1", strings.NewReader(`{
		"source_host_id":"sh-1",
		"source_vm":"logstash-1",
		"name":"Updated Logs",
		"bootstrap_servers":["kafka-1:9092","kafka-2:9092"],
		"topics":["logs","metrics"],
		"codec":"json",
		"enabled":false
	}`))
	body.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "PATCH", "/v1/orgs/test-org/kafka-capture-configs/cfg-upd-1", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if updated == nil {
		t.Fatal("expected UpdateKafkaCaptureConfig to be called")
	}
	if updated.Name != "Updated Logs" {
		t.Fatalf("expected name=Updated Logs, got %s", updated.Name)
	}
	if updated.Enabled != false {
		t.Fatalf("expected enabled=false, got %v", updated.Enabled)
	}
	resp := parseJSONResponse(rr)
	if resp["name"] != "Updated Logs" {
		t.Fatalf("expected response name=Updated Logs, got %v", resp["name"])
	}
}

func TestHandleUpdateKafkaCaptureConfig_EmptyBootstrapServers(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	existing := &store.KafkaCaptureConfig{
		ID:               "cfg-upd-bs",
		OrgID:            testOrg.ID,
		SourceHostID:     "sh-1",
		SourceVM:         "logstash-1",
		Name:             "Logs",
		BootstrapServers: store.StringSlice{"kafka-1:9092"},
		Topics:           store.StringSlice{"logs"},
		Codec:            "json",
		Enabled:          true,
	}
	ms.GetKafkaCaptureConfigFn = func(_ context.Context, id string) (*store.KafkaCaptureConfig, error) {
		if id == existing.ID {
			return existing, nil
		}
		return nil, store.ErrNotFound
	}

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("PATCH", "/v1/orgs/test-org/kafka-capture-configs/cfg-upd-bs", strings.NewReader(`{
		"source_host_id":"sh-1",
		"source_vm":"logstash-1",
		"name":"Logs",
		"bootstrap_servers":[],
		"topics":["logs"],
		"codec":"json",
		"enabled":true
	}`))
	body.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "PATCH", "/v1/orgs/test-org/kafka-capture-configs/cfg-upd-bs", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateKafkaCaptureConfig_EmptyTopics(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	existing := &store.KafkaCaptureConfig{
		ID:               "cfg-upd-topics",
		OrgID:            testOrg.ID,
		SourceHostID:     "sh-1",
		SourceVM:         "logstash-1",
		Name:             "Logs",
		BootstrapServers: store.StringSlice{"kafka-1:9092"},
		Topics:           store.StringSlice{"logs"},
		Codec:            "json",
		Enabled:          true,
	}
	ms.GetKafkaCaptureConfigFn = func(_ context.Context, id string) (*store.KafkaCaptureConfig, error) {
		if id == existing.ID {
			return existing, nil
		}
		return nil, store.ErrNotFound
	}

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("PATCH", "/v1/orgs/test-org/kafka-capture-configs/cfg-upd-topics", strings.NewReader(`{
		"source_host_id":"sh-1",
		"source_vm":"logstash-1",
		"name":"Logs",
		"bootstrap_servers":["kafka-1:9092"],
		"topics":[],
		"codec":"json",
		"enabled":true
	}`))
	body.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "PATCH", "/v1/orgs/test-org/kafka-capture-configs/cfg-upd-topics", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateKafkaCaptureConfig_NotFound(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	ms.GetKafkaCaptureConfigFn = func(_ context.Context, id string) (*store.KafkaCaptureConfig, error) {
		return nil, store.ErrNotFound
	}

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("PATCH", "/v1/orgs/test-org/kafka-capture-configs/cfg-missing", strings.NewReader(`{
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
	req := authenticatedRequest(ms, "PATCH", "/v1/orgs/test-org/kafka-capture-configs/cfg-missing", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeleteKafkaCaptureConfig_Success(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	existing := &store.KafkaCaptureConfig{
		ID:               "cfg-del-1",
		OrgID:            testOrg.ID,
		SourceHostID:     "sh-1",
		SourceVM:         "logstash-1",
		Name:             "Logs",
		BootstrapServers: store.StringSlice{"kafka-1:9092"},
		Topics:           store.StringSlice{"logs"},
		Codec:            "json",
		Enabled:          true,
	}
	ms.GetKafkaCaptureConfigFn = func(_ context.Context, id string) (*store.KafkaCaptureConfig, error) {
		if id == existing.ID {
			return existing, nil
		}
		return nil, store.ErrNotFound
	}

	var deletedID string
	ms.DeleteKafkaCaptureConfigFn = func(_ context.Context, id string) error {
		deletedID = id
		return nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/kafka-capture-configs/cfg-del-1", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if deletedID != "cfg-del-1" {
		t.Fatalf("expected deleted ID cfg-del-1, got %s", deletedID)
	}
	resp := parseJSONResponse(rr)
	if resp["deleted"] != true {
		t.Fatalf("expected deleted=true, got %v", resp["deleted"])
	}
}

func TestHandleDeleteKafkaCaptureConfig_NotFound(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	ms.GetKafkaCaptureConfigFn = func(_ context.Context, id string) (*store.KafkaCaptureConfig, error) {
		return nil, store.ErrNotFound
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/kafka-capture-configs/cfg-missing", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleListKafkaCaptureConfigs(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	ms.ListKafkaCaptureConfigsByOrgFn = func(_ context.Context, orgID string) ([]*store.KafkaCaptureConfig, error) {
		return []*store.KafkaCaptureConfig{
			{ID: "cfg-1", OrgID: orgID, Name: "Logs", BootstrapServers: store.StringSlice{"kafka:9092"}, Topics: store.StringSlice{"logs"}},
			{ID: "cfg-2", OrgID: orgID, Name: "Metrics", BootstrapServers: store.StringSlice{"kafka:9092"}, Topics: store.StringSlice{"metrics"}},
		}, nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/kafka-capture-configs", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResponse(rr)
	if resp["count"] != float64(2) {
		t.Fatalf("expected count=2, got %v", resp["count"])
	}
}

func TestHandleListKafkaCaptureConfigs_StoreError(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	ms.ListKafkaCaptureConfigsByOrgFn = func(_ context.Context, _ string) ([]*store.KafkaCaptureConfig, error) {
		return nil, fmt.Errorf("db error")
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/kafka-capture-configs", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateKafkaCaptureConfig_MissingFields(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	s := newTestServer(ms, nil)
	body := httptest.NewRequest("POST", "/v1/orgs/test-org/kafka-capture-configs", strings.NewReader(`{
		"name":"Logs"
	}`))
	body.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/kafka-capture-configs", body)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleGetSandboxKafkaStub(t *testing.T) {
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
		SendAndWaitFn: func(_ context.Context, _ string, msg *deerv1.ControlMessage, _ time.Duration) (*deerv1.HostMessage, error) {
			if msg.GetGetSandboxKafkaStub() == nil {
				t.Fatalf("expected GetSandboxKafkaStub command")
			}
			return &deerv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &deerv1.HostMessage_SandboxKafkaStubInfo{
					SandboxKafkaStubInfo: &deerv1.SandboxKafkaStubInfo{
						StubId:          "stub-1",
						SandboxId:       "SBX-1",
						CaptureConfigId: "cfg-1",
						BrokerEndpoint:  "10.0.0.10:9092",
						Topics:          []string{"logs"},
						State:           deerv1.KafkaStubState_KAFKA_STUB_STATE_RUNNING,
					},
				},
			}, nil
		},
	}

	s := newTestServerWithSender(ms, sender, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/SBX-1/kafka-stubs/stub-1", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResponse(rr)
	if resp["id"] != "stub-1" {
		t.Fatalf("expected id=stub-1, got %v", resp["id"])
	}
}

func TestHandleGetSandboxKafkaStub_NotFound(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
		return &store.Sandbox{ID: sandboxID, OrgID: testOrg.ID, HostID: "host-1"}, nil
	}

	sender := &mockHostSender{
		SendAndWaitFn: func(_ context.Context, _ string, msg *deerv1.ControlMessage, _ time.Duration) (*deerv1.HostMessage, error) {
			return &deerv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &deerv1.HostMessage_ErrorReport{
					ErrorReport: &deerv1.ErrorReport{Error: "not found"},
				},
			}, nil
		},
	}

	s := newTestServerWithSender(ms, sender, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/sandboxes/SBX-1/kafka-stubs/stub-missing", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleStartSandboxKafkaStub(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
		return &store.Sandbox{ID: sandboxID, OrgID: testOrg.ID, HostID: "host-1"}, nil
	}
	ms.UpdateSandboxKafkaStubFn = func(_ context.Context, _ *store.SandboxKafkaStub) error {
		return nil
	}

	sender := &mockHostSender{
		SendAndWaitFn: func(_ context.Context, _ string, msg *deerv1.ControlMessage, _ time.Duration) (*deerv1.HostMessage, error) {
			return &deerv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &deerv1.HostMessage_SandboxKafkaStubInfo{
					SandboxKafkaStubInfo: &deerv1.SandboxKafkaStubInfo{
						StubId:    "stub-1",
						SandboxId: "SBX-1",
						State:     deerv1.KafkaStubState_KAFKA_STUB_STATE_RUNNING,
					},
				},
			}, nil
		},
	}

	s := newTestServerWithSender(ms, sender, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/SBX-1/kafka-stubs/stub-1/start", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleStopSandboxKafkaStub(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
		return &store.Sandbox{ID: sandboxID, OrgID: testOrg.ID, HostID: "host-1"}, nil
	}
	ms.UpdateSandboxKafkaStubFn = func(_ context.Context, _ *store.SandboxKafkaStub) error {
		return nil
	}

	sender := &mockHostSender{
		SendAndWaitFn: func(_ context.Context, _ string, msg *deerv1.ControlMessage, _ time.Duration) (*deerv1.HostMessage, error) {
			return &deerv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &deerv1.HostMessage_SandboxKafkaStubInfo{
					SandboxKafkaStubInfo: &deerv1.SandboxKafkaStubInfo{
						StubId:    "stub-1",
						SandboxId: "SBX-1",
						State:     deerv1.KafkaStubState_KAFKA_STUB_STATE_STOPPED,
					},
				},
			}, nil
		},
	}

	s := newTestServerWithSender(ms, sender, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/SBX-1/kafka-stubs/stub-1/stop", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleRestartSandboxKafkaStub(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
		return &store.Sandbox{ID: sandboxID, OrgID: testOrg.ID, HostID: "host-1"}, nil
	}
	ms.UpdateSandboxKafkaStubFn = func(_ context.Context, _ *store.SandboxKafkaStub) error {
		return nil
	}

	sender := &mockHostSender{
		SendAndWaitFn: func(_ context.Context, _ string, msg *deerv1.ControlMessage, _ time.Duration) (*deerv1.HostMessage, error) {
			return &deerv1.HostMessage{
				RequestId: msg.GetRequestId(),
				Payload: &deerv1.HostMessage_SandboxKafkaStubInfo{
					SandboxKafkaStubInfo: &deerv1.SandboxKafkaStubInfo{
						StubId:    "stub-1",
						SandboxId: "SBX-1",
						State:     deerv1.KafkaStubState_KAFKA_STUB_STATE_RUNNING,
					},
				},
			}, nil
		},
	}

	s := newTestServerWithSender(ms, sender, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/SBX-1/kafka-stubs/stub-1/restart", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleTransitionSandboxKafkaStub_NotFound(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetSandboxFn = func(_ context.Context, sandboxID string) (*store.Sandbox, error) {
		return &store.Sandbox{ID: sandboxID, OrgID: testOrg.ID, HostID: "host-1"}, nil
	}

	sender := &mockHostSender{
		SendAndWaitFn: func(_ context.Context, _ string, msg *deerv1.ControlMessage, _ time.Duration) (*deerv1.HostMessage, error) {
			return nil, fmt.Errorf("host error")
		},
	}

	s := newTestServerWithSender(ms, sender, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/sandboxes/SBX-1/kafka-stubs/stub-1/start", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}
