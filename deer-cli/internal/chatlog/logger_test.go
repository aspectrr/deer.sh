package chatlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func TestNew(t *testing.T) {
	dir := t.TempDir()
	logger, sessionID, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	if !uuidPattern.MatchString(sessionID) {
		t.Fatalf("session ID %q is not a valid UUID", sessionID)
	}
	if logger.SessionID() != sessionID {
		t.Fatalf("SessionID() = %q, want %q", logger.SessionID(), sessionID)
	}
	if _, err := os.Stat(logger.FilePath()); os.IsNotExist(err) {
		t.Fatalf("log file does not exist at %q", logger.FilePath())
	}
	expectedSuffix := sessionID + ".jsonl"
	if !strings.HasSuffix(logger.FilePath(), expectedSuffix) {
		t.Fatalf("FilePath() = %q, want suffix %q", logger.FilePath(), expectedSuffix)
	}
}

func TestNewCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "chats", "nested")
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", dir)
	}
}

func TestLogSessionStart(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.LogSessionStart("gpt-4")
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events, err := logger.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	e := events[0]
	if e.Type != TypeSessionStart {
		t.Fatalf("Type = %q, want %q", e.Type, TypeSessionStart)
	}
	if e.Model != "gpt-4" {
		t.Fatalf("Model = %q, want %q", e.Model, "gpt-4")
	}
	if e.Meta == nil {
		t.Fatal("Meta is nil")
	}
	if m, ok := e.Meta["model"].(string); !ok || m != "gpt-4" {
		t.Fatalf("Meta[\"model\"] = %v, want %q", e.Meta["model"], "gpt-4")
	}
	if e.Timestamp == "" {
		t.Fatal("Timestamp is empty")
	}
	if e.SessionID != logger.SessionID() {
		t.Fatalf("SessionID = %q, want %q", e.SessionID, logger.SessionID())
	}
}

func TestLogSessionEnd(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.LogSessionEnd(5, 3)
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events, err := logger.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	e := events[0]
	if e.Type != TypeSessionEnd {
		t.Fatalf("Type = %q, want %q", e.Type, TypeSessionEnd)
	}
	if e.Meta == nil {
		t.Fatal("Meta is nil")
	}
	tc, ok := e.Meta["tool_call_count"].(float64)
	if !ok || tc != 5 {
		t.Fatalf("Meta[\"tool_call_count\"] = %v, want 5", e.Meta["tool_call_count"])
	}
	lc, ok := e.Meta["llm_call_count"].(float64)
	if !ok || lc != 3 {
		t.Fatalf("Meta[\"llm_call_count\"] = %v, want 3", e.Meta["llm_call_count"])
	}
}

func TestLogUserMessage(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.LogUserMessage("Hello, world!")
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events, err := logger.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	e := events[0]
	if e.Type != TypeUserMessage {
		t.Fatalf("Type = %q, want %q", e.Type, TypeUserMessage)
	}
	if e.Content != "Hello, world!" {
		t.Fatalf("Content = %q, want %q", e.Content, "Hello, world!")
	}
}

func TestLogLLMResponse(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	toolCalls := []ToolCallEntry{
		{ID: "tc1", Name: "run_command", Args: map[string]any{"cmd": "ls"}},
		{ID: "tc2", Name: "read_file", Args: map[string]any{"path": "/tmp/x"}},
	}
	logger.LogLLMResponse("I'll run a command.", "claude-3", toolCalls)
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events, err := logger.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	e := events[0]
	if e.Type != TypeLLMResponse {
		t.Fatalf("Type = %q, want %q", e.Type, TypeLLMResponse)
	}
	if e.Content != "I'll run a command." {
		t.Fatalf("Content = %q, want %q", e.Content, "I'll run a command.")
	}
	if e.Model != "claude-3" {
		t.Fatalf("Model = %q, want %q", e.Model, "claude-3")
	}
	if len(e.ToolCalls) != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2", len(e.ToolCalls))
	}
	if e.ToolCalls[0].ID != "tc1" || e.ToolCalls[0].Name != "run_command" {
		t.Fatalf("ToolCalls[0] = %+v, want {ID:tc1, Name:run_command}", e.ToolCalls[0])
	}
	if e.ToolCalls[1].ID != "tc2" || e.ToolCalls[1].Name != "read_file" {
		t.Fatalf("ToolCalls[1] = %+v, want {ID:tc2, Name:read_file}", e.ToolCalls[1])
	}
}

