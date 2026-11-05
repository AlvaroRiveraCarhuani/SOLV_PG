# ADR 066: Veredicto del Juez y Autoría Docente de la Nota

> **Estado:** Implementado
> **Fecha:** 2026-11-05
> **Ámbito:** Core, evaluación, libreta de notas, revisión docente
> **Relacionado:** ADR-052 (discriminador de modalidad por entorno), ADR-053 (estado de revisión y celda de nota), ADR-061 (preview de entrega desde SpeedGrader), ADR-062 (auditoría de similitud)

---

## Contexto

`JUEZ_EFIMERO` y `IDE_PERSISTENTE` convivían con un contrato de entrega único: `submissions.verdict`
era `NOT NULL`. Una entrega de IDE, que no pasa por el juez, se guardaba con el centinela `PENDING`
para satisfacer la columna. La rúbrica, al calificar, escribía `verdict = 'AC'` y `manual_override = TRUE`
para que la nota se leyera. El resultado era un veredicto que ningún juez emitió y una libreta que
contaba como «calificada automáticamente» una decisión humana.

La consecuencia no era cosmética: el banco de ejercicios contaba pendientes por veredicto mientras las
estadísticas por laboratorio ya usaban `review_status`, y el detalle de revisión escaneaba una columna
`NOT NULL` en un `string`, de modo que cualquier ausencia habría roto la consulta.

## Decisión

### 1. El veredicto es una salida del juez

`submissions.verdict` deja de ser obligatorio. Una entrega de `JUEZ_EFIMERO` lleva el veredicto que
emitió el juez; una entrega de `IDE_PERSISTENTE` no lleva veredicto, y su ausencia se representa como
ausencia, no como un código.

- La entrega de IDE se crea con `verdict = NULL` y `review_status = 'pending'`.
- La inserción usa `NULLIF(:verdict, '')`, así que el dominio sigue trabajando con cadena vacía y la
  base guarda ausencia.
- Toda lectura usa `COALESCE(verdict, '')` para no romper el escaneo en `string`.

### 2. La nota de un proyecto la escribe el docente

La rúbrica escribe `score`, `review_status = 'graded'`, `graded_by`, `reviewed_by` y `feedback_summary`.
No toca el veredicto ni `manual_override`.

La celda de libreta pasa a `source = 'manual'`, con `overridden_by` y `reason = 'Evaluado por rúbrica'`,
porque la nota la puso una persona con criterios.

### 3. El ajuste manual de un proyecto no pide veredicto

El override de veredictos queda reservado al juez: el endpoint responde 422 para entregas de IDE
(`ErrOverrideRequiresJudge`). El SpeedGrader mantiene el botón de ajuste manual, pero en modo proyecto
envía la nota por `PATCH /teacher/submissions/{id}/review` con `status = 'graded'`, que escribe la nota
de la entrega y la celda de la libreta sin inventar ningún resultado técnico.

### 4. Lo pendiente se declara con el estado de revisión

`review_pending` del banco de ejercicios deja de contar `verdict = 'PENDING'` y usa
`review_status IN ('pending', 'in_review')`. `auto_graded` cuenta solo entregas de juez, porque en IDE
no existe calificación automática.

### 5. El antiplagio y el Semgrep del entorno no son evaluación

La auditoría de similitud no lee el veredicto y ya trataba el snapshot de IDE por archivo; no cambia.
La auditoría Semgrep del entorno de práctica sigue siendo una ayuda de desarrollo: no produce veredicto
ni influye en ninguna nota. Ninguno de los dos se modifica por esta decisión.

### 6. La procedencia del veredicto viaja con el veredicto

Un veredicto guardado no dice por sí mismo quién lo escribió. `submissions.verdict_origin` lo declara
con cuatro valores: `judge` lo escribió el camino de entrega del juez, `rubric` lo escribió la rúbrica
antes de esta separación, `override` lo escribió un ajuste manual docente y `legacy` es la fila
histórica que no se puede atribuir a un camino concreto.

La columna es informativa: ninguna consulta de nota, entrega o revisión la usa para decidir. Su
propósito es que la interfaz no presente un resultado heredado como si fuera una evaluación del juez.

