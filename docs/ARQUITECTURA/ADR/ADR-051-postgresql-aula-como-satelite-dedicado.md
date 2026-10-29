# ADR-051: PostgreSQL de Aula como Satélite Dedicado y Separación del Motor de Plataforma

- **Estado:** Aprobado (implementado)
- **Fecha:** 2026-10-29
- **Contexto:** Core / Servicios Satélite, Capacidad y Aislamiento

## Contexto y Problema

El «PostgreSQL de aula» no era un servicio del aula: era la misma instancia que sostiene la plataforma. El camino completo lo confirma:

1. `ensureSatellitePostgresDatabase(ctx, s.db, dbName)` creaba la base del alumno con el pool de `DATABASE_URL` ([workspace_service.go](../../../backend/internal/core/services/workspace_service.go)), es decir la base de usuarios, sesiones y plantillas.
2. El laboratorio recibía `postgresql://postgres:postgres@<host>:5432/<db>` ([workspace_service.go](../../../backend/internal/core/services/workspace_service.go), [workspace_lifecycle.go](../../../backend/internal/core/services/workspace_lifecycle.go)).
3. Ese `<host>` salía de `getPostgresContainerHost()`: `host.docker.internal` o `solv-postgres`. El compose define el contenedor como `solv_db` y el usuario como `solv_user`; ninguno de los dos coincide con lo inyectado.
4. El laboratorio se conecta a `solv-traefik-net` y solo se une a `solv-satellites-net` cuando usa un motor gestionado. `solv_db` vive en `solv_net`. No hay ruta por nombre.
5. `applyExerciseDBInitScript` ejecutaba el SQL del ejercicio con el `DATABASE_URL` del backend ([workspace_exercise_db.go](../../../backend/internal/core/services/workspace_exercise_db.go), eliminado en esta implementación), así que el script también corría en la instancia de plataforma.

El resultado es un defecto doble. Por un lado, cualquier consulta de un alumno compite por CPU, conexiones y memoria con el control plane; un `OOM` del motor tumba la plataforma, no el laboratorio. Por otro, el DSN del aula probablemente no conecta: nombre y credenciales son inventados. Y la memoria del motor no está en ninguna cuenta, porque no pasa por el orquestador de satélites.

## Decisión

### D1 — Dos planos, dos motores

`solv_db` queda como **control plane** de SOLV: usuarios, sesiones, plantillas, workspaces y auditoría. El **data plane** de las prácticas vive en `solv-sat-postgres`, un satélite gestionado con ciclo de vida propio, volumen propio y credenciales por alumno. Ningún laboratorio recibe credenciales del control plane.

### D2 — El adaptador es la única fuente de ejecución

`managedSatelliteSpec("postgres")` declara:

| Campo | Valor |
|---|---|
| Imagen | `postgres:18-alpine` |
| Contenedor | `solv-sat-postgres` |
| Volumen | `solv_sat_data_postgres` |
| Ruta de datos | Volumen en `/var/lib/postgresql` (layout oficial de la 18+) y `PGDATA=/var/lib/postgresql/18/docker` explícito para no depender del default de la imagen |
| Red | `solv-satellites-net` |
| Puerto | 5432 |
| Memoria | 384 MB de techo de cgroup |
| Sintonización | `shared_buffers=96MB`, `max_connections=100`, `work_mem=2MB`, `maintenance_work_mem=32MB` |
| Readiness | `pg_isready -h 127.0.0.1 -U postgres` |

La contraseña del superusuario se deriva con `satelliteRootPassword()` y nunca se inyecta al laboratorio. El superusuario solo se usa con `docker exec` desde el backend.

384 MB y no 256: `shared_buffers` de 96 más 100 conexiones vivas no entran en 256. Es una estimación de diseño; se ajusta con la telemetría real de `docker stats` del nodo y, si sobra, baja.

### D3 — Identidad por alumno, sin superusuario en el laboratorio

