# Vista: Revisión de entregas por modalidad

> **Especificación de interfaz y contrato de datos**
> **Rol:** Docente
> **Estado:** Propuesta para implementación
> **Gobernanza:** SOLV Design System / [UX-PRINCIPLES](../UX-PRINCIPLES.md) / [ADR-043](../../ARQUITECTURA/ADR/ADR-043-modalidad-dual-y-gobernanza-de-recursos.md) / [ADR-052](../../ARQUITECTURA/ADR/ADR-052-banco-ejercicios-read-model-y-discriminador-entorno.md) / [ADR-053](../../ARQUITECTURA/ADR/ADR-053-revision-docente-y-libreta-de-notas.md)
> **Ruta:** `/teacher/revision/:submissionId`
> **Reemplaza el plan de:** `proyecto-grado/Observaciones/Revision de Entregas e Inconsistencias en SpeedGrader.md` (nota local, fuera del repositorio)

---

## 1. Qué responde

El SpeedGrader responde dos preguntas distintas con la misma pantalla:

- **Juez (`JUEZ_EFIMERO`, "Evaluador Automático"):** ¿el algoritmo cumple los casos de prueba y los límites de ejecución?
- **IDE (`IDE_PERSISTENTE`, "Entorno de Desarrollo"):** ¿el proyecto cumple la rúbrica y se sostiene como sistema?

Hoy la pantalla no distingue. Una entrega IDE muestra "Casos de Prueba (0)", "0 ms", "0 MB", análisis de complejidad y un botón "Re-ejecutar" que no aplican. El código de un proyecto multiarchivo se presenta como un texto continuo con cabeceras `# === FILE: ... ===`, sin árbol para navegar. La rúbrica queda en la misma columna que esos bloques vacíos.

La decisión es una sola ruta y un solo shell, con dos paneles de revisión según `environment_type`. No se bifurca la ruta ni se duplica el header, los comentarios, el override, los atajos o el guardado de la rúbrica.

---

## 2. Decisión de arquitectura

- `SpeedGraderComponent` sigue siendo el shell: carga la entrega, resuelve el modo, registra atajos, guarda comentarios, override y rúbrica, y navega con J/K.
- Dos paneles de contenido:
  - `review-algorithm-panel`: visor de un archivo, casos de prueba, métricas, complejidad y veredicto.
  - `review-project-panel`: árbol de archivos, visor multiarchivo y rúbrica fija.
- Los nombres anteriores describen el contenido. La implementación puede conservar `speed-grader-code-viewer` y `speed-grader-eval-panel` con un input `mode`, o extraer dos componentes nuevos; en ambos casos el shell y el contrato son los mismos.
- El modo se deriva de `environment_type`, que viaja en el payload. La UI no lo infiere de la presencia de rúbrica ni de `test_cases.length`.
- El header y el panel de evaluación reciben `mode` como input. No leen el string del enum repartido por el template.
- El estado de revisión sigue el flujo de [ADR-053](../../ARQUITECTURA/ADR/ADR-053-revision-docente-y-libreta-de-notas.md): guardar la rúbrica deja la entrega en `graded` y actualiza la libreta.
- Los datos compartidos se cargan una vez: la entrega, los comentarios, la rúbrica y el estado de revisión.

### 2.1 Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Dos componentes o dos rutas completas | Duplica el shell, que es la mayor parte del componente, y rompe J/K, comentarios y override. |
| `@if` dispersos en el componente actual | Es el estado que produjo la fuga: cada elemento nuevo vuelve a olvidarse en una modalidad. |
| Inferir la modalidad por la rúbrica o por los casos | Un IDE sin rúbrica o un juez con 0 casos se clasifican mal. |
| Parsear el snapshot en el navegador | Duplica la lógica del writer y deja el contrato en dos lenguajes. |
| Incluir el entorno de prueba del alumno en esta entrega | Trae contenedores, cuotas, TTL y decisiones de seguridad; merece su propio diseño. |

---

## 3. Matriz de capacidades

**No aplica** significa que el bloque no se renderiza y, cuando es dato del backend, tampoco se envía.

