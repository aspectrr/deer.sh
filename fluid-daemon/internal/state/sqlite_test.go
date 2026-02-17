package state

import (
	"context"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore(:memory:) failed: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewStore(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	if store.db == nil {
		t.Fatal("expected db to be non-nil")
	}

	// Verify tables were created by checking we can query them.
	var count int64
	if err := store.db.Model(&Sandbox{}).Count(&count).Error; err != nil {
		t.Fatalf("sandbox table query failed: %v", err)
	}
	if err := store.db.Model(&Command{}).Count(&count).Error; err != nil {
		t.Fatalf("command table query failed: %v", err)
	}
}

func TestCreateSandbox_GetSandbox(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	sb := &Sandbox{
		ID:         "SBX-test1",
		Name:       "test-sandbox",
		AgentID:    "agent-1",
		BaseImage:  "/images/ubuntu.qcow2",
		Bridge:     "br0",
		TAPDevice:  "tap0",
		MACAddress: "52:54:00:00:00:01",
		IPAddress:  "192.168.1.10",
		State:      "RUNNING",
		PID:        1234,
		VCPUs:      2,
		MemoryMB:   2048,
		TTLSeconds: 3600,
	}

	if err := store.CreateSandbox(ctx, sb); err != nil {
		t.Fatalf("CreateSandbox failed: %v", err)
	}

	got, err := store.GetSandbox(ctx, "SBX-test1")
	if err != nil {
		t.Fatalf("GetSandbox failed: %v", err)
	}

	if got.ID != sb.ID {
		t.Errorf("ID = %q, want %q", got.ID, sb.ID)
	}
	if got.Name != sb.Name {
		t.Errorf("Name = %q, want %q", got.Name, sb.Name)
	}
	if got.AgentID != sb.AgentID {
		t.Errorf("AgentID = %q, want %q", got.AgentID, sb.AgentID)
	}
	if got.BaseImage != sb.BaseImage {
		t.Errorf("BaseImage = %q, want %q", got.BaseImage, sb.BaseImage)
	}
	if got.Bridge != sb.Bridge {
		t.Errorf("Bridge = %q, want %q", got.Bridge, sb.Bridge)
	}
	if got.TAPDevice != sb.TAPDevice {
		t.Errorf("TAPDevice = %q, want %q", got.TAPDevice, sb.TAPDevice)
	}
	if got.MACAddress != sb.MACAddress {
		t.Errorf("MACAddress = %q, want %q", got.MACAddress, sb.MACAddress)
	}
	if got.IPAddress != sb.IPAddress {
		t.Errorf("IPAddress = %q, want %q", got.IPAddress, sb.IPAddress)
	}
	if got.State != sb.State {
		t.Errorf("State = %q, want %q", got.State, sb.State)
	}
	if got.PID != sb.PID {
		t.Errorf("PID = %d, want %d", got.PID, sb.PID)
	}
	if got.VCPUs != sb.VCPUs {
		t.Errorf("VCPUs = %d, want %d", got.VCPUs, sb.VCPUs)
	}
	if got.MemoryMB != sb.MemoryMB {
		t.Errorf("MemoryMB = %d, want %d", got.MemoryMB, sb.MemoryMB)
	}
	if got.TTLSeconds != sb.TTLSeconds {
		t.Errorf("TTLSeconds = %d, want %d", got.TTLSeconds, sb.TTLSeconds)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestCreateSandbox_GetSandbox_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.GetSandbox(ctx, "SBX-nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent sandbox, got nil")
	}
}

func TestListSandboxes(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	sb1 := &Sandbox{ID: "SBX-list1", Name: "sb1", State: "RUNNING"}
	sb2 := &Sandbox{ID: "SBX-list2", Name: "sb2", State: "RUNNING"}
	sb3 := &Sandbox{ID: "SBX-list3", Name: "sb3", State: "RUNNING"}

	for _, sb := range []*Sandbox{sb1, sb2, sb3} {
		if err := store.CreateSandbox(ctx, sb); err != nil {
			t.Fatalf("CreateSandbox(%s) failed: %v", sb.ID, err)
		}
	}

	// Soft-delete one sandbox.
	if err := store.DeleteSandbox(ctx, "SBX-list2"); err != nil {
		t.Fatalf("DeleteSandbox failed: %v", err)
	}

	list, err := store.ListSandboxes(ctx)
	if err != nil {
		t.Fatalf("ListSandboxes failed: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("ListSandboxes returned %d sandboxes, want 2", len(list))
	}

	ids := map[string]bool{}
	for _, sb := range list {
		ids[sb.ID] = true
	}
	if !ids["SBX-list1"] {
		t.Error("expected SBX-list1 in list")
	}
	if !ids["SBX-list3"] {
		t.Error("expected SBX-list3 in list")
	}
	if ids["SBX-list2"] {
		t.Error("SBX-list2 should not be in list (soft-deleted)")
	}
}

func TestUpdateSandbox(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	sb := &Sandbox{
		ID:        "SBX-update1",
		Name:      "before-update",
		State:     "RUNNING",
		IPAddress: "10.0.0.1",
		VCPUs:     1,
		MemoryMB:  512,
	}
	if err := store.CreateSandbox(ctx, sb); err != nil {
		t.Fatalf("CreateSandbox failed: %v", err)
	}

	// Modify fields and update.
	sb.Name = "after-update"
	sb.State = "STOPPED"
	sb.IPAddress = "10.0.0.2"
	sb.VCPUs = 4
	sb.MemoryMB = 4096

	if err := store.UpdateSandbox(ctx, sb); err != nil {
		t.Fatalf("UpdateSandbox failed: %v", err)
	}

	got, err := store.GetSandbox(ctx, "SBX-update1")
	if err != nil {
		t.Fatalf("GetSandbox failed: %v", err)
	}

	if got.Name != "after-update" {
		t.Errorf("Name = %q, want %q", got.Name, "after-update")
	}
	if got.State != "STOPPED" {
		t.Errorf("State = %q, want %q", got.State, "STOPPED")
	}
	if got.IPAddress != "10.0.0.2" {
		t.Errorf("IPAddress = %q, want %q", got.IPAddress, "10.0.0.2")
	}
	if got.VCPUs != 4 {
		t.Errorf("VCPUs = %d, want 4", got.VCPUs)
	}
	if got.MemoryMB != 4096 {
		t.Errorf("MemoryMB = %d, want 4096", got.MemoryMB)
	}
}

func TestDeleteSandbox(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	sb := &Sandbox{ID: "SBX-del1", Name: "to-delete", State: "RUNNING"}
	if err := store.CreateSandbox(ctx, sb); err != nil {
		t.Fatalf("CreateSandbox failed: %v", err)
	}

	if err := store.DeleteSandbox(ctx, "SBX-del1"); err != nil {
		t.Fatalf("DeleteSandbox failed: %v", err)
	}

	// GetSandbox should not find it (deleted_at IS NULL filter).
	_, err := store.GetSandbox(ctx, "SBX-del1")
	if err == nil {
		t.Fatal("expected error after soft delete, got nil")
	}

	// Verify the record still exists with DESTROYED state and non-nil deleted_at
	// by querying without the deleted_at filter.
	var raw Sandbox
	if err := store.db.Where("id = ?", "SBX-del1").First(&raw).Error; err != nil {
		t.Fatalf("raw query failed: %v", err)
	}
	if raw.State != "DESTROYED" {
		t.Errorf("State = %q, want %q", raw.State, "DESTROYED")
	}
	if raw.DeletedAt == nil {
		t.Error("DeletedAt should be non-nil after soft delete")
	}
}

func TestListExpiredSandboxes(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()

	// Sandbox with custom TTL of 60s, created 2 minutes ago - expired.
	sb1 := &Sandbox{
		ID:         "SBX-exp1",
		Name:       "expired-custom-ttl",
		State:      "RUNNING",
		TTLSeconds: 60,
		CreatedAt:  now.Add(-2 * time.Minute),
	}

	// Sandbox with no custom TTL, created 2 minutes ago.
	// Will expire with a 1-minute default TTL.
	sb2 := &Sandbox{
		ID:        "SBX-exp2",
		Name:      "expired-default-ttl",
		State:     "RUNNING",
		CreatedAt: now.Add(-2 * time.Minute),
	}

	// Sandbox created just now - not expired.
	sb3 := &Sandbox{
		ID:         "SBX-fresh",
		Name:       "fresh",
		State:      "RUNNING",
		TTLSeconds: 3600,
		CreatedAt:  now,
	}

	// Sandbox that is already DESTROYED - should not appear.
	sb4 := &Sandbox{
		ID:        "SBX-destroyed",
		Name:      "destroyed",
		State:     "DESTROYED",
		CreatedAt: now.Add(-10 * time.Minute),
	}

	for _, sb := range []*Sandbox{sb1, sb2, sb3, sb4} {
		if err := store.CreateSandbox(ctx, sb); err != nil {
			t.Fatalf("CreateSandbox(%s) failed: %v", sb.ID, err)
		}
	}

	defaultTTL := 1 * time.Minute
	expired, err := store.ListExpiredSandboxes(ctx, defaultTTL)
	if err != nil {
		t.Fatalf("ListExpiredSandboxes failed: %v", err)
	}

	ids := map[string]bool{}
	for _, sb := range expired {
		ids[sb.ID] = true
	}

	if !ids["SBX-exp1"] {
		t.Error("SBX-exp1 should be expired (custom TTL 60s, created 2m ago)")
	}
	if !ids["SBX-exp2"] {
		t.Error("SBX-exp2 should be expired (default TTL 1m, created 2m ago)")
	}
	if ids["SBX-fresh"] {
		t.Error("SBX-fresh should not be expired")
	}
	if ids["SBX-destroyed"] {
		t.Error("SBX-destroyed should not appear (state=DESTROYED)")
	}

	if len(expired) != 2 {
		t.Errorf("expected 2 expired sandboxes, got %d", len(expired))
	}
}

func TestListExpiredSandboxes_NoExpired(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	now := time.Now().UTC()

	sb := &Sandbox{
		ID:         "SBX-noexp",
		Name:       "not-expired",
		State:      "RUNNING",
		TTLSeconds: 3600,
		CreatedAt:  now,
	}
	if err := store.CreateSandbox(ctx, sb); err != nil {
		t.Fatalf("CreateSandbox failed: %v", err)
	}

	expired, err := store.ListExpiredSandboxes(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("ListExpiredSandboxes failed: %v", err)
	}

	if len(expired) != 0 {
		t.Errorf("expected 0 expired sandboxes, got %d", len(expired))
	}
}

func TestCreateCommand_ListSandboxCommands(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Create a sandbox first (foreign key context).
	sb := &Sandbox{ID: "SBX-cmd1", Name: "cmd-sandbox", State: "RUNNING"}
	if err := store.CreateSandbox(ctx, sb); err != nil {
		t.Fatalf("CreateSandbox failed: %v", err)
	}

	now := time.Now().UTC()

	cmd1 := &Command{
		ID:         "CMD-1",
		SandboxID:  "SBX-cmd1",
		Command:    "whoami",
		Stdout:     "root\n",
		Stderr:     "",
		ExitCode:   0,
		DurationMS: 50,
		StartedAt:  now.Add(-2 * time.Second),
		EndedAt:    now.Add(-2*time.Second + 50*time.Millisecond),
	}
	cmd2 := &Command{
		ID:         "CMD-2",
		SandboxID:  "SBX-cmd1",
		Command:    "ls /tmp",
		Stdout:     "file1\nfile2\n",
		Stderr:     "",
		ExitCode:   0,
		DurationMS: 30,
		StartedAt:  now.Add(-1 * time.Second),
		EndedAt:    now.Add(-1*time.Second + 30*time.Millisecond),
	}
	// Command for a different sandbox - should not appear in results.
	cmd3 := &Command{
		ID:         "CMD-3",
		SandboxID:  "SBX-other",
		Command:    "echo hi",
		Stdout:     "hi\n",
		ExitCode:   0,
		DurationMS: 10,
		StartedAt:  now,
		EndedAt:    now.Add(10 * time.Millisecond),
	}

	for _, cmd := range []*Command{cmd1, cmd2, cmd3} {
		if err := store.CreateCommand(ctx, cmd); err != nil {
			t.Fatalf("CreateCommand(%s) failed: %v", cmd.ID, err)
		}
	}

	commands, err := store.ListSandboxCommands(ctx, "SBX-cmd1")
	if err != nil {
		t.Fatalf("ListSandboxCommands failed: %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}

	// Results are ordered by started_at DESC, so cmd2 should be first.
	if commands[0].ID != "CMD-2" {
		t.Errorf("first command ID = %q, want CMD-2 (most recent)", commands[0].ID)
	}
	if commands[1].ID != "CMD-1" {
		t.Errorf("second command ID = %q, want CMD-1", commands[1].ID)
	}

	// Verify fields on first command.
	c := commands[0]
	if c.Command != "ls /tmp" {
		t.Errorf("Command = %q, want %q", c.Command, "ls /tmp")
	}
	if c.Stdout != "file1\nfile2\n" {
		t.Errorf("Stdout = %q, want %q", c.Stdout, "file1\nfile2\n")
	}
	if c.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", c.ExitCode)
	}
	if c.DurationMS != 30 {
		t.Errorf("DurationMS = %d, want 30", c.DurationMS)
	}

	// List commands for a sandbox with none.
	empty, err := store.ListSandboxCommands(ctx, "SBX-nonexistent")
	if err != nil {
		t.Fatalf("ListSandboxCommands for empty sandbox failed: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 commands for nonexistent sandbox, got %d", len(empty))
	}
}
