package httpdelivery

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

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
		defer func() { _ = file.Close() }()

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
				defer func() { _ = file.Close() }()
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
