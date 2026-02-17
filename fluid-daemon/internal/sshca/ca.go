// Package sshca provides SSH Certificate Authority management for ephemeral sandbox access.
//
// This package handles:
// - SSH CA key generation and storage
// - Short-lived SSH certificate issuance
// - Certificate validation and metadata
//
// Certificates are designed to be ephemeral (1-10 minutes TTL) and are
// used to provide secure, auditable access to sandbox VMs without requiring
// any persistent credentials on the VM side.
package sshca

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Common errors
var (
	ErrCANotInitialized   = errors.New("sshca: CA not initialized")
	ErrInvalidPublicKey   = errors.New("sshca: invalid public key")
	ErrInvalidTTL         = errors.New("sshca: TTL must be between 1 and 60 minutes")
	ErrCertGenFailed      = errors.New("sshca: certificate generation failed")
	ErrCAKeyNotFound      = errors.New("sshca: CA private key not found")
	ErrCAKeyPermissions   = errors.New("sshca: CA private key has insecure permissions")
	ErrSSHKeygenNotFound  = errors.New("sshca: ssh-keygen binary not found")
	ErrInvalidPrincipal   = errors.New("sshca: invalid principal")
	ErrInvalidCertOptions = errors.New("sshca: invalid certificate options")
)

// Config holds configuration for the SSH CA.
type Config struct {
	// CAKeyPath is the path to the CA private key file.
	// This key is used to sign all user certificates.
	CAKeyPath string

	// CAPubKeyPath is the path to the CA public key file.
	// This is baked into VM images for certificate verification.
	CAPubKeyPath string

	// WorkDir is the directory for temporary certificate operations.
	// Certificates are generated here before being returned to callers.
	WorkDir string

	// DefaultTTL is the default certificate lifetime if not specified.
	// Must be between 1 and 10 minutes.
	DefaultTTL time.Duration

	// MaxTTL is the maximum allowed certificate lifetime.
	// Requests for longer TTLs will be capped to this value.
	MaxTTL time.Duration

	// DefaultPrincipals are the default principals added to certificates
	// if none are specified. Usually ["sandbox"].
	DefaultPrincipals []string

	// SSHKeygenPath is the optional path to ssh-keygen binary.
	// If empty, it will be looked up in PATH.
	SSHKeygenPath string

	// EnforceKeyPermissions when true, validates CA key file permissions.
	EnforceKeyPermissions bool
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		CAKeyPath:             "/etc/virsh-sandbox/ssh_ca",
		CAPubKeyPath:          "/etc/virsh-sandbox/ssh_ca.pub",
		WorkDir:               "/tmp/sshca",
		DefaultTTL:            30 * time.Minute,
		MaxTTL:                60 * time.Minute,
		DefaultPrincipals:     []string{"sandbox"},
		SSHKeygenPath:         "",
		EnforceKeyPermissions: true,
	}
}

// CertificateRequest contains all parameters needed to issue a certificate.
type CertificateRequest struct {
	// UserID identifies the user requesting access.
	// This is embedded in the certificate identity for audit purposes.
	UserID string

	// VMID identifies the target VM/sandbox.
	// This is embedded in the certificate identity for audit purposes.
	VMID string

	// SandboxID is the internal sandbox identifier.
	SandboxID string

	// PublicKey is the user's SSH public key to be certified.
	// Must be in OpenSSH format (e.g., "ssh-ed25519 AAAA... comment").
	PublicKey string

	// TTL is the requested certificate lifetime.
	// If zero, DefaultTTL is used. If greater than MaxTTL, it's capped.
	TTL time.Duration

	// Principals are the allowed usernames for this certificate.
	// If empty, DefaultPrincipals are used.
	Principals []string

	// SourceIP is the IP address of the requester (for audit).
	SourceIP string

	// RequestTime is when the request was made.
	RequestTime time.Time
}

