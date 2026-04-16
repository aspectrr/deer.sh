package tui

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
	"github.com/aspectrr/deer.sh/deer-cli/internal/doctor"
	"github.com/aspectrr/deer.sh/deer-cli/internal/hostexec"
	"github.com/aspectrr/deer.sh/deer-cli/internal/readonly"
	"github.com/aspectrr/deer.sh/deer-cli/internal/sandbox"
)

// ConnectStep tracks the current wizard step.
type ConnectStep int

const (
	StepAddress ConnectStep = iota
	StepConnecting
	StepDoctor
	StepDeployKeys
	StepDone
)

// connectField indexes into the address-step fields. Only fieldAddress and fieldName
// have backing text inputs; fieldInsecure is a boolean toggle.
type connectField int

const (
	fieldAddress connectField = iota
	fieldName
	fieldInsecure // virtual: boolean checkbox, no text input
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

// ConnectModel implements the connect wizard modal.
type ConnectModel struct {
	step     ConnectStep
	inputs   [fieldInsecure]textinput.Model // address, name have text inputs
	focused  connectField
	insecure bool
	spinner  spinner.Model
	width    int
	height   int
	styles   Styles
	logger   *slog.Logger

	// Connection result
	service  sandbox.Service
	hostInfo *sandbox.HostInfo
	connErr  error
	addrErr  string // address validation error message

	// Doctor results
	doctorResults []doctor.CheckResult
	doctorErr     error

	// Deploy results
	keysDeployed       bool
	deployResults      *DaemonKeyDeployResultMsg
	hostDeployStatuses []HostDeployStatus
	deployHostIndex    int
	deployIdentityKey  string
}

// NewConnectModel creates a new connect wizard.
func NewConnectModel(logger *slog.Logger) ConnectModel {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	addrInput := textinput.New()
	addrInput.Placeholder = "localhost:9091"
	addrInput.Prompt = ""
	addrInput.CharLimit = 256
	addrInput.Width = 40
	addrInput.Focus()

	nameInput := textinput.New()
	nameInput.Placeholder = "optional name"
	nameInput.Prompt = ""
	nameInput.CharLimit = 128
	nameInput.Width = 40

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#166534"))

	return ConnectModel{
		step:    StepAddress,
		inputs:  [fieldInsecure]textinput.Model{addrInput, nameInput},
		focused: fieldAddress,
		spinner: s,
		styles:  DefaultStyles(),
		logger:  logger,
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
			m.logger.Warn("connect failed", "error", msg.Err)
			return m, nil
		}
		m.service = msg.Service
		m.hostInfo = msg.HostInfo
		m.connErr = nil
		m.step = StepDoctor
		m.logger.Info("connected", "hostname", msg.HostInfo.Hostname, "version", msg.HostInfo.Version)
		return m, tea.Batch(m.spinner.Tick, m.runDoctorChecks())

	case ConnectDoctorResultMsg:
		m.doctorResults = msg.Results
		m.doctorErr = msg.Err
		m.step = StepDone
		if msg.Err != nil {
			m.logger.Warn("doctor checks failed", "error", msg.Err)
		} else {
			passed := 0
			for _, r := range msg.Results {
				if r.Passed {
					passed++
				}
			}
			m.logger.Info("doctor checks complete", "passed", passed, "total", len(msg.Results))
		}
		return m, nil

	case HostKeyDeployedMsg:
		if msg.Index < len(m.hostDeployStatuses) {
			if msg.Err != nil {
				m.hostDeployStatuses[msg.Index].State = HostDeployFailed
				m.hostDeployStatuses[msg.Index].ErrMsg = msg.Err.Error()
				m.logger.Warn("host key deploy failed", "host", msg.Host, "error", msg.Err)
			} else {
				m.hostDeployStatuses[msg.Index].State = HostDeployDone
				m.logger.Info("host key deployed", "host", msg.Host)
			}
		}
		// Deploy next host or finish
		nextIdx := msg.Index + 1
		if nextIdx < len(m.hostDeployStatuses) {
			m.hostDeployStatuses[nextIdx].State = HostDeployDeploying
			m.deployHostIndex = nextIdx
			return m, deploySourceHostKey(m.hostInfo.SourceHosts[nextIdx], m.deployIdentityKey, nextIdx, m.logger)
		}
		// All done - scan host keys on daemon side, then rerun doctor checks
		m.keysDeployed = true
		m.deployResults = m.buildDeployResults()
		m.step = StepDoctor
		m.doctorResults = nil
		m.doctorErr = nil
		m.logger.Info("deploy complete, scanning host keys", "deployed", m.deployResults.Deployed, "errors", len(m.deployResults.Errors))
		return m, tea.Batch(m.spinner.Tick, m.scanSourceHostKeys())

	case ScanKeysCompleteMsg:
		m.step = StepDoctor
		m.doctorResults = nil
		m.doctorErr = nil
		m.logger.Info("host key scan complete, re-running doctor checks")
		return m, tea.Batch(m.spinner.Tick, m.runDoctorChecks())

	case DaemonKeyDeployResultMsg:
		m.keysDeployed = true
		m.deployResults = &msg
		m.step = StepDone
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward to focused text input on address step (skip the insecure toggle)
	if m.step == StepAddress && m.focused < fieldInsecure {
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
			if m.focused < fieldInsecure {
				m.inputs[m.focused].Blur()
			}
			m.focused = (m.focused + 1) % connectFieldCount
			if m.focused < fieldInsecure {
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}
			return m, nil
		case "shift+tab", "up":
			if m.focused < fieldInsecure {
				m.inputs[m.focused].Blur()
			}
			m.focused = (m.focused - 1 + connectFieldCount) % connectFieldCount
			if m.focused < fieldInsecure {
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}
			return m, nil
		case "enter":
			addr, err := m.resolveAddress()
			if err != nil {
				m.addrErr = err.Error()
				return m, nil
			}
			m.addrErr = ""
			m.step = StepConnecting
			m.connErr = nil
			return m, tea.Batch(m.spinner.Tick, m.attemptConnect(addr))
		case " ", "y", "n":
			// Toggle insecure when focused on the insecure checkbox
			if m.focused == fieldInsecure {
				if key == "n" {
					m.insecure = false
				} else {
					m.insecure = !m.insecure
				}
				return m, nil
			}
			// Otherwise let the text input handle it
			var cmd tea.Cmd
			if m.focused < fieldInsecure {
				m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
			}
			return m, cmd
		default:
			if m.focused < fieldInsecure {
				var cmd tea.Cmd
				m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
				return m, cmd
			}
			return m, nil
		}

	case StepConnecting:
		switch key {
		case "esc":
			return m, func() tea.Msg { return ConnectCloseMsg{} }
		case "enter":
			if m.connErr != nil {
				// Retry
				addr, err := m.resolveAddress()
				if err != nil {
					m.addrErr = err.Error()
					m.step = StepAddress
					return m, nil
				}
				m.connErr = nil
				m.addrErr = ""
				return m, tea.Batch(m.spinner.Tick, m.attemptConnect(addr))
			}
		}

	case StepDoctor:
		if key == "esc" {
			return m, func() tea.Msg { return ConnectCloseMsg{} }
		}

	case StepDeployKeys:
		if key == "esc" {
			return m, func() tea.Msg { return ConnectCloseMsg{} }
		}

	case StepDone:
		switch key {
		case "enter":
			// If daemon has source hosts, identity key available, and not yet deployed: deploy keys
			if !m.keysDeployed && m.hostInfo != nil && m.hostInfo.SSHIdentityPubKey != "" && len(m.hostInfo.SourceHosts) > 0 {
				m.step = StepDeployKeys
				m.deployIdentityKey = m.hostInfo.SSHIdentityPubKey
				m.hostDeployStatuses = make([]HostDeployStatus, len(m.hostInfo.SourceHosts))
				for i, sh := range m.hostInfo.SourceHosts {
					m.hostDeployStatuses[i] = HostDeployStatus{Name: sh.Address, State: HostDeployPending}
				}
				m.hostDeployStatuses[0].State = HostDeployDeploying
				m.deployHostIndex = 0
				m.logger.Info("starting key deploy", "host_count", len(m.hostInfo.SourceHosts))
				return m, tea.Batch(m.spinner.Tick, deploySourceHostKey(m.hostInfo.SourceHosts[0], m.deployIdentityKey, 0, m.logger))
			}
			// Otherwise save & close
			return m, func() tea.Msg {
				return ConnectCloseMsg{
					Saved:  true,
					Config: m.buildConfig(),
				}
			}
		case "r":
			m.step = StepDoctor
			m.doctorResults = nil
			m.doctorErr = nil
			m.deployResults = nil
			m.keysDeployed = false
			return m, tea.Batch(m.spinner.Tick, m.runDoctorChecks())
		case "esc":
			// Save & close (successful connection)
			return m, func() tea.Msg {
				return ConnectCloseMsg{
					Saved:  true,
					Config: m.buildConfig(),
				}
			}
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m ConnectModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#166534"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  Connect to Deer Daemon"))
	b.WriteString("\n\n")

	switch m.step {
	case StepAddress:
		// Address and Name text input fields
		labels := []string{"  Address:  ", "  Name:     "}
		for i := range fieldInsecure {
			prefix := "  "
			if connectField(i) == m.focused {
				prefix = "> "
			}
			fmt.Fprintf(&b, "%s%s %s\n", prefix, labels[i], m.inputs[i].View())
		}
		// Insecure boolean toggle
		insecurePrefix := "  "
		if m.focused == fieldInsecure {
			insecurePrefix = "> "
		}
		checkbox := "[ ]"
		if m.insecure {
			checkbox = "[x]"
		}
		fmt.Fprintf(&b, "%s  Insecure: %s  (space/y/n to toggle)\n", insecurePrefix, checkbox)
		if m.addrErr != "" {
			b.WriteString(errStyle.Render(fmt.Sprintf("  Error: %s\n", m.addrErr)))
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Enter: connect  Tab: next field  Esc: cancel"))

	case StepConnecting:
		addr, _ := m.resolveAddress()
		if m.connErr != nil {
			b.WriteString(errStyle.Render(fmt.Sprintf("  Failed to connect to %s", addr)))
			b.WriteString("\n")
			b.WriteString(errStyle.Render(fmt.Sprintf("  %v", m.connErr)))
			b.WriteString("\n\n")
			b.WriteString(dimStyle.Render("  Enter: retry  Esc: cancel"))
		} else {
			fmt.Fprintf(&b, "  %s Connecting to %s...", m.spinner.View(), addr)
		}

	case StepDoctor:
		fmt.Fprintf(&b, "  %s Running doctor checks...", m.spinner.View())

	case StepDeployKeys:
		b.WriteString("  Setting up source hosts (user + key)...\n\n")
		for _, hs := range m.hostDeployStatuses {
			switch hs.State {
			case HostDeployDone:
				b.WriteString(successStyle.Render(fmt.Sprintf("  v %s", hs.Name)))
			case HostDeployFailed:
				b.WriteString(errStyle.Render(fmt.Sprintf("  x %s: %s", hs.Name, hs.ErrMsg)))
			case HostDeployDeploying:
				fmt.Fprintf(&b, "  %s %s", m.spinner.View(), hs.Name)
			default:
				b.WriteString(dimStyle.Render(fmt.Sprintf("  - %s", hs.Name)))
			}
			b.WriteString("\n")
		}

	case StepDone:
		m.renderHostInfo(&b, successStyle, dimStyle)
		b.WriteString("\n")
		m.renderDoctorResults(&b, successStyle, errStyle)
		m.renderDeployResults(&b, successStyle, errStyle)
		b.WriteString("\n")
		if !m.keysDeployed && m.hostInfo != nil && m.hostInfo.SSHIdentityPubKey != "" && len(m.hostInfo.SourceHosts) > 0 {
			b.WriteString(dimStyle.Render("  Enter: setup source hosts (create user + deploy key)  r: retry checks  Esc: save & close"))
		} else {
			b.WriteString(dimStyle.Render("  Enter: save & close  r: retry checks  Esc: save & close"))
		}
	}

	b.WriteString("\n")

	content := b.String()
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
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
		fmt.Fprintf(b, "  Doctor checks skipped: %v\n", m.doctorErr)
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
			fmt.Fprintf(b, "     Fix: %s\n", r.FixCmd)
		}
	}
	fmt.Fprintf(b, "\n  %d/%d passed", passed, passed+failed)
	if failed > 0 {
		fmt.Fprintf(b, ", %d failed", failed)
	}
	b.WriteString("\n")
}

func (m ConnectModel) renderDeployResults(b *strings.Builder, successStyle, errStyle lipgloss.Style) {
	if m.deployResults == nil {
		return
	}
	b.WriteString("\n")
	if m.deployResults.Deployed > 0 {
		b.WriteString(successStyle.Render(fmt.Sprintf("  v Setup complete on %d source host(s)", m.deployResults.Deployed)))
		b.WriteString("\n")
	}
	for _, e := range m.deployResults.Errors {
		b.WriteString(errStyle.Render(fmt.Sprintf("  x Source host setup: %s", e)))
		b.WriteString("\n")
	}
}

// resolveAddress returns the address from input, defaulting to localhost:9091.
func (m ConnectModel) resolveAddress() (string, error) {
	addr := strings.TrimSpace(m.inputs[fieldAddress].Value())
	if addr == "" {
		addr = "localhost:9091"
	}

	// Append default gRPC port if not specified
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// Not in host:port format, try appending default port
		addr = net.JoinHostPort(addr, "9091")
	} else if host == "" {
		return "", fmt.Errorf("invalid address: empty host")
	} else if port == "" {
		addr = net.JoinHostPort(host, "9091")
	}

	return addr, nil
}

