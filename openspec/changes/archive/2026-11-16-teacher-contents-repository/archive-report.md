# Archive Report: teacher-contents-repository

**Change**: teacher-contents-repository
**Archived to**: `openspec/changes/archive/2026-11-16-teacher-contents-repository/`
**Archive date**: 2026-11-16
**Status**: CLOSED — PASS WITH WARNINGS

## Final State at Close

El banco de ejercicios y el catálogo de plantillas están implementados. Las 10 tareas están marcadas. Las 25 pruebas frontend y la guarda de modalidad del backend pasan.

El read model `GET /api/v1/teacher/exercises/bank` existe y el componente lo consume. La integración contra PostgreSQL no se ejecutó en esta sesión.

## Gates

- **Task Completion Gate**: PASS — 10/10 tareas marcadas.
- **Evidence Gate**: PASS — 25 pruebas frontend y `TestBulkAddTestCases_EnvironmentGuard`.
- **Integration Gate**: NOT EXECUTED — `TestSlice13_TeacherExerciseBank` requiere PostgreSQL.

## Specs Synced

| Dominio | Acción | Detalle |
|---|---|---|
| `teacher-contents-repository` | Creado | Banco global, KPIs, filtros, modalidad, acciones, catálogo de plantillas y calidad de UI. |

El archivo quedó en `openspec/specs/teacher-contents-repository/spec.md`.

La spec usa criterios numerados en lugar de escenarios Given/When/Then. Se conserva el contenido funcional y se registra como deuda de formato, no como blocker.

## Archive Contents

- `proposal.md`
- `design.md`
- `tasks.md` (10/10)
- `specs/teacher-contents-repository.spec.md`
- `verify-report.md`

## Carried-Forward Warnings

1. La agregación multi-materia y `review_pending` no se verificaron contra PostgreSQL.
2. El estado de docente sin materias solo se probó con datos simulados.
3. No se ejecutó un E2E de solicitud y aprobación de plantillas.
