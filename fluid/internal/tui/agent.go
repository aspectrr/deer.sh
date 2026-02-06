package tui

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspectrr/fluid.sh/fluid/internal/ansible"
	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/libvirt"
	"github.com/aspectrr/fluid.sh/fluid/internal/llm"
	"github.com/aspectrr/fluid.sh/fluid/internal/store"
	"github.com/aspectrr/fluid.sh/fluid/internal/telemetry"
	"github.com/aspectrr/fluid.sh/fluid/internal/vm"
)

// PendingApproval represents a sandbox creation waiting for memory approval
type PendingApproval struct {
	Request      MemoryApprovalRequest
	SourceVM     string
	SandboxName  string
	Host         *config.HostConfig // nil for local
	ResponseChan chan bool
}

// FluidAgent implements AgentRunner for the fluid CLI
type FluidAgent struct {
	cfg             *config.Config
	store           store.Store
	vmService       *vm.Service
	manager         libvirt.Manager
	llmClient       llm.Client
	playbookService *ansible.PlaybookService
	telemetry       telemetry.Service

	// Multi-host support
	multiHostMgr *libvirt.MultiHostDomainManager

	// Status callback for sending updates to TUI
	statusCallback func(tea.Msg)

	// Conversation history for context
	history []llm.Message

	// Track sandboxes created during this session for cleanup on exit
	createdSandboxes []string

	// Currently active sandbox (for status bar display)
	currentSandboxID   string
	currentSandboxHost string

	// Pending approval for memory-constrained sandbox creation
	pendingApproval *PendingApproval

	// Pending approval for network access
	pendingNetworkApproval *PendingNetworkApproval

	// Read-only mode: only query tools are available to the LLM
	readOnly bool
}

// PendingNetworkApproval represents a network access request waiting for approval
type PendingNetworkApproval struct {
	Request      NetworkApprovalRequest
	ResponseChan chan bool
}

// NewFluidAgent creates a new fluid agent
func NewFluidAgent(cfg *config.Config, store store.Store, vmService *vm.Service, manager libvirt.Manager, tele telemetry.Service) *FluidAgent {
	var llmClient llm.Client
	if cfg.AIAgent.Provider == "openrouter" {
		llmClient = llm.NewOpenRouterClient(cfg.AIAgent)
	}

	agent := &FluidAgent{
		cfg:             cfg,
		store:           store,
		vmService:       vmService,
		manager:         manager,
		llmClient:       llmClient,
		playbookService: ansible.NewPlaybookService(store, cfg.Ansible.PlaybooksDir),
		telemetry:       tele,
		history:         make([]llm.Message, 0),
	}

	// Initialize multi-host manager if hosts are configured
	if len(cfg.Hosts) > 0 {
		// Use a silent logger for multi-host manager to avoid TUI corruption
		silentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		agent.multiHostMgr = libvirt.NewMultiHostDomainManager(cfg.Hosts, silentLogger)
	}

	return agent
}

// SetStatusCallback sets the callback function for status updates
func (a *FluidAgent) SetStatusCallback(callback func(tea.Msg)) {
	a.statusCallback = callback
}

// SetReadOnly toggles read-only mode on the agent
func (a *FluidAgent) SetReadOnly(ro bool) {
	a.readOnly = ro
}

// sendStatus sends a status message through the callback if set
func (a *FluidAgent) sendStatus(msg tea.Msg) {
	if a.statusCallback != nil {
		a.statusCallback(msg)
	}
}

