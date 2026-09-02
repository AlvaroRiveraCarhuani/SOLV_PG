# Contratos de API REST y WebSocket — SOLV Backend

> **Gobernanza:** SDD / Zero-Framework Go / Contratos de Transporte  
> **Servidor Base:** `http://localhost:3000` (o Ingress `https://api.solv.uab.edu.bo`)

---

## SLICE_12 — Contratos Backend Implementados

### 1. Perfil de Usuario Autenticado (`GET /api/v1/users/me`)

Obtiene el DTO completo del usuario autenticado en sesión activa (resuelto vía ForwardAuth o token JWT).

- **Método:** `GET`
- **Ruta:** `/api/v1/users/me`
- **Headers Requeridos:**
  - `X-User-Id: <uuid_del_usuario>` (O Cookie `solv_session` / `Authorization: Bearer <jwt>`)
  - `X-Tenant-Id: 00000000-0000-0000-0000-000000000001` (Opcional, fallback por defecto)

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "id": "33333333-3333-4333-a333-333333333333",
    "email": "alovelace@uab.edu.bo",
    "first_name": "Ada",
    "last_name": "Lovelace",
    "full_name": "Ada Lovelace",
    "role": "student",
    "tenant_id": "00000000-0000-0000-0000-000000000001"
  },
  "error": "",
  "message": "Perfil de usuario obtenido exitosamente"
}
```

#### Respuestas de Error
- `401 Unauthorized`: Si falta la cabecera `X-User-Id` o la cookie de sesión.
- `404 Not Found`: Si el identificador no existe en la base de datos.

#### Comando `curl` de Prueba
```bash
curl -X GET "http://localhost:3000/api/v1/users/me" \
     -H "X-User-Id: 33333333-3333-4333-a333-333333333333" \
     -H "Content-Type: application/json"
```

---

### 2. Entregas y Asignaciones Pendientes del Estudiante (`GET /api/v1/student/assignments/due`)

Retorna la lista de ejercicios y laboratorios asignados a las materias donde el estudiante está matriculado, filtrados por fecha límite futura (`due_date > NOW()`) y ordenados por proximidad de entrega.

- **Método:** `GET`
- **Ruta:** `/api/v1/student/assignments/due`
- **Headers Requeridos:**
  - `X-User-Id: <uuid_del_estudiante>`
  - `X-Tenant-Id: 00000000-0000-0000-0000-000000000001`

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": [
    {
      "exercise_id": "66666666-6666-4666-a666-666666666666",
      "title": "Laboratorio #04: Árboles AVL",
      "description": "Implemente la rotación y balanceo en C++",
      "subject_id": "55555555-5555-4555-a555-555555555555",
      "subject_name": "Estructuras de Datos",
      "subject_code": "ED-101",
      "due_date": "2026-09-04T18:00:00Z",
      "type": "algorithm"
    }
  ],
  "error": "",
  "message": "Entregas pendientes obtenidas exitosamente"
}
```

#### Comando `curl` de Prueba
```bash
curl -X GET "http://localhost:3000/api/v1/student/assignments/due" \
     -H "X-User-Id: 44444444-4444-4444-a444-444444444444" \
     -H "Content-Type: application/json"
```

---

### 3. Hibernación Manual de Workspace (`POST /api/v1/workspaces/{id}/pause`)

Pausa un contenedor activo de OpenVSCode Server liberando memoria RAM en el servidor host y actualizando el estado de la instancia a `hibernated`. Requiere que el solicitante sea el dueño del workspace o tenga rol docente/admin.

- **Método:** `POST`
- **Ruta:** `/api/v1/workspaces/{id}/pause`
- **Headers Requeridos:**
  - `X-User-Id: <uuid_del_usuario>`
  - `X-User-Role: student | teacher | admin`

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "id": "99999999-9999-4999-a999-999999999999",
    "student_id": "77777777-7777-4777-a777-777777777777",
    "subject_id": "88888888-8888-4888-a888-888888888888",
    "status": "hibernated",
    "type": "IDE_PERSISTENTE",
    "memory_limit_mb": 256
  },
  "error": "",
  "message": "Workspace hibernado exitosamente"
}
```

#### Comando `curl` de Prueba
```bash
curl -X POST "http://localhost:3000/api/v1/workspaces/99999999-9999-4999-a999-999999999999/pause" \
     -H "X-User-Id: 77777777-7777-4777-a777-777777777777" \
     -H "X-User-Role: student" \
     -H "Content-Type: application/json"
```

---

### 4. Concentrador WebSocket para el Juez Virtual (`/ws/v1/evaluations` — ADR-037)

Canal dúplex en tiempo real para suscripción al ciclo de vida de evaluación algorítmica y feedback de pre-chequeo Semgrep (AST).

- **Protocolo:** `WSS` / `WS`
- **Ruta:** `/ws/v1/evaluations` (Alias: `/api/v1/ws/evaluations`)
- **Autenticación en Handshake:** Query param `?token=<JWT>`, cabecera `Authorization: Bearer <JWT>` o cookie `solv_session`.

#### Ciclo de Mensajes (Eventos Servidor -> Cliente)
* `CONNECTION_ESTABLISHED` (Stage: `CONNECTED`)
* `EVALUATION_PROGRESS` (Stage: `QUEUED` / `AST_CHECKING` / `COMPILING` / `RUNNING_TEST`)
* `EVALUATION_COMPLETED` (Stage: `COMPLETED` con veredicto `AC`, `WA`, `TLE`, `RE`, `AST_BLOCKED`)

#### Comando CLI de Prueba
```bash
websocat "ws://localhost:3000/ws/v1/evaluations?token=TU_JWT_AQUI"
```

---

## SLICE_13 — Contratos Backend Implementados (Creación de Labs y Docencia)

### 1. Creación de Laboratorios / Ejercicios (`POST /api/v1/exercises`)

Permite a los docentes crear un nuevo laboratorio con configuración completa (límites de memoria/tiempo, boilerplate, reglas AST y casos de prueba).

- **Método:** `POST`
- **Ruta:** `/api/v1/exercises`
- **Headers:** `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Código de Respuesta:** `201 Created` (`403` para student, `400` por payload inválido)

