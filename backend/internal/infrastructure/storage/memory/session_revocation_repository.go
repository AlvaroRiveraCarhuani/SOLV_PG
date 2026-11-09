package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

// SessionRevocationRepository solo sirve para tests unitarios aislados o
// desarrollo efímero. No es persistente ni compartido entre procesos; la
// producción usa PostgresSessionRevocationRepository.
type SessionRevocationRepository struct {
	mu      sync.Mutex
	revoked map[string]time.Time
}

func NewSessionRevocationRepository() *SessionRevocationRepository {
	return &SessionRevocationRepository{revoked: make(map[string]time.Time)}
}

func (r *SessionRevocationRepository) RevokeSession(_ context.Context, sessionID string, expiresAt time.Time) error {
	if sessionID == "" {
		return errors.New("session_id requerido")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revoked[sessionID] = expiresAt
	return nil
}

func (r *SessionRevocationRepository) IsSessionRevoked(_ context.Context, sessionID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	expiresAt, ok := r.revoked[sessionID]
	if !ok {
		return false, nil
	}
	if !expiresAt.After(time.Now()) {
		delete(r.revoked, sessionID)
		return false, nil
	}
	return true, nil
}

func (r *SessionRevocationRepository) ListRevokedSessions(_ context.Context) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	ids := make([]string, 0, len(r.revoked))
	for id, expiresAt := range r.revoked {
		if expiresAt.After(now) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *SessionRevocationRepository) DeleteExpiredSessions(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for id, expiresAt := range r.revoked {
		if !expiresAt.After(now) {
			delete(r.revoked, id)
		}
	}
	return nil
}

var _ domain.SessionRevocationRepository = (*SessionRevocationRepository)(nil)
