package tui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
)

type builtinPatternEntry struct {
	name     string
	category string
	pattern  string
	example  string
}

var builtinPatterns = []builtinPatternEntry{
	{"IPv4 Address", "IP", `\b(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\b`, "Server at 192.168.1.100 is up"},
	{"IPv6 Address", "IP", `(?i)\b(?:[0-9a-f]{1,4}:){7}[0-9a-f]{1,4}\b`, "Address: 2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
	{"API Key", "KEY", `\bsk-[a-zA-Z0-9]{20,}\b`, "token=sk-proj-abc123def456ghi789jkl012mno345"},
	{"AWS Access Key", "KEY", `\bAKIA[0-9A-Z]{16}\b`, "AWS_KEY=AKIAIOSFODNN7EXAMPLE"},
	{"SSH Private Key", "KEY", `-----BEGIN [A-Z ]*PRIVATE KEY-----`, "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA..."},
	{"Connection String", "SECRET", `\b(postgres|mysql|mongodb|redis)://[^\s]+`, "db=postgres://user:pass@localhost:5432/mydb"},
}

type redactionMode int

const (
	redactionModeList redactionMode = iota
	redactionModeAdd
)

type redactionStyles struct {
	title      lipgloss.Style
	help       lipgloss.Style
	section    lipgloss.Style
	enabled    lipgloss.Style
	disabled   lipgloss.Style
	category   lipgloss.Style
	pattern    lipgloss.Style
	custom     lipgloss.Style
	dimmed     lipgloss.Style
	error      lipgloss.Style
	success    lipgloss.Style
	indicator  lipgloss.Style
	highlight  lipgloss.Style
	redacted   lipgloss.Style
	previewBox lipgloss.Style
}

func defaultRedactionStyles() redactionStyles {
	return redactionStyles{
		title:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#166534")),
		help:       lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		section:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#15803d")),
		enabled:    lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		disabled:   lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		category:   lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")),
		pattern:    lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")),
		custom:     lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		dimmed:     lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		error:      lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")),
		success:    lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")),
		indicator:  lipgloss.NewStyle().Foreground(lipgloss.Color("#166534")),
		highlight:  lipgloss.NewStyle().Background(lipgloss.Color("#FEF3C7")).Foreground(lipgloss.Color("#92400E")),
		redacted:   lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		previewBox: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#F59E0B")).Padding(0, 1),
	}
}

type RedactionModel struct {
	cfg            *config.Config
	width          int
	height         int
	styles         redactionStyles
	customPatterns []string
	mode           redactionMode
	selected       int
	scrollY        int
	addFocused     int
	exampleInput   textinput.Model
	regexInput     textinput.Model
	addErr         string
	previewBefore  string
	previewAfter   string
	previewErr     string
}

func NewRedactionModel(cfg *config.Config) RedactionModel {
	exampleInput := textinput.New()
	exampleInput.Prompt = "Example: "
	exampleInput.Placeholder = "Text with sensitive data..."
	exampleInput.Focus()

	regexInput := textinput.New()
	regexInput.Prompt = "Regex:  "
	regexInput.Placeholder = "e.g., \\bsecret-[a-z0-9]+\\b"

	m := RedactionModel{
		cfg:            cfg,
		styles:         defaultRedactionStyles(),
		customPatterns: make([]string, len(cfg.Redact.CustomPatterns)),
		mode:           redactionModeList,
		selected:       0,
		scrollY:        0,
		addFocused:     0,
		exampleInput:   exampleInput,
		regexInput:     regexInput,
	}
	copy(m.customPatterns, cfg.Redact.CustomPatterns)

	return m
}

func (m RedactionModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m RedactionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.mode == redactionModeAdd {
				m.mode = redactionModeList
				m.exampleInput.SetValue("")
				m.regexInput.SetValue("")
				m.addErr = ""
				m.previewBefore = ""
				m.previewAfter = ""
				m.previewErr = ""
				return m, nil
			}
			return m, func() tea.Msg { return RedactionCloseMsg{Saved: false} }

		case "ctrl+s":
			return m, func() tea.Msg { return RedactionCloseMsg{Saved: true} }

		case "up":
			if m.mode == redactionModeList {
				if m.selected > 0 {
					m.selected--
					m.ensureVisible()
				}
			}
			return m, nil

		case "down":
			if m.mode == redactionModeList {
				totalItems := len(builtinPatterns) + len(m.customPatterns) + 1
				if m.selected < totalItems-1 {
					m.selected++
					m.ensureVisible()
				}
			}
			return m, nil

		case " ":
			if m.mode == redactionModeList && m.selected == 0 {
				m.cfg.Redact.Enabled = !m.cfg.Redact.Enabled
			}
			return m, nil

		case "e":
			if m.mode == redactionModeList && m.selected == 0 {
				m.cfg.Redact.Enabled = !m.cfg.Redact.Enabled
			}
			return m, nil

		case "ctrl+n":
			if m.mode == redactionModeList {
				m.mode = redactionModeAdd
				m.addFocused = 0
				m.exampleInput.Focus()
			}
			return m, nil

		case "enter":
			if m.mode == redactionModeAdd {
				regex := strings.TrimSpace(m.regexInput.Value())
				if regex == "" {
					m.addErr = "regex cannot be empty"
					return m, nil
				}
				if _, err := regexp.Compile(regex); err != nil {
					m.addErr = fmt.Sprintf("invalid regex: %v", err)
					return m, nil
				}
				for _, p := range m.customPatterns {
					if p == regex {
						m.addErr = "pattern already exists"
						return m, nil
					}
				}
				m.customPatterns = append(m.customPatterns, regex)
				sort.Strings(m.customPatterns)
				m.cfg.Redact.CustomPatterns = m.customPatterns
				m.mode = redactionModeList
				m.exampleInput.SetValue("")
				m.regexInput.SetValue("")
				m.addErr = ""
				m.previewBefore = ""
				m.previewAfter = ""
				m.previewErr = ""
				m.selected = 1 + len(m.customPatterns) - 1
				m.ensureVisible()
			}
			return m, nil

		case "tab":
			if m.mode == redactionModeAdd {
				if m.addFocused == 0 {
					m.addFocused = 1
					m.exampleInput.Blur()
					m.regexInput.Focus()
				} else {
					m.addFocused = 0
					m.regexInput.Blur()
					m.exampleInput.Focus()
				}
			}
			return m, nil

		case "shift+tab":
			if m.mode == redactionModeAdd {
				if m.addFocused == 0 {
					m.addFocused = 1
					m.exampleInput.Blur()
					m.regexInput.Focus()
				} else {
					m.addFocused = 0
					m.regexInput.Blur()
					m.exampleInput.Focus()
				}
			}
			return m, nil

		case "d":
			if m.mode == redactionModeList {
				totalBuiltins := 1 + len(builtinPatterns)
				if m.selected >= totalBuiltins {
					idx := m.selected - totalBuiltins
					if idx < len(m.customPatterns) {
						m.customPatterns = append(m.customPatterns[:idx], m.customPatterns[idx+1:]...)
						m.cfg.Redact.CustomPatterns = m.customPatterns
						totalItems := len(builtinPatterns) + len(m.customPatterns) + 1
						if m.selected >= totalItems && m.selected > 0 {
							m.selected--
						}
						m.ensureVisible()
					}
				}
			}
			return m, nil
		}
	}

	if m.mode == redactionModeAdd {
		var cmd tea.Cmd
		if m.addFocused == 0 {
			m.exampleInput, cmd = m.exampleInput.Update(msg)
			m.previewBefore = m.exampleInput.Value()
		} else {
			m.regexInput, cmd = m.regexInput.Update(msg)
		}
		cmds = append(cmds, cmd)
		m.recomputePreview()
	}

	return m, tea.Batch(cmds...)
}

