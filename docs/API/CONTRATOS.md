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

## 3. Interfaces TypeScript Mapeadas (Angular 22)

```typescript
export interface WorkspaceInstance {
  id: string;
  student_id: string;
  subject_id: string;
  tenant_id: string;
  type: 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO';
  status: 'pending' | 'running' | 'hibernated' | 'failed' | 'oom_killed';
  access_url: string;
  memory_limit_mb: number;
}
```
