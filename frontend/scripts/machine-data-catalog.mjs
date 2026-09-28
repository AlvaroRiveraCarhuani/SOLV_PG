// Catálogo mantenido de nombres técnicos que exigen marca tipográfica.
// Cada entrada: nombre, categoría, ejemplo visible y origen (fecha/PR).
// Agregar un campo técnico público exige registrarlo aquí + agregar fixture en
// machine-data-gate.test.mjs. Eliminar una regla exige actualizar su contrato.
// No inferir significado de texto arbitrario: lo no registrado no se adivina.
export const MACHINE_DATA_CATALOG = [
  { name: 'id', category: 'identificador', example: 'workspace_id', since: '2026-09-28/base' },
  { name: 'code', category: 'identificador', example: 'course code', since: '2026-09-28/base' },
  { name: 'uuid', category: 'identificador', example: 'uuid v4', since: '2026-09-28/base' },
  { name: 'hash', category: 'identificador', example: 'commit hash', since: '2026-09-28/base' },
  { name: 'version', category: 'version-tag', example: 'docker_version', since: '2026-09-28/base' },
  { name: 'verdict', category: 'veredicto-log', example: 'AC/WA/TLE', since: '2026-09-28/base' },
  { name: 'level', category: 'veredicto-log', example: 'log level', since: '2026-09-28/base' },
  { name: 'tag', category: 'version-tag', example: 'image_tag', since: '2026-09-28/base' },
  { name: 'image_tag', category: 'version-tag', example: 'img:1.2.3', since: '2026-09-28/base' },
  { name: 'workspace_id', category: 'identificador', example: 'ws-uuid', since: '2026-09-28/base' },
  { name: 'docker_version', category: 'version-tag', example: '27.x', since: '2026-09-28/base' },
  { name: 'uptime_seconds', category: 'metrica-recurso', example: '3600s', since: '2026-09-28/base' },
  { name: 'path', category: 'ruta-url', example: '/var/log/app.log', since: '2026-09-28/base' },
  { name: 'base_ram_mb', category: 'metrica-recurso', example: '512 MB', since: '2026-09-28/base' },
  { name: 'currentRAM', category: 'metrica-recurso', example: 'RAM actual', since: '2026-09-28/base' },
  { name: 'currentCPU', category: 'metrica-recurso', example: 'CPU actual', since: '2026-09-28/base' },
  { name: 'ttl_remaining_seconds', category: 'duracion-tiempo', example: 'TTL 300s', since: '2026-09-28/base' },
  { name: 'resource_type', category: 'metrica-recurso', example: 'cpu/ram', since: '2026-09-28/base' },
  { name: 'docker_image', category: 'version-tag', example: 'img:tag', since: '2026-09-28/base' },
  { name: 'memory_used_mb', category: 'metrica-recurso', example: '256 MB', since: '2026-09-28/base' },
  { name: 'memory_limit_mb', category: 'metrica-recurso', example: '1024 MB', since: '2026-09-28/base' },
  { name: 'cpu_cores', category: 'metrica-recurso', example: '2 cores', since: '2026-09-28/base' },
  { name: 'cpu_percent', category: 'metrica-recurso', example: '45%', since: '2026-09-28/base' },
  { name: 'disk_percent', category: 'metrica-recurso', example: '70%', since: '2026-09-28/base' },
  { name: 'containers_active', category: 'metrica-recurso', example: '3 activos', since: '2026-09-28/base' },
  { name: 'containers_max', category: 'metrica-recurso', example: 'máx 10', since: '2026-09-28/base' },
  { name: 'containers_hibernated', category: 'metrica-recurso', example: '2 hibernados', since: '2026-09-28/base' },
  { name: 'ram_percent', category: 'metrica-recurso', example: '62%', since: '2026-09-28/base' },
  { name: 'ram_used_gb', category: 'metrica-recurso', example: '1.2 GB', since: '2026-09-28/base' },
  { name: 'timestamp', category: 'duracion-tiempo', example: '2026-09-28T12:00', since: '2026-09-28/base' },
  { name: 'duration_ms', category: 'duracion-tiempo', example: '120 ms', since: '2026-09-28/base' },
  { name: 'ramPercentComputed', category: 'metrica-recurso', example: '62% calc', since: '2026-09-28/base' },
  { name: 'ramUsedGB', category: 'metrica-recurso', example: '1.2 GB', since: '2026-09-28/base' },
  { name: 'ramTotalGB', category: 'metrica-recurso', example: '4 GB', since: '2026-09-28/base' },
  { name: 'diskUsedGB', category: 'metrica-recurso', example: '20 GB', since: '2026-09-28/base' },
  { name: 'diskTotalGB', category: 'metrica-recurso', example: '50 GB', since: '2026-09-28/base' },
  { name: 'concurrencyPercent', category: 'metrica-recurso', example: '80%', since: '2026-09-28/base' },
  { name: 'getMemoryPercent', category: 'metrica-recurso', example: 'método %', since: '2026-09-28/base' },
  { name: 'formatUptime', category: 'duracion-tiempo', example: 'método uptime', since: '2026-09-28/base' },
  { name: 'formatTTL', category: 'duracion-tiempo', example: 'método TTL', since: '2026-09-28/base' },
];

export const TECHNICAL_MEMBER_NAMES = new Set(MACHINE_DATA_CATALOG.map((entry) => entry.name));
