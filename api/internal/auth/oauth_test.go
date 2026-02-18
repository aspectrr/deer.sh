package auth

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

func TestGitHubOAuthConfig(t *testing.T) {
	cfg := GitHubOAuthConfig("gh-id", "gh-secret", "http://localhost/callback")

	if cfg.ClientID != "gh-id" {
		t.Fatalf("ClientID = %q, want %q", cfg.ClientID, "gh-id")
	}
	if cfg.ClientSecret != "gh-secret" {
		t.Fatalf("ClientSecret = %q, want %q", cfg.ClientSecret, "gh-secret")
	}
	if cfg.RedirectURL != "http://localhost/callback" {
		t.Fatalf("RedirectURL = %q, want %q", cfg.RedirectURL, "http://localhost/callback")
	}

	wantScopes := []string{"user:email"}
	if len(cfg.Scopes) != len(wantScopes) {
		t.Fatalf("Scopes length = %d, want %d", len(cfg.Scopes), len(wantScopes))
	}
	for i, s := range cfg.Scopes {
		if s != wantScopes[i] {
			t.Fatalf("Scopes[%d] = %q, want %q", i, s, wantScopes[i])
		}
	}

	if cfg.Endpoint != github.Endpoint {
		t.Fatalf("Endpoint does not match github.Endpoint")
	}
}

func TestGoogleOAuthConfig(t *testing.T) {
	cfg := GoogleOAuthConfig("g-id", "g-secret", "http://localhost/google/callback")

	if cfg.ClientID != "g-id" {
		t.Fatalf("ClientID = %q, want %q", cfg.ClientID, "g-id")
	}
	if cfg.ClientSecret != "g-secret" {
		t.Fatalf("ClientSecret = %q, want %q", cfg.ClientSecret, "g-secret")
	}
	if cfg.RedirectURL != "http://localhost/google/callback" {
		t.Fatalf("RedirectURL = %q, want %q", cfg.RedirectURL, "http://localhost/google/callback")
	}

	wantScopes := []string{"openid", "email", "profile"}
	if len(cfg.Scopes) != len(wantScopes) {
		t.Fatalf("Scopes length = %d, want %d", len(cfg.Scopes), len(wantScopes))
	}
	for i, s := range cfg.Scopes {
		if s != wantScopes[i] {
			t.Fatalf("Scopes[%d] = %q, want %q", i, s, wantScopes[i])
		}
	}

	if cfg.Endpoint != google.Endpoint {
		t.Fatalf("Endpoint does not match google.Endpoint")
	}
}

func TestGenerateOAuthState(t *testing.T) {
	state, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("GenerateOAuthState() error = %v", err)
	}

	// Should be 64 hex chars (32 bytes)
	if len(state) != 64 {
		t.Fatalf("state length = %d, want 64", len(state))
	}

	if _, err := hex.DecodeString(state); err != nil {
		t.Fatalf("state is not valid hex: %v", err)
	}

	// Two calls should produce different values
	state2, _ := GenerateOAuthState()
	if state == state2 {
		t.Fatal("two calls returned the same state")
	}
}

func TestSetAndClearOAuthStateCookie(t *testing.T) {
	w := httptest.NewRecorder()
	SetOAuthStateCookie(w, "test-state", true)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("SetOAuthStateCookie did not set any cookie")
	}

	c := cookies[0]
	if c.Name != OAuthStateCookieName {
		t.Fatalf("cookie Name = %q, want %q", c.Name, OAuthStateCookieName)
	}
	if c.Value != "test-state" {
		t.Fatalf("cookie Value = %q, want %q", c.Value, "test-state")
	}
	if !c.HttpOnly {
		t.Fatal("cookie should be HttpOnly")
	}
	if !c.Secure {
		t.Fatal("cookie should be Secure when secure=true")
	}
	if c.MaxAge != oauthStateMaxAge {
		t.Fatalf("cookie MaxAge = %d, want %d", c.MaxAge, oauthStateMaxAge)
	}

	w2 := httptest.NewRecorder()
	ClearOAuthStateCookie(w2)
	cookies2 := w2.Result().Cookies()
	if len(cookies2) == 0 {
		t.Fatal("ClearOAuthStateCookie did not set any cookie")
	}
	if cookies2[0].MaxAge != -1 {
		t.Fatalf("cleared cookie MaxAge = %d, want -1", cookies2[0].MaxAge)
	}
}

func TestValidateOAuthState(t *testing.T) {
	// Missing state param
	req := httptest.NewRequest("GET", "/callback", nil)
	if err := ValidateOAuthState(req); err == nil {
		t.Fatal("expected error for missing state param")
	}

	// Missing cookie
	req = httptest.NewRequest("GET", "/callback?state=abc", nil)
	if err := ValidateOAuthState(req); err == nil {
		t.Fatal("expected error for missing cookie")
	}

	// Mismatched state
	req = httptest.NewRequest("GET", "/callback?state=abc", nil)
	req.AddCookie(&http.Cookie{Name: OAuthStateCookieName, Value: "xyz"})
	if err := ValidateOAuthState(req); err == nil {
		t.Fatal("expected error for mismatched state")
	}

	// Matching state
	req = httptest.NewRequest("GET", "/callback?state=matching-value", nil)
	req.AddCookie(&http.Cookie{Name: OAuthStateCookieName, Value: "matching-value"})
	if err := ValidateOAuthState(req); err != nil {
		t.Fatalf("ValidateOAuthState() unexpected error = %v", err)
	}
}
