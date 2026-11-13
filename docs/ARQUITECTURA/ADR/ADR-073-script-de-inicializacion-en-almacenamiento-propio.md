# ADR 073: Script de Inicialización en Almacenamiento Propio

> **Estado:** Implementado
> **Fecha:** 2026-11-12
> **Ámbito:** Core, Persistencia
> **Relacionado:** ADR-046 (servicios satélite y `db_init_script` multi-motor), ADR-050 (servicios satélite por rol), ADR-071 (alcance del script y tope de tamaño), ADR-072 (ejecución por entrada estándar)

---

## Contexto

El script de inicialización de base de datos se guardaba dentro de `exercises.config`, una columna
JSONB que el repositorio lee completa en cada consulta del ejercicio. Además, el script se escribía
dos veces en el mismo documento: `config.ide.db_init_script` y el espejo `config.database.init_script`.

Eso produce tres problemas concretos:

1. **Lectura costosa.** Cada apertura del editor y cada arranque de laboratorio deserializa el texto
   completo del docente, aunque la mayoría de las consultas solo necesita la configuración de entrega.
2. **Contenido mezclado con configuración.** El JSONB guarda parámetros pequeños y estables junto a un
   texto que puede medir megabytes, y cualquier `UPDATE` del config reescribe el script entero.
3. **Duplicación.** Los dos campos del script podían divergir, porque nada los mantenía sincronizados.

ADR-072 resolvió el techo de ejecución, pero dejó explícito que el techo de almacenamiento seguía
abierto y que era requisito para cualquier base precargada por materia.

## Decisión

### 1. El script vive en su propia tabla

```sql
CREATE TABLE exercise_db_init_scripts (
    exercise_id UUID PRIMARY KEY REFERENCES exercises(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    byte_size   INTEGER NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Una fila por ejercicio, con borrado en cascada cuando el ejercicio se elimina. `byte_size` queda
disponible para una cuota futura sin tener que medir el texto en cada consulta.

### 2. El config se persiste sin el script

`ExerciseConfig.Value()` es el punto único donde el dominio se convierte en columna. Ahí se serializa
una copia con `ide.db_init_script` y `database.init_script` en blanco. El valor `omitempty` de los tags
hace que las claves ni siquiera aparezcan en el JSON guardado.

La copia es superficial sobre los punteros que toca: el ejercicio en memoria conserva el texto, porque
el guardado y el arranque del laboratorio lo siguen usando en la misma operación.

### 3. La lectura por id reconstruye el ejercicio completo

`GetByIDAndTenant` consulta la tabla después de escanear la fila y vuelve a poblar `Config.IDE` y, si
el bloque existe, el espejo `Config.Database`. Así el editor docente y el aprovisionamiento reciben el
mismo objeto que antes, sin cambiar un solo consumidor.

La lista por materia no carga el script: es una consulta por ejercicio y no lo necesita. Tampoco
cambia lo que ve el alumno, porque el objeto que se serializa al rol `student` ya se construía sin ese
dato.

### 4. Un script vacío elimina la fila

Si el docente limpia el campo, el guardado borra la fila. No queda el texto anterior disponible para
un arranque posterior.

### 5. La migración mueve el contenido existente

`00057_exercise_db_init_scripts` crea la tabla, copia el script desde `config.ide.db_init_script` o
desde el espejo `config.database.init_script`, y después retira ambas claves del JSONB. La reversión
vuelve a escribir el script en `config.ide.db_init_script`, que es la ubicación que lee el backend,
antes de eliminar la tabla. Un ejercicio que solo tenía el espejo recupera el script en la clave
canónica, no en `config.database`.

## Consecuencias

- El JSONB del ejercicio deja de cargar con el texto del docente. Las consultas que no necesitan el
  script ya no lo deserializan.
- El script se escribe una sola vez, en una tabla, sin espejo persistido. Se termina la posibilidad de
  que los dos campos diverjan.
- Atender el editor y el arranque del laboratorio suma una consulta por id. Es una lectura por clave
  primaria sobre una tabla de una fila por ejercicio.
- `UpdateConfig` sigue escribiendo solo el JSONB y no toca el script. No tiene llamadores de producción
  hoy; si se usa en el futuro para cambiar el script, hay que extenderlo.
- La cuota por materia sobre bases precargadas ahora tiene dónde apoyarse (`byte_size`), pero no está
  implementada.

## Verificación

- `backend/internal/core/domain/exercise_config_value_test.go`: el config persistido no contiene
  `db_init_script` ni `init_script`, conserva la política de entrega y no vacía el script en memoria.
- `go build ./...` y `go test ./internal/... -count=1`.
- `backend/tests/integration/exercise_db_init_scripts_test.go` cubre la migración y el repositorio
  contra PostgreSQL real. Crea una base descartable en el mismo servidor de tests, aplica las
  migraciones desde cero, siembra ejercicios con las dos ubicaciones históricas del script, aplica
  `00057`, verifica el backfill y la limpieza del JSONB, ejercita `Create`, `GetByIDAndTenant` y
  `Update` del repositorio, revierte a `00056`, comprueba la restauración y vuelve a aplicar.
- El aislamiento es por base y no por esquema a propósito: la migración `00048` busca `workspaces`
  con `table_schema = 'public'`, así que un esquema propio deja la cadena incompleta. Esa dependencia
  del esquema público queda registrada como límite conocido de las migraciones.
- El test requiere `TEST_DB_DSN` o `TEST_DATABASE_URL`, igual que el resto del paquete de integración,
  y necesita permiso para crear y eliminar bases en ese servidor.
