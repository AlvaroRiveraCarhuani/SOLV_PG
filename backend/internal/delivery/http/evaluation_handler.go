package httpdelivery

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

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
		if errors.Is(err, domain.ErrModuleLocked) {
			SendError(w, http.StatusForbidden, "MODULE_LOCKED", "El módulo curricular se encuentra bloqueado")
			return
		}
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

func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, domain.ErrSeedRequiresExamPurpose) ||
		errors.Is(err, domain.ErrInvalidVisibility) ||
		errors.Is(err, domain.ErrInvalidWeight) ||
		errors.Is(err, domain.ErrEmptyExpectedOutput) ||
		errors.Is(err, domain.ErrTestCasesNotArray) ||
		errors.Is(err, domain.ErrMemoryGovernedByTemplate) ||
		errors.Is(err, domain.ErrTemplateEnvironmentMismatch) ||
		errors.Is(err, domain.ErrTemplateNotApproved) ||
		strings.Contains(err.Error(), "contrato de formato") ||
		strings.Contains(err.Error(), "inválido según contrato") {
		return true
	}
	return false
}

func normalizeExerciseConfig(body []byte, ex *domain.Exercise) {
	var raw struct {
		ASTRules   *domain.ASTRules         `json:"ast_rules"`
		Comparator *domain.ComparatorConfig `json:"comparator"`
		TestCases  []domain.TestCase        `json:"test_cases"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return
	}
	if ex.Type == domain.ExerciseTypeAlgorithm || ex.Type == "" {
		if ex.Config.Algorithm == nil {
			ex.Config.Algorithm = &domain.AlgorithmConfig{
				TimeLimitMS:   ex.TimeLimitMS,
				MemoryLimitMB: ex.MemoryLimitMB,
			}
		}
		if raw.ASTRules != nil {
			ex.Config.Algorithm.ASTRules = *raw.ASTRules
		}
		if raw.Comparator != nil {
			ex.Config.Algorithm.Comparator = raw.Comparator
		}
		if len(raw.TestCases) > 0 && len(ex.Config.Algorithm.TestCases) == 0 {
			for i := range raw.TestCases {
				raw.TestCases[i].Normalize()
			}
			ex.Config.Algorithm.TestCases = raw.TestCases
		}
	}
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

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body", "Error al leer el cuerpo de la petición")
		return
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawMap); err == nil {
		if _, hasRam := rawMap["memory_limit_mb"]; hasRam {
			SendError(w, http.StatusUnprocessableEntity, "memory_governed_by_template", domain.ErrMemoryGovernedByTemplate.Error())
			return
		}
	}

	var ex domain.Exercise
	if err := json.Unmarshal(bodyBytes, &ex); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	normalizeExerciseConfig(bodyBytes, &ex)

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
		if isValidationError(err) {
			SendError(w, http.StatusUnprocessableEntity, err.Error(), err.Error())
			return
		}
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

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body", "Error al leer el cuerpo de la petición")
		return
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawMap); err == nil {
		if _, hasRam := rawMap["memory_limit_mb"]; hasRam {
			SendError(w, http.StatusUnprocessableEntity, "memory_governed_by_template", domain.ErrMemoryGovernedByTemplate.Error())
			return
		}
	}

	var ex domain.Exercise
	if err := json.Unmarshal(bodyBytes, &ex); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	normalizeExerciseConfig(bodyBytes, &ex)

	ex.ID = exerciseID
	ex.TenantID = tenantID
	if ex.Type == "" {
		ex.Type = existing.Type
	}

	if err := h.service.UpdateExercise(r.Context(), &ex); err != nil {
		if isValidationError(err) {
			SendError(w, http.StatusUnprocessableEntity, err.Error(), err.Error())
			return
		}
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
			SendError(w, http.StatusUnprocessableEntity, fmt.Sprintf("Invalid JSON test cases: %v", err), "Formato de casos de prueba inválido")
			return
		}
	}

	for i := range testCases {
		testCases[i].Normalize()
		if err := testCases[i].Validate(); err != nil {
			SendError(w, http.StatusUnprocessableEntity, err.Error(), err.Error())
			return
		}
	}

	if err := h.service.BulkAddTestCases(r.Context(), exerciseID, tenantID, testCases); err != nil {
		if isValidationError(err) {
			SendError(w, http.StatusUnprocessableEntity, err.Error(), err.Error())
			return
		}
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
		SendError(w, http.StatusForbidden, "FORBIDDEN", "No tiene permisos para publicar ejercicios")
		return
	}

	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "MISSING_EXERCISE_ID", "ID de ejercicio faltante")
		return
	}

	tenantID := getTenantFromCtx(r)

	ex, err := h.service.PublishExercise(r.Context(), exerciseID, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "NOT_FOUND", "Ejercicio no encontrado")
			return
		}
		if errors.Is(err, domain.ErrMissingReferenceSolution) {
			SendError(w, http.StatusConflict, "REFERENCE_REQUIRED", "El ejercicio requiere una solución de referencia antes de ser publicado")
			return
		}
		if errors.Is(err, domain.ErrExerciseStale) {
			SendError(w, http.StatusConflict, "EXERCISE_STALE", "El ejercicio tiene cambios pendientes y requiere un dry-run exitoso antes de ser publicado")
			return
		}
		if errors.Is(err, services.ErrZeroPublicTestCases) || strings.Contains(err.Error(), "0 public test cases") {
			SendError(w, 422, "NO_PUBLIC_TEST_CASES", "No se puede publicar un ejercicio sin al menos un caso de prueba público")
			return
		}
		SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error al publicar el ejercicio: "+err.Error())
		return
	}

	SendJSON(w, http.StatusOK, ex, "Ejercicio publicado exitosamente")
}

// StartDryRun maneja POST /api/v1/exercises/{id}/dry-run -> 202 Accepted
func (h *EvaluationHandler) StartDryRun(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "FORBIDDEN", "No tiene permisos para ejecutar comprobación previa")
		return
	}

	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "MISSING_EXERCISE_ID", "ID de ejercicio faltante")
		return
	}

	tenantID := getTenantFromCtx(r)

	job, err := h.service.StartDryRun(r.Context(), exerciseID, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			SendError(w, http.StatusNotFound, "NOT_FOUND", "Ejercicio no encontrado")
			return
		}
		if errors.Is(err, domain.ErrMissingReferenceSolution) {
			SendError(w, http.StatusBadRequest, "REFERENCE_REQUIRED", "Debe configurar una solución de referencia antes de iniciar el dry-run")
			return
		}
		if errors.Is(err, domain.ErrRamExceedsHostCapacity) {
			SendError(w, http.StatusBadRequest, "RAM_EXCEEDS_HOST", "La memoria configurada excede la capacidad estructural del host")
			return
		}
		SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Error al iniciar trabajo de dry-run: "+err.Error())
		return
	}

	SendJSON(w, http.StatusAccepted, job, "Trabajo de dry-run iniciado exitosamente")
}

// GetDryRunJob maneja GET /api/v1/exercises/{id}/dry-run/jobs/{jobId}
func (h *EvaluationHandler) GetDryRunJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("jobId")
	if jobID == "" {
		jobID = r.PathValue("id")
	}
	if jobID == "" {
		SendError(w, http.StatusBadRequest, "MISSING_JOB_ID", "ID de trabajo faltante")
		return
	}

	job, err := h.service.GetDryRunJob(r.Context(), jobID)
	if err != nil {
		SendError(w, http.StatusNotFound, "NOT_FOUND", "Trabajo de dry-run no encontrado")
		return
	}

	SendJSON(w, http.StatusOK, job, "Estado de trabajo dry-run obtenido exitosamente")
}

func (h *EvaluationHandler) CalculateOutputs(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "Unauthorized: teacher role required", "No tiene permisos para calcular salidas de prueba")
		return
	}

	var req services.CalculateOutputsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	resp, err := h.service.CalculateOutputs(r.Context(), req)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "Error al calcular salidas con la solución de referencia")
		return
	}

	SendJSON(w, http.StatusOK, resp, "Salidas calculadas exitosamente")
}

type ValidateInputRequestDTO struct {
	Contract json.RawMessage `json:"contract"`
	Input    string          `json:"input"`
}

type ValidateInputResponseDTO struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

func (h *EvaluationHandler) ValidateInputFormat(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "Unauthorized: teacher role required", "No tiene permisos para validar formato de entrada")
		return
	}

	var req ValidateInputRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	validator := services.NewFormatValidator()
	if err := validator.ValidateContract(req.Contract); err != nil {
		SendJSON(w, http.StatusOK, ValidateInputResponseDTO{
			Valid: false,
			Error: err.Error(),
		}, "Contrato de formato inválido")
		return
	}

	valid, msg := validator.ValidateCase(req.Contract, req.Input)
	SendJSON(w, http.StatusOK, ValidateInputResponseDTO{
		Valid: valid,
		Error: msg,
	}, "Validación de entrada completada")
}

type ExerciseChecklistRequestDTO struct {
	ReferenceSolution *domain.ReferenceSolutionInput `json:"reference_solution,omitempty"`
}

func (h *EvaluationHandler) GetExerciseChecklist(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "Unauthorized: teacher role required", "No tiene permisos para consultar el checklist de publicación")
		return
	}

	exerciseID := r.PathValue("id")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "MISSING_EXERCISE_ID", "ID de ejercicio faltante")
		return
	}

	var req ExerciseChecklistRequestDTO
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	report, err := h.service.GenerateChecklist(r.Context(), exerciseID, req.ReferenceSolution)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al generar checklist de publicación")
		return
	}

	SendJSON(w, http.StatusOK, report, "Checklist de publicación generado exitosamente")
}

type GenerateCasesRequestDTO struct {
	Contract json.RawMessage `json:"contract"`
	Count    int             `json:"count"`
	Seed     *int64          `json:"seed,omitempty"`
}

type GeneratedCaseItemDTO struct {
	Input  string  `json:"input"`
	Output *string `json:"output"`
}

type GenerateCasesResponseDTO struct {
	Cases []GeneratedCaseItemDTO `json:"cases"`
}

func (h *EvaluationHandler) GenerateCases(w http.ResponseWriter, r *http.Request) {
	userRole := r.Header.Get("X-User-Role")
	if userRole != "teacher" && userRole != "admin" {
		SendError(w, http.StatusForbidden, "Unauthorized: teacher role required", "No tiene permisos para generar casos de prueba")
		return
	}

	var req GenerateCasesRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON payload", "Cuerpo de la petición inválido")
		return
	}

	if req.Count <= 0 {
		req.Count = 10
	}
	if req.Count > 100 {
		req.Count = 100
	}

	validator := services.NewFormatValidator()
	if err := validator.ValidateContract(req.Contract); err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), "El contrato de formato no es válido")
		return
	}

	cases := make([]GeneratedCaseItemDTO, 0, req.Count)
	baseSeed := time.Now().UnixNano()
	if req.Seed != nil {
		baseSeed = *req.Seed
	}

	for i := 0; i < req.Count; i++ {
		caseSeed := baseSeed + int64(i*10007)
		if req.Seed == nil {
			caseSeed = time.Now().UnixNano() + int64(i*10007+rand.Intn(1000000))
		}
		generatedInput, err := validator.GenerateCase(req.Contract, caseSeed)
		if err != nil {
			SendError(w, http.StatusInternalServerError, err.Error(), fmt.Sprintf("Error generando caso #%d", i+1))
			return
		}
		cases = append(cases, GeneratedCaseItemDTO{
			Input:  generatedInput,
			Output: nil,
		})
	}

	SendJSON(w, http.StatusOK, GenerateCasesResponseDTO{
		Cases: cases,
	}, "Casos generados exitosamente")
}

func (h *EvaluationHandler) ImportExercises(w http.ResponseWriter, r *http.Request) {
	courseID := r.PathValue("courseId")
	if courseID == "" {
		SendError(w, http.StatusBadRequest, "Course ID missing", "ID de curso faltante")
		return
	}

	dryRun := r.URL.Query().Get("dry_run") == "true"

	var content []byte
	var fileName string

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		err := r.ParseMultipartForm(10 << 20) // 10MB limit
		if err != nil {
			SendError(w, http.StatusBadRequest, err.Error(), "Error al procesar el archivo subido")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			SendError(w, http.StatusBadRequest, err.Error(), "Archivo 'file' no encontrado en el formulario")
			return
		}
		defer file.Close()

		fileName = header.Filename
		var readErr error
		content, readErr = io.ReadAll(file)
		if readErr != nil {
			SendError(w, http.StatusInternalServerError, readErr.Error(), "Error al leer el archivo")
			return
		}
	} else {
		var err error
		content, err = io.ReadAll(r.Body)
		if err != nil {
			SendError(w, http.StatusBadRequest, err.Error(), "Error al leer el cuerpo de la petición")
			return
		}
		fileName = r.URL.Query().Get("file_name")
		if fileName == "" {
			fileName = "import.json"
		}
	}

	res, err := h.service.ImportExercises(r.Context(), courseID, content, fileName, dryRun)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), err.Error())
		return
	}

	if !res.CanImport {
		SendJSON(w, http.StatusUnprocessableEntity, res, "Errores de validación en los ejercicios a importar")
		return
	}

	if dryRun {
		SendJSON(w, http.StatusOK, res, "Previsualización de importación completada")
		return
	}

	SendJSON(w, http.StatusCreated, res, fmt.Sprintf("Se importaron %d ejercicios exitosamente", res.ImportedCount))
}

func (h *EvaluationHandler) GenerateCasesFromScript(w http.ResponseWriter, r *http.Request) {
	exerciseID := r.PathValue("exerciseId")
	if exerciseID == "" {
		SendError(w, http.StatusBadRequest, "Exercise ID missing", "ID de ejercicio faltante")
		return
	}

	tenantID := domain.GetTenantID(r.Context())
	dryRun := r.URL.Query().Get("dry_run") == "true"

	var scriptCode string
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			SendError(w, http.StatusBadRequest, err.Error(), "Error al procesar el formulario")
			return
		}
		scriptCode = r.FormValue("script")
		if scriptCode == "" {
			file, _, err := r.FormFile("script")
			if err == nil {
				defer file.Close()
				buf, readErr := io.ReadAll(file)
				if readErr == nil {
					scriptCode = string(buf)
				}
			}
		}
	} else if strings.HasPrefix(contentType, "application/json") {
		var req struct {
			Script string `json:"script"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			scriptCode = req.Script
		}
	} else {
		buf, err := io.ReadAll(r.Body)
		if err == nil {
			scriptCode = string(buf)
		}
	}

	if strings.TrimSpace(scriptCode) == "" {
		SendError(w, http.StatusBadRequest, "Script is empty", "El código del script no puede estar vacío")
		return
	}

	res, err := h.service.GenerateCasesFromScript(r.Context(), exerciseID, tenantID, scriptCode, dryRun)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error(), err.Error())
		return
	}

	if !res.CanImport {
		SendJSON(w, http.StatusUnprocessableEntity, res, "Se encontraron errores de validación en los casos generados por el script")
		return
	}

	if dryRun {
		SendJSON(w, http.StatusOK, res, "Previsualización de casos generados exitosamente")
		return
	}

	SendJSON(w, http.StatusCreated, res, fmt.Sprintf("Se agregaron %d casos al ejercicio exitosamente", res.ImportedCount))
}
