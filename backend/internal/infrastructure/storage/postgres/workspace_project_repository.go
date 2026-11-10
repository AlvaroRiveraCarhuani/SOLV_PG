package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"solv-backend/internal/core/domain"
)

type PostgresWorkspaceProjectRepository struct {
	db *sqlx.DB
}

func NewPostgresWorkspaceProjectRepository(db *sqlx.DB) *PostgresWorkspaceProjectRepository {
	return &PostgresWorkspaceProjectRepository{db: db}
}

func (r *PostgresWorkspaceProjectRepository) GetProjectByExercise(ctx context.Context, tenantID, exerciseID string) (*domain.WorkspaceProject, error) {
	var project domain.WorkspaceProject
	query := `
		SELECT id, tenant_id, exercise_id, current_published_version_id, created_by, created_at, updated_at
		FROM workspace_projects
		WHERE tenant_id = $1 AND exercise_id = $2
	`
	if err := r.db.GetContext(ctx, &project, query, tenantID, exerciseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWorkspaceProjectNotFound
		}
		return nil, fmt.Errorf("failed to get workspace project: %w", err)
	}
	return &project, nil
}

func (r *PostgresWorkspaceProjectRepository) CreateProject(ctx context.Context, project *domain.WorkspaceProject) (*domain.WorkspaceProject, error) {
	if project.ID == "" {
		project.ID = uuid.NewString()
	}
	query := `
		INSERT INTO workspace_projects (id, tenant_id, exercise_id, created_by, created_at, updated_at)
		VALUES (:id, :tenant_id, :exercise_id, :created_by, NOW(), NOW())
		RETURNING id, tenant_id, exercise_id, current_published_version_id, created_by, created_at, updated_at
	`
	rows, err := r.db.NamedQueryContext(ctx, query, project)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, domain.ErrWorkspaceProjectExists
		}
		return nil, fmt.Errorf("failed to create workspace project: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, fmt.Errorf("failed to create workspace project: no row returned")
	}
	var created domain.WorkspaceProject
	if err := rows.StructScan(&created); err != nil {
		return nil, fmt.Errorf("failed to scan workspace project: %w", err)
	}
	return &created, nil
}

func (r *PostgresWorkspaceProjectRepository) GetDraftByProject(ctx context.Context, tenantID, projectID string) (*domain.WorkspaceProjectDraft, error) {
	var draft domain.WorkspaceProjectDraft
	query := `
		SELECT id, tenant_id, project_id, base_version_id, text_snapshot, content_sha256, revision, updated_by, created_at, updated_at
		FROM workspace_project_drafts
		WHERE tenant_id = $1 AND project_id = $2
	`
	if err := r.db.GetContext(ctx, &draft, query, tenantID, projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWorkspaceProjectNotFound
		}
		return nil, fmt.Errorf("failed to get workspace project draft: %w", err)
	}
	return &draft, nil
}

