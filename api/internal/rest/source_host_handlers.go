package rest

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

type discoverSourceHostsRequest struct {
	SSHConfigContent string `json:"ssh_config_content"`
}

type confirmSourceHostsRequest struct {
	Hosts []confirmSourceHost `json:"hosts"`
}

type confirmSourceHost struct {
	Name             string   `json:"name"`
	Hostname         string   `json:"hostname"`
	Type             string   `json:"type"` // "libvirt" or "proxmox"
	SSHUser          string   `json:"ssh_user"`
	SSHPort          int      `json:"ssh_port"`
	SSHIdentityFile  string   `json:"ssh_identity_file"`
	ProxmoxHost      string   `json:"proxmox_host,omitempty"`
	ProxmoxTokenID   string   `json:"proxmox_token_id,omitempty"`
	ProxmoxSecret    string   `json:"proxmox_secret,omitempty"`
	ProxmoxNode      string   `json:"proxmox_node,omitempty"`
	ProxmoxVerifySSL bool     `json:"proxmox_verify_ssl,omitempty"`
	VMs              []string `json:"vms,omitempty"`
}

func (s *Server) handleDiscoverSourceHosts(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	var req discoverSourceHostsRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if req.SSHConfigContent == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("ssh_config_content is required"))
		return
	}

	// Use the orchestrator to discover hosts via the daemon
	results, err := s.orchestrator.DiscoverSourceHosts(r.Context(), org.ID, req.SSHConfigContent)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("discovery failed"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"hosts": results,
		"count": len(results),
	})
}

func (s *Server) handleConfirmSourceHosts(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	var req confirmSourceHostsRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	if len(req.Hosts) == 0 {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("at least one host is required"))
		return
	}

	created := make([]*store.SourceHost, 0, len(req.Hosts))
	for _, h := range req.Hosts {
		hostType := h.Type
		if hostType == "" {
			hostType = "libvirt"
		}
		port := h.SSHPort
		if port == 0 {
			port = 22
		}

		sh := &store.SourceHost{
			ID:               uuid.New().String(),
			OrgID:            org.ID,
			Name:             h.Name,
			Hostname:         h.Hostname,
			Type:             hostType,
			SSHUser:          h.SSHUser,
			SSHPort:          port,
			SSHIdentityFile:  h.SSHIdentityFile,
			ProxmoxHost:      h.ProxmoxHost,
			ProxmoxTokenID:   h.ProxmoxTokenID,
			ProxmoxSecret:    h.ProxmoxSecret,
			ProxmoxNode:      h.ProxmoxNode,
			ProxmoxVerifySSL: h.ProxmoxVerifySSL,
			VMs:              h.VMs,
		}

		if err := s.store.CreateSourceHost(r.Context(), sh); err != nil {
			serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to save source host"))
			return
		}
		created = append(created, sh)
	}

	_ = serverJSON.RespondJSON(w, http.StatusCreated, map[string]any{
		"source_hosts": created,
		"count":        len(created),
	})
}

func (s *Server) handleListSourceHosts(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	hosts, err := s.store.ListSourceHostsByOrg(r.Context(), org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("list source hosts failed"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"source_hosts": hosts,
		"count":        len(hosts),
	})
}

func (s *Server) handleDeleteSourceHost(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "sourceHostID")
	if id == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("sourceHostID is required"))
		return
	}

	// Verify ownership
	host, err := s.store.GetSourceHost(r.Context(), id)
	if err != nil {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("source host not found"))
		return
	}
	if host.OrgID != org.ID {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("source host not found"))
		return
	}

	if err := s.store.DeleteSourceHost(r.Context(), id); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete source host"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"deleted": true,
	})
}
