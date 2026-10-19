# Especificación: Constructor de Entornos Componibles por Catálogo de Tecnologías y Construcción Local Determinista de Imágenes (`solv-local/stack:<sha256>`)

## 1. Resumen

Esta especificación define los requisitos funcionales, de comportamiento y de seguridad para el **Constructor de Entornos Componibles por Catálogo de Tecnologías**, abarcando:
1. **Catálogo de Componentes Tecnológicos (`template_components`)** con 3 capas (`runtime`, `addon`, `database`), paquete de logotipos vectoriales SVG oficiales en activos estáticos (`public/tech-icons/<slug>.svg`) con soporte de URL/SVG propio y monograma de respaldo, e instalación asistida por gestor de paquetes (`pip`, `npm`, `apt`).
2. **Construcción Local Determinista de Imágenes direccionada por Contenido (`solv-local/stack:<sha256>`)** con limpieza automática de imágenes huérfanas y versionado sin corte para plantillas en uso activo.
3. **Interfaz Dedicada del Constructor de Entornos (`/admin/plantillas/nueva` y `/admin/plantillas/:id/editar`)** con diseño de 2 paneles (Izquierda: Modo Visual sincronizado bidireccionalmente con Modo Avanzado de Dockerfile y Scripts; Derecha: Resumen del Entorno y Manifiesto en vivo), terminología de dominio limpia, resolución automática de comando de ejecución para el Juez Virtual y selector compartido para solicitudes docentes.

---

## 2. Requisitos y Escenarios de Aceptación

### REQ-1: Catálogo de Componentes Tecnológicos y Logotipos Vectoriales (`template-components-catalog`)

- **R1.1:** El sistema DEBE persistir los componentes tecnológicos en la tabla multi-tenant `template_components` clasificados en tres capas:
  - `runtime`: Lenguaje de ejecución principal o plataforma web (`Python`, `Node.js`, `Java`, `Go`, `C/C++`, `Rust`, `PHP`, `Angular`, `React`, `Vue`).
  - `addon`: Frameworks, librerías, drivers de base de datos, ORMs o utilidades complementarias (`FastAPI`, `Flask`, `Django`, `Streamlit`, `Spring Boot`, `Vite`, `TypeScript`, `Prisma`, `SQLAlchemy`, `Ciencia de Datos`) con lista `compatible_runtimes`.
  - `database`: Motores acoplados de persistencia o caché (`SQLite3`, `PostgreSQL`, `MySQL`, `MongoDB`, `Redis`), cuya disponibilidad efectiva en la Columna 3 del Constructor de Entornos está gobernada por las capacidades reales del nodo (`GetRuntimeCapabilities` según ADR-045, con detección en vivo de la versión mayor de PostgreSQL).
