package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

// PostgresClassAdvanceRepository guarda los avances del docente y sus
// destinatarios (ADR-058). La sesión se lee y se ajusta con el repositorio de
// aula, para que el alcance del docente y la expiración se resuelvan en un solo
// lugar.
type PostgresClassAdvanceRepository struct {
	db       *sqlx.DB
	sessions *PostgresClassSessionRepository
}

func NewPostgresClassAdvanceRepository(db *sqlx.DB, sessions *PostgresClassSessionRepository) *PostgresClassAdvanceRepository {
	return &PostgresClassAdvanceRepository{db: db, sessions: sessions}
}

var _ domain.AdvanceSnapshotRetentionStore = (*PostgresClassAdvanceRepository)(nil)

// CreateAdvance inserta el avance y sus destinatarios en una sola transacción:
// o queda el avance completo con todos sus alumnos, o no queda nada.
func (r *PostgresClassAdvanceRepository) CreateAdvance(ctx context.Context, advance domain.ClassAdvance, studentIDs []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin advance transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO class_advances (
			id, tenant_id, session_id, exercise_id, scope,
			snapshot_key, snapshot_sha256, snapshot_size_bytes, file_count,
			created_by, created_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10::uuid, $11)
	`, advance.ID, advance.TenantID, advance.SessionID, advance.ExerciseID, advance.Scope,
		advance.SnapshotKey, advance.SnapshotSHA256, advance.SnapshotSize, advance.FileCount,
		advance.CreatedBy, advance.CreatedAt); err != nil {
		return fmt.Errorf("failed to insert class advance: %w", err)
	}

	for _, studentID := range studentIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO class_advance_recipients (advance_id, student_id, tenant_id, status)
			VALUES ($1::uuid, $2::uuid, $3::uuid, 'pending')
		`, advance.ID, studentID, advance.TenantID); err != nil {
			return fmt.Errorf("failed to insert advance recipient: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit class advance: %w", err)
	}
	return nil
}

// SessionParticipants devuelve los alumnos que han pasado por el aula.
func (r *PostgresClassAdvanceRepository) SessionParticipants(ctx context.Context, tenantID, sessionID string) ([]string, error) {
	var ids []string
	if err := r.db.SelectContext(ctx, &ids, `
		SELECT student_id::text
		FROM class_session_participants
		WHERE tenant_id = $1 AND session_id = $2::uuid
		ORDER BY student_id
	`, tenantID, sessionID); err != nil {
		return nil, fmt.Errorf("failed to load session participants: %w", err)
	}
	return ids, nil
}

// ExercisePurpose devuelve 'class' o 'exam' del ejercicio del tenant.
func (r *PostgresClassAdvanceRepository) ExercisePurpose(ctx context.Context, tenantID, exerciseID string) (string, error) {
	var purpose string
	if err := r.db.GetContext(ctx, &purpose, `
		SELECT purpose FROM exercises WHERE id = $1::uuid AND tenant_id = $2
	`, exerciseID, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("failed to load exercise purpose: %w", err)
	}
	return purpose, nil
}

// advanceDeliveryQuery resuelve el avance, el destinatario y el estado de su
// sesión en una sola consulta. El filtro por student_id es la autorización: un
// avance ajeno no aparece.
const advanceDeliveryQuery = `
	SELECT ca.id, ca.tenant_id, ca.session_id, ca.exercise_id, ca.scope,
	       ca.snapshot_key, ca.snapshot_sha256, ca.snapshot_size_bytes, ca.file_count,
	       ca.created_by, ca.created_at,
	       r.status AS recipient_status, r.decided_at,
	       cs.subject_id, cs.status AS session_status, cs.grace_until,
	       e.purpose
	FROM class_advance_recipients r
	JOIN class_advances ca ON ca.id = r.advance_id
	JOIN class_sessions cs ON cs.id = ca.session_id
	JOIN exercises e ON e.id = ca.exercise_id
	WHERE r.tenant_id = $1 AND r.student_id = $2::uuid`

