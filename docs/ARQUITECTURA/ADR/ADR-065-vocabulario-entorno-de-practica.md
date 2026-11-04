# ADR-065: Vocabulario de entorno de práctica

**Estado:** Aceptado  
**Fecha:** 2026-11-04  
**Relacionado:** ADR-059 (entornos por práctica)

## Contexto

El modelo ya era uno solo: cada práctica tiene su entorno, con volumen propio y clave `(alumno, materia, práctica)`. La interfaz, en cambio, seguía diciendo «workspace» en rutas, tipos, tabla y auditoría, y la vista de laboratorio por materia ya no existía. Esa mezcla obligaba a traducir mentalmente entre «entorno» (UI) y «workspace» (API y base) y dejaba nombres que describían un contenedor de materia.

Los clientes son el propio frontend del repositorio; no hay integraciones externas que dependan de los nombres anteriores.

## Decisión

La instancia pasa a llamarse **entorno de práctica** en toda la superficie:

| Ámbito | Antes | Ahora |
| :--- | :--- | :--- |
| Tipo Go | `WorkspaceInstance` | `PracticeEnvironment` |
| Tabla | `workspaces` | `environments` |
| Columna | `workspace_id` | `environment_id` |
| Rutas | `/api/v1/workspaces/...` | `/api/v1/environments/...` |
| Servicios y handlers | `WorkspaceService`, `WorkspaceHandler` | `PracticeEnvironmentService`, `PracticeEnvironmentHandler` |
| Frontend | `WorkspaceInstance`, `startWorkspace` | `PracticeEnvironment`, `startEnvironment` |
| Auditoría | `WORKSPACE_RECONCILED_*`, `EXAM_WORKSPACE_*` | `PRACTICE_ENVIRONMENT_RECONCILED_*`, `EXAM_PRACTICE_ENVIRONMENT_*` |
| Frases de emergencia | `TERMINAR TODOS LOS WORKSPACES` | `TERMINAR TODOS LOS ENTORNOS` |

La migración `00048_environment_vocabulary.sql` renombra tabla, columnas, índices y constraints. Es idempotente y tiene `Down`.

### Lo que no cambia

- El prefijo de volumen `solv_workspace_<alumno>_<materia>_<práctica>` y la ruta `/home/workspace` dentro del contenedor. Son infraestructura: renombrarlos dejaría huérfanos los volúmenes existentes y el mount del editor apuntaría a otra carpeta.
- Los ADRs y documentos históricos, que describen el estado anterior con sus nombres.
- El veredicto y los estados del juez, que no pertenecen a este vocabulario.

## Consecuencias

- Una sola palabra para la instancia en UI, API, base y auditoría.
- El contrato HTTP cambia sin capa de compatibilidad: `/api/v1/workspaces` deja de responder.
- `PracticeEnvironment` evita la colisión con `EnvironmentType` (modalidad de la plantilla) y con `environment_requests` (pedidos del docente).
- Los registros de auditoría escritos antes del cambio conservan su acción original; las consultas nuevas usan el nombre nuevo.

## Alternativas descartadas

| Alternativa | Motivo |
| :--- | :--- |
| Llamar al tipo `Environment` | Se confunde con `EnvironmentType` (modalidad) y con `environment_requests`. |
| Mantener `workspace` en el dominio y traducir solo en la UI | Conserva la doble nomenclatura que este ADR elimina. |
| Renombrar también el prefijo del volumen | Rompe el montaje y deja huérfano el código de las prácticas en curso. |
