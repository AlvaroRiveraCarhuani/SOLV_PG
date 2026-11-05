package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"solv-backend/internal/core/domain"
)

// PostgresClassSessionRepository guarda el aula: la sesión, sus participantes y
// el enlace de los entornos que nacieron dentro de ella.
type PostgresClassSessionRepository struct {
	db *sqlx.DB
}

func NewPostgresClassSessionRepository(db *sqlx.DB) *PostgresClassSessionRepository {
	return &PostgresClassSessionRepository{db: db}
}

// sessionColumns es el SELECT común de una sesión con la materia, la práctica y
// los dos conteos que la vista muestra sin recalcular nada.
const sessionColumns = `
	cs.id,
	cs.tenant_id,
	cs.subject_id,
	s.code AS subject_code,
	s.name AS subject_name,
	cs.exercise_id,
	ex.title AS exercise_title,
	COALESCE(ex.purpose, 'class') AS purpose,
	cs.status,
	cs.opens_at,
	cs.ends_at,
	cs.closes_at,
	cs.frozen_at,
	cs.grace_until,
	COALESCE(cs.created_by::text, '') AS created_by,
	(SELECT COUNT(*) FROM enrollments en
	   WHERE en.tenant_id = cs.tenant_id AND en.subject_id = cs.subject_id) AS enrolled,
	(SELECT COUNT(*) FROM class_session_participants p
	   WHERE p.session_id = cs.id
	     AND p.last_seen_at >= NOW() - INTERVAL '120 seconds') AS present
`

// expireSessions cierra las aulas vencidas de la institución y sella la salida de
// sus participantes. Se ejecuta antes de leer o abrir: el estado "abierta" queda
// atado a su hora de fin sin depender de un proceso periódico aparte, y una
// sesión olvidada deja de bloquear la materia en la siguiente interacción.
//
// Es una sola sentencia: la CTE cierra las sesiones y el UPDATE exterior marca
// el `left_at` de quienes estaban dentro.
func (r *PostgresClassSessionRepository) expireSessions(ctx context.Context, tenantID string) error {
	if _, err := r.db.ExecContext(ctx, `
		WITH expired AS (
			UPDATE class_sessions
			SET status = 'closed',
			    closes_at = ends_at,
			    grace_until = COALESCE(grace_until, ends_at + make_interval(mins => $2)),
			    updated_at = NOW()
			WHERE tenant_id = $1
			  AND status = 'open'
			  AND ends_at <= NOW()
			RETURNING id
		)
		UPDATE class_session_participants p
		SET left_at = COALESCE(p.left_at, NOW())
		FROM expired e
		WHERE p.session_id = e.id AND p.left_at IS NULL
	`, tenantID, domain.DefaultAdvanceGraceMinutes); err != nil {
		return fmt.Errorf("failed to expire class sessions: %w", err)
	}
	return nil
}

// OpenSession abre el aula de una materia del docente por la duración pedida.
// Un segundo intento sobre la misma materia choca con el índice parcial y se
// traduce a un conflicto de dominio, no a un error de base de datos.
func (r *PostgresClassSessionRepository) OpenSession(ctx context.Context, tenantID, teacherID, subjectID string, exerciseID *string, durationMinutes int) (*domain.ClassSession, error) {
	if err := r.expireSessions(ctx, tenantID); err != nil {
		return nil, err
	}

	var subject struct {
		ID string `db:"id"`
	}
	err := r.db.GetContext(ctx, &subject, `
		SELECT s.id
		FROM subjects s
		WHERE s.id = $1::uuid AND s.tenant_id = $2
		  AND ($3 = '' OR s.teacher_id = $3::uuid)
	`, subjectID, tenantID, teacherID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to load subject for session: %w", err)
	}

	exercise := ""
	if exerciseID != nil {
		exercise = strings.TrimSpace(*exerciseID)
	}
	if exercise != "" {
		var belongs bool
		if err := r.db.GetContext(ctx, &belongs, `
			SELECT EXISTS (
				SELECT 1 FROM exercises e
				WHERE e.id = $1::uuid AND e.subject_id = $2::uuid AND e.tenant_id = $3
			)
		`, exercise, subjectID, tenantID); err != nil {
			return nil, fmt.Errorf("failed to validate exercise for session: %w", err)
		}
		if !belongs {
			// La práctica elegida no es de esta materia: el aula no se abre con
			// una consigna que el alumno no puede ver.
			return nil, domain.ErrNotFound
		}
	}

	var sessionID string
	err = r.db.GetContext(ctx, &sessionID, `
		INSERT INTO class_sessions (tenant_id, subject_id, exercise_id, status, opens_at, ends_at, created_by)
		VALUES ($1, $2::uuid, NULLIF($3, '')::uuid, 'open', NOW(),
		        NOW() + make_interval(mins => $5), NULLIF($4, '')::uuid)
		RETURNING id
	`, tenantID, subjectID, exercise, teacherID, durationMinutes)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, domain.ErrSessionAlreadyOpen
		}
		return nil, fmt.Errorf("failed to open class session: %w", err)
	}

	return r.GetSession(ctx, tenantID, teacherID, sessionID)
}

