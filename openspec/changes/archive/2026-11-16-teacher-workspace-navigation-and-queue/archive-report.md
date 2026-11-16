# Archive Report: teacher-workspace-navigation-and-queue

**Change**: teacher-workspace-navigation-and-queue
**Archived to**: `openspec/changes/archive/2026-11-16-teacher-workspace-navigation-and-queue/`
**Archive date**: 2026-11-16
**Status**: CLOSED — SUPERSEDED

## Final State at Close

El cambio implementó una navegación de tres secciones y una cola global en `/teacher/evaluaciones`. El estado vigente fue reemplazado por decisiones posteriores:

- ADR-055 retiró el centro global de evaluaciones y unificó la cola por `review_status`.
- ADR-057 fijó `HOY`, `DOCENCIA`, `CONTENIDOS` y `CUENTA` como navegación.
- ADR-062 retiró `SUPERVISIÓN` y llevó la auditoría a cada práctica.

La ruta `/teacher/evaluaciones` redirige a `/teacher/por-calificar`; la cola vigente es `TeacherReviewsComponent`, con 5 pruebas en verde.

## Gates

- **Task Completion Gate**: PASS — 8/8 tareas marcadas.
- **Evidence Gate**: PASS para la cola vigente.
- **Spec Sync Gate**: NOT APPLICABLE — la spec original fue reemplazada por ADR-055, ADR-057 y ADR-062.

## Specs Synced

No se copió ninguna spec a `openspec/specs/`. El archivo histórico se conserva dentro del cambio archivado. Fusionarlo reintroduciría rutas y secciones retiradas.

## Archive Contents

- `proposal.md`
- `design.md`
- `tasks.md` (8/8)
- `specs/teacher-workspace-navigation-and-queue.spec.md` (histórico)
- `verify-report.md`

## Carried-Forward Warnings

1. La spec original describe rutas y secciones que ya no existen.
2. No hay prueba del `TeacherLayoutComponent`; la estructura se verificó por código y ADR.
3. No se ejecutó un E2E de navegación hasta SpeedGrader.
