# Inventario de Capacidades de Endurecimiento BaaS — SOLV

Este documento mapea las 24 capacidades de endurecimiento de la plataforma **SOLV**, vinculándolas con su correspondiente Slice, Decisión de Arquitectura (ADR) y estado de implementación.

---

## Mapeo de Capacidades Técnicas (1 a 24)

| # | Capacidad Técnica | Descripción | Slice | ADR | Estado |
| :---: | :--- | :--- | :---: | :---: | :---: |
| **01** | Aislamiento de Red Host vía DOCKER-USER | Inserción de reglas `iptables` en la cadena `DOCKER-USER` para impedir el bypass de firewall de Docker. | Slice 8 | ADR-023 | **COMPLETADO** |
| **02** | Límite de Procesos e Inmunidad a ForkBombs | Restricción `--pids-limit 100` en contenedores para evitar saturación de la tabla de procesos del Kernel. | Slice 9 | - | **PENDIENTE** |
| **03** | Perfiles Seccomp y AppArmor para Ejecución | Aplicación de filtros de llamadas al sistema para mitigar escalada de privilegios en el host. | Slice 9 | - | **PENDIENTE** |
| **04** | Multi-Tenancy Lógico con Discriminador `tenant_id` | Columna `tenant_id` en PostgreSQL y middleware HTTP en Go para aislamiento por inquilino. | Slice 8 | ADR-018 | **COMPLETADO** |
| **05** | Autenticación ForwardAuth HttpOnly en Traefik v3 | Endpoint `GET /api/v1/auth/verify` para validación perimetral de cookies `solv_session` en `iframes` (<50ms). | Slice 8 | ADR-017 | **COMPLETADO** |
| **06** | Unificación del Modelo de Workspaces | Consolidación de laboratorios e IDEs efímeros en la tabla `workspaces` con discriminador `type`. | Slice 8 | ADR-019 | **COMPLETADO** |
| **07** | Sanitización y Truncado de Entradas/Salidas | Límite a 64KB en buffers `stdin`/`stdout` del Juez Virtual para prevenir consumo masivo de memoria. | Slice 9 | - | **PENDIENTE** |
| **08** | Cuotas de Disco por Volumen de Estudiante | Limitación de capacidad de almacenamiento en volúmenes Docker nombrados. | Slice 9 | ADR-001 | **PARCIAL** |
| **09** | Rate Limiting Perimetral en Traefik v3 | Limitación de tasa de solicitudes HTTP por IP y por Tenant en Traefik v3. | Slice 10 | - | **PENDIENTE** |
| **10** | Circuit Breakers y Fail-Fast en Servicios | Detección automática de saturación en Docker Daemon o PostgreSQL. | Slice 10 | - | **PARCIAL** |
| **11** | Cierre Controlado (Graceful Shutdown) | Intercepción de señales `SIGTERM`/`SIGINT` en el backend Go para destruir contenedores efímeros. | Slice 10 | - | **PENDIENTE** |
| **12** | Registro de Auditoría Inmutable (Audit Log) | Tabla de eventos relacionales auditables con sellado de tiempo. | Slice 10 | - | **PENDIENTE** |
| **13** | Pool Adaptativo de Conexiones a Base de Datos | Configuración y tuneo dinámico del pool de conexiones SQL en Go (`sqlx`). | Slice 11 | ADR-000 | **PARCIAL** |
| **14** | Warm Pool de Contenedores de Laboratorio | Mantenimiento de contenedores pre-iniciados en reserva para reducir latencia a <500ms. | Slice 11 | - | **PENDIENTE** |
| **15** | Caché de Imágenes Docker y Pre-pulling | Servicio en segundo plano para precargar imágenes base de laboratorios. | Slice 11 | - | **PENDIENTE** |
| **16** | Control de Admisión Host RAM (15% Headroom) | Rechazo proactivo de creación de laboratorios cuando la memoria libre cae por debajo del 15%. | Slice 5 | ADR-008 | **COMPLETADO** |
| **17** | Hibernación Dual por Inactividad | Hibernación tras 5 minutos de inactividad basada en validación doble (Heartbeat vs Telemetría). | Slice 5 | ADR-011 | **COMPLETADO** |
| **18** | Penalización 3-Strikes OOMKilled | Cooldown de 5 minutos al estudiante cuyo entorno sufra 3 eliminaciones consecutivas por RAM. | Slice 5 | ADR-008 | **COMPLETADO** |
| **19** | Auto-Tuning EWMA de Recursos | Suavizado exponencial del perfil de consumo de RAM por plantilla de laboratorio. | Slice 6 | ADR-014 | **PENDIENTE** |
| **20** | Blindaje Inter-Contenedor (ICC Disabled) | Aislamiento de red mediante la creación de la red Docker con `enable_icc=false`. | Slice 6 | ADR-009 | **PARCIAL** |
| **21** | Telemetría Prometheus `/metrics` | Exposición de métricas clave de salud del host y contenedores activos en `/metrics`. | Slice 6 | - | **COMPLETADO** |
| **22** | Análisis Semántico AST Inmutable (Semgrep) | Ejecución de Semgrep en modo `:ro` sobre el volumen del alumno, persistiendo en `semgrep_audit JSONB`. | Slice 7 | ADR-016 | **PARCIAL** |
| **23** | Renovación Automática TLS Wildcard DNS-01 | Desafío ACME DNS-01 con desec.io y Let's Encrypt para certificados SSL `*.solv.uab.edu.bo`. | Slice 10 | ADR-018 | **PENDIENTE** |
| **24** | Subdominios Dinámicos Seguros vía UUID Opaco | Generación de URLs de acceso basadas en UUIDv4 opacos. | Slice 4 | ADR-004 | **PENDIENTE** |
