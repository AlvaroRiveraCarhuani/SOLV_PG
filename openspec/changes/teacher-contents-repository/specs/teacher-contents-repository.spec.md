# Specification: Banco de Ejercicios y Catálogo de Plantillas Docente

## Requirements

### R1: Banco Global de Ejercicios (`/teacher/exercises`)
1. El componente `TeacherExercisesComponent` debe cargar y agregar en una sola lista reactiva todos los ejercicios y laboratorios pertenecientes a los cursos asignados al docente.
2. Debe proveer KPIs agregados:
   - Total de Ejercicios
   - Laboratorios Publicados
   - Borradores en Edición
   - Total de Entregas Recibidas
3. Debe permitir filtrar por:
   - Materia / Curso asignado
   - Modalidad (`ALGORITMO`, `DATABASE`, `WORKSPACE`)
   - Estado (`all`, `published`, `draft`)
   - Búsqueda textual por título o descripción del ejercicio.
4. Cada tarjeta o fila de ejercicio debe exponer:
   - Título, tipo de laboratorio, lenguaje/runtime, límites de recursos (RAM/tiempo).
   - Cantidad de entregas y estudiantes.
   - Acciones: Editar ejercicio (abre `ExerciseEditorModalComponent`), Generar Casos Fuzzing (abre `FuzzingModalComponent`), e Ir al curso.

### R2: Catálogo de Plantillas Docentes (`/teacher/templates`)
1. El componente `TeacherTemplatesComponent` debe consumir las plantillas disponibles vía servicio o API `/api/v1/templates`.
2. Debe permitir filtrar por categoría (Programación, Bases de Datos, Sistemas, Web) y buscar por nombre de runtime o etiqueta.
3. Debe mostrar especificaciones técnicas clave: Imagen base Docker, límites recomendados de memoria RAM y CPU, puertos expuestos y herramientas preinstaladas.
4. Debe contener un botón principal de acción "Solicitar Nueva Plantilla" que despliegue el modal `TemplateRequestModalComponent`.
5. Debe permitir crear un nuevo laboratorio con la plantilla preseleccionada.

### R3: Calidad y Gobernanza de UI
1. Todos los estilos SCSS deben adherirse estrictamente a los tokens del sistema (`_primitives.scss`), sin literales de color `#...` ni fuentes ad-hoc.
2. Los valores técnicos visibles deben usar `[machineData]` o la clase estática `font-mono`.
3. Las fechas deben renderizarse con el pipe `dateText` dentro de `<time>`.
4. El conjunto de pruebas unitarias debe cubrir la lógica de agregación, filtros y modales.
