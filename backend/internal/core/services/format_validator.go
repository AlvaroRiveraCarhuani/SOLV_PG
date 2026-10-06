package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

// FormatValidator define el contrato del validador y generador estructural.
type FormatValidator interface {
	ValidateContract(contract json.RawMessage) error
	ValidateCase(contract json.RawMessage, input string) (bool, string)
	GenerateCase(contract json.RawMessage, seed int64) (string, error)
}

type DefaultFormatValidator struct{}

func NewFormatValidator() FormatValidator {
	return &DefaultFormatValidator{}
}

// Estructuras del Contrato JSONB

type ContractLineItem struct {
	Type *string `json:"type,omitempty"`
	Min  any     `json:"min,omitempty"`
	Max  any     `json:"max,omitempty"`
}

type ContractEdgeField struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"` // "int" o "string"
	Min  any    `json:"min,omitempty"`
	Max  any    `json:"max,omitempty"`
}

type ContractLine struct {
	ID     string              `json:"id,omitempty"`
	Type   string              `json:"type"` // "int", "ints", "matrix", "edges", "string", "raw"
	Min    any                 `json:"min,omitempty"`
	Max    any                 `json:"max,omitempty"`
	Count  any                 `json:"count,omitempty"`
	Rows   any                 `json:"rows,omitempty"`
	Cols   any                 `json:"cols,omitempty"`
	Item   *ContractLineItem   `json:"item,omitempty"`
	Fields []ContractEdgeField `json:"fields,omitempty"`
	Regex  string              `json:"regex,omitempty"`
	MinLen *int                `json:"min_len,omitempty"`
	MaxLen *int                `json:"max_len,omitempty"`
}

type InputFormatContract struct {
	Version int `json:"version"`
	Input   struct {
		Lines []ContractLine `json:"lines"`
	} `json:"input"`
}

// ValidateContract valida la consistencia estática del esquema del contrato.
func (v *DefaultFormatValidator) ValidateContract(contract json.RawMessage) error {
	if len(contract) == 0 || string(contract) == "null" || string(contract) == "{}" {
		return nil
	}

	var c InputFormatContract
	if err := json.Unmarshal(contract, &c); err != nil {
		return fmt.Errorf("formato JSON del contrato inválido: %w", err)
	}

	if len(c.Input.Lines) == 0 {
		return nil
	}

	declaredIDs := make(map[string]bool)

	for idx, line := range c.Input.Lines {
		lineNum := idx + 1
		validTypes := map[string]bool{
			"int": true, "ints": true, "matrix": true,
			"edges": true, "string": true, "raw": true,
		}
		if !validTypes[line.Type] {
			return fmt.Errorf("regla %d: tipo de línea no soportado %q", lineNum, line.Type)
		}

		if line.ID != "" {
			if declaredIDs[line.ID] {
				return fmt.Errorf("regla %d: identificador duplicado %q", lineNum, line.ID)
			}
			declaredIDs[line.ID] = true
		}

		// Validar referencias en campos dinámicos
		checkRef := func(field string, val any) error {
			if val == nil {
				return nil
			}
			if strVal, ok := val.(string); ok && strings.HasPrefix(strVal, "=") {
				refID := strVal[1:]
				if !declaredIDs[refID] {
					return fmt.Errorf("regla %d (%s): referencia a variable %q inexistente o declarada posteriormente", lineNum, field, refID)
				}
			}
			return nil
		}

		if err := checkRef("min", line.Min); err != nil {
			return err
		}
		if err := checkRef("max", line.Max); err != nil {
			return err
		}
		if err := checkRef("count", line.Count); err != nil {
			return err
		}
		if err := checkRef("rows", line.Rows); err != nil {
			return err
		}
		if err := checkRef("cols", line.Cols); err != nil {
			return err
		}

		if line.Item != nil {
			if err := checkRef("item.min", line.Item.Min); err != nil {
				return err
			}
			if err := checkRef("item.max", line.Item.Max); err != nil {
				return err
			}
		}

		for fIdx, f := range line.Fields {
			fName := fmt.Sprintf("fields[%d]", fIdx)
			if err := checkRef(fName+".min", f.Min); err != nil {
				return err
			}
			if err := checkRef(fName+".max", f.Max); err != nil {
				return err
			}
		}

		if line.Regex != "" {
			if _, err := regexp.Compile(line.Regex); err != nil {
				return fmt.Errorf("regla %d: expresión regular inválida %q: %w", lineNum, line.Regex, err)
			}
		}
	}

	return nil
}

