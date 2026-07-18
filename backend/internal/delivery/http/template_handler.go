package httpdelivery

import (
	"encoding/json"
	"net/http"

	"solv-backend/internal/domain"

	"github.com/go-playground/validator/v10"
)

type TemplateHandler struct {
	repo     domain.TemplateRepository
	validate *validator.Validate
}

func NewTemplateHandler(repo domain.TemplateRepository, validate *validator.Validate) *TemplateHandler {
	return &TemplateHandler{repo: repo, validate: validate}
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto domain.CreateTemplateDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON body", "Cuerpo de la petición inválido")
		return
	}

	if err := h.validate.Struct(dto); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Campos obligatorios faltantes o inválidos")
		return
	}

	id, err := h.repo.CreateTemplate(r.Context(), dto)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "No se pudo crear la plantilla")
		return
	}

	SendJSON(w, http.StatusCreated, map[string]string{"id": id}, "Plantilla creada exitosamente")
}

func (h *TemplateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	templates, err := h.repo.GetAllTemplates(r.Context())
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "No se pudieron obtener las plantillas")
		return
	}

	SendJSON(w, http.StatusOK, templates, "Plantillas obtenidas exitosamente")
}
