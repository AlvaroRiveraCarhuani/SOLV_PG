# Mapa Global de Slices — Plataforma SOLV

Este documento define la hoja de ruta de la plataforma **SOLV** organizada en **Vertical Slices (1 a 16)**, estableciendo el alcance funcional, las decisiones de arquitectura vinculadas (ADRs) y el estado de implementación de cada extremo (Backend / Frontend / Verificación).

---

> [!NOTE]
> **Convención de Vertical Slices:**
> - **Vertical Core & MVP (Slices 1–7):** Núcleo funcional end-to-end (Backend Go, Docker, Traefik v3, Postgres, OpenVSCode Server, Semgrep).
> - **Hardening BaaS & Operabilidad (Slices 8–11):** Seguridad perimetral, multi-tenancy, resiliencia y base relacional.
> - **Experiencia de Usuario & Verticales de Rol (Slices 12–14):** Flujos completos por rol (Estudiante, Docente, Administrador) con integración backend/frontend.
> - **Servicios Transversales & Operación Avanzada (Slices 15–16):** Notificaciones proactivas y políticas de backup/retención institucional.

---

## Tabla General de Slices (1–16)

| Slice # | Título del Slice | ADRs Vinculados | Estado Backend | Estado Frontend | Estado de Verificación |
| :---: | :--- | :--- | :---: | :---: | :---: |
| **01** | Canal de Comunicación Básico | ADR-001, ADR-002 | Implementado | N/A | Verificado (Integración) |
| **02** | Persistencia e Identidad Básica | ADR-003, ADR-004 | Implementado | N/A | Verificado (Integración) |
| **03** | Juez Virtual Algorítmico y BD | ADR-005, ADR-006 | Implementado | N/A | Verificado (Integración) |
| **04** | Entorno Interactivo Web (IDE) | ADR-007, ADR-008 | Implementado | N/A | Verificado (Integración) |
| **05** | Orquestador de Recursos & QoS | ADR-009, ADR-010 | Implementado | N/A | Verificado (Integración) |
| **06** | Blindaje de Red & Telemetría | ADR-011, ADR-013 | Implementado | N/A | Verificado (Integración) |
| **07** | OpenVSCode Server & Auditoría AST | ADR-014, ADR-015 | Implementado | N/A | Verificado (Integración) |
| **08** | Hardening BaaS & Modelo Unificado | ADR-016, ADR-017, ADR-018, ADR-019 | Implementado | N/A | Verificado (Integración) |
| **09** | Esquema Académico & Seguridad | ADR-020, ADR-021, ADR-024 | Implementado | N/A | Verificado (Integración) |
| **10** | Robustez BaaS & Resiliencia | ADR-022, ADR-023, ADR-025 | Implementado | N/A | Verificado (Integración) |
| **11** | Operabilidad B2B & Migrations Lock | ADR-027, ADR-028 | Implementado | N/A | Verificado (Integración) |
| **12** | Experiencia Estudiante, Shell y Juez en Tiempo Real | ADR-012, ADR-037, ADR-029 | **Implementado** | **Pendiente** | **Verificado (PostgreSQL Real)** |
| **13** | Experiencia Docente, Cursos y Creación de Laboratorios | ADR-026, ADR-029, ADR-030, ADR-037 | Pendiente | Pendiente | Planificado |
| **14** | Panel Administrador Institucional & Gobernanza | ADR-024, ADR-027, ADR-029, ADR-030, ADR-031, ADR-032, ADR-033, ADR-036 | Pendiente | Pendiente | Planificado |
| **15** | Notificaciones Proactivas (In-App & Email) | ADR-031, ADR-032, ADR-034, ADR-035 | Pendiente | Pendiente | Planificado |
| **16** | Backups y Retención Institucional | ADR-027, ADR-034, ADR-035 | Pendiente | Pendiente | Planificado |

---

## Detalle de Slices de Aplicación (12–16)

### Slice 12: Experiencia Estudiante, Shell y Juez en Tiempo Real
- **Alcance:** Shell autenticado del estudiante, resolución de identidad `GET /api/v1/users/me`, dashboard con entregas próximas `GET /api/v1/student/assignments/due`, pausa/hibernación manual `POST /api/v1/workspaces/{id}/pause` y canal dúplex WebSocket para el Juez Virtual (`/ws/v1/evaluations`).
- **Estado Actual:** Backend implementado y verificado con PostgreSQL en Docker. Frontend pendiente.

### Slice 13: Experiencia Docente, Cursos y Creación de Laboratorios
- **Alcance:** Dashboard docente, vista de curso, wizard de creación de ejercicios/laboratorios, cola de revisión de entregas, auditoría de código con Semgrep, solicitud de plantillas Docker personalizadas y filtro por periodo académico.
- **Estado Actual:** Planificado.

### Slice 14: Panel Administrador Institucional & Gobernanza
- **Alcance:** Consola administrativa multi-tenant, gestión de estudiantes (búsqueda, reset de sesiones, bloqueo), gestión de docentes y reasignación de cursos huérfanos, periodos académicos y archivado, aprobación de plantillas Docker, control de modo mantenimiento y 5 acciones de emergencia con confirmación tipada.
- **Estado Actual:** Planificado.

### Slice 15: Notificaciones Proactivas (In-App & Email)
- **Alcance:** Centro de notificaciones in-app (campana, conteo no leídas, severidades), despachador asíncrono, canal de correo transaccional para eventos críticos y reglas anti-fatiga de alertas.
- **Estado Actual:** Planificado.

### Slice 16: Backups y Retención Institucional
- **Alcance:** Programación de copias de seguridad de PostgreSQL, retención local por días, sincronización remota configurable (S3/B2), verificación de integridad y restauración controlada.
- **Estado Actual:** Planificado.
