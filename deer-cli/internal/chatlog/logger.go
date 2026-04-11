// Package chatlog writes a per-session JSONL chat log to ~/.config/deer/chats/.
// Each TUI session gets a new UUID file containing every user message, LLM
// response, tool call, and tool result - enough to replay or audit a session.
package chatlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	TypeSessionStart = "session_start"
	TypeSessionEnd   = "session_end"
	TypeUserMessage  = "user_message"
	TypeLLMResponse  = "llm_response"
	TypeToolCall     = "tool_call"
)

// ToolCallEntry carries the name and arguments of a single tool call from an
// LLM response, logged before execution.
type ToolCallEntry struct {
	ID   string         `json:"id,omitempty"`
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// Event is one line in the per-session JSONL chat log.
type Event struct {
	Type      string `json:"type"`
	Timestamp string `json:"ts"`
	SessionID string `json:"session_id"`

	// user_message
	Content string `json:"content,omitempty"`

	// llm_response
	Model     string          `json:"model,omitempty"`
	ToolCalls []ToolCallEntry `json:"tool_calls,omitempty"`

	// tool_call result
	Tool       string         `json:"tool,omitempty"`
	Args       map[string]any `json:"args,omitempty"`
	Result     any            `json:"result,omitempty"`
	Error      string         `json:"error,omitempty"`
	DurationMS int64          `json:"duration_ms,omitempty"`

	// session_start / session_end meta
	Meta map[string]any `json:"meta,omitempty"`
}

// Logger writes events for a single session to a JSONL file.
type Logger struct {
	sessionID string
	filePath  string
	file      *os.File
	mu        sync.Mutex
}

// New creates a chat log file at dir/<session-uuid>.jsonl and returns a Logger.
// The session UUID is also returned so callers can display or reference it.
func New(dir string) (*Logger, string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, "", fmt.Errorf("create chats dir: %w", err)
	}

	id := uuid.New().String()
	path := filepath.Join(dir, id+".jsonl")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, "", fmt.Errorf("open chat log: %w", err)
	}

	return &Logger{sessionID: id, filePath: path, file: f}, id, nil
}

// SessionID returns the UUID for this session.
func (l *Logger) SessionID() string { return l.sessionID }

// FilePath returns the path to the JSONL file for this session.
func (l *Logger) FilePath() string { return l.filePath }

// ReadEvents parses the JSONL log file and returns all recorded events.
// The file must be closed (or at least flushed) before calling this.
func (l *Logger) ReadEvents() ([]Event, error) {
	data, err := os.ReadFile(l.filePath)
	if err != nil {
		return nil, fmt.Errorf("read chat log: %w", err)
	}
	var events []Event
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, nil
}

// Close flushes and closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// LogSessionStart records session metadata at startup.
func (l *Logger) LogSessionStart(model string) {
	l.write(&Event{
		Type:  TypeSessionStart,
		Model: model,
		Meta:  map[string]any{"model": model},
	})
}

// LogSessionEnd records final stats.
func (l *Logger) LogSessionEnd(toolCallCount, llmCallCount int) {
	l.write(&Event{
		Type: TypeSessionEnd,
		Meta: map[string]any{
			"tool_call_count": toolCallCount,
			"llm_call_count":  llmCallCount,
		},
	})
}

// LogUserMessage records the full text of a user message.
func (l *Logger) LogUserMessage(content string) {
	l.write(&Event{
		Type:    TypeUserMessage,
		Content: content,
	})
}

// LogLLMResponse records the full LLM response text and any tool calls it
// requested.
func (l *Logger) LogLLMResponse(content, model string, toolCalls []ToolCallEntry) {
	l.write(&Event{
		Type:      TypeLLMResponse,
		Content:   content,
		Model:     model,
		ToolCalls: toolCalls,
	})
}

// LogToolCall records a tool invocation with its full arguments, result,
// optional error, and wall-clock duration.
func (l *Logger) LogToolCall(tool string, args map[string]any, result any, err error, durationMS int64) {
	e := &Event{
		Type:       TypeToolCall,
		Tool:       tool,
		Args:       args,
		Result:     result,
		DurationMS: durationMS,
	}
	if err != nil {
		e.Error = err.Error()
	}
	l.write(e)
}

func (l *Logger) write(e *Event) {
	l.mu.Lock()
	defer l.mu.Unlock()

	e.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	e.SessionID = l.sessionID

	data, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chatlog: marshal error: %v\n", err)
		return
	}
	data = append(data, '\n')
	if _, err := l.file.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "chatlog: write error: %v\n", err)
	}
}
