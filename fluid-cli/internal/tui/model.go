package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/updater"
)

// State represents the current state of the TUI
type State int

const (
	StateIdle State = iota
	StateThinking
	StateAwaitingReview
	StateSettings
	StateMemoryApproval
)

// ConversationEntry represents a single entry in the conversation
type ConversationEntry struct {
	Role    string // "user", "assistant", "tool", "system"
	Content string
	Tool    *ToolResult
}

// Model is the main Bubble Tea model for the TUI
type Model struct {
	// UI components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model
	styles   Styles

	// State
	state        State
	conversation []ConversationEntry
	thinking     bool
	thinkingDots int
	quitting     bool
	currentInput string // The last command entered by the user

	// Agent activity status
	agentStatus     AgentStatus
	currentToolName string
	currentToolArgs map[string]any

	// Status channel for agent updates
	statusChan chan tea.Msg

	// Dimensions
	width  int
	height int
	ready  bool

	// Configuration
	title      string
	provider   string
	model      string
	cfg        *config.Config
	configPath string

	// Banner state
	showBanner bool // Show startup banner (until first message)

	// Settings screen
	settingsModel SettingsModel
	inSettings    bool

	// Memory approval dialog
	confirmModel    ConfirmModel
	inMemoryConfirm bool
	approvalChan    chan<- MemoryApprovalResult

	// Network approval dialog
	networkConfirmModel NetworkConfirmModel
	inNetworkConfirm    bool
	networkApprovalChan chan<- NetworkApprovalResult

	// Source prepare approval dialog
	sourcePrepareConfirmModel SourcePrepareConfirmModel
	inSourcePrepareConfirm    bool
	sourcePrepareApprovalChan chan<- SourcePrepareApprovalResult

	// Agent
	agentRunner AgentRunner
	readOnly    bool

	// Playbooks browser
	playbooksModel PlaybooksModel
	inPlaybooks    bool

	// Autocomplete
	suggestions     []commandSuggestion
	suggestionIndex int

	// History
	historyPath  string
	historyList  []string
	historyIndex int

	// Live command output (inline with conversation)
	liveOutputLines   []string
	liveOutputPending string
	showingLiveOutput bool
	liveOutputSandbox string
	liveOutputIndex   int // Index in conversation where live output is displayed
	currentRetry      *RetryAttemptMsg

	// Live prepare progress (inline with conversation, like live command output)
	showingLivePrepare  bool
	livePrepareSourceVM string
	livePrepareSteps    []string // completed/in-progress step descriptions
	livePrepareIndex    int      // index in conversation

	// Markdown renderer
	mdRenderer *glamour.TermRenderer

	// Cleanup state
	inCleanup       bool
	cleanupStatuses map[string]CleanupStatus // sandbox ID -> status
	cleanupErrors   map[string]string        // sandbox ID -> error message
	cleanupOrder    []string                 // ordered list of sandbox IDs
	cleanupDone     bool                     // true when cleanup is complete
	cleanupResult   *CleanupCompleteMsg      // final cleanup results
}

type commandSuggestion struct {
	name        string
	description string
}

var allCommands = []commandSuggestion{
	{"/vms", "List available VMs for cloning"},
	{"/sandboxes", "List active sandboxes"},
	{"/hosts", "List configured remote hosts"},
	{"/playbooks", "List generated Ansible playbooks"},
	{"/compact", "Summarize and compact conversation history"},
	{"/context", "Show current context token usage"},
	{"/settings", "Open configuration settings"},
	{"/clear", "Clear conversation history"},
	{"/help", "Show available commands"},
}

// AgentRunner is the interface for running agent commands
type AgentRunner interface {
	Run(input string) tea.Cmd
	Reset()
	// SetStatusCallback sets a callback for status updates during execution
	SetStatusCallback(func(tea.Msg))
	// SetReadOnly toggles read-only mode (only query tools available)
	SetReadOnly(bool)
}

// NewModel creates a new TUI model
func NewModel(title, provider, modelName string, runner AgentRunner, cfg *config.Config, configPath string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (type /settings to configure)"
	ta.Focus()
	ta.Prompt = ""
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))

	// Create markdown renderer
	mdRenderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	// Build startup message
	startupMsg := "Welcome to Fluid! Type '/help' for commands."

	// if len(cfg.Hosts) > 0 {
	// 	hostNames := make([]string, len(cfg.Hosts))
	// 	for i, h := range cfg.Hosts {
	// 		hostNames[i] = h.Name
	// 	}
	// 	startupMsg = fmt.Sprintf("Connected with %d remote hosts: %s. Type '/hosts', '/vms', or '/clear' to query or reset.",
	// 		len(cfg.Hosts), strings.Join(hostNames, ", "))
	// }

	// Create status channel for agent updates (larger buffer to prevent dropped messages)
	statusChan := make(chan tea.Msg, 100)

	// Load persistent history
	historyPath := HistoryPath()
	historyList := LoadHistory(historyPath)
	if historyList == nil {
		historyList = make([]string, 0)
	}

	m := Model{
		textarea:          ta,
		spinner:           s,
		styles:            DefaultStyles(),
		state:             StateIdle,
		conversation:      make([]ConversationEntry, 0),
		title:             title,
		provider:          provider,
		model:             modelName,
		cfg:               cfg,
		configPath:        configPath,
		agentRunner:       runner,
		mdRenderer:        mdRenderer,
		statusChan:        statusChan,
		historyPath:       historyPath,
		historyList:       historyList,
		historyIndex:      len(historyList),
		liveOutputLines:   make([]string, 0),
		liveOutputPending: "",
		showBanner:        true, // Show banner until first user message
	}

	// Set up status callback for the agent
	if runner != nil {
		runner.SetStatusCallback(func(msg tea.Msg) {
			select {
			case statusChan <- msg:
			default:
				// Channel full, drop message
			}
		})
	}

	// Add startup message
	m.conversation = append(m.conversation, ConversationEntry{
		Role:    "system",
		Content: startupMsg,
	})

	return m
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
		checkForUpdate(),
	)
}

// checkForUpdate spawns a background goroutine that checks for updates.
// Only runs once per 24 hours. Fails silently.
func checkForUpdate() tea.Cmd {
	return func() tea.Msg {
		if !updater.ShouldCheck() {
			return nil
		}
		latest, _, needsUpdate, err := updater.CheckLatest(Version)
		updater.MarkChecked()
		if err != nil || !needsUpdate {
			return nil
		}
		return UpdateAvailableMsg{Version: latest}
	}
}

