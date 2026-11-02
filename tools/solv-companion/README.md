# SOLV Companion

Extensión de VS Code para el estudiante: muestra los comentarios que el docente dejó sobre su entrega y registra la cadencia de escritura para la revisión forense.

## Requisitos

- VS Code u OpenVSCode Server 1.96 o superior.
- Una entrega de SOLV y un token de sesión válido.
- Acceso a la API de SOLV desde el editor (misma red o dominio público).

## Configuración

1. Ejecutá `SOLV: Configurar conexión`.
2. Indicá la URL base de la API, sin barra final (por ejemplo, el dominio del nodo o `http://localhost:8080` en desarrollo).
3. Indicá el identificador de la entrega.
4. Pegá el token de sesión. El token se guarda en `SecretStorage`, no en `settings.json`.

La URL y la entrega quedan en la configuración del workspace. Se pueden ajustar también en Settings:

- `solv.apiBaseUrl`
- `solv.submissionId`
- `solv.comments.refreshSeconds` (60 por defecto)
- `solv.telemetry.enabled` (activo por defecto)
- `solv.telemetry.pasteThreshold` (50 caracteres por defecto)

## Uso

- Los comentarios con `file_path` y `local_line_number` se pintan en el margen de la línea y aparecen en el hover.
- `SOLV: Actualizar comentarios` fuerza una consulta; la extensión también refresca al guardar y cada intervalo configurado.
- `SOLV: Estado de la telemetría` muestra si el capturador está activo y cuántos eventos esperan envío.

Los comentarios de rúbrica o de la modalidad juez no tienen ancla por archivo y no se pintan en una línea.

## Módulos

| Ruta | Responsabilidad |
| --- | --- |
| `src/extension.ts` | Composition root: crea las piezas y registra comandos |
| `src/config/` | Configuración del workspace y token en `SecretStorage` |
| `src/client/` | Cliente HTTP con `Bearer` para comentarios y eventos |
| `src/comments/` | Decoración de líneas, hover y refresco |
| `src/telemetry/` | Clasificación de cambios y envío por lotes |

## API

- `GET /api/v1/student/submissions/{id}/comments`
- `POST /api/v1/submissions/{id}/keystroke-events`

Ambas rutas usan el token de la sesión. La lectura de comentarios valida que la entrega pertenezca al estudiante.

## Desarrollo

```bash
cd tools/solv-companion
npm install
npm run compile
```

Para probar en el editor: abrí esta carpeta en VS Code, presioná `F5` y se abre una ventana de extensión.

## Empaquetado

```bash
npm run package
```

El script compila y genera `solv-companion-0.1.0.vsix` con `vsce`. El paquete incluye solo `out/`, `package.json` y este README; el código fuente y `node_modules` quedan fuera por `.vscodeignore`.
