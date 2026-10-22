package services_test

import (
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func customRule(pattern, ruleType, message string) domain.ASTCustomRule {
	return domain.ASTCustomRule{
		Language: "python",
		Pattern:  pattern,
		Type:     ruleType,
		Message:  message,
	}
}

func TestStaticASTAnalyzer_ValidateCustomRules(t *testing.T) {
	analyzer := services.NewStaticASTAnalyzer()

	tests := []struct {
		name        string
		code        string
		rules       domain.ASTRules
		wantMatch   bool
		wantPattern string
		wantTarget  string
		wantLine    int
	}{
		{
			name: "prohibe una funcion usada en el codigo",
			code: "valores = [3, 1, 2]\nprint(sorted(valores))\n",
			rules: domain.ASTRules{
				CustomRules: []domain.ASTCustomRule{customRule("sorted", "function", "No usar sorted")},
			},
			wantMatch:   true,
			wantPattern: "sorted",
			wantTarget:  "sorted",
			wantLine:    2,
		},
		{
			name: "no dispara por un identificador que solo contiene el patron",
			code: "is_sorted = True\nprint(is_sorted)\n",
			rules: domain.ASTRules{
				CustomRules: []domain.ASTCustomRule{customRule("sorted", "function", "No usar sorted")},
			},
			wantMatch: false,
		},
		{
			name: "no dispara por el patron dentro de un comentario",
			code: "# aqui usamos sorted() a proposito\nprint(1)\n",
			rules: domain.ASTRules{
				CustomRules: []domain.ASTCustomRule{customRule("sorted", "function", "No usar sorted")},
			},
			wantMatch: false,
		},
		{
			name: "no dispara por el patron dentro de un literal de cadena",
			code: "mensaje = 'no usar sorted aqui'\nprint(mensaje)\n",
			rules: domain.ASTRules{
				CustomRules: []domain.ASTCustomRule{customRule("sorted", "function", "No usar sorted")},
			},
			wantMatch: false,
		},
		{
			name: "detecta una regla con varios identificadores separados por coma",
			code: "print(reversed(lista))\n",
			rules: domain.ASTRules{
				CustomRules: []domain.ASTCustomRule{customRule("sorted, reversed", "function", "No usar funciones nativas de orden")},
			},
			wantMatch:   true,
			wantPattern: "reversed",
			wantTarget:  "reversed",
			wantLine:    1,
		},
		{
			name: "detecta el nombre cualificado de un metodo",
			code: "import math\nprint(math.sqrt(16))\n",
			rules: domain.ASTRules{
				CustomRules: []domain.ASTCustomRule{customRule("math.sqrt", "method", "No usar sqrt")},
			},
			wantMatch:   true,
			wantPattern: "math.sqrt",
			wantTarget:  "math.sqrt",
			wantLine:    2,
		},
		{
			name: "ignora las reglas declaradas para otro lenguaje",
			code: "print(sorted([2, 1]))\n",
			rules: domain.ASTRules{
				CustomRules: []domain.ASTCustomRule{{
					Language: "java",
					Pattern:  "sorted",
					Type:     "function",
					Message:  "No usar sorted",
				}},
			},
			wantMatch: false,
		},
		{
			name: "detecta una regla de sintaxis (bucles bloqueados)",
			code: "for i in range(3):\n    print(i)\n",
			rules: domain.ASTRules{
				CustomRules: []domain.ASTCustomRule{customRule("for,while", "syntax", "Resolucion recursiva pura")},
			},
			wantMatch:   true,
			wantPattern: "for",
			wantTarget:  "for",
			wantLine:    1,
		},
		{
			name: "sin reglas personalizadas no hay infraccion",
			code: "print(sorted([2, 1]))\n",
			rules: domain.ASTRules{
				CustomRules: nil,
			},
			wantMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violation := analyzer.ValidateCustomRules("python", tc.code, tc.rules)

			if !tc.wantMatch {
				if violation != nil {
					t.Fatalf("se esperaba ninguna infraccion y se obtuvo %+v", violation)
				}
				return
			}

			if violation == nil {
				t.Fatal("se esperaba una infraccion y no se obtuvo ninguna")
			}
			if violation.Pattern != tc.wantPattern {
				t.Errorf("patron esperado %q, obtenido %q", tc.wantPattern, violation.Pattern)
			}
			if violation.Target != tc.wantTarget {
				t.Errorf("target esperado %q, obtenido %q", tc.wantTarget, violation.Target)
			}
			if violation.Line != tc.wantLine {
				t.Errorf("linea esperada %d, obtenida %d", tc.wantLine, violation.Line)
			}
			if violation.RuleID == "" {
				t.Error("el rule_id no puede estar vacio")
			}
			if violation.Message == "" {
				t.Error("el mensaje de la infraccion no puede estar vacio")
			}
		})
	}
}

func TestStaticASTAnalyzer_ValidateCustomRules_NoFalsePositiveOnCommentsAndStrings(t *testing.T) {
	analyzer := services.NewStaticASTAnalyzer()

	code := "# sorted() esta prohibido\ntexto = 'sorted' + \"reversed\"\nprint(texto)\n"
	violation := analyzer.ValidateCustomRules("python", code, domain.ASTRules{
		CustomRules: []domain.ASTCustomRule{customRule("sorted", "function", "No usar sorted")},
	})

	if violation != nil {
		t.Fatalf("no se esperaba infraccion, se obtuvo %+v", violation)
	}
}
