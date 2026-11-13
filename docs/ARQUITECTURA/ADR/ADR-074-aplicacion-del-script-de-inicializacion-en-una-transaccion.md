# ADR 074: Aplicación del Script de Inicialización en una Transacción

> **Estado:** Implementado
> **Fecha:** 2026-11-13
> **Ámbito:** Core, Servicios satélite
> **Relacionado:** ADR-046 (servicios satélite y `db_init_script` multi-motor), ADR-051 (PostgreSQL de aula como satélite dedicado), ADR-071 (alcance del script), ADR-072 (ejecución por entrada estándar), ADR-073 (script en almacenamiento propio)

---

## Contexto

ADR-072 quitó el techo de tamaño del script, pero dejó intacto el costo de ejecución. `psql`
ejecuta cada sentencia en modo autocommit: cada `INSERT` se confirma contra el disco antes de
continuar con el siguiente. Un script con miles de sentencias sueltas paga miles de confirmaciones.

La medición se hizo con la configuración real del satélite: `postgres:18-alpine`, techo de 384 MB,
`shared_buffers=96MB`, `work_mem=2MB` y `maintenance_work_mem=32MB`.

| Script | Tamaño | Un alumno, antes |
| :--- | ---: | ---: |
| 2.000 `INSERT` | 194 KB | 5,15 s |
| 20.000 `INSERT` | 2,0 MB | 53,8 s |
| 100.000 `INSERT` | 10 MB | 5 min 7 s |

El costo es lineal y ronda los 2,7 ms por sentencia. Como el script se aplica una vez por alumno
sobre su propia base (ADR-051), una clase de 35 alumnos pagaba ese tiempo 35 veces.

Dos datos más de la misma medición, que descartan explicaciones alternativas:

- 200.000 filas cargadas con `COPY` tardan 1,7 s. El problema no es el volumen de datos, es la
  cantidad de confirmaciones.
- Copiar una base ya cargada de 23 MB tarda 0,36 s. La alternativa de precargar existe, pero es una
  pieza mucho más grande que este cambio.

## Decisión

`psql` aplica el script dentro de una sola transacción mediante `--single-transaction`.

```
psql <url> -v ON_ERROR_STOP=1 --single-transaction -f -
```

El script sigue llegando por entrada estándar (ADR-072) y `ON_ERROR_STOP=1` se mantiene: un error
corta la ejecución y la transacción no se confirma.

### Alcance

- Aplica solo a PostgreSQL. En MySQL y MariaDB envolver el script no ayudaría: el DDL provoca un
  commit implícito, así que la transacción se cerraría en la primera sentencia de estructura.
- El sellado en `_solv_exercise_migrations` sigue siendo una sentencia aparte, fuera de esta
  transacción. Un fallo al sellar después de aplicar obliga a reintentar el script en el próximo
  arranque; ese comportamiento ya existía y no cambia con este ADR.

## Consecuencias

- **Todo o nada.** Si el script falla a la mitad, la base del alumno queda como estaba. Antes podía
  quedar parcialmente cargada y el arranque fallaba igual.
- **Sentencias que no admiten transacción** (`CREATE INDEX CONCURRENTLY`, `VACUUM`, `CREATE DATABASE`,
  `ALTER SYSTEM`) ahora fallan con un error explícito. No hay camino alternativo automático; un
  script que las necesite debe resolverse fuera de este flujo. No aparecen en scripts de práctica
  normales.
- **Scripts con `BEGIN`/`COMMIT` propios** pueden cerrar la transacción antes de tiempo. `psql`
  emite un aviso, no un error, así que la atomicidad se pierde para ese caso. No se considera
  soportado.
- **El tiempo deja de depender de la cantidad de sentencias.** La comparación medida:

| Script | Un alumno, antes | Un alumno, después | Mejora |
| :--- | ---: | ---: | ---: |
| 2.000 `INSERT` | 5,15 s | 0,36 s | 14× |
| 20.000 `INSERT` | 53,8 s | 2,15 s | 25× |
| 100.000 `INSERT` | 5 min 7 s | 8,86 s | 35× |

  Los valores posteriores son la mediana de tres corridas; la variación entre corridas ronda el
  15 %. Con 35 alumnos, el mismo conjunto pasa de 3 minutos a 13 segundos, de 31 minutos a 1 minuto
  15 segundos, y de 3 horas a 5 minutos 10 segundos.
- La base precargada con copia por alumno (Fase 2) deja de ser urgente: solo se justificaría para
  volúmenes mucho mayores, donde el costo de copia y el disco duplicado sigan siendo menores que
  aplicar el script.
- `psql -q` se midió y se descartó: no reduce el tiempo de forma consistente. El detalle del método,
  las medianas y las alternativas evaluadas están en
  `docs/ARQUITECTURA/RENDIMIENTO_DEL_SCRIPT_DE_INICIALIZACION.md`.

## Verificación

- `backend/internal/core/services/practice_environment_managed_satellite_test.go`: la ejecución que
  transporta el script usa `--single-transaction` en PostgreSQL; MySQL no lo usa.
- Medición contra PostgreSQL 18 real con la configuración del satélite, con los tres tamaños de la
  tabla anterior, antes y después del cambio.
- Comprobación de reversión: un script que crea una tabla y después falla deja la base sin la tabla
  y devuelve error. La transacción no se confirma.
- `go build ./...` y `go test ./internal/core/services/... -count=1`.
