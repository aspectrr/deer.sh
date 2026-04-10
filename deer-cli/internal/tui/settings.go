package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
)

// StaticSettingsField represents the fixed configuration fields
type StaticSettingsField int

const (
	// AI Agent
	FieldAIAgentProvider StaticSettingsField = iota
	FieldAIAgentAPIKey
	FieldAIAgentModel
	FieldAIAgentEndpoint
	FieldAIAgentSiteURL
	FieldAIAgentSiteName
	FieldAIAgentTotalContextTokens
	FieldAIAgentCompactModel
	FieldAIAgentCompactThreshold

	// SSH
	FieldSSHProxyJump
	FieldSSHKeyDir
	FieldSSHSourceKeyDir
	FieldSSHCertTTL
	FieldSSHMaxTTL
	FieldSSHDefaultUser

	// Ansible
	FieldAnsibleInventoryPath
	FieldAnsiblePlaybooksDir

	// Logging
	FieldLoggingLevel
	FieldLoggingFormat

	// Telemetry
	FieldTelemetryEnabled

	// Audit
	FieldAuditEnabled
	FieldAuditLogPath
	FieldAuditMaxSizeMB

	StaticFieldCount
)

// SettingsModel is the Bubble Tea model for the settings screen
type SettingsModel struct {
	inputs     []textinput.Model
	labels     []string
	sections   []string
	focused    int
	cfg        *config.Config
	configPath string
	width      int
	height     int
	styles     Styles
	saved      bool
	err        error
	scrollY    int
}

// NewSettingsModel creates a new settings model
func NewSettingsModel(cfg *config.Config, configPath string) SettingsModel {
	m := SettingsModel{
		cfg:        cfg,
		configPath: configPath,
		styles:     DefaultStyles(),
		inputs:     make([]textinput.Model, 0),
		labels:     make([]string, 0),
		sections:   make([]string, 0),
	}

	// Static fields
	staticLabels := []string{
		// AI Agent
		"Provider:", "API Key:", "Model:", "Endpoint:", "Site URL:", "Site Name:",
		"Total Context Tokens:", "Compact Model:", "Compact Threshold:",
		// SSH
		"Proxy Jump:", "Key Dir:", "Source Key Dir:", "Cert TTL:", "Max TTL:", "Default User:",
		// Ansible
		"Inventory Path:", "Playbooks Dir:",
		// Logging
		"Log Level:", "Log Format:",
		// Telemetry
		"Enable Anonymous Usage:",
		// Audit
		"Audit Enabled:", "Log Path:", "Max Size (MB):",
	}

	staticSections := []string{
		// AI Agent
		"AI Agent", "AI Agent", "AI Agent", "AI Agent", "AI Agent", "AI Agent",
		"AI Agent", "AI Agent", "AI Agent",
		// SSH
		"SSH", "SSH", "SSH", "SSH", "SSH", "SSH",
		// Ansible
		"Ansible", "Ansible",
		// Logging
		"Logging", "Logging",
		// Telemetry
		"Telemetry",
		// Audit
		"Audit", "Audit", "Audit",
	}

	for i := range StaticFieldCount {
		t := textinput.New()
		t.Prompt = ""
		t.CharLimit = 512

		val := m.getStaticConfigValue(StaticSettingsField(i))
		t.SetValue(val)

		m.inputs = append(m.inputs, t)
		m.labels = append(m.labels, staticLabels[i])
		m.sections = append(m.sections, staticSections[i])
	}

	if len(m.inputs) > 0 {
		m.inputs[0].Focus()
	}

	return m
}

func (m SettingsModel) getStaticConfigValue(field StaticSettingsField) string {
	switch field {
	case FieldAIAgentProvider:
		return m.cfg.AIAgent.Provider
	case FieldAIAgentAPIKey:
		return m.cfg.AIAgent.APIKey
	case FieldAIAgentModel:
		return m.cfg.AIAgent.Model
	case FieldAIAgentEndpoint:
		return m.cfg.AIAgent.Endpoint
	case FieldAIAgentSiteURL:
		return m.cfg.AIAgent.SiteURL
	case FieldAIAgentSiteName:
		return m.cfg.AIAgent.SiteName
	case FieldAIAgentTotalContextTokens:
		return strconv.Itoa(m.cfg.AIAgent.TotalContextTokens)
	case FieldAIAgentCompactModel:
		return m.cfg.AIAgent.CompactModel
	case FieldAIAgentCompactThreshold:
		return strconv.FormatFloat(m.cfg.AIAgent.CompactThreshold, 'f', 2, 64)

	case FieldSSHProxyJump:
		return m.cfg.SSH.ProxyJump
	case FieldSSHKeyDir:
		return m.cfg.SSH.KeyDir
	case FieldSSHSourceKeyDir:
		return m.cfg.SSH.SourceKeyDir
	case FieldSSHCertTTL:
		return m.cfg.SSH.CertTTL.String()
	case FieldSSHMaxTTL:
		return m.cfg.SSH.MaxTTL.String()
	case FieldSSHDefaultUser:
		return m.cfg.SSH.DefaultUser

	case FieldAnsibleInventoryPath:
		return m.cfg.Ansible.InventoryPath
	case FieldAnsiblePlaybooksDir:
		return m.cfg.Ansible.PlaybooksDir

	case FieldLoggingLevel:
		return m.cfg.Logging.Level
	case FieldLoggingFormat:
		return m.cfg.Logging.Format

	case FieldTelemetryEnabled:
		return strconv.FormatBool(m.cfg.Telemetry.EnableAnonymousUsage)

	case FieldAuditEnabled:
		return strconv.FormatBool(m.cfg.Audit.Enabled)
	case FieldAuditLogPath:
		return m.cfg.Audit.LogPath
	case FieldAuditMaxSizeMB:
		return strconv.Itoa(m.cfg.Audit.MaxSizeMB)
	}
	return ""
}

