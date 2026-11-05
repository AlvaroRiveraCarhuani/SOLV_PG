---
name: solv-context
description: Core SOLV project context: architecture decisions D1-D6 (ForwardAuth, multi-tenancy), roles, and non-negotiable rules. Trigger: ONLY when the work touches authentication, sessions, tenancy/tenant_id, roles/permissions, or architectural decisions.
---

# SOLV — Contexto del Proyecto y Reglas Core

## Qué es SOLV

Plataforma académica de orquestación de laboratorios virtuales (BaaS-ready).
On-Premise single-node hoy (servidor Asus), multi-tenant ready por diseño.
Roles: student (solo ve sus laboratorios/ejercicios), teacher (crea plantillas/laboratorios, califica, revisión congelada read-only), admin (usuarios, plantillas globales, infra).

## Decisiones Inamovibles de Arquitectura (D1 - D6)

- **D1**: ForwardAuth HttpOnly cookie `solv_session` en `/api/v1/auth/verify`. PROHIBIDO JWT en URLs.
- **D2**: Multi-tenancy lógico por discriminador `tenant_id` en PostgreSQL y middleware `TenantMiddleware`.
- **D3**: Unificación de laboratorios en una sola tabla `workspaces` con discriminador `type` (`IDE_PERSISTENTE` | `JUEZ_EFIMERO`).
- **D4**: Semgrep como motor AST principal del juez (pre-chequeo estático antes del sandbox).
- **D5**: Registro de extensiones Open VSX únicamente; soporte C#/C++ vía terminal funcional.
- **D6**: Classroom SSO unidireccional + importación manual GET + panel docente de solo lectura.

## Editores y Visualizadores de Código

- **Fase A (Entorno de Evaluación)**: Monaco nativo (componente Angular, sin iframe).
- **Fase B (Laboratorios)**: iframe OpenVSCode a pantalla completa con topbar mínimo.
- **Revisión Congelada**: iframe read-only + banner informativo ámbar.

## Terminología de entornos

| Identificador interno | Nombre visible |
| :--- | :--- |
| `JUEZ_EFIMERO` | Entorno de Evaluación |
| `IDE_PERSISTENTE` | Entorno de Desarrollo |
| `AMBOS` | Entorno de Desarrollo y Evaluación |

- El identificador no se traduce ni se muestra como etiqueta. «Juez Virtual» es el motor que evalúa, no el nombre de la modalidad.
- Consultar `frontend/src/app/shared/constants/environment-types.constants.ts` y usar el pipe `environmentLabel`; nunca escribir la etiqueta a mano.
- El gate `terminology-gate` (en `npm run lint:styles` y CI) clasifica por forma: rechaza literales y celdas con forma de modalidad, variantes de mayúsculas o acentos, y nombres nuevos como «Entorno de Programación». Los nombres antiguos solo se aceptan en registros históricos que referencian la decisión vigente **ADR-067**.
- Un valor de `environment_type` que no se reconoce se muestra como estado neutro; nunca se asume Entorno de Desarrollo ni Entorno de Evaluación.

## Stack de Tecnologías (NO cambiar)

- **Backend**: Go 1.26 + `net/http.ServeMux` (SIN Gin) + `sqlx` + PostgreSQL 18.
- **Frontend**: Angular 22 + Material 3 + Lucide + SCSS.
- **Infra**: Docker Engine v27 + Traefik v3.

## Reglas de Gobernanza de Código

- Arquitectura hexagonal: `core/domain`, `core/services`, `infrastructure/`, `delivery/http`.
- Commits Conventional Commits con cuerpo obligatorio: Qué / Por qué / Cómo / Impacto.
- PROHIBIDO: Hardcodear dominios, marcas, crear componentes duplicados o usar EMOJIS en archivos de documentación (.md), comentarios o código. Usar texto plano formal, diagramas Mermaid o Lucide Icons (`lucide:name`).
