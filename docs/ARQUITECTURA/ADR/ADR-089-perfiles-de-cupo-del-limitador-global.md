# ADR-089: Perfiles de cupo del limitador global por IP

- **Estado:** Aceptado
- **Fecha:** 2026-11-18
- **Ámbito:** Seguridad, operación, interfaz

## Contexto

El limitador global (`GlobalRateLimitMiddleware`) protege al backend con un
token bucket por IP. Se incorporó el 8 de octubre de 2026 en el hardening previo
al piloto con 100 peticiones por minuto y ráfaga de 20. Ese número se escribió
una vez y nunca se contrastó con el patrón de la interfaz.

Una carga de tablero abre entre ocho y doce peticiones en paralelo
(`/api/v1/config/public`, `/api/v1/users/me`, notificaciones, contador de no
leídas, aula, panel y pendientes). Con dos o tres pestañas la ráfaga se agota
antes de que el panel pinte, el backend responde 429 y la pantalla informa un
fallo de conexión que no existe. El 10 de noviembre se agregaron
`RATE_LIMIT_IP_PER_MINUTE` y `RATE_LIMIT_IP_BURST` para que los tests de
integración no chocaran con el límite, y los workflows pasaron a 6000/500.
Desde entonces convivían tres verdades: 100/20 en el binario, 6000/500 en CI y
ningún valor declarado como referencia de despliegue.

El 18 de noviembre el valor por defecto pasó a 600/120 para que un tablero
cargue con varias pestañas. Faltaba el criterio: sin perfiles declarados, cada
entorno vuelve a elegir un número distinto y el problema reaparece en el
siguiente despliegue.

## Decisión

El cupo del limitador global se define por perfil de uso. Solo existen las dos
variables de entorno `RATE_LIMIT_IP_PER_MINUTE` y `RATE_LIMIT_IP_BURST`; no se
agregan variables por ruta ni por rol.

| Perfil | Uso previsto | Peticiones por minuto | Ráfaga |
| :--- | :--- | ---: | ---: |
| A — Local y CI | Stack en loopback, tres roles y tests automatizados sobre `127.0.0.1` | 6000 | 500 |
| B — Nodo único con NAT | Instalación On-Premise con una persona y varias pestañas, o hasta cuatro personas tras la misma dirección pública. Es el valor por defecto del binario | 600 | 120 |
| C — Aula compartida | Cohorte de hasta 30 alumnos con dos pestañas cada uno detrás de una única dirección pública y apertura simultánea de clase | 3000 | 600 |

### Criterio de dimensionamiento

El tamaño se deriva de la carga simultánea, no de una petición aislada.

1. **Ráfaga** = cargas simultáneas máximas × 10 peticiones por carga.
   - Perfil B: 12 cargas simultáneas (cuatro personas con tres pestañas) = 120.
   - Perfil C: 60 cargas simultáneas (30 alumnos con dos pestañas) = 600.
2. **Cupo por minuto** = 5 × ráfaga en los perfiles reales. Cubre el sondeo de
   notificaciones, presencia y refresco de panel (60 s cada uno) más la
   navegación normal sin acercarse al techo.
3. **Perfil A** no se dimensiona por carga de usuario: el objetivo es que
   ninguna ejecución automatizada choque con el limitador, porque todas las
   sesiones salen por `127.0.0.1` y comparten un único cubo.
4. Si la interfaz agrega llamadas a la carga inicial de una pantalla, se revisa
   la ráfaga del perfil afectado en el mismo cambio. La carga de tablero es el
   número que manda.

### Reglas de consistencia

1. **Fuente única de valores.** Los tres perfiles se declaran en este ADR. El
   perfil B vive además como valor por defecto en `ipLimitConfig`
   (`backend/internal/delivery/http/middleware/rate_limiter.go`) y como
   plantilla en `example.env`. Los workflows usan el perfil A. Cualquier otro
   archivo que necesite un valor lo referencia, no lo repite.
2. **Sin literales sueltos.** Prohibido escribir un cupo numérico en código,
   Compose, scripts o documentación fuera de los archivos anteriores.
3. **Cambio coordinado.** Modificar un valor exige actualizar en el mismo
   cambio: este ADR, `docs/SECURITY/security-hardening.md`,
   `docs/GOBERNANZA/TESTING_POR_IMPACTO.md` y `example.env`.
4. **Selección en despliegue.** Un laboratorio con una sola salida a Internet
   declara el perfil C en su `.env`; no se cambia el valor por defecto del
   binario ni se agrega una rama de código por entorno.
5. **Diagnóstico.** El backend publica `X-RateLimit-Limit`,
   `X-RateLimit-Remaining`, `X-RateLimit-Reset` y `Retry-After`. Un 429
   repetido en la carga de un panel se investiga primero contra el perfil del
   entorno y después contra la interfaz.

### Límite conocido

El cubo es por IP y no por usuario. El perfil C reduce el problema de un aula
detrás de NAT, pero no lo elimina: un cliente de esa IP consume el cupo de
todos. La corrección de fondo es contar por identidad autenticada y dejar la IP
como respaldo anónimo; eso exige resolver la sesión antes del limitador o mover
el middleware detrás de la autenticación y tendrá su propio ADR. Mientras tanto,
el perfil C es la referencia operativa para un aula compartida.

## Consecuencias

- Un despliegue nuevo no hereda un cupo pensado para una sola petición: el
  perfil B cubre una sesión real con varias pestañas.
- CI y E2E conservan 6000/500, ahora identificados como perfil A y no como una
  excepción sin nombre.
- El perfil C no se activa solo. Un aula compartida lo declara en su `.env` y
  queda registrado en la configuración del despliegue.
- Los límites específicos no cambian: inicio de sesión 5 intentos fallidos por
  IP cada 15 minutos, envíos 10 por minuto por alumno.
- El 429 sigue existiendo como protección; lo que cambia es que su umbral queda
  documentado y es el mismo en código, plantilla y documentación.

## Verificación

- `go test ./internal/delivery/http/middleware -run RateLimiter` comprueba que
  la petición siguiente a la ráfaga configurada recibe 429 y que el bloqueo por
  intentos fallidos de autenticación sigue activo.
- Una prueba de ráfaga contra el stack desplegado envía `ráfaga + 1` peticiones
  a un endpoint público: las primeras responden 200 y la última 429. Con el
  perfil B son 121 peticiones.
- `grep` de `RATE_LIMIT_IP_PER_MINUTE` y `RATE_LIMIT_IP_BURST` en el repositorio
  debe devolver solo este ADR, `example.env`, los dos workflows y los documentos
  que los referencian.
