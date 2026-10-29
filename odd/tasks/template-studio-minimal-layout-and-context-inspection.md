# Feature: Minimalismo Visual e Inspector Contextual en Template Studio
Branch: feat/template-studio-minimal-layout | Delivery strategy: ask-on-risk | Test runner: cd frontend && npm run test:ci

## Specs

- **S1**: **Eliminación de Banners Redundantes de Alcance (Modo Dual y Evaluador)**:
  - En `frontend/src/app/features/admin/templates/studio/template-studio.component.html`:
    - Retirar el banner estático `.studio-scope-banner` en el Paso 2 (tanto para `targetEnvironment() === 'AMBOS'` como para `'JUEZ_EFIMERO'`).
    - En la cabecera del Paso 2 (`card-section-header`), incorporar un botón de alcance discreto (`.btn-scope-chip`):
      - Si `targetEnvironment() === 'AMBOS'`: botón informativo con icono de información y texto `Modo Dual`.
      - Si `targetEnvironment() === 'JUEZ_EFIMERO'`: botón informativo con icono de información y texto `Evaluador Automático`.
    - Al pulsar el botón, abrir un modal desacoplado (`modal-shell`) con la explicación técnica de ejecución por consola estándar (`stdin` / `stdout`) y límites de servicios interactivos, evitando contaminar el lienzo principal con párrafos estáticos.

- **S2**: **Tarjetas de Tecnologías Compactas y Uniformes**:
  - En `template-studio.component.html` y `template-studio.component.scss`:
    - Retirar el párrafo `.comp-desc` del interior de las tarjetas `.tech-component-card` en:
      - Runtimes backend (Columna 1).
      - Plataformas web (Columna 1).
      - Complementos de framework, herramientas y testing (Columna 2).
      - Motores de bases de datos (Columna 3).
    - Preservar `[title]="item.description"` en el botón interactivo de cada tarjeta para consulta rápida por cursor (tooltip nativo) y accesibilidad.
    - Establecer una altura uniforme (~40px) centrada con fila única: Logo + Nombre + Badge + Botón de edición hermano.

- **S3**: **Inspector Contextual de Tecnología Seleccionada (Propuesta A)**:
  - En `template-studio.component.html`:
    - Incorporar el contenedor semántico `.tech-inspector-strip` inmediatamente debajo de cada subgrupo:
      - Debajo de la grilla de lenguajes principales: renderizar el inspector contextual cuando `selectedRuntimeComponent()` cuenta con descripción.
      - Debajo de la grilla de plataformas web: renderizar el inspector cuando `selectedWebComponent()` cuenta con descripción.
      - Debajo de cada grupo activo de complementos: renderizar la descripción del complemento seleccionado.
      - Debajo de la grilla de bases de datos: renderizar la descripción del motor seleccionado.
    - Estilo en `template-studio.component.scss`: barra horizontal sutil con fondo neutro de superficie, borde suave, icono en miniatura, nombre destacado y descripción legible sin afectar el alineamiento de la grilla superior.

- **S4**: **Sincronización de Tests y Calidad de Estilos**:
  - Actualizar `template-studio.component.spec.ts` para validar que:
    - Las tarjetas de tecnologías son compactas y no contienen `.comp-desc` en su estructura interna en reposo.
    - Al seleccionar una tecnología, el inspector contextual muestra la descripción técnica correspondiente.
    - El chip de modo de alcance abre el modal explicativo al interactuar.
  - Ejecución de `npm run lint:styles` y tests de la vista en verde.

## Tasks

- [x] **T1** (S1): Reemplazar `.studio-scope-banner` por `.scope-chip-btn` en la cabecera del Paso 2 e implementar el modal informativo con `modal-shell`.
- [x] **T2** (S2, S3): Compactar las tarjetas de tecnologías en las 3 columnas y construir las franjas de inspección contextual `.tech-inspector-strip` con sus estilos en `template-studio.component.scss`.
- [x] **T3** (S4): Sincronizar los tests unitarios en `template-studio.component.spec.ts` y validar el paso de linters (`npm run lint:styles`) y tests de Studio (71/71 en verde).

## Log
- **L1 (2026-10-29)**: Detección de sobrecarga cognitiva por descripciones multilínea dentro de todas las tarjetas del catálogo y saturación por el banner explicativo estático de Modo Dual. Decisión de adoptar tarjetas compactas (~40px), inspector contextual por selección y modal bajo demanda para el alcance.
- **L2 (2026-10-29)**: Implementación completada en rama `feat/template-studio-minimal-layout`. Se removieron los banners estáticos invasivos de 120px, se agregó el chip de alcance con modal desacoplado (`modal-shell`), se compactaron las tarjetas (~42px uniformes) preservando el botón de edición hermano sin controles anidados, y se integró la franja de inspección contextual `.tech-inspector-strip` en las tres columnas. Todos los 71 tests de `template-studio.component.spec.ts` y gates de estilo pasaron exitosamente.
