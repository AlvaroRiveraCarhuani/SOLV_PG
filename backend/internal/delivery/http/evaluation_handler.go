package httpdelivery

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/dto"
)

type EvaluationHandler struct {
	service  *services.EvaluationService
	validate *validator.Validate
	wsHub    *WebSocketHub
}

func NewEvaluationHandler(service *services.EvaluationService, validate *validator.Validate) *EvaluationHandler {
	return &EvaluationHandler{
		service:  service,
		validate: validate,
	}
}

func (h *EvaluationHandler) SetWebSocketHub(hub *WebSocketHub) {
	h.wsHub = hub
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

	userID := r.Header.Get("X-User-Id")
	if h.wsHub != nil && userID != "" {
		h.wsHub.EmitToUser(userID, WebSocketMessage{
			Event: "EVALUATION_PROGRESS",
			Stage: "QUEUED",
			Data:  map[string]string{"exercise_id": req.ExerciseID, "language": req.Language},
		})
	}

	result, err := h.service.Evaluate(r.Context(), req.ExerciseID, req.Language, req.SourceCodeB64)
	if err != nil {
		if h.wsHub != nil && userID != "" {
			h.wsHub.EmitToUser(userID, WebSocketMessage{
				Event: "EVALUATION_PROGRESS",
				Stage: "ERROR",
				Data:  map[string]string{"error": err.Error()},
			})
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al procesar la evaluación")
		return
	}

	if h.wsHub != nil && userID != "" {
		h.wsHub.EmitToUser(userID, WebSocketMessage{
			Event: "EVALUATION_COMPLETED",
			Stage: "COMPLETED",
			Data:  result,
		})
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

func (h *EvaluationHandler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	var ex domain.Exercise
	if err := json.NewDecoder(r.Body).Decode(&ex); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	if ex.Title == "" {
		SendError(w, http.StatusBadRequest, "Title is required", "El título del ejercicio es obligatorio")
		return
	}
	if ex.Type == "" {
		ex.Type = domain.ExerciseTypeAlgorithm
	}
	ex.TenantID = tenantID

	if err := h.service.CreateExercise(r.Context(), &ex); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al crear el ejercicio")
		return
	}

	SendJSON(w, http.StatusCreated, ex, "Ejercicio creado exitosamente")
}

