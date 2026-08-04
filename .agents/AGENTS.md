# SOLV — Contexto y Reglas del Proyecto

## Qué es SOLV
Plataforma académica de orquestación de laboratorios virtuales (BaaS-ready).
On-Premise single-node hoy (servidor Asus), multi-tenant ready por diseño.
Roles: student, teacher, admin.

## Stack (NO cambiar)
- Backend: Go 1.26 + net/http.ServeMux (SIN Gin) + sqlx + PostgreSQL 18
- Frontend: Angular 22 + Material + Lucide + SCSS
- Infra: Docker Engine v27 + Traefik v3 + OpenVSCode Server + Semgrep
- DNS/TLS: desec.io + Let's Encrypt Wildcard (ACME DNS-01). NUNCA dominios .local

## Convenciones de código
- Arquitectura hexagonal: core/domain, core/services, infrastructure/, delivery/http
- Migraciones idempotentes en postgres.go
- Tests de integración en backend/tests/integration/
- Commits Conventional Commits con cuerpo: Qué/Por qué/Cómo/Impacto

## PROHIBIDO
- Hardcodear dominios o "UAB" (todo vía config)
- Exponer test_cases de exercises al rol student
- Usar iptables normal para aislar el bridge (usar cadena DOCKER-USER o bind localhost)
- Crear tablas/componentes que dupliquen existentes
- Usar imágenes :latest (versiones fijadas)

## Semántica de estados (UI y API)
running=success · pending=warning · hibernated=neutral · failed/oom_killed=error
Veredictos juez: AC=verde · WA=rojo · TLE=ámbar · RE=naranja · AST_BLOCKED=rojo intenso
