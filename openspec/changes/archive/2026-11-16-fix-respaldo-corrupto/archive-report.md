# Archive Report: fix-respaldo-corrupto

**Change**: fix-respaldo-corrupto
**Archived to**: `openspec/changes/archive/2026-11-16-fix-respaldo-corrupto/`
**Archive date**: 2026-11-16
**Status**: CLOSED — PASS WITH WARNINGS

## Final State at Close

El contrato de verificación de respaldos, los tres estados de integridad, el tamaño sub-MB y el cierre fail-closed del disparador están implementados. Las 13 tareas están marcadas.

Las pruebas unitarias del servicio y las 20 pruebas del componente de configuración pasan. La integración HTTP con PostgreSQL no se ejecutó por falta de base y `TEST_DB_DSN`.

## Gates

- **Task Completion Gate**: PASS — 13/13 tareas marcadas.
- **Evidence Gate**: PASS — pruebas unitarias de respaldo y pruebas de UI.
- **Integration Gate**: NOT EXECUTED — `TestSlice16_Backups` requiere PostgreSQL.

## Specs Synced

| Dominio | Acción | Detalle |
|---|---|---|
| `backup-verify-contract` | Creado | Contrato de envelope, estados distintos, persistencia, tamaño sub-MB, confirmación honesta y fail-closed. |

El archivo quedó en `openspec/specs/backup-verify-contract/spec.md`.

## Archive Contents

- `proposal.md`
- `design.md`
- `exploration.md`
- `apply-progress.md`
- `tasks.md` (13/13)
- `specs/backup-verify-contract/spec.md`
- `verify-report.md`

## Carried-Forward Warnings

1. La persistencia y recarga de `last_verify_*` no se verificaron contra PostgreSQL.
2. `TriggerBackup` marca `last_verify_ok=true` al crear el respaldo tras calcular el checksum por relectura; es integridad en creación, pero conviene confirmar que la semántica de “verificar” posterior no se confunde en operación.
3. Un fallo de lectura durante `VerifyBackup` devuelve error y no persiste un outcome; el caso de archivo ilegible no tiene prueba específica.
4. La restauración real sigue fuera de alcance y no está certificada.
