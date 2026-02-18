package mcp

import (
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aspectrr/fluid.sh/fluid/internal/ansible"
	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/sandbox"
	"github.com/aspectrr/fluid.sh/fluid/internal/store"
	"github.com/aspectrr/fluid.sh/fluid/internal/telemetry"
)

const (
	// mcpAgentID identifies sandboxes created via the MCP server.
	mcpAgentID = "mcp-agent"
)

// Server wraps an MCP server that exposes fluid tools over stdio.
type Server struct {
	cfg             *config.Config
	store           store.Store
	service         sandbox.Service
	playbookService *ansible.PlaybookService
	telemetry       telemetry.Service
	logger          *slog.Logger
	mcpServer       *server.MCPServer
}

// NewServer creates a new MCP server wired to the fluid services.
func NewServer(cfg *config.Config, st store.Store, svc sandbox.Service, tele telemetry.Service, logger *slog.Logger) *Server {
	s := &Server{
		cfg:             cfg,
		store:           st,
		service:         svc,
		playbookService: ansible.NewPlaybookService(st, cfg.Ansible.PlaybooksDir),
		telemetry:       tele,
		logger:          logger,
	}

	s.mcpServer = server.NewMCPServer("fluid", "0.1.0",
		server.WithToolCapabilities(false),
	)

	s.registerTools()
	return s
}

// Serve starts the MCP server on stdio. Blocks until the connection closes.
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcpServer)
}

// registerTools registers all fluid tools on the MCP server.
func (s *Server) registerTools() {
	s.mcpServer.AddTool(mcp.NewTool("list_sandboxes",
		mcp.WithDescription("List all existing sandboxes with their state and IP addresses."),
	), s.handleListSandboxes)

	s.mcpServer.AddTool(mcp.NewTool("create_sandbox",
		mcp.WithDescription("Create a new sandbox VM by cloning from a source VM. Set live=true for current state, live=false to use cached image if available."),
		mcp.WithString("source_vm", mcp.Required(), mcp.Description("The name of the source VM to clone from (e.g., 'ubuntu-base').")),
		mcp.WithNumber("cpu", mcp.Description("Number of vCPUs (default: 2).")),
		mcp.WithNumber("memory_mb", mcp.Description("RAM in MB (default: 4096).")),
		mcp.WithBoolean("live", mcp.Description("If true, clone from the VM's live current state. If false (default), use cached image if available.")),
	), s.handleCreateSandbox)

	s.mcpServer.AddTool(mcp.NewTool("destroy_sandbox",
		mcp.WithDescription("Completely destroy a sandbox VM and remove its storage."),
		mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox to destroy.")),
	), s.handleDestroySandbox)

	s.mcpServer.AddTool(mcp.NewTool("run_command",
		mcp.WithDescription("Execute a shell command inside a sandbox via SSH."),
		mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox to run the command in.")),
		mcp.WithString("command", mcp.Required(), mcp.Description("The shell command to execute.")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Optional command timeout in seconds. 0 or omitted uses the configured default.")),
	), s.handleRunCommand)

	s.mcpServer.AddTool(mcp.NewTool("start_sandbox",
		mcp.WithDescription("Start a stopped sandbox VM."),
		mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox to start.")),
	), s.handleStartSandbox)

	s.mcpServer.AddTool(mcp.NewTool("stop_sandbox",
		mcp.WithDescription("Stop a running sandbox VM."),
		mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox to stop.")),
	), s.handleStopSandbox)

	s.mcpServer.AddTool(mcp.NewTool("get_sandbox",
		mcp.WithDescription("Get detailed information about a specific sandbox."),
		mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox.")),
	), s.handleGetSandbox)

	s.mcpServer.AddTool(mcp.NewTool("list_vms",
		mcp.WithDescription("List available source VMs that can be cloned to create sandboxes."),
	), s.handleListVMs)

	s.mcpServer.AddTool(mcp.NewTool("create_snapshot",
		mcp.WithDescription("Create a snapshot of the current sandbox state."),
		mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox.")),
		mcp.WithString("name", mcp.Description("Optional name for the snapshot.")),
	), s.handleCreateSnapshot)

	s.mcpServer.AddTool(mcp.NewTool("create_playbook",
		mcp.WithDescription("Create a new Ansible playbook."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the playbook.")),
		mcp.WithString("hosts", mcp.Description("Target hosts (default: 'all').")),
		mcp.WithBoolean("become", mcp.Description("Whether to use privilege escalation (sudo).")),
	), s.handleCreatePlaybook)

	s.mcpServer.AddTool(mcp.NewTool("add_playbook_task",
		mcp.WithDescription("Add a task to an Ansible playbook."),
		mcp.WithString("playbook_id", mcp.Required(), mcp.Description("The ID of the playbook.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the task.")),
		mcp.WithString("module", mcp.Required(), mcp.Description("Ansible module to use (e.g., 'apt', 'shell', 'copy').")),
		mcp.WithObject("params", mcp.Description("Parameters for the Ansible module.")),
	), s.handleAddPlaybookTask)

	s.mcpServer.AddTool(mcp.NewTool("edit_file",
		mcp.WithDescription("Edit a file on a sandbox VM by replacing text or create a new file."),
		mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox containing the file.")),
		mcp.WithString("path", mcp.Required(), mcp.Description("The absolute path to the file inside the sandbox.")),
		mcp.WithString("old_str", mcp.Description("The string to find and replace. If empty, the file will be created/overwritten with new_str.")),
		mcp.WithString("new_str", mcp.Required(), mcp.Description("The string to replace old_str with, or the content for a new file.")),
		mcp.WithBoolean("replace_all", mcp.Description("Replace all occurrences of old_str. Default: false.")),
	), s.handleEditFile)

	s.mcpServer.AddTool(mcp.NewTool("read_file",
		mcp.WithDescription("Read the contents of a file on a sandbox VM."),
		mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox containing the file.")),
		mcp.WithString("path", mcp.Required(), mcp.Description("The absolute path to the file inside the sandbox.")),
	), s.handleReadFile)

	s.mcpServer.AddTool(mcp.NewTool("list_playbooks",
		mcp.WithDescription("List all Ansible playbooks."),
	), s.handleListPlaybooks)

	s.mcpServer.AddTool(mcp.NewTool("get_playbook",
		mcp.WithDescription("Get the full definition of an Ansible playbook including its YAML content and all tasks."),
		mcp.WithString("playbook_id", mcp.Required(), mcp.Description("The ID of the playbook to retrieve.")),
	), s.handleGetPlaybook)

	s.mcpServer.AddTool(mcp.NewTool("run_source_command",
		mcp.WithDescription("Execute a read-only command on a source/golden VM."),
		mcp.WithString("source_vm", mcp.Required(), mcp.Description("The name of the source VM to run the command on.")),
		mcp.WithString("command", mcp.Required(), mcp.Description("The read-only diagnostic command to execute.")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Optional command timeout in seconds.")),
	), s.handleRunSourceCommand)

	s.mcpServer.AddTool(mcp.NewTool("read_source_file",
		mcp.WithDescription("Read the contents of a file on a source/golden VM. This is read-only."),
		mcp.WithString("source_vm", mcp.Required(), mcp.Description("The name of the source VM containing the file.")),
		mcp.WithString("path", mcp.Required(), mcp.Description("The absolute path to the file inside the source VM.")),
	), s.handleReadSourceFile)
}
