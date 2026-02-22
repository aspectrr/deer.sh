package tui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/aspectrr/fluid.sh/fluid/internal/paths"
)

// MaxHistoryEntries is the maximum number of history entries to keep.
const MaxHistoryEntries = 1000

// HistoryPath returns the history file path under the XDG data directory.
// History is runtime data and belongs in DataDir, not ConfigDir.
func HistoryPath() string {
	dir, err := paths.DataDir()
	if err != nil {
		// Best-effort fallback
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "share", "fluid")
	}
	return filepath.Join(dir, "history")
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
