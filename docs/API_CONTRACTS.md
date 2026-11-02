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
- **Respuestas:** `200` con el entorno en `hibernated`; `404 workspace_not_found` cuando el laboratorio no existe (un id desconocido no es un fallo interno); `500` solo si Docker o la persistencia fallan de verdad.

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

### 5. Materias y Resumen del Docente (`GET /api/v1/teacher/courses`)

Retorna la lista de materias del docente autenticado con métricas agregadas (`students_count`, `active_now`, `pending_review`, `at_risk`).

- **Método:** `GET`
- **Ruta:** `/api/v1/teacher/courses`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Código de Respuesta:** `200 OK` (`403` si el rol es student)

#### Ejemplo de Respuesta
```json
{
  "data": [
    {
      "id": "9239032b-25e0-434d-8544-85768465552f",
      "name": "Programación II",
      "code": "PROG-201",
      "students_count": 35,
      "active_now": 18,
      "pending_review": 4,
      "at_risk": 2
    }
  ],
  "error": "",
  "message": "Cursos del docente obtenidos exitosamente"
}
```

---

### 7. Estadísticas por Laboratorio de un Curso (`GET /api/v1/teacher/courses/{id}/labs`)

Retorna el desglose de métricas, entregas y veredictos para cada laboratorio del curso.

- **Método:** `GET`
- **Ruta:** `/api/v1/teacher/courses/{id}/labs`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`: Lista de estadísticas por laboratorio.
  - `404 Not Found`: Si la materia no existe o no pertenece al docente.

#### Ejemplo de Respuesta
```json
{
  "data": [
    {
      "id": "lab-uuid-1",
      "title": "Lab #01: Árboles AVL",
      "status": "published",
      "due_date": "2026-09-10T23:59:59Z",
      "submissions_count": 28,
      "students_count": 35,
      "auto_graded": 24,
      "pending_review": 4,
      "at_risk": 2,
      "verdicts_summary": {
        "AC": 20,
        "WA": 5,
        "TLE": 2,
        "RE": 1,
        "AST_BLOCKED": 0
      }
    }
  ],
  "error": "",
  "message": "Estadisticas de laboratorios obtenidas exitosamente"
}
```

---

### 7.1 Prueba de entorno de una práctica IDE (`POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/env-test`)

Valida una práctica `IDE_PERSISTENTE` sin crear entregas ni un workspace persistente. El servidor resuelve la plantilla del ejercicio (con la de la materia como respaldo), aprovisiona sus bases satélite con una identidad de prueba aislada, ejecuta el `setup_script` de la plantilla sobre un volumen temporal, comprueba que `/home/workspace` se puede escribir y que la imagen trae OpenVSCode Server. El montaje del volumen es la comprobación de repositorio base: un volumen no escribible falla el job, igual que un setup con error.

- **Método:** `POST`
- **Ruta:** `/api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/env-test`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Cuerpo:** vacío. La identidad y el tenant salen de la sesión.
- **Respuestas:**
  - `202 Accepted`: job creado en `pending`.
  - `400 invalid_environment`: la práctica no es de IDE persistente.
  - `403 forbidden`: el docente no está asignado a la materia.
  - `404 practice_not_found`: la práctica o la materia no existen en el tenant.
  - `400 validation_error`: falta plantilla, imagen o la plantilla pide más RAM que la del host.

#### Ejemplo de Respuesta
```json
{
  "data": {
    "id": "job-uuid",
    "exercise_id": "exercise-uuid",
    "subject_id": "subject-uuid",
    "image": "python:3.12-slim",
    "target_environment": "IDE_PERSISTENTE",
    "status": "pending",
    "progress": { "bytes_done": 0, "bytes_total": 0, "layer_current": 0, "layers_total": 0, "percent": 0, "current_action": "Encolado, esperando slot de ejecución..." },
    "created_at": "2026-11-01T11:00:00Z",
    "updated_at": "2026-11-01T11:00:00Z"
  },
  "error": "",
  "message": "Prueba de entorno de la práctica iniciada"
}
```

El progreso y el resultado se consultan con los endpoints genéricos del job:

| Método | Ruta | Uso |
| --- | --- | --- |
| `GET` | `/api/v1/jobs/env-test/{jobId}` | Estado, progreso y resultado. |
| `POST` | `/api/v1/jobs/env-test/{jobId}/cancel` | Cancelación del job en curso. |

Estados: `pending` (en cola), `pulling` (imagen), `testing` (satélites y setup), `success`, `failed`, `canceled`. El resultado exitoso lista `repositorio base` (volumen escribible), `openvscode-server` y un elemento por cada satélite asegurado; una dependencia que falle en el `setup_script` deja el job en `failed`. Un fallo de setup conserva la salida del comando en `error_message`.

### 7.2 Última prueba de la práctica (`GET /api/v1/teacher/exercises/{exerciseId}/env-test/latest`)

Devuelve el job más reciente de esa práctica para el docente autenticado. Sin pruebas responde `200` con `data: null`; otro docente no ve el job aunque comparta la materia.

```json
{ "data": null, "error": "", "message": "Última prueba de entorno de la práctica" }
```

### 7.3 IDE de una práctica del docente (`POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/ide`)

Abre el entorno del docente acotado a una práctica y devuelve la URL para abrirlo en una pestaña nueva. El contenedor usa el volumen propio del ejercicio (`solv_workspace_<docente>_<materia>_<práctica>`), la plantilla del ejercicio y su `db_init_script`; dos prácticas de la misma materia no comparten repositorio. Si el entorno de esa práctica ya existe corriendo devuelve el mismo, y si estaba pausado lo reanuda. No crea entregas ni inscribe al docente en el aula de la materia.

- **Método:** `POST`
- **Ruta:** `/api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/ide`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Cuerpo:** vacío. La identidad y el tenant salen de la sesión.
- **Respuestas:**
  - `200 OK`: el IDE quedó disponible y `access_url` apunta al contenedor.
  - `400 not_project_practice`: la práctica es del juez virtual y no tiene repositorio persistente.
  - `403 forbidden`: la materia no está asignada a ese docente.
  - `404 practice_not_found`: la práctica o su materia no existen en el tenant.
  - `409 workspace_frozen`: el entorno de esa práctica quedó en solo lectura.
  - `503 host_memory_exhausted` / `503 max_containers_reached`: el host o el cupo del tenant no admiten otro contenedor.
  - `503 workspace_url_unavailable`: el contenedor arrancó pero no se pudo resolver su URL.

#### Ejemplo de Respuesta
```json
{
  "data": {
    "workspace_id": "b7c1e5f2-9a53-4f0e-9a5b-3f5b0f0e7c11",
    "exercise_id": "exercise-uuid",
    "subject_id": "subject-uuid",
    "template_id": "template-uuid",
    "status": "running",
    "access_url": "http://localhost:49153",
    "volume_name": "solv_workspace_teacher-uuid_subject-uuid_exercise-uuid"
  },
  "error": "",
  "message": "IDE de la práctica listo"
}
```

`access_url` sale del dominio configurado (`BASE_DOMAIN`) cuando hay Traefik, y del puerto publicado en `127.0.0.1` cuando se trabaja en local.

---

### 8. Cola de Entregas del Curso (`GET /api/v1/teacher/courses/{id}/submissions`)

Permite al docente inspeccionar y filtrar todas las entregas de su materia.

- **Método:** `GET`
- **Ruta:** `/api/v1/teacher/courses/{id}/submissions?exercise_id=<uuid>&verdict=<string>`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Código de Respuesta:** `200 OK` (`403` si es estudiante)

#### Ejemplo de Respuesta
```json
{
  "data": [
    {
      "id": "sub-uuid-1",
      "exercise_id": "ex-uuid-1",
      "exercise_title": "Lab #01: Árboles AVL",
      "student_id": "stu-uuid-1",
      "student_name": "Carlos Ruiz",
      "student_email": "carlos@uab.edu.bo",
      "verdict": "WA",
      "score": null,
      "manual_override": false,
      "execution_time_ms": 14,
      "memory_used_mb": 28,
      "submitted_at": "2026-09-02T18:30:00Z",
      "comments_count": 2
    }
  ],
  "error": "",
  "message": "Cola de entregas obtenida exitosamente"
}
```

---

### 9. Detalle SpeedGrader Desenmascarado (`GET /api/v1/teacher/submissions/{id}/review`)

Vista de corrección para el docente con **casos de prueba privados desenmascarados**, análisis AST y punteros `next_submission_id` / `prev_submission_id`.

- **Método:** `GET`
- **Ruta:** `/api/v1/teacher/submissions/{id}/review`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`: Detalle completo para revisión.
  - `400 Bad Request`: El id de la entrega no es un UUID.
  - `404 Not Found`: La entrega no existe o no pertenece a una materia del docente.
  - `403 Forbidden`: Si un estudiante intenta consultar este endpoint.

#### Ejemplo de Respuesta
```json
{
  "data": {
    "id": "sub-uuid-1",
    "exercise_id": "ex-uuid-1",
    "exercise_title": "Lab #01: Árboles AVL",
    "subject_id": "sub-uuid-1",
    "subject_name": "Estructuras de Datos",
    "student_id": "stu-uuid-1",
    "student_name": "Carlos Ruiz",
    "environment_type": "JUEZ_EFIMERO",
    "language": "python",
    "boilerplate": "def insert(root, key):\n    pass\n",
    "code": "def insert(root, key): ...",
    "verdict": "WA",
    "score": null,
    "manual_override": false,
    "ast_result": {},
    "test_cases": [
      {
        "input": "10 20 30",
        "expected_output": "20 10 30",
        "is_hidden": false,
        "passed": true
      },
      {
        "input": "50 40 30 20 10",
        "expected_output": "40 20 50 10 30",
        "is_hidden": true,
        "passed": false
      }
    ],
    "files_available": false,
    "metrics_available": true,
    "files": null,
    "base_files": null,
    "base_files_available": false,
    "comments": [
      {
        "id": "c-uuid-1",
        "line_number": 15,
        "end_line_number": 18,
        "comment": "Revisar rotación doble a la izquierda",
        "author_name": "Profesor Revisor",
        "created_at": "2026-09-02T18:40:00Z"
      }
    ],
    "prev_submission_id": "sub-uuid-0",
    "next_submission_id": "sub-uuid-2",
    "submitted_at": "2026-09-02T18:30:00Z"
  },
  "error": "",
  "message": "Detalle de revisión SpeedGrader obtenido exitosamente"
}
```

#### Campos de modalidad

- `environment_type`: `JUEZ_EFIMERO` o `IDE_PERSISTENTE`. Sale de `exercises.environment_type`; si falta o es desconocido, el backend responde `JUEZ_EFIMERO`.
- `language`: lenguaje del ejercicio. Vacío si no está definido.
- `boilerplate`: código base del ejercicio. En juez es el texto único contra el que se calcula el diff. En IDE es el proyecto base multiarchivo (formato `# === FILE: ruta ===`); el diff compara cada archivo entregado contra su homónimo. **En IDE con `files_available = true` el campo se omite**: `base_files` ya transporta el proyecto base.
- `code`: código de la entrega. En juez es obligatorio. **En IDE con `files_available = true` se omite** para no duplicar el snapshot; si `files_available = false`, el backend envía el texto crudo para que la entrega se pueda revisar en el visor plano.
- `files_available`: `true` solo en IDE cuando el snapshot se pudo separar en archivos. Si es `false`, la vista usa el visor plano de `code`.
- `metrics_available`: `true` en juez cuando la entrega ya tiene veredicto de ejecución. En IDE es `false`. La vista muestra tiempo y memoria solo con este campo; no interpreta `0` como medición.
- `files`: arreglo derivado de `code` (`path`, `language`, `content`, `start_line`, `line_count`). En juez es `null`. `start_line` es 1-based y permite traducir los comentarios a la línea local.
- `base_files`: arreglo con el mismo formato que `files`, derivado de `boilerplate` en IDE. En juez es `null`. `base_files_available` indica si el proyecto base se pudo separar; si es `false`, el diff por archivo no se muestra.
- En IDE, `test_cases` es `[]`: los casos I/O no aplican y el backend no los busca en `exercise_config`.

El parser que arma `files` y `base_files` vive en `internal/core/domain/project_snapshot.go`. Rechaza rutas absolutas, rutas con `..`, caracteres de control, rutas duplicadas, más de 500 archivos o un snapshot mayor a 2 MiB; en esos casos el arreglo correspondiente queda vacío y su bandera en `false`.

---

### 10. Override Manual de Calificación (`POST /api/v1/submissions/{id}/override`)

Permite a un docente o administrador rectificar o convalidar el veredicto y nota de una entrega con justificación obligatoria ($\ge 10$ caracteres).

