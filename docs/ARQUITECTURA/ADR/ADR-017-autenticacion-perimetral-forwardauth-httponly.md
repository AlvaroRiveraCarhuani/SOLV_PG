# ADR-017: Autenticación Perimetral ForwardAuth vía Cookie HttpOnly Cross-Subdomain (D1)

* **Estado:** Aceptado (Evoluciona y sustituye la autenticación de ADR-004 y ADR-015)
* **Fecha:** 2026-08-04

## Contexto y Problema
Los entornos de desarrollo de los estudiantes se sirven dentro de elementos `iframe` en la interfaz de Angular apuntando a subdominios dinámicos (ej: `<uuid>.solv.uab.edu.bo`). Como el cliente de Angular almacena el token JWT en memoria por razones de seguridad, no puede inyectar automáticamente cabeceras de autorización HTTP (`Authorization: Bearer <token>`) en las solicitudes nativas del navegador para cargar los activos HTML/JS del `iframe`. Pasar el token JWT en la URL introduciría graves vulnerabilidades de filtración en historiales y logs.

## Decisión Tomada
1. Implementar la decisión de arquitectura **D1**: Autenticación del `iframe` vía cookie `HttpOnly` de dominio combinada con el middleware **ForwardAuth** en Traefik v3.
2. Tras el SSO de Google, el backend setea de forma aditiva la cookie `solv_session` con las banderas:
   * `HttpOnly: true` (inaccesible desde scripts JavaScript del cliente)
   * `Secure: true` (exclusivo para conexiones TLS/HTTPS)
   * `SameSite: Lax` (permite navegación cross-subdomain en el dominio padre)
   * `Domain: COOKIE_DOMAIN` (ej. `.solv.uab.edu.bo`)
   * `Path: /`
3. Crear el endpoint de bajísima latencia (<50ms SLA) `GET /api/v1/auth/verify` en Go (`AuthHandler.VerifyAuth`), el cual Traefik v3 consulta antes de permitir el paso de cada solicitud HTTP hacia cualquier subdominio de laboratorio.
4. En caso de cookie válida, responde `200 OK` inyectando las cabeceras `X-User-Id` y `X-User-Role`. Si falta o es inválida, responde `401 Unauthorized`. Si el JWT expiró, responde `403 Forbidden`.

## Consecuencias
* **Positivas:**
  * Seguridad Zero Trust sin exponer nunca el JWT en las URLs de los `iframes`.
  * Validación perimetral centralizada en Traefik antes de procesar solicitudes en los contenedores.
  * Compatibilidad total con la navegación cross-subdomain entre el dashboard y los laboratorios.
  * Tiempo de respuesta del endpoint de verificación menor a 0.1ms (~55µs).
