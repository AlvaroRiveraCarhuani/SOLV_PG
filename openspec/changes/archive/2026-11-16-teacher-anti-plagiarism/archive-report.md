# Archive Report: teacher-anti-plagiarism

**Change**: teacher-anti-plagiarism
**Archived to**: `openspec/changes/archive/2026-11-16-teacher-anti-plagiarism/`
**Archive date**: 2026-11-16
**Status**: CLOSED — SUPERSEDED

## Final State at Close

El cambio implementó originalmente un centro global de supervisión en `/teacher/anti-plagio`. El ADR-062 retiró ese listado y trasladó la auditoría a la ficha de cada práctica. La ruta antigua redirige a `/teacher/por-calificar` y la sección `SUPERVISIÓN` ya no existe.

El componente de auditoría por práctica sigue vigente y sus 10 pruebas pasan. El cambio se archiva como historia, no como contrato vigente.

## Gates

- **Task Completion Gate**: PASS — 6/6 tareas marcadas.
- **Evidence Gate**: PASS para el componente de práctica.
- **Spec Sync Gate**: NOT APPLICABLE — la spec original fue reemplazada por ADR-062.

## Specs Synced

No se copió ninguna spec a `openspec/specs/`. El archivo histórico se conserva dentro del cambio archivado. Fusionarlo introduciría un listado global que el producto ya no ofrece.

## Archive Contents

- `proposal.md`
- `design.md`
- `tasks.md` (6/6)
- `specs/teacher-anti-plagiarism.spec.md` (histórico)
- `verify-report.md`

## Carried-Forward Warnings

1. La spec original describe una ruta y un alcance retirados.
2. La suite no prueba la redirección ni la ausencia de `SUPERVISIÓN` en el sidebar.
3. No se ejecutó un E2E de auditoría desde la práctica.
