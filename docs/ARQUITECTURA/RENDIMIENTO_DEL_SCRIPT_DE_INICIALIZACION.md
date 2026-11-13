# Rendimiento del Script de Inicialización de Base de Datos

> **Última medición:** 2026-11-13
> **Alcance:** PostgreSQL 18, satélite de aula
> **Decisiones relacionadas:** ADR-071, ADR-072, ADR-073, ADR-074

---

## Qué responde este documento

Cuánto tarda en prepararse la base de un alumno cuando el docente carga un script de
inicialización, cómo cambió ese tiempo con la transacción de ADR-074, y qué alternativas se
evaluaron y se descartaron.

Sirve para dos cosas: justificar la decisión con números y tener una línea base contra la que medir
cualquier cambio futuro.

## Cómo se mide

La medición usa la configuración real del satélite, no una aproximación:

```
postgres:18-alpine
--memory 384m --memory-swap 384m
-c shared_buffers=96MB -c max_connections=100
-c work_mem=2MB -c maintenance_work_mem=32MB
```

Cada caso arranca de una base vacía, aplica el archivo por entrada estándar y se cronometra solo la
ejecución del cliente:

```
psql <url> -v ON_ERROR_STOP=1 --single-transaction -q -f -
```

Los archivos generan cinco tablas, inserts sueltos uno por línea y cinco índices al final. Es la
forma en que un docente suele escribir un script a mano, no un volcado de `pg_dump`.

Los tiempos absolutos dependen del disco del servidor. La proporción entre casos se mantiene.

## Punto de partida

Antes de ADR-074, `psql` ejecutaba cada sentencia en autocommit: cada `INSERT` se confirmaba contra
el disco antes de seguir.

| Script | Tamaño | Un alumno |
| :--- | ---: | ---: |
| 2.000 `INSERT` | 194 KB | 5,15 s |
| 20.000 `INSERT` | 2,0 MB | 53,8 s |
| 100.000 `INSERT` | 10 MB | 5 min 7 s |

El costo era lineal y rondaba los 2,7 ms por sentencia.

## El diagnóstico

El problema no era el tamaño del archivo ni la memoria del contenedor. Era la cantidad de
confirmaciones. Dos mediciones lo confirman:

| Caso | Tiempo |
| :--- | ---: |
| 200.000 filas cargadas con `COPY` | 1,7 s |
| Copiar una base ya cargada de 23 MB | 0,36 s |

El mismo volumen de datos entra en 1,7 segundos cuando viaja en un solo bloque. Y una base entera se
copia en menos de medio segundo. Confirmar veinte mil veces era el costo, no mover los datos.

## Qué se cambió

`psql` aplica el script dentro de una sola transacción (`--single-transaction`), así que se confirma
una vez al final en lugar de una vez por sentencia. El script sigue llegando por entrada estándar
(ADR-072) y `ON_ERROR_STOP=1` se mantiene.

Una consecuencia que importa: si el script falla a la mitad, la base queda como estaba. Antes podía
quedar parcialmente cargada.

## Resultado

Mediana de tres corridas por tamaño, después del cambio:

| Script | Antes | Después | Mejora |
| :--- | ---: | ---: | ---: |
| 2.000 `INSERT` | 5,15 s | 0,36 s | 14× |
| 20.000 `INSERT` | 53,8 s | 2,15 s | 25× |
| 100.000 `INSERT` | 5 min 7 s | 8,86 s | 35× |

Las tres corridas de 100.000 sentencias fueron 9,88 s, 8,86 s y 8,66 s. La variación entre corridas
ronda el 15 %, así que conviene comparar medianas y no valores sueltos.

### Impacto en una clase de 35 alumnos

El script se aplica una vez por alumno, sobre su propia base (ADR-051).

| Script | Antes | Después |
| :--- | ---: | ---: |
| 2.000 `INSERT` | ~3 min | 13 s |
| 20.000 `INSERT` | ~31 min | 1 min 15 s |
| 100.000 `INSERT` | ~3 h | 5 min 10 s |

## Alternativas evaluadas

| Alternativa | Resultado |
| :--- | :--- |
| Silenciar la salida con `psql -q` | Sin efecto. 100.000 sentencias dieron 12,6 s sin `-q` y 14,8 s con `-q`; la diferencia es ruido. No se agregó el flag. |
| Subir memoria del contenedor | No ataca la causa: el cuello de botella son las confirmaciones, no la memoria. |
| Reescribir el script del docente a `COPY` | Sería el máximo rendimiento, pero obliga a transformar el archivo y a explicar por qué. Se descartó por fragilidad. |
| Base precargada y copia por alumno | Copiar tarda 0,36 s, mucho menos que aplicar. Queda como opción para volúmenes mucho mayores, con el costo de duplicar disco por alumno y administrar la validez de la plantilla. |

## Límites conocidos

- La transacción aplica solo a PostgreSQL. En MySQL y MariaDB el DDL provoca un commit implícito y
  envolver el script no serviría.
- `CREATE INDEX CONCURRENTLY`, `VACUUM`, `CREATE DATABASE` y `ALTER SYSTEM` no pueden ejecutarse
  dentro de una transacción. Un script que las use falla con error explícito.
- Un script con `BEGIN`/`COMMIT` propios puede cerrar la transacción antes de tiempo. `psql` avisa,
  no falla, y la atomicidad se pierde para ese caso.
- El sellado en `_solv_exercise_migrations` sigue fuera de la transacción. Si el script se aplica y
  el sellado falla, el script se reintenta en el próximo arranque.
- No se midió el camino completo por la API, solo la aplicación del script. El arranque del
  contenedor del alumno suma un tiempo parecido en todos los casos.

## Cómo reproducir

```bash
docker run -d --rm --name solv-pg-bench \
  --memory 384m --memory-swap 384m \
  -e POSTGRES_PASSWORD=bench -e PGDATA=/var/lib/postgresql/18/docker \
  -p 127.0.0.1:55433:5432 postgres:18-alpine \
  -c shared_buffers=96MB -c max_connections=100 \
  -c work_mem=2MB -c maintenance_work_mem=32MB

docker exec solv-pg-bench psql -U postgres -d postgres -c "CREATE DATABASE bench_db;"
time docker exec -i solv-pg-bench psql -U postgres -d bench_db \
  -v ON_ERROR_STOP=1 --single-transaction -q -f - < script.sql
```

Conviene calentar con una corrida descartable antes de cronometrar: la primera medición después de
arrancar el contenedor sale más lenta.

## Cuándo volver a revisar

Si aparecen scripts que tarden decenas de segundos por alumno incluso con la transacción. En ese
punto conviene comparar contra la base precargada, que ya está medida en 0,36 s por copia.
