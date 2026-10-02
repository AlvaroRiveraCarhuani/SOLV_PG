# Proposal: Banco Centralizado de Ejercicios y Catálogo de Plantillas Docentes

## Contexto y Motivación
El rol docente necesita gestionar contenidos pedagógicos de forma transversal sin estar limitado a la navegación jerárquica estricta de materia por materia. En particular:
1. **Banco de Ejercicios (`/teacher/exercises`)**: Permite visualizar, filtrar, crear y editar todos los laboratorios y ejercicios diseñados para sus cursos, ejecutar fuzzing asistido y revisar estadísticas de entrega.
2. **Catálogo de Plantillas (`/teacher/templates`)**: Permite explorar las imágenes y entornos de ejecución aprobados en la plataforma, consultar sus capacidades técnicas (RAM, CPU, puertos) y solicitar formalmente al administrador nuevas imágenes Docker mediante el asistente de solicitudes.

## Alcance
- Crear `TeacherExercisesComponent` (`/teacher/exercises`) con agregación multi-materia, filtros por modalidad/lenguaje/estado, KPIs y enlaces directos al asistente de creación (`ExerciseEditorModalComponent`) y de fuzzing (`FuzzingModalComponent`).
- Crear `TeacherTemplatesComponent` (`/teacher/templates`) con catálogo visual de plantillas aprobadas, filtros por tecnología, y disparador del modal de solicitud de plantilla (`TemplateRequestModalComponent`).
- Registrar las rutas en `teacher.routes.ts`.
- Garantizar el cumplimiento del sistema de diseño (tokens, `dateText`, `machineData`, cero colores hexadecimales en SCSS).
