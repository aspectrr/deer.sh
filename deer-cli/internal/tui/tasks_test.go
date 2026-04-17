package tui

import "testing"

func TestTaskList_Add(t *testing.T) {
	tl := NewTaskList()
	task := tl.Add("Install nginx")
	if task.ID != "t1" {
		t.Errorf("expected ID t1, got %s", task.ID)
	}
	if task.Content != "Install nginx" {
		t.Errorf("expected content 'Install nginx', got %s", task.Content)
	}
	if task.Status != TaskPending {
		t.Errorf("expected status pending, got %s", task.Status)
	}
	if !tl.HasTasks() {
		t.Error("expected HasTasks() to be true")
	}
}

func TestTaskList_Update(t *testing.T) {
	tl := NewTaskList()
	tl.Add("Task 1")
	tl.Add("Task 2")

	updated, found := tl.Update("t1", TaskInProgress, "")
	if !found {
		t.Fatal("task t1 not found")
	}
	if updated.Status != TaskInProgress {
		t.Errorf("expected in_progress, got %s", updated.Status)
	}

	_, found = tl.Update("nonexistent", TaskCompleted, "")
	if found {
		t.Error("expected nonexistent task to not be found")
	}
}

func TestTaskList_Delete(t *testing.T) {
	tl := NewTaskList()
	tl.Add("Task 1")
	tl.Add("Task 2")

	if !tl.Delete("t1") {
		t.Error("expected delete to succeed")
	}
	tasks := tl.List()
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "t2" {
		t.Errorf("expected remaining task t2, got %s", tasks[0].ID)
	}
}

func TestTaskList_Clear(t *testing.T) {
	tl := NewTaskList()
	tl.Add("Task 1")
	tl.Add("Task 2")
	tl.Clear()
	if tl.HasTasks() {
		t.Error("expected HasTasks() to be false after Clear()")
	}
}

func TestTaskList_FormatForSystemPrompt(t *testing.T) {
	tl := NewTaskList()
	if tl.FormatForSystemPrompt() != "" {
		t.Error("expected empty prompt when no tasks")
	}

	tl.Add("Install nginx")
	tl.Add("Configure SSL")
	tl.Update("t1", TaskCompleted, "")

	prompt := tl.FormatForSystemPrompt()
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !contains(prompt, "[x]") {
		t.Error("expected [x] for completed task")
	}
	if !contains(prompt, "[ ]") {
		t.Error("expected [ ] for pending task")
	}
}

func TestTaskList_Summary(t *testing.T) {
	tl := NewTaskList()
	if tl.Summary() != "" {
		t.Error("expected empty summary when no tasks")
	}
	tl.Add("Task 1")
	tl.Add("Task 2")
	tl.Add("Task 3")
	tl.Update("t1", TaskCompleted, "")
	tl.Update("t2", TaskInProgress, "")
	summary := tl.Summary()
	if !contains(summary, "1/3 completed") {
		t.Errorf("unexpected summary: %s", summary)
	}
}

func TestRenderTaskPanel(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Content: "Install nginx", Status: TaskCompleted},
		{ID: "t2", Content: "Configure SSL", Status: TaskInProgress},
		{ID: "t3", Content: "Run tests", Status: TaskPending},
	}
	panel := renderTaskPanel(tasks, 80, false)
	if panel == "" {
		t.Fatal("expected non-empty panel")
	}
	if !contains(panel, "Tasks") {
		t.Error("expected 'Tasks' in panel")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