// Init implements tea.Model
func (m SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.inputs[m.focused].Value() != "" {
				m.inputs[m.focused].SetValue("")
				return m, nil
			}
			return m, func() tea.Msg { return SettingsCloseMsg{Saved: false} }

		case "esc":
			return m, func() tea.Msg { return SettingsCloseMsg{Saved: false} }

		case "tab", "down":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % len(m.inputs)
			m.inputs[m.focused].Focus()
			m.ensureFocusedVisible()
			return m, nil

		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.focused].Focus()
			m.ensureFocusedVisible()
			return m, nil

		case "pgdown", "ctrl+d":
			visibleItems := m.visibleItemCount()
			m.inputs[m.focused].Blur()
			m.focused = min(m.focused+visibleItems/2, len(m.inputs)-1)
			m.inputs[m.focused].Focus()
			m.ensureFocusedVisible()
			return m, nil

		case "pgup", "ctrl+u":
			visibleItems := m.visibleItemCount()
			m.inputs[m.focused].Blur()
			m.focused = max(m.focused-visibleItems/2, 0)
			m.inputs[m.focused].Focus()
			m.ensureFocusedVisible()
			return m, nil

		case "home":
			m.inputs[m.focused].Blur()
			m.focused = 0
			m.inputs[m.focused].Focus()
			m.scrollY = 0
			return m, nil

		case "end":
			m.inputs[m.focused].Blur()
			m.focused = len(m.inputs) - 1
			m.inputs[m.focused].Focus()
			m.ensureFocusedVisible()
			return m, nil

		case "ctrl+s":
			if err := m.saveConfig(); err != nil {
				m.err = err
				return m, nil
			}
			m.saved = true
			return m, func() tea.Msg { return SettingsCloseMsg{Saved: true} }

		case "enter":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % len(m.inputs)
			m.inputs[m.focused].Focus()
			m.ensureFocusedVisible()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	var cmd tea.Cmd
	if len(m.inputs) > 0 {
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m SettingsModel) visibleItemCount() int {
	if m.height <= 0 {
		return 10
	}
	available := m.height - 8
	if available < 4 {
		return 4
	}
	return available / 2
}

func (m *SettingsModel) ensureFocusedVisible() {
	visibleItems := m.visibleItemCount()
	if m.focused < m.scrollY {
		m.scrollY = m.focused
	}
	if m.focused >= m.scrollY+visibleItems {
		m.scrollY = m.focused - visibleItems + 1
	}
	if m.scrollY < 0 {
		m.scrollY = 0
	}
	maxScroll := len(m.inputs) - visibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollY > maxScroll {
		m.scrollY = maxScroll
	}
}

