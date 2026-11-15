# Feature: Cobertura de guías del panel administrador

Objetivo: cerrar la cobertura de recorridos y tutoriales del rol administrador, dejando una regla única de qué se enseña con una guía, qué se documenta como referencia y qué no se guía porque ejecuta una acción destructiva.
Branch: `main` | Delivery strategy: ask-on-risk | Test runner: `npm run test:ci`

Alcance: las trece rutas del administrador y las cinco pestañas de Configuración. No crea pantallas nuevas ni cambia contratos de backend.

## Decisiones cerradas

- **Primero la ficha, después el código.** Toda tanda de guías se documenta en `odd/tasks/` con sus specs y su matriz de verificación, y se suma a `odd/README.md`. Sin ficha no se implementa.
- **Un tutorial es un camino, no un catálogo.** El modelo de `TutorialDefinition` es lineal: cada paso avanza solo si una comprobación lee el estado real y no hay ramas. Un tutorial enseña a terminar una tarea sin romper nada; no cubre todas las variantes.
- **Los casos límite son referencia, no pasos.** Van en la entrada contextual del centro de ayuda y en la sección del manual. Ahí el formato es caso → qué hacer: dominio externo, invitación expirada, solicitud sin plantilla compatible, estudiante bloqueado sin entorno.
- **Un segundo tutorial solo con tres condiciones juntas:** el caso es frecuente, bloquea al usuario y no se descubre solo. Los candidatos admitidos son «solicitud sin plantilla compatible» e «invitación expirada».
- **Ninguna guía ejecuta una acción destructiva o irreversible.** Archivar, revocar, dar de baja, terminar contenedores y aprobar una solicitud se explican y quedan en manos del administrador. Crear o registrar sí se puede ejecutar, como ya hace el tutorial de Administradores.
- **Sin selector de escenario.** Añadirlo obliga a UI nueva en el centro de ayuda y a reiniciar estado entre escenarios; todavía no hay suficientes casos por pantalla para justificarlo.
- **El manual es el índice visible.** La tabla de la sección 11 de `docs/UI/ADMIN/MANUAL_ADMIN.md` se actualiza en el mismo cambio que la guía; `frontend/public/docs/MANUAL_ADMIN.md` es un enlace simbólico a ese archivo.
- **Criterio de aceptación de una guía:** el recorrido solo señala y no pide datos; el tutorial comprueba el estado real en cada paso con `check.isComplete` y un `pendingHint` accionable; ambos se pueden cerrar en cualquier momento.

## Estado de partida (verificado 2026-11-14)

Siete superficies tenían guías antes de esta ficha: Constructor de plantillas (recorrido y tutorial), Catálogo de plantillas (solo tutorial) y las cinco pestañas de Configuración (recorrido y tutorial cada una). El resto mostraba explicación contextual sin guía.

Registro real en código: `ContextHelpService.registerScreen` en cada pantalla con entrada en `CONTEXT_HELP_ENTRIES`; `registerGuides` solo en las pantallas con recorrido o tutorial. El botón `?` de la barra superior se oculta cuando la pantalla no tiene entrada ni guías, así que el manual no lo muestra.

## Specs

- **S0 — Regla de proceso y criterios de guía.**
  - Dejar escrita esta ficha como origen del trabajo y sumarla al índice de `odd/README.md`.
  - Fijar en la ficha las decisiones de arriba para que la siguiente tanda no las reabra.
  - Criterio: cualquier guía nueva referencia esta ficha y declara si es recorrido, tutorial o ambos.

- **S1 — Solicitudes de entornos (`/admin/solicitudes-entorno`).** *Entregada.*
  - Recorrido de siete pasos: encabezado, cola en revisión/respondidas, ficha del pedido, piezas reconocidas, piezas faltantes, uso y materias, botón «Responder».
  - Tutorial que exige ficha abierta (`selected() !== null`) y plantilla elegida (`selectedTemplateId() !== ''`); la respuesta y la decisión se explican sin ejecutarse.
  - Criterio cumplido: el tutorial no aprueba ni rechaza nada; aprobar sigue siendo una acción del administrador.
  - Archivos: `environment-requests-help.definitions.ts`, componente y plantilla de la pantalla.