- Base `solv_std_<alumno>_<materia>` y rol `u_<hmac>` con contraseña derivada de `SOLV_SATELLITE_SECRET`.
- `CREATE ROLE ... NOSUPERUSER NOCREATEDB NOCREATEROLE`.
- `CREATE DATABASE ... OWNER <rol>` y `REVOKE CONNECT ... FROM PUBLIC`, con `GRANT CONNECT` solo al rol dueño.
- El laboratorio recibe `DATABASE_URL`, `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD` y `PGDATABASE` con esas credenciales, apuntando a `solv-sat-postgres`.
- MySQL y MariaDB ya siguen este contrato; MongoDB y Redis lo completan en la fase 5 del ADR-050.

### D4 — Init declarado, ejecutado en la base del alumno

El `db_init_script` SQL se aplica con `psql` dentro del contenedor, contra la base del alumno, con `ON_ERROR_STOP=1` y el mismo guard que el resto de los motores:

1. `CREATE TABLE IF NOT EXISTS _solv_exercise_migrations (...)`.
2. Consulta por `exercise_id`; si ya está, no hace nada.
3. Ejecuta el script; si falla, no sella y se reintenta en el próximo arranque.
4. Inserta el sello.

Se retira `applyExerciseDBInitScript` del camino de laboratorios: usaba el `DATABASE_URL` del backend y por lo tanto la instancia de plataforma.

### D5 — Capacidad: una reserva por motor, la plataforma fuera de la cuenta

`postgres` entra en `managedSatelliteEngines` y suma 384 MB **una sola vez por nodo**, con la misma fórmula de admisión del ADR-050. `solv_db` deja de participar de la cuenta de laboratorios: es costo de plataforma, no de práctica. El camino heredado (un ejercicio con `db_init_script` y una plantilla sin servicios declarados) sigue levantando el satélite y cuenta la reserva.

### D6 — Ciclo de vida y fallo

- Scale-to-zero con el refcount existente: el motor se apaga cuando ningún laboratorio activo lo usa; el volumen sobrevive.
- El barrido periódico respeta un margen de gracia de dos minutos desde el último uso del motor. Un laboratorio que se está aprovisionando todavía no figura como activo en la base de datos, y sin ese margen el motor podía apagarse a mitad del arranque. La hibernación y la terminación explícitas apagan en el acto: ahí no hay carrera.
- El contenedor declara `SIGINT` como señal de parada. PostgreSQL interpreta `SIGTERM` como apagado "smart" (espera a los clientes), así que con la señal por defecto el stop terminaba en `SIGKILL` y salida 137; con `SIGINT` el apagado es "fast" y sale con código 0.
- Si el satélite no arranca o `psql` falla, el arranque del laboratorio se aborta. No se publica un laboratorio con una base rota.
- El control plane no depende del satélite: si `solv-sat-postgres` se cae, la plataforma sigue funcionando.

### D7 — Datos, respaldo y corte

- Volumen nombrado `solv_sat_data_postgres`; detener el contenedor nunca borra datos.
- `backup.sh` respalda el satélite con `pg_dumpall -U postgres`; si está detenido por scale-to-zero, lo arranca, respalda y lo deja como estaba.
- Las bases que hoy viven en `solv_db` **no se migran automáticamente**. El `db_init_script` se reaplica al arrancar, pero lo que el alumno haya creado se pierde si no se migra. El corte se hace con backup previo y, si hay datos que conservar, `pg_dump` por base y `psql` en el contenedor nuevo. No se borra nada de `solv_db` automáticamente.
- Multi-tenant en el nombre de la base queda para la fase 5, porque la derivación es compartida con MySQL, MariaDB, MongoDB y Redis.

### D8 — Red: se hereda el contrato actual, con una deuda explícita

El laboratorio se une a `solv-satellites-net` y resuelve el motor por DNS, igual que los otros cuatro. Esa red es un bridge común: un laboratorio conectado ahí puede alcanzar a otro laboratorio conectado ahí, no solo al motor. El problema ya existe para MySQL, MariaDB, MongoDB y Redis; PostgreSQL lo vuelve mayoritario. Se aborda en la tarea T6 con una medición previa (`docker network inspect`) y una decisión entre red por laboratorio o broker; no se mezcla con este ADR.