| Elemento | Juez | IDE | Decisión |
| --- | --- | --- | --- |
| Veredicto y puntaje | Sí | No: `PENDING` es estado de revisión, no veredicto | Ocultar la tarjeta de veredicto en IDE. La nota sale de la rúbrica. |
| Tiempo y memoria | Sí | No | Ocultar en IDE. En juez, mostrar solo si `metrics_available`; si no, `—`, no `0`. |
| Casos de prueba | Sí | No | Ocultar en IDE y no enviarlos en el payload. |
| Complejidad asintótica | Sí | Hoy no hay datos | Ocultar en IDE. Si se agregan restricciones, generar el análisis antes de mostrarlo. |
| Rúbrica | No | Sí | Mostrar solo en IDE. El backend ya rechaza rúbricas en juez. |
| Diff contra la base | Sí, con el `boilerplate` real del ejercicio | Por archivo, fuera de alcance en esta entrega | Corregir la fuente del código base; no adivinarla con regex. |
| Evolución temporal | Sí, si hay datos | Sí, si hay datos | Mantener en ambas. |
| Auditoría de escritura | Sí, si hay datos | Sí, si hay datos | Mantener en ambas. |
| Consola del estudiante | No | Solo con un workspace real | Quitar la sesión fabricada; ocultar si no hay workspace vivo. |
| Re-ejecutar sandbox | Sí, con el lenguaje del ejercicio | No | Ocultar en IDE. El endpoint responde 409 en IDE. |
| Comentarios inline | Sí | Sí, por archivo visible | Mantener el número de línea global y traducir en la vista. |
| Calificar / override | Sí | Sí, como excepción | La acción primaria en IDE es guardar la rúbrica. |
| Estado de revisión | Sí | Sí | Guardar la rúbrica mueve la entrega a `graded` y actualiza la libreta. |

Notas de la matriz:

- En IDE, el `verdict` almacenado sigue siendo un detalle de persistencia. La interfaz no lo presenta como veredicto oficial.
- Un juez con 0 casos es un error de configuración, no un "no aplica". Su estado vacío debe decir que faltan casos, no que la modalidad no los usa.
- La consola del estudiante no se puede construir con los datos de la entrega. Hoy `currentLiveSession` inventa un workspace `running`; eso se elimina.

---

## 4. Estructura de la vista

### 4.1 Juez

Se mantiene la disposición actual de dos columnas:

- **Izquierda:** visor de código de un archivo, con comentarios inline y diff contra el `boilerplate` del ejercicio.
- **Derecha:** veredicto, puntaje, tiempo, memoria, casos de prueba y complejidad.
- **Header:** "Re-ejecutar" visible, con el lenguaje del ejercicio; herramientas de replay y auditoría si hay datos.
- **Rúbrica:** no se renderiza.

### 4.2 IDE

Tres columnas:

```text
┌──────────────────────┬─────────────────────────────────────┬────────────────────────┐
│ Archivos             │ Código del estudiante               │ Rúbrica                │
│ 240–280 px           │ flexible                             │ 360–420 px             │
│ árbol colapsable     │ CodeMirror readonly + comentarios   │ sticky, scroll propio  │
└──────────────────────┴─────────────────────────────────────┴────────────────────────┘
```

- **Izquierda:** árbol de archivos ordenado por ruta, carpetas colapsables, archivo seleccionado marcado. Las rutas usan `font-mono`; las etiquetas humanas, `--font-sans`.
- **Centro:** encabezado con ruta, lenguaje y cantidad de líneas; editor de solo lectura; comentarios inline de la línea seleccionada.
- **Derecha:** criterios y niveles de la rúbrica, comentarios generales, score ponderado y botón de guardado. La columna queda fija y con scroll independiente.
- **Header:** sin "Re-ejecutar". El menú de herramientas oculta complejidad; replay y auditoría quedan si hay datos; la consola solo aparece con un workspace real.
- **Sin rúbrica:** estado vacío explícito, guardado deshabilitado y enlace a la práctica para configurarla.
- **Sin archivos reconocibles:** se cae al visor plano de `code` con un aviso discreto; no se inventa un árbol.

