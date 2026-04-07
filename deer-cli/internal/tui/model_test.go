package tui

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
	"github.com/aspectrr/deer.sh/deer-cli/internal/sandbox"
)

type stubModelRunner struct {
	cancelled bool
	runID     uint64
}

func (s *stubModelRunner) Run(string) tea.Cmd                      { return nil }
func (s *stubModelRunner) Reset()                                  {}
func (s *stubModelRunner) SetStatusCallback(func(tea.Msg))         {}
func (s *stubModelRunner) SetReadOnly(bool)                        {}
func (s *stubModelRunner) Cancel()                                 { s.cancelled = true }
func (s *stubModelRunner) RunID() uint64                           { return s.runID }
func (s *stubModelRunner) SetSandboxService(sandbox.Service) error { return nil }

func newTestModel(t *testing.T) (Model, *stubModelRunner) {
	t.Helper()

	runner := &stubModelRunner{runID: 1}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	model := NewModel("fluid", "test", "test-model", runner, &config.Config{}, "", logger)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	return updated.(Model), runner
}

func newTestModelWithAgent(t *testing.T) (Model, *FluidAgent) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	agent := &FluidAgent{logger: logger}
	model := NewModel("fluid", "test", "test-model", agent, &config.Config{}, "", logger)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	return updated.(Model), agent
}

func dequeueStatus(t *testing.T, model Model) tea.Msg {
	t.Helper()
	return model.listenForStatus()()
}

func TestModelEscapeCancelsAndClearsLiveState(t *testing.T) {
	model, runner := newTestModel(t)

	model.state = StateThinking
	model.thinking = true
	model.agentStatus = StatusWorking
	model.currentToolName = "create_sandbox"
	model.currentToolArgs = map[string]any{"source_vm": "ubuntu"}
	model.currentRetry = &RetryAttemptMsg{SandboxID: "SBX-1", Attempt: 2}
	model.conversation = append(model.conversation,
		ConversationEntry{Role: "assistant", Content: "working"},
		ConversationEntry{Role: "live_output", Content: "partial output"},
		ConversationEntry{Role: "live_prepare", Content: "partial prepare"},
		ConversationEntry{Role: "live_create", Content: "partial create"},
	)
	model.showingLiveOutput = true
	model.liveOutputLines = []string{"line"}
	model.liveOutputPending = "pending"
	model.liveOutputSandbox = "SBX-1"
	model.liveOutputCommand = "uname -a"
	model.liveOutputIndex = 2
	model.showingLivePrepare = true
	model.livePrepareSourceVM = "ubuntu"
	model.livePrepareSteps = []string{"step"}
	model.livePrepareIndex = 3
	model.showingLiveCreate = true
	model.liveCreateSourceVM = "ubuntu"
	model.liveCreateSteps = []string{"step"}
	model.liveCreateIndex = 4
	model.statusChan <- ToolStartMsg{ToolName: "create_sandbox"}
	model.statusChan <- SandboxCreateProgressMsg{SourceVM: "ubuntu", StepName: "Booting", StepNum: 5, Total: 7}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	if !runner.cancelled {
		t.Fatal("expected Cancel to be called")
	}
	if model.state != StateIdle {
		t.Fatalf("state = %v, want %v", model.state, StateIdle)
	}
	if model.thinking {
		t.Fatal("expected thinking to be false")
	}
	if model.showingLiveOutput || model.showingLivePrepare || model.showingLiveCreate {
		t.Fatal("expected all live views to be cleared")
	}
	if model.liveOutputLines != nil || model.liveCreateSteps != nil || model.livePrepareSteps != nil {
		t.Fatal("expected live buffers to be reset")
	}
	if model.liveOutputPending != "" {
		t.Fatalf("liveOutputPending = %q, want empty", model.liveOutputPending)
	}
	if model.currentRetry != nil {
		t.Fatal("expected retry state to be cleared")
	}
	if model.currentToolName != "" || model.currentToolArgs != nil {
		t.Fatal("expected current tool state to be cleared")
	}
	if got := len(model.statusChan); got != 0 {
		t.Fatalf("status channel length = %d, want 0", got)
	}
	if len(model.conversation) != 2 {
		t.Fatalf("conversation length = %d, want 2", len(model.conversation))
	}
	for _, entry := range model.conversation {
		if entry.Role == "live_output" || entry.Role == "live_prepare" || entry.Role == "live_create" {
			t.Fatalf("unexpected live conversation entry after cancel: %q", entry.Role)
		}
	}
}

