package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

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
	StudentID string `json:"student_id" validate:"required"`
	SubjectID string `json:"subject_id" validate:"required"`
}

func (h *WorkspaceHandler) StartWorkspace(w http.ResponseWriter, r *http.Request) {
	var req StartWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición JSON inválido")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Los campos student_id y subject_id son obligatorios")
		return
	}

	instance, err := h.service.StartWorkspace(r.Context(), req.StudentID, req.SubjectID)
	if err != nil {
		if errors.Is(err, services.ErrHostMemoryExhausted) {
			SendError(w, http.StatusServiceUnavailable, err.Error(), "Servidor saturado: La memoria RAM del host cayó por debajo del 15% de margen de seguridad. Intente más tarde.")
			return
		}
		if errors.Is(err, services.ErrOOMKilledCooldownPenalty) {
			SendError(w, http.StatusTooManyRequests, err.Error(), "Memoria Excedida: Has superado la cuota de RAM (OOMKilled) 3 veces seguidas. Espera 5 minutos antes de reiniciar.")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al iniciar el entorno de desarrollo interactivo")
		return
	}

	SendJSON(w, http.StatusOK, instance, "Entorno de desarrollo interactivo iniciado exitosamente")
}

func (h *WorkspaceHandler) TerminateWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("id")
	if workspaceID == "" {
		SendError(w, http.StatusBadRequest, "Missing workspace ID", "Se requiere el ID del workspace")
		return
	}

	if err := h.service.TerminateWorkspace(r.Context(), workspaceID); err != nil {
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
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al reiniciar el entorno")
		return
	}

	SendJSON(w, http.StatusOK, instance, "Entorno reiniciado exitosamente")
}
