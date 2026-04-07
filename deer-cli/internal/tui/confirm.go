package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MemoryApprovalRequest contains details about the memory shortage
// Deprecated: Use ResourceApprovalRequest instead for unified resource approval
type MemoryApprovalRequest struct {
	SourceVM          string
	HostName          string
	RequiredMemoryMB  int
	AvailableMemoryMB int64
	TotalMemoryMB     int64
	Warnings          []string
	Errors            []string
}

// ResourceApprovalRequest contains details about resource requirements that need approval
type ResourceApprovalRequest struct {
	SourceVM string
	HostName string

	// CPU resources
	RequestedCPUs    int
	AvailableCPUs    int
	TotalCPUs        int
	NeedsCPUApproval bool

	// Memory resources
	RequestedMemoryMB   int
	AvailableMemoryMB   int64
	TotalMemoryMB       int64
	NeedsMemoryApproval bool

	// Storage resources
	RequestedStorageMB   int64
	AvailableStorageMB   int64
	TotalStorageMB       int64
	NeedsStorageApproval bool

	Warnings []string
	Errors   []string
}

// ResourceApprovalResult is the response from the user
type ResourceApprovalResult struct {
	Approved bool
	Request  ResourceApprovalRequest
}

// ResourceApprovalRequestMsg is sent when the agent needs resource approval
type ResourceApprovalRequestMsg struct {
	Request ResourceApprovalRequest
}

// ResourceApprovalResponseMsg is sent when the user responds to the resource approval dialog
type ResourceApprovalResponseMsg struct {
	Result ResourceApprovalResult
}

// MemoryApprovalResult is the response from the user
type MemoryApprovalResult struct {
	Approved bool
	Request  MemoryApprovalRequest
}

// ConfirmModel is a Bubble Tea model for confirming memory-constrained sandbox creation
type ConfirmModel struct {
	request  MemoryApprovalRequest
	selected int // 0 = No (default safe option), 1 = Yes
	width    int
	height   int
	styles   confirmStyles

	// Result channel to communicate back to caller
	resultChan chan<- MemoryApprovalResult
}

type confirmStyles struct {
	dialog      lipgloss.Style
	title       lipgloss.Style
	warning     lipgloss.Style
	error       lipgloss.Style
	info        lipgloss.Style
	highlight   lipgloss.Style
	button      lipgloss.Style
	buttonFocus lipgloss.Style
	help        lipgloss.Style
}

func newConfirmStyles() confirmStyles {
	return confirmStyles{
		dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FACC15")).
			Padding(1, 2),
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FACC15")).
			MarginBottom(1),
		warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FACC15")),
		error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")),
		info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")),
		highlight: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8FAFC")),
		button: lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#475569")),
		buttonFocus: lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3B82F6")).
			Foreground(lipgloss.Color("#3B82F6")).
			Bold(true),
		help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B")).
			MarginTop(1),
	}
}

// NewConfirmModel creates a new confirmation dialog for memory approval
func NewConfirmModel(request MemoryApprovalRequest, resultChan chan<- MemoryApprovalResult) ConfirmModel {
	return ConfirmModel{
		request:    request,
		selected:   0, // Default to "No" for safety
		styles:     newConfirmStyles(),
		resultChan: resultChan,
	}
}

// confirmKeyMap defines keys for the confirmation dialog
type confirmKeyMap struct {
	Left   key.Binding
	Right  key.Binding
	Enter  key.Binding
	Escape key.Binding
	Tab    key.Binding
	Yes    key.Binding
	No     key.Binding
}

var confirmKeys = confirmKeyMap{
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("<-/h", "select"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("->/l", "select"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle"),
	),
	Yes: key.NewBinding(
		key.WithKeys("y", "Y"),
		key.WithHelp("y", "approve"),
	),
	No: key.NewBinding(
		key.WithKeys("n", "N"),
		key.WithHelp("n", "deny"),
	),
}

// Init implements tea.Model
func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, confirmKeys.Left):
			m.selected = 0
		case key.Matches(msg, confirmKeys.Right):
			m.selected = 1
		case key.Matches(msg, confirmKeys.Tab):
			m.selected = (m.selected + 1) % 2
		case key.Matches(msg, confirmKeys.Yes):
			m.selected = 1
			return m.confirm()
		case key.Matches(msg, confirmKeys.No), key.Matches(msg, confirmKeys.Escape):
			m.selected = 0
			return m.confirm()
		case key.Matches(msg, confirmKeys.Enter):
			return m.confirm()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m ConfirmModel) confirm() (tea.Model, tea.Cmd) {
	result := MemoryApprovalResult{
		Approved: m.selected == 1,
		Request:  m.request,
	}
	if m.resultChan != nil {
		m.resultChan <- result
	}
	return m, func() tea.Msg {
		return MemoryApprovalResponseMsg{Result: result}
	}
}

