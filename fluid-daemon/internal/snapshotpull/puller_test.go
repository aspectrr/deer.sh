package snapshotpull

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/image"
	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/state"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// mockBackend records calls and writes a dummy file.
type mockBackend struct {
	callCount atomic.Int32
	delay     time.Duration
	failErr   error
}

func (m *mockBackend) SnapshotAndPull(_ context.Context, vmName string, destPath string) error {
	m.callCount.Add(1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.failErr != nil {
		return m.failErr
	}
	// Write a dummy file
	return os.WriteFile(destPath, []byte("fake-qcow2-data"), 0o644)
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&state.CachedImage{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func setupTestImageStore(t *testing.T) *image.Store {
	t.Helper()
	dir := t.TempDir()
	store, err := image.NewStore(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestCacheKey(t *testing.T) {
	tests := []struct {
		host, vm string
		want     string
	}{
		{"host1.example.com", "my-vm", "snap-host1-example-com-my-vm"},
		{"10.0.0.1", "test_vm", "snap-10-0-0-1-test_vm"},
		{"HOST", "VM", "snap-host-vm"},
	}
	for _, tt := range tests {
		got := cacheKey(tt.host, tt.vm)
		if got != tt.want {
			t.Errorf("cacheKey(%q, %q) = %q, want %q", tt.host, tt.vm, got, tt.want)
		}
	}
}

func TestPuller_FreshPull(t *testing.T) {
	db := setupTestDB(t)
	imgStore := setupTestImageStore(t)
	puller := NewPuller(imgStore, db, nil)
	backend := &mockBackend{}

	req := PullRequest{
		SourceHost:   "host1",
		VMName:       "vm1",
		SnapshotMode: "fresh",
	}

	result, err := puller.Pull(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Cached {
		t.Error("expected Cached=false for fresh pull")
	}
	if result.ImageName == "" {
		t.Error("expected non-empty ImageName")
	}
	if backend.callCount.Load() != 1 {
		t.Errorf("expected 1 backend call, got %d", backend.callCount.Load())
	}

	// Verify file exists
	path := filepath.Join(imgStore.BaseDir(), result.ImageName+".qcow2")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected image file to exist")
	}

	// Verify DB record
	var cached state.CachedImage
	if err := db.Where("image_name = ?", result.ImageName).First(&cached).Error; err != nil {
		t.Errorf("expected cache record in DB: %v", err)
	}
}

func TestPuller_CachedHit(t *testing.T) {
	db := setupTestDB(t)
	imgStore := setupTestImageStore(t)
	puller := NewPuller(imgStore, db, nil)
	backend := &mockBackend{}

	req := PullRequest{
		SourceHost:   "host1",
		VMName:       "vm1",
		SnapshotMode: "cached",
	}

	// First pull - miss
	result1, err := puller.Pull(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("first pull error: %v", err)
	}
	if result1.Cached {
		t.Error("first pull should not be cached")
	}

	// Second pull - should be a cache hit
	result2, err := puller.Pull(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("second pull error: %v", err)
	}
	if !result2.Cached {
		t.Error("second pull should be cached")
	}

	// Backend should only be called once
	if backend.callCount.Load() != 1 {
		t.Errorf("expected 1 backend call, got %d", backend.callCount.Load())
	}
}

func TestPuller_FreshBypassesCache(t *testing.T) {
	db := setupTestDB(t)
	imgStore := setupTestImageStore(t)
	puller := NewPuller(imgStore, db, nil)
	backend := &mockBackend{}

	// First pull to populate cache
	req := PullRequest{
		SourceHost:   "host1",
		VMName:       "vm1",
		SnapshotMode: "cached",
	}
	_, err := puller.Pull(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("first pull error: %v", err)
	}

	// Fresh pull should bypass cache
	req.SnapshotMode = "fresh"
	result, err := puller.Pull(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("fresh pull error: %v", err)
	}
	if result.Cached {
		t.Error("fresh pull should not report cached")
	}
	if backend.callCount.Load() != 2 {
		t.Errorf("expected 2 backend calls, got %d", backend.callCount.Load())
	}
}

func TestPuller_DeduplicatesConcurrent(t *testing.T) {
	db := setupTestDB(t)
	imgStore := setupTestImageStore(t)
	puller := NewPuller(imgStore, db, nil)
	backend := &mockBackend{delay: 100 * time.Millisecond}

	req := PullRequest{
		SourceHost:   "host1",
		VMName:       "vm1",
		SnapshotMode: "fresh",
	}

	var wg sync.WaitGroup
	results := make([]*PullResult, 5)
	errs := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = puller.Pull(context.Background(), req, backend)
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d error: %v", i, err)
		}
	}

	// Backend should only be called once despite 5 concurrent requests
	if backend.callCount.Load() != 1 {
		t.Errorf("expected 1 backend call (deduped), got %d", backend.callCount.Load())
	}
}

func TestPuller_BackendError(t *testing.T) {
	db := setupTestDB(t)
	imgStore := setupTestImageStore(t)
	puller := NewPuller(imgStore, db, nil)
	backend := &mockBackend{failErr: fmt.Errorf("connection refused")}

	req := PullRequest{
		SourceHost:   "host1",
		VMName:       "vm1",
		SnapshotMode: "fresh",
	}

	_, err := puller.Pull(context.Background(), req, backend)
	if err == nil {
		t.Fatal("expected error from backend failure")
	}
}

func TestPuller_CacheMissWhenFileDeleted(t *testing.T) {
	db := setupTestDB(t)
	imgStore := setupTestImageStore(t)
	puller := NewPuller(imgStore, db, nil)
	backend := &mockBackend{}

	req := PullRequest{
		SourceHost:   "host1",
		VMName:       "vm1",
		SnapshotMode: "cached",
	}

	// First pull
	result1, err := puller.Pull(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("first pull error: %v", err)
	}

	// Delete the file manually
	path := filepath.Join(imgStore.BaseDir(), result1.ImageName+".qcow2")
	_ = os.Remove(path)

	// Second pull should be a cache miss
	result2, err := puller.Pull(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("second pull error: %v", err)
	}
	if result2.Cached {
		t.Error("expected cache miss when file deleted")
	}
	if backend.callCount.Load() != 2 {
		t.Errorf("expected 2 backend calls, got %d", backend.callCount.Load())
	}
}
