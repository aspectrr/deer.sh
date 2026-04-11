package rest

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	serverError "github.com/aspectrr/deer.sh/api/internal/error"
	serverJSON "github.com/aspectrr/deer.sh/api/internal/json"
	"github.com/aspectrr/deer.sh/api/internal/store"
)

type kafkaCaptureConfigRequest struct {
	SourceHostID       string   `json:"source_host_id"`
	SourceVM           string   `json:"source_vm"`
	Name               string   `json:"name"`
	BootstrapServers   []string `json:"bootstrap_servers"`
	Topics             []string `json:"topics"`
	Username           string   `json:"username"`
	Password           string   `json:"password,omitempty"`
	SASLMechanism      string   `json:"sasl_mechanism,omitempty"`
	TLSEnabled         bool     `json:"tls_enabled"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify"`
	TLSCAPEM           string   `json:"tls_ca_pem,omitempty"`
	Codec              string   `json:"codec"`
	RedactionRules     []string `json:"redaction_rules"`
	MaxBufferAgeSecs   int32    `json:"max_buffer_age_seconds"`
	MaxBufferBytes     int64    `json:"max_buffer_bytes"`
	Enabled            bool     `json:"enabled"`
}

func (s *Server) handleCreateKafkaCaptureConfig(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	var req kafkaCaptureConfigRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}
	if req.SourceHostID == "" || req.SourceVM == "" || len(req.BootstrapServers) == 0 || len(req.Topics) == 0 {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("source_host_id, source_vm, bootstrap_servers, and topics are required"))
		return
	}
	if req.Codec == "" {
		req.Codec = "json"
	}
	if req.SASLMechanism == "" {
		req.SASLMechanism = "plain"
	}

	cfg := &store.KafkaCaptureConfig{
		ID:                 uuid.New().String(),
		OrgID:              org.ID,
		SourceHostID:       req.SourceHostID,
		SourceVM:           req.SourceVM,
		Name:               req.Name,
		BootstrapServers:   store.StringSlice(req.BootstrapServers),
		Topics:             store.StringSlice(req.Topics),
		Username:           req.Username,
		Password:           req.Password,
		SASLMechanism:      req.SASLMechanism,
		TLSEnabled:         req.TLSEnabled,
		InsecureSkipVerify: req.InsecureSkipVerify,
		TLSCAPEM:           req.TLSCAPEM,
		Codec:              req.Codec,
		RedactionRules:     store.StringSlice(req.RedactionRules),
		MaxBufferAgeSecs:   req.MaxBufferAgeSecs,
		MaxBufferBytes:     req.MaxBufferBytes,
		Enabled:            req.Enabled,
		LastCaptureState:   "pending",
	}
	if err := s.store.CreateKafkaCaptureConfig(r.Context(), cfg); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to create kafka capture config"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusCreated, cfg)
}

func (s *Server) handleListKafkaCaptureConfigs(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}
	configs, err := s.store.ListKafkaCaptureConfigsByOrg(r.Context(), org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list kafka capture configs"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"kafka_capture_configs": configs,
		"count":                 len(configs),
	})
}

func (s *Server) handleGetKafkaCaptureConfig(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "configID")
	cfg, err := s.store.GetKafkaCaptureConfig(r.Context(), id)
	if err != nil || cfg.OrgID != org.ID {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("kafka capture config not found"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleUpdateKafkaCaptureConfig(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "configID")
	cfg, err := s.store.GetKafkaCaptureConfig(r.Context(), id)
	if err != nil || cfg.OrgID != org.ID {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("kafka capture config not found"))
		return
	}

	var req kafkaCaptureConfigRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	cfg.SourceHostID = req.SourceHostID
	cfg.SourceVM = req.SourceVM
	cfg.Name = req.Name
	cfg.BootstrapServers = store.StringSlice(req.BootstrapServers)
	cfg.Topics = store.StringSlice(req.Topics)
	cfg.Username = req.Username
	if req.Password != "" {
		cfg.Password = req.Password
	}
	if req.SASLMechanism != "" {
		cfg.SASLMechanism = req.SASLMechanism
	}
	cfg.TLSEnabled = req.TLSEnabled
	cfg.InsecureSkipVerify = req.InsecureSkipVerify
	if req.TLSCAPEM != "" {
		cfg.TLSCAPEM = req.TLSCAPEM
	}
	if req.Codec != "" {
		cfg.Codec = req.Codec
	}
	cfg.RedactionRules = store.StringSlice(req.RedactionRules)
	cfg.MaxBufferAgeSecs = req.MaxBufferAgeSecs
	cfg.MaxBufferBytes = req.MaxBufferBytes
	cfg.Enabled = req.Enabled
	cfg.UpdatedAt = time.Now().UTC()

	if len(cfg.BootstrapServers) == 0 {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("bootstrap_servers is required"))
		return
	}
	if len(cfg.Topics) == 0 {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("topics is required"))
		return
	}

	if err := s.store.UpdateKafkaCaptureConfig(r.Context(), cfg); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to update kafka capture config"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleDeleteKafkaCaptureConfig(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "configID")
	cfg, err := s.store.GetKafkaCaptureConfig(r.Context(), id)
	if err != nil || cfg.OrgID != org.ID {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("kafka capture config not found"))
		return
	}
	if err := s.store.DeleteKafkaCaptureConfig(r.Context(), id); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete kafka capture config"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (s *Server) handleListSandboxKafkaStubs(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}
	sandboxID := chi.URLParam(r, "sandboxID")
	stubs, err := s.orchestrator.ListSandboxKafkaStubs(r.Context(), org.ID, sandboxID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list sandbox kafka stubs"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"kafka_stubs": stubs,
		"count":       len(stubs),
	})
}

func (s *Server) handleGetSandboxKafkaStub(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}
	sandboxID := chi.URLParam(r, "sandboxID")
	stubID := chi.URLParam(r, "stubID")
	stub, err := s.orchestrator.GetSandboxKafkaStub(r.Context(), org.ID, sandboxID, stubID)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox kafka stub not found"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, stub)
}

func (s *Server) handleStartSandboxKafkaStub(w http.ResponseWriter, r *http.Request) {
	s.handleTransitionSandboxKafkaStub(w, r, "start")
}

func (s *Server) handleStopSandboxKafkaStub(w http.ResponseWriter, r *http.Request) {
	s.handleTransitionSandboxKafkaStub(w, r, "stop")
}

func (s *Server) handleRestartSandboxKafkaStub(w http.ResponseWriter, r *http.Request) {
	s.handleTransitionSandboxKafkaStub(w, r, "restart")
}

func (s *Server) handleTransitionSandboxKafkaStub(w http.ResponseWriter, r *http.Request, action string) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}
	sandboxID := chi.URLParam(r, "sandboxID")
	stubID := chi.URLParam(r, "stubID")

	var (
		stub *store.SandboxKafkaStub
		err  error
	)
	switch action {
	case "start":
		stub, err = s.orchestrator.StartSandboxKafkaStub(r.Context(), org.ID, sandboxID, stubID)
	case "stop":
		stub, err = s.orchestrator.StopSandboxKafkaStub(r.Context(), org.ID, sandboxID, stubID)
	case "restart":
		stub, err = s.orchestrator.RestartSandboxKafkaStub(r.Context(), org.ID, sandboxID, stubID)
	}
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("sandbox kafka stub not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to %s sandbox kafka stub", action))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, stub)
}
