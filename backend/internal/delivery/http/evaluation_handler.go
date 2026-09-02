package httpdelivery

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "Unauthorized: teacher role required", "No tiene permisos para crear ejercicios")
		return
	}

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
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
	if ex.Status == "" {
		ex.Status = "draft"
	}
	ex.TenantID = tenantID

	if err := h.service.CreateExercise(r.Context(), &ex); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al crear el ejercicio")
		return
	}

	SendJSON(w, http.StatusCreated, ex, "Ejercicio creado exitosamente")
}

func (h *EvaluationHandler) UpdateExercise(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "Unauthorized: teacher role required", "No tiene permisos para modificar ejercicios")
		return
	}

	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "Exercise ID is required", "ID de ejercicio faltante")
		return
	}

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	existing, err := h.service.GetExerciseByIDAndTenant(r.Context(), exerciseID, tenantID)
	if err != nil || existing == nil {
		SendError(w, http.StatusNotFound, "Exercise not found", "Ejercicio no encontrado en este tenant")
		return
	}

	var ex domain.Exercise
	if err := json.NewDecoder(r.Body).Decode(&ex); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	ex.ID = exerciseID
	ex.TenantID = tenantID
	if ex.Type == "" {
		ex.Type = existing.Type
	}

	if err := h.service.UpdateExercise(r.Context(), &ex); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al actualizar el ejercicio")
		return
	}

	SendJSON(w, http.StatusOK, ex, "Ejercicio actualizado exitosamente")
}

func (h *EvaluationHandler) BulkTestCases(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "Unauthorized: teacher role required", "No tiene permisos para gestionar casos de prueba")
		return
	}

	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "Exercise ID is required", "ID de ejercicio faltante")
		return
	}

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	ex, err := h.service.GetExerciseByIDAndTenant(r.Context(), exerciseID, tenantID)
	if err != nil || ex == nil {
		SendError(w, http.StatusNotFound, "Exercise not found", "Ejercicio no encontrado en este tenant")
		return
	}

	contentType := r.Header.Get("Content-Type")
	var testCases []domain.TestCase

	if strings.Contains(contentType, "text/csv") || strings.Contains(contentType, "application/csv") {
		reader := csv.NewReader(r.Body)
		reader.FieldsPerRecord = -1
		reader.LazyQuotes = true
		reader.TrimLeadingSpace = true

		records, err := reader.ReadAll()
		if err != nil {
			SendError(w, 422, fmt.Sprintf("CSV parse error: %v", err), "Error al procesar el archivo CSV")
			return
		}

		for idx, row := range records {
			lineNum := idx + 1
			if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
				continue
			}
			// Saltar cabecera si existe
			if idx == 0 && (strings.EqualFold(row[0], "input") || strings.EqualFold(row[0], "entrada")) {
				continue
			}
			if len(row) < 2 {
				SendError(w, 422, fmt.Sprintf("Malformed row at line %d: requires at least 2 columns (input, expected_output)", lineNum), fmt.Sprintf("Fila %d malformada en el archivo CSV", lineNum))
				return
			}

			isHidden := false
			if len(row) >= 3 {
				val := strings.ToLower(strings.TrimSpace(row[2]))
				if val == "true" || val == "1" || val == "yes" || val == "si" || val == "sí" {
					isHidden = true
				}
			}

			testCases = append(testCases, domain.TestCase{
				Input:          row[0],
				ExpectedOutput: row[1],
				IsHidden:       isHidden,
			})
		}
	} else {
		// Formato JSON
		if err := json.NewDecoder(r.Body).Decode(&testCases); err != nil {
			SendError(w, 422, fmt.Sprintf("Invalid JSON test cases: %v", err), "Formato de casos de prueba inválido")
			return
		}
	}

	if err := h.service.BulkAddTestCases(r.Context(), exerciseID, tenantID, testCases); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al guardar casos de prueba")
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"exercise_id": exerciseID,
		"added_count": len(testCases),
	}, "Casos de prueba agregados exitosamente")
}

func (h *EvaluationHandler) PublishExercise(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "Unauthorized: teacher role required", "No tiene permisos para publicar ejercicios")
		return
	}

	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "Exercise ID is required", "ID de ejercicio faltante")
		return
	}

	tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	ex, err := h.service.PublishExercise(r.Context(), exerciseID, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, err.Error(), "Ejercicio no encontrado")
			return
		}
		if errors.Is(err, services.ErrZeroPublicTestCases) || strings.Contains(err.Error(), "0 public test cases") {
			SendError(w, 422, err.Error(), "No se puede publicar un ejercicio sin al menos un caso de prueba público")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al publicar el ejercicio")
		return
	}

	SendJSON(w, http.StatusOK, ex, "Ejercicio publicado exitosamente")
}