func (m *RedactionModel) recomputePreview() {
	regex := m.regexInput.Value()
	if regex == "" || m.previewBefore == "" {
		m.previewAfter = ""
		m.previewErr = ""
		return
	}

	re, err := regexp.Compile(regex)
	if err != nil {
		m.previewErr = fmt.Sprintf("invalid regex: %v", err)
		m.previewAfter = ""
		return
	}

	m.previewErr = ""
	matches := re.FindAllStringIndex(m.previewBefore, -1)
	if len(matches) == 0 {
		m.previewAfter = m.previewBefore
		return
	}

	m.previewAfter = renderReplaced(m.previewBefore, matches, m.styles.highlight)
}

func (m *RedactionModel) ensureVisible() {
	visibleItems := m.visibleItemCount()
	totalItems := len(builtinPatterns) + len(m.customPatterns) + 1
	if m.selected < m.scrollY {
		m.scrollY = m.selected
	}
	if m.selected >= m.scrollY+visibleItems {
		m.scrollY = m.selected - visibleItems + 1
	}
	maxScroll := totalItems - visibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollY > maxScroll {
		m.scrollY = maxScroll
	}
}

func (m RedactionModel) visibleItemCount() int {
	if m.height <= 0 {
		return 10
	}
	available := m.height - 12
	if available < 4 {
		return 4
	}
	return available
}

