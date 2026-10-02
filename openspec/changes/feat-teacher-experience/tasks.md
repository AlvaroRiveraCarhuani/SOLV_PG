# Tasks: Experiencia y Flujo de Trabajo del Docente

## Phase 1: Foundation & Contracts

- [x] 1.1 Crear modelos de datos e interfaces de TypeScript en `frontend/src/app/features/teacher/models/teacher.models.ts`
- [x] 1.2 Implementar servicios de API y Signals Store en `frontend/src/app/features/teacher/services/teacher-dashboard.service.ts`, `teacher-course.service.ts` y `teacher-grading.service.ts`
- [x] 1.3 Configurar las rutas y layout con topbar/sidebar docente en `frontend/src/app/features/teacher/teacher.routes.ts` y `teacher-layout.component.ts`

## Phase 2: Teacher Dashboard

- [x] 2.1 Implementar `TeacherDashboardComponent` con tarjetas de incidentes (`critical`, `warning`, `standard`) y lista de materias asignadas
- [x] 2.2 Agregar indicador visual y contador para alumnos en estado `at_risk`

## Phase 3: Course & Exercise Management

- [x] 3.1 Implementar `TeacherCourseDetailComponent` con pestañas de Cuaderno/Ejercicios, Estudiantes matriculados y Libro de Notas
- [x] 3.2 Implementar modal `ExerciseEditorModalComponent` con validación de estados `draft` / `published` (mínimo 1 caso público obligatorio)
- [x] 3.3 Integrar importación de casos de prueba CSV con reporte de errores por número de línea
- [x] 3.4 Conectar acción de exportación de matriz de calificaciones a CSV con codificación UTF-8 BOM

## Phase 4: SpeedGrader & Code Evaluation

- [x] 4.1 Implementar `SpeedGraderComponent` con visor de código y navegación secuencial (`prev_submission_id` / `next_submission_id`)
- [x] 4.2 Renderizar resultados de ejecución con desenmascaramiento de casos de prueba privados para el docente
- [x] 4.3 Implementar creación y renderizado de comentarios in-line anclados a líneas de código (`line_number`)

## Phase 5: Verification & Quality Gates

- [x] 5.1 Validar gate de estilos SCSS (`npm run lint:styles`) verificando cero colores hex y sin selectores `solv-`
- [x] 5.2 Validar compilación limpia del frontend (`npm run build`)
