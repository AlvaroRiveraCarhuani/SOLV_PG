# Índice de Decisiones de Arquitectura (ADRs)

Este documento consolida todas las Decisiones de Arquitectura (ADRs) tomadas durante el desarrollo de la plataforma **SOLV**.

| Número | Título | Estado | Slice de Origen |
| :---: | :--- | :---: | :---: |
| 000 | [Arquitectura Hexagonal y Stack de Servidor Nulo (Zero-Framework Go)](ADR-000-arquitectura-hexagonal-zero-framework.md) | Activo | Slice 1, Slice 8 |
| 001 | [Estrategia Persistencia Datos](ADR-001-estrategia-persistencia-datos.md) | Activo | Slice 2 |
| 002 | [Autenticacion Autorizacion Manejo Sesiones](ADR-002-autenticacion-autorizacion-manejo-sesiones.md) | Activo | Slice 2 |
| 003 | [ADR 003: Elección del Lenguaje, Framework Backend y Gestión de Recursos (Go y `net/http`)](ADR-003-eleccion-lenguaje-framework-backend-go-gin.md) | Activo | Slice 1, Slice 3 |
| 004 | [Enrutamiento Dinamico Proxy Inverso Traefik](ADR-004-enrutamiento-dinamico-proxy-inverso-traefik.md) | Activo | Slice 1, Slice 4, Slice 8 |
| 005 | [Sistema Evaluacion Juez Virtual Dual](ADR-005-sistema-evaluacion-juez-virtual-dual.md) | Activo | Slice 3 |
| 006 | [Estrategia Aprovisionamiento Gestion Entornos](ADR-006-estrategia-aprovisionamiento-gestion-entornos.md) | Activo | Slice 4 |
| 007 | [Estrategia Sincronizacion Control Sesiones](ADR-007-estrategia-sincronizacion-control-sesiones.md) | Activo | Slice 5 |
| 008 | [Estrategia Asignacion Limitacion Recursos](ADR-008-estrategia-asignacion-limitacion-recursos.md) | Activo | Slice 5 |
| 009 | [Estrategia Persistencia Motor Base Datos](ADR-009-estrategia-persistencia-motor-base-datos.md) | Activo | Slice 3, Slice 6 |
| 010 | [Arquitectura Evaluacion Segura Aislamiento](ADR-010-arquitectura-evaluacion-segura-aislamiento.md) | Activo | Slice 3 |
| 011 | [Gestion Dinamica Ciclo Vida Hibernacion](ADR-011-gestion-dinamica-ciclo-vida-hibernacion.md) | Activo | Slice 5 |
| 012 | [Arquitectura Frontend Motor Interfaz](ADR-012-arquitectura-frontend-motor-interfaz.md) | Activo | Slice 12 |
| 013 | [Experiencia Desarrollador Perfiles Itinerantes](ADR-013-experiencia-desarrollador-perfiles-itinerantes.md) | Activo | Slice 13 |
| 014 | [Estrategia Operativa Observabilidad Red](ADR-014-estrategia-operativa-observabilidad-red.md) | Activo | Slice 6 |
| 015 | [Migración a la Distribución Oficial OpenVSCode Server](ADR-015-migracion-openvscode-server.md) | Activo | Slice 7, Slice 8 |
| 016 | [Motor de Auditoría Semántica AST con SemgrepWorker](ADR-016-auditoria-semantica-ast-semgrep.md) | Activo | Slice 7 |
| 017 | [Autenticación Perimetral ForwardAuth vía Cookie HttpOnly Cross-Subdomain (D1)](ADR-017-autenticacion-perimetral-forwardauth-httponly.md) | Activo | Slice 8 |
| 018 | [Automatización TLS Wildcard y Certificados SSL vía ACME DNS-01 (D2)](ADR-018-renovacion-automatica-tls-wildcard-acme-dns01.md) | Activo | Slice 8 |
| 019 | [Consolidación del Modelo de Dominio de Workspaces y Discriminador `type` (D3)](ADR-019-consolidacion-modelo-workspaces-type-discriminator.md) | Activo | Slice 8 |
| 020 | [Motor de Auditoría Semántica AST Inmutable con Montaje de Solo Lectura (D4)](ADR-020-auditoria-semantica-ast-inmutable-solo-lectura.md) | Activo | Slice 8 |
| 021 | [Registro Abierto OpenVSX para Entornos Interactivos (D5)](ADR-021-integracion-registro-abierto-openvsx.md) | Activo | Slice 8 |
| 022 | [Integración y Sincronización Unidireccional con Google Classroom (D6)](ADR-022-sincronizacion-unidireccional-google-classroom.md) | Activo | Slice 8 |
| 023 | [Protección Zero Trust del Host con Cadena `DOCKER-USER` de `iptables` y Enlace Localhost](ADR-023-proteccion-zero-trust-host-cadena-docker-user.md) | Activo | Slice 8 |
| 024 | [Esquema Académico Multi-Tenant Unificado](ADR-024-esquema-academico-multitenant.md) | Activo | Slice 9 |
| 025 | [Invitaciones a Docentes mediante Transacciones Atómicas SQL](ADR-025-invitaciones-docentes-transaccionales.md) | Activo | Slice 9 |
| 026 | [Pre-chequeo AST con Semgrep previo a la Aprovisionamiento del Contenedor](ADR-026-pre-chequeo-ast-semgrep.md) | Activo | Slice 9 |
| 027 | [Operabilidad B2B — Audit Logs, Rate Limiting y Lock de Migraciones](ADR-027-operabilidad-b2b.md) | Activo | Slice 11 |
| 028 | [Selección del Driver PostgreSQL (`lib/pq`) frente a `pgx`](ADR-028-driver-postgresql-libpq.md) | Activo | Slice 11 |
