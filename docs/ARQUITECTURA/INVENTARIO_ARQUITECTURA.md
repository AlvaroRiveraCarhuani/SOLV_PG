# Inventario Técnico de Arquitectura del Repositorio SOLV

> **Documento Oficial de Auditoría Técnica de Arquitectura**  
> **Proyecto:** SOLV (Sistema de Orquestación de Laboratorios Virtuales)  
> **Repositorio:** SOLV_PG  
> **Auditor:** Agente IA Antigravity  
> **Fecha de Inspección:** 2026-08-19  

---

## 1. Árbol del Repositorio (Nivel 3)

```text
SOLV_PG/
├── backend/
│   ├── cmd/
│   │   ├── api/
│   │   └── autotune/
│   ├── internal/
│   │   ├── core/
│   │   ├── delivery/
│   │   └── infrastructure/
│   ├── tests/
│   │   └── integration/
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── app/
│   │   ├── assets/
│   │   ├── index.html
│   │   └── main.ts
│   ├── angular.json
│   ├── package.json
│   └── tsconfig.json
├── infra/
│   ├── desec/
│   │   └── setup_dns.sh
│   ├── firewall/
│   │   └── docker-user-rules.sh
│   ├── scripts/
│   │   └── backup.sh
│   └── traefik/
│       ├── dynamic_conf.yml
│       └── traefik.yml
├── docs/
│   ├── API/
│   │   └── CONTRATOS.md
│   ├── ARQUITECTURA/
│   │   ├── ADR/
│   │   ├── CONCURRENCIA.md
│   │   ├── SEGURIDAD.md
│   │   └── STORAGE_STRATEGY.md
│   ├── BD/
│   │   └── DISENO_BD.md
│   ├── GOBERNANZA/
│   │   ├── CONVENCIONES.md
│   │   ├── INVENTARIO_BAAS.md
│   │   └── METODOLOGIA.md
│   ├── SLICES/
│   │   ├── SLICE_1/ ... SLICE_11/
│   ├── 00-INVENTARIO_BAAS.md
│   ├── 00-MAPA_SLICES.md
│   ├── REQUERIMIENTOS_SOLV.md
│   ├── SECURITY.md
│   ├── SECURITY_CHECKLIST.md
│   └── VERSIONS.md
├── tests/
│   └── stress/
│       ├── locustfile.py
│       └── requirements.txt
├── compose.yml
├── Makefile
├── SOLV_PROYECTO_DE_GRADO.md
└── TAREA.MD
```

---

## 2. Vista Lógica (Arquitectura Hexagonal - Nombres Exactos de Tipos Go)

### 2.1 Adaptadores Entrantes (Driving / Primary)
* **Handlers HTTP REST (`backend/internal/delivery/http/`)**:
  * `AuthHandler` (`auth_handler.go`)
  * `ClassroomHandler` (`classroom_handler.go`)
  * `ConfigHandler` (`config_handler.go`)
  * `EvaluationHandler` (`evaluation_handler.go`)
  * `MetricsHandler` (`metrics_handler.go`)
  * `SubjectHandler` (`subject_handler.go`)
  * `SubmissionHandler` (`submission_handler.go`)
  * `TeacherInvitationHandler` (`teacher_invitation_handler.go`)
  * `TemplateHandler` (`template_handler.go`)
  * `UserHandler` (`user_handler.go`)
  * `WorkspaceHandler` (`workspace_handler.go`)
* **Middlewares de Entrada (`backend/internal/delivery/http/middleware/`)**:
  * `TenantMiddleware` (`tenant_middleware.go`)
  * `UserRateLimiter` (`rate_limit_middleware.go`)
  * `AuditWorkerPool` (`audit_worker_pool.go`)
  * `statusCapturingResponseWriter` (`audit_middleware.go`)

### 2.2 Capa de Aplicación / Casos de Uso (`backend/internal/core/services/`)
* `AuthService` (`auth_service.go`)
* `EvaluationService` (`evaluation_service.go`)
* `EWMAProfilerServiceImpl` (`ewma_profiler.go`)
* `LabService` (`lab_service.go`)
* `QoSOrchestratorWorker` (`qos_worker.go`)
* `SemgrepWorker` (`semgrep_worker.go`)
* `StaticASTAnalyzer` (`ast_analyzer.go`)
* `SubjectService` (`subject_service.go`)
* `SubmissionService` (`submission_service.go`)
* `TeacherInvitationService` (`teacher_invitation_service.go`)
* `WorkspaceService` (`workspace_service.go`)
* `ZombieCollectorWorker` (`zombie_collector.go`)

