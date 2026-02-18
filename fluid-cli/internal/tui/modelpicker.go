package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/modelsdev"
)

// ModelPickerCloseMsg is sent when the model picker is closed.
type ModelPickerCloseMsg struct {
	Selected           bool
	Model              string
	CompactModel       string
	TotalContextTokens int
}

// ModelsLoadedMsg is sent when models finish loading.
type ModelsLoadedMsg struct {
	Models []modelsdev.Model
}

// ModelsLoadErrorMsg is sent when model loading fails.
type ModelsLoadErrorMsg struct {
	Err error
}

// ModelPickerModel is the Bubble Tea model for the /model picker.
type ModelPickerModel struct {
	models           []modelsdev.Model
	cursor           int
	width, height    int
	styles           Styles
	selectingCompact bool
	mainModel        *modelsdev.Model
	loading          bool
	loadErr          error
	scrollY          int
	cfg              *config.Config
}

// NewModelPickerModel creates a new model picker.
func NewModelPickerModel(cfg *config.Config) ModelPickerModel {
	return ModelPickerModel{
		styles:  DefaultStyles(),
		loading: true,
		cfg:     cfg,
	}
}

// Init starts the async model fetch.
func (m ModelPickerModel) Init() tea.Cmd {
	return func() tea.Msg {
		models, err := modelsdev.FetchTopModels(context.Background())
		if err != nil {
			return ModelsLoadErrorMsg{Err: err}
		}
		return ModelsLoadedMsg{Models: models}
	}
}

// Update handles messages for the model picker.
func (m ModelPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ModelsLoadedMsg:
		m.loading = false
		m.models = msg.Models
		m.cursor = 0
		m.scrollY = 0
		return m, nil

	case ModelsLoadErrorMsg:
		m.loading = false
		m.loadErr = msg.Err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, func() tea.Msg {
				return ModelPickerCloseMsg{Selected: false}
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.models)-1 {
				m.cursor++
				m.ensureVisible()
			}
			return m, nil

		case "enter":
			if m.loadErr != nil {
				return m, func() tea.Msg {
					return ModelPickerCloseMsg{Selected: false}
				}
			}
			if len(m.models) == 0 {
				return m, nil
			}

			selected := m.models[m.cursor]

			if !m.selectingCompact {
				// First selection: main model
				m.mainModel = &selected
				m.selectingCompact = true
				m.cursor = 0
				m.scrollY = 0
				return m, nil
			}

			// Second selection: compact model
			return m, func() tea.Msg {
				return ModelPickerCloseMsg{
					Selected:           true,
					Model:              m.mainModel.ID,
					CompactModel:       selected.ID,
					TotalContextTokens: m.mainModel.ContextLimit,
				}
			}
		}
	}

	return m, nil
}

// View renders the model picker.
func (m ModelPickerModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3B82F6"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	if m.selectingCompact {
		b.WriteString(titleStyle.Render("Select Compact Model"))
	} else {
		b.WriteString(titleStyle.Render("Select Model"))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Up/Down: navigate | Enter: select | Esc: cancel"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(helpStyle.Render("Loading models from OpenRouter..."))
		return b.String()
	}

	if m.loadErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Failed to load models: %v", m.loadErr)))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Press Enter or Esc to go back. Configure model manually in /settings."))
		return b.String()
	}

	if len(m.models) == 0 {
		b.WriteString(helpStyle.Render("No models found."))
		return b.String()
	}

	if m.mainModel != nil {
		selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
		b.WriteString(selectedStyle.Render(fmt.Sprintf("Main model: %s", m.mainModel.ID)))
		b.WriteString("\n\n")
	}

	visibleItems := m.visibleCount()
	start := m.scrollY
	end := start + visibleItems
	if end > len(m.models) {
		end = len(m.models)
	}

	nameWidth := 40
	if m.width > 0 && m.width < 100 {
		nameWidth = m.width / 3
	}

	for i := start; i < end; i++ {
		model := m.models[i]
		name := model.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		cost := fmt.Sprintf("$%.2f/$%.2f per 1M", model.InputCostPer1M, model.OutputCostPer1M)
		ctx := fmt.Sprintf("%dk ctx", model.ContextLimit/1000)

		var tags []string
		if model.ToolCall {
			tags = append(tags, "tools")
		}
		if model.Reasoning {
			tags = append(tags, "reasoning")
		}
		tagStr := strings.Join(tags, ",")

		line := fmt.Sprintf("%-*s  %-24s  %-10s  %s", nameWidth, name, cost, ctx, tagStr)

		if i == m.cursor {
			cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true)
			b.WriteString(cursorStyle.Render("> " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}

	if len(m.models) > visibleItems {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("  %d/%d models", m.cursor+1, len(m.models))))
	}

	return b.String()
}

func (m ModelPickerModel) visibleCount() int {
	available := m.height - 8 // header, help, padding
	if m.mainModel != nil {
		available -= 2
	}
	if available < 5 {
		return 5
	}
	return available
}

func (m *ModelPickerModel) ensureVisible() {
	visible := m.visibleCount()
	if m.cursor < m.scrollY {
		m.scrollY = m.cursor
	}
	if m.cursor >= m.scrollY+visible {
		m.scrollY = m.cursor - visible + 1
	}
}
