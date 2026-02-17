package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Message types for the TUI

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Name      string
	Args      map[string]any
	Result    map[string]any
	Error     bool
	ErrorMsg  string
	StartTime time.Time
	EndTime   time.Time
}

// AgentResponse represents a response from the agent
type AgentResponse struct {
	Content       string
	ToolResults   []ToolResult
	Done          bool
	AwaitingInput bool
}

// UserInputMsg is sent when the user submits input
type UserInputMsg struct {
	Input string
}

// AgentResponseMsg is sent when the agent responds
type AgentResponseMsg struct {
	Response AgentResponse
}

// AgentErrorMsg is sent when the agent encounters an error
type AgentErrorMsg struct {
	Err error
}

// ThinkingTickMsg is sent for the thinking animation
type ThinkingTickMsg struct{}

// ThinkingCmd returns a command for the thinking animation
func ThinkingCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return ThinkingTickMsg{}
	})
}

// AgentStatus represents what the agent is currently doing
type AgentStatus int

const (
	StatusThinking AgentStatus = iota // Waiting for LLM response
	StatusWorking                     // Executing a tool
)

// ToolStartMsg is sent when a tool starts executing
type ToolStartMsg struct {
	ToolName string
	Args     map[string]any
}

// ToolCompleteMsg is sent when a tool finishes executing
type ToolCompleteMsg struct {
	ToolName string
	Success  bool
	Result   map[string]any
	Error    string
}

// AgentDoneMsg is sent through the status channel when the agent finishes
// This unblocks the status listener
type AgentDoneMsg struct{}

// ClearThinkingMsg is sent to clear the thinking indicator
type ClearThinkingMsg struct{}

// StartAgentMsg is sent to start the agent processing
type StartAgentMsg struct {
	Input string
}

// WindowSizeMsg wraps the window size message
type WindowSizeMsg struct {
	Width  int
	Height int
}

// QuitMsg is sent when the user wants to quit
type QuitMsg struct{}

// ResetMsg is sent when the user wants to reset the conversation
type ResetMsg struct{}

// FocusInputMsg is sent to focus the input field
type FocusInputMsg struct{}

// ScrollMsg is sent to scroll the conversation view
type ScrollMsg struct {
	Direction int // positive = down, negative = up
}

// SettingsOpenMsg is sent to open the settings modal
type SettingsOpenMsg struct{}

// SettingsCloseMsg is sent when settings are closed
type SettingsCloseMsg struct {
	Saved bool
}

// ReviewRequestMsg is sent when the agent requests human review
type ReviewRequestMsg struct {
	Reason  string
	Summary map[string]any
}

// ReviewResponseMsg is sent when the user responds to a review request
type ReviewResponseMsg struct {
	Approved bool
	Feedback string
}

// TaskCompleteMsg is sent when a task is completed
type TaskCompleteMsg struct {
	Summary string
	Stats   map[string]any
}

// CompactStartMsg is sent when compaction starts
type CompactStartMsg struct{}

// CompactCompleteMsg is sent when compaction completes
type CompactCompleteMsg struct {
	PreviousTokens int
	NewTokens      int
	Summary        string
}

// CompactErrorMsg is sent when compaction fails
type CompactErrorMsg struct {
	Err error
}

// CommandOutputChunkMsg is sent when streaming output arrives from a command
type CommandOutputChunkMsg struct {
	SandboxID string
	IsStderr  bool // true for stderr, false for stdout
	Chunk     string
}

// CommandOutputResetMsg signals that live output should be reset (e.g., on retry)
type CommandOutputResetMsg struct {
	SandboxID string
}

// CommandOutputDoneMsg signals streaming is complete
type CommandOutputDoneMsg struct {
	SandboxID string
}

// RetryAttemptMsg is sent when a command is being retried
type RetryAttemptMsg struct {
	SandboxID string
	Attempt   int
	Max       int
	Delay     time.Duration
	Error     string
}

// CleanupStartMsg signals that cleanup has started
type CleanupStartMsg struct {
	SandboxIDs []string
}

// CleanupStatus represents the status of a sandbox during cleanup
type CleanupStatus string

const (
	CleanupStatusPending    CleanupStatus = "pending"
	CleanupStatusDestroying CleanupStatus = "destroying"
	CleanupStatusDestroyed  CleanupStatus = "destroyed"
	CleanupStatusFailed     CleanupStatus = "failed"
	CleanupStatusSkipped    CleanupStatus = "skipped" // Already destroyed
)

// CleanupProgressMsg is sent when a sandbox cleanup status changes
type CleanupProgressMsg struct {
	SandboxID string
	Status    CleanupStatus
	Error     string // Only set if Status is CleanupStatusFailed
}

// CleanupCompleteMsg is sent when all sandboxes have been cleaned up
type CleanupCompleteMsg struct {
	Total     int
	Destroyed int
	Failed    int
	Skipped   int
}

// SourcePrepareProgressMsg is sent during source VM preparation to show step-by-step progress
type SourcePrepareProgressMsg struct {
	SourceVM string
	StepName string
	StepNum  int
	Total    int
	Done     bool
}

// AutoReadOnlyMsg is sent when read-only mode is auto-toggled for source VM operations
type AutoReadOnlyMsg struct {
	SourceVM string
	Enabled  bool
}

// UpdateAvailableMsg is sent when a newer version is available
type UpdateAvailableMsg struct {
	Version string
}
