package snapshotpull

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewLibvirtBackend_Defaults(t *testing.T) {
	b := NewLibvirtBackend("host1.example.com", 0, "", "", nil)

	if b.sshPort != 22 {
		t.Errorf("expected default port 22, got %d", b.sshPort)
	}
	if b.sshUser != "root" {
		t.Errorf("expected default user root, got %s", b.sshUser)
	}
	if b.sshHost != "host1.example.com" {
		t.Errorf("expected host host1.example.com, got %s", b.sshHost)
	}
	// Default port 22 is omitted from URI
	if !strings.Contains(b.virshURI, "qemu+ssh://root@host1.example.com/system") {
		t.Errorf("expected qemu+ssh URI with host, got %s", b.virshURI)
	}
	if !strings.Contains(b.virshURI, "no_tty=1") {
		t.Errorf("expected no_tty=1 in URI, got %s", b.virshURI)
	}
}

func TestNewLibvirtBackend_CustomValues(t *testing.T) {
	b := NewLibvirtBackend("10.0.0.1", 2222, "admin", "/home/admin/.ssh/id_rsa", nil)

	if b.sshPort != 2222 {
		t.Errorf("expected port 2222, got %d", b.sshPort)
	}
	if b.sshUser != "admin" {
		t.Errorf("expected user admin, got %s", b.sshUser)
	}
	if b.sshIdentityFile != "/home/admin/.ssh/id_rsa" {
		t.Errorf("expected identity file /home/admin/.ssh/id_rsa, got %s", b.sshIdentityFile)
	}
	// Non-standard port is included in URI
	if !strings.Contains(b.virshURI, "qemu+ssh://admin@10.0.0.1:2222/system") {
		t.Errorf("expected qemu+ssh URI with port, got %s", b.virshURI)
	}
	if !strings.Contains(b.virshURI, "keyfile=/home/admin/.ssh/id_rsa") {
		t.Errorf("expected keyfile in URI, got %s", b.virshURI)
	}
}

func TestNewLibvirtBackend_URINoIdentity(t *testing.T) {
	b := NewLibvirtBackend("host1", 22, "root", "", nil)
	if strings.Contains(b.virshURI, "keyfile") {
		t.Errorf("expected no keyfile in URI when identity is empty, got %s", b.virshURI)
	}
}

func TestFindCleanDiskPath_UnreachableHost(t *testing.T) {
	b := NewLibvirtBackend("host1", 22, "root", "", nil)

	// findCleanDiskPath should return an error with unreachable host, not panic
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := b.findCleanDiskPath(ctx, "test-vm")
	if err == nil {
		t.Fatal("expected error with unreachable host")
	}
}

func TestCleanupAllFluidSnapshots_UnreachableHost(t *testing.T) {
	b := NewLibvirtBackend("host1", 22, "root", "", nil)

	// cleanupAllDeerSnapshots should not panic with unreachable host
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should not panic
	b.cleanupAllDeerSnapshots(ctx, "test-vm")
}

func TestIsFluidOverlay(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/var/lib/libvirt/images/test-vm-1.qcow2", false},
		{"/var/lib/libvirt/images/test-vm-1.deer-tmp-snap", true},
		{"/var/lib/libvirt/images/test-vm-1.deer-tmp-snap-1709000000", true},
		{"/data/vms/myvm.raw", false},
		{"/data/vms/myvm.deer-tmp-snap-1234567890", true},
		{"test-vm.deer-tmp-snap", true},
		{"regular-disk.qcow2", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isDeerOverlay(tt.path)
		if got != tt.expected {
			t.Errorf("isDeerOverlay(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestHasBackingFile_OutputParsing(t *testing.T) {
	// Test the string parsing logic that hasBackingFile uses.
	// We can't call the real method (needs virsh), but we verify the
	// parsing logic matches what virsh vol-dumpxml outputs.
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			"with backing file",
			`<volume type='file'>
  <name>test-vm.deer-tmp-snap-123</name>
  <backingStore>
    <path>/var/lib/libvirt/images/test-vm.qcow2</path>
    <format type='qcow2'/>
  </backingStore>
</volume>`,
			true,
		},
		{
			"no backing file (self-closing)",
			`<volume type='file'>
  <name>test-vm.qcow2</name>
  <backingStore/>
</volume>`,
			false,
		},
		{
			"empty output",
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the parsing logic from hasBackingFile
			got := strings.Contains(tt.output, "</backingStore>")
			if got != tt.expected {
				t.Errorf("backing file detection for %q = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestSnapshotAndPull_UnreachableHost(t *testing.T) {
	// With an unreachable host, SnapshotAndPull should fail at findCleanDiskPath
	// (the first virsh call), not panic or produce confusing errors.
	b := NewLibvirtBackend("host1", 22, "root", "", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := b.SnapshotAndPull(ctx, "test-vm", "/tmp/test-dest.qcow2")
	if err == nil {
		t.Fatal("expected error with unreachable host")
	}
}

func TestFluidSnapPrefix(t *testing.T) {
	// Verify the prefix constant matches what we expect
	if deerSnapPrefix != "deer-tmp-snap" {
		t.Errorf("expected deerSnapPrefix to be %q, got %q", "deer-tmp-snap", deerSnapPrefix)
	}

	// Verify a generated snap name starts with the prefix
	snapName := fmt.Sprintf("%s-%d", deerSnapPrefix, 1709000000)
	if !strings.HasPrefix(snapName, deerSnapPrefix) {
		t.Errorf("generated snap name %q does not start with prefix %q", snapName, deerSnapPrefix)
	}
}
