# Implementation Tasks: Personalización de Dashboards y Tematización

## Phase 1: Servicio Central de Layouts y Bento Grid Docente
- [ ] 1.1 Crear `DashboardLayoutService` en `frontend/src/app/core/services/dashboard-layout.service.ts` con soporte de merge, persistencia y reset.
- [ ] 1.2 Refactorizar `TeacherDashboardComponent` (`.ts`, `.html`, `.scss`) para usar Bento Grid de 12 columnas con drag-and-drop y selector de ancho (33%, 50%, 66%, 100%).
- [ ] 1.3 Escribir pruebas unitarias en `dashboard-layout.service.spec.ts` y `teacher-dashboard.component.spec.ts`.

## Phase 2: Bento Grid en Estudiante y Tematización Híbrida de Cursos
- [ ] 2.1 Refactorizar `StudentDashboardComponent` (`.ts`, `.html`, `.scss`) para integrar Bento Grid configurable.
- [ ] 2.2 Integrar selector híbrido de color de curso (8 presets curados + color picker libre) en los componentes de creación/edición de cursos.
- [ ] 2.3 Escribir pruebas unitarias en `student-dashboard.component.spec.ts`.

## Phase 3: Verificación y Calidad
- [ ] 3.1 Ejecutar `npm run lint:styles` para validar el gate de estilos y tipografía.
- [ ] 3.2 Ejecutar `npm run test:ci` para asegurar el 100% de tests en verde.