// CloseSession cierra el aula. Es idempotente: si ya estaba cerrada devuelve la
// misma sesión, sin error, porque el docente pudo haber pulsado dos veces.
func (r *PostgresClassSessionRepository) CloseSession(ctx context.Context, tenantID, teacherID, sessionID string) (*domain.ClassSession, error) {
	var closedID string
	err := r.db.GetContext(ctx, &closedID, `
		UPDATE class_sessions cs
		SET status = 'closed',
		    closes_at = COALESCE(cs.closes_at, NOW()),
		    grace_until = COALESCE(cs.grace_until, COALESCE(cs.closes_at, NOW()) + make_interval(mins => $4)),
		    updated_at = NOW()
		FROM subjects s
		WHERE cs.id = $1::uuid AND cs.tenant_id = $2
		  AND s.id = cs.subject_id
		  AND ($3 = '' OR s.teacher_id = $3::uuid)
		  AND cs.status = 'open'
		RETURNING cs.id
	`, sessionID, tenantID, teacherID, domain.DefaultAdvanceGraceMinutes)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to close class session: %w", err)
	}

	if closedID != "" {
		if _, err := r.db.ExecContext(ctx, `
			UPDATE class_session_participants
			SET left_at = COALESCE(left_at, NOW())
			WHERE session_id = $1::uuid AND left_at IS NULL
		`, sessionID); err != nil {
			return nil, fmt.Errorf("failed to stamp session departure: %w", err)
		}
	}

	// Cerrar dos veces devuelve el mismo estado; una sesión ajena o inexistente
	// devuelve ErrSessionNotFound desde aquí.
	return r.GetSession(ctx, tenantID, teacherID, sessionID)
}

// GetOpenSessionForTeacher devuelve el aula abierta del docente, o nil cuando no
// hay ninguna. Es la fila de aula del panel y el punto de entrada de la pantalla:
// una aula vencida deja de aparecer aquí, porque se cierra antes de consultar.
func (r *PostgresClassSessionRepository) GetOpenSessionForTeacher(ctx context.Context, tenantID, teacherID string) (*domain.ClassSession, error) {
	if err := r.expireSessions(ctx, tenantID); err != nil {
		return nil, err
	}

	query := `
		SELECT ` + sessionColumns + `
		FROM class_sessions cs
		JOIN subjects s ON s.id = cs.subject_id
		LEFT JOIN exercises ex ON ex.id = cs.exercise_id
		WHERE cs.tenant_id = $1
		  AND cs.status = 'open'
		  AND ($2 = '' OR s.teacher_id = $2::uuid)
		ORDER BY cs.opens_at DESC
		LIMIT 1
	`
	session := &domain.ClassSession{}
	if err := r.db.GetContext(ctx, session, query, tenantID, teacherID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load open class session: %w", err)
	}
	return session, nil
}

// GetLastClosedSessionForTeacher devuelve el aula cerrada más reciente del
// docente, o nil si nunca tuvo una. Es la lectura que permite a la pantalla
// recuperar el historial y la ventana tras recargar sin aula abierta.
func (r *PostgresClassSessionRepository) GetLastClosedSessionForTeacher(ctx context.Context, tenantID, teacherID string) (*domain.ClassSession, error) {
	if err := r.expireSessions(ctx, tenantID); err != nil {
		return nil, err
	}

	query := `
		SELECT ` + sessionColumns + `
		FROM class_sessions cs
		JOIN subjects s ON s.id = cs.subject_id
		LEFT JOIN exercises ex ON ex.id = cs.exercise_id
		WHERE cs.tenant_id = $1
		  AND cs.status = 'closed'
		  AND ($2 = '' OR s.teacher_id = $2::uuid)
		ORDER BY COALESCE(cs.closes_at, cs.ends_at) DESC
		LIMIT 1
	`
	session := &domain.ClassSession{}
	if err := r.db.GetContext(ctx, session, query, tenantID, teacherID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load last closed class session: %w", err)
	}
	return session, nil
}

