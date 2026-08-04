package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/lib/pq"

	coredomain "solv-backend/internal/core/domain"
	"solv-backend/internal/domain"
)

func (db *Database) CreateUser(ctx context.Context, dto domain.CreateUserDTO) (string, error) {
	tenantID := coredomain.GetTenantID(ctx)
	query := `
		INSERT INTO users (first_name, last_name, email, tenant_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id string
	err := db.db.QueryRowContext(ctx, query, dto.FirstName, dto.LastName, dto.Email, tenantID).Scan(&id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return "", errors.New("email already exists")
			}
		}
		return "", fmt.Errorf("failed to insert user: %w", err)
	}

	return id, nil
}

// GetUserByID retrieves a user by UUID.
func (db *Database) GetUserByID(ctx context.Context, id string) (domain.UserResponseDTO, error) {
	tenantID := coredomain.GetTenantID(ctx)
	query := `SELECT id, first_name, last_name, email, role, tenant_id FROM users WHERE id = $1 AND tenant_id = $2`
	var u domain.UserResponseDTO
	err := db.db.QueryRowContext(ctx, query, id, tenantID).Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Role, &u.TenantID)
	if err != nil {
		return domain.UserResponseDTO{}, fmt.Errorf("failed to get user by id: %w", err)
	}
	return u, nil
}

func (db *Database) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Buscamos globalmente para SSO, pero seleccionamos el tenant_id
	query := `SELECT id, first_name, last_name, email, role, tenant_id FROM users WHERE email = $1`
	var u domain.User
	err := db.db.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Role, &u.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

func (db *Database) CreateUserFromSSO(ctx context.Context, user *domain.User) (string, error) {
	// Si el struct no tiene tenant_id, lo intentamos obtener del context
	tenantID := user.TenantID
	if tenantID == "" {
		tenantID = coredomain.GetTenantID(ctx)
	}
	query := `
		INSERT INTO users (first_name, last_name, email, role, tenant_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id string
	err := db.db.QueryRowContext(ctx, query, user.FirstName, user.LastName, user.Email, user.Role, tenantID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to insert user from SSO: %w", err)
	}
	return id, nil
}
