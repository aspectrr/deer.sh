package snapshotpull

import (
	"testing"
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
}

func TestLibvirtBackend_SSHOpts(t *testing.T) {
	b := NewLibvirtBackend("host1", 2222, "user", "/path/to/key", nil)
	opts := b.sshOpts()

	if opts == "" {
		t.Fatal("expected non-empty ssh opts")
	}

	// Should contain port
	if !contains(opts, "-p 2222") {
		t.Errorf("expected opts to contain port, got: %s", opts)
	}

	// Should contain identity file
	if !contains(opts, "-i /path/to/key") {
		t.Errorf("expected opts to contain identity file, got: %s", opts)
	}
}

func TestLibvirtBackend_SSHOpts_NoIdentity(t *testing.T) {
	b := NewLibvirtBackend("host1", 22, "root", "", nil)
	opts := b.sshOpts()

	if contains(opts, "-i") {
		t.Errorf("expected no identity file flag, got: %s", opts)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
