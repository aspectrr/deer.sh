package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestWriteAndVerifyChain(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

	logger, err := NewLogger(logPath, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	logger.LogSessionStart()
	logger.LogOperation(TypeSandboxCreated, map[string]any{"sandbox_id": "sbx-1"}, nil, 150)
	logger.LogOperation(TypeCommandExecuted, map[string]any{"sandbox_id": "sbx-1", "command": "ls"}, nil, 30)
	logger.LogOperation(TypeSandboxDestroyed, map[string]any{"sandbox_id": "sbx-1"}, fmt.Errorf("not found"), 5)
	logger.LogSessionEnd()

	if err := logger.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if !valid {
		t.Fatalf("chain invalid at seq %d", brokenAt)
	}
	if brokenAt != -1 {
		t.Fatalf("expected brokenAtSeq=-1, got %d", brokenAt)
	}
}

func TestTamperDetection(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

	logger, err := NewLogger(logPath, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	logger.LogSessionStart()
	logger.LogOperation(TypeSandboxCreated, map[string]any{"sandbox_id": "sbx-1"}, nil, 10)
	logger.LogOperation(TypeCommandExecuted, map[string]any{"command": "pwd"}, nil, 5)
	logger.LogSessionEnd()
	_ = logger.Close()

	// Read all lines, tamper with the middle entry.
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	var lines [][]byte
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := make([]byte, len(scanner.Bytes()))
		copy(line, scanner.Bytes())
		lines = append(lines, line)
	}
	_ = f.Close()

	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}

	// Tamper: change meta in entry at index 2.
	var entry Entry
	if err := json.Unmarshal(lines[2], &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	entry.Meta = map[string]any{"command": "TAMPERED"}
	tampered, _ := json.Marshal(entry)
	lines[2] = tampered

	out, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for _, line := range lines {
		_, _ = out.Write(line)
		_, _ = out.Write([]byte("\n"))
	}
	_ = out.Close()

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if valid {
		t.Fatal("expected chain to be invalid after tampering")
	}
	if brokenAt != 3 {
		t.Fatalf("expected break at seq 3, got %d", brokenAt)
	}
}

func TestChainContinuity(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

	logger, err := NewLogger(logPath, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	logger.LogSessionStart()
	logger.LogOperation(TypeSandboxCreated, map[string]any{"sandbox_id": "sbx-1"}, nil, 5)
	logger.LogSessionEnd()
	_ = logger.Close()

	// Reopen and continue.
	logger2, err := NewLogger(logPath, 10)
	if err != nil {
		t.Fatalf("NewLogger (reopen): %v", err)
	}
	logger2.LogSessionStart()
	logger2.LogOperation(TypeCommandExecuted, map[string]any{"command": "ls"}, nil, 3)
	logger2.LogSessionEnd()
	_ = logger2.Close()

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if !valid {
		t.Fatalf("chain invalid at seq %d", brokenAt)
	}

	// Verify sequence numbers are continuous.
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lastSeq int64
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if e.Seq != lastSeq+1 {
			t.Fatalf("sequence gap: expected %d, got %d", lastSeq+1, e.Seq)
		}
		lastSeq = e.Seq
	}
	if lastSeq != 6 {
		t.Fatalf("expected 6 entries, got %d", lastSeq)
	}
}

func TestConcurrentWrites(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

	logger, err := NewLogger(logPath, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	const goroutines = 20
	const callsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				logger.LogOperation(
					TypeCommandExecuted,
					map[string]any{"worker": id, "iter": j},
					nil,
					int64(j),
				)
			}
		}(i)
	}
	wg.Wait()
	_ = logger.Close()

	valid, brokenAt, err := VerifyChain(logPath)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if !valid {
		t.Fatalf("chain invalid at seq %d after concurrent writes", brokenAt)
	}

	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	count := 0
	for scanner.Scan() {
		if len(scanner.Bytes()) > 0 {
			count++
		}
	}
	expected := goroutines * callsPerGoroutine
	if count != expected {
		t.Fatalf("expected %d entries, got %d", expected, count)
	}
}

func TestGenesisEntry(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "audit.jsonl")

	logger, err := NewLogger(logPath, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	logger.LogSessionStart()
	_ = logger.Close()

	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if !scanner.Scan() {
		t.Fatal("expected at least one entry")
	}

	var entry Entry
	if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if entry.PrevHash != genesisHash {
		t.Fatalf("genesis entry prev_hash = %q, want %q", entry.PrevHash, genesisHash)
	}
	if entry.Seq != 1 {
		t.Fatalf("genesis entry seq = %d, want 1", entry.Seq)
	}
	if entry.Hash == "" {
		t.Fatal("genesis entry hash is empty")
	}
}
