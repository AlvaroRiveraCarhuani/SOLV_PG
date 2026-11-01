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

type WorkspaceHandler struct {
	service  *services.WorkspaceService
	validate *validator.Validate
}

func NewWorkspaceHandler(service *services.WorkspaceService, validate *validator.Validate) *WorkspaceHandler {
	return &WorkspaceHandler{
		service:  service,
		validate: validate,
	}
}

type StartWorkspaceRequest struct {
	StudentID  string `json:"student_id"`
	SubjectID  string `json:"subject_id" validate:"required"`
	ExerciseID string `json:"exercise_id"`
}

// resolveRequestStudentID prioriza la identidad del JWT y cae al header inyectado
// por ForwardAuth cuando el request ya viene autenticado aguas arriba.
func resolveRequestStudentID(r *http.Request) string {
	if userID := strings.TrimSpace(domain.GetUserID(r.Context())); userID != "" {
		return userID
	}
	return strings.TrimSpace(r.Header.Get("X-User-Id"))
}

func (h *WorkspaceHandler) StartWorkspace(w http.ResponseWriter, r *http.Request) {
	var req StartWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición JSON inválido")
		return
	}

	// R1: la identidad del alumno sale del JWT; student_id del body solo puede
	// sobrescribirla con rol admin/teacher. Un alumno que intenta iniciar el
	// workspace de otro recibe 403.
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
		SendError(w, http.StatusForbidden, "student_id does not match the authenticated student", "No tiene permisos para iniciar un workspace de otro alumno")
		return
	} else {
		req.StudentID = authStudentID
	}

	if err := h.validate.Struct(req); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "El campo subject_id es obligatorio")
		return
	}
	if strings.TrimSpace(req.StudentID) == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	instance, err := h.service.StartWorkspaceForExercise(r.Context(), req.StudentID, req.SubjectID, req.ExerciseID)
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
		if errors.Is(err, domain.ErrWorkspaceFrozen) {
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
func (h *WorkspaceHandler) OpenPracticeIDE(w http.ResponseWriter, r *http.Request) {
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

	instance, err := h.service.StartTeacherPracticeWorkspace(r.Context(), teacherID, "", exerciseID, role)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPracticeTestForbidden):
			SendError(w, http.StatusForbidden, "forbidden", "La práctica pertenece a la materia de otro docente")
		case errors.Is(err, services.ErrPracticeIDENotProject):
			SendError(w, http.StatusBadRequest, "not_project_practice", err.Error())
		case errors.Is(err, domain.ErrWorkspaceFrozen):
			SendError(w, http.StatusConflict, "workspace_frozen", "El entorno de esa práctica quedó en solo lectura")
		case errors.Is(err, domain.ErrNotFound):
			SendError(w, http.StatusNotFound, "practice_not_found", "La práctica o su materia no existen")
		case errors.Is(err, services.ErrHostMemoryExhausted):
			SendError(w, http.StatusServiceUnavailable, "host_memory_exhausted", "Servidor saturado: la memoria del host no alcanza para abrir el entorno")
		case errors.Is(err, services.ErrMaxContainersReached):
			SendError(w, http.StatusServiceUnavailable, "max_containers_reached", "Cupo institucional lleno: se alcanzó el máximo de entornos concurrentes")
		default:
			SendError(w, http.StatusInternalServerError, "workspace_start_failed", "No se pudo abrir el IDE de la práctica")
		}
		return
	}

	if strings.TrimSpace(instance.AccessURL) == "" {
		SendError(w, http.StatusServiceUnavailable, "workspace_url_unavailable", "El entorno arrancó pero todavía no tiene una URL de acceso")
		return
	}

	SendJSON(w, http.StatusOK, map[string]any{
		"workspace_id": instance.ID,
		"exercise_id":  instance.ExerciseID,
		"subject_id":   instance.SubjectID,
		"template_id":  instance.TemplateID,
		"status":       instance.Status,
		"access_url":   instance.AccessURL,
		"volume_name":  instance.VolumeName(),
	}, "IDE de la práctica listo")
}

// GetWorkspaceForSubject devuelve el entorno del usuario autenticado en una
// materia, o nulo si todavía no abrió ninguno. La pantalla la usa antes de
// ofrecer abrir: cargar la vista no debe levantar un contenedor que nadie pidió.
func (h *WorkspaceHandler) GetWorkspaceForSubject(w http.ResponseWriter, r *http.Request) {
	userID := resolveRequestStudentID(r)
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del usuario desde la sesión")
		return
	}

	instance, err := h.service.WorkspaceForSubject(r.Context(), userID, r.URL.Query().Get("subject_id"))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrWorkspaceSubjectRequired):
			SendError(w, http.StatusBadRequest, err.Error(), "Se requiere el subject_id para consultar el entorno")
		case errors.Is(err, domain.ErrInvalidResourceID):
			SendError(w, http.StatusBadRequest, err.Error(), "La materia indicada no es un identificador válido")
		case errors.Is(err, domain.ErrNotFound):
			SendError(w, http.StatusNotFound, err.Error(), "La materia indicada no existe en esta institución")
		default:
			SendError(w, http.StatusInternalServerError, err.Error(), "Error al consultar el entorno de la materia")
		}
		return
	}

	if instance == nil {
		SendJSON(w, http.StatusOK, nil, "No hay entorno en esta materia")
		return
	}
	SendJSON(w, http.StatusOK, instance, "Entorno de la materia obtenido")
}

