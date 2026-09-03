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

### 6. Widget de Atención Prioritaria (`GET /api/v1/teacher/attention`)

Provee alertas clasificadas por severidad (`critical`, `warning`, `standard`) siguiendo el patrón *Exception-Based Reporting*.

- **Método:** `GET`
- **Ruta:** `/api/v1/teacher/attention`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Código de Respuesta:** `200 OK`

#### Ejemplo de Respuesta
```json
{
  "data": {
    "critical": [
      {
        "type": "oom_killed",
        "student_id": "77777777-7777-4777-a777-777777777777",
        "student_name": "Carlos Ruiz",
        "workspace_id": "99999999-9999-4999-a999-999999999999",
        "subject_id": "88888888-8888-4888-a888-888888888888",
        "occurred_at": "2026-09-02T10:15:00Z"
      }
    ],
    "warning": [
      {
        "type": "ast_blocked",
        "student_id": "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa",
        "student_name": "Ana Torres",
        "exercise_id": "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb",
        "rule_violated": "no_import_os",
        "occurred_at": "2026-09-02T09:30:00Z"
      }
    ],
    "standard": [
      {
        "type": "pending_review",
        "submission_id": "cccccccc-cccc-4ccc-cccc-cccccccccccc",
        "student_name": "María López",
        "exercise_title": "Lab #01: Árboles AVL",
        "submitted_at": "2026-09-02T08:45:00Z"
      }
    ]
  },
  "error": "",
  "message": "Alertas de atencion obtenidas exitosamente"
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
    "comments": [
      {
        "id": "c-uuid-1",
        "line_number": 15,
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

---

### 10. Override Manual de Calificación (`POST /api/v1/submissions/{id}/override`)

Permite a un docente o administrador rectificar o convalidar el veredicto y nota de una entrega con justificación obligatoria ($\ge 10$ caracteres).

- **Método:** `POST`
- **Ruta:** `/api/v1/submissions/{id}/override`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`: Calificación actualizada.
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

---

### 11. Comentarios In-line Anclados a Código (`POST & GET /api/v1/teacher/submissions/{id}/comments`)

Permite registrar y consultar comentarios pedagógicos anclados a líneas específicas del código fuente.

- **Método:** `POST` / `GET`
- **Ruta:** `/api/v1/teacher/submissions/{id}/comments`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `POST`: `201 Created`
  - `GET`: `200 OK` con lista de comentarios ordenados por `line_number`.

#### Ejemplo de Solicitud POST
```json
{
  "line_number": 42,
  "comment": "Cuidado con la condición de corte en el caso base"
}
```

---

### 12. Runner Efímero Docente (`POST /api/v1/teacher/submissions/{id}/run-ephemeral`)

Ejecuta código en el sandbox efímero sin persistir una nueva entrega en la base de datos (para pruebas en vivo durante la corrección).

- **Método:** `POST`
- **Ruta:** `/api/v1/teacher/submissions/{id}/run-ephemeral`
- **Headers:** `X-User-Id: <teacher_uuid>`, `X-User-Role: teacher | admin`, `X-Tenant-Id: <tenant_uuid>`
- **Respuestas:**
  - `200 OK`: Veredicto y salida de ejecución en memoria.
  - `403 Forbidden`: Si el rol es student.

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

### 13. Exportación de Calificaciones en CSV (`GET /api/v1/teacher/courses/{id}/grades/export`)

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

## Contratos Aprobados Pendientes de Implementar

### SLICE_14: Panel Administrador Institucional (Restante)

| Endpoint | Método | ADR Vinculado | Descripción de Alto Nivel | Estado |
| :--- | :---: | :---: | :--- | :---: |
| `/api/v1/admin/templates/{id}/review` | `PUT` | ADR-030 | Aprobación o rechazo de plantillas Docker solicitadas | *Pendiente* |
| `/api/v1/admin/emergency/{action}` | `POST` | ADR-032 | Ejecutar acción de emergencia con frase de confirmación tipada | *Pendiente* |

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