### 4.3 Responsive y accesibilidad

- Hasta 1280 px, el árbol se puede colapsar a un botón.
- Hasta 1024 px, el árbol se apila arriba del editor y la rúbrica debajo; sin scroll horizontal.
- El árbol usa `role="tree"`, `aria-expanded` y navegación con flechas; Enter selecciona; el foco es visible.
- El editor es de solo lectura. Los comentarios se alcanzan con teclado.
- No se depende solo del color para indicar el archivo seleccionado.

### 4.4 Reglas visuales

- Solo tokens del sistema: `var(--tenant-*)`, `var(--color-*)`, `var(--border-color)`, `transparent`, `currentColor`.
- Sin `solv-` en selectores ni clases. Nuevos selectores: `review-algorithm-panel`, `review-project-panel`, `review-file-tree`.
- Fechas con `dateText` dentro de `<time>`; conteos con `[machineData]`; rutas y contenido técnico con `font-mono`.
- Sin fondos oscuros ni azules de énfasis fuera de los tokens.
- `npm run lint:styles` debe pasar antes de cerrar cada fase.

---

## 5. Contrato de datos

### 5.1 `GET /api/v1/teacher/submissions/{id}/review`

- **Auth:** docente o administrador. Estudiante: `403`.
- **Respuesta:** `{ "data": { ... }, "error": "", "message": "..." }`.

Campos nuevos:

| Campo | Tipo | Juez | IDE | Regla |
| --- | --- | --- | --- | --- |
| `environment_type` | string | `JUEZ_EFIMERO` | `IDE_PERSISTENTE` | Siempre presente. Sale de `exercises.environment_type`; si falta, `JUEZ_EFIMERO`. |
| `language` | string | Lenguaje del ejercicio | Lenguaje del ejercicio | Siempre presente; vacío si no está definido. |
| `boilerplate` | string | Código base del ejercicio | `""` | Fuente del diff en juez. En IDE va vacío; el diff por archivo queda fuera de alcance. |
| `files_available` | bool | `false` | `true` si el snapshot se pudo estructurar | La UI decide árbol o visor plano con este campo. |
| `metrics_available` | bool | `true` si la entrega tiene veredicto de ejecución | `false` | La UI muestra tiempo y memoria solo si es `true`. En juez sin medición, muestra `—`. |
| `files` | array | `[]` | Array de archivos | Solo IDE. Es un read model derivado de `code`; no se persiste. |
| `files[].path` | string | — | Ruta relativa | Sin rutas absolutas ni `..`. Solo se muestra, no se usa para acceder al disco. |
| `files[].language` | string | — | Lenguaje inferido | Por extensión; si no se reconoce, el lenguaje del ejercicio o `plaintext`. |
| `files[].content` | string | — | Contenido del archivo | Sin cabecera ni separador; saltos normalizados a `\n`. |
| `files[].start_line` | int | — | Línea 1-based de la primera línea de contenido en `code` | Permite traducir comentarios. |
| `files[].line_count` | int | — | Cantidad de líneas de contenido | Puede ser 0 para un archivo vacío. |

Reglas del contrato:

- `code` se mantiene en ambas modalidades. En IDE conserva el snapshot completo con cabeceras; `files` es una vista derivada. No hay migración en esta entrega.
- `test_cases` es `[]` en IDE. El backend no busca casos en `exercise_config` cuando la modalidad es IDE.
- `score` se omite cuando no hay nota, como hoy.
- `execution_time_ms` y `memory_used_mb` se mantienen; la UI de IDE los ignora. No se presenta `0` como medición.
- `metrics_available` es `true` cuando la modalidad es juez y `verdict` no es `PENDING`; es `false` en IDE o en una entrega que aún no se ejecutó. Es el campo que decide si se muestran tiempo y memoria.
- `comments` no cambia; `line_number` sigue siendo global sobre `code`.
- `boilerplate` reemplaza la heurística de regex del visor. En IDE se envía vacío y el diff por archivo no se muestra.
- `files_available = false` si el snapshot está vacío, no tiene cabeceras reconocibles, tiene rutas duplicadas o supera 500 archivos o 2 MiB. En ese caso la UI usa el visor plano y muestra un aviso.
- El contenido se escapa en Angular y en CodeMirror. No se renderiza HTML.