func (h *WorkspaceHandler) TerminateWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		SendError(w, http.StatusBadRequest, "Missing workspace ID", "Se requiere el ID del workspace")
		return
	}

	if err := h.service.TerminateWorkspace(r.Context(), workspaceID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "workspace_not_found", "El laboratorio no existe o ya fue eliminado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "No se pudo finalizar el workspace")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "terminated", "message": "Workspace finalizado exitosamente. Recálculo EWMA y auditoría AST iniciados."}, "Workspace finalizado exitosamente")
}

func (h *WorkspaceHandler) GetSemgrepAudit(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		SendError(w, http.StatusBadRequest, "Missing workspace ID", "Se requiere el ID del workspace")
		return
	}

	ws, err := h.service.GetSemgrepAudit(r.Context(), workspaceID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "workspace_not_found", "El laboratorio no existe o ya fue eliminado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener auditoría AST")
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"workspace_id":  ws.ID,
		"audit_status":  "completed",
		"semgrep_audit": ws.SemgrepAudit,
	}, "Auditoría AST obtenida exitosamente")
}

func (h *WorkspaceHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		SendError(w, http.StatusBadRequest, "Missing workspace ID", "Se requiere el ID del workspace")
		return
	}

	if err := h.service.RecordHeartbeat(r.Context(), workspaceID); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al registrar el latido (heartbeat)")
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "alive", "workspace_id": workspaceID}, "Heartbeat registrado exitosamente")
}

func (h *WorkspaceHandler) RestartWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		SendError(w, http.StatusBadRequest, "Missing workspace ID", "Se requiere el ID del workspace")
		return
	}

	instance, err := h.service.RestartWorkspace(r.Context(), workspaceID)
	if err != nil {
		if errors.Is(err, services.ErrHostMemoryExhausted) {
			SendError(w, http.StatusServiceUnavailable, err.Error(), "Servidor saturado: Memoria RAM del host agotada para reiniciar el entorno.")
			return
		}
		if errors.Is(err, services.ErrOOMKilledCooldownPenalty) {
			SendError(w, http.StatusTooManyRequests, err.Error(), "Memoria Excedida: Has alcanzado el límite de 3 strikes por OOMKilled. Debes esperar el tiempo de enfriamiento (5 minutos).")
			return
		}
		if errors.Is(err, domain.ErrWorkspaceFrozen) {
			SendError(w, http.StatusConflict, err.Error(), "El examen terminó: tu entorno quedó en solo lectura")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "workspace_not_found", "El laboratorio no existe o ya fue eliminado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al reiniciar el entorno")
		return
	}

	SendJSON(w, http.StatusOK, instance, "Entorno reiniciado exitosamente")
}

func (h *WorkspaceHandler) PauseWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		SendError(w, http.StatusBadRequest, "Missing workspace ID", "Se requiere el ID del workspace")
		return
	}

	ws, err := h.service.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		SendError(w, http.StatusNotFound, "Workspace not found", "Workspace no encontrado")
		return
	}

	userID := r.Header.Get("X-User-Id")
	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" && userRole != "teacher" && (userID != "" && ws.StudentID != userID) {
		SendError(w, http.StatusForbidden, "Unauthorized to pause this workspace", "No tiene permisos para pausar este workspace")
		return
	}

	instance, err := h.service.PauseWorkspace(r.Context(), workspaceID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			SendError(w, http.StatusNotFound, "workspace_not_found", "El laboratorio no existe o ya fue eliminado")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al pausar el workspace")
		return
	}

	SendJSON(w, http.StatusOK, instance, "Workspace hibernado exitosamente")
}

type SubmitWorkspaceRequest struct {
	ExerciseID string            `json:"exercise_id" validate:"required"`
	Files      map[string]string `json:"files"`
}

// SubmitWorkspaceProject entrega el proyecto del IDE Persistente a SpeedGrader
// como una submission con el snapshot de archivos del volumen.
func (h *WorkspaceHandler) SubmitWorkspaceProject(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		SendError(w, http.StatusBadRequest, "Missing workspace ID", "Se requiere el ID del workspace")
		return
	}

	var req SubmitWorkspaceRequest
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

	submission, err := h.service.SubmitWorkspaceProject(r.Context(), workspaceID, studentID, req.ExerciseID, req.Files)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrWorkspaceForbidden):
			SendError(w, http.StatusForbidden, err.Error(), "No tiene permisos para entregar este workspace")
		case errors.Is(err, services.ErrInvalidExerciseID):
			SendError(w, http.StatusBadRequest, err.Error(), "El exercise_id no es un UUID válido")
		case errors.Is(err, domain.ErrNotFound):
			SendError(w, http.StatusNotFound, err.Error(), "El entorno que intentas entregar no existe")
		case errors.Is(err, domain.ErrWorkspaceFrozen):
			SendError(w, http.StatusConflict, err.Error(), "El examen terminó: tu entorno quedó en solo lectura")
		case errors.Is(err, services.ErrSubmissionNotConfigured), errors.Is(err, services.ErrWorkspaceVolumeUnavailable):
			SendError(w, http.StatusServiceUnavailable, err.Error(), "La entrega de proyecto no está disponible en este momento")
		default:
			SendError(w, http.StatusInternalServerError, err.Error(), "Error al entregar el proyecto del workspace")
		}
		return
	}

	SendJSON(w, http.StatusCreated, map[string]any{
		"submission_id": submission.ID,
		"workspace_id":  workspaceID,
		"exercise_id":   submission.ExerciseID,
		"verdict":       submission.Verdict,
		"score":         submission.Score,
	}, "Proyecto entregado correctamente")
}
