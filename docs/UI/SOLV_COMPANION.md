# SOLV Companion

Extensión de VS Code que acompaña al estudiante y al docente dentro de OpenVSCode Server. Lleva al editor la consigna, los avances del aula con diff nativo, los comentarios del docente sobre la entrega y la entrega del proyecto, y registra la cadencia de escritura para la revisión forense.

El código vive en `tools/solv-companion/`. No forma parte del build de Angular ni del binario de Go: se compila y se empaqueta por separado.

## Qué resuelve

El laboratorio se abre a pantalla completa, sin iframe de Angular. Toda la interacción institucional ocurre dentro del editor. Hasta ahora el alumno tenía que volver a la plataforma para ver comentarios o enterarse de un avance del docente; ahora recibe el aviso en el panel lateral, compara contra su código y decide qué archivos incorporar.

En examen no hay avances ni comentarios en línea: el entorno pasa a solo lectura al cerrar (ADR-005, ADR-059). La extensión no cambia esa regla.

## Roles aislados

Un solo paquete `.vsix` sirve a los dos roles. Al arrancar, la extensión lee `SOLV_USER_ROLE` y fija las context keys nativas:

| Rol | Context key | Qué ve |
| --- | --- | --- |
| `student` | `solv:isStudent` | Consigna, checklist de avances, diff, comentarios y entrega |
| `teacher` / `admin` | `solv:isTeacher` | Cátedra, presencia, emisión de avances y acceso al SpeedGrader |

Los comandos de entrega y de emisión se declaran con `when` sobre esas claves, así que el rol que no corresponde no ve el comando ni en la paleta.

## Interfaz

- **Barra de actividades:** contenedor `SOLV` con el isotipo vectorial y la vista `Aula` (webview).
- **Panel lateral:** cabecera de materia y docente, estado de la entrega, avances del aula, observaciones y la consigna como extracto. Es el modo de acción.
- **Checklist de avances:** `Nuevo` (verde, marcado por defecto, con `Ver contenido`), `Modificado` (ámbar, marcado, con `Ver diferencias`) y `Sin cambios` (neutro, desmarcado). Un archivo nuevo no tiene versión anterior, así que no se ofrece un diff.
- **Pestaña completa:** `SOLV: Abrir práctica en pestaña` crea un `WebviewPanel` en la columna contigua con el documento de la práctica y su índice, para leer consignas largas o trabajar en pantalla dividida. El docente conserva ahí la tarjeta `Aula` con emitir y cerrar, más pausar el examen cuando la práctica es un examen.
- **Barra de estado:** `$(mortar-board) SOLV: SIS-420` y `$(cloud-upload) Entregar Práctica` para el alumno; `$(mortar-board) SOLV: SIS-420 · Docente`, `$(broadcast) Emitir Avance` y `$(organization) N alumnos` para el docente.

El HTML y el CSS son nativos y usan solo variables `--vscode-*`: se adaptan a tema claro, oscuro y alto contraste sin librerías externas. Los iconos de la webview son SVG en línea con `currentColor`; la barra de estado usa codicons nativos de VS Code. No hay emojis en la interfaz.

## Dos modos: lateral accionable y documento

El mismo HTML se renderiza en dos modos según `renderPanelHtml(..., { isTab })`. No son dos pantallas distintas: es el mismo estado con otra jerarquía.

**Panel lateral (acción).** El orden responde a lo que exige una decisión:

1. Cabecera de materia, práctica y estado de la entrega.
2. Avance del docente con su checklist, o el aula del docente.
3. Observaciones del docente.
4. Consigna como extracto de seis líneas, con `Leer consigna completa en una pestaña` arriba del texto: el enlace es la entrada a la consigna completa y por eso va primero.
5. `Evaluación` al final, solo para el docente, con `Abrir evaluación en la plataforma` (SpeedGrader). El aula se opera arriba; la evaluación es la salida de la clase.

El extracto sale de `excerptText` (`panel-state.ts`): saca las marcas de markdown, colapsa espacios y corta en un espacio para no partir una palabra. La rúbrica no se muestra en el lateral: vive en la pestaña, donde hay ancho para la matriz de niveles.

**Pestaña (documento).** El documento es la consigna completa, la rúbrica y las observaciones. A partir de 900 px de ancho aparece un índice de anclas fijo (`Consigna`, `Rúbrica`, `Observaciones`): cada entrada salta con scroll suave y un `IntersectionObserver` resalta la sección visible. Por debajo de 900 px el índice se oculta y el contenido sigue siendo una sola columna. Las anclas no reemplazan contenido, así que la consigna se lee completa de arriba abajo.

