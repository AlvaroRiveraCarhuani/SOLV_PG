# Estado

**Aceptado**

**Actualizado:** 2026-11-09 (identidad solo desde sesión verificada; allowlists de rol, aislamiento de plantillas y state OAuth de un solo uso)

---

## 1. Contexto y Origen de la Necesidad

Para asignar correctamente los entornos y volúmenes persistentes, el sistema SOLV requiere identificar unívocamente a los usuarios.

Además, debido a que docentes y estudiantes comparten el mismo dominio institucional (`@uab.edu.bo`), el sistema necesita un mecanismo robusto de Autorización (AuthZ) para diferenciar privilegios (por ejemplo, crear laboratorios vs. unirse a laboratorios), sin depender de un acceso directo a la base de datos central de la universidad (Kardex).

---

## 2. Decisión Arquitectónica

Se implementará un sistema de identidad descentralizado y de **Control de Acceso Basado en Roles (RBAC)** híbrido a dos niveles, compuesto por los siguientes componentes:

### Autenticación (AuthN - Identidad)

- Single Sign-On (SSO) mediante OAuth 2.0 utilizando cuentas institucionales de Google Workspace.

### Autorización de Plataforma (AuthZ Global - Roles)

La asignación del rol de privilegio (**DOCENTE / ADMIN**) estará estrictamente aislada de plataformas externas para evitar ataques de escalada de privilegios. Se gestionará localmente mediante:

- **Pre-aprovisionamiento (Allowlist):**  
  Una lista blanca estática de correos institucionales pre-aprobados (*Zero-Click Onboarding*).

- **Autorización de Respaldo (Magic Links):**  
  Enlaces criptográficos de invitación de un solo uso para docentes que no se encuentren en la lista blanca inicial.

### Autorización Contextual (AuthZ de Entornos)

- Sincronización dinámica de inscripciones delegada a la API de Google Classroom.  
- El sistema cruzará la identidad del estudiante validada por SSO con las listas oficiales de las clases para otorgar acceso a los contenedores.

### Gestión de Sesión

- Emisión de **JSON Web Tokens (JWT)** firmados; la validez de la sesión se complementa con la revocación persistida.
- Cada JWT incluye un `jti` único. Los tokens heredados sin `jti` se identifican por un hash SHA-256 para no persistir nunca el token completo.
- El logout persiste la revocación en `session_revocations` (PostgreSQL) con la expiración del token. La autenticación HTTP, WebSocket y del terminal consultan esa tabla; si PostgreSQL no responde, la comprobación falla cerrada.
- Las conexiones WebSocket abiertas en otra instancia se cierran mediante un sondeo acotado de la misma tabla. La autorización posterior no depende del sondeo: vuelve a consultar el estado persistido.
- La identidad HTTP solo se toma del JWT en `Authorization: Bearer` o de la cookie HttpOnly `solv_session`. Un middleware raíz elimina `X-User-Id`, `X-User-Role`, `X-Tenant-Id`, `X-User-Email` y cabeceras equivalentes antes de entrar al mux; `WithAuth` y `WithTenant` las vuelven a fijar únicamente desde claims verificados para la propagación interna.
- Los handlers no leen cabeceras de identidad. `requestUserID`, `requestTenantID`, `requestRole`, `requestUserEmail`, `getUserIDFromCtx` y `getTenantFromCtx` consultan solo el contexto que escriben `WithAuth`/`WithTenant`; no existe fallback a headers aunque falte el saneador.
- El callback de Google valida `verified_email`, compara el dominio de forma exacta contra `allowed_domains` del tenant y crea o reutiliza el usuario con `tenant_id` y rol resueltos localmente. El JWT emitido contiene `user_id`, `email`, `role`, `tenant_id`, `jti` y `exp`. El alta administrativa `POST /api/v1/users` no participa en este aprovisionamiento interno.
- Cada intento de login inicia un `state` de 32 bytes generados con `crypto/rand` y codificados en base64url. El backend guarda el `state` en un almacén de un solo uso con vencimiento de 10 minutos y lo asocia al navegador mediante la cookie HttpOnly `solv_oauth_state` (`SameSite=Lax`, `Secure` fuera de loopback, dominio `COOKIE_DOMAIN`). El callback exige que el `state` de la URL y el de la cookie coincidan con comparación de tiempo constante, consume el valor antes del intercambio del código y rechaza con `403` los estados ausentes, incorrectos, reutilizados o vencidos. La cookie se invalida en toda respuesta del callback. El almacén vive en memoria: con varias instancias hace falta un almacén compartido para que el callback pueda completarse en una instancia distinta de la que inició el login.
- Un rol vacío o desconocido no satisface ninguna allowlist. Las operaciones docentes exigen `teacher` o `admin`; las administrativas exigen `admin`. No se usan comprobaciones por exclusión (`rol != student`).
- `POST /api/v1/users` y `POST /api/v1/templates` requieren sesión y rol `admin`. `GET /api/v1/templates` requiere sesión de docente o administrador y filtra por el tenant del claim. `GET /api/v1/config/public` selecciona el tenant solo por `slug` o por el tenant por defecto, nunca por header. `GET /metrics` acepta solo conexiones loopback o un `METRICS_TOKEN` configurado.

