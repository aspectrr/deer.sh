package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// handleHealth godoc
// @Summary      Health check
// @Description  Returns API health status
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Register ---

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type authResponse struct {
	User *userResponse `json:"user"`
}

type userResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	EmailVerified bool   `json:"email_verified"`
}

// handleRegister godoc
// @Summary      Register a new user
// @Description  Create a new user account and return a session cookie
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      registerRequest  true  "Registration details"
// @Success      201      {object}  authResponse
// @Failure      400      {object}  swaggerError
// @Failure      409      {object}  swaggerError
// @Failure      500      {object}  swaggerError
// @Router       /auth/register [post]
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("email, password, and display_name are required"))
		return
	}

	if len(req.Password) < 8 {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("password must be at least 8 characters"))
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to hash password"))
		return
	}

	user := &store.User{
		ID:           "USR-" + uuid.New().String()[:8],
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		PasswordHash: hash,
	}

	if err := s.store.CreateUser(r.Context(), user); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			serverError.RespondError(w, http.StatusConflict, fmt.Errorf("email already registered"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create user"))
		return
	}

	sess, err := auth.CreateSession(r.Context(), s.store, user.ID, r.RemoteAddr, r.UserAgent(), s.cfg.Auth.SessionTTL)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create session"))
		return
	}

	auth.SetSessionCookie(w, sess.ID, s.cfg.Auth.SessionTTL, r.TLS != nil)

	_ = serverJSON.RespondJSON(w, http.StatusCreated, authResponse{
		User: &userResponse{
			ID:            user.ID,
			Email:         user.Email,
			DisplayName:   user.DisplayName,
			AvatarURL:     user.AvatarURL,
			EmailVerified: user.EmailVerified,
		},
	})
}

// --- Login ---

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleLogin godoc
// @Summary      Log in
// @Description  Authenticate with email and password, returns a session cookie
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      loginRequest  true  "Login credentials"
// @Success      200      {object}  authResponse
// @Failure      400      {object}  swaggerError
// @Failure      401      {object}  swaggerError
// @Failure      500      {object}  swaggerError
// @Router       /auth/login [post]
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Email == "" || req.Password == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("email and password are required"))
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusUnauthorized, fmt.Errorf("invalid email or password"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to look up user"))
		return
	}

	if user.PasswordHash == "" {
		serverError.RespondError(w, http.StatusUnauthorized, fmt.Errorf("this account uses OAuth login"))
		return
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		serverError.RespondError(w, http.StatusUnauthorized, fmt.Errorf("invalid email or password"))
		return
	}

	sess, err := auth.CreateSession(r.Context(), s.store, user.ID, r.RemoteAddr, r.UserAgent(), s.cfg.Auth.SessionTTL)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create session"))
		return
	}

	auth.SetSessionCookie(w, sess.ID, s.cfg.Auth.SessionTTL, r.TLS != nil)

	_ = serverJSON.RespondJSON(w, http.StatusOK, authResponse{
		User: &userResponse{
			ID:            user.ID,
			Email:         user.Email,
			DisplayName:   user.DisplayName,
			AvatarURL:     user.AvatarURL,
			EmailVerified: user.EmailVerified,
		},
	})
}

// --- Logout ---

// handleLogout godoc
// @Summary      Log out
// @Description  Invalidate the current session and clear the session cookie
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  map[string]string
// @Security     CookieAuth
// @Router       /auth/logout [post]
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		_ = s.store.DeleteSession(r.Context(), cookie.Value)
	}
	auth.ClearSessionCookie(w)
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// --- Me ---

// handleMe godoc
// @Summary      Get current user
// @Description  Return the currently authenticated user
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  authResponse
// @Failure      401  {object}  swaggerError
// @Security     CookieAuth
// @Router       /auth/me [get]
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		serverError.RespondError(w, http.StatusUnauthorized, fmt.Errorf("not authenticated"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, authResponse{
		User: &userResponse{
			ID:            user.ID,
			Email:         user.Email,
			DisplayName:   user.DisplayName,
			AvatarURL:     user.AvatarURL,
			EmailVerified: user.EmailVerified,
		},
	})
}

// --- GitHub OAuth ---

// handleGitHubLogin godoc
// @Summary      GitHub OAuth login
// @Description  Redirect to GitHub OAuth authorization page
// @Tags         Auth
// @Success      302  "Redirect to GitHub"
// @Failure      501  {object}  swaggerError
// @Router       /auth/github [get]
func (s *Server) handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Auth.GitHub.ClientID == "" {
		serverError.RespondError(w, http.StatusNotImplemented, fmt.Errorf("GitHub OAuth not configured"))
		return
	}
	cfg := auth.GitHubOAuthConfig(s.cfg.Auth.GitHub.ClientID, s.cfg.Auth.GitHub.ClientSecret, s.cfg.Auth.GitHub.RedirectURL)
	url := cfg.AuthCodeURL("state")
	http.Redirect(w, r, url, http.StatusFound)
}

