# Feature: Motores de Base de Datos Gobernados por Catálogo y Auto-Orquestación Scale-to-Zero
Branch: main | Delivery strategy: ask-on-risk | Test runner: go test ./... & npm run test:ci

## Specs
- **S1**: **Gobernanza Abierta de Motores de Base de Datos (`layer: 'database'`) en Backend y Catálogo Institucional**:
  - En `backend/internal/core/domain/template_component.go` y `backend/internal/core/services/admin_governance_components.go`:
    - Permitir crear y actualizar componentes con `layer == "database"` validando que `service_engine` pertenezca al dominio soportado (`"none"`, `"sqlite"`, `"postgres"`, `"mysql"`, `"mariadb"`, `"mongodb"`, `"redis"`) o inferirlo automáticamente en `NormalizeCreateComponentDTO` a partir de `slug`/`name`/`icon_key`/`base_image`.
    - En `NormalizeCreateComponentDTO`: cuando `dto.Layer == ComponentLayerDatabase`, completar por defecto si vienen vacíos:
      - `CompatibleRuntimes`: `["universal"]`.
      - `SubCategory`: `"nosql"` para `mongodb`/`redis`, `"relational"` para `sqlite`/`postgres`/`mysql`/`mariadb`.
      - `BaseImage`: `"mysql:8.4"` (mysql), `"mariadb:11.4"` (mariadb), `"mongo:7.0"` (mongodb), `"redis:7.2-alpine"` (redis), `""` (sqlite/postgres).
      - `RamDeltaMB`: `0` (sqlite), `50` (postgres), `64` (redis), `128` (mysql/mariadb/mongodb).
  - En `backend/internal/core/services/ai_copilot_service.go` (`SuggestComponent`):
    - Ampliar la validación de `layer` para aceptar `"runtime"`, `"addon"` y `"database"`. Cuando `layer == "database"`, instruir al prompt y sanear en la respuesta `service_engine` (`sqlite`, `postgres`, `mysql`, `mariadb`, `mongodb`, `redis`), `base_image` con tag fijo (prohibido `:latest`) y `ram_delta_mb`.
  - Crear migración reversible `backend/migrations/00027_seed_managed_database_engines.sql` que asegure en `template_components` las semillas activas para `sqlite-3`, `postgres-16`, `mysql-84` (`base_image: 'mysql:8.4'`), `mariadb-11` (`base_image: 'mariadb:11.4'`), `mongodb-70` (`base_image: 'mongo:7.0'`) y `redis-72` (`base_image: 'redis:7.2-alpine'`).
- **S2**: **Capacidades de Nodo Auto-Orquestadas (`GetRuntimeCapabilities`) y Puertos de Dominio para Satélites Gestionados**:
  - En `backend/internal/core/domain/admin_governance.go` y `backend/internal/core/services/admin_governance_emergency.go` (`GetRuntimeCapabilities`):
    - Añadir en `SatelliteServiceCapability` el campo `ProvisioningMode string` (`json:"provisioning_mode"`): `"embedded"` (SQLite3), `"shared_host"` (PostgreSQL Aula) o `"managed_container"` (MySQL, MariaDB, MongoDB, Redis).
    - Con el motor Docker activo en el nodo, marcar `IsAvailable: true` para todos los motores soportados (`sqlite`, `postgres`, `mysql`, `mariadb`, `mongodb`, `redis`), eliminando el bloqueo que dependía de variables manuales `MYSQL_SATELLITE_HOST`, `MONGODB_SATELLITE_HOST` y `REDIS_SATELLITE_HOST` (las cuales quedan solo como override opcional si un sysadmin desea apuntar a un host externo).
  - En `backend/internal/core/domain/ports.go`:
    - Definir `ManagedSatelliteSpec` (`Engine`, `Image`, `ContainerName`, `VolumeName`, `MemoryLimitMB`, `Env`, `Cmd`, `ReadyCmd`) y `ManagedSatelliteEndpoint` (`Engine`, `Host`, `Port`, `ContainerID`).
    - Extender `WorkspaceOrchestrator` (o puerto específico inyectado en `WorkspaceService`) con `EnsureManagedSatelliteRunning(ctx context.Context, spec ManagedSatelliteSpec) (*ManagedSatelliteEndpoint, error)` y `StopManagedSatelliteIfIdle(ctx context.Context, engine string) error`.
