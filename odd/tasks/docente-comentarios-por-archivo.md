# Feature: Comentarios de Revisión Anclados al Archivo

Objetivo: que un comentario del SpeedGrader identifique **archivo y línea local** además de la línea global del snapshot, para que el visor web y la futura extensión SOLV Companion lo coloquen sin recalcular el snapshot.

Branch: `feat/review-comments-by-file` (desde `main`) | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm run test:ci`

Estado: **implementada (2026-11-02)**. Migración `00045` aplicada; backend y frontend con tests en verde. Falta commitear y, cuando exista el paquete, el consumo desde SOLV Companion.

Fuente: `proyecto-grado/Observaciones/Flujo de Laboratorios del Estudiante y Propuesta Extension VSCode.md` y `proyecto-grado/Observaciones/Propuesta Tecnica Extension VSCode SOLV Companion.md` §3.B. Decisión del usuario: «file_path y local_line_number en la BD, omitiendo code en proyectos IDE».

## Decisiones cerradas

- Se persisten `file_path` y `local_line_number` en `submission_comments`. La línea global (`line_number`) sigue existiendo y sigue siendo la que valida el backend contra `code`.
- En modalidad IDE el payload de revisión deja de duplicar `code` y `boilerplate`: `files` y `base_files` son la fuente de contenido. En juez se mantiene `code` + `boilerplate`.
- La extensión Companion no se implementa en esta ficha. Solo se deja el contrato de lectura listo para cuando exista (`tools/solv-companion/` no está en el repo).

## Qué existe hoy (verificado en el código)

- Tabla `submission_comments` en `backend/migrations/00001_baseline.sql:395`: `line_number INT NOT NULL` y `comment TEXT NOT NULL`; la fase 3 agregó `end_line_number` en `backend/migrations/00043_submission_comment_ranges.sql`. No hay `file_path` ni línea local.
- `domain.SubmissionComment` (`backend/internal/core/domain/teacher_review.go:9`) expone `LineNumber` y `EndLineNumber`. `AddCommentRequestDTO` solo acepta `line_number`, `end_line_number` y `comment`.
- `TeacherService.AddComment` (`backend/internal/core/services/teacher_submissions.go:43`) valida el rango contra `review.Code` y rechaza bloques que cruzan archivos con `ErrLineRangeSpansFiles`. Ese comportamiento se conserva.
- `PostgresTeacherRepository.AddComment` (`backend/internal/infrastructure/storage/postgres/teacher_grading_repository.go:355`) inserta las columnas actuales. `GetCommentsBySubmission` (mismo archivo, línea 376) las lee.
- El repositorio arma `Files` desde `row.Code` y `BaseFiles` desde `row.Boilerplate` con `domain.ParseProjectSnapshot` (`teacher_grading_repository.go:224-231`). El DTO devuelve `code`, `files` y `base_files` a la vez.
- La vista web traduce local↔global con `start_line` en `frontend/src/app/features/teacher/grading/speed-grader/review-line-map.ts`. Funciona, pero depende de que el snapshot llegue completo en `files`.
- La rúbrica inserta un comentario general con `line_number = 1` y sin archivo (`backend/internal/core/services/rubric_service.go:189`). Es un caso válido de comentario sin `file_path`.

## Specs

- **S1 — Migración `00045_submission_comment_anchors.sql`** (el número puede correr: verificar `ls backend/migrations` antes).
  - `ALTER TABLE submission_comments ADD COLUMN IF NOT EXISTS file_path TEXT;`
  - `ALTER TABLE submission_comments ADD COLUMN IF NOT EXISTS local_line_number INT;`
  - `CHECK`: ambos nulos o ambos con valor (`file_path <> '' AND local_line_number > 0`). Nada de archivo sin línea ni línea local sin archivo.
  - Índice parcial `(submission_id, file_path)` donde `file_path IS NOT NULL`.
  - `Down` elimina el índice, el constraint y las dos columnas. Sin backfill en SQL.
  - Criterio: la migración aplica y revierte sobre `solv_test`; una fila con `file_path` y sin `local_line_number` es rechazada.

- **S2 — Dominio y validación.**
  - `SubmissionComment` agrega `FilePath *string` y `LocalLineNumber *int` (punteros: null es válido en juez y en comentarios generales).
  - `AddCommentRequestDTO` agrega `file_path` y `local_line_number`; acepta también `local_end_line_number` opcional o deriva el local final de `end_line_number` al leer.
  - Nuevo error `ErrCommentFileUnknown` (`422 comment_file_unknown`) cuando `file_path` no existe en `files`.
  - Criterio: un comentario IDE con archivo inexistente devuelve 422; uno con archivo válido y línea local fuera de `line_count` devuelve 422.

- **S3 — Servicio: una sola fuente de verdad.**
  - Si llegan `file_path` + `local_line_number`, el servicio busca el archivo en `review.Files` y calcula `LineNumber = file.StartLine + local - 1`. Si además llega `line_number`, debe coincidir; si no coincide, 422.
  - Si llegan solo `line_number` y `end_line_number` (clientes actuales), el servicio deriva `file_path` y `local_line_number` cuando la línea cae dentro de un archivo. Si cae en una cabecera del snapshot, ambos quedan nulos.
  - Los datos viejos se resuelven al leer: si `file_path` es nulo y la línea cae dentro de un archivo, el DTO expone el archivo y la línea local derivados. No se hace backfill.
  - Criterio: un comentario creado con el cliente viejo, en una línea de `src/app.ts`, se lee con `file_path = "src/app.ts"` y la local correcta.

- **S4 — API.**
  - `POST /api/v1/teacher/submissions/{id}/comments` acepta las dos formas (archivo+local o global).
  - `GET /api/v1/teacher/submissions/{id}` devuelve en cada comentario `file_path`, `local_line_number` y `local_end_line_number` (este último calculado).
  - Documentar la compatibilidad en `docs/API_CONTRACTS.md`.
  - Criterio: repetir el `POST` con el mismo cuerpo no cambia el ancla; el `GET` devuelve los tres campos.

- **S5 — Payload por modalidad.**
  - En `TeacherSubmissionReviewDTO`, `code` y `boilerplate` llevan `omitempty`; en IDE se vacían **en la capa HTTP**, después de que el repositorio haya armado `files` y `base_files`.
  - No vaciar `code` en el repositorio: `AddComment` valida el rango sobre `review.Code`.
  - En juez no cambia nada: `code` + `boilerplate` + `test_cases`.
  - Criterio: `GET` de una entrega IDE no contiene `code` ni `boilerplate`; `files` y `base_files` conservan contenido y `start_line`.

- **S6 — Frontend.**
  - `speed-grader.component.ts` deja de derivar `codeLines` de `review().code` en IDE; usa `activeFile().line_count` y el contenido del archivo activo.
  - El alta de comentario envía `file_path` + `local_line_number` + `local_end_line_number` en IDE, y `line_number` + `end_line_number` en juez.
  - `review-line-map.ts` usa primero los campos locales del comentario y cae a la traducción por `start_line` cuando llegan nulos.
  - Criterio: un comentario de bloque L2–L5 en `src/app.ts` se guarda y se vuelve a leer igual después de recargar.

- **S7 — Tests y documentación indivisible.**
  - Unitarios: `teacher_submissions_test.go` (derivación, desacuerdo global/local, archivo desconocido), `teacher_grading_repository_test.go` si existe, `teacher_submission_read_errors_test.go` (payload IDE sin `code`).
  - Integración: extender `backend/tests/integration/slice13_teacher_review_modality_test.go` con el ancla por archivo.
  - Frontend: `review-line-map.spec.ts` y `speed-grader.component.spec.ts` con comentarios viejos y nuevos.
  - Docs: `docs/API_CONTRACTS.md`, `docs/UI/DOCENTE/REVISION_POR_MODALIDAD.md` y una sección nueva en `docs/ARQUITECTURA/IDE_PERSISTENTE.md` que explique el contrato para Companion. Actualizar ADR-060 con una nota de contrato (no hace falta ADR nuevo).

## Tasks

- [x] **T1 — S1**: migración `00045_submission_comment_anchors.sql` aplicada en la base local (versión 45). Columnas, check e índice verificados.
- [x] **T2 — S2**: dominio + DTO + `ErrCommentFileUnknown`.
- [x] **T3 — S3**: `TeacherService.AddComment` con derivación bidireccional y validación; 5 tests nuevos.
- [x] **T4 — S4**: handler con ancla, error `422 comment_file_unknown` y lectura de campos.
- [x] **T5 — S5**: `code`/`boilerplate` con `omitempty`, vaciados en la capa HTTP; `files`/`base_files` nulos en juez.
- [x] **T6 — S6**: alta de comentario con archivo y línea local; `review-line-map` prefiere el ancla persistida.
- [x] **T7 — S7**: tests de servicio, handler y frontend; contratos, ADR-060 enmendado y guía de revisión.

## Decisiones resueltas (`Decisiones para el Agente.md`)

- **D1 — Sin backfill.** El ancla se deriva al leer mientras el snapshot exista; los comentarios viejos conservan la línea global.
- **D2 — Se aceptan las dos formas.** El cliente web manda global + archivo/local; el servidor calcula la global y exige coincidencia.
- **D3 — Companion solo consume el contrato.** La extensión no está en el repo y su construcción es una ficha aparte.

## Verificación

```bash
cd backend && gofmt -l internal/core/domain internal/core/services internal/delivery/http internal/infrastructure/storage/postgres
cd backend && go build ./... && go vet ./...
cd backend && go test ./internal/core/domain ./internal/core/services ./internal/delivery/http ./internal/infrastructure/storage/postgres
cd frontend && npm run lint:styles && npm run lint:ts
cd frontend && npm run test:ci -- --include='**/speed-grader/**' --include='**/review-line-map.spec.ts'
cd frontend && CI=true npm run build
```

## Log

- **L1 (verbatim):** «Anclaje de comentarios: file_path y local_line_number en la BD, omitiendo code en proyectos IDE.»
- **L2 (verbatim):** «Si se agrega file_path y línea local a los comentarios.» — decisión abierta que esta ficha cierra.
- **Hallazgo verificado:** `AddComment` ya distingue archivos con `ProjectFileForGlobalLine`, pero no guarda esa decisión; si el snapshot cambia de formato, el comentario se re-deriva. Persistir el ancla evita depender del parser para siempre.
- **Hallazgo verificado:** el DTO devuelve `code` + `files` + `base_files` en IDE; `code` es el snapshot concatenado con cabeceras `# === FILE: ... ===`. Es el doble más caro del payload.
- **Cierre (2026-11-02):** migración `00045` aplicada (versión 45). Pasan `go build`, `go vet` y los paquetes de dominio, servicios, http y postgres; 30 tests del SpeedGrader y del mapa de líneas en el frontend. Backend reconstruido y reiniciado sobre el contrato nuevo.
