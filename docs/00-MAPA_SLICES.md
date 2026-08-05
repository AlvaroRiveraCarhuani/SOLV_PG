# Mapa Global de Slices — Plataforma SOLV

Este documento define la hoja de ruta integral de la plataforma **SOLV** (Slices 1 a 14), categorizados por tipo de entregable, su contenido de requerimientos críticos (CRITs) y su estado actual de implementación.

---

> [!NOTE]
> **Convención de Slices:**
> - **Vertical MVP (Slices 1–7):** Construcción del núcleo funcional end-to-end (Backend Go, Docker, Traefik v3, Postgres, OpenVSCode Server, Semgrep).
> - **Hardening BaaS (Slices 8–11):** Blindaje de seguridad perimetral, multi-tenancy, optimización de recursos y preparación BaaS On-Premise.
> - **Vertical UI / Frontend (Slices 12–14):** Desarrollo de la interfaz de usuario en Angular 22, componentes de dashboard y experiencia de usuario.

---

## Tabla General de Slices (1–14)

| Slice # | Título del Slice | Tipo de Slice | Requerimientos Críticos (CRITs) / Hitos Incluidos | Estado |
| :---: | :--- | :--- | :--- | :---: |
| **01** | Canal de Comunicación Básico | Vertical MVP | Levantamiento de contenedor efímero Nginx + enrutamiento dinámico en Traefik v3. | **COMPLETADO** |
| **02** | Persistencia e Identidad Básica | Vertical MVP | Autenticación Google OAuth2 SSO + Volúmenes nombrados de Docker por estudiante. | **COMPLETADO** |
| **03** | Juez Virtual Algorítmico y BD | Vertical MVP | Pipeline de evaluación algorítmica (`stdin`/`stdout`) y evaluación de scripts SQL en PostgreSQL. | **COMPLETADO** |
| **04** | Entorno Interactivo Web (IDE) | Vertical MVP | Orquestación de contenedores OpenVSCode Server con subdominios dinámicos en Traefik v3. | **COMPLETADO** |
| **05** | Orquestador de Recursos & QoS | Vertical MVP | Control de admisión de RAM (15% headroom), hibernación por inactividad y penalización 3-strikes OOMKilled. | **COMPLETADO** |
| **06** | Blindaje de Red & Telemetría | Vertical MVP | Bloqueo de tráfico inter-contenedor (ICC), telemetría Prometheus `/metrics` y auto-tuning EWMA. | **COMPLETADO** |
| **07** | OpenVSCode Server & Auditoría AST | Vertical MVP | Migración a OpenVSCode Server oficial (puerto 3000) y worker de análisis semántico AST con Semgrep. | **COMPLETADO** |
| **08** | Hardening BaaS & Modelo Unificado | **Hardening BaaS** | **CRIT-01** (DOCKER-USER), **CRIT-04** (Multi-Tenancy `tenant_id`), **CRIT-05** (ForwardAuth HttpOnly), **CRIT-06** (Unificación Workspaces). | **COMPLETADO** |
| **09** | Esquema Académico & Hardening Seguridad | **Hardening BaaS** | **CRIT-02** (Esquema Académico Multi-Tenant, Subjects, Enrollments, Submissions, Invitaciones Docentes). | **COMPLETADO** |
| **10** | Robustez BaaS & Resiliencia | **Hardening BaaS** | **CRIT-09** (Rate Limiting Traefik), **CRIT-10** (Circuit Breakers), **CRIT-11** (Graceful Shutdown), **CRIT-12** (Audit Log de Eventos). | **PENDIENTE** |
| **11** | Optimización BaaS & Caché | **Hardening BaaS** | **CRIT-13** (Pool de Conexiones DB), **CRIT-14** (Warm Pool de Contenedores), **CRIT-15** (Caché de Imágenes Docker). | **PENDIENTE** |
| **12** | Interface Estudiantil (UI) | Vertical UI | Dashboard del estudiante en Angular 22, visor de workspaces, integración de iframe y consola de estado. | **PENDIENTE** |
| **13** | Interface Docente & Calificador (UI) | Vertical UI | Panel de control de ejercicios, visualizador de reportes AST Semgrep y gestión de laboratorios. | **PENDIENTE** |
| **14** | Admin Portal & Multi-Tenant UI | Vertical UI | Panel de administración de Tenants, métricas globales de servidor Asus y configuración de plataforma. | **PENDIENTE** |

---

> [!TIP]
> Todos los Slices del 1 al 9 cuentan con suites de pruebas automatizadas en `backend/tests/integration/` que validan su funcionamiento al 100%.
