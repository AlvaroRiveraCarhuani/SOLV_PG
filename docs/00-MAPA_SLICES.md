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
| **12** | Experiencia Estudiante, Shell y Juez en Tiempo Real | ADR-012, ADR-037, ADR-029 | Implementado | En Proceso | Verificado (PostgreSQL Real) |
| **13** | Experiencia Docente, Cursos y Creación de Laboratorios | ADR-026, ADR-029, ADR-030, ADR-037 | Implementado | **Implementado** | Verificado (Integración) |
| **14** | Panel Administrador Institucional & Gobernanza | ADR-024, ADR-027, ADR-029, ADR-030, ADR-031, ADR-032, ADR-033, ADR-036 | Implementado | **Implementado** | Verificado (Integración) |
| **15** | Notificaciones Proactivas (In-App & Email) | ADR-031, ADR-032, ADR-034, ADR-035 | Implementado | Pendiente | Verificado (Integración) |
| **16** | Backups y Retención Institucional | ADR-027, ADR-034, ADR-035 | Implementado | Pendiente | Verificado (Integración) |

---

## Detalle y Estado de Ejecución por Rol

### Consola del Administrador Institucional (Slice 14) — Completada

El backend cuenta con todos los endpoints de gobernanza registrados y verificados. En el frontend, el módulo del Administrador quedó completo vista por vista en el siguiente orden:

| Submódulo / Vista | Ruta Frontend | ADRs Base | Estado | Alcance Principal |
| :--- | :--- | :--- | :---: | :--- |
| **14.1 Salud y Recursos** | `/admin/dashboard` | ADR-027 | **Implementado** | Métricas en vivo de RAM/CPU, monitor de contenedores Docker e incidentes con resolución manual. |
| **14.2 Gestión de Docentes** | `/admin/docentes` | ADR-025, ADR-036 | **Implementado** | Alta e invitación (72h), roles titular/auxiliar, filtros de estado, atajo de copiado de correo y reasignación de materias asignadas. |
| **14.3 Cursos y Periodos Académicos** | `/admin/cursos` | ADR-024, ADR-029, ADR-036 | **Implementado** | Selector de periodos semestrales, alta y edición de cursos, asignación de docente titular y vinculación de plantilla Docker. |
| **14.4 Directorio de Estudiantes** | `/admin/estudiantes` | ADR-033 | **Implementado** | Búsqueda institucional de alumnos, cursos inscritos, monitor de 3 strikes OOM-Killed y reseteo manual justificado. |
| **14.5 Plantillas Docker** | `/admin/plantillas` | ADR-030 | **Implementado** | Catálogo oficial de imágenes, bandeja de aprobación/rechazo de plantillas docentes, fijación y edición de límites de RAM base, biblioteca de modelos y verificación de entorno. |
| **14.6 Configuración y Mantenimiento** | `/admin/configuracion` | ADR-027, ADR-029, ADR-031, ADR-035, ADR-038 | **Implementado** | Personalización institucional (logo por upload real, branding, tipografía white-label por catálogo ADR-038, correo soporte) con preview en vivo, gestión de períodos con archivado formal irreversible, políticas QoS configurables con recarga en worker, respaldos con verificación SHA-256 y switch de Modo Mantenimiento. Detalle en `docs/SLICES/SLICE_14/05-SUBMODULO_14.6_CONFIGURACION.md`. |
| **14.7 Auditoría y Emergencias** | `/admin/auditoria` | ADR-027, ADR-032 | **Implementado** | Tabla de audit logs con enriquecimiento semántico y drawer de cronología por actor, más centro de control de emergencias con las 5 acciones de ADR-032 (doble confirmación tipada, motivo obligatorio, registro EMERGENCY_* en auditoría). Detalle en `docs/SLICES/SLICE_14/07-SUBMODULO_14.7_AUDITORIA.md`. |

---

### Desglose Operativo Inmediato: 14.3 Cursos y Periodos Académicos (`/admin/cursos`)

Para no perder el hilo ni desviar el alcance, la vista 14.3 se ejecuta en los siguientes 4 pasos concretos:

1. **Rutas y Navegación:**
   - Registrar la ruta `/admin/cursos` en `app.routes.ts`.
   - Activar el ítem "Cursos" en la barra lateral (`admin-layout.component.ts`) bajo la sección "ACADÉMICO Y GESTIÓN".

2. **Capa de Servicios y Contratos (`admin-courses.service.ts`):**
   - Consumir endpoints de Periodos Académicos (`GET`, `POST`, `PUT /api/v1/admin/academic-periods`, ADR-029).
   - Consumir carga de cursos y materias (`GET /api/v1/admin/dashboard/courses-load` y `GET /api/v1/subjects`, ADR-024).
   - Alta de curso (`POST /api/v1/subjects`) con asignación de periodo, docente titular y plantilla Docker.
   - Reasignación de titular (`POST /api/v1/admin/courses/{id}/reassign`, ADR-036).

3. **Interfaz y Experiencia de Usuario (`AdminCoursesComponent`):**
   - **Barra superior:** Selector de Periodo Académico activo con botón para crear/cerrar semestres.
   - **Métricas rápidas:** Total de cursos en el periodo, estudiantes activos acumulados y consumo de memoria RAM agrupado.
   - **Tabla central:** Código de materia, nombre, docente titular asignado, alumnos inscritos, plantilla asociada y consumo de recursos.
   - **Modal de Alta de Curso:** Formulario con validación reactiva (Código, Nombre, Periodo, Docente titular, Plantilla Docker).
   - **Acción directa de reasignación:** Diálogo para cambiar de docente titular con justificación requerida por auditoría.

4. **Verificación y Calidad:**
   - Compilación en limpio verificando presupuesto SCSS (< 16 kB).
   - Verificación visual directa en navegador para validar que no existan inconsistencias de diseño.

---

### Siguientes Fases de Frontend

Una vez completadas las vistas del Administrador Institucional, el desarrollo continúa en este orden:

1. **Fase 2 — Experiencia Docente (Slice 13):**
   - `/teacher/dashboard`: Dashboard general de materias y alertas de revisión.
   - `/teacher/cursos/:id`: Vista de curso (el "cuaderno"), creación de hojas/laboratorios, fijación de plantillas y gestión de entregas.
   - `/teacher/revision/:submissionId`: Calificación manual, consola efímera y comentarios sobre código de alumnos.

2. **Fase 3 — Experiencia Estudiante (Slice 12):**
   - `/student/dashboard`: Vista de materias (los cuadernos), tareas próximas y entrega de ejercicios.
   - `/student/workspace/:id`: IDE web OpenVSCode embebido, consola de ejecución y veredicto del Juez Virtual vía WebSocket en tiempo real.