### 7. El veredicto de una entrega lo respalda el servidor

El juez corre en el servidor, pero la entrega llega por otro request y el navegador elige qué veredicto
envía. `POST /api/v1/submissions` ya no confía en ese valor: la respuesta de `POST /api/v1/evaluations`
viaja con `evaluation_receipt`, un token firmado con HMAC que ata institución, ejercicio, alumno,
veredicto, métricas, huella del código evaluado, instante de emisión y un identificador propio (`jti`). El
`jti` es necesario porque el instante tiene precisión de segundos: dos evaluaciones idénticas en el mismo
segundo deben tener identidad distinta. La entrega verifica la firma, la vigencia (15 minutos) y que el
código enviado sea el evaluado, y escribe el veredicto y las métricas del recibo; el `verdict` del payload
se ignora.

- Un ejercicio de `IDE_PERSISTENTE` se rechaza en ese endpoint con 422: su corrección es docente.
- Un recibo vencido, manipulado o de otro ejercicio, alumno o institución se rechaza con 422 y no se
  persiste ninguna entrega.
- `judge` solo se escribe cuando el veredicto salió de una evaluación del servidor.
- El recibo no se guarda: es una prueba efímera del resultado que el alumno ya recibió.

El veredicto no viaja en la petición de evaluación. `POST /api/v1/evaluations` recibe ejercicio, lenguaje
y código; el juez ejecuta los casos y el recibo firma el resultado que calculó. Un campo `verdict` en el
cuerpo no se lee: no forma parte del DTO.

La entrega se identifica por el recibo: su identificador deriva del token firmado. Reenviar el mismo
recibo devuelve la entrega ya registrada —el reintento legítimo no falla ni duplica— y no crea una
segunda fila. Una evaluación nueva emite otro recibo y sí registra otra entrega.

El choque de clave solo se interpreta como reenvío cuando PostgreSQL informa `23505` sobre la tabla
`submissions` y la restricción `submissions_pkey`. Cualquier otro fallo de escritura conserva su error
propio y no se presenta como reintento.

### 8. La rúbrica es una acción docente

`POST /api/v1/submissions/{id}/rubric-evaluation` exige rol `teacher` o `admin` y una identidad. El
docente solo alcanza entregas de las materias que tiene asignadas; un administrador conserva el alcance
de su institución. Un rechazo no escribe nota, estado, comentario ni celda.

### 9. Las métricas del juez no mezclan proyectos

Las tasas de éxito por dificultad y etiqueta, los casos fallados y las sugerencias de refuerzo se
calculan solo con ejercicios de `JUEZ_EFIMERO`: se apoyan en el veredicto de casos de prueba. Un
proyecto de IDE no tiene veredicto y su nota vive en la libreta (`exercise_grades`). No se introduce un
umbral de aprobación para proyectos: la regla existente del resumen del alumno (nota >= 51 para contar
como completado) no se reutiliza aquí porque estas series no miden notas docentes. Si se quiere una
métrica de proyectos, requiere una decisión de producto aparte.

### 10. La nota manual de otro docente se cambia con motivo

La celda `source = 'manual'` conserva autor y motivo. Si otro docente cambia la nota —por la revisión o
por la rúbrica— necesita una justificación de al menos 10 caracteres, que se escribe como motivo de la
celda. La validación ocurre antes de escribir: un intento rechazado no deja la revisión a medias ni la
libreta desactualizada. El mismo docente puede ajustar su propia nota sin un motivo nuevo, y ese ajuste
conserva el motivo que ya estaba escrito. La recorrección justificada reemplaza la autoría vigente.

La celda sigue apuntando a la entrega que ya representaba: recalificar otra entrega del mismo alumno no
cambia cuál está seleccionada en la libreta. Elegir otra entrega es una decisión de producto pendiente
de confirmación, no una consecuencia implícita de la recorrección.

## Migración

