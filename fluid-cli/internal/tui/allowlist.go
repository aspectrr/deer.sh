package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/fluid.sh/fluid-cli/internal/config"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/readonly"
)

type allowlistMode int

const (
	allowlistModeList allowlistMode = iota
	allowlistModeAdd
)

type allowlistStyles struct {
	title       lipgloss.Style
	help        lipgloss.Style
	section     lipgloss.Style
	command     lipgloss.Style
	userCommand lipgloss.Style
	dimmed      lipgloss.Style
	error       lipgloss.Style
	success     lipgloss.Style
	indicator   lipgloss.Style
}

func defaultAllowlistStyles() allowlistStyles {
	return allowlistStyles{
		title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3B82F6")),
		help:        lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		section:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4")),
		command:     lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")),
		userCommand: lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		dimmed:      lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		error:       lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")),
		success:     lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		indicator:   lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")),
	}
}

type AllowlistModel struct {
	cfg              *config.Config
	width            int
	height           int
	styles           allowlistStyles
	builtinCmds      []string
	subcommandRestrs map[string][]string
	userCmds         []string
	mode             allowlistMode
	selected         int
	scrollY          int
	addInput         textinput.Model
	addErr           string
}

func NewAllowlistModel(cfg *config.Config) AllowlistModel {
	addInput := textinput.New()
	addInput.Prompt = "Command: "
	addInput.Placeholder = "e.g., custom-tool"
	addInput.Focus()

	m := AllowlistModel{
		cfg:              cfg,
		styles:           defaultAllowlistStyles(),
		builtinCmds:      readonly.AllowedCommandsList(),
		subcommandRestrs: readonly.SubcommandRestrictions(),
		userCmds:         make([]string, len(cfg.ExtraAllowedCommands)),
		mode:             allowlistModeList,
		selected:         0,
		scrollY:          0,
		addInput:         addInput,
	}
	copy(m.userCmds, cfg.ExtraAllowedCommands)
	sort.Strings(m.userCmds)

	return m
}

func (m AllowlistModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m AllowlistModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.mode == allowlistModeAdd {
				m.mode = allowlistModeList
				m.addInput.SetValue("")
				m.addErr = ""
				return m, nil
			}
			return m, func() tea.Msg { return AllowlistCloseMsg{Saved: false} }

		case "ctrl+s":
			return m, func() tea.Msg { return AllowlistCloseMsg{Saved: true} }

		case "up":
			if m.mode == allowlistModeList {
				if m.selected > 0 {
					m.selected--
					m.ensureVisible()
				}
			}
			return m, nil

		case "down":
			if m.mode == allowlistModeList {
				totalItems := len(m.builtinCmds) + len(m.userCmds)
				if m.selected < totalItems-1 {
					m.selected++
					m.ensureVisible()
				}
			}
			return m, nil

		case "ctrl+n":
			if m.mode == allowlistModeList {
				m.mode = allowlistModeAdd
				m.addInput.Focus()
			}
			return m, nil

		case "enter":
			if m.mode == allowlistModeAdd {
				newCmd := strings.TrimSpace(m.addInput.Value())
				if newCmd == "" {
					m.addErr = "command cannot be empty"
					return m, nil
				}
				for _, cmd := range m.builtinCmds {
					if cmd == newCmd {
						m.addErr = "command already in allowlist"
						return m, nil
					}
				}
				for _, cmd := range m.userCmds {
					if cmd == newCmd {
						m.addErr = "command already in allowlist"
						return m, nil
					}
				}
				m.userCmds = append(m.userCmds, newCmd)
				sort.Strings(m.userCmds)
				m.cfg.ExtraAllowedCommands = m.userCmds
				m.mode = allowlistModeList
				m.addInput.SetValue("")
				m.addErr = ""
				m.selected = len(m.builtinCmds) + len(m.userCmds) - 1
				m.ensureVisible()
			}
			return m, nil

		case "d":
			if m.mode == allowlistModeList {
				totalBuiltins := len(m.builtinCmds)
				if m.selected >= totalBuiltins {
					idx := m.selected - totalBuiltins
					m.userCmds = append(m.userCmds[:idx], m.userCmds[idx+1:]...)
					m.cfg.ExtraAllowedCommands = m.userCmds
					if m.selected >= len(m.builtinCmds)+len(m.userCmds) && m.selected > 0 {
						m.selected--
					}
					m.ensureVisible()
				}
			}
			return m, nil
		}
	}

	if m.mode == allowlistModeAdd {
		var cmd tea.Cmd
		m.addInput, cmd = m.addInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *AllowlistModel) ensureVisible() {
	visibleItems := m.visibleItemCount()
	if m.selected < m.scrollY {
		m.scrollY = m.selected
	}
	if m.selected >= m.scrollY+visibleItems {
		m.scrollY = m.selected - visibleItems + 1
	}
	totalItems := len(m.builtinCmds) + len(m.userCmds)
	maxScroll := totalItems - visibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollY > maxScroll {
		m.scrollY = maxScroll
	}
}

