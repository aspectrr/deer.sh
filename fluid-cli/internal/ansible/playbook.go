package ansible

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/aspectrr/fluid.sh/fluid/internal/store"
)

// PlaybookService manages Ansible playbook creation and rendering.
type PlaybookService struct {
	store        store.DataStore
	playbooksDir string
}

// NewPlaybookService creates a new PlaybookService.
func NewPlaybookService(st store.DataStore, playbooksDir string) *PlaybookService {
	return &PlaybookService{
		store:        st,
		playbooksDir: playbooksDir,
	}
}

// PlaybookDir returns the configured playbooks directory.
func (s *PlaybookService) PlaybookDir() string {
	return s.playbooksDir
}

// CreatePlaybookRequest contains parameters for creating a new playbook.
type CreatePlaybookRequest struct {
	Name   string `json:"name"`
	Hosts  string `json:"hosts"`
	Become bool   `json:"become"`
}

// CreatePlaybook creates a new playbook in the database.
func (s *PlaybookService) CreatePlaybook(ctx context.Context, req CreatePlaybookRequest) (*store.Playbook, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Hosts == "" {
		req.Hosts = "all"
	}

	pb := &store.Playbook{
		ID:     uuid.New().String(),
		Name:   req.Name,
		Hosts:  req.Hosts,
		Become: req.Become,
	}

	if err := s.store.CreatePlaybook(ctx, pb); err != nil {
		return nil, fmt.Errorf("create playbook: %w", err)
	}

	return pb, nil
}

// GetPlaybook retrieves a playbook by ID.
func (s *PlaybookService) GetPlaybook(ctx context.Context, id string) (*store.Playbook, error) {
	return s.store.GetPlaybook(ctx, id)
}

// GetPlaybookByName retrieves a playbook by name.
func (s *PlaybookService) GetPlaybookByName(ctx context.Context, name string) (*store.Playbook, error) {
	return s.store.GetPlaybookByName(ctx, name)
}

// ListPlaybooks lists all playbooks.
func (s *PlaybookService) ListPlaybooks(ctx context.Context, opt *store.ListOptions) ([]*store.Playbook, error) {
	return s.store.ListPlaybooks(ctx, opt)
}

// DeletePlaybook deletes a playbook and its tasks.
func (s *PlaybookService) DeletePlaybook(ctx context.Context, id string) error {
	// Get playbook to find file path
	pb, err := s.store.GetPlaybook(ctx, id)
	if err != nil {
		return err
	}

	// Delete from database
	if err := s.store.DeletePlaybook(ctx, id); err != nil {
		return err
	}

	// Remove rendered file if it exists
	if pb.FilePath != nil && *pb.FilePath != "" {
		_ = os.Remove(*pb.FilePath)
	}

	return nil
}

// AddTaskRequest contains parameters for adding a task to a playbook.
type AddTaskRequest struct {
	Name   string         `json:"name"`
	Module string         `json:"module"`
	Params map[string]any `json:"params"`
}

// AddTask adds a task to an existing playbook and re-renders the YAML.
func (s *PlaybookService) AddTask(ctx context.Context, playbookID string, req AddTaskRequest) (*store.PlaybookTask, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("task name is required")
	}
	if req.Module == "" {
		return nil, fmt.Errorf("module is required")
	}

	// Get next position
	pos, err := s.store.GetNextTaskPosition(ctx, playbookID)
	if err != nil {
		return nil, fmt.Errorf("get next position: %w", err)
	}

	task := &store.PlaybookTask{
		ID:         uuid.New().String(),
		PlaybookID: playbookID,
		Position:   pos,
		Name:       req.Name,
		Module:     req.Module,
		Params:     req.Params,
	}

	if err := s.store.CreatePlaybookTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Re-render playbook to disk
	if err := s.RenderPlaybook(ctx, playbookID); err != nil {
		return nil, fmt.Errorf("render playbook: %w", err)
	}

	return task, nil
}

// GetTask retrieves a task by ID.
func (s *PlaybookService) GetTask(ctx context.Context, id string) (*store.PlaybookTask, error) {
	return s.store.GetPlaybookTask(ctx, id)
}

// ListTasks lists all tasks for a playbook.
func (s *PlaybookService) ListTasks(ctx context.Context, playbookID string) ([]*store.PlaybookTask, error) {
	return s.store.ListPlaybookTasks(ctx, playbookID, nil)
}

// UpdateTaskRequest contains parameters for updating a task.
type UpdateTaskRequest struct {
	Name   *string        `json:"name,omitempty"`
	Module *string        `json:"module,omitempty"`
	Params map[string]any `json:"params,omitempty"`
}

