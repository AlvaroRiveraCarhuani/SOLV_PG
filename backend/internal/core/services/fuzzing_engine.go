package services

import (
	"fmt"
	"strings"

	"solv-backend/internal/core/domain"
)

// FuzzingEngine genera sistematicamente vectores de fuzzing, condiciones de borde y casos extremos.
type FuzzingEngine struct{}

func NewFuzzingEngine() *FuzzingEngine {
	return &FuzzingEngine{}
}

// GenerateTestCases sintetiza casos de prueba a partir de los tipos de parametros y categorias seleccionadas.
func (e *FuzzingEngine) GenerateTestCases(req domain.FuzzGenerationRequest) *domain.FuzzGenerationReport {
	var cases []domain.GeneratedFuzzCase
	categoriesCount := make(map[string]int)

	// Normalizar listas
	typesMap := make(map[string]bool)
	for _, t := range req.ParameterTypes {
		typesMap[strings.ToLower(strings.TrimSpace(t))] = true
	}
	if len(typesMap) == 0 {
		typesMap["integer"] = true
		typesMap["string"] = true
		typesMap["array"] = true
	}

	catMap := make(map[string]bool)
	for _, c := range req.Categories {
		catMap[strings.ToLower(strings.TrimSpace(c))] = true
	}
	if len(catMap) == 0 {
		catMap["numeric_bounds"] = true
		catMap["empty_structures"] = true
		catMap["unicode_special"] = true
		catMap["injection_escape"] = true
		catMap["large_input"] = true
	}

	// 1. Límites Numéricos
	if catMap["numeric_bounds"] {
		if typesMap["integer"] || typesMap["number"] {
			numCases := []domain.GeneratedFuzzCase{
				{
					Category:    "numeric_bounds",
					Description: "Cero absoluto (Neutro aditivo y divisor crítico)",
					Input:       "0",
					Expected:    "0",
					IsPublic:    true,
					Weight:      10,
					RiskLevel:   "boundary",
				},
				{
					Category:    "numeric_bounds",
					Description: "Entero negativo unitario (-1)",
					Input:       "-1",
					Expected:    "-1",
					IsPublic:    false,
					Weight:      10,
					RiskLevel:   "boundary",
				},
				{
					Category:    "numeric_bounds",
					Description: "Límite superior entero 32-bit con desbordamiento potencial (2^31 - 1)",
					Input:       "2147483647",
					Expected:    "2147483647",
					IsPublic:    false,
					Weight:      15,
					RiskLevel:   "critical",
				},
				{
					Category:    "numeric_bounds",
					Description: "Límite inferior entero 32-bit (-2^31)",
					Input:       "-2147483648",
					Expected:    "-2147483648",
					IsPublic:    false,
					Weight:      15,
					RiskLevel:   "critical",
				},
			}
			cases = append(cases, numCases...)
		}

		if typesMap["float"] || typesMap["double"] || typesMap["decimal"] {
			floatCases := []domain.GeneratedFuzzCase{
				{
					Category:    "numeric_bounds",
					Description: "Épsilon infinitesimal positivo (0.0000001)",
					Input:       "0.0000001",
					Expected:    "0.0000001",
					IsPublic:    false,
					Weight:      10,
					RiskLevel:   "boundary",
				},
				{
					Category:    "numeric_bounds",
					Description: "Precisión IEEE 754 de coma flotante (0.1 + 0.2)",
					Input:       "0.30000000000000004",
					Expected:    "0.3",
					IsPublic:    false,
					Weight:      15,
					RiskLevel:   "warning",
				},
			}
			cases = append(cases, floatCases...)
		}
	}

	// 2. Estructuras Vacías y Mínimas
	if catMap["empty_structures"] {
		if typesMap["string"] || typesMap["text"] {
			strCases := []domain.GeneratedFuzzCase{
				{
					Category:    "empty_structures",
					Description: "Cadena de texto completamente vacía",
					Input:       `""`,
					Expected:    `""`,
					IsPublic:    true,
					Weight:      10,
					RiskLevel:   "boundary",
				},
				{
					Category:    "empty_structures",
					Description: "Cadena compuesta exclusivamente por espacios en blanco",
					Input:       `"   "`,
					Expected:    `""`,
					IsPublic:    false,
					Weight:      10,
					RiskLevel:   "warning",
				},
			}
			cases = append(cases, strCases...)
		}

		if typesMap["array"] || typesMap["list"] || typesMap["vector"] {
			arrCases := []domain.GeneratedFuzzCase{
				{
					Category:    "empty_structures",
					Description: "Arreglo / Lista sin elementos []",
					Input:       "[]",
					Expected:    "[]",
					IsPublic:    true,
					Weight:      10,
					RiskLevel:   "boundary",
				},
				{
					Category:    "empty_structures",
					Description: "Arreglo unitario con elemento neutro [0]",
					Input:       "[0]",
					Expected:    "[0]",
					IsPublic:    false,
					Weight:      10,
					RiskLevel:   "boundary",
				},
			}
			cases = append(cases, arrCases...)
		}

		if typesMap["matrix"] || typesMap["grid"] {
			matrixCases := []domain.GeneratedFuzzCase{
				{
					Category:    "empty_structures",
					Description: "Matriz 0x0 vacía [[]]",
					Input:       "[[]]",
					Expected:    "[[]]",
					IsPublic:    false,
					Weight:      15,
					RiskLevel:   "boundary",
				},
				{
					Category:    "empty_structures",
					Description: "Matriz 1x1 unitaria [[0]]",
					Input:       "[[0]]",
					Expected:    "[[0]]",
					IsPublic:    false,
					Weight:      10,
					RiskLevel:   "boundary",
				},
			}
			cases = append(cases, matrixCases...)
		}
	}

	// 3. Unicode y Caracteres Especiales
	if catMap["unicode_special"] {
		if typesMap["string"] || typesMap["text"] {
			unicodeCases := []domain.GeneratedFuzzCase{
				{
					Category:    "unicode_special",
					Description: "Caracteres multibyte UTF-8 con tildes y diacríticos",
					Input:       `"Árbol de computación lingüística con ñ y diéresis ü"`,
					Expected:    `"Árbol de computación lingüística con ñ y diéresis ü"`,
					IsPublic:    false,
					Weight:      10,
					RiskLevel:   "warning",
				},
				{
					Category:    "unicode_special",
					Description: "Caracteres CJK y Emojis de 4 bytes",
					Input:       `"こんにちは世界 🚀💻🔥"`,
					Expected:    `"こんにちは世界 🚀💻🔥"`,
					IsPublic:    false,
					Weight:      15,
					RiskLevel:   "warning",
				},
				{
					Category:    "unicode_special",
					Description: "Secuencias de control y saltos de línea (\\n\\r\\t)",
					Input:       `"Linea1\nLinea2\r\n\tTabulacion"`,
					Expected:    `"Linea1 Linea2 Tabulacion"`,
					IsPublic:    false,
					Weight:      10,
					RiskLevel:   "warning",
				},
			}
			cases = append(cases, unicodeCases...)
		}
	}

	// 4. Inyecciones y Escapes
	if catMap["injection_escape"] {
		if typesMap["sql"] || typesMap["database"] || typesMap["string"] {
			injCases := []domain.GeneratedFuzzCase{
				{
					Category:    "injection_escape",
					Description: "Vector de inyección SQL clásico (' OR 1=1 --)",
					Input:       `"' OR '1'='1' --"`,
					Expected:    `"Invalid input or sanitized query"`,
					IsPublic:    false,
					Weight:      20,
					RiskLevel:   "critical",
				},
				{
					Category:    "injection_escape",
					Description: "Comillas dobles, simples y barras invertidas sin escapar",
					Input:       `"\"'\\\\'; DROP TABLE students; --"`,
					Expected:    `"Sanitized"`,
					IsPublic:    false,
					Weight:      20,
					RiskLevel:   "critical",
				},
			}
			cases = append(cases, injCases...)
		}
	}

	// 5. Escala Masiva y Rendimiento
	if catMap["large_input"] {
		if typesMap["array"] || typesMap["list"] {
			cases = append(cases, domain.GeneratedFuzzCase{
				Category:    "large_input",
				Description: "Arreglo masivo de 1000 elementos ordenados descendentemente (Worst case)",
				Input:       generateDescendingArray(1000),
				Expected:    "OK",
				IsPublic:    false,
				Weight:      25,
				RiskLevel:   "stress",
			})
		}
		if typesMap["string"] {
			cases = append(cases, domain.GeneratedFuzzCase{
				Category:    "large_input",
				Description: "Cadena de texto de 5000 caracteres repetitivos",
				Input:       fmt.Sprintf(`"%s"`, strings.Repeat("A", 5000)),
				Expected:    "OK",
				IsPublic:    false,
				Weight:      20,
				RiskLevel:   "stress",
			})
		}
	}

	// Limitar cantidad si se especifica
	if req.Count > 0 && len(cases) > req.Count {
		cases = cases[:req.Count]
	}

	for _, c := range cases {
		categoriesCount[c.Category]++
	}

	return &domain.FuzzGenerationReport{
		TotalGenerated: len(cases),
		Categories:     categoriesCount,
		Cases:          cases,
	}
}

func generateDescendingArray(size int) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i := size; i >= 1; i-- {
		sb.WriteString(fmt.Sprintf("%d", i))
		if i > 1 {
			sb.WriteString(",")
		}
	}
	sb.WriteString("]")
	return sb.String()
}