- **S3**: **Auto-Orquestación Docker con Red Dedicada, Tuning Liviano, Aislamiento por Alumno y Scale-to-Zero**:
  - En `backend/internal/infrastructure/docker/`:
    - Implementar `EnsureManagedSatelliteRunning`:
      - Asegurar la red interna `solv-satellites-net` (bridge con resolución DNS interna para alcanzar `solv-sat-<engine>` sin habilitar comunicación cruzada entre contenedores de distintos alumnos en `solv-traefik-net`) y conectar el workspace del estudiante a dicha red cuando su plantilla requiera un satélite `managed_container`.
      - Asegurar el volumen nombrado persistente `solv_sat_data_<engine>` y el contenedor `solv-sat-<engine>` con límites de memoria cgroups (`256 MB` para MySQL/MariaDB/MongoDB, `96 MB` para Redis) y `no-new-privileges:true`.
      - Aplicar flags de bajo consumo de RAM:
        - MySQL/MariaDB: `--innodb-buffer-pool-size=64M --performance-schema=OFF --max-connections=100`
        - MongoDB: `--wiredTigerCacheSizeGB 0.25`
        - Redis: `redis-server --maxmemory 64mb --maxmemory-policy allkeys-lru --appendonly yes`
      - Ejecutar healthcheck determinista (backoff hasta 30 s) antes de retornar el endpoint.
  - En `backend/internal/core/services/workspace_service.go`, `workspace_lifecycle.go` y `workspace_satellite.go`:
    - Generalizar `ensureExerciseSatelliteDB` para detectar cualquier motor declarado en `template.ServicesConfig.Services` (`postgres`, `mysql`, `mariadb`, `mongodb`, `redis`):
      - Para **`postgres`**: mantener el aprovisionamiento actual en `s.db` con usuario aislado o base `solv_std_<student>_<subject>` y aplicación idempotente de `db_init_script`.
      - Para **`mysql` / `mariadb`**: asegurar el satélite `solv-sat-mysql` / `solv-sat-mariadb`, crear la base aislada `solv_std_<student>_<subject>` y usuario exclusivo con contraseña determinista HMAC-SHA256 (`SOLV_SATELLITE_SECRET`), aplicar `db_init_script` de forma idempotente (tabla `_solv_exercise_migrations`) e inyectar `DATABASE_URL=mysql://<user>:<pass>@<host>:3306/<db>`, `MYSQL_HOST`, `MYSQL_PORT=3306`, `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_DATABASE`.
      - Para **`mongodb`**: asegurar el satélite `solv-sat-mongodb` e inyectar `MONGODB_URI=mongodb://<host>:27017/solv_std_<student>_<subject>`.
      - Para **`redis`**: asegurar el satélite `solv-sat-redis` e inyectar `REDIS_URL=redis://<host>:6379/0` y `REDIS_KEY_PREFIX=solv_std_<student>_<subject>:`.
    - **Scale-to-Zero**: en `HibernateWorkspace`, `PauseWorkspace`, `TerminateWorkspace` y en el ciclo de `QoSOrchestratorWorker`, verificar si quedan workspaces en estado `running` que utilicen cada motor `managed_container`; cuando el conteo llegue a `0`, detener el contenedor `solv-sat-<engine>` (preservando su volumen `solv_sat_data_<engine>`) para liberar el 100% de su RAM en el host.
- **S4**: **Tests Unitarios de Auto-Orquestación, Aislamiento y Documentación Técnica (ADR-046 + API_CONTRACTS.md)**:
  - Añadir tests unitarios en `backend/internal/core/services/` verificando:
    1. Creación de componentes `layer: "database"` y sugerencia con IA (`SuggestComponent`).
    2. `GetRuntimeCapabilities` reportando todos los motores disponibles con su `provisioning_mode`.
    3. `StartWorkspaceForExercise` aprovisionando e inyectando credenciales aisladas para `postgres`, `mysql`, `mongodb` y `redis`.
    4. Apagado automático (`Scale-to-Zero`) del satélite cuando el último workspace activo que lo requiere pasa a `hibernated` o `failed`.
  - Actualizar `docs/ARQUITECTURA/ADR/ADR-046-estudio-plantillas-componentes-construccion-local.md` y `docs/API_CONTRACTS.md`.
