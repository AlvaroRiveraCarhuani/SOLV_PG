# Manual Operativo — SLICE 12: Experiencia Estudiante y Juez en Tiempo Real

## 1. Endpoints de Backend Disponibles

| Método | Ruta | Descripción | Rol Requerido |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/users/me` | Obtiene el perfil del usuario autenticado | Autenticado (`student`, `teacher`, `admin`) |
| `GET` | `/api/v1/student/assignments/due` | Lista de entregas y laboratorios pendientes | `student` |
| `POST` | `/api/v1/workspaces/{id}/pause` | Pausa manual e hibernación del contenedor | Dueño (`student`) / `teacher` / `admin` |
| `GET / WS` | `/ws/v1/evaluations` | Canal WebSocket para eventos del Juez Virtual | Autenticado con token |
| `GET / WS` | `/api/v1/ws/evaluations` | Alias del canal WebSocket de evaluación | Autenticado con token |

## 2. Protocolo de WebSocket del Juez Virtual (ADR-037)

### Conexión Handshake
- **URL:** `ws://localhost:3000/ws/v1/evaluations?token=<TOKEN_JWT>`
- **Ping / Pong:** Ticker de ping automático cada 25 segundos desde el servidor. Si el cliente no responde en 60 segundos, la conexión se cierra.

### Formato de Mensajes (Eventos Servidor -> Cliente)
```json
{
  "event": "EVALUATION_PROGRESS",
  "submission_id": "sub-12345",
  "stage": "QUEUED",
  "data": {
    "exercise_id": "ex-avl-01",
    "language": "cpp"
  },
  "timestamp": "2026-09-02T10:00:01Z"
}
```

## 3. Requisitos Operativos
- PostgreSQL 18 ejecutando con migraciones de `exercises.subject_id` y `exercises.due_date`.
- Docker Engine activo para la gestión del ciclo de vida de contenedores OpenVSCode Server.
