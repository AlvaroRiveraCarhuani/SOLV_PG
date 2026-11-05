# ADR 062: Auditoría de Similitud por Práctica con Código Base Restado

> **Estado:** Implementado
> **Fecha:** 2026-11-02
> **Ámbito:** Docencia, auditoría de similitud, modalidad dual
> **Relacionado:** ADR-052 (banco de ejercicios y modalidad), ADR-059 (entornos por práctica), ADR-060 (comentarios por rango), [REVISION_POR_MODALIDAD](../../UI/DOCENTE/REVISION_POR_MODALIDAD.md)
>
> **Terminología vigente:** ADR-067. Los nombres de modalidad que aparecen en este ADR son los de su fecha y no se reescriben.

---

## Contexto

El motor de similitud comparaba las entregas completas. En una práctica con `exercises.boilerplate`,
dos alumnos que solo pegaban el código base aparecían con 100% de similitud y entraban al reporte como
pares sospechosos. El problema existía en las dos modalidades y el repositorio ni siquiera leía la
columna `boilerplate`, así que ninguna capa podía restarla.

Además, el reporte exponía `common_structures` con los tokens normalizados del compilador (`$LIT`,
`$ID`, `KW:RETURN`). El docente no puede abrir una línea del fuente con eso. La numeración tampoco era
confiable: `removeComments` descartaba las líneas de los bloques `/* ... */` y las líneas que quedaban
vacías al quitar un `//`, así que todos los tokens posteriores se corrían.

Por último, el acceso vivía en `SUPERVISIÓN → Control de Plagio` como listado global por materia. La
decisión que la práctica tiene código base y entregas concretas no llegaba a la pantalla: el docente
debía volver a elegir materia y práctica.

## Decisión

### 1. El código base se resta antes de calcular

`GetExerciseSubmissionsForPlagiarism` devuelve `boilerplate` y `environment_type` junto con el código.
El motor resta la base antes de generar los k-gramas de cada entrega:

- **Evaluador automático:** se tokeniza `boilerplate` y se descartan de la entrega los tokens cubiertos
  por cualquier k-grama de la base.
- **IDE:** se parsean `submissions.code` y `exercises.boilerplate` con `domain.ParseProjectSnapshot` y
  se resta archivo por archivo, con la misma ruta. Si la base no tiene cabeceras, se resta de forma
  global. Si la entrega no se puede parsear, también se resta global.

Si después de restar quedan menos de `KGramSize` tokens en alguna de las dos entregas, la similitud es
0 y el par no se reporta. La ventana de sustracción se acorta cuando la base tiene menos tokens que
`KGramSize`, para que una base corta no quede sin efecto.

### 2. La numeración de líneas no se toca

`removeComments` escribe exactamente una línea de salida por cada línea de entrada. Un bloque de
comentario deja una línea vacía en lugar de desaparecer, y una línea con solo `//` también conserva su
posición. `Token.Line` sigue siendo 1-based sobre el código original y es la base de todos los rangos.

### 3. El reporte entrega rangos, no tokens

`PlagiarismMatch` reemplaza `common_structures` por `ranges`:

```json
{
  "source_a": { "file_path": "src/tree.py", "start_line": 10, "end_line": 24 },
  "source_b": { "file_path": "src/tree.py", "start_line": 12, "end_line": 26 },
  "token_count": 140,
  "exact_match": true
}
```

- `file_path` solo existe en modalidad IDE y las líneas son locales a ese archivo. En el evaluador
  automático el campo se omite y las líneas son las del envío.
- Los tramos salen de una pasada voraz sobre los k-gramas coincidentes: se extienden hacia atrás y
  adelante mientras los tokens sean iguales, se conservan los maximales y se descartan los solapados.
- Se devuelven como máximo 10 tramos por par, en el orden del fuente.
- En IDE, un tramo se acota al archivo de su primera línea; una coincidencia que cruza archivos no se
  parte en varios tramos en esta entrega.

`exact_match` indica que el tramo coincide token a token después de la normalización (identificadores
y literales incluidos en la normalización estructural). El visor de la entrega 2 mostrará el código
fuente usando estos rangos.

### 4. La auditoría se abre desde la práctica

- La entrada `SUPERVISIÓN → Control de Plagio` sale del sidebar docente; la sección desaparece porque
  queda sin entradas.
- La ficha de la práctica muestra `[Auditoría de Similitud]` para juez e IDE.
- La ruta es `/teacher/cursos/:courseId/practicas/:exerciseId/similitud`. La pantalla no tiene
  combobox de materia ni de práctica: el contexto viene de la ruta y el botón vuelve a escanear esa
  práctica.
- `/teacher/anti-plagio` redirige a `/teacher/por-calificar`. No queda un listado global.

### 5. El par se abre con un contrato propio

El visor necesita el código de las dos entregas, no solo el resumen del reporte. `GET
/api/v1/teacher/plagiarism/pairs/{submission_a}/{submission_b}` devuelve las dos entregas, el contexto
compartido de la práctica, los tramos del par y la telemetría de escritura resumida.

- La consulta filtra por `tenant_id` y por `subjects.teacher_id`; el rol `admin` consulta sin filtro de
docente. Si una entrega no pertenece a una materia del docente, la respuesta es `403` y no se distingue
de una entrega inexistente.
- Las dos entregas deben ser de la misma práctica: si no, `422`. El mismo identificador repetido es
  `400`.
