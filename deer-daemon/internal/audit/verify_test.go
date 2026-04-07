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
	logger.LogOperation(TypeSandboxCreated, map[string]any{"sandbox_id": "sbx-1"}, nil, 5)
	logger.LogOperation(TypeCommandExecuted, map[string]any{"command": "ls"}, nil, 3)
	logger.LogSessionEnd()
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
	logger.LogOperation(TypeSandboxCreated, nil, nil, 5)
	logger.LogOperation(TypeCommandExecuted, nil, nil, 3)
	logger.LogSessionEnd()
	_ = logger.Close()

	lines := readLines(t, logPath)
	if len(lines) < 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	var entry Entry
	if err := json.Unmarshal(lines[1], &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	entry.Meta = map[string]any{"tampered": true}
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
	logger.LogOperation(TypeSandboxCreated, nil, nil, 5)
	logger.LogOperation(TypeCommandExecuted, nil, nil, 3)
	logger.LogSessionEnd()
	_ = logger.Close()

	lines := readLines(t, logPath)
	if len(lines) < 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	lines = append(lines[:1], lines[2:]...)
	writeLines(t, logPath, lines)

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if valid {
		t.Fatal("expected invalid chain after deletion")
	}
	if brokenAt != 3 {
		t.Fatalf("expected break at seq 3, got %d", brokenAt)
	}
}

func TestVerifyEmptyFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

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
