package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	"github.com/aspectrr/fluid.sh/api/internal/id"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

var oauthHTTPClient = &http.Client{Timeout: 10 * time.Second}

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
// @Failure      400      {object}  error.ErrorResponse
// @Failure      409      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
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

	if _, err := mail.ParseAddress(req.Email); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("invalid email format"))
		return
	}

	if len(req.Password) < 8 {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("password must be at least 8 characters"))
		return
	}

	if len(req.Password) > 72 {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("password must be at most 72 characters"))
		return
	}

	req.Email = strings.ToLower(req.Email)

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to hash password"))
		return
	}

	userID, err := id.Generate("USR-")
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate user ID"))
		return
	}

	user := &store.User{
		ID:           userID,
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

	rawToken, _, err := auth.CreateSession(r.Context(), s.store, user.ID, r.RemoteAddr, r.UserAgent(), s.cfg.Auth.SessionTTL)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create session"))
		return
	}

	auth.SetSessionCookie(w, rawToken, s.cfg.Auth.SessionTTL, s.cfg.Auth.SecureCookies)

	s.telemetry.Track(user.ID, "user_registered", map[string]any{"provider": "password"})

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
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
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

	if len(req.Password) > 72 {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("password too long (max 72 characters)"))
		return
	}

	req.Email = strings.ToLower(req.Email)

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

	rawToken, _, err := auth.CreateSession(r.Context(), s.store, user.ID, r.RemoteAddr, r.UserAgent(), s.cfg.Auth.SessionTTL)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create session"))
		return
	}

	auth.SetSessionCookie(w, rawToken, s.cfg.Auth.SessionTTL, s.cfg.Auth.SecureCookies)

	s.telemetry.Track(user.ID, "user_logged_in", map[string]any{"provider": "password"})

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
		_ = s.store.DeleteSession(r.Context(), auth.HashSessionToken(cookie.Value))
	}
	auth.ClearSessionCookie(w, s.cfg.Auth.SecureCookies)
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// --- Me ---

// handleMe godoc
// @Summary      Get current user
// @Description  Return the currently authenticated user
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  authResponse
// @Failure      401  {object}  error.ErrorResponse
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

// --- Onboarding ---

type onboardingRequest struct {
	OrgName        string   `json:"org_name"`
	Role           string   `json:"role"`
	UseCases       []string `json:"use_cases"`
	ReferralSource string   `json:"referral_source"`
}

// handleOnboarding godoc
// @Summary      Complete onboarding
// @Description  Create the user's first organization during onboarding
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      onboardingRequest  true  "Onboarding details"
// @Success      201      {object}  orgResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      409      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /auth/onboarding [post]
func (s *Server) handleOnboarding(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		serverError.RespondError(w, http.StatusUnauthorized, fmt.Errorf("not authenticated"))
		return
	}

	var req onboardingRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.OrgName == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("org_name is required"))
		return
	}

	slug := strings.ToLower(req.OrgName)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, slug)
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if !slugRegex.MatchString(slug) {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("org name must produce a valid slug (3-50 lowercase alphanumeric chars and hyphens)"))
		return
	}

	orgID, err := id.Generate("ORG-")
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate org ID"))
		return
	}

	org := &store.Organization{
		ID:      orgID,
		Name:    req.OrgName,
		Slug:    slug,
		OwnerID: user.ID,
	}

	baseSlug := slug
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			suffix := generateSlugSuffix()
			maxBase := maxSlugLen - 1 - len(suffix)
			b := baseSlug
			if len(b) > maxBase {
				b = strings.TrimRight(b[:maxBase], "-")
			}
			slug = b + "-" + suffix
			org.Slug = slug
		}

		err = s.store.WithTx(r.Context(), func(tx store.DataStore) error {
			if err := tx.CreateOrganization(r.Context(), org); err != nil {
				return err
			}
			memberID, err := id.Generate("MBR-")
			if err != nil {
				return fmt.Errorf("generate member ID: %w", err)
			}
			member := &store.OrgMember{
				ID:     memberID,
				OrgID:  org.ID,
				UserID: user.ID,
				Role:   store.OrgRoleOwner,
			}
			return tx.CreateOrgMember(r.Context(), member)
		})
		if err == nil {
			break
		}
		if !isDuplicateSlugErr(err) {
			break
		}
	}

	if err != nil {
		serverError.RespondErrorMsg(w, http.StatusInternalServerError, "failed to create organization", err)
		return
	}

	s.telemetry.Track(user.ID, "user_onboarded", map[string]any{
		"org_slug":        slug,
		"role":            req.Role,
		"use_cases":       req.UseCases,
		"referral_source": req.ReferralSource,
	})

	_ = serverJSON.RespondJSON(w, http.StatusCreated, toOrgResponseForOwner(org))
}

