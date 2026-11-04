# Guía de Estilos, Tokens y Regla Multi-Tenant — SOLV

> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System  
> **Gate en CI:** Stylelint (`npm run lint:styles`)  

---

## 1. Política de Cero Literales en Componentes

Para mantener la coherencia visual entre vistas y permitir la personalización multi-tenant por institución (`--tenant-primary`, familias tipográficas y escala base), queda **estrictamente prohibido escribir valores literales** de tipografía y familias tipográficas en hojas de estilo SCSS de componentes (`src/app/**/*.scss`).

Todo cambio es verificado automáticamente en el pipeline de CI mediante el script:
```bash
npm run lint:styles
```

---

## 2. Escala y Tokens Tipográficos

Las propiedades `font-size` y `font-family` deben usar exclusivamente tokens CSS del sistema:

### Tamaños (`font-size`)

| Token | Valor relativo | Equivalente aproximado | Uso recomendado |
| :--- | :---: | :---: | :--- |
| `var(--text-xs)` | `0.75rem` | 12px | Badges, micro-etiquetas, metadatos secundarios, timestamps |
| `var(--text-sm)` | `0.875rem` | 14px | Texto general de tabla, inputs, ayuda secundaria, tooltips |
| `var(--text-base)` | `1rem` | 16px | Párrafos principales, botones estándar, contenido de lectura |
| `var(--text-lg)` | `1.125rem` | 18px | Subtítulos de sección, modales, encabezados de tarjeta |
| `var(--text-xl)` | `1.25rem` | 20px | Títulos de página secundaria, encabezados de panel |
| `var(--text-2xl)` | `1.5rem` | 24px | Títulos principales de dashboard y modales de primer nivel |

> Nota: También se admite `inherit` cuando se hereda explícitamente el tamaño del contenedor padre.

### Familias Tipográficas (`font-family`)

| Token | Pila tipográfica | Uso recomendado |
| :--- | :--- | :--- |
| `var(--font-sans)` | `'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif` | Toda la interfaz de usuario: títulos, formularios, botones, tablas. |
| `var(--font-mono)` | `'JetBrains Mono', 'Fira Code', Menlo, Monaco, Consolas, monospace` | Datos de máquina: referencias OCI, UUIDs, snippets de código, puertos y comandos. |

---

## 3. Regla Multi-Tenant de Colores

1. **Colores de Marca:**
   - La identidad institucional del tenant (universidad o institución) se proyecta únicamente a través de los tokens `--tenant-*`:
     - `--tenant-primary`: Color de marca principal (botones principales, acentos de selección activa).
     - `--tenant-secondary`: Apoyo de marca.
     - `--tenant-contrast`: Texto sobre fondo primario.
   - Prohibido hardcodear colores de marca específicos (ej. azules corporativos hardcodeados `#2563EB`) en componentes genéricos.
2. **Colores Semánticos del Sistema:**
   - Éxito / AC: Verde semántico (`#15803D`, `--color-success`).
   - Advertencia / TLE: Ámbar semántico (`#D97706`, `--color-warning`).
   - Error / WA / RE: Rojo semántico (`#DC2626`, `--color-error`).
   - Neutro / Hibernated: Gris semántico (`#64748B`, `--color-neutral`).

3. **Gate de Colores en Stylelint (`color-no-hex: true`):**
   - Regla activa en `.stylelintrc.json` que prohíbe el uso de códigos hexadecimales literales en el SCSS de componentes.
   - Valores permitidos para color: `var(--tenant-*)`, `var(--color-*)`, `var(--border-color)`, `transparent` y `currentColor`.

---

## 4. Movimiento

El movimiento tiene el mismo tratamiento que la tipografía y el color: **tokens y primitivas compartidas**, nunca animaciones sueltas por vista. La fuente de verdad es `frontend/src/styles/_primitives.scss` (tokens) y `frontend/src/styles/_animations.scss` (keyframes y utilidades).

### 4.1 Tokens de duración y curva

