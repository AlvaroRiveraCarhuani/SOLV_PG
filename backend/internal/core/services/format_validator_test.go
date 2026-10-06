package services_test

import (
	"encoding/json"
	"strings"
	"testing"

	"solv-backend/internal/core/services"
)

func TestFormatValidator_ValidateContract(t *testing.T) {
	v := services.NewFormatValidator()

	t.Run("Valid complete contract with dynamic references", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "n", "type": "int", "min": 1, "max": 1000 },
					{ "id": "m", "type": "int", "min": 1, "max": 50 },
					{ "id": "a", "type": "ints", "count": "=n", "item": { "min": -1000, "max": 1000 } },
					{ "id": "mat", "type": "matrix", "rows": "=n", "cols": "=m", "item": { "type": "int", "min": 0, "max": 100 } },
					{ "id": "e", "type": "edges", "count": "=m", "fields": [
						{ "id": "u", "type": "int", "min": 1, "max": "=n" },
						{ "id": "v", "type": "int", "min": 1, "max": "=n" }
					]},
					{ "id": "s", "type": "string", "regex": "^[a-z]+$", "max_len": 50 },
					{ "id": "libre", "type": "raw" }
				]
			}
		}`)
		if err := v.ValidateContract(contract); err != nil {
			t.Fatalf("unexpected error on valid contract: %v", err)
		}
	})

	t.Run("Duplicate ID is rejected", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "n", "type": "int" },
					{ "id": "n", "type": "int" }
				]
			}
		}`)
		err := v.ValidateContract(contract)
		if err == nil {
			t.Fatalf("expected error on duplicate ID, got nil")
		}
		if !strings.Contains(err.Error(), "identificador duplicado") {
			t.Errorf("expected duplicate ID message, got %v", err)
		}
	})

	t.Run("Forward reference to undeclared ID is rejected", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "a", "type": "ints", "count": "=n" },
					{ "id": "n", "type": "int" }
				]
			}
		}`)
		err := v.ValidateContract(contract)
		if err == nil {
			t.Fatalf("expected error on forward reference, got nil")
		}
		if !strings.Contains(err.Error(), "inexistente o declarada posteriormente") {
			t.Errorf("expected undeclared ref message, got %v", err)
		}
	})

	t.Run("Invalid regex is rejected", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "s", "type": "string", "regex": "[a-z" }
				]
			}
		}`)
		err := v.ValidateContract(contract)
		if err == nil {
			t.Fatalf("expected error on invalid regex, got nil")
		}
		if !strings.Contains(err.Error(), "expresión regular inválida") {
			t.Errorf("expected invalid regex message, got %v", err)
		}
	})
}