### 2.3 Entidades de Dominio (`backend/internal/core/domain/`)
* `AuditLog` (`audit_log.go`)
* `Enrollment` (`academic.go`)
* `Exercise`, `ExerciseType`, `Verdict`, `ScanViolation`, `ScanResult`, `TestCase`, `AlgorithmConfig`, `DatabaseConfig`, `ExerciseConfig`, `EvaluationResult`, `EvaluationRunConfig`, `TestCaseRunResult`, `DBEvaluationRunConfig`, `DBEvaluationResult` (`exercise.go`)
* `LabInstance` (`lab_instance.go`)
* `LabTemplate` (`workspace.go`)
* `Subject` (`academic.go`)
* `Submission` (`academic.go`)
* `TeacherInvitation` (`academic.go`)
* `Tenant` (`tenant.go`)
* `WorkspaceInstance`, `WorkspaceContainerConfig`, `ContainerMetrics`, `EWMAState`, `ResourceProfile` (`workspace.go`)

### 2.4 Puertos / Interfaces (`backend/internal/core/domain/ports.go`, `audit_log.go`, `lab_instance.go`)
* `ASTAnalyzer` (`ports.go`)
* `AuditLogRepository` (`audit_log.go`)
* `CodeScanner` (`ports.go`)
* `ContainerOrchestrator` (`ports.go`)
* `DBEngineStrategy` (`ports.go`)
* `EvaluationRunner` (`ports.go`)
* `EWMAProfilerService` (`ports.go`)
* `ExerciseRepository` (`ports.go`)
* `HostMonitor` (`ports.go`)
* `LabContainerConfig` (`ports.go`)
* `LabInstanceRepository` (`lab_instance.go`)
* `LabTemplateRepository` (`ports.go`)
* `LanguageStrategy` (`ports.go`)
* `SubjectRepository` (`ports.go`)
* `SubmissionRepository` (`ports.go`)
* `TeacherInvitationRepository` (`ports.go`)
* `TemplateRepository` (`ports.go`)
* `TenantRepository` (`ports.go`)
* `WorkspaceOrchestrator` (`ports.go`)
* `WorkspaceRepository` (`ports.go`)

### 2.5 Adaptadores Salientes (Driven / Secondary)
* **Persistencia PostgreSQL (`backend/internal/infrastructure/storage/postgres/`)**:
  * `AuditLogRepository` (`audit_log_repository.go`)
  * `LabTemplateRepository` (`lab_template_repository.go`)
  * `PostgresExerciseRepository` (`exercise_repository.go`)
  * `PostgresSubjectRepository` (`subject_repository.go`)
  * `PostgresSubmissionRepository` (`submission_repository.go`)
  * `PostgresTeacherInvitationRepository` (`teacher_invitation_repository.go`)
  * `PostgresTenantRepository` (`tenant_repository.go`)
  * `PostgresWorkspaceRepository` (`workspace_repository.go`)
* **Orquestación de Contenedores Docker (`backend/internal/infrastructure/docker/`)**:
  * `Client` (`client.go`) - Implementa `domain.ContainerOrchestrator` y `domain.WorkspaceOrchestrator`.
  * `DockerService` (`docker_service.go`)
  * `Manager` (`manager.go`)
  * `Runner` (`runner.go`)
* **Análisis AST Semgrep (`backend/internal/core/services/` & `backend/internal/infrastructure/semgrep/`)**:
  * `SemgrepWorker` (`semgrep_worker.go`)
* **Monitoreo de Host (`backend/internal/infrastructure/system/`)**:
  * `GopsutilHostMonitor` (`host_monitor.go`)
* **Cliente Google Classroom (`backend/internal/delivery/http/`)**:
  * `ClassroomHandler` (`classroom_handler.go`) - Realiza la llamada HTTP GET directa al endpoint de Google Classroom API.

---

## 3. Vista de Procesos (Gestión del Ciclo de Vida de Contenedores y Recursos)