// PendingAdvances lista los avisos pendientes del alumno, del más nuevo al más
// viejo. Solo se muestran los que todavía se pueden aplicar.
func (r *PostgresClassAdvanceRepository) PendingAdvances(ctx context.Context, tenantID, studentID string) ([]domain.AdvanceDelivery, error) {
	deliveries := make([]domain.AdvanceDelivery, 0, 4)
	query := advanceDeliveryQuery + `
	  AND r.status = 'pending'
	ORDER BY ca.created_at DESC`
	if err := r.db.SelectContext(ctx, &deliveries, query, tenantID, studentID); err != nil {
		return nil, fmt.Errorf("failed to load pending advances: %w", err)
	}
	return deliveries, nil
}

// AdvanceForStudent devuelve el avance del alumno autenticado. Un avance de otro
// alumno, de otro tenant o inexistente responde lo mismo: no encontrado.
func (r *PostgresClassAdvanceRepository) AdvanceForStudent(ctx context.Context, tenantID, studentID, advanceID string) (*domain.AdvanceDelivery, error) {
	var delivery domain.AdvanceDelivery
	query := advanceDeliveryQuery + ` AND ca.id = $3::uuid`
	if err := r.db.GetContext(ctx, &delivery, query, tenantID, studentID, advanceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAdvanceNotFound
		}
		return nil, fmt.Errorf("failed to load class advance: %w", err)
	}
	return &delivery, nil
}

// DecideRecipient fija el estado final una sola vez. La condición status =
// 'pending' evita que dos pestañas del mismo alumno dejen estados distintos.
func (r *PostgresClassAdvanceRepository) DecideRecipient(ctx context.Context, tenantID, studentID, advanceID, status string, backupPath *string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE class_advance_recipients
		SET status = $4, decided_at = NOW(), backup_path = $5
		WHERE advance_id = $1::uuid AND student_id = $2::uuid AND tenant_id = $3
		  AND status = 'pending'
	`, advanceID, studentID, tenantID, status, backupPath)
	if err != nil {
		return fmt.Errorf("failed to decide advance recipient: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrAdvanceNotPending
	}
	return nil
}

// advanceRecipientRow es una fila plana de emisión y destinatario. La consulta
// trae nombre y correo para que el docente no cruce identificadores.
type advanceRecipientRow struct {
	AdvanceID     string     `db:"advance_id"`
	ExerciseID    string     `db:"exercise_id"`
	ExerciseTitle string     `db:"exercise_title"`
	Scope         string     `db:"scope"`
	FileCount     int        `db:"file_count"`
	CreatedAt     time.Time  `db:"created_at"`
	StudentID     string     `db:"student_id"`
	Name          string     `db:"name"`
	Email         string     `db:"email"`
	Status        string     `db:"status"`
	DecidedAt     *time.Time `db:"decided_at"`
}

// SessionAdvances devuelve las emisiones del aula con el estado de cada
// destinatario. El filtro por docente dueño es la autorización: el aula de otra
// materia no aparece.
func (r *PostgresClassAdvanceRepository) SessionAdvances(ctx context.Context, tenantID, teacherID, sessionID string) ([]domain.AdvanceWithRecipients, error) {
	// El docente tiene que ver vencidos aunque el alumno nunca vuelva a abrir el
	// laboratorio: la lectura de la sesión cierra los pendientes de una ventana
	// que ya terminó.
	if _, err := r.db.ExecContext(ctx, `
		UPDATE class_advance_recipients r
		SET status = 'expired', decided_at = COALESCE(r.decided_at, NOW())
		FROM class_advances ca
		JOIN class_sessions cs ON cs.id = ca.session_id
		WHERE r.advance_id = ca.id
		  AND r.tenant_id = $1
		  AND ca.session_id = $2::uuid
		  AND r.status = 'pending'
		  AND cs.status = 'closed'
		  AND (cs.grace_until IS NULL OR cs.grace_until <= NOW())
		  AND ($3 = '' OR EXISTS (
		      SELECT 1 FROM subjects s
		      WHERE s.id = cs.subject_id AND s.teacher_id = $3::uuid
		  ))
	`, tenantID, sessionID, teacherID); err != nil {
		return nil, fmt.Errorf("failed to close expired advance recipients: %w", err)
	}

	rows := make([]advanceRecipientRow, 0, 32)
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT ca.id AS advance_id, ca.exercise_id, e.title AS exercise_title,
		       ca.scope, ca.file_count, ca.created_at,
		       r.student_id,
		       COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email) AS name,
		       COALESCE(u.email, '') AS email,
		       r.status, r.decided_at
		FROM class_advances ca
		JOIN class_sessions cs ON cs.id = ca.session_id
		JOIN subjects s ON s.id = cs.subject_id
		JOIN exercises e ON e.id = ca.exercise_id
		JOIN class_advance_recipients r ON r.advance_id = ca.id
		LEFT JOIN users u ON u.id = r.student_id
		WHERE ca.tenant_id = $1 AND ca.session_id = $2::uuid
		  AND ($3 = '' OR s.teacher_id = $3::uuid)
		ORDER BY ca.created_at DESC, name
	`, tenantID, sessionID, teacherID); err != nil {
		return nil, fmt.Errorf("failed to load session advances: %w", err)
	}
	return groupSessionAdvances(rows), nil
}

