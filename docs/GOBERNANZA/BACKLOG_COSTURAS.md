# Backlog de Costuras Técnicas y Trabajo Futuro (SOLV)

> **Documento Oficial de Seguimiento de Costuras Técnicas y Optimizaciones Post-MVP**
> **Proyecto:** SOLV (Sistema de Orquestación de Laboratorios Virtuales)
> **Gobernanza:** SDD / Docs-as-Code

---

## 1. Visión General del Backlog

Este documento registra los requerimientos, mejoras de infraestructura y refinamientos de interfaz identificados durante la fase de desarrollo que han sido clasificados como **Costuras Futuras** o **Mejoras Post-MVP**.

Su objetivo es evitar la dispersión durante las fases activas (Slices 12 al 14 de Frontend) y servir como hoja de ruta si existe disponibilidad de tiempo o para su inclusión en la sección de *Trabajos Futuros / Recomendaciones* del documento de Proyecto de Grado.

---

## 2. Registro de Ítems y Costuras Técnicas

| ID | Título de la Costura | Tipo | Origen | Descripción / Alcance Técnico | Prioridad |
|:---:|:--- |:---:|:---:|:--- |:---:|
| **COST-01** | Sincronización de Layout Multidispositivo | Frontend / Backend | Día 2 (UI Estudiante) | Persistir en base de datos PostgreSQL la disposición personalizada de los widgets del Dashboard (`CDK DragDrop`) mediante un endpoint dedicado (`PUT /api/v1/users/me/preferences/layout`), reemplazando el `localStorage` actual. | Media |
| **COST-02** | Pipelines CI/CD DevSecOps | Infraestructura | Auditoría de Inventario | Implementación de flujos automatizados en `.github/workflows/` para ejecución de tests de integración en Go, linter `golangci-lint`, compilación Angular y escaneo de vulnerabilidades con Semgrep en cada `push` / `PR`. | Media |
| **COST-03** | Panel White-Label Institucional | BaaS / Adm. | Día 2 (UI Estudiante) | Interfaz gráfica administrativa para que el autoridad/rector suba dinámicamente el logotipo institucional, favicons y personalice la paleta de colores (`--tenant-primary`) directamente desde la web. | Baja |
| **COST-04** | Perfiles Seccomp y AppArmor para Ejecución | Seguridad | Capacidad BaaS #03 | Aplicación de perfiles restrictivos de syscalls en el Host para limitar la superficie de ataque dentro de los contenedores efímeros del Juez Virtual. | Baja |
| **COST-05** | Sanitización y Truncado de Entradas/Salidas (64KB) | Seguridad / Juez | Capacidad BaaS #07 | Truncado estricto de buffers `stdin`/`stdout` a 64KB máximo en las ejecuciones del Juez Virtual para mitigar consumo desmedido de memoria RAM ante salidas masivas. | Media |
| **COST-06** | Sistema de Reporte de Problemas / Ayuda en Laboratorio | Frontend / Backend | Día 2 (UI Estudiante) | Implementación de tabla relacional `support_tickets` y endpoints REST (`POST /api/v1/support/tickets`) para permitir al estudiante reportar problemas técnicos directamente desde la vista del laboratorio hacia el docente/soporte. | Baja |

---

## 3. Criterio de Selección y Ejecución

* **Fase Actual (Prioridad 1):** Culminar los Slices de Frontend (Slice 12: Estudiante, Slice 13: Docente, Slice 14: Admin) para consolidar la plataforma funcional de punta a punta.
* **Fase de Extensión (Prioridad 2 - Opcional):** Si el cronograma de trabajo concluye antes de la fecha límite, se abordarán secuencialmente los ítems `COST-01` a `COST-05`, generando su respectivo Slice, ADR y suite de pruebas.
