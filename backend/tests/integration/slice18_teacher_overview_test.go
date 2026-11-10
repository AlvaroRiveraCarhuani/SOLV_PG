package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
			ID          string    `json:"id"`
			SubjectID   string    `json:"subject_id"`
			SubjectCode string    `json:"subject_code"`
			EndsAt      time.Time `json:"ends_at"`
			Present     int       `json:"present"`
		} `json:"session"`
		Queue struct {
			Pending   int     `json:"pending"`
			Practices int     `json:"practices"`
			Oldest    *string `json:"oldest_submitted_at"`
		} `json:"queue"`
		DueSoon []struct {
			SubjectID     string    `json:"subject_id"`
			SubjectCode   string    `json:"subject_code"`
			SubjectName   string    `json:"subject_name"`
			ExerciseID    string    `json:"exercise_id"`
			ExerciseTitle string    `json:"exercise_title"`
			DueAt         time.Time `json:"due_at"`
			Urgency       string    `json:"urgency"`
			PendingReview int       `json:"pending_review"`
		} `json:"due_soon"`
		DueSoonTotal  int `json:"due_soon_total"`
		DueSoonHidden int `json:"due_soon_hidden"`
		Subjects      []struct {
			ID                string `json:"id"`
			Code              string `json:"code"`
			Enrolled          int    `json:"enrolled"`
			PendingReview     int    `json:"pending_review"`
			WithoutSubmission *int   `json:"without_submission"`
			NextDueTitle      string `json:"next_due_title"`
		} `json:"subjects"`
		Signals []struct {
			Type string `json:"type"`
		} `json:"signals"`
	} `json:"data"`
}

