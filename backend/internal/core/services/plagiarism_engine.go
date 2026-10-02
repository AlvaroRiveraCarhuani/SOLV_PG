package services

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"solv-backend/internal/core/domain"
)

// TokenType representa una categoría estructural en el flujo de tokens normalizado.
type TokenType string

const (
	TokKeyword     TokenType = "KW"
	TokIdentifier  TokenType = "ID"
	TokLiteral     TokenType = "LIT"
	TokOperator    TokenType = "OP"
	TokDelimiter   TokenType = "DELIM"
	TokControlFlow TokenType = "CTRL"
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
}

// ASTPlagiarismEngine implementa el análisis de similitud estructural por huella AST / Winnowing.
type ASTPlagiarismEngine struct {
	KGramSize    int     // Tamaño de ventana para k-gramas (default 4)
	MinThreshold float64 // Porcentaje mínimo de similitud para reportar (default 60.0)
}

func NewASTPlagiarismEngine() *ASTPlagiarismEngine {
	return &ASTPlagiarismEngine{
		KGramSize:    4,
		MinThreshold: 60.0,
	}
}

// Tokenize normaliza el código fuente en una secuencia de tokens estructurales canónicos.
func (e *ASTPlagiarismEngine) Tokenize(sourceCode string, language string) []Token {
	cleaned := removeComments(sourceCode, language)
	lang := strings.ToLower(strings.TrimSpace(language))

	keywords := map[string]bool{
		"if": true, "else": true, "elif": true, "for": true, "while": true,
		"def": true, "function": true, "func": true, "class": true, "return": true,
		"break": true, "continue": true, "try": true, "catch": true, "except": true,
		"finally": true, "throw": true, "raise": true, "import": true, "from": true,
		"switch": true, "case": true, "default": true, "struct": true, "interface": true,
		"var": true, "let": true, "const": true, "int": true, "float": true, "string": true,
		"bool": true, "void": true, "public": true, "private": true, "protected": true,
		"package": true, "include": true, "using": true, "namespace": true, "new": true,
		"in": true, "is": true, "not": true, "and": true, "or": true, "range": true,
	}

	operators := map[string]bool{
		"+": true, "-": true, "*": true, "/": true, "%": true, "=": true,
		"==": true, "!=": true, "<": true, ">": true, "<=": true, ">=": true,
		"+=": true, "-=": true, "*=": true, "/=": true, "&&": true, "||": true,
		"!": true, "&": true, "|": true, "^": true, "<<": true, ">>": true,
		":=": true, "->": true, "=>": true,
	}

	delimiters := map[string]bool{
		"(": true, ")": true, "{": true, "}": true, "[": true, "]": true,
		";": true, ":": true, ",": true, ".": true,
	}

	lines := strings.Split(cleaned, "\n")
	tokens := make([]Token, 0, len(lines)*5)

	wordRegex := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*|==|!=|<=|>=|\+=|-=|\*=|/=|:=|&&|\|\||->|=>|[0-9]+(?:\.[0-9]+)?|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[+\-*/%=<>&|^!(){}\[\];:,.\n])`)

	for lineIdx, line := range lines {
		matches := wordRegex.FindAllString(line, -1)
		for _, m := range matches {
			mTrim := strings.TrimSpace(m)
			if mTrim == "" {
				continue
			}

			lineNum := lineIdx + 1

			// 1. Literal numérico o string
			if strings.HasPrefix(mTrim, "\"") || strings.HasPrefix(mTrim, "'") || isNumeric(mTrim) {
				tokens = append(tokens, Token{Type: TokLiteral, Value: "$LIT", Line: lineNum})
				continue
			}

			// 2. Control flow o keyword
			if keywords[mTrim] {
				if mTrim == "if" || mTrim == "else" || mTrim == "elif" || mTrim == "for" || mTrim == "while" || mTrim == "switch" || mTrim == "case" || mTrim == "try" || mTrim == "catch" || mTrim == "except" {
					tokens = append(tokens, Token{Type: TokControlFlow, Value: strings.ToUpper(mTrim), Line: lineNum})
				} else {
					tokens = append(tokens, Token{Type: TokKeyword, Value: strings.ToUpper(mTrim), Line: lineNum})
				}
				continue
			}

			// 3. Operadores
			if operators[mTrim] {
				tokens = append(tokens, Token{Type: TokOperator, Value: mTrim, Line: lineNum})
				continue
			}

			// 4. Delimitadores
			if delimiters[mTrim] {
				tokens = append(tokens, Token{Type: TokDelimiter, Value: mTrim, Line: lineNum})
				continue
			}

			// 5. Identificador de usuario normalizado
			tokens = append(tokens, Token{Type: TokIdentifier, Value: "$ID", Line: lineNum})
		}
	}

	_ = lang
	return tokens
}

// GenerateKGrams genera secuencias contiguas de K tokens para hashing.
func (e *ASTPlagiarismEngine) GenerateKGrams(tokens []Token) []string {
	if len(tokens) < e.KGramSize {
		if len(tokens) == 0 {
			return nil
		}
		var sb strings.Builder
		for _, t := range tokens {
			sb.WriteString(string(t.Type))
			sb.WriteString(":")
			sb.WriteString(t.Value)
			sb.WriteString(";")
		}
		return []string{sb.String()}
	}

	kgrams := make([]string, 0, len(tokens)-e.KGramSize+1)
	for i := 0; i <= len(tokens)-e.KGramSize; i++ {
		var sb strings.Builder
		for j := 0; j < e.KGramSize; j++ {
			t := tokens[i+j]
			sb.WriteString(string(t.Type))
			sb.WriteString(":")
			sb.WriteString(t.Value)
			sb.WriteString(";")
		}
		kgrams = append(kgrams, sb.String())
	}
	return kgrams
}

// ComputeSimilarity calcula el coeficiente de similitud entre dos secuencias de tokens.
func (e *ASTPlagiarismEngine) ComputeSimilarity(tokensA, tokensB []Token) (float64, int, []string) {
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0.0, 0, nil
	}

	gramsA := e.GenerateKGrams(tokensA)
	gramsB := e.GenerateKGrams(tokensB)

	setA := make(map[string]int)
	for _, g := range gramsA {
		setA[g]++
	}

	setB := make(map[string]int)
	for _, g := range gramsB {
		setB[g]++
	}

	intersectionCount := 0
	commonPatterns := make([]string, 0)
	patternSeen := make(map[string]bool)

	for g, countA := range setA {
		if countB, ok := setB[g]; ok {
			minCount := countA
			if countB < minCount {
				minCount = countB
			}
			intersectionCount += minCount

			if !patternSeen[g] && len(commonPatterns) < 5 {
				patternSeen[g] = true
				cleanPat := simplifyGram(g)
				if cleanPat != "" {
					commonPatterns = append(commonPatterns, cleanPat)
				}
			}
		}
	}

	totalUnique := len(gramsA) + len(gramsB) - intersectionCount
	if totalUnique <= 0 {
		return 0.0, 0, nil
	}

	jaccard := float64(intersectionCount) / float64(totalUnique) * 100.0

	minGrams := len(gramsA)
	if len(gramsB) < minGrams {
		minGrams = len(gramsB)
	}
	containment := 0.0
	if minGrams > 0 {
		containment = float64(intersectionCount) / float64(minGrams) * 100.0
	}

	// Ponderación balanceada: 60% Jaccard + 40% Containment
	score := (0.6 * jaccard) + (0.4 * containment)
	score = math.Round(score*10) / 10

	if score > 100.0 {
		score = 100.0
	}

	return score, intersectionCount, commonPatterns
}

// AnalyzeSubmissions compara pares de entregas y reporta similitudes por encima del umbral.
func (e *ASTPlagiarismEngine) AnalyzeSubmissions(submissions []*domain.SubmissionForPlagiarism) []domain.PlagiarismMatch {
	n := len(submissions)
	if n < 2 {
		return []domain.PlagiarismMatch{}
	}

	type parsedSubmission struct {
		sub    *domain.SubmissionForPlagiarism
		tokens []Token
	}

	parsed := make([]parsedSubmission, n)
	for i, sub := range submissions {
		parsed[i] = parsedSubmission{
			sub:    sub,
			tokens: e.Tokenize(sub.Code, sub.Language),
		}
	}

	matches := make([]domain.PlagiarismMatch, 0)

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			// Mismo estudiante en distintas entregas no es plagio cruzado
			if parsed[i].sub.StudentID == parsed[j].sub.StudentID {
				continue
			}

			// Solo comparamos si pertenecen al mismo ejercicio
			if parsed[i].sub.ExerciseID != parsed[j].sub.ExerciseID {
				continue
			}

			similarity, matchingTokens, commonPatterns := e.ComputeSimilarity(parsed[i].tokens, parsed[j].tokens)

			if similarity >= e.MinThreshold {
				riskLevel := "info"
				if similarity >= 85.0 {
					riskLevel = "critical"
				} else if similarity >= 70.0 {
					riskLevel = "warning"
				}

				match := domain.PlagiarismMatch{
					SubmissionIDA:    parsed[i].sub.SubmissionID,
					StudentIDA:       parsed[i].sub.StudentID,
					StudentNameA:     parsed[i].sub.StudentName,
					SubmissionIDB:    parsed[j].sub.SubmissionID,
					StudentIDB:       parsed[j].sub.StudentID,
					StudentNameB:     parsed[j].sub.StudentName,
					ExerciseID:       parsed[i].sub.ExerciseID,
					ExerciseTitle:    parsed[i].sub.ExerciseTitle,
					Similarity:       similarity,
					RiskLevel:        riskLevel,
					MatchingTokens:   matchingTokens,
					TotalTokensA:     len(parsed[i].tokens),
					TotalTokensB:     len(parsed[j].tokens),
					CommonStructures: commonPatterns,
				}
				matches = append(matches, match)
			}
		}
	}

	// Ordenar de mayor a menor similitud
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Similarity > matches[j].Similarity
	})

	return matches
}

func removeComments(code, language string) string {
	lang := strings.ToLower(strings.TrimSpace(language))
	var out strings.Builder
	lines := strings.Split(code, "\n")

	inBlockComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if lang == "python" || lang == "py" {
			// Remover comentarios #
			if idx := strings.Index(line, "#"); idx != -1 {
				line = line[:idx]
			}
			out.WriteString(line)
			out.WriteString("\n")
			continue
		}

		// C, C++, Java, JS, Go comentarios // y /* */
		if inBlockComment {
			if endIdx := strings.Index(line, "*/"); endIdx != -1 {
				inBlockComment = false
				line = line[endIdx+2:]
			} else {
				continue
			}
		}

		if startIdx := strings.Index(line, "/*"); startIdx != -1 {
			if endIdx := strings.Index(line[startIdx+2:], "*/"); endIdx != -1 {
				line = line[:startIdx] + line[startIdx+2+endIdx+2:]
			} else {
				inBlockComment = true
				line = line[:startIdx]
			}
		}

		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}

		if strings.TrimSpace(line) != "" || trimmed == "" {
			out.WriteString(line)
			out.WriteString("\n")
		}
	}

	return out.String()
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	dotCount := 0
	for i, c := range s {
		if c == '.' {
			dotCount++
			if dotCount > 1 {
				return false
			}
			continue
		}
		if (c < '0' || c > '9') && !(i == 0 && (c == '-' || c == '+')) {
			return false
		}
	}
	return true
}

func simplifyGram(gram string) string {
	parts := strings.Split(gram, ";")
	var tokens []string
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		sub := strings.Split(p, ":")
		if len(sub) == 2 {
			tokens = append(tokens, sub[1])
		}
	}
	if len(tokens) > 0 {
		return strings.Join(tokens, " ")
	}
	return ""
}