// handleGitHubCallback godoc
// @Summary      GitHub OAuth callback
// @Description  Handle GitHub OAuth callback, create or link user, set session cookie, and redirect to dashboard
// @Tags         Auth
// @Param        code  query  string  true  "OAuth authorization code"
// @Success      302   "Redirect to dashboard"
// @Failure      400   {object}  swaggerError
// @Failure      500   {object}  swaggerError
// @Router       /auth/github/callback [get]
func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("missing code parameter"))
		return
	}

	cfg := auth.GitHubOAuthConfig(s.cfg.Auth.GitHub.ClientID, s.cfg.Auth.GitHub.ClientSecret, s.cfg.Auth.GitHub.RedirectURL)
	token, err := cfg.Exchange(r.Context(), code)
	if err != nil {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("failed to exchange code: %w", err))
		return
	}

	// Fetch user info from GitHub
	ghUser, err := fetchGitHubUser(r.Context(), token.AccessToken)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to fetch GitHub user: %w", err))
		return
	}

	user, err := s.findOrCreateOAuthUser(r.Context(), "github", fmt.Sprintf("%d", ghUser.ID), ghUser.Email, ghUser.Name, ghUser.AvatarURL, token.AccessToken, token.RefreshToken)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to process OAuth user: %w", err))
		return
	}

	sess, err := auth.CreateSession(r.Context(), s.store, user.ID, r.RemoteAddr, r.UserAgent(), s.cfg.Auth.SessionTTL)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create session"))
		return
	}

	auth.SetSessionCookie(w, sess.ID, s.cfg.Auth.SessionTTL, r.TLS != nil)
	http.Redirect(w, r, s.cfg.Frontend.URL+"/dashboard", http.StatusFound)
}

// --- Google OAuth ---

// handleGoogleLogin godoc
// @Summary      Google OAuth login
// @Description  Redirect to Google OAuth authorization page
// @Tags         Auth
// @Success      302  "Redirect to Google"
// @Failure      501  {object}  swaggerError
// @Router       /auth/google [get]
func (s *Server) handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Auth.Google.ClientID == "" {
		serverError.RespondError(w, http.StatusNotImplemented, fmt.Errorf("Google OAuth not configured"))
		return
	}
	cfg := auth.GoogleOAuthConfig(s.cfg.Auth.Google.ClientID, s.cfg.Auth.Google.ClientSecret, s.cfg.Auth.Google.RedirectURL)
	url := cfg.AuthCodeURL("state")
	http.Redirect(w, r, url, http.StatusFound)
}

// handleGoogleCallback godoc
// @Summary      Google OAuth callback
// @Description  Handle Google OAuth callback, create or link user, set session cookie, and redirect to dashboard
// @Tags         Auth
// @Param        code  query  string  true  "OAuth authorization code"
// @Success      302   "Redirect to dashboard"
// @Failure      400   {object}  swaggerError
// @Failure      500   {object}  swaggerError
// @Router       /auth/google/callback [get]
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("missing code parameter"))
		return
	}

	cfg := auth.GoogleOAuthConfig(s.cfg.Auth.Google.ClientID, s.cfg.Auth.Google.ClientSecret, s.cfg.Auth.Google.RedirectURL)
	token, err := cfg.Exchange(r.Context(), code)
	if err != nil {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("failed to exchange code: %w", err))
		return
	}

	gUser, err := fetchGoogleUser(r.Context(), token.AccessToken)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to fetch Google user: %w", err))
		return
	}

	user, err := s.findOrCreateOAuthUser(r.Context(), "google", gUser.ID, gUser.Email, gUser.Name, gUser.Picture, token.AccessToken, token.RefreshToken)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to process OAuth user: %w", err))
		return
	}

	sess, err := auth.CreateSession(r.Context(), s.store, user.ID, r.RemoteAddr, r.UserAgent(), s.cfg.Auth.SessionTTL)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create session"))
		return
	}

	auth.SetSessionCookie(w, sess.ID, s.cfg.Auth.SessionTTL, r.TLS != nil)
	http.Redirect(w, r, s.cfg.Frontend.URL+"/dashboard", http.StatusFound)
}

// --- OAuth helpers ---

type githubUserInfo struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func fetchGitHubUser(ctx context.Context, accessToken string) (*githubUserInfo, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var user githubUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	// Fetch primary email if not set
	if user.Email == "" {
		user.Email, _ = fetchGitHubPrimaryEmail(ctx, accessToken)
	}

	return &user, nil
}

func fetchGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", fmt.Errorf("no email found")
}

type googleUserInfo struct {
	ID      string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func fetchGoogleUser(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var user googleUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Server) findOrCreateOAuthUser(ctx context.Context, provider, providerID, email, name, avatarURL, accessToken, refreshToken string) (*store.User, error) {
	// Check if OAuth account exists
	oa, err := s.store.GetOAuthAccount(ctx, provider, providerID)
	if err == nil {
		// Account exists, get the user
		return s.store.GetUser(ctx, oa.UserID)
	}

	if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}

	// Check if user exists with same email
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}

	if errors.Is(err, store.ErrNotFound) {
		// Create new user
		user = &store.User{
			ID:            "USR-" + uuid.New().String()[:8],
			Email:         email,
			DisplayName:   name,
			AvatarURL:     avatarURL,
			EmailVerified: true,
		}
		if err := s.store.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}

	// Link OAuth account to user
	oauthAccount := &store.OAuthAccount{
		ID:           "OA-" + uuid.New().String()[:8],
		UserID:       user.ID,
		Provider:     provider,
		ProviderID:   providerID,
		Email:        email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	if err := s.store.CreateOAuthAccount(ctx, oauthAccount); err != nil {
		return nil, err
	}

	return user, nil
}
