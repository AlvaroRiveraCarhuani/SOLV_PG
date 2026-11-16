# Capacidad: cuántos alumnos aguantan a la vez

Este documento responde una pregunta de venta y de dimensionamiento: cuántos
alumnos puede atender un servidor con SOLV. Distingue lo que el sistema calcula
de lo que hay que medir.

## 1. Qué decide el número

La creación de un entorno pasa por dos compuertas independientes
(`practice_environment_service.go`):

| Compuerta | De dónde sale | Qué pasa si se alcanza |
|---|---|---|
| Techo por institución | `max_containers` de las políticas del servidor | `503 max_containers_reached` |
| Presupuesto de RAM del host | RAM total × `host_safety_ceiling_pct` − cuotas comprometidas − satélites | `503 host_memory_exhausted` |

Manda la menor. Por eso un techo chico en un servidor grande no protege de nada:
solo desperdicia hardware. Hasta ahora el techo por defecto era `40`, un número
elegido para la máquina de desarrollo; en una facultad con 128 GB eso deja el
servidor trabajando a un tercio de su memoria.

## 2. Cómo se calcula el techo recomendado

Si la institución nunca guardó políticas, el techo se deriva de la RAM medida:

```
techo = (RAM total × techo de seguridad% − 2048 MB) ÷ RAM por entorno
```

- **RAM total**, no disponible: lo disponible cambia con los laboratorios ya
  abiertos y no sirve para planificar.
- **2048 MB de reserva** (`domain.PlatformReserveMB`): sistema operativo, backend,
  frontend, traefik, base principal y métricas. Es un piso de arranque, no una
  medición.
- **Techo de seguridad**: 85% por defecto, configurable en 75/80/85/90.
- El resultado se acota a 500 (`domain.MaxAllowedContainers`), que es el máximo
  que acepta la validación de políticas.

Sin medición no hay número. Cuando el host no informa su memoria, la API expone
`capacity_basis: "unavailable"` y el panel de plantillas muestra «sin medición»
en el máximo del host, la RAM libre y la capacidad simultánea, en lugar de
asumir 8192 MB de total y 4096 MB libres. El techo de publicación lo sigue
poniendo la admisión del backend, que rechaza lo que no puede verificar; el
panel no promete un límite que no midió.

La API devuelve el número junto a sus supuestos en
`GET /api/v1/admin/server/policies`:

```json
{
  "ram_limit_mb": 512,
  "inactivity_minutes": 15,
  "max_containers": 213,
  "host_safety_ceiling_pct": 85,
  "recommendation": {
    "recommended_max_containers": 213,
    "host_total_mb": 131072,
    "platform_reserve_mb": 2048,
    "ram_per_lab_mb": 512,
    "host_safety_ceiling_pct": 85,
    "capacity_basis": "host_ram"
  }
}
```

`capacity_basis: "unavailable"` significa que no se pudo medir el host: en ese
caso el techo es el guardado (o el de último recurso) y no hay recomendación.

## 3. Tabla de referencia

Techo de seguridad 85%, reserva 2048 MB. Los valores son el máximo teórico de
entornos simultáneos, no una promesa de rendimiento.

| RAM total | 256 MB por entorno | 512 MB por entorno | 1024 MB por entorno |
|---:|---:|---:|---:|
| 16 GB | 19 | 23 | 4 |
| 32 GB | 100 | 50 | 25 |
| 64 GB | 209 | 104 | 52 |
| 128 GB | 427 | 213 | 106 |
| 256 GB | 500¹ | 431 | 215 |

¹ El cálculo da 862; se acota al máximo aceptado por la validación (500).

Para otras combinaciones, la fórmula de la sección 2 es la misma que aplica el
backend; el test `TestRecommendMaxContainersScalesWithHostRAM` fija estos valores.

## 4. Qué no mide esta tabla

La recomendación es un techo de memoria. Un servidor de venta se dimensiona con
lo que falta acá:

- **CPU.** Un entorno con compilación o tests satura un núcleo mientras corre.
  Un host con mucha RAM y pocos núcleos se degrada antes por CPU que por memoria.
  Esta fórmula no lo descuenta.
- **Motores satélite.** Cada motor administrado (PostgreSQL 384 MB, MySQL y
  MariaDB 128 MB, MongoDB 128 MB) consume reserva propia por nodo; la admisión la
  descuenta, la recomendación no. Un aula con muchos motores distintos soporta
  menos entornos que la tabla.
