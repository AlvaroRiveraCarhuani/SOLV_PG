package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"strings"
)

// ListAdministrators devuelve los usuarios con rol admin del tenant ordenados por fecha de creación ascendente.
func (r *PostgresAdminGovernanceRepository) ListAdministrators(
	ctx context.Context,
	tenantID string,
) ([]*domain.AdminUserItem, error) {
	query := `
		SELECT id, email, first_name, last_name, role, status, COALESCE(origin, 'manual') AS origin, created_at, last_login_at
		FROM users
		WHERE tenant_id = $1 AND role = 'admin'
		ORDER BY created_at ASC
	`
	var items []*domain.AdminUserItem
	if err := r.db.SelectContext(ctx, &items, query, tenantID); err != nil {
		return nil, fmt.Errorf("error listing administrators: %w", err)
	}
	if items == nil {
		items = []*domain.AdminUserItem{}
	}
	return items, nil
}

// CountActiveAdministrators devuelve la cantidad de administradores en estado activo para el tenant.
func (r *PostgresAdminGovernanceRepository) CountActiveAdministrators(
	ctx context.Context,
	tenantID string,
) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM users
		WHERE tenant_id = $1 AND role = 'admin' AND status = 'active'
	`
	if err := r.db.GetContext(ctx, &count, query, tenantID); err != nil {
		return 0, fmt.Errorf("error counting active administrators: %w", err)
	}
	return count, nil
}

// CreateAdministrator inserta o promueve a un usuario al rol admin en el tenant.
func (r *PostgresAdminGovernanceRepository) CreateAdministrator(
	ctx context.Context,
	tenantID, email, firstName, lastName string,
) (*domain.AdminUserItem, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	trimmedFirst := strings.TrimSpace(firstName)
	trimmedLast := strings.TrimSpace(lastName)

	// Verificar si el usuario ya existe en el tenant
	var existing domain.AdminUserItem
	queryCheck := `
		SELECT id, email, first_name, last_name, role, status, COALESCE(origin, 'manual') AS origin, created_at, last_login_at
		FROM users
		WHERE tenant_id = $1 AND LOWER(email) = $2
	`
	err := r.db.GetContext(ctx, &existing, queryCheck, tenantID, normalizedEmail)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error checking existing user: %w", err)
	}

	if err == nil {
		// Usuario existe: promover a admin y reactivar
		queryUpdate := `
			UPDATE users
			SET role = 'admin',
			    status = 'active',
			    first_name = CASE WHEN $3 <> '' THEN $3 ELSE first_name END,
			    last_name = CASE WHEN $4 <> '' THEN $4 ELSE last_name END,
			    updated_at = NOW()
			WHERE tenant_id = $1 AND id = $2
			RETURNING id, email, first_name, last_name, role, status, COALESCE(origin, 'manual') AS origin, created_at, last_login_at
		`
		var updated domain.AdminUserItem
		if err := r.db.GetContext(ctx, &updated, queryUpdate, tenantID, existing.ID, trimmedFirst, trimmedLast); err != nil {
			return nil, fmt.Errorf("error promoting user to administrator: %w", err)
		}
		return &updated, nil
	}

	// Usuario no existe: insertar nuevo registro con rol admin
	queryInsert := `
		INSERT INTO users (id, tenant_id, email, first_name, last_name, role, status, origin, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, 'admin', 'active', 'manual', NOW(), NOW())
		RETURNING id, email, first_name, last_name, role, status, COALESCE(origin, 'manual') AS origin, created_at, last_login_at
	`
	var created domain.AdminUserItem
	if err := r.db.GetContext(ctx, &created, queryInsert, tenantID, normalizedEmail, trimmedFirst, trimmedLast); err != nil {
		return nil, fmt.Errorf("error creating administrator: %w", err)
	}
	return &created, nil
}

// RevokeAdministrator revoca los privilegios de un administrador pasando su rol a student y su estado a suspendido.
func (r *PostgresAdminGovernanceRepository) RevokeAdministrator(
	ctx context.Context,
	tenantID, targetUserID string,
) error {
	query := `
		UPDATE users
		SET role = 'student',
		    status = 'suspended',
		    suspension_reason = 'Privilegios de administrador revocados',
		    updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2 AND role = 'admin'
	`
	res, err := r.db.ExecContext(ctx, query, tenantID, targetUserID)
	if err != nil {
		return fmt.Errorf("error revoking administrator: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %w", err)
	}
	if rows == 0 {
		return domain.ErrAdminNotFound
	}
	return nil
}
