# Especificación de Contratos de API REST (OpenAPI 3.0 Specification) — SOLV BaaS

El presente documento establece la especificación formal del contrato de interfaz entre el cliente (Frontend / Angular) y el servidor (Backend / Go) para el sistema de orquestación de laboratorios **SOLV**.

---

## 1. Autenticación y Perímetro (`/auth`, `/api/v1/auth`, `/api/v1/users`)

### 1.1 `GET /auth/google/login`
* **Propósito:** Redirección al consentimiento de Google OAuth2 SSO.
* **Respuesta:** `302 Found` a Google OAuth2.

### 1.2 `GET /auth/google/callback`
* **Propósito:** Intercambio de código OAuth2, aprovisionamiento de usuario e inyección de sesión.
* **Respuestas:**
  * `200 OK`: Retorna payload JSON `{"token": "<jwt>"}` y setea la cookie HttpOnly `solv_session` (`Domain: COOKIE_DOMAIN`, `SameSite: Lax`).
  * `401 Unauthorized`: Email no institucional (`@uab.edu.bo`).

### 1.3 `GET /api/v1/auth/verify` (ForwardAuth Traefik v3)
* **Propósito:** Endpoint perimetral de ultra-baja latencia (<50ms SLA) consultado por Traefik v3 para validar la cookie HttpOnly `solv_session` en solicitudes de `iframes` cross-subdomain.
* **Respuestas:**
  * `200 OK`: Cookie o Bearer válida. Cabeceras inyectadas: `X-User-Id`, `X-User-Role`.
  * `401 Unauthorized`: Sin cookie o token inválido.
  * `403 Forbidden`: Token JWT expirado.

### 1.4 `POST /api/v1/auth/logout`
* **Propósito:** Cierre de sesión y expiración limpia de la cookie HttpOnly (`MaxAge: -1`).
* **Respuesta:** `200 OK` (`{"status": "logged_out"}`).

### 1.5 `GET /api/v1/config/public`
* **Propósito:** Exposición de configuración pública no sensible (dominio base, versión de API, banderas SSO).
* **Respuesta:** `200 OK` (`{"cookie_domain": ".solv.uab.edu.bo", "version": "v0.8.0"}`).

---

## 2. Workspaces y Orquestación Unificada (`/api/v1/workspaces`)

### 2.1 `POST /api/v1/workspaces/start`
* **Propósito:** Instanciar un entorno interactivo (`type: "IDE_PERSISTENTE"`) o efímero de evaluación (`type: "JUEZ_EFIMERO"`).
* **Cabeceras:** `Authorization: Bearer <token_jwt>`, `X-Tenant-ID: <uuid>` (opcional).
* **Cuerpo:**
```json
{
  "student_id": "a1b2c3d4-e5f6-7890-abcd-1234567890ab",
  "subject_id": "c9d8e7f6-a5b4-3210-fedc-0987654321ba",
  "type": "IDE_PERSISTENTE"
}
```
* **Cabeceras de Respuesta (RFC 6585):**
  * `X-RateLimit-Limit: 5`
  * `X-RateLimit-Remaining: <N>`
  * `X-RateLimit-Reset: <timestamp_unix>`
* **Respuesta 200 OK:**
```json
{
  "id": "bd430809-b45f-4be0-95a4-472da081ec11",
  "student_id": "a1b2c3d4-e5f6-7890-abcd-1234567890ab",
  "subject_id": "c9d8e7f6-a5b4-3210-fedc-0987654321ba",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "type": "IDE_PERSISTENTE",
  "status": "running",
  "access_url": "http://bd430809-b45f-4be0-95a4-472da081ec11.solv.uab.edu.bo",
  "memory_limit_mb": 256
}
```
* **Respuesta 429 Too Many Requests (Exceso de Cuota):**
  * `Retry-After: 60`
  * `Content-Type: application/json`
```json
{
  "error": "Rate limit exceeded. Please wait before starting another workspace."
}
```

### 2.2 `DELETE /api/v1/workspaces/{id}`
* **Propósito:** Finalizar el workspace, recálculo EWMA y escaneo AST Semgrep en solo lectura (`:ro`).

### 2.3 `GET /api/v1/workspaces/{id}/audit`
* **Propósito:** Obtener la auditoría semántica AST generada por `SemgrepWorker` (`semgrep_audit JSONB`).

---

## 3. Esquema Académico (`/api/v1/subjects`, `/api/v1/submissions`, `/api/v1/invitations`, `/api/v1/classroom`)

### 3.1 `POST /api/v1/subjects` & `GET /api/v1/subjects`
* **Propósito:** Creación y listado de materias filtradas por tenant.

### 3.2 `POST /api/v1/subjects/{id}/enroll` & `GET /api/v1/subjects/{id}/students`
* **Propósito:** Inscripción de alumnos y listado de alumnos inscritos en una materia.

### 3.3 `POST /api/v1/submissions`
* **Propósito:** Registrar un intento/solución enviado al Juez Virtual.

### 3.4 `GET /api/v1/exercises/{id}/submissions`
* **Propósito:** Consultar historial de entregas de un ejercicio.
* **Regla de Filtrado por Rol:**
  * Rol `student`: Filtra únicamente las entregas del usuario solicitante (`student_id = X-User-Id`).
  * Rol `teacher` / `admin`: Retorna todas las entregas del ejercicio dentro del `tenant_id`.