| Proceso / Operación | Función Encargada | Archivo |
|---|---|---|
| **Crear Volumen / Red ICC** | `EnsureVolumeExists`, `EnsureICCDisabledNetworkExists` | `backend/internal/infrastructure/docker/client.go` |
| **Crear / Iniciar Contenedor** | `StartContainer`, `CreateWorkspaceContainer` | `backend/internal/infrastructure/docker/client.go` |
| **Ejecutar Código / Sandbox** | `RunCodeInContainer`, `RunTestCase` | `backend/internal/infrastructure/docker/client.go` |
| **Pausar / Hibernar** | `PauseContainer` | `backend/internal/infrastructure/docker/client.go` |
| **Reanudar / Unpause** | `UnpauseContainer` | `backend/internal/infrastructure/docker/client.go` |
| **Destruir / Limpiar (Kill)** | `StopAndRemoveContainer`, `RemoveContainer` | `backend/internal/infrastructure/docker/client.go` |
| **Congelar Entorno (Read-Only)** | `StartContainer` (con `config.ReadOnly = true`) | `backend/internal/infrastructure/docker/client.go` |
| **Métricas RAM/CPU (Cgroups)** | `GetContainerMetrics` | `backend/internal/infrastructure/docker/client.go` |
| **Hibernación por Inactividad** | `QoSOrchestratorWorker.Start` | `backend/internal/core/services/qos_worker.go` |
| **Recolección de Zombis** | `ZombieCollectorWorker.Start` | `backend/internal/core/services/zombie_collector.go` |
| **Pre-chequeo AST (Semgrep)** | `SemgrepWorker.ScanCode` | `backend/internal/core/services/semgrep_worker.go` |

---

## 4. Vista Física (Infraestructura, Compose y Firewall)

### 4.1 Servicios Declarados en `compose.yml`

1. **`traefik`**:
   * **Imagen:** `traefik:v3.1.2`
   * **Nombre de Contenedor:** `solv_traefik`
   * **Puertos Publicados:** `"80:80"`, `"443:443"`, `"443:443/udp"`
   * **Volúmenes:**
     * `/var/run/docker.sock:/var/run/docker.sock:ro`
     * `./infra/traefik/traefik.yml:/etc/traefik/traefik.yml:ro`
     * `./infra/traefik/dynamic_conf.yml:/etc/traefik/dynamic_conf.yml:ro`
     * `./letsencrypt/acme.json:/letsencrypt/acme.json`
2. **`postgres`**:
   * **Imagen:** `postgres:18-alpine`
   * **Nombre de Contenedor:** `solv_db`
   * **Puertos Publicados:** `"127.0.0.1:5432:5432"`
   * **Volúmenes:** `pg_data:/var/lib/postgresql/data`

### 4.2 Configuración de Red, Firewall y TLS (`infra/`)

* **Traefik Proxy (`infra/traefik/`):**
  * `traefik.yml`: Enrutamiento dinámico escuchando el socket de Docker, Ingress HTTP/HTTPS, habilitador de Let's Encrypt DNS-01 para `desec.io`.
  * `dynamic_conf.yml`: Definición de middleware ForwardAuth en `/api/v1/auth/verify` y redirección automática HTTP -> HTTPS.
* **Firewall Host / iptables (`infra/firewall/`):**
  * `docker-user-rules.sh`: Script de configuración de reglas `iptables` en la cadena `DOCKER-USER` para impedir la comunicación inter-contenedor (`enable_icc=false`) y aislar los servicios sensibles en `127.0.0.1`.
* **Automatización DNS (`infra/desec/`):**
  * `setup_dns.sh`: Script para la actualización de registros DNS en desec.io.
* **Backup de Infraestructura (`infra/scripts/`):**
  * `backup.sh`: Script de respaldo para volúmenes de PostgreSQL y datos de estudiantes.

---

## 5. Vista de Desarrollo (Documentación y CI/CD)

### 5.1 Estructura Documental en `docs/`
* `docs/00-MAPA_SLICES.md`
* `docs/00-INVENTARIO_BAAS.md`
* `docs/REQUERIMIENTOS_SOLV.md`
* `docs/SECURITY.md`
* `docs/SECURITY_CHECKLIST.md`
* `docs/VERSIONS.md`
* `docs/API/CONTRATOS.md`
* `docs/ARQUITECTURA/CONCURRENCIA.md`
* `docs/ARQUITECTURA/SEGURIDAD.md`
* `docs/ARQUITECTURA/STORAGE_STRATEGY.md`
* `docs/ARQUITECTURA/ADR/` (ADR-000 al ADR-028 + `INDEX.md`)
* `docs/BD/DISENO_BD.md`
* `docs/GOBERNANZA/CONVENCIONES.md`
* `docs/GOBERNANZA/INVENTARIO_BAAS.md`
* `docs/GOBERNANZA/METODOLOGIA.md`
* `docs/SLICES/` (Carpetas del `SLICE_1` al `SLICE_11` con sus 4 documentos técnicos correspondientes).

