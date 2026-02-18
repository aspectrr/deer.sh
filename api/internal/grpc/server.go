// Package grpc provides the gRPC server that accepts bidirectional streams
// from sandbox hosts.
package grpc

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/store"

	"google.golang.org/grpc"
)

// Server wraps a gRPC server that sandbox hosts connect to.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	handler    *StreamHandler
	registry   *registry.Registry
	store      store.Store
	logger     *slog.Logger
}

// NewServer creates a gRPC server listening on addr and registers the
// HostService stream handler.
func NewServer(
	addr string,
	reg *registry.Registry,
	st store.Store,
	logger *slog.Logger,
	heartbeatTimeout time.Duration,
	opts ...grpc.ServerOption,
) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}

	gs := grpc.NewServer(opts...)

	handler := NewStreamHandler(reg, st, logger, heartbeatTimeout)
	fluidv1.RegisterHostServiceServer(gs, handler)

	s := &Server{
		listener:   lis,
		grpcServer: gs,
		handler:    handler,
		registry:   reg,
		store:      st,
		logger:     logger.With("component", "grpc"),
	}

	return s, nil
}

// Handler returns the stream handler, allowing the orchestrator to call
// SendAndWait for dispatching commands to connected hosts.
func (s *Server) Handler() *StreamHandler {
	return s.handler
}

// Start begins accepting connections. Blocks until stopped.
func (s *Server) Start() error {
	s.logger.Info("gRPC server starting", "addr", s.listener.Addr().String())
	return s.grpcServer.Serve(s.listener)
}

// Stop performs a graceful shutdown of the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("gRPC server stopping")
	s.grpcServer.GracefulStop()
}
