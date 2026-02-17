package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/aspectrr/fluid.sh/demo-server/internal/session"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 4096
)

// Config holds WebSocket handler configuration.
type Config struct {
	FluidBin       string
	AllowedOrigins []string
	SessionTimeout time.Duration
	MaxSessions    int
	LLMAPIKey      string
	LLMModel       string
	Logger         *slog.Logger
}

// clientMessage is a message from the browser.
type clientMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Handler manages WebSocket connections and sessions.
type Handler struct {
	config   Config
	upgrader websocket.Upgrader
	sessions map[string]*session.DemoSession
	mu       sync.Mutex
	logger   *slog.Logger
}

// NewHandler creates a new WebSocket handler.
func NewHandler(cfg Config) *Handler {
	h := &Handler{
		config:   cfg,
		sessions: make(map[string]*session.DemoSession),
		logger:   cfg.Logger,
	}

	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			for _, o := range cfg.AllowedOrigins {
				if o == "*" || o == origin {
					return true
				}
			}
			return false
		},
	}

	// Start session reaper
	go h.reapExpiredSessions()

	return h
}

// HandleWebSocket upgrades the HTTP connection and manages the session.
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check session limit
	h.mu.Lock()
	if len(h.sessions) >= h.config.MaxSessions {
		h.mu.Unlock()
		http.Error(w, `{"error":"too many sessions"}`, http.StatusServiceUnavailable)
		return
	}
	h.mu.Unlock()

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	// Create session
	sess, err := session.New(session.Config{
		FluidBin:       h.config.FluidBin,
		LLMAPIKey:      h.config.LLMAPIKey,
		LLMModel:       h.config.LLMModel,
		SessionTimeout: h.config.SessionTimeout,
		Logger:         h.logger,
	})
	if err != nil {
		h.logger.Error("session creation failed", "error", err)
		conn.WriteJSON(map[string]string{"type": "error", "message": "failed to start session: " + err.Error()})
		conn.Close()
		return
	}

	h.mu.Lock()
	h.sessions[sess.ID] = sess
	h.mu.Unlock()

	h.logger.Info("new session", "session_id", sess.ID, "remote", r.RemoteAddr)

	// Send session info
	conn.WriteJSON(session.Event{
		Type:         "session_info",
		SessionID:    sess.ID,
		ExpiresInSec: sess.ExpiresInSec(),
	})

	// Start read and write goroutines
	done := make(chan struct{})
	go h.writePump(conn, sess, done)
	h.readPump(conn, sess, done)

	// Cleanup
	h.mu.Lock()
	delete(h.sessions, sess.ID)
	h.mu.Unlock()

	sess.Close()
	h.logger.Info("session ended", "session_id", sess.ID)
}

// readPump reads messages from the WebSocket and dispatches to the session.
func (h *Handler) readPump(conn *websocket.Conn, sess *session.DemoSession, done chan struct{}) {
	defer func() {
		close(done)
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				h.logger.Error("websocket read error", "error", err)
			}
			return
		}

		var msg clientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			h.logger.Error("invalid message", "error", err)
			continue
		}

		if msg.Type == "user_input" && msg.Content != "" {
			// Process in a goroutine so we don't block reads
			go sess.HandleMessage(msg.Content)
		}
	}
}

// writePump writes events from the session to the WebSocket.
func (h *Handler) writePump(conn *websocket.Conn, sess *session.DemoSession, done chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	events := sess.Events()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteJSON(event); err != nil {
				h.logger.Error("websocket write error", "error", err)
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			// Check session expiry
			if sess.IsExpired() {
				conn.WriteJSON(session.Event{
					Type:    "error",
					Message: "session expired due to inactivity",
				})
				return
			}

		case <-done:
			return
		}
	}
}

// Shutdown closes all active sessions.
func (h *Handler) Shutdown() {
	h.mu.Lock()
	sessions := make([]*session.DemoSession, 0, len(h.sessions))
	for _, s := range h.sessions {
		sessions = append(sessions, s)
	}
	h.mu.Unlock()

	for _, s := range sessions {
		s.Close()
	}
}

// reapExpiredSessions periodically checks for and closes expired sessions.
func (h *Handler) reapExpiredSessions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		expired := make([]string, 0)
		for id, s := range h.sessions {
			if s.IsExpired() {
				expired = append(expired, id)
			}
		}
		for _, id := range expired {
			if s, ok := h.sessions[id]; ok {
				h.logger.Info("reaping expired session", "session_id", id)
				go s.Close()
				delete(h.sessions, id)
			}
		}
		h.mu.Unlock()
	}
}
