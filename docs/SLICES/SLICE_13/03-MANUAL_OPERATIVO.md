# Manual Operativo — SLICE 13: Experiencia Docente, Cursos y Creación de Laboratorios

## 1. Endpoints de la Capa Docente

| Método | Ruta | Descripción | Rol Requerido |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/exercises` | Crear ejercicio con casos de prueba y AST | `teacher` / `admin` |
| `GET` | `/api/v1/subjects` | Listar materias (con filtro `?period_id=` y `?archived=`) | `teacher` / `admin` |
| `POST` | `/api/v1/templates/request` | Solicitar nueva plantilla Docker para revisión | `teacher` |
| `GET` | `/api/v1/exercises/{id}/submissions` | Listar entregas de estudiantes | `teacher` / `admin` |
| `POST` | `/api/v1/submissions/{id}/override` | Anulación o ajuste de calificación manual | `teacher` / `admin` |

## 2. Flujo Operativo de Plantillas Docker (ADR-030)
1. El docente completa la solicitud técnica de la plantilla (imagen, comandos de inicialización, límites de memoria).
2. El sistema la almacena con estado `pending_review`.
3. La plantilla no es visible para los estudiantes hasta que un Administrador la apruebe formalmente.

## 3. Filtrado y Archivado de Periodos (ADR-029)
- Las materias de periodos anteriores marcadas como archivadas entran en modo de sólo lectura para los estudiantes y se ocultan del listado principal del docente.
