# Estrategia de Almacenamiento SSD vs HDD — Nodo On-Premise SOLV

## Contexto del Hardware
La infraestructura actual de la plataforma SOLV corre sobre un servidor físico **Asus (12GB RAM, arquitectura x86_64 single-node)**. La orquestación de laboratorios virtuales mediante contenedores efímeros e interactivos impone una carga de Entrada/Salida (I/O) asimétrica entre lecturas/escrituras aleatorias de alta frecuencia y almacenamiento de datos históricos a largo plazo.

---

## Clasificación de Cargas por Tipo de Almacenamiento

### 1. Unidad de Estado Sólido (SSD — M.2 NVMe / SATA III - Mínimo 128GB)
**Propósito:** Operaciones de I/O aleatorio intensivo de baja latencia (<1ms).

* **Imágenes Docker:** Runtimes base (`gitpod/openvscode-server:1.96.0`, `postgres:18-alpine`, `semgrep/semgrep:1.100.0`, etc.).
* **Contenedores Efímeros:** Juez virtual de evaluación de código y compiladores efímeros.
* **Volúmenes Activos de Environments:** Entornos IDE activos de estudiantes donde ocurren comandos como `npm install`, compilaciones en C++/Java e instalación de paquetes.
* **Base de Datos Principal (`pg_data`):** Tablas activas de PostgreSQL 18 para asegurar transacciones en milisegundos.

### 2. Disco Duro Mecánico (HDD — SATA III 7200 RPM - Mínimo 1TB)
**Propósito:** Almacenamiento de alta capacidad para datos fríos e I/O secuencial.

* **Directorio de Archivados de Backups (`./backups/YYYY-MM/`):** Dumps comprimidos de PostgreSQL (`.sql.gz`) y empaquetados de volúmenes hibernados (`.tar.gz`).
* **Volúmenes Hibernados:** Environments de estudiantes sin actividad prolongada.
* **Logs Históricos y Telemetría:** Registros auditados de `audit_logs` archivados y logs de sistema `/var/log/solv-backup.log`.

---

## Configuración Recomendada de Docker Data-Root en SSD

Para garantizar que el motor Docker Engine utilice por defecto la unidad SSD como raíz de trabajo:

1. Editar o crear el archivo de configuración `/etc/docker/daemon.json`:
   ```json
   {
     "data-root": "/mnt/ssd/docker",
     "log-driver": "json-file",
     "log-opts": {
       "max-size": "10m",
       "max-file": "3"
     }
   }
   ```
2. Reiniciar el servicio de Docker:
   ```bash
   sudo systemctl stop docker
   sudo rsync -aP /var/lib/docker/ /mnt/ssd/docker/
   sudo systemctl start docker
   ```

---

## Almacenamiento de Objetos: SeaweedFS

Los objetos que no son bloques de disco ni tablas (hoy, los snapshots de avance de ADR-058) se guardan en **SeaweedFS** (`chrislusf/seaweedfs:4.51`), expuesto por su gateway S3 compatible. La imagen está fijada a una versión concreta, sin `:latest`.

### Por qué SeaweedFS y no un disco compartido

* Los snapshots no van en `bytea` de PostgreSQL: inflarían `pg_dump` y cada respaldo remoto.
* No van en el disco del backend: la fila y el archivo podían quedar desincronizados.
* SeaweedFS guarda objetos pequeños en volúmenes compartidos, con una API S3 estándar. El backend usa el SDK de AWS con *path-style*, sin código propio de transporte.

### Servicio en Compose

El servicio `seaweedfs` del archivo `compose.yml`:

* Arranca `weed server -s3` con datos en el volumen `seaweed_data`.
* Corre con `-volume.max=0`. Cada bucket S3 se guarda en una colección propia y el servidor trae un tope de 8 volúmenes: con el valor por defecto, crear un bucket nuevo falla en cuanto el servidor ya tiene 8. Sin tope, el alta del bucket y los snapshots no dependen de cuántos volúmenes previos existan.
* Publica el puerto S3 solo en `127.0.0.1:8333`. No es accesible desde la red.
* Está en `solv_net`; el backend lo alcanza como `http://seaweedfs:8333`.
* Genera el archivo de identidades `/etc/seaweedfs/s3.json` al arrancar, a partir de las variables de entorno. El gateway S3 de SeaweedFS exige ese archivo (`-s3.config`). Las variables `AWS_*` no bastan: el servidor no las lee para autenticar.
* Usa `SEAWEEDFS_ACCESS_KEY` y `SEAWEEDFS_SECRET_KEY`. Compose se niega a arrancar si no están definidas (`:?`), para no levantar el gateway con credenciales vacías.

