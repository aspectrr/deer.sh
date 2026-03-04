package sourcekeys

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

const (
	privateKeyName = "source_ed25519"
	publicKeyName  = "source_ed25519.pub"
)

// EnsureKeyPair generates an ed25519 key pair at keyDir if it does not already exist.
// Returns the private key path and public key contents. Idempotent.
func EnsureKeyPair(keyDir string) (privateKeyPath string, pubKeyContents string, err error) {
	privPath := filepath.Join(keyDir, privateKeyName)
	pubPath := filepath.Join(keyDir, publicKeyName)

	// Check if key pair already exists
	if _, err := os.Stat(privPath); err == nil {
		pub, err := os.ReadFile(pubPath)
		if err != nil {
			return "", "", fmt.Errorf("read existing public key: %w", err)
		}
		return privPath, string(pub), nil
	}

	// Create key directory
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return "", "", fmt.Errorf("create key directory: %w", err)
	}

	// Generate ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	// Marshal private key to OpenSSH format
	privPEM, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}

	privBytes := pem.EncodeToMemory(privPEM)
	if err := os.WriteFile(privPath, privBytes, 0o600); err != nil {
		return "", "", fmt.Errorf("write private key: %w", err)
	}
	// Ensure permissions are correct regardless of umask
	if err := os.Chmod(privPath, 0o600); err != nil {
		return "", "", fmt.Errorf("set private key permissions: %w", err)
	}

	// Marshal public key to authorized_keys format
	sshPub, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", fmt.Errorf("convert public key: %w", err)
	}
	pubContents := string(ssh.MarshalAuthorizedKey(sshPub))

	if err := os.WriteFile(pubPath, []byte(pubContents), 0o644); err != nil {
		return "", "", fmt.Errorf("write public key: %w", err)
	}

	return privPath, pubContents, nil
}

// GetPublicKey reads the public key contents from the key directory.
func GetPublicKey(keyDir string) (string, error) {
	pubPath := filepath.Join(keyDir, publicKeyName)
	data, err := os.ReadFile(pubPath)
	if err != nil {
		return "", fmt.Errorf("read public key: %w", err)
	}
	return string(data), nil
}

// GetPrivateKeyPath returns the path to the private key in the key directory.
// The returned path may not exist on disk if EnsureKeyPair has not been called.
func GetPrivateKeyPath(keyDir string) string {
	return filepath.Join(keyDir, privateKeyName)
}
