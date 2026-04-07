package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
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

func TestAllowlistDetail_Enter(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{},
	}
	m := NewAllowlistModel(cfg)

	for i, cmd := range m.builtinCmds {
		if cmd == "systemctl" {
			m.selected = i
			break
		}
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.mode != allowlistModeDetail {
		t.Error("expected detail mode")
	}
	if m.detailCommand != "systemctl" {
		t.Errorf("expected systemctl, got %s", m.detailCommand)
	}
	if len(m.detailSubcmds) == 0 {
		t.Error("expected subcommands in detail mode")
	}
}

func TestAllowlistDetail_Escape(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{},
	}
	m := NewAllowlistModel(cfg)

	for i, cmd := range m.builtinCmds {
		if cmd == "systemctl" {
			m.selected = i
			break
		}
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.mode != allowlistModeDetail {
		t.Fatal("expected detail mode")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(AllowlistModel)

	if m.mode != allowlistModeList {
		t.Error("expected back to list mode")
	}
	if m.detailCommand != "" {
		t.Error("expected detail command to be cleared")
	}
}

func TestAllowlistDetail_AddSubcommand(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{"custom-tool"},
	}
	m := NewAllowlistModel(cfg)

	m.selected = len(m.builtinCmds)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.mode != allowlistModeDetail {
		t.Fatal("expected detail mode")
	}

	m.detailAddInput.SetValue("sub1")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if len(m.detailSubcmds) != 1 {
		t.Errorf("expected 1 subcommand, got %d", len(m.detailSubcmds))
	}
	if m.detailSubcmds[0] != "sub1" {
		t.Errorf("expected sub1, got %s", m.detailSubcmds[0])
	}
	if len(m.cfg.ExtraAllowedSubcommands["custom-tool"]) != 1 {
		t.Error("expected subcommand in config")
	}
}

func TestAllowlistDetail_DeleteSubcommand(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands:    []string{"custom-tool"},
		ExtraAllowedSubcommands: map[string][]string{"custom-tool": {"sub1", "sub2"}},
	}
	m := NewAllowlistModel(cfg)

	m.selected = len(m.builtinCmds)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.mode != allowlistModeDetail {
		t.Fatal("expected detail mode")
	}

	m.detailSelected = 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(AllowlistModel)

	if len(m.detailSubcmds) != 1 {
		t.Errorf("expected 1 subcommand after delete, got %d", len(m.detailSubcmds))
	}
	if len(m.cfg.ExtraAllowedSubcommands["custom-tool"]) != 1 {
		t.Error("expected one subcommand in config after delete")
	}
}

func TestAllowlistDetail_AddDuplicateSubcommand(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands:    []string{"custom-tool"},
		ExtraAllowedSubcommands: map[string][]string{"custom-tool": {"sub1"}},
	}
	m := NewAllowlistModel(cfg)

	m.selected = len(m.builtinCmds)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	m.detailAddInput.SetValue("sub1")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.detailAddErr == "" {
		t.Error("expected error for duplicate subcommand")
	}
}

func TestAllowlistDetail_NavigateSubcommands(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands:    []string{"custom-tool"},
		ExtraAllowedSubcommands: map[string][]string{"custom-tool": {"aaa", "bbb", "ccc"}},
	}
	m := NewAllowlistModel(cfg)

	m.selected = len(m.builtinCmds)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.detailSelected != 0 {
		t.Error("expected detailSelected to be 0")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(AllowlistModel)

	if m.detailSelected != 1 {
		t.Error("expected detailSelected to be 1 after down")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(AllowlistModel)

	if m.detailSelected != 0 {
		t.Error("expected detailSelected to be 0 after up")
	}
}

func TestAllowlistDetail_ModeToggle(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands:    []string{"custom-tool"},
		ExtraAllowedSubcommands: map[string][]string{"custom-tool": {"sub1", "sub2"}},
	}
	m := NewAllowlistModel(cfg)

	m.selected = len(m.builtinCmds)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.detailModeIsAllowlist != true {
		t.Error("expected default allowlist mode for user command with subcommands")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = updated.(AllowlistModel)

	if m.detailModeIsAllowlist != false {
		t.Error("expected blocklist mode after toggle")
	}

	if m.cfg.ExtraAllowedSubcommandsMode["custom-tool"] != false {
		t.Error("expected mode to be persisted in config")
	}
}

func TestAllowlistDetail_DeleteLastSwitchToBlocklist(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands:        []string{"custom-tool"},
		ExtraAllowedSubcommands:     map[string][]string{"custom-tool": {"sub1"}},
		ExtraAllowedSubcommandsMode: map[string]bool{"custom-tool": true},
	}
	m := NewAllowlistModel(cfg)

	m.selected = len(m.builtinCmds)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.detailModeIsAllowlist != true {
		t.Fatal("expected allowlist mode")
	}

	m.detailSelected = 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(AllowlistModel)

	if m.detailModeIsAllowlist != false {
		t.Error("expected blocklist mode after deleting last subcommand")
	}
}

func TestAllowlistDetail_DefaultAllowlistForNewCommand(t *testing.T) {
	cfg := &config.Config{
		ExtraAllowedCommands: []string{"custom-tool"},
	}
	m := NewAllowlistModel(cfg)

	m.selected = len(m.builtinCmds)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AllowlistModel)

	if m.detailModeIsAllowlist != true {
		t.Error("expected default allowlist mode for new command")
	}
}
