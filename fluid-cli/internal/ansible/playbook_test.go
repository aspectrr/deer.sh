package ansible

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aspectrr/fluid.sh/fluid/internal/store"
)

// mockStore implements store.DataStore for testing playbook operations.
type mockStore struct {
	playbooks     map[string]*store.Playbook
	playbookTasks map[string]*store.PlaybookTask
}

func newMockStore() *mockStore {
	return &mockStore{
		playbooks:     make(map[string]*store.Playbook),
		playbookTasks: make(map[string]*store.PlaybookTask),
	}
}

func (m *mockStore) CreatePlaybook(ctx context.Context, pb *store.Playbook) error {
	if _, exists := m.playbooks[pb.ID]; exists {
		return store.ErrAlreadyExists
	}
	for _, existing := range m.playbooks {
		if existing.Name == pb.Name {
			return store.ErrAlreadyExists
		}
	}
	m.playbooks[pb.ID] = pb
	return nil
}

func (m *mockStore) GetPlaybook(ctx context.Context, id string) (*store.Playbook, error) {
	pb, ok := m.playbooks[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return pb, nil
}

func (m *mockStore) GetPlaybookByName(ctx context.Context, name string) (*store.Playbook, error) {
	for _, pb := range m.playbooks {
		if pb.Name == name {
			return pb, nil
		}
	}
	return nil, store.ErrNotFound
}

func (m *mockStore) ListPlaybooks(ctx context.Context, opt *store.ListOptions) ([]*store.Playbook, error) {
	result := make([]*store.Playbook, 0, len(m.playbooks))
	for _, pb := range m.playbooks {
		result = append(result, pb)
	}
	return result, nil
}

func (m *mockStore) UpdatePlaybook(ctx context.Context, pb *store.Playbook) error {
	if _, ok := m.playbooks[pb.ID]; !ok {
		return store.ErrNotFound
	}
	m.playbooks[pb.ID] = pb
	return nil
}

func (m *mockStore) DeletePlaybook(ctx context.Context, id string) error {
	if _, ok := m.playbooks[id]; !ok {
		return store.ErrNotFound
	}
	// Delete associated tasks
	for taskID, task := range m.playbookTasks {
		if task.PlaybookID == id {
			delete(m.playbookTasks, taskID)
		}
	}
	delete(m.playbooks, id)
	return nil
}

func (m *mockStore) CreatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	if _, exists := m.playbookTasks[task.ID]; exists {
		return store.ErrAlreadyExists
	}
	m.playbookTasks[task.ID] = task
	return nil
}

