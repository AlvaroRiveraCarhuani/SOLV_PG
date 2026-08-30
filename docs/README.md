# Centro de Documentación Técnica — SOLV BaaS

> **Documentación Oficial y Mapa de Arquitectura del Proyecto de Grado**  
> **Sistema de Orquestación de Laboratorios Virtuales (SOLV)**  
> **Arquitectura:** Hexagonal / Zero-Framework Go 1.26 + Angular 22 + PostgreSQL 18  
> **Gobernanza:** SDD (Spec-Driven Development) / Docs-as-Code  

---

## 1. Visión General del Producto y Requerimientos
* [`REQUERIMIENTOS_SOLV.md`](REQUERIMIENTOS_SOLV.md): Especificación oficial de Requerimientos Funcionales (RF) y No Funcionales (RNF).
* [`00-MAPA_SLICES.md`](00-MAPA_SLICES.md): Mapa de ruta de los 14 Slices Verticales del proyecto (`v0.11.0` completado).
* [`VERSIONS.md`](VERSIONS.md): Registro estricto de versiones fijadas de infraestructura Docker y dependencias.

---

## 2. Arquitectura de Software y Decisiones (ADRs)
* [`ARQUITECTURA/INVENTARIO_ARQUITECTURA.md`](ARQUITECTURA/INVENTARIO_ARQUITECTURA.md): Auditoría real de tipos Go, middlewares, adaptadores y vista de procesos.
* [`ARQUITECTURA/ADR/INDEX.md`](ARQUITECTURA/ADR/INDEX.md): Índice consolidado de los 29 Registros de Decisiones de Arquitectura (ADR-000 al ADR-028).
* [`ARQUITECTURA/CONCURRENCIA.md`](ARQUITECTURA/CONCURRENCIA.md): Modelo de control de concurrencia y límites de hardware.
* [`ARQUITECTURA/SEGURIDAD.md`](ARQUITECTURA/SEGURIDAD.md): Estrategia de seguridad, ForwardAuth HttpOnly (D1) y reglas `iptables` Zero-Trust.
* [`ARQUITECTURA/STORAGE_STRATEGY.md`](ARQUITECTURA/STORAGE_STRATEGY.md): Persistencia mediante volúmenes nombrados por estudiante/materia.

---

## 3. Diseño de Interfaz de Usuario y Experiencia (UI/UX)
* [`UI/UX-PRINCIPLES.md`](UI/UX-PRINCIPLES.md): Las 5 Leyes de Interfaz de SOLV (Dualidad de Estados, Carga Cognitiva Cero, Disclosure Progresivo).
* **Módulo Estudiante (`UI/ESTUDIANTE/`):**
  * [`UI/ESTUDIANTE/DASHBOARD.md`](UI/ESTUDIANTE/DASHBOARD.md): Vista 1 — Centro de Comando (Agenda + Laboratorios).
  * [`UI/ESTUDIANTE/LABORATORIO_ACTIVO.md`](UI/ESTUDIANTE/LABORATORIO_ACTIVO.md): Vista 2 — IDE Inmersivo (OpenVSCode Server + Protocolo `postMessage`).
  * [`UI/ESTUDIANTE/JUEZ_VIRTUAL.md`](UI/ESTUDIANTE/JUEZ_VIRTUAL.md): Vista 3 — Evaluación Algorítmica (Monaco Split-Screen + Semgrep AST).

---

## 4. Base de Datos y Contratos de Transporte API
* [`BD/DISENO_BD.md`](BD/DISENO_BD.md): Diagrama Entidad-Relación (ERD Mermaid) y DDL SQL oficial de PostgreSQL 18.
* [`API/CONTRATOS.md`](API/CONTRATOS.md): Contratos JSON de transporte REST OpenAPI para backend Go y modelos TypeScript Angular.

---

## 5. Gobernanza, Metodología y Backlog
* [`GOBERNANZA/INVENTARIO_BAAS.md`](GOBERNANZA/INVENTARIO_BAAS.md): Catálogo oficial de las 24 Capacidades Técnicas BaaS.
* [`GOBERNANZA/BACKLOG_COSTURAS.md`](GOBERNANZA/BACKLOG_COSTURAS.md): Registro de costuras técnicas y tareas post-MVP (COST-01 al COST-06).
* [`GOBERNANZA/METODOLOGIA.md`](GOBERNANZA/METODOLOGIA.md): Metodología Spec-Driven Development (SDD) e ingeniería por Rebanadas Verticales.
* [`GOBERNANZA/CONVENCIONES.md`](GOBERNANZA/CONVENCIONES.md): Convenciones de código, estándares de commit humano y vocabulario de delatores de IA prohibido.

---

## 6. Historial de Rebanadas Verticales (Slices 1 al 11)
La carpeta [`SLICES/`](SLICES/) contiene la especificación ejecutada y probada de cada slice completado del backend.