- **Método:** `POST`
- **Ruta:** `/api/v1/submissions/{id}/override`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`: Calificación actualizada. Solo se responde así cuando la fila se actualizó de verdad.
  - `400 Bad Request`: El id de la entrega no es un UUID.
  - `404 Not Found`: La entrega no existe en la institución del docente. El repositorio devuelve aquí, no deja pasar el override como si hubiera guardado la nota.
  - `422 Unprocessable Entity`: Si la justificación tiene menos de 10 caracteres.
  - `403 Forbidden`: Si el rol es student.

#### Ejemplo de Solicitud
```json
{
  "verdict": "AC",
  "override_reason": "El algoritmo implementó balanceo correcto, falla en formato de impresión.",
  "score": 90
}
```

**La lectura sigue el mismo criterio.** `GET /api/v1/submissions/{id}`, el detalle de revisión del docente y el registro de keystrokes responden `400` si el id no es un UUID y `404` si la entrega no existe en la institución. La evaluación por rúbrica de una entrega ausente también responde `404`. Un fallo real del almacén sigue siendo `500`: no se disfraza de ausencia.

---

### 11. Comentarios In-line Anclados a Código (`POST & GET /api/v1/teacher/submissions/{id}/comments`)

Permite registrar y consultar comentarios pedagógicos anclados a una línea o a un bloque de líneas del código fuente.

- **Método:** `POST` / `GET`
- **Ruta:** `/api/v1/teacher/submissions/{id}/comments`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `POST`: `201 Created`.
  - `GET`: `200 OK` con lista de comentarios ordenados por `line_number`.
  - `400 Bad Request`: El id de la entrega no es un UUID (`invalid_submission_id`).
  - `404 Not Found`: La entrega no existe en la institución del docente (`submission_not_found`).
  - `422 Unprocessable Entity`: `line_number` cae fuera del rango de líneas de `submissions.code` (`line_out_of_range`), el rango está invertido (`line_range_invalid`), el bloque cruza archivos en IDE (`line_range_spans_files`) o el `file_path` no existe en el snapshot (`comment_file_unknown`). El backend valida contra `code`, que es lo que persiste.

#### Ejemplo de Solicitud POST (una línea)
```json
{
  "line_number": 42,
  "comment": "Cuidado con la condición de corte en el caso base"
}
```

#### Ejemplo de Solicitud POST (bloque)
```json
{
  "line_number": 42,
  "end_line_number": 47,
  "comment": "El bloque no valida el body antes de responder"
}
```

`end_line_number` es opcional. Cuando se omite, el comentario aplica a una sola línea. En IDE el bloque debe pertenecer a un único archivo.

#### Ancla por archivo (IDE)

La modalidad IDE acepta además `file_path`, `local_line_number` y `local_end_line_number` (1-based).

```json
{
  "file_path": "src/app.js",
  "local_line_number": 10,
  "local_end_line_number": 14,
  "comment": "El bloque no valida el body antes de responder"
}
```

- Si llega el ancla por archivo, el backend calcula `line_number = start_line + local_line_number - 1` y valida el rango contra `code`. La respuesta guarda `file_path`, `local_line_number` y la línea global.
- Si además llega `line_number`, debe coincidir con el calculado; si no, `422 line_range_invalid`.
- Si solo llega la coordenada global (clientes anteriores), el backend deriva `file_path` y `local_line_number` cuando la línea cae dentro de un archivo. Una cabecera del snapshot queda sin ancla, como los comentarios generales.
- En juez, `file_path` y `local_line_number` no se aceptan: el comentario solo usa líneas globales.
- Los comentarios anteriores a la migración `00045_submission_comment_anchors.sql` no se migran: el backend deriva su ancla al leer cuando el snapshot sigue disponible.

---

### 12. Runner Efímero Docente (`POST /api/v1/teacher/submissions/{id}/run-ephemeral`)

Ejecuta código en el sandbox efímero sin persistir una nueva entrega en la base de datos (para pruebas en vivo durante la corrección).

- **Método:** `POST`
- **Ruta:** `/api/v1/teacher/submissions/{id}/run-ephemeral`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`: Veredicto y salida de ejecución en memoria.
  - `400 Bad Request`: El id de la entrega no es un UUID.
  - `404 Not Found`: La entrega no existe en la institución del docente.
  - `409 Conflict`: La entrega pertenece a un proyecto IDE Persistente (`ephemeral_not_supported_for_ide`). Los proyectos se corrigen con rúbrica, no re-ejecutando casos I/O.
  - `403 Forbidden`: Si el rol es student.

`GET /api/v1/teacher/submissions/{id}/timeline` y `/keystroke-events` comparten ese contrato: `400` con un id mal formado y `404` cuando la entrega no existe.

#### Ejemplo de Solicitud
```json
{
  "code": "def insert(root, key):\n    # corrección en vivo\n    pass",
  "language": "python"
}
```

#### Ejemplo de Respuesta
```json
{
  "data": {
    "submission_id": "sub-uuid-1",
    "exercise_id": "ex-uuid-1",
    "verdict": "AC",
    "execution_time_ms": 12,
    "memory_used_mb": 24,
    "message": "Ejecución efímera completada con éxito"
  },
  "error": "",
  "message": "Ejecución efímera completada exitosamente"
}
```

---

### 13. Entorno de Prueba en Vivo de una Entrega IDE (`POST /api/v1/teacher/submissions/{id}/test-environment`)

Crea un contenedor efímero con una copia del snapshot de la entrega y devuelve la URL para abrir la aplicación en una pestaña nueva. El contenedor no monta el workspace vivo del alumno y el QoS Worker lo apaga por inactividad; no existe un endpoint de «detener».

- **Método:** `POST`
- **Ruta:** `/api/v1/teacher/submissions/{id}/test-environment`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Cuerpo:** vacío (`{}`).
- **Respuestas:**
  - `200 OK`: `{ "data": { "url": "...", "expires_at": "..." }, "error": "", "message": "Entorno de prueba listo" }`.
  - `400 Bad Request`: El id de la entrega no es un UUID (`invalid_submission_id`) o el tenant no es válido (`invalid_tenant`).
  - `403 Forbidden`: El rol es `student`.
  - `404 Not Found`: La entrega no existe en la institución (`submission_not_found`).
  - `409 Conflict`: La entrega es de juez (`review_environment_requires_ide`), la práctica no define `app_run_command` (`app_run_command_missing`), el snapshot no es multiarchivo (`project_snapshot_unavailable`), la práctica no resuelve imagen (`template_image_missing`), el puerto guardado está fuera de rango (`invalid_app_port`) o ya hay un preview iniciándose (`review_environment_starting`).
  - `502 Bad Gateway`: El contenedor arrancó pero la app no escuchó en el puerto (`app_not_ready`) o el contenedor no inició (`app_start_failed`).
  - `503 Service Unavailable`: El host no tiene memoria para el preview (`oom_guard`) o el nodo no tiene orquestador (`review_environment_unavailable`).

#### Ejemplo de Respuesta
```json
{
  "data": {
    "url": "https://review-3f2a9c1b7d4e.ejemplo.edu",
    "expires_at": "2026-11-01T18:45:00Z"
  },
  "error": "",
  "message": "Entorno de prueba listo"
}
```

`expires_at` es RFC3339. La UI lo muestra con el pipe `dateText` dentro de un `<time>`; la URL se abre con `target="_blank"` y `rel="noopener noreferrer"`, nunca en un iframe. Si la entrega ya tiene un preview `running`, el endpoint devuelve la misma URL y renueva la ventana de inactividad de 15 minutos.

---

### 14. Exportación de Calificaciones en CSV (`GET /api/v1/teacher/courses/{id}/grades/export`)

Genera y transmite una matriz consolidada de notas en formato CSV compatible con hojas de cálculo (UTF-8 BOM y header `Content-Disposition`).

- **Método:** `GET`
- **Ruta:** `/api/v1/teacher/courses/{id}/grades/export?format=csv`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Headers de Respuesta:**
  - `Content-Type: text/csv; charset=utf-8`
  - `Content-Disposition: attachment; filename="calificaciones_PROG-201_20260902.csv"`
- **Respuestas:**
  - `200 OK`: Archivo CSV binario con matriz de calificaciones.
  - `403 Forbidden`: Si el rol es student.
  - `404 Not Found`: Si la materia no existe o no pertenece al docente.

#### Ejemplo de Contenido CSV
```csv
ID Estudiante,Nombre Completo,Email,Lab 1: Sockets TCP,Lab 2: RPC Concurrente,Promedio Final
stu-uuid-1,Carlos Ruiz,carlos@uab.edu.bo,100,0,50.00
stu-uuid-2,Ana Torres,ana@uab.edu.bo,100,90,95.00
stu-uuid-3,María López,maria@uab.edu.bo,0,0,0.00
```

---

---

## SLICE_14: Panel Administrador Institucional

### 1. Activar Modo Mantenimiento (`POST /api/v1/admin/maintenance/enable`)

Activa el modo mantenimiento para el tenant especificado, bloqueando el acceso a estudiantes y docentes con `503 Service Unavailable` y permitiendo el bypass de administradores.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/maintenance/enable`
- **Headers:** `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Payload:**
```json
{
  "until": "2026-09-10T00:00:00Z",
  "reason": "Actualización programada de base de datos"
}
```
- **Respuestas:**
  - `200 OK`: Modo mantenimiento activado.
  - `400 Bad Request`: Formato de fecha o payload inválido.

---

### 2. Desactivar Modo Mantenimiento (`POST /api/v1/admin/maintenance/disable`)

Desactiva el modo mantenimiento y restablece el acceso regular a la plataforma.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/maintenance/disable`
- **Headers:** `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`: Modo mantenimiento desactivado.

---

### 3. Consultar Estado de Mantenimiento (`GET /api/v1/admin/maintenance/status`)

Consulta el estado actual de mantenimiento del tenant.

- **Método:** `GET`
- **Ruta:** `/api/v1/admin/maintenance/status`
- **Headers:** `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": {
    "maintenance_mode": true,
    "maintenance_until": "2026-09-10T00:00:00Z",
    "maintenance_reason": "Actualización programada de base de datos"
  },
  "error": "",
  "message": "Estado de mantenimiento obtenido exitosamente"
}
```

---

### 4. CRUD de Periodos Académicos (`/api/v1/admin/academic-periods`)

Permite gestionar los semestres y periodos institucionales, asociar materias y archivar semestres vencidos.

- **GET `/api/v1/admin/academic-periods`**: Listar periodos del tenant.
- **POST `/api/v1/admin/academic-periods`**: Crear nuevo periodo académico.
  - Payload:
```json
{
  "name": "Primer Semestre 2026",
  "code": "I-2026",
  "start_date": "2026-02-01",
  "end_date": "2026-06-30",
  "is_active": true
}
```
  - Respuestas: `201 Created`, `422 Unprocessable Entity` (si `end_date < start_date`), `409 Conflict` (código duplicado).
- **PUT `/api/v1/admin/academic-periods/{id}`**: Actualizar periodo. Respuestas: `200 OK`, `422 Unprocessable Entity`.
- **DELETE `/api/v1/admin/academic-periods/{id}`**: Eliminar periodo sin materias asociadas. Respuestas: `204 No Content`, `409 Conflict` (si tiene materias vinculadas).

---

### 5. Reasignación de Docente Titular a Materia (`POST /api/v1/admin/courses/{id}/reassign`)

Permite transferir la titularidad docente de una materia sin perder el historial académico ni las entregas de los estudiantes.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/courses/{id}/reassign`
- **Headers:** `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Payload:**
```json
{
  "new_teacher_id": "doc-uuid-2",
  "reason": "Asignación de nuevo docente titular por licencia médica"
}
```
- **Respuestas:**
  - `200 OK`: Materia reasignada exitosamente.
  - `422 Unprocessable Entity`: Si el usuario no tiene rol docente o no existe.
  - `404 Not Found`: Si la materia no existe en el tenant.
  - `403 Forbidden`: Si el rol es student.

---

### 6. Directorio Institucional de Estudiantes (`GET /api/v1/admin/students`)

Provee al administrador un listado consolidado de estudiantes con métricas agregadas de cursos matriculados, workspaces activos y penalizaciones por consumo de memoria.

- **Método:** `GET`
- **Ruta:** `/api/v1/admin/students?search=Perez&subject_id=<uuid>&status=strikes`
- **Headers:** `X-User-Role: admin | teacher`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": [
    {
      "id": "stu-uuid-1",
      "first_name": "Juan",
      "last_name": "Pérez",
      "email": "juan@uab.edu.bo",
      "role": "student",
      "enrolled_courses_count": 3,
      "active_workspaces_count": 1,
      "oom_strike_count": 0,
      "last_oom_killed_at": null
    }
  ],
  "error": "",
  "message": "Directorio de estudiantes obtenido exitosamente"
}
```

