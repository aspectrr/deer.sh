package tui

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/charmbracelet/lipgloss"
)

type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
)

type Task struct {
	ID      string     `json:"id"`
	Content string     `json:"content"`
	Status  TaskStatus `json:"status"`
}

type TaskList struct {
	tasks  []Task
	nextID atomic.Int64
}

func NewTaskList() *TaskList {
	return &TaskList{}
}

func (tl *TaskList) Add(content string) Task {
	id := fmt.Sprintf("t%d", tl.nextID.Add(1))
	t := Task{ID: id, Content: content, Status: TaskPending}
	tl.tasks = append(tl.tasks, t)
	return t
}

func (tl *TaskList) Update(taskID string, status TaskStatus, content string) (Task, bool) {
	for i := range tl.tasks {
		if tl.tasks[i].ID == taskID {
			if status != "" {
				tl.tasks[i].Status = status
			}
			if content != "" {
				tl.tasks[i].Content = content
			}
			return tl.tasks[i], true
		}
	}
	return Task{}, false
}

func (tl *TaskList) Delete(taskID string) bool {
	for i := range tl.tasks {
		if tl.tasks[i].ID == taskID {
			tl.tasks = append(tl.tasks[:i], tl.tasks[i+1:]...)
			return true
		}
	}
	return false
}

func (tl *TaskList) List() []Task {
	result := make([]Task, len(tl.tasks))
	copy(result, tl.tasks)
	return result
}

func (tl *TaskList) Clear() {
	tl.tasks = nil
}

func (tl *TaskList) HasTasks() bool {
	return len(tl.tasks) > 0
}

func (tl *TaskList) Summary() string {
	if len(tl.tasks) == 0 {
		return ""
	}
	var b strings.Builder
	pending := 0
	inProgress := 0
	completed := 0
	for _, t := range tl.tasks {
		switch t.Status {
		case TaskPending:
			pending++
		case TaskInProgress:
			inProgress++
		case TaskCompleted:
			completed++
		}
	}
	fmt.Fprintf(&b, "Tasks: %d/%d completed", completed, len(tl.tasks))
	if inProgress > 0 {
		fmt.Fprintf(&b, ", %d in progress", inProgress)
	}
	if pending > 0 {
		fmt.Fprintf(&b, ", %d pending", pending)
	}
	return b.String()
}

func (tl *TaskList) FormatForSystemPrompt() string {
	if len(tl.tasks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Current Task List\n\n")
	for _, t := range tl.tasks {
		var icon string
		switch t.Status {
		case TaskPending:
			icon = "[ ]"
		case TaskInProgress:
			icon = "[~]"
		case TaskCompleted:
			icon = "[x]"
		}
		fmt.Fprintf(&b, "- %s %s (%s)\n", icon, t.Content, t.ID)
	}
	b.WriteString("\nUpdate tasks as you work. Mark tasks in_progress when starting, completed when done.")
	return b.String()
}

func renderTaskPanel(tasks []Task, width int, expanded bool) string {
	if len(tasks) == 0 {
		return ""
	}

	maxVisible := 3
	if expanded {
		maxVisible = len(tasks)
		if maxVisible > 15 {
			maxVisible = 15
		}
	}

	boxWidth := width - 4
	if boxWidth < 30 {
		boxWidth = 30
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6366F1")).
		Padding(0, 1).
		Width(boxWidth)

	completed := 0
	for _, t := range tasks {
		if t.Status == TaskCompleted {
			completed++
		}
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6366F1")).
		Bold(true).
		Render(fmt.Sprintf("Tasks (%d/%d)", completed, len(tasks)))

	var lines []string
	visible := tasks
	if !expanded && len(tasks) > maxVisible {
		visible = tasks[:maxVisible]
	}

	for _, t := range visible {
		var icon string
		var contentStyle lipgloss.Style
		switch t.Status {
		case TaskPending:
			icon = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("o")
			contentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
		case TaskInProgress:
			icon = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true).Render("~")
			contentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB"))
		case TaskCompleted:
			icon = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("v")
			contentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
		}

		content := t.Content
		maxContentWidth := boxWidth - 8
		if maxContentWidth < 20 {
			maxContentWidth = 20
		}
		if len(content) > maxContentWidth {
			content = content[:maxContentWidth-3] + "..."
		}
		lines = append(lines, fmt.Sprintf(" %s %s", icon, contentStyle.Render(content)))
	}

	if !expanded && len(tasks) > maxVisible {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true).Render(
			fmt.Sprintf("   ... %d more (Ctrl+T to expand)", len(tasks)-maxVisible)))
	}

	hint := ""
	if !expanded {
		hint = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Ctrl+T to expand")
	} else {
		hint = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Ctrl+T to collapse")
	}

	body := header + "\n" + strings.Join(lines, "\n")
	if hint != "" {
		body += "\n" + hint
	}

	return style.Render(body)
}