### 5.2 CI/CD
* **Estado:** **NO ENCONTRADO** (No existen flujos de trabajo en `.github/workflows/` ni scripts de integración continua automatizada en el repositorio).

---

## 6. Tabla de Decisiones de Arquitectura (ADRs)

| ID | Título | Estado | Ruta del Archivo |
| :---: | :--- | :---: | :--- |
| **ADR-000** | Arquitectura Hexagonal y Stack de Servidor Nulo (Zero-Framework Go) | Activo | [`docs/ARQUITECTURA/ADR/ADR-000-arquitectura-hexagonal-zero-framework.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-000-arquitectura-hexagonal-zero-framework.md) |
| **ADR-001** | Estrategia Persistencia Datos | Activo | [`docs/ARQUITECTURA/ADR/ADR-001-estrategia-persistencia-datos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-001-estrategia-persistencia-datos.md) |
| **ADR-002** | Autenticacion Autorizacion Manejo Sesiones | Activo | [`docs/ARQUITECTURA/ADR/ADR-002-autenticacion-autorizacion-manejo-sesiones.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-002-autenticacion-autorizacion-manejo-sesiones.md) |
| **ADR-003** | ADR 003: Elección del Lenguaje, Framework Backend y Gestión de Recursos (Go y `net/http`) | Activo | [`docs/ARQUITECTURA/ADR/ADR-003-eleccion-lenguaje-framework-backend-go-gin.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-003-eleccion-lenguaje-framework-backend-go-gin.md) |
| **ADR-004** | Enrutamiento Dinamico Proxy Inverso Traefik | Activo | [`docs/ARQUITECTURA/ADR/ADR-004-enrutamiento-dinamico-proxy-inverso-traefik.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-004-enrutamiento-dinamico-proxy-inverso-traefik.md) |
| **ADR-005** | Sistema Evaluacion Juez Virtual Dual | Activo | [`docs/ARQUITECTURA/ADR/ADR-005-sistema-evaluacion-juez-virtual-dual.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-005-sistema-evaluacion-juez-virtual-dual.md) |
| **ADR-006** | Estrategia Aprovisionamiento Gestion Entornos | Activo | [`docs/ARQUITECTURA/ADR/ADR-006-estrategia-aprovisionamiento-gestion-entornos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-006-estrategia-aprovisionamiento-gestion-entornos.md) |
| **ADR-007** | Estrategia Sincronizacion Control Sesiones | Activo | [`docs/ARQUITECTURA/ADR/ADR-007-estrategia-sincronizacion-control-sesiones.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-007-estrategia-sincronizacion-control-sesiones.md) |
| **ADR-008** | Estrategia Asignacion Limitacion Recursos | Activo | [`docs/ARQUITECTURA/ADR/ADR-008-estrategia-asignacion-limitacion-recursos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-008-estrategia-asignacion-limitacion-recursos.md) |
| **ADR-009** | Estrategia Persistencia Motor Base Datos | Activo | [`docs/ARQUITECTURA/ADR/ADR-009-estrategia-persistencia-motor-base-datos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-009-estrategia-persistencia-motor-base-datos.md) |
| **ADR-010** | Arquitectura Evaluacion Segura Aislamiento | Activo | [`docs/ARQUITECTURA/ADR/ADR-010-arquitectura-evaluacion-segura-aislamiento.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-010-arquitectura-evaluacion-segura-aislamiento.md) |
| **ADR-011** | Gestion Dinamica Ciclo Vida Hibernacion | Activo | [`docs/ARQUITECTURA/ADR/ADR-011-gestion-dinamica-ciclo-vida-hibernacion.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-011-gestion-dinamica-ciclo-vida-hibernacion.md) |
| **ADR-012** | Arquitectura Frontend Motor Interfaz | Activo | [`docs/ARQUITECTURA/ADR/ADR-012-arquitectura-frontend-motor-interfaz.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-012-arquitectura-frontend-motor-interfaz.md) |
| **ADR-013** | Experiencia Desarrollador Perfiles Itinerantes | Activo | [`docs/ARQUITECTURA/ADR/ADR-013-experiencia-desarrollador-perfiles-itinerantes.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-013-experiencia-desarrollador-perfiles-itinerantes.md) |
| **ADR-014** | Estrategia Operativa Observabilidad Red | Activo | [`docs/ARQUITECTURA/ADR/ADR-014-estrategia-operativa-observabilidad-red.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-014-estrategia-operativa-observabilidad-red.md) |
| **ADR-015** | Migración a la Distribución Oficial OpenVSCode Server | Activo | [`docs/ARQUITECTURA/ADR/ADR-015-migracion-openvscode-server.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-015-migracion-openvscode-server.md) |
| **ADR-016** | Motor de Auditoría Semántica AST con SemgrepWorker | Activo | [`docs/ARQUITECTURA/ADR/ADR-016-auditoria-semantica-ast-semgrep.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-016-auditoria-semantica-ast-semgrep.md) |
| **ADR-017** | Autenticación Perimetral ForwardAuth vía Cookie HttpOnly Cross-Subdomain (D1) | Activo | [`docs/ARQUITECTURA/ADR/ADR-017-autenticacion-perimetral-forwardauth-httponly.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-017-autenticacion-perimetral-forwardauth-httponly.md) |
| **ADR-018** | Automatización TLS Wildcard y Certificados SSL vía ACME DNS-01 | Activo | [`docs/ARQUITECTURA/ADR/ADR-018-renovacion-automatica-tls-wildcard-acme-dns01.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-018-renovacion-automatica-tls-wildcard-acme-dns01.md) |
| **ADR-019** | Consolidación del Modelo de Dominio de Workspaces y Discriminador `type` (D3) | Activo | [`docs/ARQUITECTURA/ADR/ADR-019-consolidacion-modelo-workspaces-type-discriminator.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-019-consolidacion-modelo-workspaces-type-discriminator.md) |
| **ADR-020** | Motor de Auditoría Semántica AST Inmutable con Montaje de Solo Lectura (D4) | Activo | [`docs/ARQUITECTURA/ADR/ADR-020-auditoria-semantica-ast-inmutable-solo-lectura.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-020-auditoria-semantica-ast-inmutable-solo-lectura.md) |
| **ADR-021** | Registro Abierto OpenVSX para Entornos Interactivos (D5) | Activo | [`docs/ARQUITECTURA/ADR/ADR-021-integracion-registro-abierto-openvsx.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-021-integracion-registro-abierto-openvsx.md) |
| **ADR-022** | Integración y Sincronización Unidireccional con Google Classroom (D6) | Activo | [`docs/ARQUITECTURA/ADR/ADR-022-sincronizacion-unidireccional-google-classroom.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-022-sincronizacion-unidireccional-google-classroom.md) |
| **ADR-023** | Protección Zero Trust del Host con Cadena `DOCKER-USER` de `iptables` y Enlace Localhost | Activo | [`docs/ARQUITECTURA/ADR/ADR-023-proteccion-zero-trust-host-cadena-docker-user.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-023-proteccion-zero-trust-host-cadena-docker-user.md) |
| **ADR-024** | Esquema Académico Multi-Tenant Unificado | Activo | [`docs/ARQUITECTURA/ADR/ADR-024-esquema-academico-multitenant.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-024-esquema-academico-multitenant.md) |
| **ADR-025** | Invitaciones a Docentes mediante Transacciones Atómicas SQL | Activo | [`docs/ARQUITECTURA/ADR/ADR-025-invitaciones-docentes-transaccionales.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-025-invitaciones-docentes-transaccionales.md) |
| **ADR-026** | Pre-chequeo AST con Semgrep previo a la Aprovisionamiento del Contenedor | Activo | [`docs/ARQUITECTURA/ADR/ADR-026-pre-chequeo-ast-semgrep.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-026-pre-chequeo-ast-semgrep.md) |
| **ADR-027** | Operabilidad B2B — Audit Logs, Rate Limiting y Lock de Migraciones | Activo | [`docs/ARQUITECTURA/ADR/ADR-027-operabilidad-b2b.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-027-operabilidad-b2b.md) |
| **ADR-028** | Selección del Driver PostgreSQL (`lib/pq`) frente a `pgx` | Activo | [`docs/ARQUITECTURA/ADR/ADR-028-driver-postgresql-libpq.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-028-driver-postgresql-libpq.md) |
