# Plan de benchmarking de SOLV sobre `26e109c`

**Fase:** PASO SYNC-2 (auditoría de solo lectura sobre el código).
**Fecha:** 2026-11-24.
**Nodo objetivo:** Asus i7-6700HQ, 12 GB DDR4, SSD 128 GB + HDD 1 TB, Ubuntu Server 24.04 LTS.
**Estado:** este documento no reporta ninguna medición nueva. Todo lo que figura como `NO MEDIDO` sigue sin evidencia y solo se declara el escenario que lo mediría. No se ejecutó ningún benchmark durante SYNC-2.

**Regla de evidencia (D7).** Cada fila cita archivo y línea del repositorio en `26e109c` o de la tesis. Una cifra sin registro se escribe `NO MEDIDO` y nunca se presenta como resultado.

---

## 1. Mapa de afirmaciones cuantitativas frente a evidencia

| # | Afirmación del manuscrito | Dónde se afirma | Evidencia hoy en `26e109c` | Decisión | Escenario |
|:-:|:---|:---|:---|:---|:---:|
| 1 | RSS del orquestador inferior a 50 MB | `05a:69`, `05d` (RNF-02), `04:253` | `NO MEDIDO`: no hay muestreo ni salida guardada | medir en el Asus y publicar `μ`, `σ`, máximo | E2 |
| 2 | `ForwardAuth` por debajo de 50 ms | `04:189`, `slice08_forwardauth_test.go:125-129` | cota verificada por prueba de integración que falla si se supera; no hay distribución | medir distribución y conservar la cota como aserción | E3 |
| 3 | `session_revocations` se consulta cada 2 s y cierra el canal | `05b` escenario 1 | implementado y probado; latencia de cierre no medida | medir tiempo entre revocación y cierre del WebSocket | E3 |
| 4 | Pre-chequeo en tres capas por debajo de 100 ms | `01:7`, `04:183`, `05a:55`, `05b:207` | `slice09_semgrep_precheck_test.go:171-175` solo hace `t.Logf`; no falla ni promedia | medir con `go test -bench` y con la ruta HTTP real | E4 |
| 5 | Reutilización de imagen por SHA-256 (`solv-local/stack:<sha256>`) y costo del *cold build* | `05b:22` | `NO MEDIDO`: no existe script ni salida de `docker image inspect` o `docker build` | medir *cold* y *warm* por separado | E5 |
| 6 | Provisión de entorno con `POST /api/v1/environments/start` | `05d` RF-2.1, RF-2.3 | `NO MEDIDO` como latencia; las pruebas verifican que el contenedor arranca | medir `n = 30` y reportar `μ`, `σ`, p95 | E6 |
| 7 | Reanudación tras hibernación en menos de 500 ms | `05d` RF-2.3 | `NO MEDIDO`; no hay aserción de tiempo | medir con `curl` sobre el entorno pausado | E6 |
| 8 | Capacidad de 15 a 24 entornos IDE y 35 a 40 estudiantes en 12 GB | `04:229`, `04:283` | inferencia aritmética de la Ecuación (5), no una corrida | convertir en experimento hasta el rechazo por memoria | E7 |
| 9 | Apagado automático de motores gestionados al quedar sin laboratorios | `05d` RF-2.2 (ADR-050, ADR-051) | prueba unitaria de la decisión; tiempo real de parada no medido | medir intervalo entre último cierre y contenedor detenido | E8 |
| 10 | Contención de OOM en `cgroups v2` con `oom_strike_count` y sin degradar vecinos | `05h` PRU-OE6-04 | prueba de estrés que verifica el contador; impacto en vecinos no medido | medir latencia del vecino durante el OOM | E9 |
| 11 | Descarte lateral del 100 % en la cadena `DOCKER-USER` | `05h` PRU-OE6-01 | aserción de integración; requiere daemon e `iptables` reales | repetir en el Asus y registrar porcentaje de paquetes descartados | E10 |
| 12 | Aislamiento de salida a internet en exámenes (ADR-085) | `05f` RF-4.6, `04:185` | `docker_exam_network_e2e_test.go` salta sin `EXAM_NETWORK_E2E=1` | ejecutar la prueba real y una sonda de salida a internet | E10 |
| 13 | Precisión de `ComplexityService` al clasificar `O(f(N))` | `05a`, `05b` | el servicio ajusta modelos con `R² ≥ 0,3` y desviación menor a 2 ms, pero no hay matriz de aciertos | medir aciertos sobre algoritmos de complejidad conocida | E11 |
| 14 | Similitud por k-gramas (k = 4) con resta de código base | `05a:59`, ADR-066 | umbrales y mecánica probados; precisión y exhaustividad no medidas | medir sobre un corpus con plagio inyectado y pares limpios | E12 |
| 15 | Integridad SHA-256 de avances y artefactos en SeaweedFS 4.51 | `05a:31`, `05d` RF-2.8 | pruebas unitarias de empaquetado y huella; verificación de descarga no medida | comprobar que un objeto alterado se rechaza | E13 |
| 16 | Consumo eléctrico de 65 W y 10,4 / 46,8 kWh al mes | `04:321`, `04:325`, `04:344` | `NO MEDIDO`: no existe instrumento ni registro | rebajar a estimación declarada en D1-B, salvo que se consiga medidor | sin escenario |
| 17 | P95 global de 217,59 ms y 79,73 req/s a 200 usuarios | `docs/CAPACITY/load-test-report.md` §3 | medido en i7-4600U con 7,6 GiB y Fedora 43, no en el nodo piloto | repetir en el Asus para sostener la validez externa | E1 |

