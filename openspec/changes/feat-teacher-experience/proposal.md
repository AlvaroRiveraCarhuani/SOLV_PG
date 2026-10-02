# Proposal: Experiencia y Flujo de Trabajo del Docente

## Intent

Implementar la interfaz de usuario para el rol Docente en Angular 22, permitiendo a los profesores gestionar cursos, crear y publicar ejercicios con casos de prueba, y evaluar entregas de estudiantes mediante el flujo de corrección continua (SpeedGrader) conectado a los contratos BaaS existentes (ADR-026).

## Scope

### In Scope
- **Dashboard Docente (`/teacher/dashboard`):** Reporte de incidentes por severidad (OOM-Killed, AST Blocked, Pendientes) y métrica de alumnos en riesgo (`at_risk`).
- **Gestión del Curso y Ejercicios (`/teacher/cursos/:id`):** Vista de cuaderno, alta/edición de ejercicios en estados `draft` / `published`, e importación de casos de prueba vía CSV (RFC 4180).
- **SpeedGrader y Visor de Código (`/teacher/revision/:submissionId`):** Navegación secuencial (`prev`/`next`), desenmascaramiento de casos de prueba privados (ADR-012), comentarios anclados a líneas de código y ejecución efímera.
- **Exportación de Calificaciones:** Descarga de matriz académica en formato CSV UTF-8 BOM.

### Out of Scope
- Entorno interactivo y workspace para estudiantes.
- Sistema de notificaciones vía WebSocket/Email en tiempo real.
- Modificación a endpoints de Backend (el BaaS ya está implementado y verificado con tests de integración).

## Capabilities

### New Capabilities
- `teacher-management-ui`: Flujos de interfaz gráfica para el rol docente, incluyendo dashboard de excepciones, autoría de ejercicios y calificación SpeedGrader.

### Modified Capabilities
- None

## Approach

Construir el módulo de rutas bajo `frontend/src/app/features/teacher/` con arquitectura standalone zoneless y reactividad pura con Signals. Reutilizar componentes compartidos de diseño (`kpi-card`, `search-bar`, `status-tabs`, `pagination-bar`) respetando los tokens de diseño (cero clases `solv-`, cero valores hex o tipografía hardcodeada en SCSS). Conectar a los endpoints de `/api/v1/teacher/*` respetando la propagación de headers y validaciones fail-closed.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `frontend/src/app/features/teacher/` | New | Vistas, componentes, servicios y contratos del rol docente |
| `frontend/src/app/app.routes.ts` | Modified | Registro del guard y rutas hijas `/teacher/*` |
| `frontend/src/app/core/guards/` | Modified | Verificación de acceso para rol `teacher` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Manejo complejo del estado en navegación continua de SpeedGrader | Med | Centralizar el cursor de navegación (`prev_submission_id`, `next_submission_id`) en un `TeacherGradingService` basado en Signals. |
| Inconsistencias de estilo con el Design System | Low | Ejecutar `npm run lint:styles` y verificar que los selectores y clases no usen prefijo `solv-`. |

## Rollback Plan

Revertir los commits del frontend asociados a la ruta `/teacher` y desregistrar las rutas en `app.routes.ts`. El backend permanece intacto e idempotente.

## Dependencies

- Endpoints de BaaS (`/api/v1/teacher/*`, `/api/v1/subjects/*`) ya operativos y probados.

## Success Criteria

- [ ] Rutas `/teacher/dashboard`, `/teacher/cursos/:id` y `/teacher/revision/:id` operativas y accesibles con rol docente.
- [ ] SpeedGrader permite calificar, navegar entregas y enviar comentarios por línea.
- [ ] Creación y publicación de ejercicios con importación de casos de prueba CSV validada.
- [ ] Build de frontend limpio (`npm run build`) y estilos conformes (`npm run lint:styles`).
