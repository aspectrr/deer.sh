package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-remote/internal/ansible"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/config"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/janitor"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/libvirt"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/rest"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/sshca"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/sshkeys"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/store"
	postgresStore "github.com/aspectrr/fluid.sh/fluid-remote/internal/store/postgres"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/telemetry"
	"github.com/aspectrr/fluid.sh/fluid-remote/internal/vm"
)

// @title fluid-remote API
// @version 0.1.0
// @description API for managing AI Agent VM sandboxes using libvirt
// @BasePath /

// @tag.name Sandbox
// @tag.description Sandbox lifecycle management - create, start, run commands, snapshot, and destroy sandboxes

// @tag.name VMs
// @tag.description Virtual machine listing and information

// @tag.name Ansible
// @tag.description Ansible playbook job management

// @tag.name Health
// @tag.description Health check endpoints
func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Context with OS signal cancellation
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load config from YAML (with env var overrides for backward compatibility)
	cfg, err := config.LoadWithEnvOverride(*configPath)
	if err != nil {
		slog.Error("failed to load config", "path", *configPath, "error", err)
		os.Exit(1)
	}

	// Logging setup
	logger := setupLogger(cfg.Logging.Level, cfg.Logging.Format)
	slog.SetDefault(logger)

	logger.Info("starting fluid-remote API",
		"config", *configPath,
		"addr", cfg.API.Addr,
		"db", cfg.Database.URL,
		"network", cfg.Libvirt.Network,
		"default_vcpus", cfg.VM.DefaultVCPUs,
		"default_memory_mb", cfg.VM.DefaultMemoryMB,
		"command_timeout", cfg.VM.CommandTimeout.String(),
		"ip_discovery_timeout", cfg.VM.IPDiscoveryTimeout.String(),
		"ansible_playbooks_dir", cfg.Ansible.PlaybooksDir,
	)

	st, err := postgresStore.New(ctx, store.Config{
		DatabaseURL:     cfg.Database.URL,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		AutoMigrate:     cfg.Database.AutoMigrate,
		ReadOnly:        false,
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

	// Initialize libvirt manager with config
	lvCfg := libvirt.Config{
		LibvirtURI:         cfg.Libvirt.URI,
		BaseImageDir:       cfg.Libvirt.BaseImageDir,
		WorkDir:            cfg.Libvirt.WorkDir,
		DefaultNetwork:     cfg.Libvirt.Network,
		SSHKeyInjectMethod: cfg.Libvirt.SSHKeyInjectMethod,
		SSHProxyJump:       cfg.SSH.ProxyJump,
		SocketVMNetWrapper: cfg.Libvirt.SocketVMNetWrapper,
		DefaultVCPUs:       cfg.VM.DefaultVCPUs,
		DefaultMemoryMB:    cfg.VM.DefaultMemoryMB,
	}
	// Read SSH CA public key if it exists
	if pubKeyData, err := os.ReadFile(cfg.SSH.CAPubPath); err == nil {
		lvCfg.SSHCAPubKey = string(pubKeyData)
	}
	lvMgr := libvirt.NewVirshManager(lvCfg, logger)

	// Initialize domain manager for direct libvirt queries
	domainMgr := libvirt.NewDomainManager(cfg.Libvirt.URI)

	// Initialize telemetry service.
	// Design decision: telemetry failures should not crash the application.
	// If telemetry initialization fails, we log the error and use a noop service
	// that silently discards all events. This ensures the core API functionality
	// remains available even when analytics infrastructure is unavailable.
	telemetrySvc, err := telemetry.NewService(cfg.Telemetry)
	if err != nil {
		logger.Warn("telemetry initialization failed, using noop service", "error", err)
		telemetrySvc = telemetry.NewNoopService()
	}
	defer telemetrySvc.Close()

	// Initialize SSH CA and key manager (optional - for managed credentials)
	var keyMgr sshkeys.KeyProvider
	if _, err := os.Stat(cfg.SSH.CAKeyPath); err == nil {
		// SSH CA key exists, initialize managed key support
		caCfg := sshca.Config{
			CAKeyPath:             cfg.SSH.CAKeyPath,
			CAPubKeyPath:          cfg.SSH.CAPubPath,
			WorkDir:               cfg.SSH.WorkDir,
			DefaultTTL:            cfg.SSH.CertTTL,
			MaxTTL:                cfg.SSH.MaxTTL,
			DefaultPrincipals:     []string{cfg.SSH.DefaultUser},
			EnforceKeyPermissions: true,
		}
		ca, err := sshca.NewCA(caCfg)
		if err != nil {
			logger.Error("failed to create SSH CA", "error", err)
			os.Exit(1)
		}
		if err := ca.Initialize(ctx); err != nil {
			logger.Error("failed to initialize SSH CA", "error", err)
			os.Exit(1)
		}

		keyMgrCfg := sshkeys.Config{
			KeyDir:          cfg.SSH.KeyDir,
			CertificateTTL:  cfg.SSH.CertTTL,
			RefreshMargin:   30 * time.Second,
			DefaultUsername: cfg.SSH.DefaultUser,
		}
		keyMgr, err = sshkeys.NewKeyManager(ca, keyMgrCfg, logger)
		if err != nil {
			logger.Error("failed to create SSH key manager", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := keyMgr.Close(); err != nil {
				logger.Error("failed to close SSH key manager", "error", err)
			}
		}()
		logger.Info("SSH key management enabled",
			"key_dir", cfg.SSH.KeyDir,
			"cert_ttl", cfg.SSH.CertTTL,
		)
	} else {
		logger.Info("SSH CA not found, managed credentials disabled",
			"ca_key_path", cfg.SSH.CAKeyPath,
		)
	}

	// Initialize VM service with logger and optional key manager
	vmOpts := []vm.Option{
		vm.WithLogger(logger),
		vm.WithVirshConfig(lvCfg), // Pass virsh config for remote manager creation
		vm.WithTelemetry(telemetrySvc),
	}
	if keyMgr != nil {
		vmOpts = append(vmOpts, vm.WithKeyManager(keyMgr))
	}
	vmSvc := vm.NewService(lvMgr, st, vm.Config{
		Network:            cfg.Libvirt.Network,
		DefaultVCPUs:       cfg.VM.DefaultVCPUs,
		DefaultMemoryMB:    cfg.VM.DefaultMemoryMB,
		CommandTimeout:     cfg.VM.CommandTimeout,
		IPDiscoveryTimeout: cfg.VM.IPDiscoveryTimeout,
		SSHProxyJump:       cfg.SSH.ProxyJump,
	}, vmOpts...)

	// Initialize Ansible runner
	ansibleRunner := ansible.NewRunner(cfg.Ansible.InventoryPath, cfg.Ansible.Image, cfg.Ansible.AllowedPlaybooks)

	// Initialize Ansible playbook service
	playbookSvc := ansible.NewPlaybookService(st, cfg.Ansible.PlaybooksDir)

	// Initialize multi-host manager if hosts are configured
	var multiHostMgr *libvirt.MultiHostDomainManager
	if len(cfg.Hosts) > 0 {
		multiHostMgr = libvirt.NewMultiHostDomainManager(cfg.Hosts, logger)
		logger.Info("multi-host VM listing enabled",
			"host_count", len(cfg.Hosts),
		)
	}

	// REST server setup with multi-host support

	restSrv := rest.NewServerWithMultiHost(vmSvc, domainMgr, multiHostMgr, ansibleRunner, playbookSvc, telemetrySvc)

	// Start janitor for background cleanup of expired sandboxes

	if cfg.Janitor.Enabled {

		janitorSvc := janitor.New(st, vmSvc, cfg.Janitor.DefaultTTL, logger)
		go janitorSvc.Start(ctx, cfg.Janitor.Interval)
		logger.Info("sandbox janitor enabled",
			"interval", cfg.Janitor.Interval,
			"default_ttl", cfg.Janitor.DefaultTTL,
		)
	} else {
		logger.Info("sandbox janitor disabled")
	}

	// Build http.Server so we can gracefully shutdown
	// WriteTimeout must be > IPDiscoveryTimeout to allow wait_for_ip to complete
	writeTimeout := cfg.VM.IPDiscoveryTimeout + 30*time.Second
	if writeTimeout < cfg.API.WriteTimeout {
		writeTimeout = cfg.API.WriteTimeout
	}
	httpSrv := &http.Server{
		Addr:              cfg.API.Addr,
		Handler:           restSrv.Router, // use the chi router directly for graceful shutdowns
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       cfg.API.ReadTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       cfg.API.IdleTimeout,
	}

	// Start HTTP server
	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", cfg.API.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
	}()

	// Wait for signal or server error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrCh:
		logger.Error("server error", "error", err)
	}

	// Attempt graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.API.ShutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server graceful shutdown failed", "error", err)
		_ = httpSrv.Close()
	} else {
		logger.Info("http server shut down gracefully")
	}
}

// setupLogger configures slog with level and format.
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
	jsonFmt := strings.ToLower(format) == "json"

	var handler slog.Handler
	if jsonFmt {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	return slog.New(handler)
}
