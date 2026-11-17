# Tasks: Cuota de disco por entorno y separación del disco de la base

## Phase 1: Recuperar el espacio que ya está ocupado

- [x] 1.1 Listar volúmenes con `LINK=0` y su tamaño con `docker system df -v`; guardar la lista con fecha en `docs/OPERATIONS/capacidad.md`.
- [x] 1.2 Cruzar esa lista contra los entornos registrados en base y marcar los que no corresponden a ningún entorno.
- [x] 1.3 Borrar los huérfanos confirmados uno por uno, anotando el espacio recuperado y la auditoría.
- [x] 1.4 Revisar imágenes sin contenedor y caché de construcción; recuperar solo lo que no sea imagen fijada por el catálogo.
- [x] 1.5 Registrar el espacio libre resultante como línea base de la cuota.
- [x] 1.6 Implementar en el panel el inventario de volúmenes con `LINK=0`, el cruce contra los entornos registrados, el borrado por nombre con frase y la auditoría por volumen (`DiskOrphanService`, `GET/POST /api/v1/admin/server/storage/orphans`). La corrida sobre el nodo sigue siendo la tarea 1.1–1.5.

## Phase 2: Medición

- [x] 2.1 Agregar el tamaño medido al dominio y la lectura por volumen desde el daemon: `Domain.EvaluateDiskQuota` + `MeasurePracticeEnvironmentVolumeMB` (contenedor helper con `du`, volumen en solo lectura).
- [x] 2.2 Persistir `used_mb` por entorno y exponerlo en el DTO.
- [x] 2.3 Worker de medición cada 5 minutos, con una sola lectura del daemon por ciclo.
- [x] 2.4 Pruebas unitarias del cálculo de porcentaje y estado (`disk_quota_test.go`: sin cuota, aviso, exacta, excedida, umbrales inválidos y redondeo).

## Phase 3: Cuota blanda con aplicación

- [x] 3.1 Agregar `disk_quota_mb`, `disk_warning_pct` y `disk_block_pct` a la política del servidor, con `0` como sin cuota.
- [x] 3.2 Implementar las transiciones `ok` → `disk_warning` → `disk_full` y la vuelta a `ok`.
- [x] 3.3 Al entrar en `disk_full` no se detiene el contenedor: se marca el estado y se frena la entrega con 409, porque cerrar el editor en medio del trabajo pierde código sin guardar. Se detiene solo por presión del nodo, y el alta nueva se rechaza por debajo del 5% libre (`ErrHostDiskExhausted`, 503).
- [x] 3.4 Implementar la limpieza de cachés declaradas desde la interfaz y el rearranque del entorno.
- [x] 3.5 Mostrar en la ficha del entorno el uso, el límite y el aviso, dejando claro que el límite es una cuota blanda.
- [x] 3.6 Pruebas de integración con un volumen de prueba: aviso, bloqueo, limpieza y vuelta a `ok`.

## Phase 4: Separación de la base de notas

- [ ] 4.1 Verificar el espacio libre del grupo de volúmenes para decidir entre volumen lógico propio o HDD de datos fríos.
- [ ] 4.2 Crear el almacenamiento nuevo, migrar `pg_data` con respaldo verificado previo y ventana de mantenimiento.
- [ ] 4.3 Comprobar el arranque, la restauración de un respaldo y la latencia de las consultas.
- [ ] 4.4 Dejar documentado el procedimiento de vuelta atrás y no borrar el volumen anterior hasta validar.

## Phase 5: Camino a la cuota dura

- [ ] 5.1 Crear un volumen lógico para volúmenes de entorno y formatearlo XFS con `prjquota`.
- [ ] 5.2 Implementar el asignador de identificador de proyecto por volumen y el límite `bhard`.
- [ ] 5.3 Mover los volúmenes existentes al sistema de archivos nuevo con el entorno detenido.
- [ ] 5.4 Verificar el rechazo de escritura al llegar al límite y la lectura del reporte de cuotas.

## Phase 6: Verificación y documentación

- [x] 6.1 Actualizar `docs/OPERATIONS/capacidad.md` con la dimensión de disco y el techo fijado.
- [x] 6.2 Actualizar `INVENTARIO_BAAS.md` punto 08 y la estrategia de almacenamiento.
- [x] 6.3 Escribir el ADR y su entrada en `INDEX.md` (ADR-084, cuotas de CPU, procesos y disco; cubre también el cambio de dimensionamiento).
- [ ] 6.4 Ejecutar el plan de impacto de `tools/test-impact` sobre las suites afectadas.
