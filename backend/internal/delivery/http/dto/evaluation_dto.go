package dto

import (
	"encoding/json"

	"solv-backend/internal/core/domain"
)

// StudentGeneratedCaseDTO expone al rol student el caso generado por semilla
// sin la salida esperada. La salida esperada de un caso generado es la
// respuesta del ejercicio: entregarla al estudiante convierte el snapshot en
// una solución (GOB-SEGURO-04).
type StudentGeneratedCaseDTO struct {
	ID         string  `json:"id,omitempty"`
	ExerciseID string  `json:"exercise_id,omitempty"`
	OrderIndex int     `json:"order_index"`
	Input      string  `json:"input"`
	Visibility string  `json:"visibility"`
	Weight     float64 `json:"weight"`
	IsHidden   bool    `json:"is_hidden"`
	IsSample   bool    `json:"is_sample"`
}

// StudentFailedTestCaseDTO describe el caso fallido sin la salida esperada
// cuando el caso no es público.
type StudentFailedTestCaseDTO struct {
	ID         string  `json:"id,omitempty"`
	ExerciseID string  `json:"exercise_id,omitempty"`
	OrderIndex int     `json:"order_index"`
	Input      string  `json:"input"`
	Visibility string  `json:"visibility"`
	Weight     float64 `json:"weight"`
	IsHidden   bool    `json:"is_hidden"`
	IsSample   bool    `json:"is_sample"`
}

// StudentEvaluationResponse es la respuesta de evaluación para el rol student.
type StudentEvaluationResponse struct {
	Verdict            domain.Verdict             `json:"verdict"`
	ExecutionTimeMS    int                        `json:"execution_time_ms"`
	MemoryUsedMB       float64                    `json:"memory_used_mb"`
	Message            string                     `json:"message"`
	FailedTestCase     *StudentFailedTestCaseDTO  `json:"failed_test_case,omitempty"`
	CaseResults        []domain.CaseResult        `json:"case_results,omitempty"`
	ActualJSON         string                     `json:"actual_json,omitempty"`
	GeneratedCases     []StudentGeneratedCaseDTO  `json:"generated_cases,omitempty"`
	ComplexityAnalysis *domain.ComplexityAnalysis `json:"complexity_analysis,omitempty"`
}

// ToStudentEvaluationResponse sanitiza el resultado de evaluación para el rol
// student: elimina la salida esperada de los casos generados por semilla y del
// caso fallido cuando no es público.
func ToStudentEvaluationResponse(result *domain.EvaluationResult) *StudentEvaluationResponse {
	if result == nil {
		return nil
	}

	resp := &StudentEvaluationResponse{
		Verdict:            result.Verdict,
		ExecutionTimeMS:    result.ExecutionTimeMS,
		MemoryUsedMB:       result.MemoryUsedMB,
		Message:            result.Message,
		CaseResults:        result.CaseResults,
		ActualJSON:         result.ActualJSON,
		ComplexityAnalysis: result.ComplexityAnalysis,
	}

	if failed := result.FailedTestCase; failed != nil {
		// El caso fallido viaja sin salida esperada cuando no es público: la
		// salida de un caso oculto es la respuesta del ejercicio.
		resp.FailedTestCase = &StudentFailedTestCaseDTO{
			ID:         failed.ID,
			ExerciseID: failed.ExerciseID,
			OrderIndex: failed.OrderIndex,
			Input:      failed.Input,
			Visibility: string(failed.Visibility),
			Weight:     failed.Weight,
			IsHidden:   failed.IsHidden,
			IsSample:   failed.IsSample,
		}
	}

	for _, gc := range result.GeneratedCases {
		resp.GeneratedCases = append(resp.GeneratedCases, StudentGeneratedCaseDTO{
			ID:         gc.ID,
			ExerciseID: gc.ExerciseID,
			OrderIndex: gc.OrderIndex,
			Input:      gc.Input,
			Visibility: string(gc.Visibility),
			Weight:     gc.Weight,
			IsHidden:   gc.IsHidden,
			IsSample:   gc.IsSample,
		})
	}

	return resp
}

// StripExpectedOutputFromRawCases elimina la clave de salida esperada de un
// payload JSON de casos generados. Se usa sobre el payload que envía el
// cliente para que una entrega de estudiante nunca persista salidas
// esperadas declaradas por el propio cliente.
func StripExpectedOutputFromRawCases(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}

	var cases []map[string]any
	if err := json.Unmarshal(raw, &cases); err != nil {
		return raw
	}

	for _, c := range cases {
		delete(c, "expected_output")
		delete(c, "ExpectedOutput")
	}

	cleaned, err := json.Marshal(cases)
	if err != nil {
		return raw
	}
	return json.RawMessage(cleaned)
}