- **S5**: **Desbloqueo de Columna 3 (`+ Nuevo`), Clasificación `Motor de Base de Datos` y Tarjetas 1-Clic en Frontend (Antigravity)**:
  - En `component-catalog.presets.ts`, `component-form-modal.component.*`, `template-studio.component.*` y `template-studio.utils.ts`:
    - Ampliar `ComponentCatalogKind` a `'backend' | 'web' | 'addon' | 'database'` y agregar el 4to botón **`Motor de Base de Datos`** en la barra segmentada `#comp-layer.catalog-kind-segmented`.
    - Incorporar `OFFICIAL_DATABASE_PROFILES` (`SQLite 3`, `PostgreSQL`, `MySQL`, `MariaDB`, `MongoDB`, `Redis`) en el `Catálogo Oficial (1 clic)` con selector de versión e inferencia de `service_engine`, `base_image` y huella de memoria.
    - Agregar el botón **`+ Nuevo`** en la cabecera de la **Columna 3 (Base de Datos)** de `template-studio.component.html` que abre el modal pre-posicionado en `database` (`openCreateComponentModal('database')`).
    - Mostrar en las tarjetas de la Columna 3 el badge de modo de aprovisionamiento (`En archivo +0 MB`, `Aula compartida ~50 MB`, `Auto-orquestado Scale-to-Zero`) y asegurar que todos los tests de frontend (`npm run lint:styles && npm run test:ci && CI=true npm run build`) pasen en verde.

## Tasks
- [x] **T1** (S1, S2) [delegated: Freebuff]: Habilitar `layer: 'database'` en `CreateTemplateComponent`, `NormalizeCreateComponentDTO` y `AICopilotService.SuggestComponent`, añadir migración `00027_seed_managed_database_engines.sql` y exponer `provisioning_mode` con disponibilidad activa en `GetRuntimeCapabilities` (`cd backend && go vet ./... && go test ./internal/... -count=1` OK).
- [x] **T2** (S3, S4) [delegated: Freebuff]: Implementar el orquestador Docker de satélites bajo demanda (`solv-sat-<engine>` en `solv-satellites-net`, tuning de RAM, aislamiento por alumno HMAC-SHA256, `db_init_script` multi-motor y apagado automático Scale-to-Zero), tests unitarios en Go y actualización de `ADR-046` + `API_CONTRACTS.md` (`cd backend && go vet ./... && go test ./internal/... -count=1` OK).
- [x] **T3** (S5) [delegated: Freebuff]: Habilitar la opción `Motor de Base de Datos` y `OFFICIAL_DATABASE_PROFILES` en `component-form-modal`, agregar el botón `+ Nuevo` y unión catálogo+capabilities en la Columna 3 de `template-studio`, y actualizar `.spec.ts` verificando `npm run lint:styles`, `npm run test:ci` y `CI=true npm run build` (`cd frontend && npm run lint:styles && npm run test:ci && CI=true npm run build` OK).

## Log
- **L1 (2026-10-27)**: "me parece super bueno, entonces podemos aplciarlo asi, puedes tu crear el odd para abrir esta tarea para que asi frebuff lo pueda aplicar, y tambien como hacemos siempre, puedes reevaluar tu propuesta si es la mejor, o hay puntitos mejorables, talves pqueños, grandes, criticos y demas"
- **L2 (2026-10-27)**: Freebuff implementó `T1` y `T2` (`S1`..`S4`). Verificado en el repositorio: `cd backend && go vet ./... && go test ./internal/... -count=1` pasa 100% en verde. El usuario indicó que Freebuff también ejecutará `T3` (`S5` Frontend).
- **L3 (2026-10-27)**: Freebuff implementó `T3` (`S5` Frontend) y se verificó integralmente: `cd backend && go vet ./... && go test ./internal/... -count=1` y `cd frontend && npm run lint:styles && npm run test:ci && CI=true npm run build` pasan en verde. Se separaron los commits por especificación (`S1`–`S5` + doc), sin footer `Slice` (el trabajo no pertenece a ningún slice) y sin renombrar los identificadores internos (se omitió el término "satélite" únicamente en los mensajes de commit).
