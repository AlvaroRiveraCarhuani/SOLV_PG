# Feature: Inicio "Hoy" y Navegación Docente por Tarea

Objetivo: reemplazar el panel de widgets reordenables por una pantalla de despacho y una navegación que siga el trabajo del docente, no las capacidades del sistema.
Branch: `feat/teacher-home-navigation` (desde `main` tras las dos features anteriores) | Delivery strategy: inline si el forecast se mantiene bajo 400 líneas | Test runner: `npm run test:ci`

Forecast de líneas autoradas: ~700 (frontend ~600, docs ~100). Si al implementar supera 400, partir en dos: primero navegación e inicio, después retiro de widgets.

Diseño de referencia: `docs/UI/DOCENTE/REDISENO_PANEL_DOCENTE.md` (§2.4, §3.1, §4).

## Decisiones cerradas

- **El inicio es un despacho, no un tablero.** Responde una sola pregunta: qué clase tengo y qué me espera. Una columna, no dos: a 768 px de alto dos columnas de importancia desigual roban altura a la principal.
- **Regla de sustitución.** Si hay sesión hoy, se muestra con su acción. Si no, sube la cola de calificación. Si tampoco hay cola, estado vacío con "Abrir aula". Nunca una pantalla vacía.
- **Se retira el grid reordenable.** Cuatro bloques iguales comunican que todo importa igual; además desplazan al docente la decisión de qué mirar. `dashboard-layout.service.ts` y el orden persistido en `localStorage` se eliminan.
- **Se retiran las KPIs del panel.** Repetían lo que ya dicen las tarjetas y usaban definiciones incompatibles (una sumaba matrículas, otra contaba workspaces). La cifra útil pasa a la línea de estado del encabezado.
- **Se retira el widget de atención.** Su contenido se reparte entre las excepciones del aula y la cola de calificación, con dueño explícito.
- **La telemetría sale del inicio.** Vive en el aula. En el inicio no hay ningún dato técnico.
- **"Señales" no es destino de navegación.** Es una sección dentro de la práctica.
- **La navegación se organiza por horizonte de trabajo.**

## Specs

- **S1 — Navegación por tarea.**
  - `TEACHER_NAV_SECTIONS` pasa a cuatro bloques de trabajo: `HOY` (Inicio), `DOCENCIA` (Mis materias · Por calificar · Notas), `CONTENIDOS` (Prácticas · Banco de ejercicios · Plantillas), `ANÁLISIS` (Estudiantes), `CUENTA` (Configuración).
  - "Por calificar" muestra el conteo pendiente cuando es mayor que cero.
  - Criterio: ninguna ruta queda huérfana; las rutas existentes de evaluaciones se mapean a "Por calificar" o a "Notas" sin enlaces muertos.

- **S2 — Pantalla Hoy.**
  - Franja de sesión con hora, materia, práctica, inscritos y acción; tarjeta de cola con cifra y enlaces.
  - Estados: con sesión, sin sesión con cola, sin nada (estado vacío con acción), cargando con `skeleton`, error de red con reintento.
  - Criterio: los cinco estados están implementados y probados; ninguno deja la pantalla en blanco.

- **S3 — Retiro del panel anterior.**
  - Se eliminan `dashboard-kpi-widget`, `dashboard-attention-widget`, `dashboard-telemetry-widget` y el uso de `dashboard-layout.service.ts` desde el inicio; la telemetría se reutiliza dentro del aula con datos de la sesión.
  - Se elimina la consulta `/api/v1/teacher/attention` y su read model si no queda consumidor; si se conserva para el aula, se documenta el motivo.
  - Criterio: `grep` no encuentra referencias a los widgets retirados y la suite sigue en verde; ningún endpoint queda sin consumidor sin justificar.

- **S4 — Materias como registro.**
  - La grilla auto-fill de tarjetas de 260 px se reemplaza por una lista con buscador y orden por pendientes, conservando el color de materia y la acción de entrar.
  - Criterio: con 30 materias, el docente encuentra una sin desplazarse más de una pantalla y ve cuántas tiene pendientes.

