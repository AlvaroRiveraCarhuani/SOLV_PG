package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

type aulaEnvelope struct {
	Data struct {
		ID         string `json:"id"`
		Status     string `json:"status"`
		SubjectID  string `json:"subject_id"`
		Enrolled   int    `json:"enrolled"`
		Present    int    `json:"present"`
		EndsAt     string `json:"ends_at"`
		ClosesAt   string `json:"closes_at"`
	} `json:"data"`
}

type boardEnvelope struct {
	Data struct {
		Session struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"session"`
		Summary struct {
			Enrolled          int `json:"enrolled"`
			Present           int `json:"present"`
			NotStarted        int `json:"not_started"`
			Working           int `json:"working"`
			Submitted         int `json:"submitted"`
			Stuck             int `json:"stuck"`
			Absent            int `json:"absent"`
			WithoutSubmission int `json:"without_submission"`
		} `json:"summary"`
		Students []struct {
			StudentID string `json:"student_id"`
			State     string `json:"state"`
			Present   bool   `json:"present"`
			Verdict   string `json:"verdict"`
			Attempts  int    `json:"attempts"`
		} `json:"students"`
	} `json:"data"`
}

func setupClassSessionTestServer(t *testing.T) (*httptest.Server, *database.Database) {
	t.Helper()

	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Skipf("se omite la prueba: base de datos no disponible: %v", err)
		return nil, nil
	}

	repo := postgres.NewPostgresClassSessionRepository(db.GetDB())
	service := services.NewClassSessionService(repo)
	handler := httpdelivery.NewClassSessionHandler(service)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-Id")
			if tenantID == "" {
				tenantID = "00000000-0000-0000-0000-000000000001"
			}
			ctx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	handlers := httpdelivery.Handlers{
		ClassSessionHandler: handler,
		TenantMiddleware:    tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db
}

// TestSlice19_ClassSession_Aula comprueba el contrato del aula: una sola sesión
// abierta por materia, presencia acotada a la sesión, trabados primero y cierre
// idempotente que deja de admitir presencia.
func TestSlice19_ClassSession_Aula(t *testing.T) {
	server, db := setupClassSessionTestServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	ctx := context.Background()
	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := uuid.NewString()
	otherTeacherID := uuid.NewString()
	subjectID := uuid.NewString()
	exerciseID := uuid.NewString()

	for _, id := range []string{teacherID, otherTeacherID} {
		_, err := db.GetDB().ExecContext(ctx, `
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'Docente', 'Aula', $2, 'teacher', $3)
		`, id, fmt.Sprintf("docente_aula_%s@uab.edu.bo", id[:8]), tenantID)
		if err != nil {
			t.Fatalf("no se pudo crear el docente: %v", err)
		}
	}

	_, err := db.GetDB().ExecContext(ctx, `
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Aula de Prueba', $3, $4)
	`, subjectID, tenantID, "AUL-"+subjectID[:8], teacherID)
	if err != nil {
		t.Fatalf("no se pudo crear la materia: %v", err)
	}

	studentIDs := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for i, studentID := range studentIDs {
		_, _ = db.GetDB().ExecContext(ctx, `
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, $2, 'Alumno', $3, 'student', $4)
		`, studentID, fmt.Sprintf("AlumnoAula%d", i+1), fmt.Sprintf("alumno_aula%d_%s@uab.edu.bo", i+1, studentID[:8]), tenantID)

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
		INSERT INTO exercises (id, tenant_id, subject_id, title, status, environment_type, template_id)
		VALUES ($1, $2, $3, 'Práctica del aula', 'published', 'JUEZ_EFIMERO', $4)
	`, exerciseID, tenantID, subjectID, templateID)
	if err != nil {
		t.Fatalf("no se pudo crear la práctica: %v", err)
	}

	do := func(method, path, userID, role string, payload any) *http.Response {
		t.Helper()
		var body *bytes.Reader
		if payload != nil {
			encoded, _ := json.Marshal(payload)
			body = bytes.NewReader(encoded)
		} else {
			body = bytes.NewReader(nil)
		}
		req, reqErr := http.NewRequestWithContext(ctx, method, server.URL+path, body)
		if reqErr != nil {
			t.Fatalf("no se pudo armar la petición: %v", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-Id", tenantID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("X-User-Role", role)

		res, errReq := http.DefaultClient.Do(req)
		if errReq != nil {
			t.Fatalf("la petición %s %s falló: %v", method, path, errReq)
		}
		return res
	}

	var sessionID string

	t.Run("el docente abre el aula de su materia", func(t *testing.T) {
		res := do(http.MethodPost, "/api/v1/teacher/sessions", teacherID, "teacher", map[string]string{
			"subject_id":  subjectID,
			"exercise_id": exerciseID,
		})
		defer res.Body.Close()
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("se esperaba 201, se obtuvo %d", res.StatusCode)
		}

		var body aulaEnvelope
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}
		sessionID = body.Data.ID
		if sessionID == "" {
			t.Fatal("el aula abierta no trae identificador")
		}
		if body.Data.Status != domain.ClassSessionStatusOpen {
			t.Fatalf("estado del aula: se esperaba open, se obtuvo %q", body.Data.Status)
		}
		if body.Data.Enrolled != 3 {
			t.Fatalf("inscritos: se esperaba 3, se obtuvo %d", body.Data.Enrolled)
		}
	})

	t.Run("la misma materia no admite dos aulas abiertas", func(t *testing.T) {
		res := do(http.MethodPost, "/api/v1/teacher/sessions", teacherID, "teacher", map[string]string{
			"subject_id": subjectID,
		})
		defer res.Body.Close()
		if res.StatusCode != http.StatusConflict {
			t.Fatalf("se esperaba 409, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("un docente ajeno no ve ni cierra el aula", func(t *testing.T) {
		res := do(http.MethodGet, "/api/v1/teacher/sessions/"+sessionID+"/board", otherTeacherID, "teacher", nil)
		defer res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("tablero ajeno: se esperaba 404, se obtuvo %d", res.StatusCode)
		}

		resClose := do(http.MethodPatch, "/api/v1/teacher/sessions/"+sessionID+"/close", otherTeacherID, "teacher", nil)
		defer resClose.Body.Close()
		if resClose.StatusCode != http.StatusNotFound {
			t.Fatalf("cierre ajeno: se esperaba 404, se obtuvo %d", resClose.StatusCode)
		}
	})

	t.Run("la presencia del alumno sale de su señal, no de una inscripción", func(t *testing.T) {
		res := do(http.MethodGet, "/api/v1/teacher/sessions/"+sessionID+"/board", teacherID, "teacher", nil)
		if res.StatusCode != http.StatusOK {
			res.Body.Close()
			t.Fatalf("tablero: se esperaba 200, se obtuvo %d", res.StatusCode)
		}
		var body boardEnvelope
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			res.Body.Close()
			t.Fatalf("respuesta ilegible: %v", err)
		}
		res.Body.Close()

		if body.Data.Summary.Present != 0 {
			t.Fatalf("sin señales la presencia debe ser 0, se obtuvo %d", body.Data.Summary.Present)
		}
		if body.Data.Summary.NotStarted != 3 {
			t.Fatalf("sin empezar: se esperaba 3, se obtuvo %d", body.Data.Summary.NotStarted)
		}

		// El alumno 1 entra al aula: una señal alcanza para contarlo dentro.
		resBeat := do(http.MethodPost, "/api/v1/student/sessions/"+sessionID+"/heartbeat", studentIDs[0], "student", nil)
		defer resBeat.Body.Close()
		if resBeat.StatusCode != http.StatusOK {
			t.Fatalf("heartbeat: se esperaba 200, se obtuvo %d", resBeat.StatusCode)
		}

		resBoard := do(http.MethodGet, "/api/v1/teacher/sessions/"+sessionID+"/board", teacherID, "teacher", nil)
		defer resBoard.Body.Close()
		if resBoard.StatusCode != http.StatusOK {
			t.Fatalf("tablero: se esperaba 200, se obtuvo %d", resBoard.StatusCode)
		}
		var refreshed boardEnvelope
		if err := json.NewDecoder(resBoard.Body).Decode(&refreshed); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}
		if refreshed.Data.Summary.Present != 1 {
			t.Fatalf("presencia: se esperaba 1, se obtuvo %d", refreshed.Data.Summary.Present)
		}
		if refreshed.Data.Summary.NotStarted != 2 {
			t.Fatalf("sin empezar: se esperaban 2, se obtuvo %d", refreshed.Data.Summary.NotStarted)
		}
	})

	t.Run("un alumno que no cursa la materia no puede latir en el aula", func(t *testing.T) {
		outsiderID := uuid.NewString()
		_, _ = db.GetDB().ExecContext(ctx, `
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'Alumno', 'Ajeno', $2, 'student', $3)
		`, outsiderID, fmt.Sprintf("alumno_ajeno_%s@uab.edu.bo", outsiderID[:8]), tenantID)

		res := do(http.MethodPost, "/api/v1/student/sessions/"+sessionID+"/heartbeat", outsiderID, "student", nil)
		defer res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("se esperaba 403, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("el tablero pone al trabado primero y cuenta sin entregar", func(t *testing.T) {
		_, err := db.GetDB().ExecContext(ctx, `
			INSERT INTO submissions (tenant_id, exercise_id, student_id, verdict, review_status, submitted_at)
			VALUES ($1, $2, $3, 'RE', 'pending', NOW())
		`, tenantID, exerciseID, studentIDs[1])
		if err != nil {
			t.Fatalf("no se pudo crear la entrega trabada: %v", err)
		}

		res := do(http.MethodGet, "/api/v1/teacher/sessions/"+sessionID+"/board", teacherID, "teacher", nil)
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("tablero: se esperaba 200, se obtuvo %d", res.StatusCode)
		}
		var body boardEnvelope
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}

		if body.Data.Summary.Stuck != 1 {
			t.Fatalf("trabados: se esperaba 1, se obtuvo %d", body.Data.Summary.Stuck)
		}
		if body.Data.Summary.WithoutSubmission != 2 {
			t.Fatalf("sin entregar: se esperaban 2, se obtuvo %d", body.Data.Summary.WithoutSubmission)
		}
		if len(body.Data.Students) == 0 || body.Data.Students[0].StudentID != studentIDs[1] {
			t.Fatal("la primera fila del aula debe ser el alumno trabado")
		}
		if body.Data.Students[0].State != domain.ParticipantStuck {
			t.Fatalf("estado: se esperaba %q, se obtuvo %q", domain.ParticipantStuck, body.Data.Students[0].State)
		}
	})

	t.Run("cerrar dos veces devuelve el mismo estado y corta la presencia", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			res := do(http.MethodPatch, "/api/v1/teacher/sessions/"+sessionID+"/close", teacherID, "teacher", nil)
			if res.StatusCode != http.StatusOK {
				res.Body.Close()
				t.Fatalf("cierre %d: se esperaba 200, se obtuvo %d", i+1, res.StatusCode)
			}
			var body aulaEnvelope
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				res.Body.Close()
				t.Fatalf("respuesta ilegible: %v", err)
			}
			res.Body.Close()
			if body.Data.Status != domain.ClassSessionStatusClosed {
				t.Fatalf("cierre %d: se esperaba closed, se obtuvo %q", i+1, body.Data.Status)
			}
			if body.Data.ClosesAt == "" {
				t.Fatalf("cierre %d: el aula cerrada no registra hora de cierre", i+1)
			}
		}

		resBeat := do(http.MethodPost, "/api/v1/student/sessions/"+sessionID+"/heartbeat", studentIDs[0], "student", nil)
		defer resBeat.Body.Close()
		if resBeat.StatusCode != http.StatusConflict {
			t.Fatalf("latido sobre aula cerrada: se esperaba 409, se obtuvo %d", resBeat.StatusCode)
		}

		resOpen := do(http.MethodGet, "/api/v1/teacher/sessions/open", teacherID, "teacher", nil)
		defer resOpen.Body.Close()
		if resOpen.StatusCode != http.StatusOK {
			t.Fatalf("aula abierta: se esperaba 200, se obtuvo %d", resOpen.StatusCode)
		}
		var open aulaEnvelope
		if err := json.NewDecoder(resOpen.Body).Decode(&open); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}
		if open.Data.ID != "" {
			t.Fatalf("después del cierre no debe quedar aula abierta, se obtuvo %s", open.Data.ID)
		}
	})

	t.Run("un aula fuera de rango de duración se rechaza", func(t *testing.T) {
		res := do(http.MethodPost, "/api/v1/teacher/sessions", teacherID, "teacher", map[string]any{
			"subject_id":        subjectID,
			"duration_minutes": 5,
		})
		defer res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("se esperaba 400, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("un aula vencida se cierra sola y deja abrir otra en la misma materia", func(t *testing.T) {
		res := do(http.MethodPost, "/api/v1/teacher/sessions", teacherID, "teacher", map[string]any{
			"subject_id":        subjectID,
			"duration_minutes": 45,
		})
		if res.StatusCode != http.StatusCreated {
			res.Body.Close()
			t.Fatalf("apertura: se esperaba 201, se obtuvo %d", res.StatusCode)
		}
		var opened aulaEnvelope
		if err := json.NewDecoder(res.Body).Decode(&opened); err != nil {
			res.Body.Close()
			t.Fatalf("respuesta ilegible: %v", err)
		}
		res.Body.Close()
		if opened.Data.EndsAt == "" {
			t.Fatal("el aula abierta no trae hora de fin prevista")
		}

		// Se adelanta el reloj del aula en la base: esperar 45 minutos no prueba nada
		// que esta fila no pruebe ya.
		if _, err := db.GetDB().ExecContext(ctx, `
			UPDATE class_sessions SET ends_at = NOW() - INTERVAL '1 minute' WHERE id = $1::uuid
		`, opened.Data.ID); err != nil {
			t.Fatalf("no se pudo vencer el aula: %v", err)
		}

		resOpen := do(http.MethodGet, "/api/v1/teacher/sessions/open", teacherID, "teacher", nil)
		defer resOpen.Body.Close()
		if resOpen.StatusCode != http.StatusOK {
			t.Fatalf("aula abierta: se esperaba 200, se obtuvo %d", resOpen.StatusCode)
		}
		var current aulaEnvelope
		if err := json.NewDecoder(resOpen.Body).Decode(&current); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}
		if current.Data.ID != "" {
			t.Fatalf("un aula vencida no debe seguir apareciendo abierta, se obtuvo %s", current.Data.ID)
		}

		resBeat := do(http.MethodPost, "/api/v1/student/sessions/"+opened.Data.ID+"/heartbeat", studentIDs[0], "student", nil)
		defer resBeat.Body.Close()
		if resBeat.StatusCode != http.StatusConflict {
			t.Fatalf("latido sobre aula vencida: se esperaba 409, se obtuvo %d", resBeat.StatusCode)
		}

		resAgain := do(http.MethodPost, "/api/v1/teacher/sessions", teacherID, "teacher", map[string]string{
			"subject_id": subjectID,
		})
		defer resAgain.Body.Close()
		if resAgain.StatusCode != http.StatusCreated {
			t.Fatalf("una aula vencida no debe bloquear la materia: se esperaba 201, se obtuvo %d", resAgain.StatusCode)
		}
	})
}

