package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

func TestHandleListHostTokens(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.ListHostTokensByOrgFn = func(_ context.Context, orgID string) ([]store.HostToken, error) {
		return []store.HostToken{
			{
				ID:    "HTK-1234",
				OrgID: testOrg.ID,
				Name:  "test-token",
			},
		}, nil
	}
	s := newTestServer(ms, nil)

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/hosts/tokens", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := parseJSONResponse(rr)
	tokens, ok := body["tokens"].([]any)
	if !ok {
		t.Fatal("expected tokens array in response")
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
}

func TestHandleCreateHostToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.CreateHostTokenFn = func(_ context.Context, token *store.HostToken) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/test-org/hosts/tokens",
			strings.NewReader(`{"name":"my-host-token"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/hosts/tokens", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["name"] != "my-host-token" {
			t.Fatalf("expected name 'my-host-token', got %v", body["name"])
		}
		// Token should be returned on creation.
		if body["token"] == nil || body["token"] == "" {
			t.Fatal("expected token to be returned on creation")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/test-org/hosts/tokens",
			strings.NewReader(`{}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/hosts/tokens", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("insufficient permissions - member role", func(t *testing.T) {
		ms := &mockStore{}
		ms.GetOrganizationBySlugFn = func(_ context.Context, slug string) (*store.Organization, error) {
			if slug == testOrg.Slug {
				return testOrg, nil
			}
			return nil, store.ErrNotFound
		}
		ms.GetOrgMemberFn = func(_ context.Context, orgID, userID string) (*store.OrgMember, error) {
			return &store.OrgMember{
				ID:     "MBR-regular",
				OrgID:  testOrg.ID,
				UserID: testUser.ID,
				Role:   store.OrgRoleMember,
			}, nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/test-org/hosts/tokens",
			strings.NewReader(`{"name":"forbidden-token"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/hosts/tokens", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleDeleteHostToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.DeleteHostTokenFn = func(_ context.Context, orgID, id string) error {
			if orgID != testOrg.ID {
				return store.ErrNotFound
			}
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/hosts/tokens/HTK-1234", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["status"] != "deleted" {
			t.Fatalf("expected status 'deleted', got %v", body["status"])
		}
	})

	t.Run("cross-org deletion rejected", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.DeleteHostTokenFn = func(_ context.Context, orgID, id string) error {
			// Token belongs to a different org, so scoped WHERE returns no rows
			if orgID != "ORG-other" {
				return store.ErrNotFound
			}
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/hosts/tokens/HTK-other-org", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for cross-org token, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("insufficient permissions", func(t *testing.T) {
		ms := &mockStore{}
		ms.GetOrganizationBySlugFn = func(_ context.Context, slug string) (*store.Organization, error) {
			if slug == testOrg.Slug {
				return testOrg, nil
			}
			return nil, store.ErrNotFound
		}
		ms.GetOrgMemberFn = func(_ context.Context, orgID, userID string) (*store.OrgMember, error) {
			return &store.OrgMember{
				ID:     "MBR-regular",
				OrgID:  testOrg.ID,
				UserID: testUser.ID,
				Role:   store.OrgRoleMember,
			}, nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/hosts/tokens/HTK-1234", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}
