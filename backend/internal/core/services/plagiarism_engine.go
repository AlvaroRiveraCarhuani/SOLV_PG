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

// ComputeSimilarity calcula el coeficiente de similitud entre dos secuencias de
// tokens y devuelve los tramos que comparten, con líneas del fuente.
func (e *ASTPlagiarismEngine) ComputeSimilarity(tokensA, tokensB []Token) (float64, int, []domain.PlagiarismRange) {
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0.0, 0, nil
	}

	score, intersectionCount := e.score(tokensA, tokensB)
	if score <= 0 {
		return score, intersectionCount, nil
	}

	return score, intersectionCount, e.buildRanges(tokensA, tokensB, nil, nil)
}

// score calcula Jaccard y Containment ponderados sobre los k-gramas de ambas
// secuencias. Es la parte que no necesita líneas ni archivos.
func (e *ASTPlagiarismEngine) score(tokensA, tokensB []Token) (float64, int) {
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0.0, 0
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
	for g, countA := range setA {
		if countB, ok := setB[g]; ok {
			minCount := countA
			if countB < minCount {
				minCount = countB
			}
			intersectionCount += minCount
		}
	}

	totalUnique := len(gramsA) + len(gramsB) - intersectionCount
	if totalUnique <= 0 {
		return 0.0, 0
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

	return score, intersectionCount
}

type tokenRun struct {
	aStart int
	aEnd   int
	bStart int
	bEnd   int
}

// maxRangesPerPair acota lo que el visor puede mostrar sin perder los tramos
// más largos.
const maxRangesPerPair = 10

// buildRanges encuentra los tramos que las dos secuencias comparten. Recorre los
// k-gramas coincidentes, extiende cada coincidencia hacia atrás y adelante
// mientras los tokens sigan siendo iguales y conserva los tramos maximales sin
// solaparse. En IDE el tramo se acota al archivo de su primera línea.
func (e *ASTPlagiarismEngine) buildRanges(tokensA, tokensB []Token, filesA, filesB []domain.ProjectFile) []domain.PlagiarismRange {
	k := e.KGramSize
	if k <= 0 || len(tokensA) < k || len(tokensB) < k {
		return nil
	}

	indexB := make(map[string][]int)
	for j := 0; j+k <= len(tokensB); j++ {
		gram := e.gramString(tokensB[j : j+k])
		indexB[gram] = append(indexB[gram], j)
	}

	runs := make([]tokenRun, 0)
	seen := make(map[[2]int]bool)
	// Un texto con muchos k-gramas repetidos puede disparar el producto de
	// coincidencias. El tope conserva los tramos encontrados sin colgar el request.
	const maxCandidates = 5000
	candidates := 0

	for i := 0; i+k <= len(tokensA) && candidates < maxCandidates; i++ {
		starts, ok := indexB[e.gramString(tokensA[i:i+k])]
		if !ok {
			continue
		}
		for _, j := range starts {
			key := [2]int{i, j}
			if seen[key] {
				continue
			}
			seen[key] = true
			candidates++
			if candidates > maxCandidates {
				break
			}

			aStart, bStart := i, j
			for aStart > 0 && bStart > 0 && sameToken(tokensA[aStart-1], tokensB[bStart-1]) {
				aStart--
				bStart--
			}
			aEnd, bEnd := i+k-1, j+k-1
			for aEnd+1 < len(tokensA) && bEnd+1 < len(tokensB) && sameToken(tokensA[aEnd+1], tokensB[bEnd+1]) {
				aEnd++
				bEnd++
			}

			runs = append(runs, tokenRun{aStart: aStart, aEnd: aEnd, bStart: bStart, bEnd: bEnd})
		}
	}

	// Primero los tramos más largos: así los solapados que quedan son los cortos.
	sort.SliceStable(runs, func(i, j int) bool {
		lengthI := runs[i].aEnd - runs[i].aStart
		lengthJ := runs[j].aEnd - runs[j].aStart
		if lengthI != lengthJ {
			return lengthI > lengthJ
		}
		return runs[i].aStart < runs[j].aStart
	})

	accepted := make([]tokenRun, 0, maxRangesPerPair)
	for _, run := range runs {
		if len(accepted) >= maxRangesPerPair {
			break
		}
		if overlapsAccepted(run, accepted) {
			continue
		}
		accepted = append(accepted, run)
	}

	// Se devuelven en el orden del fuente para que el visor los lea de arriba abajo.
	sort.SliceStable(accepted, func(i, j int) bool { return accepted[i].aStart < accepted[j].aStart })

	ranges := make([]domain.PlagiarismRange, 0, len(accepted))
	for _, run := range accepted {
		ranges = append(ranges, domain.PlagiarismRange{
			SourceA:    rangeSource(tokensA[run.aStart].Line, tokensA[run.aEnd].Line, filesA),
			SourceB:    rangeSource(tokensB[run.bStart].Line, tokensB[run.bEnd].Line, filesB),
			TokenCount: run.aEnd - run.aStart + 1,
			ExactMatch: true,
		})
	}
	return ranges
}

func overlapsAccepted(run tokenRun, accepted []tokenRun) bool {
	for _, other := range accepted {
		if run.aStart <= other.aEnd && other.aStart <= run.aEnd {
			return true
		}
		if run.bStart <= other.bEnd && other.bStart <= run.bEnd {
			return true
		}
	}
	return false
}

// rangeSource traduce líneas globales del snapshot a archivo y línea local. En
// el evaluador automático no hay archivos y las líneas quedan como están.
func rangeSource(startLine, endLine int, files []domain.ProjectFile) domain.PlagiarismRangeSource {
	file := domain.ProjectFileForGlobalLine(files, startLine)
	if file == nil {
		return domain.PlagiarismRangeSource{StartLine: startLine, EndLine: endLine}
	}
	start := startLine - file.StartLine + 1
	end := endLine - file.StartLine + 1
	if end < start {
		end = start
	}
	return domain.PlagiarismRangeSource{FilePath: file.Path, StartLine: start, EndLine: end}
}

func sameToken(a, b Token) bool {
	return a.Type == b.Type && a.Value == b.Value
}

func (e *ASTPlagiarismEngine) gramString(tokens []Token) string {
	var sb strings.Builder
	for _, t := range tokens {
		sb.WriteString(string(t.Type))
		sb.WriteString(":")
		sb.WriteString(t.Value)
		sb.WriteString(";")
	}
	return sb.String()
}

// AnalyzeSubmissions compara pares de entregas y reporta similitudes por encima del umbral.
// El código base de la práctica se resta antes de comparar, para que la plantilla
// compartida no produzca falsos positivos.
func (e *ASTPlagiarismEngine) AnalyzeSubmissions(submissions []*domain.SubmissionForPlagiarism) []domain.PlagiarismMatch {
	n := len(submissions)
	if n < 2 {
		return []domain.PlagiarismMatch{}
	}

	type parsedSubmission struct {
		sub    *domain.SubmissionForPlagiarism
		tokens []Token
		files  []domain.ProjectFile
	}

	parsed := make([]parsedSubmission, n)
	for i, sub := range submissions {
		tokens, files := e.prepareTokens(sub)
		parsed[i] = parsedSubmission{sub: sub, tokens: tokens, files: files}
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

			// Sin base suficiente no hay comparación: el par no se reporta.
			if len(parsed[i].tokens) < e.KGramSize || len(parsed[j].tokens) < e.KGramSize {
				continue
			}

			similarity, matchingTokens := e.score(parsed[i].tokens, parsed[j].tokens)

			if similarity >= e.MinThreshold {
				riskLevel := "info"
				if similarity >= 85.0 {
					riskLevel = "critical"
				} else if similarity >= 70.0 {
					riskLevel = "warning"
				}

				match := domain.PlagiarismMatch{
					SubmissionIDA:  parsed[i].sub.SubmissionID,
					StudentIDA:     parsed[i].sub.StudentID,
					StudentNameA:   parsed[i].sub.StudentName,
					SubmissionIDB:  parsed[j].sub.SubmissionID,
					StudentIDB:     parsed[j].sub.StudentID,
					StudentNameB:   parsed[j].sub.StudentName,
					ExerciseID:     parsed[i].sub.ExerciseID,
					ExerciseTitle:  parsed[i].sub.ExerciseTitle,
					Similarity:     similarity,
					RiskLevel:      riskLevel,
					MatchingTokens: matchingTokens,
					TotalTokensA:   len(parsed[i].tokens),
					TotalTokensB:   len(parsed[j].tokens),
					Ranges:         e.buildRanges(parsed[i].tokens, parsed[j].tokens, parsed[i].files, parsed[j].files),
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

// CompareSubmissions enfrenta dos entregas concretas y devuelve la similitud,
// los tramos compartidos y los totales posteriores a restar el código base. Es
// la operación que consume el visor del par.
func (e *ASTPlagiarismEngine) CompareSubmissions(a, b *domain.SubmissionForPlagiarism) (float64, int, []domain.PlagiarismRange, int, int) {
	tokensA, filesA := e.prepareTokens(a)
	tokensB, filesB := e.prepareTokens(b)
	if len(tokensA) < e.KGramSize || len(tokensB) < e.KGramSize {
		return 0.0, 0, nil, len(tokensA), len(tokensB)
	}

	score, matchingTokens := e.score(tokensA, tokensB)
	if score <= 0 {
		return score, matchingTokens, nil, len(tokensA), len(tokensB)
	}

	return score, matchingTokens, e.buildRanges(tokensA, tokensB, filesA, filesB), len(tokensA), len(tokensB)
}

// prepareTokens tokeniza la entrega y le resta el código base de la práctica. En
// IDE se resta archivo por archivo, con la línea global de cada archivo; si
// alguna de las dos partes no tiene cabeceras, se resta de forma global.
func (e *ASTPlagiarismEngine) prepareTokens(sub *domain.SubmissionForPlagiarism) ([]Token, []domain.ProjectFile) {
	if strings.EqualFold(strings.TrimSpace(sub.EnvironmentType), string(domain.EnvironmentTypeIDEPersistente)) {
		files, filesOK := domain.ParseProjectSnapshot(sub.Code, sub.Language)
		if filesOK {
			baseFiles, baseOK := domain.ParseProjectSnapshot(sub.Boilerplate, sub.Language)
			var baseGlobal []Token
			if !baseOK {
				baseGlobal = e.Tokenize(sub.Boilerplate, sub.Language)
			}

			tokens := make([]Token, 0)
			for _, file := range files {
				fileTokens := e.Tokenize(file.Content, file.Language)
				if baseOK {
					if baseFile := domain.ProjectFileByPath(baseFiles, file.Path); baseFile != nil {
						fileTokens = e.subtractBase(fileTokens, e.Tokenize(baseFile.Content, baseFile.Language))
					}
				} else {
					fileTokens = e.subtractBase(fileTokens, baseGlobal)
				}
				// ParseProjectSnapshot cuenta las líneas desde el inicio de Code; el
				// motor necesita esa línea global para reportar rangos.
				for idx := range fileTokens {
					fileTokens[idx].Line += file.StartLine - 1
				}
				tokens = append(tokens, fileTokens...)
			}
			return tokens, files
		}
	}

	codeTokens := e.Tokenize(sub.Code, sub.Language)
	return e.subtractBase(codeTokens, e.Tokenize(sub.Boilerplate, sub.Language)), nil
}

// subtractBase quita de la entrega los tokens cubiertos por un k-grama del
// código base. La ventana se acorta cuando el base tiene menos tokens que K.
func (e *ASTPlagiarismEngine) subtractBase(subTokens, baseTokens []Token) []Token {
	if len(subTokens) == 0 || len(baseTokens) == 0 {
		return subTokens
	}

	window := e.KGramSize
	if window <= 0 {
		return subTokens
	}
	if len(baseTokens) < window {
		window = len(baseTokens)
	}

	baseGrams := make(map[string]bool)
	for i := 0; i+window <= len(baseTokens); i++ {
		baseGrams[e.gramString(baseTokens[i:i+window])] = true
	}

	excluded := make([]bool, len(subTokens))
	for i := 0; i+window <= len(subTokens); i++ {
		if !baseGrams[e.gramString(subTokens[i:i+window])] {
			continue
		}
		for j := 0; j < window; j++ {
			excluded[i+j] = true
		}
	}

	filtered := make([]Token, 0, len(subTokens))
	for i, token := range subTokens {
		if !excluded[i] {
			filtered = append(filtered, token)
		}
	}
	return filtered
}

// removeComments quita comentarios conservando una línea por cada línea del
// fuente original. Descartar líneas desplazaba los números de línea de todos los
// tokens posteriores a un bloque /* ... */ y a una línea de solo comentario.
func removeComments(code, language string) string {
	lang := strings.ToLower(strings.TrimSpace(language))
	var out strings.Builder
	lines := strings.Split(code, "\n")

	inBlockComment := false

	for _, line := range lines {
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
			endIdx := strings.Index(line, "*/")
			if endIdx == -1 {
				out.WriteString("\n")
				continue
			}
			inBlockComment = false
			line = line[endIdx+2:]
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

		out.WriteString(line)
		out.WriteString("\n")
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
		if (c < '0' || c > '9') && (i != 0 || (c != '-' && c != '+')) {
			return false
		}
	}
	return true
}
