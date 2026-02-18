package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aspectrr/fluid.sh/fluid/internal/ansible"
	"github.com/aspectrr/fluid.sh/fluid/internal/sandbox"
)

// jsonResult marshals v to JSON and returns it as a text tool result.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

// errorResult marshals v to JSON and returns it as a tool result with IsError set.
// This gives AI agents structured error context instead of opaque error strings.
func errorResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal error result: %w", err)
	}
	result := mcp.NewToolResultText(string(data))
	result.IsError = true
	return result, nil
}

// shellEscape safely escapes a string for use in a shell command.
func shellEscape(s string) (string, error) {
	if err := validateShellArg(s); err != nil {
		return "", err
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'", nil
}

// trackToolCall records an mcp_tool_call telemetry event.
func (s *Server) trackToolCall(toolName string) {
	if s.telemetry != nil {
		s.telemetry.Track("mcp_tool_call", map[string]any{
			"tool_name": toolName,
		})
	}
}

// --- Handlers ---

func (s *Server) handleListSandboxes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("list_sandboxes")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	sandboxes, err := s.service.ListSandboxes(ctx)
	if err != nil {
		s.logger.Error("list_sandboxes failed", "error", err)
		return errorResult(map[string]any{"error": fmt.Sprintf("list sandboxes: %s", err)})
	}

	result := make([]map[string]any, 0, len(sandboxes))
	for _, sb := range sandboxes {
		item := map[string]any{
			"id":         sb.ID,
			"name":       sb.Name,
			"state":      sb.State,
			"base_image": sb.BaseImage,
			"created_at": sb.CreatedAt.Format(time.RFC3339),
		}
		if sb.IPAddress != "" {
			item["ip"] = sb.IPAddress
		}
		result = append(result, item)
	}

	return jsonResult(map[string]any{
		"sandboxes": result,
		"count":     len(result),
	})
}

func (s *Server) handleCreateSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("create_sandbox")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	sourceVM := request.GetString("source_vm", "")
	if sourceVM == "" {
		return nil, fmt.Errorf("source_vm is required")
	}
	cpu := request.GetInt("cpu", 0)
	memoryMB := request.GetInt("memory_mb", 0)
	live := request.GetBool("live", false)

	sb, err := s.service.CreateSandbox(ctx, sandbox.CreateRequest{
		SourceVM: sourceVM,
		AgentID:  mcpAgentID,
		VCPUs:    cpu,
		MemoryMB: memoryMB,
		Live:     live,
	})
	if err != nil {
		s.logger.Error("create_sandbox failed", "error", err, "source_vm", sourceVM)
		return errorResult(map[string]any{"source_vm": sourceVM, "error": fmt.Sprintf("create sandbox: %s", err)})
	}

	result := map[string]any{
		"sandbox_id": sb.ID,
		"name":       sb.Name,
		"state":      sb.State,
	}
	if sb.IPAddress != "" {
		result["ip"] = sb.IPAddress
	}
	return jsonResult(result)
}

func (s *Server) handleDestroySandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("destroy_sandbox")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	id := request.GetString("sandbox_id", "")
	if id == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}

	err := s.service.DestroySandbox(ctx, id)
	if err != nil {
		s.logger.Error("destroy_sandbox failed", "error", err, "sandbox_id", id)
		return errorResult(map[string]any{"sandbox_id": id, "error": fmt.Sprintf("destroy sandbox: %s", err)})
	}

	return jsonResult(map[string]any{
		"destroyed":  true,
		"sandbox_id": id,
	})
}

func (s *Server) handleRunCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("run_command")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	sandboxID := request.GetString("sandbox_id", "")
	command := request.GetString("command", "")
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	timeoutSec := request.GetInt("timeout_seconds", 0)

	result, err := s.service.RunCommand(ctx, sandboxID, command, timeoutSec, nil)
	if err != nil {
		s.logger.Error("run_command failed", "error", err, "sandbox_id", sandboxID, "command", command)
		resp := map[string]any{
			"sandbox_id": sandboxID,
			"command":    command,
			"error":      fmt.Sprintf("run command: %s", err),
		}
		if result != nil {
			resp["exit_code"] = result.ExitCode
			resp["stdout"] = result.Stdout
			resp["stderr"] = result.Stderr
		}
		return errorResult(resp)
	}

	return jsonResult(map[string]any{
		"sandbox_id": sandboxID,
		"exit_code":  result.ExitCode,
		"stdout":     result.Stdout,
		"stderr":     result.Stderr,
	})
}

