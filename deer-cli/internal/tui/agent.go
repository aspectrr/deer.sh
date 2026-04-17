package tui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspectrr/deer.sh/deer-cli/internal/ansible"
	"github.com/aspectrr/deer.sh/deer-cli/internal/audit"
	"github.com/aspectrr/deer.sh/deer-cli/internal/chatlog"
	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
	"github.com/aspectrr/deer.sh/deer-cli/internal/hostexec"
	"github.com/aspectrr/deer.sh/deer-cli/internal/llm"
	"github.com/aspectrr/deer.sh/deer-cli/internal/paths"
	"github.com/aspectrr/deer.sh/deer-cli/internal/readonly"
	"github.com/aspectrr/deer.sh/deer-cli/internal/redact"
	"github.com/aspectrr/deer.sh/deer-cli/internal/sandbox"
	"github.com/aspectrr/deer.sh/deer-cli/internal/skill"
	"github.com/aspectrr/deer.sh/deer-cli/internal/source"
	"github.com/aspectrr/deer.sh/deer-cli/internal/sourcekeys"
	"github.com/aspectrr/deer.sh/deer-cli/internal/sshconfig"
	"github.com/aspectrr/deer.sh/deer-cli/internal/store"
	"github.com/aspectrr/deer.sh/deer-cli/internal/telemetry"
)

const tlsDebuggingGuidance = "\n\nWhen debugging TLS/SSL issues on source hosts:\n" +
	"- If you get permission denied reading certificate files, don't retry - these files are intentionally restricted\n" +
	"- Use `journalctl -u <service> --no-pager -n 100` to find SSL errors in service logs\n" +
	"- Use `grep -i <service> /var/log/syslog` as a fallback if journalctl has no results\n" +
	"- Use `openssl s_client -connect localhost:<port>` to inspect the live certificate chain (no file access needed)\n" +
	"- Use `ls -la` on cert directories to check ownership and permissions as a diagnostic"

// PendingApproval represents a sandbox creation waiting for memory approval
type PendingApproval struct {
	Request      MemoryApprovalRequest
	SourceVM     string
	SandboxName  string
	Host         *config.HostConfig // nil for local
	ResponseChan chan bool
}

// DeerAgent implements AgentRunner for the deer CLI
type DeerAgent struct {
	cfg             *config.Config
	store           store.Store
	service         sandbox.Service
	sourceService   source.Provider
	llmClient       llm.Client
	playbookService *ansible.PlaybookService
	telemetry       telemetry.Service
	redactor        *redact.Redactor
	auditLog        *audit.Logger
	chatLog         *chatlog.Logger
	logger          *slog.Logger
	skillLoader     *skill.Loader

	// Status callback for sending updates to TUI
	statusCallback func(tea.Msg)

	// Conversation history for context
	history []llm.Message

	// Track sandboxes created during this session for cleanup on exit
	createdSandboxes []string

	// Currently active sandbox (for status bar display)
	currentSandboxID        string
	currentSandboxHost      string
	currentSandboxBaseImage string

	// Source VM tracking (for status bar and auto read-only)
	currentSourceVM string
	autoReadOnly    bool

	// displayReadOnly tracks sticky read-only display state after source VM ops.
	// Stays true after withAutoReadOnly exits until a write tool explicitly clears it.
	displayReadOnly bool

	// Pending approval for network access
	pendingNetworkApproval *PendingNetworkApproval

	// Pending approval for source command elevation
	pendingSourceAccess *PendingSourceAccess

	// Session-level elevated commands (host -> set of approved commands)
	sessionElevatedCommands map[string]map[string]bool

	// Read-only mode: only query tools are available to the LLM
	readOnly bool

	// Re-prepare warning: tracks the last host warned about re-prepare
	lastPrepareWarned string

	// Timeout for SetSandboxService swap operation (default 2s, configurable for tests)
	swapTimeout time.Duration

	// Dedup tracking for sensitive content redaction messages
	redactedSeen map[string]bool

	// Task list for tracking agent progress
	taskList *TaskList

	// cancelFunc cancels the active agent Run context when ESC is pressed.
	// mu protects cancelFunc, runID, done, currentSourceVM, autoReadOnly, and readOnly.
	cancelFunc context.CancelFunc
	runID      uint64
	done       chan struct{}
	mu         sync.Mutex
}

// PendingNetworkApproval represents a network access request waiting for approval
type PendingNetworkApproval struct {
	Request      NetworkApprovalRequest
	ResponseChan chan bool
}

// PendingSourceAccess represents a command elevation request waiting for approval
type PendingSourceAccess struct {
	Request      SourceAccessApprovalRequest
	ResponseChan chan SourceAccessApprovalResult
}

// NewDeerAgent creates a new deer agent
func NewDeerAgent(cfg *config.Config, st store.Store, svc sandbox.Service, srcSvc source.Provider, tele telemetry.Service, redactor *redact.Redactor, auditLog *audit.Logger, chatLog *chatlog.Logger, logger *slog.Logger) *DeerAgent {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	var llmClient llm.Client
	if cfg.AIAgent.Provider == "openrouter" {
		llmClient = llm.NewOpenRouterClient(cfg.AIAgent)
	}

	return &DeerAgent{
		cfg:                     cfg,
		store:                   st,
		service:                 svc,
		sourceService:           srcSvc,
		llmClient:               llmClient,
		playbookService:         ansible.NewPlaybookService(st, cfg.Ansible.PlaybooksDir),
		telemetry:               tele,
		redactor:                redactor,
		auditLog:                auditLog,
		chatLog:                 chatLog,
		logger:                  logger,
		skillLoader:             initSkillLoader(logger),
		history:                 make([]llm.Message, 0),
		swapTimeout:             2 * time.Second,
		redactedSeen:            make(map[string]bool),
		sessionElevatedCommands: make(map[string]map[string]bool),
	}
}

// initSkillLoader creates and populates a skill loader from the deer config directory.
func initSkillLoader(logger *slog.Logger) *skill.Loader {
	skillsDir, err := skill.SkillsDir()
	if err != nil {
		logger.Warn("skill loader: could not resolve skills dir", "error", err)
		return skill.NewLoader()
	}
	loader := skill.NewLoader(skillsDir)
	if count, err := loader.Discover(); err != nil {
		logger.Warn("skill loader: discover failed", "error", err)
	} else if count > 0 {
		logger.Info("skill loader: loaded skills", "count", count, "dir", skillsDir)
	}
	return loader
}

// SetStatusCallback sets the callback function for status updates
func (a *DeerAgent) SetStatusCallback(callback func(tea.Msg)) {
	a.statusCallback = callback
}

// SetReadOnly toggles read-only mode on the agent
func (a *DeerAgent) SetReadOnly(ro bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.readOnly = ro
}

// SetSandboxService hot-swaps the sandbox service (e.g. after /connect).
// Must be called after Cancel() to avoid race conditions with running agent.
// Waits for the running goroutine to finish before swapping.
func (a *DeerAgent) SetSandboxService(svc sandbox.Service) error {
	a.mu.Lock()
	if a.cancelFunc != nil {
		a.mu.Unlock()
		return fmt.Errorf("cannot swap sandbox service while agent is running; cancel first")
	}
	doneCh := a.done
	a.mu.Unlock()

	// Wait for any in-flight goroutine to finish (cancel was already called but
	// the goroutine may still be mid-tool-call).
	if doneCh != nil {
		select {
		case <-doneCh:
		case <-time.After(a.swapTimeout):
			return fmt.Errorf("timed out waiting for previous agent run to finish")
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancelFunc != nil {
		return fmt.Errorf("a new agent run started while waiting; cancel first")
	}
	if a.service != nil {
		_ = a.service.Close()
	}
	a.service = svc
	return nil
}

// sendStatus sends a status message through the callback if set
func (a *DeerAgent) sendStatus(msg tea.Msg) {
	if a.statusCallback != nil {
		a.statusCallback(msg)
	}
}

// finishRun sends the final TUI-facing status update and returns the only
// direct completion signal for Run(). AgentDoneMsg must not be queued through
// statusCallback, otherwise it can remain buffered and break the next run.
func (a *DeerAgent) finishRun(msg tea.Msg) tea.Msg {
	if msg != nil {
		a.sendStatus(msg)
	}
	return AgentDoneMsg{}
}

// sendRedactedMsg sends a SensitiveContentRedactedMsg with dedup by host/path.
// Only sends the message the first time per unique key per agent run.
func (a *DeerAgent) sendRedactedMsg(host, path string) {
	key := host
	if path != "" {
		key = host + ":" + path
	}
	if a.redactedSeen[key] {
		return
	}
	a.redactedSeen[key] = true
	a.sendStatus(SensitiveContentRedactedMsg{Host: host, Path: path})
}

// RunID returns the current run generation counter.
func (a *DeerAgent) RunID() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runID
}

// Cancel stops the currently running agent loop
func (a *DeerAgent) Cancel() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancelFunc != nil {
		a.cancelFunc()
		a.cancelFunc = nil
	}
}

// withAutoReadOnly temporarily enables read-only mode for source VM operations.
// It sets currentSourceVM, enables auto-read-only mode, and restores the previous
// state when the function returns.
func (a *DeerAgent) withAutoReadOnly(sourceVM string, fn func() (any, error)) (any, error) {
	a.mu.Lock()
	a.currentSourceVM = sourceVM
	wasAutoReadOnly := a.autoReadOnly
	var enterMsg *AutoReadOnlyMsg
	if !a.readOnly {
		a.autoReadOnly = true
		a.readOnly = true
		a.displayReadOnly = true
		enterMsg = &AutoReadOnlyMsg{SourceVM: sourceVM, Enabled: true}
	}
	a.mu.Unlock()
	if enterMsg != nil {
		a.sendStatus(*enterMsg)
	}
	defer func() {
		a.mu.Lock()
		a.currentSourceVM = ""
		if a.autoReadOnly && !wasAutoReadOnly {
			a.autoReadOnly = false
			a.readOnly = false
			// displayReadOnly stays true - cleared by the next write tool call
		}
		a.mu.Unlock()
	}()
	return fn()
}