- **S5 — Documentación.**
  - `docs/UI/DOCENTE/DASHBOARD.md` se reescribe como "Inicio"; `docs/UI/MAPA_DE_VISTAS.md` actualiza la navegación docente.
  - Criterio: no queda en la documentación ninguna referencia a widgets reordenables ni a KPIs del panel.

## Tasks

- [x] **S1**: navegación por horizonte de trabajo. Entregado con los bloques `HOY` (Inicio), `DOCENCIA` (Aula · Por calificar), `CONTENIDOS`, `SUPERVISIÓN` y `CUENTA`. Los destinos "Mis materias" y "Notas" no entran porque no existen como pantalla: la tabla de materias del inicio las cubre hasta la etapa 3, cuando la materia tenga sus pestañas.
- [x] **S2**: pantalla Hoy con sus estados. Se alimenta de `GET /api/v1/teacher/overview`, que resuelve período, fila de trabajo, cola y materias en cuatro consultas agregadas sin bucle por materia. Estados: esqueleto al cargar, filas ocultas sin contenido, vacío de materias con el selector de período, error con reintento que conserva lo cargado.
- [x] **S3**: retiro de widgets y del uso docente del servicio de layout. `dashboard-kpi-widget`, `dashboard-attention-widget` y `dashboard-courses-widget` se eliminan; la telemetría y la terminal de sombra pasan a `aula/`. `GET /api/v1/teacher/attention` se retira: ninguna pantalla leía su resultado, y su reemplazo es `signals` del resumen de `overview`. Se quitan la ruta, el read model, los tests y la llamada en `loadDashboardData`.
- [x] **S4**: sustituido por la tabla de materias del inicio con inscritos, entornos abiertos, por calificar y sin entregar. Se descarta la lista con buscador: con los conteos resueltos en el servidor, la tabla se lee sin desplazar y no hace falta una pantalla propia de materias hasta la etapa 3.
- [x] **S5**: `DASHBOARD.md` congelado con puntero al rediseño; `MAPA_DE_VISTAS.md` con las rutas reales del rol docente y las fichas de Hoy y Aula.

## Verificación ejecutada (2026-10-30)

- `npx ng build` sin errores.
- `npm run lint:styles` en verde: inline-styles, date-format, machine-data y stylelint.
- Specs del área: 17 tests en 3 archivos (Hoy, aula-telemetry-widget, shadow-terminal-modal) en verde.
- `go build ./...` y `TestSlice18_TeacherOverview_ReadModel` contra base real: la cola, los entornos abiertos y la tabla no discrepan; 403 para estudiante; 400 para `period_id` inválido.

## Log

- **L1 (verbatim):** "si ahora mismo eso de WA, RE, TLE, ya no lo estamos usando [...] para personas de primeros semestres puede ser complicado."
- **L2 (verbatim):** "tus propuestas estan medio bien, pero siento que son parches, osea ahora mismo no podemos quedarnos con el modelo que tenemos, o la iu, tiene que evolucionar a una mas potente y diseñada para lo que es, clases de programacion."
- **L3 (verbatim):** "Ahora, para arrancar hay cuatro decisiones que cambian el esquema" → respuestas registradas en las otras dos tareas; esta no introduce esquema nuevo.
- **Dependencia:** esta feature consume la cola de calificación (S3 de `docente-revision-y-libreta.md`) y el aula (S6 de `docente-aula-y-sesiones.md`). Implementarla antes deja "Hoy" sin destino para sus dos acciones.
- **Hallazgo verificado:** la grilla de materias usa `repeat(auto-fill, minmax(260px, 1fr))` dentro de 8 de 12 columnas; con 30 materias produce más de 2.000 px de alto sin buscador ni orden.

## Actualización (2026-10-30)

La pantalla `Hoy` no está en el árbol: sigue la grilla de widgets con su banner de atención y `Personalizar`. Los puntos S2 y S4 describen el objetivo de la etapa 1, no lo entregado, y S4 queda superado: la tabla se sustituye por el canvas de materias con columna de pendientes, especificado en [docente-hoy-canvas.md](docente-hoy-canvas.md) y en el ADR-057. Lo que sí quedó de esta feature es la navegación por horizonte de trabajo y el read model del resumen.