// TestSlice18_TeacherOverview_ReadModel comprueba las reglas del panel: los
// conteos los resuelve el servidor, el resumen y las materias no pueden
// discrepar, el aula abierta viaja con su materia y su hora de fin, y ninguna
// cifra de infraestructura llega a la pantalla del docente.
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
		"SELECT id FROM lab_templates WHERE target_environment='JUEZ_EFIMERO' AND status IN ('approved','APROBADA') LIMIT 1"); err != nil {
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

	newRequest := func(teacher, query, role string) *http.Request {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/teacher/overview"+query, nil)
		req.Header.Set("X-Test-Tenant-Id", tenantID)
		req.Header.Set("X-Test-User-Id", teacher)
		req.Header.Set("X-Test-User-Role", role)
		return req
	}

	t.Run("el rol estudiante no accede al resumen docente", func(t *testing.T) {
		res, err := http.DefaultClient.Do(newRequest(teacherID, "", "student"))
		if err != nil {
			t.Fatalf("la petición falló: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("se esperaba 403, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("un period_id que no es UUID se rechaza", func(t *testing.T) {
		res, err := http.DefaultClient.Do(newRequest(teacherID, "?period_id=no-es-uuid", "teacher"))
		if err != nil {
			t.Fatalf("la petición falló: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("se esperaba 400, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("el resumen y las materias no discrepan", func(t *testing.T) {
		res, err := http.DefaultClient.Do(newRequest(teacherID, "", "teacher"))
		if err != nil {
			t.Fatalf("la petición falló: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("se esperaba 200, se obtuvo %d", res.StatusCode)
		}

		raw, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}
		if strings.Contains(string(raw), `"environments":`) {
			t.Errorf("el resumen sigue publicando cifras de contenedores")
		}
		if strings.Contains(string(raw), `"next_due":`) {
			t.Errorf("el resumen sigue publicando un solo vencimiento en lugar de la lista")
		}

		var body overviewEnvelope
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}

		var pending int
		var found bool
		for _, subject := range body.Data.Subjects {
			pending += subject.PendingReview

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
			if subject.NextDueTitle != "Práctica de auditoría" {
				t.Errorf("el título del vencimiento debía viajar con su fecha, se obtuvo %q", subject.NextDueTitle)
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
		if body.Data.Session != nil {
			t.Errorf("sin aula abierta la sesión debe ser nula")
		}
		if len(body.Data.Signals) != 0 {
			t.Errorf("las señales quedan vacías hasta que exista la agrupación por causa")
		}

		if body.Data.DueSoonTotal != 1 {
			t.Errorf("vencimientos próximos: se esperaba un total de 1, se obtuvo %d", body.Data.DueSoonTotal)
		}
		if body.Data.DueSoonHidden != 0 {
			t.Errorf("sin corte, los vencimientos ocultos deben ser 0, se obtuvo %d", body.Data.DueSoonHidden)
		}
		if len(body.Data.DueSoon) != 1 {
			t.Fatalf("se esperaba un vencimiento en la lista, se obtuvo %d", len(body.Data.DueSoon))
		}
		due := body.Data.DueSoon[0]
		if due.SubjectID != subjectID || due.ExerciseID != exerciseID {
			t.Errorf("el vencimiento no apunta a la práctica sembrada: %+v", due)
		}
		if due.SubjectCode == "" || due.ExerciseTitle != "Práctica de auditoría" {
			t.Errorf("el vencimiento llega sin código de materia o sin título: %+v", due)
		}
		if due.Urgency == "" {
			t.Errorf("el vencimiento debe llegar con su urgencia calculada en el servidor: %+v", due)
		}
	})

	t.Run("el aula abierta viaja con su materia y su hora de fin", func(t *testing.T) {
		// Un docente distinto para no alterar el caso sin aula. La sesión se
		// inserta directa: la apertura por HTTP tiene su propia prueba en el aula.
		sessionTeacherID := uuid.NewString()
		sessionSubjectID := uuid.NewString()
		if _, err := db.GetDB().ExecContext(ctx, `
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'Docente', 'Aula', $2, 'teacher', $3)
		`, sessionTeacherID, fmt.Sprintf("docente_aula_%s@uab.edu.bo", sessionTeacherID[:8]), tenantID); err != nil {
			t.Fatalf("no se pudo crear el segundo docente: %v", err)
		}
		if _, err := db.GetDB().ExecContext(ctx, `
			INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
			VALUES ($1, $2, 'Aula del Panel', $3, $4)
		`, sessionSubjectID, tenantID, "AUL-"+sessionSubjectID[:8], sessionTeacherID); err != nil {
			t.Fatalf("no se pudo crear la materia del aula: %v", err)
		}

		var sessionID string
		if err := db.GetDB().GetContext(ctx, &sessionID, `
			INSERT INTO class_sessions (tenant_id, subject_id, status, opens_at, ends_at, created_by)
			VALUES ($1, $2, 'open', NOW(), NOW() + INTERVAL '90 minutes', $3)
			RETURNING id
		`, tenantID, sessionSubjectID, sessionTeacherID); err != nil {
			t.Fatalf("no se pudo abrir el aula: %v", err)
		}
		defer func() {
			_, _ = db.GetDB().ExecContext(ctx, `DELETE FROM class_sessions WHERE id = $1`, sessionID)
		}()

		res, err := http.DefaultClient.Do(newRequest(sessionTeacherID, "", "teacher"))
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
		if body.Data.Session == nil {
			t.Fatalf("con aula abierta la sesión no puede ser nula")
		}
		if body.Data.Session.SubjectCode == "" {
			t.Errorf("la sesión llega sin código de materia")
		}
		if body.Data.Session.EndsAt.IsZero() {
			t.Errorf("la sesión llega sin hora de fin: la franja no puede decir cuándo cierra")
		}
		if body.Data.DueSoonTotal != 0 || len(body.Data.DueSoon) != 0 {
			t.Errorf("sin prácticas por vencer la lista debe ir vacía, se obtuvo %d", body.Data.DueSoonTotal)
		}
	})
}
