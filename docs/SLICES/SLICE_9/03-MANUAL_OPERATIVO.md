# Manual Operativo — SLICE 9: Esquema Académico Completo

## 1. Migraciones y Preparación de Entorno
Al arrancar el backend Go (`cmd/api/main.go`), las migraciones en `postgres.go` crean automáticamente las 4 tablas académicas, la restricción Foreign Key `fk_workspaces_subject` y los 4 índices de rendimiento.

## 2. Endpoints Académicos Disponibles

| Método | Ruta | Descripción | Rol Requerido |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/v1/subjects` | Crear nueva materia | `teacher` / `admin` |
| `GET` | `/api/v1/subjects` | Listar materias del tenant | Todos |
| `POST` | `/api/v1/subjects/{id}/enroll` | Inscribir estudiante en materia | `teacher` / `admin` |
| `GET` | `/api/v1/subjects/{id}/students` | Listar estudiantes inscritos | `teacher` / `admin` |
| `POST` | `/api/v1/submissions` | Registrar solución enviada al juez | `student` |
| `GET` | `/api/v1/exercises/{id}/submissions` | Consultar entregas de un ejercicio | Todos (Filtrado por rol) |
| `POST` | `/api/v1/invitations/teachers` | Generar token de invitación docente | `admin` |

## 3. Dependencias del Host
El pre-chequeo AST de seguridad del Juez Virtual utiliza **Semgrep CLI** local para realizar análisis semántico estático previo al aprovisionamiento del contenedor sandbox.
- **Requisito:** Python 3.8+ y `semgrep` CLI instalados en el host.
- **Instalación:** `pip3 install semgrep` (o `pipx install semgrep`).
- **Verificación:** `semgrep --version`.
- **Ruta de reglas:** `backend/internal/infrastructure/semgrep/rules/{language}/forbidden.yaml`.