// evalDynamicInt evalúa un entero estático o una referencia "=id" en base al contexto acumulado.
func evalDynamicInt(val any, ctx map[string]int64, defaultVal int64) (int64, error) {
	if val == nil {
		return defaultVal, nil
	}
	switch v := val.(type) {
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		if strings.HasPrefix(v, "=") {
			refID := v[1:]
			val, ok := ctx[refID]
			if !ok {
				return 0, fmt.Errorf("variable referenciada %q no ha sido definida aún", refID)
			}
			return val, nil
		}
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("valor numérico inválido %q", v)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("tipo de valor numérico no soportado: %T", val)
	}
}

// ValidateCase valida una entrada textual contra el contrato. Devuelve (válido, mensaje en español).
func (v *DefaultFormatValidator) ValidateCase(contract json.RawMessage, input string) (bool, string) {
	if len(contract) == 0 || string(contract) == "null" || string(contract) == "{}" {
		return true, ""
	}

	var c InputFormatContract
	if err := json.Unmarshal(contract, &c); err != nil {
		return false, fmt.Sprintf("Error interno al decodificar contrato: %v", err)
	}

	if len(c.Input.Lines) == 0 {
		return true, ""
	}

	// Normalizar finales de línea \r\n -> \n
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")

	// Si la última línea es vacía por salto de línea final, quitarla si excede lo requerido
	if len(lines) > 0 && lines[len(lines)-1] == "" && len(lines) > 1 {
		lines = lines[:len(lines)-1]
	}

	ctx := make(map[string]int64)
	currLineIdx := 0

	for ruleIdx, rule := range c.Input.Lines {
		ruleNum := ruleIdx + 1

		if currLineIdx >= len(lines) {
			return false, fmt.Sprintf("Línea %d: se esperaba regla %d (%s), pero la entrada terminó prematuramente", currLineIdx+1, ruleNum, rule.Type)
		}

		switch rule.Type {
		case "int":
			lineNum := currLineIdx + 1
			lineContent := strings.TrimSpace(lines[currLineIdx])
			currLineIdx++

			if lineContent == "" {
				return false, fmt.Sprintf("Línea %d: se esperaba un entero, se encontró una línea vacía", lineNum)
			}
			tokens := strings.Fields(lineContent)
			if len(tokens) != 1 {
				return false, fmt.Sprintf("Línea %d: se esperaba exactamente 1 entero, se encontraron %d elementos", lineNum, len(tokens))
			}
			num, err := strconv.ParseInt(tokens[0], 10, 64)
			if err != nil {
				return false, fmt.Sprintf("Línea %d: %q no es un entero válido", lineNum, tokens[0])
			}

			if rule.Min != nil {
				minVal, err := evalDynamicInt(rule.Min, ctx, 0)
				if err != nil {
					return false, fmt.Sprintf("Línea %d: error al evaluar límite mínimo: %v", lineNum, err)
				}
				if num < minVal {
					return false, fmt.Sprintf("Línea %d: el valor %d es menor que el mínimo permitido (%d)", lineNum, num, minVal)
				}
			}
			if rule.Max != nil {
				maxVal, err := evalDynamicInt(rule.Max, ctx, 0)
				if err != nil {
					return false, fmt.Sprintf("Línea %d: error al evaluar límite máximo: %v", lineNum, err)
				}
				if num > maxVal {
					return false, fmt.Sprintf("Línea %d: el valor %d es mayor que el máximo permitido (%d)", lineNum, num, maxVal)
				}
			}

			if rule.ID != "" {
				ctx[rule.ID] = num
			}

		case "ints":
			lineNum := currLineIdx + 1
			lineContent := strings.TrimSpace(lines[currLineIdx])
			currLineIdx++

			tokens := strings.Fields(lineContent)
			expectedCount := int64(len(tokens))
			if rule.Count != nil {
				ec, err := evalDynamicInt(rule.Count, ctx, int64(len(tokens)))
				if err != nil {
					return false, fmt.Sprintf("Línea %d: error al evaluar count: %v", lineNum, err)
				}
				expectedCount = ec
			}

			if int64(len(tokens)) != expectedCount {
				countDesc := fmt.Sprintf("%d", expectedCount)
				if strCount, ok := rule.Count.(string); ok && strings.HasPrefix(strCount, "=") {
					refID := strCount[1:]
					countDesc = fmt.Sprintf("%d (declarado %q, %s=%d)", expectedCount, strCount, refID, ctx[refID])
				}
				return false, fmt.Sprintf("Línea %d: se esperaban %s enteros, llegaron %d", lineNum, countDesc, len(tokens))
			}

			var minVal *int64
			var maxVal *int64
			if rule.Item != nil {
				if rule.Item.Min != nil {
					m, err := evalDynamicInt(rule.Item.Min, ctx, 0)
					if err != nil {
						return false, fmt.Sprintf("Línea %d: error en item.min: %v", lineNum, err)
					}
					minVal = &m
				}
				if rule.Item.Max != nil {
					m, err := evalDynamicInt(rule.Item.Max, ctx, 0)
					if err != nil {
						return false, fmt.Sprintf("Línea %d: error en item.max: %v", lineNum, err)
					}
					maxVal = &m
				}
			}

			for idx, tok := range tokens {
				num, err := strconv.ParseInt(tok, 10, 64)
				if err != nil {
					return false, fmt.Sprintf("Línea %d (elemento %d): %q no es un entero válido", lineNum, idx+1, tok)
				}
				if minVal != nil && num < *minVal {
					return false, fmt.Sprintf("Línea %d (elemento %d): el valor %d es menor que el mínimo permitido (%d)", lineNum, idx+1, num, *minVal)
				}
				if maxVal != nil && num > *maxVal {
					return false, fmt.Sprintf("Línea %d (elemento %d): el valor %d es mayor que el máximo permitido (%d)", lineNum, idx+1, num, *maxVal)
				}
			}

		case "matrix":
			rows, err := evalDynamicInt(rule.Rows, ctx, 1)
			if err != nil {
				return false, fmt.Sprintf("Línea %d: error al evaluar matrix rows: %v", currLineIdx+1, err)
			}
			cols, err := evalDynamicInt(rule.Cols, ctx, 1)
			if err != nil {
				return false, fmt.Sprintf("Línea %d: error al evaluar matrix cols: %v", currLineIdx+1, err)
			}

			var minVal *int64
			var maxVal *int64
			if rule.Item != nil {
				if rule.Item.Min != nil {
					m, _ := evalDynamicInt(rule.Item.Min, ctx, 0)
					minVal = &m
				}
				if rule.Item.Max != nil {
					m, _ := evalDynamicInt(rule.Item.Max, ctx, 0)
					maxVal = &m
				}
			}

			for r := int64(0); r < rows; r++ {
				if currLineIdx >= len(lines) {
					return false, fmt.Sprintf("Línea %d: matriz incompleta (se esperaban %d filas, solo llegaron %d)", currLineIdx+1, rows, r)
				}
				lineNum := currLineIdx + 1
				lineContent := strings.TrimSpace(lines[currLineIdx])
				currLineIdx++

				tokens := strings.Fields(lineContent)
				if int64(len(tokens)) != cols {
					return false, fmt.Sprintf("Línea %d (fila %d de matriz): se esperaban %d columnas, llegaron %d", lineNum, r+1, cols, len(tokens))
				}
				for c, tok := range tokens {
					num, err := strconv.ParseInt(tok, 10, 64)
					if err != nil {
						return false, fmt.Sprintf("Línea %d (fila %d, col %d): %q no es un entero válido", lineNum, r+1, c+1, tok)
					}
					if minVal != nil && num < *minVal {
						return false, fmt.Sprintf("Línea %d (fila %d, col %d): valor %d menor que el mínimo (%d)", lineNum, r+1, c+1, num, *minVal)
					}
					if maxVal != nil && num > *maxVal {
						return false, fmt.Sprintf("Línea %d (fila %d, col %d): valor %d mayor que el máximo (%d)", lineNum, r+1, c+1, num, *maxVal)
					}
				}
			}

		case "edges":
			count, err := evalDynamicInt(rule.Count, ctx, 1)
			if err != nil {
				return false, fmt.Sprintf("Línea %d: error al evaluar count de edges: %v", currLineIdx+1, err)
			}

			expectedFields := len(rule.Fields)
			if expectedFields == 0 {
				expectedFields = 2 // default (u, v)
			}

			for e := int64(0); e < count; e++ {
				if currLineIdx >= len(lines) {
					return false, fmt.Sprintf("Línea %d: aristas incompletas (se esperaban %d aristas, solo llegaron %d)", currLineIdx+1, count, e)
				}
				lineNum := currLineIdx + 1
				lineContent := strings.TrimSpace(lines[currLineIdx])
				currLineIdx++

				tokens := strings.Fields(lineContent)
				if len(tokens) != expectedFields {
					return false, fmt.Sprintf("Línea %d (arista %d): se esperaban %d valores, llegaron %d", lineNum, e+1, expectedFields, len(tokens))
				}

				for fIdx, tok := range tokens {
					var minVal *int64
					var maxVal *int64
					if fIdx < len(rule.Fields) {
						if rule.Fields[fIdx].Min != nil {
							m, _ := evalDynamicInt(rule.Fields[fIdx].Min, ctx, 0)
							minVal = &m
						}
						if rule.Fields[fIdx].Max != nil {
							m, _ := evalDynamicInt(rule.Fields[fIdx].Max, ctx, 0)
							maxVal = &m
						}
					}

					num, err := strconv.ParseInt(tok, 10, 64)
					if err != nil {
						return false, fmt.Sprintf("Línea %d (arista %d, campo %d): %q no es un entero válido", lineNum, e+1, fIdx+1, tok)
					}
					if minVal != nil && num < *minVal {
						return false, fmt.Sprintf("Línea %d (arista %d, campo %d): valor %d menor que el mínimo (%d)", lineNum, e+1, fIdx+1, num, *minVal)
					}
					if maxVal != nil && num > *maxVal {
						return false, fmt.Sprintf("Línea %d (arista %d, campo %d): valor %d mayor que el máximo (%d)", lineNum, e+1, fIdx+1, num, *maxVal)
					}
				}
			}

		case "string":
			lineNum := currLineIdx + 1
			lineContent := lines[currLineIdx]
			currLineIdx++

			if rule.MinLen != nil && len(lineContent) < *rule.MinLen {
				return false, fmt.Sprintf("Línea %d: la cadena tiene longitud %d, menor que el mínimo (%d)", lineNum, len(lineContent), *rule.MinLen)
			}
			if rule.MaxLen != nil && len(lineContent) > *rule.MaxLen {
				return false, fmt.Sprintf("Línea %d: la cadena tiene longitud %d, mayor que el máximo (%d)", lineNum, len(lineContent), *rule.MaxLen)
			}
			if rule.Regex != "" {
				re, err := regexp.Compile(rule.Regex)
				if err == nil && !re.MatchString(lineContent) {
					return false, fmt.Sprintf("Línea %d: la cadena %q no coincide con la expresión regular %q", lineNum, lineContent, rule.Regex)
				}
			}

		case "raw":
			// Escotilla de escape: consume todo lo restante
			currLineIdx = len(lines)
		}
	}

	if currLineIdx < len(lines) {
		extraCount := len(lines) - currLineIdx
		return false, fmt.Sprintf("Línea %d: la entrada contiene %d línea(s) adicional(es) no especificadas en el contrato", currLineIdx+1, extraCount)
	}

	return true, ""
}

