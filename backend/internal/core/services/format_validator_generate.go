package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

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

// GenerateCase genera un caso aleatorio válido respetando el contrato y la semilla dada.
func (v *DefaultFormatValidator) GenerateCase(contract json.RawMessage, seed int64) (string, error) {
	if len(contract) == 0 || string(contract) == "null" || string(contract) == "{}" {
		return "", errors.New("contrato de formato vacío o nulo")
	}

	var c InputFormatContract
	if err := json.Unmarshal(contract, &c); err != nil {
		return "", fmt.Errorf("formato de contrato inválido: %w", err)
	}

	// Generacion pseudoaleatoria determinista controlada por semilla para reproducibilidad de casos.
	//nolint:gosec
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
			fmt.Fprintf(&sb, "%d\n", val)

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
