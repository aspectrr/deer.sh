package sourcekeys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureKeyPair(t *testing.T) {
	dir := t.TempDir()
	keyDir := filepath.Join(dir, "keys")

	privPath, pubContents, err := EnsureKeyPair(keyDir)
	if err != nil {
		t.Fatalf("EnsureKeyPair: %v", err)
	}

	if privPath != filepath.Join(keyDir, privateKeyName) {
		t.Errorf("unexpected private key path: %s", privPath)
	}

	if !strings.HasPrefix(pubContents, "ssh-ed25519 ") {
		t.Errorf("public key should start with ssh-ed25519, got: %s", pubContents[:30])
	}

	// Check private key permissions
	info, err := os.Stat(privPath)
	if err != nil {
		t.Fatalf("stat private key: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("private key should be 0600, got %o", info.Mode().Perm())
	}

	// Check public key permissions
	pubPath := filepath.Join(keyDir, publicKeyName)
	info, err = os.Stat(pubPath)
	if err != nil {
		t.Fatalf("stat public key: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("public key should be 0644, got %o", info.Mode().Perm())
	}
}

func TestEnsureKeyPairIdempotent(t *testing.T) {
	dir := t.TempDir()
	keyDir := filepath.Join(dir, "keys")

	_, pub1, err := EnsureKeyPair(keyDir)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	_, pub2, err := EnsureKeyPair(keyDir)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if pub1 != pub2 {
		t.Error("second call should return same public key")
	}
}

func TestGetPublicKey(t *testing.T) {
	dir := t.TempDir()
	keyDir := filepath.Join(dir, "keys")

	_, expected, err := EnsureKeyPair(keyDir)
	if err != nil {
		t.Fatalf("EnsureKeyPair: %v", err)
	}

	got, err := GetPublicKey(keyDir)
	if err != nil {
		t.Fatalf("GetPublicKey: %v", err)
	}

	if got != expected {
		t.Error("GetPublicKey returned different content than EnsureKeyPair")
	}
}

func TestGetPublicKeyMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := GetPublicKey(dir)
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestGetPrivateKeyPath(t *testing.T) {
	path := GetPrivateKeyPath("/some/dir")
	if path != "/some/dir/source_ed25519" {
		t.Errorf("unexpected path: %s", path)
	}
}
