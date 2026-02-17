// Package sshkeys provides managed SSH key lifecycle for sandbox command execution.
//
// This package handles ephemeral SSH keypair generation, certificate signing,
// and cleanup for the RunCommand endpoint. Keys are cached per-sandbox and
// automatically regenerated before certificate expiry.
package sshkeys

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-daemon/internal/sshca"
)

// vmNameSanitizer is a compiled regex for sanitizing VM names used in filesystem paths.
// It matches any character that is not alphanumeric, underscore, or hyphen.
var vmNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// KeyProvider provides SSH credentials for sandboxes.
type KeyProvider interface {
	// GetCredentials returns SSH credentials for a sandbox.
	// If valid cached credentials exist, they are returned.
	// Otherwise, new credentials are generated.
	GetCredentials(ctx context.Context, sandboxID, username string) (*Credentials, error)

	// GetSourceVMCredentials returns read-only SSH credentials for a source/golden VM.
	// Uses the "fluid-readonly" principal instead of "sandbox".
	GetSourceVMCredentials(ctx context.Context, sourceVMName string) (*Credentials, error)

	// CleanupSandbox removes all cached credentials for a sandbox.
	// Called when sandbox is destroyed.
	CleanupSandbox(ctx context.Context, sandboxID string) error

	// Close releases all resources.
	Close() error
}

// Credentials holds SSH key material for connecting to a sandbox.
type Credentials struct {
	// PrivateKeyPath is the path to the private key file (0600 permissions).
	PrivateKeyPath string

	// CertificatePath is the path to the certificate file (key-cert.pub).
	CertificatePath string

	// PublicKey is the public key content.
	PublicKey string

	// Username is the SSH username.
	Username string

	// ValidUntil is when the certificate expires.
	ValidUntil time.Time

	// SandboxID is the sandbox these credentials are for.
	SandboxID string
}

// IsExpired returns true if credentials are expired or will expire within margin.
func (c *Credentials) IsExpired(margin time.Duration) bool {
	return time.Now().Add(margin).After(c.ValidUntil)
}

// Config configures the KeyManager.
type Config struct {
	// KeyDir is the base directory for storing keys (default: /tmp/sandbox-keys).
	KeyDir string

	// CertificateTTL is the certificate lifetime (default: 5 minutes).
	CertificateTTL time.Duration

	// RefreshMargin is how early to regenerate before expiry (default: 30 seconds).
	RefreshMargin time.Duration

	// DefaultUsername is the default SSH username (default: "sandbox").
	DefaultUsername string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		KeyDir:          "/tmp/sandbox-keys",
		CertificateTTL:  30 * time.Minute,
		RefreshMargin:   30 * time.Second,
		DefaultUsername: "sandbox",
	}
}

// KeyManager manages ephemeral SSH keys for sandboxes.
type KeyManager struct {
	ca        *sshca.CA
	cfg       Config
	logger    *slog.Logger
	timeNowFn func() time.Time

	// Per-sandbox locks to prevent concurrent key generation.
	mu           sync.RWMutex
	sandboxLocks map[string]*sync.Mutex

	// Cached credentials per sandbox.
	credentials map[string]*Credentials
}