**Sobre la fila 16.** No es un benchmark pendiente: es una afirmación que hoy no se puede sostener con rigor. Si el Asus no tiene medidor de enchufe, D1-B debe escribirla como estimación con su base de cálculo (potencia nominal por horas de operación) y quitar la palabra «medido».

**Sobre la fila 17.** Los percentiles del informe se calcularon con un script que en `26e109c` ya no es ejecutable sin cambios (§2, trampas 2 y 3). Repetir la corrida exige corregir el script, no solo lanzarlo.

---

## 2. Validación técnica de los escenarios contra `26e109c`

1. **Cupo global por IP (ADR-089, `rate_limiter.go:59-77`).** El valor por defecto del binario es 600 peticiones por minuto con ráfaga 120 (perfil B). Todo el tráfico de `k6` sale por `127.0.0.1`, es decir, un único cubo. Una corrida de 200 usuarios virtuales con tres peticiones por iteración y pausa de 2 a 5 s ronda las 170 peticiones por segundo, muy por encima de las 100 que permite el perfil A (6000/500). Sin subir el cupo, el benchmark mide el limitador y no el sistema. Para la corrida hay que exportar `RATE_LIMIT_IP_PER_MINUTE` y `RATE_LIMIT_IP_BURST` con holgura y declarar en el informe que se usó un cupo de laboratorio, no el perfil de despliegue.

2. **Límite de envíos por alumno (`rate_limiter.go:103`).** `SubmissionRateLimitMiddleware` permite 10 envíos por minuto y por estudiante. El script actual hace un envío por iteración con pausa de 2 a 5 s, o sea entre 12 y 30 envíos por minuto para el mismo identificador, así que una parte de las respuestas será 429. Hay que subir la pausa a 6 s o más, o separar el escenario de lectura del de escritura.

3. **Recibo de evaluación obligatorio (ADR-066, `submission_handler.go:68`).** Sin `evaluation_receipt` la entrega responde 422 `invalid_evaluation_receipt`. El script de carga actual envía un veredicto inventado en el cuerpo y no pide evaluación previa, de modo que en `26e109c` fallaría en el segundo paso. El camino válido es `POST /api/v1/evaluations` y después `POST /api/v1/submissions` con el recibo devuelto. Cualquier comparación con el informe anterior tiene que declarar este cambio de contrato.

4. **Identidad simulada contra el camino real.** El script envía `X-User-Id`, `X-Tenant-Id` y `X-User-Role` directo al puerto 3000. Eso mide el backend sin Traefik y sin `ForwardAuth`. Para el escenario de autenticación hay que entrar por Traefik en el puerto 80 con la cookie real, porque los encabezados inyectados por el proxy no se producen en el camino directo.

5. **Cookie `solv_session` de prueba.** `scripts/gen-token.sh` ejecuta `go run ./cmd/gentoken/main.go -role=... -email=... -days=... -raw`, imprime el JWT y arma la línea de consola `document.cookie = "solv_session=...; path=/"`. Para `curl` el equivalente es `-H "Cookie: solv_session=$TOKEN"` o `-H "Authorization: Bearer $TOKEN"`, que `VerifyAuth` también acepta (`auth_handler.go:187-193`). El token exige un usuario existente en la base: si se genera para un correo que no está sembrado, la firma es válida pero el flujo posterior falla.

