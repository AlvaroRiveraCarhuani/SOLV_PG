# Implementation Tasks: Banco de Ejercicios y Plantillas Docentes

## Phase 1: Banco de Ejercicios (`/teacher/exercises`)
- [x] 1.1 Crear `TeacherExercisesComponent` (`.ts`, `.html`, `.scss`) con KPIs, buscador, filtros por materia/modalidad/estado y listado en grid.
- [x] 1.2 Integrar los modales `ExerciseEditorModalComponent` y `FuzzingModalComponent` para edición y generación de casos.
- [x] 1.3 Registrar ruta en `teacher.routes.ts`.
- [x] 1.4 Escribir suite de pruebas unitarias en `teacher-exercises.component.spec.ts`.

## Phase 2: Catálogo de Plantillas (`/teacher/templates`)
- [x] 2.1 Crear `TeacherTemplatesComponent` (`.ts`, `.html`, `.scss`) con listado de plantillas Docker, specs de memoria/CPU y filtros por categoría.
- [x] 2.2 Integrar `TemplateRequestModalComponent` para la solicitud formal de plantillas al administrador.
- [x] 2.3 Registrar ruta en `teacher.routes.ts`.
- [x] 2.4 Escribir suite de pruebas unitarias en `teacher-templates.component.spec.ts`.

## Phase 3: Verificación y Calidad
- [x] 3.1 Ejecutar `npm run lint:styles` para validar el gate de estilos y tipografía.
- [x] 3.2 Ejecutar `npm run test:ci` para asegurar que el 100% de los tests pasen en verde.
