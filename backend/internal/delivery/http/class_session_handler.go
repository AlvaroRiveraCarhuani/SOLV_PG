package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// ClassSessionHandler expone el aula: abrirla, cerrarla, su tablero y la señal
// de presencia del alumno.
type ClassSessionHandler struct {
	service *services.ClassSessionService
}

func NewClassSessionHandler(service *services.ClassSessionService) *ClassSessionHandler {
	return &ClassSessionHandler{service: service}
}

type openAulaRequest struct {
	SubjectID  string `json:"subject_id"`
	ExerciseID string `json:"exercise_id"`
	// DurationMinutes es opcional: sin valor, el aula dura una clase de hora y
	// media. Al vencer se cierra sola, así no bloquea la materia para siempre.
	DurationMinutes *int `json:"duration_minutes"`
}

// OpenAula abre el aula de una materia. Responde con el aula creada para que la
// pantalla no tenga que volver a preguntar cuál quedó abierta.
func (h *ClassSessionHandler) OpenAula(w http.ResponseWriter, r *http.Request) {
	if role := requestRole(r); role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden abrir un aula")
		return
	}

	var req openAulaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición JSON inválido")
		return
	}
	if strings.TrimSpace(req.SubjectID) == "" {
		SendError(w, http.StatusBadRequest, "Missing subject_id", "El campo subject_id es obligatorio para abrir un aula")
		return
	}

	var exerciseID *string
	if trimmed := strings.TrimSpace(req.ExerciseID); trimmed != "" {
		exerciseID = &trimmed
	}

	session, err := h.service.OpenAula(r.Context(), getTenantFromCtx(r), resolveRequestStudentID(r), requestRole(r), req.SubjectID, exerciseID, req.DurationMinutes)
	if err != nil {
		h.writeSessionError(w, err, "Error al abrir el aula")
		return
	}

	SendJSON(w, http.StatusCreated, session, "Aula abierta")
}

// closeAulaResponse conserva los campos de la sesión y suma el resultado del
// congelamiento. Los clientes que ya leen la sesión ignoran el campo nuevo.
type closeAulaResponse struct {
	*domain.ClassSession
	FreezeReport *services.FreezeReport `json:"freeze_report,omitempty"`
}

// CloseAula cierra el aula de la materia. Cerrar dos veces responde lo mismo.
func (h *ClassSessionHandler) CloseAula(w http.ResponseWriter, r *http.Request) {
	if role := requestRole(r); role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden cerrar un aula")
		return
	}

	sessionID := r.PathValue("id")
	session, freezeReport, err := h.service.CloseAula(r.Context(), getTenantFromCtx(r), resolveRequestStudentID(r), requestRole(r), sessionID)
	if err != nil {
		h.writeSessionError(w, err, "Error al cerrar el aula")
		return
	}

	response := closeAulaResponse{ClassSession: session}
	if freezeReport.Frozen+freezeReport.Skipped+freezeReport.Failed > 0 {
		response.FreezeReport = &freezeReport
	}
	SendJSON(w, http.StatusOK, response, "Aula cerrada")
}

type unfreezeExamRequest struct {
	// Reason explica por qué se reabre un volumen ya cerrado. Queda en la
	// auditoría junto al docente y el alumno.
	Reason string `json:"reason"`
}

// FreezeExam congela los volúmenes de un examen ya cerrado. Es idempotente y
// responde el conteo real: cuántos quedaron en solo lectura, cuántos ya estaban
// y cuántos fallaron.
func (h *ClassSessionHandler) FreezeExam(w http.ResponseWriter, r *http.Request) {
	if role := requestRole(r); role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden cerrar un examen")
		return
	}

	report, err := h.service.FreezeExam(r.Context(), getTenantFromCtx(r), resolveRequestStudentID(r), requestRole(r), r.PathValue("id"))
	if err != nil {
		h.writeSessionError(w, err, "Error al congelar el examen")
		return
	}
	SendJSON(w, http.StatusOK, report, "Examen congelado")
}

// UnfreezeExamStudent es la excepción individual: devuelve la escritura al
// laboratorio de un alumno con un motivo que queda registrado.
func (h *ClassSessionHandler) UnfreezeExamStudent(w http.ResponseWriter, r *http.Request) {
	if role := requestRole(r); role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden reabrir un entorno")
		return
	}

	var req unfreezeExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición JSON inválido")
		return
	}

	report, err := h.service.UnfreezeExamStudent(r.Context(), getTenantFromCtx(r), resolveRequestStudentID(r), requestRole(r), r.PathValue("id"), r.PathValue("studentID"), req.Reason)
	if err != nil {
		h.writeSessionError(w, err, "Error al reabrir el entorno del alumno")
		return
	}
	SendJSON(w, http.StatusOK, report, "Entorno reabierto")
}

