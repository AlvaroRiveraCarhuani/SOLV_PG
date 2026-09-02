package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

func TestJudgeLimitsTimeLimitExceededAllLanguages(t *testing.T) {
	languages := []struct {
		lang     string
		codeFile string
	}{
		{lang: "python", codeFile: filepath.Join("testdata", "algoritmia", "python", "tle_loop.py")},
		{lang: "c", codeFile: filepath.Join("testdata", "algoritmia", "c", "tle_loop.c")},
		{lang: "cpp", codeFile: filepath.Join("testdata", "algoritmia", "cpp", "tle_loop.cpp")},
		{lang: "javascript", codeFile: filepath.Join("testdata", "algoritmia", "javascript", "tle_loop.js")},
	}

	runner, teardown := setupDockerRunner(t)
	defer teardown()

	for _, l := range languages {
		t.Run(l.lang, func(t *testing.T) {
			codeBytes, err := os.ReadFile(l.codeFile)
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", l.lang, l.codeFile, err)
			}

			runConfig := domain.EvaluationRunConfig{
				Language:      l.lang,
				SourceCode:    string(codeBytes),
				MemoryLimitMB: 128,
				TimeLimitMS:   1000,
				TestCase: domain.TestCase{
					Input:          "",
					ExpectedOutput: "done",
					IsHidden:       false,
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			result, err := runner.RunTestCase(ctx, runConfig)
			if err != nil {
				t.Fatalf("[%s] RunTestCase devolvió error inesperado: %v", l.lang, err)
			}

			if result.Verdict != domain.VerdictTLE {
				t.Errorf("[%s] Se esperaba Veredicto TLE, se obtuvo %s (Detalles: %s)", l.lang, result.Verdict, result.ErrorDetails)
			} else {
				t.Logf("[%s] Prueba TLE pasada tras %v: %s", l.lang, result.ExecutionTime, result.ErrorDetails)
			}
		})
	}
}
