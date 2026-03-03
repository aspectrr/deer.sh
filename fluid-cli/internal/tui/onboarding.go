package tui

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
)

// OnboardingStep represents the current step in onboarding.
type OnboardingStep int

const (
	StepWelcome  OnboardingStep = iota // Branding + what fluid does
	StepAPIKey                         // OpenRouter API key input
	StepComplete                       // Done
)

// OnboardingModel is the Bubble Tea model for the simplified onboarding.
type OnboardingModel struct {
	step    OnboardingStep
	spinner spinner.Model

	textInput textinput.Model
	styles    Styles

	width, height int
	ready         bool

	cfg        *config.Config
	configPath string

	apiKeyValid bool
	errorMsg    string
	testing     bool
}

type apiKeyTestDoneMsg struct {
	valid bool
	err   error
}

// NewOnboardingModel creates a new onboarding model with the simplified 3-step flow.
func NewOnboardingModel(cfg *config.Config, configPath string) OnboardingModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	ti := textinput.New()
	ti.Placeholder = "sk-or-v1-..."
	ti.EchoMode = textinput.EchoPassword
	ti.Width = 60

	return OnboardingModel{
		step:       StepWelcome,
		spinner:    s,
		textInput:  ti,
		cfg:        cfg,
		configPath: configPath,
		styles:     DefaultStyles(),
	}
}

func (m OnboardingModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.EnterAltScreen)
}

func (m OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m.handleEnter()
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
		m.step = StepComplete
		return m, nil
	}

	// Forward key events to text input when on API key step
	if m.step == StepAPIKey && !m.testing {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m OnboardingModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepWelcome:
		m.step = StepAPIKey
		m.textInput.Focus()
		return m, textinput.Blink

	case StepAPIKey:
		key := strings.TrimSpace(m.textInput.Value())
		if key == "" {
			m.errorMsg = "API key is required."
			return m, nil
		}
		m.errorMsg = ""
		m.testing = true
		return m, testAPIKey(key)

	case StepComplete:
		return m, tea.Quit
	}
	return m, nil
}

func (m OnboardingModel) View() string {
	if !m.ready {
		return ""
	}

	var s strings.Builder

	switch m.step {
	case StepWelcome:
		s.WriteString(m.viewWelcome())
	case StepAPIKey:
		s.WriteString(m.viewAPIKey())
	case StepComplete:
		s.WriteString(m.viewComplete())
	}

	return s.String()
}

func (m OnboardingModel) viewWelcome() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69")).
		Render("fluid")

	return fmt.Sprintf(`

  %s - Make Infrastructure Safe for AI

  Fluid gives AI agents read-only access to your servers so they can
  diagnose issues and generate Ansible playbooks for fixes.

  What you need:
    - An OpenRouter API key (for the AI agent)
    - SSH access to your servers (from this machine)

  Press Enter to continue.

`, title)
}

func (m OnboardingModel) viewAPIKey() string {
	title := lipgloss.NewStyle().Bold(true).Render("OpenRouter API Key")

	view := fmt.Sprintf(`

  %s

  Get a key at https://openrouter.ai/keys

  %s

`, title, m.textInput.View())

	if m.testing {
		view += fmt.Sprintf("  %s Validating...\n", m.spinner.View())
	}
	if m.errorMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		view += fmt.Sprintf("  %s\n", errStyle.Render(m.errorMsg))
	}

	view += "\n  Press Enter to continue.\n"
	return view
}

func (m OnboardingModel) viewComplete() string {
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	check := checkStyle.Render("[ok]")

	return fmt.Sprintf(`

  %s API key configured.

  Next steps:
    1. Prepare a source host:   fluid source prepare <hostname>
    2. Start the agent:         fluid

  The agent can inspect your servers with read-only SSH access.
  To create sandboxes, set up a daemon host: https://fluid.sh/docs/daemon

  Press Enter to finish.

`, check)
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

// RunOnboarding runs the simplified onboarding wizard.
func RunOnboarding(cfg *config.Config, configPath string) (*config.Config, error) {
	model := NewOnboardingModel(cfg, configPath)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return cfg, err
	}
	return finalModel.(OnboardingModel).GetConfig(), nil
}
