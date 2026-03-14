package tui

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/fluid.sh/fluid-cli/internal/config"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/docsprogress"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/hostexec"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/paths"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/readonly"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/source"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/sourcekeys"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/sshconfig"
)

// probeFluidReadonly tests if the fluid-readonly user is reachable on a host.
func probeFluidReadonly(hostname, keyDir string) bool {
	keyPath := sourcekeys.GetPrivateKeyPath(keyDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run := hostexec.NewReadOnlySSHAlias(hostname, keyPath)
	_, _, code, err := run(ctx, "echo ok")
	return err == nil && code == 0
}

// Session code charset (no ambiguous chars: 0, O, I, 1)
const sessionCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateSessionCode() string {
	const chars = sessionCodeCharset
	b := make([]byte, 6)
	for i := range b {
		for {
			_, _ = rand.Read(b[i : i+1])
			idx := int(b[i])
			if idx < 256-(256%len(chars)) {
				b[i] = chars[idx%len(chars)]
				break
			}
		}
	}
	return string(b)
}

// OnboardingStep represents the current step in onboarding.
type OnboardingStep int

const (
	StepWelcome  OnboardingStep = iota // Branding + what fluid does
	StepAPIKey                         // OpenRouter API key input
	StepPrepare                        // Source host preparation
	StepComplete                       // Done
)

// OnboardingModel is the Bubble Tea model for the onboarding flow.
type OnboardingModel struct {
	step    OnboardingStep
	spinner spinner.Model

	textInput textinput.Model
	styles    Styles

	width, height int
	ready         bool

	cfg        *config.Config
	configPath string

	sessionCode string

	apiKeyValid bool
	errorMsg    string
	testing     bool

	// Prepare step state
	sshHosts       []string
	hostCursor     int
	preparingHosts map[string]bool
	preparedHosts  map[string]bool
	prepareErrors  map[string]string
	// Re-prepare confirmation state
	confirmHost         string
	confirmingReprepare bool
	probing             bool
}

type apiKeyTestDoneMsg struct {
	valid bool
	err   error
}

type onboardingPrepareDoneMsg struct {
	host string
	err  error
}

type onboardingProbeResultMsg struct {
	host            string
	alreadyPrepared bool
}

// NewOnboardingModel creates a new onboarding model.
func NewOnboardingModel(cfg *config.Config, configPath string) OnboardingModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	ti := textinput.New()
	ti.Placeholder = "sk-or-v1-..."
	ti.EchoMode = textinput.EchoPassword
	ti.Width = 60

	code := generateSessionCode()
	cfg.DocsSessionCode = code

	return OnboardingModel{
		step:           StepWelcome,
		spinner:        s,
		textInput:      ti,
		cfg:            cfg,
		configPath:     configPath,
		styles:         DefaultStyles(),
		sessionCode:    code,
		preparingHosts: make(map[string]bool),
		preparedHosts:  make(map[string]bool),
		prepareErrors:  make(map[string]string),
	}
}

func (m OnboardingModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.EnterAltScreen)
}

// registerSessionCmd registers the session code with the API (fire-and-forget).
func (m OnboardingModel) registerSessionCmd() tea.Cmd {
	return func() tea.Msg {
		docsprogress.RegisterSession(m.cfg.APIURL, m.sessionCode)
		return nil
	}
}

// openDocsCmd returns a tea.Cmd that opens the quickstart docs in a browser.
func (m OnboardingModel) openDocsCmd() tea.Cmd {
	return func() tea.Msg {
		u, err := url.JoinPath(m.cfg.WebURL, "/docs/quickstart")
		if err == nil {
			openBrowser(u + "?code=" + m.sessionCode)
		}
		return nil
	}
}

// openBrowser opens a URL in the default browser. Errors are silently ignored.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func (m OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		// Step-specific key handling
		switch m.step {
		case StepWelcome:
			if msg.String() == "enter" {
				m.step = StepAPIKey
				m.textInput.Focus()
				cmds := []tea.Cmd{textinput.Blink, m.openDocsCmd()}
				if m.cfg.APIURL != "" {
					cmds = append(cmds, m.registerSessionCmd())
				}
				return m, tea.Batch(cmds...)
			}

		case StepAPIKey:
			if msg.String() == "s" && !m.testing {
				m.step = StepPrepare
				m.sshHosts = sshconfig.ListHosts()
				return m, nil
			}
			if msg.String() == "enter" {
				return m.handleAPIKeyEnter()
			}
			if !m.testing {
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}

		case StepPrepare:
			return m.handlePrepareKey(msg)

		case StepComplete:
			if msg.String() == "enter" {
				m.cfg.DocsSessionCode = ""
				if configPath, err := paths.ConfigFile(); err == nil {
					_ = m.cfg.Save(configPath)
				}
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case apiKeyTestDoneMsg:
		m.testing = false
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("API key validation failed: %v", msg.err)
			return m, nil
		}
		if !msg.valid {
			m.errorMsg = "Invalid API key. Please check and try again."
			return m, nil
		}
		m.apiKeyValid = true
		m.cfg.AIAgent.APIKey = m.textInput.Value()
		m.step = StepPrepare
		m.sshHosts = sshconfig.ListHosts()
		return m, nil

	case onboardingProbeResultMsg:
		m.probing = false
		if msg.alreadyPrepared {
			m.confirmHost = msg.host
			m.confirmingReprepare = true
		} else {
			m.preparingHosts[msg.host] = true
			return m, m.prepareHostCmd(msg.host)
		}
		return m, nil

	case onboardingPrepareDoneMsg:
		m.preparingHosts[msg.host] = false
		if msg.err != nil {
			m.prepareErrors[msg.host] = msg.err.Error()
		} else {
			m.preparedHosts[msg.host] = true
		}
		return m, nil
	}

	return m, nil
}

