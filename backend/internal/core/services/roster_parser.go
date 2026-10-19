package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

var emailRegex = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

// ParsedRosterResult contiene el resumen estructurado de la lectura del archivo de nómina
type ParsedRosterResult struct {
	TotalDetected   int      `json:"total_detected"`
	ValidEmails     []string `json:"valid_emails"`
	DuplicateEmails []string `json:"duplicate_emails"`
	InvalidEmails   []string `json:"invalid_emails"`
}

// RosterParser provee métodos tolerantes para extraer correos institucionales desde .xlsx y .csv
type RosterParser struct{}

// NewRosterParser crea una nueva instancia del parser de planillas
func NewRosterParser() *RosterParser {
	return &RosterParser{}
}

// Parse extrae y valida correos desde un archivo subido
func (p *RosterParser) Parse(r io.Reader, filename string) (*ParsedRosterResult, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	var rawRows [][]string
	var err error

	if ext == ".xlsx" || ext == ".xlsm" || ext == ".xltx" {
		rawRows, err = p.parseXLSX(r)
	} else {
		// Por defecto asumimos CSV / TXT delimitado
		rawRows, err = p.parseCSV(r)
	}

	if err != nil {
		return nil, fmt.Errorf("error al leer archivo de nómina: %w", err)
	}

	return p.processRows(rawRows), nil
}

func (p *RosterParser) parseCSV(r io.Reader) ([][]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Detectar delimitador común (coma, punto y coma, tabulador)
	delimiter := rune(',')
	sample := string(data)
	if strings.Count(sample, ";") > strings.Count(sample, ",") {
		delimiter = ';'
	} else if strings.Count(sample, "\t") > strings.Count(sample, ",") {
		delimiter = '\t'
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1 // Permite filas de longitud variable
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	return reader.ReadAll()
}

func (p *RosterParser) parseXLSX(r io.Reader) ([][]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("el archivo Excel no contiene hojas de cálculo")
	}

	// Procesar la primera hoja activa
	return f.GetRows(sheets[0])
}

func (p *RosterParser) processRows(rows [][]string) *ParsedRosterResult {
	result := &ParsedRosterResult{
		ValidEmails:     make([]string, 0),
		DuplicateEmails: make([]string, 0),
		InvalidEmails:   make([]string, 0),
	}

	if len(rows) == 0 {
		return result
	}

	// 1. Detección heurística de columna de correo por cabecera
	emailColIdx := -1
	headerRow := rows[0]
	for idx, cell := range headerRow {
		clean := strings.ToLower(strings.TrimSpace(cell))
		if strings.Contains(clean, "correo") ||
			strings.Contains(clean, "email") ||
			strings.Contains(clean, "e-mail") ||
			strings.Contains(clean, "mail") {
			emailColIdx = idx
			break
		}
	}

	seen := make(map[string]bool)
	var rawCandidates []string

	// Si identificamos la columna, priorizamos leer esa columna
	startRow := 1
	if emailColIdx == -1 {
		startRow = 0 // Puede no tener cabeceras, escanear todo
	}

	for r := startRow; r < len(rows); r++ {
		row := rows[r]
		if emailColIdx >= 0 && emailColIdx < len(row) {
			val := strings.TrimSpace(row[emailColIdx])
			if val != "" {
				rawCandidates = append(rawCandidates, val)
			}
		} else {
			// Escanear todas las celdas de la fila buscando patrones de correo
			for _, cell := range row {
				val := strings.TrimSpace(cell)
				if val == "" {
					continue
				}
				matches := emailRegex.FindAllString(val, -1)
				rawCandidates = append(rawCandidates, matches...)
			}
		}
	}

	result.TotalDetected = len(rawCandidates)

	for _, cand := range rawCandidates {
		// Limpieza de espacios, comillas o caracteres espurios
		cand = strings.Trim(strings.ToLower(strings.TrimSpace(cand)), `"'<>;`)

		// Validar sintaxis RFC de correo
		if !isValidEmailAddress(cand) {
			result.InvalidEmails = append(result.InvalidEmails, cand)
			continue
		}

		if seen[cand] {
			result.DuplicateEmails = append(result.DuplicateEmails, cand)
		} else {
			seen[cand] = true
			result.ValidEmails = append(result.ValidEmails, cand)
		}
	}

	return result
}

func isValidEmailAddress(e string) bool {
	if len(e) < 5 || len(e) > 254 {
		return false
	}
	if !emailRegex.MatchString(e) {
		return false
	}
	_, err := mail.ParseAddress(e)
	return err == nil
}
