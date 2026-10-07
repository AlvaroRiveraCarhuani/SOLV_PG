package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type ExerciseImportValidationItem struct {
	Index    int      `json:"index"`
	Title    string   `json:"title"`
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type ExerciseImportResponse struct {
	Mode           string                         `json:"mode"`
	ExercisesCount int                            `json:"exercises_count,omitempty"`
	Exercises      []ExerciseImportValidationItem `json:"exercises,omitempty"`
	CanImport      bool                           `json:"can_import"`
	TotalValid     int                            `json:"total_valid,omitempty"`
	TotalInvalid   int                            `json:"total_invalid,omitempty"`
	ImportedCount  int                            `json:"imported_count,omitempty"`
	ImportedIDs    []string                       `json:"imported_ids,omitempty"`
}

var (
	ErrImportFileTooLarge = errors.New("el archivo excede el tamaño máximo permitido de 10 MB")
	ErrTooManyExercises   = errors.New("el archivo contiene más de 50 ejercicios (máximo 50 por archivo)")
	ErrImportEmptyFile    = errors.New("el archivo de importación está vacío")
)

// ImportExercises procesa la importación masiva de ejercicios en formato JSON o YAML.
func (s *EvaluationService) ImportExercises(ctx context.Context, courseID string, fileContent []byte, fileName string, dryRun bool) (*ExerciseImportResponse, error) {
	if len(fileContent) == 0 {
		return nil, ErrImportEmptyFile
	}
	if len(fileContent) > 10*1024*1024 {
		return nil, ErrImportFileTooLarge
	}

	exercises, err := parseImportPayload(fileContent, fileName)
	if err != nil {
		return nil, fmt.Errorf("error al parsear el archivo de importación: %w", err)
	}

	if len(exercises) == 0 {
		return nil, errors.New("no se encontraron ejercicios válidos en el archivo")
	}
	if len(exercises) > 50 {
		return nil, ErrTooManyExercises
	}

	tenantID := domain.GetTenantID(ctx)

	// Obtener ejercicios existentes en la materia para detectar colisiones de título
	existingExercises, _ := s.exerciseRepo.ListBySubject(ctx, tenantID, courseID)
	existingTitles := make(map[string]bool)
	for _, ex := range existingExercises {
		existingTitles[strings.ToLower(strings.TrimSpace(ex.Title))] = true
	}

	validationItems := make([]ExerciseImportValidationItem, 0, len(exercises))
	totalValid := 0
	totalInvalid := 0

	for i, ex := range exercises {
		ex.SubjectID = &courseID
		ex.TenantID = tenantID
		if ex.ID == "" {
			ex.ID = uuid.NewString()
		}

		itemErrors := make([]string, 0)
		if strings.TrimSpace(ex.Title) == "" {
			itemErrors = append(itemErrors, "El título del ejercicio es obligatorio ('title')")
		}
		if ex.Type == "" {
			ex.Type = domain.ExerciseTypeAlgorithm
		}
		if ex.Status == "" {
			ex.Status = "draft"
		}
		if ex.Language == "" {
			ex.Language = "python"
		}
		if ex.TimeLimitMS <= 0 {
			ex.TimeLimitMS = 1000
		}
		if ex.MemoryLimitMB <= 0 {
			ex.MemoryLimitMB = 128
		}

		if err := ex.Validate(); err != nil {
			itemErrors = append(itemErrors, err.Error())
		}
		if err := s.validateExerciseInputFormat(ex); err != nil {
			itemErrors = append(itemErrors, err.Error())
		}
		if ex.Config.Algorithm != nil {
			for idx, tc := range ex.Config.Algorithm.TestCases {
				tc.Normalize()
				if err := tc.Validate(); err != nil {
					itemErrors = append(itemErrors, fmt.Sprintf("Caso de prueba #%d inválido: %v", idx+1, err))
				}
			}
		}

		isValid := len(itemErrors) == 0
		if isValid {
			totalValid++
		} else {
			totalInvalid++
		}

		validationItems = append(validationItems, ExerciseImportValidationItem{
			Index:    i,
			Title:    ex.Title,
			Valid:    isValid,
			Errors:   itemErrors,
			Warnings: make([]string, 0),
		})
	}

	canImport := totalInvalid == 0

	if dryRun {
		return &ExerciseImportResponse{
			Mode:           "dry_run",
			ExercisesCount: len(exercises),
			Exercises:      validationItems,
			CanImport:      canImport,
			TotalValid:     totalValid,
			TotalInvalid:   totalInvalid,
		}, nil
	}

	if !canImport {
		return &ExerciseImportResponse{
			Mode:         "import",
			CanImport:    false,
			TotalValid:   totalValid,
			TotalInvalid: totalInvalid,
			Exercises:    validationItems,
		}, nil
	}

	importedIDs := make([]string, 0, len(exercises))
	for _, ex := range exercises {
		ex.Title = resolveTitleCollision(ex.Title, existingTitles)
		existingTitles[strings.ToLower(strings.TrimSpace(ex.Title))] = true

		if err := s.CreateExercise(ctx, ex); err != nil {
			return nil, fmt.Errorf("error guardando el ejercicio '%s': %w", ex.Title, err)
		}
		importedIDs = append(importedIDs, ex.ID)
	}

	return &ExerciseImportResponse{
		Mode:          "import",
		CanImport:     true,
		ImportedCount: len(importedIDs),
		ImportedIDs:   importedIDs,
	}, nil
}

func resolveTitleCollision(title string, existing map[string]bool) string {
	cleanTitle := strings.TrimSpace(title)
	if !existing[strings.ToLower(cleanTitle)] {
		return cleanTitle
	}

	candidate := cleanTitle + " (importado)"
	if !existing[strings.ToLower(candidate)] {
		return candidate
	}

	for i := 2; i <= 100; i++ {
		candidate = fmt.Sprintf("%s (importado %d)", cleanTitle, i)
		if !existing[strings.ToLower(candidate)] {
			return candidate
		}
	}
	return fmt.Sprintf("%s (importado %s)", cleanTitle, uuid.NewString()[:8])
}

func parseImportPayload(content []byte, fileName string) ([]*domain.Exercise, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	isYAML := ext == ".yaml" || ext == ".yml"

	var rawItems []map[string]interface{}

	if isYAML {
		// Intentar unmarshal como array en YAML
		var arr []map[string]interface{}
		if err := yaml.Unmarshal(content, &arr); err == nil && len(arr) > 0 {
			rawItems = arr
		} else {
			// Intentar unmarshal como objeto único en YAML
			var single map[string]interface{}
			if err := yaml.Unmarshal(content, &single); err == nil && len(single) > 0 {
				rawItems = []map[string]interface{}{single}
			} else {
				return nil, fmt.Errorf("formato YAML no válido")
			}
		}
	} else {
		// Intentar unmarshal como array en JSON
		var arr []map[string]interface{}
		if err := json.Unmarshal(content, &arr); err == nil {
			rawItems = arr
		} else {
			// Intentar unmarshal como objeto único en JSON
			var single map[string]interface{}
			if err := json.Unmarshal(content, &single); err == nil {
				rawItems = []map[string]interface{}{single}
			} else {
				return nil, fmt.Errorf("formato JSON no válido")
			}
		}
	}

	result := make([]*domain.Exercise, 0, len(rawItems))
	for _, item := range rawItems {
		ex := convertMapToExercise(item)
		if ex != nil {
			result = append(result, ex)
		}
	}

	return result, nil
}

func convertMapToExercise(item map[string]interface{}) *domain.Exercise {
	bytes, err := json.Marshal(item)
	if err != nil {
		return nil
	}

	var ex domain.Exercise

	// Verificar si viene con formato jerárquico (metadata, statement, contract, cases)
	if metaRaw, ok := item["metadata"].(map[string]interface{}); ok {
		if title, ok := metaRaw["title"].(string); ok {
			ex.Title = title
		}
		if diff, ok := metaRaw["difficulty"].(string); ok {
			ex.Difficulty = &diff
		}
		if tagsRaw, ok := metaRaw["tags"].([]interface{}); ok {
			for _, t := range tagsRaw {
				if ts, ok := t.(string); ok {
					ex.Tags = append(ex.Tags, ts)
				}
			}
		}
		if mod, ok := metaRaw["modality"].(string); ok {
			ex.Type = domain.ExerciseType(mod)
		}
		if purp, ok := metaRaw["purpose"].(string); ok {
			ex.Purpose = purp
		}
		if tLim, ok := metaRaw["time_limit_ms"].(float64); ok {
			ex.TimeLimitMS = int(tLim)
		}
		if mLim, ok := metaRaw["memory_limit_mb"].(float64); ok {
			ex.MemoryLimitMB = int(mLim)
		}
		if expComp, ok := metaRaw["expected_complexity"].(string); ok {
			ex.ExpectedComplexity = &expComp
		}

		if stmtRaw, ok := item["statement"].(map[string]interface{}); ok {
			if desc, ok := stmtRaw["description"].(string); ok {
				ex.Description = desc
			}
		}

		if refRaw, ok := item["reference_solution"].(map[string]interface{}); ok {
			if code, ok := refRaw["code"].(string); ok {
				ex.ReferenceSolution = code
			}
			if lang, ok := refRaw["language"].(string); ok {
				ex.Language = lang
			}
		}

		if bpRaw, ok := item["boilerplate"].(map[string]interface{}); ok {
			if code, ok := bpRaw[ex.Language].(string); ok {
				ex.Boilerplate = code
			} else {
				for _, v := range bpRaw {
					if codeStr, ok := v.(string); ok {
						ex.Boilerplate = codeStr
						break
					}
				}
			}
		} else if bpStr, ok := item["boilerplate"].(string); ok {
			ex.Boilerplate = bpStr
		}

		// Contract & Cases
		if ex.Config.Algorithm == nil {
			ex.Config.Algorithm = &domain.AlgorithmConfig{
				TimeLimitMS:   ex.TimeLimitMS,
				MemoryLimitMB: ex.MemoryLimitMB,
			}
		}

		if contractRaw, ok := item["contract"]; ok {
			contractBytes, _ := json.Marshal(contractRaw)
			ex.Config.Algorithm.InputFormat = contractBytes
			ex.Config.InputFormat = contractBytes
		}

		if casesRaw, ok := item["cases"].([]interface{}); ok {
			testCases := make([]domain.TestCase, 0, len(casesRaw))
			for idx, c := range casesRaw {
				if cMap, ok := c.(map[string]interface{}); ok {
					tc := domain.TestCase{
						OrderIndex: idx + 1,
						Weight:     1.0,
					}
					if input, ok := cMap["input"].(string); ok {
						tc.Input = input
					}
					if expOut, ok := cMap["expected_output"].(string); ok {
						tc.ExpectedOutput = expOut
					}
					if vis, ok := cMap["visibility"].(string); ok {
						tc.Visibility = domain.TestCaseVisibility(vis)
						if vis == "hidden" {
							tc.IsHidden = true
						}
					}
					if w, ok := cMap["weight"].(float64); ok {
						tc.Weight = w
					}
					testCases = append(testCases, tc)
				}
			}
			ex.Config.Algorithm.TestCases = testCases
		}
		return &ex
	}

	// Estructura directa plana (json unmarshal sobre struct domain.Exercise)
	if err := json.Unmarshal(bytes, &ex); err == nil && ex.Title != "" {
		if ex.Config.Algorithm == nil && len(ex.Config.InputFormat) > 0 {
			ex.Config.Algorithm = &domain.AlgorithmConfig{
				TimeLimitMS:   ex.TimeLimitMS,
				MemoryLimitMB: ex.MemoryLimitMB,
				InputFormat:   ex.Config.InputFormat,
			}
		}
		return &ex
	}

	return &ex
}