// Run executes a command and returns the result
func (a *DeerAgent) Run(input string) tea.Cmd {
	// Increment runID eagerly so the caller can read it via RunID() immediately.
	a.mu.Lock()
	a.runID++
	currentRunID := a.runID
	a.mu.Unlock()

	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		doneCh := make(chan struct{})
		a.mu.Lock()
		a.cancelFunc = cancel
		a.done = doneCh
		a.mu.Unlock()
		defer func() {
			cancel()
			a.mu.Lock()
			a.cancelFunc = nil
			a.mu.Unlock()
			close(doneCh)
		}()

		// Handle slash commands
		if strings.HasPrefix(input, "/") {
			cmdName := input
			if idx := strings.Index(input[1:], " "); idx >= 0 {
				cmdName = input[:idx+1]
			}
			a.telemetry.Track("tui_slash_command", map[string]any{"command": cmdName})

			// Commands with arguments (checked before exact match switch)
			if strings.HasPrefix(input, "/prepare ") {
				hostname := strings.TrimSpace(strings.TrimPrefix(input, "/prepare "))
				if hostname == "" {
					return a.finishRun(AgentResponseMsg{Response: AgentResponse{
						Content: "Usage: `/prepare <hostname>` - specify an SSH host to prepare",
						Done:    true,
					}})
				}
				// Probe if host is already prepared
				if probeDeerReadonly(hostname, a.cfg.SSH.SourceKeyDir) {
					if a.lastPrepareWarned != hostname {
						a.lastPrepareWarned = hostname
						return a.finishRun(AgentResponseMsg{Response: AgentResponse{
							Content: fmt.Sprintf("Host %s is already prepared. Run `/prepare %s` again to re-prepare.", hostname, hostname),
							Done:    true,
						}})
					}
					a.lastPrepareWarned = ""
				} else {
					a.lastPrepareWarned = ""
				}
				return a.runPrepareInline(ctx, hostname)
			}

			switch input {
			// case "/vms": // use /hosts instead
			// 	result, err := a.listVMs(ctx)
			// 	return a.finishRun(AgentResponseMsg{Response: AgentResponse{
			// 		Content: a.formatVMsResult(result, err),
			// 		Done:    true,
			// 	}})
			case "/sandboxes":
				result, err := a.listSandboxes(ctx)
				return a.finishRun(AgentResponseMsg{Response: AgentResponse{
					Content: a.formatSandboxesResult(result, err),
					Done:    true,
				}})
			case "/hosts":
				if a.sourceService != nil {
					hosts := a.sourceService.ListHosts()
					var lines []string
					for _, h := range hosts {
						status := "not ready"
						if h.Prepared {
							status = "ready"
						}
						lines = append(lines, fmt.Sprintf("  %s (%s) - %s", h.Name, h.Address, status))
					}
					content := "**Source Hosts:**\n" + strings.Join(lines, "\n")
					if len(hosts) == 0 {
						content = "No source hosts configured. Run: `deer source prepare <hostname>`"
					}
					return a.finishRun(AgentResponseMsg{Response: AgentResponse{Content: content, Done: true}})
				}
				result, err := a.listHostsWithVMs(ctx)
				return a.finishRun(AgentResponseMsg{Response: AgentResponse{
					Content: a.formatHostsResult(result, err),
					Done:    true,
				}})
			case "/playbooks":
				result, err := a.listPlaybooks(ctx)
				return a.finishRun(AgentResponseMsg{Response: AgentResponse{
					Content: a.formatPlaybooksResult(result, err),
					Done:    true,
				}})
			case "/compact":
				// Manual compaction
				a.sendStatus(CompactStartMsg{})
				result, err := a.Compact(ctx)
				if err != nil {
					return a.finishRun(CompactErrorMsg{Err: err})
				}
				return a.finishRun(result)
			case "/context":
				// Show context usage
				usage := a.GetContextUsage()
				tokens := a.EstimateTokens()
				maxTokens := a.cfg.AIAgent.TotalContextTokens
				threshold := a.cfg.AIAgent.CompactThreshold
				return a.finishRun(AgentResponseMsg{Response: AgentResponse{
					Content: fmt.Sprintf("Context usage: %d/%d tokens (%.1f%%)\nAuto-compact threshold: %.0f%%",
						tokens, maxTokens, usage*100, threshold*100),
					Done: true,
				}})
			case "/allowlist":
				var b strings.Builder
				b.WriteString("## Read-Only Command Allowlist\n\n")
				b.WriteString("### Default Commands\n\n")
				defaults := readonly.AllowedCommandsList()
				for i, cmd := range defaults {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString("`" + cmd + "`")
				}
				b.WriteString("\n\n")
				if len(a.cfg.ExtraAllowedCommands) > 0 {
					b.WriteString("### User-Added Commands\n\n")
					for _, cmd := range a.cfg.ExtraAllowedCommands {
						b.WriteString("- `" + cmd + "`\n")
					}
					b.WriteString("\n")
				}
				b.WriteString("Edit extra commands in `/settings` or `config.yaml` under `extra_allowed_commands`.\n")
				return a.finishRun(AgentResponseMsg{Response: AgentResponse{
					Content: b.String(),
					Done:    true,
				}})
			case "/help":
				var b strings.Builder
				b.WriteString("## Available Commands\n\n")
				b.WriteString("- **/vms**: List available VMs for cloning\n")
				b.WriteString("- **/sandboxes**: List active sandboxes\n")
				b.WriteString("- **/hosts**: List configured remote hosts\n")
				b.WriteString("- **/playbooks**: List generated Ansible playbooks\n")
				b.WriteString("- **/prepare <host>**: Prepare a host for read-only access\n")
				b.WriteString("- **/allowlist**: Show the read-only command allowlist\n")
				b.WriteString("- **/compact**: Summarize and compact conversation history\n")
				b.WriteString("- **/context**: Show current context token usage\n")
				b.WriteString("- **/settings**: Open configuration settings\n")
				b.WriteString("- **/clear**: Clear conversation history\n")
				b.WriteString("- **/help**: Show this help message\n")
				b.WriteString("\n## Keyboard Shortcuts\n\n")
				b.WriteString("- **PgUp/PgDn**: Scroll conversation history\n")
				return a.finishRun(AgentResponseMsg{Response: AgentResponse{
					Content: b.String(),
					Done:    true,
				}})
			default:
				return a.finishRun(AgentResponseMsg{Response: AgentResponse{
					Content: fmt.Sprintf("Unknown command: %s. Available: /vms, /sandboxes, /hosts, /playbooks, /prepare, /allowlist, /compact, /context, /settings", input),
					Done:    true,
				}})
			}
		}

		// Add user message to history
		a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: input})

		// Log user input to audit log (length only, not content)
		if a.auditLog != nil {
			a.auditLog.LogUserInput(len(input))
		}
		if a.chatLog != nil {
			a.chatLog.LogUserMessage(input)
		}

		// LLM client is required
		if a.llmClient == nil || a.cfg.AIAgent.APIKey == "" {
			return a.finishRun(AgentErrorMsg{Err: fmt.Errorf("LLM provider not configured. Please set OPENROUTER_API_KEY environment variable or configure it in /settings")})
		}

		// Check if auto-compaction is needed before making LLM call
		if a.NeedsCompaction() {
			a.sendStatus(CompactStartMsg{})
			compactResult, err := a.Compact(ctx)
			if err != nil {
				// Log warning but continue - don't fail the request
				a.logger.Warn("auto-compaction failed", "error", err)
				a.sendStatus(CompactErrorMsg{Err: fmt.Errorf("auto-compact failed, continuing with full context: %w", err)})
			} else {
				// Send compact complete to TUI so user knows it happened
				a.sendStatus(compactResult)
			}
		}

		// LLM-driven execution loop
		for iteration := 0; ; iteration++ {
			if ctx.Err() != nil {
				return a.finishRun(AgentCancelledMsg{RunID: currentRunID})
			}
			a.logger.Debug("LLM loop iteration", "iteration", iteration, "history_len", len(a.history))
			systemPrompt := a.cfg.AIAgent.DefaultSystem
			tools := llm.GetTools()
			// Snapshot readOnly under lock
			a.mu.Lock()
			isReadOnly := a.readOnly
			a.mu.Unlock()
			// Tool selection precedence:
			// 1. No sandbox hosts AND no prepared hosts: minimal tools, nudge to /prepare
			// 2. No sandbox hosts but has prepared hosts: source-only read access
			// 3. Has sandbox hosts but read-only mode: observation-only tools
			// 4. Default: full tool set
			if !a.cfg.HasSandboxHosts() && len(a.cfg.PreparedHosts()) == 0 {
				tools = llm.GetNoSourceTools()
				systemPrompt += "\n\nThe user has not prepared any source hosts yet. You have no access to any servers. " +
					"You can still answer questions about infrastructure, Linux, DevOps, and help plan tasks. " +
					"If the user asks you to do anything that requires server access (running commands, reading files, diagnosing issues), " +
					"let them know they need to prepare a host first with `/prepare <hostname>` or `deer source prepare <hostname>` to give you read-only SSH access to their servers."
			} else if !a.cfg.HasSandboxHosts() {
				tools = llm.GetSourceOnlyTools()
				systemPrompt += "\n\nYou have read-only SSH access to the user's servers. Use run_source_command and read_source_file to diagnose issues. You CANNOT modify anything on source hosts.\n\nWhen you identify a fix or change:\n1. Explain the diagnosis and proposed fix\n2. Say: \"This is a fix I could test in a sandbox and generate a playbook for. Set up a daemon host (https://deer.sh/docs/daemon) then use /connect to link it.\"" +
					tlsDebuggingGuidance
			} else if isReadOnly {
				tools = llm.GetReadOnlyTools()
				systemPrompt += "\n\nYou are in READ-ONLY mode. You can only query and observe - you cannot create, modify, or destroy any resources."
			}

			// Add TLS debugging guidance when the agent has source host access AND sandbox hosts.
			// This is mutually exclusive with the branch above that appends tlsDebuggingGuidance
			// when !HasSandboxHosts (source-only mode) - so the guidance is never appended twice.
			if len(a.cfg.PreparedHosts()) > 0 && a.cfg.HasSandboxHosts() && !isReadOnly {
				systemPrompt += tlsDebuggingGuidance
			}

			// Inject skills catalog into system prompt so the LLM knows what's available.
			if a.skillLoader != nil && a.skillLoader.HasSkills() {
				systemPrompt += "\n\n## Available Skills\n\n" +
					"You have access to domain-specific skills via the `list_skills` and `load_skill` tools. " +
					"Skills contain detailed procedures, runbooks, and tool-usage guidance for specific technologies.\n\n" +
					"**IMPORTANT**: When a user describes an issue, immediately `load_skill` the most relevant skill BEFORE running any diagnostic commands. " +
					"Skills tell you exactly which commands to run and in what order, preventing wasted iterations.\n\n" +
					"Available skills:\n"
				for _, entry := range a.skillLoader.Catalog() {
					desc := entry.Description
					if desc == "" {
						desc = "(no description)"
					}
					systemPrompt += fmt.Sprintf("- **%s**: %s\n", entry.Name, desc)
				}
				systemPrompt += "\nUse `load_skill` to retrieve the full content of any skill listed above."
			}

			// Inject current task list into system prompt so the LLM stays on track.
			if a.taskList != nil && a.taskList.HasTasks() {
				systemPrompt += "\n\n" + a.taskList.FormatForSystemPrompt()
			}

			// Build messages, applying redaction if enabled
			messages := append([]llm.Message{{
				Role:    llm.RoleSystem,
				Content: systemPrompt,
			}}, a.history...)

			if a.redactor != nil {
				redactedMessages := make([]llm.Message, len(messages))
				for i, msg := range messages {
					redactedMessages[i] = msg
					redactedMessages[i].Content = a.redactor.Redact(msg.Content)
					// Redact tool call arguments
					if len(msg.ToolCalls) > 0 {
						redactedTCs := make([]llm.ToolCall, len(msg.ToolCalls))
						copy(redactedTCs, msg.ToolCalls)
						for j, tc := range redactedTCs {
							redactedTCs[j].Function.Arguments = a.redactor.Redact(tc.Function.Arguments)
						}
						redactedMessages[i].ToolCalls = redactedTCs
					}
				}
				messages = redactedMessages
			}

			req := llm.ChatRequest{
				Messages: messages,
				Tools:    tools,
			}

			if a.telemetry != nil {
				a.telemetry.Track("agent_prompt_sent", map[string]any{
					"message_count": len(req.Messages),
					"provider":      a.cfg.AIAgent.Provider,
					"model":         a.cfg.AIAgent.Model,
				})
			}

			// Log LLM request to audit
			if a.auditLog != nil {
				a.auditLog.LogLLMRequest(len(req.Messages), a.EstimateTokens(), a.cfg.AIAgent.Model)
			}

			resp, err := a.llmClient.Chat(ctx, req)
			if err != nil {
				a.logger.Error("LLM chat failed", "error", err)
				return a.finishRun(AgentErrorMsg{Err: fmt.Errorf("llm chat: %w", err)})
			}

			if len(resp.Choices) == 0 {
				a.logger.Error("LLM returned no choices")
				return a.finishRun(AgentErrorMsg{Err: fmt.Errorf("llm returned no choices")})
			}

			msg := resp.Choices[0].Message

			// Log LLM response to audit
			if a.auditLog != nil {
				a.auditLog.LogLLMResponse(len(msg.Content)/4, len(msg.ToolCalls))
			}

			// Restore redacted tokens in LLM response
			if a.redactor != nil {
				msg.Content = a.redactor.Restore(msg.Content)
				for i, tc := range msg.ToolCalls {
					msg.ToolCalls[i].Function.Arguments = a.redactor.Restore(tc.Function.Arguments)
				}
			}

			if a.chatLog != nil {
				chatTCs := make([]chatlog.ToolCallEntry, 0, len(msg.ToolCalls))
				for _, tc := range msg.ToolCalls {
					var args map[string]any
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
					chatTCs = append(chatTCs, chatlog.ToolCallEntry{
						ID:   tc.ID,
						Name: tc.Function.Name,
						Args: args,
					})
				}
				a.chatLog.LogLLMResponse(msg.Content, a.cfg.AIAgent.Model, chatTCs)
			}

			a.history = append(a.history, msg)

			if len(msg.ToolCalls) > 0 {
				a.logger.Debug("LLM response contains tool calls", "tool_count", len(msg.ToolCalls))
				// Send intermediate response if there's content
				if msg.Content != "" {
					a.sendStatus(AgentResponseMsg{Response: AgentResponse{
						Content: msg.Content,
						Done:    false,
					}})
				}

				// Handle tool calls
				for _, tc := range msg.ToolCalls {
					if ctx.Err() != nil {
						return a.finishRun(AgentCancelledMsg{RunID: currentRunID})
					}
					a.logger.Debug("executing tool call", "tool", tc.Function.Name, "call_id", tc.ID)
					toolStart := time.Now()
					result, err := a.executeTool(ctx, tc)

					var toolResultContent string
					var resultMap map[string]any
					success := true
					errMsg := ""

					if err != nil {
						a.logger.Error("tool execution failed", "tool", tc.Function.Name, "error", err)
						success = false
						errMsg = err.Error()
						toolResultContent = fmt.Sprintf("Error: %v", err)
					} else {
						jsonResult, _ := json.Marshal(result)
						toolResultContent = string(jsonResult)
						// Normalize to map[string]any via JSON round-trip so all slice
						// elements are []any (not []map[string]any etc.) for the TUI renderer.
						_ = json.Unmarshal(jsonResult, &resultMap)
					}

					// Log tool call to audit and chat log
					var toolArgs map[string]any
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &toolArgs)
					if a.auditLog != nil {
						auditArgs := toolArgs
						auditResult := result
						if a.redactor != nil {
							auditArgs = a.redactor.RedactMap(toolArgs)
							auditResult = a.redactor.RedactAny(result)
						}
						a.auditLog.LogToolCall(tc.Function.Name, auditArgs, auditResult, err, time.Since(toolStart).Milliseconds())
					}
					if a.chatLog != nil {
						a.chatLog.LogToolCall(tc.Function.Name, toolArgs, result, err, time.Since(toolStart).Milliseconds())
					}

					// Send tool completion status to TUI
					a.sendStatus(ToolCompleteMsg{
						ToolName: tc.Function.Name,
						Success:  success,
						Result:   resultMap,
						Error:    errMsg,
					})

					a.history = append(a.history, llm.Message{
						Role:       llm.RoleTool,
						Content:    toolResultContent,
						ToolCallID: tc.ID,
						Name:       tc.Function.Name,
					})
				}
				// Continue loop to let LLM process tool results
				continue
			}

			// No more tool calls. Send the final response through statusCallback so
			// ToolCompleteMsg stays ordered ahead of it, then return AgentDoneMsg
			// directly as the only completion signal for this run.
			return a.finishRun(AgentResponseMsg{Response: AgentResponse{
				Content: msg.Content,
				Done:    true,
			}})
		}
	}
}

