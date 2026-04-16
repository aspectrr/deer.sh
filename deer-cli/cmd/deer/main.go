package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/aspectrr/deer.sh/deer-cli/internal/ansible"
	"github.com/aspectrr/deer.sh/deer-cli/internal/audit"
	"github.com/aspectrr/deer.sh/deer-cli/internal/chatlog"
	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
	"github.com/aspectrr/deer.sh/deer-cli/internal/doctor"
	"github.com/aspectrr/deer.sh/deer-cli/internal/hostexec"
	deermcp "github.com/aspectrr/deer.sh/deer-cli/internal/mcp"
	"github.com/aspectrr/deer.sh/deer-cli/internal/paths"
	"github.com/aspectrr/deer.sh/deer-cli/internal/readonly"
	"github.com/aspectrr/deer.sh/deer-cli/internal/redact"
	"github.com/aspectrr/deer.sh/deer-cli/internal/sandbox"
	"github.com/aspectrr/deer.sh/deer-cli/internal/skill"
	"github.com/aspectrr/deer.sh/deer-cli/internal/source"
	"github.com/aspectrr/deer.sh/deer-cli/internal/sourcekeys"
	"github.com/aspectrr/deer.sh/deer-cli/internal/sshconfig"
	"github.com/aspectrr/deer.sh/deer-cli/internal/store"
	"github.com/aspectrr/deer.sh/deer-cli/internal/store/sqlite"
	"github.com/aspectrr/deer.sh/deer-cli/internal/telemetry"
	"github.com/aspectrr/deer.sh/deer-cli/internal/tui"
	"github.com/aspectrr/deer.sh/deer-cli/internal/updater"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	cfgFile      string
	cfg          *config.Config
	globalPrompt string
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
	Use:   "deer",
	Short: "Deer.sh - Make Infrastructure Safe for AI",
	Long:  "Deer.sh is a terminal agent that AI manage infrastructure via sandboxed resources, audit trails and human approval.",
	// Default to TUI when no subcommand is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetBool("version"); v {
			short := commit
			if len(short) > 7 {
				short = short[:7]
			}
			fmt.Printf("deer %s (%s, %s)\n", version, short, date)
			return nil
		}
		if globalPrompt != "" {
			return runHeadless(globalPrompt)
		}
		return runTUI()
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server on stdio",
	Long:  "Start an MCP (Model Context Protocol) server that exposes deer tools over stdio for use with Claude Code, Cursor, and other MCP clients.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCP()
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check daemon setup on a host",
	Long:  "Validate that the deer-daemon is properly installed and configured on a sandbox host.",
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
	Short:   "Update deer to the latest version",
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
	Long:  "Set up the deer-readonly user and SSH key on a remote host. Uses ssh -G to resolve connection details from ~/.ssh/config.",
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

var sourceRunCmd = &cobra.Command{
	Use:   "run <host> <command>",
	Short: "Run a read-only command on a source host",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]
		command := strings.Join(args[1:], " ")
		timeoutSec, _ := cmd.Flags().GetInt("timeout")
		return runSourceRun(host, command, timeoutSec)
	},
}

var sourceReadFileCmd = &cobra.Command{
	Use:   "read <host> <path>",
	Short: "Read a file from a source host",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSourceReadFile(args[0], args[1])
	},
}

// --- connect command ---

var connectCmd = &cobra.Command{
	Use:           "connect <address>",
	Short:         "Connect to a deer daemon and save config",
	Long:          "Test the gRPC connection to a deer-daemon, run doctor checks via SSH, display host info, and save the daemon to your config.",
	Args:          cobra.ExactArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		insecure, _ := cmd.Flags().GetBool("insecure")
		skipSave, _ := cmd.Flags().GetBool("no-save")
		sshUser, _ := cmd.Flags().GetString("ssh-user")
		return runConnect(args[0], name, insecure, skipSave, sshUser)
	},
}

// --- sandbox commands ---

var sandboxCmd = &cobra.Command{
	Use:   "sandbox",
	Short: "Manage sandbox VMs",
}

var sandboxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sandboxes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSandboxList()
	},
}

var sandboxCreateCmd = &cobra.Command{
	Use:   "create <source_vm>",
	Short: "Create a new sandbox VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceVM := args[0]
		cpu, _ := cmd.Flags().GetInt("cpu")
		memoryMB, _ := cmd.Flags().GetInt("memory")
		live, _ := cmd.Flags().GetBool("live")
		kafkaStub, _ := cmd.Flags().GetBool("kafka-stub")
		esStub, _ := cmd.Flags().GetBool("es-stub")
		return runSandboxCreate(sourceVM, cpu, memoryMB, live, kafkaStub, esStub)
	},
}

var sandboxDestroyCmd = &cobra.Command{
	Use:   "destroy <sandbox_id>",
	Short: "Destroy a sandbox VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSandboxDestroy(args[0])
	},
}

var sandboxStartCmd = &cobra.Command{
	Use:   "start <sandbox_id>",
	Short: "Start a stopped sandbox",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSandboxStart(args[0])
	},
}

var sandboxStopCmd = &cobra.Command{
	Use:   "stop <sandbox_id>",
	Short: "Stop a running sandbox",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSandboxStop(args[0])
	},
}

var sandboxGetCmd = &cobra.Command{
	Use:   "get <sandbox_id>",
	Short: "Get sandbox details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSandboxGet(args[0])
	},
}

