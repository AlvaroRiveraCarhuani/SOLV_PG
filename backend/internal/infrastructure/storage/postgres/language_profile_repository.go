package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresLanguageProfileRepository struct {
	db *sqlx.DB
}

func NewPostgresLanguageProfileRepository(db *sqlx.DB) domain.LanguageProfileRepository {
	return &PostgresLanguageProfileRepository{db: db}
}

func (r *PostgresLanguageProfileRepository) ListProfiles(ctx context.Context) ([]*domain.LanguageProfile, error) {
	query := `
		SELECT language, default_timeout_ms, default_memory_mb, image,
		       build_command, build_timeout_ms, build_memory_mb,
		       p95_window_days, checker_sidecar_image, created_at, updated_at
		FROM language_profiles
		ORDER BY language ASC
	`
	var profiles []*domain.LanguageProfile
	err := r.db.SelectContext(ctx, &profiles, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list language profiles: %w", err)
	}
	return profiles, nil
}

func (r *PostgresLanguageProfileRepository) GetProfileByLanguage(ctx context.Context, language string) (*domain.LanguageProfile, error) {
	query := `
		SELECT language, default_timeout_ms, default_memory_mb, image,
		       build_command, build_timeout_ms, build_memory_mb,
		       p95_window_days, checker_sidecar_image, created_at, updated_at
		FROM language_profiles
		WHERE language = $1
	`
	var profile domain.LanguageProfile
	err := r.db.GetContext(ctx, &profile, query, language)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUnknownLanguage
		}
		return nil, fmt.Errorf("failed to get language profile for %s: %w", language, err)
	}
	return &profile, nil
}

func (r *PostgresLanguageProfileRepository) UpdateProfile(ctx context.Context, profile *domain.LanguageProfile, audit *domain.LanguageProfileAudit) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for language profile update: %w", err)
	}
	defer tx.Rollback()

	updateQuery := `
		UPDATE language_profiles
		SET default_timeout_ms = $1,
		    default_memory_mb = $2,
		    image = $3,
		    build_command = $4,
		    build_timeout_ms = $5,
		    build_memory_mb = $6,
		    p95_window_days = $7,
		    checker_sidecar_image = $8,
		    updated_at = NOW()
		WHERE language = $9
	`
	res, err := tx.ExecContext(ctx, updateQuery,
		profile.DefaultTimeoutMS,
		profile.DefaultMemoryMB,
		profile.Image,
		profile.BuildCommand,
		profile.BuildTimeoutMS,
		profile.BuildMemoryMB,
		profile.P95WindowDays,
		profile.CheckerSidecarImage,
		profile.Language,
	)
	if err != nil {
		return fmt.Errorf("failed to update language profile for %s: %w", profile.Language, err)
	}

	rowsAff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAff == 0 {
		return domain.ErrUnknownLanguage
	}

	if audit != nil {
		auditQuery := `
			INSERT INTO language_profile_audits (id, language, author, old_values, new_values, reason, created_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
		`
		createdAt := audit.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		_, err = tx.ExecContext(ctx, auditQuery,
			profile.Language,
			audit.Author,
			audit.OldValues,
			audit.NewValues,
			audit.Reason,
			createdAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert language profile audit: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit language profile update transaction: %w", err)
	}

	return nil
}
