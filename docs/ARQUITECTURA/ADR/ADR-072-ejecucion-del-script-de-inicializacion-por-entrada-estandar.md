# ADR 072: Ejecución del Script de Inicialización por Entrada Estándar

> **Estado:** Implementado
> **Fecha:** 2026-11-12
> **Ámbito:** Core, Servicios satélite
> **Relacionado:** ADR-046 (servicios satélite y `db_init_script` multi-motor), ADR-050 (servicios satélite por rol), ADR-051 (PostgreSQL de aula como satélite dedicado), ADR-071 (alcance del script y tope de tamaño)

---

## Contexto

ADR-071 dejó documentado un límite que no se podía resolver desde la interfaz: el script de
inicialización viajaba como argumento de los clientes de base de datos.

```
psql <url> -v ON_ERROR_STOP=1 -tAc <script>
mysql -uroot -p<pass> <base> -e <script>
```

Linux limita el tamaño total de `argv` más `envp` de un proceso a 2 MiB (`ARG_MAX`). Un script que se
acerque a ese total falla con `E2BIG` antes de ejecutar una sola sentencia, y como el entorno del
proceso también consume parte del presupuesto, el techo efectivo queda por debajo de ese número. La
interfaz solo mostraba una advertencia: no había forma de aceptar un archivo grande.

## Decisión

### 1. El script SQL se entrega por entrada estándar

Los dos clientes aceptan leer el script de la entrada estándar:

- PostgreSQL: `psql <url> -v ON_ERROR_STOP=1 -f -`.
- MySQL y MariaDB: `mysql -uroot -p<pass> <base>` sin `-e`.

El contenido ya no ocupa lugar en la línea de comandos, así que el límite del kernel deja de aplicar.
`ON_ERROR_STOP=1` se mantiene: un error del script sigue cortando la ejecución y el ejercicio no queda
sellado en `_solv_exercise_migrations`.

### 2. El orquestador suma una capacidad de entrada

`ManagedSatelliteOrchestrator` incorpora `ExecInManagedSatelliteWithInput(ctx, engine, cmd, input)`.
En el orquestador Docker crea el exec con `AttachStdin` y `StdinOnce`, escribe el contenido en el
stream adjunto y cierra el extremo de escritura para que el proceso reciba EOF. La implementación sin
entrada delega en la misma función con `input` nulo, así que el camino de las sentencias sueltas no
cambia.

Las sentencias de control (rol, base, sellado, verificación) siguen usando `ExecInManagedSatellite`:
son cortas y construidas en Go, no llevan el contenido del docente.

### 3. MongoDB queda fuera con su motivo

El init de MongoDB es JavaScript y se evalúa con `mongosh --eval`, que también recibe el script como
argumento. Pasarlo por entrada requeriría un archivo temporal dentro de un contenedor compartido por
todos los alumnos del motor, con un nombre único por base para no pisarse entre aprovisionamientos
concurrentes. No hay consumo de ese motor que lo justifique todavía, así que se deja como está y el
tope sigue aplicando solo a ese camino.

## Consecuencias

- Un script SQL puede superar los 2 MiB sin fallar por tamaño de argumentos. El umbral de la interfaz
  (`MAX_RECOMMENDED_SQL_BYTES`) deja de describir un límite del sistema y pasa a ser una recomendación
  de costo: un script muy grande demora el aprovisionamiento de cada alumno.
- El contenido del script ya no se guarda en `exercises.config`: vive en la tabla
  `exercise_db_init_scripts` (ADR-073). El techo de ejecución y el de almacenamiento quedan resueltos.
- El error de un script se reporta con el stderr del cliente, igual que antes. Un fallo sigue
  abortando el arranque sin dejar un entorno a medio inicializar.
- La verificación con tests cubre que el script llegue por entrada estándar y no como argumento en
  PostgreSQL, MySQL y MariaDB. No se ejecutó contra contenedores reales en este cambio.

## Verificación

- `backend/internal/core/services/practice_environment_managed_satellite_test.go`: el script del
  ejercicio se entrega por entrada estándar a `psql` y a `mysql`, no aparece en `cmd`, y se aplica una
  sola vez.
- `backend/internal/core/services/practice_environment_provisioning_test.go`: el doble de orquestador
  registra la entrada y clasifica las sentencias por contenido, con o sin `-e`.
- `go build ./...` y `go test ./internal/core/services/... -count=1`.