var sandboxRunCmd = &cobra.Command{
	Use:   "run <sandbox_id> <command>",
	Short: "Run a command in a sandbox",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sandboxID := args[0]
		command := strings.Join(args[1:], " ")
		timeoutSec, _ := cmd.Flags().GetInt("timeout")
		return runSandboxRun(sandboxID, command, timeoutSec)
	},
}

var sandboxSnapshotCmd = &cobra.Command{
	Use:   "snapshot <sandbox_id> [name]",
	Short: "Create a snapshot of a sandbox",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sandboxID := args[0]
		name := ""
		if len(args) > 1 {
			name = args[1]
		}
		return runSandboxSnapshot(sandboxID, name)
	},
}

// --- playbook commands ---

var playbookCmd = &cobra.Command{
	Use:   "playbook",
	Short: "Manage Ansible playbooks",
}

// --- file commands ---

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manage files on sandboxes",
}

var fileReadCmd = &cobra.Command{
	Use:   "read <sandbox_id> <path>",
	Short: "Read a file from a sandbox",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFileRead(args[0], args[1])
	},
}

var fileEditCmd = &cobra.Command{
	Use:   "edit <sandbox_id> <path>",
	Short: "Edit a file on a sandbox",
	Long:  "Edit a file on a sandbox by replacing text or creating a new file. Use --old to specify text to replace, or omit to create/overwrite the file.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sandboxID := args[0]
		path := args[1]
		oldStr, _ := cmd.Flags().GetString("old")
		newStr, _ := cmd.Flags().GetString("new")
		replaceAll, _ := cmd.Flags().GetBool("replace-all")
		return runFileEdit(sandboxID, path, oldStr, newStr, replaceAll)
	},
}

var playbookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all playbooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlaybookList()
	},
}

var playbookCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new playbook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		hosts, _ := cmd.Flags().GetString("hosts")
		become, _ := cmd.Flags().GetBool("become")
		return runPlaybookCreate(name, hosts, become)
	},
}

var playbookGetCmd = &cobra.Command{
	Use:   "get <playbook_id>",
	Short: "Get playbook details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlaybookGet(args[0])
	},
}

var playbookAddTaskCmd = &cobra.Command{
	Use:   "add-task <playbook_id> <name> <module>",
	Short: "Add a task to a playbook",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		playbookID := args[0]
		name := args[1]
		module := args[2]
		paramsJSON, _ := cmd.Flags().GetString("params")
		return runPlaybookAddTask(playbookID, name, module, paramsJSON)
	},
}

// --- audit commands ---

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Manage the audit log",
}

// --- skills commands ---

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage deer skills",
	Long:  "Install, list, and remove domain-specific skills that provide the agent with expert knowledge for technologies like Elasticsearch, Kafka, PostgreSQL, and more.",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSkillsList()
	},
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install a skill from a local path or GitHub repo",
	Long:  "Install a skill from a local directory path (./my-skill) or GitHub source (owner/repo). The skill directory must contain a SKILL.md file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSkillsInstall(args[0])
	},
}

var skillsRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSkillsRemove(args[0])
	},
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $XDG_CONFIG_HOME/deer/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&globalPrompt, "prompt", "p", "", "run agent non-interactively with prompt and print session JSON to stdout")
	rootCmd.Flags().BoolP("version", "v", false, "print version")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := paths.MaybeMigrate(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: migration failed: %v\n", err)
		}
		return nil
	}
	doctorCmd.Flags().String("host", "", "host name from config (default: localhost)")

	connectCmd.Flags().String("name", "", "display name for this daemon (default: hostname from daemon)")
	connectCmd.Flags().Bool("insecure", false, "skip TLS verification (INSECURE: use only for local/dev daemons)")
	connectCmd.Flags().Bool("no-save", false, "test connection without saving to config")
	connectCmd.Flags().String("ssh-user", "", "SSH user for doctor checks (default: from SSH config)")

	sourceCmd.AddCommand(sourcePrepareCmd)
	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(sourceRunCmd)
	sourceCmd.AddCommand(sourceReadFileCmd)

	sourceRunCmd.Flags().Int("timeout", 0, "Command timeout in seconds")
	auditCmd.AddCommand(auditVerifyCmd)
	auditCmd.AddCommand(auditShowCmd)

	sandboxCmd.AddCommand(sandboxListCmd)
	sandboxCmd.AddCommand(sandboxCreateCmd)
	sandboxCmd.AddCommand(sandboxDestroyCmd)
	sandboxCmd.AddCommand(sandboxStartCmd)
	sandboxCmd.AddCommand(sandboxStopCmd)
	sandboxCmd.AddCommand(sandboxGetCmd)
	sandboxCmd.AddCommand(sandboxRunCmd)
	sandboxCmd.AddCommand(sandboxSnapshotCmd)

	sandboxCreateCmd.Flags().Int("cpu", 0, "Number of vCPUs")
	sandboxCreateCmd.Flags().Int("memory", 0, "RAM in MB")
	sandboxCreateCmd.Flags().Bool("live", false, "Clone from live state instead of cached image")
	sandboxCreateCmd.Flags().Bool("kafka-stub", false, "Start local Redpanda Kafka broker at localhost:9092 inside the sandbox")
	sandboxCreateCmd.Flags().Bool("es-stub", false, "Start local single-node Elasticsearch at localhost:9200 inside the sandbox")
	sandboxRunCmd.Flags().Int("timeout", 0, "Command timeout in seconds")

	playbookCmd.AddCommand(playbookListCmd)
	playbookCmd.AddCommand(playbookCreateCmd)
	playbookCmd.AddCommand(playbookGetCmd)
	playbookCmd.AddCommand(playbookAddTaskCmd)

	playbookCreateCmd.Flags().String("hosts", "", "Target hosts (default: 'all')")
	playbookCreateCmd.Flags().Bool("become", false, "Use privilege escalation (sudo)")
	playbookAddTaskCmd.Flags().String("params", "", "Task parameters as JSON")

	fileCmd.AddCommand(fileReadCmd)
	fileCmd.AddCommand(fileEditCmd)

	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsInstallCmd)
	skillsCmd.AddCommand(skillsRemoveCmd)

	fileEditCmd.Flags().String("old", "", "String to find and replace")
	fileEditCmd.Flags().String("new", "", "Replacement string (required)")
	fileEditCmd.Flags().Bool("replace-all", false, "Replace all occurrences")

	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(sourceCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(sandboxCmd)
	rootCmd.AddCommand(playbookCmd)
	rootCmd.AddCommand(fileCmd)
	rootCmd.AddCommand(skillsCmd)
}