// NewKeyManager creates a new key manager.
func NewKeyManager(ca *sshca.CA, cfg Config, logger *slog.Logger) (*KeyManager, error) {
	if ca == nil {
		return nil, fmt.Errorf("sshca.CA is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	// Apply defaults.
	if cfg.KeyDir == "" {
		cfg.KeyDir = DefaultConfig().KeyDir
	}
	if cfg.CertificateTTL <= 0 {
		cfg.CertificateTTL = DefaultConfig().CertificateTTL
	}
	if cfg.RefreshMargin <= 0 {
		cfg.RefreshMargin = DefaultConfig().RefreshMargin
	}
	if cfg.DefaultUsername == "" {
		cfg.DefaultUsername = DefaultConfig().DefaultUsername
	}

	// Ensure key directory exists.
	if err := os.MkdirAll(cfg.KeyDir, 0o700); err != nil {
		return nil, fmt.Errorf("create key directory %s: %w", cfg.KeyDir, err)
	}

	return &KeyManager{
		ca:           ca,
		cfg:          cfg,
		logger:       logger,
		timeNowFn:    time.Now,
		sandboxLocks: make(map[string]*sync.Mutex),
		credentials:  make(map[string]*Credentials),
	}, nil
}

// GetCredentials implements KeyProvider.
func (m *KeyManager) GetCredentials(ctx context.Context, sandboxID, username string) (*Credentials, error) {
	if sandboxID == "" {
		return nil, fmt.Errorf("sandboxID is required")
	}
	if username == "" {
		username = m.cfg.DefaultUsername
	}

	// Get per-sandbox lock.
	lock := m.getSandboxLock(sandboxID)
	lock.Lock()
	defer lock.Unlock()

	// Check cache for valid credentials.
	cacheKey := m.cacheKey(sandboxID, username)
	m.mu.RLock()
	creds, ok := m.credentials[cacheKey]
	m.mu.RUnlock()

	if ok && !creds.IsExpired(m.cfg.RefreshMargin) {
		m.logger.Debug("using cached credentials",
			"sandbox_id", sandboxID,
			"username", username,
			"valid_until", creds.ValidUntil,
		)
		return creds, nil
	}

	// Generate new credentials.
	m.logger.Info("generating new credentials",
		"sandbox_id", sandboxID,
		"username", username,
		"ttl", m.cfg.CertificateTTL,
	)

	newCreds, err := m.generateCredentials(ctx, sandboxID, username)
	if err != nil {
		return nil, fmt.Errorf("generate credentials: %w", err)
	}

	// Cache the credentials.
	m.mu.Lock()
	m.credentials[cacheKey] = newCreds
	m.mu.Unlock()

	return newCreds, nil
}

// CleanupSandbox implements KeyProvider.
func (m *KeyManager) CleanupSandbox(ctx context.Context, sandboxID string) error {
	if sandboxID == "" {
		return fmt.Errorf("sandboxID is required")
	}

	// Get per-sandbox lock.
	lock := m.getSandboxLock(sandboxID)
	lock.Lock()
	defer lock.Unlock()

	m.logger.Info("cleaning up sandbox credentials", "sandbox_id", sandboxID)

	// Remove from cache (all usernames for this sandbox).
	m.mu.Lock()
	for key := range m.credentials {
		if m.extractSandboxID(key) == sandboxID {
			delete(m.credentials, key)
		}
	}
	m.mu.Unlock()

	// Remove key files.
	keyDir := m.sandboxKeyDir(sandboxID)
	if err := os.RemoveAll(keyDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove key directory %s: %w", keyDir, err)
	}

	// Clean up the sandbox lock.
	m.mu.Lock()
	delete(m.sandboxLocks, sandboxID)
	m.mu.Unlock()

	return nil
}

// Close implements KeyProvider.
func (m *KeyManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("closing key manager", "cached_credentials", len(m.credentials))

	// Clear all cached credentials (files are left for cleanup on sandbox destroy).
	m.credentials = make(map[string]*Credentials)
	m.sandboxLocks = make(map[string]*sync.Mutex)

	return nil
}

// getSandboxLock returns the mutex for a specific sandbox, creating one if needed.
func (m *KeyManager) getSandboxLock(sandboxID string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()

	if lock, ok := m.sandboxLocks[sandboxID]; ok {
		return lock
	}

	lock := &sync.Mutex{}
	m.sandboxLocks[sandboxID] = lock
	return lock
}

// cacheKey generates a cache key for sandbox+username.
func (m *KeyManager) cacheKey(sandboxID, username string) string {
	return sandboxID + ":" + username
}

// extractSandboxID extracts the sandbox ID from a cache key.
func (m *KeyManager) extractSandboxID(cacheKey string) string {
	for i := 0; i < len(cacheKey); i++ {
		if cacheKey[i] == ':' {
			return cacheKey[:i]
		}
	}
	return cacheKey
}

// sandboxKeyDir returns the directory for a sandbox's keys.
func (m *KeyManager) sandboxKeyDir(sandboxID string) string {
	return filepath.Join(m.cfg.KeyDir, sandboxID)
}

// generateCredentials creates new SSH credentials for a sandbox.
func (m *KeyManager) generateCredentials(ctx context.Context, sandboxID, username string) (*Credentials, error) {
	// Create sandbox key directory.
	keyDir := m.sandboxKeyDir(sandboxID)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return nil, fmt.Errorf("create sandbox key directory: %w", err)
	}

	// Generate ephemeral keypair.
	comment := fmt.Sprintf("sandbox-%s-%s", sandboxID, username)
	privateKey, publicKey, err := sshca.GenerateUserKeyPair(comment)
	if err != nil {
		return nil, fmt.Errorf("generate keypair: %w", err)
	}

	// Write key files.
	privateKeyPath := filepath.Join(keyDir, "key")
	certPath := filepath.Join(keyDir, "key-cert.pub")

	if err := os.WriteFile(privateKeyPath, []byte(privateKey), 0o600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}

	// Request certificate from CA.
	certReq := sshca.CertificateRequest{
		UserID:      fmt.Sprintf("sandbox-runner:%s", sandboxID),
		VMID:        sandboxID,
		SandboxID:   sandboxID,
		PublicKey:   publicKey,
		TTL:         m.cfg.CertificateTTL,
		Principals:  []string{username},
		SourceIP:    "internal",
		RequestTime: m.timeNowFn(),
	}

	cert, err := m.ca.IssueCertificate(ctx, &certReq)
	if err != nil {
		// Clean up the private key on failure.
		_ = os.Remove(privateKeyPath)
		return nil, fmt.Errorf("issue certificate: %w", err)
	}

	// Write certificate.
	if err := os.WriteFile(certPath, []byte(cert.Certificate), 0o644); err != nil {
		_ = os.Remove(privateKeyPath)
		return nil, fmt.Errorf("write certificate: %w", err)
	}

	m.logger.Debug("generated credentials",
		"sandbox_id", sandboxID,
		"username", username,
		"private_key_path", privateKeyPath,
		"cert_path", certPath,
		"valid_until", cert.ValidBefore,
	)

	return &Credentials{
		PrivateKeyPath:  privateKeyPath,
		CertificatePath: certPath,
		PublicKey:       publicKey,
		Username:        username,
		ValidUntil:      cert.ValidBefore,
		SandboxID:       sandboxID,
	}, nil
}

// GetSourceVMCredentials implements KeyProvider.
// Issues a cert with principal "fluid-readonly" for read-only access to golden VMs.
func (m *KeyManager) GetSourceVMCredentials(ctx context.Context, sourceVMName string) (*Credentials, error) {
	if sourceVMName == "" {
		return nil, fmt.Errorf("sourceVMName is required")
	}

	username := "fluid-readonly"

	// Use a virtual ID for locking and caching.
	virtualID := "sourcevm:" + sourceVMName
	lock := m.getSandboxLock(virtualID)
	lock.Lock()
	defer lock.Unlock()

	// Check cache.
	cacheKey := virtualID + ":" + username
	m.mu.RLock()
	creds, ok := m.credentials[cacheKey]
	m.mu.RUnlock()

	if ok && !creds.IsExpired(m.cfg.RefreshMargin) {
		m.logger.Debug("using cached source VM credentials",
			"source_vm", sourceVMName,
			"valid_until", creds.ValidUntil,
		)
		return creds, nil
	}

	m.logger.Info("generating source VM read-only credentials",
		"source_vm", sourceVMName,
		"ttl", m.cfg.CertificateTTL,
	)

	newCreds, err := m.generateSourceVMCredentials(ctx, sourceVMName)
	if err != nil {
		return nil, fmt.Errorf("generate source VM credentials: %w", err)
	}

	m.mu.Lock()
	m.credentials[cacheKey] = newCreds
	m.mu.Unlock()

	return newCreds, nil
}

// sanitizeVMName sanitizes a VM name for safe use in filesystem paths.
// It replaces any character that is not alphanumeric, underscore, or hyphen
// with an underscore. This prevents path traversal attacks using ".." or "/" in VM names.
func sanitizeVMName(name string) string {
	// Only allow: A-Z, a-z, 0-9, underscore, hyphen
	return vmNameSanitizer.ReplaceAllString(name, "_")
}

// generateSourceVMCredentials creates read-only SSH credentials for a source VM.
func (m *KeyManager) generateSourceVMCredentials(ctx context.Context, sourceVMName string) (*Credentials, error) {
	// Sanitize the VM name for safe filesystem path usage
	sanitizedName := sanitizeVMName(sourceVMName)
	keyDir := filepath.Join(m.cfg.KeyDir, "sourcevm-"+sanitizedName)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return nil, fmt.Errorf("create source VM key directory: %w", err)
	}

	comment := fmt.Sprintf("source-readonly-%s", sourceVMName)
	privateKey, publicKey, err := sshca.GenerateUserKeyPair(comment)
	if err != nil {
		return nil, fmt.Errorf("generate keypair: %w", err)
	}

	privateKeyPath := filepath.Join(keyDir, "key")
	certPath := filepath.Join(keyDir, "key-cert.pub")

	if err := os.WriteFile(privateKeyPath, []byte(privateKey), 0o600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}

	certReq := sshca.CertificateRequest{
		UserID:      fmt.Sprintf("source-readonly:%s", sourceVMName),
		VMID:        sourceVMName,
		SandboxID:   sourceVMName,
		PublicKey:   publicKey,
		TTL:         m.cfg.CertificateTTL,
		Principals:  []string{"fluid-readonly"},
		SourceIP:    "internal",
		RequestTime: m.timeNowFn(),
	}

	cert, err := m.ca.IssueCertificate(ctx, &certReq)
	if err != nil {
		_ = os.Remove(privateKeyPath)
		return nil, fmt.Errorf("issue certificate: %w", err)
	}

	if err := os.WriteFile(certPath, []byte(cert.Certificate), 0o644); err != nil {
		_ = os.Remove(privateKeyPath)
		return nil, fmt.Errorf("write certificate: %w", err)
	}

	m.logger.Debug("generated source VM credentials",
		"source_vm", sourceVMName,
		"private_key_path", privateKeyPath,
		"cert_path", certPath,
		"valid_until", cert.ValidBefore,
	)

	return &Credentials{
		PrivateKeyPath:  privateKeyPath,
		CertificatePath: certPath,
		PublicKey:       publicKey,
		Username:        "fluid-readonly",
		ValidUntil:      cert.ValidBefore,
		SandboxID:       sourceVMName,
	}, nil
}
