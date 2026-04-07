package snapshotpull

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/image"
	"github.com/aspectrr/deer.sh/deer-daemon/internal/state"

	"gorm.io/gorm"
)

// PullRequest describes what to pull.
type PullRequest struct {
	SourceHost   string // identifier for the source host (e.g. hostname)
	VMName       string // VM name on the source host
	SnapshotMode string // "cached" or "fresh"
}

// PullResult describes the outcome of a pull.
type PullResult struct {
	ImageName string
	Cached    bool
	PulledAt  time.Time
}

// inflightEntry tracks an in-progress pull and its outcome.
type inflightEntry struct {
	done   chan struct{}
	result *PullResult
	err    error
}

// Puller orchestrates snapshot pulls with caching and deduplication.
type Puller struct {
	imgStore *image.Store
	db       *gorm.DB
	logger   *slog.Logger

	mu       sync.Mutex
	inflight map[string]*inflightEntry
}

// NewPuller creates a new Puller.
func NewPuller(imgStore *image.Store, db *gorm.DB, logger *slog.Logger) *Puller {
	if logger == nil {
		logger = slog.Default()
	}
	return &Puller{
		imgStore: imgStore,
		db:       db,
		logger:   logger.With("component", "puller"),
		inflight: make(map[string]*inflightEntry),
	}
}

// Pull pulls a VM snapshot image, using the cache when appropriate.
// Concurrent pulls for the same image are deduplicated.
//
// Fresh mode: try live pull first, fall back to cache on failure.
// Cached mode: try cache first, pull on miss.
func (p *Puller) Pull(ctx context.Context, req PullRequest, backend SnapshotBackend) (*PullResult, error) {
	imageName := cacheKey(req.SourceHost, req.VMName)

	if req.SnapshotMode == "fresh" {
		result, err := p.pullOrWait(ctx, imageName, req, backend)
		if err == nil {
			return result, nil
		}
		p.logger.Warn("live pull failed, falling back to cache", "image", imageName, "error", err)
		if cached, ok := p.checkCache(ctx, imageName); ok {
			return cached, nil
		}
		return nil, err
	}

	// Cached mode: check cache first, pull on miss
	if result, ok := p.checkCache(ctx, imageName); ok {
		p.logger.Info("cache hit", "image", imageName)
		return result, nil
	}
	return p.pullOrWait(ctx, imageName, req, backend)
}

// pullOrWait performs a live pull with inflight deduplication.
func (p *Puller) pullOrWait(ctx context.Context, imageName string, req PullRequest, backend SnapshotBackend) (*PullResult, error) {
	p.mu.Lock()
	if entry, ok := p.inflight[imageName]; ok {
		p.mu.Unlock()
		select {
		case <-entry.done:
			return entry.result, entry.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	entry := &inflightEntry{done: make(chan struct{})}
	p.inflight[imageName] = entry
	p.mu.Unlock()

	result, err := p.doPull(ctx, imageName, req, backend)

	entry.result = result
	entry.err = err
	close(entry.done)

	p.mu.Lock()
	delete(p.inflight, imageName)
	p.mu.Unlock()

	return result, err
}

// doPull performs the actual snapshot pull.
func (p *Puller) doPull(ctx context.Context, imageName string, req PullRequest, backend SnapshotBackend) (*PullResult, error) {
	p.logger.Info("pulling snapshot", "image", imageName, "vm", req.VMName, "host", req.SourceHost)

	destPath := p.imgStore.BaseDir() + "/" + imageName + ".qcow2"

	// Snapshot and pull
	if err := backend.SnapshotAndPull(ctx, req.VMName, destPath); err != nil {
		return nil, fmt.Errorf("snapshot and pull: %w", err)
	}

	// Kernel extraction is no longer needed - microVM provider uses a
	// pre-downloaded kernel configured via microvm.kernel_path.

	// Get file size
	var sizeMB int64
	if info, err := os.Stat(destPath); err == nil {
		sizeMB = info.Size() / (1024 * 1024)
	}

	// Save to cache DB
	now := time.Now().UTC()
	cached := state.CachedImage{
		ID:         imageName,
		ImageName:  imageName,
		SourceHost: req.SourceHost,
		VMName:     req.VMName,
		SizeMB:     sizeMB,
		PulledAt:   now,
	}

	// Upsert
	if err := p.db.Where("image_name = ?", imageName).
		Assign(cached).
		FirstOrCreate(&cached).Error; err != nil {
		p.logger.Warn("failed to save cache metadata", "image", imageName, "error", err)
	}

	p.logger.Info("pull complete", "image", imageName, "size_mb", sizeMB)

	return &PullResult{
		ImageName: imageName,
		Cached:    false,
		PulledAt:  now,
	}, nil
}

// checkCache checks if an image is already cached and the file exists.
func (p *Puller) checkCache(ctx context.Context, imageName string) (*PullResult, bool) {
	var cached state.CachedImage
	if err := p.db.WithContext(ctx).Where("image_name = ?", imageName).First(&cached).Error; err != nil {
		return nil, false
	}

	// Verify the image still exists on disk
	if !p.imgStore.HasImage(imageName) {
		_ = p.db.Delete(&cached).Error
		return nil, false
	}

	return &PullResult{
		ImageName: cached.ImageName,
		Cached:    true,
		PulledAt:  cached.PulledAt,
	}, true
}

// cacheKey generates a sanitized cache key from host + vm name.
func cacheKey(host, vmName string) string {
	safe := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	h := safe.ReplaceAllString(host, "-")
	v := safe.ReplaceAllString(vmName, "-")
	return fmt.Sprintf("snap-%s-%s", strings.ToLower(h), strings.ToLower(v))
}
