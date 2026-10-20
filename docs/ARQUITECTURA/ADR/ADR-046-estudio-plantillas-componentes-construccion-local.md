# ADR-046: Constructor de Entornos Componibles por Catálogo de Tecnologías y Construcción Local Determinista de Imágenes (`solv-local/stack:<sha256>`)

- **Estado:** Aprobado
- **Fecha:** 2026-10-19
- **Contexto:** Core / Gestión de Entornos y Gobernanza Docker

## Contexto y Problema

El flujo original de registro de plantillas se basaba en un modal multipaso de 6 pestañas que presentaba tres limitaciones estructurales:
1. Duplicación de la selección de modelos preconfigurados entre el paso de identidad y el paso de imagen.
2. Imposibilidad de combinar tecnologías de distintos ecosistemas (por ejemplo, un lenguaje backend como Python con una plataforma frontend como Angular y un motor de base de datos acoplado) sin escribir manualmente comandos de aprovisionamiento o construir imágenes externas.
3. Exposición directa de vocabulario de infraestructura (`tags OCI`, `cgroups v2`, `memory.high`, `smoke test`) en la interfaz principal para administradores académicos y docentes.

Si las dependencias adicionales se instalan únicamente en el script de inicio (`setup_script`) cuando el estudiante abre su contenedor, un curso de 30 alumnos entrando simultáneamente multiplica el tráfico de red y el tiempo de arranque. Por otro lado, depender exclusivamente de imágenes pre-armadas en registros externos impide combinar tecnologías a medida o incorporar herramientas nuevas.

## Decisión

1. **Catálogo Extensible de Componentes Tecnológicos (`template_components`), Motores de Base de Datos Gobernados por el Nodo (ADR-045) y Activos Vectoriales Oficiales (`public/tech-icons/`):**
   - Se modela un catálogo persistente en PostgreSQL clasificado en tres capas componibles: `runtime` (lenguaje principal e interfaz web opcional), `addon` (complementos, librerías, drivers y ORMs con matriz de compatibilidad `compatible_runtimes`) y `database` (servicios acoplados).
   - Mientras que la Columna 1 (`Lenguaje y Web`) y la Columna 2 (`Complementos Compatibles`) son capas de catálogo 100% abiertas al administrador mediante el botón `+ Nuevo`, la Columna 3 (`Base de Datos`) está gobernada directamente por las capacidades reales de infraestructura del nodo (`GetRuntimeCapabilities`, ADR-045): expone exclusivamente los motores de base de datos aprovisionados y disponibles en el servidor (`is_available === true`, como SQLite3 nativo con `+0 MB RAM` y PostgreSQL de aula con `~50 MB RAM` detectando su versión mayor dinámicamente mediante `SHOW server_version;` en `PostgresVersionProbe`), omitiendo motores satélite no habilitados (`mysql`, `mongodb`, `redis`) para mantener la interfaz despejada y accionable sin opciones muertas. Los clientes de base de datos u ORMs (`Prisma`, `SQLAlchemy`, `TypeORM`) se registran en el catálogo abierto como `addon`.
   - Se incorpora un modo de alta asistida multi-gestor (`pip`, `uv`, `npm`, `apt`, `cargo`, `go`, `composer` y `custom`) en `ComponentFormModalComponent`, validado mediante expresión regular estricta (`^[a-zA-Z0-9@_./=<>-]+$`). Los gestores compilados o modernos (`uv`, `cargo`, `go`, `composer`) se sintetizan de forma segura con `install_mode: 'custom'` hacia el motor de base de datos y backend sin requerir cambios en el esquema relacional (`InstallMode`). Los desplegables utilizan de forma homogénea `<combobox>` (`ComboboxComponent`) sin elementos `<select>` nativos.
   - En lugar de exigir un valor numérico manual de megabytes, `ComponentFormModalComponent` presenta un selector de **Perfil de Consumo en Memoria (`memoryProfile`)**: para `addon` ofrece `library` (`0 MB` residentes extra, solo ocupa disco), `devserver` (`256 MB` para servidores de desarrollo en segundo plano) y `heavy` (`384 MB` para motores de ciencia de datos o JVM); para `runtime` ofrece `light` (`384 MB`), `standard` (`512 MB`) y `heavy` (`1024 MB`).
   - Se reemplaza el bloque condicional inline (`@switch`) de iconos en `TechLogoComponent` por un paquete de activos estáticos vectoriales oficiales (`frontend/public/tech-icons/<slug>.svg`) para las 31 claves del catálogo base.
   - `TechLogoComponent` y `ComponentFormModalComponent` aceptan además rutas locales (`/tech-icons/...`), URLs externas (`https://...`) o URI de datos (`data:image/svg+xml,...`) en `icon_key`, con degradación automática a monograma determinista de 2 letras ante claves personalizadas o errores de carga (`error` en `<img>`), siendo 100% compatible con despliegues On-Premise sin conexión a internet y futuros entornos híbridos o en nube.