func (s *Server) handleStartSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("start_sandbox")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	id := request.GetString("sandbox_id", "")
	if id == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}

	sb, err := s.service.StartSandbox(ctx, id)
	if err != nil {
		s.logger.Error("start_sandbox failed", "error", err, "sandbox_id", id)
		return errorResult(map[string]any{"sandbox_id": id, "error": fmt.Sprintf("start sandbox: %s", err)})
	}

	result := map[string]any{
		"started":    true,
		"sandbox_id": id,
	}
	if sb.IPAddress != "" {
		result["ip"] = sb.IPAddress
	}
	return jsonResult(result)
}

func (s *Server) handleStopSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("stop_sandbox")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	id := request.GetString("sandbox_id", "")
	if id == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}

	err := s.service.StopSandbox(ctx, id, false)
	if err != nil {
		s.logger.Error("stop_sandbox failed", "error", err, "sandbox_id", id)
		return errorResult(map[string]any{"sandbox_id": id, "error": fmt.Sprintf("stop sandbox: %s", err)})
	}

	return jsonResult(map[string]any{
		"stopped":    true,
		"sandbox_id": id,
	})
}

func (s *Server) handleGetSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("get_sandbox")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	id := request.GetString("sandbox_id", "")
	if id == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}

	sb, err := s.service.GetSandbox(ctx, id)
	if err != nil {
		s.logger.Error("get_sandbox failed", "error", err, "sandbox_id", id)
		return errorResult(map[string]any{"sandbox_id": id, "error": fmt.Sprintf("get sandbox: %s", err)})
	}

	result := map[string]any{
		"sandbox_id": sb.ID,
		"name":       sb.Name,
		"state":      sb.State,
		"base_image": sb.BaseImage,
		"agent_id":   sb.AgentID,
		"created_at": sb.CreatedAt.Format(time.RFC3339),
	}
	if sb.IPAddress != "" {
		result["ip"] = sb.IPAddress
	}

	return jsonResult(result)
}

func (s *Server) handleListVMs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("list_vms")

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	vms, err := s.service.ListVMs(ctx)
	if err != nil {
		s.logger.Error("list_vms failed", "error", err)
		return errorResult(map[string]any{"error": fmt.Sprintf("list vms: %s", err)})
	}

	result := make([]map[string]any, 0, len(vms))
	for _, vm := range vms {
		item := map[string]any{
			"name":     vm.Name,
			"state":    vm.State,
			"prepared": vm.Prepared,
		}
		if vm.IPAddress != "" {
			item["ip"] = vm.IPAddress
		}
		result = append(result, item)
	}

	return jsonResult(map[string]any{
		"vms":   result,
		"count": len(result),
		"total": len(result),
	})
}

func (s *Server) handleCreateSnapshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("create_snapshot")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	sandboxID := request.GetString("sandbox_id", "")
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}
	name := request.GetString("name", "")
	if name == "" {
		name = fmt.Sprintf("snap-%d", time.Now().Unix())
	}

	snap, err := s.service.CreateSnapshot(ctx, sandboxID, name)
	if err != nil {
		s.logger.Error("create_snapshot failed", "error", err, "sandbox_id", sandboxID)
		return errorResult(map[string]any{"sandbox_id": sandboxID, "error": fmt.Sprintf("create snapshot: %s", err)})
	}

	return jsonResult(map[string]any{
		"snapshot_id": snap.SnapshotID,
		"sandbox_id":  sandboxID,
		"name":        snap.SnapshotName,
	})
}

