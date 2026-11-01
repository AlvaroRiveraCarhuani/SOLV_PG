# Feature: Flujo de Laboratorio Docente (entorno por práctica)

Objetivo: que el docente pueda preparar, probar y ajustar cada práctica desde su detalle, con un entorno propio por práctica, sin mezclarlo con las entregas de los alumnos.

Branch: pendiente de definir | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm run test:ci`

Estado: **en curso**. S1 (fila de la materia), S2 (fichas IDE y juez), S3 (prueba de entorno), S4 (apertura del IDE por práctica) y S5 (retiro de `Mi entorno` y contexto precargado del aula) implementados; S6 pendiente.

Fuente del problema: `proyecto-grado/Observaciones/Laboratorios y Flujo del Entorno Docente.md` (§1–§5) y `proyecto-grado/Observaciones/INICIO.md`.

## Problema actual

- La acción `Probar Entorno` de cada práctica IDE (`IDE_PERSISTENTE`) no levanta ningún contenedor ni abre VS Code. Redirige a `exercise-preview`, pensada para problemas algorítmicos, y muestra una tarjeta estática con imagen y memoria. El docente no puede validar que el repositorio base, las dependencias y los satélites funcionen antes de la clase.
- `Fuzzing` aparece en todas las filas, aunque solo aplica a laboratorios de juez virtual. Hoy el código ya lo limita (`canFuzz` en `teacher-exercises.component.ts`), pero la fila de la materia (`teacher-course-detail.component.html`, botón `btn-fuzz`) no lo aplica igual.
- Cada fila de laboratorio tiene cuatro acciones (`Revisar entregas`, `Probar Entorno`, `Fuzzing`, `Editar`) sin jerarquía, y no muestra el stack del laboratorio.
- `Mi entorno` en el menú docente pide elegir una materia para abrir VS Code. Una materia tiene varias prácticas con stacks distintos, así que ese entorno no corresponde a ninguna práctica concreta.

## Decisiones cerradas

- **Entorno por práctica.** Cada práctica tiene su propio entorno. Un laboratorio puede usar una tecnología y otro, otra. El entorno no es por materia ni por sesión.
- **Entorno de prueba separado del de alumnos.** `Probar Entorno` levanta un entorno propio del docente para validar la práctica. No reutiliza el arranque de alumnos (`exercise_id` de `persistent-ide-backend-oa1`) y no crea entregas.
- **Ubicación del código en vivo.** El código en vivo del docente se vincula al aula (`/teacher/aula`) para proyectar y emitir avances. `Mi entorno` sale del menú lateral. Esto ya está en la nota de observaciones; se confirma al cerrar la ficha.
- **Ficha por modalidad.** Cada práctica muestra la ficha según `environment_type`:
  - IDE persistente: `Probar Entorno (Abrir VS Code)` como acción principal, `Editar` como secundaria, estado de satélite de base de datos, política de re-entregas y rúbrica.
  - Juez virtual: `Probar Solución (Simulador)` como principal, `Fuzzing de Casos` como secundaria y casos de prueba en pestañas.
- **Fuzzing solo para juez virtual.** Sale de las filas de IDE y de cualquier entorno de proyecto.

## Qué ya existe (verificado en el código)

- `backend/internal/core/services/env_test_service.go` y `backend/internal/core/domain/env_test_job.go`: servicio de prueba de entorno del estudio de plantillas, con trabajo asíncrono, estados (`pending`, `pulling`, `testing`, `success`, `failed`, `canceled`) y errores de descarga y ejecución. S3 reutiliza el job, sus estados y su repositorio, no su ejecución.
- `backend/internal/core/services/fuzzing_engine.go` y `teacher_sessions_fuzz.go`: generación y aplicación de casos de fuzzing, con endpoints en `router.go`.
- `frontend/.../teacher-exercises.component.spec.ts`: prueba que `canFuzz` es verdadero solo para juez virtual.

## Specs (propuestas, sin cerrar)

- **S1 — Fila de la materia.**
  - Toda la fila lleva a la práctica con un clic.
  - La fila muestra el contador de pendientes de revisión, sin botón propio.
  - Se quitan `Probar Entorno`, `Fuzzing` y `Editar` de la fila; viven en la vista de la práctica.
  - Quitar `Fuzzing` de las filas IDE va primero, como cambio aparte.
  - Criterio: en una materia con IDE y juez, `Fuzzing` aparece solo en la vista de las prácticas de juez.

- **S2 — Vista de la práctica por modalidad.**
  - Ruta de detalle del laboratorio con las dos fichas (IDE y juez).
  - Prototipos de referencia: `proyecto-grado/Observaciones/propuesta-detalle-laboratorio-ide.html` (IDE) y `propuesta-detalle-laboratorio.html` (juez).
  - Criterio: la ficha IDE no muestra casos de prueba ni simulador; la de juez no muestra VS Code.

- **S3 — Entorno de prueba del docente.**
  - `Probar Entorno` lanza un trabajo de prueba de la práctica con sus dependencias y satélites, sin entregas.
  - Reutiliza `EnvTestService` si el análisis de S3 lo confirma; si no, se decide si extenderlo o crear un flujo propio.
  - Estado visible con los estados de `EnvTestStatus`.
  - Criterio: el docente ve el progreso y el resultado; el trabajo no aparece en las entregas de ningún alumno.

- **S4 — Apertura del IDE del docente.**
  - Depende de la decisión 1 (abrir dentro de la vista o en pestaña nueva).
  - Criterio: el entorno abre con el repositorio base de la práctica y no con el de otra práctica de la misma materia.

- **S5 — Quitar `Mi entorno` del menú.**
  - El acceso al código en vivo pasa al aula.
  - Criterio: ninguna ruta del menú pide elegir materia para abrir VS Code.

- **S6 — Verificación y documentación indivisible.**
  - `cd backend && go vet ./... && go test ./...`.
  - `cd frontend && npm run lint:styles && npm run lint:ts && npm run test:ci`.
  - Actualizar `docs/API_CONTRACTS.md` con cualquier endpoint nuevo y `docs/UI/DOCENTE/` con la ficha de práctica.
  - Revisar ADR-058 (avance voluntario) porque asume un entorno por sesión de aula. Hay que decidir si se ajusta el ADR o se abre uno nuevo.

## Estado de implementación

- S1 — hecho. La fila abre la ficha con un clic, el contador de pendientes no es botón y las acciones viven en la vista. `Publicar` permanece en la fila solo para borradores.
- S2 — hecho. Ruta `/teacher/cursos/{courseId}/practicas/{labId}` con las dos fichas; Fuzzing de Casos solo aparece en juez.
- S3 — hecho. `Probar Entorno` inicia un job por práctica que aprovisiona los satélites con una identidad de prueba, ejecuta el setup de la plantilla sobre un volumen temporal y exige OpenVSCode Server. Reutiliza `EnvTestJob`, sus estados y su repositorio; no crea entregas ni workspaces. El job sobrevive al request y se puede cancelar.
- S4 — hecho. `Abrir IDE` en la ficha abre en una pestaña nueva el entorno del docente acotado a esa práctica: volumen `solv_workspace_<docente>_<materia>_<práctica>`, plantilla y `db_init_script` del ejercicio, y reutilización por `(docente, materia, práctica)`. Solo prácticas IDE; una del juez responde `400`, la materia de otro docente `403` y un volumen congelado `409`. Decisión y consecuencias en ADR-059.
- S5 — hecho. `Mi entorno` sale del menú docente y `/teacher/entorno` redirige a Hoy; el código en vivo se abre desde la ficha de cada práctica. El aula conserva su entrada en el menú como ancla permanente y precarga materia y práctica cuando se llega desde la ficha (`?subject_id=&exercise_id=`). La emisión de avances deja de buscar el entorno por materia: con sesión de práctica lee el volumen de esa práctica (ADR-059) y mantiene el entorno por materia como respaldo de aulas anteriores.
- S6 — pendiente (verificación y documentación). Cubierta parcialmente para S4 y S5: `docs/API_CONTRACTS.md` §7.3, `docs/UI/DOCENTE/DASHBOARD.md`, `docs/ARQUITECTURA/IDE_PERSISTENTE.md` §2.1, ADR-058 §5 y ADR-059.

## Dependencias

- `docente-catalogo-entornos.md`: el entorno de cada práctica sale del catálogo. Esta ficha no decide qué plantillas existen.
- `persistent-ide-backend-oa1.md`: arranque por `exercise_id`. Esta ficha lo usa solo para el alumno, no para la prueba del docente.
- `docente-avance-voluntario.md` y ADR-058: el volumen de avance y la sesión de aula dependen del entorno por práctica.
- Fuzzing: la regla de la fila no depende de las demás fichas.

## Decisiones cerradas (ronda 2)

1. **Apertura del IDE:** pestaña nueva a pantalla completa, con URL estable por práctica mientras el entorno existe. Un iframe complica WebSockets, CSP, Traefik y la sesión de OpenVSCode Server.
2. **Lista de la materia:** toda la fila lleva a la práctica con un clic. La fila muestra solo el contador de pendientes de revisión (sin botón propio). Las acciones (`Probar Entorno`, `Editar`, `Fuzzing` en juez) viven en la vista de la práctica, según los mockups `propuesta-detalle-laboratorio-ide.html` y `propuesta-detalle-laboratorio.html`.
3. **Eliminar `Mi entorno`:** sí. El aula elige la práctica de la sesión y abre su entorno.
4. **Fuzzing:** cambio aparte y previo a esta ficha (S1), porque es independiente.
5. **`EnvTestService`:** reutilizar el modelo de estados y el vocabulario, no el flujo. La prueba del docente arranca la práctica con su repositorio y satélites. Falta revisar la implementación completa antes de S3.

6. **Estado del entorno en la vista de la práctica:** no lleva encender, apagar ni reiniciar. La prueba del docente es un job efímero (`EnvTestStatus`: `pending`, `pulling`, `testing`, `success`, `failed`, `canceled`), no un contenedor persistente con energía. La vista muestra solo el estado del último job de prueba de esa práctica (sin pruebas, en curso, último resultado con su hora). Los controles de energía (`pause` / `restart` de workspaces) son del alumno y no se reutilizan aquí. El mockup IDE ya coincide: su único badge de estado es `Publicado` (publicación de la práctica), no el estado del entorno.

## Decisiones pendientes

- Ninguna abierta. S3 resolvió el vínculo con `exercise_id` y `subject_id` en `EnvTestJob`, más `GetLatestByExercise` en el repositorio y una identidad de prueba estable para los satélites.

## Fuera de alcance

- El catálogo de plantillas y su curaduría (`docente-catalogo-entornos.md`).
- La sesión de aula con conectados, telemetría y avance (`docente-aula-y-sesiones.md`, `docente-avance-voluntario.md`).
- Crear o editar la plantilla del entorno desde el docente.
