package audit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	TypeToolCall     = "tool_call"
	TypeLLMRequest   = "llm_request"
	TypeLLMResponse  = "llm_response"
	TypeUserInput    = "user_input"
	TypeSessionStart = "session_start"
	TypeSessionEnd   = "session_end"
	TypeRedaction    = "redaction"

	genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"
)

// Entry represents a single audit log record in JSONL format.
type Entry struct {
	Seq        int64          `json:"seq"`
	Timestamp  string         `json:"ts"`
	Type       string         `json:"type"`
	Tool       string         `json:"tool,omitempty"`
	Args       map[string]any `json:"args,omitempty"`
	Result     any            `json:"result,omitempty"`
	DurationMS int64          `json:"duration_ms,omitempty"`
	Error      string         `json:"error,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"`
	PrevHash   string         `json:"prev_hash"`
	Hash       string         `json:"hash"`
}

// Logger writes hash-chained audit log entries to a JSONL file.
type Logger struct {
	file         *os.File
	prevHash     string
	seq          int64
	mu           sync.Mutex
	maxSizeBytes int64
	dropped      int64
}

// NewLogger opens or creates an audit log file. It reads any existing entries
// to recover the last hash and sequence number for chain continuity.
func NewLogger(logPath string, maxSizeMB int) (*Logger, error) {
	prevHash := genesisHash
	var seq int64

	// Read existing entries to find last hash and seq.
	if f, err := os.Open(logPath); err == nil {
		scanner := bufio.NewScanner(f)
		// Allow large lines (up to 1MB).
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var e Entry
			if err := json.Unmarshal(line, &e); err != nil {
				slog.Warn("audit: skipping unparseable entry during startup", "error", err)
				continue
			}
			prevHash = e.Hash
			seq = e.Seq
		}
		if err := scanner.Err(); err != nil {
			slog.Warn("audit: error reading log during startup", "error", err)
		}
		_ = f.Close()
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open audit log: %w", err)
	}

	return &Logger{
		file:         file,
		prevHash:     prevHash,
		seq:          seq,
		maxSizeBytes: int64(maxSizeMB) * 1024 * 1024,
	}, nil
}

// Close closes the underlying log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// LogToolCall records a tool invocation.
func (l *Logger) LogToolCall(tool string, args map[string]any, result any, err error, durationMs int64) {
	entry := &Entry{
		Type:       TypeToolCall,
		Tool:       tool,
		Args:       args,
		Result:     result,
		DurationMS: durationMs,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	l.write(entry)
}

// LogLLMRequest records an outgoing LLM API request.
func (l *Logger) LogLLMRequest(messageCount int, tokenEstimate int, model string) {
	l.write(&Entry{
		Type: TypeLLMRequest,
		Meta: map[string]any{
			"message_count":  messageCount,
			"token_estimate": tokenEstimate,
			"model":          model,
		},
	})
}

// LogLLMResponse records an LLM API response.
func (l *Logger) LogLLMResponse(tokenEstimate int, toolCallCount int) {
	l.write(&Entry{
		Type: TypeLLMResponse,
		Meta: map[string]any{
			"token_estimate":  tokenEstimate,
			"tool_call_count": toolCallCount,
		},
	})
}

// LogUserInput records that user input was received. Only length is logged,
// not the content itself.
func (l *Logger) LogUserInput(inputLength int) {
	l.write(&Entry{
		Type: TypeUserInput,
		Meta: map[string]any{
			"input_length": inputLength,
		},
	})
}

// LogSessionStart records the beginning of a session.
func (l *Logger) LogSessionStart() {
	l.write(&Entry{
		Type: TypeSessionStart,
	})
}

// LogSessionEnd records the end of a session with summary stats.
func (l *Logger) LogSessionEnd(toolCallCount int, llmCallCount int) {
	l.write(&Entry{
		Type: TypeSessionEnd,
		Meta: map[string]any{
			"tool_call_count": toolCallCount,
			"llm_call_count":  llmCallCount,
		},
	})
}

// write computes the hash chain, marshals the entry to JSON, and appends it.
func (l *Logger) write(entry *Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Enforce max file size if configured.
	if l.maxSizeBytes > 0 {
		if info, err := l.file.Stat(); err == nil && info.Size() >= l.maxSizeBytes {
			l.dropped++
			slog.Warn("audit: log file at max size, dropping event", "size", info.Size(), "max", l.maxSizeBytes, "dropped_total", l.dropped)
			return
		}
	}

	l.seq++
	entry.Seq = l.seq
	entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	entry.PrevHash = l.prevHash

	entry.Hash = hashEntry(entry)
	l.prevHash = entry.Hash

	data, err := json.Marshal(entry)
	if err != nil {
		slog.Error("audit: failed to marshal entry", "error", err, "seq", entry.Seq)
		return
	}
	data = append(data, '\n')

	if _, err := l.file.Write(data); err != nil {
		slog.Error("audit: write failed", "error", err)
	}
	if err := l.file.Sync(); err != nil {
		slog.Error("audit: sync failed", "error", err)
	}
}

// hashEntry produces the SHA-256 hex digest for an entry.
// Canonical form: JSON-encode the entry without the hash field, prepend
// prev_hash + "|", then SHA-256.
//
// NOTE: This temporarily mutates entry.Hash in place and restores it via defer.
// This is safe because all callers hold l.mu.
func hashEntry(entry *Entry) string {
	// Save and clear hash so it is excluded from canonical JSON.
	savedHash := entry.Hash
	entry.Hash = ""
	defer func() { entry.Hash = savedHash }()

	canonical, err := json.Marshal(entry)
	if err != nil {
		slog.Error("audit: failed to marshal entry for hashing", "error", err)
		return ""
	}

	input := entry.PrevHash + "|" + string(canonical)
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}
