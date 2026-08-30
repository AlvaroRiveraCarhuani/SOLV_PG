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

- **Fase A (Juez Virtual)**: Monaco nativo (componente Angular, sin iframe).
- **Fase B (Laboratorios)**: iframe OpenVSCode a pantalla completa con topbar mínimo.
- **Revisión Congelada**: iframe read-only + banner informativo ámbar.

## Stack de Tecnologías (NO cambiar)

- **Backend**: Go 1.26 + `net/http.ServeMux` (SIN Gin) + `sqlx` + PostgreSQL 18.
- **Frontend**: Angular 22 + Material 3 + Lucide + SCSS.
- **Infra**: Docker Engine v27 + Traefik v3.

## Reglas de Gobernanza de Código

- Arquitectura hexagonal: `core/domain`, `core/services`, `infrastructure/`, `delivery/http`.
- Commits Conventional Commits con cuerpo obligatorio: Qué / Por qué / Cómo / Impacto.
- PROHIBIDO: Hardcodear dominios, marcas, crear componentes duplicados o usar EMOJIS en archivos de documentación (.md), comentarios o código. Usar texto plano formal, diagramas Mermaid o Lucide Icons (`lucide:name`).
