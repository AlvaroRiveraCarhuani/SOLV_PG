# ADR 093: Fundaciones del Sistema de Diseño UI

> **Estado:** Aceptado
> **Fecha:** 2026-11-22
> **Ámbito:** Interfaz, Sistema de diseño
> **Relacionado:** ADR-012 (arquitectura frontend con Angular standalone y signals), ADR-038 (tipografía white-label por tenant), ADR-067 (política terminológica de modalidades)

---

## Contexto

La especificación `docs/UI/design-system-spec.md` v1.5 fija en sus secciones 1 y 2 el contrato de tokens, la geometría del shell y los controles base. La aplicación venía de una escala Slate con tinte azulado, un topbar de 60 px, un sidebar de 240/64 px y primitivas duplicadas por vista: cada pantalla resolvía sus botones, su buscador y sus modales con estilos locales.

Ese estado producía cuatro problemas concretos:

- **Superficies azuladas.** Los grises de la interfaz compartían el matiz del azul institucional. En un producto multi-tenant, ese matiz compite con el color de la institución y hace que el tema oscuro se lea como una variante del azul.
- **Shell sin regla de ancho.** El sidebar arrancaba expandido en todas las rutas, incluidas las de trabajo intensivo donde el código o la tabla necesitan el ancho completo. El ancho colapsado de 64 px tampoco coincidía con la rejilla de iconos.
- **Controles con alturas distintas.** Los botones definían su altura por relleno, el buscador medía 34 o 38 px y los iconos de barra 36 px. La misma fila de herramientas mezclaba tres ritmos verticales.
- **Primitivas dispersas.** No existía un cajón lateral genérico, ni un popover de filtros, ni un badge semántico fuera de los estados de entorno y de los veredictos. Cada vista nueva volvía a resolver lo mismo con sus propias clases.

## Decisión

### 1. Escala neutra Zinc en tema claro y tema oscuro

`frontend/src/styles/_semantic.scss` reemplaza la escala Slate por Zinc (`#FAFAFA`, `#F4F4F5`, `#E4E4E7`, `#09090B`, `#52525B`, `#71717A`). El bloque `[data-theme="dark"]` declara superficies carbón escalonadas y tintes translúcidos para estados y veredictos. `_primitives.scss` alinea la escala `--color-gray-*` y las superficies de código con los mismos neutros.

Se completan los tokens que faltaban: `--color-success-dark`, `--color-success-bg`, `--color-success-subtle`, `--color-danger` y `--color-danger-bg`. El verde oscuro `#15803D` queda como fondo del aviso de éxito; el texto sale de `--text-inverse`, que en tema oscuro es oscuro.

### 2. Motor de tenant con luminosidad relativa

`_tenant.scss` mantiene un único color base en canales HSL. Los derivados usan `clamp()` sobre la luminosidad en lugar de valores fijos: el hover baja 6 puntos y el activo 12, con topes de 24 %/75 % y 18 %/68 %. El texto sobre el color primario sale de `--tenant-primary-text`, calculado en `TenantService` por contraste AA. En tema oscuro cambian solo las alfas de `--tenant-primary-subtle` y `--tenant-primary-border`.

### 3. Geometría del shell

| Elemento | Medida | Regla |
| --- | --- | --- |
| Topbar | `48px` | Borde inferior `--border-subtle`; sin breadcrumbs ni acciones de sesión. Conserva menú, marca, ayuda, avisos y perfil. |
| Sidebar expandido | `224px` | Borde derecho `--border-subtle`; ítem activo con `--tenant-primary-subtle`, icono y texto en `--tenant-primary`, radio `6px`. |
| Sidebar en rail | `52px` | Solo iconos de `18px`, etiqueta en `title` y `aria-current` en la ruta activa. |
| Controles de barra | `32px` | Botones, inputs, selects y toggles. Variante `.btn-sm` de `28px` para barras compactas. |
| Radios | `6px` / `8px` | Controles, inputs y botones en `6px`; tarjetas, popovers y modales en `8px`. |

El rail por defecto se resuelve en `SidebarRailService`, que escucha `NavigationEnd` y aplica la lista de rutas de la especificación. Los tres layouts consumen la misma señal; el botón del topbar la alterna. Las rutas administrativas no figuran en la tabla y arrancan expandidas.

### 4. Jerarquía de botones