func TestSandboxCreateProgressDoneClosesActiveBox(t *testing.T) {
	model, _ := newTestModel(t)

	updated, _ := model.Update(SandboxCreateProgressMsg{
		SourceVM: "ubuntu",
		StepName: "Booting microVM",
		StepNum:  5,
		Total:    7,
	})
	model = updated.(Model)

	if !model.showingLiveCreate {
		t.Fatal("expected live create box to be active after progress")
	}

	updated, _ = model.Update(SandboxCreateProgressMsg{
		SourceVM: "ubuntu",
		StepName: "Ready",
		StepNum:  1,
		Total:    7,
		Done:     true,
	})
	model = updated.(Model)

	if model.showingLiveCreate {
		t.Fatal("expected Done progress to close the live create box")
	}
	if len(model.liveCreateSteps) != 1 {
		t.Fatalf("liveCreateSteps length = %d, want 1", len(model.liveCreateSteps))
	}
	if got := model.liveCreateSteps[0]; got != "  v [1/7] Ready" {
		t.Fatalf("last live create step = %q, want %q", got, "  v [1/7] Ready")
	}
}

func TestSandboxCreateProgressMarksPreviousStepDone(t *testing.T) {
	model, _ := newTestModel(t)

	updated, _ := model.Update(SandboxCreateProgressMsg{
		SourceVM: "ubuntu",
		StepName: "Resolving source host",
		StepNum:  1,
		Total:    9,
	})
	model = updated.(Model)

	updated, _ = model.Update(SandboxCreateProgressMsg{
		SourceVM: "ubuntu",
		StepName: "Pulling fresh snapshot",
		StepNum:  2,
		Total:    9,
	})
	model = updated.(Model)

	if len(model.liveCreateSteps) != 2 {
		t.Fatalf("liveCreateSteps length = %d, want 2", len(model.liveCreateSteps))
	}
	if got := model.liveCreateSteps[0]; got != "  v [1/9] Resolving source host" {
		t.Fatalf("first live create step = %q, want %q", got, "  v [1/9] Resolving source host")
	}
	if got := model.liveCreateSteps[1]; got != "  . [2/9] Pulling fresh snapshot..." {
		t.Fatalf("second live create step = %q, want %q", got, "  . [2/9] Pulling fresh snapshot...")
	}
}

func TestSandboxCreateProgressDoneWithoutStepsDoesNotCreateBox(t *testing.T) {
	model, _ := newTestModel(t)
	initialConversationLen := len(model.conversation)

	updated, _ := model.Update(SandboxCreateProgressMsg{
		SourceVM: "ubuntu",
		Done:     true,
	})
	model = updated.(Model)

	if model.showingLiveCreate {
		t.Fatal("expected Done-only progress to leave live create inactive")
	}
	if len(model.liveCreateSteps) != 0 {
		t.Fatalf("liveCreateSteps length = %d, want 0", len(model.liveCreateSteps))
	}
	if len(model.conversation) != initialConversationLen {
		t.Fatalf("conversation length = %d, want %d", len(model.conversation), initialConversationLen)
	}
}

func TestSandboxCreateProgressDoneWithoutDetailsClosesWithoutOverwriting(t *testing.T) {
	model, _ := newTestModel(t)

	updated, _ := model.Update(SandboxCreateProgressMsg{
		SourceVM: "ubuntu",
		StepName: "Booting microVM",
		StepNum:  7,
		Total:    9,
	})
	model = updated.(Model)

	updated, _ = model.Update(SandboxCreateProgressMsg{Done: true})
	model = updated.(Model)

	if model.showingLiveCreate {
		t.Fatal("expected Done-only close to hide the live create box")
	}
	if len(model.liveCreateSteps) != 1 {
		t.Fatalf("liveCreateSteps length = %d, want 1", len(model.liveCreateSteps))
	}
	if got := model.liveCreateSteps[0]; got != "  . [7/9] Booting microVM..." {
		t.Fatalf("live create step = %q, want in-flight step to remain unchanged", got)
	}
}

