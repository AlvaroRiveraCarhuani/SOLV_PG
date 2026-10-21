package postgres

import (
	"github.com/jmoiron/sqlx"
)

type PostgresTeacherRepository struct {
	db *sqlx.DB
}

func NewPostgresTeacherRepository(db *sqlx.DB) *PostgresTeacherRepository {
	return &PostgresTeacherRepository{db: db}
}