- **S2 — Docentes (`/admin/docentes`).** *Entregada.*
  - Recorrido de cinco pasos: resumen del plantel, alta, tabla con pestañas, filtros y acciones por fila.
  - Tutorial del alta individual: exige modal abierto, correo válido y registro confirmado. El correo se lee del componente del modal con `viewChild` y su `isValidEmail()` computado, para que el avance no se habilite con un correo incompleto.
  - Reenviar, renovar, revocar y la baja solo se explican. La importación de planilla queda en el manual.
  - Criterio cumplido: tras registrar, la tabla se recarga y el tutorial cierra con la explicación del ciclo de 72 horas.
  - Archivos: `teachers-help.definitions.ts`, componente de la pantalla, tabla y modal de invitación (`data-tour`).

- **S3 — Emergencias (pestaña de `/admin/auditoria`).** *Pendiente, prioridad 1.*
  - Recorrido por el panel, la lista de operaciones, la frase de confirmación, el motivo y el historial.
  - Tutorial que exige la frase exacta escrita y un motivo de al menos 10 caracteres; **no ejecuta** ninguna operación. Termina explicando cuándo cada una es la salida correcta.
  - Criterio: con la frase incompleta el avance sigue bloqueado; ninguna operación queda registrada por el tutorial.
  - Motivo: es la pantalla más peligrosa del panel y la única donde el caso simple no existe.

- **S4 — Auditoría (`/admin/auditoria`).** *Entregada.*
  - Recorrido por pestañas, búsqueda por evento/actor/recurso, filtro por verbo de acción, tabla paginada, cronología del actor y paginación.
  - Tutorial que exige aplicar un filtro real y abrir la cronología de un actor; explica qué mirar al investigar un incidente.
  - Criterio cumplido: el registro es de solo lectura; el tutorial no edita ni borra nada y las comprobaciones leen el estado real (`hasActiveFilters`, `drawerOpen`).
  - La pantalla no tiene filtro por fecha ni exportación: la ficha se corrigió para describir lo que existe.

- **S5 — Estudiantes (`/admin/estudiantes`).** *Pendiente, prioridad 1.*
  - Recorrido por resumen, pestañas de estado, filtro de período, tabla y acciones.
  - Tutorial del desbloqueo por memoria: exige estudiante seleccionado, motivo escrito y confirmación abierta; el desbloqueo real no se ejecuta.
  - Debe explicar el modelo de faltas: 1–2 en riesgo, 3 bloqueo, y que el desbloqueo reinicia el contador y queda auditado.
  - Criterio: una celda sin motivo no habilita el avance.

- **S6 — Cursos (`/admin/cursos`).** *Entregada.*
  - Recorrido por resumen, selector de período, pestañas de estado, tabla y acciones por fila.
  - Tutorial del alta: exige formulario abierto y nombre con código; el docente es opcional y la materia puede nacer en «Sin Docente».
  - Reasignar y archivar se explican; archivar queda fuera de la guía por estar atado al período.
  - Criterio cumplido: el alta del tutorial deja una materia visible en la tabla; el archivado no se ejecuta.

- **S7 — Catálogo de tecnologías (`/admin/tecnologias`).** *Entregada.*
  - Recorrido por resumen, pestañas, filtros, tarjetas con uso y acciones de baja; el mapa del sistema se explica desde la pestaña.
  - Tutorial de baja segura: exige abrir el diálogo de una tecnología **en uso** y comprobar que la pantalla ofrece desactivar en lugar de eliminar.
  - Criterio cumplido: la guía no desactiva ni elimina nada; el caso de uso cero queda fuera porque el diálogo ofrece otra acción.

- **S8 — Dashboard de salud (`/admin/dashboard`).** *Entregada. Solo recorrido.*
  - Pasos: veredicto de salud y sus motivos, tarjetas de recursos, gráfico de carga, tabla de contenedores y alcance de «Hibernar todo».
  - No lleva tutorial: no hay un flujo por pasos que comprobar; la operación es un clic con confirmación.
  - Criterio: el recorrido no abre logs ni detiene contenedores.

- **S9 — Categorías académicas (`/admin/categorias`).** *Entregada. Solo recorrido.*
  - Pasos: lista, alta, orden por flechas o arrastre y la restricción de borrado con modelos asociados.
  - No lleva tutorial: el flujo es corto y la restricción se explica en la entrada contextual.