Para el docente el documento abre con el `Aula` (métricas, presencia, `Emitir avance a la clase` y cerrar; en examen, `Pausar examen`) y el índice suma `Aula` como primera entrada. Esa tarjeta no compite con el lateral: son los mismos controles sobre el mismo estado, y sin ella la pestaña dejaba al docente sin poder operar el aula mientras leía la consigna.

La rúbrica arranca plegada (`<details>` sin `open`) y deja el total en el resumen; su apertura se conserva entre refrescos con `host.setState`. Si hay un avance pendiente, la pestaña muestra una franja con `Revisar en el panel lateral` (`focus-panel`): el checklist y sus botones existen en un solo lugar, para no aplicar dos veces lo mismo.

**Aviso de avance nuevo.** Como el checklist vive en el lateral, un avance podía pasar inadvertido con el panel cerrado. Al detectar un avance pendiente distinto del último avisado, la extensión muestra una notificación de VS Code (`showInformationMessage`) con `Revisar archivos`, que enfoca el lateral. El identificador avisado queda en `workspaceState` (`solv.notifiedAdvance`) y se limpia cuando el avance deja de estar pendiente, así que cada avance avisa una sola vez y no en cada ciclo de un minuto. La notificación no bloquea el refresco: se dispara sin `await`.

## Diff en memoria

La extensión registra un `TextDocumentContentProvider` con el esquema virtual `solv-advance:`. Para comparar un archivo:

```typescript
const localUri = vscode.Uri.joinPath(environmentUri, relativePath);
const virtualUri = vscode.Uri.from({
  scheme: 'solv-advance',
  path: `/advances/${advanceId}/${relativePath}`
});
await vscode.commands.executeCommand('vscode.diff', localUri, virtualUri, `${relativePath} — Tu versión y el avance del docente`);
```

El contenido llega desde `GET /api/v1/class-advances/{id}/files`, en el mapa `files`. No se escriben archivos temporales en el volumen del alumno. Antes de abrir el diff o aplicar se ejecuta `environment.saveAll(false)`.

Un archivo que el alumno todavía no tiene no se compara contra un `file:` inexistente: VS Code responde `The editor could not be opened because the file was not found` y la pestaña queda vacía. En ese caso el lado izquierdo es un documento virtual vacío (`solv-advance:` con prefijo `/vacio/`), el avance se lee completo como líneas agregadas y el título dice `— Archivo nuevo del avance`.

## Contexto de la materia y estado de la entrega

El panel toma materia, práctica y estado de la entrega de `GET /api/v1/student/submissions`. Una entrega con nota se muestra como `Calificada N/100`; una entrega registrada sin nota, como `Entregada`; sin entregas, como `Borrador`. Si la consulta falla, el panel conserva lo que venga por entorno o configuración.

Al arrancar, la extensión escribe `files.exclude` en `.vscode/settings.json` para `**/.git`, `**/.cache`, `**/.openvscode-server` y `**/.solv_backups`. Solo agrega claves ausentes: una preferencia explícita del entorno se respeta.

## Aplicación selectiva

`POST /api/v1/class-advances/{id}/apply` acepta `selected_files: []string`:

```json
{ "selected_files": ["src/models/product.ts", "package.json"] }
```

Con lista, el backend respalda y sobrescribe solo esos archivos. Sin lista, aplica el avance completo, que es el contrato anterior. Una ruta que no pertenece al snapshot invalida la selección completa con `422`: la extensión nunca deja el proyecto a medio aplicar en silencio.

La copia arma un contenedor efímero sobre el volumen del alumno y tarda alrededor de 10 s, así que `apply`, `rollback` y la emisión usan `LONG_REQUEST_TIMEOUT_MS` (120 s) en vez del tope corto de las consultas. El botón `Aplicar archivos seleccionados` muestra un progreso en la notificación mientras el backend escribe: antes, el cliente abortaba a los 10 s y la petición se cancelaba a mitad de la escritura.

## Deshacer avance, por archivo

`GET /api/v1/class-advances/{id}/rollback` lista los archivos que el respaldo (`.solv_backups/<avance_id>/`) todavía puede volver a su versión anterior. Alimenta el checklist del avance aplicado y se consulta en cada refresco. Si la consulta falla, el panel mantiene el botón `Deshacer avance`, que restaura todo, en lugar de mostrar un checklist vacío.

