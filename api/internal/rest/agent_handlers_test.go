package rest

// Agent handler tests - commented out, not yet ready for integration.

/*
import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

func TestHandleListConversations(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.ListAgentConversationsByOrgFn = func(_ context.Context, orgID string) ([]*store.AgentConversation, error) {
		if orgID != testOrg.ID {
			t.Fatalf("unexpected orgID: %s", orgID)
		}
		return []*store.AgentConversation{
			{ID: "conv-1", OrgID: testOrg.ID, UserID: testUser.ID, Title: "Test Conv", Model: "gpt-4o"},
		}, nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/agent/conversations", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if resp["count"] != float64(1) {
		t.Fatalf("expected count 1, got %v", resp["count"])
	}
	convs, ok := resp["conversations"].([]any)
	if !ok || len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %v", resp["conversations"])
	}
}

func TestHandleGetConversation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetAgentConversationFn = func(_ context.Context, id string) (*store.AgentConversation, error) {
			if id == "conv-1" {
				return &store.AgentConversation{
					ID:    "conv-1",
					OrgID: testOrg.ID,
					Title: "My Conv",
				}, nil
			}
			return nil, store.ErrNotFound
		}

		s := newTestServer(ms, nil)
		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/agent/conversations/conv-1", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if resp["id"] != "conv-1" {
			t.Fatalf("expected id conv-1, got %v", resp["id"])
		}
	})

	t.Run("not_found", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetAgentConversationFn = func(_ context.Context, id string) (*store.AgentConversation, error) {
			return nil, store.ErrNotFound
		}

		s := newTestServer(ms, nil)
		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/agent/conversations/nonexistent", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("wrong_org", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetAgentConversationFn = func(_ context.Context, id string) (*store.AgentConversation, error) {
			return &store.AgentConversation{
				ID:    "conv-1",
				OrgID: "ORG-other",
				Title: "Other Org Conv",
			}, nil
		}

		s := newTestServer(ms, nil)
		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/agent/conversations/conv-1", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleDeleteConversation(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetAgentConversationFn = func(_ context.Context, id string) (*store.AgentConversation, error) {
		if id == "conv-1" {
			return &store.AgentConversation{ID: "conv-1", OrgID: testOrg.ID}, nil
		}
		return nil, store.ErrNotFound
	}
	deleted := false
	ms.DeleteAgentConversationFn = func(_ context.Context, id string) error {
		if id == "conv-1" {
			deleted = true
		}
		return nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/agent/conversations/conv-1", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !deleted {
		t.Fatal("expected DeleteAgentConversation to be called")
	}
	resp := parseJSONResponse(rr)
	if resp["deleted"] != true {
		t.Fatalf("expected deleted=true, got %v", resp["deleted"])
	}
}

func TestHandleListMessages(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.GetAgentConversationFn = func(_ context.Context, id string) (*store.AgentConversation, error) {
		if id == "conv-1" {
			return &store.AgentConversation{ID: "conv-1", OrgID: testOrg.ID}, nil
		}
		return nil, store.ErrNotFound
	}
	ms.ListAgentMessagesFn = func(_ context.Context, convID string) ([]*store.AgentMessage, error) {
		return []*store.AgentMessage{
			{ID: "msg-1", ConversationID: convID, Role: store.MessageRoleUser, Content: "hello", CreatedAt: time.Now()},
			{ID: "msg-2", ConversationID: convID, Role: store.MessageRoleAssistant, Content: "hi", CreatedAt: time.Now()},
		}, nil
	}

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/agent/conversations/conv-1/messages", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	resp := parseJSONResponse(rr)
	if resp["count"] != float64(2) {
		t.Fatalf("expected count 2, got %v", resp["count"])
	}
}

func TestHandleListModels(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)

	s := newTestServer(ms, nil)
	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/agent/models", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	models, ok := resp["models"].([]any)
	if !ok || len(models) == 0 {
		t.Fatalf("expected non-empty models list, got %v", resp["models"])
	}
}
*/