// View implements tea.Model
func (m ConfirmModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.title.Render("! Memory Warning"))
	b.WriteString("\n\n")

	// Context info
	hostInfo := "local"
	if m.request.HostName != "" {
		hostInfo = m.request.HostName
	}
	b.WriteString(m.styles.info.Render(fmt.Sprintf("Creating sandbox from: %s", m.styles.highlight.Render(m.request.SourceVM))))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render(fmt.Sprintf("Target host: %s", hostInfo)))
	b.WriteString("\n\n")

	// Memory comparison
	deficit := int64(m.request.RequiredMemoryMB) - m.request.AvailableMemoryMB
	percentAvailable := float64(m.request.AvailableMemoryMB) / float64(m.request.TotalMemoryMB) * 100

	b.WriteString(m.styles.warning.Render("Memory Status:"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Required:  %s MB\n", m.styles.highlight.Render(fmt.Sprintf("%d", m.request.RequiredMemoryMB))))
	b.WriteString(fmt.Sprintf("  Available: %s MB (%.1f%% of total)\n", m.styles.error.Render(fmt.Sprintf("%d", m.request.AvailableMemoryMB)), percentAvailable))
	b.WriteString(fmt.Sprintf("  Total:     %d MB\n", m.request.TotalMemoryMB))
	b.WriteString(fmt.Sprintf("  Deficit:   %s MB\n", m.styles.error.Render(fmt.Sprintf("%d", deficit))))
	b.WriteString("\n")

	// Warnings
	if len(m.request.Warnings) > 0 {
		b.WriteString(m.styles.warning.Render("Warnings:"))
		b.WriteString("\n")
		for _, w := range m.request.Warnings {
			b.WriteString(m.styles.warning.Render(fmt.Sprintf("  - %s", w)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Errors
	if len(m.request.Errors) > 0 {
		b.WriteString(m.styles.error.Render("Errors:"))
		b.WriteString("\n")
		for _, e := range m.request.Errors {
			b.WriteString(m.styles.error.Render(fmt.Sprintf("  - %s", e)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Warning message
	b.WriteString(m.styles.warning.Render("Proceeding may cause:"))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("  - VM performance degradation"))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("  - System instability if memory is exhausted"))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("  - Potential VM or host crashes under load"))
	b.WriteString("\n\n")

	// Question
	b.WriteString(m.styles.highlight.Render("Proceed with sandbox creation anyway?"))
	b.WriteString("\n\n")

	// Buttons - use JoinHorizontal for proper alignment
	var noBtn, yesBtn string
	if m.selected == 0 {
		noBtn = m.styles.buttonFocus.Render(" [ No ] ")
		yesBtn = m.styles.button.Render("   Yes   ")
	} else {
		noBtn = m.styles.button.Render("   No   ")
		yesBtn = m.styles.buttonFocus.Render(" [ Yes ] ")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, noBtn, "    ", yesBtn)
	b.WriteString(buttons)
	b.WriteString("\n\n")

	// Help
	b.WriteString(m.styles.help.Render("  <-/-> or Tab: select | Enter: confirm | y/n: quick select | Esc: cancel"))

	// Wrap in dialog box
	content := m.styles.dialog.Render(b.String())

	// Center the dialog
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return content
}

// MemoryApprovalResponseMsg is sent when the user responds to the memory approval dialog
type MemoryApprovalResponseMsg struct {
	Result MemoryApprovalResult
}

// MemoryApprovalRequestMsg is sent when the agent needs memory approval
type MemoryApprovalRequestMsg struct {
	Request MemoryApprovalRequest
}

// RunConfirmDialog runs a standalone confirmation dialog
func RunConfirmDialog(request MemoryApprovalRequest) (bool, error) {
	resultChan := make(chan MemoryApprovalResult, 1)
	model := NewConfirmModel(request, resultChan)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return false, err
	}

	select {
	case result := <-resultChan:
		return result.Approved, nil
	default:
		return false, nil
	}
}

// NetworkApprovalRequest contains details about the network access request
type NetworkApprovalRequest struct {
	Command     string   // The full command being executed
	SandboxID   string   // The sandbox where the command will run
	NetworkTool string   // The detected network tool (curl, wget, etc.)
	URLs        []string // Detected URLs in the command
}

// NetworkApprovalResult is the response from the user
type NetworkApprovalResult struct {
	Approved bool
	Request  NetworkApprovalRequest
}

// NetworkApprovalRequestMsg is sent when the agent needs network approval
type NetworkApprovalRequestMsg struct {
	Request NetworkApprovalRequest
}

// NetworkApprovalResponseMsg is sent when the user responds to the network approval dialog
type NetworkApprovalResponseMsg struct {
	Result NetworkApprovalResult
}

// NetworkConfirmModel is a Bubble Tea model for confirming network access
type NetworkConfirmModel struct {
	request  NetworkApprovalRequest
	selected int // 0 = No (default safe option), 1 = Yes
	width    int
	height   int
	styles   confirmStyles

	// Result channel to communicate back to caller
	resultChan chan<- NetworkApprovalResult
}

// NewNetworkConfirmModel creates a new confirmation dialog for network approval
func NewNetworkConfirmModel(request NetworkApprovalRequest, resultChan chan<- NetworkApprovalResult) NetworkConfirmModel {
	return NetworkConfirmModel{
		request:    request,
		selected:   0, // Default to "No" for safety
		styles:     newConfirmStyles(),
		resultChan: resultChan,
	}
}

// Init implements tea.Model
func (m NetworkConfirmModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m NetworkConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, confirmKeys.Left):
			m.selected = 0
		case key.Matches(msg, confirmKeys.Right):
			m.selected = 1
		case key.Matches(msg, confirmKeys.Tab):
			m.selected = (m.selected + 1) % 2
		case key.Matches(msg, confirmKeys.Yes):
			m.selected = 1
			return m.confirm()
		case key.Matches(msg, confirmKeys.No), key.Matches(msg, confirmKeys.Escape):
			m.selected = 0
			return m.confirm()
		case key.Matches(msg, confirmKeys.Enter):
			return m.confirm()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m NetworkConfirmModel) confirm() (tea.Model, tea.Cmd) {
	result := NetworkApprovalResult{
		Approved: m.selected == 1,
		Request:  m.request,
	}
	if m.resultChan != nil {
		m.resultChan <- result
	}
	return m, func() tea.Msg {
		return NetworkApprovalResponseMsg{Result: result}
	}
}

// View implements tea.Model
func (m NetworkConfirmModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.title.Render("! Network Access Request"))
	b.WriteString("\n\n")

	// Context info
	b.WriteString(m.styles.info.Render(fmt.Sprintf("Sandbox: %s", m.styles.highlight.Render(m.request.SandboxID))))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render(fmt.Sprintf("Tool: %s", m.styles.highlight.Render(m.request.NetworkTool))))
	b.WriteString("\n\n")

	// Command
	b.WriteString(m.styles.warning.Render("Command:"))
	b.WriteString("\n")
	cmd := m.request.Command
	if len(cmd) > 80 {
		cmd = cmd[:77] + "..."
	}
	b.WriteString(m.styles.info.Render(fmt.Sprintf("  %s", cmd)))
	b.WriteString("\n\n")

	// URLs detected
	if len(m.request.URLs) > 0 {
		b.WriteString(m.styles.warning.Render("URLs detected:"))
		b.WriteString("\n")
		for _, url := range m.request.URLs {
			if len(url) > 70 {
				url = url[:67] + "..."
			}
			b.WriteString(m.styles.info.Render(fmt.Sprintf("  - %s", url)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Warning message
	b.WriteString(m.styles.warning.Render("This command will access the internet."))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("  - Data may be downloaded or uploaded"))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("  - External servers will be contacted"))
	b.WriteString("\n\n")

	// Question
	b.WriteString(m.styles.highlight.Render("Allow this network access?"))
	b.WriteString("\n\n")

	// Buttons
	var noBtn, yesBtn string
	if m.selected == 0 {
		noBtn = m.styles.buttonFocus.Render(" [ No ] ")
		yesBtn = m.styles.button.Render("   Yes   ")
	} else {
		noBtn = m.styles.button.Render("   No   ")
		yesBtn = m.styles.buttonFocus.Render(" [ Yes ] ")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, noBtn, "    ", yesBtn)
	b.WriteString(buttons)
	b.WriteString("\n\n")

	// Help
	b.WriteString(m.styles.help.Render("  <-/-> or Tab: select | Enter: confirm | y/n: quick select | Esc: cancel"))

	// Wrap in dialog box
	content := m.styles.dialog.Render(b.String())

	// Center the dialog
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return content
}

// SourcePrepareApprovalRequest contains details about the source VM that needs preparation
type SourcePrepareApprovalRequest struct {
	SourceVM string
	Error    string // The connection error that triggered this
}

// SourcePrepareApprovalResult is the response from the user
type SourcePrepareApprovalResult struct {
	Approved bool
	Request  SourcePrepareApprovalRequest
}

// SourcePrepareApprovalRequestMsg is sent when the agent detects a source VM needs preparation
type SourcePrepareApprovalRequestMsg struct {
	Request SourcePrepareApprovalRequest
}

// SourcePrepareApprovalResponseMsg is sent when the user responds
type SourcePrepareApprovalResponseMsg struct {
	Result SourcePrepareApprovalResult
}

// SourcePrepareConfirmModel is a Bubble Tea model for confirming source VM preparation
type SourcePrepareConfirmModel struct {
	request  SourcePrepareApprovalRequest
	selected int // 0 = No (default safe option), 1 = Yes
	width    int
	height   int
	styles   confirmStyles

	resultChan chan<- SourcePrepareApprovalResult
}

// NewSourcePrepareConfirmModel creates a new confirmation dialog for source prepare approval
func NewSourcePrepareConfirmModel(request SourcePrepareApprovalRequest, resultChan chan<- SourcePrepareApprovalResult) SourcePrepareConfirmModel {
	return SourcePrepareConfirmModel{
		request:    request,
		selected:   0,
		styles:     newConfirmStyles(),
		resultChan: resultChan,
	}
}

// Init implements tea.Model
func (m SourcePrepareConfirmModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m SourcePrepareConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, confirmKeys.Left):
			m.selected = 0
		case key.Matches(msg, confirmKeys.Right):
			m.selected = 1
		case key.Matches(msg, confirmKeys.Tab):
			m.selected = (m.selected + 1) % 2
		case key.Matches(msg, confirmKeys.Yes):
			m.selected = 1
			return m.confirm()
		case key.Matches(msg, confirmKeys.No), key.Matches(msg, confirmKeys.Escape):
			m.selected = 0
			return m.confirm()
		case key.Matches(msg, confirmKeys.Enter):
			return m.confirm()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m SourcePrepareConfirmModel) confirm() (tea.Model, tea.Cmd) {
	result := SourcePrepareApprovalResult{
		Approved: m.selected == 1,
		Request:  m.request,
	}
	if m.resultChan != nil {
		m.resultChan <- result
	}
	return m, func() tea.Msg {
		return SourcePrepareApprovalResponseMsg{Result: result}
	}
}

// View implements tea.Model
func (m SourcePrepareConfirmModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.title.Render("! Source VM Not Prepared"))
	b.WriteString("\n\n")

	b.WriteString(m.styles.info.Render(fmt.Sprintf("Source VM: %s", m.styles.highlight.Render(m.request.SourceVM))))
	b.WriteString("\n\n")

	b.WriteString(m.styles.warning.Render("Connection Error:"))
	b.WriteString("\n")
	errMsg := m.request.Error
	if len(errMsg) > 120 {
		errMsg = errMsg[:117] + "..."
	}
	b.WriteString(m.styles.error.Render(fmt.Sprintf("  %s", errMsg)))
	b.WriteString("\n\n")

	b.WriteString(m.styles.info.Render("The source VM may not be prepared for read-only access."))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("Running 'source prepare' will:"))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("  - Start the VM if it's not running"))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("  - Create a 'fluid-readonly' user"))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("  - Install a restricted shell"))
	b.WriteString("\n")
	b.WriteString(m.styles.info.Render("  - Configure SSH certificate authentication"))
	b.WriteString("\n\n")

	b.WriteString(m.styles.highlight.Render("Prepare this source VM for read-only access?"))
	b.WriteString("\n\n")

	var noBtn, yesBtn string
	if m.selected == 0 {
		noBtn = m.styles.buttonFocus.Render(" [ No ] ")
		yesBtn = m.styles.button.Render("   Yes   ")
	} else {
		noBtn = m.styles.button.Render("   No   ")
		yesBtn = m.styles.buttonFocus.Render(" [ Yes ] ")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, noBtn, "    ", yesBtn)
	b.WriteString(buttons)
	b.WriteString("\n\n")

	b.WriteString(m.styles.help.Render("  <-/-> or Tab: select | Enter: confirm | y/n: quick select | Esc: cancel"))

	content := m.styles.dialog.Render(b.String())

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	return content
}
