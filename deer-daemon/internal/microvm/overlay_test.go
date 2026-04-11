package microvm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	_, err := CreateOverlay(context.Background(), "/nonexistent/base.qcow2", workDir, "test-id", 0)
	if err == nil {
		t.Error("expected error for missing base image")
	}
}

func TestCreateOverlay_CreatesOverlay(t *testing.T) {
	workDir := t.TempDir()
	baseImage := filepath.Join(workDir, "base.qcow2")
	if err := os.WriteFile(baseImage, []byte("base"), 0o644); err != nil {
		t.Fatalf("write base image: %v", err)
	}

	logPath := filepath.Join(workDir, "qemu-img.log")
	fakeQemuImg := filepath.Join(workDir, "qemu-img")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"" + logPath + "\"\n" +
		"case \"$1\" in\n" +
		"  create)\n" +
		"    : > \"$8\"\n" +
		"    ;;\n" +
		"esac\n"
	if err := os.WriteFile(fakeQemuImg, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake qemu-img: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})
	if err := os.Setenv("PATH", workDir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}

	overlayPath, err := CreateOverlay(context.Background(), baseImage, workDir, "test-id", 0)
	if err != nil {
		t.Fatalf("CreateOverlay returned error: %v", err)
	}
	if _, err := os.Stat(overlayPath); err != nil {
		t.Fatalf("overlay file missing: %v", err)
	}

	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read qemu-img log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(logBytes)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 qemu-img invocation (create only, no resize), got %d: %q", len(lines), string(logBytes))
	}
	if !strings.Contains(lines[0], "create -f qcow2 -b "+baseImage+" -F qcow2 "+overlayPath) {
		t.Fatalf("unexpected create invocation: %q", lines[0])
	}
}