// GetSession devuelve una sesión del docente por identificador, esté abierta o
// cerrada. Una sesión vencida se devuelve cerrada: la pantalla deja el modo en
// vivo en el siguiente refresco, sin esperar a que el docente cierre.
func (r *PostgresClassSessionRepository) GetSession(ctx context.Context, tenantID, teacherID, sessionID string) (*domain.ClassSession, error) {
	if err := r.expireSessions(ctx, tenantID); err != nil {
		return nil, err
	}

	query := `
		SELECT ` + sessionColumns + `
		FROM class_sessions cs
		JOIN subjects s ON s.id = cs.subject_id
		LEFT JOIN exercises ex ON ex.id = cs.exercise_id
		WHERE cs.id = $1::uuid AND cs.tenant_id = $2
		  AND ($3 = '' OR s.teacher_id = $3::uuid)
	`
	session := &domain.ClassSession{}
	if err := r.db.GetContext(ctx, session, query, sessionID, tenantID, teacherID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to load class session: %w", err)
	}
	return session, nil
}

// Board resuelve el aula en dos consultas fijas —la sesión y la lista de
// alumnos— sin recorrer alumnos en un bucle. El estado de cada fila se calcula
// aquí, en una sola pasada sobre las inscripciones de la materia.
func (r *PostgresClassSessionRepository) Board(ctx context.Context, tenantID, teacherID, sessionID string) (*domain.ClassSessionBoard, error) {
	session, err := r.GetSession(ctx, tenantID, teacherID, sessionID)
	if err != nil {
		return nil, err
	}

	rowsQuery := `
		WITH s AS (
			SELECT id, tenant_id, subject_id, exercise_id, opens_at
			FROM class_sessions
			WHERE id = $1::uuid AND tenant_id = $2
		),
		latest AS (
			SELECT DISTINCT ON (sub.student_id)
				sub.student_id, COALESCE(sub.verdict, '') AS verdict, sub.review_status, sub.submitted_at
			FROM submissions sub
			JOIN s ON sub.exercise_id = s.exercise_id AND sub.tenant_id = s.tenant_id
			WHERE sub.submitted_at >= s.opens_at
			ORDER BY sub.student_id, sub.submitted_at DESC
		),
		attempts AS (
			SELECT sub.student_id, COUNT(*) AS total
			FROM submissions sub
			JOIN s ON sub.exercise_id = s.exercise_id AND sub.tenant_id = s.tenant_id
			WHERE sub.submitted_at >= GREATEST(s.opens_at, NOW() - INTERVAL '10 minutes')
			GROUP BY sub.student_id
		),
		env AS (
			SELECT DISTINCT ON (w.student_id) w.student_id, w.id, w.status
			FROM environments w
			JOIN s ON w.subject_id = s.subject_id AND w.tenant_id = s.tenant_id
			WHERE w.status IN ('running', 'active', 'pending')
			ORDER BY w.student_id, w.last_heartbeat_at DESC NULLS LAST
		),
		board AS (
			SELECT
				en.student_id,
				COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email) AS name,
				u.email,
				p.joined_at,
				p.last_seen_at,
				COALESCE(p.last_seen_at >= NOW() - INTERVAL '120 seconds', FALSE) AS present,
				env.id AS environment_id,
				env.status AS environment_status,
				l.submitted_at,
				l.verdict,
				l.review_status,
				COALESCE(a.total, 0) AS attempts,
				-- El veredicto manda sobre la presencia: una entrega hecha dentro de la
				-- clase prueba que el alumno estuvo ahí, aunque su último latido se haya
				-- perdido. La presencia en vivo (columna present) sigue siendo el latido.
				CASE
					WHEN l.verdict IN ('RE', 'TLE', 'AST_BLOCKED')
					     AND l.submitted_at >= NOW() - INTERVAL '10 minutes' THEN 'trabado'
					WHEN l.submitted_at IS NOT NULL THEN 'entregado'
					WHEN p.last_seen_at IS NULL THEN 'sin_empezar'
					WHEN p.last_seen_at < NOW() - INTERVAL '120 seconds' THEN 'ausente'
					ELSE 'trabajando'
				END AS state
			FROM enrollments en
			JOIN s ON en.subject_id = s.subject_id AND en.tenant_id = s.tenant_id
			JOIN users u ON u.id = en.student_id
			LEFT JOIN class_session_participants p
				ON p.session_id = s.id AND p.student_id = en.student_id
			LEFT JOIN latest l ON l.student_id = en.student_id
			LEFT JOIN attempts a ON a.student_id = en.student_id
			LEFT JOIN env ON env.student_id = en.student_id
		)
		SELECT *
		FROM board
		ORDER BY
			CASE state
				WHEN 'trabado' THEN 0
				WHEN 'sin_empezar' THEN 1
				WHEN 'ausente' THEN 2
				WHEN 'entregado' THEN 3
				ELSE 4
			END,
			name
	`

	type boardRow struct {
		StudentID       string         `db:"student_id"`
		Name            string         `db:"name"`
		Email           string         `db:"email"`
		JoinedAt        sql.NullTime   `db:"joined_at"`
		LastSeenAt      sql.NullTime   `db:"last_seen_at"`
		Present         bool           `db:"present"`
		EnvironmentID     sql.NullString `db:"environment_id"`
		PracticeEnvironmentStatus sql.NullString `db:"environment_status"`
		SubmittedAt     sql.NullTime   `db:"submitted_at"`
		Verdict         sql.NullString `db:"verdict"`
		ReviewStatus    sql.NullString `db:"review_status"`
		Attempts        int            `db:"attempts"`
		State           string         `db:"state"`
	}

	var rows []boardRow
	if err := r.db.SelectContext(ctx, &rows, rowsQuery, sessionID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to build class session board: %w", err)
	}

	board := &domain.ClassSessionBoard{
		Session:  *session,
		Students: make([]domain.ClassSessionStudent, 0, len(rows)),
	}
	for _, row := range rows {
		student := domain.ClassSessionStudent{
			StudentID: row.StudentID,
			Name:      row.Name,
			Email:     row.Email,
			State:     row.State,
			Present:   row.Present,
			Attempts:  row.Attempts,
		}
		if row.JoinedAt.Valid {
			joined := row.JoinedAt.Time
			student.JoinedAt = &joined
		}
		if row.LastSeenAt.Valid {
			seen := row.LastSeenAt.Time
			student.LastSeenAt = &seen
		}
		if row.EnvironmentID.Valid {
			id := row.EnvironmentID.String
			student.EnvironmentID = &id
		}
		if row.PracticeEnvironmentStatus.Valid {
			status := row.PracticeEnvironmentStatus.String
			student.PracticeEnvironmentStatus = &status
		}
		if row.SubmittedAt.Valid {
			submitted := row.SubmittedAt.Time
			student.SubmittedAt = &submitted
		}
		if row.Verdict.Valid {
			verdict := row.Verdict.String
			student.Verdict = &verdict
		}
		if row.ReviewStatus.Valid {
			review := row.ReviewStatus.String
			student.ReviewStatus = &review
		}
		board.Students = append(board.Students, student)

		board.Summary.Enrolled++
		if student.Present {
			board.Summary.Present++
		}
		switch student.State {
		case domain.ParticipantNotStarted:
			board.Summary.NotStarted++
		case domain.ParticipantWorking:
			board.Summary.Working++
		case domain.ParticipantSubmitted:
			board.Summary.Submitted++
		case domain.ParticipantStuck:
			board.Summary.Stuck++
		case domain.ParticipantAbsent:
			board.Summary.Absent++
		}

		// "Sin entregar" cuenta alumnos sin entrega, no alumnos cuyo estado no es
		// "entregado": un trabado ya entregó y no puede aparecer como faltante de
		// entrega.
		if student.SubmittedAt == nil {
			board.Summary.WithoutSubmission++
		}
	}

	// Los conteos firmados por la consulta de la sesión y por la lista deben
	// coincidir: si no, la excepción se ve, no se esconde.
	board.Session.Present = board.Summary.Present
	board.Session.Enrolled = board.Summary.Enrolled
	return board, nil
}

