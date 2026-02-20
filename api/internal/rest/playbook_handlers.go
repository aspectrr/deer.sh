package rest

// Playbook handlers - commented out, not yet ready for integration.

/*
import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/aspectrr/fluid.sh/api/internal/auth"
	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// --- Playbook CRUD ---

func (s *Server) handleCreatePlaybook(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("name is required"))
		return
	}

	pb := &store.Playbook{
		ID:          uuid.New().String(),
		OrgID:       org.ID,
		Name:        req.Name,
		Description: req.Description,
	}
	if err := s.store.CreatePlaybook(r.Context(), pb); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create playbook"))
		return
	}

	if user := auth.UserFromContext(r.Context()); user != nil {
		s.telemetry.Track(user.ID, "playbook_created", map[string]any{"org_id": org.ID})
	}

	_ = serverJSON.RespondJSON(w, http.StatusCreated, pb)
}

func (s *Server) handleListPlaybooks(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	playbooks, err := s.store.ListPlaybooksByOrg(r.Context(), org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list playbooks"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"playbooks": playbooks,
		"count":     len(playbooks),
	})
}

func (s *Server) handleGetPlaybook(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	playbookID := chi.URLParam(r, "playbookID")
	pb, err := s.store.GetPlaybook(r.Context(), playbookID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("playbook not found"))
		return
	}
	if pb.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("playbook does not belong to this organization"))
		return
	}

	tasks, _ := s.store.ListPlaybookTasks(r.Context(), playbookID)
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"playbook": pb,
		"tasks":    tasks,
	})
}

func (s *Server) handleUpdatePlaybook(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	playbookID := chi.URLParam(r, "playbookID")
	pb, err := s.store.GetPlaybook(r.Context(), playbookID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("playbook not found"))
		return
	}
	if pb.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("playbook does not belong to this organization"))
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Name != nil {
		pb.Name = *req.Name
	}
	if req.Description != nil {
		pb.Description = *req.Description
	}

	if err := s.store.UpdatePlaybook(r.Context(), pb); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to update playbook"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, pb)
}

func (s *Server) handleDeletePlaybook(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	playbookID := chi.URLParam(r, "playbookID")
	pb, err := s.store.GetPlaybook(r.Context(), playbookID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("playbook not found"))
		return
	}
	if pb.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("playbook does not belong to this organization"))
		return
	}

	if err := s.store.DeletePlaybook(r.Context(), playbookID); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete playbook"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{"deleted": true, "playbook_id": playbookID})
}

// --- Playbook Task CRUD ---

func (s *Server) handleCreatePlaybookTask(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	playbookID := chi.URLParam(r, "playbookID")
	pb, err := s.store.GetPlaybook(r.Context(), playbookID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("playbook not found"))
		return
	}
	if pb.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("playbook does not belong to this organization"))
		return
	}

	var req struct {
		Name   string          `json:"name"`
		Module string          `json:"module"`
		Params json.RawMessage `json:"params"`
	}
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" || req.Module == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("name and module are required"))
		return
	}

	paramsStr := "{}"
	if len(req.Params) > 0 {
		paramsStr = string(req.Params)
	}

	task := &store.PlaybookTask{
		ID:         uuid.New().String(),
		PlaybookID: playbookID,
		Name:       req.Name,
		Module:     req.Module,
		Params:     paramsStr,
	}
	if err := s.store.CreatePlaybookTask(r.Context(), task); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create task"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusCreated, task)
}

func (s *Server) handleListPlaybookTasks(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	playbookID := chi.URLParam(r, "playbookID")
	pb, err := s.store.GetPlaybook(r.Context(), playbookID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("playbook not found"))
		return
	}
	if pb.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("playbook does not belong to this organization"))
		return
	}

	tasks, err := s.store.ListPlaybookTasks(r.Context(), playbookID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list tasks"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
		"count": len(tasks),
	})
}

func (s *Server) handleUpdatePlaybookTask(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	playbookID := chi.URLParam(r, "playbookID")
	pb, err := s.store.GetPlaybook(r.Context(), playbookID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("playbook not found"))
		return
	}
	if pb.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("playbook does not belong to this organization"))
		return
	}

	taskID := chi.URLParam(r, "taskID")
	task, err := s.store.GetPlaybookTask(r.Context(), taskID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("task not found"))
		return
	}
	if task.PlaybookID != playbookID {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("task does not belong to this playbook"))
		return
	}

	var req struct {
		Name   *string          `json:"name"`
		Module *string          `json:"module"`
		Params *json.RawMessage `json:"params"`
	}
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Module != nil {
		task.Module = *req.Module
	}
	if req.Params != nil {
		task.Params = string(*req.Params)
	}

	if err := s.store.UpdatePlaybookTask(r.Context(), task); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to update task"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, task)
}

func (s *Server) handleDeletePlaybookTask(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	playbookID := chi.URLParam(r, "playbookID")
	pb, err := s.store.GetPlaybook(r.Context(), playbookID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("playbook not found"))
		return
	}
	if pb.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("playbook does not belong to this organization"))
		return
	}

	taskID := chi.URLParam(r, "taskID")
	task, err := s.store.GetPlaybookTask(r.Context(), taskID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("task not found"))
		return
	}
	if task.PlaybookID != playbookID {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("task does not belong to this playbook"))
		return
	}

	if err := s.store.DeletePlaybookTask(r.Context(), taskID); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete task"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{"deleted": true, "task_id": taskID})
}

func (s *Server) handleReorderPlaybookTasks(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	playbookID := chi.URLParam(r, "playbookID")
	pb, err := s.store.GetPlaybook(r.Context(), playbookID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("playbook not found"))
		return
	}
	if pb.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("playbook does not belong to this organization"))
		return
	}

	var req struct {
		TaskIDs []string `json:"task_ids"`
	}
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if err := s.store.ReorderPlaybookTasks(r.Context(), playbookID, req.TaskIDs); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to reorder tasks"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{"reordered": true})
}
*/
