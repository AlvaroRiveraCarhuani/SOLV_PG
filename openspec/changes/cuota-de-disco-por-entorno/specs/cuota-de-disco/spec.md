# cuota-de-disco Specification

## Purpose

Poner un techo medido al espacio que puede ocupar cada entorno, avisar antes de llenar el
disco, liberar lo que ya está ocupado sin dueño y separar el almacenamiento de la base de
notas del de los entornos.

## Requirements

### Requirement: Medición del espacio por entorno

El sistema MUST medir el tamaño del volumen de cada entorno y MUST exponerlo como
`used_mb` junto al límite vigente y al porcentaje. La medición MUST salir de una lectura del
daemon de Docker y MUST NOT recorrer el árbol de volúmenes completo en cada ciclo.

#### Scenario: Medición de un entorno en uso
- GIVEN un entorno con un volumen de 1.2 GB y un límite de 2 GB
- WHEN el worker completa un ciclo
- THEN el entorno informa `used_mb = 1200`, `limit_mb = 2048` y `percent = 58`
- AND el estado sigue siendo `ok`.

---

### Requirement: Aviso y bloqueo por cuota

El sistema MUST marcar `disk_warning` al alcanzar el 80% del límite y `disk_full` al 100%.
En `disk_full` MUST rechazar la entrega con 409 y MUST ofrecer una acción de limpieza;
MUST NOT detener el contenedor ni montar el volumen en solo lectura, porque cerrar el
editor pierde código sin guardar y con el disco lleno el entorno podría no volver a
arrancar. El entorno MUST volver a `running` cuando el uso baje del 80%.

#### Scenario: Aviso al 80%
- GIVEN un entorno con límite de 2 GB y uso de 1.7 GB
- WHEN el worker evalúa el entorno
- THEN el estado es `disk_warning`
- AND el alumno ve un aviso con el espacio que le queda.

#### Scenario: Bloqueo al 100%
- GIVEN un entorno con límite de 2 GB y uso de 2.05 GB
- WHEN el alumno intenta entregar
- THEN la entrega se rechaza con 409 `disk_quota_exceeded`
- AND el estado es `disk_full`
- AND el editor sigue abierto
- AND la interfaz ofrece purgar `node_modules`, `.cache`, `target` y `.venv`.

#### Scenario: Vuelta a la normalidad
- GIVEN un entorno en `disk_full` cuya limpieza dejó el uso en 900 MB
- WHEN el worker evalúa el entorno
- THEN el estado es `ok`
- AND el entorno arranca de nuevo.

---

### Requirement: Cuota configurable por nodo

El límite por entorno MUST venir de la política del servidor (`disk_quota_mb`) y MUST NOT
estar escrito en el código. El valor `0` MUST significar sin cuota y MUST dejar solo la
medición y el aviso de disco del host.

#### Scenario: Cuota apagada
- GIVEN `disk_quota_mb = 0`
- WHEN un entorno supera cualquier tamaño
- THEN no se marca `disk_warning` ni `disk_full`
- AND el panel sigue mostrando el uso del disco del host.

---

### Requirement: Limpieza de volúmenes sin dueño

El sistema MUST poder listar los volúmenes con `LINK=0` y su tamaño, y MUST borrarlos solo
tras confirmación explícita. MUST NOT borrar un volumen con `LINK > 0` ni un volumen cuyo
nombre corresponda a un entorno registrado. Cada borrado MUST quedar en `audit_logs`.

#### Scenario: Volumen huérfano
- GIVEN un volumen sin contenedor asociado y sin entorno en base
- WHEN el administrador confirma la limpieza
- THEN el volumen se borra
- AND el espacio recuperado se informa en el resultado
- AND queda una entrada de auditoría por volumen.

#### Scenario: Volumen de un entorno activo
- GIVEN un volumen cuyo nombre pertenece a un entorno registrado
- WHEN se ejecuta la limpieza
- THEN el volumen no se borra
- AND se informa como omitido con el motivo.

---

### Requirement: Separación del almacenamiento de la base

Los datos de la base de notas MUST vivir en un almacenamiento distinto del que usan los
volúmenes de entorno, de modo que un disco de entornos lleno no impida registrar entregas
ni notas.

#### Scenario: Disco de entornos lleno
- GIVEN el almacenamiento de entornos al 100%
- WHEN un docente califica una entrega
- THEN la operación se registra en la base
- AND el entorno afectado sigue bloqueado por cuota, sin afectar al resto.

---

### Requirement: Camino a la cuota dura

El sistema MUST documentar que la cuota actual es blanda y MUST NOT presentarla como un
límite que el sistema operativo hace cumplir. Cuando exista un volumen lógico XFS con
`prjquota`, el sistema SHOULD asignar un identificador de proyecto por volumen y aplicar el
límite con `bhard`.

#### Scenario: Documentación honesta
- GIVEN la pantalla de cuota de un entorno
- WHEN el administrador la abre
- THEN ve que el límite es una cuota blanda con medición
- AND no se afirma que el sistema operativo rechace la escritura.
