package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aspectrr/fluid.sh/api/internal/orchestrator"
	"github.com/aspectrr/fluid.sh/api/internal/store"
	"github.com/google/uuid"
)

// ExecuteTool dispatches a tool call to the appropriate handler and returns the JSON result.
func (c *Client) ExecuteTool(ctx context.Context, orgID, name string, args json.RawMessage) (string, error) {
	var params map[string]any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid tool arguments: %w", err)
		}
	}
	if params == nil {
		params = map[string]any{}
	}

	switch name {
	// Sandbox tools
	case "create_sandbox":
		return c.execCreateSandbox(ctx, orgID, params)
	case "list_sandboxes":
		return c.execListSandboxes(ctx, orgID)
	case "get_sandbox":
		return c.execGetSandbox(ctx, params)
	case "destroy_sandbox":
		return c.execDestroySandbox(ctx, params)
	case "start_sandbox":
		return c.execStartSandbox(ctx, params)
	case "stop_sandbox":
		return c.execStopSandbox(ctx, params)
	case "run_command":
		return c.execRunCommand(ctx, params)
	case "get_sandbox_ip":
		return c.execGetSandboxIP(ctx, params)
	case "create_snapshot":
		return c.execCreateSnapshot(ctx, params)
	case "list_commands":
		return c.execListCommands(ctx, params)

	// Source VM tools
	case "list_vms":
		return c.execListVMs(ctx)
	case "prepare_source_vm":
		return c.execPrepareSourceVM(ctx, params)
	case "run_source_command":
		return c.execRunSourceCommand(ctx, params)
	case "read_source_file":
		return c.execReadSourceFile(ctx, params)

	// Host tools
	case "list_hosts":
		return c.execListHosts(ctx)
	case "get_host":
		return c.execGetHost(ctx, params)

	// Playbook tools
	case "create_playbook":
		return c.execCreatePlaybook(ctx, orgID, params)
	case "update_playbook":
		return c.execUpdatePlaybook(ctx, params)
	case "delete_playbook":
		return c.execDeletePlaybook(ctx, params)
	case "list_playbooks":
		return c.execListPlaybooks(ctx, orgID)
	case "add_playbook_task":
		return c.execAddPlaybookTask(ctx, params)
	case "update_playbook_task":
		return c.execUpdatePlaybookTask(ctx, params)
	case "delete_playbook_task":
		return c.execDeletePlaybookTask(ctx, params)
	case "reorder_playbook_tasks":
		return c.execReorderPlaybookTasks(ctx, params)

	default:
		return jsonResult(map[string]string{"error": "unknown tool: " + name})
	}
}

// --- Sandbox tool executors ---

func (c *Client) execCreateSandbox(ctx context.Context, orgID string, params map[string]any) (string, error) {
	req := orchestrator.CreateSandboxRequest{
		OrgID:     orgID,
		SourceVM:  strParam(params, "source_vm"),
		BaseImage: strParam(params, "base_image"),
		Name:      strParam(params, "name"),
		VCPUs:     intParam(params, "vcpus"),
		MemoryMB:  intParam(params, "memory_mb"),
	}
	sandbox, err := c.orchestrator.CreateSandbox(ctx, req)
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(sandbox)
}

func (c *Client) execListSandboxes(ctx context.Context, orgID string) (string, error) {
	sandboxes, err := c.orchestrator.ListSandboxesByOrg(ctx, orgID)
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"sandboxes": sandboxes, "count": len(sandboxes)})
}