---

### 7. Reset Manual de Penalizaciones OOM-Killed (`POST /api/v1/admin/students/{id}/reset-oom`)

Permite al administrador resetear a 0 los strikes de penalización por exceso de memoria de un estudiante tras una justificación pedagógica ($\ge 10$ caracteres).

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/students/{id}/reset-oom`
- **Headers:** `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Payload:**
```json
{
  "reason": "El estudiante corrigió la fuga de memoria en sus algoritmos concurrentes"
}
```
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": {
    "student_id": "stu-uuid-1",
    "workspaces_reset_count": 1,
    "message": "Penalizaciones OOM reseteadas exitosamente"
  },
  "error": "",
  "message": "Penalizaciones OOM reseteadas exitosamente"
}
```
  - `422 Unprocessable Entity`: Si la justificación tiene menos de 10 caracteres.
  - `404 Not Found`: Si el estudiante no existe en el tenant.
  - `403 Forbidden`: Si el rol es student.

---

### 8. Listado Institucional de Plantillas Docker (`GET /api/v1/admin/templates`)

Permite a los administradores y docentes inspeccionar el catálogo de plantillas base, su estado de aprobación y los parámetros de recursos asignados.

- **Método:** `GET`
- **Ruta:** `/api/v1/admin/templates?status=pending&search=rust`
- **Headers:** `X-User-Role: admin | teacher`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": [
    {
      "id": "tpl-uuid-1",
      "name": "Rust Async Lab",
      "docker_image": "solv/rust:1.75",
      "base_ram_mb": 512,
      "status": "pending",
      "rejection_reason": "",
      "reviewed_by": null,
      "reviewed_at": null,
      "requested_by": "doc-uuid-1",
      "description": "Plantilla para laboratorio concurrente",
      "created_at": "2026-09-02T22:00:00Z"
    }
  ],
  "error": "",
  "message": "Listado de plantillas obtenido exitosamente"
}
```

---

### 9. Aprobación o Rechazo de Plantillas Docker (`PUT /api/v1/admin/templates/{id}/review`)

Permite al administrador aprobar una plantilla solicitada (ajustando opcionalmente la memoria RAM base) o rechazarla con un motivo explícito.

- **Método:** `PUT`
- **Ruta:** `/api/v1/admin/templates/{id}/review`
- **Headers:** `X-User-Role: admin`, `X-User-Id: <admin_uuid>`, `X-Tenant-Id: <tenant_uuid>`
- **Payload Aprobación:**
```json
{
  "status": "approved",
  "base_ram_mb": 1024
}
```
- **Payload Rechazo:**
```json
{
  "status": "rejected",
  "rejection_reason": "La imagen solicitada contiene paquetes no autorizados para el plan docente"
}
```
- **Respuestas:**
  - `200 OK`: Plantilla revisada y actualizada exitosamente.
  - `422 Unprocessable Entity`: Si el estado no es `approved`/`rejected`, o si es `rejected` sin `rejection_reason`.
  - `404 Not Found`: Si la plantilla no existe.
  - `403 Forbidden`: Si el rol no es `admin`.

---

### 10. Acciones de Emergencia Administrativas con Confirmación Tipada (`POST /api/v1/admin/emergency/{action}`)

Permite al administrador ejecutar comandos de pánico para salvaguardar la estabilidad del host ante incidentes críticos, exigiendo una frase de confirmación exacta para prevenir ejecuciones accidentales.

- **Método:** `POST`
- **Rutas soportadas:**
  - `/api/v1/admin/emergency/terminate_all_workspaces` (Frase: `"TERMINAR TODOS LOS WORKSPACES"`)
  - `/api/v1/admin/emergency/hibernate_all_workspaces` (Frase: `"HIBERNAR TODOS LOS WORKSPACES"`)
  - `/api/v1/admin/emergency/kill_zombies` (Frase: `"LIMPIAR ZOMBIES DOCKER"`)
- **Headers:** `X-User-Role: admin`, `X-User-Id: <admin_uuid>`, `X-Tenant-Id: <tenant_uuid>`
- **Payload:**
```json
{
  "confirmation_phrase": "TERMINAR TODOS LOS WORKSPACES",
  "reason": "Saturación de memoria por procesos descontrolados"
}
```
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": {
    "action": "terminate_all_workspaces",
    "affected_count": 5,
    "executed_by": "admin-uuid-1",
    "message": "Se terminaron forzosamente 5 workspaces activos"
  },
  "error": "",
  "message": "Se terminaron forzosamente 5 workspaces activos"
}
```
  - `422 Unprocessable Entity`: Si la frase de confirmación no coincide exactamente o la acción no existe.
  - `403 Forbidden`: Si el rol no es `admin`.

---

---

## SLICE_15: Sistema de Notificaciones Proactivas (ADR-034)

### 1. Listado Paginado de Notificaciones (`GET /api/v1/notifications`)

Consulta el historial de notificaciones del usuario autenticado ordenadas cronológicamente de forma descendente, con soporte de paginación y filtro de sólo no leídas.

- **Método:** `GET`
- **Ruta:** `/api/v1/notifications?unread_only=false&page=1&limit=20`
- **Headers:** `X-User-Id: <user_uuid>`, `X-Tenant-Id: <tenant_uuid>`
- **Response Headers:** `X-Total-Count: <total_notificaciones>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": [
    {
      "id": "notif-uuid-1",
      "tenant_id": "00000000-0000-0000-0000-000000000001",
      "recipient_user_id": "user-uuid-1",
      "channel": "in_app",
      "severity": "critical",
      "title": "Alerta de Consumo OOM",
      "message": "Tu workspace fue suspendido por superar el límite de memoria",
      "event_type": "oom_warning",
      "metadata": {},
      "is_read": false,
      "read_at": null,
      "email_sent_at": null,
      "occurrence_count": 1,
      "link": "/student/workspaces",
      "created_at": "2026-09-03T06:00:00Z"
    }
  ],
  "error": "",
  "message": "Notificaciones obtenidas exitosamente"
}
```

---

### 2. Contador Ultraliviano para Campana UI (`GET /api/v1/notifications/unread-count`)

Consulta de alta velocidad optimizada mediante índice parcial en PostgreSQL (`WHERE is_read = false`), diseñada para sondeo frecuente desde el frontend sin sobrecargar el servidor.

- **Método:** `GET`
- **Ruta:** `/api/v1/notifications/unread-count`
- **Headers:** `X-User-Id: <user_uuid>`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": {
    "unread_count": 3
  },
  "error": "",
  "message": "Contador de notificaciones no leídas obtenido exitosamente"
}
```

---

### 3. Marcar Notificación Individual como Leída (`PATCH /api/v1/notifications/{id}/read`)

Marca una notificación específica como leída, asignando el timestamp de lectura actual (`read_at = NOW()`) y validando el aislamiento multi-tenant y pertenencia de usuario.

- **Método:** `PATCH`
- **Ruta:** `/api/v1/notifications/{id}/read`
- **Headers:** `X-User-Id: <user_uuid>`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": {
    "id": "notif-uuid-1",
    "status": "marked_as_read"
  },
  "error": "",
  "message": "Notificación marcada como leída"
}
```
  - `404 Not Found`: Si la notificación no existe o pertenece a otro usuario.

---

### 4. Marcar Todas las Notificaciones como Leídas (`POST /api/v1/notifications/mark-all-read`)

Actualiza en una sola operación atómica todas las notificaciones pendientes del usuario a estado leído.

- **Método:** `POST`
- **Ruta:** `/api/v1/notifications/mark-all-read`
- **Headers:** `X-User-Id: <user_uuid>`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": {
    "marked_count": 3
  },
  "error": "",
  "message": "Todas las notificaciones fueron marcadas como leídas"
}
```

---

---

## SLICE_16: Backups Configurables y Retención Institucional (ADR-035)

### 1. Consulta y Modificación de Configuración de Respaldos (`GET & PUT /api/v1/admin/backups/config`)

Permite al administrador consultar y actualizar las frecuencias y políticas de retención de las copias de seguridad locales y remotas.

- **Métodos:** `GET` / `PUT`
- **Ruta:** `/api/v1/admin/backups/config`
- **Headers:** `X-User-Id: <admin_uuid>`, `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Payload (`PUT`):**
```json
{
  "local_frequency_hours": 12,
  "local_retention_days": 14,
  "remote_enabled": true,
  "remote_provider": "backblaze_b2",
  "remote_bucket_name": "solv-institutional-backups",
  "remote_endpoint": "https://s3.us-west-002.backblazeb2.com"
}
```
- **Respuestas (`GET` / `PUT`):**
  - `200 OK`:
```json
{
  "data": {
    "id": "config-uuid-1",
    "tenant_id": "00000000-0000-0000-0000-000000000001",
    "local_frequency_hours": 12,
    "local_retention_days": 14,
    "remote_enabled": true,
    "remote_provider": "backblaze_b2",
    "remote_bucket_name": "solv-institutional-backups",
    "remote_endpoint": "https://s3.us-west-002.backblazeb2.com",
    "remote_retention_days": 30,
    "is_active": true,
    "updated_at": "2026-09-03T07:00:00Z"
  },
  "error": "",
  "message": "Configuración de respaldos obtenida exitosamente"
}
```
  - `403 Forbidden`: Si el rol no es `admin`.

---

### 2. Historial de Respaldos (`GET /api/v1/admin/backups`)

Consulta el historial paginado de todas las ejecuciones de respaldo realizadas, detallando tamaño, checksum SHA-256 inmutable y estado de ejecución.

- **Método:** `GET`
- **Ruta:** `/api/v1/admin/backups?page=1&limit=20`
- **Headers:** `X-User-Id: <admin_uuid>`, `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Response Headers:** `X-Total-Count: <total_respaldos>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": [
    {
      "id": "backup-uuid-1",
      "tenant_id": "00000000-0000-0000-0000-000000000001",
      "file_name": "solv_backup_00000000_1788418800.dump.gz",
      "file_size_bytes": 1048576,
      "sha256_checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "storage_tier": "local",
      "status": "success",
      "error_message": "",
      "started_at": "2026-09-03T07:00:00Z",
      "completed_at": "2026-09-03T07:00:02Z"
    }
  ],
  "error": "",
  "message": "Historial de respaldos obtenido exitosamente"
}
```
  - `403 Forbidden`: Si el rol no es `admin`.

---

### 3. Disparo Manual de Respaldo (`POST /api/v1/admin/backups/trigger`)

Inicia la generación inmediata de un volcado comprimido (`.dump.gz`), calcula su hash SHA-256, aplica la política de rotación de retención y notifica al administrador institucional.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/backups/trigger`
- **Headers:** `X-User-Id: <admin_uuid>`, `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `201 Created`:
```json
{
  "data": {
    "id": "backup-uuid-1",
    "tenant_id": "00000000-0000-0000-0000-000000000001",
    "file_name": "solv_backup_00000000_1788418800.dump.gz",
    "file_size_bytes": 1048576,
    "sha256_checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "storage_tier": "local",
    "status": "success",
    "started_at": "2026-09-03T07:00:00Z",
    "completed_at": "2026-09-03T07:00:02Z"
  },
  "error": "",
  "message": "Copia de seguridad ejecutada exitosamente"
}
```
  - `403 Forbidden`: Si el rol no es `admin`.

---

### 4. Verificación Criptográfica de Integridad (`POST /api/v1/admin/backups/{id}/verify`)