#### Ejemplo de Petición
```json
{
  "title": "Algoritmo Dijkstra de Caminos Mínimos",
  "description": "Implemente Dijkstra utilizando cola de prioridad",
  "type": "algorithm",
  "boilerplate": "def dijkstra(graph, start):\n    pass\n",
  "status": "draft",
  "language": "python",
  "time_limit_ms": 1500,
  "memory_limit_mb": 256,
  "config": {
    "algorithm": {
      "time_limit_ms": 1500,
      "memory_limit_mb": 256,
      "test_cases": [
        {"input": "g1, start", "expected_output": "[0, 2, 5]", "is_hidden": false},
        {"input": "g_priv1, start", "expected_output": "[0, 10, 25]", "is_hidden": true}
      ],
      "ast_rules": {
        "forbidden_imports": ["networkx", "os"],
        "forbidden_functions": ["eval", "exec"]
      }
    }
  }
}
```

---

### 2. Actualización de Ejercicio (`PUT /api/v1/exercises/{id}`)

Actualiza la información base, límites y configuración del ejercicio. Aislado por tenant (`404` si pertenece a otra institución).

- **Método:** `PUT`
- **Ruta:** `/api/v1/exercises/{id}`
- **Headers:** `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Código de Respuesta:** `200 OK`

---

### 3. Importación Masiva de Casos de Prueba (`POST /api/v1/exercises/{id}/test-cases/bulk`)

Permite anexar casos de prueba en bloque usando JSON o archivo CSV. Soporta campos con comas entre comillas, texto Unicode y formato CRLF.

- **Método:** `POST`
- **Ruta:** `/api/v1/exercises/{id}/test-cases/bulk`
- **Headers:** `Content-Type: text/csv` (o `application/json`), `X-User-Role: teacher | admin`
- **Respuestas:**
  - `200 OK`: Casos agregados exitosamente.
  - `422 Unprocessable Entity`: Si una fila del CSV está malformada (retorna el número exacto de línea).

---

### 4. Publicación de Ejercicio (`POST /api/v1/exercises/{id}/publish`)

Realiza la transición de estado `draft` -> `published`. Requiere que el ejercicio tenga al menos 1 caso de prueba público.

- **Método:** `POST`
- **Ruta:** `/api/v1/exercises/{id}/publish`
- **Headers:** `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`: Ejercicio publicado.
  - `422 Unprocessable Entity`: Si el ejercicio cuenta con 0 casos de prueba públicos.
  - `404 Not Found`: Si el ejercicio no existe en el tenant.

---

## Contratos Aprobados Pendientes de Implementar

### SLICE_14: Panel Administrador Institucional

| Endpoint | Método | ADR Vinculado | Descripción de Alto Nivel | Estado |
| :--- | :---: | :---: | :--- | :---: |
| `/api/v1/admin/maintenance` | `POST` | ADR-031 | Activar/desactivar modo mantenimiento global con bypass admin | *Pendiente* |
| `/api/v1/admin/emergency/{action}` | `POST` | ADR-032 | Ejecutar acción de emergencia con frase de confirmación tipada | *Pendiente* |
| `/api/v1/admin/students` | `GET` | ADR-033 | Listado administrativo de estudiantes con búsqueda y filtros | *Pendiente* |
| `/api/v1/admin/students/{id}/reset-oom` | `POST` | ADR-033 | Reset manual de penalizaciones OOMKilled de un estudiante | *Pendiente* |
| `/api/v1/admin/academic-periods` | `POST` / `GET` | ADR-029 | CRUD de periodos académicos institucionales | *Pendiente* |
| `/api/v1/admin/templates/{id}/review` | `PUT` | ADR-030 | Aprobación o rechazo de plantillas Docker solicitadas | *Pendiente* |
| `/api/v1/admin/courses/{id}/reassign` | `POST` | ADR-036 | Reasignación de docentes a cursos huérfanos | *Pendiente* |

### SLICE_15: Notificaciones Proactivas

| Endpoint | Método | ADR Vinculado | Descripción de Alto Nivel | Estado |
| :--- | :---: | :---: | :--- | :---: |
| `/api/v1/notifications` | `GET` | ADR-034 | Consulta paginada de notificaciones in-app del usuario | *Pendiente* |
| `/api/v1/notifications/unread-count` | `GET` | ADR-034 | Contador de notificaciones no leídas para la campana UI | *Pendiente* |
| `/api/v1/notifications/{id}/read` | `PATCH` | ADR-034 | Marcar notificación específica como leída | *Pendiente* |
| `/api/v1/notifications/mark-all-read` | `POST` | ADR-034 | Marcar todas las notificaciones como leídas | *Pendiente* |

### SLICE_16: Backups y Retención Institucional

| Endpoint | Método | ADR Vinculado | Descripción de Alto Nivel | Estado |
| :--- | :---: | :---: | :--- | :---: |
| `/api/v1/admin/backups` | `GET` | ADR-035 | Historial de respaldos ejecutados con tamaño y estado | *Pendiente* |
| `/api/v1/admin/backups/trigger` | `POST` | ADR-035 | Disparo manual de volcado `pg_dump` con checksum | *Pendiente* |
| `/api/v1/admin/backups/config` | `GET` / `PUT` | ADR-035 | Consulta y modificación de política de retención local/remota | *Pendiente* |
