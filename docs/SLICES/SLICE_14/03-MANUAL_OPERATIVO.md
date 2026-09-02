# Manual Operativo — SLICE 14: Panel Administrador Institucional

## 1. Endpoints de la Capa de Administración

| Método | Ruta | Descripción | Rol Requerido |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/admin/maintenance` | Activar/desactivar modo mantenimiento | `admin` |
| `POST` | `/api/v1/admin/emergency/{action}` | Ejecutar una de las 5 acciones de emergencia | `admin` (Requiere frase de confirmación) |
| `GET` | `/api/v1/admin/students` | Buscar y listar estudiantes con estado y métricas | `admin` |
| `POST` | `/api/v1/admin/students/{id}/reset-oom` | Resetear penalización de OOMKilled | `admin` |
| `POST` | `/api/v1/admin/academic-periods` | Crear nuevo periodo académico | `admin` |
| `PUT` | `/api/v1/admin/templates/{id}/review` | Aprobar o rechazar plantilla Docker | `admin` |
| `POST` | `/api/v1/admin/courses/{id}/reassign` | Reasignar docente a curso huérfano | `admin` |
| `GET` | `/api/v1/admin/audit-logs` | Consultar auditoría con filtros avanzados | `admin` |

## 2. Protocolo de Frases de Confirmación (Acciones de Emergencia)
Para evitar ejecuciones accidentales, los endpoints de `/api/v1/admin/emergency/{action}` validan la propiedad `confirmation_phrase` en el cuerpo JSON:
- Evicción de Contenedores: `"CONFIRMAR_EVICTION_TOTAL"`
- Purga de DB: `"CONFIRMAR_PURGA_CONEXIONES"`
- Revocación de Sesiones: `"CONFIRMAR_REVOCACION_SESIONES"`
- Reset OOM: `"CONFIRMAR_RESET_PENALIZACIONES"`
- Reinicio de Servicio: `"CONFIRMAR_REINICIO_SERVICIO"`
