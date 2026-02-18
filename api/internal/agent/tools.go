package agent

import "encoding/json"

// Tool represents an OpenRouter-compatible function tool definition.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes the function schema.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// AllTools returns all tool definitions available to the agent.
func AllTools() []Tool {
	return []Tool{
		// Sandbox tools
		{Type: "function", Function: ToolFunction{
			Name:        "create_sandbox",
			Description: "Create a new sandbox VM by cloning from a source VM. Set live=true for current state, live=false to use cached snapshot.",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"source_vm": map[string]any{"type": "string", "description": "Name of the source VM to clone from"}, "name": map[string]any{"type": "string", "description": "Optional sandbox name"}, "vcpus": map[string]any{"type": "integer", "description": "Number of vCPUs (default 2)"}, "memory_mb": map[string]any{"type": "integer", "description": "Memory in MB (default 2048)"}, "live": map[string]any{"type": "boolean", "description": "If true, clone from the VM's current live state (fresh snapshot). If false (default), use a cached image if available."}}, "required": []string{"source_vm"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "list_sandboxes",
			Description: "List all sandboxes in the organization",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "get_sandbox",
			Description: "Get details of a specific sandbox by ID",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"sandbox_id": map[string]any{"type": "string", "description": "The sandbox ID"}}, "required": []string{"sandbox_id"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "destroy_sandbox",
			Description: "Destroy a sandbox and release its resources",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"sandbox_id": map[string]any{"type": "string", "description": "The sandbox ID to destroy"}}, "required": []string{"sandbox_id"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "start_sandbox",
			Description: "Start a stopped sandbox",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"sandbox_id": map[string]any{"type": "string", "description": "The sandbox ID to start"}}, "required": []string{"sandbox_id"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "stop_sandbox",
			Description: "Stop a running sandbox",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"sandbox_id": map[string]any{"type": "string", "description": "The sandbox ID to stop"}}, "required": []string{"sandbox_id"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "run_command",
			Description: "Execute a shell command in a sandbox",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"sandbox_id": map[string]any{"type": "string", "description": "The sandbox ID"}, "command": map[string]any{"type": "string", "description": "The shell command to execute"}, "timeout_seconds": map[string]any{"type": "integer", "description": "Timeout in seconds (default 300)"}}, "required": []string{"sandbox_id", "command"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "get_sandbox_ip",
			Description: "Get the IP address of a sandbox",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"sandbox_id": map[string]any{"type": "string", "description": "The sandbox ID"}}, "required": []string{"sandbox_id"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "create_snapshot",
			Description: "Create a snapshot of a sandbox",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"sandbox_id": map[string]any{"type": "string", "description": "The sandbox ID"}, "name": map[string]any{"type": "string", "description": "Snapshot name"}}, "required": []string{"sandbox_id", "name"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "list_commands",
			Description: "List all commands executed in a sandbox",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"sandbox_id": map[string]any{"type": "string", "description": "The sandbox ID"}}, "required": []string{"sandbox_id"}}),
		}},

		// Source VM tools
		{Type: "function", Function: ToolFunction{
			Name:        "list_vms",
			Description: "List all source VMs across connected hosts",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "prepare_source_vm",
			Description: "Prepare a source VM for sandbox creation",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"vm_name": map[string]any{"type": "string", "description": "Name of the source VM"}, "ssh_user": map[string]any{"type": "string", "description": "SSH user for the VM"}, "key_path": map[string]any{"type": "string", "description": "Path to SSH key"}}, "required": []string{"vm_name"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "run_source_command",
			Description: "Execute a read-only command on a source VM",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"vm_name": map[string]any{"type": "string", "description": "Name of the source VM"}, "command": map[string]any{"type": "string", "description": "The command to execute"}, "timeout_seconds": map[string]any{"type": "integer", "description": "Timeout in seconds (default 30)"}}, "required": []string{"vm_name", "command"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "read_source_file",
			Description: "Read a file from a source VM",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"vm_name": map[string]any{"type": "string", "description": "Name of the source VM"}, "path": map[string]any{"type": "string", "description": "File path to read"}}, "required": []string{"vm_name", "path"}}),
		}},

		// Host tools
		{Type: "function", Function: ToolFunction{
			Name:        "list_hosts",
			Description: "List all connected sandbox hosts",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "get_host",
			Description: "Get details of a specific host",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"host_id": map[string]any{"type": "string", "description": "The host ID"}}, "required": []string{"host_id"}}),
		}},

		// Playbook tools
		{Type: "function", Function: ToolFunction{
			Name:        "create_playbook",
			Description: "Create a new Ansible-style playbook",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string", "description": "Playbook name"}, "description": map[string]any{"type": "string", "description": "Playbook description"}}, "required": []string{"name"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "update_playbook",
			Description: "Update a playbook's name or description",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"playbook_id": map[string]any{"type": "string", "description": "The playbook ID"}, "name": map[string]any{"type": "string", "description": "New name"}, "description": map[string]any{"type": "string", "description": "New description"}}, "required": []string{"playbook_id"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "delete_playbook",
			Description: "Delete a playbook and all its tasks",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"playbook_id": map[string]any{"type": "string", "description": "The playbook ID to delete"}}, "required": []string{"playbook_id"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "list_playbooks",
			Description: "List all playbooks in the organization",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "add_playbook_task",
			Description: "Add a task to a playbook",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"playbook_id": map[string]any{"type": "string", "description": "The playbook ID"}, "name": map[string]any{"type": "string", "description": "Task name"}, "module": map[string]any{"type": "string", "description": "Ansible module name (e.g. shell, apt, copy)"}, "params": map[string]any{"type": "object", "description": "Module parameters as key-value pairs"}}, "required": []string{"playbook_id", "name", "module"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "update_playbook_task",
			Description: "Update a playbook task",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"task_id": map[string]any{"type": "string", "description": "The task ID"}, "name": map[string]any{"type": "string", "description": "New task name"}, "module": map[string]any{"type": "string", "description": "New module name"}, "params": map[string]any{"type": "object", "description": "New module parameters"}}, "required": []string{"task_id"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "delete_playbook_task",
			Description: "Delete a task from a playbook",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"task_id": map[string]any{"type": "string", "description": "The task ID to delete"}}, "required": []string{"task_id"}}),
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "reorder_playbook_tasks",
			Description: "Reorder tasks in a playbook",
			Parameters:  mustJSON(map[string]any{"type": "object", "properties": map[string]any{"playbook_id": map[string]any{"type": "string", "description": "The playbook ID"}, "task_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Ordered list of task IDs"}}, "required": []string{"playbook_id", "task_ids"}}),
		}},
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