func TestLogToolCall(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	args := map[string]any{"cmd": "ls -la"}
	logger.LogToolCall("run_command", args, "file1\nfile2", fmt.Errorf("exit code 1"), 150)
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events, err := logger.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	e := events[0]
	if e.Type != TypeToolCall {
		t.Fatalf("Type = %q, want %q", e.Type, TypeToolCall)
	}
	if e.Tool != "run_command" {
		t.Fatalf("Tool = %q, want %q", e.Tool, "run_command")
	}
	if e.Args["cmd"] != "ls -la" {
		t.Fatalf("Args[\"cmd\"] = %v, want %q", e.Args["cmd"], "ls -la")
	}
	if e.Result != "file1\nfile2" {
		t.Fatalf("Result = %v, want %q", e.Result, "file1\nfile2")
	}
	if e.Error != "exit code 1" {
		t.Fatalf("Error = %q, want %q", e.Error, "exit code 1")
	}
	if e.DurationMS != 150 {
		t.Fatalf("DurationMS = %d, want 150", e.DurationMS)
	}
}

func TestLogToolCallNoError(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.LogToolCall("read_file", map[string]any{"path": "/tmp/x"}, "contents", nil, 42)
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events, err := logger.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}

	if events[0].Error != "" {
		t.Fatalf("Error = %q, want empty string", events[0].Error)
	}

	raw, err := os.ReadFile(logger.FilePath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	if strings.Contains(string(raw), `"error"`) {
		t.Fatalf("raw JSON should not contain \"error\" field when err is nil, got: %s", string(raw))
	}
}

func TestReadEvents(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.LogSessionStart("gpt-4")
	logger.LogUserMessage("hello")
	logger.LogLLMResponse("hi there", "gpt-4", nil)
	logger.LogToolCall("ls", nil, "file.txt", nil, 10)
	logger.LogSessionEnd(1, 1)
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events, err := logger.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents() error = %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("len(events) = %d, want 5", len(events))
	}

	wantTypes := []string{TypeSessionStart, TypeUserMessage, TypeLLMResponse, TypeToolCall, TypeSessionEnd}
	for i, want := range wantTypes {
		if events[i].Type != want {
			t.Fatalf("events[%d].Type = %q, want %q", i, events[i].Type, want)
		}
	}
	for i, e := range events {
		if e.SessionID != logger.SessionID() {
			t.Fatalf("events[%d].SessionID = %q, want %q", i, e.SessionID, logger.SessionID())
		}
	}
}

func TestReadEventsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events, err := logger.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents() error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("len(events) = %d, want 0", len(events))
	}
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := os.Stat(logger.FilePath()); os.IsNotExist(err) {
		t.Fatal("log file should still exist after Close()")
	}

	if err := logger.Close(); err == nil {
		t.Fatal("expected error on second Close(), got nil")
	}
}

func TestConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	logger, _, err := New(dir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const goroutines = 20
	const perGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				logger.LogToolCall(
					fmt.Sprintf("tool_%d", id),
					map[string]any{"iter": j},
					"ok",
					nil,
					int64(j),
				)
			}
		}(i)
	}
	wg.Wait()

	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	events, err := logger.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents() error = %v", err)
	}

	expected := goroutines * perGoroutine
	if len(events) != expected {
		t.Fatalf("len(events) = %d, want %d", len(events), expected)
	}

	for i, e := range events {
		if e.Type != TypeToolCall {
			t.Fatalf("events[%d].Type = %q, want %q", i, e.Type, TypeToolCall)
		}
		if e.SessionID != logger.SessionID() {
			t.Fatalf("events[%d].SessionID = %q, want %q", i, e.SessionID, logger.SessionID())
		}
	}

	raw, err := os.ReadFile(logger.FilePath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != expected {
		t.Fatalf("len(lines) = %d, want %d", len(lines), expected)
	}
	for i, line := range lines {
		var v map[string]any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Fatalf("line %d is not valid JSON: %v", i, err)
		}
	}
}
