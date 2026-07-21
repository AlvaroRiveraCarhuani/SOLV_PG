package httpdelivery

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"solv-backend/internal/core/services"
)

type LabHandler struct {
	service  *services.LabService
	validate *validator.Validate
}

func NewLabHandler(service *services.LabService, validate *validator.Validate) *LabHandler {
	return &LabHandler{service: service, validate: validate}
}

type StartLabRequest struct {
	UserID     string `json:"user_id" validate:"required"`
	TemplateID string `json:"template_id" validate:"required"`
	RAMLimitMB int    `json:"ram_limit_mb"`
	UserEmail  string `json:"user_email" validate:"required,email"`
}

func (h *LabHandler) Start(w http.ResponseWriter, r *http.Request) {
	var req StartLabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON body", "Cuerpo de la petición inválido")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Campos obligatorios faltantes o inválidos")
		return
	}

	dto, err := h.service.StartLab(r.Context(), req.UserID, req.TemplateID, req.RAMLimitMB, req.UserEmail)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "No se pudo iniciar el laboratorio")
		return
	}

	SendJSON(w, http.StatusOK, dto, "Laboratorio iniciado exitosamente")
}