func (m OnboardingModel) handleAPIKeyEnter() (tea.Model, tea.Cmd) {
	key := strings.TrimSpace(m.textInput.Value())
	if key == "" {
		m.errorMsg = "API key is required. Press s to skip."
		return m, nil
	}
	m.errorMsg = ""
	m.testing = true
	return m, testAPIKey(key)
}

func (m OnboardingModel) handlePrepareKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle re-prepare confirmation
	if m.confirmingReprepare {
		switch msg.String() {
		case "y":
			host := m.confirmHost
			m.confirmingReprepare = false
			m.confirmHost = ""
			m.preparingHosts[host] = true
			return m, m.prepareHostCmd(host)
		case "n":
			m.confirmingReprepare = false
			m.confirmHost = ""
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.hostCursor > 0 {
			m.hostCursor--
		}
	case "down", "j":
		if m.hostCursor < len(m.sshHosts)-1 {
			m.hostCursor++
		}
	case "enter":
		if len(m.sshHosts) > 0 && !m.probing {
			host := m.sshHosts[m.hostCursor]
			if !m.preparingHosts[host] && !m.preparedHosts[host] {
				m.probing = true
				return m, m.probeHostCmd(host)
			}
		}
	case "a":
		// Prepare all hosts concurrently (skip probe/confirm)
		var cmds []tea.Cmd
		for _, host := range m.sshHosts {
			if !m.preparingHosts[host] && !m.preparedHosts[host] {
				m.preparingHosts[host] = true
				cmds = append(cmds, m.prepareHostCmd(host))
			}
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
	case "s", "n":
		m.step = StepComplete
	}
	return m, nil
}

func (m OnboardingModel) hasAnyPrepared() bool {
	return len(m.preparedHosts) > 0
}

func (m OnboardingModel) probeHostCmd(hostname string) tea.Cmd {
	keyDir := m.cfg.SSH.SourceKeyDir
	return func() tea.Msg {
		reachable := probeFluidReadonly(hostname, keyDir)
		return onboardingProbeResultMsg{host: hostname, alreadyPrepared: reachable}
	}
}

func (m OnboardingModel) prepareHostCmd(hostname string) tea.Cmd {
	cfg := m.cfg
	return func() tea.Msg {
		// 1. Resolve SSH config
		resolved, err := sshconfig.Resolve(hostname)
		if err != nil {
			return onboardingPrepareDoneMsg{host: hostname, err: fmt.Errorf("SSH resolve: %v", err)}
		}

		// 2. Ensure SSH key pair
		_, pubKey, err := sourcekeys.EnsureKeyPair(cfg.SSH.SourceKeyDir)
		if err != nil {
			return onboardingPrepareDoneMsg{host: hostname, err: fmt.Errorf("key pair: %v", err)}
		}

		// 3. Prepare host
		sshRunFn := hostexec.NewSSHAlias(hostname)
		sshRun := readonly.SSHRunFunc(sshRunFn)
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		_, err = readonly.PrepareWithKey(ctx, sshRun, pubKey, nil, logger)
		if err != nil {
			return onboardingPrepareDoneMsg{host: hostname, err: err}
		}

		// 4. Update config
		configPath, _ := paths.ConfigFile()
		if err := source.SavePreparedHost(cfg, configPath, hostname, resolved); err != nil {
			return onboardingPrepareDoneMsg{host: hostname, err: err}
		}

		return onboardingPrepareDoneMsg{host: hostname}
	}
}

func (m OnboardingModel) View() string {
	if !m.ready {
		return ""
	}

	// Header with session code
	header := m.viewHeader()

	var content string
	switch m.step {
	case StepWelcome:
		content = m.viewWelcome()
	case StepAPIKey:
		content = m.viewAPIKey()
	case StepPrepare:
		content = m.viewPrepare()
	case StepComplete:
		content = m.viewComplete()
	}

	// Center content
	centered := lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, content)
	return header + "\n" + centered
}

func (m OnboardingModel) viewHeader() string {
	codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	code := codeStyle.Render("Session: " + m.sessionCode)

	// Right-align session code
	padding := m.width - lipgloss.Width(code)
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + code
}

