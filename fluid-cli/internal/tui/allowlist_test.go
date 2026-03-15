package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspectrr/fluid.sh/fluid-cli/internal/config"
)

func TestNewAllowlistModel(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{"custom-cmd", "another-cmd"},
	}
	m := NewAllowlistModel(cfg)

	if len(m.builtinCmds) == 0 {
		t.Error("expected non-empty builtin commands")
	}
	if len(m.userCmds) != 2 {
		t.Errorf("expected 2 user commands, got %d", len(m.userCmds))
	}
	if m.mode != allowlistModeList {
		t.Error("expected list mode")
	}
}

func TestAllowlistAdd(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{},
	}
	m := NewAllowlistModel(cfg)
	m.mode = allowlistModeAdd
	m.addInput = textinput.New()
	m.addInput.SetValue("new-cmd")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.mode != allowlistModeList {
		t.Error("expected back to list mode after add")
	}
	if len(m.userCmds) != 1 {
		t.Errorf("expected 1 user command, got %d", len(m.userCmds))
	}
	if m.userCmds[0] != "new-cmd" {
		t.Errorf("expected new-cmd, got %s", m.userCmds[0])
	}
	if m.cfg.ExtraAllowedCommands[0] != "new-cmd" {
		t.Errorf("expected config to have new-cmd, got %v", m.cfg.ExtraAllowedCommands)
	}
}

func TestAllowlistAdd_Duplicate(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{},
	}
	m := NewAllowlistModel(cfg)
	m.mode = allowlistModeAdd
	m.addInput = textinput.New()
	m.addInput.SetValue("cat")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.addErr == "" {
		t.Error("expected error for duplicate command")
	}
	if len(m.userCmds) != 0 {
		t.Error("expected no user commands added")
	}
}

func TestAllowlistAdd_Empty(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{},
	}
	m := NewAllowlistModel(cfg)
	m.mode = allowlistModeAdd
	m.addInput = textinput.New()
	m.addInput.SetValue("")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.addErr == "" {
		t.Error("expected error for empty command")
	}
}

func TestAllowlistDelete_Builtin(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{},
	}
	m := NewAllowlistModel(cfg)
	m.selected = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(AllowlistModel)

	if len(m.userCmds) != 0 {
		t.Error("expected no change when deleting builtin")
	}
}

func TestAllowlistDelete_User(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{"custom-cmd"},
	}
	m := NewAllowlistModel(cfg)
	m.selected = len(m.builtinCmds)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(AllowlistModel)

	if len(m.userCmds) != 0 {
		t.Error("expected user command to be deleted")
	}
	if len(m.cfg.ExtraAllowedCommands) != 0 {
		t.Error("expected config to have no extra commands")
	}
}

func TestAllowlistClose_NoSave(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{"custom-cmd"},
	}
	m := NewAllowlistModel(cfg)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_ = updated

	if cmd == nil {
		t.Error("expected close message")
	}
	msg := <-func() chan tea.Msg {
		ch := make(chan tea.Msg, 1)
		go func() {
			ch <- cmd()
		}()
		return ch
	}()

	closeMsg, ok := msg.(AllowlistCloseMsg)
	if !ok {
		t.Error("expected AllowlistCloseMsg")
	}
	if closeMsg.Saved {
		t.Error("expected saved=false")
	}
}

func TestAllowlistClose_Save(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{"custom-cmd"},
	}
	m := NewAllowlistModel(cfg)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	_ = updated

	if cmd == nil {
		t.Error("expected close message")
	}
}