// colorFunc returns an ANSI color wrapper when useColor is true.
func colorFunc(useColor bool, code string) func(string) string {
	return func(s string) string {
		if useColor {
			return code + s + "\033[0m"
		}
		return s
	}
}

// resolveConfigPath returns the config file path, using the flag or default.
func resolveConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}
	return paths.ConfigFile()
}

// runSourcePrepare prepares a host for read-only deer access.
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
	green := colorFunc(useColor, "\033[32m")
	red := colorFunc(useColor, "\033[31m")

	// 0. Probe if host is already prepared
	probeKeyPath := sourcekeys.GetPrivateKeyPath(loadedCfg.SSH.SourceKeyDir)
	probeCtx, probeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	probeRun := hostexec.NewReadOnlySSHAlias(hostname, probeKeyPath)
	_, _, probeCode, probeErr := probeRun(probeCtx, "echo ok")
	probeCancel()
	if probeErr == nil && probeCode == 0 {
		fmt.Printf("  Host %s already has deer-readonly access configured.\n", hostname)
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
	fmt.Printf("  Generating deer SSH key pair...\n")
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

	// 5. Deploy daemon identity key if available
	identityPubKey := config.DaemonIdentityPubKey(loadedCfg.SandboxHosts)
	if identityPubKey != "" {
		fmt.Printf("  Deploying daemon SSH key to %s...\n", hostname)
		deployCtx, deployCancel := context.WithTimeout(context.Background(), 30*time.Second)
		deployErr := readonly.DeployDaemonKey(deployCtx, sshRun, identityPubKey, logger)
		deployCancel()
		if deployErr != nil {
			fmt.Printf("  %s Daemon key deploy: %v\n", red("[warning]"), deployErr)
		} else {
			fmt.Printf("  %s Daemon SSH key deployed\n", green("[ok]"))
		}
	}

	fmt.Println()
	fmt.Printf("  %s Host %q is ready for read-only access.\n", green("[done]"), hostname)
	fmt.Printf("  Run `deer` to start the agent and inspect this host.\n")
	return nil
}

