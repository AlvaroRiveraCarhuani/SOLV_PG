package httpdelivery

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// ClassAdvanceHandler expone el avance voluntario del docente del lado del
// alumno: qué tiene pendiente, aplicarlo con respaldo y descartarlo.
type ClassAdvanceHandler struct {
	service *services.ClassAdvanceService
}

func NewClassAdvanceHandler(service *services.ClassAdvanceService) *ClassAdvanceHandler {
	return &ClassAdvanceHandler{service: service}
}

type emitAdvanceRequest struct {
	ExerciseID string   `json:"exercise_id"`
	Scope      string   `json:"scope"`
	StudentIDs []string `json:"student_ids"`
}

// PostAdvance emite un avance: empaqueta el código del entorno de clase del
// docente, lo guarda como snapshot y lo deja como aviso para los destinatarios.
func (h *ClassAdvanceHandler) PostAdvance(w http.ResponseWriter, r *http.Request) {
	if role := requestRole(r); role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores emiten avances")
		return
	}

	var req emitAdvanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición JSON inválido")
		return
	}
	if strings.TrimSpace(req.ExerciseID) == "" {
		SendError(w, http.StatusBadRequest, "Missing exercise_id", "El campo exercise_id es obligatorio para emitir un avance")
		return
	}

	teacherID := resolveRequestStudentID(r)
	advance, recipients, err := h.service.Emit(r.Context(), services.EmitAdvanceInput{
		TenantID:   getTenantFromCtx(r),
		TeacherID:  teacherID,
		Role:       requestRole(r),
		SessionID:  r.PathValue("id"),
		ExerciseID: req.ExerciseID,
		Scope:      req.Scope,
		StudentIDs: req.StudentIDs,
	})
	if err != nil {
		h.writeAdvanceError(w, err, "Error al emitir el avance")
		return
	}

	SendJSON(w, http.StatusCreated, map[string]any{
		"advance":    advance,
		"recipients": recipients,
	}, "Avance emitido")
}

// GetPendingAdvances devuelve los avisos pendientes del alumno autenticado. Es
// la consulta periódica del laboratorio; sin avisos devuelve una lista vacía.
func (h *ClassAdvanceHandler) GetPendingAdvances(w http.ResponseWriter, r *http.Request) {
	studentID := resolveRequestStudentID(r)
	if strings.TrimSpace(studentID) == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	advances, err := h.service.PendingAdvances(r.Context(), getTenantFromCtx(r), studentID)
	if err != nil {
		h.writeAdvanceError(w, err, "Error al obtener los avances pendientes")
		return
	}
	SendJSON(w, http.StatusOK, advances, "Avances pendientes obtenidos")
}

// GetSessionAdvances devuelve las emisiones del aula con el estado de cada
// destinatario: quién aplicó, quién descartó y quién sigue pendiente.
func (h *ClassAdvanceHandler) GetSessionAdvances(w http.ResponseWriter, r *http.Request) {
	if role := requestRole(r); role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores ven el estado de los avances")
		return
	}

	advances, err := h.service.SessionAdvances(
		r.Context(),
		getTenantFromCtx(r),
		resolveRequestStudentID(r),
		requestRole(r),
		r.PathValue("id"),
	)
	if err != nil {
		h.writeAdvanceError(w, err, "Error al leer el estado de los avances")
		return
	}
	SendJSON(w, http.StatusOK, advances, "Avances del aula obtenidos")
}

// GetSessionHistory devuelve el historial del aula: quién emitió, congeló,
// descongeló, ajustó la ventana, aplicó o descartó un avance. El alumno no lo ve.
func (h *ClassAdvanceHandler) GetSessionHistory(w http.ResponseWriter, r *http.Request) {
	if role := requestRole(r); role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores ven el historial de la clase")
		return
	}

	events, err := h.service.SessionHistory(
		r.Context(),
		getTenantFromCtx(r),
		resolveRequestStudentID(r),
		requestRole(r),
		r.PathValue("id"),
	)
	if err != nil {
		h.writeAdvanceError(w, err, "Error al leer el historial de la clase")
		return
	}
	SendJSON(w, http.StatusOK, events, "Historial de la clase obtenido")
}

// PreviewAdvance devuelve los archivos del aviso separados en los que se
// agregan y los que se reemplazan. Es la confirmación previa: no escribe nada.
func (h *ClassAdvanceHandler) PreviewAdvance(w http.ResponseWriter, r *http.Request) {
	studentID := resolveRequestStudentID(r)
	if strings.TrimSpace(studentID) == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	preview, err := h.service.Preview(r.Context(), getTenantFromCtx(r), studentID, r.PathValue("id"))
	if err != nil {
		h.writeAdvanceError(w, err, "Error al leer los archivos del avance")
		return
	}
	SendJSON(w, http.StatusOK, preview, "Archivos del avance obtenidos")
}

// applyAdvanceRequest es el checklist del editor. Sin cuerpo o con la lista
// vacía se aplica el avance completo, que es el contrato anterior.
type applyAdvanceRequest struct {
	SelectedFiles []string `json:"selected_files"`
}

