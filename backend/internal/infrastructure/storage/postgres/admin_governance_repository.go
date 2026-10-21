package postgres

import (
	"github.com/jmoiron/sqlx"
)

type PostgresAdminGovernanceRepository struct {
	db *sqlx.DB
}

func NewPostgresAdminGovernanceRepository(db *sqlx.DB) *PostgresAdminGovernanceRepository {
	return &PostgresAdminGovernanceRepository{db: db}
}
