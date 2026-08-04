# Manual Operativo - SLICE 6: Persistencia Relacional, EWMA y Telemetría

## 1. Verificación de Migraciones de Base de Datos en PostgreSQL

Al iniciar el servidor backend en Go, las migraciones iniciales se ejecutan automáticamente creando la tabla `lab_template_profiles`:

```bash
# Verificar conexión a PostgreSQL y la existencia de la tabla de perfiles
psql -h 127.0.0.1 -U solv_user -d solv_db -c "\d lab_template_profiles"
```

---

## 2. Configuración de la Red Aislada en Docker (`solv-traefik-net`)

Verificar que la red puente tenga deshabilitado el tráfico inter-contenedor (`enable_icc=false`):

```bash
# Inspeccionar opciones de la red en Docker Engine
docker network inspect solv-traefik-net --format '{{index .Options "com.docker.network.bridge.enable_icc"}}'

# Salida esperada:
# false
```

---

## 3. Monitoreo y Raspado de Telemetría Prometheus (`GET /metrics`)

Probar la consulta del endpoint de observabilidad localmente:

```bash
curl -i http://localhost:3000/metrics

# Respuesta esperada (HTTP 200 OK con cabecera Prometheus):
# Content-Type: text/plain; version=0.0.4; charset=utf-8
# solv_active_workspaces_total 0
# solv_host_available_memory_bytes 2058354688
# solv_host_oom_guard_bytes 1476395008
# solv_orphan_containers_reclaimed_total 0
```

---

## 4. Configuración del Archivo `prometheus.yml` (Para Grafana)

Agregar el target de SOLV en la configuración de Prometheus:

```yaml
scrape_configs:
  - job_name: 'solv-backend'
    scrape_interval: 5s
    static_configs:
      - targets: ['localhost:3000']
```
