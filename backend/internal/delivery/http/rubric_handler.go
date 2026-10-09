package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
)

type RubricHandler struct {
	rubricService domain.RubricService
}

func NewRubricHandler(rubricService domain.RubricService) *RubricHandler {
	return &RubricHandler{rubricService: rubricService}
}

func (h *RubricHandler) GetRubric(w http.ResponseWriter, r *http.Request) {
	exerciseIDStr := r.PathValue("id")
	if exerciseIDStr == "" {
		SendError(w, http.StatusBadRequest, "exercise ID is required", "ID del ejercicio requerido")
		return
	}

	exerciseID, err := uuid.Parse(exerciseIDStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid exercise ID format", "Formato de ID del ejercicio inválido")
		return
	}

	rubric, err := h.rubricService.GetByExerciseID(r.Context(), exerciseID)
	if err != nil {
		if errors.Is(err, domain.ErrRubricNotFound) {
			SendError(w, http.StatusNotFound, "rubric not found", "No se encontró rúbrica para este ejercicio")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al obtener la rúbrica")
		return
	}

	SendJSON(w, http.StatusOK, rubric, "Rúbrica obtenida exitosamente")
}

func (h *RubricHandler) SaveRubric(w http.ResponseWriter, r *http.Request) {
	exerciseIDStr := r.PathValue("id")
	if exerciseIDStr == "" {
		SendError(w, http.StatusBadRequest, "exercise ID is required", "ID del ejercicio requerido")
		return
	}

	exerciseID, err := uuid.Parse(exerciseIDStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid exercise ID format", "Formato de ID del ejercicio inválido")
		return
	}

	var rubric domain.Rubric
	if err := json.NewDecoder(r.Body).Decode(&rubric); err != nil {
		SendError(w, http.StatusBadRequest, "invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	rubric.ExerciseID = exerciseID

	if err := h.rubricService.CreateOrUpdate(r.Context(), &rubric); err != nil {
		if errors.Is(err, domain.ErrRubricNotAllowedForJudge) {
			SendError(w, http.StatusUnprocessableEntity, err.Error(), "Las rúbricas solo están permitidas para ejercicios de IDE Persistente")
			return
		}
		if errors.Is(err, domain.ErrRubricInvalidWeights) ||
			errors.Is(err, domain.ErrRubricInvalidLevels) ||
			errors.Is(err, domain.ErrRubricInvalidScore) ||
			errors.Is(err, domain.ErrRubricDuplicateLevelName) {
			SendError(w, http.StatusBadRequest, err.Error(), err.Error())
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al guardar la rúbrica")
		return
	}

	SendJSON(w, http.StatusOK, rubric, "Rúbrica guardada exitosamente")
}

func (h *RubricHandler) EvaluateSubmission(w http.ResponseWriter, r *http.Request) {
	submissionIDStr := r.PathValue("id")
	if submissionIDStr == "" {
		SendError(w, http.StatusBadRequest, "submission ID is required", "ID de la entrega requerido")
		return
	}

	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid submission ID format", "Formato de ID de entrega inválido")
		return
	}

	var req domain.RubricEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	res, err := h.rubricService.EvaluateSubmission(r.Context(), submissionID, req)
	if err != nil {
		if errors.Is(err, domain.ErrRubricSelectionMissing) {
			SendError(w, http.StatusBadRequest, err.Error(), "Falta seleccionar una opción para cada criterio de la rúbrica")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al evaluar la entrega por rúbrica")
		return
	}

	SendJSON(w, http.StatusOK, res, "Evaluación por rúbrica registrada exitosamente")
}
