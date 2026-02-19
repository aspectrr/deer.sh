package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

func TestRequireAuth_NoCookie(t *testing.T) {
	st := &mockStore{}
	handler := RequireAuth(st, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_InvalidSession(t *testing.T) {
	st := &mockStore{
		getSessionFn: func(_ context.Context, _ string) (*store.Session, error) {
			return nil, fmt.Errorf("session not found")
		},
	}
	handler := RequireAuth(st, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "bad-session-id"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_UserNotFound(t *testing.T) {
	st := &mockStore{
		getSessionFn: func(_ context.Context, id string) (*store.Session, error) {
			return &store.Session{ID: id, UserID: "user-gone"}, nil
		},
		getUserFn: func(_ context.Context, _ string) (*store.User, error) {
			return nil, fmt.Errorf("user not found")
		},
	}
	handler := RequireAuth(st, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "valid-session"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireAuth_ValidSession(t *testing.T) {
	expectedUser := &store.User{ID: "user-1", Email: "test@example.com"}

	rawToken := "good-token"
	hashedToken := HashSessionToken(rawToken)

	st := &mockStore{
		getSessionFn: func(_ context.Context, id string) (*store.Session, error) {
			if id != hashedToken {
				return nil, fmt.Errorf("not found")
			}
			return &store.Session{ID: id, UserID: expectedUser.ID}, nil
		},
		getUserFn: func(_ context.Context, id string) (*store.User, error) {
			if id != expectedUser.ID {
				return nil, fmt.Errorf("not found")
			}
			return expectedUser, nil
		},
	}

	var handlerCalled bool
	handler := RequireAuth(st, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		u := UserFromContext(r.Context())
		if u == nil {
			t.Fatal("UserFromContext returned nil")
		}
		if u.ID != expectedUser.ID {
			t.Fatalf("user ID = %q, want %q", u.ID, expectedUser.ID)
		}
		if u.Email != expectedUser.Email {
			t.Fatalf("user Email = %q, want %q", u.Email, expectedUser.Email)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: rawToken})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !handlerCalled {
		t.Fatal("inner handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
