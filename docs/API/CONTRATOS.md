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
```