## Alternativas Descartadas

- **Poner un `mem_limit` a `solv_db` y seguir usándolo para prácticas.** Protege el nodo a costa de ahogar el control plane; no separa el riesgo, lo comparte.
- **Un contenedor de PostgreSQL fijo en el compose.** Aísla el control plane, pero paga 384 MB siempre encendidos, agrega un servicio estático que operar y necesita un DSN administrativo en la configuración del backend.
- **Un PostgreSQL por laboratorio.** 30 × ~100 MB no entran y rompe el scale-to-zero.
- **Solo esquemas separados en `solv_db`.** Mismo motor, mismas conexiones, mismo `OOM`: no resuelve nada.
- **PostgreSQL externo obligatorio (`POSTGRES_SATELLITE_HOST`).** Válido para un despliegue con un motor gestionado fuera del nodo, pero no puede ser el default de un on-premise de un solo servidor.

## Consecuencias

- **Positivas:** el control plane deja de compartir destino con las prácticas; el DSN del aula apunta a un contenedor que existe y con credenciales reales; la memoria del motor entra en la admisión; el volumen y el respaldo quedan bajo el mismo patrón que los otros motores; Postgres gana la identidad por alumno de la fase 5.
- **Negativas / impacto esperado:** el primer arranque de cada periodo paga pull y readiness; el corte no migra datos; el respaldo necesita un paso nuevo; 384 MB es una estimación que hay que medir; el aula deja de estar disponible si el satélite se cae, a cambio de que la plataforma siga viva.
- **Neutras:** `solv_db` sigue igual; el compose no cambia; `services_config` no cambia.

## Relación con Otros ADRs

- **ADR-050 D3/D6:** el pipeline de N bindings y el aislamiento por credenciales se extienden a PostgreSQL; la fase 5 deja pendientes MongoDB y Redis.
- **ADR-046:** el contenedor satélite y el scale-to-zero siguen siendo el mecanismo; este ADR corrige el motor que se había quedado fuera.
- **ADR-035:** el respaldo incorpora el `pg_dumpall` del satélite.
- **ADR-045:** el catálogo deja de describir PostgreSQL como `shared_host`; pasa a `managed_container`.
- **ADR-049:** la telemetría de capacidad debe incluir la reserva de 384 MB cuando se exponga la reserva de satélites.

## Plan de Implementación

| Tarea | Alcance | Verificación | Estado |
|---|---|---|---|
| T2.1 | Adaptador: imagen, contenedor, volumen, `PGDATA`, memoria, sintonización y readiness | `TestStartWorkspaceForExercise_ProvisionsManagedSatelliteEngines/postgres` | Hecho |
| T2.2 | Rol, base y DSN por alumno; retirar `s.db` del camino de laboratorios | Test de aprovisionamiento sin pool de plataforma (`WithProvisioning` ya no recibe `*sqlx.DB`) | Hecho |
| T2.3 | Init SQL por `psql`, camino heredado y reserva de 384 MB | Test de guard/sellado y de reserva con 2 y 30 alumnos | Hecho |
| T2.4 | Capacidades y Studio: PostgreSQL `managed_container`, migración del catálogo | `TestGetRuntimeCapabilities_AllEnginesAvailableWithProvisioningMode` y migración `00031` | Hecho |
| T2.5 | Backup del satélite y documento de corte | `pg_dumpall` con arranque temporal en [backup.sh](../../../infra/scripts/backup.sh) | Hecho |
| T2.6 | Tests de regresión y actualización de ADR-050 | `go build ./...`, `go vet ./...`, `go test ./...`, `ng build`, 802 tests de Angular, `lint:styles`, `eslint` | Hecho |
| T2.7 | Hallazgos del nodo: filtro de satélites en el colector de zombies, gracia del scale-to-zero y `SIGINT` | Tests unitarios del filtro y del margen; prueba real en el nodo | Hecho |

### Corte de datos

