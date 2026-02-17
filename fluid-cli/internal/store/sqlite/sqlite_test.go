package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aspectrr/fluid.sh/fluid/internal/store"
)

func setupTestStore(t *testing.T) (store.Store, func()) {
	t.Helper()

	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "fluid-sqlite-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "test.db")
	ctx := context.Background()

	s, err := New(ctx, store.Config{
		DatabaseURL: dbPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)

	cleanup := func() {
		_ = s.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return s, cleanup
}

func TestSandboxCRUD(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a sandbox
	sb := &store.Sandbox{
		ID:          "SBX-001",
		JobID:       "JOB-001",
		AgentID:     "agent-1",
		SandboxName: "test-sandbox",
		BaseImage:   "ubuntu-base",
		Network:     "default",
		State:       store.SandboxStateCreated,
	}

	err := s.CreateSandbox(ctx, sb)
	require.NoError(t, err)
	assert.False(t, sb.CreatedAt.IsZero())
	assert.False(t, sb.UpdatedAt.IsZero())

	// Get the sandbox
	got, err := s.GetSandbox(ctx, sb.ID)
	require.NoError(t, err)
	assert.Equal(t, sb.ID, got.ID)
	assert.Equal(t, sb.SandboxName, got.SandboxName)
	assert.Equal(t, sb.State, got.State)

	// Update state
	ip := "192.168.1.100"
	err = s.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, &ip)
	require.NoError(t, err)

	got, err = s.GetSandbox(ctx, sb.ID)
	require.NoError(t, err)
	assert.Equal(t, store.SandboxStateRunning, got.State)
	require.NotNil(t, got.IPAddress)
	assert.Equal(t, ip, *got.IPAddress)

	// List sandboxes
	sandboxes, err := s.ListSandboxes(ctx, store.SandboxFilter{}, nil)
	require.NoError(t, err)
	assert.Len(t, sandboxes, 1)

	// Filter by state
	runningState := store.SandboxStateRunning
	sandboxes, err = s.ListSandboxes(ctx, store.SandboxFilter{State: &runningState}, nil)
	require.NoError(t, err)
	assert.Len(t, sandboxes, 1)

	stoppedState := store.SandboxStateStopped
	sandboxes, err = s.ListSandboxes(ctx, store.SandboxFilter{State: &stoppedState}, nil)
	require.NoError(t, err)
	assert.Len(t, sandboxes, 0)

	// Delete sandbox
	err = s.DeleteSandbox(ctx, sb.ID)
	require.NoError(t, err)

	// Verify soft delete
	_, err = s.GetSandbox(ctx, sb.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)

	// Verify sandbox doesn't appear in list
	sandboxes, err = s.ListSandboxes(ctx, store.SandboxFilter{}, nil)
	require.NoError(t, err)
	assert.Len(t, sandboxes, 0)
}

func TestSnapshotCRUD(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create sandbox first
	sb := &store.Sandbox{
		ID:          "SBX-002",
		JobID:       "JOB-002",
		AgentID:     "agent-1",
		SandboxName: "snapshot-test",
		BaseImage:   "ubuntu-base",
		Network:     "default",
		State:       store.SandboxStateRunning,
	}
	require.NoError(t, s.CreateSandbox(ctx, sb))

	// Create snapshot
	snap := &store.Snapshot{
		ID:        "SNP-001",
		SandboxID: sb.ID,
		Name:      "initial",
		Kind:      store.SnapshotKindInternal,
		Ref:       "snapshot-ref-001",
		CreatedAt: time.Now().UTC(),
	}

	err := s.CreateSnapshot(ctx, snap)
	require.NoError(t, err)

	// Get snapshot
	got, err := s.GetSnapshot(ctx, snap.ID)
	require.NoError(t, err)
	assert.Equal(t, snap.Name, got.Name)
	assert.Equal(t, snap.Kind, got.Kind)

	// Get by name
	got, err = s.GetSnapshotByName(ctx, sb.ID, "initial")
	require.NoError(t, err)
	assert.Equal(t, snap.ID, got.ID)

	// List snapshots
	snapshots, err := s.ListSnapshots(ctx, sb.ID, nil)
	require.NoError(t, err)
	assert.Len(t, snapshots, 1)
}

func TestCommandCRUD(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create sandbox first
	sb := &store.Sandbox{
		ID:          "SBX-003",
		JobID:       "JOB-003",
		AgentID:     "agent-1",
		SandboxName: "command-test",
		BaseImage:   "ubuntu-base",
		Network:     "default",
		State:       store.SandboxStateRunning,
	}
	require.NoError(t, s.CreateSandbox(ctx, sb))

	// Save command
	cmd := &store.Command{
		ID:        "CMD-001",
		SandboxID: sb.ID,
		Command:   "whoami",
		Stdout:    "root\n",
		Stderr:    "",
		ExitCode:  0,
		StartedAt: time.Now().UTC(),
		EndedAt:   time.Now().UTC(),
	}

	err := s.SaveCommand(ctx, cmd)
	require.NoError(t, err)

	// Get command
	got, err := s.GetCommand(ctx, cmd.ID)
	require.NoError(t, err)
	assert.Equal(t, cmd.Command, got.Command)
	assert.Equal(t, cmd.ExitCode, got.ExitCode)

	// List commands
	commands, err := s.ListCommands(ctx, sb.ID, nil)
	require.NoError(t, err)
	assert.Len(t, commands, 1)
}

func TestDiffCRUD(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create sandbox first
	sb := &store.Sandbox{
		ID:          "SBX-004",
		JobID:       "JOB-004",
		AgentID:     "agent-1",
		SandboxName: "diff-test",
		BaseImage:   "ubuntu-base",
		Network:     "default",
		State:       store.SandboxStateRunning,
	}
	require.NoError(t, s.CreateSandbox(ctx, sb))

	// Save diff
	diff := &store.Diff{
		ID:           "DIF-001",
		SandboxID:    sb.ID,
		FromSnapshot: "snap1",
		ToSnapshot:   "snap2",
		DiffJSON: store.ChangeDiff{
			FilesAdded:    []string{"/etc/nginx/nginx.conf"},
			FilesModified: []string{"/etc/hosts"},
		},
		CreatedAt: time.Now().UTC(),
	}

	err := s.SaveDiff(ctx, diff)
	require.NoError(t, err)

	// Get diff
	got, err := s.GetDiff(ctx, diff.ID)
	require.NoError(t, err)
	assert.Equal(t, diff.FromSnapshot, got.FromSnapshot)
	assert.Equal(t, diff.ToSnapshot, got.ToSnapshot)
	assert.Len(t, got.DiffJSON.FilesAdded, 1)
	assert.Len(t, got.DiffJSON.FilesModified, 1)

	// Get by snapshots
	got, err = s.GetDiffBySnapshots(ctx, sb.ID, "snap1", "snap2")
	require.NoError(t, err)
	assert.Equal(t, diff.ID, got.ID)
}

func TestNotFoundErrors(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Sandbox not found
	_, err := s.GetSandbox(ctx, "nonexistent")
	assert.ErrorIs(t, err, store.ErrNotFound)

	// Snapshot not found
	_, err = s.GetSnapshot(ctx, "nonexistent")
	assert.ErrorIs(t, err, store.ErrNotFound)

	// Command not found
	_, err = s.GetCommand(ctx, "nonexistent")
	assert.ErrorIs(t, err, store.ErrNotFound)

	// Diff not found
	_, err = s.GetDiff(ctx, "nonexistent")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestDuplicateSandboxName(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	sb1 := &store.Sandbox{
		ID:          "SBX-005",
		JobID:       "JOB-005",
		AgentID:     "agent-1",
		SandboxName: "duplicate-name",
		BaseImage:   "ubuntu-base",
		Network:     "default",
		State:       store.SandboxStateCreated,
	}
	require.NoError(t, s.CreateSandbox(ctx, sb1))

	// Try to create another sandbox with same name
	sb2 := &store.Sandbox{
		ID:          "SBX-006",
		JobID:       "JOB-006",
		AgentID:     "agent-1",
		SandboxName: "duplicate-name",
		BaseImage:   "ubuntu-base",
		Network:     "default",
		State:       store.SandboxStateCreated,
	}
	err := s.CreateSandbox(ctx, sb2)
	assert.ErrorIs(t, err, store.ErrAlreadyExists)

	// After deleting the first, we should be able to reuse the name
	require.NoError(t, s.DeleteSandbox(ctx, sb1.ID))

	err = s.CreateSandbox(ctx, sb2)
	require.NoError(t, err)
}

func TestTransaction(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Transaction that succeeds
	err := s.WithTx(ctx, func(tx store.DataStore) error {
		sb := &store.Sandbox{
			ID:          "SBX-TX-001",
			JobID:       "JOB-TX-001",
			AgentID:     "agent-1",
			SandboxName: "tx-test",
			BaseImage:   "ubuntu-base",
			Network:     "default",
			State:       store.SandboxStateCreated,
		}
		return tx.CreateSandbox(ctx, sb)
	})
	require.NoError(t, err)

	// Verify sandbox was created
	_, err = s.GetSandbox(ctx, "SBX-TX-001")
	require.NoError(t, err)

	// Transaction that fails should rollback
	err = s.WithTx(ctx, func(tx store.DataStore) error {
		sb := &store.Sandbox{
			ID:          "SBX-TX-002",
			JobID:       "JOB-TX-002",
			AgentID:     "agent-1",
			SandboxName: "tx-fail-test",
			BaseImage:   "ubuntu-base",
			Network:     "default",
			State:       store.SandboxStateCreated,
		}
		if err := tx.CreateSandbox(ctx, sb); err != nil {
			return err
		}
		// Force a rollback
		return assert.AnError
	})
	assert.Error(t, err)

	// Verify sandbox was not created
	_, err = s.GetSandbox(ctx, "SBX-TX-002")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestSourceVMCRUD(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Get non-existent
	_, err := s.GetSourceVM(ctx, "nonexistent")
	assert.ErrorIs(t, err, store.ErrNotFound)

	// Upsert (insert)
	now := time.Now().UTC()
	hostName := "kvm-01"
	hostAddr := "192.168.1.10"
	prepJSON := `{"user_created":true}`
	caFP := "abc123"
	svm := &store.SourceVM{
		Name:          "ubuntu-base",
		HostName:      &hostName,
		HostAddress:   &hostAddr,
		Prepared:      true,
		PreparedAt:    &now,
		PrepareJSON:   &prepJSON,
		CAFingerprint: &caFP,
	}

	err = s.UpsertSourceVM(ctx, svm)
	require.NoError(t, err)
	assert.NotEmpty(t, svm.ID)

	// Get by name
	got, err := s.GetSourceVM(ctx, "ubuntu-base")
	require.NoError(t, err)
	assert.Equal(t, "ubuntu-base", got.Name)
	assert.True(t, got.Prepared)
	require.NotNil(t, got.HostName)
	assert.Equal(t, "kvm-01", *got.HostName)
	require.NotNil(t, got.CAFingerprint)
	assert.Equal(t, "abc123", *got.CAFingerprint)

	// Upsert (update) - change fingerprint
	newFP := "def456"
	svm.CAFingerprint = &newFP
	err = s.UpsertSourceVM(ctx, svm)
	require.NoError(t, err)

	got, err = s.GetSourceVM(ctx, "ubuntu-base")
	require.NoError(t, err)
	require.NotNil(t, got.CAFingerprint)
	assert.Equal(t, "def456", *got.CAFingerprint)

	// List
	svms, err := s.ListSourceVMs(ctx)
	require.NoError(t, err)
	assert.Len(t, svms, 1)
	assert.Equal(t, "ubuntu-base", svms[0].Name)

	// Insert another
	svm2 := &store.SourceVM{
		Name:     "debian-base",
		Prepared: false,
	}
	err = s.UpsertSourceVM(ctx, svm2)
	require.NoError(t, err)

	svms, err = s.ListSourceVMs(ctx)
	require.NoError(t, err)
	assert.Len(t, svms, 2)
	// Should be alphabetically ordered
	assert.Equal(t, "debian-base", svms[0].Name)
	assert.Equal(t, "ubuntu-base", svms[1].Name)
}

func TestSourceVMUpsertValidation(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Nil svm
	err := s.UpsertSourceVM(ctx, nil)
	assert.ErrorIs(t, err, store.ErrInvalid)

	// Empty name
	err = s.UpsertSourceVM(ctx, &store.SourceVM{})
	assert.ErrorIs(t, err, store.ErrInvalid)
}

func TestPing(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	err := s.Ping(ctx)
	require.NoError(t, err)
}