func (m RedactionModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.title.Render("Redaction Patterns"))
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render("Up/down: navigate | Space/E: toggle | Ctrl+N: add | D: delete | Ctrl+S: save | Esc: close"))
	b.WriteString("\n")

	if m.mode == redactionModeAdd {
		b.WriteString(m.styles.section.Render("--- Add Custom Pattern ---"))
		b.WriteString("\n")

		if m.addFocused == 0 {
			b.WriteString(m.styles.indicator.Render("> "))
			b.WriteString(m.exampleInput.View())
		} else {
			b.WriteString("  ")
			b.WriteString(m.exampleInput.View())
		}
		b.WriteString("\n")

		if m.addFocused == 1 {
			b.WriteString(m.styles.indicator.Render("> "))
			b.WriteString(m.regexInput.View())
		} else {
			b.WriteString("  ")
			b.WriteString(m.regexInput.View())
		}
		b.WriteString("\n")

		if m.addErr != "" {
			b.WriteString(m.styles.error.Render(m.addErr))
			b.WriteString("\n")
		}

		if m.previewBefore != "" && m.previewErr == "" {
			b.WriteString("\n")
			b.WriteString(m.styles.section.Render("--- Live Preview ---"))
			b.WriteString("\n")

			b.WriteString(m.styles.dimmed.Render("Before: "))
			b.WriteString(m.previewBefore)
			b.WriteString("\n")

			b.WriteString(m.styles.dimmed.Render("After:  "))
			b.WriteString(m.previewAfter)
			b.WriteString("\n")
		}

		return b.String()
	}

	b.WriteString("\n")
	enabledStr := "Disabled"
	enabledStyle := m.styles.disabled
	if m.cfg.Redact.Enabled {
		enabledStr = "Enabled"
		enabledStyle = m.styles.enabled
	}

	prefix := "  "
	if m.selected == 0 {
		prefix = m.styles.indicator.Render("> ")
	}
	fmt.Fprintf(&b, "%sRedaction: %s (Space/E to toggle)\n", prefix, enabledStyle.Render(enabledStr))

	totalItems := len(builtinPatterns) + len(m.customPatterns) + 1
	visibleStart := m.scrollY
	visibleEnd := m.scrollY + m.visibleItemCount()
	if visibleEnd > totalItems {
		visibleEnd = totalItems
	}

	b.WriteString("\n")
	b.WriteString(m.styles.section.Render("--- Builtin Patterns ---"))
	b.WriteString("\n")

	builtinStart := 1
	for i := visibleStart; i < visibleEnd && i < builtinStart+len(builtinPatterns); i++ {
		if i >= builtinStart {
			idx := i - builtinStart
			b.WriteString(m.renderBuiltinRow(i, builtinPatterns[idx]))
		}
	}

	if len(m.customPatterns) > 0 {
		b.WriteString("\n")
		b.WriteString(m.styles.section.Render("--- Custom Patterns ---"))
		b.WriteString("\n")

		customStart := builtinStart + len(builtinPatterns)
		for i := visibleStart; i < visibleEnd; i++ {
			if i >= customStart {
				idx := i - customStart
				if idx < len(m.customPatterns) {
					b.WriteString(m.renderCustomRow(i, m.customPatterns[idx]))
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

func (m RedactionModel) renderBuiltinRow(idx int, p builtinPatternEntry) string {
	prefix := "  "
	style := m.styles.pattern
	if idx == m.selected {
		prefix = m.styles.indicator.Render("> ")
		style = m.styles.pattern.Bold(true)
	}

	highlighted := renderHighlighted(p.example, m.styles.highlight)
	return fmt.Sprintf("%s%s  %s  %s\n%s    Example: %s\n",
		prefix,
		m.styles.category.Render("["+p.category+"]"),
		style.Render(p.name),
		m.styles.dimmed.Render(p.pattern),
		prefix,
		highlighted,
	)
}

func (m RedactionModel) renderCustomRow(idx int, pattern string) string {
	prefix := "  "
	style := m.styles.custom
	if idx == m.selected {
		prefix = m.styles.indicator.Render("> ")
		style = m.styles.custom.Bold(true)
	}
	delTag := m.styles.dimmed.Render(" [D to delete]")
	return fmt.Sprintf("%s%s  %s%s\n", prefix, style.Render("Custom"), m.styles.dimmed.Render(pattern), delTag)
}

func renderHighlighted(text string, style lipgloss.Style) string {
	patterns := make([]string, len(builtinPatterns))
	for i, p := range builtinPatterns {
		patterns[i] = p.pattern
	}
	re := regexp.MustCompile(strings.Join(patterns, "|"))
	locs := re.FindAllStringIndex(text, -1)
	if len(locs) == 0 {
		return text
	}
	return renderHighlightedLocs(text, locs, style)
}

func renderHighlightedLocs(text string, locs [][]int, style lipgloss.Style) string {
	var b strings.Builder
	lastEnd := 0
	for _, loc := range locs {
		b.WriteString(text[lastEnd:loc[0]])
		b.WriteString(style.Render(text[loc[0]:loc[1]]))
		lastEnd = loc[1]
	}
	b.WriteString(text[lastEnd:])
	return b.String()
}

func renderReplaced(text string, locs [][]int, style lipgloss.Style) string {
	var b strings.Builder
	lastEnd := 0
	redactNum := 1
	for _, loc := range locs {
		b.WriteString(text[lastEnd:loc[0]])
		b.WriteString(style.Render(fmt.Sprintf("[REDACTED_%d]", redactNum)))
		redactNum++
		lastEnd = loc[1]
	}
	b.WriteString(text[lastEnd:])
	return b.String()
}

func (m RedactionModel) GetConfig() *config.Config {
	return m.cfg
}
