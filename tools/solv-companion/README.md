# SOLV Companion

Extensión de VS Code que acompaña al estudiante y al docente dentro de OpenVSCode Server. Muestra la consigna, los avances del aula con diff nativo en memoria, los comentarios del docente sobre la entrega y la entrega de la práctica. Registra además la cadencia de escritura para la revisión forense.

Un solo paquete sirve a los dos roles: las context keys `solv:isStudent` y `solv:isTeacher` se fijan al arrancar según `SOLV_USER_ROLE`.

## Requisitos

- VS Code u OpenVSCode Server 1.96 o superior.
- Funciona en espacios de trabajo no confiables (modo restringido), que es como arrancan los entornos del laboratorio. El manifiesto lo declara en `capabilities.untrustedWorkspaces`; sin esa declaración el editor desactiva la extensión y no muestra su panel.
- Una entrega de SOLV, un aula y un token de sesión válido.
- Acceso a la API de SOLV desde el editor (misma red o dominio público).

## Configuración

1. Ejecutá `SOLV: Configurar conexión`.
2. Indicá la URL base de la API, sin barra final (por ejemplo, el dominio del nodo o `http://localhost:8080` en desarrollo).
3. Indicá el identificador de la entrega.
4. Pegá el token de sesión. El token se guarda en `SecretStorage`, no en `settings.json`.

En el laboratorio la plataforma inyecta la conexión por entorno:

| Variable | Uso |
| --- | --- |
| `SOLV_API_URL` | URL base de la API: el dominio del nodo, sin sufijo ni `/api` (`https://solv.dedyn.io`) |
| `SOLV_STUDENT_TOKEN` | Token de la sesión del editor |
| `SOLV_USER_ROLE` | `student`, `teacher` o `admin` |
| `SOLV_SUBMISSION_ID` | Entrega en revisión |
| `SOLV_SESSION_ID` | Aula abierta |
| `SOLV_EXERCISE_ID` | Práctica en curso |
| `SOLV_SUBJECT_ID` / `SOLV_WORKSPACE_ID` | Materia y entorno, para entregar el proyecto |
| `SOLV_SUBJECT_CODE` / `SOLV_SUBJECT_TITLE` / `SOLV_TEACHER_NAME` | Cabecera de la materia |
| `SOLV_ASSIGNMENT_TITLE` / `SOLV_ASSIGNMENT_SUMMARY` | Consigna del panel |
| `SOLV_EXERCISE_PURPOSE` | Propósito pedagógico (`class` para práctica o `exam` para examen) |
| `SOLV_SPEEDGRADER_URL` | Acceso rápido del docente a la evaluación |

Los mismos datos se pueden fijar en `settings.json` con las claves `solv.role`, `solv.sessionId`, `solv.exerciseId`, `solv.exercisePurpose`, `solv.subjectId`, `solv.workspaceId`, `solv.subjectCode`, `solv.subjectTitle`, `solv.teacherName`, `solv.assignmentTitle`, `solv.assignmentSummary`, `solv.assignmentCriteria` y `solv.speedGraderUrl`.

Intervalos: `solv.comments.refreshSeconds` (60) y `solv.advances.refreshSeconds` (30). Telemetría: `solv.telemetry.enabled` y `solv.telemetry.pasteThreshold` (50 caracteres).

## Comandos

| Comando | Rol | Uso |
| --- | --- | --- |
| `SOLV: Abrir práctica en pestaña` | Ambos | Abre el documento de la práctica con índice de anclas, para pantalla dividida |
| `SOLV: Actualizar panel` | Ambos | Vuelve a leer avances, aula y comentarios |
| `SOLV: Entregar práctica` | Estudiante | Revisa diagnósticos, confirma y entrega el proyecto |
| `SOLV: Compartir avance con la clase` | Docente | Pide título y alcance, y emite el snapshot |
| `SOLV: Pausar o reanudar la clase` | Docente | Pausa o reanuda la emisión de avances |
| `SOLV: Ver presencia del aula` | Docente | Lista los alumnos del aula con su presencia |
| `SOLV: Abrir evaluación en el navegador` | Docente | Abre la evaluación en el navegador |
| `SOLV: Configurar conexión` | Ambos | URL, entrega y token |
| `SOLV: Actualizar comentarios` | Ambos | Fuerza la consulta de comentarios |
| `SOLV: Estado de la telemetría` | Ambos | Eventos pendientes de envío |