Lee el archivo físico almacenado en disco, computa su hash SHA-256 actual y lo compara con el registro original de la base de datos para certificar ausencia de corrupción.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/backups/{id}/verify`
- **Headers:** `X-User-Id: <admin_uuid>`, `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "data": {
    "execution_id": "backup-uuid-1",
    "file_name": "solv_backup_00000000_1788418800.dump.gz",
    "database_checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "computed_checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "is_valid": true,
    "message": "Integridad criptográfica verificada: el archivo no está corrupto ni alterado"
  },
  "error": "",
  "message": "Verificación de integridad completada"
}
```
  - `404 Not Found`: Si el respaldo no existe.
  - `403 Forbidden`: Si el rol no es `admin`.

---

### 5. Descarga Directa de Respaldo (`GET /api/v1/admin/backups/{id}/download`)

Transmite el flujo binario del archivo `.dump.gz` al cliente con cabeceras `Content-Disposition: attachment`.

- **Método:** `GET`
- **Ruta:** `/api/v1/admin/backups/{id}/download`
- **Headers:** `X-User-Id: <admin_uuid>`, `X-User-Role: admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`: Flujo binario con `Content-Type: application/gzip` y `Content-Disposition: attachment; filename="..."`.
  - `404 Not Found`: Si el archivo no se encuentra en el almacenamiento local.
  - `403 Forbidden`: Si el rol no es `admin`.

---

## Gestión de Caché y Precalentamiento de Plantillas Docker (Rol Administrador)

### 1. Estado de Caché Local de Plantillas (`GET /api/v1/admin/templates/cache-status`)

Inspecciona la presencia de las imágenes Docker de todas las plantillas del tenant en el almacenamiento local del host Docker (`InspectLocalImage`), indicando si residen en disco o requieren descarga remota.

- **Método:** `GET`
- **Ruta:** `/api/v1/admin/templates/cache-status`
- **Headers Requeridos:**
  - `X-User-Id: <admin_uuid>`
  - `X-User-Role: admin`
  - `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "items": [
    {
      "template_id": "77777777-7777-4777-a777-777777777777",
      "template_name": "Python Data Science",
      "docker_image": "python:3.11-slim",
      "cached": true,
      "size_mb": 145,
      "local_image_id": "sha256:d8f2..."
    },
    {
      "template_id": "88888888-8888-4888-a888-888888888888",
      "template_name": "Rust 1.80 Systems",
      "docker_image": "rust:1.80-slim",
      "cached": false,
      "size_mb": 0,
      "local_image_id": ""
    }
  ]
}
```
  - `403 Forbidden`: Si el rol no es `admin`.

---

### 2. Calentamiento Bajo Demanda de Plantilla Individual (`POST /api/v1/admin/templates/{id}/warm`)

Descarga y almacena localmente la imagen OCI de una plantilla específica mediante *single-flight deduplication* en el Docker Engine del host, garantizando que futuras solicitudes no bloqueen ni compitan por ancho de banda.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/templates/{id}/warm`
- **Headers Requeridos:**
  - `X-User-Id: <admin_uuid>`
  - `X-User-Role: admin`
  - `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "template_id": "88888888-8888-4888-a888-888888888888",
  "template_name": "Rust 1.80 Systems",
  "docker_image": "rust:1.80-slim",
  "cached": true,
  "size_mb": 312,
  "local_image_id": "sha256:a1b2..."
}
```
  - `404 Not Found`: Si la plantilla no existe en el tenant.
  - `502 Bad Gateway`: Si ocurre un error de red o la imagen no existe en el registro remoto (`not_found_remote` o `network_failure`).

---

### 3. Calentamiento Masivo Pre-Clase (`POST /api/v1/admin/templates/warm-all`)

Ejecuta el precalentamiento secuencial seguro de todas las plantillas registradas del tenant para asegurar la invariante *Golden Image* antes del inicio de clases presenciales.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/templates/warm-all`
- **Headers Requeridos:**
  - `X-User-Id: <admin_uuid>`
  - `X-User-Role: admin`
  - `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`:
```json
{
  "total_templates": 6,
  "pulled": 2,
  "already_cached": 4,
  "failed": 0
}
```
  - `403 Forbidden`: Si el rol no es `admin`.

---

## Módulo de Inteligencia Artificial Híbrida (Nube + Servidor Local LAN)

Contratos del Copiloto Institucional: configuración por institución, prueba de conexión, estado liviano, sugerencia de componentes de catálogo (admin) y borradores de ejercicio con sello determinista del juez (docente).

### Reglas comunes del módulo

- La configuración se persiste por tenant en `tenants.config["ai_settings"]`.
- La API Key se guarda cifrada con AES-256-GCM (`base64(nonce + ciphertext)`) y jamás se expone: las respuestas solo incluyen `has_api_key` y `api_key_masked` (ej. `AIza...9f2a`).
- Cuota horaria por usuario (`hourly_quota_per_user`, 1..200, default 15) con ventana en memoria de 1 hora. En motores `local_gpu` (`ollama_lan` o proveedor `custom_*`) el piso efectivo es de 60 solicitudes/hora, por no existir costo por token externo.
- **Gobernanza autoadaptativa del timeout**: `timeout_seconds` (5..240, default 60) es un tope, no un valor adivinado. El servidor clasifica el motor activo (`engine_profile`) y resuelve el presupuesto de tiempo por acción (`min(240, max(presupuestoAutomático, timeout_seconds))`), de modo que un tope viejo de `30` jamás recorta una operación pesada y un tope mayor fijado en Modo Experto se respeta.

| `engine_profile` | Condición | Presupuesto por acción (`suggest-component` · `suggest-template` / `repair-template` · `generate-exercise` · `test-connection`) | `auto_timeout_range_label` |
|------------------|-----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------|
| `fast_cloud` | Modelo comercial en la nube (`gemini-2.5-flash`, `gpt-4o-mini`, `deepseek-chat`, ...) | `60 s` · `90 s` · `120 s` · `30 s` | `60s – 120s según complejidad de la tarea` |
| `queued_or_reasoning` | Modelo gratuito o de razonamiento (`:free`, `r1`, `reasoner`, `thinking`, `nemotron`, `qwq`, `o1`, `o3`) | `90 s` · `120 s` · `180 s` · `60 s` | `90s – 180s según complejidad de la tarea` |
| `local_gpu` | Servidor LAN propio (`ollama_lan`) o proveedor `custom_*` | `90 s` · `120 s` · `180 s` · `60 s` | `90s – 180s según complejidad de la tarea` |

- **Cascada BYOK personal** (`/api/v1/ai/my-settings`): docentes y administradores pueden guardar su propia API Key indexada por `userID` en `ai_settings.user_overrides`. Cuando está activa, la inferencia usa el proveedor, modelo, `base_url` y clave personales, opera incluso con la IA institucional deshabilitada y **no descuenta** la cuota horaria institucional compartida.
- La latencia (`latency_ms`) se mide **después de descargar el cuerpo completo** de la respuesta del motor (no solo los headers) y acumula todos los intentos realizados.
- Blindaje de transporte: si un modelo gratuito rechaza `response_format: json_object` con `400 Bad Request`, el backend reintenta una única vez sin ese campo; las pruebas de conexión aceptan JSON envuelto en prosa de razonamiento o en fences Markdown (```` ```json ... ``` ````).
- Escudo anti-SSRF: para `gemini`/`openai`/`openrouter`/`deepseek` la URL base oficial se fuerza y se ignora cualquier `base_url` del cliente; `ollama_lan` exige `http://` o `https://` y rechaza loopback (`localhost`, `127.0.0.0/8`, `::1`), `0.0.0.0`, rangos link-local/metadata (`169.254.0.0/16`) y nombres internos de contenedores sin dominio (`postgres`, `redis`, `traefik`). Se permiten IPs privadas LAN (`10.x`, `192.168.x`, `172.16-31.x`) y dominios con punto.

#### Códigos de estado del módulo

| Código | `error` | Condición |
|--------|---------|-----------|
| `200 OK` | `""` | Operación completada |
| `400 Bad Request` | `invalid_payload`, `prompt_invalid`, `layer_invalid`, `language_invalid`, `provider_invalid`, `base_url_required`, `base_url_invalid`, `base_url_blocked`, `base_image_latest`, `base_image_unpinned`, `hourly_quota_invalid`, `timeout_invalid`, `ai_response_invalid`, `api_key_required`, `model_required` | Validación de entrada, escudo anti-SSRF o respuesta del LLM inválida |
| `403 Forbidden` | `forbidden` | Rol insuficiente (sugerencia/autorreparación de plantilla solo `admin`; borradores bloqueados para `student`; `/api/v1/ai/my-settings*` solo `teacher` y `admin`) |
| `401 Unauthorized` | texto plano | Sin sesión válida (`tm(...)`) |
| `409 Conflict` | `ai_disabled` | `domain.ErrAIDisabled`: IA deshabilitada para el instituto |
| `429 Too Many Requests` | `ai_quota_exceeded` | `domain.ErrAIQuotaExceeded`: cuota horaria agotada |
| `500 Internal Server Error` | texto del error | Falla interna o del tenant; `message` conserva el motivo técnico real (`<contexto>: <causa>`) |
| `504 Gateway Timeout` | `ai_timeout` | `domain.ErrAITimeout`: el motor superó el presupuesto de tiempo de la operación |

---

### 1. Consulta de Configuración de IA (`GET /api/v1/admin/ai/settings`)

- **Método:** `GET`
- **Ruta:** `/api/v1/admin/ai/settings`
- **Headers Requeridos:**
  - `X-User-Id: <admin_uuid>`
  - `X-User-Role: admin`
  - `X-Tenant-Id: <tenant_uuid>` (opcional, fallback por defecto)

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "enabled": false,
    "provider": "gemini",
    "model": "gemini-2.5-flash",
    "base_url": "https://generativelanguage.googleapis.com/v1beta/openai",
    "has_api_key": false,
    "api_key_masked": "",
    "hourly_quota_per_user": 15,
    "timeout_seconds": 60,
    "updated_at": "",
    "engine_profile": "fast_cloud",
    "auto_timeout_range_label": "60s – 120s según complejidad de la tarea"
  },
  "error": "",
  "message": "Configuración de IA obtenida exitosamente"
}
```

- `403 Forbidden`: Si el rol no es `admin`.

---

### 2. Actualización de Configuración de IA (`PUT /api/v1/admin/ai/settings`)

- **Método:** `PUT`
- **Ruta:** `/api/v1/admin/ai/settings`
- **Headers Requeridos:** `X-User-Id`, `X-User-Role: admin`, `X-Tenant-Id`

#### Payload de Entrada
```json
{
  "enabled": true,
  "provider": "gemini",
  "model": "gemini-2.5-flash",
  "base_url": "",
  "api_key": "AIza...",
  "clear_api_key": false,
  "hourly_quota_per_user": 15,
  "timeout_seconds": 30
}
```

Reglas:
- `provider`: `gemini` | `openai` | `openrouter` | `deepseek` | `ollama_lan`.
- En proveedores cloud se ignora `base_url` y se persiste la URL oficial; en `ollama_lan` es obligatoria y pasa por el escudo anti-SSRF.
- `api_key` vacía con `clear_api_key: false` conserva la clave previa; `clear_api_key: true` la elimina.
- Si `enabled: true` y el proveedor es cloud sin API Key (nueva ni previa): `400` con `Se requiere una API Key para habilitar un proveedor en la nube`.
- `hourly_quota_per_user` 1..200 (0 aplica default 15); `timeout_seconds` 5..120 (0 aplica default 30).

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "enabled": true,
    "provider": "gemini",
    "model": "gemini-2.5-flash",
    "base_url": "https://generativelanguage.googleapis.com/v1beta/openai",
    "has_api_key": true,
    "api_key_masked": "AIza...9f2a",
    "hourly_quota_per_user": 15,
    "timeout_seconds": 30,
    "updated_at": "2026-10-24T14:03:11Z"
  },
  "error": "",
  "message": "Configuración de IA actualizada"
}
```

- `400 Bad Request`: Validaciones (`provider_invalid`, `base_url_blocked`, `hourly_quota_invalid`, `timeout_invalid`, `api_key_required`, ...).
- `403 Forbidden`: Si el rol no es `admin`.

---

### 3. Prueba de Conexión al Motor (`POST /api/v1/admin/ai/test-connection`)

Verificación real contra el motor configurado pidiendo el JSON mínimo `{"ok":true}` y midiendo la latencia. Los fallos de validación local devuelven `400`; un fallo del motor remoto responde `200` con `ok: false`.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/ai/test-connection`
- **Headers Requeridos:** `X-User-Id`, `X-User-Role: admin`, `X-Tenant-Id`

#### Payload de Entrada
```json
{
  "provider": "gemini",
  "model": "gemini-2.5-flash",
  "base_url": "",
  "api_key": ""
}
```
> `api_key` vacía usa la clave cifrada guardada en el tenant; `provider`/`model`/`base_url` vacíos heredan los guardados.

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "ok": true,
    "provider": "gemini",
    "model": "gemini-2.5-flash",
    "latency_ms": 340,
    "message": "Conexión verificada correctamente con el motor de IA."
  },
  "error": "",
  "message": "Prueba de conexión completada"
}
```

#### Comando `curl` de Prueba
```bash
curl -X POST "http://localhost:3000/api/v1/admin/ai/test-connection" \
     -H "Authorization: Bearer <jwt_admin>" \
     -H "Content-Type: application/json" \
     -d '{"provider":"ollama_lan","model":"qwen2.5-coder:7b","base_url":"http://10.20.1.50:11434/v1","api_key":""}'
```

- `400 Bad Request`: `provider_invalid`, `base_url_required`, `base_url_blocked`.
- `403 Forbidden`: Si el rol no es `admin`.

---

### 4. Catálogo Dinámico de Modelos de IA (`POST /api/v1/admin/ai/models`)

