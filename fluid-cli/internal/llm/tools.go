package llm

// readOnlyTools is the set of tool names allowed in read-only mode.
var readOnlyTools = map[string]bool{
	"list_sandboxes":     true,
	"get_sandbox":        true,
	"list_vms":           true,
	"read_file":          true,
	"list_playbooks":     true,
	"get_playbook":       true,
	"run_source_command": true,
	"read_source_file":   true,
}

// GetReadOnlyTools returns only the tools that are safe for read-only mode.
func GetReadOnlyTools() []Tool {
	var tools []Tool
	for _, t := range GetTools() {
		if readOnlyTools[t.Function.Name] {
			tools = append(tools, t)
		}
	}
	return tools
}

// GetTools returns the list of tools available to the LLM.
func GetTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: Function{
				Name:        "list_sandboxes",
				Description: "List all existing sandboxes with their state and IP addresses.",
				Parameters: ParameterSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "create_sandbox",
				Description: "Create a new sandbox VM by cloning from a source VM.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"source_vm": {
							Type:        "string",
							Description: "The name of the source VM to clone from (e.g., 'ubuntu-base').",
						},
						"host": {
							Type:        "string",
							Description: "Optional target host name for multi-host setups.",
						},
						"cpu": {
							Type:        "integer",
							Description: "Number of vCPUs (default: 2).",
						},
						"memory_mb": {
							Type:        "integer",
							Description: "RAM in MB (default: 4096).",
						},
					},
					Required: []string{"source_vm"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "destroy_sandbox",
				Description: "Completely destroy a sandbox VM and remove its storage.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox to destroy.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "run_command",
				Description: "Execute a shell command inside a sandbox via SSH.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox to run the command in.",
						},
						"command": {
							Type:        "string",
							Description: "The shell command to execute.",
						},
					},
					Required: []string{"sandbox_id", "command"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "start_sandbox",
				Description: "Start a stopped sandbox VM.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox to start.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "stop_sandbox",
				Description: "Stop a running sandbox VM.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox to stop.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "get_sandbox",
				Description: "Get detailed information about a specific sandbox.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "list_vms",
				Description: "List available host VMs (base images) that can be cloned to create sandboxes. Does not include sandboxes - use list_sandboxes for those.",
				Parameters: ParameterSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "create_snapshot",
				Description: "Create a snapshot of the current sandbox state.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox.",
						},
						"name": {
							Type:        "string",
							Description: "Optional name for the snapshot.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "create_playbook",
				Description: "Create a new Ansible playbook.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"name": {
							Type:        "string",
							Description: "Name of the playbook.",
						},
						"hosts": {
							Type:        "string",
							Description: "Target hosts (default: 'all').",
						},
						"become": {
							Type:        "boolean",
							Description: "Whether to use privilege escalation (sudo).",
						},
					},
					Required: []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "add_playbook_task",
				Description: "Add a task to an Ansible playbook.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"playbook_id": {
							Type:        "string",
							Description: "The ID of the playbook.",
						},
						"name": {
							Type:        "string",
							Description: "Name of the task.",
						},
						"module": {
							Type:        "string",
							Description: "Ansible module to use (e.g., 'apt', 'shell', 'copy').",
						},
						"params": {
							Type:        "object",
							Description: "Parameters for the Ansible module.",
						},
					},
					Required: []string{"playbook_id", "name", "module"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "edit_file",
				Description: "Edit a file on a sandbox VM by replacing text or create a new file. This tool operates on files INSIDE the sandbox via SSH - not local files or playbooks. If old_str is empty, creates/overwrites the file with new_str. Otherwise replaces the first occurrence of old_str with new_str. For viewing playbook definitions, use get_playbook instead.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox containing the file.",
						},
						"path": {
							Type:        "string",
							Description: "The absolute path to the file inside the sandbox to edit or create.",
						},
						"old_str": {
							Type:        "string",
							Description: "The string to find and replace. If empty, the file will be created/overwritten with new_str.",
						},
						"new_str": {
							Type:        "string",
							Description: "The string to replace old_str with, or the content for a new file.",
						},
					},
					Required: []string{"sandbox_id", "path", "new_str"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "read_file",
				Description: "Read the contents of a file on a sandbox VM. This tool operates on files INSIDE the sandbox via SSH - not local files or playbooks. For viewing playbook definitions, use get_playbook instead.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox containing the file.",
						},
						"path": {
							Type:        "string",
							Description: "The absolute path to the file inside the sandbox to read.",
						},
					},
					Required: []string{"sandbox_id", "path"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "list_playbooks",
				Description: "List all Ansible playbooks that have been created. Returns playbook IDs, names, and file paths. Use get_playbook to retrieve the full contents of a specific playbook.",
				Parameters: ParameterSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "get_playbook",
				Description: "Get the full definition of an Ansible playbook including its YAML content and all tasks. Use this to view playbook contents - do NOT use read_file for playbooks.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"playbook_id": {
							Type:        "string",
							Description: "The ID of the playbook to retrieve.",
						},
					},
					Required: []string{"playbook_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "run_source_command",
				Description: "Execute a read-only command on a source/golden VM. Only diagnostic commands are allowed (ps, ls, cat, systemctl status, journalctl, etc.). This does NOT create or modify anything.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"source_vm": {
							Type:        "string",
							Description: "The name of the source VM to run the command on.",
						},
						"command": {
							Type:        "string",
							Description: "The read-only diagnostic command to execute.",
						},
					},
					Required: []string{"source_vm", "command"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "read_source_file",
				Description: "Read the contents of a file on a source/golden VM. This is read-only and does not modify the VM.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"source_vm": {
							Type:        "string",
							Description: "The name of the source VM containing the file.",
						},
						"path": {
							Type:        "string",
							Description: "The absolute path to the file inside the source VM to read.",
						},
					},
					Required: []string{"source_vm", "path"},
				},
			},
		},
	}
}
