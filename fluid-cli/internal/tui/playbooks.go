package tui

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/fluid.sh/fluid/internal/ansible"
)

// PlaybooksCloseMsg is sent when the playbooks view is closed
type PlaybooksCloseMsg struct{}

// PlaybookDeletedMsg is sent when a playbook is deleted
type PlaybookDeletedMsg struct {
	ID   string
	Name string
}

// PlaybookDeleteErrorMsg is sent when a playbook deletion fails
type PlaybookDeleteErrorMsg struct {
	Err error
}

// PlaybooksModel is a Bubble Tea model for browsing and managing playbooks
type PlaybooksModel struct {
	playbooks       []*ansible.PlaybookWithTasks
	selected        int
	width           int
	height          int
	styles          playbooksStyles
	previewViewport viewport.Model

	// Delete confirmation
	confirmingDelete bool
	deleteSelected   int // 0 = No, 1 = Yes

	// Playbook service for operations
	playbookService *ansible.PlaybookService

	// Error message
	err     error
	errTime time.Time
}

type playbooksStyles struct {
	title         lipgloss.Style
	listItem      lipgloss.Style
	listItemFocus lipgloss.Style
	panel         lipgloss.Style
	panelTitle    lipgloss.Style
	info          lipgloss.Style
	infoLabel     lipgloss.Style
	preview       lipgloss.Style
	help          lipgloss.Style
	confirmBox    lipgloss.Style
	button        lipgloss.Style
	buttonFocus   lipgloss.Style
	warning       lipgloss.Style
	error         lipgloss.Style
}

func newPlaybooksStyles() playbooksStyles {
	return playbooksStyles{
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#3B82F6")).
			MarginTop(10).
			MarginBottom(1),
		listItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			PaddingLeft(2),
		listItemFocus: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8FAFC")).
			Bold(true).
			PaddingLeft(1),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#475569")).
			Padding(1),
		panelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#06B6D4")).
			MarginBottom(1),
		info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")),
		infoLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B")),
		preview: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A5B4FC")),
		help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B")).
			MarginTop(1),
		confirmBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#EF4444")).
			Padding(1, 2),
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
		warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FACC15")),
		error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")),
	}
}

// playbooksKeyMap defines keys for the playbooks view
type playbooksKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Delete key.Binding
	Open   key.Binding
	Escape key.Binding
	Enter  key.Binding
	Left   key.Binding
	Right  key.Binding
	Tab    key.Binding
	Yes    key.Binding
	No     key.Binding
}

var playbooksKeys = playbooksKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "move down"),
	),
	Delete: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete"),
	),
	Open: key.NewBinding(
		key.WithKeys("ctrl+g"),
		key.WithHelp("ctrl+g", "open in vscode"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc/q", "back"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
	),
	Yes: key.NewBinding(
		key.WithKeys("y", "Y"),
	),
	No: key.NewBinding(
		key.WithKeys("n", "N"),
	),
}

// NewPlaybooksModel creates a new playbooks browser model
func NewPlaybooksModel(playbookService *ansible.PlaybookService) PlaybooksModel {
	vp := viewport.New(40, 10)
	vp.Style = lipgloss.NewStyle()

	return PlaybooksModel{
		playbooks:       make([]*ansible.PlaybookWithTasks, 0),
		selected:        0,
		styles:          newPlaybooksStyles(),
		previewViewport: vp,
		playbookService: playbookService,
	}
}

// LoadPlaybooks fetches playbooks from the service
func (m *PlaybooksModel) LoadPlaybooks() error {
	ctx := context.Background()

	playbooks, err := m.playbookService.ListPlaybooks(ctx, nil)
	if err != nil {
		return err
	}

	m.playbooks = make([]*ansible.PlaybookWithTasks, 0, len(playbooks))
	for _, pb := range playbooks {
		pbWithTasks, err := m.playbookService.GetPlaybookWithTasks(ctx, pb.ID)
		if err != nil {
			continue
		}
		m.playbooks = append(m.playbooks, pbWithTasks)
	}

	if m.selected >= len(m.playbooks) {
		m.selected = len(m.playbooks) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}

	return nil
}

