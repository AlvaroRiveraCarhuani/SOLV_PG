# Vigilancia externa del sitio

> **Para qué:** si el proceso que avisa es el mismo que se cae, nadie avisa. Un
> vigilante fuera del servidor manda un correo cuando el sitio deja de responder,
> aunque el servidor esté apagado, sin red o con el disco lleno.

---

## Qué se vigila

| Monitor | URL | Qué detecta |
|---|---|---|
| **API y base** | `https://<SOLV_HOST>/api/v1/health` | 200 = API y PostgreSQL responden · 503 = la API vive y la base no · sin respuesta = el sitio está caído |
| **Interfaz** (opcional) | `https://<SOLV_HOST>/` | El panel carga aunque siga habiendo un problema de datos |

`/api/v1/health` responde **503 cuando la base no contesta**. Esa es la razón de
que exista: un vigilante entiende códigos HTTP, no cuerpos JSON, y
`/api/v1/version` responde 200 incluso con PostgreSQL caído para que el
`HEALTHCHECK` del contenedor no reinicie la API por un problema ajeno.

Por qué no se vigila solo `/`: el frontend se sirve desde su propio contenedor,
así que puede estar arriba con el backend caído. Dos monitores cuestan lo mismo
que uno y distinguen los dos casos.

---

## Servicio recomendado: UptimeRobot, plan Free

Plan gratuito con 50 monitores, chequeo cada **5 minutos**, alertas por correo y
app móvil, sin tarjeta y permitido para uso comercial (documentación del
proveedor, noviembre de 2026).

**Por qué este y no otro:** es el único plan gratuito con margen de sobra para
varias instituciones, alerta por correo sin configurar nada y no requiere instalar
un agente en el servidor. Sus alternativas gratuitas bajan a 10 monitores
(StatusCake, Better Stack) o restringen el uso a proyectos personales.

**Su límite, dicho claro:** 5 minutos de intervalo significan hasta ~6 minutos de
ceguera, y durante una caída larga el proveedor espacia los rechequeos. Para un
sitio académico es suficiente: una clase no empieza sin que alguien lo note en ese
lapso. Si se necesita detección en un minuto, es un plan pago, no un cambio de
código.

### Puesta en marcha

1. Crear la cuenta en <https://uptimerobot.com> con el correo del responsable
   técnico. No hace falta tarjeta.
2. **Add New Monitor** → tipo **HTTP(s)**:
   - *Friendly name*: `SOLV API — <nombre de la institución>`
   - *URL*: `https://<SOLV_HOST>/api/v1/health`
   - *Monitoring interval*: `5 minutes`
   - *Alert contacts*: el correo dado de alta (y la app móvil si se quiere aviso
     con sonido).
3. Repetir con un monitor de tipo **Keyword** sobre `https://<SOLV_HOST>/`, con la
   palabra `SOLV` y `Alert when keyword does not exist`. Ese es el que detecta el
   frontend caído con la API viva.
4. **Probar el aviso, no confiar en él.** Con la instalación en marcha:
   ```bash
   docker compose stop backend
   ```
   Esperar hasta 6 minutos: llega el correo de caída. Levantar de nuevo y esperar
   el correo de recuperación. Un vigilante que nunca se probó es una suposición.

### Alternativa: vigilante propio

Uptime Kuma en el mismo servidor **no sirve para esto**: se cae con el servidor.
Si se quiere hospedar, tiene que vivir en otra máquina o en otro proveedor. El
plan gratuito de UptimeRobot evita mantener esa segunda máquina.

---

## Cuando llega el aviso

| El correo dice | Qué pasó | Dónde seguir |
|---|---|---|
| `503` en `/api/v1/health` | La API vive, PostgreSQL no | `incident-runbook.md` §4.3 |
| Sin respuesta / timeout | El sitio no atiende: proxy, backend o red | `incident-runbook.md` §4.1 y §4.2 |
| Falla solo el monitor de la interfaz | Backend arriba, frontend caído | `incident-runbook.md` §4.1 |
| Recuperación | Volvió a responder | Cerrar el incidente con la lista de §8 |

El aviso llega por correo, no por SMS: el plan gratuito no incluye créditos de
mensajes. Por eso el correo del responsable tiene que ser uno que se revise fuera
del horario de oficina.

---

## Límites de esta vigilancia

- **El aviso depende de internet.** Un corte del enlace del proveedor, del DNS o
  de la energía del edificio también corta la entrega del correo. No hay forma de
  avisar por un canal que se cayó con el sitio; para eso está el monitoreo del
  proveedor de hosting o la llamada de la institución.
- **No diagnostica.** Dice que algo no responde; el runbook dice qué mirar.
- **No reemplaza `/metrics`.** Prometheus y Grafana muestran tendencias y
  saturación con el sistema vivo; esto solo avisa cuando ya no responde.
- **No hay escalamiento automático.** Avisa a una persona. Quién es esa persona y
  qué hace a las 3 de la mañana está en `incident-runbook.md` §7.
