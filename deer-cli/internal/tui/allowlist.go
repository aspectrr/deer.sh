package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
	"github.com/aspectrr/deer.sh/deer-cli/internal/readonly"
)

type allowlistMode int

const (
	allowlistModeList allowlistMode = iota
	allowlistModeAdd
	allowlistModeDetail
)

type allowlistStyles struct {
	title           lipgloss.Style
	help            lipgloss.Style
	section         lipgloss.Style
	command         lipgloss.Style
	userCommand     lipgloss.Style
	dimmed          lipgloss.Style
	error           lipgloss.Style
	success         lipgloss.Style
	indicator       lipgloss.Style
	toggleAllowlist lipgloss.Style
	toggleBlocklist lipgloss.Style
}

func defaultAllowlistStyles() allowlistStyles {
	return allowlistStyles{
		title:           lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#166534")),
		help:            lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		section:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#15803d")),
		command:         lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")),
		userCommand:     lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		dimmed:          lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		error:           lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")),
		success:         lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		indicator:       lipgloss.NewStyle().Foreground(lipgloss.Color("#166534")),
		toggleAllowlist: lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Background(lipgloss.Color("#EDE9FE")),
		toggleBlocklist: lipgloss.NewStyle().Foreground(lipgloss.Color("#D97706")).Background(lipgloss.Color("#FEF3C7")),
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

	detailCommand         string
	detailSubcmds         []string
	detailIsBuiltin       map[string]bool
	detailSelected        int
	detailScrollY         int
	detailAddInput        textinput.Model
	detailAddErr          string
	detailModeIsAllowlist bool
}

func NewAllowlistModel(cfg *config.Config) AllowlistModel {
	addInput := textinput.New()
	addInput.Prompt = "Command: "
	addInput.Placeholder = "e.g., custom-tool"
	addInput.Focus()

	detailAddInput := textinput.New()
	detailAddInput.Prompt = "Subcommand: "
	detailAddInput.Placeholder = "status"
	detailAddInput.Width = 30
	detailAddInput.Focus()
	detailAddInput.SetValue("")

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
		detailAddInput:   detailAddInput,
	}
	copy(m.userCmds, cfg.ExtraAllowedCommands)
	sort.Strings(m.userCmds)

	if cfg.ExtraAllowedSubcommands == nil {
		cfg.ExtraAllowedSubcommands = make(map[string][]string)
	}
	if cfg.ExtraAllowedSubcommandsMode == nil {
		cfg.ExtraAllowedSubcommandsMode = make(map[string]bool)
	}

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
			if m.mode == allowlistModeDetail {
				m.mode = allowlistModeList
				m.detailCommand = ""
				m.detailSubcmds = nil
				m.detailIsBuiltin = nil
				m.detailSelected = 0
				m.detailScrollY = 0
				m.detailAddInput.SetValue("")
				m.detailAddErr = ""
				return m, nil
			}
			return m, func() tea.Msg { return AllowlistCloseMsg{Saved: false} }

		case "ctrl+s":
			return m, func() tea.Msg { return AllowlistCloseMsg{Saved: true} }

		case "up":
			switch m.mode {
			case allowlistModeList:
				if m.selected > 0 {
					m.selected--
					m.ensureVisible()
				}
			case allowlistModeDetail:
				if m.detailSelected > 0 {
					m.detailSelected--
					m.ensureDetailVisible()
				}
			}
			return m, nil

		case "down":
			switch m.mode {
			case allowlistModeList:
				totalItems := len(m.builtinCmds) + len(m.userCmds)
				if m.selected < totalItems-1 {
					m.selected++
					m.ensureVisible()
				}
			case allowlistModeDetail:
				if m.detailSelected < len(m.detailSubcmds)-1 {
					m.detailSelected++
					m.ensureDetailVisible()
				}
			}
			return m, nil

		case "ctrl+n":
			switch m.mode {
			case allowlistModeList:
				m.mode = allowlistModeAdd
				m.addInput.Focus()
			case allowlistModeDetail:
				m.detailAddInput.Focus()
			}
			return m, nil

		case "ctrl+t":
			if m.mode == allowlistModeDetail {
				m.detailModeIsAllowlist = !m.detailModeIsAllowlist
				m.cfg.ExtraAllowedSubcommandsMode[m.detailCommand] = m.detailModeIsAllowlist
			}
			return m, nil

		case "enter":
			switch m.mode {
			case allowlistModeAdd:
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
			case allowlistModeDetail:
				m.enterDetailAddSubcommand()
			default:
				m.enterDetailMode()
			}
			return m, nil

		case "d":
			switch m.mode {
			case allowlistModeList:
				totalBuiltins := len(m.builtinCmds)
				if m.selected >= totalBuiltins {
					idx := m.selected - totalBuiltins
					cmdToDelete := m.userCmds[idx]
					m.userCmds = append(m.userCmds[:idx], m.userCmds[idx+1:]...)
					m.cfg.ExtraAllowedCommands = m.userCmds
					delete(m.cfg.ExtraAllowedSubcommands, cmdToDelete)
					if m.selected >= len(m.builtinCmds)+len(m.userCmds) && m.selected > 0 {
						m.selected--
					}
					m.ensureVisible()
				}
			case allowlistModeDetail:
				m.deleteDetailSubcommand()
			}
			return m, nil
		}
	}

	switch m.mode {
	case allowlistModeAdd:
		var cmd tea.Cmd
		m.addInput, cmd = m.addInput.Update(msg)
		cmds = append(cmds, cmd)
	case allowlistModeDetail:
		var cmd tea.Cmd
		m.detailAddInput, cmd = m.detailAddInput.Update(msg)
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

func (m *AllowlistModel) ensureDetailVisible() {
	visibleItems := m.detailVisibleItemCount()
	if m.detailSelected < m.detailScrollY {
		m.detailScrollY = m.detailSelected
	}
	if m.detailSelected >= m.detailScrollY+visibleItems {
		m.detailScrollY = m.detailSelected - visibleItems + 1
	}
	totalItems := len(m.detailSubcmds)
	maxScroll := totalItems - visibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.detailScrollY > maxScroll {
		m.detailScrollY = maxScroll
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

func (m AllowlistModel) detailVisibleItemCount() int {
	if m.height <= 0 {
		return 10
	}
	available := m.height - 12
	if available < 4 {
		return 4
	}
	return available
}

func (m *AllowlistModel) enterDetailMode() {
	cmd := ""
	isBuiltin := false
	if m.selected < len(m.builtinCmds) {
		cmd = m.builtinCmds[m.selected]
		isBuiltin = true
	} else {
		userIdx := m.selected - len(m.builtinCmds)
		if userIdx < len(m.userCmds) {
			cmd = m.userCmds[userIdx]
		}
	}
	if cmd == "" {
		return
	}

	m.detailCommand = cmd
	m.detailSubcmds = []string{}
	m.detailIsBuiltin = make(map[string]bool)

	if builtinSubs, ok := m.subcommandRestrs[cmd]; ok {
		for _, sub := range builtinSubs {
			m.detailSubcmds = append(m.detailSubcmds, sub)
			m.detailIsBuiltin[sub] = true
		}
	}

	if userSubs, ok := m.cfg.ExtraAllowedSubcommands[cmd]; ok {
		for _, sub := range userSubs {
			if !m.detailIsBuiltin[sub] {
				m.detailSubcmds = append(m.detailSubcmds, sub)
				m.detailIsBuiltin[sub] = false
			}
		}
	}

	sort.Strings(m.detailSubcmds)

	if isBuiltin {
		m.detailModeIsAllowlist = true
	} else {
		if len(m.detailSubcmds) > 0 {
			if mode, ok := m.cfg.ExtraAllowedSubcommandsMode[cmd]; ok {
				m.detailModeIsAllowlist = mode
			} else {
				m.detailModeIsAllowlist = true
			}
		} else {
			m.detailModeIsAllowlist = true
		}
	}

	m.detailSelected = 0
	m.detailScrollY = 0
	m.detailAddInput.SetValue("")
	m.mode = allowlistModeDetail
}

func (m *AllowlistModel) enterDetailAddSubcommand() {
	newSub := strings.TrimSpace(m.detailAddInput.Value())
	if newSub == "" {
		m.detailAddErr = "subcommand cannot be empty"
		return
	}
	for _, sub := range m.detailSubcmds {
		if sub == newSub {
			m.detailAddErr = "subcommand already exists"
			return
		}
	}

	if !m.detailIsBuiltin[newSub] {
		m.cfg.ExtraAllowedSubcommands[m.detailCommand] = append(
			m.cfg.ExtraAllowedSubcommands[m.detailCommand],
			newSub,
		)
		sort.Strings(m.cfg.ExtraAllowedSubcommands[m.detailCommand])
	}

	m.detailSubcmds = append(m.detailSubcmds, newSub)
	m.detailIsBuiltin[newSub] = false
	sort.Strings(m.detailSubcmds)

	for i, sub := range m.detailSubcmds {
		if sub == newSub {
			m.detailSelected = i
			break
		}
	}

	m.detailAddInput.SetValue("")
	m.detailAddErr = ""
	m.ensureDetailVisible()
}

func (m *AllowlistModel) deleteDetailSubcommand() {
	if m.detailSelected >= len(m.detailSubcmds) {
		return
	}

	sub := m.detailSubcmds[m.detailSelected]
	isBuiltin := m.detailIsBuiltin[sub]

	if !isBuiltin {
		subs := m.cfg.ExtraAllowedSubcommands[m.detailCommand]
		for i, s := range subs {
			if s == sub {
				m.cfg.ExtraAllowedSubcommands[m.detailCommand] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}

	m.detailSubcmds = append(m.detailSubcmds[:m.detailSelected], m.detailSubcmds[m.detailSelected+1:]...)
	delete(m.detailIsBuiltin, sub)

	if m.detailSelected >= len(m.detailSubcmds) && m.detailSelected > 0 {
		m.detailSelected--
	}

	if len(m.detailSubcmds) == 0 && m.detailModeIsAllowlist {
		m.detailModeIsAllowlist = false
		m.cfg.ExtraAllowedSubcommandsMode[m.detailCommand] = false
	}

	m.ensureDetailVisible()
}

func (m AllowlistModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.title.Render("Command Allowlist"))
	b.WriteString("\n")

	if m.mode == allowlistModeDetail {
		b.WriteString(m.styles.help.Render("Up/down: navigate | Ctrl+N: add | D: delete | Ctrl+T: toggle mode | Esc: back"))
	} else {
		b.WriteString(m.styles.help.Render("Up/down: navigate | Enter: details | Ctrl+N: add | D: delete | Ctrl+S: save | Esc: close"))
	}
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

	if m.mode == allowlistModeDetail {
		return m.renderDetailView(&b)
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

	detailTag := m.styles.dimmed.Render(" [Enter for details]")

	cmd := ""
	if idx < len(m.builtinCmds) {
		cmd = m.builtinCmds[idx]

		userSubs, hasUserSubs := m.cfg.ExtraAllowedSubcommands[cmd]
		subs, hasBuiltinSubs := m.subcommandRestrs[cmd]

		if hasUserSubs && len(userSubs) > 0 {
			isAllowlist := m.cfg.ExtraAllowedSubcommandsMode[cmd]
			var subInfo string
			if isAllowlist {
				subInfo = fmt.Sprintf("(only: %s)", strings.Join(userSubs, ", "))
			} else {
				subInfo = fmt.Sprintf("(all except: %s)", strings.Join(userSubs, ", "))
			}
			return fmt.Sprintf("%s%s %s%s\n", prefix, style.Render(cmd), m.styles.dimmed.Render(subInfo), detailTag)
		}

		if hasBuiltinSubs {
			return fmt.Sprintf("%s%s %s%s\n", prefix, style.Render(cmd), m.styles.dimmed.Render("("+strings.Join(subs, ", ")+")"), detailTag)
		}
		return fmt.Sprintf("%s%s %s%s\n", prefix, style.Render(cmd), m.styles.dimmed.Render("(all subcommands)"), detailTag)
	}

	userIdx := idx - len(m.builtinCmds)
	if userIdx < len(m.userCmds) {
		cmd = m.userCmds[userIdx]

		userSubs := m.cfg.ExtraAllowedSubcommands[cmd]
		isAllowlist := m.cfg.ExtraAllowedSubcommandsMode[cmd]

		var subInfo string
		if len(userSubs) > 0 {
			if isAllowlist {
				subInfo = fmt.Sprintf("(only: %s)", strings.Join(userSubs, ", "))
			} else {
				subInfo = fmt.Sprintf("(all except: %s)", strings.Join(userSubs, ", "))
			}
		} else {
			subInfo = "(all subcommands)"
		}

		tags := ""
		if isUser {
			tags = m.styles.dimmed.Render(" [D to delete]")
		}
		tags += detailTag
		return fmt.Sprintf("%s%s %s%s\n", prefix, m.styles.userCommand.Render(cmd), m.styles.dimmed.Render(subInfo), tags)
	}

	return ""
}

func (m AllowlistModel) GetConfig() *config.Config {
	return m.cfg
}

func (m AllowlistModel) renderModeToggle() string {
	var sb strings.Builder

	sb.WriteString(m.styles.help.Render("Esc: back  "))

	if m.detailModeIsAllowlist {
		sb.WriteString(m.styles.toggleAllowlist.Render("[■ Allowlist]"))
		sb.WriteString(m.styles.help.Render("|"))
		sb.WriteString(m.styles.dimmed.Render("[□ Blocklist]"))
	} else {
		sb.WriteString(m.styles.dimmed.Render("[□ Allowlist]"))
		sb.WriteString(m.styles.help.Render("|"))
		sb.WriteString(m.styles.toggleBlocklist.Render("[■ Blocklist]"))
	}

	return sb.String()
}

func (m AllowlistModel) renderDetailView(b *strings.Builder) string {
	b.WriteString(m.styles.section.Render("--- " + m.detailCommand + " ---"))
	b.WriteString("\n")

	b.WriteString(m.renderModeToggle())
	b.WriteString("\n")

	if len(m.detailSubcmds) == 0 {
		if m.detailModeIsAllowlist {
			b.WriteString(m.styles.dimmed.Render("  (blocking all except listed)"))
		} else {
			b.WriteString(m.styles.dimmed.Render("  (allowing all except listed)"))
		}
		b.WriteString("\n")
	} else {
		visibleStart := m.detailScrollY
		visibleEnd := m.detailScrollY + m.detailVisibleItemCount()
		if visibleEnd > len(m.detailSubcmds) {
			visibleEnd = len(m.detailSubcmds)
		}

		for i := visibleStart; i < visibleEnd; i++ {
			sub := m.detailSubcmds[i]
			isBuiltin := m.detailIsBuiltin[sub]

			prefix := "  "
			style := m.styles.command
			if i == m.detailSelected {
				prefix = m.styles.indicator.Render("> ")
				style = m.styles.command.Bold(true)
			}

			tag := ""
			if isBuiltin {
				tag = m.styles.dimmed.Render(" [builtin]")
			} else {
				tag = m.styles.dimmed.Render(" [user]")
			}

			fmt.Fprintf(b, "%s%s%s\n", prefix, style.Render(sub), tag)
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.section.Render("--- Add Subcommand ---"))
	b.WriteString("\n")
	b.WriteString(m.detailAddInput.View())
	b.WriteString("\n")
	if m.detailAddErr != "" {
		b.WriteString(m.styles.error.Render(m.detailAddErr))
		b.WriteString("\n")
	}

	totalFields := len(m.detailSubcmds)
	scrollPct := 0
	if totalFields > m.detailVisibleItemCount() {
		scrollPct = (m.detailScrollY * 100) / (totalFields - m.detailVisibleItemCount())
	}

	b.WriteString("\n")
	scrollIndicator := fmt.Sprintf("Subcommand %d/%d", m.detailSelected+1, totalFields)
	if totalFields > m.detailVisibleItemCount() {
		barWidth := 20
		filledWidth := (scrollPct * barWidth) / 100
		if filledWidth < 1 && m.detailScrollY > 0 {
			filledWidth = 1
		}
		scrollBar := strings.Repeat("#", filledWidth) + strings.Repeat(".", barWidth-filledWidth)
		scrollIndicator += fmt.Sprintf(" [%s] %d%%", scrollBar, scrollPct)
	}
	b.WriteString(m.styles.help.Render(scrollIndicator))

	return b.String()
}
