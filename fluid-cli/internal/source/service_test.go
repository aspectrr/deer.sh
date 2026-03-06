package source

import (
	"context"
	"log/slog"
	"testing"

	"github.com/aspectrr/fluid.sh/fluid-cli/internal/config"
)

func TestListHosts(t *testing.T) {
	cfg := &config.Config{
		Hosts: []config.HostConfig{
			{Name: "web-01", Address: "10.0.0.1", Prepared: true},
			{Name: "db-01", Address: "10.0.0.2", Prepared: false},
		},
	}
	svc := NewService(cfg, "/tmp/key", slog.Default())

	hosts := svc.ListHosts()
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].Name != "web-01" || !hosts[0].Prepared {
		t.Errorf("unexpected host[0]: %+v", hosts[0])
	}
	if hosts[1].Name != "db-01" || hosts[1].Prepared {
		t.Errorf("unexpected host[1]: %+v", hosts[1])
	}
}

func TestFindHostNotFound(t *testing.T) {
	cfg := &config.Config{
		Hosts: []config.HostConfig{
			{Name: "web-01", Address: "10.0.0.1"},
		},
	}
	svc := NewService(cfg, "/tmp/key", slog.Default())

	_, err := svc.RunCommand(context.TODO(), "nonexistent", "ls")
	if err == nil {
		t.Error("expected error for nonexistent host")
	}
}

func TestReadFileRequiresAbsolutePath(t *testing.T) {
	cfg := &config.Config{
		Hosts: []config.HostConfig{
			{Name: "web-01", Address: "10.0.0.1", Prepared: true},
		},
	}
	svc := NewService(cfg, "/tmp/key", slog.Default())

	_, err := svc.ReadFile(context.TODO(), "web-01", "relative/path")
	if err == nil {
		t.Error("expected error for relative path")
	}
}
