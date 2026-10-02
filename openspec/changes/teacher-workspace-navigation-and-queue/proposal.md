# Proposal: Expansión de Navegación Docente y Cola Global de Entregas

## Intent
Enriquecer la experiencia del profesor en SOLV transformando el menú lateral en una barra de navegación completa dividida en 3 áreas temáticas (Docencia, Contenidos y Supervisión) y dotando al módulo de una vista centralizada de Entregas y Evaluaciones (`/teacher/evaluaciones`) con filtros globales y acceso directo a SpeedGrader.

## Scope
### In Scope
- Reestructuración de `TEACHER_NAV_SECTIONS` en `TeacherLayoutComponent` con 3 secciones semánticas:
  - `DOCENCIA`: Panel de Control (`/teacher/dashboard`), Evaluaciones & Entregas (`/teacher/evaluaciones`).
  - `CONTENIDOS`: Banco de Ejercicios (`/teacher/exercises`), Plantillas (`/teacher/templates`).
  - `SUPERVISIÓN`: Anti-Plagio AST (`/teacher/anti-plagio`).
- Actualización de `TeacherEvaluationsComponent` (`/teacher/evaluaciones`) para convertirlo en una cola centralizada de corrección de entregas de todas las materias asignadas.
- Filtros por materia/curso, veredicto (AC, WA, TLE, RE, AST_BLOCKED) y estado de revisión.
- KPIs de corrección docente (Total entregas, Pendientes de revisión manual, Auto-calificadas, Casos en riesgo).
- Enlace directo a `SpeedGraderComponent` (`/teacher/revision/:submissionId`).

### Out of Scope
- Implementación de la base de datos para banco de ejercicios (Fase 2).
- Motor de comparación AST entre múltiples entregas en lote (Fase 3).

## Approach
1. Actualizar `TEACHER_NAV_SECTIONS` y el componente de sidebar para reflejar la navegación completa y placeholders informativos para secciones de fases siguientes.
2. Extender `TeacherCourseService` / `TeacherDashboardService` para proporcionar la lista agregada de entregas docentes con filtrado reactivo mediante Angular Signals y Computed.
3. Diseñar `TeacherEvaluationsComponent` con diseño responsivo usando tokens semánticos de SOLV, directiva `machineData` y componentes compartidos (`kpi-card`, `skeleton-loader`, `form-field`).
4. Validar con tests unitarios en frontend y verificar el gate de estilos (`lint:styles`).

## Rollback Plan
Si se detecta alguna regresión, revertir los cambios de rutas y `TEACHER_NAV_SECTIONS` mediante Git sin afectar la lógica core de cursos ni el backend.
