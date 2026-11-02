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
- **Panel lateral:** cabecera de materia y docente, estado de la entrega, consigna con criterios, avances, comentarios y aula del docente.
- **Checklist de avances:** `Nuevo` (verde, marcado por defecto), `Modificado` (ámbar, marcado, con `Ver diferencias`) y `Sin cambios` (neutro, desmarcado).
- **Pestaña completa:** `SOLV: Abrir consigna en pestaña` crea un `WebviewPanel` en la columna contigua, para leer consignas largas o trabajar en pantalla dividida.
- **Barra de estado:** `$(mortar-board) SOLV: SIS-420` y `$(cloud-upload) Entregar Práctica` para el alumno; `$(mortar-board) SOLV: SIS-420 · Docente`, `$(broadcast) Emitir Avance` y `$(organization) N alumnos` para el docente.

El HTML y el CSS son nativos y usan solo variables `--vscode-*`: se adaptan a tema claro, oscuro y alto contraste sin librerías externas. Los iconos de la webview son SVG en línea con `currentColor`; la barra de estado usa codicons nativos de VS Code. No hay emojis en la interfaz.

## Diff en memoria

La extensión registra un `TextDocumentContentProvider` con el esquema virtual `solv-advance:`. Para comparar un archivo:

```typescript
const localUri = vscode.Uri.joinPath(workspaceUri, relativePath);
const virtualUri = vscode.Uri.from({
  scheme: 'solv-advance',
  path: `/advances/${advanceId}/${relativePath}`
});
await vscode.commands.executeCommand('vscode.diff', localUri, virtualUri, `${relativePath} — Tu versión y el avance del docente`);
```

El contenido llega desde `GET /api/v1/class-advances/{id}/files`, en el mapa `files`. No se escriben archivos temporales en el volumen del alumno. Antes de abrir el diff o aplicar se ejecuta `workspace.saveAll(false)`.

## Contexto de la materia y estado de la entrega

El panel toma materia, práctica y estado de la entrega de `GET /api/v1/student/submissions`. Una entrega con nota se muestra como `Calificada N/100`; una entrega registrada sin nota, como `Entregada`; sin entregas, como `Borrador`. Si la consulta falla, el panel conserva lo que venga por entorno o configuración.

Al arrancar, la extensión escribe `files.exclude` en `.vscode/settings.json` para `**/.git`, `**/.cache`, `**/.openvscode-server` y `**/.solv_backups`. Solo agrega claves ausentes: una preferencia explícita del workspace se respeta.

## Aplicación selectiva

`POST /api/v1/class-advances/{id}/apply` acepta `selected_files: []string`:

```json
{ "selected_files": ["src/models/product.ts", "package.json"] }
```

Con lista, el backend respalda y sobrescribe solo esos archivos. Sin lista, aplica el avance completo, que es el contrato anterior. Una ruta que no pertenece al snapshot invalida la selección completa con `422`: la extensión nunca deja el proyecto a medio aplicar en silencio.

## Deshacer avance

`POST /api/v1/class-advances/{id}/rollback` restaura los archivos desde `.solv_backups/<avance_id>/`, deja el destinatario en `pending` y registra `CLASS_ADVANCE_ROLLBACK` en la auditoría. Exige que el avance esté aplicado (`409` si no) y que exista respaldo (`409` si el avance solo agregó archivos nuevos). El respaldo se borra después de reabrir el aviso; si la reapertura falla, se conserva y el alumno puede reintentar.

## Entrega con pre-flight

`SOLV: Entregar práctica` evalúa `vscode.languages.getDiagnostics()`. Si hay errores de severidad `Error`, muestra una advertencia con dos salidas: `Revisar errores` (abre el archivo y posiciona el cursor en el primer error) o `Entregar igualmente`. Después pide confirmación, guarda todas las pestañas y entrega con `POST /api/v1/workspaces/{id}/submit`; la práctica se resuelve por `solv.workspaceId` o consultando el entorno de la materia.

## Emisión del docente

