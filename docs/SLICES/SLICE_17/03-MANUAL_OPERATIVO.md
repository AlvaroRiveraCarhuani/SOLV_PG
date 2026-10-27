# Manual Operativo — Slice 17: Panel de Salud y Recursos

## Qué cambió para el operador

- El historial del gráfico ya no se siembra al arrancar: se acumula desde el momento del despliegue (1 muestra/minuto, 7 días de retención).
- Estados `Sin Verificar` (unknown) y `Huérfano` (orphaned): un contenedor ausente libera cupo y el alumno lo recrea con restart; un daemon caído no destruye registros y la plataforma reintenta cada 30 s.
- El badge de salud muestra `Operativo / Degradado / Crítico` con el motivo (daemon caído, huérfanos, OOM, RAM sobre techo).
- La capacidad informada (X/Y activos) es la misma que aplica la admisión: `max_containers` de la política o RAM disponible (`capacity_basis` lo explica bajo la tarjeta).
- La ruta de disco que se muestra es la medida real del nodo.

## Consultas útiles

- Ver la salud completa: `GET /api/v1/admin/metrics/health?window=24h`
- Auditoría de transiciones: buscar `WORKSPACE_RECONCILED_*` en el registro de auditoría (actor `system`).
- Detección de daemon caído: `unknown` persistente más de 5 ciclos (~2,5 min) emite una alerta única.