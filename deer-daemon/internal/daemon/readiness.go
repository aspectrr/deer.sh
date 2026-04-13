package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"
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
	ready   map[string]bool
	readyIP map[string]string
	refs    map[string]int
	logger  *slog.Logger
	server  *http.Server
}

// NewReadinessServer creates a readiness server listening on the given address.
// addr should be "bridgeIP:port" (e.g., "10.0.0.1:9092").
func NewReadinessServer(addr string, logger *slog.Logger) *ReadinessServer {
	rs := &ReadinessServer{
		waiters: make(map[string]chan struct{}),
		ready:   make(map[string]bool),
		readyIP: make(map[string]string),
		refs:    make(map[string]int),
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
	return rs.Serve(ln)
}

// Serve handles readiness callbacks on an existing listener.
func (rs *ReadinessServer) Serve(ln net.Listener) error {
	rs.logger.Info("readiness server serving", "addr", ln.Addr().String())
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
	if _, ok := rs.waiters[sandboxID]; !ok {
		rs.waiters[sandboxID] = make(chan struct{})
		rs.ready[sandboxID] = false
		rs.readyIP[sandboxID] = ""
	}
	rs.refs[sandboxID]++
}

// Unregister removes a sandbox's waiter channel (cleanup).
func (rs *ReadinessServer) Unregister(sandboxID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if refs := rs.refs[sandboxID]; refs > 1 {
		rs.refs[sandboxID] = refs - 1
		return
	}
	delete(rs.refs, sandboxID)
	delete(rs.waiters, sandboxID)
	delete(rs.ready, sandboxID)
	delete(rs.readyIP, sandboxID)
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
	if len(sandboxID) > 128 || !isValidSandboxID(sandboxID) {
		http.Error(w, "invalid sandbox_id", http.StatusBadRequest)
		return
	}

	rs.logger.Info("phone_home received", "sandbox_id", sandboxID)

	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		remoteIP = host
	}

	rs.mu.Lock()
	ch, ok := rs.waiters[sandboxID]
	if ok {
		rs.ready[sandboxID] = true
		rs.readyIP[sandboxID] = remoteIP
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
	rs.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "ok\n")
}

// WasReady reports whether the sandbox posted readiness.
func (rs *ReadinessServer) WasReady(sandboxID string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.ready[sandboxID]
}

func (rs *ReadinessServer) ReadyIP(sandboxID string) string {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.readyIP[sandboxID]
}

var sandboxIDRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func isValidSandboxID(id string) bool {
	return sandboxIDRe.MatchString(id)
}
