package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

// PostgresSessionRevocationRepository guarda solo el identificador de sesión
// (jti o hash heredado) y su expiración. El JWT completo nunca se persiste.
type PostgresSessionRevocationRepository struct {
	db *sqlx.DB
}

func NewPostgresSessionRevocationRepository(db *sqlx.DB) *PostgresSessionRevocationRepository {
	return &PostgresSessionRevocationRepository{db: db}
}

func (r *PostgresSessionRevocationRepository) RevokeSession(ctx context.Context, sessionID string, expiresAt time.Time) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session_id requerido")
	}
	if expiresAt.IsZero() {
		return errors.New("expires_at requerido")
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO session_revocations (session_id, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (session_id) DO UPDATE
		SET expires_at = GREATEST(session_revocations.expires_at, EXCLUDED.expires_at),
		    revoked_at = NOW()
	`, sessionID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("no se pudo persistir la revocación de sesión: %w", err)
	}
	return nil
}

func (r *PostgresSessionRevocationRepository) IsSessionRevoked(ctx context.Context, sessionID string) (bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, nil
	}

	var revoked bool
	err := r.db.GetContext(ctx, &revoked, `
		SELECT EXISTS (
			SELECT 1
			FROM session_revocations
			WHERE session_id = $1
			  AND expires_at > NOW()
		)
	`, sessionID)
	if err != nil {
		return false, fmt.Errorf("no se pudo consultar la revocación de sesión: %w", err)
	}
	return revoked, nil
}

func (r *PostgresSessionRevocationRepository) ListRevokedSessions(ctx context.Context) ([]string, error) {
	var sessionIDs []string
	if err := r.db.SelectContext(ctx, &sessionIDs, `
		SELECT session_id
		FROM session_revocations
		WHERE expires_at > NOW()
	`); err != nil {
		return nil, fmt.Errorf("no se pudieron listar las sesiones revocadas: %w", err)
	}
	return sessionIDs, nil
}

func (r *PostgresSessionRevocationRepository) DeleteExpiredSessions(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `
		DELETE FROM session_revocations
		WHERE expires_at <= NOW()
	`); err != nil {
		return fmt.Errorf("no se pudieron limpiar las revocaciones vencidas: %w", err)
	}
	return nil
}

var _ domain.SessionRevocationRepository = (*PostgresSessionRevocationRepository)(nil)
