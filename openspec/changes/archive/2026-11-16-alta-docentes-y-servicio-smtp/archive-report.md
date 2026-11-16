# Archive Report: alta-docentes-y-servicio-smtp

**Change**: alta-docentes-y-servicio-smtp
**Archived to**: `openspec/changes/archive/2026-11-16-alta-docentes-y-servicio-smtp/`
**Archive date**: 2026-11-16
**Status**: CLOSED — PASS WITH WARNINGS

## Final State at Close

El alta directa, la importación masiva tolerante, la previsualización y el adaptador SMTP están implementados. Las 25 tareas están marcadas y las pruebas acotadas pasan.

La verificación encontró y corrigió un defecto de contrato: el backend devuelve `registered_count` y el frontend leía `created_count`, por lo que el aviso de importación masiva mostraba `undefined`. El modelo, el componente y el mock de prueba quedaron alineados con la respuesta real.

## Gates

- **Task Completion Gate**: PASS — 25/25 tareas marcadas.
- **Evidence Gate**: PASS — backend con `go test` en mailer, services y delivery; frontend con 18 pruebas del área; gate de estilos limpio.
- **Integration Gate**: NOT EXECUTED — no hay PostgreSQL levantado ni `TEST_DB_DSN`.

## Specs Synced

| Dominio | Acción | Detalle |
|---|---|---|
| `admin-teachers` | Creado | Spec de alta directa, importación tolerante, previsualización y notificación. |
| `smtp-service` | Creado | Spec de configuración SMTP, fallback local, cola asíncrona y plantilla institucional. |

Los archivos quedaron en:

- `openspec/specs/admin-teachers/spec.md`
- `openspec/specs/smtp-service/spec.md`

## Archive Contents

- `proposal.md`
- `design.md`
- `tasks.md` (25/25)
- `specs/admin-teachers/spec.md`
- `specs/smtp-service/spec.md`
- `verify-report.md`

## Carried-Forward Warnings

1. No se ejecutó integración PostgreSQL para promoción, idempotencia y persistencia masiva.
2. No hay prueba automatizada de `.xlsx`, aunque la rama `excelize` existe.
3. No existe `admin-teachers.service.spec.ts`; la cobertura es de componente.
4. El camino `send_notification=true` no tiene prueba de entrega; la prueba de cola es de humo.
5. No se probó relay SMTP real ni TLS 465.
6. No se ejecutó el E2E de alta directa e inicio de sesión con Google SSO.
7. La implementación quedó en `TeacherInvitationService`/`TeacherInvitationHandler` y `cmd/api/main.go`, no en los archivos que nombran las tareas. Es una desalineación de trazabilidad, no un cambio de contrato.
