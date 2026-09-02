# Inventario Técnico de Arquitectura del Repositorio SOLV

> **Documento Oficial de Auditoría Técnica de Arquitectura**  
> **Proyecto:** SOLV (Sistema de Orquestación de Laboratorios Virtuales)  
> **Repositorio:** SOLV_PG  
> **Última Actualización:** 2026-09-02 (Alineación SDD Slices 12 a 16)  

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
│   ├── API_CONTRACTS.md
│   ├── ARQUITECTURA/
│   │   ├── ADR/ (ADR-000 a ADR-037)
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
│   │   ├── SLICE_1/ ... SLICE_16/
│   ├── 00-INVENTARIO_BAAS.md
│   ├── 00-MAPA_SLICES.md
│   ├── REQUERIMIENTOS_SOLV.md
│   ├── SECURITY.md
│   ├── SECURITY_CHECKLIST.md
│   └── VERSIONS.md
├── compose.yml
└── Makefile
```

---

## 2. Vista Lógica (Arquitectura Hexagonal - Nombres Exactos de Tipos Go)

### 2.1 Adaptadores Entrantes (Driving / Primary)
* **Handlers HTTP REST y WebSocket (`backend/internal/delivery/http/`)**:
  * `AdminHandler` (`admin_handler.go`) — *Implementado*
  * `AuthHandler` (`auth_handler.go`) — *Implementado*
  * `ClassroomHandler` (`classroom_handler.go`) — *Implementado*
  * `ConfigHandler` (`config_handler.go`) — *Implementado*
  * `EvaluationHandler` (`evaluation_handler.go`) — *Implementado*
  * `MetricsHandler` (`metrics_handler.go`) — *Implementado*
  * `StudentHandler` (`student_handler.go`) — *Implementado*
  * `SubjectHandler` (`subject_handler.go`) — *Implementado*
  * `SubmissionHandler` (`submission_handler.go`) — *Implementado*
  * `TeacherInvitationHandler` (`teacher_invitation_handler.go`) — *Implementado*
  * `TemplateHandler` (`template_handler.go`) — *Implementado*
  * `UserHandler` (`user_handler.go`) — *Implementado*
  * `WorkspaceHandler` (`workspace_handler.go`) — *Implementado*
  * `WebSocketHandler` (`websocket_handler.go` — ADR-037) — *Implementado*
* **Middlewares de Entrada (`backend/internal/delivery/http/middleware/`)**:
  * `TenantMiddleware` (`tenant_middleware.go`) — *Implementado*
  * `UserRateLimiter` (`rate_limit_middleware.go`) — *Implementado*
  * `AuditWorkerPool` (`audit_worker_pool.go`) — *Implementado*
  * `statusCapturingResponseWriter` (`audit_middleware.go`) — *Implementado*
  * `MaintenanceMiddleware` — *Aprobado Pendiente (ADR-031 / Slice 14)*

### 2.2 Capa de Aplicación / Casos de Uso (`backend/internal/core/services/`)
* **Implementados:**
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
  * `WebSocketHub` (`delivery/http/websocket_hub.go` — ADR-037 / Slice 12)
  * `WorkspaceService` (`workspace_service.go`)
  * `ZombieCollectorWorker` (`zombie_collector.go`)
* **Aprobados Pendientes de Implementación:**
  * `AcademicPeriodService` — *Aprobado (ADR-029 / Slice 14)*
  * `TemplateApprovalService` — *Aprobado (ADR-030 / Slice 14)*
  * `EmergencyActionsService` — *Aprobado (ADR-032 / Slice 14)*
  * `StudentManagementService` — *Aprobado (ADR-033 / Slice 14)*
  * `NotificationDispatcher` — *Aprobado (ADR-034 / Slice 15)*
  * `BackupWorker` — *Aprobado (ADR-035 / Slice 16)*
  * `CourseReassignmentService` — *Aprobado (ADR-036 / Slice 14)*

### 2.3 Entidades de Dominio (`backend/internal/core/domain/`)
* `AuditLog` (`audit_log.go`)
* `DueAssignment` (`exercise.go`)
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

---

## 3. Tabla de Decisiones de Arquitectura (ADRs)

| ID | Título | Estado | Ruta del Archivo |
| :---: | :--- | :---: | :--- |
| **ADR-000** | Arquitectura Hexagonal y Stack de Servidor Nulo (Zero-Framework Go) | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-000-arquitectura-hexagonal-zero-framework.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-000-arquitectura-hexagonal-zero-framework.md) |
| **ADR-001** | Estrategia Persistencia Datos | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-001-estrategia-persistencia-datos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-001-estrategia-persistencia-datos.md) |
| **ADR-002** | Autenticacion Autorizacion Manejo Sesiones | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-002-autenticacion-autorizacion-manejo-sesiones.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-002-autenticacion-autorizacion-manejo-sesiones.md) |
| **ADR-003** | Elección del Lenguaje, Framework Backend y Gestión de Recursos (Go y `net/http`) | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-003-eleccion-lenguaje-framework-backend-go-gin.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-003-eleccion-lenguaje-framework-backend-go-gin.md) |
| **ADR-004** | Enrutamiento Dinamico Proxy Inverso Traefik | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-004-enrutamiento-dinamico-proxy-inverso-traefik.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-004-enrutamiento-dinamico-proxy-inverso-traefik.md) |
| **ADR-005** | Sistema Evaluacion Juez Virtual Dual | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-005-sistema-evaluacion-juez-virtual-dual.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-005-sistema-evaluacion-juez-virtual-dual.md) |
| **ADR-006** | Estrategia Aprovisionamiento Gestion Entornos | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-006-estrategia-aprovisionamiento-gestion-entornos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-006-estrategia-aprovisionamiento-gestion-entornos.md) |
| **ADR-007** | Estrategia Sincronizacion Control Sesiones | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-007-estrategia-sincronizacion-control-sesiones.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-007-estrategia-sincronizacion-control-sesiones.md) |
| **ADR-008** | Estrategia Asignacion Limitacion Recursos | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-008-estrategia-asignacion-limitacion-recursos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-008-estrategia-asignacion-limitacion-recursos.md) |
| **ADR-009** | Estrategia Persistencia Motor Base Datos | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-009-estrategia-persistencia-motor-base-datos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-009-estrategia-persistencia-motor-base-datos.md) |
| **ADR-010** | Arquitectura Evaluacion Segura Aislamiento | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-010-arquitectura-evaluacion-segura-aislamiento.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-010-arquitectura-evaluacion-segura-aislamiento.md) |
| **ADR-011** | Gestion Dinamica Ciclo Vida Hibernacion | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-011-gestion-dinamica-ciclo-vida-hibernacion.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-011-gestion-dinamica-ciclo-vida-hibernacion.md) |
| **ADR-012** | Arquitectura Frontend Motor Interfaz | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-012-arquitectura-frontend-motor-interfaz.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-012-arquitectura-frontend-motor-interfaz.md) |
| **ADR-013** | Experiencia Desarrollador Perfiles Itinerantes | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-013-experiencia-desarrollador-perfiles-itinerantes.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-013-experiencia-desarrollador-perfiles-itinerantes.md) |
| **ADR-014** | Estrategia Operativa Observabilidad Red | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-014-estrategia-operativa-observabilidad-red.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-014-estrategia-operativa-observabilidad-red.md) |
| **ADR-015** | Migración a la Distribución Oficial OpenVSCode Server | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-015-migracion-openvscode-server.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-015-migracion-openvscode-server.md) |
| **ADR-016** | Motor de Auditoría Semántica AST con SemgrepWorker | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-016-auditoria-semantica-ast-semgrep.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-016-auditoria-semantica-ast-semgrep.md) |
| **ADR-017** | Autenticación Perimetral ForwardAuth vía Cookie HttpOnly Cross-Subdomain (D1) | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-017-autenticacion-perimetral-forwardauth-httponly.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-017-autenticacion-perimetral-forwardauth-httponly.md) |
| **ADR-018** | Automatización TLS Wildcard y Certificados SSL vía ACME DNS-01 | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-018-renovacion-automatica-tls-wildcard-acme-dns01.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-018-renovacion-automatica-tls-wildcard-acme-dns01.md) |
| **ADR-019** | Consolidación del Modelo de Dominio de Workspaces y Discriminador `type` (D3) | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-019-consolidacion-modelo-workspaces-type-discriminator.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-019-consolidacion-modelo-workspaces-type-discriminator.md) |
| **ADR-020** | Motor de Auditoría Semántica AST Inmutable con Montaje de Solo Lectura (D4) | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-020-auditoria-semantica-ast-inmutable-solo-lectura.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-020-auditoria-semantica-ast-inmutable-solo-lectura.md) |
| **ADR-021** | Registro Abierto OpenVSX para Entornos Interactivos (D5) | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-021-integracion-registro-abierto-openvsx.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-021-integracion-registro-abierto-openvsx.md) |
| **ADR-022** | Integración y Sincronización Unidireccional con Google Classroom (D6) | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-022-sincronizacion-unidireccional-google-classroom.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-022-sincronizacion-unidireccional-google-classroom.md) |
| **ADR-023** | Protección Zero Trust del Host con Cadena `DOCKER-USER` de `iptables` y Enlace Localhost | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-023-proteccion-zero-trust-host-cadena-docker-user.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-023-proteccion-zero-trust-host-cadena-docker-user.md) |
| **ADR-024** | Esquema Académico Multi-Tenant Unificado | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-024-esquema-academico-multitenant.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-024-esquema-academico-multitenant.md) |
| **ADR-025** | Invitaciones a Docentes mediante Transacciones Atómicas SQL | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-025-invitaciones-docentes-transaccionales.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-025-invitaciones-docentes-transaccionales.md) |
| **ADR-026** | Pre-chequeo AST con Semgrep previo a la Aprovisionamiento del Contenedor | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-026-pre-chequeo-ast-semgrep.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-026-pre-chequeo-ast-semgrep.md) |
| **ADR-027** | Operabilidad B2B — Audit Logs, Rate Limiting y Lock de Migraciones | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-027-operabilidad-b2b.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-027-operabilidad-b2b.md) |
| **ADR-028** | Selección del Driver PostgreSQL (`lib/pq`) frente a `pgx` | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-028-driver-postgresql-libpq.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-028-driver-postgresql-libpq.md) |
| **ADR-029** | Periodos Académicos y Archivado de Cursos | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-029-periodos-academicos-archivado-cursos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-029-periodos-academicos-archivado-cursos.md) |
| **ADR-030** | Catálogo de Plantillas Docker con Flujo de Aprobación | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-030-catalogo-plantillas-docker-aprobacion.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-030-catalogo-plantillas-docker-aprobacion.md) |
| **ADR-031** | Modo Mantenimiento Global con Bypass Administrativo | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-031-modo-mantenimiento-global.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-031-modo-mantenimiento-global.md) |
| **ADR-032** | Cinco Acciones de Emergencia del Administrador | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-032-acciones-emergencia-administrador.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-032-acciones-emergencia-administrador.md) |
| **ADR-033** | Gestión Administrativa de Estudiantes (Búsqueda, Reset, Estado) | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-033-gestion-estudiantes-administrador.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-033-gestion-estudiantes-administrador.md) |
| **ADR-034** | Sistema de Notificaciones Proactivas (In-App y Email) | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-034-sistema-notificaciones-proactivas.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-034-sistema-notificaciones-proactivas.md) |
| **ADR-035** | Backups Configurables con Política de Retención Local y Remota | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-035-backups-configurables-retencion.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-035-backups-configurables-retencion.md) |
| **ADR-036** | Reasignación de Docentes para Cursos Huérfanos | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-036-reasignacion-docentes-cursos-huerfanos.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-036-reasignacion-docentes-cursos-huerfanos.md) |
| **ADR-037** | WebSocket Hub para Evaluación del Juez Virtual en Tiempo Real | Aprobado | [`docs/ARQUITECTURA/ADR/ADR-037-websocket-hub-evaluacion-juez-virtual.md`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/docs/ARQUITECTURA/ADR/ADR-037-websocket-hub-evaluacion-juez-virtual.md) |