// UpdateTask updates an existing task and re-renders the playbook.
func (s *PlaybookService) UpdateTask(ctx context.Context, taskID string, req UpdateTaskRequest) (*store.PlaybookTask, error) {
	task, err := s.store.GetPlaybookTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Module != nil {
		task.Module = *req.Module
	}
	if req.Params != nil {
		task.Params = req.Params
	}

	if err := s.store.UpdatePlaybookTask(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	// Re-render playbook
	if err := s.RenderPlaybook(ctx, task.PlaybookID); err != nil {
		return nil, fmt.Errorf("render playbook: %w", err)
	}

	return task, nil
}

// DeleteTask removes a task from a playbook and re-renders.
func (s *PlaybookService) DeleteTask(ctx context.Context, taskID string) error {
	task, err := s.store.GetPlaybookTask(ctx, taskID)
	if err != nil {
		return err
	}
	playbookID := task.PlaybookID

	if err := s.store.DeletePlaybookTask(ctx, taskID); err != nil {
		return err
	}

	// Re-render playbook
	return s.RenderPlaybook(ctx, playbookID)
}

// ReorderTasksRequest contains the new task order.
type ReorderTasksRequest struct {
	TaskIDs []string `json:"task_ids"`
}

// ReorderTasks reorders tasks in a playbook and re-renders.
func (s *PlaybookService) ReorderTasks(ctx context.Context, playbookID string, taskIDs []string) error {
	if err := s.store.ReorderPlaybookTasks(ctx, playbookID, taskIDs); err != nil {
		return err
	}
	return s.RenderPlaybook(ctx, playbookID)
}

// RenderPlaybook generates the YAML file from the database state.
func (s *PlaybookService) RenderPlaybook(ctx context.Context, playbookID string) error {
	pb, err := s.store.GetPlaybook(ctx, playbookID)
	if err != nil {
		return fmt.Errorf("get playbook: %w", err)
	}

	tasks, err := s.store.ListPlaybookTasks(ctx, playbookID, nil)
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	yamlContent, err := s.renderYAML(pb, tasks)
	if err != nil {
		return fmt.Errorf("render yaml: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(s.playbooksDir, 0o750); err != nil {
		return fmt.Errorf("create playbooks dir: %w", err)
	}

	// Write to file
	filePath := filepath.Join(s.playbooksDir, pb.Name+".yml")
	if err := os.WriteFile(filePath, yamlContent, 0o640); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	// Update playbook with file path
	pb.FilePath = &filePath
	if err := s.store.UpdatePlaybook(ctx, pb); err != nil {
		return fmt.Errorf("update playbook path: %w", err)
	}

	return nil
}

// ExportPlaybook returns the YAML content without writing to disk.
func (s *PlaybookService) ExportPlaybook(ctx context.Context, playbookID string) ([]byte, error) {
	pb, err := s.store.GetPlaybook(ctx, playbookID)
	if err != nil {
		return nil, fmt.Errorf("get playbook: %w", err)
	}

	tasks, err := s.store.ListPlaybookTasks(ctx, playbookID, nil)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return s.renderYAML(pb, tasks)
}

// ansiblePlay represents a single play in an Ansible playbook.
type ansiblePlay struct {
	Name   string        `yaml:"name"`
	Hosts  string        `yaml:"hosts"`
	Become bool          `yaml:"become,omitempty"`
	Tasks  []ansibleTask `yaml:"tasks"`
}

// ansibleTask represents a task in YAML format.
type ansibleTask map[string]any

// renderYAML converts playbook and tasks to Ansible YAML format.
func (s *PlaybookService) renderYAML(pb *store.Playbook, tasks []*store.PlaybookTask) ([]byte, error) {
	ansibleTasks := make([]ansibleTask, 0, len(tasks))
	for _, t := range tasks {
		task := ansibleTask{
			"name": t.Name,
		}
		// Add module with its params
		if len(t.Params) > 0 {
			task[t.Module] = t.Params
		} else {
			task[t.Module] = nil
		}
		ansibleTasks = append(ansibleTasks, task)
	}

	play := ansiblePlay{
		Name:   pb.Name,
		Hosts:  pb.Hosts,
		Become: pb.Become,
		Tasks:  ansibleTasks,
	}

	// Ansible playbook is a list of plays
	playbook := []ansiblePlay{play}

	return yaml.Marshal(playbook)
}

// PlaybookWithTasks combines a playbook with its tasks for API responses.
type PlaybookWithTasks struct {
	Playbook *store.Playbook       `json:"playbook"`
	Tasks    []*store.PlaybookTask `json:"tasks"`
}

// GetPlaybookWithTasks retrieves a playbook along with all its tasks.
func (s *PlaybookService) GetPlaybookWithTasks(ctx context.Context, playbookID string) (*PlaybookWithTasks, error) {
	pb, err := s.store.GetPlaybook(ctx, playbookID)
	if err != nil {
		return nil, err
	}

	tasks, err := s.store.ListPlaybookTasks(ctx, playbookID, nil)
	if err != nil {
		return nil, err
	}

	return &PlaybookWithTasks{
		Playbook: pb,
		Tasks:    tasks,
	}, nil
}

// GetPlaybookWithTasksByName retrieves a playbook by name along with all its tasks.
func (s *PlaybookService) GetPlaybookWithTasksByName(ctx context.Context, name string) (*PlaybookWithTasks, error) {
	pb, err := s.store.GetPlaybookByName(ctx, name)
	if err != nil {
		return nil, err
	}

	tasks, err := s.store.ListPlaybookTasks(ctx, pb.ID, nil)
	if err != nil {
		return nil, err
	}

	return &PlaybookWithTasks{
		Playbook: pb,
		Tasks:    tasks,
	}, nil
}
