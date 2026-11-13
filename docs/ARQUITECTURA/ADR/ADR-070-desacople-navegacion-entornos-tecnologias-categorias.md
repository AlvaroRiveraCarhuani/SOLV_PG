# ADR-070: Desacople de Navegación entre Catálogo de Entornos, Tecnologías y Categorías Académicas

**Estado:** Aceptado  
**Fecha:** 2026-11-12  
**Área:** Frontend (`Angular 22 Standalone`, `Zoneless`, `Signals`, `Tokens Design System`)  
**Relacionado:** ADR-046 (constructor de plantillas y componentes), ADR-054 (catálogo de entornos docente), ADR-065 (vocabulario de entorno de práctica), ADR-067 (política terminológica de modalidades)

---

## 1. Contexto

La interfaz del administrador concentraba en una sola ruta (`/admin/modelos-categorias`, `ModelLibraryComponent`) tres responsabilidades con niveles de abstracción dispares:
1. **Bloques técnicos atómicos** (`template_components`): Runtimes, complementos, bases de datos y mapa de dependencias que alimentan el Template Studio.
2. **Entornos institucionales ensamblados** (`template_models`): Modelos oficiales derivados de plantillas aprobadas.
3. **Taxonomía pedagógica** (`template_categories`): Categorías académicas del campus ("Ciencia de Datos & IA", "Desarrollo Web & Cloud", etc.).

Esta acumulación generaba una rotura de contrato en la experiencia del administrador: el menú anunciaba "Modelos y Categorías", pero al entrar el contenido dominante era el catálogo de software y librerías Docker. Asimismo, la vista de plantillas utilizaba la denominación técnica de infraestructura ("Plantillas Docker"), desalineada del vocabulario canónico del proyecto ("Catálogo de Entornos").

---

## 2. Decisión

### 2.1 Desacople de Vistas y Rutas Especializadas

Se divide la gestión en dos vistas dedicadas y se reubica la taxonomía en su contexto de dominio:

1. **`/admin/tecnologias` (`TechCatalogComponent`):**  
   Ubicada en la sección **INFRAESTRUCTURA**. Centraliza la gestión de componentes técnicos (lenguajes, frameworks, complementos, bases de datos) y el visualizador del Mapa del Sistema.
2. **`/admin/categorias` (`AcademicCategoriesComponent`):**  
   Ubicada en la sección **ACADÉMICO Y GESTIÓN**. Centraliza la gestión de categorías académicas, su ordenamiento prioritario y la protección contra eliminación de categorías en uso.
3. **`/admin/plantillas` (`AdminTemplatesComponent`):**  
   Se unifica la denominación visible en cabecera a **"Catálogo de Entornos"** y el botón de acción a **"Nuevo Entorno"**, integrando los entornos ejecutables y modelos oficiales bajo la misma sombrilla funcional.
4. **Redirección de compatibilidad:**  
   La ruta histórica `/admin/modelos-categorias` se mantiene configurada con `redirectTo: 'tecnologias'` para no romper marcadores ni referencias previas.

---

## 3. Consecuencias

* **Cohesión de Dominio:** Cada pantalla atiende una sola responsabilidad funcional.
* **Vocabulario Unificado:** Se elimina la denominación de infraestructura en favor del lenguaje centrado en el dominio del laboratorio.
* **Integración con Help Center:** Se definieron entradas contextuales independientes (`tech-catalog` y `academic-categories`) en el catálogo de ayuda del sistema.
