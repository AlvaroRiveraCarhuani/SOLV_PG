package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"solv-backend/internal/domain"

	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	repo     domain.UserRepository
	validate *validator.Validate
}

func NewUserHandler(repo domain.UserRepository, validate *validator.Validate) *UserHandler {
	return &UserHandler{repo: repo, validate: validate}
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto domain.CreateUserDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON body", "Cuerpo de la petición inválido")
		return
	}

	if err := h.validate.Struct(dto); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Campos obligatorios faltantes o inválidos")
		return
	}

	id, err := h.repo.CreateUser(r.Context(), dto)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "already exists") {
			status = http.StatusConflict
		}
		SendError(w, status, err.Error(), "No se pudo crear el usuario")
		return
	}

	SendJSON(w, http.StatusCreated, map[string]string{"id": id}, "Usuario creado exitosamente")
}
