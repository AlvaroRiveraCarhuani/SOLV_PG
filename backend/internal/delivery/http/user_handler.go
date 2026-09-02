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

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		SendError(w, http.StatusUnauthorized, "Missing user ID in headers", "Usuario no autenticado")
		return
	}

	user, err := h.repo.GetUserByID(r.Context(), userID)
	if err != nil {
		SendError(w, http.StatusNotFound, "User not found", "Usuario no encontrado")
		return
	}

	fullName := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if fullName == "" {
		fullName = user.FirstName
	}

	resp := map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"full_name":  fullName,
		"role":       user.Role,
		"tenant_id":  user.TenantID,
	}

	SendJSON(w, http.StatusOK, resp, "Perfil de usuario obtenido exitosamente")
}
