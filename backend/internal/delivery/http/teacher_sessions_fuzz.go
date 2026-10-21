package httpdelivery

import (
	"encoding/json"
	"net/http"

	"solv-backend/internal/core/domain"
)

func (h *TeacherHandler) GetLiveSessions(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")

	sessions, err := h.service.ListLiveSessions(r.Context(), tenantID, teacherID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al listar sesiones de estudiantes en vivo")
		return
	}

	SendJSON(w, http.StatusOK, sessions, "Sesiones en vivo obtenidas exitosamente")
}

func (h *TeacherHandler) GenerateFuzzCases(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	var req domain.FuzzGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON body", "Cuerpo de solicitud inválido")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")

	report, err := h.service.GenerateFuzzCases(r.Context(), tenantID, teacherID, req)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al generar casos de prueba")
		return
	}

	SendJSON(w, http.StatusOK, report, "Casos de prueba sintetizados exitosamente")
}

func (h *TeacherHandler) ApplyFuzzCases(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get("X-User-Role")
	if role == "student" {
		SendError(w, http.StatusForbidden, "Forbidden", "Acceso denegado: solo docentes y administradores pueden acceder a este recurso")
		return
	}

	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "Missing exercise ID", "El identificador del ejercicio es requerido")
		return
	}

	var req domain.ApplyFuzzCasesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON body", "Cuerpo de solicitud inválido")
		return
	}

	tenantID := getTenantFromCtx(r)
	teacherID := r.Header.Get("X-User-Id")

	addedCount, err := h.service.ApplyFuzzCases(r.Context(), tenantID, teacherID, exerciseID, req.Cases)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al aplicar casos de prueba")
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"added_count": addedCount,
		"exercise_id": exerciseID,
	}, "Casos de prueba incorporados exitosamente al ejercicio")
}