// runConnect tests a daemon connection, runs doctor checks, and saves config.
func runConnect(addr, name string, insecure, skipSave bool, sshUser string) error {
	// Append default gRPC port if not specified
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "9091")
	}

	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	useColor := os.Getenv("NO_COLOR") == ""
	green := colorFunc(useColor, "\033[32m")
	red := colorFunc(useColor, "\033[31m")
	dim := colorFunc(useColor, "\033[90m")

	if insecure {
		fmt.Println("  WARNING: using insecure TLS (no certificate verification)")
	}

	// 1. Connect and health check
	fmt.Printf("\n  Connecting to %s...\n", addr)

	cpCfg := config.ControlPlaneConfig{
		DaemonAddress:  addr,
		DaemonInsecure: insecure,
	}
	svc, err := sandbox.NewRemoteService(addr, cpCfg)
	if err != nil {
		fmt.Printf("  %s Failed to dial: %v\n", red("[error]"), err)
		return err
	}
	defer func() {
		_ = svc.Close()
	}()

	// 2. Health check with its own timeout
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer healthCancel()
	if err := svc.Health(healthCtx); err != nil {
		fmt.Printf("  %s Health check failed: %v\n", red("[error]"), err)
		return err
	}
	fmt.Printf("  %s Health check passed\n", green("[ok]"))

	// 3. Get host info with its own timeout
	infoCtx, infoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer infoCancel()
	info, err := svc.GetHostInfo(infoCtx)
	if err != nil {
		fmt.Printf("  %s Failed to get host info: %v\n", red("[error]"), err)
		return err
	}
	fmt.Printf("  %s Host info retrieved\n\n", green("[ok]"))

	fmt.Printf("  Hostname:    %s\n", info.Hostname)
	fmt.Printf("  Version:     %s\n", info.Version)
	fmt.Printf("  CPUs:        %d\n", info.TotalCPUs)
	fmt.Printf("  Memory:      %d MB\n", info.TotalMemoryMB)
	fmt.Printf("  Sandboxes:   %d active\n", info.ActiveSandboxes)
	fmt.Printf("  Images:      %d available\n", len(info.BaseImages))
	fmt.Println()

	// 4. Doctor checks via gRPC
	fmt.Printf("  Running doctor checks...\n\n")
	doctorCtx, doctorCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer doctorCancel()
	checkResults, doctorErr := svc.DoctorCheck(doctorCtx)
	if doctorErr != nil {
		fmt.Printf("  %s Doctor checks failed: %v\n\n", red("[error]"), doctorErr)
	} else {
		doctorResults := make([]doctor.CheckResult, len(checkResults))
		for i, r := range checkResults {
			doctorResults[i] = doctor.CheckResult{
				Name:     r.Name,
				Category: r.Category,
				Passed:   r.Passed,
				Message:  r.Message,
				FixCmd:   r.FixCmd,
			}
		}
		doctor.PrintResults(doctorResults, os.Stdout, useColor)
		fmt.Println()
	}

	// 5. Save config
	if skipSave {
		fmt.Println(dim("  --no-save: config not modified"))
		fmt.Println()
		return nil
	}

	if name == "" {
		if info.Hostname != "" {
			name = info.Hostname
		} else {
			name = "default"
		}
	}

	entry := config.SandboxHostConfig{
		Name:                 name,
		DaemonAddress:        addr,
		Insecure:             insecure,
		SSHUser:              sshUser,
		DaemonIdentityPubKey: info.SSHIdentityPubKey,
	}

	loadedCfg.SandboxHosts = config.UpsertSandboxHost(loadedCfg.SandboxHosts, entry)

	if err := loadedCfg.Save(configPath); err != nil {
		fmt.Printf("  %s Failed to save config: %v\n", red("[error]"), err)
		return err
	}
	fmt.Printf("  %s Saved %q (%s) to config\n", green("[ok]"), name, addr)

	// Deploy daemon identity key to all prepared source hosts
	if info.SSHIdentityPubKey != "" {
		preparedHosts := loadedCfg.PreparedHosts()
		if len(preparedHosts) > 0 {
			fmt.Println()
			fmt.Println("  Deploying daemon SSH key to prepared hosts...")
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			for _, h := range preparedHosts {
				sshRunFn := hostexec.NewSSHAlias(h.Name)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err := readonly.DeployDaemonKey(ctx, readonly.SSHRunFunc(sshRunFn), info.SSHIdentityPubKey, logger)
				cancel()
				if err != nil {
					fmt.Printf("  %s %s: %v\n", dim("[skip]"), h.Name, err)
				} else {
					fmt.Printf("  %s %s\n", green("[ok]"), h.Name)
				}
			}
		}
	}

	fmt.Println()
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
		fmt.Println("  Run: deer source prepare <hostname>")
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
	logPath := filepath.Join(filepath.Dir(configPath), "deer-mcp.log")
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

	srv := deermcp.NewServer(cfg, core.store, svc, core.source, core.telemetry, logger)
	return srv.Serve()
}

