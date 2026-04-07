package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	got := HistoryPath()
	assert.Equal(t, filepath.Join(tmp, "fluid", "history"), got)
}

func TestLoadHistory_NoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent")
	assert.Nil(t, LoadHistory(path))
}

func TestLoadHistory_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o600))
	assert.Nil(t, LoadHistory(path))
}

func TestLoadHistory_NormalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	content := "list\ncreate foo\nrun SBX-1 whoami\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	got := LoadHistory(path)
	assert.Equal(t, []string{"list", "create foo", "run SBX-1 whoami"}, got)
}

func TestLoadHistory_ExceedsMax(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")

	// Write MaxHistoryEntries + 50 entries
	var lines []string
	total := MaxHistoryEntries + 50
	for i := 0; i < total; i++ {
		lines = append(lines, "cmd"+strings.Repeat("x", 3))
	}
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600))

	got := LoadHistory(path)
	assert.Len(t, got, MaxHistoryEntries)
	// Should be the last MaxHistoryEntries entries
	assert.Equal(t, lines[50], got[0])
	assert.Equal(t, lines[total-1], got[MaxHistoryEntries-1])

	// File on disk should also be truncated
	reloaded := LoadHistory(path)
	assert.Len(t, reloaded, MaxHistoryEntries)
}

func TestAppendHistory_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	AppendHistory(path, "hello")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(data))

	// Check permissions
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestAppendHistory_AppendsToExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	require.NoError(t, os.WriteFile(path, []byte("first\n"), 0o600))

	AppendHistory(path, "second")

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\n", string(data))
}

func TestAppendHistory_UnwritablePath(t *testing.T) {
	// Should not panic on unwritable path
	assert.NotPanics(t, func() {
		AppendHistory("/nonexistent/dir/history", "test")
	})
}

func TestIntegration_LoadAppendLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")

	// Start empty
	assert.Nil(t, LoadHistory(path))

	// Append entries
	AppendHistory(path, "list")
	AppendHistory(path, "create foo")
	AppendHistory(path, "run SBX-1 whoami")

	// Load and verify
	got := LoadHistory(path)
	assert.Equal(t, []string{"list", "create foo", "run SBX-1 whoami"}, got)

	// Append more
	AppendHistory(path, "destroy SBX-1")

	// Reload
	got = LoadHistory(path)
	assert.Equal(t, []string{"list", "create foo", "run SBX-1 whoami", "destroy SBX-1"}, got)
}
