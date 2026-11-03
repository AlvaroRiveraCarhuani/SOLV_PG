package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

// PostgresEnvironmentRequestRepository guarda las solicitudes de entorno del
// docente. Las piezas y materias viajan como JSONB porque son una foto del
// pedido: no se editan por partes ni necesitan integridad referencial.
type PostgresEnvironmentRequestRepository struct {
	db *sqlx.DB
}

func NewPostgresEnvironmentRequestRepository(db *sqlx.DB) *PostgresEnvironmentRequestRepository {
	return &PostgresEnvironmentRequestRepository{db: db}
}

type environmentRequestRow struct {
	ID                 string         `db:"id"`
	TenantID           string         `db:"tenant_id"`
	RequestedBy        string         `db:"requested_by"`
	RequesterName      sql.NullString `db:"requester_name"`
	Title              string         `db:"title"`
	Description        string         `db:"description"`
	TeacherText        string         `db:"teacher_text"`
	Uses               []byte         `db:"uses"`
	TargetEnvironment  string         `db:"target_environment"`
	StackConfig        []byte         `db:"stack_config"`
	Components         []byte         `db:"components"`
	MissingComponents  []byte         `db:"missing_components"`
	Subjects           []byte         `db:"subjects"`
	StackSignature     string         `db:"stack_signature"`
	Status             string         `db:"status"`
	ReviewNote         string         `db:"review_note"`
	ReviewedBy         sql.NullString `db:"reviewed_by"`
	ReviewedAt         sql.NullTime   `db:"reviewed_at"`
	ResolvedTemplateID sql.NullString `db:"resolved_template_id"`
	CreatedAt          time.Time      `db:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at"`
}

const environmentRequestSelect = `
	SELECT
		er.id::text AS id,
		er.tenant_id::text AS tenant_id,
		er.requested_by::text AS requested_by,
		NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), '') AS requester_name,
		er.title,
		er.description,
		er.teacher_text,
		COALESCE(er.uses, '[]'::jsonb) AS uses,
		er.target_environment,
		COALESCE(er.stack_config, '{}'::jsonb) AS stack_config,
		COALESCE(er.components, '[]'::jsonb) AS components,
		COALESCE(er.missing_components, '[]'::jsonb) AS missing_components,
		COALESCE(er.subjects, '[]'::jsonb) AS subjects,
		er.stack_signature,
		er.status,
		COALESCE(er.review_note, '') AS review_note,
		er.reviewed_by::text AS reviewed_by,
		er.reviewed_at,
		er.resolved_template_id::text AS resolved_template_id,
		er.created_at,
		er.updated_at
	FROM environment_requests er
	LEFT JOIN users u ON u.id = er.requested_by
`

