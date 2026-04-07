package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyValidChain(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

	logger, err := NewLogger(logPath, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	logger.LogSessionStart()
	logger.LogToolCall("ls", nil, "ok", nil, 5)
	logger.LogLLMRequest(1, 100, "test-model")
	logger.LogLLMResponse(50, 1)
	logger.LogSessionEnd(1, 1)
	_ = logger.Close()

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if !valid {
		t.Fatalf("expected valid chain, broken at seq %d", brokenAt)
	}
	if brokenAt != -1 {
		t.Fatalf("expected brokenAtSeq=-1, got %d", brokenAt)
	}
}

func TestVerifyDetectModifiedEntry(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

	logger, err := NewLogger(logPath, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	logger.LogSessionStart()
	logger.LogToolCall("ls", nil, "ok", nil, 5)
	logger.LogToolCall("pwd", nil, "/home", nil, 3)
	logger.LogSessionEnd(2, 0)
	_ = logger.Close()

	// Read lines.
	lines := readLines(t, logPath)
	if len(lines) < 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	// Modify entry at index 1 (seq 2): change the result.
	var entry Entry
	if err := json.Unmarshal(lines[1], &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	entry.Result = "MODIFIED"
	modified, _ := json.Marshal(entry)
	lines[1] = modified

	writeLines(t, logPath, lines)

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if valid {
		t.Fatal("expected invalid chain after modification")
	}
	if brokenAt != 2 {
		t.Fatalf("expected break at seq 2, got %d", brokenAt)
	}
}

func TestVerifyDetectDeletedEntry(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

	logger, err := NewLogger(logPath, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	logger.LogSessionStart()
	logger.LogToolCall("ls", nil, "ok", nil, 5)
	logger.LogToolCall("pwd", nil, "/home", nil, 3)
	logger.LogSessionEnd(2, 0)
	_ = logger.Close()

	// Read lines and delete the second entry (index 1).
	lines := readLines(t, logPath)
	if len(lines) < 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	// Remove entry at index 1.
	lines = append(lines[:1], lines[2:]...)
	writeLines(t, logPath, lines)

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if valid {
		t.Fatal("expected invalid chain after deletion")
	}
	// The break should be detected at the entry that now follows the gap.
	// Entry at index 1 after deletion was originally seq 3, whose prev_hash
	// points to seq 2's hash (which is now missing).
	if brokenAt != 3 {
		t.Fatalf("expected break at seq 3, got %d", brokenAt)
	}
}

func TestVerifyEmptyFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

	// Create empty file.
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = f.Close()

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if !valid {
		t.Fatal("expected valid for empty file")
	}
	if brokenAt != -1 {
		t.Fatalf("expected brokenAtSeq=-1, got %d", brokenAt)
	}
}

func TestVerifyNonexistentFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "does_not_exist.jsonl")

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if !valid {
		t.Fatal("expected valid for nonexistent file")
	}
	if brokenAt != -1 {
		t.Fatalf("expected brokenAtSeq=-1, got %d", brokenAt)
	}
}

// readLines reads all non-empty lines from a JSONL file.
func readLines(t *testing.T, path string) [][]byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = f.Close() }()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		line := make([]byte, len(scanner.Bytes()))
		copy(line, scanner.Bytes())
		lines = append(lines, line)
	}
	return lines
}

// writeLines writes lines to a file, each followed by a newline.
func writeLines(t *testing.T, path string, lines [][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = f.Close() }()
	for _, line := range lines {
		_, _ = f.Write(line)
		_, _ = f.Write([]byte("\n"))
	}
}
