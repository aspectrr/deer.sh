package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

const (
	SessionCookieName = "fluid_session"
	sessionTokenLen   = 32
)

func generateSessionToken() (string, error) {
	b := make([]byte, sessionTokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func CreateSession(ctx context.Context, st store.Store, userID, ip, ua string, ttl time.Duration) (*store.Session, error) {
	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	sess := &store.Session{
		ID:        token,
		UserID:    userID,
		IPAddress: ip,
		UserAgent: ua,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}

	if err := st.CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return sess, nil
}

func SetSessionCookie(w http.ResponseWriter, token string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}
