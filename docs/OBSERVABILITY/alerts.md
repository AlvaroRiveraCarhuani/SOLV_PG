# Reglas de Alerta Operativa para SOLV Judge (Prometheus)

Este documento define el catálogo estándar de alertas Prometheus recomendadas para la operación en producción del Juez Virtual SOLV.

---

## 1. HighSubmissionFailureRate

* **Nombre de alerta:** `HighSubmissionFailureRate`
* **Severidad:** `warning`
* **Expresión PromQL:**
  ```promql
  sum(rate(solv_submissions_total{verdict=~"WA|RE|TLE"}[5m]))
  /
  sum(rate(solv_submissions_total[5m])) > 0.50
  ```
* **Duración (`for`):** `5m`
* **Descripción:** Se dispara cuando más del 50% de los envíos evaluados en los últimos 5 minutos resultan en fallos (Wrong Answer, Runtime Error o Time Limit Exceeded).
* **Diagnóstico & Mitigación:**
  1. Verificar si un examen o tarea masiva contiene casos de prueba con especificación rota o discrepancias en el evaluador.
  2. Revisar si hay un problema en la imagen del runner que esté provocando Runtime Errors masivos.
  3. Comprobar los logs estructurados JSON filtrando por `verdict="RE"` y `verdict="TLE"`.

---

## 2. DockerContainerSaturation

* **Nombre de alerta:** `DockerContainerSaturation`
* **Severidad:** `critical`
* **Expresión PromQL:**
  ```promql
  (solv_docker_containers_active / 64) > 0.80
  ```
  *(Considerando una capacidad máxima de referencia de 64 contenedores concurrentes por nodo)*
* **Duración (`for`):** `1m`
* **Descripción:** Se dispara cuando la cantidad de contenedores Docker efímeros concurrentes supera el 80% de la capacidad de concurrencia asignada al nodo.
* **Diagnóstico & Mitigación:**
  1. Inspeccionar si hay contenedores zombis acumulados no liberados por el runtime.
  2. Verificar si el worker de limpieza periódica de contenedores (`ZombieCollectorWorker`) está en ejecución.
  3. Revisar el estado de memoria RAM y CPU del nodo servidor con `docker ps` y `htop`.

---

## 3. EvaluatorQueueBacklog

* **Nombre de alerta:** `EvaluatorQueueBacklog`
* **Severidad:** `warning`
* **Expresión PromQL:**
  ```promql
  solv_evaluator_queue_depth > 50
  ```
* **Duración (`for`):** `2m`
* **Descripción:** Se dispara cuando la cola de evaluaciones pendientes del juez supera los 50 envíos en espera durante más de 2 minutos consecutivos.
* **Diagnóstico & Mitigación:**
  1. Revisar si hay entregas con timeouts altos monopolizando la capacidad de ejecución.
  2. Ajustar la concurrencia de workers en el pool de evaluación si los recursos del servidor lo permiten.
  3. Verificar si el daemon de Docker está respondiendo con lentitud a las operaciones de creación y arranque.

---

## 4. HighHTTPLatency

* **Nombre de alerta:** `HighHTTPLatency`
* **Severidad:** `warning`
* **Expresión PromQL:**
  ```promql
  histogram_quantile(0.95, sum(rate(solv_http_request_duration_seconds_bucket[5m])) by (le)) > 5.0
  ```
* **Duración (`for`):** `3m`
* **Descripción:** Se dispara cuando el percentil 95 (P95) de la latencia de las peticiones HTTP a la API supera los 5 segundos durante una ventana de 3 minutos.
* **Diagnóstico & Mitigación:**
  1. Identificar rutas lentas agrupando por etiqueta `path` en `solv_http_request_duration_seconds`.
  2. Verificar si hay contención de bloqueos en PostgreSQL o consultas no indexadas.
  3. Comprobar si el middleware de observabilidad reporta tiempos anómalos en los logs JSON de petición.