// classHistoryRow es una fila de audit_logs ya unida al actor. El metadata se
// guarda como JSONB y se traduce a los campos que la vista necesita.
type classHistoryRow struct {
	ID         string    `db:"id"`
	Action     string    `db:"action"`
	ActorID    string    `db:"actor_id"`
	ActorName  string    `db:"actor_name"`
	ActorEmail string    `db:"actor_email"`
	ActorRole  string    `db:"actor_role"`
	Metadata   []byte    `db:"metadata"`
	CreatedAt  time.Time `db:"created_at"`
}

// classHistoryMetadata es el detalle que cada acción deja en el audit log.
type classHistoryMetadata struct {
	AdvanceID     string     `json:"advance_id"`
	ExerciseID    string     `json:"exercise_id"`
	Scope         string     `json:"scope"`
	Recipients    int        `json:"recipients"`
	FileCount     int        `json:"file_count"`
	FilesWritten  int        `json:"files_written"`
	Minutes       int        `json:"minutes"`
	PreviousUntil *time.Time `json:"previous_until"`
	Until         *time.Time `json:"until"`
}

// SessionHistory lee los movimientos del aula desde audit_logs. El filtro por
// resource_type y resource_id usa el índice parcial de la migración 00039: el
// historial no recorre todo el registro de la institución.
func (r *PostgresClassAdvanceRepository) SessionHistory(ctx context.Context, tenantID, sessionID string, limit int) ([]domain.ClassHistoryEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	rows := make([]classHistoryRow, 0, 32)
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT a.id, a.action, a.actor_id,
		       COALESCE(NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), ''),
		                u.email, a.actor_id::text) AS actor_name,
		       COALESCE(u.email, '') AS actor_email,
		       COALESCE(u.role, '') AS actor_role,
		       a.metadata, a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_id
		WHERE a.tenant_id = $1
		  AND a.resource_type = 'class_session'
		  AND a.resource_id = $2::uuid
		  AND a.action LIKE 'CLASS\_%' ESCAPE '\'
		ORDER BY a.created_at DESC
		LIMIT $3
	`, tenantID, sessionID, limit); err != nil {
		return nil, fmt.Errorf("failed to load class session history: %w", err)
	}

	events := make([]domain.ClassHistoryEvent, 0, len(rows))
	for _, row := range rows {
		event := domain.ClassHistoryEvent{
			ID:         row.ID,
			Action:     row.Action,
			ActorID:    row.ActorID,
			ActorName:  row.ActorName,
			ActorEmail: row.ActorEmail,
			ActorRole:  row.ActorRole,
			CreatedAt:  row.CreatedAt,
		}
		if len(row.Metadata) > 0 {
			var meta classHistoryMetadata
			if err := json.Unmarshal(row.Metadata, &meta); err == nil {
				event.Scope = meta.Scope
				event.Recipients = meta.Recipients
				event.FileCount = meta.FileCount
				event.FilesWritten = meta.FilesWritten
				event.Minutes = meta.Minutes
				event.PreviousUntil = meta.PreviousUntil
				event.Until = meta.Until
				if meta.AdvanceID != "" {
					advanceID := meta.AdvanceID
					event.AdvanceID = &advanceID
				}
				if meta.ExerciseID != "" {
					exerciseID := meta.ExerciseID
					event.ExerciseID = &exerciseID
				}
			}
		}
		events = append(events, event)
	}
	return events, nil
}

// groupSessionAdvances agrupa las filas por emisión y cuenta los estados. Las
// filas llegan ordenadas por emisión y nombre, así que el resultado conserva
// ese orden y los destinatarios de cada avance quedan alfabéticos.
func groupSessionAdvances(rows []advanceRecipientRow) []domain.AdvanceWithRecipients {
	out := make([]domain.AdvanceWithRecipients, 0, 4)
	index := make(map[string]int, 4)
	for _, row := range rows {
		position, ok := index[row.AdvanceID]
		if !ok {
			out = append(out, domain.AdvanceWithRecipients{
				ID:            row.AdvanceID,
				ExerciseID:    row.ExerciseID,
				ExerciseTitle: row.ExerciseTitle,
				Scope:         row.Scope,
				FileCount:     row.FileCount,
				CreatedAt:     row.CreatedAt,
				Recipients:    make([]domain.AdvanceRecipientSummary, 0, 8),
			})
			position = len(out) - 1
			index[row.AdvanceID] = position
		}

		advance := &out[position]
		advance.Recipients = append(advance.Recipients, domain.AdvanceRecipientSummary{
			StudentID: row.StudentID,
			Name:      row.Name,
			Email:     row.Email,
			Status:    row.Status,
			DecidedAt: row.DecidedAt,
		})
		advance.Counts.Total++
		switch row.Status {
		case domain.AdvanceRecipientApplied:
			advance.Counts.Applied++
		case domain.AdvanceRecipientDismissed:
			advance.Counts.Dismissed++
		case domain.AdvanceRecipientPending:
			advance.Counts.Pending++
		case domain.AdvanceRecipientFailed:
			advance.Counts.Failed++
		case domain.AdvanceRecipientExpired:
			advance.Counts.Expired++
		}
	}
	return out
}

// SetFrozen fija o limpia frozen_at. Congelar dos veces conserva la primera hora.
func (r *PostgresClassAdvanceRepository) SetFrozen(ctx context.Context, tenantID, teacherID, sessionID string, frozen bool) (*domain.ClassSession, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE class_sessions cs
		SET frozen_at = CASE WHEN $4 THEN COALESCE(cs.frozen_at, NOW()) ELSE NULL END,
		    updated_at = NOW()
		FROM subjects s
		WHERE s.id = cs.subject_id
		  AND cs.id = $1::uuid AND cs.tenant_id = $2
		  AND ($3 = '' OR s.teacher_id = $3::uuid)
	`, sessionID, tenantID, teacherID, frozen)
	if err != nil {
		return nil, fmt.Errorf("failed to update class freeze: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, domain.ErrSessionNotFound
	}
	return r.sessions.GetSession(ctx, tenantID, teacherID, sessionID)
}