// runTUI launches the interactive TUI
// runHeadless runs the agent with a single prompt and writes the full session
// as a JSON array to stdout. Uses the same service setup as runTUI but skips
// the Bubbletea model entirely.
func runHeadless(prompt string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	cfg, err = tui.EnsureConfigExists(configPath)
	if err != nil {
		return fmt.Errorf("ensure config: %w", err)
	}

	fileLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	core, err := initCoreServices(cfg, fileLogger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()
	if core.auditLog != nil {
		defer func() { _ = core.auditLog.Close() }()
	}

	svc := initSandboxService(cfg, fileLogger)
	defer func() { _ = svc.Close() }()

	if cfg.ChatsDir == "" {
		return fmt.Errorf("chats_dir not configured")
	}
	chatLogger, _, err := chatlog.New(cfg.ChatsDir)
	if err != nil {
		return fmt.Errorf("open chat log: %w", err)
	}
	chatLogger.LogSessionStart(cfg.AIAgent.Model)

	agent := tui.NewDeerAgent(cfg, core.store, svc, core.source, core.telemetry, core.redactor, core.auditLog, chatLogger, fileLogger)

	ctx := context.Background()
	if _, err := agent.RunHeadless(ctx, prompt); err != nil {
		chatLogger.LogSessionEnd(0, 0)
		_ = chatLogger.Close()
		return fmt.Errorf("agent: %w", err)
	}

	chatLogger.LogSessionEnd(0, 0)
	_ = chatLogger.Close()

	events, err := chatLogger.ReadEvents()
	if err != nil {
		return fmt.Errorf("read chat log: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(events)
}

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
	logPath := filepath.Join(filepath.Dir(configPath), "deer.log")
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

	var chatLogger *chatlog.Logger
	if cfg.ChatsDir != "" {
		var sessionID string
		var chatErr error
		chatLogger, sessionID, chatErr = chatlog.New(cfg.ChatsDir)
		if chatErr != nil {
			fileLogger.Warn("failed to open chat log", "error", chatErr)
		} else {
			defer func() { _ = chatLogger.Close() }()
			chatLogger.LogSessionStart(cfg.AIAgent.Model)
			fileLogger.Info("chat log", "session_id", sessionID, "path", cfg.ChatsDir+"/"+sessionID+".jsonl")
		}
	}

	agent := tui.NewDeerAgent(cfg, core.store, svc, core.source, core.telemetry, core.redactor, core.auditLog, chatLogger, fileLogger)

	model := tui.NewModel("deer", "daemon", "vm-agent", agent, cfg, configPath, fileLogger)
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
	if sh.Insecure {
		fmt.Printf("  \033[33m[warning]\033[0m: connecting to %s with TLS verification disabled (from saved config)\n", sh.DaemonAddress)
	}
	svc, err := sandbox.NewRemoteService(sh.DaemonAddress, config.ControlPlaneConfig{
		DaemonAddress:  sh.DaemonAddress,
		DaemonInsecure: sh.Insecure,
		DaemonCAFile:   sh.CAFile,
	})
	if err != nil {
		logger.Warn("failed to connect to sandbox daemon, falling back to noop", "address", sh.DaemonAddress, "error", err)
		return sandbox.NewNoopService()
	}
	return svc
}

// --- sandbox command handlers ---

func runSandboxList() error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() {
		if err := core.store.Close(); err != nil {
			logger.Error("failed to close store", "error", err)
		}
	}()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() {
		if err := svc.Close(); err != nil {
			logger.Error("failed to close sandbox service", "error", err)
		}
	}()

	sandboxes, err := svc.ListSandboxes(ctx)
	if err != nil {
		return fmt.Errorf("list sandboxes: %w", err)
	}

	if len(sandboxes) == 0 {
		fmt.Println("  No sandboxes found.")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-20s %-15s %-20s %-15s %s\n", "ID", "NAME", "STATE", "BASE IMAGE", "IP")
	fmt.Printf("  %-20s %-15s %-20s %-15s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 15), strings.Repeat("-", 20), strings.Repeat("-", 15), strings.Repeat("-", 15))
	for _, sb := range sandboxes {
		ip := "-"
		if sb.IPAddress != "" {
			ip = sb.IPAddress
		}
		fmt.Printf("  %-20s %-15s %-20s %-15s %s\n", sb.ID, sb.Name, sb.State, sb.BaseImage, ip)
	}
	fmt.Println()
	return nil
}

func runSandboxCreate(sourceVM string, cpu, memoryMB int, live, kafkaStub, esStub bool) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() {
		if err := core.store.Close(); err != nil {
			logger.Error("failed to close store", "error", err)
		}
	}()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() {
		if err := svc.Close(); err != nil {
			logger.Error("failed to close sandbox service", "error", err)
		}
	}()

	sb, err := svc.CreateSandbox(ctx, sandbox.CreateRequest{
		SourceVM:                  sourceVM,
		AgentID:                   "cli",
		VCPUs:                     cpu,
		MemoryMB:                  memoryMB,
		Live:                      live,
		SimpleKafkaBroker:         kafkaStub,
		SimpleElasticsearchBroker: esStub,
	})
	if err != nil {
		return fmt.Errorf("create sandbox: %w", err)
	}

	fmt.Printf("  Created sandbox %s (%s)\n", sb.ID, sb.Name)
	if sb.IPAddress != "" {
		fmt.Printf("  IP: %s\n", sb.IPAddress)
	}
	return nil
}

func runSandboxDestroy(sandboxID string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() {
		if err := core.store.Close(); err != nil {
			logger.Error("failed to close store", "error", err)
		}
	}()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() {
		if err := svc.Close(); err != nil {
			logger.Error("failed to close sandbox service", "error", err)
		}
	}()

	err = svc.DestroySandbox(ctx, sandboxID)
	if err != nil {
		return fmt.Errorf("destroy sandbox: %w", err)
	}

	fmt.Printf("  Destroyed sandbox %s\n", sandboxID)
	return nil
}

func runSandboxStart(sandboxID string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() { _ = svc.Close() }()

	sb, err := svc.StartSandbox(ctx, sandboxID)
	if err != nil {
		return fmt.Errorf("start sandbox: %w", err)
	}

	fmt.Printf("  Started sandbox %s\n", sandboxID)
	if sb.IPAddress != "" {
		fmt.Printf("  IP: %s\n", sb.IPAddress)
	}
	return nil
}

func runSandboxStop(sandboxID string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() { _ = svc.Close() }()

	err = svc.StopSandbox(ctx, sandboxID, false)
	if err != nil {
		return fmt.Errorf("stop sandbox: %w", err)
	}

	fmt.Printf("  Stopped sandbox %s\n", sandboxID)
	return nil
}

func runSandboxGet(sandboxID string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() { _ = svc.Close() }()

	sb, err := svc.GetSandbox(ctx, sandboxID)
	if err != nil {
		return fmt.Errorf("get sandbox: %w", err)
	}

	fmt.Println()
	fmt.Printf("  ID:         %s\n", sb.ID)
	fmt.Printf("  Name:       %s\n", sb.Name)
	fmt.Printf("  State:      %s\n", sb.State)
	fmt.Printf("  Base Image: %s\n", sb.BaseImage)
	fmt.Printf("  Agent ID:   %s\n", sb.AgentID)
	fmt.Printf("  Created:    %s\n", sb.CreatedAt.Format(time.RFC3339))
	if sb.IPAddress != "" {
		fmt.Printf("  IP:         %s\n", sb.IPAddress)
	}
	fmt.Println()
	return nil
}

func runSandboxRun(sandboxID, command string, timeoutSec int) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() { _ = svc.Close() }()

	result, err := svc.RunCommand(ctx, sandboxID, command, timeoutSec, nil)
	if err != nil {
		return fmt.Errorf("run command: %w", err)
	}

	fmt.Printf("  Exit code: %d\n", result.ExitCode)
	if result.Stdout != "" {
		fmt.Println("  STDOUT:")
		fmt.Println(indentLines(result.Stdout, "    "))
	}
	if result.Stderr != "" {
		fmt.Println("  STDERR:")
		fmt.Println(indentLines(result.Stderr, "    "))
	}
	return nil
}

func runSandboxSnapshot(sandboxID, name string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() { _ = svc.Close() }()

	if name == "" {
		name = fmt.Sprintf("snap-%d", time.Now().Unix())
	}

	snap, err := svc.CreateSnapshot(ctx, sandboxID, name)
	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	fmt.Printf("  Created snapshot %s (%s)\n", snap.SnapshotID, snap.SnapshotName)
	return nil
}

// indentLines indents each line of text with the given prefix
func indentLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, prefix+line)
	}
	return strings.Join(result, "\n")
}

