# Feature: Extensión VS Code SOLV Companion para Laboratorios

Objetivo: transformar OpenVSCode Server en el entorno de desarrollo oficial e integrado de SOLV bajo el modelo de pantalla completa (GitHub Codespaces / Gitpod), eliminando el iframe web e integrando recepción de avances con diff nativo, checklist selectivo, rollback seguro, entrega directa y comentarios pedagógicos.

Branch: `feat/companion-full-experience` (desde `main`) | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm test` en `tools/solv-companion/`

Estado: **especificación cerrada para implementación**. Los contratos base de variables de entorno y montaje condicional ya están en `main` (commit `55d75f6`). Esta ficha define la experiencia completa de UI/UX, sincronización, diff y entrega.

Fuentes:
- `proyecto-grado/Observaciones/Flujo de Laboratorios del Estudiante y Propuesta Extension VSCode.md`
- `proyecto-grado/Observaciones/Propuesta Tecnica Extension VSCode SOLV Companion.md`
- `docs/ARQUITECTURA/ADR/ADR-058-avance-voluntario-del-docente-en-aula.md`
- `docs/UI/SOLV_COMPANION.md`

---

## 1. Contexto Pedagógico y de Aula

1. **Eliminación del Iframe (Modelo Codespaces):**
   El laboratorio virtual del estudiante se abre a pantalla completa en su propia pestaña del navegador. Se eliminan las barras web contenedoras de Angular que recortaban píxeles útiles de programación. Toda la interacción institucional ocurre desde adentro del editor mediante la extensión preinstalada `SOLV Companion`.

2. **Avance en Vivo vs Examen:**
   - **En exámenes (congelados/contrarreloj):** no se emiten avances ni se comentan líneas; el alumno programa contra el tiempo y el entorno pasa a solo lectura al sonar la campana (ADR-059).
   - **En clase guiada (laboratorio en vivo):** el docente programa en su propio entorno y emite avances con código base o soluciones intermedias. El alumno recibe el aviso en tiempo real y decide si acepta el código, qué archivos incorporar y cómo comparar sus cambios.
   - **En revisión asíncrona (post-entrega):** el alumno abre su entrega calificada para revisar observaciones pedagógicas dejadas por el docente en el SpeedGrader.
3. **Regla Estricta de Aislamiento de Roles:**
   Esta ficha implementa exclusivamente la experiencia del **ESTUDIANTE** (recepción de avances, diff, checklist, rollback, comentarios y entrega). El docente emite avances desde la plataforma web (`/teacher/aula`). El agente implementador **no debe mezclar ni implementar controles de emisión docente adentro de la extensión** en esta fase para evitar contaminar la sesión del alumno.

---

## 2. Presentación Visual e Interfaz (UI/UX)

La interfaz se construye con **HTML + CSS Nativo** utilizando las variables CSS temáticas de VS Code (`var(--vscode-...)`). Cero librerías externas o frameworks pesados en el webview. Se adapta en forma automática a temas oscuros, claros y alto contraste. Ocupa **0 píxeles extra** de pantalla cuando está colapsada (`Ctrl+B`).

### A. Barra Lateral de Actividades (Activity Bar)
- **Ícono:** Isotipo vectorial SVG monocromático de SOLV (24x24) ubicado en la barra lateral izquierda.
- **Panel Webview:**
  1. **Cabecera de Contexto:**
     - Materia (ej. `SIS-420: Desarrollo Web Fullstack`)
     - Docente a cargo (ej. `Ing. Carlos Mendoza`)
     - Estado de la entrega: `Borrador`, `Entregada` o `Calificada (Score: 85/100)`
  2. **Consigna y Objetivos:**
     - Resumen pedagógico del ejercicio y criterios de evaluación.
  3. **Módulo de Avance de Clase (cuando hay avance pendiente):**
     - Alerta no invasiva: *«El docente compartió una actualización (Práctica 1)»*.
     - **Checklist de Archivos con Badges Inteligentes:**
       - Archivo nuevo para el alumno: badge verde `(Nuevo)` -> seleccionado por defecto.
       - Archivo modificado por el alumno: badge ámbar `(Modificado)` -> acción `[Ver diferencias]` al lado.
       - Archivo idéntico: badge neutro `(Sin cambios)` -> desmarcado por defecto.
     - Botones de acción:
       - `[Comparar y Aplicar Selección]`
       - `[Descartar Avance]`
     - **Acción de Reversión:**
       - Si ya se aplicó un avance: botón `[Deshacer último avance]` que restaura la versión anterior respaldada.
  4. **Comentarios Pedagógicos del Docente:**
     - Lista de observaciones ancladas agrupadas por archivo.
     - Al hacer click en un comentario, el editor abre el archivo y posiciona el cursor en la línea comentada.

### B. Barra Inferior de Estado (Status Bar)
- Elemento discreto izquierdo: `$(mortar-board) SOLV: SIS-420` (al hacer click enfoca el panel de SOLV).
- Elemento de acción: `$(cloud-upload) Entregar Práctica` (gatilla el diálogo de confirmación y entrega).
- Monitor de conexión: `$(check) SOLV` o `$(sync~spin) Conectando...`. En caso de microcorte, reintenta en silencio y solo muestra advertencia discreta si el fallo persiste más de 2 minutos. Cero popups flotantes intrusivos.

### C. Modo Pestaña Completa (Editor Tab / Split View)
- La extensión permite abrir la consigna y el estado del laboratorio como una pestaña completa en el editor (`vscode.window.createWebviewPanel` con comando `SOLV: Abrir Consigna en Pestaña`).
- **Ergonomía de pantalla dividida:** El estudiante puede trabajar con la consigna a la izquierda y su código a la derecha (`Split Editor`), o maximizarla para leer enunciados largos, diagramas y tablas sin estrecharse en la barra lateral.

---

## 3. Especificación Técnica y Contratos (Sin Parches)

### A. Diff Nativo en Memoria (`vscode.diff`)
- La extensión registra un `TextDocumentContentProvider` con el esquema virtual `solv-advance:`.
- **Ajuste en Backend Go:** El endpoint `GET /api/v1/class-advances/{id}/files` debe devolver el contenido completo de cada archivo en un mapa `files: map[string]string` además de las listas `added` y `overwritten`.
- Para inspeccionar diferencias de un archivo modificado, la extensión ejecuta:
  ```typescript
  const localUri = vscode.Uri.file(path.join(workspaceRoot, relativePath));
  const virtualUri = vscode.Uri.parse(`solv-advance:/advances/${advanceId}/${relativePath}`);
  await vscode.commands.executeCommand('vscode.diff', localUri, virtualUri, `${relativePath} (Tu versión ↔ Avance docente)`);
  ```
- **Ventaja técnica:** Cero archivos temporales basura en el disco del alumno; el motor de diferencias de VS Code opera puramente en memoria.

### B. Aplicación Selectiva Granular (Checklist)
- **Ajuste en Backend Go:** `POST /api/v1/class-advances/{id}/apply` debe aceptar un payload JSON opcional:
  ```json
  {
    "selected_files": ["src/models/product.ts", "package.json"]
  }
  ```
- Si `selected_files` está presente y no está vacío, el servicio en Go solo respalda y sobreescribe los archivos incluidos en la lista, respetando el resto del proyecto del alumno. Si se omite, aplica todos (compatibilidad hacia atrás).

### C. Deshacer Avance (Rollback con Auditoría)
- Cuando el backend aplica un avance, respalda los archivos sobreescritos en `/home/workspace/.solv_backups/<advance_id>/`.
- **Nuevo Endpoint en Backend Go:**
  `POST /api/v1/class-advances/{id}/rollback`
  - Restaura los archivos desde `.solv_backups/<advance_id>/` al volumen activo del alumno.
  - Registra el evento de auditoría `domain.AuditClassAdvanceRollback`.
  - Actualiza el estado de la entrega/destinatario para permitir re-aplicar si fuera necesario.

### D. Sellos de Excelencia Operativa
1. **Guardado Previo Forzoso (`saveAll`):**
   Antes de abrir cualquier diff o ejecutar `apply`, la extensión ejecuta `await vscode.workspace.saveAll(false)` para asegurar que las pestañas en memoria no entren en conflicto con los archivos del disco.
2. **Pre-flight Check antes de Entregar:**
   Al presionar `[Entregar Práctica]`, la extensión evalúa `vscode.languages.getDiagnostics()`. Si existen errores de sintaxis (`DiagnosticSeverity.Error`), despliega una advertencia pedagógica antes de confirmar:
   *«Se detectaron 2 errores de compilación en tu código. ¿Deseas entregar de todas formas o prefieres revisarlos?»*
3. **Limpieza de Espacio de Trabajo:**
   La extensión configura en `.vscode/settings.json`:
   ```json
   {
     "files.exclude": {
       "**/.git": true,
       "**/.cache": true,
       "**/.openvscode-server": true,
       "**/.solv_backups": true
     }
   }
   ```
4. **Corrección de Hover Duplicado:**
   En `CommentDecorator`, eliminar la doble registración (`hoverMessage` en la decoración + `registerHoverProvider`) para que el tooltip docente se renderice exactamente una vez.

---

## 4. Plan de Trabajo para el Agente Implementador

### Paso 1: Ajustes en Backend (Go)
1. Extender `GET /api/v1/class-advances/{id}/files` para retornar el mapa `files` con contenido de texto de cada archivo.
2. Extender `POST /api/v1/class-advances/{id}/apply` para admitir `selected_files: []string`.
3. Implementar `POST /api/v1/class-advances/{id}/rollback` para restaurar desde `.solv_backups/<advance_id>/` y auditar.
4. Agregar tests unitarios para los 3 cambios en `backend/internal/core/services/` y `backend/internal/delivery/http/`.

### Paso 2: Corrección y Servicios Base en Extensión (TypeScript)
1. Corregir el hover duplicado en `comment-decorator.ts`.
2. Crear `src/advances/advance-client.ts` para interactuar con los endpoints de avances, files, apply y rollback.
3. Crear `src/advances/advance-content-provider.ts` implementando `vscode.TextDocumentContentProvider` para el esquema `solv-advance:`.

### Paso 3: Webview del Panel Lateral y Status Bar
1. Crear `src/views/solv-panel-view.ts` implementando `vscode.WebviewViewProvider`.
2. Diseñar el HTML/CSS nativo con variables `--vscode-*` para las 4 secciones (Sesión, Consigna, Avances con checklist, Comentarios).
3. Conectar la mensajería bidireccional entre la Webview y la extensión (`postMessage` / `onDidReceiveMessage`) para las acciones: ver diff, aplicar selección, deshacer y refrescar.
4. Crear `src/status/status-bar-controller.ts` para los accesos directos en la barra inferior.

### Paso 4: Pre-flight de Entrega y Empaquetado
1. Implementar el comando `solv.submit` con validación de diagnostics y llamada a `POST /api/v1/workspaces/{id}/submit`.
2. Correr `npm test` y `npm run package` para generar el `.vsix` limpio y validado.