// listenForStatus returns a command that listens for status updates from the agent
func (m Model) listenForStatus() tea.Cmd {
	return func() tea.Msg {
		return <-m.statusChan
	}
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle SettingsCloseMsg first, before delegating to settings
	if closeMsg, ok := msg.(SettingsCloseMsg); ok {
		m.inSettings = false
		m.state = StateIdle
		if closeMsg.Saved {
			m.cfg = m.settingsModel.GetConfig()
			m.addSystemMessage("Settings saved. Some changes may require restart.")
		} else {
			m.addSystemMessage("Settings cancelled.")
		}
		m.updateViewportContent(false)
		m.textarea.Focus()
		return m, nil
	}

	// Handle PlaybooksCloseMsg
	if _, ok := msg.(PlaybooksCloseMsg); ok {
		m.inPlaybooks = false
		m.state = StateIdle
		m.updateViewportContent(false)
		m.textarea.Focus()
		return m, nil
	}

	// If in playbooks mode, delegate to playbooks model
	if m.inPlaybooks {
		var cmd tea.Cmd
		playbooksModel, cmd := m.playbooksModel.Update(msg)
		m.playbooksModel = playbooksModel.(PlaybooksModel)
		return m, cmd
	}

	// If in settings mode, delegate to settings model
	if m.inSettings {
		var cmd tea.Cmd
		settingsModel, cmd := m.settingsModel.Update(msg)
		m.settingsModel = settingsModel.(SettingsModel)
		return m, cmd
	}

	// Handle memory approval response first, before delegating to confirm model
	if approvalResp, ok := msg.(MemoryApprovalResponseMsg); ok {
		m.inMemoryConfirm = false
		m.state = StateThinking // Go back to thinking while agent processes
		m.thinking = true
		m.thinkingDots = 0

		// Send response to the agent
		if agent, ok := m.agentRunner.(*FluidAgent); ok {
			agent.HandleApprovalResponse(approvalResp.Result.Approved)
		}

		if approvalResp.Result.Approved {
			m.addSystemMessage("Memory warning acknowledged. Proceeding with sandbox creation...")
		} else {
			m.addSystemMessage("Sandbox creation cancelled due to insufficient memory.")
		}

		m.updateViewportContent(true)
		// Restart both the thinking animation and status listener
		return m, tea.Batch(ThinkingCmd(), m.listenForStatus())
	}

	// Handle network approval response
	if networkResp, ok := msg.(NetworkApprovalResponseMsg); ok {
		m.inNetworkConfirm = false
		m.state = StateThinking // Go back to thinking while agent processes
		m.thinking = true
		m.thinkingDots = 0

		// Send response to the agent
		if agent, ok := m.agentRunner.(*FluidAgent); ok {
			agent.HandleNetworkApprovalResponse(networkResp.Result.Approved)
		}

		if networkResp.Result.Approved {
			m.addSystemMessage("Network access approved. Proceeding with command...")
		} else {
			m.addSystemMessage("Network access denied.")
		}

		m.updateViewportContent(true)
		// Restart both the thinking animation and status listener
		return m, tea.Batch(ThinkingCmd(), m.listenForStatus())
	}

	// Handle source prepare approval response
	if spResp, ok := msg.(SourcePrepareApprovalResponseMsg); ok {
		m.inSourcePrepareConfirm = false
		m.state = StateThinking
		m.thinking = true
		m.thinkingDots = 0

		if agent, ok := m.agentRunner.(*FluidAgent); ok {
			agent.HandleSourcePrepareApprovalResponse(spResp.Result.Approved)
		}

		if spResp.Result.Approved {
			m.addSystemMessage("Preparing source VM for read-only access...")
		} else {
			m.addSystemMessage("Source VM preparation declined.")
		}

		m.updateViewportContent(true)
		return m, tea.Batch(ThinkingCmd(), m.listenForStatus())
	}

	// If in memory confirmation mode, delegate to confirm model
	if m.inMemoryConfirm {
		var cmd tea.Cmd
		confirmModel, cmd := m.confirmModel.Update(msg)
		m.confirmModel = confirmModel.(ConfirmModel)
		return m, cmd
	}

	// If in network confirmation mode, delegate to network confirm model
	if m.inNetworkConfirm {
		var cmd tea.Cmd
		networkModel, cmd := m.networkConfirmModel.Update(msg)
		m.networkConfirmModel = networkModel.(NetworkConfirmModel)
		return m, cmd
	}

	// If in source prepare confirmation mode, delegate to source prepare confirm model
	if m.inSourcePrepareConfirm {
		var cmd tea.Cmd
		spModel, cmd := m.sourcePrepareConfirmModel.Update(msg)
		m.sourcePrepareConfirmModel = spModel.(SourcePrepareConfirmModel)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if !m.inSettings && !m.inPlaybooks && !m.inMemoryConfirm &&
			!m.inNetworkConfirm && !m.inSourcePrepareConfirm && !m.inCleanup {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.viewport.ScrollUp(3)
			case tea.MouseButtonWheelDown:
				m.viewport.ScrollDown(3)
			}
		}
		return m, nil

	case tea.KeyMsg:
		keyStr := msg.String()

		if m.quitting && keyStr != "ctrl+c" {
			m.quitting = false
		}

		// Handle autocomplete navigation if suggestions are visible
		if len(m.suggestions) > 0 {
			switch keyStr {
			case "tab":
				m.textarea.SetValue(m.suggestions[m.suggestionIndex].name + " ")
				m.textarea.SetCursor(len(m.textarea.Value()))
				m.suggestions = nil
				return m, nil
			case "up":
				m.suggestionIndex--
				if m.suggestionIndex < 0 {
					m.suggestionIndex = len(m.suggestions) - 1
				}
				return m, nil
			case "down":
				m.suggestionIndex++
				if m.suggestionIndex >= len(m.suggestions) {
					m.suggestionIndex = 0
				}
				return m, nil
			case "esc":
				m.suggestions = nil
				return m, nil
			}
		} else {
			// Handle command history navigation
			switch keyStr {
			case "up":
				if len(m.historyList) > 0 && m.historyIndex > 0 {
					m.historyIndex--
					m.textarea.SetValue(m.historyList[m.historyIndex])
					m.textarea.SetCursor(len(m.textarea.Value()))
					return m, nil
				}
			case "down":
				if len(m.historyList) > 0 && m.historyIndex < len(m.historyList) {
					m.historyIndex++
					if m.historyIndex == len(m.historyList) {
						m.textarea.Reset()
					} else {
						m.textarea.SetValue(m.historyList[m.historyIndex])
						m.textarea.SetCursor(len(m.textarea.Value()))
					}
					return m, nil
				}
			}
		}

		switch keyStr {
		case "pgup":
			m.viewport.PageUp()
			return m, nil
		case "pgdown":
			m.viewport.PageDown()
			return m, nil
		case "shift+tab":
			m.readOnly = !m.readOnly
			if m.agentRunner != nil {
				m.agentRunner.SetReadOnly(m.readOnly)
			}
			// Clear auto read-only so manual toggle sticks
			if agent, ok := m.agentRunner.(*FluidAgent); ok {
				agent.ClearAutoReadOnly()
			}
			mode := "edit"
			if m.readOnly {
				mode = "read-only"
			}
			m.addSystemMessage(fmt.Sprintf("Switched to %s mode.", mode))
			m.updateViewportContent(false)
			return m, nil
		case "ctrl+c":
			// If already in cleanup, allow force quit
			if m.inCleanup {
				return m, tea.Quit
			}
			if m.textarea.Value() != "" {
				m.textarea.Reset()
				return m, nil
			}
			if !m.quitting {
				m.quitting = true
				m.conversation = make([]ConversationEntry, 0)
				m.updateViewportContent(true)
				return m, nil
			}
			// Second ctrl-c with quitting=true: start cleanup
			if agent, ok := m.agentRunner.(*FluidAgent); ok {
				sandboxes := agent.GetCreatedSandboxes()
				if len(sandboxes) > 0 {
					m.inCleanup = true
					m.cleanupOrder = sandboxes
					m.cleanupStatuses = make(map[string]CleanupStatus)
					m.cleanupErrors = make(map[string]string)
					for _, id := range sandboxes {
						m.cleanupStatuses[id] = CleanupStatusPending
					}
					return m, m.startCleanup()
				}
			}
			return m, tea.Quit
		case "ctrl+r":
			m.conversation = make([]ConversationEntry, 0)
			m.addSystemMessage("Conversation reset.")
			if m.agentRunner != nil {
				m.agentRunner.Reset()
			}
			m.updateViewportContent(true)
			return m, nil
		case "enter":
			if m.state == StateIdle && m.textarea.Value() != "" {
				input := strings.TrimSpace(m.textarea.Value())
				m.textarea.Reset()
				m.suggestions = nil

				// Add to history (in-memory + persistent)
				if len(m.historyList) == 0 || m.historyList[len(m.historyList)-1] != input {
					m.historyList = append(m.historyList, input)
					AppendHistory(m.historyPath, input)
				}
				m.historyIndex = len(m.historyList)

				m.currentInput = input

				// Handle /settings command
				if input == "/settings" || input == "settings" {
					m.inSettings = true
					m.settingsModel = NewSettingsModel(m.cfg, m.configPath)
					return m, m.settingsModel.Init()
				}

				// Handle /clear command
				if input == "/clear" || input == "clear" {
					m.conversation = make([]ConversationEntry, 0)
					if m.agentRunner != nil {
						m.agentRunner.Reset()
					}
					m.addSystemMessage("Conversation cleared.")
					m.updateViewportContent(true)
					return m, nil
				}

				// Handle /playbooks command - open playbooks browser
				if input == "/playbooks" || input == "playbooks" {
					if agent, ok := m.agentRunner.(*FluidAgent); ok {
						m.inPlaybooks = true
						m.playbooksModel = NewPlaybooksModel(agent.playbookService)
						if err := m.playbooksModel.LoadPlaybooks(); err != nil {
							m.inPlaybooks = false
							m.addSystemMessage(fmt.Sprintf("Failed to load playbooks: %v", err))
							m.updateViewportContent(true)
							return m, nil
						}
						// Send window size to playbooks model
						if m.width > 0 && m.height > 0 {
							playbooksModel, _ := m.playbooksModel.Update(tea.WindowSizeMsg{
								Width:  m.width,
								Height: m.height,
							})
							m.playbooksModel = playbooksModel.(PlaybooksModel)
						}
						return m, nil
					}
					m.addSystemMessage("Playbooks browser not available.")
					m.updateViewportContent(true)
					return m, nil
				}

				// Add user message
				m.addUserMessage(input)

				// Start thinking
				m.state = StateThinking
				m.thinking = true
				m.thinkingDots = 0
				m.updateViewportContent(true)

				// Run agent
				if m.agentRunner != nil {
					return m, tea.Batch(
						m.agentRunner.Run(input),
						ThinkingCmd(),
						m.listenForStatus(),
					)
				}
			}
		case "esc":
			if m.state == StateSettings {
				m.state = StateIdle
				m.textarea.Focus()
			}
		}

	case SettingsCloseMsg:
		m.inSettings = false
		m.state = StateIdle
		if msg.Saved {
			m.cfg = m.settingsModel.GetConfig()
			m.addSystemMessage("Settings saved. Some changes may require restart.")
		} else {
			m.addSystemMessage("Settings cancelled.")
		}
		m.updateViewportContent(false)
		m.textarea.Focus()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate banner height if shown
		bannerHeight := 0
		if m.showBanner && m.state == StateIdle {
			bannerHeight = GetBannerLogoHeight() + 2 // Logo lines + spacing
		}

		// Calculate suggestion height if shown
		suggestionHeight := 0
		if len(m.suggestions) > 0 {
			suggestionHeight = len(m.suggestions)
			if suggestionHeight > 5 {
				suggestionHeight = 6 // 5 items + "... more" line
			}
			suggestionHeight += 2 // border
		}

		// inputHeight depends on content, but initially or on resize we use current textarea height
		// +2 accounts for the border (top and bottom)
		inputHeight := m.textarea.Height() + 2
		statusBarHeight := 1 // Bottom status bar
		conversationHeight := m.height - bannerHeight - inputHeight - statusBarHeight - suggestionHeight

		if !m.ready {
			m.viewport = viewport.New(m.width, conversationHeight)
			m.viewport.YPosition = bannerHeight
			m.viewport.KeyMap = viewport.KeyMap{}
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = conversationHeight
		}

		m.textarea.SetWidth(m.width - 4)
		m.updateViewportContent(false)

	case ThinkingTickMsg:
		if m.thinking {
			m.thinkingDots = (m.thinkingDots + 1) % 4
			m.updateViewportContent(false)
			return m, ThinkingCmd()
		}

	case AgentDoneMsg:
		// Agent finished, don't restart the status listener
		return m, nil

	case ToolStartMsg:
		m.agentStatus = StatusWorking
		m.currentToolName = msg.ToolName
		m.currentToolArgs = msg.Args
		m.updateViewportContent(false)
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case ToolCompleteMsg:
		// Add tool result to conversation
		tr := ToolResult{
			Name:   msg.ToolName,
			Args:   m.currentToolArgs, // Capture args from when tool started
			Result: msg.Result,
			Error:  !msg.Success,
		}
		if msg.Error != "" {
			tr.ErrorMsg = msg.Error
		}
		m.addToolResult(tr)
		// Switch back to thinking while waiting for next LLM response
		m.agentStatus = StatusThinking
		m.currentToolName = ""
		m.currentToolArgs = nil
		m.updateViewportContent(false)
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case CommandOutputResetMsg:
		// Reset live output (e.g., on retry)
		if m.showingLiveOutput && m.liveOutputSandbox == msg.SandboxID {
			m.liveOutputLines = nil
			m.liveOutputPending = ""
			if m.liveOutputIndex < len(m.conversation) {
				m.conversation[m.liveOutputIndex].Content = "(retrying...)"
			}
			m.updateViewportContent(false)
		}
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case RetryAttemptMsg:
		m.currentRetry = &msg
		m.updateViewportContent(false)
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case CommandOutputChunkMsg:
		// Clear retry when output arrives
		m.currentRetry = nil

		// Filter SSH warning from chunk
		chunk := filterSSHWarning(msg.Chunk)
		if chunk == "" {
			// Nothing left after filtering, just continue listening
			return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)
		}

		if !m.showingLiveOutput {
			// Add a new conversation entry for live output
			m.showingLiveOutput = true
			m.liveOutputSandbox = msg.SandboxID
			m.liveOutputLines = nil
			m.liveOutputPending = ""
			m.liveOutputIndex = len(m.conversation)
			m.conversation = append(m.conversation, ConversationEntry{
				Role:    "live_output",
				Content: "",
			})
		}

		// Append chunk to pending
		m.liveOutputPending += chunk

		// Process complete lines
		if strings.Contains(m.liveOutputPending, "\n") {
			lines := strings.Split(m.liveOutputPending, "\n")
			// The last element is the new pending buffer (after the last \n)
			if len(lines) > 0 {
				complete := lines[:len(lines)-1]
				// Filter SSH warnings from completed lines
				for _, line := range complete {
					if !isSSHWarningLine(line) {
						m.liveOutputLines = append(m.liveOutputLines, line)
					}
				}
				m.liveOutputPending = lines[len(lines)-1]

				// Keep buffer size manageable (last 100 lines is plenty since we only show 20)
				if len(m.liveOutputLines) > 100 {
					m.liveOutputLines = m.liveOutputLines[len(m.liveOutputLines)-100:]
				}
			}
		}

		// Update the live output entry in place
		if m.liveOutputIndex < len(m.conversation) {
			m.conversation[m.liveOutputIndex].Content = m.formatLiveOutput()
		}
		m.updateViewportContent(false)
		m.viewport.GotoBottom() // Auto-scroll
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case CommandOutputDoneMsg:
		m.showingLiveOutput = false
		m.currentRetry = nil
		// Clear output buffer as it's no longer needed (result will be shown via ToolCompleteMsg)
		m.liveOutputLines = nil
		m.liveOutputPending = ""
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case SourcePrepareProgressMsg:
		if !m.showingLivePrepare {
			m.showingLivePrepare = true
			m.livePrepareSourceVM = msg.SourceVM
			m.livePrepareSteps = nil
			m.livePrepareIndex = len(m.conversation)
			m.conversation = append(m.conversation, ConversationEntry{
				Role:    "live_prepare",
				Content: "",
			})
		}

		if msg.Done {
			// Mark the last step as done
			if len(m.livePrepareSteps) > 0 {
				m.livePrepareSteps[len(m.livePrepareSteps)-1] = fmt.Sprintf("  v [%d/%d] %s", msg.StepNum, msg.Total, msg.StepName)
			}
			// If this was the last step, close the live box
			if msg.StepNum == msg.Total {
				m.showingLivePrepare = false
			}
		} else {
			// New step starting
			m.livePrepareSteps = append(m.livePrepareSteps, fmt.Sprintf("  . [%d/%d] %s...", msg.StepNum, msg.Total, msg.StepName))
		}

		// Update the conversation entry in place
		if m.livePrepareIndex < len(m.conversation) {
			m.conversation[m.livePrepareIndex].Content = strings.Join(m.livePrepareSteps, "\n")
		}
		m.updateViewportContent(false)
		m.viewport.GotoBottom()
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case AutoReadOnlyMsg:
		m.readOnly = msg.Enabled
		if msg.Enabled {
			m.addSystemMessage(fmt.Sprintf("Auto read-only: accessing source VM %s", msg.SourceVM))
		} else {
			m.addSystemMessage("Auto read-only: restored edit mode")
		}
		m.updateViewportContent(false)
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case AgentResponseMsg:
		// Add assistant message (tool results were already sent via ToolCompleteMsg)
		if msg.Response.Content != "" {
			m.addAssistantMessage(msg.Response.Content)
		}

		if !msg.Response.Done {
			m.updateViewportContent(true)
			return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)
		}

		m.thinking = false
		m.state = StateIdle
		m.agentStatus = StatusThinking
		m.currentToolName = ""

		// Check for review request or completion
		if msg.Response.AwaitingInput {
			// Handle review - we'd need more context here
			m.state = StateAwaitingReview
		}

		m.updateViewportContent(true)
		m.textarea.Focus()
		return m, nil

	case AgentErrorMsg:
		m.thinking = false
		m.state = StateIdle
		m.addSystemMessage(fmt.Sprintf("Error: %v", msg.Err))
		m.updateViewportContent(true)
		m.textarea.Focus()
		return m, nil

	case CompactStartMsg:
		m.addSystemMessage("Compacting conversation context...")
		m.updateViewportContent(false)
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case CompactCompleteMsg:
		m.addSystemMessage(fmt.Sprintf("Context compacted: %d -> %d tokens (%.1f%% reduction)",
			msg.PreviousTokens, msg.NewTokens,
			100*(1-float64(msg.NewTokens)/float64(msg.PreviousTokens))))
		m.updateViewportContent(false)
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case CompactErrorMsg:
		m.addSystemMessage(fmt.Sprintf("Compaction warning: %v", msg.Err))
		m.updateViewportContent(false)
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case ReviewResponseMsg:
		m.state = StateIdle
		if msg.Approved {
			m.addSystemMessage("Review approved.")
			if m.agentRunner != nil {
				m.state = StateThinking
				m.thinking = true
				return m, tea.Batch(
					m.agentRunner.Run("Review approved by human. You may proceed."),
					ThinkingCmd(),
				)
			}
		} else {
			m.addSystemMessage("Review rejected. Please provide feedback.")
		}
		m.textarea.Focus()
		m.updateViewportContent(true)
		return m, nil

	case MemoryApprovalRequestMsg:
		// Show the memory approval confirmation dialog
		m.inMemoryConfirm = true
		m.state = StateMemoryApproval
		m.thinking = false

		// Create result channel for the confirmation
		resultChan := make(chan MemoryApprovalResult, 1)
		m.approvalChan = resultChan
		m.confirmModel = NewConfirmModel(msg.Request, resultChan)

		// Update dimensions for the confirm model
		if m.width > 0 && m.height > 0 {
			confirmModel, _ := m.confirmModel.Update(tea.WindowSizeMsg{
				Width:  m.width,
				Height: m.height,
			})
			m.confirmModel = confirmModel.(ConfirmModel)
		}

		return m, nil

	case NetworkApprovalRequestMsg:
		// Show the network approval confirmation dialog
		m.inNetworkConfirm = true
		m.state = StateMemoryApproval // Reuse the same state for approval dialogs
		m.thinking = false

		// Create result channel for the confirmation
		resultChan := make(chan NetworkApprovalResult, 1)
		m.networkApprovalChan = resultChan
		m.networkConfirmModel = NewNetworkConfirmModel(msg.Request, resultChan)

		// Update dimensions for the confirm model
		if m.width > 0 && m.height > 0 {
			networkModel, _ := m.networkConfirmModel.Update(tea.WindowSizeMsg{
				Width:  m.width,
				Height: m.height,
			})
			m.networkConfirmModel = networkModel.(NetworkConfirmModel)
		}

		return m, nil

	case SourcePrepareApprovalRequestMsg:
		// Show the source prepare approval confirmation dialog
		m.inSourcePrepareConfirm = true
		m.state = StateMemoryApproval
		m.thinking = false

		resultChan := make(chan SourcePrepareApprovalResult, 1)
		m.sourcePrepareApprovalChan = resultChan
		m.sourcePrepareConfirmModel = NewSourcePrepareConfirmModel(msg.Request, resultChan)

		if m.width > 0 && m.height > 0 {
			spModel, _ := m.sourcePrepareConfirmModel.Update(tea.WindowSizeMsg{
				Width:  m.width,
				Height: m.height,
			})
			m.sourcePrepareConfirmModel = spModel.(SourcePrepareConfirmModel)
		}

		return m, nil

	case UpdateAvailableMsg:
		m.addSystemMessage(fmt.Sprintf("Update available: v%s - run `fluid update`", msg.Version))
		m.updateViewportContent(false)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case CleanupProgressMsg:
		m.cleanupStatuses[msg.SandboxID] = msg.Status
		if msg.Error != "" {
			m.cleanupErrors[msg.SandboxID] = msg.Error
		}
		// Continue listening for more updates and keep spinner going
		return m, tea.Batch(m.listenForStatus(), m.spinner.Tick)

	case CleanupCompleteMsg:
		m.cleanupDone = true
		m.cleanupResult = &msg
		// Give user a moment to see the results, then quit
		return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return tea.Quit()
		})

	}

	// Update textarea (skip for mouse events handled above)
	var taCmd tea.Cmd
	m.textarea, taCmd = m.textarea.Update(msg)
	cmds = append(cmds, taCmd)

	// Update autocomplete suggestions
	m.updateSuggestions()

	// Dynamic height adjustment for textarea
	lineCount := m.textarea.LineCount()
	if lineCount < 1 {
		lineCount = 1
	}

	// Calculate a dynamic maximum height for the input box.
	// We want to keep at least some lines visible for the conversation history.
	// Reserve space for: Header (1), Viewport min (2), Help (1), Suggestions (var), and margins (2).
	reservedHeight := 6
	if len(m.suggestions) > 0 {
		suggestionHeight := len(m.suggestions)
		if suggestionHeight > 5 {
			suggestionHeight = 6
		}
		reservedHeight += suggestionHeight + 2
	}

	maxInputHeight := m.height - reservedHeight
	if maxInputHeight < 1 {
		maxInputHeight = 1
	}
	// Also cap it at 50% of screen height to ensure conversation remains visible
	if maxInputHeight > m.height/2 && m.height > 10 {
		maxInputHeight = m.height / 2
	}

	if lineCount > maxInputHeight {
		lineCount = maxInputHeight
	}

	if lineCount != m.textarea.Height() {
		m.textarea.SetHeight(lineCount)
	}

	// Always recalculate viewport height to stay in sync with View()
	bannerHeight := 0
	if m.showBanner && m.state == StateIdle {
		bannerHeight = GetBannerLogoHeight() + 2
	}
	inputHeight := lineCount + 2 // textarea lines + border
	statusBarHeight := 1
	suggestionHeight := 0
	if len(m.suggestions) > 0 {
		suggestionHeight = len(m.suggestions)
		if suggestionHeight > 5 {
			suggestionHeight = 6 // 5 items + "... more" line
		}
		suggestionHeight += 2 // border
	}
	conversationHeight := m.height - bannerHeight - inputHeight - statusBarHeight - suggestionHeight
	if conversationHeight > 0 {
		m.viewport.Height = conversationHeight
	}

	// Update viewport on resize events
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m Model) View() string {
	// Show cleanup page if cleaning up
	if m.inCleanup {
		return m.renderCleanupView()
	}

	// Show memory approval dialog if in confirmation mode
	if m.inMemoryConfirm {
		return m.confirmModel.View()
	}

	// Show network approval dialog if in confirmation mode
	if m.inNetworkConfirm {
		return m.networkConfirmModel.View()
	}

	// Show source prepare approval dialog if in confirmation mode
	if m.inSourcePrepareConfirm {
		return m.sourcePrepareConfirmModel.View()
	}

	// Show settings screen if in settings mode
	if m.inSettings {
		return m.settingsModel.View()
	}

	// Show playbooks browser if in playbooks mode
	if m.inPlaybooks {
		return m.playbooksModel.View()
	}

	if !m.ready {
		return "Initializing..."
	}

	// Build banner (if shown)
	var banner string
	bannerHeight := 0
	if m.showBanner && m.state == StateIdle && !m.quitting {
		modelName := m.model
		if m.cfg != nil && m.cfg.AIAgent.Model != "" {
			modelName = m.cfg.AIAgent.Model
		}
		hostsString := "No Hosts Configured"
		if m.cfg != nil && len(m.cfg.Hosts) > 0 {
			hosts := make([]string, 0, len(m.cfg.Hosts))
			for _, h := range m.cfg.Hosts {
				hosts = append(hosts, h.Name)
			}
			if len(hosts) > 3 {
				hosts = append(hosts[:3], "...")
			}
			hostsString = strings.Join(hosts, ", ")
		}
		banner = RenderBanner(modelName, hostsString, m.provider, m.width)
		bannerHeight = lipgloss.Height(banner)
	}

	// Build suggestions menu (if any)
	var suggestions string
	suggestionHeight := 0
	if len(m.suggestions) > 0 {
		suggestionStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1).
			Width(m.width - 2)

		var sb strings.Builder
		displayCount := 0
		maxDisplay := 5

		startIdx := 0
		if m.suggestionIndex >= maxDisplay {
			startIdx = m.suggestionIndex - maxDisplay + 1
		}

		for i := startIdx; i < len(m.suggestions) && displayCount < maxDisplay; i++ {
			s := m.suggestions[i]
			name := s.name
			desc := s.description
			if i == m.suggestionIndex {
				sb.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(" > "+name) + "  " + lipgloss.NewStyle().Foreground(mutedColor).Render(desc) + "\n")
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(textColor).Render("   "+name) + "  " + lipgloss.NewStyle().Foreground(mutedColor).Render(desc) + "\n")
			}
			displayCount++
		}

		if len(m.suggestions) > maxDisplay {
			if startIdx+maxDisplay < len(m.suggestions) {
				sb.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Italic(true).Render(fmt.Sprintf("   ... %d more", len(m.suggestions)-(startIdx+maxDisplay))))
			}
		}

		suggestions = suggestionStyle.Render(strings.TrimSpace(sb.String()))
		suggestionHeight = lipgloss.Height(suggestions)
	}

	// Build input area
	inputBox := m.styles.Border.Width(m.width - 2).Render(
		m.styles.InputPrompt.Render("$ ") + m.textarea.View(),
	)
	inputHeight := lipgloss.Height(inputBox)

	// Build status bar
	sandboxID, sandboxHost := m.getCurrentSandbox()
	contextUsage := m.getContextUsage()
	modelName := m.model
	if m.cfg != nil && m.cfg.AIAgent.Model != "" {
		modelName = m.cfg.AIAgent.Model
	}
	sourceVM := m.getCurrentSourceVM()
	sandboxBaseImage := m.getCurrentSandboxBaseImage()
	statusBar := RenderStatusBarBottom(modelName, sandboxID, sandboxHost, sandboxBaseImage, sourceVM, contextUsage, m.readOnly, m.width)
	statusHeight := lipgloss.Height(statusBar)

	// Calculate viewport height to fill remaining space
	viewportHeight := m.height - bannerHeight - suggestionHeight - inputHeight - statusHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	// Build viewport content
	var viewportContent string
	if m.quitting {
		warnStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EAB308")).
			Width(m.width).
			Height(viewportHeight).
			Align(lipgloss.Right).
			AlignVertical(lipgloss.Bottom)
		viewportContent = warnStyle.Render("Press Ctrl+C again to close fluid and destroy all sandboxes created during this session.")
	} else {
		// Ensure viewport fills its allocated height
		viewportContent = lipgloss.NewStyle().
			Height(viewportHeight).
			Width(m.width).
			Render(m.viewport.View())
	}

	// Join all components vertically
	var parts []string
	if banner != "" {
		parts = append(parts, banner)
	}
	parts = append(parts, viewportContent)
	if suggestions != "" {
		parts = append(parts, suggestions)
	}
	parts = append(parts, inputBox)
	parts = append(parts, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// Helper methods

// getContextUsage returns the current context usage as a percentage (0.0 to 1.0)
// This includes any live output that's being streamed but not yet added to agent history
func (m *Model) getContextUsage() float64 {
	if agent, ok := m.agentRunner.(*FluidAgent); ok {
		baseUsage := agent.GetContextUsage()

		// Add estimate for live output not yet in history
		if m.showingLiveOutput {
			// Estimate tokens for live output: chars * tokensPerChar
			tokensPerChar := 0.25 // default
			if m.cfg != nil && m.cfg.AIAgent.TokensPerChar > 0 {
				tokensPerChar = m.cfg.AIAgent.TokensPerChar
			}

			liveChars := 0
			for _, line := range m.liveOutputLines {
				liveChars += len(line)
			}
			liveChars += len(m.liveOutputPending)

			maxTokens := 64000
			if m.cfg != nil && m.cfg.AIAgent.TotalContextTokens > 0 {
				maxTokens = m.cfg.AIAgent.TotalContextTokens
			}

			liveTokens := float64(liveChars) * tokensPerChar
			liveUsage := liveTokens / float64(maxTokens)

			return baseUsage + liveUsage
		}

		return baseUsage
	}
	return 0
}

// getCurrentSandbox returns the currently active sandbox ID and host
func (m *Model) getCurrentSandbox() (id string, host string) {
	if agent, ok := m.agentRunner.(*FluidAgent); ok {
		return agent.GetCurrentSandbox()
	}
	return "", ""
}

func (m *Model) getCurrentSourceVM() string {
	if agent, ok := m.agentRunner.(*FluidAgent); ok {
		return agent.GetCurrentSourceVM()
	}
	return ""
}

func (m *Model) getCurrentSandboxBaseImage() string {
	if agent, ok := m.agentRunner.(*FluidAgent); ok {
		return agent.GetCurrentSandboxBaseImage()
	}
	return ""
}

func (m *Model) updateSuggestions() {
	val := m.textarea.Value()
	if !strings.HasPrefix(val, "/") {
		m.suggestions = nil
		m.suggestionIndex = 0
		return
	}

	m.suggestions = nil
	for _, cmd := range allCommands {
		if strings.HasPrefix(cmd.name, val) {
			m.suggestions = append(m.suggestions, cmd)
		}
	}

	if m.suggestionIndex >= len(m.suggestions) {
		m.suggestionIndex = 0
	}
}

func (m *Model) addUserMessage(content string) {
	m.conversation = append(m.conversation, ConversationEntry{
		Role:    "user",
		Content: content,
	})
	// Hide banner after first user message
	m.showBanner = false
}

func (m *Model) addAssistantMessage(content string) {
	m.conversation = append(m.conversation, ConversationEntry{
		Role:    "assistant",
		Content: content,
	})
}

func (m *Model) addSystemMessage(content string) {
	m.conversation = append(m.conversation, ConversationEntry{
		Role:    "system",
		Content: content,
	})
}

func (m *Model) addToolResult(tr ToolResult) {
	m.conversation = append(m.conversation, ConversationEntry{
		Role: "tool",
		Tool: &tr,
	})
}

// formatLiveOutput formats the live output for display, truncating to last N lines
func (m *Model) formatLiveOutput() string {
	var lines []string
	if len(m.liveOutputLines) > 0 {
		lines = append(lines, m.liveOutputLines...)
	}
	// Add pending content if it's not empty, or if we have no lines yet (so we show something)
	if m.liveOutputPending != "" || len(lines) == 0 {
		lines = append(lines, m.liveOutputPending)
	}

	// Show last 20 lines to keep viewport manageable
	if len(lines) > 20 {
		lines = append([]string{"... (truncated)"}, lines[len(lines)-20:]...)
	}

	// Word wrap each line to fit the box width (box is m.width - 6 with padding)
	maxLineWidth := m.width - 10
	if maxLineWidth < 20 {
		maxLineWidth = 20
	}
	var wrappedLines []string
	for _, line := range lines {
		wrapped := wrapText(line, maxLineWidth)
		wrappedLines = append(wrappedLines, wrapped...)
	}

	return strings.Join(wrappedLines, "\n")
}

// wrapText wraps text to a maximum width, breaking on word boundaries where possible
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 || len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	remaining := text

	for len(remaining) > maxWidth {
		// Find last space within maxWidth
		breakPoint := maxWidth
		for i := maxWidth; i >= 0; i-- {
			if i < len(remaining) && remaining[i] == ' ' {
				breakPoint = i
				break
			}
		}
		// If no space found, break at maxWidth
		if breakPoint == 0 {
			breakPoint = maxWidth
		}

		lines = append(lines, remaining[:breakPoint])
		remaining = strings.TrimLeft(remaining[breakPoint:], " ")
	}

	if len(remaining) > 0 {
		lines = append(lines, remaining)
	}

	return lines
}