// --- playbook command handlers ---

func runPlaybookList() error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	playbookSvc := ansible.NewPlaybookService(core.store, loadedCfg.Ansible.PlaybooksDir)

	playbooks, err := playbookSvc.ListPlaybooks(ctx, nil)
	if err != nil {
		return fmt.Errorf("list playbooks: %w", err)
	}

	if len(playbooks) == 0 {
		fmt.Println("  No playbooks found.")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-20s %-30s %s\n", "ID", "NAME", "CREATED")
	fmt.Printf("  %-20s %-30s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 30), strings.Repeat("-", 20))
	for _, pb := range playbooks {
		path := ""
		if pb.FilePath != nil {
			path = *pb.FilePath
		}
		fmt.Printf("  %-20s %-30s %s\n", pb.ID, pb.Name, pb.CreatedAt.Format("2006-01-02"))
		if path != "" {
			fmt.Printf("  %s%s\n", strings.Repeat(" ", 22), path)
		}
	}
	fmt.Println()
	return nil
}

func runPlaybookCreate(name, hosts string, become bool) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	playbookSvc := ansible.NewPlaybookService(core.store, loadedCfg.Ansible.PlaybooksDir)

	pb, err := playbookSvc.CreatePlaybook(ctx, ansible.CreatePlaybookRequest{
		Name:   name,
		Hosts:  hosts,
		Become: become,
	})
	if err != nil {
		return fmt.Errorf("create playbook: %w", err)
	}

	fmt.Printf("  Created playbook %s (%s)\n", pb.ID, pb.Name)
	if pb.FilePath != nil {
		fmt.Printf("  Path: %s\n", *pb.FilePath)
	}
	return nil
}

func runPlaybookGet(playbookID string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	playbookSvc := ansible.NewPlaybookService(core.store, loadedCfg.Ansible.PlaybooksDir)

	pbWithTasks, err := playbookSvc.GetPlaybookWithTasks(ctx, playbookID)
	if err != nil {
		return fmt.Errorf("get playbook: %w", err)
	}

	yamlContent, err := playbookSvc.ExportPlaybook(ctx, playbookID)
	if err != nil {
		return fmt.Errorf("export playbook: %w", err)
	}

	fmt.Println()
	fmt.Printf("  ID:      %s\n", pbWithTasks.Playbook.ID)
	fmt.Printf("  Name:    %s\n", pbWithTasks.Playbook.Name)
	fmt.Printf("  Hosts:   %s\n", pbWithTasks.Playbook.Hosts)
	fmt.Printf("  Become:  %v\n", pbWithTasks.Playbook.Become)
	fmt.Printf("  Created: %s\n", pbWithTasks.Playbook.CreatedAt.Format(time.RFC3339))
	if pbWithTasks.Playbook.FilePath != nil {
		fmt.Printf("  Path:    %s\n", *pbWithTasks.Playbook.FilePath)
	}

	if len(pbWithTasks.Tasks) > 0 {
		fmt.Println("\n  Tasks:")
		for _, t := range pbWithTasks.Tasks {
			fmt.Printf("    [%d] %s (%s)\n", t.Position, t.Name, t.Module)
		}
	}

	fmt.Println("\n  YAML Content:")
	fmt.Println(indentLines(string(yamlContent), "    "))
	fmt.Println()
	return nil
}

func runPlaybookAddTask(playbookID, name, module, paramsJSON string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	playbookSvc := ansible.NewPlaybookService(core.store, loadedCfg.Ansible.PlaybooksDir)

	var params map[string]any
	if paramsJSON != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return fmt.Errorf("parse params JSON: %w", err)
		}
	}

	task, err := playbookSvc.AddTask(ctx, playbookID, ansible.AddTaskRequest{
		Name:   name,
		Module: module,
		Params: params,
	})
	if err != nil {
		return fmt.Errorf("add task: %w", err)
	}

	fmt.Printf("  Added task %s to playbook %s\n", task.ID, playbookID)
	fmt.Printf("  Position: %d\n", task.Position)
	return nil
}

// --- file command handlers ---