// Run executes a command and returns the result
func (a *FluidAgent) Run(input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Handle slash commands
		if strings.HasPrefix(input, "/") {
			switch input {
			case "/vms":
				a.sendStatus(AgentDoneMsg{})
				result, err := a.listVMs(ctx)
				return AgentResponseMsg{Response: AgentResponse{
					Content: a.formatVMsResult(result, err),
					Done:    true,
				}}
			case "/sandboxes":
				a.sendStatus(AgentDoneMsg{})
				result, err := a.listSandboxes(ctx)
				return AgentResponseMsg{Response: AgentResponse{
					Content: a.formatSandboxesResult(result, err),
					Done:    true,
				}}
			case "/hosts":
				a.sendStatus(AgentDoneMsg{})
				result, err := a.listHostsWithVMs(ctx)
				return AgentResponseMsg{Response: AgentResponse{
					Content: a.formatHostsResult(result, err),
					Done:    true,
				}}
			case "/playbooks":
				a.sendStatus(AgentDoneMsg{})
				result, err := a.listPlaybooks(ctx)
				return AgentResponseMsg{Response: AgentResponse{
					Content: a.formatPlaybooksResult(result, err),
					Done:    true,
				}}
			case "/compact":
				// Manual compaction
				a.sendStatus(CompactStartMsg{})
				result, err := a.Compact(ctx)
				a.sendStatus(AgentDoneMsg{})
				if err != nil {
					return CompactErrorMsg{Err: err}
				}
				// The Compact function returns a CompactCompleteMsg,
				// but here we are in a func returning tea.Msg.
				// result is already CompactCompleteMsg.
				return result
			case "/context":
				// Show context usage
				a.sendStatus(AgentDoneMsg{})
				usage := a.GetContextUsage()
				tokens := a.EstimateTokens()
				maxTokens := a.cfg.AIAgent.TotalContextTokens
				threshold := a.cfg.AIAgent.CompactThreshold
				return AgentResponseMsg{Response: AgentResponse{
					Content: fmt.Sprintf("Context usage: %d/%d tokens (%.1f%%)\nAuto-compact threshold: %.0f%%",
						tokens, maxTokens, usage*100, threshold*100),
					Done: true,
				}}
			case "/help":
				a.sendStatus(AgentDoneMsg{})
				var b strings.Builder
				b.WriteString("## Available Commands\n\n")
				b.WriteString("- **/vms**: List available VMs for cloning\n")
				b.WriteString("- **/sandboxes**: List active sandboxes\n")
				b.WriteString("- **/hosts**: List configured remote hosts\n")
				b.WriteString("- **/playbooks**: List generated Ansible playbooks\n")
				b.WriteString("- **/compact**: Summarize and compact conversation history\n")
				b.WriteString("- **/context**: Show current context token usage\n")
				b.WriteString("- **/settings**: Open configuration settings\n")
				b.WriteString("- **/clear**: Clear conversation history\n")
				b.WriteString("- **/help**: Show this help message\n")
				return AgentResponseMsg{Response: AgentResponse{
					Content: b.String(),
					Done:    true,
				}}
			default:
				a.sendStatus(AgentDoneMsg{})
				return AgentResponseMsg{Response: AgentResponse{
					Content: fmt.Sprintf("Unknown command: %s. Available: /vms, /sandboxes, /hosts, /playbooks, /compact, /context, /settings", input),
					Done:    true,
				}}
			}
		}

		// Add user message to history
		a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: input})

		// LLM client is required
		if a.llmClient == nil || a.cfg.AIAgent.APIKey == "" {
			a.sendStatus(AgentDoneMsg{})
			return AgentErrorMsg{Err: fmt.Errorf("LLM provider not configured. Please set OPENROUTER_API_KEY environment variable or configure it in /settings")}
		}

		// Check if auto-compaction is needed before making LLM call
		if a.NeedsCompaction() {
			a.sendStatus(CompactStartMsg{})
			compactResult, err := a.Compact(ctx)
			if err != nil {
				// Log warning but continue - don't fail the request
				a.sendStatus(CompactErrorMsg{Err: fmt.Errorf("auto-compact failed, continuing with full context: %w", err)})
			} else {
				// Send compact complete to TUI so user knows it happened
				a.sendStatus(compactResult)
			}
		}

		// LLM-driven execution loop
		for {
			systemPrompt := a.cfg.AIAgent.DefaultSystem
			tools := llm.GetTools()
			if a.readOnly {
				tools = llm.GetReadOnlyTools()
				systemPrompt += "\n\nYou are in READ-ONLY mode. You can only query and observe - you cannot create, modify, or destroy any resources.  Available tools: list_sandboxes, get_sandbox, list_vms, read_file, list_playbooks, get_playbook."
			}

			req := llm.ChatRequest{
				Messages: append([]llm.Message{{
					Role:    llm.RoleSystem,
					Content: systemPrompt,
				}}, a.history...),
				Tools: tools,
			}

			if a.telemetry != nil {
				a.telemetry.Track("agent_prompt_sent", map[string]any{
					"message_count": len(req.Messages),
					"provider":      a.cfg.AIAgent.Provider,
					"model":         a.cfg.AIAgent.Model,
				})
			}

			resp, err := a.llmClient.Chat(ctx, req)
			if err != nil {
				a.sendStatus(AgentDoneMsg{})
				return AgentErrorMsg{Err: fmt.Errorf("llm chat: %w", err)}
			}

			if len(resp.Choices) == 0 {
				a.sendStatus(AgentDoneMsg{})
				return AgentErrorMsg{Err: fmt.Errorf("llm returned no choices")}
			}

			msg := resp.Choices[0].Message
			a.history = append(a.history, msg)

			if len(msg.ToolCalls) > 0 {
				// Send intermediate response if there's content
				if msg.Content != "" {
					a.sendStatus(AgentResponseMsg{Response: AgentResponse{
						Content: msg.Content,
						Done:    false,
					}})
				}

				// Handle tool calls
				for _, tc := range msg.ToolCalls {
					result, err := a.executeTool(ctx, tc)

					var toolResultContent string
					var resultMap map[string]any
					success := true
					errMsg := ""

					if err != nil {
						success = false
						errMsg = err.Error()
						toolResultContent = fmt.Sprintf("Error: %v", err)
					} else {
						if m, ok := result.(map[string]any); ok {
							resultMap = m
						}
						jsonResult, _ := json.Marshal(result)
						toolResultContent = string(jsonResult)
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

			// No more tool calls, return final response
			// Tool results were already sent via ToolCompleteMsg
			// Send done message to unblock status listener
			a.sendStatus(AgentDoneMsg{})
			return AgentResponseMsg{Response: AgentResponse{
				Content: msg.Content,
				Done:    true,
			}}
		}
	}
}

// executeTool dispatches tool calls to internal methods
func (a *FluidAgent) executeTool(ctx context.Context, tc llm.ToolCall) (any, error) {
	// Parse args for status message
	var args map[string]any
	_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

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
		var args struct {
			SourceVM string `json:"source_vm"`
			Host     string `json:"host"`
			CPU      int    `json:"cpu"`
			MemoryMB int    `json:"memory_mb"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.createSandbox(ctx, args.SourceVM, args.Host, args.CPU, args.MemoryMB)
	case "destroy_sandbox":
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.destroySandbox(ctx, args.SandboxID)
	case "run_command":
		var args struct {
			SandboxID string `json:"sandbox_id"`
			Command   string `json:"command"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.runCommand(ctx, args.SandboxID, args.Command)
	case "start_sandbox":
		var args struct {
			SandboxID string `json:"sandbox_id"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.startSandbox(ctx, args.SandboxID)
	case "stop_sandbox":
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
		var args struct {
			SandboxID string `json:"sandbox_id"`
			Name      string `json:"name"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.createSnapshot(ctx, args.SandboxID, args.Name)
	case "create_playbook":
		var args ansible.CreatePlaybookRequest
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
		return a.playbookService.CreatePlaybook(ctx, args)
	case "add_playbook_task":
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
	default:
		return nil, fmt.Errorf("unknown tool: %s", tc.Function.Name)
	}
}

// Reset clears the conversation history
func (a *FluidAgent) Reset() {
	a.history = make([]llm.Message, 0)
}

// EstimateTokens estimates the token count for the current conversation history
func (a *FluidAgent) EstimateTokens() int {
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
func (a *FluidAgent) GetContextUsage() float64 {
	maxTokens := a.cfg.AIAgent.TotalContextTokens
	if maxTokens <= 0 {
		maxTokens = 64000
	}
	return float64(a.EstimateTokens()) / float64(maxTokens)
}

// NeedsCompaction returns true if the context is at or above the compaction threshold
func (a *FluidAgent) NeedsCompaction() bool {
	threshold := a.cfg.AIAgent.CompactThreshold
	if threshold <= 0 {
		threshold = 0.9
	}
	return a.GetContextUsage() >= threshold
}

// Compact summarizes the conversation history using a smaller LLM and resets with the summary
func (a *FluidAgent) Compact(ctx context.Context) (CompactCompleteMsg, error) {
	if len(a.history) == 0 {
		return CompactCompleteMsg{}, fmt.Errorf("no conversation history to compact")
	}

	previousTokens := a.EstimateTokens()

	// Build the conversation text for summarization
	var convText strings.Builder
	convText.WriteString("Conversation history to summarize:\n\n")
	for _, msg := range a.history {
		switch msg.Role {
		case llm.RoleUser:
			convText.WriteString(fmt.Sprintf("User: %s\n\n", msg.Content))
		case llm.RoleAssistant:
			if msg.Content != "" {
				convText.WriteString(fmt.Sprintf("Assistant: %s\n\n", msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				convText.WriteString(fmt.Sprintf("Assistant called tool: %s(%s)\n\n", tc.Function.Name, tc.Function.Arguments))
			}
		case llm.RoleTool:
			convText.WriteString(fmt.Sprintf("Tool result (%s): %s\n\n", msg.Name, msg.Content))
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
		return CompactCompleteMsg{}, fmt.Errorf("compaction LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
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

	return CompactCompleteMsg{
		PreviousTokens: previousTokens,
		NewTokens:      newTokens,
		Summary:        summary,
	}, nil
}

// RunCompact executes the compaction as a tea.Cmd
func (a *FluidAgent) RunCompact() tea.Cmd {
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

func (a *FluidAgent) listSandboxes(ctx context.Context) (map[string]any, error) {
	sandboxes, err := a.vmService.GetSandboxes(ctx, store.SandboxFilter{}, nil)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(sandboxes))
	for _, sb := range sandboxes {
		item := map[string]any{
			"id":         sb.ID,
			"name":       sb.SandboxName,
			"state":      sb.State,
			"base_image": sb.BaseImage,
			"created_at": sb.CreatedAt.Format(time.RFC3339),
		}
		if sb.IPAddress != nil {
			item["ip"] = *sb.IPAddress
		}
		if sb.HostName != nil {
			item["host"] = *sb.HostName
		}
		if sb.HostAddress != nil {
			item["host_address"] = *sb.HostAddress
		}
		result = append(result, item)
	}

	return map[string]any{
		"sandboxes": result,
		"count":     len(result),
	}, nil
}

func (a *FluidAgent) createSandbox(ctx context.Context, sourceVM, hostName string, cpu, memoryMB int) (map[string]any, error) {
	if sourceVM == "" {
		return nil, fmt.Errorf("source-vm is required (e.g., create ubuntu-base)")
	}

	// Determine target host and manager
	var host *config.HostConfig
	var mgr libvirt.Manager

	if a.multiHostMgr != nil {
		var err error
		host, err = a.findHostForSourceVM(ctx, sourceVM, hostName)
		if err != nil {
			return nil, fmt.Errorf("find host for source VM: %w", err)
		}
	}

	if host != nil {
		mgr = a.vmService.GetRemoteManager(host)
	} else {
		mgr = a.vmService.GetManager()
	}

	// Use provided values or defaults
	cpuCount := cpu
	if cpuCount <= 0 {
		cpuCount = a.vmService.GetDefaultCPUs()
	}
	memMB := memoryMB
	if memMB <= 0 {
		memMB = a.vmService.GetDefaultMemory()
	}

	// Sandbox name will be auto-generated as sbx-{id}

	validation := a.vmService.CheckResourcesForSandbox(ctx, mgr, sourceVM, cpuCount, memMB)

	// If source VM is invalid, fail immediately
	if !validation.SourceVMValid {
		return nil, fmt.Errorf("source VM validation failed: %s", strings.Join(validation.VMErrors, "; "))
	}

	// If resources are insufficient, request human approval
	if validation.NeedsApproval {
		hostNameStr := "local"
		if host != nil {
			hostNameStr = host.Name
		}

		request := MemoryApprovalRequest{
			SourceVM:          sourceVM,
			HostName:          hostNameStr,
			RequiredMemoryMB:  validation.ResourceCheck.RequiredMemoryMB,
			AvailableMemoryMB: validation.ResourceCheck.AvailableMemoryMB,
			TotalMemoryMB:     validation.ResourceCheck.TotalMemoryMB,
			Warnings:          validation.ResourceCheck.Warnings,
			Errors:            validation.ResourceCheck.Errors,
		}

		// Create a channel to wait for approval response
		responseChan := make(chan bool, 1)

		// Store pending approval and send request to TUI
		a.pendingApproval = &PendingApproval{
			Request:      request,
			SourceVM:     sourceVM,
			SandboxName:  "", // Auto-generated as sbx-{id}
			Host:         host,
			ResponseChan: responseChan,
		}

		// Send the approval request to the TUI
		a.sendStatus(MemoryApprovalRequestMsg{Request: request})

		// Wait for response (this blocks the agent until user responds)
		approved := <-responseChan
		a.pendingApproval = nil

		if !approved {
			return nil, fmt.Errorf("sandbox creation denied: insufficient memory (need %d MB, have %d MB available) - human approval was not granted",
				validation.ResourceCheck.RequiredMemoryMB, validation.ResourceCheck.AvailableMemoryMB)
		}
	}

	// Proceed with creation (resources approved or sufficient)
	if host != nil {
		// Create on remote host
		sb, ip, err := a.vmService.CreateSandboxOnHost(ctx, host, sourceVM, "tui-agent", "", cpuCount, memMB, nil, true, true)
		if err != nil {
			return nil, err
		}

		// Track the created sandbox for cleanup on exit
		a.createdSandboxes = append(a.createdSandboxes, sb.ID)

		// Set as current sandbox for status bar display
		a.currentSandboxID = sb.ID
		a.currentSandboxHost = host.Name

		result := map[string]any{
			"sandbox_id": sb.ID,
			"name":       sb.SandboxName,
			"state":      sb.State,
			"host":       host.Name,
		}
		if ip != "" {
			result["ip"] = ip
		}
		return result, nil
	}

	// Fall back to local creation
	sb, ip, err := a.vmService.CreateSandbox(ctx, sourceVM, "tui-agent", "", cpuCount, memMB, nil, true, true)
	if err != nil {
		return nil, err
	}

	// Track the created sandbox for cleanup on exit
	a.createdSandboxes = append(a.createdSandboxes, sb.ID)

	// Set as current sandbox for status bar display
	a.currentSandboxID = sb.ID
	a.currentSandboxHost = "local"

	result := map[string]any{
		"sandbox_id": sb.ID,
		"name":       sb.SandboxName,
		"state":      sb.State,
	}
	if ip != "" {
		result["ip"] = ip
	}

	return result, nil
}

// HandleApprovalResponse handles the response from the memory approval dialog
func (a *FluidAgent) HandleApprovalResponse(approved bool) {
	if a.pendingApproval != nil && a.pendingApproval.ResponseChan != nil {
		a.pendingApproval.ResponseChan <- approved
	}
}

// HandleNetworkApprovalResponse handles the response from the network approval dialog
func (a *FluidAgent) HandleNetworkApprovalResponse(approved bool) {
	if a.pendingNetworkApproval != nil && a.pendingNetworkApproval.ResponseChan != nil {
		a.pendingNetworkApproval.ResponseChan <- approved
	}
}

// findHostForSourceVM finds the host that has the given source VM.
// If hostName is specified, only that host is checked.
// Returns nil if no remote host has the VM (fallback to local).
func (a *FluidAgent) findHostForSourceVM(ctx context.Context, sourceVM, hostName string) (*config.HostConfig, error) {
	if a.multiHostMgr == nil {
		return nil, nil
	}

	// If specific host requested, check only that host
	if hostName != "" {
		hosts := a.multiHostMgr.GetHosts()
		for i := range hosts {
			if hosts[i].Name == hostName {
				return &hosts[i], nil
			}
		}
		return nil, fmt.Errorf("host %q not found in configuration", hostName)
	}

	// Search all hosts for the source VM
	host, err := a.multiHostMgr.FindHostForVM(ctx, sourceVM)
	if err != nil {
		// Not found on any remote host - will try local
		return nil, nil
	}

	return host, nil
}

func (a *FluidAgent) destroySandbox(ctx context.Context, id string) (map[string]any, error) {
	_, err := a.vmService.DestroySandbox(ctx, id)
	if err != nil {
		return nil, err
	}

	// Clear current sandbox if this was the one being destroyed
	if id == a.currentSandboxID {
		a.currentSandboxID = ""
		a.currentSandboxHost = ""
	}

	return map[string]any{
		"destroyed":  true,
		"sandbox_id": id,
	}, nil
}

func (a *FluidAgent) runCommand(ctx context.Context, sandboxID, command string) (map[string]any, error) {
	// Update current sandbox if different (user is working with this sandbox)
	if sandboxID != "" && sandboxID != a.currentSandboxID {
		a.currentSandboxID = sandboxID
		// Try to get host info from sandbox
		if sb, err := a.vmService.GetSandbox(ctx, sandboxID); err == nil && sb.HostName != nil {
			a.currentSandboxHost = *sb.HostName
		} else {
			a.currentSandboxHost = "local"
		}
	}

	// Check if command requires network access and request approval
	networkTool, urls := detectNetworkAccess(command)
	if networkTool != "" {
		request := NetworkApprovalRequest{
			Command:     command,
			SandboxID:   sandboxID,
			NetworkTool: networkTool,
			URLs:        urls,
		}

		// Create a channel to wait for approval response
		responseChan := make(chan bool, 1)

		// Store pending approval and send request to TUI
		a.pendingNetworkApproval = &PendingNetworkApproval{
			Request:      request,
			ResponseChan: responseChan,
		}

		// Send the approval request to the TUI
		a.sendStatus(NetworkApprovalRequestMsg{Request: request})

		// Wait for response (this blocks the agent until user responds)
		approved := <-responseChan
		a.pendingNetworkApproval = nil

		if !approved {
			return map[string]any{
				"sandbox_id": sandboxID,
				"error":      "network access denied by user",
				"exit_code":  -1,
			}, nil
		}
	}

	// Create callback to send chunks to TUI for live output
	outputCallback := func(chunk vm.OutputChunk) {
		if chunk.IsRetry && chunk.Retry != nil {
			// Signal retry to reset live output and show warning
			a.sendStatus(CommandOutputResetMsg{SandboxID: sandboxID})
			a.sendStatus(RetryAttemptMsg{
				SandboxID: sandboxID,
				Attempt:   chunk.Retry.Attempt,
				Max:       chunk.Retry.Max,
				Delay:     chunk.Retry.Delay,
				Error:     chunk.Retry.Error,
			})
			return
		}

		// nil Data but NOT a retry usually means a reset signal from elsewhere
		if chunk.Data == nil {
			a.sendStatus(CommandOutputResetMsg{SandboxID: sandboxID})
			return
		}

		a.sendStatus(CommandOutputChunkMsg{
			SandboxID: sandboxID,
			IsStderr:  chunk.IsStderr,
			Chunk:     string(chunk.Data),
		})
	}

	user := a.cfg.SSH.DefaultUser
	result, err := a.vmService.RunCommandWithCallback(ctx, sandboxID, user, "", command, 0, nil, outputCallback)

	// Signal streaming complete
	a.sendStatus(CommandOutputDoneMsg{SandboxID: sandboxID})

	if err != nil {
		// Return partial result if available
		if result != nil {
			return map[string]any{
				"sandbox_id": sandboxID,
				"exit_code":  result.ExitCode,
				"stdout":     result.Stdout,
				"stderr":     result.Stderr,
				"error":      err.Error(),
			}, nil
		}
		return nil, err
	}

	return map[string]any{
		"sandbox_id": sandboxID,
		"exit_code":  result.ExitCode,
		"stdout":     result.Stdout,
		"stderr":     result.Stderr,
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
func (a *FluidAgent) editFile(ctx context.Context, sandboxID, path, oldStr, newStr string) (map[string]any, error) {
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required - this tool operates on files inside a sandbox VM")
	}

	// Resolve to absolute path if not already
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	user := a.cfg.SSH.DefaultUser

	// If old_str is empty, create/overwrite the file
	if oldStr == "" {
		// Use base64 encoding to safely transfer content over SSH
		// This avoids issues with heredocs, special characters, and shell escaping
		encoded := base64.StdEncoding.EncodeToString([]byte(newStr))
		cmd := fmt.Sprintf("echo '%s' | base64 -d > '%s'", encoded, path)

		result, err := a.vmService.RunCommand(ctx, sandboxID, user, "", cmd, 0, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
		if result.ExitCode != 0 {
			return nil, fmt.Errorf("failed to create file: %s", result.Stderr)
		}
		return map[string]any{
			"sandbox_id": sandboxID,
			"path":       path,
			"action":     "created_file",
		}, nil
	}

	// Read the original file using base64 to handle binary/special chars
	readResult, err := a.vmService.RunCommand(ctx, sandboxID, user, "", fmt.Sprintf("base64 '%s'", path), 0, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if readResult.ExitCode != 0 {
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
	writeCmd := fmt.Sprintf("echo '%s' | base64 -d > '%s'", encoded, path)

	writeResult, err := a.vmService.RunCommand(ctx, sandboxID, user, "", writeCmd, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	if writeResult.ExitCode != 0 {
		return nil, fmt.Errorf("failed to write file: %s", writeResult.Stderr)
	}

	return map[string]any{
		"sandbox_id": sandboxID,
		"path":       path,
		"action":     "edited",
	}, nil
}

// readFile reads the contents of a file on a sandbox VM via SSH.
// This operates on files inside the sandbox - not local files or playbooks.
func (a *FluidAgent) readFile(ctx context.Context, sandboxID, path string) (map[string]any, error) {
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required - this tool operates on files inside a sandbox VM. For playbooks, use get_playbook instead")
	}

	// Resolve to absolute path if not already
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %s", path)
	}

	user := a.cfg.SSH.DefaultUser
	// Use base64 to safely transfer content that may contain special characters
	result, err := a.vmService.RunCommand(ctx, sandboxID, user, "", fmt.Sprintf("base64 '%s'", path), 0, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to read file: %s", result.Stderr)
	}

	// Decode the base64 content
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(result.Stdout))
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	return map[string]any{
		"sandbox_id": sandboxID,
		"path":       path,
		"content":    string(decoded),
	}, nil
}

// getPlaybook retrieves a playbook's full definition including YAML content and tasks.
// This is the correct way to view playbook definitions - not read_file.
func (a *FluidAgent) getPlaybook(ctx context.Context, playbookID string) (map[string]any, error) {
	if playbookID == "" {
		return nil, fmt.Errorf("playbook_id is required")
	}

	// Get playbook with tasks
	pbWithTasks, err := a.playbookService.GetPlaybookWithTasks(ctx, playbookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playbook: %w", err)
	}

	// Export the YAML content
	yamlContent, err := a.playbookService.ExportPlaybook(ctx, playbookID)
	if err != nil {
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

func (a *FluidAgent) startSandbox(ctx context.Context, id string) (map[string]any, error) {
	ip, err := a.vmService.StartSandbox(ctx, id, true)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"started":    true,
		"sandbox_id": id,
	}
	if ip != "" {
		result["ip"] = ip
	}

	return result, nil
}

func (a *FluidAgent) stopSandbox(ctx context.Context, id string) (map[string]any, error) {
	err := a.vmService.StopSandbox(ctx, id, false)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"stopped":    true,
		"sandbox_id": id,
	}, nil
}

func (a *FluidAgent) getSandbox(ctx context.Context, id string) (map[string]any, error) {
	sb, err := a.vmService.GetSandbox(ctx, id)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"sandbox_id": sb.ID,
		"name":       sb.SandboxName,
		"state":      sb.State,
		"base_image": sb.BaseImage,
		"network":    sb.Network,
		"agent_id":   sb.AgentID,
		"created_at": sb.CreatedAt.Format(time.RFC3339),
		"updated_at": sb.UpdatedAt.Format(time.RFC3339),
	}
	if sb.IPAddress != nil {
		result["ip"] = *sb.IPAddress
	}
	if sb.HostName != nil {
		result["host"] = *sb.HostName
	}
	if sb.HostAddress != nil {
		result["host_address"] = *sb.HostAddress
	}

	return result, nil
}

func (a *FluidAgent) listVMs(ctx context.Context) (map[string]any, error) {
	// If multihost manager is configured, query remote hosts
	if a.multiHostMgr != nil {
		return a.listVMsFromHosts(ctx)
	}

	// Fall back to local virsh
	return a.listVMsLocal(ctx)
}

// listVMsFromHosts queries all configured remote hosts for VMs (excludes sandboxes)
func (a *FluidAgent) listVMsFromHosts(ctx context.Context) (map[string]any, error) {
	listResult, err := a.multiHostMgr.ListDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list domains from hosts: %w", err)
	}

	result := make([]map[string]any, 0)
	for _, domain := range listResult.Domains {
		// Skip sandboxes (names starting with "sbx-")
		if strings.HasPrefix(domain.Name, "sbx-") {
			continue
		}
		item := map[string]any{
			"name":         domain.Name,
			"state":        domain.State.String(),
			"host":         domain.HostName,
			"host_address": domain.HostAddress,
		}
		if domain.UUID != "" {
			item["uuid"] = domain.UUID
		}
		result = append(result, item)
	}

	// Include any host errors in the response
	response := map[string]any{
		"vms":   result,
		"count": len(result),
	}

	if len(listResult.HostErrors) > 0 {
		errors := make([]map[string]any, 0, len(listResult.HostErrors))
		for _, he := range listResult.HostErrors {
			errors = append(errors, map[string]any{
				"host":    he.HostName,
				"address": he.HostAddress,
				"error":   he.Error,
			})
		}
		response["host_errors"] = errors
	}

	return response, nil
}

// listVMsLocal queries local virsh for VMs (excludes sandboxes)
func (a *FluidAgent) listVMsLocal(ctx context.Context) (map[string]any, error) {
	// Use virsh list --all --name to get all VMs
	cmd := exec.CommandContext(ctx, "virsh", "list", "--all", "--name")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("virsh list: %w: %s", err, stderr.String())
	}

	result := make([]map[string]any, 0)
	for _, name := range strings.Split(stdout.String(), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Skip sandboxes (names starting with "sbx-")
		if strings.HasPrefix(name, "sbx-") {
			continue
		}
		result = append(result, map[string]any{
			"name":  name,
			"state": "unknown",
			"host":  "local",
		})
	}

	return map[string]any{
		"vms":   result,
		"count": len(result),
	}, nil
}

func (a *FluidAgent) createSnapshot(ctx context.Context, sandboxID, name string) (map[string]any, error) {
	if name == "" {
		name = fmt.Sprintf("snap-%d", time.Now().Unix())
	}

	snap, err := a.vmService.CreateSnapshot(ctx, sandboxID, name, false)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"snapshot_id": snap.ID,
		"sandbox_id":  sandboxID,
		"name":        snap.Name,
		"kind":        snap.Kind,
	}, nil
}

// Formatting helpers

func (a *FluidAgent) formatVMsResult(result map[string]any, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list VMs: %v", err)
	}

	vms, ok := result["vms"].([]map[string]any)
	if !ok || len(vms) == 0 {
		return "No VMs found."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d VM(s) available for cloning:\n\n", len(vms)))

	// Group VMs by host if host information is present
	hostVMs := make(map[string][]map[string]any)
	for _, vm := range vms {
		host := "local"
		if h, ok := vm["host"].(string); ok && h != "" {
			host = h
		}
		hostVMs[host] = append(hostVMs[host], vm)
	}

	// Display VMs grouped by host
	for host, hvms := range hostVMs {
		if len(hostVMs) > 1 || host != "local" {
			b.WriteString(fmt.Sprintf("### Host: %s\n", host))
		}
		for _, vm := range hvms {
			state := "unknown"
			if s, ok := vm["state"].(string); ok {
				state = s
			}
			b.WriteString(fmt.Sprintf("- **%s** (%s)\n", vm["name"], state))
		}
		b.WriteString("\n")
	}

	// Display any host errors
	if hostErrors, ok := result["host_errors"].([]map[string]any); ok && len(hostErrors) > 0 {
		b.WriteString("### Host Errors\n")
		for _, he := range hostErrors {
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", he["host"], he["error"]))
		}
	}

	return b.String()
}

func (a *FluidAgent) formatSandboxesResult(result map[string]any, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list sandboxes: %v", err)
	}

	sandboxes, ok := result["sandboxes"].([]map[string]any)
	if !ok || len(sandboxes) == 0 {
		return "No sandboxes found."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d sandbox(es):\n\n", len(sandboxes)))

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
			b.WriteString(fmt.Sprintf("### Host: %s\n", host))
		}
		for _, sb := range sbs {
			state := "unknown"
			if s, ok := sb["state"].(string); ok {
				state = s
			}
			name := sb["name"]
			id := sb["id"]
			baseImage := sb["base_image"]

			b.WriteString(fmt.Sprintf("- **%s** (%s)\n", name, id))
			b.WriteString(fmt.Sprintf("  State: %s | Base: %s", state, baseImage))
			if ip, ok := sb["ip"].(string); ok {
				b.WriteString(fmt.Sprintf(" | IP: %s", ip))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// listHostsWithVMs queries all hosts and returns VMs differentiated by type (host VM vs sandbox)
func (a *FluidAgent) listHostsWithVMs(ctx context.Context) (map[string]any, error) {
	// Get sandboxes from database
	sandboxes, err := a.vmService.GetSandboxes(ctx, store.SandboxFilter{}, nil)
	if err != nil {
		return nil, fmt.Errorf("list sandboxes: %w", err)
	}

	// Build a set of sandbox names for quick lookup
	sandboxNames := make(map[string]bool)
	for _, sb := range sandboxes {
		sandboxNames[sb.SandboxName] = true
	}

	// Get all domains from libvirt
	var domains []map[string]any
	var hostErrors []map[string]any

	if a.multiHostMgr != nil {
		listResult, err := a.multiHostMgr.ListDomains(ctx)
		if err != nil {
			return nil, fmt.Errorf("list domains from hosts: %w", err)
		}
		for _, domain := range listResult.Domains {
			isSandbox := strings.HasPrefix(domain.Name, "sbx-") || sandboxNames[domain.Name]
			domains = append(domains, map[string]any{
				"name":         domain.Name,
				"state":        domain.State.String(),
				"host":         domain.HostName,
				"host_address": domain.HostAddress,
				"type":         vmType(isSandbox),
			})
		}
		for _, he := range listResult.HostErrors {
			hostErrors = append(hostErrors, map[string]any{
				"host":    he.HostName,
				"address": he.HostAddress,
				"error":   he.Error,
			})
		}
	} else {
		// Local virsh
		cmd := exec.CommandContext(ctx, "virsh", "list", "--all", "--name")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("virsh list: %w: %s", err, stderr.String())
		}

		for _, name := range strings.Split(stdout.String(), "\n") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			isSandbox := strings.HasPrefix(name, "sbx-") || sandboxNames[name]
			domains = append(domains, map[string]any{
				"name":  name,
				"state": "unknown",
				"host":  "local",
				"type":  vmType(isSandbox),
			})
		}
	}

	response := map[string]any{
		"domains": domains,
		"count":   len(domains),
	}
	if len(hostErrors) > 0 {
		response["host_errors"] = hostErrors
	}

	return response, nil
}

// vmType returns "sandbox" or "host_vm" based on whether the domain is a sandbox
func vmType(isSandbox bool) string {
	if isSandbox {
		return "sandbox"
	}
	return "host_vm"
}

func (a *FluidAgent) formatHostsResult(result map[string]any, err error) string {
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
	b.WriteString(fmt.Sprintf("Total: %d host VM(s), %d sandbox(es)\n\n", totalHostVMs, totalSandboxes))

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

		b.WriteString(fmt.Sprintf("## %s\n", host))
		b.WriteString(fmt.Sprintf("Host VMs: %d | Sandboxes: %d\n\n", hostVMCount, sandboxCount))

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
				b.WriteString(fmt.Sprintf("- %s (%s)\n", d["name"], state))
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
				b.WriteString(fmt.Sprintf("- %s (%s)\n", d["name"], state))
			}
			b.WriteString("\n")
		}
	}

	// Display any host errors
	if hostErrors, ok := result["host_errors"].([]map[string]any); ok && len(hostErrors) > 0 {
		b.WriteString("## Host Errors\n")
		for _, he := range hostErrors {
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", he["host"], he["error"]))
		}
	}

	return b.String()
}

func (a *FluidAgent) listPlaybooks(ctx context.Context) (map[string]any, error) {
	playbooks, err := a.playbookService.ListPlaybooks(ctx, nil)
	if err != nil {
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

func (a *FluidAgent) formatPlaybooksResult(result map[string]any, err error) string {
	if err != nil {
		return fmt.Sprintf("Failed to list playbooks: %v", err)
	}

	playbooks, ok := result["playbooks"].([]map[string]any)
	if !ok || len(playbooks) == 0 {
		return "No playbooks found."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d playbook(s):\n\n", len(playbooks)))
	for _, pb := range playbooks {
		name := pb["name"].(string)
		path := pb["path"].(string)

		absPath, _ := filepath.Abs(path)
		// OSC 8 hyperlink
		link := fmt.Sprintf("\033]8;;file://%s\033\\%s\033]8;;\033\\", absPath, path)

		b.WriteString(fmt.Sprintf("- **%s**: %s\n", name, link))
	}
	return b.String()
}

// Cleanup destroys all sandboxes created during this session.
// This is called when the TUI exits to ensure no orphaned VMs are left running.
func (a *FluidAgent) Cleanup(ctx context.Context) error {
	if len(a.createdSandboxes) == 0 {
		return nil
	}

	var errs []error
	for _, id := range a.createdSandboxes {
		// Check if sandbox still exists before destroying
		if _, err := a.vmService.GetSandbox(ctx, id); err != nil {
			// Sandbox no longer exists (already destroyed by user), skip
			continue
		}

		if _, err := a.vmService.DestroySandbox(ctx, id); err != nil {
			errs = append(errs, fmt.Errorf("destroy sandbox %s: %w", id, err))
			// Continue trying to destroy others even if one fails
		}
	}

	// Clear the list
	a.createdSandboxes = nil

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}
	return nil
}

// CreatedSandboxCount returns the number of sandboxes created during this session.
func (a *FluidAgent) CreatedSandboxCount() int {
	return len(a.createdSandboxes)
}

// GetCreatedSandboxes returns a copy of the sandbox IDs created during this session.
func (a *FluidAgent) GetCreatedSandboxes() []string {
	result := make([]string, len(a.createdSandboxes))
	copy(result, a.createdSandboxes)
	return result
}

// ClearCreatedSandboxes clears the list of created sandboxes.
func (a *FluidAgent) ClearCreatedSandboxes() {
	a.createdSandboxes = nil
}

// CleanupWithProgress destroys all sandboxes, sending progress updates through the status callback.
// Each sandbox gets its own 60-second timeout to avoid one slow destroy blocking others.
func (a *FluidAgent) CleanupWithProgress(sandboxIDs []string) {
	total := len(sandboxIDs)
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
		if _, err := a.vmService.GetSandbox(ctx, id); err != nil {
			// Already destroyed
			cancel()
			skipped++
			a.sendStatus(CleanupProgressMsg{
				SandboxID: id,
				Status:    CleanupStatusSkipped,
			})
			continue
		}

		// Destroy the sandbox
		if _, err := a.vmService.DestroySandbox(ctx, id); err != nil {
			failed++
			a.sendStatus(CleanupProgressMsg{
				SandboxID: id,
				Status:    CleanupStatusFailed,
				Error:     err.Error(),
			})
		} else {
			destroyed++
			a.sendStatus(CleanupProgressMsg{
				SandboxID: id,
				Status:    CleanupStatusDestroyed,
			})
		}
		cancel()
	}

	// Clear the created sandboxes list
	a.createdSandboxes = nil

	// Send completion message
	a.sendStatus(CleanupCompleteMsg{
		Total:     total,
		Destroyed: destroyed,
		Failed:    failed,
		Skipped:   skipped,
	})
}

// GetCurrentSandbox returns the currently active sandbox ID and host
func (a *FluidAgent) GetCurrentSandbox() (id string, host string) {
	return a.currentSandboxID, a.currentSandboxHost
}

// SetCurrentSandbox sets the currently active sandbox
func (a *FluidAgent) SetCurrentSandbox(id string, host string) {
	a.currentSandboxID = id
	a.currentSandboxHost = host
}