// Certificate represents an issued SSH certificate.
type Certificate struct {
	// ID is a unique identifier for this certificate.
	ID string

	// Identity is the certificate identity string embedded in the cert.
	// Format: "user:{UserID}-vm:{VMID}-sbx:{SandboxID}"
	Identity string

	// Certificate is the OpenSSH certificate content.
	// This is the content of the -cert.pub file.
	Certificate string

	// SerialNumber is the certificate serial number.
	SerialNumber uint64

	// ValidAfter is when the certificate becomes valid.
	ValidAfter time.Time

	// ValidBefore is when the certificate expires.
	ValidBefore time.Time

	// Principals are the usernames allowed by this certificate.
	Principals []string

	// CriticalOptions lists certificate critical options.
	CriticalOptions map[string]string

	// Extensions lists certificate extensions.
	Extensions []string

	// Request is the original request that created this certificate.
	Request *CertificateRequest

	// IssuedAt is when the certificate was issued.
	IssuedAt time.Time
}

// CA manages SSH certificate authority operations.
type CA struct {
	cfg         Config
	mu          sync.RWMutex
	serialNum   uint64
	sshKeygen   string
	caPubKey    string
	timeNowFn   func() time.Time
	initialized bool
}

// Option configures the CA during construction.
type Option func(*CA)

// WithTimeNow overrides the clock (useful for tests).
func WithTimeNow(fn func() time.Time) Option {
	return func(ca *CA) { ca.timeNowFn = fn }
}

// NewCA creates a new SSH Certificate Authority manager.
func NewCA(cfg Config, opts ...Option) (*CA, error) {
	ca := &CA{
		cfg:       cfg,
		serialNum: 0,
		timeNowFn: time.Now,
	}

	for _, opt := range opts {
		opt(ca)
	}

	// Locate ssh-keygen
	sshKeygen := cfg.SSHKeygenPath
	if sshKeygen == "" {
		path, err := exec.LookPath("ssh-keygen")
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSSHKeygenNotFound, err)
		}
		sshKeygen = path
	}
	ca.sshKeygen = sshKeygen

	// Create work directory
	if err := os.MkdirAll(cfg.WorkDir, 0o700); err != nil {
		return nil, fmt.Errorf("create work directory: %w", err)
	}

	return ca, nil
}

// Initialize validates the CA configuration and loads the CA public key.
// This must be called before issuing certificates.
func (ca *CA) Initialize(ctx context.Context) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Check CA private key exists
	if _, err := os.Stat(ca.cfg.CAKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrCAKeyNotFound, ca.cfg.CAKeyPath)
	}

	// Validate CA key permissions
	if ca.cfg.EnforceKeyPermissions {
		info, err := os.Stat(ca.cfg.CAKeyPath)
		if err != nil {
			return fmt.Errorf("stat CA key: %w", err)
		}
		mode := info.Mode().Perm()
		// Key should be readable only by owner (0600 or 0400)
		if mode&0o077 != 0 {
			return fmt.Errorf("%w: %s has mode %o, expected 0600 or 0400",
				ErrCAKeyPermissions, ca.cfg.CAKeyPath, mode)
		}
	}

	// Load CA public key
	pubKeyBytes, err := os.ReadFile(ca.cfg.CAPubKeyPath)
	if err != nil {
		return fmt.Errorf("read CA public key: %w", err)
	}
	ca.caPubKey = strings.TrimSpace(string(pubKeyBytes))

	// Initialize serial number with random value
	var serialBytes [8]byte
	if _, err := rand.Read(serialBytes[:]); err != nil {
		return fmt.Errorf("initialize serial: %w", err)
	}
	ca.serialNum = uint64(serialBytes[0])<<56 |
		uint64(serialBytes[1])<<48 |
		uint64(serialBytes[2])<<40 |
		uint64(serialBytes[3])<<32 |
		uint64(serialBytes[4])<<24 |
		uint64(serialBytes[5])<<16 |
		uint64(serialBytes[6])<<8 |
		uint64(serialBytes[7])

	ca.initialized = true
	return nil
}