// View implements tea.Model
func (m SettingsModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#166534"))
	b.WriteString(titleStyle.Render("Settings"))
	b.WriteString("\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	b.WriteString(helpStyle.Render("Tab/arrows: navigate | Ctrl+S: save | Esc: cancel"))
	b.WriteString("\n")

	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#15803d"))

	visibleItems := m.visibleItemCount()
	visibleStart := m.scrollY
	visibleEnd := m.scrollY + visibleItems

	currentSection := ""
	renderedCount := 0

	for i := 0; i < len(m.inputs); i++ {
		if i < visibleStart {
			if m.sections[i] != currentSection {
				currentSection = m.sections[i]
			}
			continue
		}
		if i >= visibleEnd {
			break
		}

		if m.sections[i] != currentSection {
			currentSection = m.sections[i]
			if renderedCount > 0 {
				b.WriteString("\n")
			}
			b.WriteString(sectionStyle.Render("--- " + currentSection + " ---"))
			b.WriteString("\n")
		}

		b.WriteString(m.renderField(i))
		renderedCount++
	}

	totalFields := len(m.inputs)
	scrollPct := 0
	if totalFields > visibleItems {
		scrollPct = (m.scrollY * 100) / (totalFields - visibleItems)
	}

	b.WriteString("\n")
	scrollIndicator := fmt.Sprintf("Field %d/%d", m.focused+1, totalFields)
	if totalFields > visibleItems {
		barWidth := 20
		filledWidth := (scrollPct * barWidth) / 100
		if filledWidth < 1 && m.scrollY > 0 {
			filledWidth = 1
		}
		scrollBar := strings.Repeat("#", filledWidth) + strings.Repeat(".", barWidth-filledWidth)
		scrollIndicator += fmt.Sprintf(" [%s] %d%%", scrollBar, scrollPct)
	}
	b.WriteString(helpStyle.Render(scrollIndicator))

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	if m.saved {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
		b.WriteString("\n")
		b.WriteString(successStyle.Render("Settings saved!"))
	}

	return b.String()
}

func (m SettingsModel) renderField(idx int) string {
	labelStyle := lipgloss.NewStyle().Width(26).Foreground(lipgloss.Color("#9CA3AF"))
	inputStyle := lipgloss.NewStyle()

	focusIndicator := "  "
	if idx == m.focused {
		focusIndicator = "> "
		inputStyle = inputStyle.Foreground(lipgloss.Color("#166534"))
	}

	return fmt.Sprintf("%s%s %s\n",
		focusIndicator,
		labelStyle.Render(m.labels[idx]),
		inputStyle.Render(m.inputs[idx].View()),
	)
}

func (m *SettingsModel) saveConfig() error {
	getStatic := func(field StaticSettingsField) string {
		idx := int(field)
		if idx < len(m.inputs) {
			return m.inputs[idx].Value()
		}
		return ""
	}

	// AI Agent
	m.cfg.AIAgent.Provider = getStatic(FieldAIAgentProvider)
	m.cfg.AIAgent.APIKey = getStatic(FieldAIAgentAPIKey)
	m.cfg.AIAgent.Model = getStatic(FieldAIAgentModel)
	m.cfg.AIAgent.Endpoint = getStatic(FieldAIAgentEndpoint)
	m.cfg.AIAgent.SiteURL = getStatic(FieldAIAgentSiteURL)
	m.cfg.AIAgent.SiteName = getStatic(FieldAIAgentSiteName)
	if v, err := strconv.Atoi(getStatic(FieldAIAgentTotalContextTokens)); err == nil {
		m.cfg.AIAgent.TotalContextTokens = v
	}
	m.cfg.AIAgent.CompactModel = getStatic(FieldAIAgentCompactModel)
	if v, err := strconv.ParseFloat(getStatic(FieldAIAgentCompactThreshold), 64); err == nil {
		m.cfg.AIAgent.CompactThreshold = v
	}

	// SSH
	m.cfg.SSH.ProxyJump = getStatic(FieldSSHProxyJump)
	m.cfg.SSH.KeyDir = getStatic(FieldSSHKeyDir)
	m.cfg.SSH.SourceKeyDir = getStatic(FieldSSHSourceKeyDir)
	if v, err := parseDuration(getStatic(FieldSSHCertTTL)); v > 0 && err == nil {
		m.cfg.SSH.CertTTL = v
	}
	if v, err := parseDuration(getStatic(FieldSSHMaxTTL)); v > 0 && err == nil {
		m.cfg.SSH.MaxTTL = v
	}
	m.cfg.SSH.DefaultUser = getStatic(FieldSSHDefaultUser)

	// Ansible
	m.cfg.Ansible.InventoryPath = getStatic(FieldAnsibleInventoryPath)
	m.cfg.Ansible.PlaybooksDir = getStatic(FieldAnsiblePlaybooksDir)

	// Logging
	m.cfg.Logging.Level = getStatic(FieldLoggingLevel)
	m.cfg.Logging.Format = getStatic(FieldLoggingFormat)

	// Telemetry
	m.cfg.Telemetry.EnableAnonymousUsage = getStatic(FieldTelemetryEnabled) == "true"

	// Audit
	m.cfg.Audit.Enabled = getStatic(FieldAuditEnabled) == "true"
	m.cfg.Audit.LogPath = getStatic(FieldAuditLogPath)
	if v, err := strconv.Atoi(getStatic(FieldAuditMaxSizeMB)); err == nil {
		m.cfg.Audit.MaxSizeMB = v
	}

	// Ensure config directory exists
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	return m.cfg.Save(m.configPath)
}

// GetConfig returns the current config
func (m SettingsModel) GetConfig() *config.Config {
	return m.cfg
}

// EnsureConfigExists checks if config exists and creates it with defaults if not.
// Both paths call LoadWithEnvOverride so env vars always take precedence over
// the YAML file, whether the config already existed or was freshly created.
func EnsureConfigExists(configPath string) (*config.Config, error) {
	if _, err := os.Stat(configPath); err == nil {
		cfg, _, err := config.LoadWithEnvOverride(configPath)
		return cfg, err
	}

	cfg := config.DefaultConfig()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	if err := cfg.Save(configPath); err != nil {
		return nil, err
	}

	// Apply env overrides on fresh config too
	cfg, _, err := config.LoadWithEnvOverride(configPath)
	return cfg, err
}

// parseDuration wraps time.ParseDuration for settings use.
func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
