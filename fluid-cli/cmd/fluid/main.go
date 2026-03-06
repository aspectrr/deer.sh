package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/aspectrr/fluid.sh/fluid-cli/internal/audit"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/config"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/doctor"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/hostexec"
	fluidmcp "github.com/aspectrr/fluid.sh/fluid-cli/internal/mcp"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/paths"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/readonly"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/redact"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/sandbox"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/source"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/sourcekeys"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/sshconfig"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/store"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/store/sqlite"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/telemetry"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/tui"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/updater"
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

// --- source commands ---

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Manage source hosts for read-only access",
}

var sourcePrepareCmd = &cobra.Command{
	Use:   "prepare <hostname>",
	Short: "Prepare a host for read-only access",
	Long:  "Set up the fluid-readonly user and SSH key on a remote host. Uses ssh -G to resolve connection details from ~/.ssh/config.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname := args[0]
		return runSourcePrepare(hostname)
	},
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured source hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSourceList()
	},
}

// --- audit commands ---

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Manage the audit log",
}

var auditVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify hash chain integrity of the audit log",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuditVerify()
	},
}

var auditShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show recent audit log entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuditShow()
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

	sourceCmd.AddCommand(sourcePrepareCmd)
	sourceCmd.AddCommand(sourceListCmd)
	auditCmd.AddCommand(auditVerifyCmd)
	auditCmd.AddCommand(auditShowCmd)

	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(sourceCmd)
	rootCmd.AddCommand(auditCmd)
}

// resolveConfigPath returns the config file path, using the flag or default.
func resolveConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}
	return paths.ConfigFile()
}