- No se devuelven `test_cases`, rúbrica, comentarios ni eventos de teclado. Las métricas son totales:
  tiempo activo, keystrokes, eventos de pegado y porcentaje pegado. Sin eventos, `events_available` es
  `false` y la vista no inventa un porcentaje.
- En IDE cada lado trae `files` con el snapshot parseado y el ejercicio trae `base_files`. En el
  evaluador cada lado trae `code`. El visor no vuelve a bajar el proyecto base desde el editor.

### 6. El visor enfrenta las dos entregas en tres columnas

La pantalla conserva el reporte y los filtros, y agrega el espacio de revisión:

| Columna | Evaluador automático | IDE persistente |
| --- | --- | --- |
| Izquierda | Pares y resumen de cada entrega: veredicto, complejidad, tiempo y memoria | Pares y árbol de archivos afectados; la base descartada queda plegada |
| Centro | Código de la entrega A con los tramos resaltados | Los dos archivos enfrentados, con tramos resaltados por lado y scroll sincronizado |
| Derecha | Código de la entrega B, métricas del juez y acciones de calificación | Cadencia de escritura de ambas entregas, nota de decisión docente y acciones de calificación |

- El alternador de contraparte aparece cuando otra entrega del reporte comparte uno de los dos
  identificadores: permite recorrer el cluster sin volver al listado.
- `code-editor` recibe `highlightRanges` y `scrollRatio`. El resaltado es una decoración de línea; el
  scroll se comparte como fracción 0..1 y se puede desacoplar.
- Los veredictos conservan su semántica (`AC` verde, `WA` rojo, `TLE` ámbar, `RE` naranja,
  `AST_BLOCKED` rojo intenso) y se acompañan de texto para el docente.

## Contrato de API

Las rutas existentes se conservan:

| Método | Ruta | Uso |
| --- | --- | --- |
| `GET` | `/api/v1/teacher/plagiarism?subject_id=` | Reporte por materia (sin cambios de forma) |
| `GET` | `/api/v1/teacher/courses/{id}/plagiarism?exercise_id=` | Reporte acotado a una práctica |
| `GET` | `/api/v1/teacher/plagiarism/pairs/{submission_a}/{submission_b}` | Documento del visor para un par de la misma práctica |

`PlagiarismReport` conserva `subject_id`, `exercise_id`, `exercise_title`, `analyzed_at`,
`total_submissions`, `suspect_pairs_count` y `matches`. Cada `PlagiarismMatch` conserva los
identificadores de las dos entregas y estudiantes, `similarity`, `risk_level`, `matching_tokens`,
`total_tokens_a` y `total_tokens_b`. El campo `common_structures` desaparece y en su lugar llega
`ranges`.

## Consecuencias

- Dos entregas que solo comparten el código base ya no aparecen en el reporte.
- Una copia real sigue superando el umbral y ahora se puede ubicar en el archivo y las líneas del
  fuente.
- El conteo de tokens y el porcentaje bajan cuando la base era gran parte del proyecto: el número
  ahora mide lo que el alumno escribió, no lo que la plantilla le dio.
- La pantalla deja de ser un listado global. Los enlaces viejos a `/teacher/anti-plagio` no rompen:
  vuelven a la cola de revisión.
- El visor de tres columnas enfrenta las dos entregas sin salir de la práctica y muestra la base
  descartada como contexto, no como coincidencia.
- El endpoint del par vuelve a comparar solo esas dos entregas; no reutiliza el reporte completo ni
  recalcula todos los pares.
- El costo del motor crece con la sustracción y con la búsqueda de tramos; el tope de candidatos y de
  tramos acota el peor caso de un par.

## Alternativas descartadas

| Alternativa | Motivo |
| --- | --- |
| Comparar el código completo y filtrar en la UI | El falso positivo sigue consumiendo el reporte y el docente no puede distinguir la plantilla de la copia |
| Restar la base global en IDE | Borra coincidencias legítimas entre archivos distintos y oculta copias reales |
| Seguir devolviendo `common_structures` | Los tokens del compilador no permiten abrir el fuente y obligan al docente a interpretar jerga interna |
| Conservar `/teacher/anti-plagio` como listado | Mantiene dos modelos de acceso y el docente vuelve a elegir materia y práctica |
| Dejar `removeComments` como estaba | Cualquier rango posterior a un bloque comentado apunta a la línea equivocada |

## Verificación

- `go build ./...`, `go vet ./...` y `go test ./internal/core/services ./internal/delivery/http ./internal/infrastructure/storage/postgres`.
- Tests nuevos del motor: base 100% igual no reporta, copia parcial devuelve el tramo esperado,
  comentarios multilínea no desplazan líneas y un IDE de dos archivos devuelve rangos en `main.py`.
- Tests del par: lados y métricas, prácticas distintas (`422`), entrega ajena (`403`), archivos IDE
  separados y ambos lados en la respuesta HTTP.
- Specs del frontend de la auditoría: carga del par, árbol de archivos, rangos por archivo, cadencia y
  traducción de veredictos; más el modal de evaluaciones.
- `CI=true npm run build` y `npm run lint:styles`.
