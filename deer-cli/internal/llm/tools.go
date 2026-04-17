package llm

// readOnlyTools is the set of tool names allowed in read-only mode.
var readOnlyTools = map[string]bool{
	"list_sandboxes":        true,
	"get_sandbox":           true,
	"list_vms":              true,
	"read_file":             true,
	"list_playbooks":        true,
	"get_playbook":          true,
	"run_source_command":    true,
	"read_source_file":      true,
	"request_source_access": true,
	"list_hosts":            true,
	"list_skills":           true,
	"load_skill":            true,
	"add_task":              true,
	"update_task":           true,
	"delete_task":           true,
	"list_tasks":            true,
}

// sourceOnlyTools is the set of tool names available when no sandbox hosts are configured.
var sourceOnlyTools = map[string]bool{
	"run_source_command":    true,
	"read_source_file":      true,
	"request_source_access": true,
	"list_hosts":            true,
	"create_playbook":       true,
	"add_playbook_task":     true,
	"list_playbooks":        true,
	"get_playbook":          true,
	"list_skills":           true,
	"load_skill":            true,
	"add_task":              true,
	"update_task":           true,
	"delete_task":           true,
	"list_tasks":            true,
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

// GetSourceOnlyTools returns tools for source-only mode (no sandbox hosts configured).
func GetSourceOnlyTools() []Tool {
	var tools []Tool
	for _, t := range GetTools() {
		if sourceOnlyTools[t.Function.Name] {
			tools = append(tools, t)
		}
	}
	return tools
}

// noSourceTools is the set of tool names available when no sources are prepared at all.
var noSourceTools = map[string]bool{
	"list_hosts":        true,
	"create_playbook":   true,
	"add_playbook_task": true,
	"list_playbooks":    true,
	"get_playbook":      true,
	"list_skills":       true,
	"load_skill":        true,
	"add_task":          true,
	"update_task":       true,
	"delete_task":       true,
	"list_tasks":        true,
}

// GetNoSourceTools returns tools for when no source hosts are prepared.
func GetNoSourceTools() []Tool {
	var tools []Tool
	for _, t := range GetTools() {
		if noSourceTools[t.Function.Name] {
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
				Description: "Create a new sandbox VM by cloning from a base image. Use list_vms first to see available base images for cloning.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"source_vm": {
							Type:        "string",
							Description: "The name of the base VM image to clone from. Must be a name returned by list_vms.",
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
						"live": {
							Type:        "boolean",
							Description: "If true, clone from the VM's live current state. If false (default), use cached image if available.",
						},
						"kafka_stub": {
							Type:        "boolean",
							Description: "If true, start a local Redpanda Kafka broker (localhost:9092) inside the sandbox. Use when the source service depends on Kafka. After creation, update service configs to use localhost:9092 instead of the original Kafka address.",
						},
						"es_stub": {
							Type:        "boolean",
							Description: "If true, start a local single-node Elasticsearch (localhost:9200) inside the sandbox. Use together with kafka_stub for logstash pipelines to verify data flows correctly through the pipeline. Logstash output should be pointed to localhost:9200.",
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
				Description: "List available base VM images that can be cloned to create sandboxes. These are the valid values for the source_vm parameter in create_sandbox. Does not include sandboxes - use list_sandboxes for those.",
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
				Description: "Execute a read-only command on a source host. Only diagnostic commands are allowed (ps, ls, cat, systemctl status, journalctl, etc.). This does NOT create or modify anything. If a command is blocked, use request_source_access to ask the human for approval.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"host": {
							Type:        "string",
							Description: "The name of the source host to run the command on.",
						},
						"command": {
							Type:        "string",
							Description: "The read-only diagnostic command to execute.",
						},
					},
					Required: []string{"host", "command"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "read_source_file",
				Description: "Read the contents of a file on a source host. This is read-only and does not modify the host.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"host": {
							Type:        "string",
							Description: "The name of the source host containing the file.",
						},
						"path": {
							Type:        "string",
							Description: "The absolute path to the file on the source host to read.",
						},
					},
					Required: []string{"host", "path"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "request_source_access",
				Description: "Request human approval to run a command that was blocked by the read-only allowlist on a source host. Use this when a diagnostic command is denied and you genuinely need it for troubleshooting. The human will see your reason and can approve or deny the request.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"host": {
							Type:        "string",
							Description: "The name of the source host to run the command on.",
						},
						"command": {
							Type:        "string",
							Description: "The command that was blocked by the read-only allowlist.",
						},
						"reason": {
							Type:        "string",
							Description: "Why you need this command. Be specific about what information it provides and why it is necessary for diagnosis.",
						},
					},
					Required: []string{"host", "command", "reason"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "verify_pipeline_output",
				Description: "Query the local Elasticsearch stub inside a sandbox to verify logstash pipeline output. Use this after configuring logstash to confirm data is flowing correctly through the pipeline. Requires es_stub=true during sandbox creation.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"sandbox_id": {
							Type:        "string",
							Description: "The ID of the sandbox running the Elasticsearch stub.",
						},
						"index": {
							Type:        "string",
							Description: "Elasticsearch index pattern to query (e.g. 'logstash-*'). Defaults to '_all'.",
						},
						"query": {
							Type:        "string",
							Description: "Optional Elasticsearch query string (lucene syntax). Defaults to match_all.",
						},
						"size": {
							Type:        "integer",
							Description: "Maximum number of documents to return. Defaults to 10.",
						},
					},
					Required: []string{"sandbox_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "list_hosts",
				Description: "List all configured source hosts (production systems) with their preparation status. These are for read-only investigation via run_source_command, NOT for create_sandbox.",
				Parameters: ParameterSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "list_skills",
				Description: "List all available skills. Skills provide domain-specific knowledge (e.g. Elasticsearch deployment, Kafka operations, on-call debugging). Use this to discover what skills are available, then use load_skill to get the full content.",
				Parameters: ParameterSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "load_skill",
				Description: "Load the full content of a skill by name. Use this after list_skills to retrieve detailed domain knowledge. The loaded skill provides procedures, runbooks, and tool usage guidance for specific technologies.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"name": {
							Type:        "string",
							Description: "The name of the skill to load (must match a name from list_skills).",
						},
					},
					Required: []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "add_task",
				Description: "Add a new task to the task list. Use this to track the steps you plan to take. Tasks help the human follow your progress.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"content": {
							Type:        "string",
							Description: "A short description of the task (imperative form, e.g. 'Install nginx').",
						},
					},
					Required: []string{"content"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "update_task",
				Description: "Update a task's status or content. Use this to mark tasks as in_progress when you start them and completed when done.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"task_id": {
							Type:        "string",
							Description: "The ID of the task to update.",
						},
						"status": {
							Type:        "string",
							Description: "New status: 'pending', 'in_progress', or 'completed'.",
							Enum:        []string{"pending", "in_progress", "completed"},
						},
						"content": {
							Type:        "string",
							Description: "Updated task description (optional).",
						},
					},
					Required: []string{"task_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "delete_task",
				Description: "Delete a task from the task list by ID.",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]Property{
						"task_id": {
							Type:        "string",
							Description: "The ID of the task to delete.",
						},
					},
					Required: []string{"task_id"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "list_tasks",
				Description: "List all current tasks and their statuses. Returns the full task list so you can review progress.",
				Parameters: ParameterSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			},
		},
	}
}