func (c *Client) execGetSandbox(ctx context.Context, params map[string]any) (string, error) {
	sandbox, err := c.orchestrator.GetSandbox(ctx, strParam(params, "sandbox_id"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(sandbox)
}

func (c *Client) execDestroySandbox(ctx context.Context, params map[string]any) (string, error) {
	err := c.orchestrator.DestroySandbox(ctx, strParam(params, "sandbox_id"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"destroyed": true, "sandbox_id": strParam(params, "sandbox_id")})
}

func (c *Client) execStartSandbox(ctx context.Context, params map[string]any) (string, error) {
	err := c.orchestrator.StartSandbox(ctx, strParam(params, "sandbox_id"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"started": true, "sandbox_id": strParam(params, "sandbox_id")})
}

func (c *Client) execStopSandbox(ctx context.Context, params map[string]any) (string, error) {
	err := c.orchestrator.StopSandbox(ctx, strParam(params, "sandbox_id"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"stopped": true, "sandbox_id": strParam(params, "sandbox_id")})
}

func (c *Client) execRunCommand(ctx context.Context, params map[string]any) (string, error) {
	result, err := c.orchestrator.RunCommand(ctx, strParam(params, "sandbox_id"), strParam(params, "command"), intParam(params, "timeout_seconds"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(result)
}

func (c *Client) execGetSandboxIP(ctx context.Context, params map[string]any) (string, error) {
	sandbox, err := c.orchestrator.GetSandbox(ctx, strParam(params, "sandbox_id"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"sandbox_id": sandbox.ID, "ip_address": sandbox.IPAddress})
}

func (c *Client) execCreateSnapshot(ctx context.Context, params map[string]any) (string, error) {
	result, err := c.orchestrator.CreateSnapshot(ctx, strParam(params, "sandbox_id"), strParam(params, "name"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(result)
}

func (c *Client) execListCommands(ctx context.Context, params map[string]any) (string, error) {
	commands, err := c.orchestrator.ListCommands(ctx, strParam(params, "sandbox_id"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"commands": commands, "count": len(commands)})
}

// --- Source VM tool executors ---

func (c *Client) execListVMs(ctx context.Context) (string, error) {
	vms, err := c.orchestrator.ListVMs(ctx)
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"vms": vms, "count": len(vms)})
}

func (c *Client) execPrepareSourceVM(ctx context.Context, params map[string]any) (string, error) {
	result, err := c.orchestrator.PrepareSourceVM(ctx, strParam(params, "vm_name"), strParam(params, "ssh_user"), strParam(params, "key_path"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(result)
}

func (c *Client) execRunSourceCommand(ctx context.Context, params map[string]any) (string, error) {
	result, err := c.orchestrator.RunSourceCommand(ctx, strParam(params, "vm_name"), strParam(params, "command"), intParam(params, "timeout_seconds"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(result)
}

func (c *Client) execReadSourceFile(ctx context.Context, params map[string]any) (string, error) {
	result, err := c.orchestrator.ReadSourceFile(ctx, strParam(params, "vm_name"), strParam(params, "path"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(result)
}

// --- Host tool executors ---

func (c *Client) execListHosts(ctx context.Context) (string, error) {
	hosts, err := c.orchestrator.ListHosts(ctx)
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"hosts": hosts, "count": len(hosts)})
}

func (c *Client) execGetHost(ctx context.Context, params map[string]any) (string, error) {
	host, err := c.orchestrator.GetHost(ctx, strParam(params, "host_id"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(host)
}

// --- Playbook tool executors ---

func (c *Client) execCreatePlaybook(ctx context.Context, orgID string, params map[string]any) (string, error) {
	pb := &store.Playbook{
		ID:          uuid.New().String(),
		OrgID:       orgID,
		Name:        strParam(params, "name"),
		Description: strParam(params, "description"),
	}
	if err := c.store.CreatePlaybook(ctx, pb); err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(pb)
}

func (c *Client) execUpdatePlaybook(ctx context.Context, params map[string]any) (string, error) {
	pb, err := c.store.GetPlaybook(ctx, strParam(params, "playbook_id"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	if name := strParam(params, "name"); name != "" {
		pb.Name = name
	}
	if desc := strParam(params, "description"); desc != "" {
		pb.Description = desc
	}
	if err := c.store.UpdatePlaybook(ctx, pb); err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(pb)
}

func (c *Client) execDeletePlaybook(ctx context.Context, params map[string]any) (string, error) {
	if err := c.store.DeletePlaybook(ctx, strParam(params, "playbook_id")); err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"deleted": true, "playbook_id": strParam(params, "playbook_id")})
}

func (c *Client) execListPlaybooks(ctx context.Context, orgID string) (string, error) {
	playbooks, err := c.store.ListPlaybooksByOrg(ctx, orgID)
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"playbooks": playbooks, "count": len(playbooks)})
}

func (c *Client) execAddPlaybookTask(ctx context.Context, params map[string]any) (string, error) {
	paramsJSON := "{}"
	if p, ok := params["params"]; ok {
		b, _ := json.Marshal(p)
		paramsJSON = string(b)
	}

	// Get current max sort order
	tasks, _ := c.store.ListPlaybookTasks(ctx, strParam(params, "playbook_id"))
	sortOrder := len(tasks)

	task := &store.PlaybookTask{
		ID:         uuid.New().String(),
		PlaybookID: strParam(params, "playbook_id"),
		SortOrder:  sortOrder,
		Name:       strParam(params, "name"),
		Module:     strParam(params, "module"),
		Params:     paramsJSON,
	}
	if err := c.store.CreatePlaybookTask(ctx, task); err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(task)
}

func (c *Client) execUpdatePlaybookTask(ctx context.Context, params map[string]any) (string, error) {
	task, err := c.store.GetPlaybookTask(ctx, strParam(params, "task_id"))
	if err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	if name := strParam(params, "name"); name != "" {
		task.Name = name
	}
	if module := strParam(params, "module"); module != "" {
		task.Module = module
	}
	if p, ok := params["params"]; ok {
		b, _ := json.Marshal(p)
		task.Params = string(b)
	}
	if err := c.store.UpdatePlaybookTask(ctx, task); err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(task)
}

func (c *Client) execDeletePlaybookTask(ctx context.Context, params map[string]any) (string, error) {
	if err := c.store.DeletePlaybookTask(ctx, strParam(params, "task_id")); err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"deleted": true, "task_id": strParam(params, "task_id")})
}

func (c *Client) execReorderPlaybookTasks(ctx context.Context, params map[string]any) (string, error) {
	var taskIDs []string
	if ids, ok := params["task_ids"]; ok {
		if arr, ok := ids.([]any); ok {
			for _, id := range arr {
				if s, ok := id.(string); ok {
					taskIDs = append(taskIDs, s)
				}
			}
		}
	}
	if err := c.store.ReorderPlaybookTasks(ctx, strParam(params, "playbook_id"), taskIDs); err != nil {
		return jsonResult(map[string]string{"error": err.Error()})
	}
	return jsonResult(map[string]any{"reordered": true})
}

// --- Helpers ---

func strParam(params map[string]any, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intParam(params map[string]any, key string) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func jsonResult(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"failed to marshal result"}`, nil
	}
	return string(b), nil
}