// SetGrace fija grace_until solo en una sesión cerrada del docente y alinea el
// estado de los destinatarios: al reabrir la ventana, los vencidos vuelven a
// pendientes; al cerrarla, los pendientes pasan a vencidos.
func (r *PostgresClassAdvanceRepository) SetGrace(ctx context.Context, tenantID, teacherID, sessionID string, graceUntil time.Time) (*domain.ClassSession, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin advance window transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		UPDATE class_sessions cs
		SET grace_until = $4, updated_at = NOW()
		FROM subjects s
		WHERE s.id = cs.subject_id
		  AND cs.id = $1::uuid AND cs.tenant_id = $2 AND cs.status = 'closed'
		  AND ($3 = '' OR s.teacher_id = $3::uuid)
	`, sessionID, tenantID, teacherID, graceUntil)
	if err != nil {
		return nil, fmt.Errorf("failed to update advance grace window: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, domain.ErrSessionNotFound
	}

	if graceUntil.After(time.Now()) {
		if _, err := tx.ExecContext(ctx, `
			UPDATE class_advance_recipients r
			SET status = 'pending', decided_at = NULL
			FROM class_advances ca
			WHERE r.advance_id = ca.id
			  AND ca.session_id = $1::uuid
			  AND r.tenant_id = $2
			  AND r.status = 'expired'
		`, sessionID, tenantID); err != nil {
			return nil, fmt.Errorf("failed to reopen expired advance recipients: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
			UPDATE class_advance_recipients r
			SET status = 'expired', decided_at = COALESCE(r.decided_at, NOW())
			FROM class_advances ca
			WHERE r.advance_id = ca.id
			  AND ca.session_id = $1::uuid
			  AND r.tenant_id = $2
			  AND r.status = 'pending'
		`, sessionID, tenantID); err != nil {
			return nil, fmt.Errorf("failed to close pending advance recipients: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit advance window: %w", err)
	}
	return r.sessions.GetSession(ctx, tenantID, teacherID, sessionID)
}

