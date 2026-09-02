# SLICE 13 — Verificación y Evidencia de Pruebas

## Inventario de Pruebas de Integración (PostgreSQL 18 Real)

Ubicación: `backend/tests/integration/`

| Archivo de Prueba | Cobertura / Alcance |
| :--- | :--- |
| `slice13_teacher_exercise_creation_test.go` | Creación de ejercicios, bulk import CSV RFC 4180, validación de 0 casos públicos en `publish`, aislamiento cross-tenant. |
| `slice13_teacher_dashboard_test.go` | Métricas de materias (`active_now`, `pending_review`, `at_risk`), widget de atención prioritaria por severidad, estadísticas de laboratorio. |
| `slice13_teacher_review_test.go` | Cola de entregas, vista SpeedGrader con desenmascaramiento privado, navegación temporal prev/next, validación 422 de override ($\ge 10$ chars), comentarios in-line. |
| `slice13_teacher_runner_and_export_test.go` | Runner efímero en memoria (comprobación de no persistencia en BD) y exportación de calificaciones en CSV con UTF-8 BOM y promedios exactos. |

---

## Comando de Ejecución y Reproducción

```bash
DATABASE_URL="postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable" go test -count=1 -v -run "TestSlice13" ./tests/integration/...
```

### Salida Real de Ejecución (100% pasando)
```
=== RUN   TestSlice13_TeacherDashboard_CompleteSuite
--- PASS: TestSlice13_TeacherDashboard_CompleteSuite (0.30s)
=== RUN   TestSlice13_Commit1_CreateAndVerifyDirectDB
--- PASS: TestSlice13_Commit1_CreateAndVerifyDirectDB (0.08s)
=== RUN   TestSlice13_Commit1_PublishTransitionsAndZeroPublicValidation
--- PASS: TestSlice13_Commit1_PublishTransitionsAndZeroPublicValidation (0.10s)
=== RUN   TestSlice13_Commit1_BulkImportCSV_MalformedAndQuoting
--- PASS: TestSlice13_Commit1_BulkImportCSV_MalformedAndQuoting (0.12s)
=== RUN   TestSlice13_Commit1_AuthorizationAndCrossTenantIsolation
--- PASS: TestSlice13_Commit1_AuthorizationAndCrossTenantIsolation (0.19s)
=== RUN   TestSlice13_TeacherReviewAndSpeedGraderSuite
--- PASS: TestSlice13_TeacherReviewAndSpeedGraderSuite (0.26s)
=== RUN   TestSlice13_TeacherRunnerAndExportSuite
--- PASS: TestSlice13_TeacherRunnerAndExportSuite (0.19s)
PASS
ok  	solv-backend/tests/integration	1.260s
```