| Token | Valor | Uso |
| :--- | :---: | :--- |
| `var(--motion-instant)` | `80ms` | Feedback inmediato: press, toggle, checkbox |
| `var(--motion-fast)` | `150ms` | Cambios de estado, hover, controles, chips |
| `var(--motion-base)` | `220ms` | Entrada de contenido: tarjetas, bloques, paneles |
| `var(--motion-slow)` | `300ms` | Paneles grandes y cajones con desplazamiento |
| `var(--motion-ease-standard)` | `cubic-bezier(0.2, 0, 0, 1)` | De un estado a otro |
| `var(--motion-ease-enter)` | `cubic-bezier(0.16, 1, 0.3, 1)` | Algo que aparece |
| `var(--motion-ease-exit)` | `cubic-bezier(0.4, 0, 1, 1)` | Algo que desaparece |
| `var(--motion-rise)` | `8px` | Desplazamiento vertical de entrada |
| `var(--motion-stagger-step)` | `40ms` | Espera entre elementos de una lista |
| `var(--motion-stagger-max)` | `320ms` | Tope de espera acumulada |
| `var(--motion-spin-duration)` | `1s` | Giro de espera; vive fuera de la escala porque es un bucle |

Los alias `--transition-fast` y `--transition-normal` siguen existiendo para el código histórico, pero apuntan a los tokens de arriba.

### 4.2 Catálogo de primitivas

Cada intención tiene **una** animación asignada. Si una vista necesita otra cosa, se agrega acá y se documenta; no se declara un `@keyframes` nuevo en el componente. Antes de escribir movimiento nuevo, la galería interna de la sección 4.7 muestra cada primitiva en vivo.

| Primitiva | Keyframe | Cuándo |
| :--- | :--- | :--- |
| `.motion-fade` | `solv-fade-in` | Estado vacío, bloque que solo cambia de opacidad |
| `.motion-rise` | `solv-fade-in-up` | Contenido que entra: tarjetas, bloques de formulario, filas |
| `.motion-panel` | `solv-panel-in` | Diálogos y paneles que se abren sobre la vista |
| `.motion-drawer` | `solv-drawer-in` | Cajones laterales |
| `.motion-chip` | `solv-chip-in` | Pieza que se suma a una lista ya visible |
| `.motion-backdrop` | `solv-fade-in` | Fondo oscuro de modales |
| `.animate-dropdown` | `solv-dropdown-enter` | Menús y popovers |
| `.motion-spin` | `solv-spin` | Icono o anillo que indica espera; es el único giro permitido |
| `solv-progress-indeterminate` | — | Barra de espera sin porcentaje |

**Salidas.** Se activan con `animate.leave="<clase>"` y duran `--motion-fast` con `--motion-ease-exit`; un menú usa `--motion-instant`.

| Primitiva | Keyframe | Cuándo |
| :--- | :--- | :--- |
| `.motion-fade-out` | `solv-fade-out` | Bloque que solo pierde opacidad |
| `.motion-backdrop-out` | `solv-fade-out` | Fondo oscuro de un overlay propio |
| `.motion-panel-out` | `solv-panel-out` | Panel o diálogo que se cierra |
| `.motion-drawer-out` | `solv-drawer-out` | Cajón lateral que se retira |
| `.motion-dropdown-out` | `solv-dropdown-out` | Menú o popover |
| `.motion-modal-out` | `solv-modal-wait` | Modal completo: fondo y diálogo salen juntos |

`.motion-modal-out` se coloca en el elemento que el template retira: el fondo y el diálogo llevan el fundido y la escala, y el elemento marcado solo aporta la duración que Angular espera (`solv-modal-wait`, una animación que no cambia píxeles) porque puede ser un envoltorio `display: contents`. Todas las salidas llevan `pointer-events: none` para que lo que se está yendo no intercepte clics.

### 4.3 Escalonado de listas

Para que una grilla entre en cascada no se escriben `nth-child` con delays a mano. Se aplica `.motion-rise.motion-stagger` y se numera cada elemento:

```html
<article class="template-card motion-rise motion-stagger" [style.--motion-index]="i">
```

El sistema calcula la espera y la corta en `--motion-stagger-max`, así la tarjeta veinte no tarda un segundo en aparecer.

### 4.4 Reglas