// buildConfig creates a SandboxHostConfig from wizard inputs.
func (m ConnectModel) buildConfig() config.SandboxHostConfig {
	addr, _ := m.resolveAddress()
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	if name == "" && m.hostInfo != nil && m.hostInfo.Hostname != "" {
		name = m.hostInfo.Hostname
	}
	if name == "" {
		name = "default"
	}

	cfg := config.SandboxHostConfig{
		Name:          name,
		DaemonAddress: addr,
		Insecure:      m.insecure,
	}
	if m.hostInfo != nil {
		cfg.DaemonIdentityPubKey = m.hostInfo.SSHIdentityPubKey
	}
	return cfg
}

// attemptConnect dials the daemon and checks health + host info.
func (m ConnectModel) attemptConnect(addr string) tea.Cmd {
	insecure := m.insecure
	m.logger.Info("attempting connect", "address", addr, "insecure", insecure)

	return func() tea.Msg {
		cpCfg := config.ControlPlaneConfig{
			DaemonAddress:  addr,
			DaemonInsecure: insecure,
		}
		svc, err := sandbox.NewRemoteService(addr, cpCfg)
		if err != nil {
			return ConnectHealthResultMsg{Err: fmt.Errorf("dial: %w", err)}
		}

		healthCtx, healthCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer healthCancel()

		if err := svc.Health(healthCtx); err != nil {
			_ = svc.Close()
			return ConnectHealthResultMsg{Err: fmt.Errorf("health check: %w", err)}
		}

		infoCtx, infoCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer infoCancel()

		info, err := svc.GetHostInfo(infoCtx)
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

// runDoctorChecks runs doctor checks via the daemon's gRPC DoctorCheck RPC.
func (m ConnectModel) runDoctorChecks() tea.Cmd {
	svc := m.service
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		results, err := svc.DoctorCheck(ctx)
		if err != nil {
			return ConnectDoctorResultMsg{Err: err}
		}
		doctorResults := make([]doctor.CheckResult, len(results))
		for i, r := range results {
			doctorResults[i] = doctor.CheckResult{
				Name:     r.Name,
				Category: r.Category,
				Passed:   r.Passed,
				Message:  r.Message,
				FixCmd:   r.FixCmd,
			}
		}
		return ConnectDoctorResultMsg{Results: doctorResults}
	}
}

// scanSourceHostKeys calls the daemon's ScanSourceHostKeys RPC to add source
// host SSH keys to the daemon's known_hosts file.
func (m ConnectModel) scanSourceHostKeys() tea.Cmd {
	svc := m.service
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		results, err := svc.ScanSourceHostKeys(ctx)
		return ScanKeysCompleteMsg{Results: results, Err: err}
	}
}