Credenciales: definirlas en el archivo `.env` del servidor, nunca en el repositorio. Para rotarlas, cambiar las variables y reiniciar el servicio; el archivo de identidades se regenera en cada arranque.

### Código

* `backend/internal/infrastructure/storage/seaweedfs/advance_snapshot_store.go`: implementa `domain.AdvanceSnapshotStore`. Calcula SHA-256 al subir, rechaza objetos por encima de `MaxBytes` y verifica el hash al leer (`GetVerified`). Un hash que no coincide devuelve `ErrChecksumMismatch`. Borrar un objeto inexistente no es error. `PutFiles` empaqueta los archivos del entorno del docente y los sube en un solo paso; al construir el almacén se crea el bucket si no existe.
* `backend/internal/infrastructure/storage/seaweedfs/archive.go`: empaqueta el ejercicio en tar.gz. Aplica las exclusiones y el tope de archivos que llegan por `ArchiveOptions`, y omite enlaces simbólicos. Un patrón terminado en `*` excluye cualquier segmento de la ruta con ese prefijo, así que `.solv_*` también filtra `.solv_backups/<avance>/`.

Ninguno de los dos lee variables de entorno. `cmd/api` los construye con `SEAWEEDFS_ENDPOINT`, `SEAWEEDFS_BUCKET`, `SEAWEEDFS_ACCESS_KEY`, `SEAWEEDFS_SECRET_KEY`, `SEAWEEDFS_REGION`, `ADVANCE_SNAPSHOT_MAX_BYTES`, `ADVANCE_SNAPSHOT_MAX_FILES`, `ADVANCE_SNAPSHOT_EXCLUDES` y `ADVANCE_APPLY_CONCURRENCY` (aplicaciones simultáneas, 4 por defecto; el resto espera en cola). Sin endpoint, bucket o credenciales el backend arranca igual: emitir y aplicar avances responde 503.

### Retención

El snapshot no se borra al cerrar el aula: la ventana puede reabrirse hasta 7 días después del cierre. Se conserva **8 días desde `closes_at`** (7 del máximo de la ventana más 1 de margen) y después se elimina el objeto. El worker de retención corre al arrancar el backend y cada 24 horas: marca `snapshot_purged_at` en la fila y borra los objetos huérfanos de más de 24 horas. Las filas y el historial no se tocan.

Los respaldos del alumno no viven en SeaweedFS: quedan en `.solv_backups/` dentro de su volumen y la política los limita a los 5 más recientes. La política completa está en [retencion-snapshots-y-respaldos.md](../../proyecto-grado/avances/retencion-snapshots-y-respaldos.md).

### Qué falta

* Respaldo del volumen `seaweed_data`: `scripts/backup.sh` lo empaqueta en `seaweed_data_<fecha>.tar.gz` junto al dump de PostgreSQL. El nombre del volumen sale de `SEAWEED_VOLUME` (por defecto `solv_pg_seaweed_data`) y se rota a 7 días como el resto. Falta probarlo con el volumen real una vez que el servicio esté levantado.
* Comprobar la retención con objetos reales de 7 y 9 días. El recorrido de emisión, aplicación y vencimiento ya tiene prueba propia: `backend/tests/integration/class_advance_docker_e2e_test.go`, que se activa con `ADVANCE_E2E=1` y usa Docker y SeaweedFS reales.

---

## Invariante de Plantilla Base (Golden Image Invariant)

Para erradicar la contención de ancho de banda y fallas en el inicio de laboratorios en aulas de cómputo, SOLV adopta la **invariante de imagen base local**:

1. **Pre-almacenamiento en SSD (`overlay2`):** Toda plantilla aprobada o creada como oficial por el administrador debe residir en las capas locales de Docker (`InspectLocalImage == true`). Al momento de certificar una plantilla, el backend fuerza la descarga previa antes de consolidar el registro en la base de datos.
2. **Cero Pull en Tiempo de Clase:** El inicio de un entorno de estudiante asume y verifica la presencia en disco SSD. Se precalientan las imágenes del día con el endpoint `/api/v1/admin/templates/warm-all` o el botón *Calentar Caché* del panel administrativo.
3. **Aislamiento Offline:** Si la red institucional sufre microcortes o lentitud durante una sesión presencial, los laboratorios arrancan en menos de 2 segundos a partir del almacenamiento local NVMe/SSD sin depender de Docker Hub.