- **S10 — Catálogo de plantillas (`/admin/plantillas`).** *Entregada. Solo recorrido.*
  - Recorrido de cinco pasos: resumen, acciones de encabezado, pestañas, filtros y fichas. El tutorial de revisión ya existía.
  - Criterio: queda a la par del Constructor, que sí tiene recorrido y tutorial.

- **S11 — Verificación y documentación indivisible.**
  - `cd frontend && npx ng test --watch=false`, `npx tsc -p tsconfig.app.json --noEmit`, `npx eslint` sobre los archivos tocados y `npx ng build --configuration development`.
  - Actualizar la tabla de la sección 11 del manual por cada guía entregada.
  - Crear las fichas de UI que faltan para las pantallas sin documento en `docs/UI/ADMIN/`: cursos, estudiantes, categorías, tecnologías y solicitudes. Una guía no debe describir un flujo que no tiene especificación escrita.
  - Criterio: el conjunto de tests de frontend pasa, el build compila y la tabla del manual coincide con lo registrado en código.

## Tasks

- [x] **T1 (S0)**: ficha creada, decisiones fijadas y fila agregada al índice de `odd/README.md`.
- [x] **T2 (S1)**: recorrido y tutorial de Solicitudes de entornos, con `data-tour` en ficha, piezas, uso y panel de revisión.
  - Estado: **implementada en el árbol de trabajo (2026-11-14), sin commit**. El tutorial valida ficha abierta y plantilla; no aprueba ni rechaza.
- [x] **T3 (S2)**: recorrido y tutorial de Docentes, con `data-tour` en resumen, alta, tabla, filtros, acciones por fila y modal.
  - Estado: **implementada en el árbol de trabajo (2026-11-14), sin commit**. El correo se valida leyendo el componente del modal con `viewChild`; el registro se detecta con una señal que se activa al confirmar.
- [x] **T4 (S1, S2, S11)**: test de las definiciones en `admin-help-guides.spec.ts` y actualización de la tabla del manual.
  - Estado: **cerrado**. La suite de frontend completa pasó en verde al cerrar S1 y S2 (145 archivos, 1111 tests). Tras sumar Auditoría, Emergencias, Cursos y Tecnologías, el test de definiciones tiene 7 casos en verde y los tests del centro de ayuda suman 26, con `tsc`, ESLint y build de desarrollo sin errores.
- [x] **T5 (S3)**: recorrido y tutorial de Emergencias, con `data-tour` en encabezado, franja del host, catálogo, historial y formulario de confirmación.
  - Estado: **implementada en el árbol de trabajo (2026-11-14), sin commit**. El tutorial valida frase exacta y motivo de 10 caracteres; no ejecuta ninguna operación.
- [x] **T6 (S4)**: recorrido y tutorial del Registro de Auditoría, con `data-tour` en pestañas, filtros, tabla, cronología, paginación y panel del actor.
  - Estado: **implementada en el árbol de trabajo (2026-11-14), sin commit**. El tutorial exige un filtro aplicado y una cronología abierta; el registro sigue siendo de solo lectura.
  - Se agregó la entrada contextual `audit-log` y se corrigió la entrada `auditoria-overview`, que prometía un filtro por severidad inexistente.
- [ ] **T7 (S5)**: guías de Estudiantes.
- [x] **T8 (S6)**: recorrido y tutorial de Cursos, con `data-tour` en resumen, período, pestañas, tabla, acciones y modal de alta.
  - Estado: **implementada en el árbol de trabajo (2026-11-14), sin commit**. El tutorial valida nombre y código leyendo el modal con `viewChild` y termina antes de confirmar: no crea la materia. La señal `createdInSession` se retiró.
- [x] **T9 (S7)**: recorrido y tutorial del Catálogo de tecnologías, con `data-tour` en resumen, pestañas, filtros, grilla, acciones y diálogo de baja.
  - Estado: **implementada en el árbol de trabajo (2026-11-14), sin commit**. El tutorial exige una búsqueda escrita y explica uso, origen del catálogo y desactivación; no abre diálogos ni depende del icono de eliminar, que el catálogo oficial no muestra.