// RunHeadless runs a single prompt through the agent synchronously and returns
// the final LLM response text. It is the non-interactive equivalent of Run(),
// with no TUI coupling: no slash commands, no sendStatus calls, no tea.Cmd.
// The full session is still written to the chatlog and audit log as normal.
func (a *DeerAgent) RunHeadless(ctx context.Context, input string) (string, error) {
	// Add user message to history.
	a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: input})
	if a.auditLog != nil {
		a.auditLog.LogUserInput(len(input))
	}
	if a.chatLog != nil {
		a.chatLog.LogUserMessage(input)
	}

	if a.llmClient == nil || a.cfg.AIAgent.APIKey == "" {
		return "", fmt.Errorf("LLM provider not configured - set OPENROUTER_API_KEY or configure in settings")
	}

	if a.NeedsCompaction() {
		if _, err := a.Compact(ctx); err != nil {
			a.logger.Warn("auto-compaction failed", "error", err)
		}
	}

	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		systemPrompt := a.cfg.AIAgent.DefaultSystem
		tools := llm.GetTools()

		a.mu.Lock()
		isReadOnly := a.readOnly
		a.mu.Unlock()

		if !a.cfg.HasSandboxHosts() && len(a.cfg.PreparedHosts()) == 0 {
			tools = llm.GetNoSourceTools()
		} else if !a.cfg.HasSandboxHosts() {
			tools = llm.GetSourceOnlyTools()
			systemPrompt += "\n\nYou have read-only SSH access to the user's servers. Use run_source_command and read_source_file to diagnose issues. You CANNOT modify anything on source hosts.\n\nWhen you identify a fix or change:\n1. Explain the diagnosis and proposed fix\n2. Say: \"This is a fix I could test in a sandbox and generate a playbook for. Set up a daemon host (https://deer.sh/docs/daemon) then use /connect to link it.\""
		} else if isReadOnly {
			tools = llm.GetReadOnlyTools()
			systemPrompt += "\n\nYou are in READ-ONLY mode. You can only query and observe - you cannot create, modify, or destroy any resources."
		}

		if len(a.cfg.PreparedHosts()) > 0 && a.cfg.HasSandboxHosts() && !isReadOnly {
			systemPrompt += tlsDebuggingGuidance
		}

		if a.skillLoader != nil && a.skillLoader.HasSkills() {
			systemPrompt += "\n\n## Available Skills\n\n" +
				"You have access to domain-specific skills via the `list_skills` and `load_skill` tools. " +
				"Skills contain detailed procedures, runbooks, and tool-usage guidance for specific technologies.\n\n" +
				"**IMPORTANT**: When a user describes an issue, immediately `load_skill` the most relevant skill BEFORE running any diagnostic commands. " +
				"Skills tell you exactly which commands to run and in what order, preventing wasted iterations.\n\n" +
				"Available skills:\n"
			for _, entry := range a.skillLoader.Catalog() {
				desc := entry.Description
				if desc == "" {
					desc = "(no description)"
				}
				systemPrompt += fmt.Sprintf("- **%s**: %s\n", entry.Name, desc)
			}
			systemPrompt += "\nUse `load_skill` to retrieve the full content of any skill listed above."
		}

		if a.taskList != nil && a.taskList.HasTasks() {
			systemPrompt += "\n\n" + a.taskList.FormatForSystemPrompt()
		}

		messages := append([]llm.Message{{Role: llm.RoleSystem, Content: systemPrompt}}, a.history...)

		if a.redactor != nil {
			redacted := make([]llm.Message, len(messages))
			for i, msg := range messages {
				redacted[i] = msg
				redacted[i].Content = a.redactor.Redact(msg.Content)
				if len(msg.ToolCalls) > 0 {
					tcs := make([]llm.ToolCall, len(msg.ToolCalls))
					copy(tcs, msg.ToolCalls)
					for j, tc := range tcs {
						tcs[j].Function.Arguments = a.redactor.Redact(tc.Function.Arguments)
					}
					redacted[i].ToolCalls = tcs
				}
			}
			messages = redacted
		}

		req := llm.ChatRequest{Messages: messages, Tools: tools}

		if a.auditLog != nil {
			a.auditLog.LogLLMRequest(len(req.Messages), a.EstimateTokens(), a.cfg.AIAgent.Model)
		}

		resp, err := a.llmClient.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("llm chat: %w", err)
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("llm returned no choices")
		}

		msg := resp.Choices[0].Message

		if a.auditLog != nil {
			a.auditLog.LogLLMResponse(len(msg.Content)/4, len(msg.ToolCalls))
		}

		if a.redactor != nil {
			msg.Content = a.redactor.Restore(msg.Content)
			for i, tc := range msg.ToolCalls {
				msg.ToolCalls[i].Function.Arguments = a.redactor.Restore(tc.Function.Arguments)
			}
		}

		if a.chatLog != nil {
			chatTCs := make([]chatlog.ToolCallEntry, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				chatTCs = append(chatTCs, chatlog.ToolCallEntry{ID: tc.ID, Name: tc.Function.Name, Args: args})
			}
			a.chatLog.LogLLMResponse(msg.Content, a.cfg.AIAgent.Model, chatTCs)
		}

		a.history = append(a.history, msg)

		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}

		for _, tc := range msg.ToolCalls {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			toolStart := time.Now()
			result, toolErr := a.executeTool(ctx, tc)

			var toolResultContent string
			if toolErr != nil {
				toolResultContent = fmt.Sprintf("Error: %v", toolErr)
			} else {
				jsonResult, _ := json.Marshal(result)
				toolResultContent = string(jsonResult)
			}

			var toolArgs map[string]any
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &toolArgs)
			if a.auditLog != nil {
				auditArgs := toolArgs
				auditResult := result
				if a.redactor != nil {
					auditArgs = a.redactor.RedactMap(toolArgs)
					auditResult = a.redactor.RedactAny(result)
				}
				a.auditLog.LogToolCall(tc.Function.Name, auditArgs, auditResult, toolErr, time.Since(toolStart).Milliseconds())
			}
			if a.chatLog != nil {
				a.chatLog.LogToolCall(tc.Function.Name, toolArgs, result, toolErr, time.Since(toolStart).Milliseconds())
			}

			a.history = append(a.history, llm.Message{
				Role:       llm.RoleTool,
				Content:    toolResultContent,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}
}

