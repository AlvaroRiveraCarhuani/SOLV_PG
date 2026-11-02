# Feature: Auditoría de Similitud dentro de la Práctica

Objetivo: que el control de plagio se abra desde una práctica concreta, con su código base ya restado, rangos de línea reales y un visor que enfrente las dos entregas sin exponer tokens del compilador.

Branch: `feat/plagiarism-practice-scope` (desde `main`) | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm run test:ci`

Estado: **Entrega 1 implementada (2026-11-02)**. Motor con sustracción de base y rangos, numeración estable y navegación desde la práctica. La Entrega 2 (endpoint del par y visor) sigue pendiente. `go build`, `go vet`, tests de backend y frontend, `lint:styles` y `CI=true npm run build` en verde; los e2e no se ejecutaron en esta sesión.

Fuente: `proyecto-grado/Observaciones/Control de Plagio y Deteccion de Similitud.md`. Decisión del usuario: «Sacarlo del menú lateral, moverlo a la práctica y sustraer el código base del cálculo».

## Decisiones cerradas

- La entrada `SUPERVISIÓN → Control de Plagio` sale del sidebar docente.
- El acceso principal pasa a la ficha de la práctica, con la pantalla filtrada a esa práctica.
- El motor resta el código base (`exercises.boilerplate`) antes de calcular Jaccard/Containment.
- La interfaz deja de mostrar `common_structures` con `$LIT`/`$ID`; muestra rangos de línea y código fuente.
- Los datos forenses de tipeo (`submission_keystroke_events`) se muestran junto al par, sin botones de trámite ni decisión automática.

## Qué existe hoy (verificado en el código)

- Sidebar: `frontend/src/app/features/teacher/layout/teacher-layout.component.ts:30-36` tiene la sección `SUPERVISIÓN` con la única entrada `Control de Plagio` (`/teacher/anti-plagio`). Al quitarla, la sección queda vacía y debe desaparecer.
- Ruta: `frontend/src/app/features/teacher/teacher.routes.ts:100` carga `TeacherPlagiarismComponent`. E2E que la visitan: `frontend/e2e/flows/05-badges-and-grading/J07-speedgrader-ephemeral-and-plagiarism.spec.mjs:320` y `frontend/e2e/flows/04-edge-cases/12-anti-plagiarism-supervision.spec.mjs`.
- Página: `frontend/src/app/features/teacher/supervision/anti-plagio/teacher-plagiarism.component.ts` pide materia y práctica con dos combobox; renderiza chips de `common_structures` en `.html:191-200`.
- API: `GET /api/v1/teacher/plagiarism` y `GET /api/v1/teacher/courses/{id}/plagiarism` (`backend/internal/delivery/http/router.go:540-541`); el handler acepta `subject_id` y `exercise_id` (`teacher_plagiarism_timeline.go:10`).
- Repositorio: `PostgresTeacherRepository.GetExerciseSubmissionsForPlagiarism` (`backend/internal/infrastructure/storage/postgres/teacher_analytics_repository.go:139`) **no** selecciona `ex.boilerplate`; solo `sub.code`, `ex.language`.
- Motor: `ASTPlagiarismEngine.Tokenize` guarda `Token.Line` (`plagiarism_engine.go:22`), pero `removeComments` descarta las líneas de bloque comentadas (`plagiarism_engine.go:312-320`), así que la numeración se desplaza si hay `/* ... */` multilínea.
- `ComputeSimilarity` (`plagiarism_engine.go:157`) genera k-gramas de ambos lados sin restar base y devuelve `commonPatterns` con `simplifyGram` (tokens normalizados).
- `AnalyzeSubmissions` (`plagiarism_engine.go:225`) agrupa por `ExerciseID`, salta el mismo alumno y ordena por similitud. `PlagiarismMatch` (`domain/teacher_review.go:190`) no tiene rangos.
- Telemetría: `TeacherService.GetKeystrokeEvents` y `GetSubmissionTimeline` ya existen (`backend/internal/core/services/teacher_plagiarism_timeline.go:45,103`); `SubmissionKeystrokeReport` trae tiempo total, pegado y porcentaje (`domain/keystroke.go:30`).

## Specs

### Entrega 1 — Motor y reubicación

- **S1 — Navegación contextual.**
  - Quitar la sección `SUPERVISIÓN` de `TEACHER_NAV_SECTIONS`.
  - Agregar `Auditoría de Similitud` en la cabecera de la ficha de práctica, junto a `Abrir IDE` / `Probar Solución`. Visible para juez y para IDE.
  - Ruta nueva: `/teacher/cursos/:courseId/practicas/:exerciseId/similitud`, que reutiliza `TeacherPlagiarismComponent` en modo bloqueado (sin combobox de materia; la práctica fija).
  - `/teacher/anti-plagio` queda como redirección a `/teacher/por-calificar` o como selector de práctica; **no** conserva el listado global. Actualizar J07 y el flujo 12.
  - Criterio: desde una práctica se abre su auditoría sin elegir materia; el sidebar no muestra `Control de Plagio`.

- **S2 — Sustracción del código base.**
  - `SubmissionForPlagiarism` agrega `Boilerplate string` y `EnvironmentType string`; el repositorio selecciona `ex.boilerplate` y `ex.environment_type`.
  - Para juez: k-gramas base = `GenerateKGrams(Tokenize(ex.boilerplate))`; se descartan de la entrega antes de Jaccard/Containment.
  - Para IDE: parsear `sub.code` y `ex.boilerplate` con `domain.ParseProjectSnapshot`; restar por archivo de la misma ruta. Si el base no tiene cabeceras, restar global.
  - Si tras restar quedan menos de `KGramSize` tokens, la similitud es 0 y el par no se reporta.
  - Criterio: dos entregas que solo comparten el código base dan 0% y no aparecen; una copia real sigue superando el umbral.

- **S3 — Rangos de línea.**
  - Conservar la numeración al quitar comentarios: `removeComments` escribe una línea vacía en vez de `continue` en los bloques.
  - Reemplazar `CommonStructures []string` por `Ranges []PlagiarismRange` (máximo 10 por par) con `source_a`/`source_b` = `{file_path?, start_line, end_line}`, `token_count` y `exact_match`.
  - Calcular los tramos con una pasada voraz sobre k-gramas coincidentes: extender hacia adelante y atrás mientras los tokens sean iguales, quedarse con los tramos maximales y descartar solapados.
  - En IDE, `file_path` sale de `ProjectFileForGlobalLine`; en juez queda vacío.
  - Criterio: el par de prueba devuelve un tramo con líneas reales, sin `$LIT` ni `$ID`, y el código de las líneas citadas coincide con el fuente.

- **S4 — Verificación y documentación indivisible.**
  - Tests de motor: base 100% igual → 0%; copia parcial → rango esperado; comentarios multilínea no desplazan líneas; IDE de dos archivos → rangos por archivo.
  - Actualizar `teacher-plagiarism.component.spec.ts` y `teacher_plagiarism_engine` si existe.
  - ADR nuevo en `docs/ARQUITECTURA/ADR/` (sustracción de base y rangos) con entrada en `INDEX.md`.
  - `docs/API_CONTRACTS.md` con el contrato de `PlagiarismMatch` y `docs/UI/` con la nueva ubicación.

### Entrega 2 — Visor y contexto forense

- **S5 — Endpoint de par.**
  - `GET /api/v1/teacher/plagiarism/pairs/{submission_a}/{submission_b}`: valida mismo ejercicio, docente dueño de ambas y devuelve `{exercise, a: {submission_id, student, code|files, metrics}, b: {...}, ranges}`.
  - No devuelve `test_cases`, rúbrica ni comentarios. `metrics` = `SubmissionKeystrokeReport` resumido.
  - Criterio: un docente ajeno recibe 403; dos entregas de prácticas distintas reciben 422.

- **S6 — Vista de tres columnas.**
  - Columna 1: pares ordenados por riesgo, con badge de porcentaje y alerta forense.
  - Columna 2: dos visores `code-editor` en solo lectura, con resaltado por rango y scroll sincronizado; selector de archivo en IDE.
  - Columna 3: tiempo activo, pulsaciones, porcentaje de pegado y enlaces a SpeedGrader de cada alumno.
  - Sin botones de «confirmar plagio» ni correos: la decisión es la nota en SpeedGrader.
  - Criterio: el docente ve las dos entregas y el tramo resaltado sin cambiar de pestaña.

- **S7 — Clústeres (opcional, si el tiempo alcanza).**
  - Agrupar coincidencias transitivas en un cluster con autor sugerido y receptores; el visor mantiene dos columnas y un selector de contraparte.
  - Criterio: tres alumnos con el mismo código producen un cluster, no tres pares sueltos.

## Tasks

- [x] **T1 — S1**: se retiró la sección `SUPERVISIÓN`, se agregó `[Auditoría de Similitud]` en la ficha, la ruta contextual reemplaza el listado y J07 navega a ella. El flujo 12 solo consulta la API: no necesitaba cambios.
- [x] **T2 — S2**: `SubmissionForPlagiarism` con `Boilerplate` y `EnvironmentType`; el repositorio los selecciona; el motor resta por archivo en IDE y global en juez.
- [x] **T3 — S3**: `removeComments` conserva una línea por línea; `PlagiarismRange` reemplaza `common_structures`; tramos maximales sin solape, máximo 10.
- [x] **T4 — S4**: 4 tests nuevos de motor, specs del frontend, ADR-062 con entrada en `INDEX.md`, `docs/API_CONTRACTS.md` y `docs/UI/MAPA_DE_VISTAS.md`.
- [ ] **T5 — S5**: handler + servicio + ruta del par, con permisos y tests.
- [ ] **T6 — S6**: visor de tres columnas y spec.
- [ ] **T7 — S7**: clústeres, si se confirma el alcance.

## Referencias obligatorias para la Entrega 2

El visor se divide en dos pantallas según modalidad y debe seguir estos diseños ya aprobados:

- `proyecto-grado/Observaciones/propuesta-plagio-evaluador.html` — monoarchivo/algorítmico: sin árbol de archivos, veredicto de tests y complejidad.
- `proyecto-grado/Observaciones/propuesta-plagio-desarrollo.html` — multiarchivo IDE: árbol de archivos afectados frente al código base descartado, alternador de contrapartes en clústeres de 3+ y botones directos de calificar sin jerga.

## Dudas abiertas

- **D1 — Ruta.** Recomendación: ruta contextual nueva y retirar el listado global; actualizar los dos e2e. Alternativa conservadora: mantener `/teacher/anti-plagio` solo por URL y agregar la ruta contextual; menos trabajo, pero deja dos modelos.
- **D2 — Alcance por entregas.** Recomendación: dos entregas. La primera cambia resultados y navegación; la segunda es UI nueva. Todo junto supera las 1000 líneas.
- **D3 — Modalidades.** Recomendación: juez e IDE. En IDE la sustracción es por archivo; en juez es global. Si se quiere solo IDE, la ficha se reduce, pero el falso positivo del 100% también existe en juez.
- **D4 — Sustracción por archivo o global en IDE.** Recomendación: por archivo con la misma ruta; si el base no tiene cabeceras, global. Restar global borra coincidencias legítimas entre archivos distintos.

## Verificación

```bash
cd backend && gofmt -l internal/core/services internal/core/domain internal/infrastructure/storage/postgres
cd backend && go build ./... && go vet ./...
cd backend && go test ./internal/core/services -run 'Plagiarism|Similarity|KGram'
cd backend && go test ./internal/delivery/http ./internal/infrastructure/storage/postgres
cd frontend && npm run lint:styles && npm run lint:ts
cd frontend && npm run test:ci -- --include='**/anti-plagio/**'
cd frontend && CI=true npm run build
cd frontend && npx playwright test e2e/flows/05-badges-and-grading/J07-speedgrader-ephemeral-and-plagiarism.spec.mjs --config=playwright.e2e.config.mjs
```

## Log

- **L1 (verbatim):** «Reubicación de Anti-Plagio: Sacarlo del menú lateral, moverlo a la práctica y sustraer el código base del cálculo.»
- **L2 (verbatim):** «devolver rangos de líneas en vez de filtrar tokens raros del compilador ($LIT $ID), y mover la auditoría adentro de la práctica.»
- **Hallazgo verificado:** el repositorio no selecciona `ex.boilerplate`, así que el motor no puede restar la base aunque se lo proponga; es el primer cambio de la cadena.
- **Hallazgo verificado:** `removeComments` descarta líneas de bloque comentadas; sin corregirlo, cualquier rango posterior a un `/* ... */` apunta a la línea equivocada.
- **Hallazgo verificado:** J07 y el flujo 12 navegan a `/teacher/anti-plagio`; retirar la ruta sin actualizarlos rompe dos e2e existentes.
- **Corrección (2026-11-02):** el flujo 12 no visita la ruta, solo consulta la API; solo J07 necesitó el cambio de URL y de título.
- **Cierre de la Entrega 1:** D1 se resolvió con la ruta contextual y la redirección de `/teacher/anti-plagio`; D2 mantiene dos entregas; D3 cubre juez e IDE; D4 resta por archivo con la misma ruta y cae a global cuando falta la cabecera.