- [x] **T10 (S8)**: recorrido del Dashboard de salud, de seis pasos, sin tutorial.
  - Estado: **implementada en el árbol de trabajo (2026-11-14), sin commit**. Señala veredicto, acciones, recursos, gráfico, carga por materia y contenedores; no abre logs ni detiene nada.
- [x] **T11 (S9)**: recorrido de Categorías académicas, de cuatro pasos, sin tutorial.
  - Estado: **implementada en el árbol de trabajo (2026-11-14), sin commit**. Explica el alta, el orden y la restricción de borrado con modelos asociados.
- [x] **T12 (S10)**: recorrido del Catálogo de plantillas, de cinco pasos, sumado al tutorial que ya existía.
  - Estado: **implementada en el árbol de trabajo (2026-11-14), sin commit**. Queda a la par del Constructor: la pantalla ofrece recorrido y tutorial. El paso de fichas apunta a la lista de la pestaña visible, no solo a la bandeja de aprobación. El tutorial cambia de guion cuando no hay pendientes.
- [x] **T13 (S11)**: fichas de UI faltantes y verificación final de la tanda.
  - Estado: **cerrado**. Creadas `CURSOS.md`, `ESTUDIANTES.md`, `CATEGORIAS.md`, `TECNOLOGIAS.md` y `SOLICITUDES.md` en `docs/UI/ADMIN`, con propósito, composición, reglas, contratos y guías de cada pantalla. Verificado tras T10–T12: 8 tests de definiciones y 27 del centro de ayuda en verde, `tsc`, ESLint, build de desarrollo y `terminology-gate` sin errores.

## Matriz de verificación

| Pantalla | Entrada contextual | Recorrido | Tutorial | Estado |
|---|---|:---:|:---:|---|
| Dashboard de salud | Sí | Sí | No aplica | Implementada, sin commit |
| Docentes | Sí | Sí | Sí | Implementada, sin commit |
| Cursos | Sí | Sí | Sí | Implementada, sin commit |
| Estudiantes | Sí | Pendiente | Pendiente | Planificada |
| Catálogo de entornos | Sí | Sí | Sí (previo) | Implementada, sin commit |
| Constructor de plantillas | Sí | Sí (previo) | Sí (previo) | Cerrada |
| Solicitudes de entornos | Sí | Sí | Sí | Implementada, sin commit |
| Catálogo de tecnologías | Sí | Sí | Sí | Implementada, sin commit |
| Categorías académicas | Sí | Sí | No aplica | Implementada, sin commit |
| Auditoría | Sí | Sí | Sí | Implementada, sin commit |
| Emergencias | Sí | Sí | Sí | Implementada, sin commit |
| Configuración (5 pestañas) | Sí | Sí (previo) | Sí (previo) | Cerrada |
| Notificaciones | Sí | No aplica | No aplica | Cerrada |
| Manual del administrador | No aplica | No aplica | No aplica | Cerrada |

## Log