// ApplyAdvance copia el avance al laboratorio del alumno con respaldo previo y
// sin pausar el contenedor. Con selected_files solo toca los archivos elegidos.
func (h *ClassAdvanceHandler) ApplyAdvance(w http.ResponseWriter, r *http.Request) {
	studentID := resolveRequestStudentID(r)
	if strings.TrimSpace(studentID) == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	var req applyAdvanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "El cuerpo debe indicar selected_files como lista de rutas")
		return
	}

	result, err := h.service.ApplySelection(r.Context(), getTenantFromCtx(r), studentID, r.PathValue("id"), req.SelectedFiles)
	if err != nil {
		h.writeAdvanceError(w, err, "Error al aplicar el avance")
		return
	}
	SendJSON(w, http.StatusOK, result, "Avance aplicado")
}

// RollbackAdvance deshace el último avance aplicado desde el respaldo del
// propio avance y deja el aviso pendiente otra vez.
func (h *ClassAdvanceHandler) RollbackAdvance(w http.ResponseWriter, r *http.Request) {
	studentID := resolveRequestStudentID(r)
	if strings.TrimSpace(studentID) == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	result, err := h.service.Rollback(r.Context(), getTenantFromCtx(r), studentID, r.PathValue("id"))
	if err != nil {
		h.writeAdvanceError(w, err, "Error al deshacer el avance")
		return
	}
	SendJSON(w, http.StatusOK, result, "Avance deshecho")
}

// DismissAdvance descarta el aviso sin tocar el entorno del alumno.
func (h *ClassAdvanceHandler) DismissAdvance(w http.ResponseWriter, r *http.Request) {
	studentID := resolveRequestStudentID(r)
	if strings.TrimSpace(studentID) == "" {
		SendError(w, http.StatusUnauthorized, "Missing student identity", "No se pudo resolver la identidad del alumno desde la sesión")
		return
	}

	if err := h.service.Dismiss(r.Context(), getTenantFromCtx(r), studentID, r.PathValue("id")); err != nil {
		h.writeAdvanceError(w, err, "Error al descartar el avance")
		return
	}
	SendJSON(w, http.StatusOK, map[string]bool{"ok": true}, "Avance descartado")
}

// FreezeSession congela la clase: bloquea las emisiones nuevas y deja intacto el
// trabajo del alumno. Descongelar recorre el mismo camino.
func (h *ClassAdvanceHandler) FreezeSession(w http.ResponseWriter, r *http.Request) {
	h.setFrozen(w, r, true)
}

// UnfreezeSession descongela la clase.
func (h *ClassAdvanceHandler) UnfreezeSession(w http.ResponseWriter, r *http.Request) {
	h.setFrozen(w, r, false)
}

func (h *ClassAdvanceHandler) setFrozen(w http.ResponseWriter, r *http.Request, frozen bool) {
	if role := requestRole(r); role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores congelan la clase")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := resolveRequestStudentID(r)
	role := requestRole(r)
	sessionID := r.PathValue("id")

	var (
		session *domain.ClassSession
		err     error
	)
	if frozen {
		session, err = h.service.Freeze(r.Context(), tenantID, teacherID, role, sessionID)
	} else {
		session, err = h.service.Unfreeze(r.Context(), tenantID, teacherID, role, sessionID)
	}
	if err != nil {
		h.writeAdvanceError(w, err, "Error al cambiar la congelación de la clase")
		return
	}

	if frozen {
		SendJSON(w, http.StatusOK, session, "Clase congelada")
		return
	}
	SendJSON(w, http.StatusOK, session, "Clase descongelada")
}

type graceAdvanceRequest struct {
	AddMinutes *int       `json:"add_minutes"`
	Until      *time.Time `json:"until"`
	Close      bool       `json:"close"`
}

// ExtendGrace ajusta la ventana de disponibilidad de una clase cerrada: suma
// minutos, fija una fecha límite para una tarea o la cierra en el momento.
// Solo aplica con el aula cerrada; con la clase en curso responde 409.
func (h *ClassAdvanceHandler) ExtendGrace(w http.ResponseWriter, r *http.Request) {
	if role := requestRole(r); role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores ajustan la ventana")
		return
	}

	var req graceAdvanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición JSON inválido")
		return
	}
	actions := 0
	if req.AddMinutes != nil {
		actions++
	}
	if req.Until != nil {
		actions++
	}
	if req.Close {
		actions++
	}
	if actions != 1 {
		SendError(w, http.StatusBadRequest, "Missing window action", "Indicá una sola acción: add_minutes, until o close")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := resolveRequestStudentID(r)
	role := requestRole(r)
	sessionID := r.PathValue("id")

	var (
		session *domain.ClassSession
		err     error
	)
	switch {
	case req.Close:
		session, err = h.service.CloseGrace(r.Context(), tenantID, teacherID, role, sessionID)
	case req.AddMinutes != nil:
		session, err = h.service.AddGrace(r.Context(), tenantID, teacherID, role, sessionID, *req.AddMinutes)
	default:
		session, err = h.service.SetGraceDeadline(r.Context(), tenantID, teacherID, role, sessionID, *req.Until)
	}
	if err != nil {
		h.writeAdvanceError(w, err, "Error al ajustar la ventana de disponibilidad")
		return
	}
	SendJSON(w, http.StatusOK, session, "Ventana de disponibilidad actualizada")
}

