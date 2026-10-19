# Propuesta: Constructor de Entornos Componibles por Catálogo de Tecnologías y Construcción Local Determinista de Imágenes (`solv-local/stack:<sha256>`)

## Intención

El flujo anterior de creación de plantillas (`TemplateCreateModalComponent`) presentaba inconsistencias estructurales y de experiencia de usuario:
1. **Duplicación de selección entre el Paso 2 y el Paso 3:** Seleccionar un modelo preconfigurado (`template_models`) en el Paso 2 (`step-identity`) saltaba al Paso 3 (`step-image`), donde se renderizaba la misma grilla de `template_models` y se sobrescribían los campos de identidad ingresados por el usuario.
2. **Imposibilidad de componer pilas tecnológicas multicapa:** El Paso 3 forzaba la elección excluyente de una única imagen base (`python` o `node` o `go`), impidiendo que administradores o docentes combinaran tecnologías complementarias (por ejemplo, Python + Angular + MySQL).
3. **Configuración dispersa de bases de datos:** Los motores de datos aparecían tanto como imágenes base independientes en el Paso 3 como servicios satélite acoplados en el Paso 5 (`step-resources`).
4. **Sobrecarga de jerga de infraestructura en un modal de 6 pasos:** La interfaz exponía conceptos de bajo nivel (`OCI tags`, `cgroups v2`, `memory.high`, `smoke test`, `tools_declared`) dentro de un modal de 6 pestañas con desplazamiento horizontal.

Este cambio reemplaza el modal de 6 pasos por una **Pantalla Dedicada del Constructor de Entornos** con diseño de **2 paneles en vivo** (Izquierda: Selección Visual de Arquitectura y Tecnologías en 3 Capas sincronizada bidireccionalmente con el Modo Avanzado de Dockerfile y Scripts; Derecha: Resumen del Entorno y Verificación en Vivo), respaldada por **Construcción Local Determinista de Imágenes direccionada por Contenido (`solv-local/stack:<sha256>`)** y un **Catálogo de Tecnologías** extensible.

## Alcance

### Dentro del Alcance
- **Catálogo Extensible de Componentes Tecnológicos (`template_components`):**
  - Migración idempotente en PostgreSQL y datos iniciales para tecnologías curadas en 3 capas componibles:
    1. `runtime` (Lenguaje principal y plataforma web opcional: Python, Node.js, Java, Go, C/C++, Rust, PHP, Angular, React, Vue).
    2. `addon` (Frameworks y librerías filtrados por compatibilidad con el lenguaje principal: FastAPI, Flask, Django, Streamlit, Spring Boot, Maven, Vite, TypeScript, Ciencia de Datos).
    3. `database` (Servicios acoplados de persistencia y caché: Sin Base de Datos, SQLite3, PostgreSQL, MySQL, MongoDB, Redis).
  - Gestión administrativa y modal asistido de registro de nuevas tecnologías con soporte de gestores de paquetes (`pip`, `npm`, `apt` validados por expresión regular estricta) e instrucciones personalizadas, junto con galería de iconos vectoriales SVG oficiales (`TechLogoComponent` con monograma determinista de respaldo).
  - Integración de la pestaña **"Catálogo de Tecnologías"** dentro de `/admin/modelos-categorias` (`ModelLibraryComponent`).
- **Construcción Local Determinista de Imágenes en el Nodo (`ImageBuilderPort`):**
  - Cuando una plantilla combina múltiples capas (ej. Python 3.12 base + Angular/Node + FastAPI), el backend calcula una firma SHA-256 canónica de los componentes ordenados y construye una imagen local inmutable (`solv-local/stack:<sha256>`) una única vez durante la verificación/publicación, aprovechando la caché de capas `overlay2` de Docker para que los espacios de trabajo de los estudiantes inicien en `< 1s` sin descargas por alumno.
  - Las plantillas de un solo lenguaje sin capas adicionales resuelven directamente a su imagen oficial fijada sin reconstrucciones innecesarias.
  - Limpieza automática de imágenes locales `solv-local/stack:*` huérfanas que no estén referenciadas por plantillas activas o pendientes.
  - Versionado sin corte al editar los componentes de una plantilla activa: los nuevos laboratorios usan el nuevo hash construido mientras los contenedores en ejecución permanecen intactos.
- **Pantalla Dedicada del Constructor de Entornos (`/admin/plantillas/nueva` y `/admin/plantillas/:id/editar`):**
  - Diseño a página completa con panel lateral derecho fijo de **Resumen del Entorno** (Resumen humano + pestaña `< /> Manifiesto` JSON + cálculo de concurrencia de estudiantes en tiempo real + verificación y publicación).
  - Terminología académica y de ingeniería clara en el Modo Visual, manteniendo sincronización bidireccional completa con el **Modo Avanzado (Dockerfile / Scripts)**.
  - Resolución automática del comando de compilación y ejecución cuando el propósito es **Evaluador Automático de Ejercicios (`JUEZ_EFIMERO`)** según el lenguaje principal elegido.
