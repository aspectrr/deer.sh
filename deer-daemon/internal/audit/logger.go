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
	TypeSandboxCreated   = "sandbox_created"
	TypeSandboxDestroyed = "sandbox_destroyed"
	TypeSandboxStarted   = "sandbox_started"
	TypeSandboxStopped   = "sandbox_stopped"
	TypeCommandExecuted  = "command_executed"
	TypeSnapshotCreated  = "snapshot_created"
	TypeSourceCommand    = "source_command"
	TypeFileRead         = "file_read"
	TypeSessionStart     = "session_start"
	TypeSessionEnd       = "session_end"

	genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"
)

// Entry represents a single audit log record in JSONL format.
type Entry struct {
	Seq        int64          `json:"seq"`
	Timestamp  string         `json:"ts"`
	Type       string         `json:"type"`
	Meta       map[string]any `json:"meta,omitempty"`
	DurationMS int64          `json:"duration_ms,omitempty"`
	Error      string         `json:"error,omitempty"`
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

	if f, err := os.Open(logPath); err == nil {
		scanner := bufio.NewScanner(f)
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

// LogOperation records a daemon operation.
func (l *Logger) LogOperation(opType string, meta map[string]any, err error, durationMs int64) {
	entry := &Entry{
		Type:       opType,
		Meta:       meta,
		DurationMS: durationMs,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	l.write(entry)
}

// LogSessionStart records the beginning of a daemon session.
func (l *Logger) LogSessionStart() {
	l.write(&Entry{
		Type: TypeSessionStart,
	})
}

// LogSessionEnd records the end of a daemon session.
func (l *Logger) LogSessionEnd() {
	l.write(&Entry{
		Type: TypeSessionEnd,
	})
}

// write computes the hash chain, marshals the entry to JSON, and appends it.
func (l *Logger) write(entry *Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

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
func hashEntry(entry *Entry) string {
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