2. **Construcción Local Determinista de Imágenes direccionada por Contenido (`solv-local/stack:<sha256>`), Bootstrap Multi-Distro y Compatibilidad de Imágenes Base:**
   - Cuando una plantilla utiliza únicamente un lenguaje base sin capas adicionales de construcción, resuelve directamente a su imagen oficial fijada.
   - Cuando combina múltiples capas (ej. `Python 3.12 + Angular + FastAPI`), `TemplateComposerService` calcula un hash SHA-256 determinista de 12 caracteres sobre la imagen base y las capas ordenadas, generando el tag inmutable `solv-local/stack:<sha256>`.
   - Al combinar una Plataforma Web (`Angular`, `React`, `Vue`) con un lenguaje base que no sea Node.js (ej. Python, Java, Go, Rust), el generador inyecta un bootstrap multi-distro tolerante a distribuciones Alpine y Debian/Ubuntu:
     `RUN (command -v apk >/dev/null 2>&1 && apk add --no-cache nodejs npm) || (apt-get update && apt-get install -y --no-install-recommends nodejs npm && rm -rf /var/lib/apt/lists/*)`
   - La imagen semilla para `java-21` se actualiza de `eclipse-temurin:21-jdk-alpine` a `eclipse-temurin:21-jdk-jammy` (vía migración `00023_update_java21_base_image.sql`) para garantizar compatibilidad con `glibc` y la extensión de OpenVSCode Server en el nodo.
   - Durante la verificación del entorno (`EnvTestService` + `ImageBuilderPort`), si el tag ya existe en el daemon Docker local del nodo, se reutiliza de inmediato (< 0.2 s); si no existe, se construye una única vez en el servidor aprovechando la caché de capas `overlay2`.
   - Se ejecuta recolección de basura automática (`PruneOrphanStackImages`) para eliminar imágenes `solv-local/stack:*` no referenciadas por plantillas activas o pendientes.

3. **Modelo de Memoria por Perfiles Académicos, Barra Segmentada de 3 Tramos Reales (`memory-bar`), Estimación Unificada de Capacidad y Candados de Seguridad:**
   - Tanto `TemplateComposerService` en Go como `TemplateStudioComponent` en Angular calculan la memoria recomendada (`recommendedRamMB`) y el mínimo requerido (`minimumFloorMB`) a partir de la huella de los procesos en ejecución (`editorBaseMB = 210 MB` en `IDE_PERSISTENTE` o `32 MB` en `JUEZ_EFIMERO`, más `satelliteBaseMB` del servicio acoplado como `50 MB` en PostgreSQL Aula o `0 MB` en SQLite, más la huella del proceso activo), sin sumar linealmente las librerías instaladas en disco (`pip`/`npm`/`apt` con perfil `library`), evitando inflar cuotas por paquetes pasivos.
   - En lugar de botones estáticos con nombres fijos de tecnologías, la Sección 3 de `TemplateStudioComponent` sintetiza dinámicamente dos perfiles académicos claros orientados a la planificación docente:
     - **Perfil Estándar (Recomendado):** asigna la cuota óptima recomendada (`recommendedRamMB`) para clases semanales, algoritmos y prácticas guiadas con la mayor cantidad de alumnos simultáneos.
     - **Perfil Intensivo (Holgura para proyectos):** habilita el siguiente escalón de memoria (`intensiveRamMB`) para compilaciones pesadas, ejecución de múltiples pruebas o manejo de datos extensos (omitiéndose si la selección base ya alcanza el tope máximo del servidor para evitar claves duplicadas).
   - Se unifica la métrica de capacidad del aula eliminando duplicaciones: la capacidad nominal (`nominalConcurrentStudents`) se calcula dividiendo la memoria permitida del servidor entre la cuota por alumno (`maxAllowedRamMB / baseRamMB`), sin volver a sumar la base de datos al divisor puesto que ya se encuentra contemplada en la cuota base.
   - La barra gráfica segmentada `.memory-bar` divide la cuota en 3 tramos reales cuya suma cierra con precisión al 100% de la memoria asignada:
     1. `Motor de evaluación` / `Editor de código` (`editorBaseMB`: 32 MB o 210 MB).
     2. `Tecnologías elegidas` (`stackBaseMB = minimumFloorMB - editorBaseMB`), reflejando la memoria requerida por el stack seleccionado (lenguaje, web, complementos y base de datos).
     3. `Espacio libre para los programas del alumno` (`usableMB = totalMB - editorBaseMB - stackBaseMB`), de modo que `editorBaseMB + stackBaseMB + usableMB === baseRamMB`.
   - Se aplican candados de seguridad en frontend y backend (`AdminGovernanceService.CreateOfficialTemplate` y `UpdateOfficialTemplate`): bloqueo ante valores inferiores al mínimo requerido (`ErrRamBelowVitalFloor`) o superiores al límite del servidor (`ErrRamExceedsHostCapacity`), reajuste automático de la memoria al nivel recomendado si un cambio de stack en el Paso 2 eleva el piso por encima de la memoria configurada, y aviso preventivo cuando la cuota supera la memoria libre actual del servidor.