Antes de aplicar el cambio en un nodo con datos:

```sql
-- En la instancia de plataforma: inventario de bases de aula existentes.
SELECT datname FROM pg_database WHERE datname LIKE 'solv_std_%';
```

```bash
# Respaldo previo de la instancia de plataforma.
docker exec solv_db pg_dumpall -U solv_user > respaldo_previo.sql

# Migración opcional, base por base, hacia el satélite nuevo.
docker exec solv_db pg_dump -U solv_user -d solv_std_xxxxxxxx_yyyyyyyy > aula.sql
docker exec -i solv-sat-postgres psql -U postgres -d solv_std_xxxxxxxx_yyyyyyyy < aula.sql
```

Sin ese paso, el satélite arranca vacío y el `db_init_script` reaplica el esquema del ejercicio. No se elimina nada de `solv_db`.

### Verificación ejecutada

Sobre el nodo de desarrollo, con el daemon de Docker real y la API corriendo:

- **Contenedor.** `solv-sat-postgres` se creó con `postgres:18-alpine`, 384 MB de cgroup (402 653 184 bytes) y el volumen `solv_sat_data_postgres` montado en `/var/lib/postgresql` con `PGDATA=/var/lib/postgresql/18/docker`.
- **Identidad.** El rol del alumno quedó `rolsuper=false`, `rolcreatedb=false`, `rolcreaterole=false`; la base `solv_std_std0001_subj001` tiene a ese rol como dueño y `has_database_privilege('public', ..., 'CONNECT')` devuelve `false`. El intento de `CREATE DATABASE` con el rol del alumno devuelve "permission denied to create database".
- **Init y sello.** El `db_init_script` se aplicó dentro de la base del alumno y el sello quedó en `_solv_exercise_migrations` con una sola fila tras dos arranques. Un tercer arranque después de apagar el motor conservó datos y sello.
- **Red.** Un contenedor distinto en `solv-satellites-net` resolvió `solv-sat-postgres` por DNS y consultó con las credenciales del alumno; el `docker exec` con `127.0.0.1` no se usa desde el laboratorio.
- **Migración.** `00031` se aplicó sobre `solv_db` con `go run ./cmd/migrate` (versión 31) y una segunda pasada no tuvo nada que hacer. La fila `postgres-16` quedó con `PostgreSQL 18 (Aula)`, `postgres:18-alpine`, `postgres`, 384 MB y `relational`.
- **Respaldo.** `backup.sh` respaldó el motor encendido y también el caso detenido: lo encendió, esperó `pg_isready`, generó `solv_sat_postgres_<ts>.sql.gz` con las bases y roles del aula y lo dejó detenido como estaba.
- **Limpieza.** El motor se detuvo en 0 y sobrevivió al worker de QoS; no hubo `OOMKilled` ni salida 137.
- `go build ./...`, `go vet ./...` y `go test ./...` sin fallos, incluidos los tests de integración contra Docker. El frontend pasa `ng build`, los 802 tests unitarios, los gates de estilos y fechas y `eslint`.
- Limitación conocida: el test de integración `TestJudgeDatabaseDryRunAndEvaluationAllEngines` falló una vez por timeout del subtest de MySQL y pasó al reintentarse; no toca este cambio pero conviene vigilar su flakiness.

### Defectos encontrados al probar

1. **El colector de zombies destruía los satélites.** `ListAllManagedContainers` incluía cualquier contenedor con la etiqueta `solv.managed`, y el colector borra todo lo que no sea un laboratorio registrado en la base. El satélite desaparecía a los pocos segundos de crearse. Afectaba a todos los motores gestionados, no solo a PostgreSQL. Se excluyen ahora los contenedores con etiqueta `solv.satellite`, con test unitario del filtro.
2. **Scale-to-zero apagaba el motor a mitad del aprovisionamiento.** El barrido periódico solo mira laboratorios activos en la base; se corrigió con el margen de gracia de D6.
3. **`docker stop` mataba el motor con `SIGKILL`.** Se corrigió con `SIGINT` como señal de parada (D6).