- **Disco.** El límite práctico suele ser la I/O: 60 IDEs indexando a la vez
  pesan más en disco que en RAM.
- **Arranque simultáneo.** El peor caso no es régimen sino los primeros minutos
  de una clase, cuando todos abren a la vez y cada contenedor paga su arranque.
- **Red.** Cada IDE mantiene un WebSocket; con un solo servidor y 100 alumnos, el
  tráfico del editor es bajo, pero la descarga inicial de dependencias no.

## 5. Primera medición del host de desarrollo

No es la tabla de venta: es el punto de partida con números medidos y lo que
falta para completarla.

- Fecha: 2026-11-14. Host: 7831 MB de RAM, 4 núcleos, SSD.
- Imagen: `gitpod/openvscode-server:1.96.0`, límite 512 MB por entorno
  (`--memory-reservation` al 75%).
- Comando: `scripts/capacity-check.sh --count 3 --memory 512 --wave 5`.

| Medición | Resultado |
|---|---|
| Arranque hasta responder HTTP (frio, imagen en caché) | 2030 ms el primero, 612–617 ms los siguientes |
| Memoria por entorno en reposo, sin alumno conectado | 31 MB el primero, 19 MB el resto |
| Memoria disponible del host antes / después | 1054 MB / 1255 MB |

Este número **no** sirve para dimensionar: sin un alumno conectado el editor no
levanta su extension host ni indexa el workspace. Lo que deja fijado es el piso
del proceso (20–30 MB) y que el arranque en frío de la imagen no es el cuello de
botella. La memoria por entorno que importa en clase la determinan el cliente
conectado, la extensión de lenguaje y el tamaño del repositorio del alumno, no el
editor en reposo.

Con ese piso, la política de 512 MB por entorno es entre 15 y 25 veces el consumo
en reposo: el límite realista lo va a fijar la medición con alumnos, no el techo
de RAM de la tabla de la sección 3.

Para completarla en el servidor de la facultad, el mismo comando con los alumnos
conectados:

```bash
bash scripts/capacity-check.sh --count 30 --memory 512
bash scripts/capacity-check.sh --count 30 --memory 512 --wave 0   # todos a la vez
```

El segundo ensayo (oleada 0) es el que corresponde al inicio de una clase y da el
tiempo de arranque p95 que se usa en la propuesta.

## 6. Cómo se mide de verdad

La tabla de la sección 3 es el techo teórico. Para vender hay que medir en el
servidor concreto, porque los números que importan son los del arranque de una
clase completa.

```bash
# Línea base antes de abrir nada
free -m
docker stats --no-stream --format 'table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}'
uptime

# Durante la oleada: cada 5 s, en otra terminal
while sleep 5; do date +%H:%M:%S; free -m | sed -n '2p'; uptime; done
```

Procedimiento:

1. Abrir entornos en oleadas de 10 y anotar, por oleada: memoria usada del host,
   `MemAvailable`, load average y tiempo desde el clic hasta que el IDE responde.
2. Repetir con arranque simultáneo (todos en menos de un minuto). Es la medición
   que se presenta: la diferencia con las oleadas suele ser el doble de tiempo.
3. Registrar los `503` que aparezcan y con qué error (`max_containers_reached` o
   `host_memory_exhausted`): dicen qué compuerta se cruzó primero.
4. Detenerse cuando `MemAvailable` baje del 10%, el load average supere los
   núcleos o el arranque pase de 60 segundos. Ese es el número real del servidor.

Planilla para la propuesta (una fila por medición):

| Fecha | Servidor (núcleos / RAM) | Alumnos | Arranque p50 | Arranque p95 | RAM usada | 503 | Observación |
|---|---|---|---|---|---|---|---|
| | | | | | | | |

## 7. Cómo se cambia el techo

```bash
curl -X PUT https://<host>/api/v1/admin/server/policies \
  -H "Authorization: Bearer <token-admin>" \
  -H "Content-Type: application/json" \
  -d '{"max_containers": 120}'
```

En Configuración → Servidor el campo aparece como «Laboratorios Simultáneos por
Institución», con el techo vigente, la recomendación y un botón para adoptarla;
también se puede ajustar por API con un token de administrador. El valor se
guarda y el worker QoS lo recarga en su próximo ciclo. Bajarlo no
expulsa entornos abiertos: la admisión lo aplica a los nuevos, así que una clase
en curso termina normalmente. Un valor por debajo de lo que ya corre se respeta
como techo de admisión pero el panel sigue informando los entornos vivos.