4. **Modo Avanzado con CodeMirror 6 (`<code-editor>`), Diagnóstico Estático en Vivo, Importación/Exportación y Blindaje de Publicación:**
   - Se reemplaza el uso de `<textarea>` planos en el Modo Avanzado por el componente institucional `<code-editor>` basado en CodeMirror 6, con extensiones de resaltado sintáctico para `dockerfile` y `shell` (`setup.sh`).
   - El editor de `Dockerfile` permanece siempre visible en el Modo Avanzado, permitiendo inspeccionar y modificar la receta Docker independientemente de si la imagen es construida localmente o resuelta directamente.
   - Se proveen herramientas directas en el navegador (sin dependencias externas) para **Importar/Descargar `Dockerfile`**, **Importar/Descargar `setup.sh`** y **Exportar `template-manifest.json`**.
   - Si el administrador edita manualmente el `Dockerfile` en el Modo Avanzado, se activa la guarda `hasCustomDockerfile = true`, preservando el código personalizado ante cambios en la selección visual, e incorporando un botón explícito `"Restablecer desde Selección Visual"` para reiniciar la sincronización.
   - **Diagnóstico Estático en Vivo del Dockerfile (`dockerfileDiagnostics`):**
     - Bloqueo por falta de instrucción `FROM`.
     - Bloqueo por uso de `:latest` o referencias a imágenes sin tag fijo de versión.
     - Advertencia ante comandos `apt-get install` que omitan el flag desatendido `-y` o `--assume-yes`.
   - **Blindaje de Publicación de Plantillas (`requiresVerificationBeforePublish`):**
     - Si la plantilla requiere construcción local (`needsLocalBuild === true` o imagen `solv-local/stack:*`), el botón `"Publicar Plantilla Oficial"` se mantiene deshabilitado hasta que el entorno haya sido comprobado satisfactoriamente (`activeEnvTestJob()?.status === 'success'`).
     - Si el Dockerfile presenta diagnósticos de severidad `error`, tanto la comprobación como la publicación se bloquean preventivamente.
   - Se desmantela de forma definitiva el antiguo wizard multipaso (`template-create-modal`), redirigiendo cualquier acceso vía `?nueva=1` directamente a la ruta dedicada del Studio (`/admin/plantillas/nueva`).
   - Se separan los contratos de dominio y DTOs en `frontend/src/app/features/admin/models/admin-templates.models.ts`. `AdminTemplatesService` opera exclusivamente como servicio de transporte HTTP hacia la API REST (`/api/v1/admin/template-components` y `/api/v1/admin/templates/capabilities`), sin listas estáticas en memoria ni catálogos de respaldo locales.

## Consecuencias

- **Positivas:** Arranque instantáneo (< 1 segundo) para los estudiantes sin descargas repetidas en horario de clase; experiencia de desarrollo y administración de nivel Enterprise con edición de código real en el navegador; diagnóstico preventivo que evita imágenes rotas en el registro o construcciones congeladas; blindaje absoluto que impide publicar imágenes locales no probadas; operación sin conexión asegurada tras la publicación; soporte extensible para gestores de paquetes modernos (`uv`, `cargo`, `go`, `composer`).
- **Negativas / Reducción de impacto:** La primera construcción de una combinación nueva toma entre 15 y 45 segundos durante la verificación del administrador, lo cual se reduce informando el avance por etapas en `EnvTestJob` y reutilizando el hash en futuras plantillas con los mismos componentes tecnológicos.