// GenerateCase genera un caso aleatorio válido respetando el contrato y la semilla dada.
func (v *DefaultFormatValidator) GenerateCase(contract json.RawMessage, seed int64) (string, error) {
	if len(contract) == 0 || string(contract) == "null" || string(contract) == "{}" {
		return "", errors.New("contrato de formato vacío o nulo")
	}

	var c InputFormatContract
	if err := json.Unmarshal(contract, &c); err != nil {
		return "", fmt.Errorf("formato de contrato inválido: %w", err)
	}

	r := rand.New(rand.NewSource(seed))
	ctx := make(map[string]int64)
	var sb strings.Builder

	randIntRange := func(min, max int64) int64 {
		if min > max {
			min, max = max, min
		}
		if min == max {
			return min
		}
		return min + r.Int63n(max-min+1)
	}

	for _, line := range c.Input.Lines {
		switch line.Type {
		case "int":
			minVal, _ := evalDynamicInt(line.Min, ctx, 1)
			maxVal, _ := evalDynamicInt(line.Max, ctx, 100)
			if maxVal < minVal {
				maxVal = minVal + 10
			}
			val := randIntRange(minVal, maxVal)
			if line.ID != "" {
				ctx[line.ID] = val
			}
			sb.WriteString(fmt.Sprintf("%d\n", val))

		case "ints":
			count, _ := evalDynamicInt(line.Count, ctx, 5)
			if count < 0 {
				count = 1
			}
			minVal := int64(1)
			maxVal := int64(100)
			if line.Item != nil {
				if line.Item.Min != nil {
					minVal, _ = evalDynamicInt(line.Item.Min, ctx, 1)
				}
				if line.Item.Max != nil {
					maxVal, _ = evalDynamicInt(line.Item.Max, ctx, 100)
				}
			}
			var nums []string
			for i := int64(0); i < count; i++ {
				nums = append(nums, strconv.FormatInt(randIntRange(minVal, maxVal), 10))
			}
			sb.WriteString(strings.Join(nums, " ") + "\n")

		case "matrix":
			rows, _ := evalDynamicInt(line.Rows, ctx, 3)
			cols, _ := evalDynamicInt(line.Cols, ctx, 3)
			if rows < 1 {
				rows = 1
			}
			if cols < 1 {
				cols = 1
			}
			minVal := int64(0)
			maxVal := int64(100)
			if line.Item != nil {
				if line.Item.Min != nil {
					minVal, _ = evalDynamicInt(line.Item.Min, ctx, 0)
				}
				if line.Item.Max != nil {
					maxVal, _ = evalDynamicInt(line.Item.Max, ctx, 100)
				}
			}
			for i := int64(0); i < rows; i++ {
				var rowNums []string
				for j := int64(0); j < cols; j++ {
					rowNums = append(rowNums, strconv.FormatInt(randIntRange(minVal, maxVal), 10))
				}
				sb.WriteString(strings.Join(rowNums, " ") + "\n")
			}

		case "edges":
			count, _ := evalDynamicInt(line.Count, ctx, 3)
			if count < 1 {
				count = 1
			}
			fields := line.Fields
			if len(fields) == 0 {
				fields = []ContractEdgeField{
					{Type: "int", Min: 1, Max: 10},
					{Type: "int", Min: 1, Max: 10},
				}
			}
			for i := int64(0); i < count; i++ {
				var edgeVals []string
				for _, f := range fields {
					minVal, _ := evalDynamicInt(f.Min, ctx, 1)
					maxVal, _ := evalDynamicInt(f.Max, ctx, 10)
					edgeVals = append(edgeVals, strconv.FormatInt(randIntRange(minVal, maxVal), 10))
				}
				sb.WriteString(strings.Join(edgeVals, " ") + "\n")
			}

		case "string":
			length := 10
			if line.MaxLen != nil && *line.MaxLen > 0 {
				length = *line.MaxLen
				if length > 20 {
					length = 15
				}
			}
			if line.MinLen != nil && *line.MinLen > length {
				length = *line.MinLen
			}
			const chars = "abcdefghijklmnopqrstuvwxyz"
			var strBytes []byte
			for i := 0; i < length; i++ {
				strBytes = append(strBytes, chars[r.Intn(len(chars))])
			}
			sb.WriteString(string(strBytes) + "\n")

		case "raw":
			sb.WriteString("raw sample data\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n"), nil
}