## Experiencia del estudiante

- El panel lateral prioriza la acción: avance del docente, observaciones y la consigna como extracto de seis líneas. El extracto abre con `Leer consigna completa en una pestaña`, que muestra el documento completo en una pestaña del editor. La rúbrica no aparece en el lateral.
- `SOLV: Abrir práctica en pestaña` abre el documento completo (consigna y rúbrica) con un índice fijo de `Consigna`, `Rúbrica` y `Observaciones` desde 900 px de ancho; el índice salta con scroll suave y resalta la sección visible. La rúbrica arranca plegada con su total en el resumen y recuerda si la abriste.
- El índice se arma con las secciones que existen: sin rúbrica no hay entrada `Rúbrica`, y el docente suma `Aula` como primera entrada.
- Con un avance pendiente, la pestaña muestra una franja `Revisar en el panel lateral`: el checklist y sus botones viven solo en el lateral.
- Cuando hay un avance, el checklist clasifica cada archivo: `Nuevo` (verde, marcado, con `Ver contenido`), `Modificado` (ámbar, marcado, con `Ver diferencias`) y `Sin cambios` (neutro, desmarcado).
- `Ver contenido` y `Ver diferencias` abren `vscode.diff` entre el archivo local y un documento virtual `solv-advance:`: el contenido del avance se sirve desde memoria, sin archivos temporales. Si el archivo todavía no existe en el proyecto, el lado izquierdo es un documento virtual vacío y el avance se lee completo como agregado (`— Archivo nuevo del avance`), en lugar del error `file was not found`; por eso un archivo nuevo no ofrece diferencias.
- Al llegar un avance nuevo, la extensión muestra una notificación de VS Code con `Revisar archivos`, que lleva al panel lateral: el lateral puede estar cerrado y el alumno no tiene que vigilar la extensión. Se avisa una vez por avance (queda en el estado del workspace).
- Antes de comparar o aplicar se ejecuta `workspace.saveAll(false)`, para que el disco y las pestañas en memoria coincidan.
- `Aplicar archivos seleccionados` aplica solo las rutas marcadas y muestra un progreso mientras el backend escribe: la copia tarda alrededor de 10 s y `apply`, `rollback` y la emisión usan un tope de 120 s en vez del de 10 s de las consultas. `Descartar avance` cierra el aviso sin tocar el proyecto. Con el avance aplicado, el panel lista los archivos que el respaldo todavía puede restaurar y `Deshacer seleccionados` devuelve solo los marcados: el resto sigue aplicado y respaldado, y recién al restaurar el último el aviso queda pendiente otra vez. Los archivos que el avance agregó no se borran; los cambios posteriores del alumno sobre los reemplazados se pierden.
- El inventario de deshacer sale de `GET /api/v1/class-advances/{id}/rollback`. Si esa consulta falla, el panel ofrece `Deshacer avance` completo en lugar de un checklist vacío.
- Los comentarios se agrupan por archivo; al hacer click se abre el archivo con el cursor en la línea comentada.
- La barra de estado adapta su acción según el propósito: `$(cloud-upload) Entregar Examen` para evaluaciones o `$(cloud-upload) Entregar Práctica` para clases formativas, además del estado de la conexión.
- La rúbrica oficial se adapta al propósito pedagógico: muestra el badge y título `Rúbrica de evaluación` (ámbar) para exámenes o `Rúbrica de práctica` (azul) para clases formativas.
- El estado de la entrega sale de `GET /api/v1/student/submissions`: sin entrega queda `Borrador`, con entrega `Entregada` y con nota `Calificada`. Esa misma consulta completa materia, práctica y propósito cuando el entorno no los trae.
- Al arrancar, la extensión fija `files.exclude` del workspace para `**/.git`, `**/.cache`, `**/.openvscode-server` y `**/.solv_backups`, sin pisar claves ya definidas.

## Experiencia del docente

