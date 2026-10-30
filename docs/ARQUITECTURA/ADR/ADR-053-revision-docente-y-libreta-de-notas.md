# ADR 053: Estado de Revisión Docente Explícito y Celda de Nota por Práctica

> **Estado:** Aprobado
> **Fecha:** 2026-10-30
> **Ámbito:** Core, Docencia, Libreta de notas
> **Relacionado:** ADR-052 (read model del banco de ejercicios), ADR-043 (modalidad dual IDE/juez), ADR-021 (rúbricas)

---

## Contexto

El área docente arrastraba dos defectos estructurales que la interfaz no podía corregir:

1. **La cola de corrección se deducía del veredicto.** `pending_review` en el resumen de materias y la cola de entregas se armaban con `WA`, `RE`, `TLE`, `failed` y `oom_killed`. Eso mostraba como tarea pendiente un ejercicio de juez ya calificado automáticamente y **ocultaba** las entregas de un laboratorio IDE, que permanecen en `PENDING` hasta que el docente las corrige con rúbrica en SpeedGrader.
2. **La libreta inventaba notas.** `GetCourseGradesMatrix` recalculaba la mejor nota en cada lectura y, cuando la entrega no tenía `score`, asignaba 0 salvo que el veredicto fuera `AC`. Una entrega IDE sin corregir aparecía como un cero, de modo que "no entregó" y "no lo corregí" se veían igual.

El esquema ya guardaba `score`, `graded_by`, `manual_override` y `override_reason` por entrega, pero no existía un estado de revisión ni una entidad que dijera **qué entrega cuenta** para la nota de un alumno en una práctica.

---

## Decisión

### 1. Estado de revisión declarado en la entrega

`submissions.review_status` con cinco valores y transiciones validadas en el dominio:

```
not_required   ← juez que califica solo (no vuelve a la cola)
pending        → in_review → graded → returned
                 ↑            │         │
                 └────────────┴─────────┘ (reabrir o devolver)
```

- `reviewed_at`, `reviewed_by`, `feedback_summary` y `returned_at` acompañan al estado.
- Un reintento crea una **entrega nueva**; no reabre una revisión cerrada.
- Calificar en SpeedGrader lleva la entrega de `pending`/`in_review` a `graded`.
- La transición se valida en el servicio y se escribe con una condición sobre el estado leído, para que dos docentes no pisen la misma corrección.

### 2. Celda de nota por alumno y práctica

Nueva tabla `exercise_grades` con PK `(student_id, exercise_id)` que guarda:

- `selected_submission_id`: qué entrega cuenta.
- `score`: nota de la celda (`NULL` = sin corregir).
- `source`: `auto` (sugerida por el sistema) o `manual` (elegida por el docente).
- `overridden_by` y `reason`: quién la cambió y por qué.

Regla de selección: el sistema sugiere la mejor entrega calificada; el docente puede cambiarla. Cambiar la nota respecto de la calificación de la entrega exige un motivo de al menos 10 caracteres. Toda selección queda en `audit_logs`.

### 3. La libreta no fabrica ceros

- `grades[exercise]` viaja como `null` cuando no hay nota.
- La interfaz muestra `—` y "sin corregir" en esos casos.
- El promedio se calcula solo sobre prácticas calificadas y la fila expone `graded_count` y `pending_count`.
- La exportación CSV deja la celda vacía en lugar de escribir 0.

### 4. Cola agrupada por práctica

`GET /api/v1/teacher/reviews` devuelve `{exercise_id, title, subject, pending, total, oldest_submitted_at}` por práctica, ordenada por antigüedad. `total` es el número de inscritos en la materia; `pending` cuenta `review_status IN ('pending','in_review')`.

---

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Mantener la deducción por veredicto y agregar `PENDING` a la lista | Sigue mezclando dos hechos distintos: el resultado del juez y el trabajo humano pendiente. Un `WA` de un ejercicio de clase ya está calificado; un `PENDING` de un IDE es trabajo real. |
| Reutilizar `verdict = 'PENDING'` como estado de revisión | Un solo campo para dos conceptos; impide distinguir "el juez aún no terminó" de "el docente aún no corrigió". |
| Seguir calculando `max(score)` en cada lectura | No permite elegir la entrega que cuenta, no audita el cambio y obliga a inventar un valor cuando falta la nota. |
| Guardar la nota solo en `submissions.score` | Es una nota por entrega, no por alumno y práctica; no hay celda con la que construir una libreta estable. |

---

## Consecuencias

**Positivas**

- La cola de corrección coincide con el trabajo real y deja de mostrar resultados automáticos.
- La libreta deja de castigar con 0 el trabajo no corregido.
- Queda trazabilidad de quién cambió una nota y por qué.
- El mismo estado habilita el aula en vivo y las notificaciones al alumno.

**Costos y cuidados**

- Migración `00034` con backfill: IDE sin nota → `pending`; celdas creadas solo para parejas con nota real.
- El `CHECK` de `review_status` se recrea con `DROP … IF EXISTS` + `ADD`, porque el migrador parte los bloques `DO` por el punto y coma interno.
- `StudentGradesRow.grades` cambia de `map[string]int` a `map[string]*int`: cualquier consumidor debe tratar el nulo como "sin corregir", nunca como 0.
- El promedio histórico de una materia cambia cuando hay prácticas sin corregir: ahora refleja solo lo calificado y lo declara con `graded_count`.

---

## Corrección posterior (2026-10-30)

La prueba contra la base real mostró dos huecos de la primera implementación:

- El `INSERT` de `submissions` no escribía `review_status`, así que toda entrega nueva nacía `not_required` y la cola nunca se habría llenado.
- La entrega de un laboratorio IDE se creaba con `score = 0`. El backfill leyó ese 0 como una nota real y marcó la entrega como `not_required`; la libreta creó una celda con 0 para cada una.

La migración `00035_teacher_review_correction.sql` devuelve esas 21 entregas a `pending`, borra el score fabricado y elimina las 21 celdas que nacieron de él. El alta de la entrega ya no asigna `Score` y declara el estado inicial según la modalidad: IDE → `pending`, juez → `not_required`.