func (m AllowlistModel) visibleItemCount() int {
	if m.height <= 0 {
		return 10
	}
	available := m.height - 10
	if available < 4 {
		return 4
	}
	return available
}

func (m AllowlistModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.title.Render("Command Allowlist"))
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render("Up/down: navigate | Ctrl+N: add | D: delete | Ctrl+S: save | Esc: close"))
	b.WriteString("\n")

	if m.mode == allowlistModeAdd {
		b.WriteString(m.styles.section.Render("--- Add Command ---"))
		b.WriteString("\n")
		b.WriteString(m.addInput.View())
		b.WriteString("\n")
		if m.addErr != "" {
			b.WriteString(m.styles.error.Render(m.addErr))
			b.WriteString("\n")
		}
		return b.String()
	}

	totalItems := len(m.builtinCmds) + len(m.userCmds)
	visibleStart := m.scrollY
	visibleEnd := m.scrollY + m.visibleItemCount()
	if visibleEnd > totalItems {
		visibleEnd = totalItems
	}

	b.WriteString(m.styles.section.Render("--- Builtin Commands ---"))
	b.WriteString("\n")

	for i := visibleStart; i < visibleEnd && i < len(m.builtinCmds); i++ {
		b.WriteString(m.renderCommandRow(i, false))
	}

	if len(m.userCmds) > 0 {
		b.WriteString("\n")
		b.WriteString(m.styles.section.Render("--- User Commands ---"))
		b.WriteString("\n")

		userStart := len(m.builtinCmds)
		for i := visibleStart; i < visibleEnd; i++ {
			if i >= userStart {
				userIdx := i - userStart
				if userIdx < len(m.userCmds) {
					b.WriteString(m.renderCommandRow(i, true))
				}
			}
		}
	}

	totalFields := totalItems
	scrollPct := 0
	if totalFields > m.visibleItemCount() {
		scrollPct = (m.scrollY * 100) / (totalFields - m.visibleItemCount())
	}

	b.WriteString("\n")
	scrollIndicator := fmt.Sprintf("Item %d/%d", m.selected+1, totalFields)
	if totalFields > m.visibleItemCount() {
		barWidth := 20
		filledWidth := (scrollPct * barWidth) / 100
		if filledWidth < 1 && m.scrollY > 0 {
			filledWidth = 1
		}
		scrollBar := strings.Repeat("#", filledWidth) + strings.Repeat(".", barWidth-filledWidth)
		scrollIndicator += fmt.Sprintf(" [%s] %d%%", scrollBar, scrollPct)
	}
	b.WriteString(m.styles.help.Render(scrollIndicator))

	return b.String()
}

func (m AllowlistModel) renderCommandRow(idx int, isUser bool) string {
	prefix := "  "
	style := m.styles.command
	if idx == m.selected {
		prefix = m.styles.indicator.Render("> ")
		if isUser {
			style = m.styles.userCommand.Bold(true)
		} else {
			style = m.styles.command.Bold(true)
		}
	}

	cmd := ""
	if idx < len(m.builtinCmds) {
		cmd = m.builtinCmds[idx]
		subs, ok := m.subcommandRestrs[cmd]
		if ok {
			return fmt.Sprintf("%s%s %s\n", prefix, style.Render(cmd), m.styles.dimmed.Render("("+strings.Join(subs, ", ")+")"))
		}
		return fmt.Sprintf("%s%s %s\n", prefix, style.Render(cmd), m.styles.dimmed.Render("(all subcommands)"))
	}

	userIdx := idx - len(m.builtinCmds)
	if userIdx < len(m.userCmds) {
		cmd = m.userCmds[userIdx]
		delTag := ""
		if isUser {
			delTag = m.styles.dimmed.Render(" [D to delete]")
		}
		return fmt.Sprintf("%s%s%s\n", prefix, m.styles.userCommand.Render(cmd), delTag)
	}

	return ""
}

func (m AllowlistModel) GetConfig() *config.Config {
	return m.cfg
}
