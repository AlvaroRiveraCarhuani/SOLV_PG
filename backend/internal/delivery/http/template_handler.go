package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strconv"

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

type PaginationMeta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type PaginatedResponse struct {
	Data interface{}    `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

func (h *TemplateHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	templates, err := h.repo.GetAllTemplates(r.Context())
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "No se pudieron obtener las plantillas")
		return
	}

	total := len(templates)

	response := PaginatedResponse{
		Data: templates,
		Meta: PaginationMeta{
			Total: total,
			Page:  page,
			Limit: limit,
		},
	}

	SendJSON(w, http.StatusOK, response, "Plantillas obtenidas exitosamente")
}
