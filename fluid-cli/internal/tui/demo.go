package tui

import (
	"time"
)

// DemoStep represents a single step in the demo
type DemoStep struct {
	Type       DemoStepType
	Content    string // For messages
	ToolName   string // For tool calls
	ToolArgs   map[string]any
	ToolResult map[string]any
	ToolError  bool
	Delay      time.Duration // Delay before this step
}

// DemoStepType indicates what kind of step this is
type DemoStepType int

const (
	DemoStepUserMessage DemoStepType = iota
	DemoStepAssistantMessage
	DemoStepToolStart
	DemoStepToolComplete
)

// GetDemoScript returns the scripted demo conversation
func GetDemoScript() []DemoStep {
	return []DemoStep{
		// User sends the initial request
		{
			Type:    DemoStepUserMessage,
			Content: "Hey can you create a sandbox from test-vm-3 and install an httpd server, make sure it runs and serve a simple HTML page from it and create an Ansible playbook to reproduce it.",
			Delay:   500 * time.Millisecond,
		},

		// Assistant responds
		{
			Type:    DemoStepAssistantMessage,
			Content: "I'll help you with that! Let me create a sandbox from `test-vm-3`, set up Apache HTTP server, verify it's working, and then create an Ansible playbook to reproduce the setup.",
			Delay:   1 * time.Second,
		},

		// Create sandbox
		{
			Type:     DemoStepToolStart,
			ToolName: "create_sandbox",
			ToolArgs: map[string]any{
				"source_vm_name": "test-vm-3",
			},
			Delay: 500 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "create_sandbox",
			ToolResult: map[string]any{
				"sandbox_id": "SBX-demo1234",
				"name":       "sbx-demo",
				"state":      "RUNNING",
				"ip_address": "192.168.122.50",
			},
			Delay: 2 * time.Second,
		},

		// Check OS
		{
			Type:     DemoStepToolStart,
			ToolName: "run_command",
			ToolArgs: map[string]any{
				"sandbox_id": "SBX-demo1234",
				"command":    "cat /etc/os-release | head -2",
			},
			Delay: 500 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "run_command",
			ToolResult: map[string]any{
				"exit_code": 0,
				"stdout":    "NAME=\"Ubuntu\"\nVERSION=\"22.04.3 LTS (Jammy Jellyfish)\"",
				"stderr":    "",
			},
			Delay: 1500 * time.Millisecond,
		},

		// Update packages
		{
			Type:     DemoStepToolStart,
			ToolName: "run_command",
			ToolArgs: map[string]any{
				"sandbox_id": "SBX-demo1234",
				"command":    "apt update",
			},
			Delay: 500 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "run_command",
			ToolResult: map[string]any{
				"exit_code": 0,
				"stdout":    "Hit:1 http://archive.ubuntu.com/ubuntu jammy InRelease\nGet:2 http://archive.ubuntu.com/ubuntu jammy-updates InRelease [119 kB]\nReading package lists... Done\nBuilding dependency tree... Done\n45 packages can be upgraded.",
				"stderr":    "",
			},
			Delay: 2500 * time.Millisecond,
		},

		// Install Apache
		{
			Type:     DemoStepToolStart,
			ToolName: "run_command",
			ToolArgs: map[string]any{
				"sandbox_id": "SBX-demo1234",
				"command":    "apt install -y apache2",
			},
			Delay: 500 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "run_command",
			ToolResult: map[string]any{
				"exit_code": 0,
				"stdout":    "Reading package lists... Done\nBuilding dependency tree... Done\nThe following NEW packages will be installed:\n  apache2 apache2-bin apache2-data apache2-utils\n0 upgraded, 4 newly installed, 0 to remove.\nSetting up apache2 (2.4.52-1ubuntu4.6) ...\nEnabling module mpm_event.\nEnabling module authz_core.\nEnabling site 000-default.\nCreated symlink /etc/systemd/system/multi-user.target.wants/apache2.service",
				"stderr":    "",
			},
			Delay: 3 * time.Second,
		},

		// Create HTML page
		{
			Type:     DemoStepToolStart,
			ToolName: "run_command",
			ToolArgs: map[string]any{
				"sandbox_id": "SBX-demo1234",
				"command":    "echo '<!DOCTYPE html><html><head><title>Fluid Demo</title></head><body><h1>Hello from Fluid!</h1><p>This page was set up automatically.</p></body></html>' > /var/www/html/index.html",
			},
			Delay: 500 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "run_command",
			ToolResult: map[string]any{
				"exit_code": 0,
				"stdout":    "",
				"stderr":    "",
			},
			Delay: 1 * time.Second,
		},

		// Start Apache
		{
			Type:     DemoStepToolStart,
			ToolName: "run_command",
			ToolArgs: map[string]any{
				"sandbox_id": "SBX-demo1234",
				"command":    "systemctl start apache2 && systemctl status apache2 | head -3",
			},
			Delay: 500 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "run_command",
			ToolResult: map[string]any{
				"exit_code": 0,
				"stdout":    "● apache2.service - The Apache HTTP Server\n     Loaded: loaded (/lib/systemd/system/apache2.service; enabled)\n     Active: active (running) since Sat 2026-01-25 12:00:00 UTC",
				"stderr":    "",
			},
			Delay: 1500 * time.Millisecond,
		},

		// Test with curl
		{
			Type:     DemoStepToolStart,
			ToolName: "run_command",
			ToolArgs: map[string]any{
				"sandbox_id": "SBX-demo1234",
				"command":    "curl -s localhost",
			},
			Delay: 500 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "run_command",
			ToolResult: map[string]any{
				"exit_code": 0,
				"stdout":    "<!DOCTYPE html><html><head><title>Fluid Demo</title></head><body><h1>Hello from Fluid!</h1><p>This page was set up automatically.</p></body></html>",
				"stderr":    "",
			},
			Delay: 1 * time.Second,
		},

		// Create playbook
		{
			Type:     DemoStepToolStart,
			ToolName: "create_playbook",
			ToolArgs: map[string]any{
				"name":        "httpd-setup",
				"description": "Install and configure Apache HTTP server with custom page",
			},
			Delay: 500 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "create_playbook",
			ToolResult: map[string]any{
				"id":   "PB-demo5678",
				"name": "httpd-setup",
			},
			Delay: 1 * time.Second,
		},

		// Add playbook tasks
		{
			Type:     DemoStepToolStart,
			ToolName: "add_playbook_task",
			ToolArgs: map[string]any{
				"playbook_name": "httpd-setup",
				"task_name":     "Update apt cache",
				"module":        "apt",
			},
			Delay: 300 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "add_playbook_task",
			ToolResult: map[string]any{
				"task_id": "TASK-001",
				"name":    "httpd-setup",
			},
			Delay: 500 * time.Millisecond,
		},

		{
			Type:     DemoStepToolStart,
			ToolName: "add_playbook_task",
			ToolArgs: map[string]any{
				"playbook_name": "httpd-setup",
				"task_name":     "Install Apache",
				"module":        "apt",
			},
			Delay: 300 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "add_playbook_task",
			ToolResult: map[string]any{
				"task_id": "TASK-002",
				"name":    "httpd-setup",
			},
			Delay: 500 * time.Millisecond,
		},

		{
			Type:     DemoStepToolStart,
			ToolName: "add_playbook_task",
			ToolArgs: map[string]any{
				"playbook_name": "httpd-setup",
				"task_name":     "Create index.html",
				"module":        "copy",
			},
			Delay: 300 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "add_playbook_task",
			ToolResult: map[string]any{
				"task_id": "TASK-003",
				"name":    "httpd-setup",
			},
			Delay: 500 * time.Millisecond,
		},

		{
			Type:     DemoStepToolStart,
			ToolName: "add_playbook_task",
			ToolArgs: map[string]any{
				"playbook_name": "httpd-setup",
				"task_name":     "Start and enable Apache",
				"module":        "systemd",
			},
			Delay: 300 * time.Millisecond,
		},
		{
			Type:     DemoStepToolComplete,
			ToolName: "add_playbook_task",
			ToolResult: map[string]any{
				"task_id": "TASK-004",
				"name":    "httpd-setup",
			},
			Delay: 500 * time.Millisecond,
		},

		// Final assistant message
		{
			Type: DemoStepAssistantMessage,
			Content: `Done! Here's what I accomplished:

**Sandbox Created**
- ID: SBX-demo1234
- IP: 192.168.122.50

**Apache HTTP Server**
- Installed and running
- Custom page at /var/www/html/index.html
- Verified working with curl

**Ansible Playbook: httpd-setup**
The playbook includes 4 tasks:
1. Update apt cache
2. Install Apache
3. Create custom index.html
4. Start and enable Apache service

You can run this playbook on any Ubuntu server to reproduce this setup.`,
			Delay: 5 * time.Second,
		},
	}
}