// executeTool dispatches tool calls to internal methods
func (a *DeerAgent) executeTool(ctx context.Context, tc llm.ToolCall) (any, error) {
	// Parse args for status message
	var args map[string]any
	_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

	a.logger.Debug("tool dispatch", "tool", tc.Function.Name, "args", args)

	// Send tool start status
	a.sendStatus(ToolStartMsg{
		ToolName: tc.Function.Name,
		Args:     args,
	})

	if a.telemetry != nil {
		a.telemetry.Track("agent_tool_call", map[string]any{
			"tool_name": tc.Function.Name,
		})
	}

	switch tc.Function.Name {
	case "list_sandboxes":
		return a.listSandboxes(ctx)
	case "create_sandbox":
		a.clearStickyReadOnly()
		var args struct {
			SourceVM                  string `json:"source_vm"`
			Host                      string `json:"host"`
			CPU                       int    `json:"cpu"`
			MemoryMB                  int    `json:"memory_mb"`
			Live                      bool   `json:"live"`
			SimpleKafkaBroker         bool   `json:"kafka_stub"`
			SimpleElasticsearchBroker bool   `json:"es_stub"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.createSandbox(ctx, args.SourceVM, args.Host, args.CPU, args.MemoryMB, args.Live, args.SimpleKafkaBroker, args.SimpleElasticsearchBroker)
	case "destroy_sandbox":
		a.clearStickyReadOnly()
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.destroySandbox(ctx, args.SandboxID)
	case "run_command":
		a.clearStickyReadOnly()
		var args struct {
			SandboxID string `json:"sandbox_id"`
			Command   string `json:"command"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.runCommand(ctx, args.SandboxID, args.Command)
	case "start_sandbox":
		a.clearStickyReadOnly()
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.startSandbox(ctx, args.SandboxID)
	case "stop_sandbox":
		a.clearStickyReadOnly()
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.stopSandbox(ctx, args.SandboxID)
	case "get_sandbox":
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.getSandbox(ctx, args.SandboxID)
	case "list_vms":
		return a.listVMs(ctx)
	case "create_snapshot":
		a.clearStickyReadOnly()
		var args struct {
			SandboxID string `json:"sandbox_id"`
			Name      string `json:"name"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.createSnapshot(ctx, args.SandboxID, args.Name)
	case "create_playbook":
		a.clearStickyReadOnly()
		var args ansible.CreatePlaybookRequest
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.playbookService.CreatePlaybook(ctx, args)
	case "add_playbook_task":
		a.clearStickyReadOnly()
		var args struct {
			PlaybookID string         `json:"playbook_id"`
			Name       string         `json:"name"`
			Module     string         `json:"module"`
			Params     map[string]any `json:"params"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.playbookService.AddTask(ctx, args.PlaybookID, ansible.AddTaskRequest{
			Name:   args.Name,
			Module: args.Module,
			Params: args.Params,
		})
	case "edit_file":
		a.clearStickyReadOnly()
		var args struct {
			SandboxID string `json:"sandbox_id"`
			Path      string `json:"path"`
			OldStr    string `json:"old_str"`
			NewStr    string `json:"new_str"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.editFile(ctx, args.SandboxID, args.Path, args.OldStr, args.NewStr)
	case "read_file":
		var args struct {
			SandboxID string `json:"sandbox_id"`
			Path      string `json:"path"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.readFile(ctx, args.SandboxID, args.Path)
	case "list_playbooks":
		return a.listPlaybooks(ctx)
	case "get_playbook":
		var args struct {
			PlaybookID string `json:"playbook_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.getPlaybook(ctx, args.PlaybookID)
	case "run_source_command":
		var args struct {
			Host    string `json:"host"`
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		if a.sourceService != nil {
			var result *source.CommandResult
			_, cmdErr := a.withAutoReadOnly(args.Host, func() (any, error) {
				var innerErr error
				result, innerErr = a.sourceService.RunCommandStreaming(ctx, args.Host, args.Command,
					func(chunk string, isStderr bool) {
						redacted, _ := a.redactContent(chunk)
						a.sendStatus(CommandOutputChunkMsg{
							SandboxID: args.Host,
							IsStderr:  isStderr,
							Chunk:     redacted,
						})
					})
				return result, innerErr
			})
			if cmdErr != nil {
				return nil, cmdErr
			}
			a.sendStatus(CommandOutputDoneMsg{SandboxID: args.Host})
			stdout, stdoutRedacted := a.redactContent(result.Stdout)
			stderr, stderrRedacted := a.redactContent(result.Stderr)
			if stdoutRedacted || stderrRedacted {
				a.sendRedactedMsg(args.Host, "")
			}
			return map[string]any{
				"host":      args.Host,
				"exit_code": result.ExitCode,
				"stdout":    stdout,
				"stderr":    stderr,
			}, nil
		}
		return a.withAutoReadOnly(args.Host, func() (any, error) {
			return a.runSourceCommand(ctx, args.Host, args.Command)
		})
	case "read_source_file":
		var args struct {
			Host string `json:"host"`
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		if a.sourceService != nil {
			var content string
			_, cmdErr := a.withAutoReadOnly(args.Host, func() (any, error) {
				var innerErr error
				content, innerErr = a.sourceService.ReadFile(ctx, args.Host, args.Path)
				return content, innerErr
			})
			if cmdErr != nil {
				return nil, cmdErr
			}
			content, wasRedacted := a.redactContent(content)
			if wasRedacted {
				a.sendRedactedMsg(args.Host, args.Path)
			}
			a.sendStatus(CommandOutputStartMsg{SandboxID: args.Host})
			a.sendStatus(CommandOutputChunkMsg{
				SandboxID: args.Host,
				Chunk:     content + "\n",
			})
			a.sendStatus(CommandOutputDoneMsg{SandboxID: args.Host})
			return map[string]any{
				"source_vm": args.Host,
				"path":      args.Path,
				"content":   content,
			}, nil
		}
		return a.withAutoReadOnly(args.Host, func() (any, error) {
			return a.readSourceFile(ctx, args.Host, args.Path)
		})
	case "verify_pipeline_output":
		var args struct {
			SandboxID string `json:"sandbox_id"`
			Index     string `json:"index"`
			Query     string `json:"query"`
			Size      int    `json:"size"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.verifyPipelineOutput(ctx, args.SandboxID, args.Index, args.Query, args.Size)
	case "request_source_access":
		var args struct {
			Host    string `json:"host"`
			Command string `json:"command"`
			Reason  string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.handleRequestSourceAccess(ctx, args.Host, args.Command, args.Reason)
	case "list_hosts":
		if a.sourceService != nil {
			hosts := a.sourceService.ListHosts()
			hostList := make([]map[string]any, 0, len(hosts))
			for _, h := range hosts {
				hostList = append(hostList, map[string]any{
					"name":     h.Name,
					"address":  h.Address,
					"prepared": h.Prepared,
				})
			}
			return map[string]any{"hosts": hostList, "count": len(hostList)}, nil
		}
		return a.listHostsWithVMs(ctx)
	case "list_skills":
		return a.handleListSkills()
	case "load_skill":
		var args struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.handleLoadSkill(args.Name)
	case "add_task":
		var args struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.handleAddTask(args.Content)
	case "update_task":
		var args struct {
			TaskID  string     `json:"task_id"`
			Status  TaskStatus `json:"status"`
			Content string     `json:"content"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.handleUpdateTask(args.TaskID, args.Status, args.Content)
	case "delete_task":
		var args struct {
			TaskID string `json:"task_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.handleDeleteTask(args.TaskID)
	case "list_tasks":
		return a.handleListTasks()
	default:
		a.logger.Error("unknown tool name", "tool", tc.Function.Name)
		return nil, fmt.Errorf("unknown tool: %s", tc.Function.Name)
	}
}

// Reset clears the conversation history
func (a *DeerAgent) Reset() {
	a.logger.Debug("conversation reset", "previous_message_count", len(a.history))
	a.history = make([]llm.Message, 0)
	if a.taskList != nil {
		a.taskList.Clear()
	}
}

// EstimateTokens estimates the token count for the current conversation history
func (a *DeerAgent) EstimateTokens() int {
	tokensPerChar := a.cfg.AIAgent.TokensPerChar
	if tokensPerChar <= 0 {
		tokensPerChar = 0.25 // default
	}

	totalChars := 0
	// Include system prompt
	totalChars += len(a.cfg.AIAgent.DefaultSystem)

	// Include all messages
	for _, msg := range a.history {
		totalChars += len(msg.Content)
		// Account for tool calls
		for _, tc := range msg.ToolCalls {
			totalChars += len(tc.Function.Name)
			totalChars += len(tc.Function.Arguments)
		}
	}

	return int(float64(totalChars) * tokensPerChar)
}

// GetContextUsage returns the current context usage as a percentage
func (a *DeerAgent) GetContextUsage() float64 {
	maxTokens := a.cfg.AIAgent.TotalContextTokens
	if maxTokens <= 0 {
		maxTokens = 64000
	}
	return float64(a.EstimateTokens()) / float64(maxTokens)
}

// NeedsCompaction returns true if the context is at or above the compaction threshold
func (a *DeerAgent) NeedsCompaction() bool {
	threshold := a.cfg.AIAgent.CompactThreshold
	if threshold <= 0 {
		threshold = 0.9
	}
	return a.GetContextUsage() >= threshold
}

// Compact summarizes the conversation history using a smaller LLM and resets with the summary
func (a *DeerAgent) Compact(ctx context.Context) (CompactCompleteMsg, error) {
	if len(a.history) == 0 {
		return CompactCompleteMsg{}, fmt.Errorf("no conversation history to compact")
	}

	previousTokens := a.EstimateTokens()
	a.logger.Info("compaction starting", "previous_tokens", previousTokens, "message_count", len(a.history))

	// Build the conversation text for summarization
	var convText strings.Builder
	convText.WriteString("Conversation history to summarize:\n\n")
	for _, msg := range a.history {
		switch msg.Role {
		case llm.RoleUser:
			fmt.Fprintf(&convText, "User: %s\n\n", msg.Content)
		case llm.RoleAssistant:
			if msg.Content != "" {
				fmt.Fprintf(&convText, "Assistant: %s\n\n", msg.Content)
			}
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&convText, "Assistant called tool: %s(%s)\n\n", tc.Function.Name, tc.Function.Arguments)
			}
		case llm.RoleTool:
			fmt.Fprintf(&convText, "Tool result (%s): %s\n\n", msg.Name, msg.Content)
		}
	}

	// Create compaction request
	compactPrompt := `Your task is to create a detailed summary of the conversation so far, paying close attention to the user's explicit requests and your previous actions. This summary will be used as context when continuing the conversation, so preserve critical information including:
- What was accomplished
- Current work in progress
- Files involved
- Next steps
- Key user requests or constraints

` + convText.String()

	// Use compact model or fall back to main model
	compactModel := a.cfg.AIAgent.CompactModel
	if compactModel == "" {
		compactModel = a.cfg.AIAgent.Model
	}

	req := llm.ChatRequest{
		Model: compactModel,
		Messages: []llm.Message{
			{
				Role:    llm.RoleUser,
				Content: compactPrompt,
			},
		},
	}

	resp, err := a.llmClient.Chat(ctx, req)
	if err != nil {
		a.logger.Error("compaction LLM call failed", "error", err)
		return CompactCompleteMsg{}, fmt.Errorf("compaction LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		a.logger.Error("compaction LLM returned no choices")
		return CompactCompleteMsg{}, fmt.Errorf("compaction LLM returned no choices")
	}

	summary := resp.Choices[0].Message.Content

	// Reset history with the summary as initial context
	a.history = []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: "[Context from previous conversation]\n" + summary,
		},
		{
			Role:    llm.RoleAssistant,
			Content: "I understand the context from our previous conversation. I'm ready to continue helping you. What would you like to do next?",
		},
	}

	newTokens := a.EstimateTokens()
	a.logger.Info("compaction complete", "previous_tokens", previousTokens, "new_tokens", newTokens)

	return CompactCompleteMsg{
		PreviousTokens: previousTokens,
		NewTokens:      newTokens,
		Summary:        summary,
	}, nil
}

// RunCompact executes the compaction as a tea.Cmd
func (a *DeerAgent) RunCompact() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		a.sendStatus(CompactStartMsg{})

		result, err := a.Compact(ctx)
		if err != nil {
			return CompactErrorMsg{Err: err}
		}

		return result
	}
}

// Command implementations

func (a *DeerAgent) listSandboxes(ctx context.Context) (map[string]any, error) {
	sandboxes, err := a.service.ListSandboxes(ctx)
	if err != nil {
		a.logger.Error("list sandboxes query failed", "error", err)
		return nil, err
	}

	a.logger.Debug("list sandboxes", "count", len(sandboxes))
	result := make([]map[string]any, 0, len(sandboxes))
	for _, sb := range sandboxes {
		item := map[string]any{
			"id":         sb.ID,
			"name":       sb.Name,
			"state":      sb.State,
			"base_image": sb.BaseImage,
			"created_at": sb.CreatedAt.Format(time.RFC3339),
		}
		if sb.IPAddress != "" {
			item["ip_address"] = sb.IPAddress
		}
		result = append(result, item)
	}

	return map[string]any{
		"sandboxes": result,
		"count":     len(result),
	}, nil
}

func normalizeVMName(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func (a *DeerAgent) createSandbox(ctx context.Context, sourceVM, hostName string, cpu, memoryMB int, live bool, simpleKafkaBroker bool, simpleElasticsearchBroker bool) (map[string]any, error) {
	if sourceVM == "" {
		return nil, fmt.Errorf("source-vm is required - call list_vms first to see available VM images for cloning")
	}

	// Validate the source VM exists before attempting creation.
	vms, err := a.service.ListVMs(ctx)
	if err == nil {
		found := false
		names := make([]string, 0, len(vms))
		resolvedName := sourceVM
		for _, v := range vms {
			names = append(names, v.Name)
			if v.Name == sourceVM {
				found = true
				resolvedName = v.Name
			} else if normalizeVMName(v.Name) == normalizeVMName(sourceVM) {
				found = true
				resolvedName = v.Name
			}
		}
		if !found {
			return nil, fmt.Errorf("source VM %q not found - call list_vms to see available VM images for cloning. Available: %s", sourceVM, strings.Join(names, ", "))
		}
		sourceVM = resolvedName
	}

	a.logger.Info("sandbox creation attempt", "source_vm", sourceVM, "cpu", cpu, "memory_mb", memoryMB, "live", live, "kafka_stub", simpleKafkaBroker, "es_stub", simpleElasticsearchBroker)
	lastStepNum := 0
	lastTotal := 0

	sb, err := a.service.CreateSandboxStream(ctx, sandbox.CreateRequest{
		SourceVM:                  sourceVM,
		AgentID:                   "tui-agent",
		VCPUs:                     cpu,
		MemoryMB:                  memoryMB,
		Live:                      live,
		SimpleKafkaBroker:         simpleKafkaBroker,
		SimpleElasticsearchBroker: simpleElasticsearchBroker,
	}, func(step string, stepNum, total int) {
		lastStepNum = stepNum
		lastTotal = total
		a.sendStatus(SandboxCreateProgressMsg{
			SourceVM: sourceVM,
			StepName: step,
			StepNum:  stepNum,
			Total:    total,
		})
	})
	if err != nil {
		a.sendStatus(SandboxCreateProgressMsg{Done: true, SourceVM: sourceVM})
		a.logger.Error("sandbox creation failed", "source_vm", sourceVM, "error", err)
		return nil, err
	}
	a.logger.Info("sandbox created", "sandbox_id", sb.ID, "ip", sb.IPAddress)

	doneMsg := SandboxCreateProgressMsg{
		SourceVM: sourceVM,
		Done:     true,
	}
	if lastTotal > 0 {
		doneMsg.StepName = "Ready"
		doneMsg.StepNum = lastTotal
		doneMsg.Total = lastTotal
		if lastStepNum > lastTotal {
			doneMsg.StepNum = lastStepNum
			doneMsg.Total = lastStepNum
		}
	}
	a.sendStatus(doneMsg)

	// Track the created sandbox for cleanup on exit
	a.createdSandboxes = append(a.createdSandboxes, sb.ID)

	// Set as current sandbox for status bar display
	a.currentSandboxID = sb.ID
	a.currentSandboxBaseImage = sb.BaseImage

	result := map[string]any{
		"sandbox_id": sb.ID,
		"name":       sb.Name,
		"state":      sb.State,
	}
	if sb.IPAddress != "" {
		result["ip"] = sb.IPAddress
	}

	return result, nil
}

// HandleApprovalResponse handles the response from the memory approval dialog
func (a *DeerAgent) HandleApprovalResponse(approved bool) {
	// No-op in remote mode - daemon handles resource checking
	a.logger.Debug("memory approval response (no-op in remote mode)", "approved", approved)
}

// HandleNetworkApprovalResponse handles the response from the network approval dialog
func (a *DeerAgent) HandleNetworkApprovalResponse(approved bool) {
	a.logger.Info("network approval response", "approved", approved)
	if a.pendingNetworkApproval != nil && a.pendingNetworkApproval.ResponseChan != nil {
		a.pendingNetworkApproval.ResponseChan <- approved
	}
}

// HandleSourceAccessResponse handles the response from the source command elevation dialog
func (a *DeerAgent) HandleSourceAccessResponse(result SourceAccessApprovalResult) {
	a.logger.Info("source access response", "approved", result.Approved, "session", result.Session)
	if a.pendingSourceAccess != nil && a.pendingSourceAccess.ResponseChan != nil {
		a.pendingSourceAccess.ResponseChan <- result
	}
}

// isSessionElevated checks if a command has been approved for the session on a given host.
func (a *DeerAgent) isSessionElevated(host, command string) bool {
	if a.sessionElevatedCommands == nil {
		return false
	}
	cmds, ok := a.sessionElevatedCommands[host]
	if !ok {
		return false
	}
	return cmds[command]
}

// addSessionElevated records a command as approved for the rest of the session.
func (a *DeerAgent) addSessionElevated(host, command string) {
	if a.sessionElevatedCommands == nil {
		a.sessionElevatedCommands = make(map[string]map[string]bool)
	}
	if _, ok := a.sessionElevatedCommands[host]; !ok {
		a.sessionElevatedCommands[host] = make(map[string]bool)
	}
	a.sessionElevatedCommands[host][command] = true
}

// handleRequestSourceAccess handles the request_source_access tool.
// It prompts the human for approval and, if granted, executes the command
// with validation bypassed.
func (a *DeerAgent) handleRequestSourceAccess(ctx context.Context, host, command, reason string) (map[string]any, error) {
	if a.sourceService == nil {
		return nil, fmt.Errorf("no source service configured")
	}
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}
	if reason == "" {
		return nil, fmt.Errorf("reason is required - explain why you need this command")
	}

	// Check session cache first
	if a.isSessionElevated(host, command) {
		a.logger.Info("using session-elevated command", "host", host, "command", command)
		result, err := a.sourceService.RunCommandElevated(ctx, host, command)
		if err != nil {
			return nil, err
		}
		stdout, stdoutRedacted := a.redactContent(result.Stdout)
		stderr, stderrRedacted := a.redactContent(result.Stderr)
		if stdoutRedacted || stderrRedacted {
			a.sendRedactedMsg(host, "")
		}
		return map[string]any{
			"host":      host,
			"exit_code": result.ExitCode,
			"stdout":    stdout,
			"stderr":    stderr,
			"elevated":  true,
		}, nil
	}

	// Send approval request to TUI and block
	request := SourceAccessApprovalRequest{
		Host:    host,
		Command: command,
		Reason:  reason,
	}
	responseChan := make(chan SourceAccessApprovalResult, 1)
	a.pendingSourceAccess = &PendingSourceAccess{
		Request:      request,
		ResponseChan: responseChan,
	}
	a.sendStatus(SourceAccessApprovalRequestMsg{Request: request})

	var result SourceAccessApprovalResult
	select {
	case result = <-responseChan:
	case <-ctx.Done():
		a.pendingSourceAccess = nil
		return map[string]any{
			"host":      host,
			"error":     "elevation request cancelled: context deadline exceeded",
			"exit_code": -1,
		}, nil
	}
	a.pendingSourceAccess = nil

	if !result.Approved {
		return map[string]any{
			"host":      host,
			"error":     "command elevation denied by user",
			"exit_code": -1,
		}, nil
	}

	// Cache for session if requested
	if result.Session {
		a.addSessionElevated(host, command)
	}

	// Execute the elevated command
	cmdResult, err := a.sourceService.RunCommandElevated(ctx, host, command)
	if err != nil {
		return nil, err
	}
	stdout, stdoutRedacted := a.redactContent(cmdResult.Stdout)
	stderr, stderrRedacted := a.redactContent(cmdResult.Stderr)
	if stdoutRedacted || stderrRedacted {
		a.sendRedactedMsg(host, "")
	}
	return map[string]any{
		"host":      host,
		"exit_code": cmdResult.ExitCode,
		"stdout":    stdout,
		"stderr":    stderr,
		"elevated":  true,
	}, nil
}

// HandleSourcePrepareApprovalResponse handles the response from the source prepare approval dialog
func (a *DeerAgent) HandleSourcePrepareApprovalResponse(approved bool) {
	// No-op in remote mode - daemon handles source VM preparation
	a.logger.Debug("source prepare approval response (no-op in remote mode)", "approved", approved)
}

// runPrepareInline runs source host preparation inline in the TUI, sending progress via SourcePrepareProgressMsg.
func (a *DeerAgent) runPrepareInline(ctx context.Context, hostname string) tea.Msg {
	identityPubKey := config.DaemonIdentityPubKey(a.cfg.SandboxHosts)
	totalSteps := 4
	if identityPubKey != "" {
		totalSteps = 5
	}

	// 1. Resolve SSH config
	a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Resolving SSH config", StepNum: 1, Total: totalSteps})
	resolved, err := sshconfig.Resolve(hostname)
	if err != nil {
		return a.finishRun(AgentResponseMsg{Response: AgentResponse{
			Content: fmt.Sprintf("Failed to resolve SSH config for %s: %v", hostname, err),
			Done:    true,
		}})
	}
	a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Resolving SSH config", StepNum: 1, Total: totalSteps, Done: true})

	// 2. Ensure SSH key pair
	a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Generating SSH key pair", StepNum: 2, Total: totalSteps})
	_, pubKey, err := sourcekeys.EnsureKeyPair(a.cfg.SSH.SourceKeyDir)
	if err != nil {
		return a.finishRun(AgentResponseMsg{Response: AgentResponse{
			Content: fmt.Sprintf("Failed to generate key pair: %v", err),
			Done:    true,
		}})
	}
	a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Generating SSH key pair", StepNum: 2, Total: totalSteps, Done: true})

	// 3. Prepare host for read-only access
	a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Preparing host", StepNum: 3, Total: totalSteps})
	sshRunFn := hostexec.NewSSHAlias(hostname)
	sshRun := readonly.SSHRunFunc(sshRunFn)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	_, err = readonly.PrepareWithKey(ctx, sshRun, pubKey, nil, logger)
	if err != nil {
		return a.finishRun(AgentResponseMsg{Response: AgentResponse{
			Content: fmt.Sprintf("Preparation failed for %s: %v", hostname, err),
			Done:    true,
		}})
	}
	a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Preparing host", StepNum: 3, Total: totalSteps, Done: true})

	a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Saving config", StepNum: 4, Total: totalSteps})
	configPath, _ := paths.ConfigFile()
	if err := source.SavePreparedHost(a.cfg, configPath, hostname, resolved); err != nil {
		a.logger.Warn("failed to save config after prepare", "error", err)
	}
	a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Saving config", StepNum: 4, Total: totalSteps, Done: true})

	// 5. Deploy daemon identity key if available
	if identityPubKey != "" {
		a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Deploying daemon SSH key", StepNum: 5, Total: totalSteps})
		deployErr := readonly.DeployDaemonKey(ctx, sshRun, identityPubKey, logger)
		if deployErr != nil {
			a.logger.Warn("daemon key deploy failed (non-fatal)", "host", hostname, "error", deployErr)
		}
		a.sendStatus(SourcePrepareProgressMsg{SourceVM: hostname, StepName: "Deploying daemon SSH key", StepNum: 5, Total: totalSteps, Done: true})
	}

	return a.finishRun(AgentResponseMsg{Response: AgentResponse{
		Content: fmt.Sprintf("Host %s is prepared.", hostname),
		Done:    true,
	}})
}

func (a *DeerAgent) destroySandbox(ctx context.Context, id string) (map[string]any, error) {
	err := a.service.DestroySandbox(ctx, id)
	if err != nil {
		a.logger.Error("destroy sandbox failed", "sandbox_id", id, "error", err)
		return nil, err
	}
	a.logger.Info("sandbox destroyed", "sandbox_id", id)

	// Clear current sandbox if this was the one being destroyed
	if id == a.currentSandboxID {
		a.currentSandboxID = ""
		a.currentSandboxHost = ""
		a.currentSandboxBaseImage = ""
	}

	return map[string]any{
		"destroyed":  true,
		"sandbox_id": id,
	}, nil
}

func (a *DeerAgent) runCommand(ctx context.Context, sandboxID, command string) (map[string]any, error) {
	truncCmd := command
	if len(truncCmd) > 120 {
		truncCmd = truncCmd[:120] + "..."
	}
	a.logger.Debug("run command", "sandbox_id", sandboxID, "command", truncCmd)

	// Update current sandbox if different (user is working with this sandbox)
	if sandboxID != "" && sandboxID != a.currentSandboxID {
		a.currentSandboxID = sandboxID
		if sb, err := a.service.GetSandbox(ctx, sandboxID); err == nil {
			a.currentSandboxBaseImage = sb.BaseImage
		} else {
			a.currentSandboxBaseImage = ""
		}
	}

	// Check if command requires network access and request approval
	networkTool, urls := detectNetworkAccess(command)
	if networkTool != "" {
		a.logger.Warn("network access detected, requesting approval", "tool", networkTool, "urls", urls, "sandbox_id", sandboxID)
		request := NetworkApprovalRequest{
			Command:     command,
			SandboxID:   sandboxID,
			NetworkTool: networkTool,
			URLs:        urls,
		}

		responseChan := make(chan bool, 1)
		a.pendingNetworkApproval = &PendingNetworkApproval{
			Request:      request,
			ResponseChan: responseChan,
		}
		a.sendStatus(NetworkApprovalRequestMsg{Request: request})

		var approved bool
		select {
		case approved = <-responseChan:
		case <-ctx.Done():
			a.pendingNetworkApproval = nil
			return map[string]any{
				"sandbox_id": sandboxID,
				"error":      "network approval cancelled: context deadline exceeded",
				"exit_code":  -1,
			}, nil
		}
		a.pendingNetworkApproval = nil
		a.logger.Info("network approval result", "approved", approved, "tool", networkTool, "sandbox_id", sandboxID)

		if !approved {
			return map[string]any{
				"sandbox_id": sandboxID,
				"error":      "network access denied by user",
				"exit_code":  -1,
			}, nil
		}
	}

	a.sendStatus(CommandOutputStartMsg{SandboxID: sandboxID})

	result, err := a.service.RunCommand(ctx, sandboxID, command, 0, nil)
	if err != nil {
		a.logger.Error("command execution failed", "sandbox_id", sandboxID, "error", err)
		a.sendStatus(CommandOutputDoneMsg{SandboxID: sandboxID})
		if result != nil {
			stdout, stdoutRedacted := a.redactContent(result.Stdout)
			stderr, stderrRedacted := a.redactContent(result.Stderr)
			if stdoutRedacted || stderrRedacted {
				a.sendRedactedMsg(sandboxID, "")
			}
			return map[string]any{
				"sandbox_id": sandboxID,
				"exit_code":  result.ExitCode,
				"stdout":     stdout,
				"stderr":     stderr,
				"error":      err.Error(),
			}, nil
		}
		return nil, err
	}

	stdout, stdoutRedacted := a.redactContent(result.Stdout)
	stderr, stderrRedacted := a.redactContent(result.Stderr)
	if stdoutRedacted || stderrRedacted {
		a.sendRedactedMsg(sandboxID, "")
	}

	// Show output in live output box
	if stdout != "" {
		a.sendStatus(CommandOutputChunkMsg{SandboxID: sandboxID, Chunk: stdout})
	}
	if stderr != "" {
		a.sendStatus(CommandOutputChunkMsg{SandboxID: sandboxID, IsStderr: true, Chunk: stderr})
	}
	a.sendStatus(CommandOutputDoneMsg{SandboxID: sandboxID})

	return map[string]any{
		"sandbox_id": sandboxID,
		"exit_code":  result.ExitCode,
		"stdout":     stdout,
		"stderr":     stderr,
	}, nil
}

// detectNetworkAccess checks if a command uses network tools and extracts URLs
// Returns the network tool name (empty if none) and any URLs found
func detectNetworkAccess(command string) (string, []string) {
	// Network tools that require approval
	networkTools := []string{"curl", "wget", "nc", "netcat", "ssh", "scp", "rsync", "ftp", "sftp", "telnet", "nmap", "ping"}

	cmdLower := strings.ToLower(command)
	var detectedTool string

	for _, tool := range networkTools {
		// Check if the tool appears as a command (not part of another word)
		// Look for tool at start, after pipe, after &&, after ;, or after whitespace
		patterns := []string{
			tool + " ",   // tool at start or after space
			"|" + tool,   // after pipe
			"| " + tool,  // after pipe with space
			"&&" + tool,  // after &&
			"&& " + tool, // after && with space
			";" + tool,   // after ;
			"; " + tool,  // after ; with space
			"$(" + tool,  // in subshell
			"`" + tool,   // in backticks
		}

		for _, pattern := range patterns {
			if strings.Contains(cmdLower, pattern) || strings.HasPrefix(cmdLower, tool+" ") || cmdLower == tool {
				detectedTool = tool
				break
			}
		}
		if detectedTool != "" {
			break
		}
	}

	if detectedTool == "" {
		return "", nil
	}

	// Extract URLs from the command
	var urls []string
	// Simple URL pattern matching
	words := strings.Fields(command)
	for _, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") ||
			strings.HasPrefix(word, "ftp://") || strings.HasPrefix(word, "sftp://") {
			urls = append(urls, word)
		}
	}

	return detectedTool, urls
}

