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

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
)

// StaticSettingsField represents the fixed configuration fields
type StaticSettingsField int

const (
	// Telemetry
	FieldTelemetryEnabled StaticSettingsField = iota

	// Libvirt
	FieldLibvirtURI
	FieldLibvirtNetwork
	FieldLibvirtBaseImageDir
	FieldLibvirtWorkDir
	FieldLibvirtSSHKeyInjectMethod
	FieldLibvirtSocketVMNetWrapper

	// VM
	FieldVMDefaultVCPUs
	FieldVMDefaultMemoryMB
	FieldVMCommandTimeout
	FieldVMIPDiscoveryTimeout

	// SSH
	FieldSSHProxyJump
	FieldSSHCAKeyPath
	FieldSSHCAPubPath
	FieldSSHKeyDir
	FieldSSHCertTTL
	FieldSSHMaxTTL
	FieldSSHWorkDir
	FieldSSHDefaultUser

	// Ansible
	FieldAnsibleInventoryPath
	FieldAnsiblePlaybooksDir

	// Logging
	FieldLoggingLevel
	FieldLoggingFormat

	// AI Agent
	FieldAIAgentProvider
	FieldAIAgentAPIKey
	FieldAIAgentModel
	FieldAIAgentEndpoint
	FieldAIAgentSiteURL
	FieldAIAgentSiteName
	FieldAIAgentTotalContextTokens
	FieldAIAgentCompactModel
	FieldAIAgentCompactThreshold

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

	// Helper to track how many hosts we currently have inputs for
	hostCount int
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

	// 1. Initialize Host Inputs
	m.hostCount = len(cfg.Hosts)

	for i, h := range cfg.Hosts {
		m.addHostInput(i+1, h.Name, h.Address, h.SSHVMUser)
	}

	// 2. Initialize Static Inputs
	// We'll use a temporary map or slice to order them correctly matching StaticSettingsField order
	staticLabels := []string{
		// Telemetry
		"Enable Anonymous Usage:",
		// Libvirt
		"Libvirt URI:", "Network:", "Base Image Dir:", "Work Dir:", "SSH Key Inject Method:", "Socket VMNet Wrapper:",
		// VM
		"Default vCPUs:", "Default Memory (MB):", "Command Timeout:", "IP Discovery Timeout:",
		// SSH
		"Proxy Jump:", "CA Key Path:", "CA Pub Path:", "Key Dir:", "Cert TTL:", "Max TTL:", "Work Dir:", "Default User:",
		// Ansible
		"Inventory Path:", "Playbooks Dir:",
		// Logging
		"Log Level:", "Log Format:",
		// AI Agent
		"Provider:", "API Key:", "Model:", "Endpoint:", "Site URL:", "Site Name:", "Total Context Tokens:", "Compact Model:", "Compact Threshold:",
	}

	staticSections := []string{
		// Telemetry
		"Telemetry",
		// Libvirt
		"Libvirt", "Libvirt", "Libvirt", "Libvirt", "Libvirt", "Libvirt",
		// VM
		"VM Defaults", "VM Defaults", "VM Defaults", "VM Defaults",
		// SSH
		"SSH", "SSH", "SSH", "SSH", "SSH", "SSH", "SSH", "SSH",
		// Ansible
		"Ansible", "Ansible",
		// Logging
		"Logging", "Logging",
		// AI Agent
		"AI Agent", "AI Agent", "AI Agent", "AI Agent", "AI Agent", "AI Agent", "AI Agent", "AI Agent", "AI Agent",
	}

	// Create inputs for static fields
	for i := range StaticFieldCount {
		t := textinput.New()
		t.Prompt = ""
		t.CharLimit = 512

		// Set value based on config
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

func (m *SettingsModel) addHostInput(num int, name, addr, vmUser string) {
	// Create three inputs: Name, Address, VM User
	tName := textinput.New()
	tName.Prompt = ""
	tName.CharLimit = 512
	tName.SetValue(name)

	tAddr := textinput.New()
	tAddr.Prompt = ""
	tAddr.CharLimit = 512
	tAddr.SetValue(addr)

	tVMUser := textinput.New()
	tVMUser.Prompt = ""
	tVMUser.CharLimit = 512
	tVMUser.SetValue(vmUser)

	// Append to lists
	m.inputs = append(m.inputs, tName, tAddr, tVMUser)
	m.labels = append(m.labels, fmt.Sprintf("Host %d Name:", num), fmt.Sprintf("Host %d Address:", num), fmt.Sprintf("Host %d VM User:", num))
	m.sections = append(m.sections, "Hosts", "Hosts", "Hosts")
}

// Helper to get static config value by enum
func (m SettingsModel) getStaticConfigValue(field StaticSettingsField) string {
	switch field {
	case FieldTelemetryEnabled:
		return strconv.FormatBool(m.cfg.Telemetry.EnableAnonymousUsage)

	case FieldLibvirtURI:
		return m.cfg.Libvirt.URI
	case FieldLibvirtNetwork:
		return m.cfg.Libvirt.Network
	case FieldLibvirtBaseImageDir:
		return m.cfg.Libvirt.BaseImageDir
	case FieldLibvirtWorkDir:
		return m.cfg.Libvirt.WorkDir
	case FieldLibvirtSSHKeyInjectMethod:
		return m.cfg.Libvirt.SSHKeyInjectMethod
	case FieldLibvirtSocketVMNetWrapper:
		return m.cfg.Libvirt.SocketVMNetWrapper

	case FieldVMDefaultVCPUs:
		return strconv.Itoa(m.cfg.VM.DefaultVCPUs)
	case FieldVMDefaultMemoryMB:
		return strconv.Itoa(m.cfg.VM.DefaultMemoryMB)
	case FieldVMCommandTimeout:
		return m.cfg.VM.CommandTimeout.String()
	case FieldVMIPDiscoveryTimeout:
		return m.cfg.VM.IPDiscoveryTimeout.String()

	case FieldSSHProxyJump:
		return m.cfg.SSH.ProxyJump
	case FieldSSHCAKeyPath:
		return m.cfg.SSH.CAKeyPath
	case FieldSSHCAPubPath:
		return m.cfg.SSH.CAPubPath
	case FieldSSHKeyDir:
		return m.cfg.SSH.KeyDir
	case FieldSSHCertTTL:
		return m.cfg.SSH.CertTTL.String()
	case FieldSSHMaxTTL:
		return m.cfg.SSH.MaxTTL.String()
	case FieldSSHWorkDir:
		return m.cfg.SSH.WorkDir
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
	}
	return ""
}

// Add a new host to the lists
func (m *SettingsModel) addNewHost() {
	m.hostCount++
	num := m.hostCount

	tName := textinput.New()
	tName.Prompt = ""
	tName.CharLimit = 512

	tAddr := textinput.New()
	tAddr.Prompt = ""
	tAddr.CharLimit = 512

	tVMUser := textinput.New()
	tVMUser.Prompt = ""
	tVMUser.CharLimit = 512

	// Insert before static fields
	// Current host inputs end at (num-1)*3
	insertIdx := (num - 1) * 3

	// Helper to insert into slice
	insertInput := func(slice []textinput.Model, idx int, items ...textinput.Model) []textinput.Model {
		return append(slice[:idx], append(items, slice[idx:]...)...)
	}
	insertString := func(slice []string, idx int, items ...string) []string {
		return append(slice[:idx], append(items, slice[idx:]...)...)
	}

	m.inputs = insertInput(m.inputs, insertIdx, tName, tAddr, tVMUser)
	m.labels = insertString(m.labels, insertIdx, fmt.Sprintf("Host %d Name:", num), fmt.Sprintf("Host %d Address:", num), fmt.Sprintf("Host %d VM User:", num))
	m.sections = insertString(m.sections, insertIdx, "Hosts", "Hosts", "Hosts")

	// If focus was after insertion point, shift it
	if m.focused >= insertIdx {
		m.focused += 3
	}
	// Focus the new name field
	m.inputs[m.focused].Blur()
	m.focused = insertIdx
	m.inputs[m.focused].Focus()
	m.ensureFocusedVisible()
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

		case "ctrl+n":
			m.addNewHost()
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

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3B82F6"))
	b.WriteString(titleStyle.Render("Settings"))
	b.WriteString("\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	b.WriteString(helpStyle.Render("Tab/↑↓: navigate | Ctrl+N: add host | Ctrl+S: save | Esc: cancel"))
	b.WriteString("\n")

	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4"))

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
			b.WriteString(sectionStyle.Render("─── " + currentSection + " ───"))
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
		scrollBar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)
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
		focusIndicator = "▶ "
		inputStyle = inputStyle.Foreground(lipgloss.Color("#3B82F6"))
	}

	return fmt.Sprintf("%s%s %s\n",
		focusIndicator,
		labelStyle.Render(m.labels[idx]),
		inputStyle.Render(m.inputs[idx].View()),
	)
}

func (m *SettingsModel) saveConfig() error {
	// 1. Save Hosts
	m.cfg.Hosts = make([]config.HostConfig, 0, m.hostCount)
	hostInputCount := m.hostCount * 3

	for i := 0; i < hostInputCount; i += 3 {
		name := m.inputs[i].Value()
		addr := m.inputs[i+1].Value()
		vmUser := m.inputs[i+2].Value()
		if name != "" || addr != "" {
			m.cfg.Hosts = append(m.cfg.Hosts, config.HostConfig{
				Name:      name,
				Address:   addr,
				SSHVMUser: vmUser,
			})
		}
	}

	// 2. Save Static Fields
	// Helper to access static inputs relative to host inputs
	getStatic := func(field StaticSettingsField) string {
		idx := hostInputCount + int(field)
		if idx < len(m.inputs) {
			return m.inputs[idx].Value()
		}
		return ""
	}

	m.cfg.Telemetry.EnableAnonymousUsage = getStatic(FieldTelemetryEnabled) == "true"

	m.cfg.Libvirt.URI = getStatic(FieldLibvirtURI)
	m.cfg.Libvirt.Network = getStatic(FieldLibvirtNetwork)
	m.cfg.Libvirt.BaseImageDir = getStatic(FieldLibvirtBaseImageDir)
	m.cfg.Libvirt.WorkDir = getStatic(FieldLibvirtWorkDir)
	m.cfg.Libvirt.SSHKeyInjectMethod = getStatic(FieldLibvirtSSHKeyInjectMethod)
	m.cfg.Libvirt.SocketVMNetWrapper = getStatic(FieldLibvirtSocketVMNetWrapper)

	if v, err := strconv.Atoi(getStatic(FieldVMDefaultVCPUs)); err == nil {
		m.cfg.VM.DefaultVCPUs = v
	}
	if v, err := strconv.Atoi(getStatic(FieldVMDefaultMemoryMB)); err == nil {
		m.cfg.VM.DefaultMemoryMB = v
	}
	if v, err := time.ParseDuration(getStatic(FieldVMCommandTimeout)); err == nil {
		m.cfg.VM.CommandTimeout = v
	}
	if v, err := time.ParseDuration(getStatic(FieldVMIPDiscoveryTimeout)); err == nil {
		m.cfg.VM.IPDiscoveryTimeout = v
	}

	m.cfg.SSH.ProxyJump = getStatic(FieldSSHProxyJump)
	m.cfg.SSH.CAKeyPath = getStatic(FieldSSHCAKeyPath)
	m.cfg.SSH.CAPubPath = getStatic(FieldSSHCAPubPath)
	m.cfg.SSH.KeyDir = getStatic(FieldSSHKeyDir)
	if v, err := time.ParseDuration(getStatic(FieldSSHCertTTL)); err == nil {
		m.cfg.SSH.CertTTL = v
	}
	if v, err := time.ParseDuration(getStatic(FieldSSHMaxTTL)); err == nil {
		m.cfg.SSH.MaxTTL = v
	}
	m.cfg.SSH.WorkDir = getStatic(FieldSSHWorkDir)
	m.cfg.SSH.DefaultUser = getStatic(FieldSSHDefaultUser)

	m.cfg.Ansible.InventoryPath = getStatic(FieldAnsibleInventoryPath)
	m.cfg.Ansible.PlaybooksDir = getStatic(FieldAnsiblePlaybooksDir)

	m.cfg.Logging.Level = getStatic(FieldLoggingLevel)
	m.cfg.Logging.Format = getStatic(FieldLoggingFormat)

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
// Returns the config and any permission warnings.
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

	return cfg, nil
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