Descubre dinámicamente los modelos ofrecidos por el proveedor de IA. Si `api_key` se envía en el payload, se utiliza en memoria; si se omite, se descifra la clave guardada en el tenant. Aplica el escudo anti-SSRF (proveedores cloud usan URL oficial fija, `ollama_lan` valida host de LAN).
Mantiene una caché en memoria de 5 minutos por `(tenant_id, provider, base_url)` para llamadas con credenciales guardadas y `refresh: false`.
Normaliza identificadores (remueve el prefijo `models/` en Gemini), excluye modelos no aptos para chat o código (embeddings, whisper, tts, dall-e, moderación, etc.), clasifica el nivel/precio (`free`, `paid`, `local`, `by_account`) y ordena los resultados (OpenRouter: gratuitos primero y luego alfabético; resto de proveedores: orden alfabético por ID).

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/ai/models`
- **Headers Requeridos:** `X-User-Id`, `X-User-Role: admin`, `X-Tenant-Id`

#### Payload de Entrada
```json
{
  "provider": "openrouter",
  "base_url": "",
  "api_key": "sk-or-...",
  "refresh": false
}
```
> `provider` obligatorio ("gemini" | "openai" | "openrouter" | "deepseek" | "ollama_lan"); `base_url` opcional (solo editable en `ollama_lan`); `api_key` opcional (si se omite se usa la guardada del tenant; requerida para cloud); `refresh` booleano opcional (fuerza consulta en vivo ignorando la caché).

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "provider": "openrouter",
    "cached": false,
    "count": 2,
    "models": [
      {
        "id": "meta-llama/llama-3.3-70b-instruct:free",
        "name": "Llama 3.3 70B Instruct (free)",
        "description": "Modelo de código y razonamiento",
        "tier": "free",
        "pricing_hint": "Gratis",
        "context_length": 131072
      },
      {
        "id": "anthropic/claude-3.5-sonnet",
        "name": "Claude 3.5 Sonnet",
        "description": "Modelo de alto rendimiento",
        "tier": "paid",
        "pricing_hint": "De pago",
        "context_length": 200000
      }
    ]
  },
  "error": "",
  "message": "Modelos de IA obtenidos"
}
```

- `400 Bad Request`: `provider_invalid`, `base_url_required`, `base_url_blocked`, `api_key_required`.
- `403 Forbidden`: Si el rol no es `admin`.

---

### 5. Estado Liviano de IA (`GET /api/v1/ai/status`)

- **Método:** `GET`
- **Ruta:** `/api/v1/ai/status`
- **Headers Requeridos:** `X-User-Id` (cualquier usuario autenticado del tenant)

Resuelve por usuario la política autoadaptativa vigente (`engine_profile`, `auto_timeout_range_label`) y la cascada personal: si el usuario tiene una API Key personal activa, `enabled` refleja su motor personal, `using_personal_key` pasa a `true` y `institutional_enabled` conserva el estado de la base del instituto.

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "enabled": true,
    "provider": "gemini",
    "provider_label": "Google Gemini",
    "model": "gemini-2.5-flash",
    "engine_profile": "fast_cloud",
    "auto_timeout_range_label": "60s – 120s según complejidad de la tarea",
    "using_personal_key": false,
    "institutional_enabled": true
  },
  "error": "",
  "message": "Estado de IA obtenido"
}
```

- `401 Unauthorized`: Sin sesión válida.

---

### 6. Sugerencia de Componente de Catálogo (`POST /api/v1/admin/ai/suggest-component`)

Genera un componente tecnológico a partir de la descripción del administrador. El JSON del LLM se sanea sin confianza ciega: se limpian fences Markdown, se aplica `NormalizeCreateComponentDTO`, se rechazan imágenes `:latest` o sin tag fijo y la capa de build se sintetiza de forma determinista desde `install_mode + package_names`.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/ai/suggest-component`
- **Headers Requeridos:** `X-User-Id`, `X-User-Role: admin`, `X-Tenant-Id`

#### Payload de Entrada
```json
{
  "prompt": "PyTorch CPU y scikit-learn para Inteligencia Artificial",
  "layer": "addon",
  "target_runtime": "python"
}
```
> `prompt` 3..600 caracteres (obligatorio); `layer` opcional (`runtime` | `addon` | `database`, default `addon`); `target_runtime` opcional.

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "component": {
      "name": "PyTorch CPU + scikit-learn",
      "slug": "python-ml-stack",
      "layer": "addon",
      "sub_category": "toolkit",
      "description": "Framework de machine learning para Python",
      "icon_key": "python",
      "base_image": "python:3.12-slim-bookworm",
      "compatible_runtimes": ["python"],
      "install_mode": "pip",
      "package_names": ["torch", "scikit-learn"],
      "build_layer_snippet": "RUN pip install --no-cache-dir torch scikit-learn",
      "setup_snippet": "",
      "judge_entrypoint": "",
      "tools_provided": [],
      "ram_delta_mb": 0
    },
    "rationale": "torch cubre el entrenamiento y scikit-learn los modelos clásicos.",
    "latency_ms": 812
  },
  "error": "",
  "message": "Sugerencia de componente generada"
}
```

Códigos: `200 OK`; `400 Bad Request` (`prompt_invalid`, `layer_invalid`, `base_image_latest`, `base_image_unpinned`, `build_layer_invalid`, `ai_response_invalid`); `403 Forbidden` si el rol no es `admin`; `409 Conflict` (`ai_disabled`); `429 Too Many Requests` (`ai_quota_exceeded`); `504 Gateway Timeout` (`ai_timeout`).

#### Capa `database` (motores de base de datos)

Cuando el administrador pide un componente con `layer: "database"`, el saneo es cerrado y no depende de la salida del LLM: la capa pedida manda sobre la que devuelva el motor, `service_engine` se canonicaliza dentro del dominio soportado (`sqlite`, `postgres`, `mysql`, `mariadb`, `mongodb`, `redis`) con `postgres` como respaldo, `sub_category` pasa a `relational`/`nosql`, `base_image` se rellena con tag fijo (`mysql:8.4`, `mariadb:11.4`, `mongo:7.0`, `redis:7.2-alpine`), `install_mode` queda en `none` sin paquetes y `ram_delta_mb` toma la huella del motor (0 sqlite, 50 postgres, 64 redis, 128 mysql/mariadb/mongodb). La raíz de la respuesta repite el motor resuelto en `service_engine`, además de `component.service_engine`.

```json
{
  "data": {
    "component": {
      "name": "MariaDB 11.4",
      "slug": "mariadb-114",
      "layer": "database",
      "sub_category": "relational",
      "icon_key": "mysql",
      "base_image": "mariadb:11.4",
      "compatible_runtimes": ["universal"],
      "install_mode": "none",
      "package_names": [],
      "build_layer_snippet": "",
      "ram_delta_mb": 128
    },
    "service_engine": "mariadb",
    "rationale": "MariaDB cubre el laboratorio relacional con la sintaxis clásica de SQL.",
    "latency_ms": 798
  },
  "error": "",
  "message": "Sugerencia de componente generada"
}
```

---

### 7. Propuesta de Arquitectura de Entorno — Copiloto (`POST /api/v1/admin/ai/suggest-template`)

El administrador describe en lenguaje natural qué debe resolver el entorno (librerías, paquetes, base de datos, script de preparación) y el Copiloto devuelve una **propuesta completa de arquitectura**, sin tocar nada: runtime principal, plataforma web, complementos, motor de base de datos, RAM base, script de setup, nombre y descripción sugeridos, con justificación técnica. El backend no aplica ni persiste: quien resuelve contra el catálogo real del tenant es el frontend (por id, slug o `icon_key`), marca las piezas que hoy no existen como no disponibles y deja que el administrador decida si **aplicar al constructor** o **descartar** antes de que se sincronice el constructor visual y el manifiesto técnico del Modo Avanzado.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/ai/suggest-template`
- **Headers Requeridos:** `X-User-Id`, `X-User-Role: admin`, `X-Tenant-Id`

#### Payload de Entrada
```json
{
  "prompt": "Entorno para Algoritmos Avanzados con Python 3.12, PyTest y SQLite para pruebas de concurrencia",
  "target_environment": "JUEZ_EFIMERO"
}
```
> `prompt` 3..600 caracteres (obligatorio); `target_environment` opcional (`JUEZ_EFIMERO` | `IDE_PERSISTENTE` | `AMBOS`, default `JUEZ_EFIMERO`).

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "name": "Algoritmos con Python y SQLite",
    "description": "Juez automático para estructuras de datos",
    "runtime_slug": "python",
    "web_platform_slug": "",
    "addon_slugs": ["pytest"],
    "database_slug": "sqlite",
    "setup_script": "export PIP_NO_CACHE=1",
    "base_ram_mb": 768,
    "rationale": "Python 3.12 con pytest para los casos de prueba y SQLite como servidor de datos acoplado.",
    "latency_ms": 812
  },
  "error": "",
  "message": "Propuesta de arquitectura generada"
}
```
> El servicio normaliza al responder: slugs en minúscula sin duplicados (`addon_slugs`), `base_ram_mb` acotado a [256, 4096] con default 512, y `rationale` obligatorio. La propuesta siempre se sanea contra slugs canónicos de ecosistema; el frontend es quien valida la existencia real en el catálogo del tenant antes de aplicar.

Códigos: `200 OK`; `400 Bad Request` (`prompt_invalid`, `target_environment_invalid`, `ai_response_invalid`, `runtime_slug_missing`); `403 Forbidden` si el rol no es `admin`; `409 Conflict` (`ai_disabled`); `429 Too Many Requests` (`ai_quota_exceeded`); `504 Gateway Timeout` (`ai_timeout`).

---

### 8. Borrador de Ejercicio con Sello del Juez (`POST /api/v1/teacher/ai/generate-exercise`)

Genera el borrador completo de un ejercicio (consigna, boilerplate, solución de referencia y casos). Cuando `verify_with_runner` no es `false` y hay runner disponible, cada caso se ejecuta con la solución de referencia y `expected_output` se sobrescribe con la salida real del runner (sello determinista): los veredictos `CE`, `RE`, `TLE` o `MLE` de la referencia dejan el caso sin verificar y detallan el motivo en `verification_warning`.

- **Método:** `POST`
- **Ruta:** `/api/v1/teacher/ai/generate-exercise`
- **Headers Requeridos:** `X-User-Id`, `X-Tenant-Id`; el rol `student` recibe `403`.

#### Payload de Entrada
```json
{
  "prompt": "Leer dos enteros desde stdin e imprimir su suma",
  "language": "python",
  "difficulty": "easy",
  "public_cases_count": 2,
  "hidden_cases_count": 3,
  "verify_with_runner": true
}
```
> `prompt` 5..1500 caracteres; `language` obligatorio; `difficulty` default `medium`; `public_cases_count` default 2 (1..5); `hidden_cases_count` default 3 (1..10); `verify_with_runner` nil equivale a `true`.

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "title": "Suma de dos enteros",
    "description": "# Suma\nLea dos enteros e imprima su suma.",
    "difficulty": "easy",
    "language": "python",
    "boilerplate": "a, b = map(int, input().split())\n# imprima a + b",
    "reference_solution": "a, b = map(int, input().split())\nprint(a + b)",
    "test_cases": [
      {
        "order_index": 1,
        "input": "1 2\n",
        "expected_output": "3",
        "visibility": "example",
        "weight": 1,
        "verified_by_runner": true
      },
      {
        "order_index": 2,
        "input": "10 5\n",
        "expected_output": "15",
        "visibility": "public",
        "weight": 1,
        "verified_by_runner": true
      }
    ],
    "runner_verified_all": true,
    "latency_ms": 1470
  },
  "error": "",
  "message": "Borrador de ejercicio generado"
}
```

Cuando la solución de referencia falla en el runner la respuesta sigue siendo `200` con `runner_verified_all: false` y `verification_warning` indicando el caso y el veredicto; `verification_warning` también se emite si el motor no devolvió `reference_solution`.

Códigos: `200 OK`; `400 Bad Request` (`prompt_invalid`, `language_invalid`, `ai_response_invalid`); `403 Forbidden` (rol `student`); `409 Conflict` (`ai_disabled`); `429 Too Many Requests` (`ai_quota_exceeded`); `504 Gateway Timeout` (`ai_timeout`).

---

### 9. Autorreparación Diagnóstica de Plantilla (`POST /api/v1/admin/ai/repair-template`)

Repara el manifiesto técnico de una plantilla cuando falla **Comprobar funcionamiento**: recibe la radiografía fresca del fallo del contenedor (sin historial de chat) y devuelve el `Dockerfile`, el `setup_script`, `tools_declared`, el `entrypoint` y `base_ram_mb` corregidos junto a la causa raíz explicada en español. Pensado para aplicarse desde el Modo Visual y el Modo Avanzado del constructor.

