package rest

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
)

func generateSessionCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		for {
			_, _ = rand.Read(b[i : i+1])
			idx := int(b[i])
			if idx < 256-(256%len(chars)) {
				b[i] = chars[idx%len(chars)]
				break
			}
		}
	}
	return string(b)
}

type docsSession struct {
	StorageKey     string
	CompletedSteps map[int]bool
	CreatedAt      time.Time
}

type docsProgressStore struct {
	mu       sync.Mutex
	sessions map[string]*docsSession
}

var docsProgress = &docsProgressStore{
	sessions: make(map[string]*docsSession),
}

func (d *docsProgressStore) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()
	cutoff := time.Now().Add(-1 * time.Hour)
	for code, s := range d.sessions {
		if s.CreatedAt.Before(cutoff) {
			delete(d.sessions, code)
		}
	}
}

type docsRegisterRequest struct {
	StorageKey string `json:"storage_key"`
}

type docsRegisterResponse struct {
	SessionCode string `json:"session_code"`
}

func (s *Server) handleDocsProgressRegister(w http.ResponseWriter, r *http.Request) {
	var req docsRegisterRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}

	docsProgress.cleanup()

	code := generateSessionCode()

	docsProgress.mu.Lock()
	if len(docsProgress.sessions) >= 10000 {
		docsProgress.mu.Unlock()
		serverError.RespondError(w, http.StatusServiceUnavailable, fmt.Errorf("too many active sessions"))
		return
	}
	docsProgress.sessions[code] = &docsSession{
		StorageKey:     req.StorageKey,
		CompletedSteps: make(map[int]bool),
		CreatedAt:      time.Now(),
	}
	docsProgress.mu.Unlock()

	_ = serverJSON.RespondJSON(w, http.StatusOK, docsRegisterResponse{SessionCode: code})
}

type completeRequest struct {
	SessionCode string `json:"session_code"`
	StepIndex   int    `json:"step_index"`
}

func (s *Server) handleDocsProgressComplete(w http.ResponseWriter, r *http.Request) {
	var req completeRequest
	if err := serverJSON.DecodeJSON(r.Context(), r, &req); err != nil {
		serverError.RespondError(w, http.StatusBadRequest, err)
		return
	}
	if req.SessionCode == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("session code is required"))
		return
	}

	docsProgress.mu.Lock()
	session, ok := docsProgress.sessions[req.SessionCode]
	if ok {
		session.CompletedSteps[req.StepIndex] = true
	}
	docsProgress.mu.Unlock()

	if !ok {
		serverError.RespondError(w, http.StatusNotFound, fmt.Errorf("session not found"))
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type progressResponse struct {
	CompletedSteps []int `json:"completed_steps"`
}

func (s *Server) handleDocsProgressGet(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		serverError.RespondError(w, http.StatusBadRequest, fmt.Errorf("code is required"))
		return
	}

	docsProgress.mu.Lock()
	session, ok := docsProgress.sessions[code]
	var steps []int
	if ok {
		for idx := range session.CompletedSteps {
			steps = append(steps, idx)
		}
	}
	docsProgress.mu.Unlock()

	if !ok {
		_ = serverJSON.RespondJSON(w, http.StatusOK, progressResponse{CompletedSteps: []int{}})
		return
	}

	_ = serverJSON.RespondJSON(w, http.StatusOK, progressResponse{CompletedSteps: steps})
}
