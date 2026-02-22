package rest

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	"github.com/aspectrr/fluid.sh/api/internal/id"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// handleListHosts godoc
// @Summary      List hosts
// @Description  List all connected sandbox hosts
// @Tags         Hosts
// @Produce      json
// @Param        slug  path      string  true  "Organization slug"
// @Success      200   {object}  map[string]interface{}
// @Failure      403   {object}  error.ErrorResponse
// @Failure      404   {object}  error.ErrorResponse
// @Failure      500   {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug}/hosts [get]
func (s *Server) handleListHosts(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	hosts, err := s.orchestrator.ListHosts(r.Context(), org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list hosts"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"hosts": hosts,
		"count": len(hosts),
	})
}

// handleGetHost godoc
// @Summary      Get host
// @Description  Get details of a specific connected host
// @Tags         Hosts
// @Produce      json
// @Param        slug    path      string  true  "Organization slug"
// @Param        hostID  path      string  true  "Host ID"
// @Success      200     {object}  orchestrator.HostInfo
// @Failure      403     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug}/hosts/{hostID} [get]
func (s *Server) handleGetHost(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	hostID := chi.URLParam(r, "hostID")
	host, err := s.orchestrator.GetHost(r.Context(), hostID, org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("host not found or not connected"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, host)
}

// --- Host Tokens ---

type createHostTokenRequest struct {
	Name string `json:"name"`
}

type hostTokenResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Token     string `json:"token,omitempty"` // Only set on creation.
	CreatedAt string `json:"created_at"`
}

// handleCreateHostToken godoc
// @Summary      Create host token
// @Description  Generate a new host authentication token (owner or admin only)
// @Tags         Host Tokens
// @Accept       json
// @Produce      json
// @Param        slug     path      string                  true  "Organization slug"
// @Param        request  body      createHostTokenRequest  true  "Token details"
// @Success      201      {object}  hostTokenResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug}/hosts/tokens [post]
func (s *Server) handleCreateHostToken(w http.ResponseWriter, r *http.Request) {
	org, member, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	if member.Role != store.OrgRoleOwner && member.Role != store.OrgRoleAdmin {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("insufficient permissions"))
		return
	}

	var req createHostTokenRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Name == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("name is required"))
		return
	}

	// Generate raw token.
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate token"))
		return
	}
	rawToken := hex.EncodeToString(rawBytes)

	tokenID, err := id.Generate("HTK-")
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate token ID"))
		return
	}

	token := &store.HostToken{
		ID:        tokenID,
		OrgID:     org.ID,
		Name:      req.Name,
		TokenHash: auth.HashToken(rawToken),
	}

	if err := s.store.CreateHostToken(r.Context(), token); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create host token"))
		return
	}

	if user := auth.UserFromContext(r.Context()); user != nil {
		s.telemetry.Track(user.ID, "host_token_created", map[string]any{"org_id": org.ID})
	}

	_ = serverJSON.RespondJSON(w, http.StatusCreated, hostTokenResponse{
		ID:        token.ID,
		Name:      token.Name,
		Token:     rawToken,
		CreatedAt: token.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// handleListHostTokens godoc
// @Summary      List host tokens
// @Description  List all host tokens for the organization
// @Tags         Host Tokens
// @Produce      json
// @Param        slug  path      string  true  "Organization slug"
// @Success      200   {object}  map[string]interface{}
// @Failure      403   {object}  error.ErrorResponse
// @Failure      404   {object}  error.ErrorResponse
// @Failure      500   {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug}/hosts/tokens [get]
func (s *Server) handleListHostTokens(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	tokens, err := s.store.ListHostTokensByOrg(r.Context(), org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list host tokens"))
		return
	}

	result := make([]hostTokenResponse, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, hostTokenResponse{
			ID:        t.ID,
			Name:      t.Name,
			CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"tokens": result,
		"count":  len(result),
	})
}

// handleDeleteHostToken godoc
// @Summary      Delete host token
// @Description  Delete a host token (owner or admin only)
// @Tags         Host Tokens
// @Produce      json
// @Param        slug     path      string  true  "Organization slug"
// @Param        tokenID  path      string  true  "Token ID"
// @Success      200      {object}  map[string]string
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /v1/orgs/{slug}/hosts/tokens/{tokenID} [delete]
func (s *Server) handleDeleteHostToken(w http.ResponseWriter, r *http.Request) {
	org, member, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	if member.Role != store.OrgRoleOwner && member.Role != store.OrgRoleAdmin {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("insufficient permissions"))
		return
	}

	tokenID := chi.URLParam(r, "tokenID")
	if err := s.store.DeleteHostToken(r.Context(), org.ID, tokenID); err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("host token not found"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
