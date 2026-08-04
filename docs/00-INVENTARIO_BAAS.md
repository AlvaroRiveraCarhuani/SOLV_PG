# Inventario de Requerimientos Críticos (CRITs) y Decisiones de Arquitectura — SOLV BaaS

Este documento detalla el estado actual de los 24 Requerimientos Críticos (CRIT-01 a CRIT-24) de la arquitectura BaaS de **SOLV**, así como las 6 Decisiones Inamovibles de Arquitectura (D1–D6) establecidas para el proyecto.

---

## 1. Decisiones Inamovibles de Arquitectura (D1 – D6)

> [!IMPORTANT]
> **Reglas de Oro del Sistema:**
>
> 1. **D1 — Autenticación Perimetral ForwardAuth:** Autenticación de subdominios de laboratorio mediante cookie HttpOnly de dominio (`solv_session`) y middleware ForwardAuth en Traefik v3 (`GET /api/v1/auth/verify`). Queda prohibido exponer tokens JWT en la URL.
> 2. **D2 — Multi-Tenancy Lógico por Discriminador `tenant_id`:** Aislamiento multi-tenant a nivel de base de datos mediante la columna `tenant_id` en todas las tablas transaccionales y middleware de contexto en Go, compartiendo un esquema relacional unificado.
> 3. **D3 — Unificación del Dominio de Workspaces:** Consolidación de laboratorios e IDEs efímeros/persistentes en la tabla única `workspaces` con discriminador `type ENUM('IDE_PERSISTENTE', 'JUEZ_EFIMERO')`. Eliminación completa de la tabla legacy `lab_instances`.
> 4. **D4 — Aislamiento de Red Host vía Cadena `DOCKER-USER`:** Protección del bridge de Docker mediante reglas de `iptables` en la cadena `DOCKER-USER`, enlazando servicios internos a `127.0.0.1` e impidiendo acceso directo no autorizado desde redes externas al host.
> 5. **D5 — Parametrización Total de Entorno (Cero Hardcode):** Prohibición estricta de hardcodear dominios institucionales o nombres de organización en el código fuente. Toda la configuración de dominios se canaliza a través de variables de entorno (`COOKIE_DOMAIN`, `DESEC_TOKEN`, etc.).
> 6. **D6 — Auditoría Semántica AST Inmutable:** El worker de análisis estático (`SemgrepWorker`) debe ejecutar contenedores efímeros montando el volumen del estudiante de forma inmutable en **Solo Lectura (`/src:ro`)**, persistiendo el resultado en PostgreSQL (`semgrep_audit JSONB`).

---

## 2. Inventario de Requerimientos Críticos (CRIT-01 a CRIT-24)

### Estado General

* **Completados (8):** CRIT-01, CRIT-04, CRIT-05, CRIT-06, CRIT-16, CRIT-17, CRIT-18, CRIT-21
* **Parciales (5):** CRIT-08, CRIT-10, CRIT-13, CRIT-20, CRIT-22
* **Pendientes (11):** CRIT-02, CRIT-03, CRIT-07, CRIT-09, CRIT-11, CRIT-12, CRIT-14, CRIT-15, CRIT-19, CRIT-23, CRIT-24

---

### Detalle Individual de Requerimientos Críticos