func (m OnboardingModel) viewWelcome() string {
	var b strings.Builder

	// Render BannerLogo
	for _, line := range BannerLogo {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	brandStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	b.WriteString(brandStyle.Render("FLUID"))
	b.WriteString("\n")

	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	b.WriteString(subtitleStyle.Render("Make Infrastructure Safe for AI"))
	b.WriteString("\n\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	b.WriteString(infoStyle.Render("Fluid gives AI agents read-only access to your servers"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("so they can diagnose issues and generate Ansible playbooks."))
	b.WriteString("\n\n")

	b.WriteString(infoStyle.Render("What you need:"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  - An OpenRouter API key (for the AI agent)"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  - SSH access to your servers (from this machine)"))
	b.WriteString("\n\n")

	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	b.WriteString(hintStyle.Render("Press Enter to continue."))

	return b.String()
}

func (m OnboardingModel) viewAPIKey() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	b.WriteString(titleStyle.Render("OpenRouter API Key"))
	b.WriteString("\n\n")

	linkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	b.WriteString("Get a key at ")
	b.WriteString(linkStyle.Render("https://openrouter.ai/keys"))
	b.WriteString("\n\n")

	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	if m.testing {
		b.WriteString(m.spinner.View())
		b.WriteString(" Validating...\n")
	}

	if m.errorMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		b.WriteString(errStyle.Render(m.errorMsg))
		b.WriteString("\n")
	}

	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	b.WriteString(hintStyle.Render("Press Enter to continue, s to skip."))

	return b.String()
}

func (m OnboardingModel) viewPrepare() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	b.WriteString(titleStyle.Render("Prepare Source Hosts"))
	b.WriteString("\n")

	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	b.WriteString(subtitleStyle.Render("Select hosts to set up for read-only access."))
	b.WriteString("\n")

	linkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	b.WriteString(subtitleStyle.Render("Learn more: "))
	srcPrepURL, _ := url.JoinPath(m.cfg.WebURL, "/docs/source-prepare")
	b.WriteString(linkStyle.Render(srcPrepURL))
	b.WriteString("\n\n")

	if len(m.sshHosts) == 0 {
		b.WriteString(subtitleStyle.Render("No SSH hosts found in ~/.ssh/config."))
		b.WriteString("\n\n")
		b.WriteString(subtitleStyle.Render("Press s to skip."))
		return b.String()
	}

	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	for i, host := range m.sshHosts {
		cursor := "  "
		if i == m.hostCursor {
			cursor = cursorStyle.Render("> ")
		}

		var status string
		if m.preparedHosts[host] {
			status = checkStyle.Render(" [ok]")
		} else if m.preparingHosts[host] {
			status = " " + m.spinner.View() + " " + dimStyle.Render("preparing")
		} else if errMsg, ok := m.prepareErrors[host]; ok {
			status = errStyle.Render(" [error: " + errMsg + "]")
		}

		b.WriteString(cursor + host + status + "\n")
	}

	if m.probing {
		b.WriteString("\n")
		b.WriteString(m.spinner.View() + " " + dimStyle.Render("Checking host..."))
		b.WriteString("\n")
	} else if m.confirmingReprepare {
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		b.WriteString("\n")
		b.WriteString(warnStyle.Render(m.confirmHost + " is already prepared. Re-prepare? (y/n)"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.hasAnyPrepared() {
		b.WriteString(dimStyle.Render("Enter: prepare selected  |  a: prepare all  |  n: next"))
	} else {
		b.WriteString(dimStyle.Render("Enter: prepare selected  |  a: prepare all  |  s: skip"))
	}

	return b.String()
}

func (m OnboardingModel) viewComplete() string {
	var b strings.Builder

	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	b.WriteString(checkStyle.Render("[ok]"))
	b.WriteString(" Setup complete.\n\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	b.WriteString(infoStyle.Render("Next steps:"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  1. Prepare a source host:  fluid source prepare <hostname>"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  2. Start the agent:        fluid"))
	b.WriteString("\n\n")

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	b.WriteString(dimStyle.Render("The agent can inspect your servers with read-only SSH access."))
	b.WriteString("\n")
	daemonURL, _ := url.JoinPath(m.cfg.WebURL, "/docs/daemon")
	b.WriteString(dimStyle.Render("To create sandboxes, set up a daemon host: " + daemonURL))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Then use /connect in the TUI to connect to it."))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Press Enter to finish."))

	return b.String()
}

// GetConfig returns the current config.
func (m OnboardingModel) GetConfig() *config.Config {
	return m.cfg
}

// testAPIKey validates the API key by making a lightweight request to OpenRouter.
func testAPIKey(key string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
		if err != nil {
			return apiKeyTestDoneMsg{valid: false, err: err}
		}
		req.Header.Set("Authorization", "Bearer "+key)

		resp, err := client.Do(req)
		if err != nil {
			return apiKeyTestDoneMsg{valid: false, err: err}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return apiKeyTestDoneMsg{valid: false}
		}
		return apiKeyTestDoneMsg{valid: true}
	}
}

// RunOnboarding runs the onboarding wizard.
func RunOnboarding(cfg *config.Config, configPath string) (*config.Config, error) {
	model := NewOnboardingModel(cfg, configPath)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		cfg.DocsSessionCode = ""
		return cfg, err
	}
	result := finalModel.(OnboardingModel).GetConfig()
	result.DocsSessionCode = ""
	return result, nil
}