### 3.5 `POST /api/v1/invitations/teachers` & `POST /api/v1/invitations/teachers/accept`
* **Propósito:** Emisión y aceptación de invitaciones a docentes.
* **Regla Transaccional:** La aceptación valida que el email del usuario en sesión (`X-User-Email`) coincida con el email de la invitación y actualiza el rol a `teacher` con `used = TRUE` en una transacción atómica.

### 3.6 `GET /api/v1/classroom/import`
* **Propósito:** Importación manual unidireccional de nóminas desde Google Classroom (D6 compliant).

### 3.7 `GET /api/v1/exercises/{id}`
* **Propósito:** Obtener la información técnica de un ejercicio para el Juez Virtual.
* **Regla de Seguridad por Rol (CRIT-03):**
  * Rol `student`: Retorna el **DTO público** (`ExercisePublicResponse`) enmascarando los `test_cases` del Juez Virtual, `reference_solution`, `validation_query` y `expected_json` para prevenir la lectura de respuestas desde DevTools del navegador.
  * Rol `teacher` / `admin`: Retorna el objeto `Exercise` completo incluyendo el bloque `config` con los `test_cases` completos.

### 3.8 `POST /api/v1/evaluations`
* **Propósito:** Evaluar la solución del estudiante contra los casos de prueba del ejercicio y devolver el veredicto del Juez Virtual.
* **Regla de Seguridad por Rol (GOB-SEGURO-01):** El rol `student` nunca recibe datos de casos ocultos en ninguna parte de la respuesta:
  * `failed_test_case.input` y `failed_test_case.expected_output` se enmascaran con `[OCULTO POR EXAMEN]` cuando el caso es `hidden` o el ejercicio corre en modo examen con semilla por estudiante.
  * `case_results[].message` (detalle por caso) se reemplaza por `<VERDICT>. Detalle: [CASO DE EVALUACIÓN OCULTO]` en los casos ocultos: el detalle original del comparador incluye la salida esperada y permitiría reconstruirla por iteración.
  * Los casos `example` y `public` conservan su detalle completo (`Esperado` / `Obtenido`) porque son visibles para el estudiante en el enunciado.
* **Regla de Seguridad por Rol (GOB-SEGURO-04):** En los exámenes con `per_student_seed` la entrada de cada caso se genera con la semilla `sha256(ejercicio:estudiante)`, de modo que dos estudiantes reciben casos distintos. La salida esperada de esos casos es la respuesta del ejercicio y **nunca** se entrega al rol `student`:
  * `POST /api/v1/evaluations` responde con el DTO `StudentEvaluationResponse`: los `generated_cases` viajan sin `expected_output` y el `failed_test_case` de un caso oculto no lleva salida esperada.
  * El evento WebSocket `EVALUATION_COMPLETED` emite el mismo DTO saneado.
  * `POST /api/v1/submissions` ignora el `generated_cases` declarado por el cliente: el servidor guarda el snapshot autoritativo calculado durante la evaluación (tabla `evaluation_generated_cases`) y el rol `student` tampoco puede leer la entrega de otro estudiante (`GET /api/v1/submissions/{id}` responde `403`).
* **Verificación:** `backend/internal/core/services/evaluation_hidden_case_isolation_test.go`, `backend/internal/delivery/http/dto/evaluation_dto_test.go` y los specs E2E `frontend/e2e/flows/02-medium/J03-verdict-matrix-and-hidden-isolation.spec.mjs` y `frontend/e2e/flows/03-hard/J05-curricular-unlock-and-exam-seed.spec.mjs` (assert sobre el payload HTTP y sobre el DOM del rol `student`).

---

## 4. Interfaces TypeScript Mapeadas (Angular 22)

```typescript
export interface Subject {
  id: string;
  tenant_id: string;
  name: string;
  code: string;
  classroom_course_id?: string;
  created_at: string;
}

export interface Submission {
  id: string;
  tenant_id: string;
  exercise_id: string;
  student_id: string;
  workspace_id?: string;
  code: string;
  verdict: 'AC' | 'WA' | 'TLE' | 'RE' | 'AST_BLOCKED';
  ast_result: any;
  execution_time_ms: number;
  memory_used_mb: number;
  submitted_at: string;
}

export interface TeacherInvitation {
  id: string;
  tenant_id: string;
  token: string;
  email: string;
  used: boolean;
  expires_at: string;
}

export interface TemplateCacheStatus {
  template_id: string;
  template_name: string;
  docker_image: string;
  cached: boolean;
  size_mb: number;
  local_image_id: string;
}

export interface WarmAllSummary {
  total_templates: number;
  pulled: number;
  already_cached: number;
  failed: number;
}
```

---

## 5. Plantillas Docker y Caché del Host (`/api/v1/admin/templates`)

### 5.1 `GET /api/v1/admin/templates/cache-status`
* **Propósito:** Inspeccionar el estado de presencia en disco local (`InspectLocalImage`) para cada plantilla homologada en el nodo Docker host.
* **Respuesta 200 OK:** `{"items": [TemplateCacheStatus, ...]}`

### 5.2 `POST /api/v1/admin/templates/{id}/warm`
* **Propósito:** Descarga forzada (*pull*) y certificación de presencia local bajo demanda con deduplicación *single-flight*.
* **Respuesta 200 OK:** Retorna el `TemplateCacheStatus` actualizado con `cached: true` y `size_mb`.

### 5.3 `POST /api/v1/admin/templates/warm-all`
* **Propósito:** Precalentamiento masivo pre-clase de todas las plantillas activas para evitar contención de red y saturación de Docker Hub (HTTP 429).
* **Respuesta 200 OK:** Retorna `WarmAllSummary` con el total de plantillas procesadas y descargadas.