| Código | Requerimiento Crítico | Descripción Técnica | Estado Actual | Slice Asociado |
| :--- | :--- | :--- | :---: | :---: |
| **CRIT-01** | Aislamiento de Red Host via DOCKER-USER | Inserción de reglas `iptables` en la cadena `DOCKER-USER` para impedir el bypass por defecto de Docker y restringir la exposición pública de contenedores de laboratorio. | **COMPLETADO** | Slice 8 |
| **CRIT-02** | Límite de Procesos e Inmunidad a ForkBombs | Restricción `--pids-limit 100` en contenedores efímeros y persistentes para evitar denegación de servicio en el Kernel anfitrión. | **PENDIENTE** | Slice 9 |
| **CRIT-03** | Perfiles Seccomp y AppArmor para Ejecución | Aplicación de perfiles de llamadas al sistema (Seccomp) para restringir invocaciones a `ptrace`, `sys_admin` y syscalls de alto riesgo. | **PENDIENTE** | Slice 9 |
| **CRIT-04** | Multi-Tenancy Lógico con Discriminador `tenant_id` | Columna `tenant_id UUID` en tablas de Postgres, inyectada y validada mediante middleware HTTP en Go a partir de claims JWT. | **COMPLETADO** | Slice 8 |
| **CRIT-05** | Autenticación ForwardAuth HttpOnly en Traefik v3 | Endpoint `GET /api/v1/auth/verify` en Go para validación perimetral de cookies HttpOnly `solv_session` en subdominios de iframes (<50ms). | **COMPLETADO** | Slice 8 |
| **CRIT-06** | Unificación de Tablas de Workspaces | Consolidación de `workspaces` y `lab_instances` en una sola entidad relacional con discriminador `type ('IDE_PERSISTENTE' / 'JUEZ_EFIMERO')`. | **COMPLETADO** | Slice 8 |
| **CRIT-07** | Sanitización y Truncado de Entradas/Salidas | Limitación estricta a 64KB en buffers `stdin`/`stdout` del Juez Virtual para prevenir ataques por consumo excesivo de memoria en logs. | **PENDIENTE** | Slice 9 |
| **CRIT-08** | Cuotas de Disco por Volumen de Estudiante | Limitación de capacidad de almacenamiento en volúmenes Docker (XFS/ext4 quota) para impedir saturación del disco anfitrión. | **PARCIAL** | Slice 9 |
| **CRIT-09** | Rate Limiting Perimetral en Traefik v3 | Limitación de tasa de solicitudes HTTP por IP y por Tenant en Traefik para mitigar ataques por fuerza bruta o inundación. | **PENDIENTE** | Slice 10 |
| **CRIT-10** | Circuit Breakers y Fail-Fast en Servicios | Detección automática de saturación en Docker Daemon o PostgreSQL, fallando rápidamente antes de acumular peticiones en cola. | **PARCIAL** | Slice 10 |
| **CRIT-11** | Cierre Controlado (Graceful Shutdown) | Intercepción de señales `SIGTERM`/`SIGINT` en el backend Go para destruir contenedores efímeros y cerrar conexiones antes de salir. | **PENDIENTE** | Slice 10 |
| **CRIT-12** | Registro de Auditoría Immutable (Audit Log) | Tabla de eventos relacionales auditables (creación, acceso, errores de seguridad) con sellado de tiempo para cumplimiento administrativo. | **PENDIENTE** | Slice 10 |
| **CRIT-13** | Pool Adaptativo de Conexiones a Base de Datos | Configuración y tuneo dinámico del pool de conexiones SQL (`SetMaxOpenConns`, `SetMaxIdleConns`) según la carga útil. | **PARCIAL** | Slice 11 |
| **CRIT-14** | Warm Pool de Contenedores de Laboratorio | Mantenimiento de una reserva de contenedores pre-iniciados en estado tibio para reducir la latencia de arranque a <500ms. | **PENDIENTE** | Slice 11 |
| **CRIT-15** | Caché de Imágenes Docker y Pre-pulling | Servicio en segundo plano para precargar imágenes base de laboratorios en el daemon local antes del inicio de clases. | **PENDIENTE** | Slice 11 |
| **CRIT-16** | Control de Admisión Host RAM (15% Headroom) | Rechazo proactivo de creación de laboratorios cuando la memoria libre del host cae por debajo del 15% para evitar Kernel Panic. | **COMPLETADO** | Slice 5 |
| **CRIT-17** | Hibernación Dual por Inactividad | Hibernación de entornos tras 5 minutos de inactividad basada en validación doble (Intención HTTP Heartbeat vs Realidad Telemetría). | **COMPLETADO** | Slice 5 |
| **CRIT-18** | Penalización 3-Strikes OOMKilled | Bloqueo temporal (cooldown de 5 min) al estudiante cuyo entorno sufra 3 eliminaciones consecutivas por exceso de memoria RAM. | **COMPLETADO** | Slice 5 |
| **CRIT-19** | Auto-Tuning EWMA de Recursos | Suavizado exponencial del perfil de consumo de memoria RAM por plantilla de laboratorio para ajustar cuotas recomendadas. | **PENDIENTE** | Slice 6 |
| **CRIT-20** | Blindaje Inter-Contenedor (ICC Disabled) | Aislamiento de red mediante la creación de la red Docker con opción `com.docker.network.bridge.enable_icc=false`. | **PARCIAL** | Slice 6 |
| **CRIT-21** | Telemetría Prometheus `/metrics` | Exposición de métricas clave de salud del host, memoria consumida, contenedores activos y veredictos en `/metrics`. | **COMPLETADO** | Slice 6 |
| **CRIT-22** | Análisis Semántico AST Inmutable (Semgrep) | Ejecución de Semgrep en modo `:ro` sobre el volumen del alumno, inyectando la auditoría semántica en la columna `semgrep_audit JSONB`. | **PARCIAL** | Slice 7 |
| **CRIT-23** | Renovación Automática TLS Wildcard DNS-01 | Automatización del protocolo ACME DNS-01 con desec.io y Let's Encrypt para certificados SSL `*.solv.uab.edu.bo`. | **PENDIENTE** | Slice 10 |
| **CRIT-24** | Subdominios Dinámicos Seguros vía UUID Opaco | Generación de URLs de acceso basadas en UUIDv4 opacos en lugar de nombres legibles para evitar ataques de enumeración. | **PENDIENTE** | Slice 4 |
