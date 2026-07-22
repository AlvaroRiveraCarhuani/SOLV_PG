package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
	dockerinfra "solv-backend/internal/infrastructure/docker"
)

func setupDockerRunner(t *testing.T) (domain.EvaluationRunner, func()) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("Failed to initialize Docker client for tests: %v", err)
	}

	runner := dockerinfra.NewDockerEvaluationRunner(cli)

	teardown := func() {
		_ = cli.Close()
	}

	return runner, teardown
}

func TestJudgeIOAcceptedAllLanguages(t *testing.T) {
	languages := []struct {
		lang     string
		codeFile string
		inFile   string
		outFile  string
	}{
		{lang: "python", codeFile: filepath.Join("testdata", "algoritmia", "python", "ac_sum.py"), inFile: filepath.Join("testdata", "algoritmia", "python", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "python", "ac_sum.out")},
		{lang: "c", codeFile: filepath.Join("testdata", "algoritmia", "c", "ac_sum.c"), inFile: filepath.Join("testdata", "algoritmia", "c", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "c", "ac_sum.out")},
		{lang: "cpp", codeFile: filepath.Join("testdata", "algoritmia", "cpp", "ac_sum.cpp"), inFile: filepath.Join("testdata", "algoritmia", "cpp", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "cpp", "ac_sum.out")},
		{lang: "csharp", codeFile: filepath.Join("testdata", "algoritmia", "csharp", "ac_sum.cs"), inFile: filepath.Join("testdata", "algoritmia", "csharp", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "csharp", "ac_sum.out")},
		{lang: "java", codeFile: filepath.Join("testdata", "algoritmia", "java", "Solution.java"), inFile: filepath.Join("testdata", "algoritmia", "java", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "java", "ac_sum.out")},
		{lang: "javascript", codeFile: filepath.Join("testdata", "algoritmia", "javascript", "ac_sum.js"), inFile: filepath.Join("testdata", "algoritmia", "javascript", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "javascript", "ac_sum.out")},
	}

	runner, teardown := setupDockerRunner(t)
	defer teardown()

	for _, l := range languages {
		t.Run(l.lang, func(t *testing.T) {
			codeBytes, err := os.ReadFile(l.codeFile)
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", l.lang, l.codeFile, err)
			}
			inBytes, err := os.ReadFile(l.inFile)
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", l.lang, l.inFile, err)
			}
			outBytes, err := os.ReadFile(l.outFile)
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", l.lang, l.outFile, err)
			}

			runConfig := domain.EvaluationRunConfig{
				Language:      l.lang,
				SourceCode:    string(codeBytes),
				MemoryLimitMB: 128,
				TimeLimitMS:   5000,
				TestCase: domain.TestCase{
					Input:          string(inBytes),
					ExpectedOutput: strings.TrimSpace(string(outBytes)),
					IsHidden:       false,
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := runner.RunTestCase(ctx, runConfig)
			if err != nil {
				t.Fatalf("[%s] RunTestCase devolvió error inesperado: %v", l.lang, err)
			}

			if result.Verdict != domain.VerdictAC {
				t.Errorf("[%s] Se esperaba Veredicto AC, se obtuvo %s (Detalles: %s)", l.lang, result.Verdict, result.ErrorDetails)
			} else {
				t.Logf("[%s] Pruebas AC pasadas. Salida: %q, Tiempo: %v", l.lang, result.ActualOutput, result.ExecutionTime)
			}
		})
	}
}

func TestJudgeIOWrongAnswerAllLanguages(t *testing.T) {
	languages := []struct {
		lang     string
		codeFile string
		inFile   string
		outFile  string
	}{
		{lang: "python", codeFile: filepath.Join("testdata", "algoritmia", "python", "wa_logic.py"), inFile: filepath.Join("testdata", "algoritmia", "python", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "python", "ac_sum.out")},
		{lang: "c", codeFile: filepath.Join("testdata", "algoritmia", "c", "wa_logic.c"), inFile: filepath.Join("testdata", "algoritmia", "c", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "c", "ac_sum.out")},
		{lang: "cpp", codeFile: filepath.Join("testdata", "algoritmia", "cpp", "wa_logic.cpp"), inFile: filepath.Join("testdata", "algoritmia", "cpp", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "cpp", "ac_sum.out")},
		{lang: "csharp", codeFile: filepath.Join("testdata", "algoritmia", "csharp", "wa_logic.cs"), inFile: filepath.Join("testdata", "algoritmia", "csharp", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "csharp", "ac_sum.out")},
		{lang: "java", codeFile: filepath.Join("testdata", "algoritmia", "java", "wa_logic.java"), inFile: filepath.Join("testdata", "algoritmia", "java", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "java", "ac_sum.out")},
		{lang: "javascript", codeFile: filepath.Join("testdata", "algoritmia", "javascript", "wa_logic.js"), inFile: filepath.Join("testdata", "algoritmia", "javascript", "ac_sum.in"), outFile: filepath.Join("testdata", "algoritmia", "javascript", "ac_sum.out")},
	}

	runner, teardown := setupDockerRunner(t)
	defer teardown()

	for _, l := range languages {
		t.Run(l.lang, func(t *testing.T) {
			codeBytes, err := os.ReadFile(l.codeFile)
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", l.lang, l.codeFile, err)
			}
			inBytes, err := os.ReadFile(l.inFile)
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", l.lang, l.inFile, err)
			}
			outBytes, err := os.ReadFile(l.outFile)
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", l.lang, l.outFile, err)
			}

			runConfig := domain.EvaluationRunConfig{
				Language:      l.lang,
				SourceCode:    string(codeBytes),
				MemoryLimitMB: 128,
				TimeLimitMS:   5000,
				TestCase: domain.TestCase{
					Input:          string(inBytes),
					ExpectedOutput: strings.TrimSpace(string(outBytes)),
					IsHidden:       false,
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := runner.RunTestCase(ctx, runConfig)
			if err != nil {
				t.Fatalf("[%s] RunTestCase devolvió error inesperado: %v", l.lang, err)
			}

			if result.Verdict != domain.VerdictWA {
				t.Errorf("[%s] Se esperaba Veredicto WA, se obtuvo %s", l.lang, result.Verdict)
			} else {
				t.Logf("[%s] Pruebas WA pasadas exitosamente: %s", l.lang, result.ErrorDetails)
			}
		})
	}
}
