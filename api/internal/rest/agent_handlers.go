package rest

// Agent handlers - commented out, not yet ready for integration.

/*
import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aspectrr/fluid.sh/api/internal/agent"
	"github.com/aspectrr/fluid.sh/api/internal/auth"
	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

// handleAgentChat streams an SSE response for the agent chat.
func (s *Server) handleAgentChat(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}
	user := auth.UserFromContext(r.Context())

	var req agent.ChatRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}
	if req.Message == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("message is required"))
		return
	}

	if s.agentClient == nil {
		serverError.RespondError(w, http.StatusServiceUnavailable, fmt.Errorf("agent not configured: OPENROUTER_API_KEY not set"))
		return
	}

	s.telemetry.Track(user.ID, "agent_chat_sent", map[string]any{"org_id": org.ID, "model": req.Model})

	s.agentClient.StreamChat(r.Context(), w, org.ID, user.ID, req)
}

// handleListConversations returns all conversations for the org.
func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	conversations, err := s.store.ListAgentConversationsByOrg(r.Context(), org.ID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list conversations"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"conversations": conversations,
		"count":         len(conversations),
	})
}

// handleGetConversation returns a specific conversation.
func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	convID := chi.URLParam(r, "conversationID")
	conv, err := s.store.GetAgentConversation(r.Context(), convID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("conversation not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get conversation"))
		return
	}
	if conv.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("conversation does not belong to this organization"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, conv)
}

// handleListMessages returns messages for a conversation.
func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	convID := chi.URLParam(r, "conversationID")
	conv, err := s.store.GetAgentConversation(r.Context(), convID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("conversation not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get conversation"))
		return
	}
	if conv.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("conversation does not belong to this organization"))
		return
	}

	messages, err := s.store.ListAgentMessages(r.Context(), convID)
	if err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to list messages"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"messages": messages,
		"count":    len(messages),
	})
}

// handleDeleteConversation deletes a conversation and its messages.
func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	org, _, ok := s.resolveOrgMembership(w, r)
	if !ok {
		return
	}

	convID := chi.URLParam(r, "conversationID")
	conv, err := s.store.GetAgentConversation(r.Context(), convID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("conversation not found"))
			return
		}
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to get conversation"))
		return
	}
	if conv.OrgID != org.ID {
		serverError.RespondError(w, http.StatusForbidden, fmt.Errorf("conversation does not belong to this organization"))
		return
	}

	if err := s.store.DeleteAgentConversation(r.Context(), convID); err != nil {
		serverError.RespondError(w, http.StatusInternalServerError, fmt.Errorf("failed to delete conversation"))
		return
	}
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"deleted":         true,
		"conversation_id": convID,
	})
}

// handleListModels returns available LLM models with pricing.
func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]any{
		"models": agent.AvailableModels(),
	})
}
*/
