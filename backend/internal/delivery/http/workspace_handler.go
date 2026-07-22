package httpdelivery

import (
	"encoding/json"
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
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al iniciar el entorno de desarrollo interactivo")
		return
	}

	SendJSON(w, http.StatusOK, instance, "Entorno de desarrollo interactivo iniciado exitosamente")
}
