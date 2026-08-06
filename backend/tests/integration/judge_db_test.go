package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

func TestJudgeDatabaseDryRunAndEvaluationAllEngines(t *testing.T) {
	engines := []struct {
		engine  string
		dbDir   string
		initExt string
		refExt  string
		valExt  string
		waExt   string
	}{
		{
			engine:  "postgres",
			dbDir:   filepath.Join("testdata", "bases_de_datos", "postgres"),
			initExt: "init.sql",
			refExt:  "reference.sql",
			valExt:  "validation.sql",
			waExt:   "student_wa.sql",
		},
		{
			engine:  "mysql",
			dbDir:   filepath.Join("testdata", "bases_de_datos", "mysql"),
			initExt: "init.sql",
			refExt:  "reference.sql",
			valExt:  "validation.sql",
			waExt:   "student_wa.sql",
		},
		{
			engine:  "mongodb",
			dbDir:   filepath.Join("testdata", "bases_de_datos", "mongodb"),
			initExt: "init.js",
			refExt:  "reference.js",
			valExt:  "validation.js",
			waExt:   "student_wa.js",
		},
	}

	runner, teardown := setupDockerRunner(t)
	defer teardown()

	for _, e := range engines {
		t.Run(e.engine, func(t *testing.T) {
			initBytes, err := os.ReadFile(filepath.Join(e.dbDir, e.initExt))
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", e.engine, e.initExt, err)
			}
			refBytes, err := os.ReadFile(filepath.Join(e.dbDir, e.refExt))
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", e.engine, e.refExt, err)
			}
			valBytes, err := os.ReadFile(filepath.Join(e.dbDir, e.valExt))
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", e.engine, e.valExt, err)
			}
			waBytes, err := os.ReadFile(filepath.Join(e.dbDir, e.waExt))
			if err != nil {
				t.Fatalf("[%s] Error leyendo %s: %v", e.engine, e.waExt, err)
			}

			dryRunConfig := domain.DBEvaluationRunConfig{
				Engine:          e.engine,
				InitScript:      string(initBytes),
				SolutionSQL:     string(refBytes),
				ValidationQuery: string(valBytes),
				MemoryLimitMB:   256,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
			defer cancel()

			t.Logf("[%s] 1. Ejecutando Dry Run para autogenerar expected_json...", e.engine)
			expectedJSON, err := runner.RunDBDryRun(ctx, dryRunConfig)
			if err != nil {
				t.Fatalf("[%s] Dry Run falló con error: %v", e.engine, err)
			}

			expectedJSONTrim := strings.TrimSpace(expectedJSON)
			if expectedJSONTrim == "" {
				t.Fatalf("[%s] Dry Run devolvió un JSON de estado vacío", e.engine)
			}
			t.Logf("[%s] Expected JSON autogenerado exitosamente: %s", e.engine, expectedJSONTrim)

			// 2. Probar solución correcta del alumno (AC)
			t.Logf("[%s] 2. Evaluando solución correcta del alumno...", e.engine)
			acConfig := domain.DBEvaluationRunConfig{
				Engine:          e.engine,
				InitScript:      string(initBytes),
				SolutionSQL:     string(refBytes),
				ValidationQuery: string(valBytes),
				MemoryLimitMB:   256,
			}

			acRes, err := runner.RunDBEvaluation(ctx, acConfig)
			if err != nil {
				t.Fatalf("[%s] Error en RunDBEvaluation para AC: %v", e.engine, err)
			}

			if acRes.Verdict != domain.VerdictAC {
				t.Errorf("[%s] Se esperaba AC para solución correcta, se obtuvo %s (Detalles: %s)", e.engine, acRes.Verdict, acRes.ErrorDetails)
			} else {
				t.Logf("[%s] Evaluación DB AC Exitosa. Salida obtenida: %s", e.engine, acRes.ResultingJSON)
			}

			// 3. Probar solución errónea del alumno (WA)
			t.Logf("[%s] 3. Evaluando solución errónea del alumno...", e.engine)
			waConfig := domain.DBEvaluationRunConfig{
				Engine:          e.engine,
				InitScript:      string(initBytes),
				SolutionSQL:     string(waBytes),
				ValidationQuery: string(valBytes),
				MemoryLimitMB:   256,
			}

			waRes, err := runner.RunDBEvaluation(ctx, waConfig)
			if err != nil {
				t.Fatalf("[%s] Error en RunDBEvaluation para WA: %v", e.engine, err)
			}

			waJSONTrim := strings.TrimSpace(waRes.ResultingJSON)
			if waJSONTrim == expectedJSONTrim {
				t.Errorf("[%s] Se esperaba que la solución errónea devuelva un JSON DISTINTO al esperado, pero coincidió: %s", e.engine, waJSONTrim)
			} else {
				t.Logf("[%s] Evaluación DB WA Exitosa! El JSON obtenido (%s) difiere del esperado (%s)", e.engine, waJSONTrim, expectedJSONTrim)
			}
		})
	}
}
