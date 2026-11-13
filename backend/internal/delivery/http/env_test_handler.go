package httpdelivery

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// EnvTestHandler gestiona las peticiones HTTP del ciclo de vida de pruebas de entorno asíncronas
type EnvTestHandler struct {
	svc         *services.EnvTestService
	practiceSvc *services.PracticeEnvTestService
}

// NewEnvTestHandler crea una nueva instancia del handler HTTP para pruebas de entorno
func NewEnvTestHandler(svc *services.EnvTestService) *EnvTestHandler {
	return &EnvTestHandler{svc: svc}
}

// SetPracticeService habilita las pruebas de entorno por práctica.
func (h *EnvTestHandler) SetPracticeService(svc *services.PracticeEnvTestService) *EnvTestHandler {
	h.practiceSvc = svc
	return h
}

// StartEnvTest inicia un nuevo job asíncrono de verificación de entorno
// POST /api/v1/jobs/env-test
func (h *EnvTestHandler) StartEnvTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}

	var req domain.StartEnvTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_json", "Cuerpo de solicitud inválido")
		return
	}

	req.OwnerID = domain.GetUserID(r.Context())
	req.TenantID = getTenantFromCtx(r)
	job, err := h.svc.StartJob(r.Context(), req)
	if err != nil {
		SendError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	SendJSON(w, http.StatusAccepted, job, "Job de prueba de entorno iniciado")
}

// StartPracticeEnvTest inicia la prueba de una práctica IDE con sus satélites y su
// repositorio base, sin crear entregas ni environments.
// POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/env-test
func (h *EnvTestHandler) StartPracticeEnvTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}
	if h.practiceSvc == nil {
		SendError(w, http.StatusServiceUnavailable, "practice_env_test_unavailable", "La prueba de entorno por práctica no está disponible")
		return
	}

	role := domain.GetUserRole(r.Context())
	if role != "teacher" && role != "admin" {
		SendError(w, http.StatusForbidden, "forbidden", "Solo docentes y administradores pueden probar el entorno de una práctica")
		return
	}

	exerciseID := r.PathValue("exerciseId")
	if strings.TrimSpace(exerciseID) == "" {
		SendError(w, http.StatusBadRequest, "missing_exercise", "Identificador de práctica requerido")
		return
	}

	job, err := h.practiceSvc.Start(r.Context(), services.PracticeEnvTestRequest{
		TeacherID:  domain.GetUserID(r.Context()),
		TenantID:   getTenantFromCtx(r),
		Role:       role,
		CourseID:   r.PathValue("courseId"),
		ExerciseID: exerciseID,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPracticeTestForbidden):
			SendError(w, http.StatusForbidden, "forbidden", err.Error())
		case errors.Is(err, services.ErrPracticeTestRequiresIDE):
			SendError(w, http.StatusBadRequest, "invalid_environment", err.Error())
		case errors.Is(err, domain.ErrNotFound):
			SendError(w, http.StatusNotFound, "practice_not_found", "La práctica o su materia no existen")
		default:
			SendError(w, http.StatusBadRequest, "validation_error", err.Error())
		}
		return
	}

	SendJSON(w, http.StatusAccepted, job, "Prueba de entorno de la práctica iniciada")
}

// GetLatestPracticeEnvTest devuelve la última prueba de una práctica para el docente.
// GET /api/v1/teacher/exercises/{exerciseId}/env-test/latest
func (h *EnvTestHandler) GetLatestPracticeEnvTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}
	if h.practiceSvc == nil {
		SendError(w, http.StatusServiceUnavailable, "practice_env_test_unavailable", "La prueba de entorno por práctica no está disponible")
		return
	}

	exerciseID := r.PathValue("exerciseId")
	if strings.TrimSpace(exerciseID) == "" {
		SendError(w, http.StatusBadRequest, "missing_exercise", "Identificador de práctica requerido")
		return
	}

	job, err := h.practiceSvc.Latest(
		r.Context(),
		domain.GetUserID(r.Context()),
		getTenantFromCtx(r),
		domain.GetUserRole(r.Context()),
		exerciseID,
	)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "latest_env_test_error", "No se pudo consultar la última prueba de la práctica")
		return
	}

	SendJSON(w, http.StatusOK, job, "Última prueba de entorno de la práctica")
}

