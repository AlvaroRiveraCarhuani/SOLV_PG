# Estrategia de Almacenamiento SSD vs HDD — Nodo On-Premise SOLV

## Contexto del Hardware
La infraestructura actual de la plataforma SOLV corre sobre un servidor físico **Asus (12GB RAM, arquitectura x86_64 single-node)**. La orquestación de laboratorios virtuales mediante contenedores efímeros e interactivos impone una carga de Entrada/Salida (I/O) asimétrica entre lecturas/escrituras aleatorias de alta frecuencia y almacenamiento de datos históricos a largo plazo.

---

## Clasificación de Cargas por Tipo de Almacenamiento

### 1. Unidad de Estado Sólido (SSD — M.2 NVMe / SATA III - Mínimo 128GB)
**Propósito:** Operaciones de I/O aleatorio intensivo de baja latencia (<1ms).

* **Imágenes Docker:** Runtimes base (`gitpod/openvscode-server:1.96.0`, `postgres:18-alpine`, `semgrep/semgrep:1.100.0`, etc.).
* **Contenedores Efímeros:** Juez virtual de evaluación de código y compiladores efímeros.
* **Volúmenes Activos de Workspaces:** Entornos IDE activos de estudiantes donde ocurren comandos como `npm install`, compilaciones en C++/Java e instalación de paquetes.
* **Base de Datos Principal (`pg_data`):** Tablas activas de PostgreSQL 18 para asegurar transacciones en milisegundos.

### 2. Disco Duro Mecánico (HDD — SATA III 7200 RPM - Mínimo 1TB)
**Propósito:** Almacenamiento de alta capacidad para datos fríos e I/O secuencial.

* **Directorio de Archivados de Backups (`./backups/YYYY-MM/`):** Dumps comprimidos de PostgreSQL (`.sql.gz`) y empaquetados de volúmenes hibernados (`.tar.gz`).
* **Volúmenes Hibernados:** Workspaces de estudiantes sin actividad prolongada.
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

## Invariante de Plantilla Base (Golden Image Invariant)

Para erradicar la contención de ancho de banda y fallas en el inicio de laboratorios en aulas de cómputo, SOLV adopta la **invariante de imagen base local**:

1. **Pre-almacenamiento en SSD (`overlay2`):** Toda plantilla aprobada o creada como oficial por el administrador debe residir en las capas locales de Docker (`InspectLocalImage == true`). Al momento de certificar una plantilla, el backend fuerza la descarga previa antes de consolidar el registro en la base de datos.
2. **Cero Pull en Tiempo de Clase:** El inicio de un workspace de estudiante asume y verifica la presencia en disco SSD. Se precalientan las imágenes del día con el endpoint `/api/v1/admin/templates/warm-all` o el botón *Calentar Caché* del panel administrativo.
3. **Aislamiento Offline:** Si la red institucional sufre microcortes o lentitud durante una sesión presencial, los laboratorios arrancan en menos de 2 segundos a partir del almacenamiento local NVMe/SSD sin depender de Docker Hub.

