package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/agent"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/audit"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/config"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/daemon"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/id"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/image"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/janitor"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/microvm"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/network"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/provider"
	lxcProvider "github.com/aspectrr/deer.sh/deer-daemon/internal/provider/lxc"
	microvmProvider "github.com/aspectrr/deer.sh/deer-daemon/internal/provider/microvm"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/redact"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/snapshotpull"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sourcevm"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sshca"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/sshkeys"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/state"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/telemetry"
)

const version = "0.1.0"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Load config
	cfgPath := *configPath
	if cfgPath == "" {
		home, _ := os.UserHomeDir()
		cfgPath = filepath.Join(home, ".deer", "daemon.yaml")
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	// Ensure host ID
	if cfg.HostID == "" {
		hostID, err := id.GenerateRaw()
		if err != nil {
			return fmt.Errorf("generate host ID: %w", err)
		}
		cfg.HostID = hostID
		_ = config.Save(cfgPath, cfg)
		logger.Info("generated host ID", "host_id", cfg.HostID)
	}

	logger.Info("deer-daemon starting",
		"host_id", cfg.HostID,
		"config", cfgPath,
		"provider", cfg.Provider,
	)

	// Initialize telemetry
	tele, err := telemetry.NewService(telemetry.Config{
		EnableAnonymousUsage: cfg.Telemetry.EnableAnonymousUsage,
	}, cfg.HostID)
	if err != nil {
		tele = telemetry.NewNoopService()
	}
	defer tele.Close()
	tele.Track("daemon_session_start", map[string]any{"provider": cfg.Provider})

	// Initialize SQLite state store
	st, err := state.NewStore(cfg.State.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	logger.Info("state store initialized", "db_path", cfg.State.DBPath)

	// Initialize provider based on config
	var prov provider.SandboxProvider
	var keyMgr sshkeys.KeyProvider

	var caPubKey string

	switch cfg.Provider {
	case "lxc":
		prov, err = initLXCProvider(cfg, logger)
		if err != nil {
			return err
		}
		logger.Info("LXC provider initialized",
			"host", cfg.LXC.Host,
			"node", cfg.LXC.Node,
		)
	default: // "microvm" or empty (default)
		prov, keyMgr, caPubKey, err = initMicroVMProvider(ctx, cfg, logger)
		if err != nil {
			return err
		}
		for _, w := range microvm.ValidateAccel(runtime.GOOS, cfg.MicroVM.Accel) {
			logger.Warn("accelerator performance warning", "warning", w, "accel", cfg.MicroVM.Accel)
		}
	}

	// Recover state from any running sandboxes
	if err := prov.RecoverState(ctx); err != nil {
		logger.Warn("state recovery failed", "error", err)
	}

	// Initialize janitor
	destroyFn := func(ctx context.Context, sandboxID string) error {
		if err := prov.DestroySandbox(ctx, sandboxID); err != nil {
			return err
		}
		return st.DeleteSandbox(ctx, sandboxID)
	}

	jan := janitor.New(st, destroyFn, cfg.Janitor.DefaultTTL, logger)
	go jan.Start(ctx, cfg.Janitor.Interval)

	// Initialize snapshot puller
	imgStore, err := image.NewStore(cfg.Image.BaseDir, logger)
	if err != nil {
		return fmt.Errorf("init image store for puller: %w", err)
	}
	puller := snapshotpull.NewPuller(imgStore, st.DB(), logger)

	// Initialize redactor
	redactor := redact.New()

	// Initialize audit logger
	var auditLog *audit.Logger
	if cfg.Audit.Enabled {
		if err := os.MkdirAll(filepath.Dir(cfg.Audit.LogPath), 0o755); err != nil {
			logger.Warn("failed to create audit log directory", "error", err)
		} else {
			al, err := audit.NewLogger(cfg.Audit.LogPath, cfg.Audit.MaxSizeMB)
			if err != nil {
				logger.Warn("failed to initialize audit logger", "error", err)
			} else {
				auditLog = al
				auditLog.LogSessionStart()
				defer auditLog.LogSessionEnd()
				defer func() { _ = auditLog.Close() }()
				logger.Info("audit logger initialized", "path", cfg.Audit.LogPath)
			}
		}
	}

	// Read daemon identity pub key for sharing with CLI
	var identityPubKey string
	identityGenerated, identityErr := sshca.EnsureSSHCA(cfg.SSH.IdentityFile, cfg.SSH.IdentityFile+".pub", "deer-daemon-identity")
	if identityErr != nil {
		logger.Warn("SSH identity key generation failed", "error", identityErr)
	} else if identityGenerated {
		logger.Info("SSH identity key generated", "path", cfg.SSH.IdentityFile)
	}
	if pubKeyData, err := os.ReadFile(cfg.SSH.IdentityFile + ".pub"); err == nil {
		identityPubKey = strings.TrimSpace(string(pubKeyData))
		logger.Info("loaded SSH identity pub key", "path", cfg.SSH.IdentityFile+".pub")
	} else {
		logger.Warn("SSH identity pub key not found, daemon key deployment will be unavailable", "path", cfg.SSH.IdentityFile+".pub", "error", err)
	}

	// Start DaemonService gRPC server (inbound from CLI)
	if cfg.Daemon.Enabled {
		daemonSrv := daemon.NewServer(cfg, prov, st, puller, keyMgr, tele, redactor, auditLog, cfg.HostID, version, cfg.SSH.IdentityFile, caPubKey, identityPubKey, logger)
		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("panic recovered in gRPC handler", "method", info.FullMethod, "panic", r)
						err = status.Errorf(codes.Internal, "internal error: %v", r)
					}
				}()
				return handler(ctx, req)
			}),
			grpc.StreamInterceptor(func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("panic recovered in gRPC stream handler", "method", info.FullMethod, "panic", r)
						err = status.Errorf(codes.Internal, "internal error: %v", r)
					}
				}()
				return handler(srv, ss)
			}),
		)
		deerv1.RegisterDaemonServiceServer(grpcServer, daemonSrv)

		lis, err := net.Listen("tcp", cfg.Daemon.ListenAddr)
		if err != nil {
			return fmt.Errorf("listen %s: %w", cfg.Daemon.ListenAddr, err)
		}
		logger.Info("daemon gRPC server listening", "addr", cfg.Daemon.ListenAddr)

		go func() {
			if err := grpcServer.Serve(lis); err != nil {
				logger.Error("daemon gRPC server error", "error", err)
			}
		}()
		go func() {
			<-ctx.Done()
			grpcServer.GracefulStop()
		}()
	}

	logger.Info("sandbox-host ready",
		"host_id", cfg.HostID,
		"control_plane", cfg.ControlPlane.Address,
		"provider", cfg.Provider,
	)

	if cfg.ControlPlane.Address != "" {
		// Initialize gRPC agent client
		agentClient := agent.NewClient(
			agent.Config{
				HostID:          cfg.HostID,
				Version:         version,
				Address:         cfg.ControlPlane.Address,
				Token:           cfg.ControlPlane.Token,
				Insecure:        cfg.ControlPlane.Insecure,
				CertFile:        cfg.ControlPlane.CertFile,
				KeyFile:         cfg.ControlPlane.KeyFile,
				CAFile:          cfg.ControlPlane.CAFile,
				SSHIdentityFile: cfg.SSH.IdentityFile,
			},
			prov,
			st,
			puller,
			logger,
		)

		// Start gRPC agent in background (reconnects automatically)
		agentErrCh := make(chan error, 1)
		go func() {
			agentErrCh <- agentClient.Run(ctx)
		}()

		// Wait for shutdown signal or agent fatal error
		select {
		case <-ctx.Done():
			logger.Info("sandbox-host shutting down")
		case err := <-agentErrCh:
			if err != nil && ctx.Err() == nil {
				logger.Error("agent error", "error", err)
				return err
			}
		}
	} else {
		logger.Info("control plane not configured, skipping agent client")
		<-ctx.Done()
		logger.Info("sandbox-host shutting down")
	}

	return nil
}

