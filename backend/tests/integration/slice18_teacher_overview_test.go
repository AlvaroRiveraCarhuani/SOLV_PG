package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

type overviewEnvelope struct {
	Data struct {
		Period *struct {
			ID       string `json:"id"`
			Code     string `json:"code"`
			IsActive bool   `json:"is_active"`
		} `json:"period"`
		Session *struct {
			ID string `json:"id"`
		} `json:"session"`
		NextDue *struct {
			SubjectID     string `json:"subject_id"`
			ExerciseTitle string `json:"exercise_title"`
		} `json:"next_due"`
		Queue struct {
			Pending   int     `json:"pending"`
			Practices int     `json:"practices"`
			Oldest    *string `json:"oldest_submitted_at"`
		} `json:"queue"`
		Environments struct {
			OpenNow int `json:"open_now"`
		} `json:"environments"`
		Subjects []struct {
			ID                string `json:"id"`
			Code              string `json:"code"`
			Enrolled          int    `json:"enrolled"`
			OpenEnvironments  int    `json:"open_environments"`
			PendingReview     int    `json:"pending_review"`
			WithoutSubmission *int   `json:"without_submission"`
		} `json:"subjects"`
		Signals []struct {
			Type string `json:"type"`
		} `json:"signals"`
	} `json:"data"`
}

// TestSlice18_TeacherOverview_ReadModel comprueba las tres reglas del panel:
// los conteos los resuelve el servidor, la franja de trabajo y la tabla no
// pueden discrepar, y una excepción sin acción del docente no viaja.
func TestSlice18_TeacherOverview_ReadModel(t *testing.T) {
	server, db := setupTeacherDashboardTestServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	ctx := context.Background()
	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()

	_, err := db.GetDB().ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Docente', 'Resumen', $2, 'teacher', $3)
	`, teacherID, fmt.Sprintf("docente_overview_%s@uab.edu.bo", teacherID[:8]), tenantID)
	if err != nil {
		t.Fatalf("no se pudo crear el docente: %v", err)
	}

	_, err = db.GetDB().ExecContext(ctx, `
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Auditoría del Panel', $3, $4)
	`, subjectID, tenantID, "AUD-"+subjectID[:8], teacherID)
	if err != nil {
		t.Fatalf("no se pudo crear la materia: %v", err)
	}

	// Tres inscritos, dos con entrega pendiente de revisión y una práctica por
	// vencer: el resumen tiene que decir exactamente eso.
	studentIDs := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for i, studentID := range studentIDs {
		_, _ = db.GetDB().ExecContext(ctx, `
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, $2, 'Alumno', $3, 'student', $4)
		`, studentID, fmt.Sprintf("AlumnoResumen%d", i+1), fmt.Sprintf("alumno_resumen%d_%s@uab.edu.bo", i+1, studentID[:8]), tenantID)

		_, _ = db.GetDB().ExecContext(ctx, `
			INSERT INTO enrollments (tenant_id, student_id, subject_id)
			VALUES ($1, $2, $3)
		`, tenantID, studentID, subjectID)
	}

	var templateID string
	if err := db.GetDB().GetContext(ctx, &templateID,
		"SELECT id FROM lab_templates WHERE target_environment='JUEZ_EFIMERO' AND status='approved' LIMIT 1"); err != nil {
		t.Skipf("se omite la prueba: no hay plantilla de juez aprobada: %v", err)
	}

	_, err = db.GetDB().ExecContext(ctx, `
		INSERT INTO exercises (id, tenant_id, subject_id, title, status, due_date, template_id, environment_type)
		VALUES ($1, $2, $3, 'Práctica de auditoría', 'published', $4, $5, 'JUEZ_EFIMERO')
	`, exerciseID, tenantID, subjectID, time.Now().Add(48*time.Hour), templateID)
	if err != nil {
		t.Fatalf("no se pudo crear la práctica: %v", err)
	}

	for _, studentID := range studentIDs[:2] {
		_, err = db.GetDB().ExecContext(ctx, `
			INSERT INTO submissions (tenant_id, exercise_id, student_id, verdict, review_status)
			VALUES ($1, $2, $3, 'AC', 'pending')
		`, tenantID, exerciseID, studentID)
		if err != nil {
			t.Fatalf("no se pudo crear la entrega: %v", err)
		}
	}

	newRequest := func(query, role string) *http.Request {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/teacher/overview"+query, nil)
		req.Header.Set("X-Tenant-Id", tenantID)
		req.Header.Set("X-User-Id", teacherID)
		req.Header.Set("X-User-Role", role)
		return req
	}

	t.Run("el rol estudiante no accede al resumen docente", func(t *testing.T) {
		res, err := http.DefaultClient.Do(newRequest("", "student"))
		if err != nil {
			t.Fatalf("la petición falló: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("se esperaba 403, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("un period_id que no es UUID se rechaza", func(t *testing.T) {
		res, err := http.DefaultClient.Do(newRequest("?period_id=no-es-uuid", "teacher"))
		if err != nil {
			t.Fatalf("la petición falló: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("se esperaba 400, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("el resumen y la tabla de materias no discrepan", func(t *testing.T) {
		res, err := http.DefaultClient.Do(newRequest("", "teacher"))
		if err != nil {
			t.Fatalf("la petición falló: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
		}

		var body overviewEnvelope
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}

		var pending, open int
		var found bool
		for _, subject := range body.Data.Subjects {
			pending += subject.PendingReview
			open += subject.OpenEnvironments

			if subject.ID != subjectID {
				continue
			}
			found = true
			if subject.Enrolled != 3 {
				t.Errorf("inscritos: se esperaba 3, se obtuvo %d", subject.Enrolled)
			}
			if subject.PendingReview != 2 {
				t.Errorf("por calificar: se esperaba 2, se obtuvo %d", subject.PendingReview)
			}
			if subject.WithoutSubmission == nil || *subject.WithoutSubmission != 1 {
				t.Errorf("sin entregar: se esperaba 1, se obtuvo %v", subject.WithoutSubmission)
			}
		}
		if !found {
			t.Fatalf("la materia sembrada no aparece en el resumen")
		}

		if body.Data.Queue.Pending != pending {
			t.Errorf("la cola dice %d y las materias suman %d", body.Data.Queue.Pending, pending)
		}
		if body.Data.Queue.Practices != 1 {
			t.Errorf("prácticas con pendientes: se esperaba 1, se obtuvo %d", body.Data.Queue.Practices)
		}
		if body.Data.Environments.OpenNow != open {
			t.Errorf("entornos abiertos dice %d y las materias suman %d", body.Data.Environments.OpenNow, open)
		}
		if body.Data.Session != nil {
			t.Errorf("sin aula abierta la sesión debe ser nula")
		}
		if len(body.Data.Signals) != 0 {
			t.Errorf("las señales quedan vacías hasta que exista la sesión de aula")
		}
		if body.Data.NextDue == nil || body.Data.NextDue.SubjectID != subjectID {
			t.Errorf("el vencimiento más cercano debía ser la práctica sembrada")
		}
	})
}
