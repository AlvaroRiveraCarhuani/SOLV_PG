package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// StartDryRun maneja POST /api/v1/exercises/{id}/dry-run -> 202 Accepted
func (h *EvaluationHandler) StartDryRun(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "FORBIDDEN", "No tiene permisos para ejecutar comprobación previa")
		return
	}

	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "MISSING_EXERCISE_ID", "ID de ejercicio faltante")
		return
	}

	tenantID := getTenantFromCtx(r)

	job, err := h.service.StartDryRun(r.Context(), exerciseID, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "NOT_FOUND", "Ejercicio no encontrado")
			return
		}
		if errors.Is(err, domain.ErrMissingReferenceSolution) {
			SendError(w, http.StatusBadRequest, "REFERENCE_REQUIRED", "Debe configurar una solución de referencia antes de iniciar el dry-run")
			return
		}
		if errors.Is(err, domain.ErrRamExceedsHostCapacity) {
			SendError(w, http.StatusBadRequest, "RAM_EXCEEDS_HOST", "La memoria configurada excede la capacidad estructural del host")
			return
		}
		SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error al iniciar trabajo de dry-run: "+err.Error())
		return
	}

	SendJSON(w, http.StatusAccepted, job, "Trabajo de dry-run iniciado exitosamente")
}

// GetDryRunJob maneja GET /api/v1/exercises/{id}/dry-run/jobs/{jobId}
func (h *EvaluationHandler) GetDryRunJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("jobId")
	if jobID == "" {
		jobID = r.PathValue("id")
	}
	if jobID == "" {
		SendError(w, http.StatusBadRequest, "MISSING_JOB_ID", "ID de trabajo faltante")
		return
	}

	job, err := h.service.GetDryRunJob(r.Context(), jobID)
	if err != nil {
		SendError(w, http.StatusNotFound, "NOT_FOUND", "Trabajo de dry-run no encontrado")
		return
	}

	SendJSON(w, http.StatusOK, job, "Estado de trabajo dry-run obtenido exitosamente")
}

func (h *EvaluationHandler) CalculateOutputs(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "Unauthorized: teacher role required", "No tiene permisos para calcular salidas de prueba")
		return
	}

	var req services.CalculateOutputsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	resp, err := h.service.CalculateOutputs(r.Context(), req)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al calcular salidas con la solución de referencia")
		return
	}

	SendJSON(w, http.StatusOK, resp, "Salidas calculadas exitosamente")
}
