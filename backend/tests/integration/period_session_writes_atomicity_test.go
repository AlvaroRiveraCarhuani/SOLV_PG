package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// Activar un periodo, archivar los vencidos, cerrar un aula, vincular un entorno
// y dar de alta una materia tocan más de una tabla. Estas pruebas fuerzan un
// fallo en la segunda escritura y comprueban contra PostgreSQL real que la
// primera no quedó guardada.
func TestAcademicAndSessionWritesAreAtomic(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		skipOrFailInfra(t, "PostgreSQL no disponible para las pruebas de escrituras académicas: %v", err)
		return
	}
	defer func() { _ = db.GetDB().Close() }()

	ctx := context.Background()
	periods := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	sessions := postgres.NewPostgresClassSessionRepository(db.GetDB())
	subjects := postgres.NewPostgresSubjectRepository(db.GetDB())

	tenantID := uuid.NewString()
	suffix := tenantID[:8]
	teacherID := uuid.NewString()
	studentID := uuid.NewString()
	subjectID := uuid.NewString()
	periodActiveID := uuid.NewString()
	periodInactiveID := uuid.NewString()
	periodExpiredID := uuid.NewString()

	if _, errTenant := db.GetDB().Exec(`INSERT INTO tenants (id, name, slug) VALUES ($1, 'Escrituras académicas', $2)`,
		tenantID, "it-academic-"+suffix); errTenant != nil {
		t.Fatalf("no se pudo sembrar el tenant de la prueba: %v", errTenant)
	}
	if _, errUser := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id) VALUES
		($1, 'Docente', 'Atómico', $2, 'teacher', $4),
		($3, 'Alumno', 'Atómico', $5, 'student', $4)
	`, teacherID, "teacher_atomic_"+suffix+"@uab.edu.bo", studentID, tenantID, "student_atomic_"+suffix+"@uab.edu.bo"); errUser != nil {
		t.Fatalf("no se pudieron sembrar los usuarios de la prueba: %v", errUser)
	}
	if _, errSubject := db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Aula atómica', $3, $4)
	`, subjectID, tenantID, "AULA-"+suffix, teacherID); errSubject != nil {
		t.Fatalf("no se pudo sembrar la materia de la prueba: %v", errSubject)
	}
	if _, errPeriods := db.GetDB().Exec(`
		INSERT INTO academic_periods (id, tenant_id, name, code, start_date, end_date, is_active) VALUES
		($1, $4, 'Periodo activo', $5, CURRENT_DATE - 30, CURRENT_DATE + 30, true),
		($2, $4, 'Periodo inactivo', $6, CURRENT_DATE - 30, CURRENT_DATE + 30, false),
		($3, $4, 'Periodo vencido', $7, CURRENT_DATE - 90, CURRENT_DATE - 1, false)
	`, periodActiveID, periodInactiveID, periodExpiredID, tenantID,
		"ACT-"+suffix, "INA-"+suffix, "VEN-"+suffix); errPeriods != nil {
		t.Fatalf("no se pudieron sembrar los periodos de la prueba: %v", errPeriods)
	}
	if _, errLink := db.GetDB().Exec(`
		UPDATE subjects SET academic_period_id = $1 WHERE id = $2
	`, periodExpiredID, subjectID); errLink != nil {
		t.Fatalf("no se pudo vincular la materia al periodo vencido: %v", errLink)
	}
	defer func() {
		_, _ = db.GetDB().Exec(`DELETE FROM environments WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM class_sessions WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM subject_catalog WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM academic_periods WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE tenant_id = $1`, tenantID)
		_, _ = db.GetDB().Exec(`DELETE FROM tenants WHERE id = $1`, tenantID)
	}()

	t.Run("activar un periodo no deja la institución sin periodo activo", func(t *testing.T) {
		trigger := "solv_it_block_period_update"
		installFailingTrigger(t, db.GetDB(), trigger, "academic_periods", "BEFORE", "UPDATE",
			"NEW.id = '"+periodInactiveID+"'::uuid")
		defer dropFailingTrigger(db.GetDB(), trigger, "academic_periods")

		errUpdate := periods.Update(ctx, &domain.AcademicPeriod{
			ID:        periodInactiveID,
			TenantID:  tenantID,
			Name:      "Periodo inactivo",
			Code:      "INA-" + suffix,
			StartDate: time.Now().AddDate(0, -1, 0),
			EndDate:   time.Now().AddDate(0, 1, 0),
			IsActive:  true,
		})
		if errUpdate == nil {
			t.Fatal("la activación debía fallar al guardar el periodo")
		}

		var active int
		if errCount := db.GetDB().GetContext(ctx, &active, `
			SELECT COUNT(*) FROM academic_periods WHERE tenant_id = $1 AND is_active = true
		`, tenantID); errCount != nil {
			t.Fatalf("no se pudo contar los periodos activos: %v", errCount)
		}
		if active != 1 {
			t.Fatalf("la activación fallida dejó %d periodo(s) activo(s), esperaba 1", active)
		}
	})

	t.Run("archivar periodos vencidos no deja materias sin sellar", func(t *testing.T) {
		trigger := "solv_it_block_subject_seal"
		installFailingTrigger(t, db.GetDB(), trigger, "subjects", "BEFORE", "UPDATE", "")
		defer dropFailingTrigger(db.GetDB(), trigger, "subjects")

		if _, errArchive := periods.ArchiveExpiredPeriods(ctx); errArchive == nil {
			t.Fatal("el archivo debía fallar al sellar las materias")
		}

		var archived bool
		if errRead := db.GetDB().GetContext(ctx, &archived, `
			SELECT is_archived FROM academic_periods WHERE id = $1
		`, periodExpiredID); errRead != nil {
			t.Fatalf("no se pudo leer el periodo vencido: %v", errRead)
		}
		if archived {
			t.Fatal("el archivo fallido dejó el periodo sellado sin sus materias")
		}
	})

	t.Run("cerrar el aula saca a los participantes o no cierra", func(t *testing.T) {
		sessionID := uuid.NewString()
		if _, errSession := db.GetDB().Exec(`
			INSERT INTO class_sessions (id, tenant_id, subject_id, status, opens_at, ends_at, created_by)
			VALUES ($1, $2, $3, 'open', NOW(), NOW() + INTERVAL '1 hour', $4)
		`, sessionID, tenantID, subjectID, teacherID); errSession != nil {
			t.Fatalf("no se pudo sembrar el aula: %v", errSession)
		}
		if _, errParticipant := db.GetDB().Exec(`
			INSERT INTO class_session_participants (session_id, student_id, tenant_id)
			VALUES ($1, $2, $3)
		`, sessionID, studentID, tenantID); errParticipant != nil {
			t.Fatalf("no se pudo sembrar la presencia: %v", errParticipant)
		}

		trigger := "solv_it_block_session_departure"
		installFailingTrigger(t, db.GetDB(), trigger, "class_session_participants", "BEFORE", "UPDATE", "")
		_, errClose := sessions.CloseSession(ctx, tenantID, teacherID, sessionID)
		dropFailingTrigger(db.GetDB(), trigger, "class_session_participants")
		if errClose == nil {
			t.Fatal("el cierre debía fallar al sellar la salida")
		}

		var status string
		if errRead := db.GetDB().GetContext(ctx, &status, `
			SELECT status FROM class_sessions WHERE id = $1
		`, sessionID); errRead != nil {
			t.Fatalf("no se pudo leer el aula: %v", errRead)
		}
		if status != "open" {
			t.Fatalf("el cierre fallido dejó el aula en estado %q, esperaba abierta", status)
		}

		// Sin el fallo, el reintento cierra el aula y sella la salida.
		if _, errRetry := sessions.CloseSession(ctx, tenantID, teacherID, sessionID); errRetry != nil {
			t.Fatalf("el cierre válido fue rechazado: %v", errRetry)
		}
		var leftAt *time.Time
		if errRead := db.GetDB().GetContext(ctx, &leftAt, `
			SELECT left_at FROM class_session_participants WHERE session_id = $1 AND student_id = $2
		`, sessionID, studentID); errRead != nil {
			t.Fatalf("no se pudo leer la salida del participante: %v", errRead)
		}
		if leftAt == nil {
			t.Fatal("el cierre válido no selló la salida del participante")
		}
	})

	t.Run("vincular un entorno al aula guarda vínculo y presencia juntos", func(t *testing.T) {
		sessionID := uuid.NewString()
		environmentID := uuid.NewString()
		if _, errSession := db.GetDB().Exec(`
			INSERT INTO class_sessions (id, tenant_id, subject_id, status, opens_at, ends_at, created_by)
			VALUES ($1, $2, $3, 'open', NOW(), NOW() + INTERVAL '1 hour', $4)
		`, sessionID, tenantID, subjectID, teacherID); errSession != nil {
			t.Fatalf("no se pudo sembrar el aula: %v", errSession)
		}
		if _, errEnv := db.GetDB().Exec(`
			INSERT INTO environments (id, student_id, subject_id, tenant_id, access_url)
			VALUES ($1, $2, $3, $4, 'http://localhost:3000')
		`, environmentID, studentID, subjectID, tenantID); errEnv != nil {
			t.Fatalf("no se pudo sembrar el entorno: %v", errEnv)
		}

		trigger := "solv_it_block_presence_insert"
		installFailingTrigger(t, db.GetDB(), trigger, "class_session_participants", "BEFORE", "INSERT", "")
		errAttach := sessions.AttachPracticeEnvironmentToOpenSession(ctx, tenantID, studentID, subjectID, environmentID)
		dropFailingTrigger(db.GetDB(), trigger, "class_session_participants")
		if errAttach == nil {
			t.Fatal("el vínculo debía fallar al registrar la presencia")
		}

		var linkedSession *string
		if errRead := db.GetDB().GetContext(ctx, &linkedSession, `
			SELECT session_id FROM environments WHERE id = $1
		`, environmentID); errRead != nil {
			t.Fatalf("no se pudo leer el entorno: %v", errRead)
		}
		if linkedSession != nil {
			t.Fatalf("el vínculo fallido dejó el entorno apuntando al aula %q", *linkedSession)
		}

		if errRetry := sessions.AttachPracticeEnvironmentToOpenSession(ctx, tenantID, studentID, subjectID, environmentID); errRetry != nil {
			t.Fatalf("el vínculo válido fue rechazado: %v", errRetry)
		}
		var linkedEnvironment string
		if errRead := db.GetDB().GetContext(ctx, &linkedEnvironment, `
			SELECT COALESCE(environment_id::text, '') FROM class_session_participants
			WHERE session_id = $1 AND student_id = $2
		`, sessionID, studentID); errRead != nil {
			t.Fatalf("no se pudo leer la presencia: %v", errRead)
		}
		if linkedEnvironment != environmentID {
			t.Fatalf("la presencia quedó con el entorno %q, esperaba %q", linkedEnvironment, environmentID)
		}
	})

	t.Run("el alta de materia no deja una entrada suelta en el catálogo", func(t *testing.T) {
		code := "ATOM-" + suffix
		missingTeacher := uuid.NewString()
		errCreate := subjects.Create(ctx, &domain.Subject{
			ID:        uuid.NewString(),
			TenantID:  tenantID,
			Name:      "Materia atómica",
			Code:      code,
			TeacherID: &missingTeacher,
		})
		if errCreate == nil {
			t.Fatal("el alta debía fallar por el docente inexistente")
		}

		var catalog int
		if errCount := db.GetDB().GetContext(ctx, &catalog, `
			SELECT COUNT(*) FROM subject_catalog WHERE tenant_id = $1 AND upper(code) = upper($2)
		`, tenantID, code); errCount != nil {
			t.Fatalf("no se pudo contar el catálogo: %v", errCount)
		}
		if catalog != 0 {
			t.Fatalf("el alta fallida dejó %d entrada(s) en el catálogo", catalog)
		}
	})
}

func installFailingTrigger(t *testing.T, db *sqlx.DB, name, table, timing, event, when string) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE OR REPLACE FUNCTION ` + name + `() RETURNS trigger AS $$
		BEGIN RAISE EXCEPTION 'fallo simulado por la prueba'; END;
		$$ LANGUAGE plpgsql
	`); err != nil {
		t.Fatalf("no se pudo crear el disparador %s: %v", name, err)
	}
	clause := ""
	if when != "" {
		clause = " WHEN (" + when + ")"
	}
	if _, err := db.Exec(`
		CREATE TRIGGER ` + name + ` ` + timing + ` ` + event + ` ON ` + table + `
		FOR EACH ROW` + clause + ` EXECUTE FUNCTION ` + name + `()
	`); err != nil {
		t.Fatalf("no se pudo instalar el disparador %s: %v", name, err)
	}
}

func dropFailingTrigger(db *sqlx.DB, name, table string) {
	_, _ = db.Exec(`DROP TRIGGER IF EXISTS ` + name + ` ON ` + table)
	_, _ = db.Exec(`DROP FUNCTION IF EXISTS ` + name + `()`)
}