Ejemplo abreviado:

```json
{
  "data": {
    "id": "8f1c2a9e-1c3d-4a5b-9e7f-2b6c8d0a1e34",
    "exercise_id": "0d6f1a2b-3c4d-4e5f-8a9b-0c1d2e3f4a5b",
    "exercise_title": "Práctica 3: API Express con pruebas unitarias",
    "subject_id": "5a4b3c2d-1e0f-4a9b-8c7d-6e5f4a3b2c1d",
    "subject_name": "Desarrollo Web Fullstack",
    "student_id": "9e8d7c6b-5a4f-4e3d-2c1b-0a9f8e7d6c5b",
    "student_name": "Sofía Méndez Claros",
    "student_email": "sofia.mendez@uab.edu.bo",
    "environment_type": "IDE_PERSISTENTE",
    "language": "javascript",
    "boilerplate": "",
    "code": "# === FILE: app.js ===\nimport express from \"express\";\n\n# === FILE: package.json ===\n{\"name\":\"practica-3\"}\n",
    "files_available": true,
    "files": [
      {
        "path": "app.js",
        "language": "javascript",
        "content": "import express from \"express\";\n",
        "start_line": 2,
        "line_count": 1
      },
      {
        "path": "package.json",
        "language": "json",
        "content": "{\"name\":\"practica-3\"}\n",
        "start_line": 5,
        "line_count": 1
      }
    ],
    "verdict": "PENDING",
    "manual_override": false,
    "metrics_available": false,
    "execution_time_ms": 0,
    "memory_used_mb": 0,
    "test_cases": [],
    "comments": [
      {
        "id": "c1a2b3c4-d5e6-4f7a-8b9c-0d1e2f3a4b5c",
        "submission_id": "8f1c2a9e-1c3d-4a5b-9e7f-2b6c8d0a1e34",
        "author_id": "11111111-1111-4111-8111-111111111111",
        "author_name": "Docente",
        "line_number": 2,
        "comment": "Falta validar el body antes de responder.",
        "created_at": "2026-11-01T10:15:00Z"
      }
    ],
    "next_submission_id": null,
    "prev_submission_id": null,
    "submitted_at": "2026-10-26T19:47:00Z"
  },
  "error": "",
  "message": "Detalle de revisión SpeedGrader obtenido exitosamente"
}
```

### 5.2 Comentarios

`POST /api/v1/teacher/submissions/{id}/comments` no cambia de forma:

```json
{
  "line_number": 2,
  "comment": "Falta validar el body antes de responder."
}
```

- `line_number` es global, 1-based, sobre `code`, incluidas las cabeceras.
- Traducción en la vista: `global = start_line + local - 1`; `local = global - start_line + 1`.
- Al comentar en IDE, la UI envía la línea global del archivo seleccionado. Al mostrar, resta el mismo offset.
- El backend valida que la línea exista en `code`. Fuera de rango: `422 line_out_of_range`.
- Un comentario cuya línea cae en una cabecera, como el comentario general de la rúbrica en la línea 1, se muestra en la lista de comentarios generales y no se ancla a un archivo.
- Los comentarios existentes siguen funcionando; no hay migración.
- Queda fuera de esta entrega agregar `file_path` y guardar la línea local. Si se necesita, se hace con migración y backfill.

### 5.3 `POST /api/v1/teacher/submissions/{id}/run-ephemeral`

- Solo para `JUEZ_EFIMERO`.
- Request: `{ "code": "...", "language": "python" }`. `language` es obligatorio y debe coincidir con el lenguaje del ejercicio.
- IDE: `409 ephemeral_not_supported_for_ide`.
- La UI oculta la acción en IDE y deja de enviar `language: 'python'` fijo.

### 5.4 Rúbrica y estado de revisión