### Autorización de la superficie administrativa y de los entornos

- Las rutas `/api/v1/admin/...` se montan después de `WithTenant` con una allowlist explícita: `adminOnly` para las operaciones exclusivamente administrativas y `staffOnly` solo para el catálogo técnico que el docente necesita leer (`GET /api/v1/admin/templates` y `GET /api/v1/admin/template-components`). Las mutaciones conservan la auditoría, ya con el contexto autenticado disponible.
- `POST /api/v1/jobs/env-test` exige `teacher` o `admin`; no es una operación de estudiante ni una vía alternativa para modificar plantillas oficiales.
- `Terminate`, `GetSemgrepAudit`, `Heartbeat`, `Restart` y `Pause` de un entorno de práctica resuelven primero el entorno con el tenant del contexto. Un estudiante solo puede operar un entorno cuyo `student_id` coincide con su identidad; un docente o administrador conserva la gestión dentro de su tenant. Un entorno ajeno responde `403` y uno inexistente o de otro tenant responde `404`.
- `GET /api/v1/config/public` serializa un DTO de lista blanca (`tenant_id`, `slug`, `institution_name`, `logo_url`, color primario, dominio base, correo de soporte y familias tipográficas). No devuelve `tenants.config` completo ni campos de IA, credenciales cifradas, `user_overrides` o políticas internas.

---

## 3. Justificación y Eficiencia (Primeros Principios)

### Delegación de Identidad vs. Soberanía de Roles

- Al usar Google SSO, el sistema SOLV elimina completamente la necesidad de almacenar contraseñas locales, reduciendo significativamente la superficie de ataque.
- Se garantiza que solo usuarios institucionales activos puedan acceder al sistema.

Para mitigar el riesgo de **Privilege Escalation** (por ejemplo, estudiantes creando clases en Classroom para obtener permisos):

- SOLV mantiene la soberanía del rol.
- La base de datos local actúa como autoridad suprema:
  - Si un usuario no está en la Allowlist o no fue invitado → su rol será **ESTUDIANTE**.

---

### Automatización vs. Vendor Lock-in

- El uso de la API de Google Classroom se limita a consultar inscripciones.
- Esto permite automatizar el acceso a laboratorios sin intervención manual.

Para evitar dependencia total de Google:

- El núcleo de SOLV implementará el **Patrón Adaptador (Adapter Pattern)**.
- Esto abstrae el proveedor externo y permite migrar en el futuro (por ejemplo, a Moodle).

---

### Seguridad de Administración (Zero Trust)

El registro de docentes excepcionales se realizará mediante enlaces criptográficos de un solo uso que:

- expiran automáticamente  
- se invalidan al consumirse  

Esto evita filtraciones o accesos no autorizados.

Medidas adicionales:

- Mitigación de **Forced Browsing**:
  - Responder `404 Not Found` en lugar de `403 Forbidden`.
- Protección de endpoints administrativos:
  - Reglas de **IP Allowlisting** en el proxy inverso (Traefik).

---

## 4. Consecuencias y Trade-offs (Análisis de Costo)

### Costo

El desarrollo inicial requiere implementar:

- adaptadores para consumir APIs externas  
- archivo de configuración de Allowlist  
- módulo de generación de tokens criptográficos  
- sistema de invitaciones seguras  

Esto incrementa la complejidad del backend (FastAPI).

---

### Ventaja

El sistema final ofrece:

- autonomía frente a plataformas externas en el control de roles  
- escalabilidad para miles de usuarios  
- mínima carga administrativa para el docente  
- seguridad empresarial basada en **Defensa en Profundidad**  

Además, la arquitectura es resistente a ataques internos de escalada de privilegios.