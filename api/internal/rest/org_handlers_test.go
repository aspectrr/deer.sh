package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

func TestHandleCreateOrg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/",
			strings.NewReader(`{"name":"My Org","slug":"my-org"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["name"] != "My Org" {
			t.Fatalf("expected name 'My Org', got %v", body["name"])
		}
		if body["slug"] != "my-org" {
			t.Fatalf("expected slug 'my-org', got %v", body["slug"])
		}
	})

	t.Run("invalid slug", func(t *testing.T) {
		ms := &mockStore{}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/",
			strings.NewReader(`{"name":"Bad Org","slug":"A!"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("duplicate explicit slug returns 409", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return store.ErrAlreadyExists
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/",
			strings.NewReader(`{"name":"Dup Org","slug":"dup-org"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("duplicate auto slug auto-resolves with suffix", func(t *testing.T) {
		ms := &mockStore{}
		calls := 0
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			calls++
			if calls == 1 {
				return store.ErrAlreadyExists
			}
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/",
			strings.NewReader(`{"name":"Dup Org"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		slug, ok := body["slug"].(string)
		if !ok {
			t.Fatal("expected slug in response")
		}
		if !strings.HasPrefix(slug, "dup-org-") {
			t.Fatalf("expected slug to start with 'dup-org-', got %q", slug)
		}
		if len(slug) != len("dup-org-")+4 {
			t.Fatalf("expected 4-char hex suffix, got slug %q", slug)
		}
	})
}

func TestHandleListOrgs(t *testing.T) {
	ms := &mockStore{}
	ms.ListOrganizationsByUserFn = func(_ context.Context, userID string) ([]*store.Organization, error) {
		return []*store.Organization{testOrg}, nil
	}
	s := newTestServer(ms, nil)

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := parseJSONResponse(rr)
	orgs, ok := body["organizations"].([]any)
	if !ok {
		t.Fatal("expected organizations array in response")
	}
	if len(orgs) != 1 {
		t.Fatalf("expected 1 org, got %d", len(orgs))
	}
}

func TestHandleGetOrg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["slug"] != testOrg.Slug {
			t.Fatalf("expected slug %s, got %v", testOrg.Slug, body["slug"])
		}
	})

	t.Run("not found", func(t *testing.T) {
		ms := &mockStore{}
		ms.GetOrganizationBySlugFn = func(_ context.Context, slug string) (*store.Organization, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/nonexistent", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("not member", func(t *testing.T) {
		ms := &mockStore{}
		ms.GetOrganizationBySlugFn = func(_ context.Context, slug string) (*store.Organization, error) {
			return testOrg, nil
		}
		ms.GetOrgMemberFn = func(_ context.Context, orgID, userID string) (*store.OrgMember, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleUpdateOrg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.UpdateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("PATCH", "/v1/orgs/test-org",
			strings.NewReader(`{"name":"Updated Org"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "PATCH", "/v1/orgs/test-org", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["name"] != "Updated Org" {
			t.Fatalf("expected name 'Updated Org', got %v", body["name"])
		}
	})

	t.Run("insufficient permissions", func(t *testing.T) {
		ms := &mockStore{}
		ms.GetOrganizationBySlugFn = func(_ context.Context, slug string) (*store.Organization, error) {
			return testOrg, nil
		}
		ms.GetOrgMemberFn = func(_ context.Context, orgID, userID string) (*store.OrgMember, error) {
			return &store.OrgMember{
				ID:     "MBR-member1",
				OrgID:  testOrg.ID,
				UserID: testUser.ID,
				Role:   store.OrgRoleMember,
			}, nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("PATCH", "/v1/orgs/test-org",
			strings.NewReader(`{"name":"Updated Org"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "PATCH", "/v1/orgs/test-org", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleDeleteOrg(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.DeleteOrganizationFn = func(_ context.Context, id string) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["status"] != "deleted" {
			t.Fatalf("expected status 'deleted', got %v", body["status"])
		}
	})

	t.Run("not owner", func(t *testing.T) {
		ms := &mockStore{}
		notOwnedOrg := &store.Organization{
			ID:      "ORG-other",
			Name:    "Other Org",
			Slug:    "other-org",
			OwnerID: "USR-someone-else",
		}
		ms.GetOrganizationBySlugFn = func(_ context.Context, slug string) (*store.Organization, error) {
			if slug == "other-org" {
				return notOwnedOrg, nil
			}
			return nil, store.ErrNotFound
		}
		ms.GetOrgMemberFn = func(_ context.Context, orgID, userID string) (*store.OrgMember, error) {
			return &store.OrgMember{
				ID:     "MBR-admin1",
				OrgID:  notOwnedOrg.ID,
				UserID: testUser.ID,
				Role:   store.OrgRoleAdmin,
			}, nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/other-org", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleListMembers(t *testing.T) {
	ms := &mockStore{}
	setupOrgMembership(ms)
	ms.ListOrgMembersFn = func(_ context.Context, orgID string) ([]*store.OrgMember, error) {
		return []*store.OrgMember{testMember}, nil
	}
	s := newTestServer(ms, nil)

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/orgs/test-org/members", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := parseJSONResponse(rr)
	members, ok := body["members"].([]any)
	if !ok {
		t.Fatal("expected members array in response")
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
}

func TestHandleAddMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)

		targetUser := &store.User{
			ID:          "USR-target",
			Email:       "target@example.com",
			DisplayName: "Target User",
		}
		ms.GetUserByEmailFn = func(_ context.Context, email string) (*store.User, error) {
			if email == targetUser.Email {
				return targetUser, nil
			}
			return nil, store.ErrNotFound
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/test-org/members",
			strings.NewReader(`{"email":"target@example.com","role":"member"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/members", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["user_id"] != targetUser.ID {
			t.Fatalf("expected user_id %s, got %v", targetUser.ID, body["user_id"])
		}
	})

	t.Run("missing email", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/test-org/members",
			strings.NewReader(`{"role":"member"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/members", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("cannot add owner role", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/orgs/test-org/members",
			strings.NewReader(`{"email":"target@example.com","role":"owner"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/orgs/test-org/members", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleRemoveMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		ms.GetOrgMemberByIDFn = func(_ context.Context, orgID, memberID string) (*store.OrgMember, error) {
			return &store.OrgMember{
				ID:     memberID,
				OrgID:  orgID,
				UserID: "USR-target",
				Role:   store.OrgRoleMember,
			}, nil
		}
		ms.DeleteOrgMemberFn = func(_ context.Context, orgID, id string) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/members/MBR-target1234", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["status"] != "removed" {
			t.Fatalf("expected status 'removed', got %v", body["status"])
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
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/members/MBR-target1234", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("IDOR cross-org member deletion returns 404", func(t *testing.T) {
		ms := &mockStore{}
		setupOrgMembership(ms)
		// GetOrgMemberByID scopes by org_id: member exists in another org but not in testOrg
		ms.GetOrgMemberByIDFn = func(_ context.Context, orgID, memberID string) (*store.OrgMember, error) {
			if orgID == testOrg.ID {
				return nil, store.ErrNotFound
			}
			return &store.OrgMember{ID: memberID, OrgID: orgID, Role: store.OrgRoleMember}, nil
		}
		ms.DeleteOrgMemberFn = func(_ context.Context, orgID, id string) error {
			if orgID == testOrg.ID {
				return store.ErrNotFound
			}
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := authenticatedRequest(ms, "DELETE", "/v1/orgs/test-org/members/MBR-other-org", nil)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for cross-org IDOR, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}