`POST /api/v1/class-advances/{id}/rollback` acepta `paths: []string` opcional. Con la lista restaura solo esos archivos, deja el destinatario en `applied` y conserva el respaldo del resto; sin lista restaura todo y deja el destinatario en `pending`. La respuesta informa `restored`, `remaining` y `fully_restored`, y la extensión usa `remaining` para volver a pintar el checklist. Una ruta sin respaldo invalida la selección completa (`422`): restaurar la mitad de lo pedido y responder éxito dejaría el avance a medias. Cada archivo restaurado sale del respaldo; el aviso vuelve a `pending` y el respaldo se borra recién cuando no queda nada por restaurar. Si esa reapertura falla, el respaldo se conserva y el alumno puede reintentar. Cada movimiento queda como `CLASS_ADVANCE_ROLLBACK` en la auditoría, con los archivos restaurados y cuántos quedan.

## Entrega con pre-flight

`SOLV: Entregar práctica` evalúa `vscode.languages.getDiagnostics()`. Si hay errores de severidad `Error`, muestra una advertencia con dos salidas: `Revisar errores` (abre el archivo y posiciona el cursor en el primer error) o `Entregar igualmente`. Después pide confirmación, guarda todas las pestañas y entrega con `POST /api/v1/environments/{id}/submit`; la práctica se resuelve por `solv.environmentId` o consultando el entorno de la materia.

## Aviso de examen pausado (alumno)

El ciclo del alumno consulta `GET /api/v1/student/sessions/active` una vez por vuelta del temporizador, en la misma llamada que el latido. Si el aula trae `frozen_at`, el Companion avisa una sola vez por pausa y el panel muestra la tarjeta **Examen en pausa** con la hora, mientras la barra de estado dice `Examen en pausa`. El entorno está montado en solo lectura: el aviso explica por qué el editor no guarda. Al reanudarse, avisa que la escritura volvió. Cerrar el examen no se anuncia como reanudación: sin aula abierta, el aviso se limpia en silencio.

## Emisión y Gestión del Aula por el Docente

- `SOLV: Iniciar clase`: Despliega un QuickPick nativo con duraciones estándar (60, 90, 120, 180 min) y opción `Personalizado...` (15 a 480 min). Ejecuta `POST /api/v1/teacher/sessions` con la duración elegida y la práctica del entorno (`SOLV_EXERCISE_ID`): un aula sin práctica no admite avances.
- `SOLV: Cerrar clase`: Disponible cuando el aula está abierta; pide confirmación y ejecuta `PATCH /api/v1/teacher/sessions/{id}/close`.
- **Autodescubrimiento del aula**: Si el docente abrió el aula desde la web o se reinició el contenedor, la extensión autodescubre la sesión activa con `GET /api/v1/teacher/sessions/open` y sincroniza métricas y presencia.
- `SOLV: Emitir avance a la clase`: Pide el título con `showInputBox`, el alcance con QuickPick (`Toda la clase` o `Seleccionados`), los alumnos con `canPickMany` cuando el alcance es por alumno y, al final, qué compartir: `Todo el proyecto` o `Elegir archivos…`, que abre un QuickPick múltiple con las rutas del entorno y viaja como `selected_files`. Así se comparte una parte del trabajo y el resto queda como actividad. Llama a `POST /api/v1/teacher/sessions/{id}/advances`.
- `SOLV: Pausar o reanudar el examen`: pausa en vivo un examen abierto con `PATCH /api/v1/teacher/sessions/{id}/freeze` y lo reanuda con `unfreeze`. Los entornos de los alumnos pasan a solo lectura y vuelven a escritura. No aplica a clases.
- **Aula recordada en el editor**: al iniciar la clase desde el panel, el identificador del aula queda en la configuración de la sesión del editor; si el aula se abrió desde la web, el autodescubrimiento hace lo mismo. Emitir un avance y pausar el examen leen ese identificador y, si quedó vacío, caen al aula del tablero. Al cerrar la clase el identificador se borra y el panel vuelve a ofrecer iniciar la clase.
- **Aula sin alumnos**: emitir un avance en un aula sin alumnos matriculados avisa y no emite. Con alcance `Seleccionados` y una lista vacía lo dice en vez de pedir una selección.

## Consigna Renderizada

La consigna se escribe en Markdown desde la plataforma y el panel la renderiza en el editor: títulos, listas, citas, separadores, bloques de código, énfasis y enlaces `http`/`https`. Antes se mostraba el texto crudo, con la sintaxis a la vista.

La conversión vive en `src/views/markdown.ts` y no usa librerías: el paquete de la extensión excluye `node_modules`, así que una dependencia de ejecución no viajaría ni en el `.vsix` ni en la imagen del laboratorio. El texto se escapa antes de interpretar el marcado, de modo que la consigna no puede inyectar etiquetas ni atributos en la webview.

