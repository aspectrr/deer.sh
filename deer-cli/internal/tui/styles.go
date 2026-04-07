package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	primaryColor   = lipgloss.Color("#3B82F6") // Blue
	secondaryColor = lipgloss.Color("#06B6D4") // Cyan
	successColor   = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	textColor      = lipgloss.Color("#F9FAFB") // Light gray
	borderColor    = lipgloss.Color("#374151") // Medium gray
)

// Styles contains all the styles used in the TUI
type Styles struct {
	// App-level styles
	App lipgloss.Style

	// Status bar
	StatusBar     lipgloss.Style
	StatusBarText lipgloss.Style

	// Messages
	UserMessage      lipgloss.Style
	AssistantMessage lipgloss.Style

	// Tool calls
	ToolSuccess      lipgloss.Style
	ToolError        lipgloss.Style
	ToolDetails      lipgloss.Style
	ToolDetailsError lipgloss.Style
	ToolName         lipgloss.Style

	// Input
	InputPrompt lipgloss.Style
	Input       lipgloss.Style

	// Thinking indicator
	Thinking lipgloss.Style

	// Help
	Help     lipgloss.Style
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style

	// Borders
	Border       lipgloss.Style
	FocusedStyle lipgloss.Style

	// Conversation area
	Conversation lipgloss.Style

	// Modals
	ModalTitle   lipgloss.Style
	ModalContent lipgloss.Style
	ModalBorder  lipgloss.Style

	// Buttons
	ButtonPrimary lipgloss.Style
	ButtonSuccess lipgloss.Style
	ButtonError   lipgloss.Style
}

// DefaultStyles returns the default styles for the TUI
func DefaultStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle().
			Padding(0),

		StatusBar: lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(textColor).
			Padding(0, 1).
			Bold(true),

		StatusBarText: lipgloss.NewStyle().
			Foreground(textColor),

		UserMessage: lipgloss.NewStyle().
			Foreground(successColor).
			PaddingLeft(2).
			PaddingBottom(1),

		AssistantMessage: lipgloss.NewStyle().
			Foreground(textColor).
			PaddingLeft(2).
			PaddingBottom(1),

		ToolSuccess: lipgloss.NewStyle().
			Foreground(secondaryColor).
			PaddingLeft(4),

		ToolError: lipgloss.NewStyle().
			Foreground(errorColor).
			PaddingLeft(4),

		ToolDetails: lipgloss.NewStyle().
			Foreground(mutedColor).
			PaddingLeft(6),

		ToolDetailsError: lipgloss.NewStyle().
			Foreground(errorColor).
			PaddingLeft(6),

		ToolName: lipgloss.NewStyle().
			Bold(true),

		InputPrompt: lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true),

		Input: lipgloss.NewStyle().
			Foreground(textColor),

		Thinking: lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			PaddingLeft(2),

		Help: lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1),

		HelpKey: lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(mutedColor),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor),

		FocusedStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor),

		Conversation: lipgloss.NewStyle().
			Padding(1, 0),

		ModalTitle: lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(textColor).
			Padding(0, 2).
			Bold(true).
			Align(lipgloss.Center),

		ModalContent: lipgloss.NewStyle().
			Padding(1, 2),

		ModalBorder: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2),

		ButtonPrimary: lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(textColor).
			Padding(0, 2).
			Bold(true),

		ButtonSuccess: lipgloss.NewStyle().
			Background(successColor).
			Foreground(textColor).
			Padding(0, 2).
			Bold(true),

		ButtonError: lipgloss.NewStyle().
			Background(errorColor).
			Foreground(textColor).
			Padding(0, 2).
			Bold(true),
	}
}
