package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

type mockSecurityExerciseRepo struct {
	exercise *domain.Exercise
}

func (m *mockSecurityExerciseRepo) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	return m.exercise, nil
}

func (m *mockSecurityExerciseRepo) Create(ctx context.Context, exercise *domain.Exercise) error {
	return nil
}

func (m *mockSecurityExerciseRepo) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	return nil
}

func TestExerciseSecurityDTOFiltering(t *testing.T) {
	ex := &domain.Exercise{
		ID:          "ex-sec-101",
		Title:       "Suma de dos números",
		Description: "Escriba un programa que sume A + B.",
		Type:        domain.ExerciseTypeAlgorithm,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: domain.TestCases{
					{Input: "2 3\n", ExpectedOutput: "5\n", IsHidden: false},
					{Input: "100 200\n", ExpectedOutput: "300\n", IsHidden: true}, // Caso de prueba secreto
				},
				ASTRules: domain.ASTRules{
					ForbiddenImports:   []string{"os"},
					ForbiddenFunctions: []string{"eval"},
				},
				TimeLimitMS:   1000,
				MemoryLimitMB: 128,
			},
		},
	}

	repo := &mockSecurityExerciseRepo{exercise: ex}
	evalService := services.NewEvaluationService(repo, nil, nil, nil)
	val := validator.New()
	handler := httpdelivery.NewEvaluationHandler(evalService, val)

	t.Run("Test 1: Estudiante consulta ejercicio -> respuesta NO contiene test_cases", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/exercises/ex-sec-101", nil)
		req.SetPathValue("id", "ex-sec-101")
		req.Header.Set("X-User-Role", "student")

		w := httptest.NewRecorder()
		handler.GetExerciseByID(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Se esperaba 200 OK, se obtuvo %d", w.Code)
		}

		body := w.Body.String()
		if strings.Contains(body, "test_cases") {
			t.Errorf("VIOLACIÓN DE SEGURIDAD: La respuesta para rol student contiene 'test_cases': %s", body)
		}
		if strings.Contains(body, "100 200") || strings.Contains(body, "300") {
			t.Errorf("VIOLACIÓN DE SEGURIDAD: La respuesta para rol student filtra valores de test_cases secretos: %s", body)
		}

		// Debe contener los campos públicos
		if !strings.Contains(body, "ex-sec-101") || !strings.Contains(body, "Suma de dos números") {
			t.Errorf("La respuesta pública debe incluir ID y Title: %s", body)
		}
		t.Logf("PASS: Respuesta para student enmascara test_cases correctamente: %s", body)
	})

	t.Run("Test 2: Docente consulta ejercicio -> respuesta SÍ contiene test_cases completosa", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/exercises/ex-sec-101", nil)
		req.SetPathValue("id", "ex-sec-101")
		req.Header.Set("X-User-Role", "teacher")

		w := httptest.NewRecorder()
		handler.GetExerciseByID(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Se esperaba 200 OK, se obtuvo %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "test_cases") {
			t.Errorf("Se esperaba que la respuesta para teacher contenga 'test_cases': %s", body)
		}
		if !strings.Contains(body, "100 200") {
			t.Errorf("Se esperaba que la respuesta para teacher contenga los valores de test_cases: %s", body)
		}
		t.Logf("PASS: Respuesta para teacher incluye test_cases completos: %s", body)
	})

	t.Run("Test 3: Validación estricta de estructura JSON hacia rol student", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/exercises/ex-sec-101", nil)
		req.SetPathValue("id", "ex-sec-101")
		req.Header.Set("X-User-Role", "student")

		w := httptest.NewRecorder()
		handler.GetExerciseByID(w, req)

		var respStruct struct {
			Success bool            `json:"success"`
			Data    json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal(w.Body.Bytes(), &respStruct); err != nil {
			t.Fatalf("Error al desmarshallear wrapper JSON: %v", err)
		}

		var rawMap map[string]interface{}
		if err := json.Unmarshal(respStruct.Data, &rawMap); err != nil {
			t.Fatalf("Error al desmarshallear data payload: %v", err)
		}

		if _, exists := rawMap["config"]; exists {
			t.Errorf("VIOLACIÓN CRÍTICA DE SEGURIDAD: El payload 'data' contiene la clave 'config' completa")
		}
		if _, exists := rawMap["test_cases"]; exists {
			t.Errorf("VIOLACIÓN CRÍTICA DE SEGURIDAD: El payload 'data' contiene la clave 'test_cases'")
		}
		t.Logf("PASS: Validación estricta confirmó ausencia de campos sensibles en el DTO de estudiante.")
	})
}
