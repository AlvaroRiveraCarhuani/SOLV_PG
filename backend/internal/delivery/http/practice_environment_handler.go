package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type PracticeEnvironmentHandler struct {
	service  *services.PracticeEnvironmentService
	validate *validator.Validate
}

func NewPracticeEnvironmentHandler(service *services.PracticeEnvironmentService, validate *validator.Validate) *PracticeEnvironmentHandler {
	return &PracticeEnvironmentHandler{
		service:  service,
		validate: validate,
	}
}

type StartPracticeEnvironmentRequest struct {
	StudentID string `json:"student_id"`
	SubjectID string `json:"subject_id" validate:"required"`
	// ExerciseID es obligatorio: el entorno y su volumen pertenecen a una
	// práctica. Sin práctica no hay nada que abrir.
	ExerciseID string `json:"exercise_id" validate:"required"`
}

// resolveRequestStudentID prioriza la identidad del JWT y cae al header inyectado
// por ForwardAuth cuando el request ya viene autenticado aguas arriba.
func resolveRequestStudentID(r *http.Request) string {
	if userID := strings.TrimSpace(domain.GetUserID(r.Context())); userID != "" {
		return userID
	}
	return strings.TrimSpace(r.Header.Get("X-User-Id"))
}

func (h *PracticeEnvironmentHandler) StartPracticeEnvironment(w http.ResponseWriter, r *http.Request) {
	var req StartPracticeEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición JSON inválido")
		return
	}

	// R1: la identidad del alumno sale del JWT; student_id del body solo puede
	// sobrescribirla con rol admin/teacher. Un alumno que intenta iniciar el
	// environment de otro recibe 403.
	authStudentID := resolveRequestStudentID(r)
	role := strings.TrimSpace(domain.GetUserRole(r.Context()))
	if role == "" {
		role = strings.TrimSpace(r.Header.Get("X-User-Role"))
	}
	bodyStudentID := strings.TrimSpace(req.StudentID)

	if role == "admin" || role == "teacher" {
		if bodyStudentID == "" {
			req.StudentID = authStudentID
		}
	} else if bodyStudentID != "" && bodyStudentID != authStudentID {
		SendError(w, http.StatusForbidden, "student_id does not match the authenticated student", "No tiene permisos para iniciar un environment de otro alumno")
		return
	} else {
		req.StudentID = authStudentID
	}

	if err := h.validate.Struct(req); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Los campos subject_id y exercise_id son obligatorios: el entorno pertenece a una práctica")
		return
	}
	if strings.TrimSpace(req.StudentID) == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	instance, err := h.service.StartPracticeEnvironmentForExercise(r.Context(), req.StudentID, req.SubjectID, req.ExerciseID)
	if err != nil {
		if errors.Is(err, services.ErrHostMemoryExhausted) {
			SendError(w, http.StatusServiceUnavailable, err.Error(), "Servidor saturado: La memoria RAM del host cayó por debajo del 15% de margen de seguridad. Intente más tarde.")
			return
		}
		if errors.Is(err, services.ErrMaxContainersReached) {
			SendError(w, http.StatusServiceUnavailable, err.Error(), "Cupo institucional lleno: se alcanzó el máximo de entornos concurrentes configurado. Intente más tarde.")
			return
		}
		if errors.Is(err, services.ErrOOMKilledCooldownPenalty) {
			SendError(w, http.StatusTooManyRequests, err.Error(), "Memoria Excedida: Has superado la cuota de RAM (OOMKilled) 3 veces seguidas. Espera 5 minutos antes de reiniciar.")
			return
		}
		if errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
			SendError(w, http.StatusConflict, err.Error(), "El examen terminó: tu entorno quedó en solo lectura")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al iniciar el entorno de desarrollo interactivo")
		return
	}

	SendJSON(w, http.StatusOK, instance, "Entorno de desarrollo interactivo iniciado exitosamente")
}