6. **Semgrep y las tres capas.** `findSemgrepBinary()` (`semgrep_worker.go:164-173`) busca primero `$HOME/.local/bin/semgrep` y después el `PATH`; el escaneo de volúmenes usa la imagen `semgrep/semgrep` en solo lectura. Si el binario no está, el escaneo degrada en abierto (`semgrep_worker.go:76-82`) y la medición del pre-chequeo se hace con dos capas, no con tres. Antes de E4 hay que confirmar `semgrep --version` y que el archivo de reglas exista.

7. **Red de examen.** El nombre real es `solv-exam-net` con `Internal=true` y los motores del examen viven en `solv-exam-satellites-net` (ADR-085). `exam_network` es el nombre de la columna de `exercises`, no el de la red. La prueba `TestExamNetwork_E2E_Isolation` se salta salvo que se defina `EXAM_NETWORK_E2E=1`, y la red solo existe si el orquestador ya la creó. La Figura 3.6 se corrigió en SYNC-2 para mostrar `solv-exam-net`.

8. **Métricas internas.** `GET /metrics` existe (`router.go:608`) y expone contadores con `promhttp`, pero si `METRICS_TOKEN` está definido exige `Authorization: Bearer` y, si no lo está, solo acepta loopback (`metrics_access.go:14-34`). El informe anterior atribuye la telemetría a Prometheus; en `compose.yml` no hay Prometheus ni Grafana, así que la captura se hace con `curl` directo o con `docker stats`, y así debe redactarse.

9. **Costo de disco y cuota blanda.** Cada entorno reserva hasta 1024 MB de cuota blanda (ADR-084) y el alta se rechaza por debajo del 5 % de disco libre. Una corrida de capacidad con imágenes `openvscode-server` y varios motores de base de datos puede consumir más de 20 GB en el SSD de 128 GB. Hay que limpiar entre escenarios (`docker system prune`, volúmenes de prueba) y registrar `df -h` antes y después.

10. **Datos sembrados.** `load-tests/seed_load_test_data.sql` crea 200 estudiantes con identificadores `00000000-0000-0000-0001-<12 dígitos>`, que es exactamente la convención que espera `exam-deadline-peak.js` (`vuIndex`). El tenant es `00000000-0000-0000-0000-000000000001` y hay 10 ejercicios. Si se cambia el número de usuarios virtuales por encima de 200, el script repite identificadores y mide contención artificial sobre el mismo alumno.

11. **Hibernación y reanudación.** El umbral son 15 minutos de inactividad sobre `last_heartbeat_at` (`05d` RF-2.3). Medir la reanudación en menos de 500 ms no exige esperar la hibernación real: alcanza con pausar el contenedor por la API y cronometrar el `start`. Si se quiere medir el ciclo completo, el escenario dura al menos 16 minutos y hay que decirlo en el informe.

12. **`DOCKER-USER` y privilegios.** La cadena se inspecciona con `iptables -S DOCKER-USER` y las reglas se derivan de la etiqueta `solv.exam.no-egress=true` mediante `infra/firewall/docker-user-rules.sh`. Requiere root en el Asus y un daemon que aplique las reglas al arrancar; sin eso, el escenario 10 mide una configuración distinta de la documentada.

---

## 3. Scripts y activos reutilizables

| Activo | Ruta | Qué aporta | Qué falta antes de usarlo |
|:---|:---|:---|:---|
| Ensayo de deadline | `load-tests/exam-deadline-peak.js` | etapas 50/100/200 usuarios, umbrales p95 y tasa de error | recibo de evaluación, cupo por IP, pausa de 6 s y camino por Traefik |
| Siembra de datos | `load-tests/seed_load_test_data.sql` | tenant, materia, 10 ejercicios y 200 estudiantes | nada, se ejecuta con `psql` sobre `solv_db` |
| Capacidad de IDEs | `scripts/capacity-check.sh` | arranque por contenedor en ms, memoria real, piso sobre RAM | corrida con `--count` alto y limpieza posterior |
| Token de sesión | `scripts/gen-token.sh` | JWT real y cookie `solv_session` | `cmd/gentoken` compilado y usuario sembrado |
| Bench de impacto | `tools/test-impact/bench.mjs` | pesos orientativos por suite para planificar corridas | son estimaciones de segundos, no mediciones del nodo |
| ForwardAuth | `backend/tests/integration/slice08_forwardauth_test.go` | aserción activa de 50 ms | complementar con distribución p95 y p99 |
| Pre-chequeo | `backend/tests/integration/slice09_semgrep_precheck_test.go` | objetivo de 100 ms con `t.Logf` | convertir en benchmark con `-bench` y `benchstat` |
| Red de examen | `backend/internal/infrastructure/docker/docker_exam_network_e2e_test.go` | aislamiento real con `EXAM_NETWORK_E2E=1` | daemon, `iptables` y red creada |
| Línea base de carga | `docs/CAPACITY/load-test-report.md` | percentiles del i7-4600U para comparar | declarar el cambio de equipo y de contrato |
| Métricas | `GET /metrics` | contadores HTTP y de Docker | `METRICS_TOKEN` o loopback |

