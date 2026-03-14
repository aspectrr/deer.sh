package tui

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/fluid.sh/fluid-cli/internal/config"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/doctor"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/hostexec"
	"github.com/aspectrr/fluid.sh/fluid-cli/internal/sandbox"
)

// ConnectStep tracks the current wizard step.
type ConnectStep int

const (
	StepAddress ConnectStep = iota
	StepConnecting
	StepDoctor
	StepDone
)

// connectField indexes into the text inputs on the address step.
type connectField int

const (
	fieldAddress connectField = iota
	fieldName
	fieldInsecure
	connectFieldCount
)

// ConnectHealthResultMsg carries the result of the gRPC health + host info attempt.
type ConnectHealthResultMsg struct {
	Service  sandbox.Service
	HostInfo *sandbox.HostInfo
	Err      error
}

// ConnectDoctorResultMsg carries doctor check results.
type ConnectDoctorResultMsg struct {
	Results []doctor.CheckResult
	Err     error
}

// ConnectModel implements the 4-step connect wizard modal.
type ConnectModel struct {
	step    ConnectStep
	inputs  [connectFieldCount]textinput.Model
	focused connectField
	spinner spinner.Model
	width   int
	height  int
	styles  Styles

	// Connection result
	service  sandbox.Service
	hostInfo *sandbox.HostInfo
	connErr  error

	// Doctor results
	doctorResults []doctor.CheckResult
	doctorErr     error

	// Config for read-only hosts (passed to NewRemoteService)
	hosts []config.HostConfig
}

// NewConnectModel creates a new connect wizard.
func NewConnectModel(hosts []config.HostConfig) ConnectModel {
	addrInput := textinput.New()
	addrInput.Placeholder = "localhost:9091"
	addrInput.Prompt = ""
	addrInput.CharLimit = 256
	addrInput.Focus()

	nameInput := textinput.New()
	nameInput.Placeholder = "optional name"
	nameInput.Prompt = ""
	nameInput.CharLimit = 128

	insecureInput := textinput.New()
	insecureInput.Placeholder = "true"
	insecureInput.Prompt = ""
	insecureInput.CharLimit = 5
	insecureInput.SetValue("true")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))

	return ConnectModel{
		step:    StepAddress,
		inputs:  [connectFieldCount]textinput.Model{addrInput, nameInput, insecureInput},
		focused: fieldAddress,
		spinner: s,
		styles:  DefaultStyles(),
		hosts:   hosts,
	}
}