// isSSHWarningLine checks if a line is the common SSH known hosts warning
func isSSHWarningLine(line string) bool {
	// Common SSH warning patterns to filter
	patterns := []string{
		"Warning: Permanently added",
		"to the list of known hosts",
	}
	for _, p := range patterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	return false
}

// filterSSHWarning removes SSH known hosts warnings from a chunk of text
func filterSSHWarning(chunk string) string {
	// If the chunk contains a warning, filter it out
	if !strings.Contains(chunk, "Warning:") && !strings.Contains(chunk, "known hosts") {
		return chunk
	}

	// Split by lines and filter
	lines := strings.Split(chunk, "\n")
	var filtered []string
	for _, line := range lines {
		if !isSSHWarningLine(line) {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}

func (m *Model) updateViewportContent(forceScroll bool) {
	var b strings.Builder

	for _, entry := range m.conversation {
		switch entry.Role {
		case "user":
			// Render user message like the input box: blue $ and bordered box with white text
			userBox := m.styles.Border.Width(m.width - 6).Render(
				m.styles.InputPrompt.Render("$ ") + lipgloss.NewStyle().Foreground(textColor).Render(entry.Content),
			)
			b.WriteString(lipgloss.NewStyle().PaddingLeft(2).PaddingBottom(1).Render(userBox))
			b.WriteString("\n")
		case "assistant":
			// Render markdown
			rendered := entry.Content
			if m.mdRenderer != nil {
				if r, err := m.mdRenderer.Render(entry.Content); err == nil {
					rendered = r
				}
			}
			b.WriteString(m.styles.AssistantMessage.Render(rendered))
			b.WriteString("\n")
		case "system":
			b.WriteString(m.styles.Thinking.Render(entry.Content))
			b.WriteString("\n")
		case "warning":
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Width(m.width).Align(lipgloss.Right)
			b.WriteString(style.Render(entry.Content))
			b.WriteString("\n")
		case "tool":
			if entry.Tool != nil {
				b.WriteString(m.renderToolResult(*entry.Tool))
				b.WriteString("\n")
			}
		case "live_output":
			// Styled box for live command output
			boxWidth := m.width - 6
			if boxWidth < 30 {
				boxWidth = 30
			}

			style := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#3B82F6")).
				Padding(0, 1).
				Width(boxWidth)

			header := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3B82F6")).
				Bold(true).
				Render(fmt.Sprintf("$ Live output (%s):", m.liveOutputSandbox))

			// Content is already word-wrapped by formatLiveOutput
			content := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94A3B8")).
				Width(boxWidth - 4). // Account for padding and border
				Render(entry.Content)

			b.WriteString(style.Render(header + "\n" + content))
			b.WriteString("\n")

		case "live_prepare":
			// Styled box for source VM preparation progress
			boxWidth := m.width - 6
			if boxWidth < 30 {
				boxWidth = 30
			}

			style := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#EAB308")).
				Padding(0, 1).
				Width(boxWidth)

			header := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EAB308")).
				Bold(true).
				Render(fmt.Sprintf("Preparing %s for read-only access:", m.livePrepareSourceVM))

			content := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94A3B8")).
				Width(boxWidth - 4).
				Render(entry.Content)

			b.WriteString(style.Render(header + "\n" + content))
			b.WriteString("\n")
		}
	}

	// Add status indicator if active
	if m.thinking {
		dots := strings.Repeat(".", m.thinkingDots)
		var statusText string
		switch m.agentStatus {
		case StatusWorking:
			if m.currentToolName != "" {
				statusText = fmt.Sprintf(" Working: %s", m.currentToolName)
				// Show relevant context for specific tools
				if m.currentToolArgs != nil {
					switch m.currentToolName {
					case "run_command":
						if cmd, ok := m.currentToolArgs["command"].(string); ok {
							// Truncate long commands
							if len(cmd) > 60 {
								cmd = cmd[:57] + "..."
							}
							statusText = fmt.Sprintf(" Running: %s", cmd)
						}
					case "create_sandbox":
						if src, ok := m.currentToolArgs["source_vm_name"].(string); ok {
							statusText = fmt.Sprintf(" Creating sandbox from: %s", src)
						}
					case "destroy_sandbox":
						if id, ok := m.currentToolArgs["sandbox_id"].(string); ok {
							statusText = fmt.Sprintf(" Destroying: %s", id)
						}
					case "start_sandbox", "stop_sandbox":
						if id, ok := m.currentToolArgs["sandbox_id"].(string); ok {
							action := "Starting"
							if m.currentToolName == "stop_sandbox" {
								action = "Stopping"
							}
							statusText = fmt.Sprintf(" %s: %s", action, id)
						}
					}
				}
			} else {
				statusText = " Working"
			}
		default:
			statusText = " Thinking"
			if strings.HasPrefix(m.currentInput, "/") {
				cmd := strings.TrimPrefix(m.currentInput, "/")
				if cmd == "hosts" || cmd == "vms" || cmd == "playbooks" || cmd == "sandboxes" {
					statusText = " Pulling " + cmd
				}
			}
		}
		b.WriteString(m.styles.Thinking.Render(m.spinner.View() + statusText + dots))
		b.WriteString("\n")
	}

	wasAtBottom := m.viewport.AtBottom()
	m.viewport.SetContent(b.String())
	if forceScroll || wasAtBottom {
		m.viewport.GotoBottom()
	}
}

func (m *Model) renderToolResult(tr ToolResult) string {
	var b strings.Builder

	if tr.Error {
		icon := "x"
		b.WriteString(m.styles.ToolError.Render(fmt.Sprintf("  %s %s", icon, tr.Name)))
		b.WriteString("\n")
		if tr.ErrorMsg != "" {
			// Truncate long error messages
			errMsg := tr.ErrorMsg
			if len(errMsg) > 200 {
				errMsg = errMsg[:197] + "..."
			}
			b.WriteString(m.styles.ToolDetailsError.Render(fmt.Sprintf("      Error: %s", errMsg)))
		}
	} else {
		icon := "v"
		b.WriteString(m.styles.ToolSuccess.Render(fmt.Sprintf("  %s %s", icon, tr.Name)))
		b.WriteString("\n")

		// Format result based on tool type
		if tr.Result != nil {
			formatted := m.formatToolOutput(tr.Name, tr.Args, tr.Result)
			b.WriteString(formatted)
		}
	}

	return b.String()
}

// formatToolOutput formats tool results in a human-readable way
func (m *Model) formatToolOutput(toolName string, args, result map[string]any) string {
	var b strings.Builder

	switch toolName {
	case "run_command":
		// Show the command that was run
		if args != nil {
			if cmd, ok := args["command"].(string); ok {
				// Truncate long commands
				if len(cmd) > 80 {
					cmd = cmd[:77] + "..."
				}
				b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      $ %s", cmd)))
				b.WriteString("\n")
			}
		}
		// Show exit code
		if exitCode, ok := result["exit_code"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      exit: %v", exitCode)))
			b.WriteString("\n")
		}
		// Show stdout (truncated)
		if stdout, ok := result["stdout"].(string); ok && stdout != "" {
			stdout = strings.TrimSpace(stdout)
			lines := strings.Split(stdout, "\n")
			if len(lines) > 5 {
				lines = append(lines[:5], fmt.Sprintf("... (%d more lines)", len(lines)-5))
			}
			for _, line := range lines {
				if len(line) > 100 {
					line = line[:97] + "..."
				}
				b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      %s", line)))
				b.WriteString("\n")
			}
		}
		// Show stderr if present
		if stderr, ok := result["stderr"].(string); ok && stderr != "" {
			stderr = strings.TrimSpace(stderr)
			// Filter out SSH warnings
			lines := strings.Split(stderr, "\n")
			var filteredLines []string
			for _, line := range lines {
				if !isSSHWarningLine(line) && strings.TrimSpace(line) != "" {
					filteredLines = append(filteredLines, line)
				}
			}
			if len(filteredLines) > 3 {
				filteredLines = append(filteredLines[:3], "...")
			}
			for _, line := range filteredLines {
				if len(line) > 100 {
					line = line[:97] + "..."
				}
				b.WriteString(m.styles.ToolDetailsError.Render(fmt.Sprintf("      stderr: %s", line)))
				b.WriteString("\n")
			}
		}

	case "list_sandboxes":
		if sandboxes, ok := result["sandboxes"].([]any); ok {
			if len(sandboxes) == 0 {
				b.WriteString(m.styles.ToolDetails.Render("      No sandboxes found"))
				b.WriteString("\n")
			} else {
				b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Found %d sandbox(es)", len(sandboxes))))
				b.WriteString("\n")
				for i, sb := range sandboxes {
					if i >= 5 {
						b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ... and %d more", len(sandboxes)-5)))
						b.WriteString("\n")
						break
					}
					if sbMap, ok := sb.(map[string]any); ok {
						id := sbMap["id"]
						name := sbMap["name"]
						state := sbMap["state"]
						ip := sbMap["ip_address"]
						line := fmt.Sprintf("      - %v (%v) %v", name, id, state)
						if ip != nil && ip != "" {
							line += fmt.Sprintf(" @ %v", ip)
						}
						b.WriteString(m.styles.ToolDetails.Render(line))
						b.WriteString("\n")
					}
				}
			}
		}

	case "create_sandbox":
		if id, ok := result["sandbox_id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ID: %v", id)))
			b.WriteString("\n")
		}
		if name, ok := result["name"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Name: %v", name)))
			b.WriteString("\n")
		}
		if ip, ok := result["ip_address"]; ok && ip != nil && ip != "" {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      IP: %v", ip)))
			b.WriteString("\n")
		}
		if state, ok := result["state"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      State: %v", state)))
			b.WriteString("\n")
		}

	case "get_sandbox":
		if id, ok := result["id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ID: %v", id)))
			b.WriteString("\n")
		}
		if name, ok := result["name"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Name: %v", name)))
			b.WriteString("\n")
		}
		if state, ok := result["state"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      State: %v", state)))
			b.WriteString("\n")
		}
		if ip, ok := result["ip_address"]; ok && ip != nil && ip != "" {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      IP: %v", ip)))
			b.WriteString("\n")
		}
		if host, ok := result["host"]; ok && host != nil && host != "" {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Host: %v", host)))
			b.WriteString("\n")
		}

	case "destroy_sandbox":
		if id, ok := result["sandbox_id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Destroyed: %v", id)))
			b.WriteString("\n")
		}

	case "start_sandbox", "stop_sandbox":
		if id, ok := result["sandbox_id"]; ok {
			action := "Started"
			if toolName == "stop_sandbox" {
				action = "Stopped"
			}
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      %s: %v", action, id)))
			b.WriteString("\n")
		}
		if ip, ok := result["ip_address"]; ok && ip != nil && ip != "" {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      IP: %v", ip)))
			b.WriteString("\n")
		}

	case "list_vms":
		if vms, ok := result["vms"].([]any); ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Found %d VM(s)", len(vms))))
			b.WriteString("\n")
			for i, vm := range vms {
				if i >= 10 {
					b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ... and %d more", len(vms)-10)))
					b.WriteString("\n")
					break
				}
				if vmMap, ok := vm.(map[string]any); ok {
					name := vmMap["name"]
					state := vmMap["state"]
					host := vmMap["host"]
					line := fmt.Sprintf("      - %v (%v)", name, state)
					if host != nil && host != "" {
						line += fmt.Sprintf(" on %v", host)
					}
					b.WriteString(m.styles.ToolDetails.Render(line))
					b.WriteString("\n")
				}
			}
		}

	case "create_snapshot":
		if id, ok := result["snapshot_id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Snapshot: %v", id)))
			b.WriteString("\n")
		}
		if name, ok := result["name"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Name: %v", name)))
			b.WriteString("\n")
		}

	case "add_playbook_task", "create_playbook", "get_playbook":
		if name, ok := result["name"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Playbook: %v", name)))
			b.WriteString("\n")
		}
		if id, ok := result["id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      ID: %v", id)))
			b.WriteString("\n")
		}
		if taskID, ok := result["task_id"]; ok {
			b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      Task ID: %v", taskID)))
			b.WriteString("\n")
		}

	default:
		// Generic formatting for unknown tools
		content := fmt.Sprintf("%v", result)
		if len(content) > 150 {
			content = content[:147] + "..."
		}
		b.WriteString(m.styles.ToolDetails.Render(fmt.Sprintf("      -> %s", content)))
		b.WriteString("\n")
	}

	return b.String()
}