// --- GitHub OAuth ---

// handleGitHubLogin godoc
// @Summary      GitHub OAuth login
// @Description  Redirect to GitHub OAuth authorization page
// @Tags         Auth
// @Success      302  "Redirect to GitHub"
// @Failure      501  {object}  error.ErrorResponse
// @Router       /auth/github [get]
func (s *Server) handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Auth.GitHub.ClientID == "" {
		serverError.RespondError(w, http.StatusNotImplemented, fmt.Errorf("GitHub OAuth not configured"))
		return
	}
	state, err := auth.GenerateOAuthState()
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate oauth state"))
		return
	}
	auth.SetOAuthStateCookie(w, state, s.cfg.Auth.SecureCookies)
	cfg := auth.GitHubOAuthConfig(s.cfg.Auth.GitHub.ClientID, s.cfg.Auth.GitHub.ClientSecret, s.cfg.Auth.GitHub.RedirectURL)
	url := cfg.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusFound)
}

// handleGitHubCallback godoc
// @Summary      GitHub OAuth callback
// @Description  Handle GitHub OAuth callback, create or link user, set session cookie, and redirect to dashboard
// @Tags         Auth
// @Param        code   query  string  true  "OAuth authorization code"
// @Param        state  query  string  true  "OAuth CSRF state parameter"
// @Success      302    "Redirect to dashboard"
// @Failure      400    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Router       /auth/github/callback [get]
func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if err := auth.ValidateOAuthState(r); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("invalid oauth state: %w", err))
		return
	}
	auth.ClearOAuthStateCookie(w)

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
	ghUser, emailVerified, err := fetchGitHubUser(r.Context(), token.AccessToken)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to fetch GitHub user: %w", err))
		return
	}

	user, err := s.findOrCreateOAuthUser(r.Context(), "github", fmt.Sprintf("%d", ghUser.ID), ghUser.Email, ghUser.Name, ghUser.AvatarURL, token.AccessToken, token.RefreshToken, emailVerified)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to process OAuth user: %w", err))
		return
	}

	rawToken, _, err := auth.CreateSession(r.Context(), s.store, user.ID, r.RemoteAddr, r.UserAgent(), s.cfg.Auth.SessionTTL)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create session"))
		return
	}

	auth.SetSessionCookie(w, rawToken, s.cfg.Auth.SessionTTL, s.cfg.Auth.SecureCookies)
	s.telemetry.Track(user.ID, "user_logged_in", map[string]any{"provider": "github"})
	http.Redirect(w, r, s.cfg.Frontend.URL+"/dashboard", http.StatusFound)
}

// --- Google OAuth ---

// handleGoogleLogin godoc
// @Summary      Google OAuth login
// @Description  Redirect to Google OAuth authorization page
// @Tags         Auth
// @Success      302  "Redirect to Google"
// @Failure      501  {object}  error.ErrorResponse
// @Router       /auth/google [get]
func (s *Server) handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Auth.Google.ClientID == "" {
		serverError.RespondError(w, http.StatusNotImplemented, fmt.Errorf("google OAuth not configured"))
		return
	}
	state, err := auth.GenerateOAuthState()
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate oauth state"))
		return
	}
	auth.SetOAuthStateCookie(w, state, s.cfg.Auth.SecureCookies)
	cfg := auth.GoogleOAuthConfig(s.cfg.Auth.Google.ClientID, s.cfg.Auth.Google.ClientSecret, s.cfg.Auth.Google.RedirectURL)
	url := cfg.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusFound)
}