// ListPracticeStudentDatabases lista las bases de los alumnos que ya abrieron la práctica.
// GET /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/student-databases
func (h *EnvTestHandler) ListPracticeStudentDatabases(w http.ResponseWriter, r *http.Request) {
	if h.practiceSvc == nil {
		SendError(w, http.StatusServiceUnavailable, "practice_env_test_unavailable", "La prueba de entorno por práctica no está disponible")
		return
	}

	role := domain.GetUserRole(r.Context())
	if role != "teacher" && role != "admin" {
		SendError(w, http.StatusForbidden, "forbidden", "Solo docentes y administradores pueden ver las bases de la práctica")
		return
	}

	databases, err := h.practiceSvc.ListStudentDatabases(
		r.Context(),
		domain.GetUserID(r.Context()),
		getTenantFromCtx(r),
		role,
		r.PathValue("courseId"),
		r.PathValue("exerciseId"),
	)
	if err != nil {
		h.sendPracticeDatabaseError(w, err, "No se pudieron consultar las bases de la práctica")
		return
	}

	SendJSON(w, http.StatusOK, databases, "Bases de la práctica por alumno")
}

// ResetPracticeStudentDatabase borra y recrea la base de un alumno con el script vigente.
// POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/students/{studentId}/database/reset
func (h *EnvTestHandler) ResetPracticeStudentDatabase(w http.ResponseWriter, r *http.Request) {
	if h.practiceSvc == nil {
		SendError(w, http.StatusServiceUnavailable, "practice_env_test_unavailable", "La prueba de entorno por práctica no está disponible")
		return
	}

	role := domain.GetUserRole(r.Context())
	if role != "teacher" && role != "admin" {
		SendError(w, http.StatusForbidden, "forbidden", "Solo docentes y administradores pueden reiniciar la base de una práctica")
		return
	}

	studentID := strings.TrimSpace(r.PathValue("studentId"))
	err := h.practiceSvc.ResetStudentDatabase(
		r.Context(),
		domain.GetUserID(r.Context()),
		getTenantFromCtx(r),
		role,
		r.PathValue("courseId"),
		r.PathValue("exerciseId"),
		studentID,
	)
	if err != nil {
		h.sendPracticeDatabaseError(w, err, "No se pudo reiniciar la base de la práctica")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"student_id": studentID}, "Base de la práctica reiniciada")
}

// sendPracticeDatabaseError traduce los errores del ciclo de vida de bases a respuestas
// HTTP concretas. Los fallos operativos se registran y no se expone el detalle interno.
func (h *EnvTestHandler) sendPracticeDatabaseError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, services.ErrPracticeTestForbidden):
		SendError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, services.ErrPracticeTestRequiresIDE):
		SendError(w, http.StatusBadRequest, "invalid_environment", err.Error())
	case errors.Is(err, services.ErrStudentPracticeNotStarted):
		SendError(w, http.StatusConflict, "practice_not_started", err.Error())
	case errors.Is(err, services.ErrPracticeDatabaseNotConfigured):
		SendError(w, http.StatusBadRequest, "database_not_configured", err.Error())
	case errors.Is(err, services.ErrPracticeDatabasesUnavailable):
		SendError(w, http.StatusServiceUnavailable, "databases_unavailable", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		SendError(w, http.StatusNotFound, "practice_not_found", "La práctica o su materia no existen")
	default:
		log.Printf("[PracticeDatabases] %s: %v", fallback, err)
		SendError(w, http.StatusInternalServerError, "database_operation_failed", fallback)
	}
}

// GetEnvTest consulta el estado y progreso en tiempo real de un job
// GET /api/v1/jobs/env-test/{id}
func (h *EnvTestHandler) GetEnvTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		// Fallback para rutas sin path values estándar
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) > 0 {
			id = parts[len(parts)-1]
		}
	}

	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "Identificador de job requerido")
		return
	}

	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil || !ownsEnvTestJob(r, job) {
		SendError(w, http.StatusNotFound, "job_not_found", "El job de prueba especificado no existe")
		return
	}

	SendJSON(w, http.StatusOK, job, "Detalle del job de prueba de entorno")
}

// CancelEnvTest cancela la ejecución de un job activo
// POST /api/v1/jobs/env-test/{id}/cancel
func (h *EnvTestHandler) CancelEnvTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			id = parts[len(parts)-2]
		}
	}

	if id == "" {
		SendError(w, http.StatusBadRequest, "missing_id", "Identificador de job requerido")
		return
	}

	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil || !ownsEnvTestJob(r, job) {
		SendError(w, http.StatusNotFound, "job_not_found", "El job de prueba especificado no existe")
		return
	}

	var cancelErr error
	if h.practiceSvc != nil {
		cancelErr = h.practiceSvc.CancelJob(r.Context(), id)
	}
	if err := h.svc.CancelJob(r.Context(), id); err != nil && cancelErr == nil {
		cancelErr = err
	}
	if cancelErr != nil {
		SendError(w, http.StatusInternalServerError, "cancel_error", "Error al cancelar el job")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "canceled"}, "Job cancelado correctamente")
}

func ownsEnvTestJob(r *http.Request, job *domain.EnvTestJob) bool {
	if job == nil {
		return false
	}
	ownerID := domain.GetUserID(r.Context())
	return ownerID != "" && job.OwnerID == ownerID && job.TenantID == getTenantFromCtx(r)
}