func (r *environmentRequestRow) toDomain() *domain.EnvironmentRequest {
	req := &domain.EnvironmentRequest{
		ID:                 r.ID,
		TenantID:           r.TenantID,
		RequestedBy:        r.RequestedBy,
		RequesterName:      r.RequesterName.String,
		Title:              r.Title,
		Description:        r.Description,
		TeacherText:        r.TeacherText,
		TargetEnvironment:  r.TargetEnvironment,
		StackSignature:     r.StackSignature,
		Status:             r.Status,
		ReviewNote:         r.ReviewNote,
		ReviewedBy:         r.ReviewedBy.String,
		ResolvedTemplateID: r.ResolvedTemplateID.String,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
	if r.ReviewedAt.Valid {
		reviewedAt := r.ReviewedAt.Time
		req.ReviewedAt = &reviewedAt
	}
	_ = json.Unmarshal(r.Uses, &req.Uses)
	_ = json.Unmarshal(r.StackConfig, &req.StackConfig)
	_ = json.Unmarshal(r.Components, &req.Components)
	_ = json.Unmarshal(r.MissingComponents, &req.MissingComponents)
	_ = json.Unmarshal(r.Subjects, &req.Subjects)
	if req.Uses == nil {
		req.Uses = []string{}
	}
	if req.Components == nil {
		req.Components = []domain.EnvironmentRequestComponent{}
	}
	if req.MissingComponents == nil {
		req.MissingComponents = []domain.EnvironmentRequestMissingComponent{}
	}
	if req.Subjects == nil {
		req.Subjects = []domain.EnvironmentRequestSubject{}
	}
	return req
}

func (r *PostgresEnvironmentRequestRepository) Create(
	ctx context.Context, req *domain.EnvironmentRequest,
) (*domain.EnvironmentRequest, error) {
	uses, _ := json.Marshal(req.Uses)
	stackConfig, _ := json.Marshal(req.StackConfig)
	components, _ := json.Marshal(req.Components)
	missing, _ := json.Marshal(req.MissingComponents)
	subjects, _ := json.Marshal(req.Subjects)

	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO environment_requests (
			tenant_id, requested_by, title, description, teacher_text, uses,
			target_environment, stack_config, components, missing_components,
			subjects, stack_signature, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id::text
	`,
		req.TenantID, req.RequestedBy, req.Title, req.Description, req.TeacherText,
		string(uses), req.TargetEnvironment, string(stackConfig), string(components),
		string(missing), string(subjects), req.StackSignature, req.Status,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("error creating environment request: %w", err)
	}
	return r.GetByID(ctx, req.TenantID, id)
}

func (r *PostgresEnvironmentRequestRepository) FindPendingBySignature(
	ctx context.Context, tenantID, signature string,
) (*domain.EnvironmentRequest, error) {
	if strings.TrimSpace(signature) == "" {
		return nil, nil
	}
	var row environmentRequestRow
	err := r.db.GetContext(ctx, &row, environmentRequestSelect+`
		WHERE er.tenant_id = $1
		  AND er.stack_signature = $2
		  AND er.status = 'EN_REVISION'
		ORDER BY er.created_at ASC
		LIMIT 1
	`, tenantID, signature)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error finding pending environment request: %w", err)
	}
	return row.toDomain(), nil
}

func (r *PostgresEnvironmentRequestRepository) AddSubject(
	ctx context.Context, requestID string, subject domain.EnvironmentRequestSubject,
) error {
	payload, err := json.Marshal([]domain.EnvironmentRequestSubject{subject})
	if err != nil {
		return fmt.Errorf("error encoding subject: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE environment_requests
		SET subjects = COALESCE(subjects, '[]'::jsonb) || $2::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		  AND NOT EXISTS (
			SELECT 1 FROM jsonb_array_elements(COALESCE(subjects, '[]'::jsonb)) s
			WHERE ($3 <> '' AND s->>'subject_id' = $3)
			   OR ($3 = '' AND s->>'subject_name' = $4)
		  )
	`, requestID, string(payload), subject.SubjectID, subject.SubjectName)
	if err != nil {
		return fmt.Errorf("error adding subject to environment request: %w", err)
	}
	return nil
}

func (r *PostgresEnvironmentRequestRepository) ListMine(
	ctx context.Context, tenantID, userID string,
) ([]*domain.EnvironmentRequest, error) {
	var rows []environmentRequestRow
	err := r.db.SelectContext(ctx, &rows, environmentRequestSelect+`
		WHERE er.tenant_id = $1 AND er.requested_by = $2
		ORDER BY er.created_at DESC
	`, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("error listing my environment requests: %w", err)
	}
	result := make([]*domain.EnvironmentRequest, len(rows))
	for i := range rows {
		result[i] = rows[i].toDomain()
	}
	return result, nil
}

func (r *PostgresEnvironmentRequestRepository) GetByID(
	ctx context.Context, tenantID, id string,
) (*domain.EnvironmentRequest, error) {
	var row environmentRequestRow
	err := r.db.GetContext(ctx, &row, environmentRequestSelect+`
		WHERE er.tenant_id = $1 AND er.id = $2
	`, tenantID, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrEnvironmentRequestNotFound
		}
		return nil, fmt.Errorf("error getting environment request: %w", err)
	}
	return row.toDomain(), nil
}

func (r *PostgresEnvironmentRequestRepository) ListForReview(
	ctx context.Context, tenantID, status string,
) ([]*domain.EnvironmentRequest, error) {
	args := []interface{}{tenantID}
	query := environmentRequestSelect + ` WHERE er.tenant_id = $1`
	if strings.TrimSpace(status) != "" {
		query += ` AND er.status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY er.created_at ASC`

	var rows []environmentRequestRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("error listing environment requests for review: %w", err)
	}
	result := make([]*domain.EnvironmentRequest, len(rows))
	for i := range rows {
		result[i] = rows[i].toDomain()
	}
	return result, nil
}

func (r *PostgresEnvironmentRequestRepository) Review(
	ctx context.Context, tenantID, id, reviewerID string, dto domain.ReviewEnvironmentRequestDTO,
) (*domain.EnvironmentRequest, error) {
	var templateID interface{}
	if strings.TrimSpace(dto.ResolvedTemplateID) != "" {
		templateID = strings.TrimSpace(dto.ResolvedTemplateID)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE environment_requests
		SET status = $3,
		    review_note = $4,
		    reviewed_by = NULLIF($5, '')::uuid,
		    reviewed_at = NOW(),
		    resolved_template_id = $6::uuid,
		    updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id, dto.Status, dto.ReviewNote, reviewerID, templateID)
	if err != nil {
		return nil, fmt.Errorf("error reviewing environment request: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, domain.ErrEnvironmentRequestNotFound
	}
	return r.GetByID(ctx, tenantID, id)
}

func (r *PostgresEnvironmentRequestRepository) FindApprovedTemplateByImage(
	ctx context.Context, tenantID, image string,
) (*domain.EnvironmentRequestTemplateRef, error) {
	if strings.TrimSpace(image) == "" {
		return nil, nil
	}
	var ref domain.EnvironmentRequestTemplateRef
	err := r.db.GetContext(ctx, &ref, fmt.Sprintf(`
		SELECT id::text AS id, name
		FROM lab_templates lt
		WHERE (lt.tenant_id = $1 OR lt.tenant_id IS NULL)
		  AND lt.docker_image = $2
		  AND %s
		ORDER BY lt.created_at ASC
		LIMIT 1
	`, approvedTemplateStatusClause("lt")), tenantID, image)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error finding approved template by image: %w", err)
	}
	return &ref, nil
}