func (m *mockStore) GetPlaybookTask(ctx context.Context, id string) (*store.PlaybookTask, error) {
	task, ok := m.playbookTasks[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return task, nil
}

func (m *mockStore) ListPlaybookTasks(ctx context.Context, playbookID string, opt *store.ListOptions) ([]*store.PlaybookTask, error) {
	result := make([]*store.PlaybookTask, 0)
	for _, task := range m.playbookTasks {
		if task.PlaybookID == playbookID {
			result = append(result, task)
		}
	}
	// Sort by position
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Position > result[j].Position {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

func (m *mockStore) UpdatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	if _, ok := m.playbookTasks[task.ID]; !ok {
		return store.ErrNotFound
	}
	m.playbookTasks[task.ID] = task
	return nil
}

func (m *mockStore) DeletePlaybookTask(ctx context.Context, id string) error {
	if _, ok := m.playbookTasks[id]; !ok {
		return store.ErrNotFound
	}
	delete(m.playbookTasks, id)
	return nil
}

func (m *mockStore) ReorderPlaybookTasks(ctx context.Context, playbookID string, taskIDs []string) error {
	for i, taskID := range taskIDs {
		task, ok := m.playbookTasks[taskID]
		if !ok || task.PlaybookID != playbookID {
			return store.ErrNotFound
		}
		task.Position = i
	}
	return nil
}

func (m *mockStore) GetNextTaskPosition(ctx context.Context, playbookID string) (int, error) {
	maxPos := -1
	for _, task := range m.playbookTasks {
		if task.PlaybookID == playbookID && task.Position > maxPos {
			maxPos = task.Position
		}
	}
	return maxPos + 1, nil
}

// Stub implementations for other DataStore methods
func (m *mockStore) CreateSandbox(ctx context.Context, sb *store.Sandbox) error {
	return nil
}

func (m *mockStore) GetSandbox(ctx context.Context, id string) (*store.Sandbox, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetSandboxByVMName(ctx context.Context, vmName string) (*store.Sandbox, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListSandboxes(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
	return nil, nil
}
func (m *mockStore) UpdateSandbox(ctx context.Context, sb *store.Sandbox) error { return nil }
func (m *mockStore) UpdateSandboxState(ctx context.Context, id string, newState store.SandboxState, ipAddr *string) error {
	return nil
}
func (m *mockStore) DeleteSandbox(ctx context.Context, id string) error { return nil }
func (m *mockStore) CreateSnapshot(ctx context.Context, sn *store.Snapshot) error {
	return nil
}

func (m *mockStore) GetSnapshot(ctx context.Context, id string) (*store.Snapshot, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetSnapshotByName(ctx context.Context, sandboxID, name string) (*store.Snapshot, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListSnapshots(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Snapshot, error) {
	return nil, nil
}
func (m *mockStore) SaveCommand(ctx context.Context, cmd *store.Command) error { return nil }
func (m *mockStore) GetCommand(ctx context.Context, id string) (*store.Command, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListCommands(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Command, error) {
	return nil, nil
}
func (m *mockStore) SaveDiff(ctx context.Context, d *store.Diff) error { return nil }
func (m *mockStore) GetDiff(ctx context.Context, id string) (*store.Diff, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetDiffBySnapshots(ctx context.Context, sandboxID, fromSnapshot, toSnapshot string) (*store.Diff, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) CreateChangeSet(ctx context.Context, cs *store.ChangeSet) error { return nil }
func (m *mockStore) GetChangeSet(ctx context.Context, id string) (*store.ChangeSet, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetChangeSetByJob(ctx context.Context, jobID string) (*store.ChangeSet, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) CreatePublication(ctx context.Context, p *store.Publication) error { return nil }
func (m *mockStore) UpdatePublicationStatus(ctx context.Context, id string, status store.PublicationStatus, commitSHA, prURL, errMsg *string) error {
	return nil
}

func (m *mockStore) GetPublication(ctx context.Context, id string) (*store.Publication, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetSourceVM(ctx context.Context, name string) (*store.SourceVM, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) UpsertSourceVM(ctx context.Context, svm *store.SourceVM) error {
	return nil
}

func (m *mockStore) ListSourceVMs(ctx context.Context) ([]*store.SourceVM, error) {
	return nil, nil
}

func TestCreatePlaybook(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	pb, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{
		Name:   "test-playbook",
		Hosts:  "all",
		Become: true,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, pb.ID)
	assert.Equal(t, "test-playbook", pb.Name)
	assert.Equal(t, "all", pb.Hosts)
	assert.True(t, pb.Become)
}

func TestCreatePlaybookDuplicate(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	_, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{Name: "test-playbook", Hosts: "all"})
	require.NoError(t, err)

	_, err = svc.CreatePlaybook(ctx, CreatePlaybookRequest{Name: "test-playbook", Hosts: "all"})
	assert.ErrorIs(t, err, store.ErrAlreadyExists)
}

func TestAddTask(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	pb, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{
		Name:  "test-playbook",
		Hosts: "all",
	})
	require.NoError(t, err)

	task, err := svc.AddTask(ctx, pb.ID, AddTaskRequest{
		Name:   "Install nginx",
		Module: "apt",
		Params: map[string]any{"name": "nginx", "state": "present"},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "Install nginx", task.Name)
	assert.Equal(t, "apt", task.Module)
	assert.Equal(t, 0, task.Position)
}

func TestAddMultipleTasks(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	pb, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{Name: "test-playbook", Hosts: "all"})
	require.NoError(t, err)

	task1, err := svc.AddTask(ctx, pb.ID, AddTaskRequest{Name: "Task 1", Module: "shell", Params: map[string]any{"cmd": "echo 1"}})
	require.NoError(t, err)
	assert.Equal(t, 0, task1.Position)

	task2, err := svc.AddTask(ctx, pb.ID, AddTaskRequest{Name: "Task 2", Module: "shell", Params: map[string]any{"cmd": "echo 2"}})
	require.NoError(t, err)
	assert.Equal(t, 1, task2.Position)

	task3, err := svc.AddTask(ctx, pb.ID, AddTaskRequest{Name: "Task 3", Module: "shell", Params: map[string]any{"cmd": "echo 3"}})
	require.NoError(t, err)
	assert.Equal(t, 2, task3.Position)
}

func TestRenderPlaybook(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	pb, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{
		Name:   "nginx-setup",
		Hosts:  "webservers",
		Become: true,
	})
	require.NoError(t, err)

	_, err = svc.AddTask(ctx, pb.ID, AddTaskRequest{
		Name:   "Install nginx",
		Module: "apt",
		Params: map[string]any{"name": "nginx", "state": "present"},
	})
	require.NoError(t, err)

	_, err = svc.AddTask(ctx, pb.ID, AddTaskRequest{
		Name:   "Start nginx",
		Module: "service",
		Params: map[string]any{"name": "nginx", "state": "started"},
	})
	require.NoError(t, err)

	// Check file was created
	filePath := filepath.Join(tmpDir, "nginx-setup.yml")
	assert.FileExists(t, filePath)

	// Check content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)

	yamlStr := string(content)
	assert.Contains(t, yamlStr, "name: nginx-setup")
	assert.Contains(t, yamlStr, "hosts: webservers")
	assert.Contains(t, yamlStr, "become: true")
	assert.Contains(t, yamlStr, "Install nginx")
	assert.Contains(t, yamlStr, "apt:")
	assert.Contains(t, yamlStr, "Start nginx")
	assert.Contains(t, yamlStr, "service:")
}

func TestExportPlaybook(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	pb, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{
		Name:   "test-export",
		Hosts:  "all",
		Become: false,
	})
	require.NoError(t, err)

	_, err = svc.AddTask(ctx, pb.ID, AddTaskRequest{
		Name:   "Echo hello",
		Module: "shell",
		Params: map[string]any{"cmd": "echo hello"},
	})
	require.NoError(t, err)

	yaml, err := svc.ExportPlaybook(ctx, pb.ID)
	require.NoError(t, err)

	yamlStr := string(yaml)
	assert.Contains(t, yamlStr, "name: test-export")
	assert.Contains(t, yamlStr, "hosts: all")
	assert.Contains(t, yamlStr, "Echo hello")
	assert.Contains(t, yamlStr, "shell:")
}

func TestDeleteTask(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	pb, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{Name: "test-playbook", Hosts: "all"})
	require.NoError(t, err)

	task, err := svc.AddTask(ctx, pb.ID, AddTaskRequest{Name: "Task 1", Module: "shell", Params: map[string]any{"cmd": "echo 1"}})
	require.NoError(t, err)

	err = svc.DeleteTask(ctx, task.ID)
	require.NoError(t, err)

	_, err = svc.GetTask(ctx, task.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestDeletePlaybook(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	pb, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{Name: "test-playbook", Hosts: "all"})
	require.NoError(t, err)

	_, err = svc.AddTask(ctx, pb.ID, AddTaskRequest{Name: "Task 1", Module: "shell", Params: map[string]any{"cmd": "echo 1"}})
	require.NoError(t, err)

	err = svc.DeletePlaybook(ctx, pb.ID)
	require.NoError(t, err)

	_, err = svc.GetPlaybook(ctx, pb.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestReorderTasks(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	pb, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{Name: "test-playbook", Hosts: "all"})
	require.NoError(t, err)

	task1, _ := svc.AddTask(ctx, pb.ID, AddTaskRequest{Name: "Task 1", Module: "shell", Params: map[string]any{"cmd": "echo 1"}})
	task2, _ := svc.AddTask(ctx, pb.ID, AddTaskRequest{Name: "Task 2", Module: "shell", Params: map[string]any{"cmd": "echo 2"}})
	task3, _ := svc.AddTask(ctx, pb.ID, AddTaskRequest{Name: "Task 3", Module: "shell", Params: map[string]any{"cmd": "echo 3"}})

	// Reorder: 3, 1, 2
	err = svc.ReorderTasks(ctx, pb.ID, []string{task3.ID, task1.ID, task2.ID})
	require.NoError(t, err)

	tasks, err := svc.ListTasks(ctx, pb.ID)
	require.NoError(t, err)

	assert.Equal(t, "Task 3", tasks[0].Name)
	assert.Equal(t, "Task 1", tasks[1].Name)
	assert.Equal(t, "Task 2", tasks[2].Name)
}

func TestUpdateTask(t *testing.T) {
	ms := newMockStore()
	tmpDir := t.TempDir()
	svc := NewPlaybookService(ms, tmpDir)
	ctx := context.Background()

	pb, err := svc.CreatePlaybook(ctx, CreatePlaybookRequest{Name: "test-playbook", Hosts: "all"})
	require.NoError(t, err)

	task, err := svc.AddTask(ctx, pb.ID, AddTaskRequest{
		Name:   "Original name",
		Module: "shell",
		Params: map[string]any{"cmd": "echo original"},
	})
	require.NoError(t, err)

	newName := "Updated name"
	newModule := "command"
	updated, err := svc.UpdateTask(ctx, task.ID, UpdateTaskRequest{
		Name:   &newName,
		Module: &newModule,
		Params: map[string]any{"cmd": "echo updated"},
	})
	require.NoError(t, err)

	assert.Equal(t, "Updated name", updated.Name)
	assert.Equal(t, "command", updated.Module)
	assert.Equal(t, "echo updated", updated.Params["cmd"])
}
