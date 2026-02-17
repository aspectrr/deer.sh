package snapshotpull

import (
	"testing"
)

func TestNewProxmoxBackend(t *testing.T) {
	b := NewProxmoxBackend("https://pve.example.com:8006", "user@pam!token", "secret", "node1", false, nil)

	if b.host != "https://pve.example.com:8006" {
		t.Errorf("expected host https://pve.example.com:8006, got %s", b.host)
	}
	if b.tokenID != "user@pam!token" {
		t.Errorf("expected tokenID user@pam!token, got %s", b.tokenID)
	}
	if b.secret != "secret" {
		t.Errorf("expected secret 'secret', got %s", b.secret)
	}
	if b.node != "node1" {
		t.Errorf("expected node node1, got %s", b.node)
	}
}

func TestNewProxmoxBackend_TrimsTrailingSlash(t *testing.T) {
	b := NewProxmoxBackend("https://pve.example.com:8006/", "tok", "sec", "n1", false, nil)

	if b.host != "https://pve.example.com:8006" {
		t.Errorf("expected trailing slash trimmed, got %s", b.host)
	}
}