// editFile edits a file on a sandbox by replacing old_str with new_str, or creates the file if old_str is empty.
// This operates on files inside the sandbox VM via SSH.
func (a *DeerAgent) editFile(ctx context.Context, sandboxID, path, oldStr, newStr string) (map[string]any, error) {
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required - this tool operates on files inside a sandbox VM")
	}

	// Resolve to absolute path if not already
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	// If old_str is empty, create/overwrite the file
	if oldStr == "" {
		a.logger.Debug("creating file", "sandbox_id", sandboxID, "path", path)
		encoded := base64.StdEncoding.EncodeToString([]byte(newStr))
		cmd := fmt.Sprintf("echo '%s' | base64 -d | sudo tee '%s' > /dev/null", encoded, path)

		result, err := a.service.RunCommand(ctx, sandboxID, cmd, 0, nil)
		if err != nil {
			a.logger.Error("failed to create file", "sandbox_id", sandboxID, "path", path, "error", err)
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
		if result.ExitCode != 0 {
			a.logger.Error("failed to create file", "sandbox_id", sandboxID, "path", path, "stderr", result.Stderr)
			return nil, fmt.Errorf("failed to create file: %s", result.Stderr)
		}
		a.sendStatus(CommandOutputStartMsg{SandboxID: sandboxID})
		a.sendStatus(CommandOutputChunkMsg{SandboxID: sandboxID, Chunk: fmt.Sprintf("Created %s\n", path)})
		a.sendStatus(CommandOutputDoneMsg{SandboxID: sandboxID})
		return map[string]any{
			"sandbox_id": sandboxID,
			"path":       path,
			"action":     "created_file",
		}, nil
	}

	a.logger.Debug("editing file", "sandbox_id", sandboxID, "path", path)
	// Read the original file using base64 to handle binary/special chars
	readResult, err := a.service.RunCommand(ctx, sandboxID, fmt.Sprintf("sudo base64 '%s'", path), 0, nil)
	if err != nil {
		a.logger.Error("failed to read file for edit", "sandbox_id", sandboxID, "path", path, "error", err)
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if readResult.ExitCode != 0 {
		a.logger.Error("failed to read file for edit", "sandbox_id", sandboxID, "path", path, "stderr", readResult.Stderr)
		return nil, fmt.Errorf("failed to read file: %s", readResult.Stderr)
	}

	// Decode the base64 content
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(readResult.Stdout))
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}
	original := string(decoded)

	// Check if old_str exists
	if !strings.Contains(original, oldStr) {
		return map[string]any{
			"sandbox_id": sandboxID,
			"path":       path,
			"action":     "old_str_not_found",
		}, nil
	}

	// Replace first occurrence only
	edited := strings.Replace(original, oldStr, newStr, 1)

	// Write the edited content back using base64
	encoded := base64.StdEncoding.EncodeToString([]byte(edited))
	writeCmd := fmt.Sprintf("echo '%s' | base64 -d | sudo tee '%s' > /dev/null", encoded, path)

	writeResult, err := a.service.RunCommand(ctx, sandboxID, writeCmd, 0, nil)
	if err != nil {
		a.logger.Error("failed to write file", "sandbox_id", sandboxID, "path", path, "error", err)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	if writeResult.ExitCode != 0 {
		a.logger.Error("failed to write file", "sandbox_id", sandboxID, "path", path, "stderr", writeResult.Stderr)
		return nil, fmt.Errorf("failed to write file: %s", writeResult.Stderr)
	}

	a.sendStatus(CommandOutputStartMsg{SandboxID: sandboxID})
	a.sendStatus(CommandOutputChunkMsg{SandboxID: sandboxID, Chunk: fmt.Sprintf("Edited %s\n", path)})
	a.sendStatus(CommandOutputDoneMsg{SandboxID: sandboxID})

	return map[string]any{
		"sandbox_id": sandboxID,
		"path":       path,
		"action":     "edited",
	}, nil
}

