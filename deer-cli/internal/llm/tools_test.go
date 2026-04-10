package llm

import (
	"sort"
	"testing"
)

func toolNames(tools []Tool) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Function.Name
	}
	sort.Strings(names)
	return names
}

func TestGetNoSourceTools(t *testing.T) {
	tools := GetNoSourceTools()
	names := toolNames(tools)

	expected := []string{
		"add_playbook_task",
		"create_playbook",
		"get_playbook",
		"list_hosts",
		"list_playbooks",
	}

	if len(names) != len(expected) {
		t.Fatalf("GetNoSourceTools() returned %d tools, want %d: got %v", len(names), len(expected), names)
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("GetNoSourceTools()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestGetNoSourceToolsExcludesServerTools(t *testing.T) {
	tools := GetNoSourceTools()
	excluded := map[string]bool{
		"run_source_command": true,
		"read_source_file":   true,
		"run_command":        true,
		"create_sandbox":     true,
		"destroy_sandbox":    true,
		"list_sandboxes":     true,
		"read_file":          true,
		"edit_file":          true,
	}

	for _, tool := range tools {
		if excluded[tool.Function.Name] {
			t.Errorf("GetNoSourceTools() should not include %q", tool.Function.Name)
		}
	}
}

func TestGetSourceOnlyTools(t *testing.T) {
	tools := GetSourceOnlyTools()
	names := toolNames(tools)

	expected := []string{
		"add_playbook_task",
		"create_playbook",
		"get_playbook",
		"list_hosts",
		"list_playbooks",
		"read_source_file",
		"run_source_command",
	}

	if len(names) != len(expected) {
		t.Fatalf("GetSourceOnlyTools() returned %d tools, want %d: got %v", len(names), len(expected), names)
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("GetSourceOnlyTools()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestGetReadOnlyTools(t *testing.T) {
	tools := GetReadOnlyTools()
	names := toolNames(tools)

	expected := []string{
		"get_playbook",
		"get_sandbox",
		"list_hosts",
		"list_playbooks",
		"list_sandboxes",
		"list_vms",
		"read_file",
		"read_source_file",
		"run_source_command",
	}

	if len(names) != len(expected) {
		t.Fatalf("GetReadOnlyTools() returned %d tools, want %d: got %v", len(names), len(expected), names)
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("GetReadOnlyTools()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestNoSourceToolsIsSubsetOfSourceOnlyTools(t *testing.T) {
	sourceOnly := make(map[string]bool)
	for _, tool := range GetSourceOnlyTools() {
		sourceOnly[tool.Function.Name] = true
	}

	for _, tool := range GetNoSourceTools() {
		if !sourceOnly[tool.Function.Name] {
			t.Errorf("GetNoSourceTools() includes %q which is not in GetSourceOnlyTools()", tool.Function.Name)
		}
	}
}

func TestAllFilteredToolsExistInGetTools(t *testing.T) {
	allNames := make(map[string]bool)
	for _, tool := range GetTools() {
		allNames[tool.Function.Name] = true
	}

	for name := range noSourceTools {
		if !allNames[name] {
			t.Errorf("noSourceTools references %q which does not exist in GetTools()", name)
		}
	}
	for name := range sourceOnlyTools {
		if !allNames[name] {
			t.Errorf("sourceOnlyTools references %q which does not exist in GetTools()", name)
		}
	}
	for name := range readOnlyTools {
		if !allNames[name] {
			t.Errorf("readOnlyTools references %q which does not exist in GetTools()", name)
		}
	}
}