// SaveDraft usa revisión optimista. expectedRevision=0 crea el borrador; un
// borrador existente devuelve conflicto. Un expectedRevision positivo exige que
// la revisión actual coincida.
func (r *PostgresWorkspaceProjectRepository) SaveDraft(ctx context.Context, draft *domain.WorkspaceProjectDraft, expectedRevision int64) (*domain.WorkspaceProjectDraft, error) {
	if draft.ID == "" {
		draft.ID = uuid.NewString()
	}
	draft.ContentSHA256 = hashWorkspaceProjectText(draft.TextSnapshot)
	if expectedRevision <= 0 {
		query := `
			INSERT INTO workspace_project_drafts
				(id, tenant_id, project_id, base_version_id, text_snapshot, content_sha256, revision, updated_by, created_at, updated_at)
			VALUES
				(:id, :tenant_id, :project_id, :base_version_id, :text_snapshot, :content_sha256, 1, :updated_by, NOW(), NOW())
			RETURNING id, tenant_id, project_id, base_version_id, text_snapshot, content_sha256, revision, updated_by, created_at, updated_at
		`
		rows, err := r.db.NamedQueryContext(ctx, query, draft)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				return nil, domain.ErrWorkspaceDraftRevisionConflict
			}
			return nil, fmt.Errorf("failed to create workspace project draft: %w", err)
		}
		defer func() { _ = rows.Close() }()
		if !rows.Next() {
			return nil, fmt.Errorf("failed to create workspace project draft: no row returned")
		}
		var saved domain.WorkspaceProjectDraft
		if err := rows.StructScan(&saved); err != nil {
			return nil, fmt.Errorf("failed to scan workspace project draft: %w", err)
		}
		return &saved, nil
	}

	query := `
		UPDATE workspace_project_drafts
		SET base_version_id = :base_version_id,
		    text_snapshot = :text_snapshot,
		    content_sha256 = :content_sha256,
		    revision = revision + 1,
		    updated_by = :updated_by,
		    updated_at = NOW()
		WHERE tenant_id = :tenant_id
		  AND project_id = :project_id
		  AND revision = :expected_revision
		RETURNING id, tenant_id, project_id, base_version_id, text_snapshot, content_sha256, revision, updated_by, created_at, updated_at
	`
	args := map[string]interface{}{
		"tenant_id":         draft.TenantID,
		"project_id":        draft.ProjectID,
		"base_version_id":   draft.BaseVersionID,
		"text_snapshot":     draft.TextSnapshot,
		"content_sha256":    draft.ContentSHA256,
		"updated_by":        draft.UpdatedBy,
		"expected_revision": expectedRevision,
	}
	rows, err := r.db.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to update workspace project draft: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, domain.ErrWorkspaceDraftRevisionConflict
	}
	var saved domain.WorkspaceProjectDraft
	if err := rows.StructScan(&saved); err != nil {
		return nil, fmt.Errorf("failed to scan workspace project draft: %w", err)
	}
	return &saved, nil
}

// Publish serializa las publicaciones con FOR UPDATE del proyecto, crea la
// versión, actualiza el espejo del ejercicio y mueve el puntero del proyecto en
// una sola transacción.
func (r *PostgresWorkspaceProjectRepository) Publish(ctx context.Context, tenantID, projectID, userID string) (*domain.WorkspaceProjectVersion, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin workspace publish: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var project domain.WorkspaceProject
	if err := tx.GetContext(ctx, &project, `
		SELECT id, tenant_id, exercise_id, current_published_version_id, created_by, created_at, updated_at
		FROM workspace_projects
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE
	`, projectID, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWorkspaceProjectNotFound
		}
		return nil, fmt.Errorf("failed to lock workspace project: %w", err)
	}

	var draft domain.WorkspaceProjectDraft
	if err := tx.GetContext(ctx, &draft, `
		SELECT id, tenant_id, project_id, base_version_id, text_snapshot, content_sha256, revision, updated_by, created_at, updated_at
		FROM workspace_project_drafts
		WHERE project_id = $1 AND tenant_id = $2
		FOR UPDATE
	`, projectID, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWorkspaceDraftEmpty
		}
		return nil, fmt.Errorf("failed to lock workspace project draft: %w", err)
	}
	if strings.TrimSpace(draft.TextSnapshot) == "" {
		return nil, domain.ErrWorkspaceDraftEmpty
	}

	files, ok := domain.ParseProjectSnapshot(draft.TextSnapshot, "")
	if !ok || len(files) == 0 {
		return nil, fmt.Errorf("%w: snapshot de proyecto inválido", domain.ErrWorkspaceDraftEmpty)
	}
	manifestEntries := make([]domain.WorkspaceProjectManifestEntry, 0, len(files))
	totalBytes := 0
	for _, file := range files {
		totalBytes += len(file.Content)
		manifestEntries = append(manifestEntries, domain.WorkspaceProjectManifestEntry{
			Path:  file.Path,
			Bytes: len(file.Content),
			Lines: file.LineCount,
		})
	}
	manifestBytes, err := json.Marshal(manifestEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to encode project manifest: %w", err)
	}
	contentHash := hashWorkspaceProjectText(draft.TextSnapshot)

	var nextVersion int
	if err := tx.GetContext(ctx, &nextVersion, `
		SELECT COALESCE(MAX(version_number), 0) + 1
		FROM workspace_project_versions
		WHERE project_id = $1 AND tenant_id = $2
	`, projectID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to resolve next project version: %w", err)
	}

	version := &domain.WorkspaceProjectVersion{
		ID:            uuid.NewString(),
		TenantID:      tenantID,
		ProjectID:     project.ID,
		ExerciseID:    project.ExerciseID,
		VersionNumber: nextVersion,
		TextSnapshot:  draft.TextSnapshot,
		Manifest:      json.RawMessage(manifestBytes),
		ContentSHA256: contentHash,
		FileCount:     len(files),
		TotalBytes:    totalBytes,
		CreatedBy:     nullableString(userID),
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO workspace_project_versions
			(id, tenant_id, project_id, exercise_id, version_number, text_snapshot, manifest, content_sha256, file_count, total_bytes, created_by, published_at, created_at)
		VALUES
			(:id, :tenant_id, :project_id, :exercise_id, :version_number, :text_snapshot, :manifest, :content_sha256, :file_count, :total_bytes, :created_by, NOW(), NOW())
	`, version); err != nil {
		return nil, fmt.Errorf("failed to insert project version: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE exercises
		SET boilerplate = $1,
		    boilerplate_sha256 = $2,
		    boilerplate_source_version_id = $3
		WHERE id = $4 AND tenant_id = $5
	`, draft.TextSnapshot, contentHash, version.ID, project.ExerciseID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to update exercise boilerplate mirror: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE workspace_projects
		SET current_published_version_id = $1,
		    updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`, version.ID, project.ID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to move project published pointer: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE workspace_project_drafts
		SET base_version_id = $1,
		    updated_at = NOW()
		WHERE project_id = $2 AND tenant_id = $3
	`, version.ID, project.ID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to update draft base version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit workspace publish: %w", err)
	}
	version.PublishedAt = time.Now().UTC()
	return version, nil
}

