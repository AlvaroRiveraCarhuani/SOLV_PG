# ADR 017 a 022: Hardening BaaS, Modelo de Dominio Unificado y Blindaje Perimetral

## Contexto y Problema
En la evolución de la plataforma **SOLV** hacia una arquitectura On-Premise robusta y lista para BaaS (Backend-as-a-Service), la separación histórica entre la tabla `workspaces` (IDEs persistentes) y `lab_instances` (ejecuciones efímeras del juez) generaba inconsistencias relacionales, duplicidad de repositorios y deuda técnica.

Asimismo, la exposición de contenedores de laboratorio mediante subdominios dinámicos alojados en `iframes` cross-subdomain presentaba riesgos de seguridad: la incapacidad de Angular para inyectar tokens JWT Bearer en peticiones HTML de `iframe` forzaba la necesidad de un esquema de autenticación perimetral transparente a nivel de red (Traefik v3).

Finalmente, el comportamiento predeterminado de Docker Engine de manipular reglas de `iptables` sobreescribía los firewalls tradicionales del host, exponiendo puertos internos y permitiendo bypasses directos sin pasar por la capa de enrutamiento y autenticación de Traefik.

---

> [!WARNING]
> **Actualización de ADRs Anteriores (Evolución de Arquitectura):**
> - **ADR-004 & ADR-015 (Superseded en autenticación):** La autenticación directa en contenedores o parámetros de URL queda oficialmente reemplazada por la autenticación perimetral **ForwardAuth** mediante cookie HttpOnly cross-subdomain (`ADR-017`).
> - **ADR-000 (Evolución del modelo de datos):** El modelo relacional unifica la orquestación bajo el recurso `workspaces`, marcando la deprecación definitiva de `lab_instances` (`ADR-019`).

---

## Decisiones Tomadas (D1 – D6)

### 1. ADR 017 (D1): Autenticación Perimetral ForwardAuth vía Cookie HttpOnly Cross-Subdomain
* **Decisión:** Implementar un middleware ForwardAuth en Traefik v3 que consulte el endpoint interno en Go `GET /api/v1/auth/verify` antes de servir cualquier solicitud HTTP a subdominios de laboratorios.
* **Mecanismo:** La sesión autenticada se transporta mediante la cookie de dominio `solv_session` con banderas `HttpOnly: true`, `Secure: true`, `SameSite: Lax`, `Path: /` y `Domain: COOKIE_DOMAIN`.
* **Rendimiento:** El endpoint `VerifyAuth` valida la firma HMAC-SHA256 del JWT y su fecha de expiración en menos de 50 microsegundos (<50ms SLA), inyectando headers `X-User-Id` y `X-User-Role` a los servicios aguas abajo.

### 2. ADR 018 (D2): Arquitectura Multi-Tenant Lógica por Discriminador `tenant_id`
* **Decisión:** Adoptar un esquema relacional compartido en PostgreSQL donde todas las tablas transaccionales incluyen la columna `tenant_id UUID NOT NULL`.
* **Aislamiento:** Un middleware HTTP en Go (`TenantMiddleware`) extrae el tenant del claim JWT y del header `X-Tenant-ID`, inyectándolo en el `context.Context` de la solicitud y aplicando filtros automáticos en las capas de repositorio (`WHERE tenant_id = $1`).

### 3. ADR 019 (D3): Consolidación del Modelo de Dominio de Workspaces y Discriminador `type`
* **Decisión:** Eliminar físicamente la tabla `lab_instances` y unificar la orquestación en la tabla `workspaces` mediante la columna discriminadora `type VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE'`.
* **Valores:** `type = 'IDE_PERSISTENTE'` para laboratorios interactivos VS Code y `type = 'JUEZ_EFIMERO'` para ejecuciones de evaluación automatizada.
* **Migración:** La migración SQL en `postgres.go` es 100% idempotente (`ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS type ...` y transferencia condicional de registros).

### 4. ADR 020 (D4): Blindaje Perimetral de Red y Cadena `DOCKER-USER` de `iptables`
* **Decisión:** Crear el script de infraestructura [docker-user-rules.sh](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/infra/firewall/docker-user-rules.sh) para interceptar el tráfico en la cadena `DOCKER-USER` de `iptables` antes de que Docker aplique sus reglas NAT.
* **Aislamiento:** Forzar el enlace de PostgreSQL a `127.0.0.1:5432` y denegar el acceso público directo a los puentes de Docker, garantizando que el único punto de entrada expuesto sea Traefik v3 en los puertos 80 y 443.

### 5. ADR 021 (D5): Parametrización Total de Dominios Organizacionales (Cero Hardcode)
* **Decisión:** Desacoplar completamente el código fuente de cualquier dominio específico (ej. `uab.edu.bo`).
* **Configuración:** Toda URL, cookie o certificado se parametriza a través de las variables de entorno `COOKIE_DOMAIN`, `DESEC_TOKEN`, `GOOGLE_REDIRECT_URL` y `DATABASE_URL`.

### 6. ADR 022 (D6): Auditoría Semántica AST Inmutable con Montaje de Solo Lectura
* **Decisión:** Garantizar que el servicio `SemgrepWorker` ejecute escaneos AST montando el volumen del estudiante de forma strictly inmutable (`:ro`).
* **Persistencia:** La auditoría resultante se inyecta directamente en la columna `semgrep_audit JSONB` de la tabla `workspaces`.

---

## Consecuencias

* **Positivas:**
  * Eliminación total de deuda técnica relacional al unificar `workspaces`.
  * Protección Zero Trust en subdominios de iframes sin exponer tokens en la URL.
  * Inmunidad contra el bypass de firewall por parte de Docker Engine.
  * Multi-Tenancy nativo listo para producción BaaS.
* **Negativas / Desafíos:**
  * Requiere la ejecución previa del script de firewall `docker-user-rules.sh` con privilegios `sudo` en el servidor anfitrión.
