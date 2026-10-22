package dto_test

import (
	"encoding/json"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/delivery/http/dto"
)

func TestToStudentEvaluationResponse_OcultaSalidaEsperadaDeCasosGenerados(t *testing.T) {
	result := &domain.EvaluationResult{
		Verdict:         domain.VerdictAC,
		ExecutionTimeMS: 42,
		Message:         "¡Solución Aceptada!",
		GeneratedCases: []domain.TestCase{
			{
				ID:             "gen-1",
				OrderIndex:     1,
				Input:          "17 23",
				ExpectedOutput: "40",
				Visibility:     domain.TestCaseVisibilityHidden,
				IsHidden:       true,
				Weight:         1,
			},
		},
	}

	response := dto.ToStudentEvaluationResponse(result)
	if response == nil {
		t.Fatal("la respuesta no puede ser nil")
	}

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta: %v", err)
	}

	if strings.Contains(string(payload), "expected_output") {
		t.Fatalf("la respuesta del estudiante expone la salida esperada: %s", payload)
	}
	if strings.Contains(string(payload), "40") {
		t.Fatalf("la respuesta del estudiante expone la salida esperada del caso: %s", payload)
	}
	if len(response.GeneratedCases) != 1 {
		t.Fatalf("se esperaba 1 caso generado, se obtuvieron %d", len(response.GeneratedCases))
	}
	if response.GeneratedCases[0].Input != "17 23" {
		t.Errorf("la entrada del caso generado debe seguir visible, se obtuvo %q", response.GeneratedCases[0].Input)
	}
}

func TestToStudentEvaluationResponse_Nil(t *testing.T) {
	if dto.ToStudentEvaluationResponse(nil) != nil {
		t.Fatal("una evaluacion nula debe producir una respuesta nula")
	}
}

func TestStripExpectedOutputFromRawCases(t *testing.T) {
	raw := json.RawMessage(`[{"input":"1 2","expected_output":"3","visibility":"hidden"},{"input":"3 4","expected_output":"7"}]`)

	cleaned := dto.StripExpectedOutputFromRawCases(raw)
	payload := string(cleaned)

	if strings.Contains(payload, "expected_output") {
		t.Fatalf("el payload saneado conserva la salida esperada: %s", payload)
	}
	if !strings.Contains(payload, "input") {
		t.Fatalf("el payload saneado debe conservar la entrada: %s", payload)
	}

	var cases []map[string]any
	if err := json.Unmarshal(cleaned, &cases); err != nil {
		t.Fatalf("el payload saneado no es JSON valido: %v", err)
	}
	if len(cases) != 2 {
		t.Fatalf("se esperaban 2 casos, se obtuvieron %d", len(cases))
	}
}

func TestStripExpectedOutputFromRawCases_PayloadInvalido(t *testing.T) {
	raw := json.RawMessage(`no-es-json`)
	if got := dto.StripExpectedOutputFromRawCases(raw); string(got) != string(raw) {
		t.Fatalf("un payload invalido debe devolverse sin cambios, se obtuvo %s", got)
	}
}
