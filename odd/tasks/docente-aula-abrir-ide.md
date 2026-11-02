# Feature: Abrir mi IDE desde el Aula

Objetivo: que el docente abra el entorno de la práctica que está dando clase desde la cabecera del Aula, sin salir de la supervisión ni pasar por la ficha de la materia.

Branch: `feat/aula-open-ide` (desde `main`) | Delivery strategy: ask-on-risk | Test runner: `npm run test:ci`

Estado: **planificada**. Bloqueada por D1 (aula sin práctica). El resto se puede implementar ya.

Fuente: `proyecto-grado/Observaciones/Laboratorios y Flujo del Entorno Docente.md` §12. Decisión del usuario: «Acceso directo en el Aula: El botón directo para abrir el IDE sin salir de la clase».

## Decisiones cerradas

- El botón vive en la cabecera de `/teacher/aula`, junto a `Emitir avance`.
- Abre el entorno de la práctica de la sesión (`ClassSession.exercise_id`) reutilizando `TeacherEnvironmentService.openPracticeIDE`.
- Se abre en pestaña nueva, nunca en iframe. La pestaña se reserva en el clic para que el navegador no la bloquee.
- No hay botón de detener: el QoS apaga el contenedor por inactividad (ADR-011).

## Qué existe hoy (verificado en el código)

- Cabecera: `frontend/src/app/features/teacher/aula/aula.component.html:27-72` tiene presencia, `Emitir avance`, `Congelar/Descongelar` y `Cerrar aula`. No hay acción de IDE.
- Modelo: `ClassSession` incluye `exercise_id: string | null` y `exercise_title` (`frontend/src/app/features/teacher/models/teacher.models.ts:803`).
- Servicio: `TeacherEnvironmentService.openPracticeIDE(courseId, exerciseId)` llama a `POST /api/v1/teacher/courses/{id}/exercises/{exerciseId}/ide` y devuelve `{access_url, workspace_id, ...}` (`frontend/src/app/features/teacher/services/teacher-environment.service.ts:82`).
- Patrón de apertura: `PracticeDetailComponent.openIde()` (`frontend/src/app/features/teacher/courses/practice-detail/practice-detail.component.ts:246`) reserva `window.open('', '_blank')`, cierra la pestaña si falla y cae a `window.location.assign` si el navegador bloqueó la ventana emergente.
- El Aula ya importa `LucideTerminal` y no tiene spec propio (`frontend/src/app/features/teacher/aula/` no contiene `.spec.ts`).
- El aula permite abrirse sin práctica: `openAula(subjectId, selectedExerciseId() || null, duration)` (`aula.component.ts:304-312`).

## Specs

- **S1 — Acción en la cabecera.**
  - Botón `Abrir mi IDE` con icono `LucideLaptop`, variante secundaria, entre presencia y `Emitir avance`.
  - Deshabilitado si no hay `aula.exercise_id` o si `isOpeningIde()` es verdadero.
  - Mientras abre: spinner + `Abriendo IDE…`; al fallar: mensaje visible debajo de la cabecera con `describePracticeIDEError`.
  - Criterio: con una sesión de práctica IDE, el botón abre la URL del entorno en una pestaña nueva.

- **S2 — Método del componente.**
  - `openIde()` en `AulaComponent` reserva la pestaña, llama a `openPracticeIDE(aula.subject_id, aula.exercise_id)` y navega al `access_url`.
  - Si la respuesta no trae URL, cierra la pestaña y muestra el error.
  - Si `window.open` devuelve `null`, navega en la pestaña actual.
  - Criterio: un error del backend no deja una pestaña en blanco.

- **S3 — Regla de práctica nula.**
  - Con `exercise_id = null`, el botón queda deshabilitado y el `title` dice por qué: «Abrí el aula con una práctica para usar tu IDE».
  - No se abre el entorno por materia: ese modelo está retirado (ADR-059).
  - Criterio: un aula sin práctica no ofrece una acción que no puede resolver.

- **S4 — Tests y documentación.**
  - Crear `aula.component.spec.ts` con un doble de `TeacherEnvironmentService`: abre en pestaña nueva, cierra al fallar, deshabilitado sin práctica.
  - Actualizar `docs/UI/DOCENTE/DASHBOARD.md` o la guía del Aula con la acción y su regla.
  - Criterio: los tres casos quedan cubiertos sin Docker.

## Tasks

- [ ] **T1 — S1**: `aula.component.html` (botón, spinner, error) y `aula.component.scss` si hace falta.
- [ ] **T2 — S2**: `aula.component.ts` (`openIde`, señales de estado, import de `TeacherEnvironmentService` y `LucideLaptop`).
- [ ] **T3 — S3**: regla de `exercise_id` nulo y texto de ayuda.
- [ ] **T4 — S4**: spec nuevo y documentación.

## Dudas abiertas

- **D1 — Aula sin práctica.** Recomendación: deshabilitar con explicación. Alternativa: abrir un selector de práctica antes de lanzar el entorno; más pasos y duplica la decisión que ya vive en la ficha.
- **D2 — Formulario manual de apertura.** El documento pide que la clase nazca desde la práctica o la materia, sin formulario. Esta ficha **no** lo cambia; solo agrega el botón. Confirmar si se abre una ficha aparte para el arranque contextual.
- **D3 — Indicador de clase activa en el sidebar.** `Corregir.md` pide un punto verde o «En curso» en el menú. No es parte de esta decisión; queda como pendiente separado.
- **D4 — Botón «Iniciar Aula» en la cabecera de la materia.** También aparece en `Corregir.md` y tampoco está en esta decisión. Pendiente separado.

## Verificación

```bash
cd frontend && npx eslint src/app/features/teacher/aula/aula.component.ts
cd frontend && npm run test:ci -- --include='**/aula/aula.component.spec.ts'
cd frontend && npm run lint:styles
cd frontend && CI=true npm run build
```

## Log

- **L1 (verbatim):** «Acceso directo en el Aula: El botón directo para abrir el IDE sin salir de la clase.»
- **L2 (verbatim):** «Para entender cómo debe conectarse el botón [Abrir mi IDE] adentro del Aula (/teacher/aula), y por qué el inicio de clase debe nacer directo desde la práctica sin formularios manuales.»
- **Nota:** L2 mezcla dos temas. El botón es esta ficha; el arranque de clase sin formulario es una decisión aparte (D2).
