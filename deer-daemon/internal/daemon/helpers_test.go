package daemon

import (
	"testing"

	"github.com/aspectrr/deer.sh/deer-daemon/internal/config"
	deerv1 "github.com/aspectrr/deer.sh/proto/gen/go/deer/v1"
)

func TestSourceHostConns_Defaults(t *testing.T) {
	s := &Server{
		cfg: &config.Config{
			SourceHosts: []config.SourceHostConfig{
				{Address: "10.0.0.1"},
			},
		},
	}

	conns := s.sourceHostConns()
	if len(conns) != 1 {
		t.Fatalf("got %d conns, want 1", len(conns))
	}
	c := conns[0]
	if c.SshHost != "10.0.0.1" {
		t.Errorf("host: got %q, want 10.0.0.1", c.SshHost)
	}
	if c.SshUser != "deer-daemon" {
		t.Errorf("user: got %q, want deer-daemon", c.SshUser)
	}
	if c.SshPort != 22 {
		t.Errorf("port: got %d, want 22", c.SshPort)
	}
	if c.Type != "libvirt" {
		t.Errorf("type: got %q, want libvirt", c.Type)
	}
}

func TestSourceHostConns_CustomValues(t *testing.T) {
	s := &Server{
		cfg: &config.Config{
			SourceHosts: []config.SourceHostConfig{
				{Address: "10.0.0.1", SSHUser: "admin", SSHPort: 2222, Type: "proxmox"},
				{Address: "10.0.0.2"},
			},
		},
	}

	conns := s.sourceHostConns()
	if len(conns) != 2 {
		t.Fatalf("got %d conns, want 2", len(conns))
	}

	// First host: custom values
	if conns[0].SshUser != "admin" {
		t.Errorf("host0 user: got %q, want admin", conns[0].SshUser)
	}
	if conns[0].SshPort != 2222 {
		t.Errorf("host0 port: got %d, want 2222", conns[0].SshPort)
	}
	if conns[0].Type != "proxmox" {
		t.Errorf("host0 type: got %q, want proxmox", conns[0].Type)
	}

	// Second host: defaults
	if conns[1].SshUser != "deer-daemon" {
		t.Errorf("host1 user: got %q, want deer-daemon", conns[1].SshUser)
	}
	if conns[1].SshPort != 22 {
		t.Errorf("host1 port: got %d, want 22", conns[1].SshPort)
	}
}

func TestSourceHostConns_Empty(t *testing.T) {
	s := &Server{cfg: &config.Config{}}
	conns := s.sourceHostConns()
	if len(conns) != 0 {
		t.Errorf("got %d conns, want 0", len(conns))
	}
}

func TestResolveSourceHost_CacheHit(t *testing.T) {
	expected := &deerv1.SourceHostConnection{
		Type: "libvirt", SshHost: "10.0.0.1", SshPort: 22, SshUser: "deer-daemon",
	}
	s := &Server{
		cfg:         &config.Config{},
		vmHostCache: map[string]*deerv1.SourceHostConnection{"my-vm": expected},
	}

	conn, err := s.resolveSourceHost(t.Context(), "my-vm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.SshHost != "10.0.0.1" {
		t.Errorf("got host %q, want 10.0.0.1", conn.SshHost)
	}
}

func TestResolveSourceHost_NotFound_NoHosts(t *testing.T) {
	s := &Server{
		cfg:         &config.Config{},
		vmHostCache: make(map[string]*deerv1.SourceHostConnection),
	}

	_, err := s.resolveSourceHost(t.Context(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for VM not found")
	}
}
