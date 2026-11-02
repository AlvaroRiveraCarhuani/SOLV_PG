package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
)

// GetReviewEnvironmentContext arma el contexto del preview desde la entrega, su
// práctica y la plantilla. La autorización es la misma que la revisión docente:
// el submission debe pertenecer al tenant. El teacherID no filtra la consulta
// porque un docente del tenant puede revisar el curso; se conserva en la firma
// del puerto para cuando el repositorio necesite auditoría por actor.
func (r *PostgresTeacherRepository) GetReviewEnvironmentContext(ctx context.Context, tenantID, _ string, submissionID string) (*domain.ReviewEnvironmentContext, error) {
	if _, err := uuid.Parse(strings.TrimSpace(submissionID)); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidResourceID, submissionID)
	}
	if strings.TrimSpace(tenantID) == "" {
		return nil, domain.ErrInvalidTenant
	}

	query := `
		SELECT sub.id, sub.exercise_id,
		       COALESCE(ex.environment_type, 'JUEZ_EFIMERO') AS environment_type,
		       sub.code,
		       COALESCE(ex.language, '') AS language,
		       COALESCE(ex.config, '{}'::jsonb) AS config,
		       COALESCE(tpl.docker_image, '') AS docker_image,
		       COALESCE(tpl.setup_script, '') AS setup_script,
		       COALESCE(tpl.base_ram_mb, 0) AS base_ram_mb
		FROM submissions sub
		JOIN exercises ex ON sub.exercise_id = ex.id
		LEFT JOIN lab_templates tpl ON tpl.id = ex.template_id
		WHERE sub.id = $1 AND sub.tenant_id = $2
	`
	var row struct {
		SubmissionID    string `db:"id"`
		ExerciseID      string `db:"exercise_id"`
		EnvironmentType string `db:"environment_type"`
		Code            string `db:"code"`
		Language        string `db:"language"`
		Config          []byte `db:"config"`
		DockerImage     string `db:"docker_image"`
		SetupScript     string `db:"setup_script"`
		BaseRAMMB       int64  `db:"base_ram_mb"`
	}
	if err := r.db.GetContext(ctx, &row, query, submissionID, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to read review environment context: %w", err)
	}

	var config domain.ExerciseConfig
	if len(row.Config) > 0 {
		if err := json.Unmarshal(row.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to parse exercise config for review environment: %w", err)
		}
	}

	context := &domain.ReviewEnvironmentContext{
		SubmissionID:     row.SubmissionID,
		ExerciseID:       row.ExerciseID,
		EnvironmentType:  row.EnvironmentType,
		Code:             row.Code,
		Language:         row.Language,
		TemplateImage:    strings.TrimSpace(row.DockerImage),
		TemplateSetup:    strings.TrimSpace(row.SetupScript),
		TemplateMemoryMB: row.BaseRAMMB,
	}
	if config.IDE != nil {
		context.AppRunCommand = strings.TrimSpace(config.IDE.AppRunCommand)
		context.AppPort = config.IDE.AppPort
	}
	return context, nil
}
