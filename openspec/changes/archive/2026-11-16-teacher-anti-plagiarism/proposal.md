# Proposal: Centro de Supervisión y Auditoría Anti-Plagio AST

## Contexto y Motivación
La integridad académica en laboratorios de programación requiere que el docente pueda supervisar en tiempo real y auditar similitudes estructurales en el código de las entregas de sus estudiantes mediante análisis de Árboles de Sintaxis Abstracta (AST) y Semgrep.
Actualmente existía un modal básico por materia, pero el docente necesita una vista completa de supervisión en `/teacher/anti-plagio` para:
1. Auditar transversalmente todas sus materias o filtrar por materia y laboratorio específico.
2. Visualizar métricas globales de riesgo (coincidencias críticas, advertencias y estudiantes involucrados).
3. Comparar lado a lado las coincidencias de AST y acceder directamente al SpeedGrader para emitir veredictos o comentarios.

## Alcance
- Crear `TeacherPlagiarismComponent` en `frontend/src/app/features/teacher/supervision/anti-plagio/teacher-plagiarism.component.*`
- Conectar la ruta `/teacher/anti-plagio` en `teacher.routes.ts`.
- Diseñar la vista con KPIs, selector reactivo de materia/laboratorio, lista de coincidencias y panel de comparación estructural.
- Mantener compatibilidad con el modal rápido `PlagiarismModalComponent`.
- Validar con pruebas unitarias y aplicar tokens de diseño estrictos.