func (s *Server) handleCreatePlaybook(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("create_playbook")

	name := request.GetString("name", "")
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	hosts := request.GetString("hosts", "")
	become := request.GetBool("become", false)

	pb, err := s.playbookService.CreatePlaybook(ctx, ansible.CreatePlaybookRequest{
		Name:   name,
		Hosts:  hosts,
		Become: become,
	})
	if err != nil {
		s.logger.Error("create_playbook failed", "error", err, "name", name)
		return errorResult(map[string]any{"name": name, "error": fmt.Sprintf("create playbook: %s", err)})
	}

	result := map[string]any{
		"id":         pb.ID,
		"name":       pb.Name,
		"hosts":      pb.Hosts,
		"become":     pb.Become,
		"created_at": pb.CreatedAt.Format(time.RFC3339),
	}
	if pb.FilePath != nil {
		result["file_path"] = *pb.FilePath
	}
	return jsonResult(result)
}

func (s *Server) handleAddPlaybookTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("add_playbook_task")

	playbookID := request.GetString("playbook_id", "")
	if playbookID == "" {
		return nil, fmt.Errorf("playbook_id is required")
	}
	name := request.GetString("name", "")
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	module := request.GetString("module", "")
	if module == "" {
		return nil, fmt.Errorf("module is required")
	}

	args := request.GetArguments()
	var params map[string]any
	if p, ok := args["params"]; ok {
		if m, ok := p.(map[string]any); ok {
			params = m
		}
	}

	task, err := s.playbookService.AddTask(ctx, playbookID, ansible.AddTaskRequest{
		Name:   name,
		Module: module,
		Params: params,
	})
	if err != nil {
		s.logger.Error("add_playbook_task failed", "error", err, "playbook_id", playbookID)
		return errorResult(map[string]any{"playbook_id": playbookID, "error": fmt.Sprintf("add playbook task: %s", err)})
	}

	return jsonResult(map[string]any{
		"id":          task.ID,
		"playbook_id": playbookID,
		"name":        task.Name,
		"module":      task.Module,
		"position":    task.Position,
	})
}

func (s *Server) handleEditFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("edit_file")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	sandboxID := request.GetString("sandbox_id", "")
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}
	path, err := validateFilePath(request.GetString("path", ""))
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	oldStr := request.GetString("old_str", "")
	newStr := request.GetString("new_str", "")

	escapedPath, err := shellEscape(path)
	if err != nil {
		s.logger.Error("edit_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
		return errorResult(map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("invalid path: %s", err)})
	}

	if oldStr == "" {
		// Create/overwrite file
		if err := checkFileSize(int64(len(newStr))); err != nil {
			s.logger.Error("edit_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
			return errorResult(map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("file too large: %s", err)})
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(newStr))
		cmd := fmt.Sprintf("base64 -d > %s << '--FLUID_B64--'\n%s\n--FLUID_B64--", escapedPath, encoded)
		result, err := s.service.RunCommand(ctx, sandboxID, cmd, 0, nil)
		if err != nil {
			s.logger.Error("edit_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
			resp := map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("create file: %s", err)}
			if result != nil {
				resp["exit_code"] = result.ExitCode
				resp["stderr"] = result.Stderr
			}
			return errorResult(resp)
		}
		if result.ExitCode != 0 {
			s.logger.Error("edit_file failed", "error", fmt.Sprintf("exit code %d", result.ExitCode), "sandbox_id", sandboxID, "path", path)
			return errorResult(map[string]any{
				"sandbox_id": sandboxID, "path": path,
				"exit_code": result.ExitCode, "stderr": result.Stderr,
				"error": fmt.Sprintf("create file failed with exit code %d", result.ExitCode),
			})
		}
		return jsonResult(map[string]any{
			"sandbox_id": sandboxID,
			"path":       path,
			"action":     "created_file",
		})
	}

	// Read existing file
	readResult, err := s.service.RunCommand(ctx, sandboxID, fmt.Sprintf("base64 %s", escapedPath), 0, nil)
	if err != nil {
		s.logger.Error("edit_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
		resp := map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("read file for edit: %s", err)}
		if readResult != nil {
			resp["exit_code"] = readResult.ExitCode
			resp["stderr"] = readResult.Stderr
		}
		return errorResult(resp)
	}
	if readResult.ExitCode != 0 {
		s.logger.Error("edit_file failed", "error", fmt.Sprintf("exit code %d", readResult.ExitCode), "sandbox_id", sandboxID, "path", path)
		return errorResult(map[string]any{
			"sandbox_id": sandboxID, "path": path,
			"exit_code": readResult.ExitCode, "stderr": readResult.Stderr,
			"error": fmt.Sprintf("read file failed with exit code %d", readResult.ExitCode),
		})
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(readResult.Stdout))
	if err != nil {
		s.logger.Error("edit_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
		return errorResult(map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("decode file content: %s", err)})
	}
	original := string(decoded)

	if !strings.Contains(original, oldStr) {
		return jsonResult(map[string]any{
			"sandbox_id": sandboxID,
			"path":       path,
			"action":     "old_str_not_found",
		})
	}

	replaceAll := request.GetBool("replace_all", false)
	n := 1
	if replaceAll {
		n = -1
	}
	edited := strings.Replace(original, oldStr, newStr, n)
	if err := checkFileSize(int64(len(edited))); err != nil {
		s.logger.Error("edit_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
		return errorResult(map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("edited file too large: %s", err)})
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(edited))
	writeCmd := fmt.Sprintf("base64 -d > %s << '--FLUID_B64--'\n%s\n--FLUID_B64--", escapedPath, encoded)
	writeResult, err := s.service.RunCommand(ctx, sandboxID, writeCmd, 0, nil)
	if err != nil {
		s.logger.Error("edit_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
		resp := map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("write file: %s", err)}
		if writeResult != nil {
			resp["exit_code"] = writeResult.ExitCode
			resp["stderr"] = writeResult.Stderr
		}
		return errorResult(resp)
	}
	if writeResult.ExitCode != 0 {
		s.logger.Error("edit_file failed", "error", fmt.Sprintf("exit code %d", writeResult.ExitCode), "sandbox_id", sandboxID, "path", path)
		return errorResult(map[string]any{
			"sandbox_id": sandboxID, "path": path,
			"exit_code": writeResult.ExitCode, "stderr": writeResult.Stderr,
			"error": fmt.Sprintf("write file failed with exit code %d", writeResult.ExitCode),
		})
	}

	return jsonResult(map[string]any{
		"sandbox_id": sandboxID,
		"path":       path,
		"action":     "edited",
	})
}

