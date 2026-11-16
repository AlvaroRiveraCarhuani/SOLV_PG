# Proposal: Cuota de disco por entorno y separación del disco de la base

## Contexto y Motivación

Medición del nodo piloto (`asus-server`, 2026-11-16):

| Dato | Valor |
|---|---|
| Sistema de archivos de `/var/lib/docker` | ext4 sobre LVM (`ubuntu-vg-ubuntu-lv`) |
| Disco total / usado / libre | 98 GB / 59 GB / 34 GB (64%) |
| Volúmenes locales | 168, 19,9 GB, **95% recuperable** (18,99 GB) |
| Imágenes | 28, 18,6 GB, 3,6 GB recuperables |
| Contenedores activos | 1 (`solv_dev_db`) |
| Base de datos | volumen `pg_data` dentro del mismo `/var/lib/docker` |

No existe cuota de disco por entorno: lo confirma `docs/GOBERNANZA/INVENTARIO_BAAS.md`
punto 08 y no hay `StorageOpt` en el código. El volumen del alumno puede crecer hasta
llenar el disco.

Con el disco al 64% y 34 GB libres, el escenario no es hipotético: un grupo de 40 alumnos
instalando dependencias lo llena. Y como la base de notas vive en el mismo sistema de
archivos, un disco lleno deja el sistema sin poder registrar entregas ni notas.

Dos hechos técnicos acotan las opciones:

- `StorageOpt size` limita la capa escribible del contenedor, **no** los volúmenes
  nombrados, que es donde vive el workspace.
- Los contenedores corren con uid 1000. El kernel no distingue alumnos por dueño de
  archivo, así que las cuotas por usuario no sirven.

## Alcance

- **Recuperar lo que ya está ocupado sin dueño**: volúmenes con `LINK=0` y caché de
  construcción, con auditoría y sin tocar ningún volumen de alumno.
- **Medir el disco de cada entorno** y exponerlo junto a la cuota.
- **Cuota blanda con aplicación**: aviso al 80%, bloqueo al 100% con acción de limpieza
  (purgar cachés de dependencias y builds).
- **Separar la base de notas** del sistema de archivos de los entornos.
- **Dejar preparado el camino a la cuota dura**: un volumen lógico nuevo formateado XFS con
  `prjquota` para los volúmenes de entorno, con el asignador de cuota por volumen.

## Fuera de Alcance

- Límites de I/O (`blkio`): cambio aparte.
- Retención de imágenes y política de actualización.
- Cuota de correo, respaldos o logs de la plataforma.

## Capabilities

### New Capabilities

- `cuota-de-disco`: medición y techo por entorno, con aviso, bloqueo y limpieza, y
  separación del almacenamiento de la base de notas.

## Riesgos y Plan de Rollback

| Riesgo | Impacto | Control |
|---|---|---|
| Borrar un volumen de alumno por error al limpiar huérfanos | Alto, irreversible | Solo se borran volúmenes con `LINK=0`, con lista previa, auditoría por volumen y confirmación por frase; nunca un volumen cuyo nombre coincida con un entorno en base |
| El bloqueo al 100% impide al alumno borrar lo que le sobra | Alto, deja al alumno sin salida | El bloqueo detiene el contenedor y ofrece la limpieza de cachés desde la interfaz; no monta en solo lectura hasta que el excedente se reduce |
| El costo de medir cada volumen degrada el nodo | Medio | Se mide con `docker system df -v` (una lectura del daemon) y no con `du` sobre el árbol completo |
| Mover la base de notas rompe el arranque | Alto | Ventana de mantenimiento, respaldo previo verificado, y vuelta atrás cambiando el montaje del volumen |
| XFS con `prjquota` exige formatear | Medio, requiere disco | Se crea un volumen lógico nuevo; el actual no se formatea |

**Rollback:** la cuota blanda se apaga con `DISK_QUOTA_MODE=off`. La separación de la base
se revierte apuntando el contenedor al volumen anterior, que no se borra hasta validar el
nuevo. El camino XFS no se aplica hasta tener el volumen lógico listo.

## Impacto Documental

- ADR nuevo: "Cuota de disco por entorno, medición y separación del almacenamiento de la
  base", con entrada en `docs/ARQUITECTURA/ADR/INDEX.md`.
- `docs/OPERATIONS/capacidad.md`: agregar la dimensión de disco, hoy ausente de la tabla.
- `docs/GOBERNANZA/INVENTARIO_BAAS.md` punto 08: actualizar el estado.
- `docs/ARQUITECTURA/STORAGE_STRATEGY.md`: reflejar el volumen lógico nuevo y el rol del
  HDD de datos fríos.