// IssueCertificate generates a short-lived SSH certificate for the given request.
func (ca *CA) IssueCertificate(ctx context.Context, req *CertificateRequest) (*Certificate, error) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if !ca.initialized {
		return nil, ErrCANotInitialized
	}

	// Validate request
	if err := ca.validateRequest(req); err != nil {
		return nil, err
	}

	// Determine TTL
	ttl := req.TTL
	if ttl == 0 {
		ttl = ca.cfg.DefaultTTL
	}
	if ttl < time.Minute {
		return nil, fmt.Errorf("%w: minimum TTL is 1 minute", ErrInvalidTTL)
	}
	if ttl > ca.cfg.MaxTTL {
		ttl = ca.cfg.MaxTTL
	}

	// Determine principals
	principals := req.Principals
	if len(principals) == 0 {
		principals = ca.cfg.DefaultPrincipals
	}

	// Validate principals
	for _, p := range principals {
		if p == "" || strings.ContainsAny(p, " \t\n\r") {
			return nil, fmt.Errorf("%w: %q", ErrInvalidPrincipal, p)
		}
	}

	// Generate unique certificate ID
	certID := ca.generateCertID()

	// Build certificate identity
	identity := fmt.Sprintf("user:%s-vm:%s-sbx:%s-cert:%s",
		req.UserID, req.VMID, req.SandboxID, certID)

	// Increment serial number
	ca.serialNum++
	serial := ca.serialNum

	// Calculate validity window
	now := ca.timeNowFn()
	validAfter := now.Add(-time.Minute) // Allow 1 minute clock skew
	validBefore := now.Add(ttl)

	// Format validity for ssh-keygen
	validityStr := fmt.Sprintf("+%dm", int(ttl.Minutes()))

	// Create temporary directory for this certificate
	tempDir, err := os.MkdirTemp(ca.cfg.WorkDir, "cert-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Write public key to temp file
	pubKeyPath := filepath.Join(tempDir, "user_key.pub")
	if err := os.WriteFile(pubKeyPath, []byte(req.PublicKey), 0o600); err != nil {
		return nil, fmt.Errorf("write public key: %w", err)
	}

	// Build ssh-keygen command
	// ssh-keygen -s CA_KEY -I IDENTITY -n PRINCIPALS -V VALIDITY -z SERIAL -O OPTIONS KEY.pub
	args := []string{
		"-s", ca.cfg.CAKeyPath,
		"-I", identity,
		"-n", strings.Join(principals, ","),
		"-V", validityStr,
		"-z", fmt.Sprintf("%d", serial),
		// Security options - disable forwarding but allow PTY for tmux
		"-O", "no-port-forwarding",
		"-O", "no-agent-forwarding",
		"-O", "no-X11-forwarding",
		// Note: permit-pty is enabled by default, so we don't need to specify it
		pubKeyPath,
	}

	cmd := exec.CommandContext(ctx, ca.sshKeygen, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %v: %s", ErrCertGenFailed, err, stderr.String())
	}

	// Read generated certificate
	certPath := filepath.Join(tempDir, "user_key-cert.pub")
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read certificate: %w", err)
	}

	cert := &Certificate{
		ID:              certID,
		Identity:        identity,
		Certificate:     strings.TrimSpace(string(certBytes)),
		SerialNumber:    serial,
		ValidAfter:      validAfter,
		ValidBefore:     validBefore,
		Principals:      principals,
		CriticalOptions: map[string]string{},
		Extensions: []string{
			"permit-pty",
		},
		Request:  req,
		IssuedAt: now,
	}

	return cert, nil
}

// GetPublicKey returns the CA public key content.
// This is the key that should be baked into VM images.
func (ca *CA) GetPublicKey() (string, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	if !ca.initialized {
		return "", ErrCANotInitialized
	}
	return ca.caPubKey, nil
}

// EnsureSSHCA checks if the SSH CA key pair exists at the given paths.
// If not, it generates a new CA key pair.
// Returns (created bool, error) where created is true if new keys were generated.
func EnsureSSHCA(keyPath, pubKeyPath, comment string) (bool, error) {
	// Check if both files already exist
	keyExists := fileExists(keyPath)
	pubExists := fileExists(pubKeyPath)

	if keyExists && pubExists {
		// Both files exist, nothing to do
		return false, nil
	}

	// If only one exists, that's an inconsistent state
	if keyExists != pubExists {
		return false, fmt.Errorf("inconsistent SSH CA state: key exists=%v, pub exists=%v", keyExists, pubExists)
	}

	// Neither exists, generate new CA
	if err := GenerateCA(keyPath, comment); err != nil {
		return false, err
	}

	return true, nil
}

