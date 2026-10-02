# Technical Design: Banco de Ejercicios y Catálogo de Plantillas Docente

## Architecture Overview
La gestión de contenidos se compone de dos vistas standalone con Signals nativos en Angular 22:
- `TeacherExercisesComponent`: Centraliza el catálogo de laboratorios existentes y la creación/edición mediante modales asistentes.
- `TeacherTemplatesComponent`: Catálogo de entornos de ejecución y disparador de solicitudes al administrador.

## Component Structure & Data Flow

```
[TeacherLayoutComponent]
   ├── /teacher/exercises ──> [TeacherExercisesComponent]
   │                             ├── Filters & KPIs (Signals)
   │                             ├── [ExerciseEditorModalComponent] (Wizard de Creación/Edición)
   │                             └── [FuzzingModalComponent] (Generador de Casos)
   │
   └── /teacher/templates ──> [TeacherTemplatesComponent]
                                 ├── Filters & Search (Signals)
                                 ├── Template Grid (Docker specs, RAM/CPU)
                                 └── [TemplateRequestModalComponent] (Solicitud formal al Admin)
```

## State Management
- `TeacherCourseService`: Extendido para recuperar laboratorios agregados por materia y plantillas de runtime.
- Computed Signals para la reactividad en tiempo real de búsqueda y filtrado combinado.