func (s *Server) handleReadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("read_file")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	sandboxID := request.GetString("sandbox_id", "")
	if sandboxID == "" {
		return nil, fmt.Errorf("sandbox_id is required")
	}
	path, err := validateFilePath(request.GetString("path", ""))
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	escapedPath, err := shellEscape(path)
	if err != nil {
		s.logger.Error("read_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
		return errorResult(map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("invalid path: %s", err)})
	}
	result, err := s.service.RunCommand(ctx, sandboxID, fmt.Sprintf("base64 %s", escapedPath), 0, nil)
	if err != nil {
		s.logger.Error("read_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
		resp := map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("read file: %s", err)}
		if result != nil {
			resp["exit_code"] = result.ExitCode
			resp["stderr"] = result.Stderr
		}
		return errorResult(resp)
	}
	if result.ExitCode != 0 {
		s.logger.Error("read_file failed", "error", fmt.Sprintf("exit code %d", result.ExitCode), "sandbox_id", sandboxID, "path", path)
		return errorResult(map[string]any{
			"sandbox_id": sandboxID, "path": path,
			"exit_code": result.ExitCode, "stderr": result.Stderr,
			"error": fmt.Sprintf("read file failed with exit code %d", result.ExitCode),
		})
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(result.Stdout))
	if err != nil {
		s.logger.Error("read_file failed", "error", err, "sandbox_id", sandboxID, "path", path)
		return errorResult(map[string]any{"sandbox_id": sandboxID, "path": path, "error": fmt.Sprintf("decode file content: %s", err)})
	}

	return jsonResult(map[string]any{
		"sandbox_id": sandboxID,
		"path":       path,
		"content":    string(decoded),
	})
}