// fileExists returns true if the file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GenerateCA creates a new SSH CA key pair.
// This should typically only be called once during initial setup.
func GenerateCA(keyPath, comment string) error {
	// Ensure directory exists
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create CA directory: %w", err)
	}

	// Find ssh-keygen
	sshKeygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSSHKeygenNotFound, err)
	}

	// Generate Ed25519 key pair
	args := []string{
		"-t", "ed25519",
		"-f", keyPath,
		"-N", "", // No passphrase
		"-C", comment,
	}

	cmd := exec.Command(sshKeygen, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("generate CA key: %v: %s", err, stderr.String())
	}

	// Set secure permissions on private key
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return fmt.Errorf("set CA key permissions: %w", err)
	}

	return nil
}

// GenerateUserKeyPair generates a new SSH key pair for a user.
// Returns the private key, public key, and any error.
func GenerateUserKeyPair(comment string) (privateKey, publicKey string, err error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "sshkey-")
	if err != nil {
		return "", "", fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	keyPath := filepath.Join(tempDir, "user_key")

	// Find ssh-keygen
	sshKeygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrSSHKeygenNotFound, err)
	}

	// Generate Ed25519 key pair
	args := []string{
		"-t", "ed25519",
		"-f", keyPath,
		"-N", "", // No passphrase
		"-C", comment,
	}

	cmd := exec.Command(sshKeygen, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("generate user key: %v: %s", err, stderr.String())
	}

	// Read keys
	privKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("read private key: %w", err)
	}

	pubKeyBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", "", fmt.Errorf("read public key: %w", err)
	}

	return string(privKeyBytes), strings.TrimSpace(string(pubKeyBytes)), nil
}

// validateRequest validates a certificate request.
func (ca *CA) validateRequest(req *CertificateRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("%w: UserID is required", ErrInvalidCertOptions)
	}
	if req.VMID == "" {
		return fmt.Errorf("%w: VMID is required", ErrInvalidCertOptions)
	}
	if req.PublicKey == "" {
		return fmt.Errorf("%w: PublicKey is required", ErrInvalidCertOptions)
	}

	// Basic validation of public key format
	parts := strings.SplitN(req.PublicKey, " ", 3)
	if len(parts) < 2 {
		return fmt.Errorf("%w: must be in OpenSSH format", ErrInvalidPublicKey)
	}

	keyType := parts[0]
	validTypes := []string{
		"ssh-rsa", "ssh-ed25519", "ecdsa-sha2-nistp256",
		"ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521",
	}
	found := false
	for _, t := range validTypes {
		if keyType == t {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: unsupported key type %q", ErrInvalidPublicKey, keyType)
	}

	// Validate base64 encoding of key
	if _, err := base64.StdEncoding.DecodeString(parts[1]); err != nil {
		return fmt.Errorf("%w: invalid base64 encoding", ErrInvalidPublicKey)
	}

	return nil
}

// generateCertID generates a unique certificate identifier.
func (ca *CA) generateCertID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to time-based ID if random fails (extremely unlikely)
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b[:8])
}

// CertInfo extracts information from a certificate for display/audit.
type CertInfo struct {
	ID           string
	Identity     string
	Serial       uint64
	ValidAfter   time.Time
	ValidBefore  time.Time
	Principals   []string
	Extensions   []string
	IsExpired    bool
	TimeToExpiry time.Duration
}

// GetCertInfo parses certificate info for display purposes.
func (c *Certificate) GetCertInfo() *CertInfo {
	now := time.Now()
	return &CertInfo{
		ID:           c.ID,
		Identity:     c.Identity,
		Serial:       c.SerialNumber,
		ValidAfter:   c.ValidAfter,
		ValidBefore:  c.ValidBefore,
		Principals:   c.Principals,
		Extensions:   c.Extensions,
		IsExpired:    now.After(c.ValidBefore),
		TimeToExpiry: c.ValidBefore.Sub(now),
	}
}

// SSHConnectCommand returns the SSH command string for connecting with this certificate.
func (c *Certificate) SSHConnectCommand(privateKeyPath, certPath, vmIP string, port int) string {
	if port == 0 {
		port = 22
	}
	principal := "sandbox"
	if len(c.Principals) > 0 {
		principal = c.Principals[0]
	}
	return fmt.Sprintf("ssh -i %s -o CertificateFile=%s -o StrictHostKeyChecking=no -p %d %s@%s",
		privateKeyPath, certPath, port, principal, vmIP)
}