// Init implements tea.Model.
func (m ConnectModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// Update implements tea.Model.
func (m ConnectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ConnectHealthResultMsg:
		if msg.Err != nil {
			m.connErr = msg.Err
			m.step = StepConnecting // stay on connecting step to show error
			return m, nil
		}
		m.service = msg.Service
		m.hostInfo = msg.HostInfo
		m.connErr = nil
		m.step = StepDoctor
		return m, tea.Batch(m.spinner.Tick, m.runDoctorChecks())

	case ConnectDoctorResultMsg:
		m.doctorResults = msg.Results
		m.doctorErr = msg.Err
		m.step = StepDone
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward to focused input on address step
	if m.step == StepAddress {
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ConnectModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.step {
	case StepAddress:
		switch key {
		case "esc":
			return m, func() tea.Msg { return ConnectCloseMsg{} }
		case "tab", "down":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % connectFieldCount
			m.inputs[m.focused].Focus()
			return m, textinput.Blink
		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused - 1 + connectFieldCount) % connectFieldCount
			m.inputs[m.focused].Focus()
			return m, textinput.Blink
		case "enter":
			addr := m.resolveAddress()
			if addr == "" {
				return m, nil
			}
			m.step = StepConnecting
			m.connErr = nil
			return m, tea.Batch(m.spinner.Tick, m.attemptConnect(addr))
		default:
			var cmd tea.Cmd
			m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
			return m, cmd
		}

	case StepConnecting:
		switch key {
		case "esc":
			return m, func() tea.Msg { return ConnectCloseMsg{} }
		case "enter":
			if m.connErr != nil {
				// Retry
				addr := m.resolveAddress()
				m.connErr = nil
				return m, tea.Batch(m.spinner.Tick, m.attemptConnect(addr))
			}
		}

	case StepDoctor:
		if key == "esc" {
			return m, func() tea.Msg { return ConnectCloseMsg{} }
		}

	case StepDone:
		switch key {
		case "enter":
			return m, func() tea.Msg {
				return ConnectCloseMsg{
					Saved:  true,
					Config: m.buildConfig(),
				}
			}
		case "esc":
			return m, func() tea.Msg { return ConnectCloseMsg{} }
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m ConnectModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3B82F6"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  Connect to Fluid Daemon"))
	b.WriteString("\n\n")

	switch m.step {
	case StepAddress:
		labels := []string{"  Address:", "  Name:   ", "  Insecure:"}
		for i := range connectFieldCount {
			prefix := "  "
			if connectField(i) == m.focused {
				prefix = "> "
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, labels[i], m.inputs[i].View()))
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Enter: connect  Tab: next field  Esc: cancel"))

	case StepConnecting:
		addr := m.resolveAddress()
		if m.connErr != nil {
			b.WriteString(errStyle.Render(fmt.Sprintf("  Failed to connect to %s", addr)))
			b.WriteString("\n")
			b.WriteString(errStyle.Render(fmt.Sprintf("  %v", m.connErr)))
			b.WriteString("\n\n")
			b.WriteString(dimStyle.Render("  Enter: retry  Esc: cancel"))
		} else {
			b.WriteString(fmt.Sprintf("  %s Connecting to %s...", m.spinner.View(), addr))
		}

	case StepDoctor:
		b.WriteString(fmt.Sprintf("  %s Running doctor checks...", m.spinner.View()))

	case StepDone:
		m.renderHostInfo(&b, successStyle, dimStyle)
		b.WriteString("\n")
		m.renderDoctorResults(&b, successStyle, errStyle)
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Enter: save & close  Esc: close without saving"))
	}

	b.WriteString("\n")
	return b.String()
}

func (m ConnectModel) renderHostInfo(b *strings.Builder, successStyle, dimStyle lipgloss.Style) {
	b.WriteString(successStyle.Render("  Connected!"))
	b.WriteString("\n\n")
	if m.hostInfo != nil {
		info := m.hostInfo
		fmt.Fprintf(b, "  Hostname:    %s\n", info.Hostname)
		fmt.Fprintf(b, "  Version:     %s\n", info.Version)
		fmt.Fprintf(b, "  CPUs:        %d\n", info.TotalCPUs)
		fmt.Fprintf(b, "  Memory:      %d MB\n", info.TotalMemoryMB)
		fmt.Fprintf(b, "  Sandboxes:   %d active\n", info.ActiveSandboxes)
		fmt.Fprintf(b, "  Images:      %d available\n", len(info.BaseImages))
	}
}

func (m ConnectModel) renderDoctorResults(b *strings.Builder, successStyle, errStyle lipgloss.Style) {
	if m.doctorErr != nil {
		b.WriteString(fmt.Sprintf("  Doctor checks skipped: %v\n", m.doctorErr))
		return
	}
	if len(m.doctorResults) == 0 {
		b.WriteString("  Doctor checks skipped (no SSH access)\n")
		return
	}

	passed, failed := 0, 0
	for _, r := range m.doctorResults {
		if r.Passed {
			passed++
			b.WriteString(successStyle.Render(fmt.Sprintf("  v %s", r.Message)))
		} else {
			failed++
			b.WriteString(errStyle.Render(fmt.Sprintf("  x %s", r.Message)))
		}
		b.WriteString("\n")
		if !r.Passed && r.FixCmd != "" {
			b.WriteString(fmt.Sprintf("     Fix: %s\n", r.FixCmd))
		}
	}
	b.WriteString(fmt.Sprintf("\n  %d/%d passed", passed, passed+failed))
	if failed > 0 {
		b.WriteString(fmt.Sprintf(", %d failed", failed))
	}
	b.WriteString("\n")
}

// resolveAddress returns the address from input, defaulting to localhost:9091.
func (m ConnectModel) resolveAddress() string {
	addr := strings.TrimSpace(m.inputs[fieldAddress].Value())
	if addr == "" {
		addr = "localhost:9091"
	}
	// Append default gRPC port if not specified
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "9091")
	}
	return addr
}

// buildConfig creates a SandboxHostConfig from wizard inputs.
func (m ConnectModel) buildConfig() config.SandboxHostConfig {
	addr := m.resolveAddress()
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	if name == "" && m.hostInfo != nil && m.hostInfo.Hostname != "" {
		name = m.hostInfo.Hostname
	}
	if name == "" {
		name = "default"
	}
	insecure := strings.TrimSpace(m.inputs[fieldInsecure].Value())

	return config.SandboxHostConfig{
		Name:          name,
		DaemonAddress: addr,
		Insecure:      insecure == "" || insecure == "true" || insecure == "yes" || insecure == "1",
	}
}

// attemptConnect dials the daemon and checks health + host info.
func (m ConnectModel) attemptConnect(addr string) tea.Cmd {
	insecure := strings.TrimSpace(m.inputs[fieldInsecure].Value())
	isInsecure := insecure == "" || insecure == "true" || insecure == "yes" || insecure == "1"
	hosts := m.hosts

	return func() tea.Msg {
		cpCfg := config.ControlPlaneConfig{
			DaemonAddress:  addr,
			DaemonInsecure: isInsecure,
		}
		svc, err := sandbox.NewRemoteService(addr, cpCfg, hosts)
		if err != nil {
			return ConnectHealthResultMsg{Err: fmt.Errorf("dial: %w", err)}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := svc.Health(ctx); err != nil {
			_ = svc.Close()
			return ConnectHealthResultMsg{Err: fmt.Errorf("health check: %w", err)}
		}

		info, err := svc.GetHostInfo(ctx)
		if err != nil {
			_ = svc.Close()
			return ConnectHealthResultMsg{Err: fmt.Errorf("get host info: %w", err)}
		}

		return ConnectHealthResultMsg{
			Service:  svc,
			HostInfo: info,
		}
	}
}

// runDoctorChecks runs doctor checks over SSH to the host.
func (m ConnectModel) runDoctorChecks() tea.Cmd {
	addr := m.resolveAddress()
	return func() tea.Msg {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
		// Skip doctor checks for localhost - they require SSH
		if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "" {
			return ConnectDoctorResultMsg{}
		}

		run := hostexec.NewSSHAlias(host)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		results := doctor.RunAll(ctx, run)
		return ConnectDoctorResultMsg{Results: results}
	}
}

// GetService returns the connected sandbox service, or nil.
func (m ConnectModel) GetService() sandbox.Service {
	return m.service
}
