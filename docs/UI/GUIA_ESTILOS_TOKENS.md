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

## 4. Política de Archivos Legacy e Ignore-List

Los componentes históricos de administración que aún poseen colores hex pendientes de migración se encuentran aislados temporalmente en el archivo `.stylelintignore`, cada uno con su respectivo comentario `// TODO: salir de la lista exige limpiar el archivo en el mismo PR`.

**Regla de salida:** Cuando un componente o módulo sea modificado o refactorizado, debe sanearse y eliminarse de `.stylelintignore` en el mismo PR. Todo archivo nuevo nace cumpliendo estrictamente sin poder ser añadido a la ignore-list.