// startCleanup begins the cleanup process for all created sandboxes
func (m Model) startCleanup() tea.Cmd {
	agent, ok := m.agentRunner.(*FluidAgent)
	if !ok {
		return func() tea.Msg {
			return CleanupCompleteMsg{Total: 0}
		}
	}

	// Start cleanup in background, progress will come through status channel
	go agent.CleanupWithProgress(m.cleanupOrder)

	// Return commands to listen for status updates and keep spinner going
	return tea.Batch(m.listenForStatus(), m.spinner.Tick)
}

// renderCleanupView renders the cleanup page showing sandbox destruction progress
func (m Model) renderCleanupView() string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F59E0B")).
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(2).
		MarginBottom(2)

	b.WriteString(headerStyle.Render("Cleaning Up Sandboxes"))
	b.WriteString("\n\n")

	// Sandbox list
	listStyle := lipgloss.NewStyle().
		PaddingLeft(4)

	for _, id := range m.cleanupOrder {
		status := m.cleanupStatuses[id]
		var statusIcon, statusColor string

		switch status {
		case CleanupStatusPending:
			statusIcon = "o"
			statusColor = "#6B7280" // gray
		case CleanupStatusDestroying:
			statusIcon = m.spinner.View()
			statusColor = "#3B82F6" // blue
		case CleanupStatusDestroyed:
			statusIcon = "v"
			statusColor = "#10B981" // green
		case CleanupStatusFailed:
			statusIcon = "x"
			statusColor = "#EF4444" // red
		case CleanupStatusSkipped:
			statusIcon = "-"
			statusColor = "#6B7280" // gray
		}

		idStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(statusColor))

		line := fmt.Sprintf("%s %s", statusIcon, id)
		if errMsg, hasErr := m.cleanupErrors[id]; hasErr {
			line += fmt.Sprintf(" - %s", errMsg)
		}
		if status == CleanupStatusSkipped {
			line += " (already destroyed)"
		}

		b.WriteString(listStyle.Render(idStyle.Render(line)))
		b.WriteString("\n")
	}

	// Summary line if complete
	if m.cleanupDone && m.cleanupResult != nil {
		b.WriteString("\n")
		summaryStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981")).
			PaddingLeft(4)

		summary := fmt.Sprintf("Cleanup complete: %d destroyed", m.cleanupResult.Destroyed)
		if m.cleanupResult.Failed > 0 {
			summary += fmt.Sprintf(", %d failed", m.cleanupResult.Failed)
		}
		if m.cleanupResult.Skipped > 0 {
			summary += fmt.Sprintf(", %d skipped", m.cleanupResult.Skipped)
		}
		b.WriteString(summaryStyle.Render(summary))
		b.WriteString("\n")
	}

	// Footer hint
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(2)

	if m.cleanupDone {
		b.WriteString(footerStyle.Render("Exiting..."))
	} else {
		b.WriteString(footerStyle.Render("Press Ctrl+C to force quit"))
	}

	return b.String()
}