func TestModelConsecutiveRunsAfterPrepareShowSourceToolResults(t *testing.T) {
	model, _ := newTestModel(t)

	model.addUserMessage("/prepare test-vm-1")
	model.state = StateThinking
	model.thinking = true
	model.currentInput = "/prepare test-vm-1"
	model.statusChan <- AgentResponseMsg{Response: AgentResponse{
		Content: "Host test-vm-1 is prepared.",
		Done:    true,
	}}

	updated, _ := model.Update(dequeueStatus(t, model))
	model = updated.(Model)

	if model.state != StateIdle {
		t.Fatalf("state after prepare = %v, want %v", model.state, StateIdle)
	}
	if model.thinking {
		t.Fatal("expected thinking to stop after prepare completion")
	}
	if got := len(model.statusChan); got != 0 {
		t.Fatalf("status channel length after prepare = %d, want 0", got)
	}

	model.addUserMessage("Hey can you investigate the nginx issue on test-vm-1?")
	model.state = StateThinking
	model.thinking = true
	model.currentInput = "Hey can you investigate the nginx issue on test-vm-1?"
	model.statusChan <- ToolStartMsg{
		ToolName: "run_source_command",
		Args: map[string]any{
			"host":    "test-vm-1",
			"command": "systemctl status nginx",
		},
	}

	updated, _ = model.Update(dequeueStatus(t, model))
	model = updated.(Model)

	if model.currentToolName != "run_source_command" {
		t.Fatalf("currentToolName = %q, want run_source_command", model.currentToolName)
	}

	model.statusChan <- ToolCompleteMsg{
		ToolName: "run_source_command",
		Success:  true,
		Result: map[string]any{
			"exit_code": 0,
			"stdout":    "nginx.service - active\n",
			"stderr":    "",
		},
	}

	updated, _ = model.Update(dequeueStatus(t, model))
	model = updated.(Model)

	model.statusChan <- AgentResponseMsg{Response: AgentResponse{
		Content: "nginx is active on test-vm-1.",
		Done:    true,
	}}

	updated, _ = model.Update(dequeueStatus(t, model))
	model = updated.(Model)

	if model.state != StateIdle {
		t.Fatalf("state after second run = %v, want %v", model.state, StateIdle)
	}

	toolCount := 0
	for _, entry := range model.conversation {
		if entry.Role == "tool" {
			toolCount++
		}
	}
	if toolCount != 1 {
		t.Fatalf("tool entry count = %d, want 1", toolCount)
	}

	view := model.View()
	if !strings.Contains(view, "run_source_command") {
		t.Fatalf("view missing tool name: %q", view)
	}
	if !strings.Contains(view, "systemctl status nginx") {
		t.Fatalf("view missing source command: %q", view)
	}
	if !strings.Contains(view, "nginx is active on test-vm-1.") {
		t.Fatalf("view missing final assistant response: %q", view)
	}
}

func TestCleanupStartMsgInitializesCleanupView(t *testing.T) {
	model, _ := newTestModel(t)

	updated, cmd := model.Update(CleanupStartMsg{SandboxIDs: []string{"sbx-1", "sbx-2"}})
	model = updated.(Model)

	if !model.inCleanup {
		t.Fatal("expected cleanup mode to be active")
	}
	if model.quitting {
		t.Fatal("expected quitting to be cleared when cleanup starts")
	}
	if len(model.cleanupOrder) != 2 {
		t.Fatalf("cleanupOrder length = %d, want 2", len(model.cleanupOrder))
	}
	if model.cleanupStatuses["sbx-1"] != CleanupStatusPending || model.cleanupStatuses["sbx-2"] != CleanupStatusPending {
		t.Fatalf("cleanupStatuses = %v, want both pending", model.cleanupStatuses)
	}
	if cmd == nil {
		t.Fatal("expected CleanupStartMsg to continue status listening")
	}
	view := model.View()
	if !strings.Contains(view, "Cleaning Up Sandboxes") {
		t.Fatalf("cleanup view missing header: %q", view)
	}
}

func TestStartCleanupReturnsCleanupStartMsg(t *testing.T) {
	model, agent := newTestModelWithAgent(t)
	model.cleanupOrder = []string{"sbx-1"}
	agent.createdSandboxes = []string{"sbx-1"}

	cmd := model.startCleanup()
	if cmd == nil {
		t.Fatal("expected cleanup command")
	}
	msg := cmd()
	startMsg, ok := msg.(CleanupStartMsg)
	if !ok {
		t.Fatalf("cleanup command returned %T, want CleanupStartMsg", msg)
	}
	if len(startMsg.SandboxIDs) != 1 || startMsg.SandboxIDs[0] != "sbx-1" {
		t.Fatalf("CleanupStartMsg = %+v, want sandbox sbx-1", startMsg)
	}
}