// writeAdvanceError traduce las reglas del aula: un avance ajeno no se revela
// (404) y un aviso que ya no se puede aplicar responde 409 con su motivo.
func (h *ClassAdvanceHandler) writeAdvanceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, domain.ErrInvalidTenant):
		SendError(w, http.StatusUnauthorized, err.Error(), "No se pudo resolver la institución de la sesión")
	case errors.Is(err, domain.ErrInvalidResourceID):
		SendError(w, http.StatusBadRequest, err.Error(), "El identificador indicado no tiene un formato válido")
	case errors.Is(err, domain.ErrAdvanceNotFound):
		SendError(w, http.StatusNotFound, err.Error(), "El avance no existe o no está dirigido a este alumno")
	case errors.Is(err, domain.ErrSessionNotFound):
		SendError(w, http.StatusNotFound, err.Error(), "El aula no existe o no pertenece al docente")
	case errors.Is(err, domain.ErrNotFound):
		SendError(w, http.StatusNotFound, err.Error(), "La práctica indicada no existe en esta institución")
	case errors.Is(err, domain.ErrAdvanceSessionClosed):
		SendError(w, http.StatusConflict, err.Error(), "El aula no está abierta: no se emiten avances")
	case errors.Is(err, domain.ErrAdvanceFrozen):
		SendError(w, http.StatusConflict, err.Error(), "La clase está congelada: descongélala para emitir")
	case errors.Is(err, domain.ErrAdvanceSourceMissing):
		SendError(w, http.StatusConflict, err.Error(), "Abre tu entorno de clase y escribe el código antes de emitir")
	case errors.Is(err, domain.ErrAdvanceScopeInvalid):
		SendError(w, http.StatusUnprocessableEntity, err.Error(), "Elige toda la clase o al menos un alumno")
	case errors.Is(err, domain.ErrAdvanceNoRecipients):
		SendError(w, http.StatusUnprocessableEntity, err.Error(), "Todavía no hay alumnos en el aula")
	case errors.Is(err, domain.ErrAdvanceRecipientNotInSession):
		SendError(w, http.StatusUnprocessableEntity, err.Error(), "Algún alumno elegido no está en el aula")
	case errors.Is(err, domain.ErrAdvanceNotPending):
		SendError(w, http.StatusConflict, err.Error(), "El avance ya fue aplicado o descartado")
	case errors.Is(err, domain.ErrAdvanceSelectionInvalid):
		SendError(w, http.StatusUnprocessableEntity, err.Error(), "Los archivos elegidos no pertenecen al avance")
	case errors.Is(err, domain.ErrAdvanceRollbackNotApplied):
		SendError(w, http.StatusConflict, err.Error(), "Solo se deshace un avance que ya aplicaste")
	case errors.Is(err, domain.ErrAdvanceBackupMissing):
		SendError(w, http.StatusConflict, err.Error(), "El avance no reemplazó archivos: no hay versión anterior para restaurar")
	case errors.Is(err, domain.ErrAdvanceExamBlocked):
		SendError(w, http.StatusConflict, err.Error(), "En examen no se aplican avances")
	case errors.Is(err, domain.ErrAdvanceGraceExpired):
		SendError(w, http.StatusConflict, err.Error(), "La ventana para aplicar el avance venció")
	case errors.Is(err, domain.ErrAdvanceGraceInvalid):
		SendError(w, http.StatusBadRequest, err.Error(), "La ventana debe estar entre 1 y 480 minutos")
	case errors.Is(err, domain.ErrAdvanceGraceExceeded):
		SendError(w, http.StatusUnprocessableEntity, err.Error(), "La ventana no puede superar los 7 días desde el cierre")
	case errors.Is(err, domain.ErrAdvanceGraceDeadline):
		SendError(w, http.StatusUnprocessableEntity, err.Error(), "La fecha límite debe ser futura y no superar los 7 días desde el cierre")
	case errors.Is(err, domain.ErrAdvanceNotCloseable):
		SendError(w, http.StatusConflict, err.Error(), "Cerrá el aula para extender la ventana de rezago")
	case errors.Is(err, domain.ErrAdvanceWorkspaceStopped):
		SendError(w, http.StatusConflict, err.Error(), "Abre tu laboratorio para aplicar el avance")
	case errors.Is(err, domain.ErrAdvanceSnapshotTooLarge):
		SendError(w, http.StatusUnprocessableEntity, err.Error(), "El avance supera el tamaño permitido")
	case errors.Is(err, services.ErrAdvanceStorageUnavailable):
		SendError(w, http.StatusServiceUnavailable, err.Error(), message)
	default:
		SendError(w, http.StatusInternalServerError, err.Error(), message)
	}
}
