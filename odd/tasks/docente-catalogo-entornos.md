# Feature: Catálogo de Entornos del Docente

Objetivo: que el docente elija el entorno donde van a programar sus alumnos por lo que **incluye** —lenguaje, framework, base de datos— y no por el nombre de una imagen Docker ni por cifras de RAM y CPU.

Branch: `feat/teacher-environments-catalog` (desde `main`) | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm run test:ci`

Forecast de líneas autoradas: ~900 (backend ~450, frontend ~350, docs ~100). Supera el presupuesto de 400 → definir estrategia de entrega antes de empezar.

Diseño de referencia: `docs/ARQUITECTURA/ADR/ADR-054-catalogo-entornos-docente.md` y `docs/UI/DOCENTE/CATALOGO_ENTORNOS.md`.

## Decisiones cerradas

- **La unidad es el entorno, no la imagen.** El catálogo del docente se arma con `template_models` y resuelve sus piezas con `stack_config` contra `template_components`. Motivo: la tarjeta actual solo puede mostrar `docker_image`, `base_ram_mb` y `target_environment` porque el endpoint legacy devuelve ocho columnas; el docente no puede saber qué herramientas trae.
- **La plantilla ejecutable es implementación.** Se muestra en "Requisitos técnicos" y decide `available`. Si no hay plantilla, la acción es "Solicitar", no "Usar".
- **Las plantillas sin modelo no se ocultan.** Se agrupan como "Otros entornos" y muestran el stack que declaren; si no declaran nada, lo dicen.
- **La portada no lleva KPIs.** El conteo vive en la barra de filtros y cambia con el resultado. Motivo: la fila actual repite el inventario y el KPI de oficiales siempre da el total porque el mapeo asume `is_official = true`.
- **Los recursos se mueven al detalle.** RAM y CPU son una consecuencia de la elección, no un criterio para elegir.
- **"Oficial" es un flag explícito.** Se agrega `is_official` a `template_models` y se define para `lab_templates`; ninguna capa asume verdadero.
- **El buscador indexa piezas.** "PostgreSQL" tiene que encontrar todo lo que lo trae, sin conocer la categoría del administrador.

## Specs

- **S1 — Contrato y read model.**
  - `GET /api/v1/teacher/environments?q=&component=&environment_type=` con `{data:[...], meta:{total}}`.
  - Cada ítem: `id`, `kind` (`curated` | `direct`), `name`, `summary`, `description`, `environment_type`, `category`, `components[]` (`slug`, `name`, `layer`, `icon_key`), `available`, `template_id`, `official`.
  - Resolución del stack en una sola lectura: `jsonb_array_elements` de `stack_config` + `JOIN` a `template_components`; sin N+1.
  - Criterio: con siete modelos semilla, la respuesta trae sus piezas con nombre e icono; una plantilla sin modelo aparece como `kind: "direct"`.

- **S2 — Oficial y disponibilidad.**
  - Migración con `template_models.is_official BOOLEAN NOT NULL DEFAULT FALSE` y el equivalente en `lab_templates`; sin default verdadero en ninguna capa.
  - `available` se deriva de una plantilla aprobada vinculada por `source_template_id` o `model_id`.
  - Criterio: el catálogo deja de mostrar todas las tarjetas como oficiales; un modelo sin plantilla devuelve `available: false`.

- **S3 — Curaduría de los modelos semilla.**
  - Completar `stack_config` de los siete modelos sembrados en `00001_baseline.sql` (Python Data Science, ML, Node, Go, C/C++, Rust, Java) con las piezas de `template_components`.
  - Fallback de presentación: si no hay `stack_config`, usar `tools`; si tampoco, "Sin detalle declarado".
  - Criterio: ninguna tarjeta del catálogo semilla queda sin herramientas visibles.

- **S4 — Portada.**
  - Retirar la fila de KPIs y las cifras de RAM/CPU de la tarjeta.
  - Tarjeta con modalidad en palabras, nombre humano, resumen, chips por capa y acciones "Ver detalle" / "Usar" o "Solicitar".
  - Buscador por nombre, resumen y pieza; filtros por modalidad.
  - Criterio: buscar "PostgreSQL" devuelve todos los entornos que lo incluyen; la tarjeta no muestra `docker_image`.

- **S5 — Detalle.**
  - Panel lateral con "Qué incluye" (principal → complementos → base de datos), "Requisitos técnicos" plegado y acción principal.
  - Si `available` es falso, explicar el motivo y ofrecer "Solicitar entorno".
  - Criterio: el detalle se cierra con Escape y no pierde el foco al volver a la tarjeta.

- **S6 — Verificación y documentación indivisible.**
  - `cd backend && go vet ./... && go test ./...`; test del read model con fixtures de piezas y modelos.
  - `cd frontend && npm run lint:styles && npm run lint:ts && npm run test:ci && CI=true npm run build`.
  - `docs/API_CONTRACTS.md` con el endpoint; ADR-054 ya escrito; `CATALOGO_ENTORNOS.md` actualizado si cambia la pantalla.

## Tasks

- [ ] **S1**: contrato y read model del catálogo, con resolución de piezas y pruebas de fixtures.
- [ ] **S2**: migración de `is_official`, regla de `available` y retiro del default verdadero.
- [ ] **S3**: curaduría de `stack_config` en los modelos semilla y fallback de presentación.
- [ ] **S4**: portada sin KPIs, tarjeta por herramientas y filtros.
- [ ] **S5**: detalle con árbol de piezas, requisitos técnicos y solicitud.
- [ ] **S6**: verificación y documentación.

## Decisiones pendientes

1. **Etiqueta visible:** "Plantillas de Entorno" o "Entornos". El concepto es entorno; la etiqueta se confirma con el usuario antes de tocar la navegación.
2. **Qué hace "Usar":** abrir el editor de práctica con el entorno preseleccionado o crear un laboratorio dentro de una materia. Cambia el flujo y el contrato de la acción.
3. **Plantillas huérfanas:** mostrarlas siempre en "Otros entornos" o solo cuando la categoría no tenga modelos.
4. **Política de oficiales:** criterio del administrador para marcar un entorno (revisión, origen institucional o ambos).
5. **Alcance del catálogo:** solo `IDE_PERSISTENTE` en la primera entrega o también `JUEZ_EFIMERO` con su tiempo límite.

## Fuera de alcance

- La sesión de laboratorio en vivo (el "aula" con conectados y telemetría). Es otra tarea.
- El constructor de entornos del administrador y el copiloto de arquitectura.
- Crear piezas nuevas desde el catálogo del docente.

## Log

- **L1 (verbatim):** "Planteemos un docente queriendo crear un entorno y tiene en su mente x tecnologias, se metera en el catologo de plantillas, pero o me topo con cosas como esta… Y digo, uy, que es una imagen? que es eso?"
- **L2 (verbatim):** "no se de cada plantilla que tiene, solo tengo algo como imagen, y ya."
- **L3 (verbatim):** "al docente no le importa si eso ocupa 1024 o 1 gpu."
- **L4 (verbatim):** "si dejamos 2 en ese caso, que dejemos 2, no creo que sea bueno dejar ese espacio solo para 2 cosas."
- **Hallazgo verificado:** el mapeo del frontend hace `is_official: item.is_official ?? true` y el API no devuelve el campo; el KPI de oficiales cuenta el total del catálogo.
- **Hallazgo verificado:** `GET /api/v1/templates` sale de `internal/infrastructure/database/template_repository.go` y devuelve ocho columnas; `description`, `category_id`, `tools_declared` y `stack_config` existen en `lab_templates` pero no se consultan.
- **Hallazgo verificado:** `template_models` tiene siete modelos sembrados con título, descripción, herramientas y categoría; el docente no los ve.