// ExpiredAdvanceSnapshots devuelve los avances cuyo objeto ya puede borrarse:
// el aula está cerrada, el cierre quedó antes del corte y la ventana ya venció.
// La consulta no filtra por tenant porque el barrido recorre la instalación; la
// clave del objeto lleva el tenant y el borrado se hace contra la fila.
func (r *PostgresClassAdvanceRepository) ExpiredAdvanceSnapshots(ctx context.Context, cutoff time.Time, limit int) ([]domain.ClassAdvance, error) {
	if limit <= 0 {
		limit = domain.AdvanceSnapshotPurgeBatch
	}
	advances := make([]domain.ClassAdvance, 0, limit)
	if err := r.db.SelectContext(ctx, &advances, `
		SELECT ca.id, ca.tenant_id, ca.snapshot_key
		FROM class_advances ca
		JOIN class_sessions cs ON cs.id = ca.session_id
		WHERE ca.snapshot_purged_at IS NULL
		  AND ca.snapshot_key <> ''
		  AND cs.status = 'closed'
		  AND COALESCE(cs.closes_at, cs.ends_at) <= $1
		  AND (cs.grace_until IS NULL OR cs.grace_until <= NOW())
		ORDER BY COALESCE(cs.closes_at, cs.ends_at), ca.id
		LIMIT $2
	`, cutoff, limit); err != nil {
		return nil, fmt.Errorf("failed to load expired advance snapshots: %w", err)
	}
	return advances, nil
}

// MarkSnapshotPurged deja constancia de que el objeto se borró. Si la fila ya
// estaba marcada no cambia nada; un borrado repetido del mismo objeto es seguro.
func (r *PostgresClassAdvanceRepository) MarkSnapshotPurged(ctx context.Context, tenantID, advanceID string, purgedAt time.Time) error {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE class_advances
		SET snapshot_purged_at = $3
		WHERE id = $1::uuid AND tenant_id = $2 AND snapshot_purged_at IS NULL
	`, advanceID, tenantID, purgedAt); err != nil {
		return fmt.Errorf("failed to mark advance snapshot purged: %w", err)
	}
	return nil
}

// AdvanceSnapshotKeys devuelve todas las claves que todavía están referenciadas
// por una fila. El barrido de huérfanos las excluye, incluso si el objeto ya se
// borró: la fila conserva el registro de lo emitido.
func (r *PostgresClassAdvanceRepository) AdvanceSnapshotKeys(ctx context.Context) ([]string, error) {
	keys := make([]string, 0, 32)
	if err := r.db.SelectContext(ctx, &keys, `
		SELECT snapshot_key FROM class_advances WHERE snapshot_key <> ''
	`); err != nil {
		return nil, fmt.Errorf("failed to load advance snapshot keys: %w", err)
	}
	return keys, nil
}