`SOLV: Emitir avance a la clase` pide el título con `showInputBox`, el alcance con QuickPick (`Toda la clase` o `Seleccionados`) y, con selección, los alumnos con `canPickMany`. Después llama a `POST /api/v1/teacher/sessions/{id}/advances` y confirma con el número de destinatarios.

## Configuración

| Ajuste | Valor por defecto | Uso |
| --- | --- | --- |
| `solv.apiBaseUrl` | vacío | URL base de la API |
| `solv.submissionId` | vacío | Entrega en revisión |
| `solv.role` | `student` | Rol del editor; `SOLV_USER_ROLE` tiene prioridad |
| `solv.sessionId` | vacío | Aula abierta |
| `solv.exerciseId` | vacío | Práctica en curso |
| `solv.subjectId` / `solv.workspaceId` | vacío | Materia y entorno para entregar |
| `solv.subjectCode` / `solv.subjectTitle` / `solv.teacherName` | vacío | Cabecera del panel |
| `solv.assignmentTitle` / `solv.assignmentSummary` / `solv.assignmentCriteria` | vacío | Consigna |
| `solv.speedGraderUrl` | vacío | Evaluación del docente |
| `solv.comments.refreshSeconds` | 60 | Intervalo de comentarios |
| `solv.advances.refreshSeconds` | 30 | Intervalo de avances |
| `solv.telemetry.enabled` | `true` | Capturador de escritura |
| `solv.telemetry.pasteThreshold` | 50 | Caracteres que cuentan como pegado |

El token se guarda en `SecretStorage` y nunca se escribe en `settings.json`, en la salida ni en el repositorio.

## Comentarios

- Solo se anclan los comentarios con `file_path` y `local_line_number`.
- El bloque se extiende con la diferencia entre `end_line_number` y `line_number`; el rango se recorta al tamaño del documento.
- El hover muestra autor y texto una sola vez: la decoración pinta el margen y el `HoverProvider` aporta el contenido.
- Los comentarios de rúbrica o de la modalidad juez no tienen ancla por archivo y no se pintan en una línea.
- Los colores `solv.comment.background` y `solv.comment.border` se adaptan a tema claro, oscuro y alto contraste.

## Telemetría

- Solo se capturan documentos con esquema `file` dentro de una carpeta del workspace.
- Cada cambio produce un evento `insert`, `delete` o `paste`. Un bloque igual o mayor al umbral, o un reemplazo multilínea de al menos 10 caracteres, se marca como pegado.
- El búfer se limita a 5000 eventos y se envía en lotes de 200 cada 5 segundos y al guardar.
- El contenido del evento se recorta a 4000 caracteres. El backend sigue aplicando su propio tope de 10 000 eventos por entrega.
- Si la API no responde, se conserva el búfer y se avisa como máximo una vez por minuto.

## Conexión

Los avances y el aula se consultan cada 30 segundos. Un microcorte se reintenta en silencio; la barra de estado muestra `Conectando…` y pasa a sin conexión solo si el fallo persiste más de dos minutos. No hay popups flotantes.

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
- En producción, el despliegue final hornea la carpeta compilada dentro de la imagen base de OpenVSCode Server en `/home/.openvscode-server/extensions/solv.solv-companion-0.1.0`.

## Montaje en Entornos de Laboratorio

El backend de Go inyecta la extensión y su configuración sin intervención del estudiante:

1. **Variables de entorno inyectadas al contenedor**: `SOLV_API_URL`, `SOLV_STUDENT_TOKEN`, `SOLV_SUBMISSION_ID` y el resto de las claves de la tabla de configuración cuando la plataforma las conoce.
2. **Carga en OpenVSCode Server**: en desarrollo, `docker_container.go` detecta `tools/solv-companion` (o `SOLV_COMPANION_HOST_PATH`) y monta la carpeta como bind mount de solo lectura en `/home/.openvscode-server/extensions/solv.solv-companion-0.1.0:ro`. El flag `--extensions-dir` asegura que el editor cargue la extensión al iniciar.
3. **Red**: `ExtraHosts: ["host.docker.internal:host-gateway"]` permite que el contenedor hable con el backend local.
