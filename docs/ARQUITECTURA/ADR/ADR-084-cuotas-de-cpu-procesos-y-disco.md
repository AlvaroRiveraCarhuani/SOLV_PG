# ADR-084 — Cuotas de CPU, procesos y disco por entorno

- **Estado:** Aceptado
- **Fecha:** 2026-11-16
- **Ámbito:** Capacidad del nodo, ciclo de vida de entornos
- **Documentos relacionados:** `openspec/changes/dimensionamiento-practica-ram-cpu`,
  `openspec/changes/cuota-de-disco-por-entorno`, `docs/OPERATIONS/capacidad.md`

## Contexto

Antes de esta decisión el contenedor de práctica tenía memoria acotada por política,
pero **ninguna cuota de CPU y ningún tope de procesos**. Un entorno podía usar todos
los núcleos del nodo y abrir procesos sin freno. El disco tampoco tenía cuota: el
volumen del alumno crecía hasta llenar el servidor, y la única señal era el nodo
quedándose sin espacio.

Los números de partida no son inventados: salen de la medición de capacidad del nodo.

| Dato medido | Valor |
| :--- | :--- |
| Disco total / usado / libre | 98 GB / 59 GB / 34 GB (64%) |
| Volúmenes locales | 168 volúmenes, 19,9 GB, 95% recuperable |
| Imágenes | 28, 18,6 GB, 3,6 GB recuperables |
| Alumnos esperados por grupo | 40 |

La cuota de disco por entorno se deriva de esa medición:
`(34 GB libres + 19 GB recuperables) × 0,85 ÷ 40 alumnos ≈ 1,1 GB`, redondeado a
**1024 MB** por entorno.

## Decisión

### 1. La cuota vive en la política del nodo, no en el código

`ServerPolicies` declara los tres límites y su valor por defecto:

| Campo | Default | Significado |
| :--- | :---: | :--- |
| `practice_cpu_millis` | 1000 | Cuota absoluta de CPU por entorno; `0` = sin límite |
| `pids_limit` | 512 | Procesos por contenedor; el mismo tope que el entorno de revisión |
| `disk_quota_mb` | 1024 | Cuota **blanda** por entorno; `0` = sin cuota |
| `disk_warning_pct` | 80 | Umbral de aviso |
| `disk_block_pct` | 100 | Umbral de bloqueo de entrega |

Se aplican en `StartPracticeEnvironmentContainer` como `NanoCPUs` y `PidsLimit`.
Sin política, el arranque conserva el comportamiento anterior (un núcleo, 512
procesos), así que una política vieja no deja el nodo sin tope.

### 2. La cuota de disco es blanda y se mide fuera del camino de lectura

No existe cuota dura: `StorageOpt size` no cubre volúmenes nombrados y `prjquota`
exige un volumen lógico XFS que hoy no está. Se documenta como cuota blanda.

La medición la hace un barrido cada **5 minutos**, no la pantalla. El barrido mide
cada volumen abierto con un contenedor helper (`du -sm`, volumen en solo lectura) y
guarda `disk_used_mb` y `disk_state` (`ok` / `warning` / `full`). Leer el entorno
devuelve esos valores guardados más la cuota vigente; no consulta Docker.

La entrega de proyecto sigue midiendo en el momento de escribir: es el punto donde
el volumen crece y donde el aviso importa de verdad.

### 3. Al 100% se frena la entrega, no el editor

| Estado | Qué ve el alumno | Qué hace el sistema |
| :--- | :--- | :--- |
| `warning` (≥80%) | "Tu espacio está al 82% (840 MB de 1024 MB)" | Nada; puede seguir trabajando |
| `full` (≥100%) | "Tu espacio está lleno… borrá archivos generados y volvé a entregar" | La entrega devuelve 409 `disk_quota_exceeded` |

Cerrar el contenedor en medio del trabajo pierde código sin guardar, y con el disco
lleno el entorno puede no volver a arrancar: dejaría al alumno encerrado sin salida.
Frenar la entrega corta el crecimiento del volumen igual, porque la entrega es lo que
lo hace crecer.

### 4. El nodo crítico sí detiene altas

`HostFreeDiskPercent` lee el espacio libre del sistema de archivos. Por debajo del
**5%** libre, el alta de entornos nuevos se rechaza con 503 `host_disk_exhausted`.
Los entornos ya abiertos no se tocan: la negativa es para los nuevos, y el umbral se
elige bajo para no castigar a quien está trabajando.

## Consecuencias

- Un entorno no puede acaparar el nodo por CPU ni abrir procesos sin control.
- El disco tiene tres escalones visibles y ninguno destruye trabajo del alumno.
- La pantalla no paga una medición de Docker por refresco.
- La cuota es blanda: entre dos barridos, un volumen puede pasarse de la cuota. Queda
  anotado en `capacidad.md` como límite conocido.
- Sin cuota dura, la protección real del nodo es el rechazo de altas por debajo del 5%.
- La limpieza de cachés por entorno y la liberación de volúmenes sin dueño ya están
  en la interfaz: el inventario cruza `LINK=0` contra los entornos registrados, borra
  solo `solv_workspace_*` confirmados por frase y audita cada volumen.
- Pendiente: tests de integración de la cuota, separación de `pg_data` y cuota dura
  XFS con `prjquota` cuando exista el volumen lógico.