| Archivo | Contenido | Reversión |
| :--- | :--- | :--- |
| `00049_submission_verdict_optional.sql` | `ALTER COLUMN verdict DROP NOT NULL` | Devuelve `PENDING` a las filas nulas y restaura `NOT NULL`. Es una acción de despliegue, no una política de datos |
| `00050_retire_pending_sentinel.sql` | Convierte `PENDING` en ausencia solo para entregas de IDE | Irreversible por diseño, como `00035` |
| `00051_submission_verdict_origin.sql` | Agrega `verdict_origin` con su `CHECK` y clasifica las filas existentes por evidencia | Elimina la columna y su `CHECK`. Los veredictos no se tocan |

Orden de despliegue: primero el código que lee con `COALESCE`, después `00049`, y al final `00050`.
Las escrituras de `PENDING` se retiraron cuando la columna ya admitía nulos. `00051` va al final: el
código nuevo escribe la columna pero tolera leerla nula.

Orden de reversión: `00051`, `00050` y `00049`, en ese orden.

- `00051` Down elimina la columna y su `CHECK`. Es la única operación con pérdida: el origen es el único
dato nuevo, y una fila que era `judge` vuelve a `legacy` si se aplica de nuevo la migración, porque la
firma que la identificaba ya no existe. No se pierde ningún veredicto ni ninguna nota.
- `00050` Down no ejecuta SQL: devolver el centinela a las filas que hoy declaran su ausencia no
distingue las históricas de las nuevas. La reversión de esta migración es restaurar la base.
- `00049` Down devuelve `PENDING` a las entregas sin veredicto y restaura `NOT NULL`. Es una acción de
despliegue para volver a una versión anterior del código, no una limpieza de datos.

### Datos históricos

| Registro | Tratamiento |
| :--- | :--- |
| IDE con `verdict = 'PENDING'` | Pasa a ausencia. No cambia la nota ni el estado de revisión |
| IDE con `verdict = 'AC'` y `override_reason = 'Evaluado por rúbrica'` | Intacto. La nota es válida; el veredicto no es del juez y queda como legado documentado |
| IDE con veredicto por override manual | Intacto, misma razón |
| `JUEZ_EFIMERO` con veredicto declarado por el navegador | Intacto. No es verificable y no se convierte en evaluación |

Ningún registro histórico se convirtió en evaluación verificada y ninguna nota se modificó.

La clasificación de origen sigue la misma regla: solo se atribuye lo que dejó una firma verificable.

| Evidencia en la fila | `verdict_origin` |
| :--- | :--- |
| `override_reason = 'Evaluado por rúbrica'` en un ejercicio de IDE | `rubric` |
| `manual_override = TRUE` en un ejercicio de IDE, sin la firma de rúbrica | `override` |
| Cualquier otra fila con veredicto | `legacy` |
| Fila sin veredicto | `NULL` |

La firma de rúbrica tiene precedencia sobre `manual_override` porque la rúbrica escribía ambas cosas.
Los veredictos del juez que declaró el navegador caen en `legacy`: el servidor no puede demostrar su
procedencia, así que no se etiquetan como `judge`.

## Consecuencias

- La entrega de IDE deja de tener un veredicto artificial y la libreta deja de contar como automática
  una corrección humana.
- Las vistas del alumno muestran el badge de veredicto solo cuando existe un veredicto real. En un
  proyecto, la nota y la marca de modalidad comunican el estado.
- El contrato de revisión del juez devuelve `files: []` en lugar de `null`, que es lo que la interfaz
  ya esperaba.
- Una práctica de IDE sin rúbrica sigue siendo calificable: el ajuste manual escribe la nota por la
  revisión docente sin exigir criterios.
- `submissions.graded_by` nombra al docente que asignó la nota vigente: una recorrección por otro
  docente lo reemplaza, y la celda de la libreta registra el mismo actor en `overridden_by`. El
  historial de quién corrigió antes queda en la fila de la revisión, no en `graded_by`.
- Las vistas del docente muestran la procedencia en lugar del resultado técnico cuando el veredicto no
es del juez, y omiten el badge cuando no hay veredicto. La información histórica no se elimina:
  cambia la etiqueta, no el dato.
- Un registro `legacy` conserva su código técnico y lo acompaña con «Resultado heredado». La rúbrica
  histórica y el ajuste docente siguen mostrando solo su procedencia, porque ese código lo escribió un
  camino que no es el juez.
