package rest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/orchestrator"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// handleCreateSandbox godoc
// @Summary      Create sandbox
// @Description  Create a new sandbox in the organization from a source VM or base image
// @Tags         Sandboxes
// @Accept       json
// @Produce      json
// @Param        slug     path      string                          true  "Organization slug"
// @Param        request  body      orchestrator.CreateSandboxRequest  true  "Sandbox configuration"
// @Success      201      {object}  store.Sandbox
// @Failure      400      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes [post]
func (s *Server) handleCreateSandbox(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	var req orchestrator.CreateSandboxRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.SourceVM == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("source_vm is required"))
		return
	}

	req.OrgID = org.ID

	sandbox, err := s.orchestrator.CreateSandbox(r.Context(), req)
	if err != nil {
		s.logger.Error("failed to create sandbox", "error", err)
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create sandbox"))
		return
	}

	if user := auth.UserFromContext(r.Context()); user != nil {
		s.telemetry.Track(user.ID, "sandbox_created", map[string]any{"org_id": org.ID, "source_vm": req.SourceVM})
	}

	_ = serverJSON.RespondJSON(w, http.StatusCreated, sandbox)
}

// handleListSandboxes godoc
// @Summary      List sandboxes
// @Description  List all sandboxes in the organization
// @Tags         Sandboxes
// @Produce      json
// @Param        slug  path      string  true  "Organization slug"
// @Success      200   {object}  map[string]interface{}
// @Failure      403   {object}  error.ErrorResponse
// @Failure      404   {object}  error.ErrorResponse
// @Failure      500   {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes [get]
func (s *Server) handleListSandboxes(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	sandboxes, err := s.orchestrator.ListSandboxesByOrg(r.Context(), org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list sandboxes"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"sandboxes": sandboxes,
		"count":     len(sandboxes),
	})
}

// handleGetSandbox godoc
// @Summary      Get sandbox
// @Description  Get sandbox details by ID
// @Tags         Sandboxes
// @Produce      json
// @Param        slug       path      string  true  "Organization slug"
// @Param        sandboxID  path      string  true  "Sandbox ID"
// @Success      200        {object}  store.Sandbox
// @Failure      403        {object}  error.ErrorResponse
// @Failure      404        {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes/{sandboxID} [get]
func (s *Server) handleGetSandbox(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	sandboxID := chi.URLParam(r, "sandboxID")
	sandbox, err := s.orchestrator.GetSandbox(r.Context(), org.ID, sandboxID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox not found"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, sandbox)
}

// handleDestroySandbox godoc
// @Summary      Destroy sandbox
// @Description  Destroy a sandbox and release its resources
// @Tags         Sandboxes
// @Produce      json
// @Param        slug       path      string  true  "Organization slug"
// @Param        sandboxID  path      string  true  "Sandbox ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      403        {object}  error.ErrorResponse
// @Failure      404        {object}  error.ErrorResponse
// @Failure      500        {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes/{sandboxID} [delete]
func (s *Server) handleDestroySandbox(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgRole(w, r, store.OrgRoleAdmin)
	if !ok {
		return
	}

	sandboxID := chi.URLParam(r, "sandboxID")

	if err := s.orchestrator.DestroySandbox(r.Context(), org.ID, sandboxID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox not found"))
			return
		}
		s.logger.Error("failed to destroy sandbox", "sandbox_id", sandboxID, "error", err)
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to destroy sandbox"))
		return
	}

	if user := auth.UserFromContext(r.Context()); user != nil {
		s.telemetry.Track(user.ID, "sandbox_destroyed", map[string]any{"org_id": org.ID})
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"destroyed":  true,
		"sandbox_id": sandboxID,
	})
}

// handleRunCommand godoc
// @Summary      Run command
// @Description  Execute a command in a sandbox
// @Tags         Sandboxes
// @Accept       json
// @Produce      json
// @Param        slug       path      string                          true  "Organization slug"
// @Param        sandboxID  path      string                          true  "Sandbox ID"
// @Param        request    body      orchestrator.RunCommandRequest  true  "Command to run"
// @Success      200        {object}  store.Command
// @Failure      400        {object}  error.ErrorResponse
// @Failure      403        {object}  error.ErrorResponse
// @Failure      404        {object}  error.ErrorResponse
// @Failure      500        {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes/{sandboxID}/run [post]
func (s *Server) handleRunCommand(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	sandboxID := chi.URLParam(r, "sandboxID")

	var req orchestrator.RunCommandRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Command == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("command is required"))
		return
	}

	const maxCommandLen = 65536 // 64 KiB
	if len(req.Command) > maxCommandLen {
		serverError.RespondError(w, http.StatusBadRequest,
			fmt.Errorf("command exceeds maximum length of %d bytes", maxCommandLen))
		return
	}

	const maxTimeoutSec = 3600
	if req.TimeoutSec > maxTimeoutSec {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("timeout_seconds must be <= %d", maxTimeoutSec))
		return
	}

	result, err := s.orchestrator.RunCommand(r.Context(), org.ID, sandboxID, req.Command, req.TimeoutSec)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox not found"))
			return
		}
		s.logger.Error("failed to run command", "sandbox_id", sandboxID, "error", err)
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to run command"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, result)
}