// redactContent runs the Redactor on content and returns whether any redaction occurred.
// If the redactor is nil (redaction disabled), content passes through unchanged.
func (a *DeerAgent) redactContent(content string) (string, bool) {
	if a.redactor == nil {
		return content, false
	}
	result := a.redactor.Redact(content)
	return result, result != content
}

// readFile reads the contents of a file on a sandbox VM via SSH.
// This operates on files inside the sandbox - not local files or playbooks.
func (a *DeerAgent) readFile(ctx context.Context, sandboxID, path string) (map[string]any, error) {
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required - this tool operates on files inside a sandbox VM. For playbooks, use get_playbook instead")
	}

	// Resolve to absolute path if not already
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	a.logger.Debug("read file", "sandbox_id", sandboxID, "path", path)
	// Use base64 to safely transfer content that may contain special characters
	result, err := a.service.RunCommand(ctx, sandboxID, fmt.Sprintf("base64 '%s'", path), 0, nil)
	if err != nil {
		a.logger.Error("failed to read file", "sandbox_id", sandboxID, "path", path, "error", err)
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if result.ExitCode != 0 {
		a.logger.Error("failed to read file", "sandbox_id", sandboxID, "path", path, "stderr", result.Stderr)
		return nil, fmt.Errorf("failed to read file: %s", result.Stderr)
	}

	// Decode the base64 content
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(result.Stdout))
	if err != nil {
		a.logger.Error("failed to decode file content", "sandbox_id", sandboxID, "path", path, "error", err)
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	content := string(decoded)
	content, wasRedacted := a.redactContent(content)
	if wasRedacted {
		a.sendRedactedMsg(sandboxID, path)
	}

	// Show file content in live output box
	a.sendStatus(CommandOutputStartMsg{SandboxID: sandboxID})
	a.sendStatus(CommandOutputChunkMsg{SandboxID: sandboxID, Chunk: content + "\n"})
	a.sendStatus(CommandOutputDoneMsg{SandboxID: sandboxID})

	return map[string]any{
		"sandbox_id": sandboxID,
		"path":       path,
		"content":    content,
	}, nil
}

func (a *DeerAgent) verifyPipelineOutput(ctx context.Context, sandboxID, index, query string, size int) (map[string]any, error) {
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}
	if index == "" {
		index = "_all"
	}
	if size <= 0 {
		size = 10
	}

	esURL := fmt.Sprintf("http://localhost:9200/%s/_search?size=%d", index, size)
	if query != "" {
		esURL += "&q=" + query
	}

	curlCmd := fmt.Sprintf("curl -sf '%s'", esURL)
	result, err := a.service.RunCommand(ctx, sandboxID, curlCmd, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch query failed: %w", err)
	}

	var esResp map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &esResp); err != nil {
		return map[string]any{
			"sandbox_id": sandboxID,
			"raw_output": result.Stdout,
			"error":      "failed to parse elasticsearch response",
		}, nil
	}

	hits, _ := esResp["hits"].(map[string]any)
	total := float64(0)
	if hits != nil {
		if t, ok := hits["total"].(float64); ok {
			total = t
		}
	}

	return map[string]any{
		"sandbox_id":   sandboxID,
		"total_hits":   int(total),
		"index":        index,
		"raw_response": result.Stdout,
	}, nil
}

