package domain_test

import (
	"encoding/json"
	"testing"

	"solv-backend/internal/core/domain"
)

func TestRubricCriterion_UnmarshalJSON(t *testing.T) {
	t.Run("Canonical schema with name and weight", func(t *testing.T) {
		raw := `{"id":"1","name":"Arquitectura","description":"Separación de capas","weight":40}`
		var c domain.RubricCriterion
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Name != "Arquitectura" || c.Weight != 40 {
			t.Errorf("expected Name Arquitectura and Weight 40, got Name=%q, Weight=%v", c.Name, c.Weight)
		}
	})

	t.Run("Legacy seed schema with title and max_score", func(t *testing.T) {
		raw := `{"title":"Diseño y Tipado","max_score":35,"description":"Uso de interfaces"}`
		var c domain.RubricCriterion
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Name != "Diseño y Tipado" || c.Weight != 35 {
			t.Errorf("expected Name 'Diseño y Tipado' and Weight 35, got Name=%q, Weight=%v", c.Name, c.Weight)
		}
	})
}