func (r *PostgresWorkspaceProjectRepository) ListVersions(ctx context.Context, tenantID, projectID string) ([]*domain.WorkspaceProjectVersion, error) {
	var versions []*domain.WorkspaceProjectVersion
	query := `
		SELECT id, tenant_id, project_id, exercise_id, version_number, text_snapshot, manifest, content_sha256, file_count, total_bytes, created_by, published_at, created_at
		FROM workspace_project_versions
		WHERE tenant_id = $1 AND project_id = $2
		ORDER BY version_number DESC
	`
	if err := r.db.SelectContext(ctx, &versions, query, tenantID, projectID); err != nil {
		return nil, fmt.Errorf("failed to list project versions: %w", err)
	}
	if versions == nil {
		versions = make([]*domain.WorkspaceProjectVersion, 0)
	}
	return versions, nil
}

func (r *PostgresWorkspaceProjectRepository) GetVersion(ctx context.Context, tenantID, versionID string) (*domain.WorkspaceProjectVersion, error) {
	var version domain.WorkspaceProjectVersion
	query := `
		SELECT id, tenant_id, project_id, exercise_id, version_number, text_snapshot, manifest, content_sha256, file_count, total_bytes, created_by, published_at, created_at
		FROM workspace_project_versions
		WHERE tenant_id = $1 AND id = $2
	`
	if err := r.db.GetContext(ctx, &version, query, tenantID, versionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWorkspaceProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project version: %w", err)
	}
	return &version, nil
}

func (r *PostgresWorkspaceProjectRepository) GetLegacyReference(ctx context.Context, tenantID, exerciseID string) (*domain.WorkspaceLegacyProjectReference, error) {
	var reference domain.WorkspaceLegacyProjectReference
	query := `
		SELECT id, tenant_id, exercise_id, text_snapshot, content_sha256, source, captured_at
		FROM workspace_legacy_project_references
		WHERE tenant_id = $1 AND exercise_id = $2
	`
	if err := r.db.GetContext(ctx, &reference, query, tenantID, exerciseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWorkspaceProjectNotFound
		}
		return nil, fmt.Errorf("failed to get legacy project reference: %w", err)
	}
	return &reference, nil
}

func hashWorkspaceProjectText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
