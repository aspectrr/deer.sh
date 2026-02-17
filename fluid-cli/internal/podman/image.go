package podman

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aspectrr/fluid.sh/fluid/internal/workflow"
)

// ImageBuilder handles Podman image building operations.
type ImageBuilder struct {
	// podmanPath is the path to the podman binary.
	podmanPath string
}

// ImageBuilderConfig configures the image builder.
type ImageBuilderConfig struct {
	// PodmanPath is the path to the podman binary.
	// If empty, "podman" is looked up in PATH.
	PodmanPath string
}

// NewImageBuilder creates a new ImageBuilder with the given configuration.
func NewImageBuilder(cfg ImageBuilderConfig) *ImageBuilder {
	podmanPath := cfg.PodmanPath
	if podmanPath == "" {
		podmanPath = "podman"
	}
	return &ImageBuilder{
		podmanPath: podmanPath,
	}
}

// ImageResult contains the result of building an image.
type ImageResult struct {
	// ImageID is the full image ID.
	ImageID string

	// ImageTag is the human-readable image tag (e.g., "vmclone/node-c:20251215T183000Z").
	ImageTag string

	// Cleanup is a function to remove the image.
	Cleanup workflow.CleanupFunc
}

// BuildImage builds a Podman image from a root filesystem archive.
// The image is tagged as vmclone/<vmName>:<UTC timestamp>.
func (b *ImageBuilder) BuildImage(ctx context.Context, archivePath string, vmName string, workDir string) (*ImageResult, error) {
	// Generate image tag with timestamp
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	imageTag := fmt.Sprintf("vmclone/%s:%s", vmName, timestamp)

	// Create a temporary directory for the build context
	buildDir := filepath.Join(workDir, "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, workflow.NewWorkflowError(
			workflow.StageBuildImage,
			workflow.ErrImageBuildFailed,
			fmt.Sprintf("failed to create build directory: %v", err),
		)
	}
	defer func() { _ = os.RemoveAll(buildDir) }()

	// Copy or link the archive to the build context
	archiveBaseName := filepath.Base(archivePath)
	buildArchivePath := filepath.Join(buildDir, archiveBaseName)

	// Create a hard link if possible, otherwise copy
	if err := os.Link(archivePath, buildArchivePath); err != nil {
		// Fall back to copy if hard link fails (e.g., cross-filesystem)
		if err := copyFile(archivePath, buildArchivePath); err != nil {
			return nil, workflow.NewWorkflowError(
				workflow.StageBuildImage,
				workflow.ErrImageBuildFailed,
				fmt.Sprintf("failed to copy archive to build context: %v", err),
			)
		}
	}

	// Generate Containerfile
	containerfile := generateContainerfile(archiveBaseName)
	containerfilePath := filepath.Join(buildDir, "Containerfile")
	if err := os.WriteFile(containerfilePath, []byte(containerfile), 0o644); err != nil {
		return nil, workflow.NewWorkflowError(
			workflow.StageBuildImage,
			workflow.ErrImageBuildFailed,
			fmt.Sprintf("failed to write Containerfile: %v", err),
		)
	}

	// Build the image
	imageID, err := b.buildImage(ctx, buildDir, imageTag)
	if err != nil {
		return nil, workflow.NewWorkflowError(
			workflow.StageBuildImage,
			workflow.ErrImageBuildFailed,
			fmt.Sprintf("podman build failed: %v", err),
		)
	}

	return &ImageResult{
		ImageID:  imageID,
		ImageTag: imageTag,
		Cleanup: func() error {
			return b.RemoveImage(context.Background(), imageTag)
		},
	}, nil
}

// generateContainerfile creates a Containerfile for importing the root filesystem.
func generateContainerfile(archiveName string) string {
	// Use scratch as base - we're importing a complete root filesystem
	// ADD automatically extracts tar archives
	return fmt.Sprintf(`# Auto-generated Containerfile for VM clone
FROM scratch

# Add the root filesystem archive
ADD %s /

# Set container environment marker
ENV container=podman

# Default to shell - can be overridden at runtime
CMD ["/bin/sh"]
`, archiveName)
}

// buildImage executes podman build and returns the image ID.
func (b *ImageBuilder) buildImage(ctx context.Context, buildDir string, imageTag string) (string, error) {
	args := []string{
		"build",
		"--tag", imageTag,
		"--file", "Containerfile",
		"--format", "oci",
		"--quiet",
		buildDir,
	}

	cmd := exec.CommandContext(ctx, b.podmanPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("build failed: %w: %s", err, stderr.String())
	}

	// podman build --quiet outputs just the image ID
	imageID := strings.TrimSpace(stdout.String())
	if imageID == "" {
		// If --quiet didn't give us an ID, inspect the image
		return b.getImageID(ctx, imageTag)
	}

	return imageID, nil
}

// getImageID retrieves the full image ID for a given tag.
func (b *ImageBuilder) getImageID(ctx context.Context, imageTag string) (string, error) {
	args := []string{
		"inspect",
		"--format", "{{.Id}}",
		imageTag,
	}

	cmd := exec.CommandContext(ctx, b.podmanPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("inspect failed: %w: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RemoveImage removes a Podman image by tag or ID.
func (b *ImageBuilder) RemoveImage(ctx context.Context, imageRef string) error {
	args := []string{
		"rmi",
		"--force",
		imageRef,
	}

	cmd := exec.CommandContext(ctx, b.podmanPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("image removal failed: %w: %s", err, stderr.String())
	}

	return nil
}

// ImageExists checks if an image with the given tag exists.
func (b *ImageBuilder) ImageExists(ctx context.Context, imageTag string) (bool, error) {
	args := []string{
		"image",
		"exists",
		imageTag,
	}

	cmd := exec.CommandContext(ctx, b.podmanPath, args...)
	err := cmd.Run()

	if err == nil {
		return true, nil
	}

	// Exit code 1 means image doesn't exist
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
	}

	return false, fmt.Errorf("failed to check image existence: %w", err)
}

// ListImages lists images matching a filter pattern.
func (b *ImageBuilder) ListImages(ctx context.Context, filter string) ([]string, error) {
	args := []string{
		"images",
		"--format", "{{.Repository}}:{{.Tag}}",
	}

	if filter != "" {
		args = append(args, "--filter", fmt.Sprintf("reference=%s", filter))
	}

	cmd := exec.CommandContext(ctx, b.podmanPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("list images failed: %w: %s", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// GetImageInfo retrieves information about an image.
type ImageInfo struct {
	ID      string
	Tag     string
	Created time.Time
	Size    int64
}

// InspectImage retrieves detailed information about an image.
func (b *ImageBuilder) InspectImage(ctx context.Context, imageRef string) (*ImageInfo, error) {
	args := []string{
		"inspect",
		"--format", "{{.Id}}|{{.Created}}|{{.Size}}",
		imageRef,
	}

	cmd := exec.CommandContext(ctx, b.podmanPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("inspect failed: %w: %s", err, stderr.String())
	}

	parts := strings.Split(strings.TrimSpace(stdout.String()), "|")
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected inspect output format")
	}

	created, _ := time.Parse(time.RFC3339, parts[1])
	var size int64
	if _, err := fmt.Sscanf(parts[2], "%d", &size); err != nil {
		return nil, fmt.Errorf("failed to parse size: %w", err)
	}

	return &ImageInfo{
		ID:      parts[0],
		Tag:     imageRef,
		Created: created,
		Size:    size,
	}, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
