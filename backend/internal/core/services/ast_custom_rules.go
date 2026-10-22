package services

import (
	"fmt"
	"strings"

	"solv-backend/internal/core/domain"
)

// customRuleToken representa un identificador del código fuente del estudiante
// con su posición (línea) y su nombre cualificado cuando forma parte de una
// cadena de acceso con puntos (por ejemplo "math.sqrt").
type customRuleToken struct {
	text      string
	line      int
	dotBefore bool
	qualified string
}

// lexIdentifiers extrae los identificadores del código fuente ignorando
// comentarios y literales de cadena. Sin este filtrado, una prohibición como
// "sorted" dispararía falsos positivos sobre el comentario "# usar sorted()"
// o sobre la cadena "el algoritmo sorted" (GOB-SEGURO-02).
func lexIdentifiers(code string) []customRuleToken {
	tokens := make([]customRuleToken, 0, 64)
	line := 1
	pendingDot := false

	for i := 0; i < len(code); {
		ch := code[i]

		switch {
		case ch == '\n':
			line++
			pendingDot = false
			i++
		case ch == ' ' || ch == '\t' || ch == '\r':
			i++
		case ch == '#':
			for i < len(code) && code[i] != '\n' {
				i++
			}
		case ch == '/' && i+1 < len(code) && code[i+1] == '/':
			for i < len(code) && code[i] != '\n' {
				i++
			}
		case ch == '/' && i+1 < len(code) && code[i+1] == '*':
			i += 2
			for i < len(code) {
				if code[i] == '\n' {
					line++
				}
				if code[i] == '*' && i+1 < len(code) && code[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
		case ch == '"' || ch == '\'' || ch == '`':
			pendingDot = false
			i = skipStringLiteral(code, i, &line)
		case ch == '.':
			pendingDot = true
			i++
		case ch >= '0' && ch <= '9':
			pendingDot = false
			for i < len(code) && ((code[i] >= '0' && code[i] <= '9') || code[i] == '.') {
				i++
			}
		case isIdentifierStart(ch):
			start := i
			for i < len(code) && isIdentifierPart(code[i]) {
				i++
			}
			tokens = append(tokens, customRuleToken{
				text:      code[start:i],
				line:      line,
				dotBefore: pendingDot,
			})
			pendingDot = false
		default:
			pendingDot = false
			i++
		}
	}

	// Nombre cualificado: reconstruye cadenas de acceso con puntos.
	for j := range tokens {
		if tokens[j].dotBefore && j > 0 {
			tokens[j].qualified = tokens[j-1].qualified + "." + tokens[j].text
			continue
		}
		tokens[j].qualified = tokens[j].text
	}

	return tokens
}

// skipStringLiteral avanza más allá de un literal de cadena (simple o triple)
// sin interpretar su contenido como código.
func skipStringLiteral(code string, start int, line *int) int {
	quote := code[start]
	i := start

	triple := i+2 < len(code) && code[i+1] == quote && code[i+2] == quote
	if triple {
		i += 3
		for i < len(code) {
			if code[i] == '\\' {
				i += 2
				continue
			}
			if code[i] == '\n' {
				(*line)++
			}
			if code[i] == quote && i+2 < len(code) && code[i+1] == quote && code[i+2] == quote {
				return i + 3
			}
			i++
		}
		return i
	}

	i++
	for i < len(code) {
		if code[i] == '\\' {
			i += 2
			continue
		}
		if code[i] == '\n' {
			(*line)++
			i++
			continue
		}
		if code[i] == quote {
			return i + 1
		}
		i++
	}
	return i
}

func isIdentifierStart(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isIdentifierPart(ch byte) bool {
	return isIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

// splitPatterns separa los patrones declarados por el docente. La UI permite
// ingresar varios identificadores separados por coma en una sola regla.
func splitPatterns(pattern string) []string {
	parts := strings.Split(pattern, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ruleAppliesToLanguage descarta las reglas declaradas para otro lenguaje.
func ruleAppliesToLanguage(rule domain.ASTCustomRule, language string) bool {
	ruleLang := normalizeLanguage(rule.Language)
	if ruleLang == "" {
		return true
	}
	if language == "" {
		return true
	}
	return ruleLang == language
}

// normalizeLanguage canonicaliza los alias de lenguaje usados por la UI.
func normalizeLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "py", "python3":
		return "python"
	case "c++", "cpp":
		return "cpp"
	case "c#", "cs":
		return "csharp"
	case "js", "node", "nodejs":
		return "javascript"
	default:
		return strings.ToLower(strings.TrimSpace(language))
	}
}

// matchCustomRule busca el identificador prohibido dentro del flujo de tokens.
// La comparación es por identificador completo, no por subcadena: prohibir
// "sorted" no debe coincidir con "is_sorted" ni con "sorted_list".
func matchCustomRule(rule domain.ASTCustomRule, pattern string, tokens []customRuleToken) *domain.CustomRuleViolation {
	dotted := strings.Contains(pattern, ".")

	for _, token := range tokens {
		matched := false
		if dotted {
			matched = token.qualified == pattern
		} else {
			matched = token.text == pattern
		}
		if !matched {
			continue
		}

		ruleType := strings.TrimSpace(rule.Type)
		if ruleType == "" {
			ruleType = "function"
		}
		ruleLang := normalizeLanguage(rule.Language)

		message := strings.TrimSpace(rule.Message)
		if message == "" {
			message = fmt.Sprintf("El uso de '%s' no está permitido para este ejercicio.", pattern)
		}

		return &domain.CustomRuleViolation{
			RuleID:   fmt.Sprintf("custom_rule:%s:%s:%s", ruleLang, ruleType, pattern),
			Type:     ruleType,
			Language: ruleLang,
			Pattern:  pattern,
			Target:   token.qualified,
			Line:     token.line,
			Message:  message,
		}
	}

	return nil
}

// ValidateCustomRules evalúa las reglas personalizadas declaradas por el
// docente en `ast_rules.custom_rules`. Devuelve la primera infracción
// detectada o nil cuando el código cumple todas las reglas (GOB-SEGURO-02).
func (a *StaticASTAnalyzer) ValidateCustomRules(language string, sourceCode string, rules domain.ASTRules) *domain.CustomRuleViolation {
	if len(rules.CustomRules) == 0 {
		return nil
	}

	lang := normalizeLanguage(language)
	tokens := lexIdentifiers(sourceCode)

	for _, rule := range rules.CustomRules {
		if !ruleAppliesToLanguage(rule, lang) {
			continue
		}
		for _, pattern := range splitPatterns(rule.Pattern) {
			if violation := matchCustomRule(rule, pattern, tokens); violation != nil {
				return violation
			}
		}
	}

	return nil
}
