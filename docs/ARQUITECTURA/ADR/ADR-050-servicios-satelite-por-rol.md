# ADR-050: Servicios Satélite Declarativos por Rol, Pin del Entorno y Contratos de Capacidad y Aislamiento

- **Estado:** Aprobado (definición; implementación por fases)
- **Fecha:** 2026-10-29
- **Contexto:** Core / Gestión de Entornos, Plantillas y Servicios Satélite

## Contexto y Problema

El ADR-046 dejó un único motor satélite por plantilla y una columna «Base de Datos» de selección excluyente. La consulta de por qué Redis y PostgreSQL no pueden convivir destapó que el límite no era de interfaz, sino de contrato, y que alrededor había cinco problemas estructurales:

1. **Un solo motor en todo el camino.** El Studio guarda un único `selectedDatabaseId` ([template-studio.component.ts:168](../../../frontend/src/app/features/admin/templates/studio/template-studio.component.ts#L168)), el payload construye como máximo un `ServiceRequirement` ([template-studio.component.ts:1219](../../../frontend/src/app/features/admin/templates/studio/template-studio.component.ts#L1219)), el compositor resuelve un solo `cfg.DatabaseID` ([template_composer_service.go:176](../../../backend/internal/core/services/template_composer_service.go#L176)), el aprovisionador toma el primer motor declarado ([workspace_satellite.go:46](../../../backend/internal/core/services/workspace_satellite.go#L46)) y el scale-to-zero resuelve un motor por laboratorio ([workspace_satellite.go:382](../../../backend/internal/core/services/workspace_satellite.go#L382)). El modelo `services_config.services` ya es una lista, pero nadie la recorre completa.

2. **La memoria de servicios compartidos se cobra a cada alumno.** Los satélites son un contenedor por nodo, no por alumno. Sin embargo, el Studio suma `db.ram_delta_mb` a la cuota de cada estudiante ([template-studio.utils.ts:679](../../../frontend/src/app/features/admin/templates/studio/template-studio.utils.ts#L679)), el backend repite el cálculo con una tabla de megabytes hardcodeada ([admin_governance_templates.go:85](../../../backend/internal/core/services/admin_governance_templates.go#L85)) y la admisión solo suma `ws.MemoryLimitMB` ([workspace_service.go:349](../../../backend/internal/core/services/workspace_service.go#L349)): el cgroup del satélite no se reserva en ninguna parte. Además hay tres números para lo mismo: el catálogo declara 128 MB, el piso vital usa 128 MB y el contenedor real tiene techo de 256 MB ([workspace_satellite.go:149](../../../backend/internal/core/services/workspace_satellite.go#L149)). Con 30 alumnos, 128 MB por cabeza reservan 3.84 GB por un contenedor de 256 MB.

3. **No hay reproducibilidad del entorno de un ejercicio.** El ejercicio apunta a una plantilla que se actualiza in situ ([template_components_repository.go:403](../../../backend/internal/infrastructure/storage/postgres/template_components_repository.go#L403)); el `TemplateSummary` es un resumen transitorio que se arma desde la plantilla viva ([evaluation_exercise_crud.go:129](../../../backend/internal/core/services/evaluation_exercise_crud.go#L129)) y el `ex.MemoryLimitMB` que se guarda al crear el ejercicio se ignora al iniciar el laboratorio ([workspace_service.go:290](../../../backend/internal/core/services/workspace_service.go#L290)). Editar una plantilla cambia imagen, RAM y motor de ejercicios ya publicados.

4. **El aislamiento es por convención, no por credenciales.** (Corregido para PostgreSQL en [ADR-051](ADR-051-postgresql-aula-como-satelite-dedicado.md).) PostgreSQL de aula inyectaba `postgres:postgres` como superusuario ([workspace_service.go:427](../../../backend/internal/core/services/workspace_service.go#L427)); el compose de referencia define `solv_db` con `solv_user/solv_password` y no existe `solv-postgres`, así que esa ruta o no conecta o apunta a un contenedor con credenciales por defecto. MongoDB se expone sin autenticación y Redis comparte el DB 0 con un `REDIS_KEY_PREFIX` que el cliente puede ignorar. Solo MySQL/MariaDB crea base y usuario por alumno, y ninguna derivación incluye tenant.

5. **El catálogo no gobierna la ejecución.** `managedSatelliteSpec` hardcodea imágenes, techos de memoria y comandos de readiness, mientras el catálogo publica `DefaultImage`, `version` y `base_ram_mb` que el aprovisionador no lee. El `db_init_script` se aplica solo en MySQL/MariaDB y PostgreSQL; MongoDB y Redis lo ignoran en silencio ([workspace_satellite.go:268](../../../backend/internal/core/services/workspace_satellite.go#L268)).

## Decisión

### D1 — El rol es el contrato; el motor es la implementación

Una plantilla declara **bindings de servicio**, cada uno con un rol:

| Rol | Cardinalidad | Motores iniciales |
|---|---|---|
| `primary_database` | 0..1 | `sqlite`, `postgres`, `mysql`, `mariadb`, `mongodb` |
| `cache` | 0..1 | `redis` |
| `broker`, `object_storage`, `search` | 0..1 | reservados para una fase posterior |

```
ServiceBinding {
  role:         primary_database | cache | ...
  engine:       postgres | mysql | mariadb | mongodb | redis | sqlite
  component_id: uuid del catálogo (trazabilidad)
  version:      versión pedida (vacío = la del adaptador)
  env_alias:    alias estable para el laboratorio ("db", "cache")
}
```

- **Invariantes de dominio:** como máximo una `primary_database`, un binding por motor, rol coherente con el motor (`redis → cache`, el resto → `primary_database`) y `env_alias` único. MySQL y PostgreSQL no pueden convivir porque ambos son primarios; PostgreSQL y Redis sí, porque son roles distintos.
- **Política, no dominio:** cuántos auxiliares se permiten por rol. Hoy el Studio ofrece uno; el modelo admite más sin cambiar el dominio.
- `category` de `ServiceRequirement` queda como alias de lectura de `role`; no se agrega un tercer nombre.
- La validación vive en `core/domain` y se ejecuta en `CreateOfficialTemplate` y `UpdateOfficialTemplate`. Hoy no existe: la API acepta cualquier `services_config`, incluidos motores desconocidos y duplicados.

### D2 — Catálogo y adaptador: una sola fuente por responsabilidad

| Capa | Decide | No decide |
|---|---|---|
| Catálogo (`template_components`) | Qué motores se ofrecen, su rol, disponibilidad y metadata pedagógica | Imagen efectiva, límites, aislamiento |
| Adaptador de motor (registro compilado en Go) | Imagen y versión, reserva y techo de RAM, readiness, aislamiento, tipo de init, identidad del satélite, variables de entorno | Qué se muestra en el catálogo |
| Binding de plantilla | Rol, motor, alias y versión pedida | Cómo se aprovisiona |

El adaptador es la única fuente de ejecución. Un registro compilado de adaptadores es el tamaño correcto para seis motores en un nodo; un plugin cargado en runtime o un service mesh agregan modos de fallo sin resolver ningún problema de este despliegue.

### D3 — Resolución y aprovisionamiento de N bindings

- `ResolveTemplateBindings(template) []ResolvedService` valida y ordena; reemplaza a `templateSatelliteEngine`, que hoy devuelve el primero.
- El arranque asegura **todos** los satélites, espera readiness de todos, inyecta el entorno de todos y conecta el laboratorio a `solv-satellites-net` si al menos uno es gestionado. Si uno falla, se aborta el arranque completo; los contenedores y volúmenes compartidos no se destruyen.
- El scale-to-zero cuenta el conjunto de motores por laboratorio y lleva refcount por identidad de satélite.
- La identidad del satélite pasa a `(engine, major, tuning_profile)` (`solv-sat-<engine>-<major>`). Se implementa recién cuando el adaptador sea la fuente de la imagen y los límites (fase 7); antes sería una segunda fuente de verdad.

### D4 — Pin del entorno al publicar

- Al publicar un ejercicio se persiste `environment_snapshot` (JSONB) con: `template_id`, imagen, entrypoint, timeout, `base_ram_mb`, bindings resueltos, contrato de init y `verification_fingerprint`.
- El workspace lee el pin. Para ejercicios anteriores al corte, resuelve la plantilla viva y registra el fallback.
- Editar una plantilla solo afecta publicaciones nuevas; la UI avisa cuántos ejercicios publicados la usan.
- **Alternativa descartada:** versionado completo de plantillas. Resuelve lo mismo con más superficie (nuevas versiones, rutas, UI de comparación) y no es necesario para este corte.

### D5 — Capacidad: reserva del nodo separada de la cuota del alumno

- Cada adaptador declara `expected_mb` (lo que se muestra al docente) y `reserve_mb` (techo de cgroups que la admisión descuenta).
- La cuota del laboratorio incluye solo los procesos del alumno (editor, runtime, web, complementos, clientes). Los satélites se reservan **una vez por identidad** en el nodo.
- El piso vital y el Studio usan los valores del adaptador; se eliminan las tablas de megabytes hardcodeadas.
- Capacidad del aula: `(allocatable_mb − Σ reserve_mb) / per_lab_quota_mb`, con el techo de concurrencia del tenant como límite adicional.
- La interfaz rotula el costo del satélite como del **nodo**, no por alumno, y distingue «sin laboratorios activos» de «reservado».

### D6 — Aislamiento por motor y por scope

El scope de toda credencial y nombre es `tenant + alumno + materia + alias`:

| Motor | Contrato |
|---|---|
| PostgreSQL | Rol y base por scope; `REVOKE CONNECT ... FROM PUBLIC`; sin superusuario en el laboratorio; DSN administrativo separado del de la aplicación |
| MySQL / MariaDB | Base y usuario por scope (ya existe); derivación con tenant, rol y motor |
| MongoDB | `--auth`; usuario con `readWrite` solo sobre su base |
| Redis | ACL por usuario con patrón de claves del scope y comandos peligrosos denegados (`flushall`, `flushdb`, `config`, `shutdown`) |
| SQLite | Archivo dentro del laboratorio |

Si un motor no se puede aislar con su adaptador, el binding se rechaza en la validación. No se publica una plantilla con aislamiento por convención.

### D7 — Init declarado por el adaptador

- El adaptador declara `init_kind`: `sql`, `mongo-js`, `redis-cmds` o `none`.
- Un `db_init_script` en un motor sin init se rechaza al guardar el ejercicio; para este corte el init aplica al primario (SQL o Mongo) y Redis no lo usa.
- El sellado de aplicación es por `(exercise_id, service_alias)` en cada motor, no por ejercicio en un solo motor. Mientras el único motor con init sea el primario, el alias es constante (`db`) y el sello por `exercise_id` es equivalente.

### D8 — Alcance del juez

Los bindings aplican a `IDE_PERSISTENTE`. `JUEZ_EFIMERO` y `AMBOS` siguen sin servicios y sin red. Un juez que necesite base de datos es otro diseño (satélite efímero por corrida o red acotada) y otro ADR.

## Consecuencias

- **Positivas:** «PostgreSQL + Redis» se puede expresar; la capacidad del nodo refleja lo que realmente se reserva; un ejercicio publicado corre igual en marzo y en junio; el aislamiento se verifica con credenciales; agregar un motor nuevo es un adaptador, no una cadena de `switch`.
- **Negativas / impacto esperado:** la identidad versionada requiere migrar nombres de contenedor y volúmenes; cambiar la derivación de credenciales rota las claves existentes (el reaprovisionamiento reescribe el usuario, pero hay que probarlo con datos); el primer arranque de cada motor sigue costando pull y readiness; la superficie de tests crece con cada adaptador.
- **Neutras:** `services_config` sigue en JSON; `database_id` queda como alias heredado; no se crea una tabla nueva de bindings, porque la plantilla publicada es un artefacto y no una referencia viva al catálogo.

## Relación con Otros ADRs

- **ADR-046 §7:** reemplazada por D1–D7 en lo relativo al motor único y al aislamiento por prefijo. El resto del ADR (contenedores, red interna, scale-to-zero, construcción local) sigue vigente.
- **ADR-045:** el catálogo de satélites sigue siendo la fuente de disponibilidad y de versión detectada en vivo.
- **ADR-008 / ADR-043:** la admisión y el QoS se ajustan a la separación entre cuota del laboratorio y reserva del nodo.
- **ADR-024:** el scope de aislamiento incorpora el tenant.
- **ADR-030:** la aprobación de plantillas incorpora la validación de bindings.
- **ADR-049:** la telemetría de capacidad debe exponer la reserva de satélites; queda como trabajo posterior.

## Fases de Implementación

| Fase | Alcance | Estado |
|---|---|---|
| 1 | Contrato de bindings, roles, invariantes y validación server-side | Implementado |
| 2 | Capacidad: reserva del nodo separada de la cuota del alumno y fórmula de admisión | Implementado |
| 3 | Pin del entorno al publicar y lectura en el arranque del workspace | Pendiente |
| 4 | Adaptadores, pipeline de N bindings, init por motor y scale-to-zero por conjunto | Implementado |
| 5 | Aislamiento por motor y scope con tenant | Pendiente |
| 6 | Studio (base principal + auxiliares), copiloto y modal docente | Pendiente |
| 7 | Identidad versionada del satélite y catálogo como fuente de la imagen | Diferida |

El orden no es negociable en dos puntos: la fase 5 no puede quedar después de la 6 (no se habilita Redis auxiliar sin ACL), y la fase 2 no podía quedar después de la 4 (aprovisionar dos motores con la cuenta de memoria anterior empeoraba la admisión).

La fase 1 y la fase 4 están acopladas: con el pipeline aprovisionando varios bindings, la validación deja de ser un tope de un servicio y pasa a exigir el contrato (un primario, rol coherente, alias único). Ambas se implementaron juntas: `NormalizeServiceBindings` en `core/domain` se ejecuta en el alta y en la edición de plantillas, y el arranque del laboratorio recorre todos los bindings antes de inyectar el entorno.

### Detalle de lo implementado en las fases 1 y 4

- **Contrato:** `ServiceRequirement.Role` con `primary_database | cache`; `category` queda como alias derivado del rol y `env_alias` por defecto `db` / `cache`. La validación rechaza motor desconocido, dos primarios, motor repetido, rol incoherente y alias inválido o repetido.
- **Pipeline:** `templateServiceBindings` resuelve y ordena (primario primero), `ensureExerciseSatelliteDB` asegura todos los bindings, mezcla el entorno de cada uno y solo conecta el laboratorio a `solv-satellites-net` si al menos uno es un contenedor gestionado. Si un motor falla, el arranque se aborta.
- **Init por motor:** SQL (PostgreSQL/MySQL/MariaDB) usa `_solv_exercise_migrations`; MongoDB aplica el script como JavaScript de mongosh con el mismo guard y sellado, contra la base aislada del alumno. Redis no usa `db_init_script` y un primario SQLite tampoco: el ejercicio se rechaza al guardarse (`ErrDBInitScriptUnsupported`) en lugar de ignorar el script al arrancar.
- **Scale-to-zero:** el conjunto de motores de cada laboratorio se resuelve desde su plantilla y se lleva refcount por motor; un motor se apaga solo cuando ningún laboratorio lo usa.
- **Host externo:** si el sysadmin declara `*_SATELLITE_HOST`, SOLV no levanta el contenedor ni intenta aprovisionar dentro de él; solo inyecta credenciales hacia ese destino.
- **Adaptador de motor:** la capa de ejecución por motor queda en `managedSatelliteSpec` (imagen, límites, readiness) y `InitKind` (contrato de init). El catálogo como fuente de la imagen y la identidad versionada del satélite siguen en la fase 7.

### Detalle de lo implementado en la fase 2

- **La cuota del laboratorio deja de pagar el satélite:** `calculateMinimumVitalFloorMB` ya no suma +50/+128/+64 por motor y el Studio dejó de sumar `ram_delta_mb` al `base_ram_mb` del laboratorio. El piso queda en 210 MB para IDE y 64 MB para juez; el cliente de base de datos entra en el margen de procesos del alumno.
- **Reserva del nodo:** `nodeSatelliteReserveMB` suma el techo de cgroup de cada motor gestionado una sola vez, con la unión de los motores de los laboratorios activos y los de la plantilla solicitada. `managedSatelliteSpec.MemoryLimitMB` es la fuente del techo (MySQL, MariaDB y MongoDB 256 MB; Redis 96 MB; PostgreSQL de aula 384 MB desde [ADR-051](ADR-051-postgresql-aula-como-satelite-dedicado.md), que lo saca de la instancia de plataforma).
- **Admisión:** `admissionBudgetDenied` compara `committed + lab + reserve` contra `host × ceiling%`. La reserva se cuenta una vez por motor, no por alumno. Si no se puede resolver la plantilla de un laboratorio activo se reserva el conjunto completo de motores, para no admitir de más; un motor con `*_SATELLITE_HOST` externo no reserva memoria del nodo.
- **Interfaz:** la tarjeta del motor rotula su costo como del nodo («nodo ~50 MB») y el resumen de capacidad por alumno ya no incluye ese delta.
- **Tests:** `TestNodeSatelliteReserveMB_CountsEachEngineOnce` fija que la reserva es la misma con 2 que con 30 laboratorios, y `TestAdmissionBudgetDenied` cubre el alumno 30 y el caso que la cuenta anterior negaba.
- **Pendiente de la fase 2:** `expected_mb` sigue viniendo del catálogo (`ram_delta_mb`) y no del adaptador; unificarlo y exponer la reserva en la telemetría de capacidad corresponde a la fase 7 y a ADR-049.

Pendiente fuera de este corte: aislamiento por credenciales con tenant (fase 5), pin del entorno al publicar (fase 3), multi-selección en el Studio (fase 6) e identidad versionada del satélite (fase 7).

## Alternativas Descartadas

- **Multi-selección libre en la Columna 3.** Permite MySQL + PostgreSQL y no cambia el contrato ni el pipeline: es el parche que este ADR evita.
- **Tabla normalizada de bindings.** La plantilla es un artefacto inmutable; el JSON alcanza y evita una FK que se rompe cuando un componente del catálogo se desactiva.
- **Un contenedor de motor por laboratorio.** 30 alumnos × 256 MB no entran en el nodo; el aislamiento tiene que vivir dentro del motor.
- **Manifiesto `.solv/services.json`.** Las variables por binding alcanzan; un archivo nuevo es un contrato más que versionar y testear sin caso de uso.
- **Plugin runtime / service mesh.** Sobre-ingeniería para un nodo y seis motores.