- El panel muestra la cátedra, la presencia (`presentes / matriculados`), las entregas y el estado de emisión.
- El lateral del docente cierra con la consigna y, al final, `Evaluación` con `Abrir evaluación en la plataforma`: el aula se opera arriba y el SpeedGrader queda como salida de la clase.
- La pestaña abre con esa misma tarjeta `Aula` (métricas, presencia, `Emitir avance a la clase`, congelar y cerrar) antes de la consigna: el docente opera el aula mientras lee, sin volver al lateral.
- `Emitir avance` pide el título en un `showInputBox`, el alcance (`Toda la clase` o `Seleccionados`) y, con selección, los alumnos en un QuickPick.
- La barra de estado muestra `$(mortar-board) SOLV: SIS-420 · Docente`, `$(broadcast) Emitir Avance` y `$(organization) N alumnos`.

## Contratos usados

| Método | Ruta | Uso |
| --- | --- | --- |
| `GET` | `/api/v1/class-advances/pending` | Avisos pendientes del alumno |
| `GET` | `/api/v1/class-advances/{id}/files` | `added`, `overwritten` y `files` con el contenido |
| `POST` | `/api/v1/class-advances/{id}/apply` | `selected_files: []string` opcional |
| `GET` | `/api/v1/class-advances/{id}/rollback` | Archivos del respaldo todavía restaurables |
| `POST` | `/api/v1/class-advances/{id}/rollback` | `paths: []string` opcional: restaura la selección o todo el respaldo |
| `POST` | `/api/v1/class-advances/{id}/dismiss` | Descarta el aviso |
| `GET` | `/api/v1/teacher/sessions/{id}/board` | Presencia y entregas del aula |
| `POST` | `/api/v1/teacher/sessions/{id}/advances` | Emisión del docente |
| `PATCH` | `/api/v1/teacher/sessions/{id}/freeze` y `/unfreeze` | Congelación del aula |
| `POST` | `/api/v1/workspaces/{id}/submit` | Entrega del proyecto |
| `GET` | `/api/v1/student/submissions` | Materia, práctica y estado de la entrega del alumno |
| `GET` | `/api/v1/student/submissions/{id}/comments` | Comentarios de la entrega |
| `POST` | `/api/v1/submissions/{id}/keystroke-events` | Lote de eventos de escritura |

## Módulos

| Ruta | Responsabilidad |
| --- | --- |
| `src/extension.ts` | Composition root: roles, paneles, comandos y refresco |
| `src/config/` | Configuración del workspace, entorno y token en `SecretStorage` |
| `src/client/` | Cliente HTTP con `Bearer`, timeouts y entrega del proyecto |
| `src/advances/advance-client.ts` | Endpoints de avances, aula y emisión |
| `src/advances/advance-content-provider.ts` | Documentos virtuales `solv-advance:` para el diff |
| `src/advances/advance-controller.ts` | Estado del avance, selección, diff, aplicación y rollback |
| `src/views/panel-state.ts` | Modelo del panel y clasificación del checklist |
| `src/views/panel-html.ts` | HTML y CSS nativo sobre variables `--vscode-*` |
| `src/views/markdown.ts` | Renderizado de la consigna Markdown, con escapado previo |
| `src/views/solv-panel-view.ts` | Webview de la barra lateral |
| `src/views/assignment-tab.ts` | Pestaña completa de la consigna |
| `src/status/status-bar-controller.ts` | Barra de estado por rol |
| `src/comments/` | Decoración de líneas, hover y refresco |
| `src/telemetry/` | Clasificación de cambios y envío por lotes |

## Desarrollo

```bash
cd tools/solv-companion
npm install
npm run compile
npm test
```

Los tests cubren la clasificación de escritura y la clasificación del checklist de avances.

Para probar en el editor: abrí esta carpeta en VS Code, presioná `F5` y simulá el rol con `SOLV_USER_ROLE=teacher` o `SOLV_USER_ROLE=student` en el entorno de depuración.

## Empaquetado

```bash
npm run package
```

El script compila y genera `solv-companion-0.1.0.vsix` con `vsce`. El paquete incluye `out/`, `package.json`, el README, el CHANGELOG y los recursos; el código fuente, los tests y `node_modules` quedan fuera por `.vscodeignore`.
