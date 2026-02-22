package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/doctor"
	"github.com/aspectrr/fluid.sh/fluid/internal/hostexec"
	fluidmcp "github.com/aspectrr/fluid.sh/fluid/internal/mcp"
	"github.com/aspectrr/fluid.sh/fluid/internal/paths"
	"github.com/aspectrr/fluid.sh/fluid/internal/sandbox"
	"github.com/aspectrr/fluid.sh/fluid/internal/store"
	"github.com/aspectrr/fluid.sh/fluid/internal/store/sqlite"
	"github.com/aspectrr/fluid.sh/fluid/internal/telemetry"
	"github.com/aspectrr/fluid.sh/fluid/internal/tui"
	"github.com/aspectrr/fluid.sh/fluid/internal/updater"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	cfgFile string
	cfg     *config.Config
)

func main() {
	// Set TUI version from ldflags
	tui.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "fluid",
	Short: "Fluid - Make Infrastructure Safe for AI",
	Long:  "Fluid is a terminal agent that AI manage infrastructure via sandboxed resources, audit trails and human approval.",
	// Default to TUI when no subcommand is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetBool("version"); v {
			short := commit
			if len(short) > 7 {
				short = short[:7]
			}
			fmt.Printf("fluid %s (%s, %s)\n", version, short, date)
			return nil
		}
		return runTUI()
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server on stdio",
	Long:  "Start an MCP (Model Context Protocol) server that exposes fluid tools over stdio for use with Claude Code, Cursor, and other MCP clients.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCP()
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check daemon setup on a host",
	Long:  "Validate that the fluid-daemon is properly installed and configured on a sandbox host.",
	RunE: func(cmd *cobra.Command, args []string) error {
		hostName, _ := cmd.Flags().GetString("host")

		configPath := cfgFile
		if configPath == "" {
			var err error
			configPath, err = paths.ConfigFile()
			if err != nil {
				return fmt.Errorf("determine config path: %w", err)
			}
		}

		loadedCfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		ctx := context.Background()
		var run hostexec.RunFunc

		if hostName == "" || hostName == "localhost" {
			run = hostexec.NewLocal()
		} else {
			// Find host in config
			var found bool
			for _, h := range loadedCfg.Hosts {
				if h.Name == hostName {
					user := h.SSHUser
					if user == "" {
						user = "root"
					}
					port := h.SSHPort
					if port == 0 {
						port = 22
					}
					run = hostexec.NewSSH(h.Address, user, port)
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("host %q not found in config", hostName)
			}
		}

		useColor := os.Getenv("NO_COLOR") == ""
		fmt.Println()
		fmt.Println("  Checking daemon health...")
		fmt.Println()

		results := doctor.RunAll(ctx, run)
		allPassed := doctor.PrintResults(results, os.Stdout, useColor)
		fmt.Println()

		if !allPassed {
			os.Exit(1)
		}
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"upgrade"},
	Short:   "Update fluid to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		latest, url, needsUpdate, err := updater.CheckLatest(version)
		if err != nil {
			return fmt.Errorf("check for updates: %w", err)
		}
		if !needsUpdate {
			fmt.Printf("Already up to date (%s)\n", version)
			return nil
		}
		fmt.Printf("Updating %s -> %s...\n", version, latest)
		if err := updater.Update(url); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		fmt.Printf("Updated to %s\n", latest)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $XDG_CONFIG_HOME/fluid/config.yaml)")
	rootCmd.Flags().BoolP("version", "v", false, "print version")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := paths.MaybeMigrate(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: migration failed: %v\n", err)
		}
		return nil
	}
	doctorCmd.Flags().String("host", "", "host name from config (default: localhost)")
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(doctorCmd)
}

// runMCP launches the MCP server on stdio
func runMCP() error {
	configPath := cfgFile
	if configPath == "" {
		var err error
		configPath, err = paths.ConfigFile()
		if err != nil {
			return fmt.Errorf("determine config path: %w", err)
		}
	}

	var err error
	cfg, err = tui.EnsureConfigExists(configPath)
	if err != nil {
		return fmt.Errorf("ensure config: %w", err)
	}

	// Log to file - stdout is the MCP transport
	logPath := filepath.Join(filepath.Dir(configPath), "fluid-mcp.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		logFile = nil
	}
	var logger *slog.Logger
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
		logger = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	svc, st, tele, err := initServicesForMCPTUI(cfg, logger)
	if err != nil {
		return fmt.Errorf("init services: %w", err)
	}
	defer func() { _ = svc.Close() }()
	defer func() { _ = st.Close() }()

	srv := fluidmcp.NewServer(cfg, st, svc, tele, logger)
	return srv.Serve()
}

// runTUI launches the interactive TUI
func runTUI() error {
	configPath := cfgFile
	if configPath == "" {
		var err error
		configPath, err = paths.ConfigFile()
		if err != nil {
			return fmt.Errorf("determine config path: %w", err)
		}
	}

	var err error
	cfg, err = tui.EnsureConfigExists(configPath)
	if err != nil {
		return fmt.Errorf("ensure config: %w", err)
	}

	// Check if onboarding is needed (first run)
	if !cfg.OnboardingComplete {
		updatedCfg, err := tui.RunOnboarding(cfg, configPath)
		if err != nil {
			return fmt.Errorf("onboarding: %w", err)
		}
		cfg = updatedCfg
		cfg.OnboardingComplete = true
		if err := cfg.Save(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save onboarding status: %v\n", err)
		}
	}

	// Log to file to avoid corrupting the TUI
	logPath := filepath.Join(filepath.Dir(configPath), "fluid.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open log file %s: %v\n", logPath, err)
		logFile = nil
	}
	var fileLogger *slog.Logger
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
		fileLogger = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		fileLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	svc, st, tele, err := initServicesForMCPTUI(cfg, fileLogger)
	if err != nil {
		return fmt.Errorf("init services: %w", err)
	}
	defer func() { _ = svc.Close() }()
	defer func() { _ = st.Close() }()

	agent := tui.NewFluidAgent(cfg, st, svc, tele, fileLogger)

	model := tui.NewModel("fluid", "local", "vm-agent", agent, cfg, configPath)
	return tui.Run(model)
}

// initServicesForMCPTUI creates sandbox.Service, store, and telemetry for MCP/TUI modes.
func initServicesForMCPTUI(loadedCfg *config.Config, logger *slog.Logger) (sandbox.Service, store.Store, telemetry.Service, error) {
	ctx := context.Background()
	st, err := sqlite.New(ctx, store.Config{AutoMigrate: true})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open store: %w", err)
	}

	tele, err := telemetry.NewService(loadedCfg.Telemetry)
	if err != nil {
		tele = telemetry.NewNoopService()
	}

	daemonAddr := loadedCfg.ControlPlane.DaemonAddress
	if daemonAddr == "" {
		daemonAddr = "localhost:9091"
	}

	svc, err := sandbox.NewRemoteService(daemonAddr, loadedCfg.ControlPlane)
	if err != nil {
		_ = st.Close()
		tele.Close()
		return nil, nil, nil, fmt.Errorf("connect to daemon at %s: %w", daemonAddr, err)
	}

	return svc, st, tele, nil
}
