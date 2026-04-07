package tui

import (
	"regexp"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspectrr/deer.sh/deer-cli/internal/config"
)

func TestNewRedactionModel(t *testing.T) {
	cfg := &config.Config{
		Redact: config.RedactConfig{
			Enabled:        true,
			CustomPatterns: []string{"custom-pattern", "another-pattern"},
		},
	}
	m := NewRedactionModel(cfg)

	if !m.cfg.Redact.Enabled {
		t.Error("expected redaction enabled")
	}
	if len(m.customPatterns) != 2 {
		t.Errorf("expected 2 custom patterns, got %d", len(m.customPatterns))
	}
	if m.mode != redactionModeList {
		t.Error("expected list mode")
	}
}

func TestRedactionAdd_ValidRegex(t *testing.T) {
	cfg := &config.Config{
		Redact: config.RedactConfig{
			Enabled:        true,
			CustomPatterns: []string{},
		},
	}
	m := NewRedactionModel(cfg)
	m.mode = redactionModeAdd
	m.regexInput = textinput.New()
	m.regexInput.SetValue(`\b\d{3}-\d{4}\b`)
	m.exampleInput = textinput.New()
	m.exampleInput.SetValue("Call 555-1234 for help")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(RedactionModel)

	if m.mode != redactionModeList {
		t.Error("expected back to list mode after add")
	}
	if len(m.customPatterns) != 1 {
		t.Errorf("expected 1 custom pattern, got %d", len(m.customPatterns))
	}
	if m.cfg.Redact.CustomPatterns[0] != `\b\d{3}-\d{4}\b` {
		t.Errorf("expected custom pattern, got %v", m.cfg.Redact.CustomPatterns)
	}
}

func TestRedactionAdd_InvalidRegex(t *testing.T) {
	cfg := &config.Config{
		Redact: config.RedactConfig{
			Enabled:        true,
			CustomPatterns: []string{},
		},
	}
	m := NewRedactionModel(cfg)
	m.mode = redactionModeAdd
	m.regexInput = textinput.New()
	m.regexInput.SetValue(`[invalid`)
	m.exampleInput = textinput.New()
	m.exampleInput.SetValue("some text")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(RedactionModel)

	if m.addErr == "" {
		t.Error("expected error for invalid regex")
	}
	if len(m.customPatterns) != 0 {
		t.Error("expected no patterns added")
	}
}

func TestRedactionDelete_Builtin(t *testing.T) {
	cfg := &config.Config{
		Redact: config.RedactConfig{
			Enabled:        true,
			CustomPatterns: []string{},
		},
	}
	m := NewRedactionModel(cfg)
	m.selected = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(RedactionModel)

	if len(m.customPatterns) != 0 {
		t.Error("expected no change when deleting builtin")
	}
}

func TestRedactionDelete_Custom(t *testing.T) {
	cfg := &config.Config{
		Redact: config.RedactConfig{
			Enabled:        true,
			CustomPatterns: []string{"custom-pattern"},
		},
	}
	m := NewRedactionModel(cfg)
	m.selected = 1 + len(builtinPatterns) + 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(RedactionModel)

	if len(m.customPatterns) != 0 {
		t.Logf("customPatterns: %v", m.customPatterns)
		t.Logf("cfg.CustomPatterns: %v", m.cfg.Redact.CustomPatterns)
		t.Logf("selected: %d, builtinPatterns: %d", m.selected, len(builtinPatterns))
		t.Error("expected custom pattern to be deleted")
	}
	if len(m.cfg.Redact.CustomPatterns) != 0 {
		t.Error("expected config to have no custom patterns")
	}
}

func TestRedactionToggle(t *testing.T) {
	cfg := &config.Config{
		Redact: config.RedactConfig{
			Enabled:        false,
			CustomPatterns: []string{},
		},
	}
	m := NewRedactionModel(cfg)
	m.selected = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(RedactionModel)

	if !m.cfg.Redact.Enabled {
		t.Error("expected redaction to be enabled after toggle")
	}
}

func TestRedactionPreview(t *testing.T) {
	cfg := &config.Config{
		Redact: config.RedactConfig{
			Enabled:        true,
			CustomPatterns: []string{},
		},
	}
	m := NewRedactionModel(cfg)
	m.mode = redactionModeAdd
	m.regexInput.SetValue(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	m.exampleInput.SetValue("Server at 192.168.1.100 is up")
	m.previewBefore = m.exampleInput.Value()

	m.recomputePreview()

	if m.previewAfter == "" {
		t.Error("expected preview after redaction")
	}
	if m.previewErr != "" {
		t.Errorf("unexpected preview error: %s", m.previewErr)
	}
}

func TestRedactionClose_NoSave(t *testing.T) {
	cfg := &config.Config{
		Redact: config.RedactConfig{
			Enabled:        true,
			CustomPatterns: []string{"custom-pattern"},
		},
	}
	m := NewRedactionModel(cfg)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_ = updated

	if cmd == nil {
		t.Error("expected close message")
	}
}

func TestRenderHighlighted(t *testing.T) {
	text := "Server at 192.168.1.100 is up"
	re := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b|` +
		`\b[A-Za-z0-9+/]{20,}=*\b|` +
		`\bAKIA[0-9A-Z]{16}\b|` +
		`-----BEGIN [A-Z ]+PRIVATE KEY-----|` +
		`\b(postgres|mysql|mongodb|redis)://[^\s]+`)
	locs := re.FindAllStringIndex(text, -1)
	if len(locs) != 1 {
		t.Fatalf("expected 1 match, got %d", len(locs))
	}
	if locs[0][0] != 10 || locs[0][1] != 23 {
		t.Errorf("expected match at [10,23], got %v", locs[0])
	}
}

func TestRenderReplaced(t *testing.T) {
	text := "Server at 192.168.1.100 is up"
	locs := [][]int{{12, 25}}
	result := renderReplaced(text, locs, defaultRedactionStyles().highlight)

	if result == text {
		t.Error("expected redaction to change the text")
	}
}