func TestFormatValidator_ValidateCase_Types(t *testing.T) {
	v := services.NewFormatValidator()

	t.Run("int type validation", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "n", "type": "int", "min": 1, "max": 10 }
				]
			}
		}`)

		ok, msg := v.ValidateCase(contract, "5")
		if !ok || msg != "" {
			t.Errorf("expected valid case, got ok=%v, msg=%q", ok, msg)
		}

		ok, msg = v.ValidateCase(contract, "15")
		if ok || !strings.Contains(msg, "mayor que el máximo permitido (10)") {
			t.Errorf("expected max limit error, got ok=%v, msg=%q", ok, msg)
		}

		ok, msg = v.ValidateCase(contract, "0")
		if ok || !strings.Contains(msg, "menor que el mínimo permitido (1)") {
			t.Errorf("expected min limit error, got ok=%v, msg=%q", ok, msg)
		}

		ok, msg = v.ValidateCase(contract, "5 10")
		if ok || !strings.Contains(msg, "se esperaba exactamente 1 entero") {
			t.Errorf("expected token count error, got ok=%v, msg=%q", ok, msg)
		}
	})

	t.Run("ints with dynamic count =n", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "n", "type": "int", "min": 1, "max": 10 },
					{ "id": "a", "type": "ints", "count": "=n", "item": { "min": 0, "max": 100 } }
				]
			}
		}`)

		// Correcto
		input := "4\n10 20 30 40"
		ok, msg := v.ValidateCase(contract, input)
		if !ok || msg != "" {
			t.Errorf("expected valid case, got ok=%v, msg=%q", ok, msg)
		}

		// Cantidad errónea
		inputMismatch := "4\n10 20 30"
		ok, msg = v.ValidateCase(contract, inputMismatch)
		if ok || !strings.Contains(msg, "Línea 2: se esperaban 4 (declarado \"=n\", n=4) enteros, llegaron 3") {
			t.Errorf("expected dynamic count error in spanish, got ok=%v, msg=%q", ok, msg)
		}

		// Elemento fuera de rango
		inputOutOfRange := "4\n10 20 150 40"
		ok, msg = v.ValidateCase(contract, inputOutOfRange)
		if ok || !strings.Contains(msg, "Línea 2 (elemento 3): el valor 150 es mayor que el máximo permitido (100)") {
			t.Errorf("expected item max error, got ok=%v, msg=%q", ok, msg)
		}
	})

	t.Run("matrix rows and cols", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "r", "type": "int", "min": 2, "max": 5 },
					{ "id": "c", "type": "int", "min": 2, "max": 5 },
					{ "id": "mat", "type": "matrix", "rows": "=r", "cols": "=c", "item": { "min": 0, "max": 9 } }
				]
			}
		}`)

		input := "2\n3\n1 2 3\n4 5 6"
		ok, msg := v.ValidateCase(contract, input)
		if !ok || msg != "" {
			t.Errorf("expected valid matrix, got ok=%v, msg=%q", ok, msg)
		}

		// Columna faltante en fila 2
		inputBadCol := "2\n3\n1 2 3\n4 5"
		ok, msg = v.ValidateCase(contract, inputBadCol)
		if ok || !strings.Contains(msg, "Línea 4 (fila 2 de matriz): se esperaban 3 columnas, llegaron 2") {
			t.Errorf("expected col mismatch error, got ok=%v, msg=%q", ok, msg)
		}

		// Fila faltante
		inputIncomplete := "2\n3\n1 2 3"
		ok, msg = v.ValidateCase(contract, inputIncomplete)
		if ok || !strings.Contains(msg, "matriz incompleta") {
			t.Errorf("expected incomplete matrix error, got ok=%v, msg=%q", ok, msg)
		}
	})

	t.Run("edges graph validation", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "n", "type": "int", "min": 2, "max": 10 },
					{ "id": "m", "type": "int", "min": 1, "max": 10 },
					{ "id": "e", "type": "edges", "count": "=m", "fields": [
						{ "id": "u", "type": "int", "min": 1, "max": "=n" },
						{ "id": "v", "type": "int", "min": 1, "max": "=n" },
						{ "id": "w", "type": "int", "min": -10, "max": 10 }
					]}
				]
			}
		}`)

		input := "4\n2\n1 2 5\n2 4 -3"
		ok, msg := v.ValidateCase(contract, input)
		if !ok || msg != "" {
			t.Errorf("expected valid edges, got ok=%v, msg=%q", ok, msg)
		}

		// Nodo u fuera de rango (> n)
		inputBadNode := "4\n2\n5 2 5\n2 4 -3"
		ok, msg = v.ValidateCase(contract, inputBadNode)
		if ok || !strings.Contains(msg, "Línea 3 (arista 1, campo 1): valor 5 mayor que el máximo (4)") {
			t.Errorf("expected edge node limit error, got ok=%v, msg=%q", ok, msg)
		}
	})

	t.Run("string regex and length", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "s", "type": "string", "regex": "^[a-z]+$", "min_len": 3, "max_len": 6 }
				]
			}
		}`)

		ok, msg := v.ValidateCase(contract, "hola")
		if !ok || msg != "" {
			t.Errorf("expected valid string, got ok=%v, msg=%q", ok, msg)
		}

		ok, msg = v.ValidateCase(contract, "HOLA")
		if ok || !strings.Contains(msg, "no coincide con la expresión regular") {
			t.Errorf("expected regex mismatch error, got ok=%v, msg=%q", ok, msg)
		}

		ok, msg = v.ValidateCase(contract, "hi")
		if ok || !strings.Contains(msg, "menor que el mínimo (3)") {
			t.Errorf("expected min_len error, got ok=%v, msg=%q", ok, msg)
		}
	})

	t.Run("raw escape hatch", func(t *testing.T) {
		contract := json.RawMessage(`{
			"version": 1,
			"input": {
				"lines": [
					{ "id": "n", "type": "int" },
					{ "id": "libre", "type": "raw" }
				]
			}
		}`)

		input := "42\ncualquier texto\nsin formato especifico\n12345 @#$%"
		ok, msg := v.ValidateCase(contract, input)
		if !ok || msg != "" {
			t.Errorf("expected valid raw escape hatch, got ok=%v, msg=%q", ok, msg)
		}
	})
}

func TestFormatValidator_PropertyReflexivity(t *testing.T) {
	v := services.NewFormatValidator()

	contract := json.RawMessage(`{
		"version": 1,
		"input": {
			"lines": [
				{ "id": "n", "type": "int", "min": 2, "max": 5 },
				{ "id": "m", "type": "int", "min": 1, "max": 4 },
				{ "id": "a", "type": "ints", "count": "=n", "item": { "min": 1, "max": 50 } },
				{ "id": "mat", "type": "matrix", "rows": "=n", "cols": "=m", "item": { "min": 0, "max": 10 } },
				{ "id": "e", "type": "edges", "count": "=m", "fields": [
					{ "id": "u", "type": "int", "min": 1, "max": "=n" },
					{ "id": "v", "type": "int", "min": 1, "max": "=n" }
				]},
				{ "id": "s", "type": "string", "min_len": 5, "max_len": 10 }
			]
		}
	}`)

	for seed := int64(1); seed <= 20; seed++ {
		genInput, err := v.GenerateCase(contract, seed)
		if err != nil {
			t.Fatalf("seed %d: error generating case: %v", seed, err)
		}
		ok, msg := v.ValidateCase(contract, genInput)
		if !ok {
			t.Fatalf("seed %d: generated input failed validation: %s\nGenerated payload:\n%s", seed, msg, genInput)
		}
	}
}