func (s *Server) handleListPlaybooks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("list_playbooks")

	playbooks, err := s.playbookService.ListPlaybooks(ctx, nil)
	if err != nil {
		s.logger.Error("list_playbooks failed", "error", err)
		return errorResult(map[string]any{"error": fmt.Sprintf("list playbooks: %s", err)})
	}

	result := make([]map[string]any, 0, len(playbooks))
	for _, pb := range playbooks {
		path := ""
		if pb.FilePath != nil && *pb.FilePath != "" {
			path = *pb.FilePath
		} else if s.cfg.Ansible.PlaybooksDir != "" {
			path = filepath.Join(s.cfg.Ansible.PlaybooksDir, pb.Name+".yml")
		}
		result = append(result, map[string]any{
			"id":         pb.ID,
			"name":       pb.Name,
			"path":       path,
			"created_at": pb.CreatedAt.Format(time.RFC3339),
		})
	}

	return jsonResult(map[string]any{
		"playbooks": result,
		"count":     len(result),
	})
}

func (s *Server) handleGetPlaybook(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("get_playbook")

	playbookID := request.GetString("playbook_id", "")
	if playbookID == "" {
		return nil, fmt.Errorf("playbook_id is required")
	}

	pbWithTasks, err := s.playbookService.GetPlaybookWithTasks(ctx, playbookID)
	if err != nil {
		s.logger.Error("get_playbook failed", "error", err, "playbook_id", playbookID)
		return errorResult(map[string]any{"playbook_id": playbookID, "error": fmt.Sprintf("get playbook: %s", err)})
	}

	yamlContent, err := s.playbookService.ExportPlaybook(ctx, playbookID)
	if err != nil {
		s.logger.Error("get_playbook failed", "error", err, "playbook_id", playbookID)
		return errorResult(map[string]any{"playbook_id": playbookID, "error": fmt.Sprintf("export playbook: %s", err)})
	}

	tasks := make([]map[string]any, 0, len(pbWithTasks.Tasks))
	for _, t := range pbWithTasks.Tasks {
		tasks = append(tasks, map[string]any{
			"id":       t.ID,
			"position": t.Position,
			"name":     t.Name,
			"module":   t.Module,
			"params":   t.Params,
		})
	}

	result := map[string]any{
		"id":           pbWithTasks.Playbook.ID,
		"name":         pbWithTasks.Playbook.Name,
		"hosts":        pbWithTasks.Playbook.Hosts,
		"become":       pbWithTasks.Playbook.Become,
		"tasks":        tasks,
		"yaml_content": string(yamlContent),
		"created_at":   pbWithTasks.Playbook.CreatedAt.Format(time.RFC3339),
	}
	if pbWithTasks.Playbook.FilePath != nil {
		result["file_path"] = *pbWithTasks.Playbook.FilePath
	}

	return jsonResult(result)
}

func (s *Server) handleRunSourceCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("run_source_command")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	sourceVM := request.GetString("source_vm", "")
	if sourceVM == "" {
		return nil, fmt.Errorf("source_vm is required")
	}
	command := request.GetString("command", "")
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	timeoutSec := request.GetInt("timeout_seconds", 0)

	result, err := s.service.RunSourceCommand(ctx, sourceVM, command, timeoutSec)
	if err != nil {
		s.logger.Error("run_source_command failed", "error", err, "source_vm", sourceVM, "command", command)
		resp := map[string]any{
			"source_vm": sourceVM,
			"command":   command,
			"error":     fmt.Sprintf("run source command: %s", err),
		}
		if result != nil {
			resp["exit_code"] = result.ExitCode
			resp["stdout"] = result.Stdout
			resp["stderr"] = result.Stderr
		}
		return errorResult(resp)
	}

	return jsonResult(map[string]any{
		"source_vm": sourceVM,
		"exit_code": result.ExitCode,
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
	})
}

func (s *Server) handleReadSourceFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.trackToolCall("read_source_file")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	sourceVM := request.GetString("source_vm", "")
	if sourceVM == "" {
		return nil, fmt.Errorf("source_vm is required")
	}
	path, err := validateFilePath(request.GetString("path", ""))
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	content, err := s.service.ReadSourceFile(ctx, sourceVM, path)
	if err != nil {
		s.logger.Error("read_source_file failed", "error", err, "source_vm", sourceVM, "path", path)
		return errorResult(map[string]any{"source_vm": sourceVM, "path": path, "error": fmt.Sprintf("read source file: %s", err)})
	}

	return jsonResult(map[string]any{
		"source_vm": sourceVM,
		"path":      path,
		"content":   content,
	})
}