// getPlaybook retrieves a playbook's full definition including YAML content and tasks.
// This is the correct way to view playbook definitions - not read_file.
func (a *DeerAgent) getPlaybook(ctx context.Context, playbookID string) (map[string]any, error) {
	if playbookID == "" {
		return nil, fmt.Errorf("playbook_id is required")
	}

	// Get playbook with tasks
	pbWithTasks, err := a.playbookService.GetPlaybookWithTasks(ctx, playbookID)
	if err != nil {
		a.logger.Error("failed to get playbook", "playbook_id", playbookID, "error", err)
		return nil, fmt.Errorf("failed to get playbook: %w", err)
	}

	// Export the YAML content
	yamlContent, err := a.playbookService.ExportPlaybook(ctx, playbookID)
	if err != nil {
		a.logger.Error("failed to export playbook", "playbook_id", playbookID, "error", err)
		return nil, fmt.Errorf("failed to export playbook: %w", err)
	}

	// Build task list
	tasks := make([]map[string]any, 0, len(pbWithTasks.Tasks))
	for _, t := range pbWithTasks.Tasks {
		tasks = append(tasks, map[string]any{
			"id":       t.ID,
			"position": t.Position,
			"name":     t.Name,
			"module":   t.Module,
			"params":   t.Params,
		})
	}

	result := map[string]any{
		"id":           pbWithTasks.Playbook.ID,
		"name":         pbWithTasks.Playbook.Name,
		"hosts":        pbWithTasks.Playbook.Hosts,
		"become":       pbWithTasks.Playbook.Become,
		"tasks":        tasks,
		"yaml_content": string(yamlContent),
		"created_at":   pbWithTasks.Playbook.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if pbWithTasks.Playbook.FilePath != nil {
		result["file_path"] = *pbWithTasks.Playbook.FilePath
	}

	return result, nil
}

func (a *DeerAgent) startSandbox(ctx context.Context, id string) (map[string]any, error) {
	sb, err := a.service.StartSandbox(ctx, id)
	if err != nil {
		a.logger.Error("start sandbox failed", "sandbox_id", id, "error", err)
		return nil, err
	}
	a.logger.Info("sandbox started", "sandbox_id", id, "ip", sb.IPAddress)

	result := map[string]any{
		"started":    true,
		"sandbox_id": id,
	}
	if sb.IPAddress != "" {
		result["ip"] = sb.IPAddress
	}

	return result, nil
}

func (a *DeerAgent) stopSandbox(ctx context.Context, id string) (map[string]any, error) {
	err := a.service.StopSandbox(ctx, id, false)
	if err != nil {
		a.logger.Error("stop sandbox failed", "sandbox_id", id, "error", err)
		return nil, err
	}
	a.logger.Info("sandbox stopped", "sandbox_id", id)

	return map[string]any{
		"stopped":    true,
		"sandbox_id": id,
	}, nil
}

func (a *DeerAgent) getSandbox(ctx context.Context, id string) (map[string]any, error) {
	sb, err := a.service.GetSandbox(ctx, id)
	if err != nil {
		a.logger.Error("get sandbox failed", "sandbox_id", id, "error", err)
		return nil, err
	}

	result := map[string]any{
		"sandbox_id": sb.ID,
		"name":       sb.Name,
		"state":      sb.State,
		"base_image": sb.BaseImage,
		"agent_id":   sb.AgentID,
		"created_at": sb.CreatedAt.Format(time.RFC3339),
	}
	if sb.IPAddress != "" {
		result["ip"] = sb.IPAddress
	}

	return result, nil
}

func (a *DeerAgent) listVMs(ctx context.Context) (map[string]any, error) {
	vms, err := a.service.ListVMs(ctx)
	if err != nil {
		a.logger.Error("list VMs failed", "error", err)
		return nil, err
	}

	result := make([]map[string]any, 0, len(vms))
	for _, v := range vms {
		item := map[string]any{
			"name":     v.Name,
			"state":    v.State,
			"prepared": v.Prepared,
		}
		if v.IPAddress != "" {
			item["ip"] = v.IPAddress
		}
		result = append(result, item)
	}

	return map[string]any{
		"vms":   result,
		"count": len(result),
	}, nil
}

func (a *DeerAgent) createSnapshot(ctx context.Context, sandboxID, name string) (map[string]any, error) {
	if name == "" {
		name = fmt.Sprintf("snap-%d", time.Now().Unix())
	}

	snap, err := a.service.CreateSnapshot(ctx, sandboxID, name)
	if err != nil {
		a.logger.Error("create snapshot failed", "sandbox_id", sandboxID, "name", name, "error", err)
		return nil, err
	}
	a.logger.Info("snapshot created", "sandbox_id", sandboxID, "snapshot_id", snap.SnapshotID, "name", snap.SnapshotName)

	return map[string]any{
		"snapshot_id": snap.SnapshotID,
		"sandbox_id":  sandboxID,
		"name":        snap.SnapshotName,
	}, nil
}

// Formatting helpers

// formatVMsResult - use list_hosts instead
// func (a *DeerAgent) formatVMsResult(result map[string]any, err error) string {
// 	if err != nil {
// 		return fmt.Sprintf("Failed to list VMs: %v", err)
// 	}
//
// 	vms, ok := result["vms"].([]map[string]any)
// 	if !ok || len(vms) == 0 {
// 		return "No VMs found."
// 	}
//
// 	var b strings.Builder
// 	b.WriteString(fmt.Sprintf("Found %d VM(s) available for cloning:\n\n", len(vms)))
//
// 	// Group VMs by host if host information is present
// 	hostVMs := make(map[string][]map[string]any)
// 	for _, vm := range vms {
// 		host := "local"
// 		if h, ok := vm["host"].(string); ok && h != "" {
// 			host = h
// 		}
// 		hostVMs[host] = append(hostVMs[host], vm)
// 	}
//
// 	// Display VMs grouped by host
// 	for host, hvms := range hostVMs {
// 		if len(hostVMs) > 1 || host != "local" {
// 			b.WriteString(fmt.Sprintf("### Host: %s\n", host))
// 		}
// 		for _, vm := range hvms {
// 			state := "unknown"
// 			if s, ok := vm["state"].(string); ok {
// 				state = s
// 			}
// 			b.WriteString(fmt.Sprintf("- **%s** (%s)\n", vm["name"], state))
// 		}
// 		b.WriteString("\n")
// 	}
//
// 	// Display any host errors
// 	if hostErrors, ok := result["host_errors"].([]map[string]any); ok && len(hostErrors) > 0 {
// 		b.WriteString("### Host Errors\n")
// 		for _, he := range hostErrors {
// 			b.WriteString(fmt.Sprintf("- **%s**: %s\n", he["host"], he["error"]))
// 		}
// 	}
//
// 	return b.String()
// }

func (a *DeerAgent) formatSandboxesResult(result map[string]any, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list sandboxes: %v", err)
	}

	sandboxes, ok := result["sandboxes"].([]map[string]any)
	if !ok || len(sandboxes) == 0 {
		return "No sandboxes found."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d sandbox(es):\n\n", len(sandboxes))

	// Group sandboxes by host if host information is present
	hostSandboxes := make(map[string][]map[string]any)
	for _, sb := range sandboxes {
		host := "local"
		if h, ok := sb["host"].(string); ok && h != "" {
			host = h
		}
		hostSandboxes[host] = append(hostSandboxes[host], sb)
	}

	// Display sandboxes grouped by host
	for host, sbs := range hostSandboxes {
		if len(hostSandboxes) > 1 || host != "local" {
			fmt.Fprintf(&b, "### Host: %s\n", host)
		}
		for _, sb := range sbs {
			state := "unknown"
			if s, ok := sb["state"].(string); ok {
				state = s
			}
			name := sb["name"]
			id := sb["id"]
			baseImage := sb["base_image"]

			fmt.Fprintf(&b, "- **%s** (%s)\n", name, id)
			fmt.Fprintf(&b, "  State: %s | Base: %s", state, baseImage)
			if ip, ok := sb["ip"].(string); ok {
				fmt.Fprintf(&b, " | IP: %s", ip)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// listHostsWithVMs returns host info from the daemon
func (a *DeerAgent) listHostsWithVMs(ctx context.Context) (map[string]any, error) {
	info, err := a.service.GetHostInfo(ctx)
	if err != nil {
		a.logger.Error("get host info failed", "error", err)
		return nil, err
	}

	// Build a combined view with VMs and sandboxes
	vms, _ := a.service.ListVMs(ctx)
	sandboxes, _ := a.service.ListSandboxes(ctx)

	domains := make([]map[string]any, 0)
	for _, v := range vms {
		domains = append(domains, map[string]any{
			"name":  v.Name,
			"state": v.State,
			"host":  info.Hostname,
			"type":  "host_vm",
		})
	}
	for _, sb := range sandboxes {
		domains = append(domains, map[string]any{
			"name":  sb.Name,
			"state": sb.State,
			"host":  info.Hostname,
			"type":  "sandbox",
		})
	}

	return map[string]any{
		"domains": domains,
		"count":   len(domains),
	}, nil
}

func (a *DeerAgent) formatHostsResult(result map[string]any, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list hosts: %v", err)
	}

	domains, ok := result["domains"].([]map[string]any)
	if !ok || len(domains) == 0 {
		return "No domains found on any host."
	}

	var b strings.Builder

	// Group domains by host
	hostDomains := make(map[string][]map[string]any)
	for _, d := range domains {
		host := "local"
		if h, ok := d["host"].(string); ok && h != "" {
			host = h
		}
		hostDomains[host] = append(hostDomains[host], d)
	}

	// Count totals
	totalHostVMs := 0
	totalSandboxes := 0
	for _, ds := range hostDomains {
		for _, d := range ds {
			if d["type"] == "sandbox" {
				totalSandboxes++
			} else {
				totalHostVMs++
			}
		}
	}

	b.WriteString("# Hosts Overview\n\n")
	fmt.Fprintf(&b, "Total: %d host VM(s), %d sandbox(es)\n\n", totalHostVMs, totalSandboxes)

	// Display domains grouped by host
	for host, ds := range hostDomains {
		// Count per host
		hostVMCount := 0
		sandboxCount := 0
		for _, d := range ds {
			if d["type"] == "sandbox" {
				sandboxCount++
			} else {
				hostVMCount++
			}
		}

		fmt.Fprintf(&b, "## %s\n", host)
		fmt.Fprintf(&b, "Host VMs: %d | Sandboxes: %d\n\n", hostVMCount, sandboxCount)

		// Display host VMs first
		if hostVMCount > 0 {
			b.WriteString("**Host VMs (available for cloning):**\n")
			for _, d := range ds {
				if d["type"] != "host_vm" {
					continue
				}
				state := "unknown"
				if s, ok := d["state"].(string); ok {
					state = s
				}
				fmt.Fprintf(&b, "- %s (%s)\n", d["name"], state)
			}
			b.WriteString("\n")
		}

		// Display sandboxes
		if sandboxCount > 0 {
			b.WriteString("**Sandboxes (ephemeral VMs):**\n")
			for _, d := range ds {
				if d["type"] != "sandbox" {
					continue
				}
				state := "unknown"
				if s, ok := d["state"].(string); ok {
					state = s
				}
				fmt.Fprintf(&b, "- %s (%s)\n", d["name"], state)
			}
			b.WriteString("\n")
		}
	}

	// Display any host errors
	if hostErrors, ok := result["host_errors"].([]map[string]any); ok && len(hostErrors) > 0 {
		b.WriteString("## Host Errors\n")
		for _, he := range hostErrors {
			fmt.Fprintf(&b, "- **%s**: %s\n", he["host"], he["error"])
		}
	}

	return b.String()
}

func (a *DeerAgent) listPlaybooks(ctx context.Context) (map[string]any, error) {
	playbooks, err := a.playbookService.ListPlaybooks(ctx, nil)
	if err != nil {
		a.logger.Error("list playbooks failed", "error", err)
		return nil, err
	}

	result := make([]map[string]any, 0, len(playbooks))
	for _, pb := range playbooks {
		path := ""
		if pb.FilePath != nil && *pb.FilePath != "" {
			path = *pb.FilePath
		} else {
			path = filepath.Join(a.cfg.Ansible.PlaybooksDir, pb.Name+".yml")
		}
		result = append(result, map[string]any{
			"id":         pb.ID,
			"name":       pb.Name,
			"path":       path,
			"created_at": pb.CreatedAt.Format(time.RFC3339),
		})
	}

	return map[string]any{
		"playbooks": result,
		"count":     len(result),
	}, nil
}

func (a *DeerAgent) formatPlaybooksResult(result map[string]any, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list playbooks: %v", err)
	}

	playbooks, ok := result["playbooks"].([]map[string]any)
	if !ok || len(playbooks) == 0 {
		return "No playbooks found."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d playbook(s):\n\n", len(playbooks))
	for _, pb := range playbooks {
		name := pb["name"].(string)
		path := pb["path"].(string)

		absPath, _ := filepath.Abs(path)
		// OSC 8 hyperlink
		link := fmt.Sprintf("\033]8;;file://%s\033\\%s\033]8;;\033\\", absPath, path)

		fmt.Fprintf(&b, "- **%s**: %s\n", name, link)
	}
	return b.String()
}

// runSourceCommand executes a read-only command on a source/golden VM.
func (a *DeerAgent) runSourceCommand(ctx context.Context, sourceVM, command string) (map[string]any, error) {
	truncCmd := command
	if len(truncCmd) > 120 {
		truncCmd = truncCmd[:120] + "..."
	}
	a.logger.Debug("run source command", "source_vm", sourceVM, "command", truncCmd)

	result, err := a.service.RunSourceCommand(ctx, sourceVM, command, 0)
	if err != nil {
		a.logger.Error("source command failed", "source_vm", sourceVM, "error", err)
		if result != nil {
			stdout, stdoutRedacted := a.redactContent(result.Stdout)
			stderr, stderrRedacted := a.redactContent(result.Stderr)
			if stdoutRedacted || stderrRedacted {
				a.sendRedactedMsg(sourceVM, "")
			}
			return map[string]any{
				"source_vm": sourceVM,
				"exit_code": result.ExitCode,
				"stdout":    stdout,
				"stderr":    stderr,
				"error":     err.Error(),
			}, nil
		}
		return nil, err
	}

	stdout, stdoutRedacted := a.redactContent(result.Stdout)
	stderr, stderrRedacted := a.redactContent(result.Stderr)
	if stdoutRedacted || stderrRedacted {
		a.sendRedactedMsg(sourceVM, "")
	}
	return map[string]any{
		"source_vm": sourceVM,
		"exit_code": result.ExitCode,
		"stdout":    stdout,
		"stderr":    stderr,
	}, nil
}

// shellEscape safely escapes a string for use in a shell command.
// It uses POSIX single-quote escaping: wrap in single quotes and replace
// any single quotes with '\” (end quote, escaped quote, start quote).
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// readSourceFile reads a file from a source/golden VM.
func (a *DeerAgent) readSourceFile(ctx context.Context, sourceVM, path string) (map[string]any, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	a.logger.Debug("read source file", "source_vm", sourceVM, "path", path)

	content, err := a.service.ReadSourceFile(ctx, sourceVM, path)
	if err != nil {
		a.logger.Error("failed to read file from source VM", "source_vm", sourceVM, "path", path, "error", err)
		return nil, fmt.Errorf("failed to read file from source VM: %w", err)
	}

	content, wasRedacted := a.redactContent(content)
	if wasRedacted {
		a.sendRedactedMsg(sourceVM, path)
	}

	// Show file content in live output box
	a.sendStatus(CommandOutputStartMsg{SandboxID: sourceVM})
	a.sendStatus(CommandOutputChunkMsg{
		SandboxID: sourceVM,
		Chunk:     content + "\n",
	})
	a.sendStatus(CommandOutputDoneMsg{SandboxID: sourceVM})

	return map[string]any{
		"source_vm": sourceVM,
		"path":      path,
		"content":   content,
	}, nil
}

// Cleanup destroys all sandboxes created during this session.
// This is called when the TUI exits to ensure no orphaned VMs are left running.
func (a *DeerAgent) Cleanup(ctx context.Context) error {
	if len(a.createdSandboxes) == 0 {
		return nil
	}

	a.logger.Info("cleanup starting", "sandbox_count", len(a.createdSandboxes))

	var errs []error
	for _, id := range a.createdSandboxes {
		// Check if sandbox still exists before destroying
		if _, err := a.service.GetSandbox(ctx, id); err != nil {
			// Sandbox no longer exists (already destroyed by user), skip
			continue
		}

		if err := a.service.DestroySandbox(ctx, id); err != nil {
			a.logger.Warn("cleanup: failed to destroy sandbox", "sandbox_id", id, "error", err)
			errs = append(errs, fmt.Errorf("destroy sandbox %s: %w", id, err))
			// Continue trying to destroy others even if one fails
		}
	}

	// Clear the list
	a.createdSandboxes = nil

	a.logger.Info("cleanup complete", "errors", len(errs))
	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}
	return nil
}

// CreatedSandboxCount returns the number of sandboxes created during this session.
func (a *DeerAgent) CreatedSandboxCount() int {
	return len(a.createdSandboxes)
}

// GetCreatedSandboxes returns a copy of the sandbox IDs created during this session.
func (a *DeerAgent) GetCreatedSandboxes() []string {
	result := make([]string, len(a.createdSandboxes))
	copy(result, a.createdSandboxes)
	return result
}

// ClearCreatedSandboxes clears the list of created sandboxes.
func (a *DeerAgent) ClearCreatedSandboxes() {
	a.createdSandboxes = nil
}

// CleanupWithProgress destroys all sandboxes, sending progress updates through the status callback.
// Each sandbox gets its own 60-second timeout to avoid one slow destroy blocking others.
func (a *DeerAgent) CleanupWithProgress(sandboxIDs []string) {
	total := len(sandboxIDs)
	a.logger.Info("cleanup with progress starting", "total", total)
	destroyed := 0
	failed := 0
	skipped := 0

	// Per-sandbox timeout - 60s should be enough for remote hosts
	const perSandboxTimeout = 60 * time.Second

	for _, id := range sandboxIDs {
		// Send progress: destroying
		a.sendStatus(CleanupProgressMsg{
			SandboxID: id,
			Status:    CleanupStatusDestroying,
		})

		// Small delay to let the UI update
		time.Sleep(50 * time.Millisecond)

		// Create a fresh context for each sandbox destruction
		ctx, cancel := context.WithTimeout(context.Background(), perSandboxTimeout)

		// Check if sandbox still exists
		if _, err := a.service.GetSandbox(ctx, id); err != nil {
			// Already destroyed
			cancel()
			skipped++
			a.logger.Debug("cleanup: sandbox already gone", "sandbox_id", id)
			a.sendStatus(CleanupProgressMsg{
				SandboxID: id,
				Status:    CleanupStatusSkipped,
			})
			continue
		}

		// Destroy the sandbox
		if err := a.service.DestroySandbox(ctx, id); err != nil {
			failed++
			a.logger.Warn("cleanup: failed to destroy sandbox", "sandbox_id", id, "error", err)
			a.sendStatus(CleanupProgressMsg{
				SandboxID: id,
				Status:    CleanupStatusFailed,
				Error:     err.Error(),
			})
		} else {
			destroyed++
			a.logger.Debug("cleanup: sandbox destroyed", "sandbox_id", id)
			a.sendStatus(CleanupProgressMsg{
				SandboxID: id,
				Status:    CleanupStatusDestroyed,
			})
		}
		cancel()
	}

	// Clear the created sandboxes list
	a.createdSandboxes = nil

	a.logger.Info("cleanup with progress complete", "total", total, "destroyed", destroyed, "failed", failed, "skipped", skipped)

	// Send completion message
	a.sendStatus(CleanupCompleteMsg{
		Total:     total,
		Destroyed: destroyed,
		Failed:    failed,
		Skipped:   skipped,
	})
}

// GetCurrentSandbox returns the currently active sandbox ID and host
func (a *DeerAgent) GetCurrentSandbox() (id string, host string) {
	return a.currentSandboxID, a.currentSandboxHost
}

// SetCurrentSandbox sets the currently active sandbox
func (a *DeerAgent) SetCurrentSandbox(id string, host string) {
	a.currentSandboxID = id
	a.currentSandboxHost = host
}

// GetCurrentSandboxBaseImage returns the base image of the current sandbox
func (a *DeerAgent) GetCurrentSandboxBaseImage() string {
	return a.currentSandboxBaseImage
}

// GetCurrentSourceVM returns the source VM currently being operated on
func (a *DeerAgent) GetCurrentSourceVM() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentSourceVM
}

// ClearAutoReadOnly clears the auto read-only flag (for manual override via Shift+Tab)
func (a *DeerAgent) ClearAutoReadOnly() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.autoReadOnly = false
	a.displayReadOnly = false
}

// clearStickyReadOnly clears the sticky read-only display state when a write tool executes.
// Sends AutoReadOnlyMsg{Enabled: false} to the model if sticky display was active.
func (a *DeerAgent) clearStickyReadOnly() {
	a.mu.Lock()
	if !a.displayReadOnly {
		a.mu.Unlock()
		return
	}
	a.displayReadOnly = false
	a.mu.Unlock()
	a.sendStatus(AutoReadOnlyMsg{Enabled: false})
}

func (a *DeerAgent) handleListSkills() (map[string]any, error) {
	if a.skillLoader == nil {
		return map[string]any{"skills": []any{}, "count": 0}, nil
	}
	entries := a.skillLoader.Catalog()
	skills := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		skills = append(skills, map[string]any{
			"name":        e.Name,
			"description": e.Description,
		})
	}
	return map[string]any{"skills": skills, "count": len(skills)}, nil
}