// Init implements tea.Model
func (m PlaybooksModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m PlaybooksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle delete confirmation mode
		if m.confirmingDelete {
			switch {
			case key.Matches(msg, playbooksKeys.Left):
				m.deleteSelected = 0
			case key.Matches(msg, playbooksKeys.Right):
				m.deleteSelected = 1
			case key.Matches(msg, playbooksKeys.Tab):
				m.deleteSelected = (m.deleteSelected + 1) % 2
			case key.Matches(msg, playbooksKeys.Yes):
				m.deleteSelected = 1
				return m.confirmDelete()
			case key.Matches(msg, playbooksKeys.No), key.Matches(msg, playbooksKeys.Escape):
				m.confirmingDelete = false
				m.deleteSelected = 0
			case key.Matches(msg, playbooksKeys.Enter):
				return m.confirmDelete()
			}
			return m, nil
		}

		// Normal mode
		switch {
		case key.Matches(msg, playbooksKeys.Up):
			if m.selected > 0 {
				m.selected--
				m.updatePreview()
			}
		case key.Matches(msg, playbooksKeys.Down):
			if m.selected < len(m.playbooks)-1 {
				m.selected++
				m.updatePreview()
			}
		case key.Matches(msg, playbooksKeys.Delete):
			if len(m.playbooks) > 0 {
				m.confirmingDelete = true
				m.deleteSelected = 0 // Default to No
			}
		case key.Matches(msg, playbooksKeys.Open):
			return m.openInVSCode()
		case key.Matches(msg, playbooksKeys.Escape):
			return m, func() tea.Msg { return PlaybooksCloseMsg{} }
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case PlaybookDeletedMsg:
		// Reload playbooks after deletion
		if err := m.LoadPlaybooks(); err != nil {
			m.err = err
			m.errTime = time.Now()
		}
		m.updatePreview()

	case PlaybookDeleteErrorMsg:
		m.err = msg.Err
		m.errTime = time.Now()
	}

	// Update viewport
	var cmd tea.Cmd
	m.previewViewport, cmd = m.previewViewport.Update(msg)

	return m, cmd
}

func (m *PlaybooksModel) updateLayout() {
	// Calculate dimensions for the preview viewport
	// Left panel: ~35% of width, Right panel: ~65% of width
	// Reserve space for borders, titles, help, and top padding
	previewWidth := m.width*65/100 - 4
	previewHeight := m.height - 15 // Extra space for top padding and details

	if previewWidth < 20 {
		previewWidth = 20
	}
	if previewHeight < 5 {
		previewHeight = 5
	}

	m.previewViewport.Width = previewWidth
	m.previewViewport.Height = previewHeight
	m.updatePreview()
}

func (m *PlaybooksModel) updatePreview() {
	if len(m.playbooks) == 0 || m.selected >= len(m.playbooks) {
		m.previewViewport.SetContent("No playbook selected")
		return
	}

	pb := m.playbooks[m.selected]

	// Get YAML content
	ctx := context.Background()
	yamlContent, err := m.playbookService.ExportPlaybook(ctx, pb.Playbook.ID)
	if err != nil {
		m.previewViewport.SetContent(fmt.Sprintf("Error loading preview: %v", err))
		return
	}

	// Apply syntax highlighting
	highlighted := highlightYAML(string(yamlContent))
	m.previewViewport.SetContent(highlighted)
	m.previewViewport.GotoTop()
}

// highlightYAML applies syntax highlighting to YAML content
func highlightYAML(content string) string {
	// Get YAML lexer
	lexer := lexers.Get("yaml")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Use a dark theme that works well in terminals
	style := styles.Get("tokyonight-moon")
	if style == nil {
		style = styles.Fallback
	}

	// Use terminal256 formatter for colored output
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize and format
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return content
	}

	return buf.String()
}

func (m PlaybooksModel) confirmDelete() (tea.Model, tea.Cmd) {
	m.confirmingDelete = false

	if m.deleteSelected == 0 {
		// User chose No
		return m, nil
	}

	// User chose Yes - delete the playbook
	if len(m.playbooks) == 0 || m.selected >= len(m.playbooks) {
		return m, nil
	}

	pb := m.playbooks[m.selected]
	return m, func() tea.Msg {
		ctx := context.Background()
		if err := m.playbookService.DeletePlaybook(ctx, pb.Playbook.ID); err != nil {
			return PlaybookDeleteErrorMsg{Err: err}
		}
		return PlaybookDeletedMsg{ID: pb.Playbook.ID, Name: pb.Playbook.Name}
	}
}

func (m PlaybooksModel) openInVSCode() (tea.Model, tea.Cmd) {
	if len(m.playbooks) == 0 || m.selected >= len(m.playbooks) {
		return m, nil
	}

	pb := m.playbooks[m.selected]
	if pb.Playbook.FilePath == nil || *pb.Playbook.FilePath == "" {
		m.err = fmt.Errorf("playbook has no file path")
		m.errTime = time.Now()
		return m, nil
	}

	filePath := *pb.Playbook.FilePath

	// Run code command in background
	return m, func() tea.Msg {
		cmd := exec.Command("code", filePath)
		if err := cmd.Start(); err != nil {
			return PlaybookDeleteErrorMsg{Err: fmt.Errorf("failed to open in vscode: %w", err)}
		}
		return nil
	}
}

