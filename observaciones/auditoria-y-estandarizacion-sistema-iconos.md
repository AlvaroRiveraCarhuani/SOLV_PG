# Auditoría y propuesta de estandarización: jerarquía, dimensiones y peso óptico del sistema de iconos

> [!IMPORTANT]
> **Carácter del documento**: informe técnico de diagnóstico y planificación. No modifica código en este paso. Diagnostica las causas visuales y métricas por las cuales los iconos se perciben desproporcionados, borrosos o pequeños en la interfaz, y define el plan de tareas paso a paso para su estandarización.

---

## 1. Diagnóstico Técnico y Causas Raíz

Tras una inspección visual comparativa y una auditoría directa del código fuente en `frontend/src/` (`styles.scss`, `_primitives.scss`, `tech-logo`, `sidebar`, `topbar`, y vistas de plantillas admin/docente), se identificaron cuatro factores estructurales que causan la percepción de degradación visual:

### A. La trampa del contenedor anidado en `tech-logo` (Efecto "Favicon atrapado")
* **Ubicación**: [`tech-logo.component.ts:156-157`](frontend/src/app/features/admin/templates/components/tech-logo/tech-logo.component.ts#L156-L157) y [`tech-logo.component.scss:6-9`](frontend/src/app/features/admin/templates/components/tech-logo/tech-logo.component.scss#L6-L9).
* **Mecanismo**: El componente `tech-logo` fue concebido originalmente para tarjetas de selección grandes (`size="32"`). Cuenta con:
  * Contenedor con `border: 1px solid var(--border-subtle)`
  * `padding: var(--space-1)` (4px por lado)
  * Cálculo interno forzado de dimensiones: `[width]="resolvedSize() - 8"` y `[height]="resolvedSize() - 8"`.
* **Consecuencia real**:
  * En la barra de filtros ([`admin-templates.component.html:131`](frontend/src/app/features/admin/templates/admin-templates.component.html#L131)) se invoca con `[size]="16"`. La imagen real mide `16 - 8 = 8px`.
  * En las tarjetas de catálogo ([`admin-templates.component.html:454`](frontend/src/app/features/admin/templates/admin-templates.component.html#L454) y [`teacher-templates.component.html:142`](frontend/src/app/features/teacher/templates/teacher-templates.component.html#L142)) se invoca con `[size]="18"`. La imagen real mide `18 - 8 = 10px`.
  * En la vista de tabla ([`admin-templates.component.html:315`](frontend/src/app/features/admin/templates/admin-templates.component.html#L315)) se invoca con `[size]="14"`. La imagen real mide `14 - 8 = 6px`.
* **Impacto visual**: Un gráfico vectorial complejo (como el vapor de la taza de Java, las dos serpientes de Python o el monograma de .NET) renderizado a 6px–10px dentro de un recuadro gris con padding, colocado a su vez dentro de un chip o pastilla que ya tiene borde y fondo, genera una doble caja (*box-in-box*) y convierte el logo en un punto indistinguible.

---

### B. Falta de tokens de iconos y uso de números impares/fraccionarios
* **Ubicación**: [`frontend/src/styles.scss:307, 743, 786`](frontend/src/styles.scss#L307) y [`frontend/src/styles/_primitives.scss`](frontend/src/styles/_primitives.scss).
* **Mecanismo**: `_primitives.scss` define tokens para espacio (`--space-*`), radios (`--radius-*`), tipografía (`--text-*`) y movimiento (`--motion-*`), pero **no posee una escala de tokens para iconos**.
* **Consecuencia real**:
  * En `styles.scss` proliferan medidas arbitrarias e impares:
    * `.btn-icon`: `width: 15px; height: 15px;`
    * `.search-icon`: `width: 15px; height: 15px;`
    * `.reload-icon`: `width: 15px; height: 15px;`
    * `.action-icon` / `.meta-icon`: `width: 14px; height: 14px;`
* **Impacto visual (Subpixel rasterization)**:
  * Los iconos de la librería (Lucide) se diseñan en un *viewBox* de **24×24px** con trazo de **2px**.
  * Escalar 24px a 15px implica un factor fraccionario de `0.625`. El trazo pasa a medir `1.25px`. Al no alinearse con la rejilla física de píxeles enteros ni semienteros limpios, el motor del navegador aplica *antialiasing* borroso.
  * Además, los iconos de 14px–15px con trazo atenuado carecen de peso óptico frente a la tipografía regular (`--text-sm` = 14px / peso 500), dando una sensación de fragilidad o palidez.

---

### C. Desproporción espacial en la barra lateral colapsada
* **Ubicación**: [`frontend/src/app/shared/components/sidebar/sidebar.component.scss:21, 95-101`](frontend/src/app/shared/components/sidebar/sidebar.component.scss#L95-L101).
* **Mecanismo**:
  * El ancho de la barra colapsada es de `64px`.
  * La clase `.nav-icon` aplica `width: 18px; height: 18px;` sin un contenedor visual definido.
* **Impacto visual**:
  * Un glifo lineal de 18px flotando aislado en un carril vertical de 64px de ancho produce un exceso de espacio negativo sin delimitación táctil ni anclaje visual.
  * Por el contrario, en la cabecera superior ([`topbar.component.scss:27-50`](frontend/src/app/shared/components/topbar/topbar.component.scss#L27-L50)), los botones de acción usan un contenedor de `36×36px` con un icono centrado de `20px`, resultando nítidos y visualmente estables.

---

### D. Desequilibrio de escala en botones primarios de ancho completo
* **Ubicación**: [`frontend/src/app/features/teacher/templates/teacher-templates.component.scss:286-293`](frontend/src/app/features/teacher/templates/teacher-templates.component.scss#L286-L293).
* **Mecanismo**: En la tarjeta docente, el botón «Usar en laboratorio» ocupa el 100% del ancho del card (más de 300px) con una altura de control de 38px–40px, mientras que su icono interno (`lucideSparkles`) se fuerza a `16px`.
* **Impacto visual**: El icono queda diminuto en relación con la masa superficial del botón azul, perdiendo jerarquía de acción.

---

## 2. Propuesta de Arquitectura y Sistema de Tokens

Para erradicar valores mágicos y mantener la disciplina de diseño de SOLV, se formula la siguiente estandarización:

### A. Escala de Tokens para Iconos (Tier 1: `_primitives.scss`)

```scss
// Escala Modular de Iconos (Base en rejilla de 4px / viewBox 24x24)
--icon-xs: 12px; // Micro-indicadores (chevrons en tags, copiado compacto)
--icon-sm: 16px; // Base estándar UI (metadata, botones estándar, chips, search-bar)
--icon-md: 20px; // Navegación y controles de cabecera (sidebar, topbar, botones destacados)
--icon-lg: 24px; // Cabeceras de sección y modales principales
--icon-xl: 32px; // Estados vacíos compactos / badges destacados
```

### B. Matriz de Uso por Componente y Rol Visual

| Componente / Contexto | Token | Dimensión | Contenedor Visual / Hitbox | Justificación |
|---|---|---|---|---|
| **Sidebar colapsada** | `--icon-md` | 20px | Tile 36×36px (radio 8px) | Anclaje óptico sobre columna de 64px. |
| **Topbar (acciones)** | `--icon-md` | 20px | Botón 36×36px | Mantiene el estándar actual que ya funciona bien. |
| **Buscador global** | `--icon-sm` | 16px | Posición absoluta `left: 12px` | Factor entero limpio para input de 38px. |
| **Botón primario ancho** | `--icon-md` o `--icon-sm` | 18px–20px | Gap 8px, centrado | Equilibrio contra la masa de fondo del botón. |
| **Botones secundarios** | `--icon-sm` | 16px | Gap 6px | Jerarquía subordinada a la acción principal. |
| **Metadata de tarjetas** | `--icon-sm` | 16px | Flex centrado al *cap-height* | Evita desvanecimiento frente a texto de 14px. |
| **Tech chips (logos)** | `--icon-sm` | 16px | **Modo glifo directo** (sin caja) | Elimina la "caja dentro de caja" y devuelve nitidez. |
| **Acciones de fila (tablas)** | `--icon-sm` | 16px | Botón 28×28px o 32×32px | Zona táctil adecuada sin alterar densidad. |

---

### C. Rediseño de `tech-logo`: Separación de Modos (`glyph` vs `boxed`)

Para resolver el problema de los chips de tecnologías sin romper las tarjetas grandes:

1. **Modo `boxed` (por defecto para selección/catálogo amplio)**:
   * Mantiene el fondo, radio y padding para tamaños `>= 24px` (ej. en modales de selección o grids de presets).
2. **Modo `inline` / `glyph` (para chips, filtros y tablas)**:
   * Sin padding externo (`padding: 0`).
   * Sin fondo (`background: transparent`).
   * Sin borde (`border: none`).
   * La imagen o SVG ocupa el 100% del tamaño solicitado (16px exactos).
   * Si es monograma, texto plano de alto contraste o insignia circular compacta de 16px.

---

## 3. Plan de Tareas para Ejecución

Cuando se autorice la implementación, las tareas se ejecutarán en el siguiente orden estricto para validar linting y no romper estilos:

### Fase 1: Fundaciones en el Sistema de Diseño
- [ ] **T1.1**: Declarar los tokens `--icon-xs`, `--icon-sm`, `--icon-md`, `--icon-lg`, `--icon-xl` en [`frontend/src/styles/_primitives.scss`](frontend/src/styles/_primitives.scss).
- [ ] **T1.2**: Documentar el contrato de iconos en [`solv-design-system/SKILL.md`](.agents/skills/solv-design-system/SKILL.md).

### Fase 2: Componente `tech-logo`
- [ ] **T2.1**: Agregar `@Input() variant: 'boxed' | 'inline' = 'boxed'` en [`tech-logo.component.ts`](frontend/src/app/features/admin/templates/components/tech-logo/tech-logo.component.ts).
- [ ] **T2.2**: En modo `inline`, renderizar el SVG/imagen ocupando directamente los 16px sin padding ni borde en [`tech-logo.component.scss`](frontend/src/app/features/admin/templates/components/tech-logo/tech-logo.component.scss).
- [ ] **T2.3**: Actualizar los chips de filtros, tarjetas y tablas en `admin-templates.component.html` y `teacher-templates.component.html` para usar `variant="inline"` y `[size]="16"`.

### Fase 3: Navegación y Shell Global
- [ ] **T3.1**: Actualizar [`sidebar.component.scss`](frontend/src/app/shared/components/sidebar/sidebar.component.scss) para elevar los iconos a `var(--icon-md)` (20px) e introducir el tile centrado de 36×36px en modo colapsado.
- [ ] **T3.2**: Estandarizar `.search-icon` a `var(--icon-sm)` (16px) en [`search-bar.component.scss`](frontend/src/app/shared/components/search-bar/search-bar.component.scss) y [`styles.scss`](frontend/src/styles.scss).
- [ ] **T3.3**: Corregir `.btn-icon` global en [`styles.scss`](frontend/src/styles.scss) de 15px a 16px (`var(--icon-sm)`).

### Fase 4: Tarjetas de Catálogo (Admin y Docente)
- [ ] **T4.1**: Ajustar `.meta-icon` en [`admin-templates.component.scss`](frontend/src/app/features/admin/templates/admin-templates.component.scss) de 14px a 16px (`var(--icon-sm)`).
- [ ] **T4.2**: Ajustar `.btn-icon` de los botones de acción en [`teacher-templates.component.scss`](frontend/src/app/features/teacher/templates/teacher-templates.component.scss) a 18px para el botón primario y 16px para los secundarios.
- [ ] **T4.3**: Alinear verticalmente el icono al centro óptico de la tipografía (evitar desalineación por línea base).

### Fase 5: Validación y Control de Calidad
- [ ] **T5.1**: Ejecutar `npm run lint:styles` para comprobar que ningún componente viole las reglas de tokens y que no haya valores residuales no permitidos.
- [ ] **T5.2**: Inspección visual en navegador de las vistas de Administrador y Docente comprobando nitidez a escala 100% y pantallas de alta densidad (HiDPI).