Un bloque de código cuya forma es un árbol de carpetas (raíz única y conectores `├──`, `└──`, `│`) se renderiza como árbol: iconos de carpeta y archivo, sangría con guías y nombres en monoespaciado, sin los caracteres de dibujo. Cualquier otro bloque sigue siendo código con desplazamiento horizontal.

## Rúbrica Oficial Adaptativa en la Consigna

La extensión consulta `GET /api/v1/exercises/{id}/rubric` usando el ejercicio de la entrega o de la configuración institucional. El formato se adapta responsivamente según el contenedor:

1. **Modo compacto (pestaña angosta, < 600 px):**
   - Presenta un totalizador superior (`Total: N pts`).
   - Cada criterio se muestra en una tarjeta tipo acordeón (`<details>`) con su puntaje visible y descripción concisa.
   - Al desplegar la tarjeta, se revelan las pastillas de niveles de desempeño con su puntaje ponderado calculado (`score% · pts`).

2. **Modo matriz (pestaña del editor, ≥ 600 px):**
   - La pestaña se abre con `Leer consigna completa en una pestaña` (`solv.openTab`) en el extracto del lateral, o desde la paleta con `SOLV: Abrir práctica en pestaña`.
   - Los criterios se transforman mediante media queries CSS en una matriz de evaluación analítica con columnas de ponderación y tarjetas de niveles en cuadrícula horizontal.
   - Si el ejercicio no cuenta con rúbrica registrada, se utiliza la lista definida en `solv.assignmentCriteria`.

La apertura de la rúbrica y de cada criterio se guarda en el estado de la webview, así que sobrevive a los refrescos del panel: la persona pliega lo que no necesita y no se le vuelve a abrir en la siguiente actualización.

## Latido de Presencia Automático (Heartbeat)

Para el rol `student`, la extensión consulta `GET /api/v1/student/sessions/active` en cada ciclo de actualización. Al detectar un aula abierta de su materia, emite un latido en segundo plano a `POST /api/v1/student/sessions/{id}/heartbeat`. Esto alimenta la tabla de presencia y actualiza el contador de alumnos presentes en el tablero del docente en tiempo real.

## Configuración

| Ajuste | Valor por defecto | Uso |
| --- | --- | --- |
| `solv.apiBaseUrl` | vacío | URL base de la API |
| `solv.submissionId` | vacío | Entrega en revisión |
| `solv.role` | `student` | Rol del editor; `SOLV_USER_ROLE` tiene prioridad |
| `solv.sessionId` | vacío | Aula abierta |
| `solv.exerciseId` | vacío | Práctica en curso |
| `solv.subjectId` / `solv.environmentId` | vacío | Materia y entorno para entregar |
| `solv.subjectCode` / `solv.subjectTitle` / `solv.teacherName` | vacío | Cabecera del panel |
| `solv.assignmentTitle` / `solv.assignmentSummary` / `solv.assignmentCriteria` | vacío | Consigna |
| `solv.speedGraderUrl` | vacío | Evaluación del docente |
| `solv.comments.refreshSeconds` | 60 | Intervalo de comentarios |
| `solv.advances.refreshSeconds` | 30 | Intervalo de avances |
| `solv.telemetry.enabled` | `true` | Capturador de escritura |
| `solv.telemetry.pasteThreshold` | 50 | Caracteres que cuentan como pegado |

El token se guarda en `SecretStorage` y nunca se escribe en `settings.json`, en la salida ni en el repositorio.

En los entornos de práctica el backend inyecta `SOLV_API_URL` con el dominio del nodo (`https://solv.dedyn.io` en este despliegue). Traefik publica el API bajo `/api` en ese mismo host, así que la URL base no lleva sufijo: no existe un subdominio `api.<dominio>`. La variable de entorno tiene prioridad sobre `solv.apiBaseUrl`; un despliegue que necesite otra URL define `SOLV_API_URL` en el entorno del backend.

## Comentarios

- Solo se anclan los comentarios con `file_path` y `local_line_number`.
- El bloque se extiende con la diferencia entre `end_line_number` y `line_number`; el rango se recorta al tamaño del documento.
- El hover muestra autor y texto una sola vez: la decoración pinta el margen y el `HoverProvider` aporta el contenido.
- Los comentarios de rúbrica o de la modalidad juez no tienen ancla por archivo y no se pintan en una línea.
- Sin entrega configurada no se consulta nada y no se avisa: el IDE de la práctica del docente no tiene entrega y el alumno no tiene ninguna hasta que entrega. El aviso de configuración aparece solo cuando falta la URL de la API.
- Los colores `solv.comment.background` y `solv.comment.border` se adaptan a tema claro, oscuro y alto contraste.