- **Método:** `POST`
- **Ruta:** `/api/v1/admin/ai/repair-template`
- **Headers Requeridos:** `X-User-Id`, `X-User-Role: admin`, `X-Tenant-Id`
- **Presupuesto de tiempo:** `90 s` en `fast_cloud`, `120 s` en `queued_or_reasoning` / `local_gpu`.

#### Payload de Entrada
```json
{
  "name": "Laboratorio de Estadística",
  "description": "Python con NumPy",
  "target_environment": "python",
  "docker_image": "python:3.12-slim",
  "dockerfile_content": "FROM python:3.12-slim\nCMD [\"python3\"]",
  "setup_script": "set -euo pipefail",
  "tools_declared": ["python3", "pip"],
  "entrypoint": "python /app/main.py",
  "base_ram_mb": 4096,
  "error_message": "exec format error: gcc: not found",
  "failed_tools": ["gcc"]
}
```
> `failed_tools` lista las herramientas devueltas con `present: false` en la verificación. `error_message` es el texto crudo reportado por el contenedor.

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "docker_image": "solv-local/stack:laboratorio-de-estadistica",
    "dockerfile_content": "FROM python:3.12-slim-bookworm\nRUN apt-get update && apt-get install -y --no-install-recommends gcc\nCMD [\"python3\"]",
    "setup_script": "set -euo pipefail\npip install --no-cache-dir numpy==2.1.0",
    "tools_declared": ["python3", "pip", "gcc"],
    "entrypoint": "python3 /opt/judge/run.py",
    "base_ram_mb": 4096,
    "rationale": "El contenedor falló porque gcc no estaba instalado en la imagen; se agregó la capa de build y se ajustó la memoria.",
    "latency_ms": 5420
  },
  "error": "",
  "message": "Plantilla reparada por el Copiloto IA"
}
```

Guardrails deterministas aplicados sobre la salida del motor:
- `base_image_latest` (`400`) si `docker_image` o cualquier instrucción `FROM` del `Dockerfile` llevan `:latest`.
- Si el `Dockerfile` define capas `RUN`, `docker_image` recibe o conserva el prefijo `solv-local/stack:<slug>` para forzar el build local.
- `ai_response_invalid` (`400`) si falta `rationale` o el JSON del motor es inválido.
- `base_ram_mb` siempre recortado al rango `256..4096`; `tools_declared` depurado (sin vacíos ni duplicados); los campos omitidos conservan el valor actual.

Códigos: `200 OK`; `400 Bad Request` (`invalid_payload`, `base_image_latest`, `ai_response_invalid`); `403 Forbidden` si el rol no es `admin`; `409 Conflict` (`ai_disabled`); `429 Too Many Requests` (`ai_quota_exceeded`); `504 Gateway Timeout` (`ai_timeout`).

---

### 10. API Key Personal en Cascada (BYOK) — `/api/v1/ai/my-settings`

Cascada de credenciales por usuario: la base institucional sigue siendo el motor por defecto y cada docente/administrador puede activar su propia API Key. Las credenciales personales se almacenan en `ai_settings.user_overrides[userID]` con el mismo cifrado AES-256-GCM y las mismas validaciones que la configuración institucional.

| Método y ruta | Uso |
|---------------|-----|
| `GET /api/v1/ai/my-settings` | Lee la configuración personal (nunca el ciphertext) |
| `PUT /api/v1/ai/my-settings` | Guarda o limpia la API Key personal |
| `POST /api/v1/ai/my-settings/test-connection` | Prueba la conexión con la clave personal |
| `POST /api/v1/ai/my-settings/models` | Lista los modelos disponibles con la clave personal |

- **Headers Requeridos:** `X-User-Id`, `X-User-Role: teacher` o `admin`, `X-Tenant-Id`
- **`403 Forbidden`** para `student` en los cuatro endpoints.
- La prueba de conexión acepta razonamiento y fences (ver reglas comunes) y usa `30 s` en `fast_cloud`, `60 s` en `queued_or_reasoning` / `local_gpu`.

#### Payload de `PUT /api/v1/ai/my-settings`
```json
{
  "enabled": true,
  "provider": "openai",
  "model": "gpt-4o-mini",
  "base_url": "",
  "api_key": "sk-personal-00000001",
  "clear_api_key": false
}
```
> `provider` vacío reutiliza el de la base institucional; en proveedores cloud la URL oficial se fuerza y `base_url` se ignora. `clear_api_key: true` elimina la clave guardada.

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "data": {
    "enabled": true,
    "provider": "openai",
    "provider_label": "OpenAI",
    "model": "gpt-4o-mini",
    "base_url": "https://api.openai.com/v1",
    "has_api_key": true,
    "api_key_masked": "sk-p...0001",
    "updated_at": "2026-10-25T14:03:11Z",
    "institutional_enabled": true,
    "institutional_provider_label": "Google Gemini",
    "institutional_model": "gemini-2.5-flash"
  },
  "error": "",
  "message": "Configuración personal de IA actualizada"
}
```

Efecto sobre las invocaciones generativas (`suggest-component`, `generate-exercise`, `repair-template`): si el `userID` tiene un override activo y válido, el motor usa proveedor, modelo, `base_url` y API Key personales **sin descontar cuota institucional** y **aunque `enabled` institucional sea `false`**.

Códigos: `200 OK`; `400 Bad Request` (`invalid_payload`, `provider_invalid`, `base_url_required`, `base_url_blocked`, `model_required`, `api_key_required`, `user_required`); `403 Forbidden` (rol distinto de `teacher`/`admin`); `500 Internal Server Error`.

---

## Motores de Base de Datos Gobernados por Catálogo y Auto-Orquestación Scale-to-Zero (ADR-046)

### 1. Capacidades del Nodo (`GET /api/v1/admin/templates/capabilities`)

Los motores de base de datos se ofrecen desde el catálogo y ya no dependen de que un administrador de sistema declare hosts a mano: con el motor Docker activo, todos los motores soportados figuran disponibles (`is_available: true`) y cada uno declara cómo se aprovisiona (`provisioning_mode`).

| Motor | `provisioning_mode` | `base_ram_mb` | `default_image` |
|-------|---------------------|---------------|-----------------|
| `sqlite` | `embedded` | 0 | — |
| `postgres` | `shared_host` | 50 | — (versión resuelta en vivo con `PostgresVersionProbe`) |
| `mysql` | `managed_container` | 128 | `mysql:8.4` |
| `mariadb` | `managed_container` | 128 | `mariadb:11.4` |
| `mongodb` | `managed_container` | 128 | `mongo:7.0` |
| `redis` | `managed_container` | 64 | `redis:7.2-alpine` |

```json
{
  "data": {
    "satellite_services": [
      {
        "category": "database",
        "engine": "mariadb",
        "label": "MariaDB",
        "version": "11.4",
        "env_var": "DATABASE_URL",
        "is_available": true,
        "base_ram_mb": 128,
        "provisioning_mode": "managed_container",
        "default_image": "mariadb:11.4"
      }
    ]
  }
}
```

El modo de aprovisionamiento es la clave de lectura de la Columna 3 del Constructor: `embedded` no consume RAM, `shared_host` comparte un motor del aula y `managed_container` explica que la memoria se consume solo mientras hay laboratorios activos, porque el motor se apaga solo cuando nadie lo usa.

### 2. Alta y Edición de Motores en el Catálogo (`POST|PUT|PATCH /api/v1/admin/template-components`)

El administrador registra, activa y edita motores con `layer: "database"` igual que cualquier otro componente del catálogo:

- `service_engine` debe pertenecer al dominio soportado (`none`, `sqlite`, `postgres`, `mysql`, `mariadb`, `mongodb`, `redis`); se aceptan los alias `postgresql` y `mongo` y se canonicalizan. En la edición, un motor fuera de esa lista responde `422 Unprocessable Entity`.
- Si el motor no se declara, se infiere de `slug`, `name`, `icon_key` o `base_image` y, cuando no es reconocible, el componente cae en el PostgreSQL de aula.
- Los valores por defecto completados en servidor son `compatible_runtimes: ["universal"]`, `sub_category` (`relational`/`nosql`), `base_image` con tag fijo y `ram_delta_mb` del motor; `install_mode` queda en `none` porque un motor se aprovisiona como servicio y nunca se instala dentro de la imagen del laboratorio.
- La migración `00027_seed_managed_database_engines.sql` deja las seis semillas institucionales activas (`sqlite-3`, `postgres-16`, `mysql-84`, `mariadb-11`, `mongodb-70`, `redis-72`) con su motor, imagen y huella de memoria.

### 3. Efecto en el Arranque del Laboratorio y Apagado Automático

Cuando la plantilla del ejercicio declara un motor `managed_container`, al iniciar (o reactivar) el laboratorio SOLV levanta el contenedor `solv-sat-<engine>` en la red interna `solv-satellites-net` —con volumen nombrado `solv_sat_data_<engine>`, techo de memoria en cgroups y `no-new-privileges`— y conecta el laboratorio del alumno a esa red. El aislamiento no es por contenedor sino por credenciales derivadas por HMAC-SHA256 de alumno y materia (`u_<12hex>`), sobre la base `solv_std_<student>_<subject>`:

| Motor | Variables inyectadas al laboratorio |
|-------|-------------------------------------|
| `mysql`, `mariadb` | `DATABASE_URL=mysql://<user>:<pass>@<host>:3306/<db>`, `MYSQL_HOST`, `MYSQL_PORT`, `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_DATABASE` |
| `mongodb` | `MONGODB_URI=mongodb://<host>:27017/<db>`, `MONGO_DATABASE` |
| `redis` | `REDIS_URL=redis://<host>:6379/0`, `REDIS_HOST`, `REDIS_PORT`, `REDIS_KEY_PREFIX=<db>:` |

El `db_init_script` del ejercicio se aplica una sola vez por `exercise_id`, sellado en `_solv_exercise_migrations` dentro de la base del alumno. Si el motor no puede levantarse, el inicio del laboratorio falla y no queda registrado como activo.

**Scale-to-Zero:** al hibernar, pausar o terminar un laboratorio —y en cada ciclo del worker de QoS— SOLV verifica si queda algún laboratorio en estado `running` que use cada motor gestionado. Cuando el conteo llega a `0`, el contenedor `solv-sat-<engine>` se detiene y libera toda su memoria del host; el contenedor no se elimina y su volumen de datos sobrevive, de modo que la próxima sesión lo reactiva sin reconstruir nada.

**Override de destino:** `MYSQL_SATELLITE_HOST`, `MARIADB_SATELLITE_HOST`, `MONGODB_SATELLITE_HOST` y `REDIS_SATELLITE_HOST` dejan de ser un requisito de disponibilidad y pasan a ser un override opcional para apuntar a un motor externo ya provisionado. Con un override presente, SOLV no levanta ni apaga contenedores para ese motor.

---

## Slice 17 — Panel de Salud y Recursos (Telemetría Verdadera)

### 1. Estado de Salud y Recursos del Host (`GET /api/v1/admin/metrics/health`)

Devuelve el estado del nodo y del tenant desde **hechos medidos** (ADR-049): telemetría del host vía el puerto `domain.HostTelemetry` (gopsutil + daemon Docker), conteos por estado del repositorio (`GROUP BY status`), capacidad calculada con `min(policy, RAM disponible)` e historial persistido en `host_metrics_samples`.

- **Método:** `GET`
- **Ruta:** `/api/v1/admin/metrics/health`
- **Query opcional:** `?window=60m|24h` (ventana del historial; default `60m`, tope = retención de 7 días; también acepta sufijo de días, p. ej. `7d`).
- **Headers requeridos:** `X-User-Id` / sesión admin, `X-Tenant-Id` (opcional, fallback por defecto).

#### Ejemplo de Respuesta (`200 OK`)
```json
{
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "running_labs": 3,
  "hibernated_labs": 1,
  "oom_killed_labs": 1,
  "unknown_labs": 0,
  "orphaned_labs": 1,
  "total_ram_alloc_mb": 1536,
  "health_status": "critical",
  "health_reasons": ["orphaned_workspaces", "oom_killed_workspaces"],
  "uptime_seconds": 123456,
  "host_hardware": {
    "ram_used_bytes": 6227705856,
    "ram_total_bytes": 8159223808,
    "ram_percent": 76.3,
    "cpu_cores": 8,
    "cpu_percent": 14.2,
    "disk_used_bytes": 197000000000,
    "disk_total_bytes": 549000000000,
    "disk_percent": 36,
    "disk_path": "/var/lib/docker"
  },
  "docker": {
    "server_version": "27.1.1",
    "storage_driver": "overlay2",
    "healthy": true
  },
  "load_history": [
    {
      "timestamp": "2026-10-27T17:12:00-04:00",
      "ram_percent": 76.1,
      "ram_used_gb": 5.8,
      "cpu_percent": 14.0,
      "active_containers": 3
    }
  ],
  "host_safety_ceiling_pct": 85,
  "estimated_max_containers": 7,
  "capacity_basis": "mem_available",
  "max_containers_policy": 40,
  "ram_per_lab_mb": 512
}
```