- **Selección de Tecnologías en Solicitudes Docentes:**
  - Los docentes que solicitan un nuevo entorno desde `/teacher/templates` (`TemplateRequestModalComponent`) seleccionan los componentes del entorno desde el mismo catálogo para enviar una especificación estructurada lista para auditar y aprobar.
- **Gobernanza y Documentación:**
  - Registro de `ADR-046` y actualización de `docs/ARQUITECTURA/ADR/INDEX.md`.

### Fuera del Alcance
- Publicación en registros Docker distribuidos multi-nodo (SOLV opera en nodo único On-Premise; las imágenes construidas residen en el daemon Docker local del servidor).
- Modificación del aislamiento de contenedores de estudiantes (`DOCKER-USER` y `cgroups v2` permanecen sin cambios y consumen `docker_image`, `setup_script` y `services_config`).

## Capacidades

### Nuevas Capacidades
- `template-components-catalog`: Persistencia en base de datos, API REST (`/api/v1/admin/template-components`), generador asistido de instrucciones por gestor de paquetes (`pip`/`npm`/`apt`) y catálogo vectorial `TechLogoComponent`.
- `deterministic-image-building`: Síntesis de `Dockerfile` direccionado por contenido y construcción/caché local `solv-local/stack:<sha256>` durante la verificación de entorno, con poda de imágenes huérfanas y actualización de plantillas sin corte.
- `template-environment-studio-ui`: Pantalla dedicada del Constructor de Entornos (`/admin/plantillas/nueva`, `/admin/plantillas/:id/editar`) con selector de 3 columnas filtrado por compatibilidad, Modo Avanzado (`</>`) sincronizado, cálculo de capacidad y flujo docente compartido.

### Capacidades Modificadas
- `admin-templates` y `model-library`: Rutas actualizadas hacia el Constructor de Entornos; `ModelLibraryComponent` extendido con la pestaña **Catálogo de Tecnologías**; tablas `lab_templates` y `template_models` extendidas con `stack_config JSONB`.

## Áreas Afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `backend/migrations/00022_template_components_and_studio.sql` | Nuevo | Migración SQL para `template_components`, columna `stack_config JSONB` y semilla del catálogo tecnológico |
| `backend/internal/core/domain/template_component.go` y `admin_governance.go` | Nuevo / Modificado | Modelos de dominio (`TemplateComponent`, `StackConfig`, `ComposedTemplatePlan`, `ImageBuilderPort`) |
| `backend/internal/core/services/template_composer_service.go` | Nuevo | Compilador determinista SHA-256 de capas de imagen, validación de paquetes `pip`/`npm`/`apt` |
| `backend/internal/core/services/env_test_service.go` | Modificado | Soporte de construcción local de imágenes (`ImageBuilderPort`) durante la verificación y limpieza de imágenes huérfanas |
| `backend/internal/delivery/http/admin_academic_handler.go` y `router.go` | Modificado | Endpoints para `/api/v1/admin/template-components` + previsualización compuesta |
| `frontend/src/app/features/admin/templates/components/tech-logo/` | Modificado | Catálogo de logotipos vectoriales SVG en activos estáticos + monograma de respaldo |
| `frontend/src/app/features/admin/templates/studio/` | Nuevo | `TemplateStudioComponent` (`/admin/plantillas/nueva`, `/admin/plantillas/:id/editar`) |
| `frontend/src/app/features/admin/templates/components/model-library/` | Modificado | Pestaña "Catálogo de Tecnologías" + `ComponentFormModalComponent` |
| `frontend/src/app/features/teacher/templates/` | Modificado | Selección de tecnologías del catálogo en solicitudes docentes |
| `docs/ARQUITECTURA/ADR/` | Modificado | `ADR-046` y actualización de `INDEX.md` |

## Criterios de Aceptación

- [x] Se elimina la duplicación entre el Paso 2 y el Paso 3; `TemplateStudioComponent` funciona en pantalla dedicada (`/admin/plantillas/nueva` y `/admin/plantillas/:id/editar`).
- [x] El administrador puede combinar 1 Lenguaje Principal + 1 Plataforma Web opcional + Complementos compatibles + 1 Base de Datos, observando la actualización en tiempo real en el panel derecho (**Resumen del Entorno** y `< /> Manifiesto`).
- [x] Alternar entre **Modo Visual** y **Modo Avanzado (Dockerfile / Scripts)** conserva el estado de manera bidireccional.
- [x] Las combinaciones multicapa construyen una imagen determinista `solv-local/stack:<sha256>` verificada mediante contenedor de prueba, y las imágenes huérfanas se eliminan automáticamente.
- [x] El administrador puede registrar una nueva tecnología en el catálogo (con icono SVG oficial o monograma e instalación asistida `pip`/`npm`/`apt`) desde `/admin/modelos-categorias` o desde las columnas del constructor.
- [x] Los tests de backend (`go test ./...`), los tests de frontend (`ng test`), Stylelint (`npm run lint:styles`) y ESLint (`npm run lint:ts`) pasan sin literales hexadecimales ni prefijos `solv-`.