// isLeakedMouseSequence checks whether runes are a leaked SGR mouse escape
// sequence. When rapid mouse events fragment across read boundaries, bubbletea
// consumes the leading ESC as KeyEscape and the remaining bytes arrive as
// KeyRunes. Fragments can be full ("[<65;80;25M"), concatenated
// ("[<65;80;25M[<65;80;25M"), or partial (";25M", "<65;80").
// All fragments consist solely of SGR mouse characters: [ < 0-9 ; M m.
// Normal keystrokes produce single-rune events; paste uses bracketed paste.
// A multi-rune event containing only these characters is always a leak.
func isLeakedMouseSequence(runes []rune) bool {
	if len(runes) < 2 {
		return false
	}
	hasSGRChar := false
	for _, r := range runes {
		switch {
		case r >= '0' && r <= '9':
		case r == ';', r == '[', r == '<', r == 'M', r == 'm':
			hasSGRChar = true
		default:
			return false
		}
	}
	return hasSGRChar
}

// mouseEventFilter drops leaked SGR mouse escape sequences before they reach
// Update(). This is the bubbletea-idiomatic way to handle input fragmentation.
func mouseEventFilter(_ tea.Model, msg tea.Msg) tea.Msg {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyRunes {
			if isLeakedMouseSequence(keyMsg.Runes) {
				return nil
			}
			// \x1b[ (CSI introducer) parsed as alt+[ when the sequence fragments
			// at the 2-byte boundary. This is never legitimate user input - terminals
			// don't emit bare \x1b[ for Alt+[.
			if keyMsg.Alt && len(keyMsg.Runes) == 1 && keyMsg.Runes[0] == '[' {
				return nil
			}
		}
	}
	return msg
}

// Run starts the TUI application
func Run(m Model) error {
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithFilter(mouseEventFilter),
	)
	_, err := p.Run()
	return err
}
