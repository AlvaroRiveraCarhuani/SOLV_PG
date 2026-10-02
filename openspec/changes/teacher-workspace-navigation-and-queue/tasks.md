# Implementation Tasks: Navegación Docente y Cola Centralizada de Entregas

## Phase 1: Navegación y Estructura
- [x] 1.1 Actualizar `TEACHER_NAV_SECTIONS` en `frontend/src/app/features/teacher/layout/teacher-layout.component.ts` con las 3 secciones semánticas.
- [x] 1.2 Registrar rutas en `teacher.routes.ts` para `/teacher/evaluaciones` y vistas de contenido complementario.

## Phase 2: Cola de Entregas y Evaluaciones
- [x] 2.1 Extender `TeacherCourseService` para obtener la agregación de entregas de todas las materias del docente.
- [x] 2.2 Diseñar `TeacherEvaluationsComponent` (`.ts`, `.html`, `.scss`) con KPIs, barra de filtros reactivos y tabla con enlace a SpeedGrader.
- [x] 2.3 Aplicar tokens de diseño, directiva `machineData` y pipe `dateText` según las reglas del proyecto.

## Phase 3: Verificación y Calidad
- [x] 3.1 Escribir pruebas unitarias en `teacher-evaluations.component.spec.ts`.
- [x] 3.2 Ejecutar `npm run lint:styles` para validar el gate de estilos.
- [x] 3.3 Ejecutar `npm run test:ci` para asegurar que el conjunto de tests pase en verde.
