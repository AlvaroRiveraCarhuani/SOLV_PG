package services

import (
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

func TestASTPlagiarismEngine_VariableRenamingAndWhitespace(t *testing.T) {
	engine := NewASTPlagiarismEngine()

	// Código Alumno 1 (Python)
	codeA := `
def calcular_promedio(lista_notas):
    # Sumar todas las notas de la lista
    suma_total = 0
    for nota in lista_notas:
        if nota >= 0:
            suma_total += nota
    promedio = suma_total / len(lista_notas)
    return promedio
`

	// Código Alumno 2: Variables renombradas, sin comentarios, diferente espaciado
	codeB := `
def calcular(n):
    s = 0
    for x in n:
        if x >= 0:
            s += x
    res = s / len(n)
    return res
`

	tokensA := engine.Tokenize(codeA, "python")
	tokensB := engine.Tokenize(codeB, "python")

	if len(tokensA) == 0 || len(tokensB) == 0 {
		t.Fatalf("Tokens no deberían estar vacíos (A: %d, B: %d)", len(tokensA), len(tokensB))
	}

	similarity, matches, _ := engine.ComputeSimilarity(tokensA, tokensB)

	if similarity < 85.0 {
		t.Errorf("Esperada similitud >= 85.0%% para código renombrado estructuralmente idéntico, obtenido: %.2f%% (coincidencias: %d)", similarity, matches)
	}
}

func TestASTPlagiarismEngine_DissimilarAlgorithms(t *testing.T) {
	engine := NewASTPlagiarismEngine()

	// Código Alumno 1: Fibonacci Recursivo
	codeA := `
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n - 1) + fibonacci(n - 2)
`

	// Código Alumno 2: Búsqueda Binaria
	codeB := `
def binary_search(arr, target):
    low = 0
    high = len(arr) - 1
    while low <= high:
        mid = (low + high) // 2
        if arr[mid] == target:
            return mid
        elif arr[mid] < target:
            low = mid + 1
        else:
            high = mid - 1
    return -1
`

	tokensA := engine.Tokenize(codeA, "python")
	tokensB := engine.Tokenize(codeB, "python")

	similarity, _, _ := engine.ComputeSimilarity(tokensA, tokensB)

	if similarity > 50.0 {
		t.Errorf("Esperada similitud baja (< 50.0%%) para algoritmos diferentes, obtenido: %.2f%%", similarity)
	}
}

func TestASTPlagiarismEngine_AnalyzeSubmissions(t *testing.T) {
	engine := NewASTPlagiarismEngine()

	subs := []*domain.SubmissionForPlagiarism{
		{
			SubmissionID:  "sub-1",
			ExerciseID:    "ex-algo-1",
			ExerciseTitle: "Suma de Arreglos",
			StudentID:     "student-1",
			StudentName:   "Carlos Gómez",
			Language:      "python",
			Code: `
def solve(arr):
    total = 0
    for v in arr:
        total += v
    return total
`,
			SubmittedAt: time.Now(),
		},
		{
			SubmissionID:  "sub-2",
			ExerciseID:    "ex-algo-1",
			ExerciseTitle: "Suma de Arreglos",
			StudentID:     "student-2",
			StudentName:   "Ana Martínez",
			Language:      "python",
			Code: `
def solve(nums):
    # acumulador
    acc = 0
    for n in nums:
        acc += n
    return acc
`,
			SubmittedAt: time.Now(),
		},
		{
			SubmissionID:  "sub-3",
			ExerciseID:    "ex-algo-1",
			ExerciseTitle: "Suma de Arreglos",
			StudentID:     "student-3",
			StudentName:   "Pedro Pascal",
			Language:      "python",
			Code: `
def solve(arr):
    # Usando recursión diferente
    if not arr:
        return 0
    return arr[0] + solve(arr[1:])
`,
			SubmittedAt: time.Now(),
		},
	}

	matches := engine.AnalyzeSubmissions(subs)

	if len(matches) == 0 {
		t.Fatalf("Esperado al menos 1 match sospechoso entre estudiante 1 y 2, obtenido 0")
	}

	topMatch := matches[0]
	if topMatch.RiskLevel != "critical" && topMatch.RiskLevel != "warning" {
		t.Errorf("Nivel de riesgo esperado critical o warning, obtenido: %s", topMatch.RiskLevel)
	}

	if (topMatch.StudentIDA != "student-1" && topMatch.StudentIDB != "student-1") ||
		(topMatch.StudentIDA != "student-2" && topMatch.StudentIDB != "student-2") {
		t.Errorf("Match esperado entre student-1 y student-2, obtenido: %s y %s", topMatch.StudentIDA, topMatch.StudentIDB)
	}
}

const boilerplateSuma = `def solve(arr):
    total = 0
    for v in arr:
        total += v
    return total
`

func submission(id, student, code string) *domain.SubmissionForPlagiarism {
	return &domain.SubmissionForPlagiarism{
		SubmissionID:    id,
		ExerciseID:      "ex-suma",
		ExerciseTitle:   "Suma de Arreglos",
		StudentID:       student,
		StudentName:     student,
		Language:        "python",
		Code:            code,
		EnvironmentType: "JUEZ_EFIMERO",
		SubmittedAt:     time.Now(),
	}
}

func TestASTPlagiarismEngine_BoilerplateSubtractionRemovesFalsePositive(t *testing.T) {
	engine := NewASTPlagiarismEngine()

	subA := submission("sub-a", "student-1", boilerplateSuma)
	subA.Boilerplate = boilerplateSuma
	subB := submission("sub-b", "student-2", boilerplateSuma)
	subB.Boilerplate = boilerplateSuma

	matches := engine.AnalyzeSubmissions([]*domain.SubmissionForPlagiarism{subA, subB})

	if len(matches) != 0 {
		t.Fatalf("Dos entregas que solo comparten el código base no deben reportarse, obtenidos %d pares", len(matches))
	}
}

func TestASTPlagiarismEngine_RangePointsToRealSourceLines(t *testing.T) {
	engine := NewASTPlagiarismEngine()

	copied := boilerplateWithCopy()
	subA := submission("sub-a", "student-1", copied)
	subA.Boilerplate = boilerplateSuma
	subB := submission("sub-b", "student-2", copied)
	subB.Boilerplate = boilerplateSuma

	matches := engine.AnalyzeSubmissions([]*domain.SubmissionForPlagiarism{subA, subB})
	if len(matches) != 1 {
		t.Fatalf("Esperado 1 par con código copiado además del base, obtenidos %d", len(matches))
	}

	ranges := matches[0].Ranges
	if len(ranges) == 0 {
		t.Fatalf("El par debe traer al menos un tramo de líneas")
	}

	first := ranges[0]
	if first.SourceA.StartLine < 6 || first.SourceA.EndLine > 8 {
		t.Errorf("El tramo debe caer en la función copiada (líneas 6-7), obtenido %d-%d", first.SourceA.StartLine, first.SourceA.EndLine)
	}
	if first.SourceA.FilePath != "" || first.SourceB.FilePath != "" {
		t.Errorf("En el evaluador automático los tramos no llevan archivo: A=%q B=%q", first.SourceA.FilePath, first.SourceB.FilePath)
	}
	if first.TokenCount <= 0 {
		t.Errorf("El tramo debe informar sus tokens, obtenido %d", first.TokenCount)
	}
}

func TestRemoveCommentsKeepsLineNumbers(t *testing.T) {
	code := `int a = 1;
/* bloque
   de varias lineas */
int b = 2;
// solo comentario
int c = 3;`

	cleaned := removeComments(code, "cpp")
	cleanedLines := strings.Split(cleaned, "\n")
	if len(cleanedLines) < 6 || cleanedLines[3] != "int b = 2;" || cleanedLines[5] != "int c = 3;" {
		t.Fatalf("las líneas del fuente no conservan su posición: %q", cleanedLines)
	}

	engine := NewASTPlagiarismEngine()
	tokens := engine.Tokenize(code, "cpp")
	last := tokens[len(tokens)-1]
	if last.Line != 6 {
		t.Fatalf("El último token pertenece a la línea 6, obtenido %d", last.Line)
	}
}

func TestASTPlagiarismEngine_IDERangesPerFile(t *testing.T) {
	engine := NewASTPlagiarismEngine()

	baseSnapshot := "# === FILE: base.py ===\ndef helper(x):\n    return x + 1\n"
	subSnapshot := "# === FILE: base.py ===\ndef helper(x):\n    return x + 1\n\n# === FILE: main.py ===\ndef solve(arr):\n    total = 0\n    for v in arr:\n        total += v\n    return total\n"
	otherSnapshot := "# === FILE: base.py ===\ndef helper(x):\n    return x + 1\n\n# === FILE: main.py ===\ndef resolver(valores):\n    acc = 0\n    for item in valores:\n        acc += item\n    return acc\n"

	subA := submission("sub-a", "student-1", subSnapshot)
	subA.Boilerplate = baseSnapshot
	subA.EnvironmentType = "IDE_PERSISTENTE"
	subB := submission("sub-b", "student-2", otherSnapshot)
	subB.Boilerplate = baseSnapshot
	subB.EnvironmentType = "IDE_PERSISTENTE"

	matches := engine.AnalyzeSubmissions([]*domain.SubmissionForPlagiarism{subA, subB})
	if len(matches) != 1 {
		t.Fatalf("Esperado 1 par en IDE, obtenidos %d", len(matches))
	}

	ranges := matches[0].Ranges
	if len(ranges) == 0 {
		t.Fatalf("El par IDE debe traer tramos")
	}
	for _, r := range ranges {
		if r.SourceA.FilePath != "main.py" || r.SourceB.FilePath != "main.py" {
			t.Errorf("El tramo debe pertenecer a main.py; A=%q B=%q", r.SourceA.FilePath, r.SourceB.FilePath)
		}
		if r.SourceA.StartLine < 1 || r.SourceA.EndLine > 5 {
			t.Errorf("Las líneas deben ser locales a main.py (1-5), obtenido %d-%d", r.SourceA.StartLine, r.SourceA.EndLine)
		}
	}
}

func boilerplateWithCopy() string {
	return boilerplateSuma + "def extra(n):\n    return n * 2\n"
}
