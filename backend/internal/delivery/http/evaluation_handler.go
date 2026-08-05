package httpdelivery

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/dto"
)

type EvaluationHandler struct {
	service  *services.EvaluationService
	validate *validator.Validate
}

func NewEvaluationHandler(service *services.EvaluationService, validate *validator.Validate) *EvaluationHandler {
	return &EvaluationHandler{
		service:  service,
		validate: validate,
	}
}

type EvaluationRequest struct {
	ExerciseID    string `json:"exercise_id" validate:"required"`
	Language      string `json:"language" validate:"required"`
	SourceCodeB64 string `json:"source_code_b64" validate:"required"`
}

func (h *EvaluationHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var req EvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Campos obligatorios faltantes o inválidos")
		return
	}

	result, err := h.service.Evaluate(r.Context(), req.ExerciseID, req.Language, req.SourceCodeB64)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al procesar la evaluación")
		return
	}

	SendJSON(w, http.StatusOK, result, "Evaluación procesada exitosamente")
}

func (h *EvaluationHandler) GetExerciseByID(w http.ResponseWriter, r *http.Request) {
	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "Exercise ID missing", "ID de ejercicio faltante")
		return
	}

	exercise, err := h.service.GetExerciseByID(r.Context(), exerciseID)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error(), "Ejercicio no encontrado")
		return
	}

	userRole := r.Header.Get("X-User-Role")
	if userRole == "" {
		userRole = "student"
	}

	if userRole == "teacher" || userRole == "admin" {
		SendJSON(w, http.StatusOK, exercise, "Ejercicio obtenido exitosamente")
		return
	}

	// Rol student: DTO público sin test_cases
	publicResp := dto.ToExercisePublicResponse(exercise)
	SendJSON(w, http.StatusOK, publicResp, "Ejercicio obtenido exitosamente")
}
