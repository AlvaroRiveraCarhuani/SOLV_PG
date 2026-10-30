# Specification: Banco de Ejercicios y Catálogo de Plantillas Docente

## Requirements

### R1: Banco Global de Ejercicios (`/teacher/exercises`)
1. El componente `TeacherExercisesComponent` debe consumir el read model `GET /api/v1/teacher/exercises/bank`, que agrega en una sola lista todos los ejercicios y laboratorios de las materias asignadas al docente, con datos reales: `environment_type`, `type`, lenguaje, límites RAM/tiempo, materia (código/nombre) y métricas de entrega.
2. Debe proveer KPIs agregados:
   - Total de Ejercicios
   - Laboratorios Publicados
   - Borradores en Edición
   - Total de Entregas Recibidas
3. Debe permitir filtrar por:
   - Materia / Curso asignado
   - Entorno (`environment_type`): `JUEZ_EFIMERO` (Juez Virtual) y `IDE_PERSISTENTE` (IDE Persistente)
   - Estado (`all`, `published`, `draft`)
   - Búsqueda textual por título o descripción del ejercicio.
4. La modalidad se deriva únicamente de `environment_type`; la columna `type` queda acotada a `algorithm | database | project` (CHECK en base de datos, migración `00033`).
5. El contador de revisión por tarjeta usa `review_pending` con semántica por entorno: juez → veredictos `WA/RE/TLE` sin override; IDE → entregas `PENDING` en la cola de SpeedGrader.
6. Cada tarjeta o fila de ejercicio debe exponer:
   - Título, entorno (Juez Virtual / Workspace Persistente), lenguaje/runtime real, límites de recursos (RAM/tiempo).
   - Cantidad de entregas y `review_pending`.
   - Acciones: **Fuzzing solo para entornos `JUEZ_EFIMERO`** (abre `FuzzingModalComponent`), Editar ejercicio (abre el editor), e Ir al curso.
7. Los casos de prueba I/O y el fuzzing son exclusivos del juez virtual: `BulkAddTestCases` rechaza ejercicios `IDE_PERSISTENTE` (error `ErrTestCasesRequireJudge`), que se corrigen con rúbrica en SpeedGrader.

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