#### Semántica de campos

| Campo | Significado |
|-------|-------------|
| `running_labs` | Conteo real de `workspaces.status = 'running'` del tenant. |
| `hibernated_labs`, `oom_killed_labs`, `unknown_labs`, `orphaned_labs` | Desglose por estado del agregado `CountWorkspacesByStatus` (nunca un filtro de una lista de running). |
| `health_status` | `operational \| degraded \| critical`. `critical` si `docker.healthy == false` o `orphaned_labs > 0`; `degraded` si `oom_killed_labs > 0`, `unknown_labs > 0` o RAM sobre el techo de seguridad. |
| `health_reasons` | Códigos estables: `docker_unreachable`, `orphaned_workspaces`, `oom_killed_workspaces`, `unknown_workspaces`, `ram_above_safety_ceiling`. |
| `host_hardware` | Lectura medida del nodo (RAM, CPU, disco con su ruta real `disk_path`). Ausente si la telemetría no está inyectada. |
| `docker` | `server_version`, `storage_driver` y `healthy` del daemon real. Ausente si la telemetría no está inyectada. |
| `load_history` | Últimos samples reales persistidos por el worker (1 fila/min, retención 7 días). `timestamp` es RFC3339 del momento de la medición. **Nunca** se siembra historia sintética. |
| `estimated_max_containers` | `min(max_containers_policy, floor(availableMB / ram_per_lab_mb))`, nunca menor que `running_labs`. |
| `capacity_basis` | `"policy"` (mandó `max_containers`, o no se pudo medir RAM) \| `"mem_available"` (mandó la RAM disponible). |
| `host_safety_ceiling_pct` | Techo de seguridad de la política vigente del tenant (75–90). |

#### Estados de workspace nuevos (ADR-049)

| Estado | Origen | Efecto |
|--------|--------|--------|
| `unknown` | El daemon de Docker no respondió durante la reconciliación (ciclo 30 s). | No destruye nada; retiene cupo en `GetActiveWorkspaces`; alerta de auditoría si persiste > 5 ciclos; reintenta en el siguiente ciclo. |
| `orphaned` | El daemon respondió y el contenedor del registro ya no existe. | Libera cupo; el alumno lo recrea con restart; transición auditada. |

`GET /api/v1/admin/dashboard/containers` expone ambos estados en `status` del DTO (ya no colapsan a `hibernated`).

---

## Revisión Docente y Libreta de Notas (ADR-053)

El trabajo manual del docente deja de deducirse del veredicto del juez. `submissions.review_status` declara el estado (`not_required | pending | in_review | graded | returned`) y `exercise_grades` guarda qué entrega cuenta para la nota de cada alumno en cada práctica.

### `GET /api/v1/teacher/reviews`

Cola real de corrección agrupada por práctica. Parámetro opcional `subject_id`.

```json
{
  "data": [
    {
      "exercise_id": "uuid",
      "exercise_title": "Lista enlazada",
      "subject_id": "uuid",
      "subject_name": "Programación I",
      "environment_type": "IDE_PERSISTENTE",
      "pending": 7,
      "total": 12,
      "oldest_submitted_at": "2026-10-28T10:00:00Z"
    }
  ]
}
```

`pending` cuenta `review_status IN ('pending','in_review')`; `total` es el número de inscritos en la materia.

### `PATCH /api/v1/teacher/submissions/{id}/review`

```json
{ "status": "graded", "feedback_summary": "Buen trabajo", "score": 85 }
```

- Transiciones válidas: `pending → in_review | graded`, `in_review → pending | graded`, `graded → in_review | returned`, `returned → in_review | graded`. Reintentar el mismo estado es idempotente.
- Al pasar a `graded` o `returned` se actualiza la celda de nota y se registra auditoría.
- Respuesta: `{ "submission_id", "review_status", "cell": { "selected_submission_id", "score", "source", "reason" } | null }`.

### `PUT /api/v1/teacher/grades/{studentId}/exercises/{exerciseId}`

```json
{ "selected_submission_id": "uuid", "score": 90, "reason": "Corrigió en clase y volvió a entregar" }
```

- Fija qué entrega cuenta para la nota. La nota es opcional: sin `score` se usa la de la entrega elegida.
- Cambiar la nota respecto de la entrega exige `reason` de al menos 10 caracteres.
- Respuesta: la celda resultante.

### Errores

| Código | Cuándo |
| --- | --- |
| `409` | Transición de revisión no permitida o entrega modificada por otro docente. |
| `422` | Estado desconocido, nota fuera de 0–100 o falta el motivo del cambio de nota. |
| `400` | La entrega elegida no pertenece a ese alumno y esa práctica. |
| `403` | Rol estudiante o materia ajena al docente. |
| `503` | El almacén de revisión no está configurado. |

### Efectos en contratos existentes

- `GET /api/v1/teacher/courses/{id}/grades/matrix`: `grades[exercise_id]` puede ser `null` (sin corregir). Cada fila añade `graded_count` y `pending_count`; `average` se calcula solo sobre prácticas calificadas.
- `GET /api/v1/teacher/courses/{id}/grades/export`: la celda sin corregir se exporta vacía.
- `GET /api/v1/teacher/courses/{id}/submissions`: cada entrega incluye `review_status`.
- El resumen de materias (`pending_review`) ahora cuenta el estado de revisión, no los veredictos.


## Avances del docente en el aula (ADR-058)

Durante el aula, el docente congela la clase cuando empieza un examen y decide cuánto tiempo más queda disponible un avance después de cerrar la sesión. Los tres endpoints siguientes responden con la sesión resultante.

### `PATCH /api/v1/teacher/sessions/{id}/freeze`

Sin cuerpo. Devuelve `200` con la sesión y `frozen_at` con la hora de la congelación. Congelar dos veces conserva la hora de la primera; congela la emisión de avances nuevos, no el editor del alumno.

```json
{ "data": { "id": "uuid", "status": "open", "frozen_at": "2026-10-31T15:04:05Z" }, "error": "", "message": "Clase congelada" }
```

### `PATCH /api/v1/teacher/sessions/{id}/unfreeze`

Sin cuerpo. Devuelve `200` con la sesión y `frozen_at` en `null`.

### `PATCH /api/v1/teacher/sessions/{id}/grace`

Ajusta la ventana de disponibilidad de un aula cerrada. El cuerpo lleva exactamente una acción:

```json
{ "add_minutes": 20 }
```

```json
{ "until": "2026-11-02T23:00:00Z" }
```

```json
{ "close": true }
```

- Al cerrar el aula —a mano o al vencer `ends_at`— el servidor fija `grace_until = cierre + 30 minutos`. No hay que pulsar nada.
- `add_minutes` va de 1 a 480 y suma al final vigente. Si la ventana ya venció, cuenta desde ahora.
- `until` es una fecha futura que reemplaza el final vigente. El máximo es de 7 días desde el cierre de la clase.
- `close` fija el final en el momento.
- Solo aplica con el aula cerrada. Con la sesión abierta responde `409`.
- Al reabrir la ventana, los destinatarios `expired` vuelven a `pending`; al cerrarla, los `pending` pasan a `expired`. `applied`, `dismissed` y `failed` no se revierten.
- Devuelve `200` con la sesión y su `grace_until`.

```json
{ "data": { "id": "uuid", "status": "closed", "grace_until": "2026-10-31T16:30:00Z" }, "error": "", "message": "Ventana de disponibilidad actualizada" }
```

### Errores

| Código | Cuándo |
| --- | --- |
| `403` | Rol estudiante en cualquiera de las tres rutas. |
| `404` | La sesión no existe, es de otra institución o pertenece a otro docente. |
| `409` | La ventana se intenta ajustar con el aula abierta. |
| `400` | Cuerpo JSON inválido, más de una acción o `add_minutes` fuera de 1–480. |
| `422` | La fecha límite no es futura o la ventana supera los 7 días desde el cierre. |

### Rutas relacionadas

`POST /api/v1/teacher/sessions/{id}/advances` emite el avance, y `/api/v1/class-advances/pending`, `/{id}/files`, `/{id}/apply` y `/{id}/dismiss` son el lado del alumno. Su contrato está en [ADR-058](ARQUITECTURA/ADR/ADR-058-avance-voluntario-del-docente-en-aula.md) §10.

### `GET /api/v1/teacher/sessions/{id}/advances`

Emisiones del aula con el estado de cada destinatario: quién aplicó, quién descartó y quién sigue pendiente.

```json
{ "data": [ { "id": "uuid", "exercise_id": "uuid", "exercise_title": "Recorridos", "scope": "class", "file_count": 2, "created_at": "2026-10-31T15:30:00Z", "counts": { "total": 4, "applied": 1, "dismissed": 1, "pending": 1, "failed": 0, "expired": 1 }, "recipients": [ { "student_id": "uuid", "name": "Ana Pérez", "email": "ana@solv.test", "status": "applied", "decided_at": "2026-10-31T15:32:00Z" } ] } ], "error": "", "message": "Avances del aula obtenidos" }
```

- Del más nuevo al más viejo; dentro de cada emisión, los destinatarios van por nombre.
- `status` es `pending`, `applied`, `dismissed`, `failed` o `expired`. `decided_at` es `null` mientras el aviso siga pendiente.
- `403` para el rol estudiante; `404` si el aula no existe o pertenece a otro docente.

### `GET /api/v1/teacher/sessions/{id}/history`

Historial de la clase: emisiones, congelaciones, cambios de ventana y decisiones de los alumnos, del más nuevo al más viejo. El alumno no lo ve.

```json
{ "data": [ { "id": "uuid", "action": "CLASS_ADVANCE_GRACE_EXTENDED", "actor_id": "uuid", "actor_name": "Prof. García", "actor_email": "garcia@solv.test", "actor_role": "teacher", "created_at": "2026-10-31T15:42:00Z", "advance_id": "uuid", "exercise_id": "uuid", "minutes": 20, "previous_until": "2026-10-31T16:30:00Z", "until": "2026-10-31T16:50:00Z" } ], "error": "", "message": "Historial de la clase obtenido" }
```

- `action` es una de `CLASS_ADVANCE_EMITTED`, `CLASS_FROZEN`, `CLASS_UNFROZEN`, `CLASS_ADVANCE_GRACE_EXTENDED`, `CLASS_ADVANCE_GRACE_DEADLINE_SET`, `CLASS_ADVANCE_GRACE_CLOSED`, `CLASS_ADVANCE_APPLIED` o `CLASS_ADVANCE_DISMISSED`.
- `actor_name`, `actor_email` y `actor_role` vienen resueltos de la misma consulta. En emitir, aplicar y descartar se agrega el `advance_id`; en emitir, `recipients` y `file_count`; en aplicar, `files_written`; en los cambios de ventana, `minutes`, `previous_until` y `until`.
- El historial no guarda el código del avance, solo sus datos y los cambios de la ventana. Devuelve hasta 200 movimientos; el registro completo queda en `/api/v1/admin/audit-logs`.
- `403` para el rol estudiante; `404` si el aula no existe o pertenece a otro docente. El administrador puede leer cualquier aula de su institución.
- La pantalla refresca la lectura al abrir la clase, al emitir, al congelar o descongelar, al ajustar la ventana y con `[Actualizar]`.

### `GET /api/v1/class-advances/{id}/files`

Lectura previa del aviso: qué archivos se agregan y cuáles se reemplazan en el laboratorio del alumno. No escribe nada.

```json
{ "data": { "advance_id": "uuid", "exercise_id": "uuid", "file_count": 2, "added": ["utils.py"], "overwritten": ["main.py"] }, "error": "", "message": "Archivos del avance obtenidos" }
```

- `404` si el avance no existe o no está dirigido al alumno autenticado; un avance ajeno no se revela.
- `409` si ya fue aplicado o descartado, si el ejercicio es de examen o si la ventana de rezago venció (el destinatario pasa a `expired`).
- `503` si el backend arrancó sin el almacén de snapshots: el aviso sigue visible, pero no se puede leer su contenido.

`GET /api/v1/class-advances/pending` solo devuelve avisos que todavía se pueden aplicar: los vencidos se cierran como `expired` en esa misma consulta y dejan de mostrarse.

