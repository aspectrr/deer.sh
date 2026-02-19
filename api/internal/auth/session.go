package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// HashSessionToken returns the SHA-256 hex digest of a raw session token.
// The raw token is returned to the user in a cookie; the hash is stored in the DB.
func HashSessionToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

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

// CreateSession generates a random token, stores its SHA-256 hash as the
// session ID in the database, and returns both the raw token (for the cookie)
// and the session record.
func CreateSession(ctx context.Context, st store.Store, userID, ip, ua string, ttl time.Duration) (rawToken string, sess *store.Session, err error) {
	raw, err := generateSessionToken()
	if err != nil {
		return "", nil, err
	}

	sess = &store.Session{
		ID:        HashSessionToken(raw),
		UserID:    userID,
		IPAddress: ip,
		UserAgent: ua,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}

	if err := st.CreateSession(ctx, sess); err != nil {
		return "", nil, fmt.Errorf("create session: %w", err)
	}
	return raw, sess, nil
}

func SetSessionCookie(w http.ResponseWriter, token string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
