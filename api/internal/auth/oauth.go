package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

const (
	OAuthStateCookieName = "fluid_oauth_state"
	oauthStateLen        = 32
	oauthStateMaxAge     = 600 // 10 minutes
)

func GenerateOAuthState() (string, error) {
	b := make([]byte, oauthStateLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func SetOAuthStateCookie(w http.ResponseWriter, state string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   oauthStateMaxAge,
	})
}

func ClearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func ValidateOAuthState(r *http.Request) error {
	state := r.URL.Query().Get("state")
	if state == "" {
		return fmt.Errorf("missing state parameter")
	}

	cookie, err := r.Cookie(OAuthStateCookieName)
	if err != nil {
		return fmt.Errorf("missing oauth state cookie")
	}

	if subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		return fmt.Errorf("state mismatch")
	}

	return nil
}

func GitHubOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
}

func GoogleOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}