1. **Prohibido** un `@keyframes` nuevo en un componente para entradas, salidas, modales, cajones, chips o dropdowns: ya existe la primitiva.
2. **Prohibido** animar por decoración: nada de rebotes, parallax, iconos que giran sin motivo ni sombras que laten.
3. Duraciones **máximo `--motion-slow` (300ms)**; el bucle solo se admite para comunicar espera (`.motion-spin` y la barra indeterminada).
4. Toda animación lleva su equivalente en `@media (prefers-reduced-motion: reduce)`. Las utilidades `.motion-*` lo resuelven solas; el resto sigue la sección 4.5.
5. Nunca animar `width`, `height`, `top` ni `left` en listas: sólo `opacity` y `transform`.
6. Cuando se toca una vista con keyframes locales, se migran a las primitivas en ese mismo cambio (misma política que la ignore-list de Stylelint).
7. Excepción declarada: `bannerSlideDown` en `admin-dashboard` (aviso que baja 6px desde el borde superior). Ninguna primitiva desciende sin agregar escala ni pertenece a un aviso; se mantiene local hasta que exista una primitiva de aviso superior.
8. Los tokens de duración y curva no se escriben como literales en SCSS de componentes; el gate de estilos ya bloquea hexadecimales y tamaños de fuente literales, y este contrato se revisa en code review.
9. Una salida va con `animate.leave` sobre **el elemento que el template retira**. Si el modal es un componente propio, el atributo va en su etiqueta de uso (`<teacher-invite-modal animate.leave="motion-modal-out">`), no en el `<modal-shell>` interno: Angular solo anima el nodo que el `@if` saca del DOM y no ejecuta la salida de un hijo de un componente que se destruye.
10. Una salida no se inventa para cada cierre: solo se migran overlays que hoy desaparecen de golpe. Un elemento sin estado de cierre no lleva `animate.leave` decorativo.
11. La entrada y la salida del mismo elemento van en el mismo cambio: si se abre con `.motion-panel`, se cierra con `.motion-panel-out`; si se abre con `.animate-dropdown`, se cierra con `.motion-dropdown-out`.

### 4.5 Movimiento reducido

Con `prefers-reduced-motion: reduce` activo:

| Qué | Cómo se comporta |
| :--- | :--- |
| Primitivas de entrada (`.motion-fade`, `.motion-rise`, `.motion-panel`, `.motion-drawer`, `.motion-chip`, `.motion-backdrop`, `.animate-dropdown`) | Se resuelven en 1ms y el elemento queda en su estado final. Vive en `_animations.scss`. |
| Primitivas de salida (`.motion-*-out`) | 1ms, no `none`: Angular retira el nodo al recibir el evento de fin, y sin animación no habría evento. El elemento sale de inmediato y no queda colgado en el DOM. |
| Animaciones locales de componente (pulsos de estado, shimmer, skeleton, cursor, aviso superior) | Se apagan con `animation: none` en el propio componente, al final del archivo. |
| Estilos globales heredados (`.toast-notification`, `.highlight-target-pulse`, `.animate-pulse-subtle`) | Se apagan en `styles.scss`. |
| Barras de progreso indeterminadas | Pierden el barrido y pasan a una pista tramada estática. Nunca se rellenan al 100%: eso diría que el proceso terminó. El texto de estado y el contador de segundos siguen visibles. |
| `.motion-spin` | Sigue girando: es la señal de una operación en curso y va siempre junto a un texto ("Cargando…", "Comprobando…"). Un giro decorativo no se apaga, se elimina del diseño. |

Dos condiciones para cualquier animación nueva:

1. Ningún estado puede depender del movimiento. Al apagarlo, el color, la etiqueta y el texto tienen que seguir explicando qué pasa ("Tipeando", "Ejecutando", "EN VIVO", el aviso de éxito).
2. El bloque `@media` va después de la regla animada y con la misma especificidad (`.tarjeta .punto`, no `.punto`), para que el override gane por orden y no haga falta `!important`.

### 4.6 Gate de movimiento

`frontend/scripts/motion-gate.mjs` revisa todos los `.scss` de `src/` y corre dentro de `npm run lint:styles`, igual que los gates de estilos inline, fechas y datos técnicos. Bloquea:

1. Un `@keyframes` que redefina una primitiva (`solv-*`) o que repita exactamente el cuerpo de una de ellas.
2. Nombres de giro (`spin`, `spinner`, `solvSpin`) fuera de `_animations.scss`: el giro es `.motion-spin`.
3. Un `@keyframes` de entrada en un componente: si no se usa en un bucle `infinite`, exige estar registrado como excepción.
4. Un archivo con animación activa y sin bloque `prefers-reduced-motion`.
5. Una declaración `animation` que apunte a un keyframe que no existe en el archivo ni en el catálogo global.

Lo que el gate no molesta: los bucles de estado (pulsos, shimmer, skeleton, cursor), que siguen siendo locales y solo necesitan su versión reducida; el uso de primitivas globales desde `styles.scss` y `_animations.scss`; y los archivos que solo declaran `animation: none`.

**Cómo justificar una excepción**

Solo aplica a animaciones locales que no son bucle y que no tienen primitiva equivalente (el caso vigente es `bannerSlideDown`). El procedimiento:

1. Añadir la entrada a `MOTION_EXCEPTIONS` en `frontend/scripts/motion-gate.mjs` con `file` y `reason`; el gate rechaza entradas sin archivo o con un motivo de menos de 20 caracteres.
2. La excepción vale únicamente en el archivo declarado; el mismo nombre en otro archivo vuelve a fallar.
3. Explicar en el PR por qué ninguna primitiva cubre el caso.
4. No hay excepciones para redefinir primitivas ni para giros: eso se resuelve usando `.motion-*`.

### 4.7 Galería de animaciones (herramienta interna)

`/dev/animaciones` es una página de consulta para quien escribe interfaz. Pide sesión de administración (`authGuard` + `adminGuard`), no aparece en la navegación y no forma parte de las vistas de estudiante ni de docente. Vive en `frontend/src/app/features/dev/motion-gallery/`.

| Bloque | Qué muestra |
| :--- | :--- |
| Indicadores de carga | `.motion-spin`, `skeleton` con `shimmer` y con `pulse`, `status-badge` en `running`, `typing-dot` (documentado, sin vista previa: vive en `aula-telemetry-widget`) |
| Entradas | `.motion-fade`, `.motion-rise`, `.motion-panel`, `.motion-backdrop`, `.animate-dropdown`, `.motion-drawer`, `.motion-chip`, `.motion-stagger` |
| Salidas | `.motion-fade-out`, `.motion-backdrop-out`, `.motion-panel-out`, `.motion-drawer-out`, `.motion-dropdown-out`, `.motion-modal-out` |
| Otros efectos | `.highlight-target-pulse` y la tabla de efectos locales por vista |

Cada tarjeta trae la clase, para qué sirve, la muestra en vivo, duración y curva, y su propio botón de reproducción. La barra superior reproduce todas las entradas, todas las salidas o reinicia las muestras. Las muestras usan las primitivas reales de `_animations.scss` y los componentes compartidos (`skeleton`, `status-badge`): la galería no tiene copias ni keyframes propios.

El control **Normal / Reducido** aplica, solo dentro de la página, las mismas reglas de `prefers-reduced-motion: reduce` que ya usa SOLV (atributo `data-motion-demo` y reglas acotadas en el SCSS de la galería). El resto de la aplicación sigue obedeciendo la preferencia real del sistema. Al entrar, la página arranca en el modo que pide el sistema y lo avisa.

Se actualiza junto con el catálogo: al agregar o retirar una primitiva, al cambiar una duración y al cerrar una deuda de movimiento local. La tabla de efectos locales lista lo que todavía vive en componentes; los overlays que siguen desapareciendo de golpe están documentados ahí como deuda y esta página no los migra.

---

## 5. Política de Archivos Legacy e Ignore-List

Los componentes históricos de administración que aún poseen colores hex pendientes de migración se encuentran aislados temporalmente en el archivo `.stylelintignore`, cada uno con su respectivo comentario `// TODO: salir de la lista exige limpiar el archivo en el mismo PR`.

**Regla de salida:** Cuando un componente o módulo sea modificado o refactorizado, debe sanearse y eliminarse de `.stylelintignore` en el mismo PR. Todo archivo nuevo nace cumpliendo estrictamente sin poder ser añadido a la ignore-list.

