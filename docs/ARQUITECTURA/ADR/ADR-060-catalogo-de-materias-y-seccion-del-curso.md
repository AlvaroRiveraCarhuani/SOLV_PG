# ADR 060: Catálogo de Materias Separado de la Oferta del Periodo

> **Estado:** Implementado
> **Fecha:** 2026-11-04
> **Ámbito:** Academia, Core, Base de datos
> **Relacionado:** ADR-057 (canvas de materias), ADR-059 (entornos por práctica)

---

## Contexto

`subjects` cumplía dos papeles a la vez. Era el catálogo —el código `SIS-211` y el nombre
"Estructuras de Datos y Algoritmos"— y era la oferta: docente, periodo académico e inscritos
(`00001_baseline.sql:266`). Los niveles de materia y de curso quedaron colapsados en una sola fila,
mientras que el tercer nivel, la clase, sí existía aparte (`class_sessions`).

El colapso funciona mientras hay una carrera con una sola oferta por materia y periodo. Deja de
funcionar en tres puntos concretos:

1. **No hay continuidad entre periodos.** `SIS-211` de 2026-1 y de 2026-2 son dos filas sin
   parentesco. No se puede preguntar cómo le fue a un estudiante en Estructuras de Datos atravesando
   semestres, ni sacar un histórico por materia.
2. **Nada impide duplicados.** No existe `UNIQUE(tenant_id, code)` ni equivalente. Dos filas
   "SIS-211" en el mismo periodo son posibles, y el estudiante vería la materia dos veces en su
   portada sin ningún error que lo delate.
3. **El término está indefinido.** El campo `classroom_course_id` muestra el origen: en Google
   Classroom "course" es la sección. "Curso" entró con la integración y nunca se definió. La propia
   portada del docente dice "Mis cursos" en el nav y "Materias del período" en el `aria-label`.

## Decisión

### 1. El catálogo es una tabla nueva; la oferta sigue siendo `subjects`

`subject_catalog` guarda `tenant_id`, `code` y `name`, con `UNIQUE(tenant_id, upper(code))`. `subjects`
gana `subject_catalog_id` y `section`.

No se crea una tabla de secciones: `section` es una columna del curso. Un curso se identifica por
materia, periodo y sección, y el índice único `ux_subjects_course_identity` lo hace cumplir sobre las
filas no archivadas. Los cursos archivados quedan fuera para no bloquear el alta del periodo
siguiente.

### 2. `subjects.name` y `subjects.code` son el snapshot del curso

No se eliminan. Si el catálogo se renombra, los cursos anteriores conservan el nombre con el que se
dictaron, igual que una factura conserva el precio del momento. Esto evita además reescribir las
noventa y cuatro consultas que hoy leen `s.code` y `s.name` desde `subjects`: el SQL es texto y un
cambio de columna fallaría en ejecución, no en compilación.

El alta de un curso hereda el nombre vigente del catálogo. El primer curso fija el nombre; los
siguientes lo toman del catálogo aunque el formulario traiga otro, para que dos cursos del mismo
código no se llamen distinto.

### 3. El código del curso se compone con la sección

Un curso de la materia `SIS-211` en la sección `1` se identifica como `SIS-211-1`. El código del
catálogo nunca lleva sección: dos paralelos son la misma materia dictada dos veces, y el vínculo entre
ambos es `subject_catalog_id`, no la igualdad de cadenas.

La sección se limita a letras y números porque viaja dentro del código visible, y se normaliza a
mayúsculas. El alta rechaza una sección con espacios o barras en lugar de generar un identificador
ilegible.

### 4. La migración se detiene si hay duplicados

`00047_subject_catalog` hace el alta del catálogo con `DISTINCT ON (tenant_id, upper(code))`,
prefiriendo el curso vigente y, entre iguales, el más reciente. Antes de crear el índice único revisa
las combinaciones repetidas de materia, periodo y sección. Si encuentra alguna, aborta con el conteo
y la consulta para corregirlas.

Es el único paso que puede bloquear un despliegue, y es deliberado: el índice único no admite atajos,
y elegir una fila por sorteo perdería la otra en silencio.

## Consecuencias

- El histórico por materia queda habilitado: dos cursos del mismo `subject_catalog_id` en periodos
  distintos ya son comparables.
- El alta de materia suma el campo de sección y muestra el identificador resultante antes de
  enviar. Una sección vacía produce el código limpio (`SIS-211`).
- Dos secciones del mismo periodo comparten una sola fila de catálogo y se distinguen por el código
  del curso y por `section`.
- `subjects.name` puede diferir del catálogo en los cursos archivados. Es lo esperado: son el
  registro de cómo se dictó cada periodo.
- La baja de un tenant arrastra su catálogo (`ON DELETE CASCADE`). No se agrega un motivo nuevo para
  que esa baja falle: `subjects` ya la protege con `RESTRICT`.
- Queda pendiente la edición del catálogo desde la interfaz y su propagación a los cursos vigentes.
  Hasta que exista, renombrar un curso cambia el snapshot y no el catálogo.