func runFileRead(sandboxID, path string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() { _ = svc.Close() }()

	validatedPath, err := deermcp.ValidateFilePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	escapedPath, err := deermcp.ShellEscape(validatedPath)
	if err != nil {
		return fmt.Errorf("escape path: %w", err)
	}

	result, err := svc.RunCommand(ctx, sandboxID, fmt.Sprintf("base64 %s", escapedPath), 0, nil)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("read file failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(result.Stdout))
	if err != nil {
		return fmt.Errorf("decode file content: %w", err)
	}

	fmt.Println(string(decoded))
	return nil
}

func runFileEdit(sandboxID, path, oldStr, newStr string, replaceAll bool) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	svc := initSandboxService(loadedCfg, logger)
	defer func() { _ = svc.Close() }()

	validatedPath, err := deermcp.ValidateFilePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	escapedPath, err := deermcp.ShellEscape(validatedPath)
	if err != nil {
		return fmt.Errorf("escape path: %w", err)
	}

	if oldStr == "" {
		// Create/overwrite file
		if err := deermcp.CheckFileSize(int64(len(newStr))); err != nil {
			return fmt.Errorf("file too large: %w", err)
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(newStr))
		cmd := fmt.Sprintf("base64 -d > %s << '--DEER_B64--'\n%s\n--DEER_B64--", escapedPath, encoded)
		result, err := svc.RunCommand(ctx, sandboxID, cmd, 0, nil)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("create file failed with exit code %d: %s", result.ExitCode, result.Stderr)
		}
		fmt.Printf("  Created file %s on sandbox %s\n", path, sandboxID)
		return nil
	}

	// Read existing file
	readResult, err := svc.RunCommand(ctx, sandboxID, fmt.Sprintf("base64 %s", escapedPath), 0, nil)
	if err != nil {
		return fmt.Errorf("read file for edit: %w", err)
	}
	if readResult.ExitCode != 0 {
		return fmt.Errorf("read file failed with exit code %d: %s", readResult.ExitCode, readResult.Stderr)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(readResult.Stdout))
	if err != nil {
		return fmt.Errorf("decode file content: %w", err)
	}
	original := string(decoded)

	if !strings.Contains(original, oldStr) {
		return fmt.Errorf("old_str not found in file")
	}

	n := 1
	if replaceAll {
		n = -1
	}
	edited := strings.Replace(original, oldStr, newStr, n)
	if err := deermcp.CheckFileSize(int64(len(edited))); err != nil {
		return fmt.Errorf("edited file too large: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(edited))
	writeCmd := fmt.Sprintf("base64 -d > %s << '--DEER_B64--'\n%s\n--DEER_B64--", escapedPath, encoded)
	writeResult, err := svc.RunCommand(ctx, sandboxID, writeCmd, 0, nil)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	if writeResult.ExitCode != 0 {
		return fmt.Errorf("write file failed with exit code %d: %s", writeResult.ExitCode, writeResult.Stderr)
	}

	fmt.Printf("  Edited file %s on sandbox %s\n", path, sandboxID)
	return nil
}

// --- source command handlers ---

func runSourceRun(host, command string, timeoutSec int) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()
	if timeoutSec > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()
	}

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	result, err := core.source.RunCommand(ctx, host, command)
	if err != nil {
		return fmt.Errorf("run source command: %w", err)
	}

	fmt.Printf("  Exit code: %d\n", result.ExitCode)
	if result.Stdout != "" {
		fmt.Println("  STDOUT:")
		fmt.Println(indentLines(result.Stdout, "    "))
	}
	if result.Stderr != "" {
		fmt.Println("  STDERR:")
		fmt.Println(indentLines(result.Stderr, "    "))
	}
	return nil
}