- **L1 (2026-11-14):** auditoría de cobertura del rol administrador. Se revisaron rutas, el registro de `ContextHelpService` en cada componente y las entradas de `help-content.constants.ts`. Hallazgo: siete superficies con guías y once con solo explicación contextual.
- **L2 (2026-11-14):** decisión del usuario de implementar primero Solicitudes de entorno y Docentes, siguiendo el patrón de Configuración. Quedaron implementadas en el árbol de trabajo, sin commit.
- **L3 (2026-11-14):** decisión sobre los casos difíciles: no meterlos en el mismo tutorial lineal. Un tutorial por camino principal; los casos límite como referencia; un segundo tutorial solo si el caso bloquea y no se descubre solo. Candidatos admitidos: solicitud sin plantilla compatible e invitación expirada.
- **L4 (2026-11-14):** se descartó el selector de escenario al inicio del tutorial: exige UI nueva en el centro de ayuda y reinicio de estado, y todavía no hay suficientes casos por pantalla.
- **L5 (2026-11-14):** regla de proceso acordada: documentar la tarea en `odd/` antes de escribir código, con specs y matriz de verificación, y actualizar el manual en el mismo cambio.
- **L6 (2026-11-14):** T5 y T6 implementadas. Emergencias quedó con recorrido de cinco pasos y tutorial de siete que termina antes de confirmar; Auditoría estrenó la entrada `audit-log`, recorrido de cinco pasos y tutorial de cinco. Se corrigió la entrada contextual vieja que hablaba de filtro por severidad: la pantalla filtra por verbo de acción y busca por evento, actor o recurso. Tests de las definiciones: 5 en verde.
- **L7 (2026-11-14):** T8 y T9 implementadas. Cursos estrenó recorrido de cinco pasos y tutorial de siete; el docente quedó opcional en el alta, corregido en la spec. Tecnologías estrenó recorrido de cinco pasos y tutorial de cuatro centrado en la baja de una tecnología en uso. Tests de las definiciones: 7 en verde.
- **L8 (2026-11-14):** T10 a T12 implementadas. Dashboard (6 pasos), Categorías (4 pasos) y Catálogo de entornos (5 pasos) quedaron solo con recorrido; el catálogo suma el recorrido al tutorial de revisión que ya tenía. Tests de las definiciones: 8 en verde.
- **L21 (2026-11-14):** los comandos declarados estaban siempre visibles en tarjetas y tabla del catálogo, donde el admin escanea estados y no audita. Pasaron a `Detalles técnicos` en las tarjetas del catálogo activo y se quitaron de la tabla y de las tarjetas de pendientes/rechazadas. Docs: `PLANTILLAS_AS_BUILT.md` §9.3.
- **L20 (2026-11-14):** el detalle del docente mostraba la dinámica de evaluación para cualquier modalidad distinta de Entorno de Desarrollo, incluidas las plantillas duales (Entorno de Desarrollo y Evaluación). Ahora cada modalidad describe lo suyo, la dual menciona ambas y una modalidad desconocida no muestra el bloque. Docs: `docs/UI/DOCENTE/CATALOGO_ENTORNOS.md` §5.
- **L19 (2026-11-14):** el detalle del docente titulaba el bloque «Herramientas y Tecnologías Incluidas». Pasó a «Tecnologías incluidas» para que «Herramientas» quede reservado a los comandos declarados, como en el catálogo del admin y la tarjeta del docente.
- **L18 (2026-11-14):** la tarjeta y el detalle del docente rellenaban descripción, categoría, memoria e insignia oficial con valores inferidos cuando faltaba el dato. Ahora omiten descripción y categoría, tratan «General» como ausencia de categoría, no muestran memoria sin dato y la insignia oficial exige `is_official === true`. Docs: `docs/UI/DOCENTE/CATALOGO_ENTORNOS.md` §5.
- **L17 (2026-11-14):** el detalle del entorno del docente mostraba «1 Core(s)» cuando la API no devuelve CPU, y marcaba cada tecnología como «Instalado». La fila de CPU se oculta sin dato y la etiqueta pasó a «Incluido». Docs: `docs/UI/DOCENTE/CATALOGO_ENTORNOS.md` §5.
- **L16 (2026-11-14):** la tarjeta del docente llamaba «Herramientas incluidas» a las tecnologías, mientras el catálogo del admin reserva «Herramientas» para los comandos declarados. La etiqueta pasó a «Tecnologías incluidas» en el catálogo y en las tarjetas de ejercicios del docente, sin cambios de datos ni de chips. Docs: `docs/UI/DOCENTE/CATALOGO_ENTORNOS.md` §4.
- **L15 (2026-11-14):** la tarjeta del catálogo de plantillas mostraba el stack con iconos, pero no las herramientas declaradas (`tools_declared`), que sí aparecen en la ficha del catálogo de tecnologías. Se añadió la fila «Herramientas» con chips monoespaciados en las tarjetas de catálogo, pendientes y rechazadas, y el stack completo con iconos de 14 px bajo el nombre, más la fila «Herramientas», en la vista de tabla de las tres pestañas. Solo lectura: no cambia el alta ni la auditoría. Docs: `PLANTILLAS_AS_BUILT.md` §9.3. Verificado con una plantilla temporal de 5 tecnologías y 8 herramientas: el stack ocupó 3 líneas y las herramientas otras 3, sin desbordar la celda; la fila se borró al terminar. Se retiró el icono grande de la cabecera de la celda, que duplicaba la primera tecnología y forzaba más alto de fila; el stack lo reemplaza. En la tarjeta, las herramientas pasaron a monoespaciado secundario sin borde ni fondo para que el stack conserve la jerarquía de identidad.
- **L14 (2026-11-14):** el tutorial de plantillas pedía «Revisar y Aprobar» y «Rechazar» aunque la bandeja de aprobación no existiera. Ahora `buildCatalogTutorial` elige un guion de tres pasos cuando `pendingCount()` es cero: explica por qué no está la pestaña y qué se hace en las fichas del catálogo activo (activar o pausar, editar cuota, duplicar). El guion de revisión queda para cuando hay pendientes, conservando la salida si la bandeja se vacía a mitad de la guía. Manual §11, hint del centro de ayuda y un test nuevo cubren el cambio. Verificado en navegador con una plantilla pendiente temporal (borrada al terminar): la pestaña apareció con contador, el tutorial abrió 5 pasos, el paso de revisión resaltó las tarjetas y el último exigió «Catálogo Activo» antes de «Finalizar». El recorrido también se comprobó en los dos estados: con 1 pendiente y con la bandeja vacía mantiene sus 5 pasos, todos con elemento visible, y el último resalta la grilla de fichas en ambos casos. En vista tabla la comprobación se repitió en los dos estados: el quinto paso resalta el contenedor de la tabla, también visible.
- **L13 (2026-11-14):** el recorrido del Catálogo de plantillas perdía su quinto paso sin aprobaciones pendientes: `templates-list` solo existía en esa bandeja. El atributo se replicó en las listas del catálogo activo y de rechazadas, en tarjetas y tabla, y el texto del paso pasó a describir la ficha y las acciones de cada pestaña. Verificado en la pestaña «Catálogo Activo» con 11 plantillas.
- **L12 (2026-11-14):** el tutorial de Cursos ya no exige crear una materia real. El paso de confirmación pasó a ser explicativo y el último sigue cerrando en las acciones de la fila; se quitó `hasCreated` del contexto y la señal `createdInSession`. La guía queda utilizable en capacitación sin dejar datos de prueba, y el formulario se cierra con «Cancelar».
- **L11 (2026-11-14):** popovers del recorrido. Driver.js colocaba la burbuja encima del objetivo en pantallas angostas; ahora el servicio busca la banda libre superior o inferior tras el desplazamiento suave, limita la altura con desplazamiento interno cuando la banda es menor que la burbuja y vuelve a ajustar al desplazar o redimensionar. El ancho se acota al viewport. Verificado a 390×844 en el recorrido de Salud: pasos 3 y 4 quedan fuera del bloque señalado; el paso 5 resalta un contenedor más alto que la ventana y mantiene la posición del motor.
- **L10 (2026-11-14):** revisión en navegador de las guías del administrador. El tutorial de Tecnologías quedaba bloqueado: exigía abrir la baja de una tecnología en uso, pero el catálogo oficial no renderiza el botón de eliminar y el paso no permitía avanzar. Se rehizo sin ese diálogo: exige una búsqueda, explica «En uso: N», el origen oficial y el interruptor. También se detectó que el recorrido de Plantillas pierde su paso de fichas sin aprobaciones pendientes y que el tutorial de Cursos crea una materia real; ambas quedan anotadas, sin cambios en este trabajo.
- **L9 (2026-11-14):** T13 cerrada. Cinco fichas nuevas en `docs/UI/ADMIN` describen el comportamiento real: la de Estudiantes declara que su guía sigue planificada (T7) y la de Cursos corrige el alta con docente opcional. `terminology-gate` en verde.
- **Hallazgo verificado:** la tabla de la sección 11 del manual decía la verdad antes de esta ficha; ahora incluye Solicitudes de entornos y Docentes. La copia del backend tiene terminología anterior a la política de modalidades; se actualizó solo la tabla y no se tocó el resto del texto.
- **Limitación declarada:** el plan de impacto salió L3 por 287 archivos cambiados que ya existían en el árbol antes de esta tarea; no se ejecutó completo. Las dos pantallas no se verificaron visualmente en el navegador: falta comprobar el resaltado y la posición de los popovers con datos reales.