// runSourcePrepare prepares a host for read-only fluid access.
func runSourcePrepare(hostname string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	useColor := os.Getenv("NO_COLOR") == ""
	green := func(s string) string {
		if useColor {
			return "\033[32m" + s + "\033[0m"
		}
		return s
	}
	red := func(s string) string {
		if useColor {
			return "\033[31m" + s + "\033[0m"
		}
		return s
	}

	// 0. Probe if host is already prepared
	probeKeyPath := sourcekeys.GetPrivateKeyPath(loadedCfg.SSH.SourceKeyDir)
	probeCtx, probeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	probeRun := hostexec.NewReadOnlySSHAlias(hostname, probeKeyPath)
	_, _, probeCode, probeErr := probeRun(probeCtx, "echo ok")
	probeCancel()
	if probeErr == nil && probeCode == 0 {
		fmt.Printf("  Host %s already has fluid-readonly access configured.\n", hostname)
		fmt.Print("  Re-prepare? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" {
			fmt.Println("  Skipped.")
			return nil
		}
	}

	// 1. Resolve SSH connection details
	fmt.Printf("  Resolving %s via ssh config...\n", hostname)
	resolved, err := sshconfig.Resolve(hostname)
	if err != nil {
		return fmt.Errorf("resolve SSH config for %s: %w", hostname, err)
	}
	fmt.Printf("  %s Resolved: %s@%s:%d\n", green("[ok]"), resolved.User, resolved.Hostname, resolved.Port)

	// 2. Generate dedicated key pair
	fmt.Printf("  Generating fluid SSH key pair...\n")
	privPath, pubKey, err := sourcekeys.EnsureKeyPair(loadedCfg.SSH.SourceKeyDir)
	if err != nil {
		return fmt.Errorf("generate key pair: %w", err)
	}
	fmt.Printf("  %s Key pair at %s\n", green("[ok]"), privPath)

	// 3. SSH to host using the original alias so ~/.ssh/config is fully applied
	fmt.Printf("  Preparing %s for read-only access...\n", hostname)
	sshRunFn := hostexec.NewSSHAlias(hostname)
	sshRun := readonly.SSHRunFunc(sshRunFn)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	progress := func(p readonly.PrepareProgress) {
		if !p.Done {
			fmt.Printf("    [%d/%d] %s...\n", p.Step+1, p.Total, p.StepName)
		} else {
			fmt.Printf("    [%d/%d] %s %s\n", p.Step+1, p.Total, p.StepName, green("[ok]"))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	_, err = readonly.PrepareWithKey(ctx, sshRun, pubKey, progress, logger)
	if err != nil {
		fmt.Printf("  %s Preparation failed: %v\n", red("[error]"), err)
		return err
	}

	// 4. Update config
	if err := source.SavePreparedHost(loadedCfg, configPath, hostname, resolved); err != nil {
		return fmt.Errorf("saving config after prepare: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s Host %q is ready for read-only access.\n", green("[done]"), hostname)
	fmt.Printf("  Run `fluid` to start the agent and inspect this host.\n")
	return nil
}

// runSourceList lists configured source hosts.
func runSourceList() error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(loadedCfg.Hosts) == 0 {
		fmt.Println("  No source hosts configured.")
		fmt.Println("  Run: fluid source prepare <hostname>")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-20s %-25s %-10s\n", "NAME", "ADDRESS", "STATUS")
	fmt.Printf("  %-20s %-25s %-10s\n", strings.Repeat("-", 20), strings.Repeat("-", 25), strings.Repeat("-", 10))
	for _, h := range loadedCfg.Hosts {
		status := "not ready"
		if h.Prepared {
			status = "ready"
		}
		addr := h.Address
		if h.SSHPort != 0 && h.SSHPort != 22 {
			addr = fmt.Sprintf("%s:%d", h.Address, h.SSHPort)
		}
		fmt.Printf("  %-20s %-25s %-10s\n", h.Name, addr, status)
	}
	fmt.Println()
	return nil
}

// runAuditVerify verifies audit log hash chain integrity.
func runAuditVerify() error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logPath := loadedCfg.Audit.LogPath
	if logPath == "" {
		return fmt.Errorf("audit log path not configured")
	}

	valid, brokenAt, err := audit.VerifyChain(logPath)
	if err != nil {
		return fmt.Errorf("verify audit chain: %w", err)
	}

	if valid {
		fmt.Println("  Audit log chain is valid.")
	} else {
		fmt.Printf("  Audit log chain is BROKEN at sequence %d.\n", brokenAt)
		os.Exit(1)
	}
	return nil
}

// runAuditShow shows recent audit log entries.
func runAuditShow() error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logPath := loadedCfg.Audit.LogPath
	if logPath == "" {
		return fmt.Errorf("audit log path not configured")
	}

	entries, err := audit.ReadRecent(logPath, 50)
	if err != nil {
		return fmt.Errorf("read audit log: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("  No audit entries found.")
		return nil
	}

	for _, e := range entries {
		line := fmt.Sprintf("  [%d] %s %s", e.Seq, e.Timestamp, e.Type)
		if e.Tool != "" {
			line += fmt.Sprintf(" tool=%s", e.Tool)
		}
		if e.Error != "" {
			line += fmt.Sprintf(" error=%s", e.Error)
		}
		fmt.Println(line)
	}
	return nil
}

// runMCP launches the MCP server on stdio
func runMCP() error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

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

	core, err := initCoreServices(cfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()
	if core.auditLog != nil {
		defer func() { _ = core.auditLog.Close() }()
	}

	core.telemetry.Track("cli_session_start", map[string]any{"mode": "mcp"})

	svc := initSandboxService(cfg, logger)
	defer func() { _ = svc.Close() }()

	srv := fluidmcp.NewServer(cfg, core.store, svc, core.source, core.telemetry, logger)
	return srv.Serve()
}

// runTUI launches the interactive TUI
func runTUI() error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

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

	core, err := initCoreServices(cfg, fileLogger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()
	if core.auditLog != nil {
		defer func() { _ = core.auditLog.Close() }()
	}

	core.telemetry.Track("cli_session_start", map[string]any{"mode": "tui"})

	svc := initSandboxService(cfg, fileLogger)
	defer func() { _ = svc.Close() }()

	agent := tui.NewFluidAgent(cfg, core.store, svc, core.source, core.telemetry, core.redactor, core.auditLog, fileLogger)

	model := tui.NewModel("fluid", "daemon", "vm-agent", agent, cfg, configPath)
	return tui.Run(model)
}

// coreServices bundles the services returned by initCoreServices.
type coreServices struct {
	store     store.Store
	telemetry telemetry.Service
	source    *source.Service
	redactor  *redact.Redactor
	auditLog  *audit.Logger
}

// initCoreServices creates store, telemetry, source service, redactor, and audit logger.
// Always succeeds for the essential services (no gRPC needed).
func initCoreServices(loadedCfg *config.Config, logger *slog.Logger) (*coreServices, error) {
	ctx := context.Background()
	st, err := sqlite.New(ctx, store.Config{AutoMigrate: true})
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	tele, err := telemetry.NewService(loadedCfg.Telemetry)
	if err != nil {
		tele = telemetry.NewNoopService()
	}

	// Ensure source SSH keys exist
	keyPath := sourcekeys.GetPrivateKeyPath(loadedCfg.SSH.SourceKeyDir)
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		_, _, _ = sourcekeys.EnsureKeyPair(loadedCfg.SSH.SourceKeyDir)
	}

	srcSvc := source.NewService(loadedCfg, keyPath, logger)

	// Create redactor if enabled
	var r *redact.Redactor
	if loadedCfg.Redact.Enabled {
		var opts []redact.Option
		// Inject known config values for detection
		var hosts, addresses, keyPaths []string
		for _, h := range loadedCfg.Hosts {
			if h.Name != "" {
				hosts = append(hosts, h.Name)
			}
			if h.Address != "" {
				addresses = append(addresses, h.Address)
			}
		}
		if loadedCfg.SSH.SourceKeyDir != "" {
			keyPaths = append(keyPaths, loadedCfg.SSH.SourceKeyDir)
		}
		if loadedCfg.SSH.KeyDir != "" {
			keyPaths = append(keyPaths, loadedCfg.SSH.KeyDir)
		}
		opts = append(opts, redact.WithConfigValues(hosts, addresses, keyPaths))

		if len(loadedCfg.Redact.Allowlist) > 0 {
			opts = append(opts, redact.WithAllowlist(loadedCfg.Redact.Allowlist))
		}
		if len(loadedCfg.Redact.CustomPatterns) > 0 {
			opts = append(opts, redact.WithCustomPatterns(loadedCfg.Redact.CustomPatterns))
		}
		r = redact.New(opts...)
	}

	// Create audit logger if enabled
	var al *audit.Logger
	if loadedCfg.Audit.Enabled && loadedCfg.Audit.LogPath != "" {
		// Ensure audit log directory exists
		auditDir := filepath.Dir(loadedCfg.Audit.LogPath)
		if err := os.MkdirAll(auditDir, 0o755); err != nil {
			logger.Warn("could not create audit log directory", "path", auditDir, "error", err)
		} else {
			al, err = audit.NewLogger(loadedCfg.Audit.LogPath, loadedCfg.Audit.MaxSizeMB)
			if err != nil {
				logger.Warn("could not open audit log", "path", loadedCfg.Audit.LogPath, "error", err)
			} else {
				al.LogSessionStart()
			}
		}
	}

	return &coreServices{
		store:     st,
		telemetry: tele,
		source:    srcSvc,
		redactor:  r,
		auditLog:  al,
	}, nil
}

// initSandboxService creates a sandbox service. Returns NoopService if no sandbox hosts configured.
func initSandboxService(loadedCfg *config.Config, logger *slog.Logger) sandbox.Service {
	if !loadedCfg.HasSandboxHosts() {
		logger.Info("no sandbox hosts configured, using noop sandbox service")
		return sandbox.NewNoopService()
	}

	// Use the first sandbox host
	sh := loadedCfg.SandboxHosts[0]
	svc, err := sandbox.NewRemoteService(sh.DaemonAddress, config.ControlPlaneConfig{
		DaemonAddress:  sh.DaemonAddress,
		DaemonInsecure: sh.Insecure,
		DaemonCAFile:   sh.CAFile,
	}, loadedCfg.Hosts)
	if err != nil {
		logger.Warn("failed to connect to sandbox daemon, falling back to noop", "address", sh.DaemonAddress, "error", err)
		return sandbox.NewNoopService()
	}
	return svc
}
