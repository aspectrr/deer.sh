package image

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func createFile(t *testing.T, path string, sizeBytes int) {
	t.Helper()
	data := make([]byte, sizeBytes)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}
}

func TestNewStore_CreatesDir(t *testing.T) {
	base := filepath.Join(t.TempDir(), "images", "nested")

	s, err := NewStore(base, slog.Default())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	info, err := os.Stat(s.BaseDir())
	if err != nil {
		t.Fatalf("expected directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", s.BaseDir())
	}
}

func TestList_Empty(t *testing.T) {
	base := t.TempDir()

	s, err := NewStore(base, slog.Default())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	images, err := s.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(images) != 0 {
		t.Errorf("expected 0 images, got %d", len(images))
	}
}

func TestList_WithImages(t *testing.T) {
	base := t.TempDir()

	// Create two qcow2 files, one with a kernel.
	createFile(t, filepath.Join(base, "ubuntu.qcow2"), 2*1024*1024)
	createFile(t, filepath.Join(base, "debian.qcow2"), 1*1024*1024)
	createFile(t, filepath.Join(base, "ubuntu.vmlinux"), 100)

	// Also create a non-qcow2 file that should be ignored.
	createFile(t, filepath.Join(base, "notes.txt"), 10)

	s, err := NewStore(base, slog.Default())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	images, err := s.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}

	byName := make(map[string]ImageInfo)
	for _, img := range images {
		byName[img.Name] = img
	}

	// Check ubuntu entry.
	ubuntu, ok := byName["ubuntu"]
	if !ok {
		t.Fatal("expected image named 'ubuntu'")
	}
	if !ubuntu.HasKernel {
		t.Error("expected ubuntu to have kernel")
	}
	if ubuntu.SizeMB != 2 {
		t.Errorf("expected ubuntu SizeMB=2, got %d", ubuntu.SizeMB)
	}
	if ubuntu.Path != filepath.Join(base, "ubuntu.qcow2") {
		t.Errorf("unexpected path: %s", ubuntu.Path)
	}

	// Check debian entry.
	debian, ok := byName["debian"]
	if !ok {
		t.Fatal("expected image named 'debian'")
	}
	if debian.HasKernel {
		t.Error("expected debian to NOT have kernel")
	}
	if debian.SizeMB != 1 {
		t.Errorf("expected debian SizeMB=1, got %d", debian.SizeMB)
	}
}

func TestListNames(t *testing.T) {
	base := t.TempDir()

	createFile(t, filepath.Join(base, "alpha.qcow2"), 100)
	createFile(t, filepath.Join(base, "beta.qcow2"), 100)

	s, err := NewStore(base, slog.Default())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	names, err := s.ListNames()
	if err != nil {
		t.Fatalf("ListNames failed: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["alpha"] || !nameSet["beta"] {
		t.Errorf("expected alpha and beta, got %v", names)
	}
}

func TestGetImagePath(t *testing.T) {
	base := t.TempDir()

	createFile(t, filepath.Join(base, "myimage.qcow2"), 100)

	s, err := NewStore(base, slog.Default())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	path, err := s.GetImagePath("myimage")
	if err != nil {
		t.Fatalf("GetImagePath failed: %v", err)
	}

	expected := filepath.Join(base, "myimage.qcow2")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestGetImagePath_Missing(t *testing.T) {
	base := t.TempDir()

	s, err := NewStore(base, slog.Default())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	_, err = s.GetImagePath("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing image, got nil")
	}
}

func TestGetKernelPath(t *testing.T) {
	base := t.TempDir()

	createFile(t, filepath.Join(base, "myimage.qcow2"), 100)
	createFile(t, filepath.Join(base, "myimage.vmlinux"), 100)

	s, err := NewStore(base, slog.Default())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	path, err := s.GetKernelPath("myimage")
	if err != nil {
		t.Fatalf("GetKernelPath failed: %v", err)
	}

	expected := filepath.Join(base, "myimage.vmlinux")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}

	// Missing kernel should error.
	_, err = s.GetKernelPath("nope")
	if err == nil {
		t.Fatal("expected error for missing kernel, got nil")
	}
}

func TestHasImage(t *testing.T) {
	base := t.TempDir()

	createFile(t, filepath.Join(base, "present.qcow2"), 100)

	s, err := NewStore(base, slog.Default())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if !s.HasImage("present") {
		t.Error("expected HasImage to return true for existing image")
	}
	if s.HasImage("absent") {
		t.Error("expected HasImage to return false for missing image")
	}
}

func TestBaseDir(t *testing.T) {
	base := t.TempDir()

	s, err := NewStore(base, slog.Default())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if s.BaseDir() != base {
		t.Errorf("expected BaseDir=%s, got %s", base, s.BaseDir())
	}
}
