package services

import (
	"strings"
	"testing"
)

func TestRosterParser_CSVWithSynonymHeaders(t *testing.T) {
	parser := NewRosterParser()

	csvContent := `Nombre Completo,Correo Institucional del Docente,Departamento
Carlos Mendoza,cmendoza@uab.edu.bo,Sistemas
Ana Gutierrez,agutierrez@uab.edu.bo,Redes
Jorge Torrez,jtorrez@uab.edu.bo,Programacion
`
	res, err := parser.Parse(strings.NewReader(csvContent), "docentes.csv")
	if err != nil {
		t.Fatalf("Unexpected parse error: %v", err)
	}

	if res.TotalDetected != 3 {
		t.Errorf("Expected 3 detected emails, got %d", res.TotalDetected)
	}
	if len(res.ValidEmails) != 3 {
		t.Errorf("Expected 3 valid emails, got %d", len(res.ValidEmails))
	}
	if res.ValidEmails[0] != "cmendoza@uab.edu.bo" {
		t.Errorf("Unexpected first email: %s", res.ValidEmails[0])
	}
}

func TestRosterParser_CSVWithSemicolonsAndDuplicates(t *testing.T) {
	parser := NewRosterParser()

	csvContent := `ID;Docente;E-mail;Observaciones
1;Juan Perez;jperez@uab.edu.bo;Titular
2;Juan Perez;jperez@uab.edu.bo;Duplicado
3;Invalido;correo-invalido;Error
4;Maria Soto;msoto@uab.edu.bo;Auxiliar
`
	res, err := parser.Parse(strings.NewReader(csvContent), "nomina.csv")
	if err != nil {
		t.Fatalf("Unexpected parse error: %v", err)
	}

	if len(res.ValidEmails) != 2 {
		t.Errorf("Expected 2 valid unique emails, got %d", len(res.ValidEmails))
	}
	if len(res.DuplicateEmails) != 1 {
		t.Errorf("Expected 1 duplicate email, got %d", len(res.DuplicateEmails))
	}
	if len(res.InvalidEmails) != 1 {
		t.Errorf("Expected 1 invalid email, got %d", len(res.InvalidEmails))
	}
}

func TestRosterParser_RegexFallbackNoHeaders(t *testing.T) {
	parser := NewRosterParser()

	rawText := `Datos aleatorios sin orden
profesor1@uab.edu.bo en la celda A
alguna otra cosa 12345
segundo profesor profesor2@uab.edu.bo
`
	res, err := parser.Parse(strings.NewReader(rawText), "lista.txt")
	if err != nil {
		t.Fatalf("Unexpected parse error: %v", err)
	}

	if len(res.ValidEmails) != 2 {
		t.Errorf("Expected 2 valid emails extracted via regex, got %d", len(res.ValidEmails))
	}
}