// TestSlice19_LastClosedAula comprueba que la pantalla recupera la última aula
// cerrada del docente al recargar, sin mostrar aulas de otros docentes.
func TestSlice19_LastClosedAula(t *testing.T) {
	server, db := setupClassSessionTestServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	ctx := context.Background()
	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := uuid.NewString()
	otherTeacherID := uuid.NewString()
	subjectID := uuid.NewString()

	for _, id := range []string{teacherID, otherTeacherID} {
		_, err := db.GetDB().ExecContext(ctx, `
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'Docente', 'Último', $2, 'teacher', $3)
		`, id, fmt.Sprintf("docente_ultimo_%s@uab.edu.bo", id[:8]), tenantID)
		if err != nil {
			t.Fatalf("no se pudo crear el docente: %v", err)
		}
	}

	_, err := db.GetDB().ExecContext(ctx, `
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Última Aula', $3, $4)
	`, subjectID, tenantID, "ULT-"+subjectID[:8], teacherID)
	if err != nil {
		t.Fatalf("no se pudo crear la materia: %v", err)
	}

	do := func(method, path, userID, role string, payload any) *http.Response {
		t.Helper()
		var body *bytes.Reader
		if payload != nil {
			encoded, _ := json.Marshal(payload)
			body = bytes.NewReader(encoded)
		} else {
			body = bytes.NewReader(nil)
		}
		req, reqErr := http.NewRequestWithContext(ctx, method, server.URL+path, body)
		if reqErr != nil {
			t.Fatalf("no se pudo armar la petición: %v", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-Id", tenantID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("X-User-Role", role)

		res, errReq := http.DefaultClient.Do(req)
		if errReq != nil {
			t.Fatalf("la petición %s %s falló: %v", method, path, errReq)
		}
		return res
	}

	lastClosed := func(userID, role string) aulaEnvelope {
		t.Helper()
		res := do(http.MethodGet, "/api/v1/teacher/sessions/last-closed", userID, role, nil)
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("last-closed: se esperaba 200, se obtuvo %d", res.StatusCode)
		}
		var body aulaEnvelope
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}
		return body
	}

	t.Run("sin aula cerrada responde data nulo", func(t *testing.T) {
		if got := lastClosed(teacherID, "teacher"); got.Data.ID != "" {
			t.Fatalf("se esperaba data nulo, se obtuvo la sesión %s", got.Data.ID)
		}
	})

	var sessionID string
	t.Run("tras cerrar, el docente recupera esa aula", func(t *testing.T) {
		res := do(http.MethodPost, "/api/v1/teacher/sessions", teacherID, "teacher", map[string]string{
			"subject_id": subjectID,
		})
		var opened aulaEnvelope
		_ = json.NewDecoder(res.Body).Decode(&opened)
		res.Body.Close()
		sessionID = opened.Data.ID
		if sessionID == "" {
			t.Fatal("no se pudo abrir el aula")
		}

		resClose := do(http.MethodPatch, "/api/v1/teacher/sessions/"+sessionID+"/close", teacherID, "teacher", nil)
		resClose.Body.Close()
		if resClose.StatusCode != http.StatusOK {
			t.Fatalf("cierre: se esperaba 200, se obtuvo %d", resClose.StatusCode)
		}

		got := lastClosed(teacherID, "teacher")
		if got.Data.ID != sessionID {
			t.Fatalf("última aula cerrada: se esperaba %s, se obtuvo %q", sessionID, got.Data.ID)
		}
		if got.Data.Status != domain.ClassSessionStatusClosed {
			t.Fatalf("estado: se esperaba closed, se obtuvo %q", got.Data.Status)
		}
		if got.Data.ClosesAt == "" {
			t.Fatal("la aula cerrada no trae hora de cierre para la ventana")
		}
	})

	t.Run("un docente ajeno no recibe el aula cerrada", func(t *testing.T) {
		if got := lastClosed(otherTeacherID, "teacher"); got.Data.ID != "" {
			t.Fatalf("el docente ajeno no debe ver el aula, se obtuvo %s", got.Data.ID)
		}
	})

	t.Run("un alumno recibe 403 y no ve aulas", func(t *testing.T) {
		res := do(http.MethodGet, "/api/v1/teacher/sessions/last-closed", otherTeacherID, "student", nil)
		defer res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("alumno: se esperaba 403, se obtuvo %d", res.StatusCode)
		}
	})

	t.Run("el admin ve la última aula cerrada del tenant", func(t *testing.T) {
		if got := lastClosed(otherTeacherID, "admin"); got.Data.ID != sessionID {
			t.Fatalf("admin: se esperaba %s, se obtuvo %q", sessionID, got.Data.ID)
		}
	})
}