---

## 4. Plan cerrado para el Asus

**Prerrequisitos antes de E1.** El Asus debe estar en `git pull` con `26e109c` (`git rev-parse --short HEAD`), con `.env` completo, `make up` sano, `docker info` operativo, `semgrep --version` disponible, al menos 40 GB libres en el SSD y `METRICS_TOKEN` definido si se van a capturar métricas. El ensayo se corre sin usuarios reales conectados y con el perfil de cupo de laboratorio declarado.

| Esc. | Objetivo | Comando base | Tiempo | Disco | Requisitos |
|:---:|:---|:---|:---:|:---:|:---|
| E0 | Preparación, siembra y línea base | `psql -f load-tests/seed_load_test_data.sql`, `df -h`, `docker info` | 20 min | 2 GB | `26e109c`, `.env`, Docker |
| E1 | Repetir carga de 200 usuarios en el Asus | `k6 run load-tests/exam-deadline-peak.js` corregido | 30 min | 3 GB | recibo, cupo, Traefik |
| E2 | RSS del orquestador en reposo y en carga | `ps -o rss= -p $(pgrep -f solv-api)` muestreado 60 s | 5 min | 0 | backend arriba |
| E3 | Distribución de `ForwardAuth` y cierre por revocación | `k6` o `hyperfine` con cookie real contra `/api/v1/auth/verify` | 15 min | 0 | token, usuario |
| E4 | Pre-chequeo en tres capas | `go test -bench . -benchtime 200x -count 10` y ruta HTTP | 15 min | 0 | Semgrep presente |
| E5 | *Cold build* y *warm hit* por SHA-256 | `docker image rm`, `time docker build`, `hyperfine 'docker image inspect'` | 40 min | 5 GB | imágenes base descargadas |
| E6 | Provisión y reanudación | 30 × `POST /environments/start` y `start` tras pausa | 30 min | 8 GB | cupo de memoria |
| E7 | Capacidad de IDEs hasta el rechazo | `scripts/capacity-check.sh --count 24 --wave 0` | 45 min | 12 GB | imagen local |
| E8 | Apagado de motores gestionados | abrir y cerrar el último laboratorio, cronometrar la parada | 20 min | 3 GB | Docker |
| E9 | OOM en `cgroups v2` | contenedor infractor más vecino midiendo latencia | 25 min | 2 GB | límites ADR-084 |
| E10 | Aislamiento lateral y de salida | `EXAM_NETWORK_E2E=1 go test ./internal/infrastructure/docker` y sonda de red | 25 min | 1 GB | root, `iptables` |
| E11 | Precisión de `ComplexityService` | 10 algoritmos de complejidad conocida por la ruta de evaluación | 20 min | 0 | backend arriba |
| E12 | Similitud con y sin código base | pares sintéticos con porcentaje conocido de copia | 25 min | 1 GB | corpus preparado |
| E13 | Integridad SHA-256 en SeaweedFS | subir, alterar y descargar un avance y un artefacto | 15 min | 1 GB | SeaweedFS arriba |

**Tiempo total estimado:** 5 h 45 min de corrida más 30 min de limpieza y captura. **Pico de disco:** 12 GB en E7 más 5 GB de imágenes en E5, o sea alrededor de 25 GB de trabajo sobre el SSD de 128 GB, con la regla del 5 % libre de ADR-084 como límite duro.

**Orden mínimo si hay poco tiempo:** E0, E1, E3, E4, E6, E7. Esos seis escenarios cubren las afirmaciones que hoy usa el artículo (latencia, autenticación, pre-chequeo, provisión y capacidad) y dejan E9 a E13 como evidencia de aislamiento y autoría para la defensa.

**Cierre esperado.** Cada escenario produce una fila con `μ`, `σ`, p95, p99, `n` y método, o la palabra `NO MEDIDO`. El informe que salga de esta campaña reemplaza los percentiles del i7-4600U y alimenta la Tabla 1.2 de D1-B.