- `GET /api/v1/exercises/{id}/rubric` se mantiene. Si no hay rúbrica, responde `404 rubric_not_found` en lugar de un `500` genérico; la vista muestra el estado vacío. Hoy el handler devuelve el código `rubric not found`; se normaliza a `rubric_not_found`.
- `POST /api/v1/submissions/{id}/rubric-evaluation` se mantiene en forma.
- **Requisito de cierre:** al guardar la rúbrica, la entrega pasa a `review_status = graded`, se escriben `reviewed_at` y `reviewed_by`, y se actualiza `exercise_grades` en la misma operación. Hoy `DefaultRubricService.EvaluateSubmission` actualiza `score`, `verdict`, `manual_override` y `graded_by`, pero no el estado de revisión ni la libreta. Ese hueco se corrige como parte de esta especificación.
- La respuesta `200` solo se devuelve si la nota, el estado y la libreta quedaron escritos. Si falla la libreta, la UI no muestra el banner de éxito.
- `PATCH /api/v1/teacher/submissions/{id}/review` se mantiene para transiciones explícitas (`in_review`, `returned`, etc.).

### 5.5 Errores

| Código | Cuándo | Cuerpo |
| --- | --- | --- |
| `400 invalid_submission_id` | UUID inválido | `{ "error": "invalid_submission_id", "message": "..." }` |
| `403 Forbidden` | Rol estudiante | Mensaje de acceso denegado |
| `404 submission_not_found` | No existe o no pertenece al docente | Mensaje de no encontrado |
| `404 rubric_not_found` | La práctica IDE no tiene rúbrica | Mensaje con enlace a la práctica |
| `409 ephemeral_not_supported_for_ide` | Se intenta ejecutar el sandbox en IDE | Mensaje de acción no aplicable |
| `422 line_out_of_range` | Comentario fuera del código | Mensaje con el rango válido |

---

## 6. Parser del snapshot

### 6.1 Formato de entrada

El writer de `workspace_submission.go` genera:

```text
# === FILE: app.js ===
import express from "express";

# === FILE: package.json ===
{"name":"practica-3"}
```

- Cabecera: línea que empieza con `# === FILE: ` y termina con ` ===`.
- La ruta es el texto entre ambos marcadores, sin espacios al inicio ni al final.
- Entre bloques hay exactamente una línea en blanco, agregada por el writer.

### 6.2 Reglas del parser

- Una cabecera solo cuenta si está al inicio de una línea.
- El contenido de un archivo es todo lo que sigue a su cabecera hasta la siguiente cabecera o el final.
- Se descarta la línea en blanco separadora que precede a la siguiente cabecera. Las líneas en blanco internas se conservan.
- Se normalizan `\r\n` y `\r` a `\n`.
- No se recorta la indentación ni los espacios al inicio de las líneas.
- `start_line` es la línea 1-based de la primera línea de contenido en `code`; `line_count` cuenta solo líneas de contenido.
- Un archivo vacío es válido: `content = ""` y `line_count = 0`. Ruta duplicada, ruta absoluta, ruta con `..`, caracteres de control, cabecera sin ruta o formato no reconocido: `files_available = false` y se usa el visor plano.
- Límites: 500 archivos y 2 MiB de snapshot. Si se superan, `files_available = false`.
- La función es pura y vive en `internal/core/domain/project_snapshot.go`, con tests al lado.

### 6.3 Lenguajes por extensión

| Extensión | Lenguaje |
| --- | --- |
| `.py` | `python` |
| `.js`, `.mjs`, `.cjs`, `.jsx` | `javascript` |
| `.ts`, `.tsx` | `typescript` |
| `.java` | `java` |
| `.c`, `.h`, `.cpp`, `.hpp`, `.cc` | `cpp` |
| `.go` | `go` |
| `.sql` | `sql` |
| `.json` | `json` |
| `.html`, `.htm` | `html` |
| `.css` | `css` |
| `.scss` | `scss` |
| `.md` | `markdown` |
| Otra | Lenguaje del ejercicio o `plaintext` |

### 6.4 Tests del parser