## Telemetría

- Solo se capturan documentos con esquema `file` dentro de una carpeta del entorno.
- Cada cambio produce un evento `insert`, `delete` o `paste`. Un bloque igual o mayor al umbral, o un reemplazo multilínea de al menos 10 caracteres, se marca como pegado.
- El búfer se limita a 5000 eventos y se envía en lotes de 200 cada 5 segundos y al guardar.
- El contenido del evento se recorta a 4000 caracteres. El backend sigue aplicando su propio tope de 10 000 eventos por entrega.
- Si la API no responde, se conserva el búfer y se avisa como máximo una vez por minuto.

## Conexión

Los avances y el aula se consultan cada 30 segundos. Un microcorte se reintenta en silencio; la barra de estado muestra `Conectando…` y pasa a sin conexión solo si el fallo persiste más de dos minutos. No hay popups flotantes.

Un `404` sin cuerpo JSON es el que devuelve el proxy, o una ruta que el backend no conoce: se informa como ruta inexistente de la API, con el `path` consultado y el estado, en lugar de atribuirlo a la entrega. Los mensajes de dominio siguen saliendo del cuerpo que responde el backend.

## Compilar y empaquetar

```bash
cd tools/solv-companion
npm install
npm run compile
npm test
npm run package
```

`npm test` compila y ejecuta los tests de `out/**/*.test.js`: clasificación de escritura y clasificación del checklist de avances. `npm run package` genera `solv-companion-0.1.0.vsix` con `vsce`.

Para probar en el editor: abrir `tools/solv-companion/` en VS Code, presionar `F5` y usar la ventana de extensión con `SOLV_USER_ROLE=student` o `SOLV_USER_ROLE=teacher`.

## Límites conocidos

- El token se renueva cuando la persona vuelve a iniciar su sesión en la plataforma.
- Aplicar y deshacer requieren el laboratorio en ejecución.
- La emisión del docente necesita un aula abierta y un entorno de clase con código; sin ellos la API responde `409`.
- En producción, el despliegue final hornea la carpeta compilada dentro de la imagen base de OpenVSCode Server en `/home/.openvscode-server/extensions/solv.solv-companion-0.1.0`, junto a las extensiones incorporadas de la imagen.

## Montaje en Entornos de Laboratorio

El backend de Go inyecta la extensión y su configuración sin intervención del estudiante:

1. **Variables de entorno inyectadas al contenedor**: `SOLV_API_URL`, `SOLV_STUDENT_TOKEN`, `SOLV_SUBMISSION_ID` y el resto de las claves de la tabla de configuración cuando la plataforma las conoce.
2. **Carga en OpenVSCode Server**: en desarrollo, `docker_container.go` detecta `tools/solv-companion` (o `SOLV_COMPANION_HOST_PATH`) y monta la carpeta como bind mount de solo lectura en `/home/.openvscode-server/extensions/solv.solv-companion-0.1.0:ro`, junto a las extensiones incorporadas que trae la imagen y a las que el Constructor instala al armar la plantilla.
   El montaje no va al directorio del alumno (`--extensions-dir`): al escanear una extensión instalada, el editor reescribe su `package.json`, y sobre un montaje de solo lectura ese intento falla con `EROFS` y la extensión queda marcada como no instalada —el editor la ofrece para instalar en vez de cargarla—.
   Con el backend corriendo en contenedor (`compose.yml`) el proceso no ve el árbol del repositorio, y como el bind lo resuelve el daemon de Docker sobre el host, la ruta debe venir en `SOLV_COMPANION_HOST_PATH`. Sin esa variable el entorno arranca sin la extensión, sin error.
   El montaje lo aplica el backend al crear el contenedor, no la imagen: toda práctica o proyecto nuevo la recibe sin hornearla en cada plantilla.
3. **Red**: `ExtraHosts: ["host.docker.internal:host-gateway"]` permite que el contenedor hable con el backend local.
4. **Modo restringido**: el espacio de trabajo del laboratorio no está marcado como confiable y el editor arranca en modo restringido, que desactiva las extensiones sin declaración explícita. Por eso `package.json` declara `capabilities.untrustedWorkspaces.supported: true`: la extensión lee el proyecto y consulta la API, no ejecuta código del espacio de trabajo. Sin esa declaración desaparecen el contenedor SOLV de la barra de actividad, la barra de estado y el panel.
