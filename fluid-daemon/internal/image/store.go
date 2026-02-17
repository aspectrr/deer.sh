// Package image manages base QCOW2 images and kernel extraction.
package image

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Store manages base QCOW2 images available for sandbox creation.
type Store struct {
	baseDir string
	logger  *slog.Logger
}

// ImageInfo describes a base image.
type ImageInfo struct {
	Name      string // filename without extension
	Path      string // full path to QCOW2 file
	SizeMB    int64  // file size in MB
	HasKernel bool   // whether a kernel has been extracted
}

// NewStore creates an image store for the given base directory.
func NewStore(baseDir string, logger *slog.Logger) (*Store, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create image dir: %w", err)
	}

	return &Store{
		baseDir: baseDir,
		logger:  logger.With("component", "image"),
	}, nil
}

// List returns all available base images.
func (s *Store) List() ([]ImageInfo, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read image dir: %w", err)
	}

	var images []ImageInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".qcow2") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".qcow2")
		fullPath := filepath.Join(s.baseDir, entry.Name())

		// Check for extracted kernel
		kernelPath := filepath.Join(s.baseDir, name+".vmlinux")
		hasKernel := fileExists(kernelPath)

		images = append(images, ImageInfo{
			Name:      name,
			Path:      fullPath,
			SizeMB:    info.Size() / (1024 * 1024),
			HasKernel: hasKernel,
		})
	}

	return images, nil
}

// ListNames returns just the names of available base images.
func (s *Store) ListNames() ([]string, error) {
	images, err := s.List()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(images))
	for i, img := range images {
		names[i] = img.Name
	}
	return names, nil
}

// GetImagePath returns the full path to a base image by name.
func (s *Store) GetImagePath(name string) (string, error) {
	path := filepath.Join(s.baseDir, name+".qcow2")
	if !fileExists(path) {
		// Try without adding .qcow2 (in case name already has extension)
		if fileExists(filepath.Join(s.baseDir, name)) {
			return filepath.Join(s.baseDir, name), nil
		}
		return "", fmt.Errorf("base image %q not found in %s", name, s.baseDir)
	}
	return path, nil
}

// GetKernelPath returns the path to the extracted kernel for a base image.
func (s *Store) GetKernelPath(name string) (string, error) {
	path := filepath.Join(s.baseDir, name+".vmlinux")
	if !fileExists(path) {
		return "", fmt.Errorf("kernel for %q not found (run kernel extraction first)", name)
	}
	return path, nil
}

// HasImage checks if a base image exists.
func (s *Store) HasImage(name string) bool {
	_, err := s.GetImagePath(name)
	return err == nil
}

// BaseDir returns the base image directory.
func (s *Store) BaseDir() string {
	return s.baseDir
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