func initMicroVMProvider(ctx context.Context, cfg *config.Config, logger *slog.Logger) (provider.SandboxProvider, sshkeys.KeyProvider, string, error) {
	// Initialize microVM manager
	vmMgr, err := microvm.NewManager(cfg.MicroVM.QEMUBinary, cfg.MicroVM.WorkDir, logger)
	if err != nil {
		logger.Warn("microVM manager initialization failed", "error", err)
		vmMgr = nil
	} else {
		logger.Info("microVM manager initialized", "work_dir", cfg.MicroVM.WorkDir)
	}

	// Initialize network manager
	netMgr := network.NewNetworkManager(
		cfg.Network.DefaultBridge,
		cfg.Network.BridgeMap,
		cfg.Network.DHCPMode,
		logger,
	)
	logger.Info("network manager initialized",
		"default_bridge", cfg.Network.DefaultBridge,
		"dhcp_mode", cfg.Network.DHCPMode,
	)

	// Initialize image store
	imgStore, err := image.NewStore(cfg.Image.BaseDir, logger)
	if err != nil {
		return nil, nil, "", err
	}
	images, _ := imgStore.ListNames()
	logger.Info("image store initialized",
		"base_dir", cfg.Image.BaseDir,
		"images", len(images),
	)

	// Initialize SSH CA and key manager (graceful fallback to nil)
	var keyMgr sshkeys.KeyProvider
	var srcVMMgr *sourcevm.Manager
	var caPubKey string

	ca, caErr := sshca.NewCA(sshca.Config{
		CAKeyPath:             cfg.SSH.CAKeyPath,
		CAPubKeyPath:          cfg.SSH.CAPubKeyPath,
		WorkDir:               cfg.SSH.KeyDir,
		DefaultTTL:            cfg.SSH.CertTTL,
		MaxTTL:                60 * time.Minute,
		DefaultPrincipals:     []string{cfg.SSH.DefaultUser},
		EnforceKeyPermissions: false,
	})
	if caErr != nil {
		logger.Warn("SSH CA initialization failed", "error", caErr)
	} else {
		generated, ensureErr := sshca.EnsureSSHCA(cfg.SSH.CAKeyPath, cfg.SSH.CAPubKeyPath, "deer-daemon-ca")
		if ensureErr != nil {
			logger.Warn("SSH CA key generation failed", "error", ensureErr)
		} else if generated {
			logger.Info("SSH CA key generated", "path", cfg.SSH.CAKeyPath)
		}
		if initErr := ca.Initialize(ctx); initErr != nil {
			logger.Warn("SSH CA key loading failed - source VM operations will use ad-hoc connections only", "error", initErr)
		} else {
			// Extract CA public key for sharing via gRPC
			if pubKey, err := ca.GetPublicKey(); err != nil {
				logger.Warn("failed to get CA public key", "error", err)
			} else {
				caPubKey = pubKey
			}

			km, kmErr := sshkeys.NewKeyManager(ca, sshkeys.Config{
				KeyDir:          cfg.SSH.KeyDir,
				CertificateTTL:  cfg.SSH.CertTTL,
				DefaultUsername: cfg.SSH.DefaultUser,
			}, logger)
			if kmErr != nil {
				logger.Warn("SSH key manager initialization failed", "error", kmErr)
			} else {
				keyMgr = km
				logger.Info("SSH CA and key manager initialized")

				// Initialize source VM manager
				srcVMMgr = sourcevm.NewManager(
					cfg.Libvirt.URI,
					cfg.Libvirt.Network,
					km,
					cfg.SSH.DefaultUser,
					cfg.SSH.ProxyJump,
					cfg.SSH.IdentityFile,
					caPubKey,
					logger,
				)
				logger.Info("source VM manager initialized",
					"libvirt_uri", cfg.Libvirt.URI,
					"network", cfg.Libvirt.Network,
				)
			}
		}
	}

	// Discover bridge IP for cloud-init phone_home readiness signaling
	bridgeIP, _ := network.GetBridgeIP(cfg.Network.DefaultBridge)
	if bridgeIP != "" {
		logger.Info("bridge IP discovered for phone_home", "bridge", cfg.Network.DefaultBridge, "ip", bridgeIP)
	}

	// Start readiness HTTP server for cloud-init phone_home callbacks
	var readiness *daemon.ReadinessServer
	if bridgeIP != "" {
		readinessAddr := bridgeIP + ":9092"
		readiness = daemon.NewReadinessServer(readinessAddr, logger)
		go func() {
			if err := readiness.Start(); err != nil && err != http.ErrServerClosed {
				logger.Warn("readiness server error", "error", err)
			}
		}()
		logger.Info("readiness server started", "addr", readinessAddr)
	}

	redpandaCacheURL := ""
	if cfg.MicroVM.RedpandaCachePath != "" {
		redpandaCacheURL = "file://" + cfg.MicroVM.RedpandaCachePath
		logger.Info("Redpanda cache configured", "path", cfg.MicroVM.RedpandaCachePath)
	}
	disableCloudInit := cfg.MicroVM.DisableCloudInit
	if disableCloudInit {
		logger.Info("cloud-init disabled (pre-baked images)")
	}

	// Build the microVM provider. When readiness is nil (no bridge IP),
	// pass nil directly to avoid the nil-typed-pointer-in-interface trap
	// where a nil *ReadinessServer stored in a ReadinessWaiter interface
	// is non-nil, causing a panic on method calls.
	if readiness != nil {
		return microvmProvider.New(vmMgr, netMgr, imgStore, srcVMMgr, keyMgr, cfg.MicroVM.KernelPath, cfg.MicroVM.InitrdPath, cfg.MicroVM.RootDevice, cfg.MicroVM.Accel, cfg.MicroVM.IPDiscoveryTimeout, cfg.MicroVM.ReadinessTimeout, caPubKey, bridgeIP, readiness, redpandaCacheURL, disableCloudInit, cfg.MicroVM.SocketVMNetClient, cfg.MicroVM.SocketVMNetPath, logger), keyMgr, caPubKey, nil
	}
	return microvmProvider.New(vmMgr, netMgr, imgStore, srcVMMgr, keyMgr, cfg.MicroVM.KernelPath, cfg.MicroVM.InitrdPath, cfg.MicroVM.RootDevice, cfg.MicroVM.Accel, cfg.MicroVM.IPDiscoveryTimeout, cfg.MicroVM.ReadinessTimeout, caPubKey, bridgeIP, nil, redpandaCacheURL, disableCloudInit, cfg.MicroVM.SocketVMNetClient, cfg.MicroVM.SocketVMNetPath, logger), keyMgr, caPubKey, nil
}

func initLXCProvider(cfg *config.Config, logger *slog.Logger) (provider.SandboxProvider, error) {
	lxcCfg := lxcProvider.Config{
		Host:      cfg.LXC.Host,
		TokenID:   cfg.LXC.TokenID,
		Secret:    cfg.LXC.Secret,
		Node:      cfg.LXC.Node,
		Storage:   cfg.LXC.Storage,
		Bridge:    cfg.LXC.Bridge,
		VMIDStart: cfg.LXC.VMIDStart,
		VMIDEnd:   cfg.LXC.VMIDEnd,
		VerifySSL: cfg.LXC.VerifySSL,
		Timeout:   cfg.LXC.Timeout,
	}

	return lxcProvider.New(lxcCfg, logger)
}
