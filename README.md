# SOLV — Sistema de Orquestación de Laboratorios Virtuales

> **Plataforma Académica de Laboratorios e IDEs Virtuales (BaaS-Ready)**  
> **Proyecto de Grado** | Universidad Adventista de Bolivia (UAB)  

---

## Stack Tecnológico de Ingeniería

<p align="left">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Angular-DD0031?style=for-the-badge&logo=angular&logoColor=white" alt="Angular" />
  <img src="https://img.shields.io/badge/PostgreSQL-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/Traefik-24A1DE?style=for-the-badge&logo=traefik&logoColor=white" alt="Traefik" />
  <img src="https://img.shields.io/badge/Semgrep-000000?style=for-the-badge&logo=semgrep&logoColor=white" alt="Semgrep" />
</p>

| Capa de Arquitectura | Tecnología / Herramienta | Versión Fijada | Propósito y Función |
|---|---|---|---|
| **Lenguaje Backend** | **Go (Golang)** | `1.26` | Servidor HTTP nulo (`net/http`), concurrencia y orquestación de Docker |
| **Framework Frontend** | **Angular Standalone** | `22` | Interfaz reactiva basada en Signals, Material 3 e integración Iframe |
| **Base de Datos** | **PostgreSQL** | `18-alpine` | Persistencia relacional multi-tenant con aislamiento por `tenant_id` |
| **Orquestación Sandbox** | **Docker Engine SDK** | `v27.0` | Contenedores efímeros/persistentes en entornos aislados (`network_mode: none`) |
| **Proxy Ingress / TLS** | **Traefik Reverse Proxy** | `v3.1.2` | Terminación TLS Wildcard, ForwardAuth HttpOnly y Rate Limiting |
| **Auditoría AST** | **Semgrep CLI** | `1.100.0` | Análisis estático y semántico de código previo a la asignación de recursos |

---

## Arquitectura de Software y Decisiones Clave (D1–D6)

La arquitectura de **SOLV** se implementa bajo el patrón de **Arquitectura Hexagonal (Puertos y Adaptadores)** desacoplada, regida por seis decisiones inamovibles de diseño:

```text
                               ┌────────────────────────────────────────┐
                               │       CLIENTE WEB / ANGULAR 22         │
                               └───────────────────┬────────────────────┘
                                                   │ Cookie HttpOnly solv_session
                                                   ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ TRAEFIK V3 INGRESS PROXY (ForwardAuth GET /api/v1/auth/verify)                                  │
└──────────────────────────────────────────────────┬───────────────────────────────────────────────┘
                                                   │
                                                   ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ BACKEND GO 1.26 (Servidor Nulo - net/http.ServeMux)                                              │
│                                                                                                  │
│  [ Driving Adapters ]   ──>   [ Core Application / Domain ]   ──>   [ Driven Adapters ]          │
│  - HTTP Handlers              - WorkspaceService                    - Postgres (sqlx)            │
│  - Tenant Middleware          - EvaluationService                   - Docker Engine SDK          │
│  - Audit Middleware           - SemgrepWorker (AST Audit)           - Gopsutil Host Monitor      │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Tabla de Decisiones Inamovibles

- **ForwardAuth Cookie HttpOnly (D1):** Autenticación validada mediante cookie `solv_session` integrada con el middleware ForwardAuth de Traefik v3 (`GET /api/v1/auth/verify`). Se elimina el uso de JWTs en parámetros de URL para prevenir fugas de credenciales.
- **Multi-Tenancy Lógico por `tenant_id` (D2):** Aislamiento multi-inquilino en PostgreSQL mediante la columna `tenant_id` en todas las tablas relacionales y validación de ámbito vía `TenantMiddleware`.
- **Consolidación en `workspaces` (D3):** Unificación de laboratorios interactivos y jueces de evaluación en la entidad relacional `workspaces` con discriminador de tipo (`type`).
- **Pre-chequeo AST Semgrep (D4):** Análisis estático y semántico de código fuente en `<100ms` previo a la provisión de contenedores sandbox efímeros.
- **Entorno Interactivo OpenVSX (D5):** Registro de extensiones libre y soporte completo de lenguajes compilados (C#/C++) vía terminal en OpenVSCode Server.
- **Google Classroom Unidireccional (D6):** Importación manual de listas de estudiantes mediante peticiones `GET` de solo lectura.

---

## Estructura General del Repositorio

```text
SOLV_PG/
├── backend/                # Código fuente Backend en Go 1.26 (Arquitectura Hexagonal)
├── frontend/               # Código fuente Frontend en Angular 22 (Standalone & Signals)
├── infra/                  # Configuraciones de Traefik v3, firewall iptables y scripts de backup
├── docs/                   # Centro de Documentación Técnica viva (Docs-as-Code)
│   ├── ARQUITECTURA/       # Inventario de tipos Go, ADR-000 a ADR-028, Concurrencia y Seguridad
│   ├── UI/                 # Wireframes ASCII Art, diagramas Mermaid HD y Principios UX
│   ├── BD/                 # Modelo Relacional (ERD Mermaid) y DDL PostgreSQL 18
│   ├── API/                # Contratos OpenAPI y payloads JSON REST
│   └── GOBERNANZA/         # Backlog de costuras, capacidades BaaS y metodología SDD
├── compose.yml             # Orquestación Docker Compose (Traefik v3 + PostgreSQL 18)
└── Makefile                # Automatización de tareas de compilación y pruebas de integración
```

---

## Guía de Inicio Rápido

### 1. Clonar el repositorio y configurar variables de entorno
```bash
git clone https://github.com/tu-usuario/SOLV_PG.git
cd SOLV_PG
cp .env.example .env
```

### 2. Desplegar servicios de infraestructura
```bash
docker compose up -d
```

### 3. Ejecutar el servidor backend en Go
```bash
cd backend
go run cmd/api/main.go
```

---

## Centro de Documentación Técnica

Para consultar la especificación detallada de requerimientos, decisiones de arquitectura, modelo relacional de base de datos o wireframes visuales, ingrese al **[Centro de Documentación (docs/README.md)](docs/README.md)**.
