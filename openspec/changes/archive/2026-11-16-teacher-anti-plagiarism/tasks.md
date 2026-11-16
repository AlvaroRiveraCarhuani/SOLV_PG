# Implementation Tasks: Auditoría Anti-Plagio AST

## Phase 1: Implementación de la Vista de Supervisión
- [x] 1.1 Crear `TeacherPlagiarismComponent` (`.ts`, `.html`, `.scss`) en `frontend/src/app/features/teacher/supervision/anti-plagio/`.
- [x] 1.2 Implementar control bar con selectores de materia y laboratorio, KPIs y panel dual (lista de matches + visor AST).
- [x] 1.3 Registrar la ruta `/teacher/anti-plagio` en `teacher.routes.ts`.

## Phase 2: Verificación y Calidad
- [x] 2.1 Escribir suite de pruebas unitarias en `teacher-plagiarism.component.spec.ts`.
- [x] 2.2 Ejecutar `npm run lint:styles` para validar el gate de estilos.
- [x] 2.3 Ejecutar `npm run test:ci` para asegurar el 100% de tests en verde.