// OpenSessionsForStudent lista las aulas abiertas de las materias del alumno.
// El cliente lo sondea para saber si tiene que latir.
func (r *PostgresClassSessionRepository) OpenSessionsForStudent(ctx context.Context, tenantID, studentID string) ([]domain.ClassSession, error) {
	if err := r.expireSessions(ctx, tenantID); err != nil {
		return nil, err
	}

	query := `
		SELECT
			cs.id,
			cs.tenant_id,
			cs.subject_id,
			s.code AS subject_code,
			s.name AS subject_name,
			cs.exercise_id,
			ex.title AS exercise_title,
			cs.status,
			cs.opens_at,
			cs.ends_at,
			cs.closes_at,
			(SELECT COUNT(*)
			   FROM enrollments e2
			  WHERE e2.tenant_id = cs.tenant_id AND e2.subject_id = cs.subject_id) AS enrolled,
			(SELECT COUNT(*)
			   FROM class_session_participants p
			  WHERE p.session_id = cs.id
			    AND p.tenant_id = cs.tenant_id
			    AND p.last_seen_at >= NOW() - INTERVAL '120 seconds') AS present
		FROM class_sessions cs
		JOIN subjects s ON s.id = cs.subject_id
		JOIN enrollments en
			ON en.subject_id = cs.subject_id
			AND en.student_id = $2::uuid
			AND en.tenant_id = cs.tenant_id
		LEFT JOIN exercises ex ON ex.id = cs.exercise_id
		WHERE cs.tenant_id = $1 AND cs.status = 'open'
		ORDER BY cs.opens_at DESC
	`
	sessions := make([]domain.ClassSession, 0)
	if err := r.db.SelectContext(ctx, &sessions, query, tenantID, studentID); err != nil {
		return nil, fmt.Errorf("failed to load student open sessions: %w", err)
	}
	return sessions, nil
}

