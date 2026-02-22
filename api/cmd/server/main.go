package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aspectrr/fluid.sh/api/docs"
	"github.com/aspectrr/fluid.sh/api/internal/auth"
	"github.com/aspectrr/fluid.sh/api/internal/config"
	grpcServer "github.com/aspectrr/fluid.sh/api/internal/grpc"
	"github.com/aspectrr/fluid.sh/api/internal/orchestrator"
	"github.com/aspectrr/fluid.sh/api/internal/registry"
	"github.com/aspectrr/fluid.sh/api/internal/rest"
	"github.com/aspectrr/fluid.sh/api/internal/store"
	postgresStore "github.com/aspectrr/fluid.sh/api/internal/store/postgres"
	"github.com/aspectrr/fluid.sh/api/internal/telemetry"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// @title          Fluid API
// @version        1.0
// @description    API for managing sandboxes, organizations, billing, and hosts
// @host           api.fluid.sh
// @BasePath       /
// @securityDefinitions.apikey CookieAuth
// @in             cookie
// @name           session
func main() {
	// Load .env if present (no error if missing - production uses real env vars)
	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logger := setupLogger(cfg.Logging.Level, cfg.Logging.Format)
	slog.SetDefault(logger)

	redactedDB := cfg.Database.URL
	if u, err := url.Parse(cfg.Database.URL); err == nil && u.User != nil {
		u.User = url.UserPassword("***", "***")
		redactedDB = u.String()
	}

	logger.Info("starting fluid API",
		"rest_addr", cfg.API.Addr,
		"grpc_addr", cfg.GRPC.Address,
		"db", redactedDB,
	)

	// 1. Initialize shared Postgres store.
	st, err := postgresStore.New(ctx, store.Config{
		DatabaseURL:     cfg.Database.URL,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		AutoMigrate:     cfg.Database.AutoMigrate,
		EncryptionKey:   cfg.EncryptionKey,
	})
	if err != nil {
		logger.Error("failed to initialize store", "error", err)
		os.Exit(1)
	}
	defer func() {
		if cerr := st.Close(); cerr != nil {
			logger.Error("failed to close store", "error", cerr)
		}
	}()

	// 2. Initialize host registry (in-memory).
	reg := registry.New()

	// 3. Initialize gRPC server with host token auth.
	grpcOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(auth.HostTokenStreamInterceptor(st)),
	}
	if cfg.GRPC.TLSCertFile != "" && cfg.GRPC.TLSKeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(cfg.GRPC.TLSCertFile, cfg.GRPC.TLSKeyFile)
		if err != nil {
			logger.Error("failed to load gRPC TLS credentials", "error", err)
			os.Exit(1)
		}
		grpcOpts = append([]grpc.ServerOption{grpc.Creds(creds)}, grpcOpts...)
		logger.Info("gRPC TLS enabled")
	} else {
		logger.Warn("gRPC server running WITHOUT TLS - host bearer tokens will be sent in plaintext")
	}

	grpcSrv, err := grpcServer.NewServer(
		cfg.GRPC.Address,
		reg,
		st,
		logger,
		cfg.Orchestrator.HeartbeatTimeout,
		grpcOpts...,
	)
	if err != nil {
		logger.Error("failed to initialize gRPC server", "error", err)
		os.Exit(1)
	}

	// 4. Initialize orchestrator.
	orch := orchestrator.New(
		reg,
		st,
		grpcSrv.Handler(),
		logger,
		cfg.Orchestrator.DefaultTTL,
		cfg.Orchestrator.HeartbeatTimeout,
	)

	// 5. Agent client - commented out, not yet ready for integration.
	// var agentClient *agent.Client
	// if cfg.Agent.OpenRouterAPIKey != "" {
	// 	agentClient = agent.NewClient(cfg.Agent, st, orch, logger)
	// 	logger.Info("agent client initialized", "model", cfg.Agent.DefaultModel)
	// } else {
	// 	logger.Warn("OPENROUTER_API_KEY not set, agent chat disabled")
	// }

	// 6. Initialize telemetry.
	tel := telemetry.New(cfg.PostHog.APIKey, cfg.PostHog.Endpoint)
	defer tel.Close()

	// 7. Initialize REST server.
	srv := rest.NewServer(st, cfg, orch, tel, docs.OpenAPIYAML)

	httpSrv := &http.Server{
		Addr:              cfg.API.Addr,
		Handler:           srv.Router,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       cfg.API.ReadTimeout,
		WriteTimeout:      cfg.API.WriteTimeout,
		IdleTimeout:       cfg.API.IdleTimeout,
	}

	// 8. Start gRPC server in background.
	grpcErrCh := make(chan error, 1)
	go func() {
		logger.Info("gRPC server listening", "addr", cfg.GRPC.Address)
		if err := grpcSrv.Start(); err != nil {
			grpcErrCh <- err
		}
	}()

	// 9. Start REST server in background.
	httpErrCh := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", "addr", cfg.API.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErrCh <- err
		}
	}()

	// 10. Wait for signal or error.
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-grpcErrCh:
		logger.Error("gRPC server error", "error", err)
	case err := <-httpErrCh:
		logger.Error("HTTP server error", "error", err)
	}

	// 11. Graceful shutdown: stop HTTP first (drain in-flight requests),
	// then stop gRPC so streaming daemons stay connected during drain.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.API.ShutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server graceful shutdown failed", "error", err)
		_ = httpSrv.Close()
	} else {
		logger.Info("HTTP server shut down gracefully")
	}

	grpcSrv.Stop()
	logger.Info("gRPC server stopped")
}

func setupLogger(levelStr, format string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	return slog.New(handler)
}
