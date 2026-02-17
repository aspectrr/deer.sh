package sshca

import (
	"context"
	"errors"
	"time"
)

// Store errors
var (
	ErrCertNotFound       = errors.New("sshca: certificate not found")
	ErrSessionNotFound    = errors.New("sshca: session not found")
	ErrCertAlreadyRevoked = errors.New("sshca: certificate already revoked")
)

// CertificateRecord represents a persisted SSH certificate.
type CertificateRecord struct {
	// ID is the unique certificate identifier.
	ID string `json:"id" db:"id"`

	// SandboxID links this certificate to a sandbox.
	SandboxID string `json:"sandbox_id" db:"sandbox_id"`

	// UserID identifies the user who requested the certificate.
	UserID string `json:"user_id" db:"user_id"`

	// VMID identifies the target VM.
	VMID string `json:"vm_id" db:"vm_id"`

	// Identity is the certificate identity string.
	Identity string `json:"identity" db:"identity"`

	// SerialNumber is the certificate serial number.
	SerialNumber uint64 `json:"serial_number" db:"serial_number"`

	// Principals are the allowed usernames.
	Principals []string `json:"principals" db:"principals"`

	// PublicKeyFingerprint is the SHA256 fingerprint of the user's public key.
	PublicKeyFingerprint string `json:"public_key_fingerprint" db:"public_key_fingerprint"`

	// ValidAfter is when the certificate becomes valid.
	ValidAfter time.Time `json:"valid_after" db:"valid_after"`

	// ValidBefore is when the certificate expires.
	ValidBefore time.Time `json:"valid_before" db:"valid_before"`

	// SourceIP is the IP address of the requester.
	SourceIP string `json:"source_ip,omitempty" db:"source_ip"`

	// Status indicates the certificate state.
	Status CertStatus `json:"status" db:"status"`

	// RevokedAt is when the certificate was revoked (if applicable).
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`

	// RevokeReason explains why the certificate was revoked.
	RevokeReason string `json:"revoke_reason,omitempty" db:"revoke_reason"`

	// IssuedAt is when the certificate was issued.
	IssuedAt time.Time `json:"issued_at" db:"issued_at"`

	// LastUsedAt tracks when the certificate was last used for connection.
	LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

// CertStatus represents the state of a certificate.
type CertStatus string

const (
	// CertStatusActive indicates the certificate is valid and usable.
	CertStatusActive CertStatus = "ACTIVE"

	// CertStatusExpired indicates the certificate has passed its ValidBefore time.
	CertStatusExpired CertStatus = "EXPIRED"

	// CertStatusRevoked indicates the certificate was manually revoked.
	CertStatusRevoked CertStatus = "REVOKED"

	// CertStatusUsed indicates the certificate was used and session ended.
	CertStatusUsed CertStatus = "USED"
)

// AccessSession tracks a user's SSH session using a certificate.
type AccessSession struct {
	// ID is the unique session identifier.
	ID string `json:"id" db:"id"`

	// CertificateID links to the certificate used for this session.
	CertificateID string `json:"certificate_id" db:"certificate_id"`

	// SandboxID links to the accessed sandbox.
	SandboxID string `json:"sandbox_id" db:"sandbox_id"`

	// UserID identifies the user.
	UserID string `json:"user_id" db:"user_id"`

	// VMID identifies the target VM.
	VMID string `json:"vm_id" db:"vm_id"`

	// VMIPAddress is the IP address used to connect.
	VMIPAddress string `json:"vm_ip_address" db:"vm_ip_address"`

	// SourceIP is the IP address the user connected from.
	SourceIP string `json:"source_ip,omitempty" db:"source_ip"`

	// Status indicates the session state.
	Status SessionStatus `json:"status" db:"status"`

	// StartedAt is when the session began.
	StartedAt time.Time `json:"started_at" db:"started_at"`

	// EndedAt is when the session ended (if applicable).
	EndedAt *time.Time `json:"ended_at,omitempty" db:"ended_at"`

	// DurationSeconds is the session duration in seconds.
	DurationSeconds *int `json:"duration_seconds,omitempty" db:"duration_seconds"`

	// DisconnectReason explains why the session ended.
	DisconnectReason string `json:"disconnect_reason,omitempty" db:"disconnect_reason"`
}

// SessionStatus represents the state of an access session.
type SessionStatus string

const (
	// SessionStatusPending indicates the session is waiting for connection.
	SessionStatusPending SessionStatus = "PENDING"

	// SessionStatusActive indicates the session is currently connected.
	SessionStatusActive SessionStatus = "ACTIVE"

	// SessionStatusEnded indicates the session ended normally.
	SessionStatusEnded SessionStatus = "ENDED"

	// SessionStatusExpired indicates the session ended due to certificate expiry.
	SessionStatusExpired SessionStatus = "EXPIRED"

	// SessionStatusRevoked indicates the session was terminated by revocation.
	SessionStatusRevoked SessionStatus = "REVOKED"

	// SessionStatusError indicates the session ended due to an error.
	SessionStatusError SessionStatus = "ERROR"
)

// CertificateFilter provides filtering options for certificate queries.
type CertificateFilter struct {
	SandboxID *string
	UserID    *string
	VMID      *string
	Status    *CertStatus
	// ActiveOnly filters to certificates that are currently valid (not expired/revoked).
	ActiveOnly bool
	// IssuedAfter filters to certificates issued after this time.
	IssuedAfter *time.Time
	// IssuedBefore filters to certificates issued before this time.
	IssuedBefore *time.Time
}

// SessionFilter provides filtering options for session queries.
type SessionFilter struct {
	CertificateID *string
	SandboxID     *string
	UserID        *string
	Status        *SessionStatus
	// ActiveOnly filters to currently active sessions.
	ActiveOnly bool
	// StartedAfter filters to sessions started after this time.
	StartedAfter *time.Time
}

// ListOptions provides pagination and ordering options.
type ListOptions struct {
	Limit   int
	Offset  int
	OrderBy string
	Asc     bool
}

// CertificateStore defines persistence operations for SSH certificates.
type CertificateStore interface {
	// CreateCertificate persists a new certificate record.
	CreateCertificate(ctx context.Context, cert *CertificateRecord) error

	// GetCertificate retrieves a certificate by ID.
	GetCertificate(ctx context.Context, id string) (*CertificateRecord, error)

	// GetCertificateBySerial retrieves a certificate by serial number.
	GetCertificateBySerial(ctx context.Context, serial uint64) (*CertificateRecord, error)

	// ListCertificates retrieves certificates matching the filter.
	ListCertificates(ctx context.Context, filter CertificateFilter, opts *ListOptions) ([]*CertificateRecord, error)

	// UpdateCertificateStatus updates the status of a certificate.
	UpdateCertificateStatus(ctx context.Context, id string, status CertStatus) error

	// RevokeCertificate marks a certificate as revoked.
	RevokeCertificate(ctx context.Context, id string, reason string) error

	// UpdateCertificateLastUsed updates the last used timestamp.
	UpdateCertificateLastUsed(ctx context.Context, id string, at time.Time) error

	// ExpireCertificates marks all expired certificates as EXPIRED.
	// Returns the number of certificates updated.
	ExpireCertificates(ctx context.Context) (int, error)

	// DeleteCertificate removes a certificate record (for cleanup).
	DeleteCertificate(ctx context.Context, id string) error

	// CreateSession persists a new access session record.
	CreateSession(ctx context.Context, session *AccessSession) error

	// GetSession retrieves a session by ID.
	GetSession(ctx context.Context, id string) (*AccessSession, error)

	// ListSessions retrieves sessions matching the filter.
	ListSessions(ctx context.Context, filter SessionFilter, opts *ListOptions) ([]*AccessSession, error)

	// UpdateSessionStatus updates the status of a session.
	UpdateSessionStatus(ctx context.Context, id string, status SessionStatus, reason string) error

	// EndSession marks a session as ended.
	EndSession(ctx context.Context, id string, endedAt time.Time, reason string) error

	// GetActiveSessions returns all currently active sessions.
	GetActiveSessions(ctx context.Context) ([]*AccessSession, error)

	// GetSessionsByCertificate returns all sessions for a certificate.
	GetSessionsByCertificate(ctx context.Context, certID string) ([]*AccessSession, error)
}

// AccessRequest represents a request for sandbox access.
// This is used as input to the access service.
type AccessRequest struct {
	// SandboxID is the target sandbox.
	SandboxID string `json:"sandbox_id"`

	// UserID identifies the requesting user.
	UserID string `json:"user_id"`

	// PublicKey is the user's SSH public key.
	PublicKey string `json:"public_key"`

	// TTLMinutes is the requested access duration in minutes (1-10).
	TTLMinutes int `json:"ttl_minutes,omitempty"`

	// SourceIP is the IP address of the requester (populated by server).
	SourceIP string `json:"-"`

	// RequestTime is when the request was made (populated by server).
	RequestTime time.Time `json:"-"`
}

// AccessResponse contains the issued certificate and connection details.
type AccessResponse struct {
	// CertificateID is the ID of the issued certificate.
	CertificateID string `json:"certificate_id"`

	// Certificate is the SSH certificate content.
	Certificate string `json:"certificate"`

	// VMIPAddress is the IP address of the sandbox VM.
	VMIPAddress string `json:"vm_ip_address"`

	// SSHPort is the SSH port (usually 22).
	SSHPort int `json:"ssh_port"`

	// Username is the SSH username to use (usually "sandbox").
	Username string `json:"username"`

	// ValidUntil is when the certificate expires.
	ValidUntil time.Time `json:"valid_until"`

	// TTLSeconds is the remaining validity in seconds.
	TTLSeconds int `json:"ttl_seconds"`

	// ConnectCommand is an example SSH command for connecting.
	ConnectCommand string `json:"connect_command"`
}

// IsExpired returns true if the certificate has expired.
func (c *CertificateRecord) IsExpired() bool {
	return time.Now().After(c.ValidBefore)
}

// IsActive returns true if the certificate is active and not expired.
func (c *CertificateRecord) IsActive() bool {
	now := time.Now()
	return c.Status == CertStatusActive &&
		now.After(c.ValidAfter) &&
		now.Before(c.ValidBefore)
}

// TimeToExpiry returns the duration until the certificate expires.
// Returns 0 if already expired.
func (c *CertificateRecord) TimeToExpiry() time.Duration {
	remaining := time.Until(c.ValidBefore)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Duration returns the session duration.
// Returns 0 if the session hasn't ended.
func (s *AccessSession) Duration() time.Duration {
	if s.EndedAt == nil {
		return time.Since(s.StartedAt)
	}
	return s.EndedAt.Sub(s.StartedAt)
}

// AuditEntry represents an audit log entry for SSH access events.
type AuditEntry struct {
	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Event is the type of event (e.g., "cert_issued", "session_started", "session_ended").
	Event string `json:"event"`

	// UserID is the user involved.
	UserID string `json:"user_id"`

	// SandboxID is the sandbox involved.
	SandboxID string `json:"sandbox_id"`

	// CertificateID is the certificate involved (if applicable).
	CertificateID string `json:"certificate_id,omitempty"`

	// SessionID is the session involved (if applicable).
	SessionID string `json:"session_id,omitempty"`

	// SourceIP is the source IP address.
	SourceIP string `json:"source_ip,omitempty"`

	// Details contains additional event-specific information.
	Details map[string]any `json:"details,omitempty"`
}