`.btn-primary` usa el color del tenant y su texto calculado, con un máximo de una instancia por pantalla. `.btn-secondary` es la acción neutra: fondo de superficie, borde sutil y hover sobre `--bg-surface-subtle`. `.btn-ghost` no tiene borde y solo cambia el fondo al pasar el cursor. `.btn-outline` se conserva como alias heredado de `.btn-secondary` para no romper las vistas que todavía lo usan.

### 5. Primitivas compartidas

| Primitiva | Archivo | Contrato |
| --- | --- | --- |
| Badge semántico | `shared/components/semantic-badge/` | Tonos success, warning, error, neutral e info; variante suave o plana; punto de `6px` opcional. |
| Lista densa | `styles.scss` (`.list-dense*`) | Filas de `40px` separadas por borde, sin tarjeta por fila; estado activo con barra interior del tenant. |
| Barra de búsqueda | `shared/components/search-bar/` | `32px`, badge `/` y atajo que enfoca la primera instancia de la página. |
| Popover de filtros | `shared/components/filter-popover/` | Disparador secundario, contador circular, panel de dos columnas, cierre por clic fuera o Escape y acción `Limpiar`. |
| Cajón lateral | `shared/components/drawer/` | Panel derecho de `380px`, fondo, Escape, foco al abrir y slots de cuerpo y pie. |
| Confirmación | `shared/components/confirm-modal/` | Signals, `alertdialog`, botones del sistema y textos formales. |

Las etiquetas y los textos nuevos se escriben en español neutro formal, con trato de usted. Los estados de entorno siguen en `status-badge` y los veredictos en `verdict-badge`: las primitivas nuevas no los reemplazan.

### 6. Lo que no cambia

- No se renombran rutas ni se migran vistas de administración, docencia o estudiante. El rail se resuelve sobre las rutas vigentes y admite los alias de la especificación.
- No se agrega un interruptor de tema: el contrato define los tokens de tema oscuro, no un control de usuario.
- `status-badge`, `verdict-badge`, `status-tabs`, `view-switcher` y `pagination-bar` conservan su contrato y sus consumidores.
- Las vistas con estilos locales siguen su curso de migración por rol; esta fase solo entrega la base compartida.

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Mantener la escala Slate y cambiar solo el color de marca | El matiz azulado sigue presente en bordes, textos y tema oscuro; el white-label no se lee neutro. |
| Guardar el estado del rail en cada layout | Tres copias del mismo estado y tres reglas de ruta distintas. Un servicio con señal única evita la deriva. |
| Resolver el rail con `@media` por ancho de pantalla | La decisión depende de la tarea de la vista, no del tamaño de la ventana: una tabla densa necesita el ancho aunque la pantalla sea grande. |
| Un componente de badge por dominio | Duplica tonos, tamaños y puntos. El badge genérico cubre etiquetas; los badges de entorno y veredicto conservan su semántica propia. |
| Eliminar `.btn-outline` en esta fase | Rompe decenas de plantillas que todavía lo usan. Se mantiene como alias y se retira cuando las vistas migren. |

## Consecuencias

- Los tokens globales cambian de valor, no de nombre: las vistas existentes heredan la escala neutra y las alturas nuevas sin tocar su SCSS.
- El topbar pierde 12 px de alto y el sidebar gana la regla de rail. Las vistas que calculan `calc(100vh - 48px)` quedan alineadas con la nueva altura.
- `TenantService` sigue siendo el único punto que escribe `--tenant-primary-*`; el CSS solo deriva hover, activo, sutil y borde.
- Los componentes nuevos no declaran hexadecimales ni tamaños de fuente literales: consumen tokens y pasan el gate de estilos.
- La documentación de tokens (`docs/UI/GUIA_ESTILOS_TOKENS.md`) incorpora la escala Zinc, el tema oscuro, la geometría del shell y las primitivas.

## Verificación

- `npm run build` en `frontend/`.
- `npm run lint:styles` (gates de estilos inline, fechas, datos máquina, movimiento, terminología y stylelint).
- Los specs de los componentes compartidos afectados (`search-bar`, `topbar`).
- Revisión de las rutas con rail por defecto: inicio del estudiante, listado y detalle de materias, evaluaciones, resolución de ejercicios, inicio y detalle del docente, ficha de práctica, editor, aula, cola de revisión y mesa de revisión.
