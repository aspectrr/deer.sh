package auth

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionCookieName(t *testing.T) {
	if SessionCookieName != "fluid_session" {
		t.Fatalf("SessionCookieName = %q, want %q", SessionCookieName, "fluid_session")
	}
}

func TestSetSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	ttl := 24 * time.Hour
	SetSessionCookie(w, "test-token", ttl, true)

	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("SetSessionCookie did not set any cookie")
	}

	c := cookies[0]

	if c.Name != SessionCookieName {
		t.Fatalf("cookie Name = %q, want %q", c.Name, SessionCookieName)
	}
	if c.Value != "test-token" {
		t.Fatalf("cookie Value = %q, want %q", c.Value, "test-token")
	}
	if c.Path != "/" {
		t.Fatalf("cookie Path = %q, want %q", c.Path, "/")
	}
	if !c.HttpOnly {
		t.Fatal("cookie HttpOnly = false, want true")
	}
	if !c.Secure {
		t.Fatal("cookie Secure = false, want true")
	}

	expectedMaxAge := int(ttl.Seconds())
	if c.MaxAge != expectedMaxAge {
		t.Fatalf("cookie MaxAge = %d, want %d", c.MaxAge, expectedMaxAge)
	}
}

func TestClearSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	ClearSessionCookie(w, true)

	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("ClearSessionCookie did not set any cookie")
	}

	c := cookies[0]

	if c.Name != SessionCookieName {
		t.Fatalf("cookie Name = %q, want %q", c.Name, SessionCookieName)
	}
	if c.MaxAge != -1 {
		t.Fatalf("cookie MaxAge = %d, want -1", c.MaxAge)
	}
}
