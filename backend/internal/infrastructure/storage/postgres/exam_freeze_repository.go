package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

// PostgresExamFreezeRepository resuelve las dos lecturas del congelamiento de
// examen y la marca del laboratorio. Vive aparte del repositorio de aulas
// porque el congelamiento cruza sesión, participantes y laboratorios.
type PostgresExamFreezeRepository struct {
	db *sqlx.DB
}

func NewPostgresExamFreezeRepository(db *sqlx.DB) domain.ExamFreezeStore {
	return &PostgresExamFreezeRepository{db: db}
}

// SessionIsExam comprueba que la sesión sea de la institución y que su práctica
// tenga purpose = exam. Una sesión sin práctica no es examen.
func (r *PostgresExamFreezeRepository) SessionIsExam(ctx context.Context, tenantID, sessionID string) (bool, error) {
	var isExam bool
	err := r.db.GetContext(ctx, &isExam, `
		SELECT EXISTS (
			SELECT 1
			FROM class_sessions cs
			JOIN exercises e ON e.id = cs.exercise_id AND e.tenant_id = cs.tenant_id
			WHERE cs.id = $1::uuid AND cs.tenant_id = $2 AND e.purpose = 'exam'
		)
	`, sessionID, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to check whether session %s is an exam: %w", sessionID, err)
	}
	return isExam, nil
}

// ParticipantWorkspaces devuelve el laboratorio de cada participante en la
// materia del aula. Los alumnos sin entorno no aparecen: no hay volumen que
// congelar y el barrido no tiene que inventar filas.
func (r *PostgresExamFreezeRepository) ParticipantWorkspaces(ctx context.Context, tenantID, sessionID string) ([]domain.ExamFreezeWorkspace, error) {
	var rows []domain.ExamFreezeWorkspace
	err := r.db.SelectContext(ctx, &rows, `
		SELECT w.id AS workspace_id, w.student_id, w.status, w.frozen_at
		FROM class_session_participants p
		JOIN class_sessions cs ON cs.id = p.session_id AND cs.tenant_id = p.tenant_id
		JOIN workspaces w
		  ON w.student_id = p.student_id
		 AND w.subject_id = cs.subject_id
		 AND w.tenant_id = cs.tenant_id
		WHERE p.session_id = $1::uuid AND p.tenant_id = $2
		ORDER BY p.student_id
	`, sessionID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list participant workspaces for session %s: %w", sessionID, err)
	}
	return rows, nil
}

// ClosedExamSessions lista las aulas cerradas de examen que todavía tienen
// algún laboratorio editable. El barrido las recorre una vez y, al marcarlas,
// dejan de aparecer.
func (r *PostgresExamFreezeRepository) ClosedExamSessions(ctx context.Context, limit int) ([]domain.ExamSessionRef, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []domain.ExamSessionRef
	err := r.db.SelectContext(ctx, &rows, `
		SELECT cs.tenant_id, cs.id::text AS session_id
		FROM class_sessions cs
		JOIN exercises e ON e.id = cs.exercise_id AND e.tenant_id = cs.tenant_id
		JOIN class_session_participants p ON p.session_id = cs.id AND p.tenant_id = cs.tenant_id
		JOIN workspaces w
		  ON w.student_id = p.student_id
		 AND w.subject_id = cs.subject_id
		 AND w.tenant_id = cs.tenant_id
		WHERE cs.status = 'closed'
		  AND e.purpose = 'exam'
		  AND w.frozen_at IS NULL
		GROUP BY cs.tenant_id, cs.id
		ORDER BY MIN(COALESCE(cs.closes_at, cs.ends_at))
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list closed exam sessions to freeze: %w", err)
	}
	return rows, nil
}

// SetWorkspaceFrozen marca el congelamiento y ajusta el estado en una sola
// sentencia: no puede quedar un laboratorio con frozen_at viejo y estado
// running, ni al revés.
func (r *PostgresExamFreezeRepository) SetWorkspaceFrozen(ctx context.Context, tenantID, workspaceID string, frozen bool, at time.Time) error {
	status := domain.WorkspaceStatusRunning
	if frozen {
		status = domain.WorkspaceStatusFrozen
	}
	query := `
		UPDATE workspaces
		SET frozen_at = CASE WHEN $3::boolean THEN $4::timestamptz ELSE NULL END,
		    status = $5,
		    updated_at = NOW()
		WHERE id = $1::uuid AND tenant_id = $2
	`
	if _, err := r.db.ExecContext(ctx, query, workspaceID, tenantID, frozen, at, status); err != nil {
		return fmt.Errorf("failed to set frozen state for workspace %s: %w", workspaceID, err)
	}
	return nil
}