El historial de la clase es de solo lectura: no se edita ni se borra desde la pantalla. El administrador conserva el registro completo en `/api/v1/admin/audit-logs`.

---

## Congelamiento del examen (ADR-005 §5)

Al terminar un examen, el volumen de cada alumno se monta en solo lectura y el laboratorio pasa a `frozen` con `frozen_at`. El bloqueo lo aplica el sistema de archivos: no se recupera recargando la página ni pidiendo reanudar.

Las respuestas de sesión (`GET /teacher/sessions/open`, `POST /teacher/sessions`, `GET /teacher/sessions/{id}/board` y `PATCH /close`) incluyen `purpose`, que vale `exam` cuando la práctica de la sesión es un examen y `class` en el resto de los casos, incluida un aula sin práctica. La pantalla del docente usa ese campo para mostrar los controles del examen.

**Aula y laboratorio: 400 no es 404.** Los endpoints del aula y del entorno distinguen dos casos. Un identificador con formato inválido responde `400` con `identificador con formato inválido`: en el cuerpo de `POST /teacher/sessions` (`subject_id`, `exercise_id`) y en el `subject_id` de consulta de `GET /api/v1/workspaces` y del entorno por práctica. Un aula o una materia que no existe responde `404`: el id del aula no corresponde a ninguna fila, o el almacén no encuentra la materia del entorno. Así el panel no confunde "no existe" con "el servidor falló" ni con "la petición está mal armada".

### `GET /api/v1/teacher/sessions/last-closed`

Devuelve la última aula cerrada del docente, con su `grace_until` y `purpose`, para que al recargar la página el aula muestre su historial y su ventana de disponibilidad. Si no hay ninguna aula cerrada, `data` es `null`. Solo lo usa el rol `teacher` con el alcance de sus materias; el alumno recibe 403.

### `POST /api/v1/teacher/sessions/{id}/exam-freeze`

Sin cuerpo. Repite el cierre del examen sobre los participantes y devuelve el conteo real:

```json
{ "data": { "frozen": 28, "skipped": 5, "failed": 2 }, "error": "", "message": "Examen congelado" }
```

- `frozen`: volúmenes que quedaron en solo lectura en esta llamada.
- `skipped`: ya estaban congelados; es idempotente.
- `failed`: no se pudieron reconstruir. El barrido de 15 minutos los reintenta.
- `409` si el aula no es un examen o si sigue abierta.
- `403` para el rol estudiante; `404` si el aula no existe o pertenece a otro docente.

### `POST /api/v1/teacher/sessions/{id}/students/{studentID}/unfreeze`

Excepción individual: devuelve la escritura al laboratorio de un alumno de ese examen.

```json
{ "reason": "Se le cortó la luz a mitad del examen" }
```

- `reason` es obligatorio y queda en `audit_logs` con el docente, el alumno y el aula.
- Responde `200` con el mismo conteo.
- `400` sin motivo o con un identificador de ruta mal formado; `404` si el alumno no participó de esa aula o si su laboratorio ya no existe.

### Entrega anticipada: `POST /api/v1/workspaces/{id}/submit`

La entrega del proyecto de un examen cierra solo el volumen de ese alumno (ADR-005 §5.2). El cuerpo sigue siendo el de la entrega del IDE:

```json
{ "exercise_id": "uuid" }
```

- `files` es opcional y queda para herramientas que necesiten escribir archivos en el volumen justo antes de la entrega. El solver del alumno no lo envía: entrega el volumen de la práctica tal como está en el IDE. Si se manda, esos archivos se escriben primero y reemplazan los que tengan el mismo nombre.
- Si el ejercicio de la entrega tiene `purpose = exam`, después de registrar la entrega el servidor monta el volumen de ese alumno en solo lectura. El resto de la clase sigue escribiendo.
- Una entrega posterior sobre un volumen congelado responde `409` con `El examen terminó: tu entorno quedó en solo lectura`: escribir por esta vía sería seguir editando después del cierre.
- `404` si el entorno no existe, y `403` si pertenece a otro alumno. Un entorno ausente no es un fallo del servidor: la entrega responde lo mismo que cualquier otro recurso que no está.
- La respuesta de la entrega no cambia; el cierre se ve en el laboratorio (`status: "frozen"` y `frozen_at`).

### `PATCH /api/v1/teacher/sessions/{id}/close`

El cierre del aula conserva su contrato y suma `freeze_report` cuando el aula era un examen:

```json
{ "data": { "id": "uuid", "status": "closed", "ends_at": "2026-10-31T16:00:00Z", "freeze_report": { "frozen": 28, "skipped": 5, "failed": 2 } }, "error": "", "message": "Aula cerrada" }
```

Un aula de práctica no congela volúmenes y responde sin `freeze_report`.

- `404` si el aula no existe o pertenece a otro docente. Un id que no es UUID tampoco identifica ningún aula y responde el mismo `404`: la diferencia con el `400` de validación está en lo que la petición declara, no en el formato del camino.

### Estado del laboratorio

`GET /api/v1/workspaces?subject_id={uuid}` y el dashboard del alumno incluyen `status: "frozen"` y `frozen_at`. Al intentar `POST /api/v1/workspaces/start` o `POST /api/v1/workspaces/{id}/restart` con el volumen congelado, la respuesta es `409` con `El examen terminó: tu entorno quedó en solo lectura`.

**Entorno por práctica del alumno.** `POST /api/v1/workspaces/start` acepta `exercise_id`. Con práctica, el entorno y el volumen se acotan a `(alumno, materia, práctica)` y la respuesta incluye `exercise_id`; abrir otra práctica no reutiliza el entorno de la materia ni escribe en su volumen. Reabrir la misma práctica devuelve el mismo entorno. Sin `exercise_id` se conserva el entorno por materia. La emisión y la aplicación de avances (ADR-058) usan el volumen de la práctica cuando existe.

---

## Entorno de clase del docente

El avance que se emite desde el aula sale del entorno del docente de la práctica de la sesión (ADR-058 §5, ADR-059). Ese entorno se abre desde la ficha de la práctica (`POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/ide`); la pantalla `/teacher/entorno` se retiró y su ruta redirige a Hoy. La lectura por materia que describe este apartado sigue disponible para consultar el entorno vigente sin levantar contenedores.

### `GET /api/v1/workspaces?subject_id={uuid}`

- Sin cuerpo. La identidad se resuelve desde la sesión (`X-User-Id` o el token): el parámetro elige la materia, nunca a la persona.
- Responde `200` con el entorno vigente en esa materia, o con `data: null` si el usuario todavía no abrió uno. Un entorno leído no levanta contenedores.
- `400` si falta `subject_id` o no es un UUID.
- `404` si el almacén no encuentra la materia indicada.
- `401` si no se puede resolver la identidad.

```json
{
  "data": {
    "workspace_id": "uuid",
    "student_id": "uuid",
    "subject_id": "uuid",
    "status": "running",
    "access_url": "https://<workspace-id>.<dominio-base>",
    "memory_limit_mb": 512,
    "last_heartbeat_at": "2026-10-31T15:04:05Z"
  },
  "error": "",
  "message": "Entorno de la materia obtenido"
}
```

Sin entorno en la materia:

```json
{ "data": null, "error": "", "message": "No hay entorno en esta materia" }
```

### Rutas que usa la pantalla para operar el entorno

| Método | Ruta | Efecto |
| --- | --- | --- |
| `POST` | `/api/v1/workspaces/start` | Abre el entorno de la materia con `{ "subject_id": "uuid" }`. Con uno corriendo devuelve el mismo; con uno en pausa lo reanuda. |
| `POST` | `/api/v1/workspaces/{id}/pause` | Lo deja en `hibernated` sin perder el volumen: el código del docente queda guardado. |

El entorno vive sobre el volumen `solv_workspace_<usuario>_<materia>`, que es exactamente el que lee la emisión de avances. Por eso lo que el docente escribe en esta pantalla es lo que se copia al aula, y sin entorno abierto la emisión responde `409`.

---

## Auditoría de similitud por práctica (ADR-062)

El reporte dejó de comparar el código completo: el motor resta `exercises.boilerplate` antes de generar los k-gramas. Dos entregas que solo comparten la base dan 0% y no aparecen. La pantalla vive en la ficha de la práctica; `/teacher/anti-plagio` redirige a `/teacher/por-calificar`.

### `GET /api/v1/teacher/courses/{id}/plagiarism?exercise_id={uuid}`

- Mismos headers y permisos que el resto del panel docente: un estudiante recibe `403`.
- `exercise_id` es opcional. La ficha siempre lo envía, así que el reporte queda acotado a esa práctica.
- Sin `exercise_id`, el reporte conserva el alcance por materia.

```json
{
  "data": {
    "subject_id": "uuid",
    "exercise_id": "uuid",
    "exercise_title": "Árboles Binarios",
    "analyzed_at": "2026-11-02T11:00:00Z",
    "total_submissions": 20,
    "suspect_pairs_count": 1,
    "matches": [
      {
        "submission_id_a": "uuid",
        "student_id_a": "uuid",
        "student_name_a": "Ana García",
        "submission_id_b": "uuid",
        "student_id_b": "uuid",
        "student_name_b": "Carlos Gómez",
        "exercise_id": "uuid",
        "exercise_title": "Árboles Binarios",
        "similarity": 92,
        "risk_level": "critical",
        "matching_tokens": 140,
        "total_tokens_a": 150,
        "total_tokens_b": 155,
        "ranges": [
          {
            "source_a": { "file_path": "src/tree.py", "start_line": 10, "end_line": 24 },
            "source_b": { "file_path": "src/tree.py", "start_line": 12, "end_line": 26 },
            "token_count": 140,
            "exact_match": true
          }
        ]
      }
    ]
  },
  "error": "",
  "message": "Análisis de similitud estructural AST completado exitosamente"
}
```

- `ranges` reemplaza a `common_structures`. Cada par trae como máximo 10 tramos, en el orden del fuente y sin solaparse.
- `file_path` solo aparece en modalidad IDE: las líneas son locales a ese archivo. En el evaluador automático el campo se omite y las líneas son las del envío completo.
- `token_count` cuenta los tokens del tramo después de restar la base de la práctica.
- Los totales `total_tokens_a` y `total_tokens_b` también son posteriores a la sustracción.

### `GET /api/v1/teacher/plagiarism/pairs/{submission_a}/{submission_b}`

- El docente solo accede a entregas de sus materias; cualquier entrega ajena responde `403` sin
  distinguirla de una inexistente. El rol `admin` consulta sin filtro de docente.
- Las dos entregas deben pertenecer a la misma práctica. Si no, `422`. El mismo identificador dos veces
  es `400`.
- No devuelve `test_cases`, rúbrica, comentarios ni eventos de teclado. Las métricas son totales.

```json
{
  "data": {
    "exercise": {
      "id": "uuid",
      "title": "Árboles Binarios",
      "subject_id": "uuid",
      "subject_name": "Estructuras de Datos",
      "environment_type": "IDE_PERSISTENTE",
      "expected_complexity": "O(log n)",
      "base_files": [
        { "path": "src/node.py", "language": "python", "content": "...", "start_line": 1, "line_count": 12 }
      ],
      "base_files_available": true
    },
    "a": {
      "submission_id": "uuid",
      "student_id": "uuid",
      "student_name": "Ana García",
      "student_email": "ana@solv.test",
      "verdict": "AC",
      "language": "python",
      "complexity": "O(log n)",
      "execution_time_ms": 120,
      "memory_used_mb": 18,
      "files": [
        { "path": "src/tree.py", "language": "python", "content": "...", "start_line": 1, "line_count": 40 }
      ],
      "files_available": true,
      "metrics": {
        "total_time_ms": 65000,
        "total_keystrokes": 42,
        "paste_events_count": 1,
        "paste_percentage": 12.5,
        "events_available": true
      }
    },
    "b": { "submission_id": "uuid", "verdict": "WA", "code": "..." },
    "similarity": 92,
    "risk_level": "critical",
    "ranges": []
  },
  "error": "",
  "message": "Par de entregas listo"
}
```

- En IDE cada lado trae `files` y se omite `code`; en el evaluador cada lado trae `code` y se omite
  `files`. `base_files` solo existe en IDE.
- `metrics` no incluye eventos: tiempo activo, keystrokes, eventos de pegado y porcentaje. Sin
  telemetría, `events_available` es `false` y los totales quedan en cero.
- Los `ranges` son los del par, calculados de nuevo con el código base restado.

### Rutas del cliente

| Ruta | Efecto |
| --- | --- |
| `/teacher/cursos/:courseId/practicas/:exerciseId/similitud` | Auditoría de esa práctica en tres columnas, sin selector de materia |
| `/teacher/anti-plagio` | Redirección a `/teacher/por-calificar`; no conserva el listado global |
