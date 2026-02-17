package microvm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveOverlay(t *testing.T) {
	workDir := t.TempDir()
	sandboxID := "test-sandbox"

	// Create sandbox dir with files
	sandboxDir := filepath.Join(workDir, sandboxID)
	if err := os.MkdirAll(sandboxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sandboxDir, "disk.qcow2"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Remove
	if err := RemoveOverlay(workDir, sandboxID); err != nil {
		t.Fatal(err)
	}

	// Verify removed
	if _, err := os.Stat(sandboxDir); !os.IsNotExist(err) {
		t.Error("sandbox dir should be removed")
	}
}

func TestCreateOverlay_MissingBase(t *testing.T) {
	workDir := t.TempDir()
	_, err := CreateOverlay(context.Background(), "/nonexistent/base.qcow2", workDir, "test-id")
	if err == nil {
		t.Error("expected error for missing base image")
	}
}