- Un archivo.
- Dos o más archivos, con orden alfabético y offsets correctos.
- Un archivo vacío, con `line_count = 0`.
- Snapshot vacío.
- Sin cabeceras.
- Cabecera con ruta vacía.
- Ruta duplicada.
- Ruta absoluta y ruta con `..`.
- CRLF.
- Contenido que incluye una línea parecida a una cabecera pero no al inicio de línea.
- Archivo que termina con líneas en blanco.
- Límite de archivos y de tamaño.
- Propiedad: al reconstruir `code` desde `files` con las cabeceras, el resultado coincide salvo la normalización documentada del separador.

---

## 7. Criterios de aceptación

- [ ] **AC-01:** `GET /api/v1/teacher/submissions/{id}/review` incluye `environment_type` y `language` en ambas modalidades. Los valores coinciden con el ejercicio. Un tipo faltante o desconocido cae en `JUEZ_EFIMERO` sin `500`.
- [ ] **AC-02:** Una entrega IDE con dos o más archivos devuelve `files_available = true` y `files` con las rutas en orden alfabético. `start_line` y `line_count` ubican cada archivo dentro de `code`. La reconstrucción desde `files` coincide con `code` salvo el separador normalizado.
- [ ] **AC-03:** El panel IDE no renderiza veredicto, tiempo, memoria, casos, complejidad ni resultado efímero. Renderiza el árbol y la rúbrica. El panel juez renderiza casos, veredicto y métricas solo cuando `metrics_available` es `true`; si no, muestra `—`. No renderiza la rúbrica.
- [ ] **AC-04:** El header IDE no muestra "Re-ejecutar". El header juez sí lo muestra y envía el lenguaje del ejercicio. `run-ephemeral` responde `409` en IDE.
- [ ] **AC-05:** Seleccionar un archivo muestra su contenido y su lenguaje. El árbol funciona con teclado y a 1024 px sin scroll horizontal. No hay selectores `solv-` ni colores hex; `npm run lint:styles` pasa.
- [ ] **AC-06:** Un comentario en la línea local 3 de `src/app.js` se guarda con la línea global correcta, reaparece en la misma línea al recargar y se lista en los comentarios. Un comentario de juez sigue funcionando.
- [ ] **AC-07:** Guardar la rúbrica devuelve `200` solo después de escribir `score`, `review_status = graded`, `reviewed_at`, `reviewed_by` y la celda de `exercise_grades`. La entrega sale de la cola de pendientes y el banner de éxito aparece solo en ese caso.
- [ ] **AC-08:** Si la práctica IDE no tiene rúbrica, el panel muestra un estado vacío explícito y deshabilita el guardado. No queda un panel en blanco ni un `500` sin tratar.
- [ ] **AC-09:** `currentLiveSession` no fabrica un workspace. "Consola del Estudiante" se oculta en IDE salvo que exista un workspace real.
- [ ] **AC-10:** El flujo e2e J08 sigue pasando y suma aserciones de revisión IDE: sin bloques de juez, árbol visible, cambio de archivo y guardado de rúbrica. J07 cubre la regresión de juez.
- [ ] **AC-11:** No hay migración de base de datos en esta entrega. `code`, `line_number` y los comentarios existentes siguen siendo compatibles.
- [ ] **AC-12:** Pasan los tests de Go de los paquetes tocados, los tests de Angular de los componentes modificados y los dos flujos e2e. No se agregan imágenes `:latest` ni dominios hardcodeados.
- [ ] **AC-13:** El diff de juez usa `boilerplate` del payload. El componente no adivina el código base con regex. En IDE no se muestra diff por archivo.

---

## 8. Plan de implementación

