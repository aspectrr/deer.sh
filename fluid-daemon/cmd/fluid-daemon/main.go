package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/agent"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/config"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/daemon"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/id"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/image"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/janitor"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/microvm"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/network"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider"
	lxcProvider "github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider/lxc"
	microvmProvider "github.com/aspectrr/fluid.sh/fluid-daemon/internal/provider/microvm"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/snapshotpull"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/state"
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
		cfgPath = filepath.Join(home, ".fluid", "daemon.yaml")
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

	logger.Info("fluid-daemon starting",
		"host_id", cfg.HostID,
		"config", cfgPath,
		"provider", cfg.Provider,
	)

	// Initialize SQLite state store
	st, err := state.NewStore(cfg.State.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	logger.Info("state store initialized", "db_path", cfg.State.DBPath)

	// Initialize provider based on config
	var prov provider.SandboxProvider

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
		prov, err = initMicroVMProvider(ctx, cfg, logger)
		if err != nil {
			return err
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

	// Start DaemonService gRPC server (inbound from CLI)
	if cfg.Daemon.Enabled {
		daemonSrv := daemon.NewServer(prov, st, puller, cfg.HostID, version, logger)
		grpcServer := grpc.NewServer()
		fluidv1.RegisterDaemonServiceServer(grpcServer, daemonSrv)

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

	// Initialize gRPC agent client
	agentClient := agent.NewClient(
		agent.Config{
			HostID:   cfg.HostID,
			Version:  version,
			Address:  cfg.ControlPlane.Address,
			Insecure: cfg.ControlPlane.Insecure,
			CertFile: cfg.ControlPlane.CertFile,
			KeyFile:  cfg.ControlPlane.KeyFile,
			CAFile:   cfg.ControlPlane.CAFile,
		},
		prov,
		st,
		puller,
		logger,
	)

	logger.Info("sandbox-host ready",
		"host_id", cfg.HostID,
		"control_plane", cfg.ControlPlane.Address,
		"provider", cfg.Provider,
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

	return nil
}

func initMicroVMProvider(ctx context.Context, cfg *config.Config, logger *slog.Logger) (provider.SandboxProvider, error) {
	// Initialize microVM manager
	vmMgr, err := microvm.NewManager(cfg.MicroVM.QEMUBinary, cfg.MicroVM.WorkDir, logger)
	if err != nil {
		logger.Warn("microVM manager initialization failed (qemu not available)", "error", err)
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
		return nil, err
	}
	images, _ := imgStore.ListNames()
	logger.Info("image store initialized",
		"base_dir", cfg.Image.BaseDir,
		"images", len(images),
	)

	return microvmProvider.New(vmMgr, netMgr, imgStore, nil, logger), nil
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