// OpenPracticeIDE abre (o reutiliza) el IDE propio del docente para una práctica
// concreta y devuelve la URL para abrirlo en una pestaña nueva. El contenedor
// arranca con el volumen y la plantilla de esa práctica: el repositorio de una
// práctica no se mezcla con el de otra de la misma materia.
// POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/ide
func (h *PracticeEnvironmentHandler) OpenPracticeIDE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}

	role := strings.TrimSpace(domain.GetUserRole(r.Context()))
	if role == "" {
		role = strings.TrimSpace(r.Header.Get("X-User-Role"))
	}
	if role != "teacher" && role != "admin" {
		SendError(w, http.StatusForbidden, "forbidden", "Solo docentes y administradores pueden abrir el IDE de una práctica")
		return
	}

	exerciseID := strings.TrimSpace(r.PathValue("exerciseId"))
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "missing_exercise", "Identificador de práctica requerido")
		return
	}

	teacherID := resolveRequestStudentID(r)
	if teacherID == "" {
		SendError(w, http.StatusUnauthorized, "missing_identity", "No se pudo resolver la identidad del docente desde la sesión")
		return
	}

	courseID := strings.TrimSpace(r.PathValue("courseId"))
	instance, err := h.service.StartTeacherPracticePracticeEnvironment(r.Context(), teacherID, courseID, exerciseID, role)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPracticeTestForbidden):
			SendError(w, http.StatusForbidden, "forbidden", "La práctica pertenece a la materia de otro docente")
		case errors.Is(err, services.ErrPracticeIDENotProject):
			SendError(w, http.StatusBadRequest, "not_project_practice", err.Error())
		case errors.Is(err, domain.ErrPracticeEnvironmentFrozen):
			SendError(w, http.StatusConflict, "environment_frozen", "El entorno de esa práctica quedó en solo lectura")
		case errors.Is(err, domain.ErrNotFound):
			SendError(w, http.StatusNotFound, "practice_not_found", "La práctica o su materia no existen")
		case errors.Is(err, services.ErrHostMemoryExhausted):
			SendError(w, http.StatusServiceUnavailable, "host_memory_exhausted", "Servidor saturado: la memoria del host no alcanza para abrir el entorno")
		case errors.Is(err, services.ErrMaxContainersReached):
			SendError(w, http.StatusServiceUnavailable, "max_containers_reached", "Cupo institucional lleno: se alcanzó el máximo de entornos concurrentes")
		default:
			SendError(w, http.StatusInternalServerError, "environment_start_failed", "No se pudo abrir el IDE de la práctica")
		}
		return
	}

	if strings.TrimSpace(instance.AccessURL) == "" {
		SendError(w, http.StatusServiceUnavailable, "environment_url_unavailable", "El entorno arrancó pero todavía no tiene una URL de acceso")
		return
	}

	SendJSON(w, http.StatusOK, map[string]any{
		"environment_id": instance.ID,
		"exercise_id":    instance.ExerciseID,
		"subject_id":     instance.SubjectID,
		"template_id":    instance.TemplateID,
		"status":         instance.Status,
		"access_url":     instance.AccessURL,
		"volume_name":    instance.VolumeName(),
	}, "IDE de la práctica listo")
}

func (h *PracticeEnvironmentHandler) TerminatePracticeEnvironment(w http.ResponseWriter, r *http.Request) {
	environmentID := r.PathValue("id")
	if environmentID == "" {
		SendError(w, http.StatusBadRequest, "Missing environment ID", "Se requiere el ID del environment")
		return
	}

	if err := h.service.TerminatePracticeEnvironment(r.Context(), environmentID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "environment_not_found", "El laboratorio no existe o ya fue eliminado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "No se pudo finalizar el environment")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "terminated", "message": "PracticeEnvironment finalizado exitosamente. Recálculo EWMA y auditoría AST iniciados."}, "PracticeEnvironment finalizado exitosamente")
}

func (h *PracticeEnvironmentHandler) GetSemgrepAudit(w http.ResponseWriter, r *http.Request) {
	environmentID := r.PathValue("id")
	if environmentID == "" {
		SendError(w, http.StatusBadRequest, "Missing environment ID", "Se requiere el ID del environment")
		return
	}

	ws, err := h.service.GetSemgrepAudit(r.Context(), environmentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "environment_not_found", "El laboratorio no existe o ya fue eliminado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener auditoría AST")
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"environment_id": ws.ID,
		"audit_status":   "completed",
		"semgrep_audit":  ws.SemgrepAudit,
	}, "Auditoría AST obtenida exitosamente")
}

func (h *PracticeEnvironmentHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	environmentID := r.PathValue("id")
	if environmentID == "" {
		SendError(w, http.StatusBadRequest, "Missing environment ID", "Se requiere el ID del environment")
		return
	}

	if err := h.service.RecordHeartbeat(r.Context(), environmentID); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al registrar el latido (heartbeat)")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "alive", "environment_id": environmentID}, "Heartbeat registrado exitosamente")
}

