package tui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// MaxHistoryEntries is the maximum number of history entries to keep.
const MaxHistoryEntries = 1000

// HistoryPath derives the history file path from the config file path.
// e.g. ~/.fluid/config.yaml -> ~/.fluid/history
func HistoryPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "history")
}

// LoadHistory reads the history file and returns the last MaxHistoryEntries lines.
// Returns nil on any error (missing file, permission denied, etc.).
// If the file exceeds MaxHistoryEntries, it is truncated on disk.
func LoadHistory(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	if scanner.Err() != nil {
		return nil
	}

	if len(lines) == 0 {
		return nil
	}

	// Truncate if over limit
	if len(lines) > MaxHistoryEntries {
		lines = lines[len(lines)-MaxHistoryEntries:]
		// Best-effort rewrite to truncate file on disk
		_ = writeLines(path, lines)
	}

	return lines
}

// AppendHistory appends a single entry to the history file.
// Silently ignores all errors.
func AppendHistory(path string, entry string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.WriteString(entry + "\n")
}

// writeLines overwrites a file with the given lines.
func writeLines(path string, lines []string) error {
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.WriteString(strings.Join(lines, "\n") + "\n")
	return err
}