// GetCurrentAula devuelve el aula abierta del docente. Sin aula, data es nulo:
// la pantalla muestra el estado vacío sin tratar el caso como un error.
func (h *ClassSessionHandler) GetCurrentAula(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.CurrentAula(r.Context(), getTenantFromCtx(r), resolveRequestStudentID(r), requestRole(r))
	if err != nil {
		h.writeSessionError(w, err, "Error al obtener el aula abierta")
		return
	}
	if session == nil {
		SendJSON(w, http.StatusOK, nil, "No hay aula abierta")
		return
	}
	SendJSON(w, http.StatusOK, session, "Aula abierta obtenida")
}

// GetBoard devuelve el tablero del aula: marcador y alumnos ordenados por
// urgencia docente.
func (h *ClassSessionHandler) GetBoard(w http.ResponseWriter, r *http.Request) {
	board, err := h.service.Board(r.Context(), getTenantFromCtx(r), resolveRequestStudentID(r), requestRole(r), r.PathValue("id"))
	if err != nil {
		h.writeSessionError(w, err, "Error al obtener el tablero del aula")
		return
	}
	SendJSON(w, http.StatusOK, board, "Tablero del aula obtenido")
}

// GetActiveAulas lista las aulas abiertas del alumno autenticado. Es el sondeo
// que le dice al cliente si tiene que emitir su presencia.
func (h *ClassSessionHandler) GetActiveAulas(w http.ResponseWriter, r *http.Request) {
	studentID := resolveRequestStudentID(r)
	if strings.TrimSpace(studentID) == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	aulas, err := h.service.StudentAulas(r.Context(), getTenantFromCtx(r), studentID)
	if err != nil {
		h.writeSessionError(w, err, "Error al obtener las aulas abiertas")
		return
	}
	SendJSON(w, http.StatusOK, aulas, "Aulas abiertas obtenidas")
}

// PostHeartbeat recibe la señal de presencia del alumno en el aula abierta.
func (h *ClassSessionHandler) PostHeartbeat(w http.ResponseWriter, r *http.Request) {
	studentID := resolveRequestStudentID(r)
	if strings.TrimSpace(studentID) == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	if err := h.service.Heartbeat(r.Context(), getTenantFromCtx(r), studentID, r.PathValue("id")); err != nil {
		h.writeSessionError(w, err, "Error al registrar la presencia en el aula")
		return
	}
	SendJSON(w, http.StatusOK, map[string]bool{"ok": true}, "Presencia registrada")
}

// requestRole lee el rol de la sesión con el mismo fallback que el resto de los
// handlers, para que las pruebas y los proxies que solo copian cabeceras sigan
// funcionando.
func requestRole(r *http.Request) string {
	if role := strings.TrimSpace(domain.GetUserRole(r.Context())); role != "" {
		return role
	}
	return strings.TrimSpace(r.Header.Get("X-User-Role"))
}

func (h *ClassSessionHandler) writeSessionError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, domain.ErrInvalidTenant):
		SendError(w, http.StatusUnauthorized, err.Error(), "No se pudo resolver la institución de la sesión")
	case errors.Is(err, domain.ErrSessionAlreadyOpen):
		SendError(w, http.StatusConflict, err.Error(), "Ya hay un aula abierta en esta materia")
	case errors.Is(err, domain.ErrSessionNotOpen):
		SendError(w, http.StatusConflict, err.Error(), "El aula ya está cerrada")
	case errors.Is(err, domain.ErrSessionNotFound):
		SendError(w, http.StatusNotFound, err.Error(), "El aula no existe o no pertenece al docente")
	case errors.Is(err, domain.ErrStudentNotEnrolled):
		SendError(w, http.StatusForbidden, err.Error(), "El alumno no está inscrito en la materia del aula")
	case errors.Is(err, domain.ErrSessionDurationInvalid):
		SendError(w, http.StatusBadRequest, err.Error(), "La duración del aula está fuera del rango permitido")
	case errors.Is(err, domain.ErrExamFreezeNotAllowed):
		SendError(w, http.StatusConflict, err.Error(), "Solo se congelan los exámenes del docente")
	case errors.Is(err, domain.ErrExamSessionOpen):
		SendError(w, http.StatusConflict, err.Error(), "El examen sigue abierto")
	case errors.Is(err, domain.ErrFreezeReasonRequired):
		SendError(w, http.StatusBadRequest, err.Error(), "Escribí el motivo para reabrir el entorno")
	case errors.Is(err, domain.ErrFreezeStudentNotInSession):
		SendError(w, http.StatusNotFound, err.Error(), "El alumno no participó de ese examen")
	case errors.Is(err, domain.ErrNotFound):
		SendError(w, http.StatusBadRequest, err.Error(), "La materia o la práctica indicada no corresponden")
	default:
		SendError(w, http.StatusInternalServerError, err.Error(), message)
	}
}
