package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ReadinessServer listens for cloud-init phone_home callbacks from sandboxes.
// When a sandbox's cloud-init finishes (sshd restarted, CA key installed),
// it POSTs to /ready/{sandbox_id}, which unblocks WaitReady.
type ReadinessServer struct {
	mu      sync.Mutex
	waiters map[string]chan struct{}
	logger  *slog.Logger
	server  *http.Server
}

// NewReadinessServer creates a readiness server listening on the given address.
// addr should be "bridgeIP:port" (e.g., "10.0.0.1:9092").
func NewReadinessServer(addr string, logger *slog.Logger) *ReadinessServer {
	rs := &ReadinessServer{
		waiters: make(map[string]chan struct{}),
		logger:  logger.With("component", "readiness"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ready/", rs.handleReady)

	rs.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return rs
}

// Start begins listening. Blocks until the server is shut down.
func (rs *ReadinessServer) Start() error {
	rs.logger.Info("readiness server starting", "addr", rs.server.Addr)
	ln, err := net.Listen("tcp", rs.server.Addr)
	if err != nil {
		return fmt.Errorf("readiness listen: %w", err)
	}
	return rs.server.Serve(ln)
}

// Shutdown gracefully stops the server.
func (rs *ReadinessServer) Shutdown(ctx context.Context) error {
	return rs.server.Shutdown(ctx)
}

// Register creates a waiter channel for a sandbox ID. Must be called before
// WaitReady so the channel exists when phone_home arrives.
func (rs *ReadinessServer) Register(sandboxID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.waiters[sandboxID] = make(chan struct{})
}

// Unregister removes a sandbox's waiter channel (cleanup).
func (rs *ReadinessServer) Unregister(sandboxID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.waiters, sandboxID)
}

// WaitReady blocks until the sandbox's phone_home POST arrives or timeout expires.
func (rs *ReadinessServer) WaitReady(sandboxID string, timeout time.Duration) error {
	rs.mu.Lock()
	ch, ok := rs.waiters[sandboxID]
	rs.mu.Unlock()

	if !ok {
		return fmt.Errorf("sandbox %s not registered for readiness", sandboxID)
	}

	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("readiness timeout for sandbox %s after %v", sandboxID, timeout)
	}
}

// handleReady handles POST /ready/{sandbox_id} from cloud-init phone_home.
func (rs *ReadinessServer) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract sandbox ID from path: /ready/{sandbox_id}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ready/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing sandbox_id", http.StatusBadRequest)
		return
	}
	sandboxID := parts[0]

	rs.logger.Info("phone_home received", "sandbox_id", sandboxID)

	rs.mu.Lock()
	ch, ok := rs.waiters[sandboxID]
	rs.mu.Unlock()

	if ok {
		select {
		case <-ch:
			// Already signaled
		default:
			close(ch)
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok\n")
}