func (h *PracticeEnvironmentHandler) RestartPracticeEnvironment(w http.ResponseWriter, r *http.Request) {
	environmentID := r.PathValue("id")
	if environmentID == "" {
		SendError(w, http.StatusBadRequest, "Missing environment ID", "Se requiere el ID del environment")
		return
	}

	instance, err := h.service.RestartPracticeEnvironment(r.Context(), environmentID)
	if err != nil {
		if errors.Is(err, services.ErrHostMemoryExhausted) {
			SendError(w, http.StatusServiceUnavailable, err.Error(), "Servidor saturado: Memoria RAM del host agotada para reiniciar el entorno.")
			return
		}
		if errors.Is(err, services.ErrOOMKilledCooldownPenalty) {
			SendError(w, http.StatusTooManyRequests, err.Error(), "Memoria Excedida: Has alcanzado el límite de 3 strikes por OOMKilled. Debes esperar el tiempo de enfriamiento (5 minutos).")
			return
		}
		if errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
			SendError(w, http.StatusConflict, err.Error(), "El examen terminó: tu entorno quedó en solo lectura")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "environment_not_found", "El laboratorio no existe o ya fue eliminado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al reiniciar el entorno")
		return
	}

	SendJSON(w, http.StatusOK, instance, "Entorno reiniciado exitosamente")
}

func (h *PracticeEnvironmentHandler) PausePracticeEnvironment(w http.ResponseWriter, r *http.Request) {
	environmentID := r.PathValue("id")
	if environmentID == "" {
		SendError(w, http.StatusBadRequest, "Missing environment ID", "Se requiere el ID del environment")
		return
	}

	ws, err := h.service.GetPracticeEnvironmentByID(r.Context(), environmentID)
	if err != nil {
		SendError(w, http.StatusNotFound, "PracticeEnvironment not found", "PracticeEnvironment no encontrado")
		return
	}

	userID := r.Header.Get("X-User-Id")
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" && userRole != "teacher" && (userID != "" && ws.StudentID != userID) {
		SendError(w, http.StatusForbidden, "Unauthorized to pause this environment", "No tiene permisos para pausar este environment")
		return
	}

	instance, err := h.service.PausePracticeEnvironment(r.Context(), environmentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "environment_not_found", "El laboratorio no existe o ya fue eliminado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al pausar el environment")
		return
	}

	SendJSON(w, http.StatusOK, instance, "PracticeEnvironment hibernado exitosamente")
}

type SubmitPracticeEnvironmentRequest struct {
	ExerciseID string            `json:"exercise_id" validate:"required"`
	Files      map[string]string `json:"files"`
}

// SubmitPracticeEnvironmentProject entrega el proyecto del IDE Persistente a SpeedGrader
// como una submission con el snapshot de archivos del volumen.
func (h *PracticeEnvironmentHandler) SubmitPracticeEnvironmentProject(w http.ResponseWriter, r *http.Request) {
	environmentID := r.PathValue("id")
	if environmentID == "" {
		SendError(w, http.StatusBadRequest, "Missing environment ID", "Se requiere el ID del environment")
		return
	}

	var req SubmitPracticeEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición JSON inválido")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "El campo exercise_id es obligatorio")
		return
	}

	studentID := resolveRequestStudentID(r)
	if studentID == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	submission, err := h.service.SubmitPracticeEnvironmentProject(r.Context(), environmentID, studentID, req.ExerciseID, req.Files)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPracticeEnvironmentForbidden):
			SendError(w, http.StatusForbidden, err.Error(), "No tiene permisos para entregar este environment")
		case errors.Is(err, services.ErrInvalidExerciseID):
			SendError(w, http.StatusBadRequest, err.Error(), "El exercise_id no es un UUID válido")
		case errors.Is(err, domain.ErrNotFound):
			SendError(w, http.StatusNotFound, err.Error(), "El entorno que intentas entregar no existe")
		case errors.Is(err, domain.ErrPracticeEnvironmentFrozen):
			SendError(w, http.StatusConflict, err.Error(), "El examen terminó: tu entorno quedó en solo lectura")
		case errors.Is(err, services.ErrSubmissionNotConfigured), errors.Is(err, services.ErrPracticeEnvironmentVolumeUnavailable):
			SendError(w, http.StatusServiceUnavailable, err.Error(), "La entrega de proyecto no está disponible en este momento")
		default:
			SendError(w, http.StatusInternalServerError, err.Error(), "Error al entregar el proyecto del environment")
		}
		return
	}

	SendJSON(w, http.StatusCreated, map[string]any{
		"submission_id":  submission.ID,
		"environment_id": environmentID,
		"exercise_id":    submission.ExerciseID,
		"verdict":        submission.Verdict,
		"verdict_origin": submission.VerdictOrigin,
		"score":          submission.Score,
	}, "Proyecto entregado correctamente")
}