- Lo mismo vale para el resumen de evaluaciones del alumno: la lista incluye ejercicios de IDE y
  `last_verdict` toma el veredicto de la entrega más reciente, así que el resumen viaja con
  `last_verdict_origin` del mismo registro. Sin ese campo, un proyecto corregido con rúbrica se leería
  como un resultado del juez.

## Alternativas descartadas

- **Mantener `PENDING` como estado de presentación.** Conserva el centinela en la base y obliga a
  distinguir dos significados del mismo código.
- **Derivar `AC` en la consulta cuando hay nota.** Reintroduce el veredicto inventado en la capa de
  lectura, donde nadie lo audita.
- **Evaluación automática del snapshot para IDE.** Se descartó antes de implementar: el IDE conserva su
  enfoque de desarrollo y la evaluación docente con rúbrica.

## Verificación

- Pruebas unitarias de servicios y transporte en verde.
- `TestSlice13_TeacherReviewModalityAndRubricClosure`: la rúbrica califica con 50, deja `verdict` nulo,
  no marca `manual_override` y escribe la celda con `source = 'manual'` y `overridden_by` del docente.
- `TestSlice13_TeacherExerciseBank`: el pendiente de IDE se cuenta por `review_status`.
- `TestProjectManualGrade_RecordsScoreProvenanceAndAuthor`: la corrección manual de un proyecto sin
  rúbrica escribe la nota, `graded_by` y la celda manual; una recorrección por un segundo docente
  reemplaza la autoría de la nota vigente; una entrega IDE en `not_required` pasa a `graded`.
- `TestEvaluate_RejectsIDEExercise`: `POST /api/v1/evaluations` rechaza un ejercicio de IDE sin
  ejecutar el juez.
- `TestGetStudentEvaluationsClassifiesAcademicStatus`: el resumen de evaluaciones conserva el origen del
  último resultado; un proyecto con `verdict_origin = 'rubric'` no se presenta como evaluación del juez.
- `TestCreateSubmission_*`: una entrega de IDE se rechaza, un veredicto sin recibo del servidor no se
  persiste, el recibo manda sobre el payload y no se reutiliza en otro ejercicio, alumno, institución ni
  código; reenviar el mismo recibo devuelve la misma entrega y una evaluación nueva registra otra.
- `TestEvaluate_ComputesVerdictOnServerAndIgnoresClientVerdict`: el juez ejecuta los casos, el veredicto
  sale de ellos y el recibo lo firma; un `verdict` en el cuerpo se ignora.
- `TestSubmissionReceipt_RegistersOneSubmissionPerEvaluation`: contra PostgreSQL, un recibo deja una
  sola fila y el reenvío devuelve esa misma entrega.
- `TestSubmissionReceipt_ConcurrentReplayPersistsOneSubmission`: dos solicitudes simultáneas con el mismo
  recibo dejan una fila y recuperan el mismo identificador.
- `TestSubmissionReceipt_MappingDoesNotMaskOtherErrors`: un fallo distinto del choque de la clave no se
  reporta como reintento.
- `TestSignEvaluationReceipt_EachEvaluationHasOwnIdentity`: dos emisiones idénticas en el mismo segundo
  producen tokens y entregas distintas.
- `TestRubricEvaluation_RejectsStudentAndAnonymous` y `TestRubricEvaluation_RequiresSubjectOwnership`:
  el alumno no califica, el docente ajeno recibe 404 y el rechazo no deja nota ni celda.
- `TestManualGradeCell_RequiresJustificationFromAnotherTeacher`: la nota manual de otro docente no se
  reemplaza sin motivo, la recorrección justificada conserva autor y motivo, y un ajuste propio sin
  motivo nuevo no borra el motivo anterior.
- `TestCourseAnalytics_SeparatesJudgeFromIDEGrades`: un proyecto de IDE con nota docente no altera las
  tasas de éxito ni los casos fallados del juez.
- Migraciones 49, 50 y 51 aplicadas sobre PostgreSQL 18 y probadas en ambas direcciones donde son
  reversibles. El backfill de `00051` se verificó sobre filas sembradas: `rubric`, `override`,
  `legacy` y nulo según la tabla anterior.
