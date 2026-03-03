package audit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// VerifyChain reads a JSONL audit log and verifies the hash chain integrity.
// Returns valid=true and brokenAtSeq=-1 when the chain is intact.
// If a break is found, brokenAtSeq is the sequence number of the first
// invalid entry.
func VerifyChain(logPath string) (valid bool, brokenAtSeq int64, err error) {
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, -1, nil
		}
		return false, -1, fmt.Errorf("open audit log: %w", err)
	}
	defer f.Close()

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

		// Check prev_hash links to the previous entry's hash.
		if entry.PrevHash != prevHash {
			return false, entry.Seq, nil
		}

		// Recompute hash from canonical form.
		recomputed := recomputeHash(&entry)
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
	defer f.Close()

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
			continue
		}
		all = append(all, entry)
	}

	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}

// recomputeHash computes the expected hash for an entry using the same
// canonical form as the logger: JSON-encode entry without hash field,
// prepend prev_hash + "|", SHA-256 hex.
func recomputeHash(entry *Entry) string {
	savedHash := entry.Hash
	entry.Hash = ""
	defer func() { entry.Hash = savedHash }()

	canonical, err := json.Marshal(entry)
	if err != nil {
		return ""
	}

	input := entry.PrevHash + "|" + string(canonical)
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}