func runSourceReadFile(host, path string) error {
	configPath, err := resolveConfigPath()
	if err != nil {
		return fmt.Errorf("determine config path: %w", err)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	core, err := initCoreServices(loadedCfg, logger)
	if err != nil {
		return fmt.Errorf("init core services: %w", err)
	}
	defer func() { _ = core.store.Close() }()
	defer core.telemetry.Close()

	validatedPath, err := deermcp.ValidateFilePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	content, err := core.source.ReadFile(ctx, host, validatedPath)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	fmt.Println(content)
	return nil
}

// --- skills command handlers ---

func runSkillsList() error {
	skillsDir, err := skill.SkillsDir()
	if err != nil {
		return fmt.Errorf("resolve skills dir: %w", err)
	}

	loader := skill.NewLoader(skillsDir)
	count, err := loader.Discover()
	if err != nil {
		return fmt.Errorf("discover skills: %w", err)
	}

	if count == 0 {
		fmt.Println("  No skills installed.")
		fmt.Println()
		fmt.Println("  Install a skill with: deer skills install <source>")
		fmt.Println("  Sources can be local paths (./my-skill) or GitHub repos (owner/repo)")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-25s %-50s %s\n", "NAME", "DESCRIPTION", "VERSION")
	fmt.Printf("  %-25s %-50s %s\n", strings.Repeat("-", 25), strings.Repeat("-", 50), strings.Repeat("-", 10))
	for _, s := range loader.List() {
		desc := s.Description
		if len(desc) > 48 {
			desc = desc[:48] + "..."
		}
		ver := s.Version
		if ver == "" {
			ver = "-"
		}
		fmt.Printf("  %-25s %-50s %s\n", s.Name, desc, ver)
	}
	fmt.Println()
	return nil
}

func runSkillsInstall(source string) error {
	skillsDir, err := skill.EnsureSkillsDir()
	if err != nil {
		return fmt.Errorf("ensure skills dir: %w", err)
	}

	useColor := os.Getenv("NO_COLOR") == ""
	green := colorFunc(useColor, "\033[32m")
	red := colorFunc(useColor, "\033[31m")

	var srcDir string
	var srcType string
	var lockSource string

	// Check for GitHub source patterns:
	//   owner/repo
	//   owner/repo//path/to/skill
	//   https://github.com/owner/repo
	//   github.com/owner/repo
	if isGitHubSource(source) {
		srcType = "github"
		lockSource = source

		repoRef, subPath := parseGitHubSource(source)
		fmt.Printf("  Cloning %s...\n", repoRef)

		tmpDir, err := os.MkdirTemp("", "deer-skill-*")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		cloneURL := "https://github.com/" + repoRef + ".git"
		if err := gitCloneShallow(cloneURL, tmpDir); err != nil {
			fmt.Printf("  %s git clone failed: %v\n", red("[error]"), err)
			fmt.Printf("  Check that %s exists and is accessible\n", repoRef)
			return err
		}

		if subPath != "" {
			srcDir = filepath.Join(tmpDir, subPath)
		} else {
			srcDir = tmpDir
		}
	} else {
		// Local path
		srcType = "local"
		absPath, err := filepath.Abs(source)
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}
		srcDir = absPath
		lockSource = absPath
	}

	// Validate source has SKILL.md
	skillMD := filepath.Join(srcDir, "SKILL.md")
	data, err := os.ReadFile(skillMD)
	if err != nil {
		if os.IsNotExist(err) {
			if srcType == "github" {
				return fmt.Errorf("%s does not contain a SKILL.md file. Use owner/repo//path/to/skill if the skill is in a subdirectory", srcDir)
			}
			return fmt.Errorf("%s does not contain a SKILL.md file", srcDir)
		}
		return fmt.Errorf("read SKILL.md: %w", err)
	}

	s, err := skill.Parse(data)
	if err != nil {
		return fmt.Errorf("parse SKILL.md: %w", err)
	}

	// Copy to skills directory
	destDir := filepath.Join(skillsDir, s.Name)
	if srcDir != destDir {
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("create skill dir: %w", err)
		}
		if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), data, 0o644); err != nil {
			return fmt.Errorf("write SKILL.md: %w", err)
		}
	}

	// Update lock file
	lf, err := skill.LoadLock()
	if err != nil {
		return fmt.Errorf("load lock: %w", err)
	}
	lf.Add(s.Name, skill.LockEntry{
		Source:     lockSource,
		SourceType: srcType,
	})
	if err := lf.Save(); err != nil {
		return fmt.Errorf("save lock: %w", err)
	}

	fmt.Printf("  %s Installed skill %q from %s\n", green("[ok]"), s.Name, lockSource)
	if s.Description != "" {
		fmt.Printf("  %s\n", s.Description)
	}
	return nil
}

// isGitHubSource returns true if the source looks like a GitHub reference.
func isGitHubSource(source string) bool {
	if strings.HasPrefix(source, "github.com/") {
		return true
	}
	if strings.HasPrefix(source, "https://github.com/") {
		return true
	}
	// owner/repo pattern: exactly one slash, no leading dot or slash
	if !strings.HasPrefix(source, ".") && !strings.HasPrefix(source, "/") {
		parts := strings.SplitN(source, "//", 2)
		repo := parts[0]
		slashCount := strings.Count(repo, "/")
		if slashCount == 1 && !strings.Contains(repo, " ") {
			return true
		}
	}
	return false
}

// parseGitHubSource returns the repo reference (owner/repo) and optional subpath.
// Supports: owner/repo, owner/repo//path/to/skill, github.com/owner/repo, https://github.com/owner/repo
func parseGitHubSource(source string) (repoRef, subPath string) {
	// Strip URL prefixes
	source = strings.TrimPrefix(source, "https://")
	source = strings.TrimPrefix(source, "github.com/")

	// Split on // for subpath
	if idx := strings.Index(source, "//"); idx >= 0 {
		repoRef = source[:idx]
		subPath = source[idx+2:]
	} else {
		repoRef = source
	}
	return repoRef, subPath
}

// gitCloneShallow does a depth-1 clone into the target directory.
func gitCloneShallow(url, dir string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", "--quiet", url, dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func runSkillsRemove(name string) error {
	skillsDir, err := skill.SkillsDir()
	if err != nil {
		return fmt.Errorf("resolve skills dir: %w", err)
	}

	useColor := os.Getenv("NO_COLOR") == ""
	green := colorFunc(useColor, "\033[32m")

	// Find and remove the skill directory
	skillDir := filepath.Join(skillsDir, name)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		// Try case-insensitive search
		entries, readErr := os.ReadDir(skillsDir)
		if readErr != nil {
			return fmt.Errorf("read skills dir: %w", readErr)
		}
		for _, e := range entries {
			if strings.EqualFold(e.Name(), name) {
				skillDir = filepath.Join(skillsDir, e.Name())
				break
			}
		}
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("remove skill directory: %w", err)
	}

	// Update lock file
	lf, err := skill.LoadLock()
	if err != nil {
		return fmt.Errorf("load lock: %w", err)
	}
	lf.Remove(name)
	if err := lf.Save(); err != nil {
		return fmt.Errorf("save lock: %w", err)
	}

	fmt.Printf("  %s Removed skill %q\n", green("[done]"), name)
	return nil
}