// GetService returns the connected sandbox service, or nil.
func (m ConnectModel) GetService() sandbox.Service {
	return m.service
}

// deploySourceHostKey sets up a source host for daemon access: creates the
// deer-daemon user (if missing), adds it to libvirt, and deploys the daemon
// identity key. The CLI user's SSH config is used for authentication (the user
// must have sudo access on the source host).
func deploySourceHostKey(sh sandbox.SourceHostInfo, identityPubKey string, index int, logger *slog.Logger) tea.Cmd {
	return func() tea.Msg {
		sshRunFn := hostexec.NewSSHAlias(sh.Address)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := readonly.SetupSourceHost(ctx, readonly.SSHRunFunc(sshRunFn), identityPubKey, logger)
		return HostKeyDeployedMsg{Host: sh.Address, Index: index, Err: err}
	}
}

// buildDeployResults aggregates per-host statuses into a DaemonKeyDeployResultMsg.
func (m ConnectModel) buildDeployResults() *DaemonKeyDeployResultMsg {
	var deployed, skipped int
	var errs []string
	for _, hs := range m.hostDeployStatuses {
		switch hs.State {
		case HostDeployDone:
			deployed++
		case HostDeployFailed:
			skipped++
			errs = append(errs, fmt.Sprintf("%s: %s", hs.Name, hs.ErrMsg))
		}
	}
	return &DaemonKeyDeployResultMsg{Deployed: deployed, Skipped: skipped, Errors: errs}
}