// handleStartSandbox godoc
// @Summary      Start sandbox
// @Description  Start a stopped sandbox
// @Tags         Sandboxes
// @Produce      json
// @Param        slug       path      string  true  "Organization slug"
// @Param        sandboxID  path      string  true  "Sandbox ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      403        {object}  error.ErrorResponse
// @Failure      404        {object}  error.ErrorResponse
// @Failure      500        {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes/{sandboxID}/start [post]
func (s *Server) handleStartSandbox(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	sandboxID := chi.URLParam(r, "sandboxID")

	if err := s.orchestrator.StartSandbox(r.Context(), org.ID, sandboxID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox not found"))
			return
		}
		s.logger.Error("failed to start sandbox", "sandbox_id", sandboxID, "error", err)
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to start sandbox"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"started":    true,
		"sandbox_id": sandboxID,
	})
}

// handleStopSandbox godoc
// @Summary      Stop sandbox
// @Description  Stop a running sandbox
// @Tags         Sandboxes
// @Produce      json
// @Param        slug       path      string  true  "Organization slug"
// @Param        sandboxID  path      string  true  "Sandbox ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      403        {object}  error.ErrorResponse
// @Failure      404        {object}  error.ErrorResponse
// @Failure      500        {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes/{sandboxID}/stop [post]
func (s *Server) handleStopSandbox(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	sandboxID := chi.URLParam(r, "sandboxID")

	if err := s.orchestrator.StopSandbox(r.Context(), org.ID, sandboxID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox not found"))
			return
		}
		s.logger.Error("failed to stop sandbox", "sandbox_id", sandboxID, "error", err)
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to stop sandbox"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"stopped":    true,
		"sandbox_id": sandboxID,
	})
}

// handleGetSandboxIP godoc
// @Summary      Get sandbox IP
// @Description  Get the IP address of a sandbox
// @Tags         Sandboxes
// @Produce      json
// @Param        slug       path      string  true  "Organization slug"
// @Param        sandboxID  path      string  true  "Sandbox ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      403        {object}  error.ErrorResponse
// @Failure      404        {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes/{sandboxID}/ip [get]
func (s *Server) handleGetSandboxIP(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	sandboxID := chi.URLParam(r, "sandboxID")

	sandbox, err := s.orchestrator.GetSandbox(r.Context(), org.ID, sandboxID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox not found"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"sandbox_id": sandboxID,
		"ip_address": sandbox.IPAddress,
	})
}

// handleCreateSnapshot godoc
// @Summary      Create snapshot
// @Description  Create a snapshot of a sandbox
// @Tags         Sandboxes
// @Accept       json
// @Produce      json
// @Param        slug       path      string                       true  "Organization slug"
// @Param        sandboxID  path      string                       true  "Sandbox ID"
// @Param        request    body      orchestrator.SnapshotRequest true  "Snapshot details"
// @Success      201        {object}  orchestrator.SnapshotResponse
// @Failure      400        {object}  error.ErrorResponse
// @Failure      403        {object}  error.ErrorResponse
// @Failure      404        {object}  error.ErrorResponse
// @Failure      500        {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes/{sandboxID}/snapshot [post]
func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	sandboxID := chi.URLParam(r, "sandboxID")

	var req orchestrator.SnapshotRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	result, err := s.orchestrator.CreateSnapshot(r.Context(), org.ID, sandboxID, req.Name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox not found"))
			return
		}
		s.logger.Error("failed to create snapshot", "sandbox_id", sandboxID, "error", err)
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create snapshot"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusCreated, result)
}

// handleListCommands godoc
// @Summary      List commands
// @Description  List all commands executed in a sandbox
// @Tags         Sandboxes
// @Produce      json
// @Param        slug       path      string  true  "Organization slug"
// @Param        sandboxID  path      string  true  "Sandbox ID"
// @Success      200        {object}  map[string]interface{}
// @Failure      403        {object}  error.ErrorResponse
// @Failure      404        {object}  error.ErrorResponse
// @Failure      500        {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sandboxes/{sandboxID}/commands [get]
func (s *Server) handleListCommands(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	sandboxID := chi.URLParam(r, "sandboxID")

	commands, err := s.orchestrator.ListCommands(r.Context(), org.ID, sandboxID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list commands"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"commands": commands,
		"count":    len(commands),
	})
}

// resolveOrgMembership resolves org from {slug} URL param and verifies user membership.
// Returns the org, member, and true if successful; writes error response and returns false otherwise.
func (s *Server) resolveOrgMembership(w http.ResponseWriter, r *http.Request) (*store.Organization, *store.OrgMember, bool) {
	slug := chi.URLParam(r, "slug")
	user := auth.UserFromContext(r.Context())
	if user == nil {
		serverError.RespondError(w, http.StatusUnauthorized, fmt.Errorf("authentication required"))
		return nil, nil, false
	}

	org, err := s.store.GetOrganizationBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("organization not found"))
			return nil, nil, false
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get organization"))
		return nil, nil, false
	}

	member, err := s.store.GetOrgMember(r.Context(), org.ID, user.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("not a member of this organization"))
		return nil, nil, false
	}

	return org, member, true
}

// resolveOrgRole resolves org membership and checks the member has at least the given role.
func (s *Server) resolveOrgRole(w http.ResponseWriter, r *http.Request, minRole store.OrgRole) (*store.Organization, *store.OrgMember, bool) {
	org, member, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return nil, nil, false
	}
	if !hasMinRole(member.Role, minRole) {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("insufficient permissions"))
		return nil, nil, false
	}
	return org, member, true
}

var roleRanks = map[store.OrgRole]int{
	store.OrgRoleOwner:  3,
	store.OrgRoleAdmin:  2,
	store.OrgRoleMember: 1,
}

func hasMinRole(role, minRole store.OrgRole) bool {
	return roleRanks[role] >= roleRanks[minRole]
}
