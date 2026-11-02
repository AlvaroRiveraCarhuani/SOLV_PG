# SOLV Companion

Extensión de VS Code que acompaña al estudiante dentro de OpenVSCode Server. Muestra los comentarios que el docente dejó sobre su entrega y registra la cadencia de escritura para la revisión forense.

El código vive en `tools/solv-companion/`. No forma parte del build de Angular ni del binario de Go: se compila y se empaqueta por separado.

## Qué resuelve

El SpeedGrader deja comentarios anclados a `file_path` y `local_line_number`. Hasta ahora el estudiante tenía que abrir la entrega en la plataforma para verlos. La extensión los lleva al editor, en el margen y en el hover de la línea comentada, sin salir del proyecto.

La telemetría de teclado ya existía en el backend. La extensión la alimenta desde el editor con inserciones, borrados y pegados masivos, para que el docente vea la cadencia real de la entrega.

## Módulos

| Ruta | Responsabilidad |
| --- | --- |
| `src/extension.ts` | Composition root: crea configuración, cliente, decorador, controlador y capturador; registra comandos |
| `src/config/solv-settings.ts` | URL, entrega, intervalos y token en `SecretStorage` |
| `src/client/solv-client.ts` | Cliente HTTP con `Bearer`, timeouts y mensajes de error |
| `src/comments/comment-decorator.ts` | Decoración de líneas, margen, hover y limpieza |
| `src/comments/comments-controller.ts` | Refresco por guardado, intervalo y cambio de configuración |
| `src/telemetry/keystroke-classifier.ts` | Clasificación pura de inserciones, borrados y pegados |
| `src/telemetry/keystroke-capturer.ts` | Captura, búfer acotado y envío por lotes |

## Configuración

1. `SOLV: Configurar conexión`.
2. URL base de la API, sin barra final.
3. Identificador de la entrega.
4. Token de sesión.

La URL y la entrega quedan en la configuración del workspace. El token se guarda en `SecretStorage` y nunca se escribe en `settings.json`, en la salida ni en el repositorio.

| Ajuste | Valor por defecto | Uso |
| --- | --- | --- |
| `solv.apiBaseUrl` | vacío | URL base de la API |
| `solv.submissionId` | vacío | Entrega que se está revisando |
| `solv.comments.refreshSeconds` | 60 | Intervalo de consulta de comentarios |
| `solv.telemetry.enabled` | `true` | Activa el capturador |
| `solv.telemetry.pasteThreshold` | 50 | Caracteres insertados de una vez que cuentan como pegado |

## Comentarios

- Solo se anclan los comentarios con `file_path` y `local_line_number`.
- El bloque se extiende con la diferencia entre `end_line_number` y `line_number`; el rango se recorta al tamaño del documento.
- Los comentarios de rúbrica o de la modalidad juez no tienen ancla por archivo y no se pintan en una línea.
- El hover muestra el autor y el texto; no interpreta Markdown del comentario.
- Los colores `solv.comment.background` y `solv.comment.border` se adaptan a tema claro, oscuro y alto contraste.

## Telemetría

- Solo se capturan documentos con esquema `file` dentro de una carpeta del workspace.
- Cada cambio produce un evento `insert`, `delete` o `paste`. Un bloque igual o mayor al umbral, o un reemplazo multilínea de al menos 10 caracteres, se marca como pegado.
- El búfer se limita a 5000 eventos y se envía en lotes de 200 cada 5 segundos y al guardar.
- El contenido del evento se recorta a 4000 caracteres. El backend sigue aplicando su propio tope de 10 000 eventos por entrega.
- Si la API no responde, se conserva el búfer y se avisa como máximo una vez por minuto.

## Contrato usado

| Método | Ruta | Uso |
| --- | --- | --- |
| `GET` | `/api/v1/student/submissions/{id}/comments` | Comentarios de una entrega propia |
| `POST` | `/api/v1/submissions/{id}/keystroke-events` | Lote de eventos de escritura |

La lectura valida que la entrega pertenezca al estudiante de la sesión; una entrega ajena responde `404`. Ambas rutas aceptan `Authorization: Bearer <token>` o la cookie de sesión.

## Compilar y empaquetar

```bash
cd tools/solv-companion
npm install
npm run compile
npm test
npm run package
```

`npm run package` compila y genera `solv-companion-0.1.0.vsix` con `vsce`. El paquete incluye `out/`, `package.json`, el README y la licencia; el fuente, `node_modules` y los tests quedan fuera por `.vscodeignore`.

Para probar en el editor: abrir `tools/solv-companion/` en VS Code, presionar `F5` y usar la ventana de extensión.

## Límites conocidos

- La extensión no descubre la entrega sola: el estudiante o el entorno deben configurar `solv.submissionId`.
- El token se renueva a mano cuando expira; no hay flujo OAuth dentro del editor.
- El despliegue del `.vsix` en la imagen del entorno no está automatizado en esta entrega.