func (a *DeerAgent) handleLoadSkill(name string) (map[string]any, error) {
	if a.skillLoader == nil {
		return nil, fmt.Errorf("no skills loaded")
	}
	s := a.skillLoader.Get(name)
	if s == nil {
		return nil, fmt.Errorf("skill %q not found; use list_skills to see available skills", name)
	}
	return map[string]any{
		"name":        s.Name,
		"description": s.Description,
		"version":     s.Version,
		"content":     s.Content,
	}, nil
}

func (a *DeerAgent) handleAddTask(content string) (map[string]any, error) {
	if a.taskList == nil {
		a.taskList = NewTaskList()
	}
	t := a.taskList.Add(content)
	a.notifyTasks()
	return map[string]any{
		"id":      t.ID,
		"content": t.Content,
		"status":  string(t.Status),
		"message": "Task added",
	}, nil
}

func (a *DeerAgent) handleUpdateTask(taskID string, status TaskStatus, content string) (map[string]any, error) {
	if a.taskList == nil {
		return nil, fmt.Errorf("no tasks to update")
	}
	t, found := a.taskList.Update(taskID, status, content)
	if !found {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	a.notifyTasks()
	return map[string]any{
		"id":      t.ID,
		"content": t.Content,
		"status":  string(t.Status),
		"message": "Task updated",
	}, nil
}

func (a *DeerAgent) handleDeleteTask(taskID string) (map[string]any, error) {
	if a.taskList == nil {
		return nil, fmt.Errorf("no tasks to delete")
	}
	if !a.taskList.Delete(taskID) {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	a.notifyTasks()
	return map[string]any{
		"id":      taskID,
		"message": "Task deleted",
	}, nil
}

func (a *DeerAgent) handleListTasks() (map[string]any, error) {
	if a.taskList == nil {
		return map[string]any{"tasks": []any{}, "count": 0}, nil
	}
	tasks := a.taskList.List()
	result := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, map[string]any{
			"id":      t.ID,
			"content": t.Content,
			"status":  string(t.Status),
		})
	}
	return map[string]any{"tasks": result, "count": len(result)}, nil
}

func (a *DeerAgent) notifyTasks() {
	if a.taskList == nil || a.statusCallback == nil {
		return
	}
	a.statusCallback(TasksUpdatedMsg{Tasks: a.taskList.List()})
}

// GetTasks returns the current task list for TUI display.
func (a *DeerAgent) GetTasks() []Task {
	if a.taskList == nil {
		return nil
	}
	return a.taskList.List()
}