| Fase | Alcance | Archivos principales | Terminado cuando |
| --- | --- | --- | --- |
| 0 | Contrato, parser y cierre de revisión | `backend/internal/core/domain/teacher_review.go`, `backend/internal/core/domain/project_snapshot.go`, `backend/internal/core/services/rubric_service.go`, `backend/internal/infrastructure/storage/postgres/teacher_grading_repository.go`, `frontend/src/app/features/teacher/models/teacher.models.ts` | AC-01, AC-02, AC-07 |
| 1 | Corte por modalidad | `frontend/src/app/features/teacher/grading/speed-grader/speed-grader.component.*`, `components/speed-grader-header/*`, `components/speed-grader-eval-panel/*`, `backend/internal/core/services/teacher_ephemeral.go` | AC-03, AC-04, AC-08, AC-09 |
| 2 | Árbol de archivos y comentarios | `frontend/src/app/features/teacher/grading/speed-grader/components/review-file-tree/`, `components/speed-grader-code-viewer/*`, `speed-grader.component.html` | AC-05, AC-06, AC-10, AC-11 |
| 3 | Entorno de prueba del alumno (diferido) | Job nuevo sobre el patrón de `practice_env_test`, ADR propia | Criterios de la ADR de la fase 3 |

Notas de implementación:

- En la fase 1, limpiar los inputs duplicados de `<speed-grader-code-viewer>` en `speed-grader.component.html`.
- El parser se integra en el servicio o repositorio de revisión, no en el handler HTTP. El handler solo transporta el DTO.
- La fase 2 no cambia el esquema de comentarios. La traducción de líneas vive en la vista y tiene tests unitarios.
- La fase 3 no se mezcla con las anteriores. Su contrato se define en una ADR aparte.

---

## 9. Documentación y gobernanza

- **Esta especificación:** `docs/UI/DOCENTE/REVISION_POR_MODALIDAD.md`.
- **ADR:** nueva ADR de revisión por modalidad o actualización de [ADR-053](../../ARQUITECTURA/ADR/ADR-053-revision-docente-y-libreta-de-notas.md) antes de implementar la fase 2.
- **Contrato de API:** actualizar `docs/API_CONTRACTS.md` con los campos nuevos al cerrar la fase 0.
- **IDE persistente:** actualizar `docs/ARQUITECTURA/IDE_PERSISTENTE.md` con el read model `files[]` y la traducción de líneas al cerrar la fase 2.
- **Mapa de vistas:** no cambia la ruta; solo se ajusta la descripción si el alcance de la vista cambia.
- **E2E:** ampliar `frontend/e2e/flows/02-medium/J08-persistent-ide-lifecycle.spec.mjs` y mantener J07 como regresión de juez.

---

## 10. Fuera de alcance y decisiones abiertas

- **Entorno de prueba del alumno:** queda para la fase 3. Hay que elegir entre conectarse al workspace vivo en modo sombra, crear un contenedor efímero desde el snapshot o superponer el snapshot sobre la plantilla. También queda abierto si la app se muestra en iframe o en pestaña; muchas aplicaciones bloquean iframe por CSP o `X-Frame-Options`.
- **Diff por archivo en IDE:** necesita los archivos de la plantilla base. En juez, el diff debe usar `exercise.boilerplate`, no una heurística.
- **Anclaje por archivo de comentarios:** se mantiene la línea global en esta entrega. Agregar `file_path` y línea local requiere migración y backfill.
- **Tercera modalidad:** si aparece, introducir un `review_mode` explícito en lugar de repartir otro booleano por el componente.
- **Duplicación de `code` y `files`:** es aceptable por ahora. Si el payload crece, se evalúa omitir `code` en IDE en una versión posterior.

---

## 11. Riesgos

- **Tamaño del payload:** `code` y `files` duplican el snapshot. Se acota a 2 MiB y 500 archivos; si crece, se mide antes de cambiar el contrato.
- **Ambigüedad del parser:** el separador en blanco entre archivos es una convención del writer. Se documenta y se prueba con archivos que terminan en blanco. `code` sigue siendo la fuente canónica.
- **Estado de revisión:** conectar la rúbrica con la libreta puede introducir dependencias circulares. Si aparece, se resuelve en un servicio de aplicación, no en el handler.
- **Densidad de la vista:** tres columnas a 1366 px dejan poco ancho para el código. Se prueba a 1024, 1280 y 1440 px, y el árbol se colapsa.
- **Accesibilidad:** el árbol es el componente nuevo con más riesgo. Se prueba con teclado y con foco visible, no solo con mouse.
- **E2E con CodeMirror:** usar selectores estables y esperar por texto del archivo, no por clases internas del editor.