- **R1.2:** Cada componente tecnológico DEBE definir un `icon_key` resuelto por `TechLogoComponent` mediante activos estáticos vectoriales oficiales en `frontend/public/tech-icons/<slug>.svg` (31 iconos oficiales sin etiquetas `<text>` artificiales), aceptando además rutas personalizadas (`/tech-icons/...`, `https://...`, `data:image/svg+xml,...`) y degradando automáticamente a un monograma determinista de 2 letras cuando la clave no existe en el catálogo o cuando el recurso de imagen emite error de carga.
- **R1.3:** El administrador DEBE poder registrar una nueva tecnología en el catálogo para las capas `runtime` y `addon` mediante **Modo Asistido por Gestor de Paquetes** (`install_mode`: `pip` | `npm` | `apt` + `package_names`) o **Modo Avanzado** (`custom`), pudiendo elegir un icono de la galería visual o indicar una URL/ruta `.svg` personalizada con vista previa en vivo, y seleccionando un **Perfil de Consumo en Memoria (`memoryProfile`)** en lugar de adivinar megabytes manualmente (`library` -> `0 MB`, `devserver` -> `256 MB`, `heavy` -> `384 MB` para complementos; `light` -> `384 MB`, `standard` -> `512 MB`, `heavy` -> `1024 MB` para lenguajes principales). Los motores de la Columna 3 (`Base de Datos`) no incluyen botón `+ Nuevo`, ya que son gobernados por `RuntimeCapabilities.satellite_services` del nodo (mostrando `+0 MB RAM` para SQLite3, `~50 MB RAM` y versión detectada en vivo para PostgreSQL, y estado deshabilitado `No habilitado en este nodo` cuando un motor satélite no está aprovisionado).
- **R1.4:** En el Modo Asistido por Gestor de Paquetes, el backend DEBE validar cada nombre de paquete contra la expresión regular `^[a-zA-Z0-9@_./=<>-]+$` (rechazando metacaracteres de shell `;`, `&`, `|`, `$`, `` ` ``, espacios) con HTTP `422 Unprocessable Entity` antes de sintetizar instrucciones `RUN` deterministas y sin caché.

#### Escenario 1.1: Resolución de Logotipos Vectoriales Oficiales y Respaldo a Monograma
- **Dado** que `TechLogoComponent` recibe una clave de ecosistema conocida (como `python`, `angular`, `django`, `flask`, `laravel`, `postgres`, `mysql`, `sqlite`, `redis` o `bun`)
- **Cuando** el componente se renderiza
- **Entonces** DEBE mostrar un elemento `<img class="tech-logo-img">` apuntando al activo vectorial oficial `/tech-icons/<clave>.svg`.
- **Y Cuando** recibe una URL personalizada (`https://.../icono.svg`) que falla al cargar o una clave desconocida (`elixir`)
- **Entonces** DEBE conmutar sin parpadeo roto al monograma determinista de 2 letras (`.tech-logo-monogram`).

#### Escenario 1.2: Alta Asistida de un Componente del Catálogo sin Escribir Bash
- **Dado** que el administrador abre el modal para registrar una nueva tecnología desde la Columna 2 o desde `/admin/modelos-categorias`
- **Cuando** ingresa el nombre `"Django 5 LTS"`, selecciona capa `addon`, compatibilidad `python`, icono `django` (o ruta `.svg` propia), elige `Paquete de Python (pip)` e introduce `django djangorestframework`
- **Entonces** el backend valida los nombres de paquete, sintetiza la instrucción `RUN pip install --no-cache-dir django djangorestframework`, persiste el componente en el catálogo y lo habilita de inmediato en la Columna 2 con su icono vectorial.

---

### REQ-2: Construcción Local Determinista de Imágenes y Limpieza de Huérfanas (`deterministic-image-building`)

- **R2.1:** Cuando una plantilla se compone de un único componente `runtime` sin capas adicionales de construcción, el compositor DEBE resolver directamente a la `base_image` fijada sin ejecutar `docker build`.
- **R2.2:** Cuando una plantilla combina múltiples capas de construcción, el backend DEBE calcular un hash SHA-256 canónico de 12 caracteres hexadecimales sobre la imagen base y las capas ordenadas, produciendo el tag local `solv-local/stack:<sha256>`.
- **R2.3:** Durante la verificación del entorno (`EnvTestJob`), si `solv-local/stack:<sha256>` ya existe en la caché del daemon Docker local, el servicio DEBE omitir la reconstrucción y pasar directamente a comprobar los binarios declarados; si no existe, DEBE construir la imagen localmente informando las etapas de progreso.
- **R2.4 (Versionado sin Corte):** Actualizar las tecnologías de una plantilla en uso por cursos activos DEBE asignar el nuevo `docker_image` (`solv-local/stack:<nuevo-sha256>`) para futuros espacios de trabajo, sin detener ni reiniciar los contenedores de estudiantes que ya están en ejecución.
- **R2.5 (Limpieza Automática de Huérfanas):** Tras actualizar o eliminar una plantilla, o al finalizar una construcción local, el backend DEBE identificar y eliminar las imágenes locales `solv-local/stack:*` que no estén referenciadas por ninguna plantilla no rechazada ni modelo activo.

---

### REQ-3: Interfaz Dedicada del Constructor de Entornos, Estado Inicial Limpio y Sincronización Bidireccional (`template-environment-studio-ui`)

- **R3.1:** Las rutas `/admin/plantillas/nueva` y `/admin/plantillas/:id/editar` DEBEN renderizar `TemplateStudioComponent` en pantalla dedicada de dos columnas (Izquierda: Propósito, Identidad, Selección Visual de 3 Columnas / Modo Avanzado de Dockerfile y Scripts, y Capacidad; Derecha: Panel fijo con Resumen del Entorno en vivo, pestaña de manifiesto y botones de verificación y publicación), sin barras de combinaciones preconfiguradas en código.
- **R3.2:** En la creación de una nueva plantilla o solicitud docente, el estado inicial DEBE comenzar vacío (`selectedRuntimeId = null`, `selectedCategoryId = null`, `dockerImage = ''`), sin preseleccionar ningún lenguaje ni categoría por defecto, y sin inyectar carpetas fijas (`/backend`, `/frontend`, `/src`) en `/home/workspace` desde el Modo Visual. Seleccionar o deseleccionar tecnologías en el Modo Visual DEBE actualizar en tiempo real la imagen resuelta, el Dockerfile generado, los programas declarados, la memoria sugerida y el nombre sugerido.
- **R3.3:** El cálculo de memoria recomendada (`recommendedRamMB`) y del piso vital (`minimumFloorMB`) tanto en `TemplateComposerService` como en `TemplateStudioComponent` DEBE basarse en los procesos residentes activos (`IDE Web 210 MB` o `Sandbox Juez 32 MB`, servicio de base de datos acoplado como `50 MB` en PostgreSQL Aula o `0 MB` en SQLite, y huella del proceso de ejecución activo), sin sumar linealmente librerías instaladas en disco (`pip`/`npm`/`apt` con perfil `library`), alineándose con los niveles de ADR-045 (`384`, `512`, `768`, `1024`, `2048 MB` en IDE; `128`, `256`, `512 MB` en Juez). La Sección 3 DEBE renderizar la barra gráfica segmentada `.memory-bar`, el simulador de alumnos simultáneos, el ajuste personalizado en MB con candados de piso vital y techo del servidor (respaldados en `AdminGovernanceService`), e incluir `resource_profile: { min_mb, high_mb, max_mb }` en el payload de creación y actualización.
- **R3.4:** Todos los componentes nuevos y modificados DEBEN usar Angular 22 standalone con Signals (`signal`, `computed`), control flow nativo (`@if`, `@for`), selectores en kebab-case sin prefijo `solv-`, y hojas de estilo SCSS gobernadas exclusivamente por tokens CSS sin colores hexadecimales literales.
