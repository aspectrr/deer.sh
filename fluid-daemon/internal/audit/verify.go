package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// VerifyChain reads a JSONL audit log and verifies the hash chain integrity.
// Returns valid=true and brokenAtSeq=-1 when the chain is intact.
func VerifyChain(logPath string) (valid bool, brokenAtSeq int64, err error) {
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, -1, nil
		}
		return false, -1, fmt.Errorf("open audit log: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	prevHash := genesisHash
	count := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			return false, int64(count + 1), fmt.Errorf("parse entry: %w", err)
		}
		count++

		if entry.PrevHash != prevHash {
			return false, entry.Seq, nil
		}

		recomputed := hashEntry(&entry)
		if recomputed != entry.Hash {
			return false, entry.Seq, nil
		}

		prevHash = entry.Hash
	}

	if err := scanner.Err(); err != nil {
		return false, -1, fmt.Errorf("read audit log: %w", err)
	}

	return true, -1, nil
}

// ReadRecent reads the last n entries from the audit log.
func ReadRecent(logPath string, n int) ([]Entry, error) {
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	defer func() { _ = f.Close() }()

	var all []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			slog.Warn("audit: skipping unparseable log line", "error", err)
			continue
		}
		all = append(all, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read audit log: %w", err)
	}

	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}
