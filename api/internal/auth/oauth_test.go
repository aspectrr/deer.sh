package auth

import (
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
