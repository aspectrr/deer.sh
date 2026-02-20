package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

func TestHandleHealth(t *testing.T) {
	ms := &mockStore{}
	s := newTestServer(ms, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/health", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := parseJSONResponse(rr)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", body["status"])
	}
}

func TestHandleRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateUserFn = func(_ context.Context, u *store.User) error {
			return nil
		}
		ms.CreateSessionFn = func(_ context.Context, s *store.Session) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/register",
			strings.NewReader(`{"email":"new@example.com","password":"password123","display_name":"New User"}`))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		user, ok := body["user"].(map[string]any)
		if !ok {
			t.Fatal("expected user in response")
		}
		if user["email"] != "new@example.com" {
			t.Fatalf("expected email new@example.com, got %v", user["email"])
		}

		// Check that a session cookie was set.
		cookies := rr.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == auth.SessionCookieName {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected session cookie to be set")
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		ms := &mockStore{}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/register",
			strings.NewReader(`{"email":"new@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("short password", func(t *testing.T) {
		ms := &mockStore{}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/register",
			strings.NewReader(`{"email":"new@example.com","password":"short","display_name":"New User"}`))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateUserFn = func(_ context.Context, u *store.User) error {
			return store.ErrAlreadyExists
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/register",
			strings.NewReader(`{"email":"existing@example.com","password":"password123","display_name":"Existing"}`))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleLogin(t *testing.T) {
	password := "password123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	loginUser := &store.User{
		ID:           "USR-login1234",
		Email:        "login@example.com",
		DisplayName:  "Login User",
		PasswordHash: hash,
	}

	t.Run("success", func(t *testing.T) {
		ms := &mockStore{}
		ms.GetUserByEmailFn = func(_ context.Context, email string) (*store.User, error) {
			if email == loginUser.Email {
				return loginUser, nil
			}
			return nil, store.ErrNotFound
		}
		ms.CreateSessionFn = func(_ context.Context, s *store.Session) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/login",
			strings.NewReader(`{"email":"login@example.com","password":"password123"}`))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		user, ok := body["user"].(map[string]any)
		if !ok {
			t.Fatal("expected user in response")
		}
		if user["email"] != loginUser.Email {
			t.Fatalf("expected email %s, got %v", loginUser.Email, user["email"])
		}

		cookies := rr.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == auth.SessionCookieName {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected session cookie to be set")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		ms := &mockStore{}
		ms.GetUserByEmailFn = func(_ context.Context, email string) (*store.User, error) {
			if email == loginUser.Email {
				return loginUser, nil
			}
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/login",
			strings.NewReader(`{"email":"login@example.com","password":"wrongpassword"}`))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("user not found", func(t *testing.T) {
		ms := &mockStore{}
		ms.GetUserByEmailFn = func(_ context.Context, email string) (*store.User, error) {
			return nil, store.ErrNotFound
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/login",
			strings.NewReader(`{"email":"nonexistent@example.com","password":"password123"}`))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		ms := &mockStore{}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/login",
			strings.NewReader(`{"email":"login@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleLogout(t *testing.T) {
	ms := &mockStore{}
	ms.DeleteSessionFn = func(_ context.Context, id string) error {
		return nil
	}
	s := newTestServer(ms, nil)

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "POST", "/v1/auth/logout", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := parseJSONResponse(rr)
	if body["status"] != "logged out" {
		t.Fatalf("expected status 'logged out', got %v", body["status"])
	}

	// Check that the session cookie is cleared.
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName && c.MaxAge == -1 {
			return
		}
	}
	t.Fatal("expected session cookie to be cleared")
}

func TestHandleOnboarding(t *testing.T) {
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
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"My Company","role":"devops","use_cases":["ci","testing"],"referral_source":"github"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["name"] != "My Company" {
			t.Fatalf("expected name 'My Company', got %v", body["name"])
		}
		if body["slug"] != "my-company" {
			t.Fatalf("expected slug 'my-company', got %v", body["slug"])
		}
		if body["owner_id"] != testUser.ID {
			t.Fatalf("expected owner_id %s, got %v", testUser.ID, body["owner_id"])
		}
	})

	t.Run("success with minimal fields", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"test-org-name"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["slug"] != "test-org-name" {
			t.Fatalf("expected slug 'test-org-name', got %v", body["slug"])
		}
	})

	t.Run("missing org_name", func(t *testing.T) {
		ms := &mockStore{}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"role":"devops"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("invalid slug from org_name", func(t *testing.T) {
		ms := &mockStore{}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"A!"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("special chars apostrophe", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"Collin's Team"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["slug"] != "collins-team" {
			t.Fatalf("expected slug 'collins-team', got %v", body["slug"])
		}
	})

	t.Run("special chars ampersand", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"Acme & Co."}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["slug"] != "acme-co" {
			t.Fatalf("expected slug 'acme-co', got %v", body["slug"])
		}
	})

	t.Run("numbers in name", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"Team 42"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["slug"] != "team-42" {
			t.Fatalf("expected slug 'team-42', got %v", body["slug"])
		}
	})

	t.Run("leading trailing special chars", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"--My Org--"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["slug"] != "my-org" {
			t.Fatalf("expected slug 'my-org', got %v", body["slug"])
		}
	})

	t.Run("consecutive special chars stripped", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"Foo...Bar"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["slug"] != "foobar" {
			t.Fatalf("expected slug 'foobar', got %v", body["slug"])
		}
	})

	t.Run("underscores become hyphens parens stripped", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"My_Org (Dev)"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["slug"] != "my-org-dev" {
			t.Fatalf("expected slug 'my-org-dev', got %v", body["slug"])
		}
	})

	t.Run("all special chars fails", func(t *testing.T) {
		ms := &mockStore{}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"!!!@@@"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("mixed case preserved as lowercase", func(t *testing.T) {
		ms := &mockStore{}
		ms.CreateOrganizationFn = func(_ context.Context, org *store.Organization) error {
			return nil
		}
		ms.CreateOrgMemberFn = func(_ context.Context, m *store.OrgMember) error {
			return nil
		}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"MyAwesomeTeam"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		if body["slug"] != "myawesometeam" {
			t.Fatalf("expected slug 'myawesometeam', got %v", body["slug"])
		}
	})

	t.Run("duplicate slug auto-resolves with suffix", func(t *testing.T) {
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
		bodyReq := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"Existing Org"}`))
		bodyReq.Header.Set("Content-Type", "application/json")
		req := authenticatedRequest(ms, "POST", "/v1/auth/onboarding", bodyReq)
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
		}

		body := parseJSONResponse(rr)
		slug, ok := body["slug"].(string)
		if !ok {
			t.Fatal("expected slug in response")
		}
		if !strings.HasPrefix(slug, "existing-org-") {
			t.Fatalf("expected slug to start with 'existing-org-', got %q", slug)
		}
		if len(slug) != len("existing-org-")+4 {
			t.Fatalf("expected 4-char hex suffix, got slug %q", slug)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		ms := &mockStore{}
		s := newTestServer(ms, nil)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/onboarding",
			strings.NewReader(`{"org_name":"My Company"}`))
		req.Header.Set("Content-Type", "application/json")
		s.Router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestHandleMe(t *testing.T) {
	ms := &mockStore{}
	s := newTestServer(ms, nil)

	rr := httptest.NewRecorder()
	req := authenticatedRequest(ms, "GET", "/v1/auth/me", nil)
	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := parseJSONResponse(rr)
	user, ok := body["user"].(map[string]any)
	if !ok {
		t.Fatal("expected user in response")
	}
	if user["id"] != testUser.ID {
		t.Fatalf("expected user id %s, got %v", testUser.ID, user["id"])
	}
	if user["email"] != testUser.Email {
		t.Fatalf("expected email %s, got %v", testUser.Email, user["email"])
	}
}
