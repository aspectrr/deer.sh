package rest

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/orchestrator"
)

// handleListVMs godoc
// @Summary      List source VMs
// @Description  List all source VMs across connected hosts
// @Tags         Source VMs
// @Produce      json
// @Param        slug  path      string  true  "Organization slug"
// @Success      200   {object}  map[string]interface{}
// @Failure      403   {object}  error.ErrorResponse
// @Failure      404   {object}  error.ErrorResponse
// @Failure      500   {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/vms [get]
func (s *Server) handleListVMs(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	vms, err := s.orchestrator.ListVMs(r.Context(), org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list VMs"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"vms":   vms,
		"count": len(vms),
	})
}

// handlePrepareSourceVM godoc
// @Summary      Prepare source VM
// @Description  Prepare a source VM for sandbox cloning
// @Tags         Source VMs
// @Accept       json
// @Produce      json
// @Param        slug     path      string                       true  "Organization slug"
// @Param        vm       path      string                       true  "Source VM name"
// @Param        request  body      orchestrator.PrepareRequest  true  "SSH credentials"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sources/{vm}/prepare [post]
func (s *Server) handlePrepareSourceVM(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	vm := chi.URLParam(r, "vm")

	var req orchestrator.PrepareRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	result, err := s.orchestrator.PrepareSourceVM(r.Context(), org.ID, vm, req.SSHUser, req.SSHKeyPath)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to prepare source VM"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, result)
}

// handleRunSourceCommand godoc
// @Summary      Run source command
// @Description  Execute a read-only command on a source VM
// @Tags         Source VMs
// @Accept       json
// @Produce      json
// @Param        slug     path      string                          true  "Organization slug"
// @Param        vm       path      string                          true  "Source VM name"
// @Param        request  body      orchestrator.RunSourceRequest   true  "Command to run"
// @Success      200      {object}  orchestrator.SourceCommandResult
// @Failure      400      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sources/{vm}/run [post]
func (s *Server) handleRunSourceCommand(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	vm := chi.URLParam(r, "vm")

	var req orchestrator.RunSourceRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Command == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("command is required"))
		return
	}

	if len(req.Command) > 4096 {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("command too long (max 4096 bytes)"))
		return
	}

	result, err := s.orchestrator.RunSourceCommand(r.Context(), org.ID, vm, req.Command, req.TimeoutSec)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to run source command"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, result)
}

// handleReadSourceFile godoc
// @Summary      Read source file
// @Description  Read a file from a source VM
// @Tags         Source VMs
// @Accept       json
// @Produce      json
// @Param        slug     path      string                          true  "Organization slug"
// @Param        vm       path      string                          true  "Source VM name"
// @Param        request  body      orchestrator.ReadSourceRequest  true  "File path"
// @Success      200      {object}  orchestrator.SourceFileResult
// @Failure      400      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     CookieAuth
// @Router       /orgs/{slug}/sources/{vm}/read [post]
func (s *Server) handleReadSourceFile(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	vm := chi.URLParam(r, "vm")

	var req orchestrator.ReadSourceRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Path == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("path is required"))
		return
	}

	cleaned := filepath.Clean(req.Path)
	if !strings.HasPrefix(cleaned, "/") || strings.Contains(cleaned, "..") {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("invalid file path"))
		return
	}
	req.Path = cleaned

	result, err := s.orchestrator.ReadSourceFile(r.Context(), org.ID, vm, req.Path)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to read source file"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, result)
}
