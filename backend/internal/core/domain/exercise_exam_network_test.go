package domain

import (
	"errors"
	"testing"
)

func TestNormalizeExamNetwork(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "cadena vacia normaliza a none", input: "", expected: ExamNetworkNone},
		{name: "espacios en blanco normaliza a none", input: "   ", expected: ExamNetworkNone},
		{name: "none con espacios", input: " none ", expected: ExamNetworkNone},
		{name: "egress con espacios", input: " egress  ", expected: ExamNetworkEgress},
		{name: "otra cadena se conserva recortada", input: " unknown ", expected: "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeExamNetwork(tc.input)
			if got != tc.expected {
				t.Fatalf("NormalizeExamNetwork(%q) = %q; esperaba %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestValidateExamNetwork(t *testing.T) {
	tests := []struct {
		name        string
		purpose     string
		network     string
		reason      string
		expectedErr error
	}{
		{
			name:        "red none por defecto en examen sin motivo es valido",
			purpose:     string(ExercisePurposeExam),
			network:     ExamNetworkNone,
			reason:      "",
			expectedErr: nil,
		},
		{
			name:        "red vacia en examen se normaliza a none y es valido",
			purpose:     string(ExercisePurposeExam),
			network:     "",
			reason:      "",
			expectedErr: nil,
		},
		{
			name:        "red none en practica de clase es valido",
			purpose:     string(ExercisePurposeClass),
			network:     ExamNetworkNone,
			reason:      "",
			expectedErr: nil,
		},
		{
			name:        "red egress en examen con motivo suficiente es valido",
			purpose:     string(ExercisePurposeExam),
			network:     ExamNetworkEgress,
			reason:      "Necesita descargar paquetes de npm para el build",
			expectedErr: nil,
		},
		{
			name:        "red egress en examen sin motivo devuelve error",
			purpose:     string(ExercisePurposeExam),
			network:     ExamNetworkEgress,
			reason:      "",
			expectedErr: ErrExamNetworkReasonRequired,
		},
		{
			name:        "red egress en examen con motivo menor a 10 caracteres devuelve error",
			purpose:     string(ExercisePurposeExam),
			network:     ExamNetworkEgress,
			reason:      "corto",
			expectedErr: ErrExamNetworkReasonRequired,
		},
		{
			name:        "red egress en practica de clase devuelve error de proposito",
			purpose:     string(ExercisePurposeClass),
			network:     ExamNetworkEgress,
			reason:      "Necesita descargar paquetes de npm para el build",
			expectedErr: ErrExamNetworkRequiresExam,
		},
		{
			name:        "red con valor no reconocido devuelve error de politica invalida",
			purpose:     string(ExercisePurposeExam),
			network:     "internet_libre",
			reason:      "motivo cualquiera",
			expectedErr: ErrExamNetworkInvalid,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateExamNetwork(tc.purpose, tc.network, tc.reason)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("ValidateExamNetwork(%q, %q, %q) error = %v; esperaba %v",
					tc.purpose, tc.network, tc.reason, err, tc.expectedErr)
			}
		})
	}
}
