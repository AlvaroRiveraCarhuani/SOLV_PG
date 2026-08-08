# Matriz de Versiones Fijadas de Producción (CRIT-08)

Queda estrictamente prohibido el uso de la etiqueta `:latest` en cualquier servicio o runtime de producción.

| Componente / Servicio | Versión Fijada | Registro / Imagen Docker | Propósito |
|-----------------------|----------------|--------------------------|-----------|
| **OpenVSCode Server** | `1.96.0` | `gitpod/openvscode-server:1.96.0` | Entorno IDE persistente para estudiantes |
| **Semgrep CLI** | `1.100.0` | `semgrep/semgrep:1.100.0` | Motor de pre-chequeo y auditoría de AST |
| **Traefik Reverse Proxy** | `v3.1.2` | `traefik:v3.1.2` | Reverse proxy y terminación ACME TLS |
| **PostgreSQL Database** | `18-alpine` | `postgres:18-alpine` | Base de datos relacional principal |
| **Prometheus** | `v2.51.1` | `prom/prometheus:v2.51.1` | Recolección de métricas de telemetría |
| **Grafana** | `10.4.1` | `grafana/grafana:10.4.1` | Tableros de monitoreo de operaciones |
