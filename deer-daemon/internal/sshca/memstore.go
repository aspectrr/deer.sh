// Package sshca provides SSH Certificate Authority management for ephemeral sandbox access.
package sshca

import (
	"context"
	"sync"
	"time"
)

// MemoryStore provides an in-memory implementation of CertificateStore.
// This is primarily useful for development and testing.
// For production use, implement a database-backed store.
type MemoryStore struct {
	mu           sync.RWMutex
	certificates map[string]*CertificateRecord
	sessions     map[string]*AccessSession
}

// NewMemoryStore creates a new in-memory certificate store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		certificates: make(map[string]*CertificateRecord),
		sessions:     make(map[string]*AccessSession),
	}
}

// CreateCertificate persists a new certificate record.
func (s *MemoryStore) CreateCertificate(ctx context.Context, cert *CertificateRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.certificates[cert.ID] = cert
	return nil
}

// GetCertificate retrieves a certificate by ID.
func (s *MemoryStore) GetCertificate(ctx context.Context, id string) (*CertificateRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cert, ok := s.certificates[id]
	if !ok {
		return nil, ErrCertNotFound
	}
	return cert, nil
}

// GetCertificateBySerial retrieves a certificate by serial number.
func (s *MemoryStore) GetCertificateBySerial(ctx context.Context, serial uint64) (*CertificateRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, cert := range s.certificates {
		if cert.SerialNumber == serial {
			return cert, nil
		}
	}
	return nil, ErrCertNotFound
}

// ListCertificates retrieves certificates matching the filter.
func (s *MemoryStore) ListCertificates(ctx context.Context, filter CertificateFilter, opts *ListOptions) ([]*CertificateRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*CertificateRecord
	now := time.Now()

	for _, cert := range s.certificates {
		// Apply filters
		if filter.SandboxID != nil && cert.SandboxID != *filter.SandboxID {
			continue
		}
		if filter.UserID != nil && cert.UserID != *filter.UserID {
			continue
		}
		if filter.VMID != nil && cert.VMID != *filter.VMID {
			continue
		}
		if filter.Status != nil && cert.Status != *filter.Status {
			continue
		}
		if filter.ActiveOnly {
			if cert.Status != CertStatusActive || now.After(cert.ValidBefore) {
				continue
			}
		}
		if filter.IssuedAfter != nil && cert.IssuedAt.Before(*filter.IssuedAfter) {
			continue
		}
		if filter.IssuedBefore != nil && cert.IssuedAt.After(*filter.IssuedBefore) {
			continue
		}

		results = append(results, cert)
	}

	// Apply pagination
	if opts != nil {
		if opts.Offset > 0 && opts.Offset < len(results) {
			results = results[opts.Offset:]
		} else if opts.Offset >= len(results) {
			results = nil
		}
		if opts.Limit > 0 && opts.Limit < len(results) {
			results = results[:opts.Limit]
		}
	}

	return results, nil
}

// UpdateCertificateStatus updates the status of a certificate.
func (s *MemoryStore) UpdateCertificateStatus(ctx context.Context, id string, status CertStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cert, ok := s.certificates[id]
	if !ok {
		return ErrCertNotFound
	}

	cert.Status = status
	return nil
}

// RevokeCertificate marks a certificate as revoked.
func (s *MemoryStore) RevokeCertificate(ctx context.Context, id string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cert, ok := s.certificates[id]
	if !ok {
		return ErrCertNotFound
	}

	if cert.Status == CertStatusRevoked {
		return ErrCertAlreadyRevoked
	}

	now := time.Now()
	cert.Status = CertStatusRevoked
	cert.RevokedAt = &now
	cert.RevokeReason = reason
	return nil
}

// UpdateCertificateLastUsed updates the last used timestamp.
func (s *MemoryStore) UpdateCertificateLastUsed(ctx context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cert, ok := s.certificates[id]
	if !ok {
		return ErrCertNotFound
	}

	cert.LastUsedAt = &at
	return nil
}

// ExpireCertificates marks all expired certificates as EXPIRED.
func (s *MemoryStore) ExpireCertificates(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0

	for _, cert := range s.certificates {
		if cert.Status == CertStatusActive && now.After(cert.ValidBefore) {
			cert.Status = CertStatusExpired
			count++
		}
	}

	return count, nil
}

// DeleteCertificate removes a certificate record.
func (s *MemoryStore) DeleteCertificate(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.certificates, id)
	return nil
}

// CreateSession persists a new access session record.
func (s *MemoryStore) CreateSession(ctx context.Context, session *AccessSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ID] = session
	return nil
}

// GetSession retrieves a session by ID.
func (s *MemoryStore) GetSession(ctx context.Context, id string) (*AccessSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// ListSessions retrieves sessions matching the filter.
func (s *MemoryStore) ListSessions(ctx context.Context, filter SessionFilter, opts *ListOptions) ([]*AccessSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*AccessSession

	for _, session := range s.sessions {
		// Apply filters
		if filter.CertificateID != nil && session.CertificateID != *filter.CertificateID {
			continue
		}
		if filter.SandboxID != nil && session.SandboxID != *filter.SandboxID {
			continue
		}
		if filter.UserID != nil && session.UserID != *filter.UserID {
			continue
		}
		if filter.Status != nil && session.Status != *filter.Status {
			continue
		}
		if filter.ActiveOnly && session.Status != SessionStatusActive && session.Status != SessionStatusPending {
			continue
		}
		if filter.StartedAfter != nil && session.StartedAt.Before(*filter.StartedAfter) {
			continue
		}

		results = append(results, session)
	}

	// Apply pagination
	if opts != nil {
		if opts.Offset > 0 && opts.Offset < len(results) {
			results = results[opts.Offset:]
		} else if opts.Offset >= len(results) {
			results = nil
		}
		if opts.Limit > 0 && opts.Limit < len(results) {
			results = results[:opts.Limit]
		}
	}

	return results, nil
}

// UpdateSessionStatus updates the status of a session.
func (s *MemoryStore) UpdateSessionStatus(ctx context.Context, id string, status SessionStatus, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	session.Status = status
	session.DisconnectReason = reason
	return nil
}

// EndSession marks a session as ended.
func (s *MemoryStore) EndSession(ctx context.Context, id string, endedAt time.Time, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	session.Status = SessionStatusEnded
	session.EndedAt = &endedAt
	session.DisconnectReason = reason

	duration := int(endedAt.Sub(session.StartedAt).Seconds())
	session.DurationSeconds = &duration

	return nil
}

// GetActiveSessions returns all currently active sessions.
func (s *MemoryStore) GetActiveSessions(ctx context.Context) ([]*AccessSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*AccessSession
	for _, session := range s.sessions {
		if session.Status == SessionStatusActive || session.Status == SessionStatusPending {
			results = append(results, session)
		}
	}
	return results, nil
}

// GetSessionsByCertificate returns all sessions for a certificate.
func (s *MemoryStore) GetSessionsByCertificate(ctx context.Context, certID string) ([]*AccessSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*AccessSession
	for _, session := range s.sessions {
		if session.CertificateID == certID {
			results = append(results, session)
		}
	}
	return results, nil
}

// Verify MemoryStore implements CertificateStore at compile time.
var _ CertificateStore = (*MemoryStore)(nil)
