# Arquitectura del Harness E2E Multi-Rol con Playwright

## 1. Propósito y Alcance

Este directorio (`docs/E2E/`) documenta las pruebas de sistema End-to-End (E2E) que atraviesan las cuatro capas reales de la plataforma:

1. **Frontend Angular 22** (vistas standalone, Signals, validaciones y formularios reactivos).
2. **Backend Go 1.26** (`net/http.ServeMux`, middlewares de sesión, multi-tenancy y casos de uso).
3. **PostgreSQL 18** (persistencia transaccional real en `solv_db`).
4. **Docker Engine** (comprobación de entornos `env-test`, síntesis de imágenes y ejecución aislada del Juez Virtual).

A diferencia de las pruebas unitarias o de tipografía aislada (`frontend/e2e/typography.spec.mjs`), los flujos en `frontend/e2e/flows/` validan contratos completos entre los tres roles del sistema: **Administrador (`admin`)**, **Docente (`teacher`)** y **Estudiante (`student`)**.

---

## 2. Bypass Determinista de Google OAuth2 (Inyección de `solv_session`)

### 2.1. Problema con la pantalla externa de Google Workspace
En uso humano, al ingresar a `/login` y presionar `Continuar con Google Workspace`, el navegador redirige a `/auth/google/login` y luego a `accounts.google.com`. Los navegadores controlados por automatización son bloqueados por las protecciones anti-bot de Google (CAPTCHA, verificación de dispositivo o bloqueo de Chromium headless).

### 2.2. Solución Arquitectónica (Alineada con D1 — ForwardAuth y JWT)
El middleware `WithAuth` del backend (`backend/internal/delivery/http/middleware.go`) y el servicio `AuthService` (`frontend/src/app/core/services/auth.service.ts`) autentican cada solicitud mediante:
- La cookie `HttpOnly` llamada `solv_session`, o
- El encabezado `Authorization: Bearer <token>` leído desde `sessionStorage.getItem('solv_token')`.

El token es un JWT firmado con `HS256` usando el `JWT_SECRET` del entorno local y contiene los claims:
- `user_id`: UUID real existente en la tabla `users` de PostgreSQL.
- `email`: Correo institucional asociado al usuario.
- `role`: Rol activo (`admin`, `teacher` o `student`).
- `tenant_id`: UUID del tenant institucional (`00000000-0000-0000-0000-000000000001`).
- `exp`: Timestamp Unix de expiración.

El fixture `frontend/e2e/fixtures/auth-roles.fixture.mjs`:
1. Verifica o aprovisiona en PostgreSQL los tres actores canónicos (`admin`, `teacher`, `student`) antes de iniciar la corrida.
2. Firma un JWT válido para cada rol con el `JWT_SECRET` activo del backend.
3. Crea tres `BrowserContext` independientes en Playwright (`adminContext`, `teacherContext`, `studentContext`), inyectando tanto la cookie `solv_session` como `sessionStorage['solv_token']` antes de navegar.
4. De este modo, cada rol navega directamente a `/admin/...`, `/teacher/...` o `/student/...`, autenticándose contra `GET /api/v1/users/me` en el backend real sin pasar por `accounts.google.com`.

---

## 3. Ejecución Fail-Fast y Diagnóstico Forense Automático

### 3.1. Corte Inmediato al Primer Fallo (`maxFailures: 1` + `mode: 'serial'`)
Los flujos multi-rol tienen dependencia secuencial estricta (el docente necesita la plantilla publicada por el administrador; el estudiante necesita el ejercicio publicado por el docente). Por ello:
- `frontend/playwright.e2e.config.mjs` configura `maxFailures: 1`, `workers: 1` y `fullyParallel: false`.
- Cada suite de flujo declara `test.describe.configure({ mode: 'serial' })`.
- Ante el primer `expect` fallido, error de red no esperado o timeout de un componente, Playwright detiene la corrida inmediatamente.

### 3.2. Capturador Forense (`frontend/e2e/fixtures/forensic-logger.fixture.mjs`)
Durante cada paso, el fixture forense intercepta de forma pasiva:
- **Respuestas HTTP de error (`status >= 400`)**: método, URL, payload enviado y cuerpo JSON devuelto por el backend Go.
- **Errores de consola y excepciones no capturadas (`pageerror` / `console.error`)**: stacktrace exacto en el navegador.
- **Estado de la vista al fallar**: URL actual, rol activo (`admin` / `teacher` / `student`), título del paso y captura de pantalla (`screenshot: 'only-on-failure'`).

Si un paso falla, el fixture escribe automáticamente un archivo estructurado en:
`frontend/e2e/test-results/forensic-last-failure.json`

Esto permite inspeccionar de inmediato qué endpoint o selector falló, corregir la causa raíz y relanzar el flujo sin adivinar.

---

## 4. Comandos de Ejecución

Desde el directorio `frontend/`:

```bash
# Ejecutar todos los flujos E2E con corte al primer error (headless)
npm run test:e2e:flows

# Ejecutar viendo el navegador en vivo (headed)
npx playwright test --config=playwright.e2e.config.mjs --headed
```