Si el host se queda sin memoria, primero se baja el techo y se hibernan entornos;
el procedimiento de emergencia está en `docs/OPERATIONS/incident-runbook.md` §6.

## 8. Préstamo de memoria cuando el nodo tiene hueco

La cuota de un entorno no es un número fijo. La admisión toma la base que fijan
la política del tenant y la plantilla, y le suma un préstamo mientras el nodo
tenga lugar (`loanQuotaMB`, ADR-082):

```
presupuesto = RAM total × techo de seguridad − comprometido − satélites
préstamo    = el perfil más alto (512 … 4096) que deja sitio a otro igual
```

El préstamo nunca deja sin lugar al siguiente alumno: un perfil entra solo si
`2 × perfil ≤ presupuesto`. En un nodo de 12 GB el primer entorno recibe
4096 MB, el segundo 2048 y el tercero 1536; cuando la clase se llena, la cuota
vuelve a los 512 MB de la base. En el nodo de medición de la sección 5 (8 GB) el
primero recibe 2048 MB.

El préstamo no sobrevive a la hibernación: al despertar, el entorno vuelve a
pedir cuota y recibe la que quepa en ese momento. Esa es la válvula que devuelve
la memoria a la clase sin apagar a nadie y sin bajarle el límite a un contenedor
en ejecución.

Lo que no cambia: si ni la base entra, el entorno se rechaza con
`host_memory_exhausted`, y el techo por contenedor sigue siendo 4096 MB, que es
el perfil Intensivo que el administrador ya podía definir en una plantilla.

## 9. Disco: cuánto ocupa un laboratorio y qué pasa cuando se llena

La cuota de disco es **blanda y se mide**. No hay un límite que el sistema de
archivos haga cumplir: `StorageOpt size` cubre la capa escribible del
contenedor y no los volúmenes nombrados, y `prjquota` necesita un volumen
lógico XFS que el nodo todavía no tiene (sección 10).

| Pieza | Valor | Dónde se configura |
| :--- | :--- | :--- |
| Cuota por entorno | 1024 MB | `disk_quota_mb` de la política del servidor (`0` = sin cuota) |
| Aviso | 80% | `disk_warning_pct` |
| Bloqueo de entrega | 100% | `disk_block_pct` |
| Barrido de medición | cada 5 minutos | `DiskUsageWorker` |
| Piso de disco del nodo | 5% libre | `DefaultHostDiskFloorPct` |

El barrido mide un volumen por entorno abierto con un contenedor efímero
(`du -sm`, volumen en solo lectura), guarda `disk_used_mb` y `disk_state`, y la
ficha lee lo guardado: refrescar la pantalla no abre contenedores. La entrega
mide en el momento de escribir, que es cuando el volumen crece.

| Estado | Umbral | Qué ve el alumno |
| :--- | :--- | :--- |
| `ok` | < 80% | nada |
| `disk_warning` | ≥ 80% | aviso con el uso, el límite y que la cuota es blanda; puede seguir trabajando y entregando |
| `disk_full` | ≥ 100% | la entrega responde 409 `disk_quota_exceeded` y la ficha ofrece liberar espacio |

El editor **no** se cierra al llenarse la cuota: cerrarlo en medio del trabajo
pierde código sin guardar, y con el disco lleno el entorno podría no volver a
arrancar. Frenar la entrega corta el crecimiento igual, porque la entrega es lo
que escribe el proyecto.

El botón «Liberar espacio» de la ficha borra solo rutas que la herramienta
regenera (`node_modules`, `.cache`, `dist`, `build`, `target`, `__pycache__`,
`.pytest_cache`, `.next`), vuelve a medir y deja el estado en `ok` si baja del
umbral. No toca código del alumno ni un examen congelado.

### Liberación de volúmenes sin dueño

`GET /api/v1/admin/server/storage/orphans` cruza los volúmenes con `LINK=0`
(una lectura de `/system/df`, no un `du` del árbol) contra todos los entornos
registrados, en cualquier estado. `POST .../orphans/prune` borra solo los
nombres elegidos, exige la frase `LIBERAR VOLUMENES HUERFANOS` y deja una
entrada `ORPHAN_VOLUME_RELEASED` en `audit_logs` por volumen.

Tres barreras, en este orden:

1. Solo `solv_workspace_*`. La base (`pg_data`), los respaldos y cualquier
   volumen de otra pila quedan fuera por nombre, aunque estén sin contenedor:
   con el stack detenido, `pg_data` también aparece con `LINK=0`.
2. Ningún entorno registrado puede reclamarlo, aunque esté hibernado, congelado
   o fallido.
3. El plan se recalcula al borrar y el daemon se llama sin `force`: un volumen
   que se montó mientras la pantalla estaba abierta se rechaza, no se destruye.

La corrida inicial es una tarea de servidor: abrir Configuración → Servidor →
Espacio en Disco, guardar la lista con fecha y liberar solo lo que el panel
marque como recuperable. No hay worker automático: un volumen recién creado
puede no estar reclamado en base todavía.

Límites conocidos:

- Entre dos barridos, un volumen puede pasarse de la cuota. El freno del nodo
  es el rechazo de altas por debajo del 5% libre (`503 host_disk_exhausted`).
- La liberación no borra intentos de examen: se conservan como evidencia
  (ADR-083) y se liberan a mano.

## 10. Separación del disco de la base y camino a la cuota dura

Pendiente de ejecución en el servidor. Nada de esta sección está aplicado.

Estado medido (2026-11-16, `asus-server`): `/var/lib/docker` es ext4 sobre LVM
(`ubuntu-vg-ubuntu-lv`), 98 GB totales con 34 GB libres; `pg_data` vive en el
mismo sistema de archivos que los volúmenes de laboratorio. Un disco de
entornos lleno deja la plataforma sin poder registrar entregas ni notas.

### 10.1 Base de notas en su propio volumen lógico

1. Verificar el espacio del grupo: `vgs ubuntu-vg` y `lvs`. Si no hay hueco, usar
   el HDD de datos fríos.
2. Respaldo verificado antes de mover: crear respaldo desde el panel y
   comprobar el checksum con «Verificar».
3. Ventana de mantenimiento: activar Modo Mantenimiento y detener el stack.
4. Crear el volumen lógico (`lvcreate -L 20G -n solv_pg ubuntu-vg`), formatearlo
   ext4 y montarlo en `/var/lib/solv/pg` con entrada en `/etc/fstab`.
5. Copiar `pg_data` con `rsync -aHAX` conservando dueño y permisos; apuntar el
   servicio `postgres` a la ruta nueva (`/var/lib/solv/pg:/var/lib/postgresql`).
6. Arrancar y comprobar: `pg_isready`, conteo de tablas, una entrega de prueba y
   la latencia de una consulta de notas.
7. Vuelta atrás: restaurar el montaje anterior y arrancar. El volumen viejo no se
   borra hasta que el nuevo haya servido una jornada completa.

### 10.2 Volumen XFS con `prjquota` para cuota dura

1. Crear un volumen lógico para los volúmenes de entorno, formatearlo (`mkfs.xfs
   -f`), montarlo con `defaults,prjquota` en `/etc/fstab` y remontar.
2. Mover los volúmenes existentes con el entorno detenido: copiar el contenido
   con el mismo nombre y montar cada uno en el sistema de archivos nuevo.
3. Asignar un identificador de proyecto por volumen
   (`xfs_quota -x -c 'project -s ...'`) y fijar `bhard` con la cuota de la
   política.
4. Verificar el rechazo de escritura al llegar al límite y leer el reporte con
   `xfs_quota -x -c 'report -h'`.

Mientras ese volumen no exista, la cuota sigue siendo blanda y la interfaz no
afirma que el sistema operativo rechace la escritura.

## Volúmenes de examen e intentos

Cada examen nace en su propio volumen (`_exam_<n>`, ADR-083). El volumen de la
práctica queda aparte, y cada recuperatorio agrega uno más. Nada los borra solo:
el intento anterior se conserva como evidencia.

Para dimensionar el disco, el espacio de exámenes es:

```
alumnos × intentos (1 + recuperatorios) × tamaño del proyecto base
```

Un examen con 40 alumnos, dos intentos y proyectos de 20 MB ocupa cerca de
1,6 GB en volúmenes, además de lo que se haya escrito dentro. La limpieza de
volúmenes de intentos cerrados es una tarea pendiente: hoy se liberan a mano.

El estado del entorno que ve el alumno (`preparing`, `ready`, `closed`,
`failed`) se calcula del registro, así que no agrega tráfico ni columnas.