// handleGoogleCallback godoc
// @Summary      Google OAuth callback
// @Description  Handle Google OAuth callback, create or link user, set session cookie, and redirect to dashboard
// @Tags         Auth
// @Param        code   query  string  true  "OAuth authorization code"
// @Param        state  query  string  true  "OAuth CSRF state parameter"
// @Success      302    "Redirect to dashboard"
// @Failure      400    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Router       /auth/google/callback [get]
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if err := auth.ValidateOAuthState(r); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("invalid oauth state: %w", err))
		return
	}
	auth.ClearOAuthStateCookie(w)

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

	user, err := s.findOrCreateOAuthUser(r.Context(), "google", gUser.ID, gUser.Email, gUser.Name, gUser.Picture, token.AccessToken, token.RefreshToken, gUser.EmailVerified)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to process OAuth user: %w", err))
		return
	}

	rawToken, _, err := auth.CreateSession(r.Context(), s.store, user.ID, r.RemoteAddr, r.UserAgent(), s.cfg.Auth.SessionTTL)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create session"))
		return
	}

	auth.SetSessionCookie(w, rawToken, s.cfg.Auth.SessionTTL, s.cfg.Auth.SecureCookies)
	s.telemetry.Track(user.ID, "user_logged_in", map[string]any{"provider": "google"})
	http.Redirect(w, r, s.cfg.Frontend.URL+"/dashboard", http.StatusFound)
}

// --- OAuth helpers ---

type githubUserInfo struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func fetchGitHubUser(ctx context.Context, accessToken string) (*githubUserInfo, bool, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("GitHub user API returned status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var user githubUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, false, err
	}

	emailVerified := false
	if user.Email == "" {
		email, verified, err := fetchGitHubPrimaryEmail(ctx, accessToken)
		if err == nil {
			user.Email = email
			emailVerified = verified
		}
	} else {
		// Email from /user endpoint - check verification via emails API
		_, verified, err := fetchGitHubPrimaryEmail(ctx, accessToken)
		if err == nil {
			emailVerified = verified
		}
	}

	return &user, emailVerified, nil
}

func fetchGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, bool, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("GitHub emails API returned status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", false, err
	}
	// Prefer verified+primary
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, true, nil
		}
	}
	// Fall back to any primary
	for _, e := range emails {
		if e.Primary {
			return e.Email, e.Verified, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, emails[0].Verified, nil
	}
	return "", false, fmt.Errorf("no email found")
}

type googleUserInfo struct {
	ID            string `json:"sub"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	EmailVerified bool   `json:"email_verified"`
}

func fetchGoogleUser(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo API returned status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var user googleUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Server) findOrCreateOAuthUser(ctx context.Context, provider, providerID, email, name, avatarURL, accessToken, refreshToken string, emailVerified bool) (*store.User, error) {
	email = strings.ToLower(email)

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

	// If user was found by email, refuse to link if OAuth email is not verified
	if err == nil && !emailVerified {
		return nil, fmt.Errorf("oauth email not verified, cannot link to existing account")
	}

	if errors.Is(err, store.ErrNotFound) {
		// Create new user
		newUserID, err := id.Generate("USR-")
		if err != nil {
			return nil, fmt.Errorf("generate user ID: %w", err)
		}
		user = &store.User{
			ID:            newUserID,
			Email:         email,
			DisplayName:   name,
			AvatarURL:     avatarURL,
			EmailVerified: emailVerified,
		}
		if err := s.store.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}

	// Link OAuth account to user
	oaID, err := id.Generate("OA-")
	if err != nil {
		return nil, fmt.Errorf("generate oauth account ID: %w", err)
	}
	oauthAccount := &store.OAuthAccount{
		ID:           oaID,
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
