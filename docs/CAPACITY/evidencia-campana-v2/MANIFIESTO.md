# Campaña experimental v2 — manifiesto de evidencia

- **Commit medido:** `6ea0589c0ccdc78ce291dd8ebe23dcee1f62f268` (rama `main`, merge del PR #1)
- **Nodo:** `asus-server`, Intel Core i7-6700HQ (4 núcleos / 8 hilos), 11.852 MiB de RAM física, Ubuntu 24.04, kernel `6.8.0-151-generic`, Docker Engine `29.3.1`, cgroups v2
- **Ventana de la campaña:** `2026-11-27T11:05:40Z` → `2026-11-27T12:18:34Z`
- **Bitácora de ejecución:** `campaign_run.log` (una línea por etapa y por réplica, con `start_utc`/`end_utc`)
- **Resumen calculado:** `raw/metrics_summary.json` y `raw/metrics_summary.txt`

La campaña responde a las observaciones 8, 9 y 10 del tutor. Se ejecutó sobre el nodo piloto real, por HTTPS a través de Traefik, con sesiones firmadas y el mismo flujo autenticado de la campaña v1. No se modificó ningún archivo de `proyecto-grado/` ni se hizo commit.

---

## 1. Qué se midió y por qué

| Observación | Pregunta | Escenario que la responde |
| :--- | :--- | :--- |
| 8 | ¿El sistema sostiene una carga de aula realista (10–35 alumnos) sin degradarse? | E1 escalonado + E1-cal de calibración |
| 9 | ¿Dónde está el techo real del flujo síncrono de evaluación? | Saturación 40 → 60 → 100 VUs |
| 10 | ¿Se mantiene estable en el tiempo y qué pasa con la provisión en frío/caliente? | Soak de 30 min + E6 frío/caliente |

La campaña v1 (`docs/CAPACITY/pilot-raw/`) midió 200 usuarios virtuales durante 60 s y encontró que ninguna de las 204 evaluaciones terminó. Dejó sin responder qué pasa **por debajo** de ese extremo: si 10 alumnos ya rompen el flujo, si el techo está en 20 o en 35, y si el nodo aguanta media hora. Eso es lo que se mide aquí.

---

## 2. Condiciones de contorno declaradas

### 2.1 Pool de cuentas y datos

- 40 estudiantes de aula temporales (`campaign-v2-sNN@test.local`) + 1 estudiante dedicado al IDE (`campaign-v2-ide@test.local`), con sesiones JWT firmadas con el `JWT_SECRET` local (mismo contrato de claims que `backend/cmd/gentoken`).
- Materia propia de la campaña `CS-CAP-V2` con 10 prácticas de juez efímero y 3 proyectos IDE, todos con `purpose = class`. Se usó `class` y no `exam` a propósito: un examen bloquea la reentrega (`AllowsResubmit()` devuelve `false`), y con un solo envío por par (alumno, ejercicio) el flujo de envíos se agotaría a los pocos minutos de carga. Todas las cifras de envío corresponden a prácticas con reentrega habilitada.
- Cada VU usa una identidad distinta del pool (35 VUs → 35 alumnos), de modo que ningún `429` provenga de un bloqueo por usuario.

### 2.2 Cupos del limitador (leídos del código y de `.env`, no supuestos)

| Limitador | Cupo | Clave | Fuente |
| :--- | ---: | :--- | :--- |
| Global por IP (`GlobalRateLimitMiddleware`) | 120.000/min, ráfaga 10.000 | IP del cliente (`X-Forwarded-For`) | `.env` (`RATE_LIMIT_IP_PER_MINUTE`, `RATE_LIMIT_IP_BURST`) |
| Envíos de evaluación (`middleware.NewUserRateLimiter(10, 10)`) | 10/min, ráfaga 10 | `user_id` del JWT | `backend/cmd/api/main.go:579-581` |
| Login | 5/min, ráfaga 5 | usuario | ídem |

Con *think time* de 12–18 s por envío, cada alumno produce ~3 envíos/min: un tercio del cupo. No se observó ni un `429` en toda la campaña (ver §3 y §5). Los cupos del perfil C de ADR-089 no se tocaron.

### 2.3 Preparación del entorno (intervención declarada)

`POST /api/v1/evaluations` devolvía **HTTP 500** al inicio de la campaña:

```
{"error":"error en pre-chequeo AST Semgrep: semgrep execution error: exec: \"semgrep\": executable file not found in $PATH"}
```

El binario no existía dentro de `solv_backend`. Se instaló `semgrep==1.186.0` (la misma versión que usó la campaña v1) dentro del contenedor en ejecución. Log en `raw/environment_prep_semgrep.log`. Sin esa intervención no hay recibo de evaluación y, por lo tanto, no hay envíos: toda la campaña habría medido un error de despliegue. Con semgrep el flujo responde 200 y emite recibo.

### 2.4 Un defecto funcional observado, que no se corrigió

Todas las evaluaciones devuelven veredicto `RE` con el mensaje `python: can't open file '/runner/solution.py': [Errno 2] No such file or directory` (3 de 3 casos). El runner monta `/runner` y ejecuta `python /runner/solution.py`, pero el archivo no está. Es un defecto de HEAD, ajeno al alcance de esta campaña: **las latencias de evaluación miden el costo de arranque de contenedor + Semgrep + ciclo de casos, no el costo de ejecutar el código del alumno**. Se declara para que nadie lea los tiempos como si el juez hubiera ejecutado la solución.

---

## 3. E1 — escalonado realista de aula (obs. 8, 9 y 10)

### 3.1 Diseño

- Niveles: **10, 20 y 35 VUs**; **3 réplicas independientes por nivel** de 60 s cada una, con **30 s de enfriamiento**.
- Rampa de 10 s hasta el nivel objetivo y 50 s de meseta (evita el pico artificial de 35 alumnos entrando en el mismo milisegundo).
- *Think time*: 3–5 s tras leer el ejercicio; 12–18 s tras cada envío.
- Cada VU: `GET /api/v1/exercises/{id}` → `POST /api/v1/evaluations` (recibo) → `POST /api/v1/submissions` → `GET /api/v1/submissions/{id}`.
- Cliente HTTP: k6 con `timeout` de 60 s (el mismo presupuesto que usa la interfaz para declarar un fallo).

### 3.2 Réplicas y ventanas

| Nivel | Réplica | `start_utc` | `end_utc` |
| ---: | ---: | :--- | :--- |
| 10 | 1 | 2026-11-27T11:16:36Z | 2026-11-27T11:18:06Z |
| 10 | 2 | 2026-11-27T11:18:36Z | 2026-11-27T11:20:06Z |
| 10 | 3 | 2026-11-27T11:20:36Z | 2026-11-27T11:22:06Z |
| 20 | 1 | 2026-11-27T11:22:06Z | 2026-11-27T11:23:37Z |
| 20 | 2 | 2026-11-27T11:24:07Z | 2026-11-27T11:25:37Z |
| 20 | 3 | 2026-11-27T11:26:07Z | 2026-11-27T11:27:40Z |
| 35 | 1 | 2026-11-27T11:27:40Z | 2026-11-27T11:29:11Z |
| 35 | 2 | 2026-11-27T11:29:41Z | 2026-11-27T11:31:12Z |
| 35 | 3 | 2026-11-27T11:31:42Z | 2026-11-27T11:33:13Z |

### 3.3 Lectura de ejercicios (3 réplicas agregadas)

| VUs | N | Éxito | Media | σ | p50 | p95 | p99 | Mín | Máx |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 10 | 30 | 100 % (30/30) | 13,26 ms | 12,62 ms | 8,98 ms | 30,37 ms | 72,66 ms | 6,43 ms | 72,66 ms |
| 20 | 60 | 100 % (60/60) | 20,73 ms | 28,99 ms | 11,21 ms | 45,91 ms | 207,50 ms | 6,98 ms | 207,50 ms |
| 35 | 105 | 100 % (105/105) | 17,82 ms | 12,69 ms | 16,29 ms | 32,76 ms | 66,17 ms | 6,45 ms | 103,55 ms |

### 3.4 Envío de evaluación

| VUs | Evaluaciones intentadas | Completadas (< 60 s) | % completadas | Envíos creados | Latencia observada |
| ---: | ---: | ---: | ---: | ---: | :--- |
| 10 | 30 | 0 | 0 % | 0 | 30 timeouts del cliente a 60.000,69 ± 0,35 ms |
| 20 | 60 | 0 | 0 % | 0 | 60 timeouts del cliente a 60.000,67 ± 0,33 ms |
| 35 | 105 | 0 | 0 % | 0 | 105 timeouts del cliente a 60.000,63 ± 0,33 ms |

**Distribución de códigos HTTP:** lectura `200` en 195/195; evaluación `0` (timeout del cliente, sin respuesta) en 195/195; envío: no hubo petición porque no hubo recibo.

### 3.5 Lectura

La lectura de ejercicios no se inmuta: p95 entre 30 y 46 ms en los tres niveles, con 195/195 respuestas 200. El envío de evaluación no completa **ni una** petición en 60 s, ni siquiera con 10 alumnos. La causa no es un `429` (no hubo ninguno) ni un error del servidor: la petición sigue viva cuando el cliente se rinde. La §5 pone número a esa espera: **112 s de media a 10 VUs**.

Con 60 s de ventana y una evaluación que tarda ~112 s, cada VU alcanza a intentar una sola evaluación por réplica. Eso explica que N sea exactamente igual al número de VUs en las tres filas.

---

## 4. Saturación / punto de quiebre

Rampa controlada 40 → 60 → 100 VUs, 20 s de subida y 45/60 s de meseta por escalón, *think time* corto (2–4 s) para buscar el techo y no simular aula. Ventana: `2026-11-27T11:33:13Z` → `2026-11-27T11:37:44Z`.

| Métrica | N | Resultado |
| :--- | ---: | :--- |
| Lecturas | 306 | 100 % `200` |
| Evaluaciones | 235 | 0 completadas (100 % timeout del cliente a 60 s) |
| Envíos | 0 | — |
| Búsqueda de `429` | 541 peticiones | 0 respuestas `429` |

**Techo observado:** no hay un nivel de VUs en el que el flujo síncrono empiece a degradarse de forma gradual; ya está saturado en el escalón más bajo de esta rampa (40 VUs). La campaña v1 lo había visto a 200 VUs y lo atribuyó al daemon de Docker. La v2 muestra que el techo está **muy por debajo de 40**, y la calibración de §5 lo acota: a 10 VUs la evaluación tarda 112 s.

---

## 5. E1-cal — calibración del tiempo real de evaluación

Corrida complementaria, fuera de las 9 réplicas de E1: 10 VUs, 180 s de meseta, **timeout de cliente de 240 s** para poder observar la respuesta en lugar de cortarla a 60 s. Ventana: `2026-11-27T12:12:26Z` → `2026-11-27T12:16:07Z`.

| Métrica | N | Éxito | Media | σ | p50 | p95 | Mín | Máx |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Evaluación (`POST /evaluations`) | 11 | 100 % `200` | 111.725,44 ms | 10.587,33 ms | 115.696,83 ms | 121.697,24 ms | 83.361,96 ms | 121.697,24 ms |
| Envío (`POST /submissions`) | 11 | 100 % `201` | 154,61 ms | 111,21 ms | 122,80 ms | 330,64 ms | 35,18 ms | 330,64 ms |
| Lectura | 20 | 100 % `200` | 20,60 ms | 10,55 ms | 16,17 ms | 32,50 ms | 9,40 ms | 57,28 ms |

Se intentaron 11 evaluaciones y las 11 completaron (100 %) porque el timeout del cliente se amplió a 240 s; el resto de la ventana de 180 s se consumió esperando. Cada VU alcanzó a leer dos veces y a enviar una sola evaluación: 20 lecturas frente a 11 envíos. El envío, una vez que existe recibo, es barato: p50 123 ms.

Mediciones puntuales de contraste (una sola evaluación, sin concurrencia):

| Momento | `start_utc` | HTTP | Total |
| :--- | :--- | ---: | ---: |
| Antes de la campaña (una evaluación aislada) | 2026-11-27T11:07:58Z | 200 | 17.857,76 ms |
| Después del soak de 30 min | 2026-11-27T12:12:04Z | 200 | 18.653,22 ms |

**Lectura:** el costo de una evaluación aislada es ~18 s. Con 10 alumnos concurrentes sube a ~112 s, un factor de 6,2. La latencia no crece de forma lineal con la concurrencia: el cuello es el arranque serializado de contenedores del daemon (3 casos de prueba por evaluación, más Semgrep). Ese es el techo real del flujo síncrono.

---

## 6. E6 — frío vs. caliente con repeticiones controladas (obs. 8 y 10)

### 6.1 Definición explícita de las condiciones

| Condición | Definición operativa | Qué se elimina antes de medir | Volumen | Imagen |
| :--- | :--- | :--- | :--- | :--- |
| **Frío** (N = 12) | `DELETE /api/v1/environments/{id}` + `docker rm -f` + borrado de la fila en `environments` + `docker volume rm` + `POST /environments/start` | Contenedor, fila y volumen del intento | Nuevo en las 12 corridas (`volume_state = removed_before_run`) | En caché local (`image_cached = true` en 12/12) |
| **Caliente** (N = 12) | `POST /environments/{id}/pause` + espera de 5 s + `POST /environments/start` | Nada: se reutiliza el contenedor pausado | Existente | En caché |

Ambas series: 3 bloques de 4 corridas, 30 s entre bloques. Se verificó con `docker inspect` que el contenedor estuviera `Paused=true` antes de medir la reanudación (12/12). Ventana: `2026-11-27T11:11:27Z` → `2026-11-27T11:16:36Z`.

> **Diferencia con la v1:** el frío de la v1 (15.842,88 ms) fue la primera provisión absoluta del par (alumno, ejercicio) e incluyó la creación de la base satélite. El frío de la v2 mide el caso de aula: la base satélite ya existe y lo que se crea desde cero es el contenedor y el volumen. No son la misma condición y no deben compararse como si lo fueran.

### 6.2 Provisión en frío

| N | Éxito | Media | σ | p50 | p95 | Mín | Máx |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 12 | 100 % (`200` 12/12) | 2.610,64 ms | 257,17 ms | 2.495,26 ms | 3.165,12 ms | 2.268,92 ms | 3.165,12 ms |

Errores HTTP: ninguno. Volumen nuevo en 12/12 corridas.

### 6.3 Hibernación y reanudación en caliente

| Operación | N | Éxito | Media | σ | p50 | p95 | Mín | Máx |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Hibernación (`pause`) | 12 | `200` 12/12 | 150,97 ms | 14,87 ms | 148,56 ms | 176,04 ms | 130,65 ms | 176,04 ms |
| Reanudación (`start` sobre pausado) | 12 | `200` 12/12 | 291,14 ms | 67,04 ms | 271,74 ms | 495,85 ms | 237,74 ms | 495,85 ms |

**Lectura:** la reanudación por `docker unpause` es de 291 ms, no de 8,9 s. La v1 midió 8.939,50 ms porque su entorno se había quedado sin contenedor y el servicio lo reconstruyó: lo que midió fue una reconstrucción, no una reanudación. Con el contenedor realmente pausado, el ciclo hibernar → esperar 5 s → reanudar cuesta ~442 ms en total. La observación 8 se responde con esto: el problema de tiempos no está en el ciclo de vida del IDE, está en la evaluación.

---

## 7. Soak de 30 minutos (obs. 10)

### 7.1 Condiciones

- 15 VUs de aula con el mismo flujo y *think time* de E1, más consulta de estado de la entrega.
- 2 entornos IDE activos en paralelo. Se pidieron 3: el tercero fue **rechazado con `503` en 89 ms** por la política de admisión del nodo, con 2 ya activos y la carga de evaluación en curso. El rango pedido era 2 a 3.
- Muestreo del host cada 60 s en `raw/soak_metrics.csv`.
- Ventana: `2026-11-27T11:38:45Z` → `2026-11-27T12:09:29Z`.

### 7.2 Cortes de minuto 0 / 10 / 20 / 30

| Corte | `timestamp_utc` | MemAvailable | VmRSS `solv-api-dev` | loadavg 1 min | Disco libre | Contenedores | Entornos IDE |
| ---: | :--- | ---: | ---: | ---: | ---: | ---: | ---: |
| 0 | 2026-11-27T11:39:18Z | 8.439 MiB | no válido (ver §10.5) | 47,53 | 843,01 GB | 11 | 4 |
| 10 | 2026-11-27T11:49:04Z | 8.067 MiB | 56,21 MiB | 12,25 | 843,00 GB | 12 | 4 |
| 20 | 2026-11-27T11:59:08Z | 8.317 MiB | 54,02 MiB | 12,78 | 843,00 GB | 9 | 2 |
| 30 | 2026-11-27T12:09:13Z | 8.355 MiB | 52,97 MiB | 13,46 | 842,99 GB | 12 | 2 |

Serie VmRSS (28 muestras válidas): media 53,99 MiB, σ 1,75 MiB, p50 54,30, p95 56,21, mín 50,30, máx 56,30. **Crecimiento neto del período: −2,66 MiB.** Memoria disponible mínima: 8.048 MiB (68 % de la RAM física). Disco: 842,99–843,01 GB libres, sin deriva.

### 7.3 Tráfico del soak

| Métrica | N | Éxito | Media | σ | p50 | p95 | p99 | Mín | Máx |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Lectura | 344 | 100 % `200` | 19,27 ms | 13,29 ms | 16,96 ms | 34,54 ms | 60,33 ms | 6,83 ms | 193,39 ms |
| Evaluación | 331 | 0 completadas | timeout 60 s | — | — | — | — | — | — |
| Envío | 0 | — | — | — | — | — | — | — | — |
| `429` | 0 | — | — | — | — | — | — | — | — |

Gráfica: `raw/soak_30m.png` (memoria disponible, VmRSS, loadavg y p95 de latencia por minuto).

### 7.4 Lectura

El nodo **no pierde memoria ni filtraciones**: 8,0–8,4 GiB disponibles durante 30 min, RSS del orquestador estable con crecimiento negativo y sin contenedores acumulados (9–16 según el minuto; los de evaluación son efímeros y se retiran). Pero la estabilidad se consigue **sin completar evaluaciones**: de 331 intentos, ninguno terminó dentro de los 60 s del cliente. El sistema no retiene la cola porque el cliente abandona; el daemon libera el trabajo. Es estabilidad por abandono, no por capacidad.

---

## 8. Escenarios de A.1.4 que quedaron pendientes

| Escenario | Estado en v2 | Motivo |
| :--- | :--- | :--- |
| **E5** — construcción en frío y en caliente de imágenes (40 min) | **No ejecutado** | El nodo no expone una superficie de construcción de imágenes: las plantillas usan imágenes precompiladas (`solv-local/stack:<hash>`), tal como ocurría en la v1. Construir 5 imágenes en frío y 5 en caliente excede la ventana de esta campaña y no mide el flujo del alumno. |
| **E8** — apagado por inactividad de 16 a 20 min | **Observado, N = 2** | No se planificó como escenario, pero ocurrió solo durante el soak: el recolector de QoS hibernó los dos IDE a los **15 min 01 s** y **15 min 29 s** de su alta. Un minuto después, el reconciliador registró uno de esos contenedores como `oom_killed` (exit 137). Evidencia: `raw/e8_idle_shutdown_observed.txt`. No se presenta como ensayo controlado de E8. |

---

## 9. Hallazgos que responden a las observaciones

1. **Obs. 8 — aula realista.** La lectura de ejercicios sostiene 35 alumnos (p95 32,76 ms, 100 % `200`). El envío de evaluación no sostiene **10**: 0 de 30 completadas en 60 s, con una latencia real de 112 s medida aparte. Un aula de 35 alumnos no puede evaluar en la práctica.
2. **Obs. 9 — punto de quiebre.** No hay degradación gradual: una evaluación aislada cuesta ~18 s; 10 concurrentes, ~112 s; a 40 VUs ya no completa ninguna. El techo del flujo síncrono está entre **1 y 10 evaluaciones concurrentes**, no en 200. El cuello es el arranque serializado de contenedores (3 por evaluación) más el pre-chequeo Semgrep, no la base de datos ni Traefik.
3. **Obs. 10 — resistencia y ciclo de vida.** 30 min de carga no producen fuga de memoria (RSS 54 ± 1,75 MiB, crecimiento −2,66 MiB) ni deriva de disco. La contraparte es que el soak tampoco completa evaluaciones. El ciclo frío/caliente del IDE es barato (2,6 s / 0,29 s) y el apagado por inactividad aparece a los ~15 min, pero termina con un contenedor marcado `oom_killed`, lo que ensucia la métrica de OOM de la plataforma.
4. **Hallazgo lateral, bloqueante para el flujo real.** Toda evaluación devuelve `RE` porque el runner no encuentra `/runner/solution.py`. La campaña mide el costo del andamiaje (contenedor + Semgrep), no el de ejecutar la solución del alumno. Cualquier cifra de latencia de evaluación debe leerse con esa salvedad.

---

## 10. Limitaciones y validez

1. **Generador y servidor comparten CPU.** k6 corre en el mismo nodo (8 hilos). Las latencias incluyen la competencia del generador; es la misma condición de la v1, y por eso los valores son comparables entre campañas, no con un cliente externo.
2. **Intervención de entorno.** Sin `semgrep` en el contenedor, toda evaluación era `500`. Se instaló `1.186.0` y se declaró en §2.3. La cifra de 18 s por evaluación incluye ese pre-chequeo.
3. **Reentrega habilitada.** Los ejercicios son `purpose = class`. Con `exam`, el segundo envío del mismo par (alumno, ejercicio) devuelve `403 resubmission_not_allowed` y el flujo de envíos se agota.
4. **Frío sin creación de base satélite.** Ver la nota de §6.1: el frío de v2 es el de aula, no el de la primera provisión absoluta de la v1.
5. **Hueco de muestreo en el soak.** Entre `11:39:18Z` y `11:43:01Z` faltan ~3 muestras: el primer muestreador tomaba el RSS del supervisor `air` (6,79 MiB) en lugar del binario Go; se corrigió a `pgrep -x solv-api-dev` y se reanudó sobre el mismo CSV. La muestra del minuto 0 queda marcada como no válida para VmRSS (`vmrss_valid: false` en `raw/metrics_summary.json`) y excluida de la serie.
6. **Entornos IDE del soak.** Fueron 2 y no 3 por rechazo de admisión (`503`), y el recolector de inactividad los hibernó a los ~15 min. Las cifras de memoria de la segunda mitad del soak corresponden a un nodo sin IDE activos.
7. **Energía.** No se midió consumo: el nodo no expone sensores legibles sin root y no hay medidor de enchufe. Igual que en la v1.

---

## 11. Comandos exactos ejecutados

Todos los scripts viven en `scripts/` y se ejecutaron desde la raíz del repositorio.

```bash
# Estado inicial del host
bash docs/CAPACITY/evidencia-campana-v2/scripts/00_host_baseline.sh \
  > docs/CAPACITY/evidencia-campana-v2/raw/host_baseline.txt

# Siembra y tokens
docker exec -i solv_db sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1' \
  < docs/CAPACITY/evidencia-campana-v2/scripts/01_seed_campaign.sql
python3 docs/CAPACITY/evidencia-campana-v2/scripts/02_mint_tokens.py

# Preparación declarada del entorno (semgrep dentro de solv_backend)
docker exec solv_backend sh -lc 'curl -fsSL https://bootstrap.pypa.io/get-pip.py -o /tmp/get-pip.py; \
  python3 /tmp/get-pip.py --break-system-packages -q; \
  python3 -m pip install --break-system-packages --no-cache-dir semgrep==1.186.0'

# E6 frío/caliente (12 + 12, 3 bloques de 4, 30 s entre bloques)
python3 docs/CAPACITY/evidencia-campana-v2/scripts/e6_cold_warm.py

# E1 (3 niveles x 3 réplicas de 60 s, 30 s de enfriamiento)
bash docs/CAPACITY/evidencia-campana-v2/scripts/run_e1.sh

# Saturación 40 -> 60 -> 100 VUs
k6 run --quiet --summary-time-unit ms \
  --summary-trend-stats "avg,min,med,p(50),p(90),p(95),p(99),max,count" \
  --out csv=docs/CAPACITY/evidencia-campana-v2/raw/breakpoint.csv \
  -e SUMMARY_OUT=docs/CAPACITY/evidencia-campana-v2/raw/breakpoint_summary.json \
  docs/CAPACITY/evidencia-campana-v2/scripts/e1_breakpoint.js

# Soak de 30 min (15 VUs + IDE + muestreo cada 60 s)
bash docs/CAPACITY/evidencia-campana-v2/scripts/run_soak.sh

# Calibración con timeout de 240 s (10 VUs, 180 s)
k6 run --quiet --summary-time-unit ms \
  --summary-trend-stats "avg,min,med,p(50),p(90),p(95),p(99),max,count" \
  --out csv=docs/CAPACITY/evidencia-campana-v2/raw/calibration/L10_timeout240.csv \
  -e VUS=10 -e DURATION=180s -e RAMP=10s -e TIMEOUT=240s \
  -e SUMMARY_OUT=docs/CAPACITY/evidencia-campana-v2/raw/calibration/L10_timeout240_summary.json \
  docs/CAPACITY/evidencia-campana-v2/scripts/e1_classroom.js

# Resumen, gráfica y limpieza
python3 docs/CAPACITY/evidencia-campana-v2/scripts/summarize.py
/tmp/mplvenv/bin/python docs/CAPACITY/evidencia-campana-v2/scripts/plot_soak.py
bash docs/CAPACITY/evidencia-campana-v2/scripts/cleanup.sh
```

Cada corrida de k6 dejó su CSV crudo (`--out csv`) y su resumen JSON. Las réplicas de E1 están en `raw/e1/L{nivel}_r{réplica}.csv`.

---

## 12. Limpieza y estado final

`cleanup.sh` eliminó, en este orden: contenedores de los entornos de la campaña, volúmenes de los estudiantes de campaña, y en PostgreSQL las entregas, eventos de teclado, comentarios, calificaciones, recordatorios, casos generados, métricas de ejecución, entornos, matrículas, casos de prueba, ejercicios, materias y usuarios temporales.

Verificación al cierre (`2026-11-27T12:18:34Z`):

| Comprobación | Resultado |
| :--- | :--- |
| Usuarios `campaign-v2-%` | 0 |
| Materias de campaña | 0 |
| Ejercicios de campaña | 0 |
| Entregas de estudiantes de campaña | 0 |
| Entornos de campaña | 0 |
| Volúmenes con identificador de campaña | 0 |
| Contenedores en ejecución | 9 (los 9 del estado inicial) |

`docker ps` final: `solv_traefik`, `solv_frontend`, `solv_backend`, `solv_db`, `solv_dev_db`, `solv_seaweedfs`, `solv-sat-postgres`, y dos contenedores del entorno heredado de la campaña v1 (`solv-environment-b4fb3748-…` y su proxy), que **no** pertenecen a esta campaña y se conservaron. Salida completa en `raw/docker_ps_after_cleanup.txt`.

No se eliminaron volúmenes colgantes preexistentes (188) ni datos de la campaña v1: limpiar eso habría borrado trabajo ajeno a esta corrida.

---

## 13. Inventario

```
docs/CAPACITY/evidencia-campana-v2/
├── MANIFIESTO.md                     este documento
├── SHA256SUMS.txt                    checksums de scripts/ y raw/
├── campaign_run.log                  bitácora con timestamps ISO-8601
├── scripts/
│   ├── 00_host_baseline.sh
│   ├── 01_seed_campaign.sql
│   ├── 02_mint_tokens.py
│   ├── e1_classroom.js
│   ├── e1_breakpoint.js
│   ├── run_e1.sh
│   ├── e6_cold_warm.py
│   ├── soak_classroom.js
│   ├── soak_sampler.py
│   ├── start_ide_envs.sh
│   ├── run_soak.sh
│   ├── summarize.py
│   ├── plot_soak.py
│   ├── cleanup.sh
│   └── campaign_run.sh
└── raw/
    ├── host_baseline.txt
    ├── environment_prep_semgrep.log
    ├── e6_cold.csv · e6_warm.csv · e6_raw.json
    ├── e1/L{10,20,35}_r{1,2,3}.csv · .log · _summary.json
    ├── breakpoint.csv · breakpoint_summary.json · campaign_stdout.log · smoke_read.json
    ├── soak_metrics.csv · soak_k6.csv · soak_k6.log · soak_summary.json
    ├── soak_ide_environments.txt · soak_sampler.log
    ├── calibration/L10_timeout240.csv · _summary.json · .log · single_eval_post_soak.txt
    ├── metrics_summary.json · metrics_summary.txt
    ├── soak_30m.png
    ├── e8_idle_shutdown_observed.txt
    ├── docker_ps_after_cleanup.txt
    └── tokens-campaign.json · token-ide.json
```
