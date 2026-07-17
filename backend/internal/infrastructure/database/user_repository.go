package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key value") {
			return "", errors.New("email already exists")
		}
		return "", fmt.Errorf("failed to insert user: %w", err)
	}

	return id, nil
}
