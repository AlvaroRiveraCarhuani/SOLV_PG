package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/lib/pq"

	"solv-backend/internal/domain"
)

func (db *Database) CreateUser(ctx context.Context, dto domain.CreateUserDTO) (string, error) {
	query := `
		INSERT INTO users (first_name, last_name, email)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id string
	err := db.db.QueryRowContext(ctx, query, dto.FirstName, dto.LastName, dto.Email).Scan(&id)
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
	query := `SELECT id, first_name, last_name, email, role FROM users WHERE id = $1`
	var u domain.UserResponseDTO
	err := db.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Role)
	if err != nil {
		return domain.UserResponseDTO{}, fmt.Errorf("failed to get user by id: %w", err)
	}
	return u, nil
}

func (db *Database) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, first_name, last_name, email, role FROM users WHERE email = $1`
	var u domain.User
	err := db.db.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

func (db *Database) CreateUserFromSSO(ctx context.Context, user *domain.User) (string, error) {
	query := `
		INSERT INTO users (first_name, last_name, email, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id string
	err := db.db.QueryRowContext(ctx, query, user.FirstName, user.LastName, user.Email, user.Role).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to insert user from SSO: %w", err)
	}
	return id, nil
}