// View implements tea.Model
func (m PlaybooksModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Add top padding
	b.WriteString("\n")

	// Title
	b.WriteString(m.styles.title.Render("  Playbooks"))
	b.WriteString("\n")

	// Calculate panel widths
	leftWidth := m.width*35/100 - 2
	rightWidth := m.width*65/100 - 2

	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 30 {
		rightWidth = 30
	}

	// Left panel - playbook list
	var leftContent strings.Builder
	leftContent.WriteString(m.styles.panelTitle.Render("Playbooks"))
	leftContent.WriteString("\n")

	if len(m.playbooks) == 0 {
		leftContent.WriteString(m.styles.info.Render("No playbooks found"))
	} else {
		// Calculate visible items
		visibleHeight := m.height - 10
		if visibleHeight < 3 {
			visibleHeight = 3
		}

		startIdx := 0
		if m.selected >= visibleHeight {
			startIdx = m.selected - visibleHeight + 1
		}

		for i := startIdx; i < len(m.playbooks) && i < startIdx+visibleHeight; i++ {
			pb := m.playbooks[i]
			name := pb.Playbook.Name
			if len(name) > leftWidth-4 {
				name = name[:leftWidth-7] + "..."
			}

			if i == m.selected {
				leftContent.WriteString(m.styles.listItemFocus.Render("> " + name))
			} else {
				leftContent.WriteString(m.styles.listItem.Render("  " + name))
			}
			leftContent.WriteString("\n")
		}

		// Scroll indicator
		if len(m.playbooks) > visibleHeight {
			leftContent.WriteString("\n")
			leftContent.WriteString(m.styles.info.Render(fmt.Sprintf("  %d/%d", m.selected+1, len(m.playbooks))))
		}
	}

	leftPanel := m.styles.panel.Width(leftWidth).Render(leftContent.String())

	// Right panel - details and preview
	var rightContent strings.Builder

	if len(m.playbooks) > 0 && m.selected < len(m.playbooks) {
		pb := m.playbooks[m.selected]

		// Details section
		rightContent.WriteString(m.styles.panelTitle.Render("Details"))
		rightContent.WriteString("\n")

		rightContent.WriteString(m.styles.infoLabel.Render("Name:     "))
		rightContent.WriteString(m.styles.info.Render(pb.Playbook.Name))
		rightContent.WriteString("\n")

		path := "-"
		if pb.Playbook.FilePath != nil && *pb.Playbook.FilePath != "" {
			path = *pb.Playbook.FilePath
			if len(path) > rightWidth-12 {
				path = "..." + path[len(path)-(rightWidth-15):]
			}
		}
		rightContent.WriteString(m.styles.infoLabel.Render("Path:     "))
		rightContent.WriteString(m.styles.info.Render(path))
		rightContent.WriteString("\n")

		rightContent.WriteString(m.styles.infoLabel.Render("Created:  "))
		rightContent.WriteString(m.styles.info.Render(pb.Playbook.CreatedAt.Format("2006-01-02 15:04")))
		rightContent.WriteString("\n")

		rightContent.WriteString(m.styles.infoLabel.Render("Tasks:    "))
		rightContent.WriteString(m.styles.info.Render(fmt.Sprintf("%d", len(pb.Tasks))))
		rightContent.WriteString("\n\n")

		// Preview section
		rightContent.WriteString(m.styles.panelTitle.Render("Preview"))
		rightContent.WriteString("\n")
		rightContent.WriteString(m.styles.preview.Render(m.previewViewport.View()))
	} else {
		rightContent.WriteString(m.styles.info.Render("Select a playbook to view details"))
	}

	rightPanel := m.styles.panel.Width(rightWidth).Render(rightContent.String())

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
	b.WriteString(panels)
	b.WriteString("\n")

	// Error message
	if m.err != nil && time.Since(m.errTime) < 5*time.Second {
		b.WriteString(m.styles.error.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	// Help line
	helpText := "up/down: navigate | x: delete | ctrl+g: open in vscode | esc: back"
	b.WriteString(m.styles.help.Render(helpText))

	// Overlay delete confirmation if active
	if m.confirmingDelete {
		return m.renderWithConfirmDialog(b.String())
	}

	return b.String()
}

func (m PlaybooksModel) renderWithConfirmDialog(base string) string {
	if len(m.playbooks) == 0 || m.selected >= len(m.playbooks) {
		return base
	}

	pb := m.playbooks[m.selected]

	var dialog strings.Builder
	dialog.WriteString(m.styles.warning.Bold(true).Render("Delete Playbook?"))
	dialog.WriteString("\n\n")
	dialog.WriteString(m.styles.info.Render(fmt.Sprintf("Are you sure you want to delete '%s'?", pb.Playbook.Name)))
	dialog.WriteString("\n")
	dialog.WriteString(m.styles.info.Render("This action cannot be undone."))
	dialog.WriteString("\n\n")

	// Buttons
	var noBtn, yesBtn string
	if m.deleteSelected == 0 {
		noBtn = m.styles.buttonFocus.Render(" [ No ] ")
		yesBtn = m.styles.button.Render("   Yes   ")
	} else {
		noBtn = m.styles.button.Render("   No   ")
		yesBtn = m.styles.buttonFocus.Render(" [ Yes ] ")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, noBtn, "    ", yesBtn)
	dialog.WriteString(buttons)
	dialog.WriteString("\n\n")
	dialog.WriteString(m.styles.help.Render("<-/-> or Tab: select | Enter: confirm | y/n: quick select | Esc: cancel"))

	confirmDialog := m.styles.confirmBox.Render(dialog.String())

	// Center the dialog over the base view
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, confirmDialog)
	}

	return confirmDialog
}
