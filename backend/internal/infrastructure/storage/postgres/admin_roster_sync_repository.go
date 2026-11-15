package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"solv-backend/internal/core/domain"

	"github.com/jmoiron/sqlx"
)

// SyncSubjectRoster resuelve las personas de la nómina por identificador
// externo o correo, las matricula en la materia y, si se pide reconciliación,
// quita a quienes ya no figuran en el origen.
//
// Todo ocurre en una transacción: una nómina aplicada a medias deja la materia
// en un estado que nadie pidió, y reconcilia contra una lista que ya no
// corresponde.
func (r *PostgresAdminGovernanceRepository) SyncSubjectRoster(
	ctx context.Context,
	tenantID, subjectID string,
	req domain.RosterSyncRequest,
) (*domain.RosterSyncOutcome, error) {
	outcome := &domain.RosterSyncOutcome{}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error abriendo la transacción de nómina: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Sin esta validación, reconciliar contra una materia de otra institución
	// borraría matrículas ajenas.
	var subjectExists bool
	if err := tx.GetContext(ctx, &subjectExists,
		`SELECT EXISTS (SELECT 1 FROM subjects WHERE id = $1 AND tenant_id = $2)`,
		subjectID, tenantID); err != nil {
		return nil, fmt.Errorf("error validando la materia: %w", err)
	}
	if !subjectExists {
		return nil, errors.New("la materia no existe en la institución")
	}

	source := req.SourceSystem
	if source == "" {
		source = "roster"
	}

	studentIDs := make([]string, 0, len(req.Entries))
	for _, entry := range req.Entries {
		studentID, matchedExternal, created, err := upsertRosterStudent(ctx, tx, tenantID, source, entry)
		if err != nil {
			return nil, err
		}
		if studentID == "" {
			outcome.SkippedNotFound++
			continue
		}
		if matchedExternal {
			outcome.MatchedByExternalID++
		}
		if created {
			outcome.CreatedStudents++
		}
		studentIDs = append(studentIDs, studentID)

		// xmax = 0 distingue la fila insertada de la que ya existía: el panel
		// informa matrículas nuevas, no filas tocadas.
		var inserted bool
		if err := tx.GetContext(ctx, &inserted, `
			INSERT INTO enrollments (id, tenant_id, student_id, subject_id, enrolled_at, source_system, external_id, last_synced_at)
			VALUES (gen_random_uuid(), $1, $2, $3, NOW(), $4, NULLIF($5, ''), NOW())
			ON CONFLICT (tenant_id, student_id, subject_id) DO UPDATE
			SET source_system   = EXCLUDED.source_system,
			    external_id     = COALESCE(EXCLUDED.external_id, enrollments.external_id),
			    last_synced_at  = NOW()
			RETURNING (xmax = 0)`,
			tenantID, studentID, subjectID, source, entry.ExternalID); err != nil {
			return nil, fmt.Errorf("error matriculando %s: %w", entry.Email, err)
		}
		if inserted {
			outcome.EnrolledStudents++
		}
	}

	if req.Reconcile && len(studentIDs) > 0 {
		res, err := tx.ExecContext(ctx, `
			DELETE FROM enrollments
			WHERE tenant_id = $1
			  AND source_system = $2
			  AND subject_id = $3
			  AND NOT (student_id = ANY($4::uuid[]))`,
			tenantID, source, subjectID, arrayTextLiteral(studentIDs))
		if err != nil {
			return nil, fmt.Errorf("error reconciliando las bajas de la nómina: %w", err)
		}
		removed, _ := res.RowsAffected()
		outcome.RemovedStudents = int(removed)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error confirmando la nómina: %w", err)
	}

	return outcome, nil
}

// upsertRosterStudent devuelve el id de la cuenta que corresponde a la fila,
// si la resolvió por identificador externo, y si la creó. Un id vacío con error
// nulo significa que la fila se descartó: el correo pertenece a una cuenta que
// no es de estudiante.
func upsertRosterStudent(
	ctx context.Context,
	tx *sqlx.Tx,
	tenantID, source string,
	entry domain.RosterEntry,
) (id string, matchedExternal bool, created bool, err error) {
	if entry.ExternalID != "" {
		err = tx.GetContext(ctx, &id,
			`SELECT id FROM users WHERE tenant_id = $1 AND external_id = $2`,
			tenantID, entry.ExternalID)
		switch {
		case err == nil:
			_, uerr := tx.ExecContext(ctx, `
				UPDATE users
				SET last_synced_at = NOW(),
				    source_system = COALESCE(NULLIF(source_system, ''), $3),
				    updated_at = NOW()
				WHERE id = $1 AND tenant_id = $2`,
				id, tenantID, source)
			return id, true, false, uerr
		case !errors.Is(err, sql.ErrNoRows):
			return "", false, false, fmt.Errorf("error buscando el identificador externo %q: %w", entry.ExternalID, err)
		}
	}

	// El correo es único por cuenta en toda la instalación. Si ya existe una
	// cuenta con ese correo y no es estudiante, la fila se descarta: cambiarle
	// el rol a un docente porque aparece en una planilla sería peor que
	// saltarse la fila, y el conteo lo reporta.
	var existing struct {
		ID   string `db:"id"`
		Role string `db:"role"`
	}
	err = tx.GetContext(ctx, &existing,
		`SELECT id, role FROM users WHERE tenant_id = $1 AND LOWER(email) = LOWER($2)`,
		tenantID, entry.Email)
	switch {
	case err == nil:
		if existing.Role != "student" {
			return "", false, false, nil
		}
		_, uerr := tx.ExecContext(ctx, `
			UPDATE users
			SET external_id    = COALESCE(NULLIF(external_id, ''), NULLIF($3, '')),
			    source_system  = COALESCE(NULLIF(source_system, ''), $4),
			    last_synced_at = NOW(),
			    updated_at     = NOW()
			WHERE id = $1 AND tenant_id = $2`,
			existing.ID, tenantID, entry.ExternalID, source)
		return existing.ID, entry.ExternalID != "", false, uerr
	case !errors.Is(err, sql.ErrNoRows):
		return "", false, false, fmt.Errorf("error buscando el correo %q: %w", entry.Email, err)
	}

	if err := tx.GetContext(ctx, &id, `
		INSERT INTO users (id, tenant_id, email, first_name, last_name, role, status, origin, external_id, source_system, last_synced_at, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, 'student', 'active', 'manual', NULLIF($5, ''), NULLIF($6, ''), NOW(), NOW())
		RETURNING id`,
		tenantID, entry.Email, entry.FirstName, entry.LastName, entry.ExternalID, source); err != nil {
		return "", false, false, fmt.Errorf("error creando la cuenta %q: %w", entry.Email, err)
	}

	return id, false, true, nil
}