// TouchParticipant registra o refresca la presencia del alumno. Un aula cerrada
// no admite presencia (ErrSessionNotOpen) y una materia que el alumno no cursa
// tampoco (ErrStudentNotEnrolled): son dos errores distintos porque el cliente
// debe dejar de latir en el primero y no debería haber llegado al segundo.
func (r *PostgresClassSessionRepository) TouchParticipant(ctx context.Context, tenantID, studentID, sessionID string) error {
	// Un aula vencida no admite presencia: se cierra antes de leer su estado.
	if err := r.expireSessions(ctx, tenantID); err != nil {
		return err
	}

	var session struct {
		Status    string `db:"status"`
		SubjectID string `db:"subject_id"`
	}
	if err := r.db.GetContext(ctx, &session, `
		SELECT status, subject_id
		FROM class_sessions
		WHERE id = $1::uuid AND tenant_id = $2
	`, sessionID, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrSessionNotFound
		}
		return fmt.Errorf("failed to load session for heartbeat: %w", err)
	}
	if session.Status != domain.ClassSessionStatusOpen {
		return domain.ErrSessionNotOpen
	}

	var enrolled bool
	if err := r.db.GetContext(ctx, &enrolled, `
		SELECT EXISTS (
			SELECT 1 FROM enrollments
			WHERE tenant_id = $1 AND subject_id = $2::uuid AND student_id = $3::uuid
		)
	`, tenantID, session.SubjectID, studentID); err != nil {
		return fmt.Errorf("failed to validate student enrollment: %w", err)
	}
	if !enrolled {
		return domain.ErrStudentNotEnrolled
	}

	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO class_session_participants (session_id, student_id, tenant_id, joined_at, last_seen_at)
		VALUES ($1::uuid, $2::uuid, $3, NOW(), NOW())
		ON CONFLICT (session_id, student_id)
		DO UPDATE SET last_seen_at = NOW(), left_at = NULL
	`, sessionID, studentID, tenantID); err != nil {
		return fmt.Errorf("failed to record session presence: %w", err)
	}
	return nil
}

// AttachPracticeEnvironmentToOpenSession deja constancia de en qué aula nació el entorno y
// marca la presencia del alumno. Si la materia no tiene aula abierta no hace
// nada: el trabajo fuera de clase sigue siendo trabajo, sin sesión.
func (r *PostgresClassSessionRepository) AttachPracticeEnvironmentToOpenSession(ctx context.Context, tenantID, studentID, subjectID, environmentID string) error {
	// Un entorno que nace después del fin de clase no pertenece a esa sesión.
	if err := r.expireSessions(ctx, tenantID); err != nil {
		return err
	}

	var sessionID string
	err := r.db.GetContext(ctx, &sessionID, `
		SELECT cs.id
		FROM class_sessions cs
		WHERE cs.tenant_id = $1 AND cs.subject_id = $2::uuid AND cs.status = 'open'
		LIMIT 1
	`, tenantID, subjectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to resolve open session for environment: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, `
		UPDATE environments
		SET session_id = $1::uuid
		WHERE id = $2::uuid AND tenant_id = $3 AND session_id IS NULL
	`, sessionID, environmentID, tenantID); err != nil {
		return fmt.Errorf("failed to link environment to session: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO class_session_participants (session_id, student_id, tenant_id, joined_at, last_seen_at, environment_id)
		VALUES ($1::uuid, $2::uuid, $3, NOW(), NOW(), $4::uuid)
		ON CONFLICT (session_id, student_id)
		DO UPDATE SET last_seen_at = NOW(), left_at = NULL, environment_id = EXCLUDED.environment_id
	`, sessionID, studentID, tenantID, environmentID); err != nil {
		return fmt.Errorf("failed to record session presence for environment: %w", err)
	}
	return nil
}
