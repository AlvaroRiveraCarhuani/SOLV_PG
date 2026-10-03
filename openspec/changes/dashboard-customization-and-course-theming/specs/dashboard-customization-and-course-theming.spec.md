# Specification: Personalización Libre de Dashboards y Tematización Híbrida

## Requirements

### R1: Bento Grid Configurable (Docente y Estudiante)
1. El dashboard debe organizarse en una grilla CSS de 12 columnas.
2. Cada widget debe soportar anchos parametrizables:
   - `colSpan: 12` (100% - Ancho completo)
   - `colSpan: 8` (66% - Dos tercios)
   - `colSpan: 6` (50% - Mitad de pantalla)
   - `colSpan: 4` (33% - Un tercio / Compacto)
3. En modo personalización (`isCustomizing = true`):
   - Los widgets deben poder reordenarse mediante drag-and-drop con `@angular/cdk/drag-drop`.
   - Cada tarjeta de widget debe mostrar controles para:
     - Cambiar su ancho (selector rápido: 33%, 50%, 66%, 100%).
     - Ocultar o mostrar el widget.
   - Debe existir un botón "Restablecer Diseño por Defecto".
4. En pantallas móviles (`< 768px`), todos los widgets deben colapsar automáticamente al 100% de ancho respetando el orden elegido.

### R2: Tematización Híbrida de Cursos
1. La entidad Curso (`Course`) debe almacenar el color temático en `course.color`.
2. El selector de color debe ofrecer:
   - 8 presets curados de alto contraste: `emerald`, `indigo`, `purple`, `amber`, `rose`, `cyan`, `teal`, `slate`.
   - Opción libre para ingresar un color hexadecimal personalizado.
3. Las tarjetas de curso deben aplicar el color temático en su acento visual superior, en el badge de código y en las etiquetas de materias.

### R3: Preferencia Personal de Tipografías
1. El usuario debe poder seleccionar su tipografía de interfaz (`--font-sans`) y de código/datos (`--font-mono`) desde el catálogo curado (`curated-fonts.ts`).
2. Las preferencias deben guardarse en almacenamiento local y aplicarse dinámicamente en el elemento raíz `<html>`.

### R4: Calidad y Gobernanza
1. Los estilos SCSS deben respetar estrictamente el sistema de tokens (`_primitives.scss`) sin colores hexadecimales en hojas de estilo.
2. Los valores técnicos visibles deben usar `[machineData]` o la clase `font-mono`.
3. Pruebas unitarias completas para el servicio de layout y los componentes de dashboard.
