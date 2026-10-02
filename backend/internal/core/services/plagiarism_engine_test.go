package services

import (
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
