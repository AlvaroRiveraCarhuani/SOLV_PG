# Diseño Técnico - SLICE 6: Persistencia Relacional, EWMA, Zombie Collector y Telemetría

## 1. Esquema de Base de Datos PostgreSQL (`lab_template_profiles`)

```sql
CREATE TABLE IF NOT EXISTS lab_template_profiles (
    signature_hash VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    base_image VARCHAR(255) NOT NULL,
    setup_script TEXT NOT NULL DEFAULT '',
    resource_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Estructura del Documento JSONB (`resource_profile`):
```json
{
  "signature_hash": "a591a6d4c...",
  "base_memory_mb": 256,
  "max_quota_mb": 554,
  "ewma_state": {
    "current_ewma_mb": 442.95,
    "sample_count": 11,
    "last_updated_at": "2026-07-28T20:54:40Z"
  }
}
```

---

## 2. Flujo del Profiler EWMA con Control de Concurrencia Dual

```
                     ┌────────────────────────────────────────┐
                     │ Sesión de Laboratorio Finalizada / OOM │
                     └───────────────────┬────────────────────┘
                                         │
                                         ▼
                     ┌────────────────────────────────────────┐
                     │ EWMAProfilerService (RecordSessionPeak)│
                     └───────────────────┬────────────────────┘
                                         │
                   ┌─────────────────────┴─────────────────────┐
                   ▼                                           ▼
      [1. Bloqueo sync.Mutex]                    [2. Transacción PostgreSQL]
      Garantiza ordenamiento local               SELECT ... FOR UPDATE
      en la memoria de Go                        Recalcula S_t = 0.2*Y_t + 0.8*S_{t-1}
                                                 Aplica MaxQuota = S_t * 1.25
                                                 Actualiza JSONB atómicamente
```

---

## 3. Arquitectura del Reconciliador de Huérfanos (`ZombieCollectorWorker`)

```
                           ┌─────────────────────────┐
                           │  ZombieCollectorWorker  │ (Ticker 30s)
                           └────────────┬────────────┘
                                        │
                                        ▼
                           ┌─────────────────────────┐
                           │      TryLock (sync)     │ Evita ejecuciones
                           └────────────┬────────────┘ superpuestas
                                        │
                     ┌──────────────────┴──────────────────┐
                     ▼                                     ▼
        ListAllManagedContainers                GetActiveWorkspaces
           (Docker API Engine)                   (PostgreSQL DB)
                     │                                     │
                     └──────────────────┬──────────────────┘
                                        │
                                        ▼
                       Diferencia: [Docker - PostgreSQL]
                                        │
                         (Si existe contenedor huérfano)
                                        ▼
                           StopAndRemoveContainer (Docker)
                                        │
                          reclaimedCountAtomic++ (Counter)
```

---

## 4. Estructura de Métricas Prometheus (`GET /metrics`)

| Nombre de la Métrica | Tipo | Descripción |
| :--- | :--- | :--- |
| `solv_active_workspaces_total` | Gauge | Número total de entornos activos (running/pending). |
| `solv_host_available_memory_bytes` | Gauge | Memoria RAM física disponible en el host (bytes). |
| `solv_host_oom_guard_bytes` | Gauge | Umbral seguro del host calibrado (`OOM_GUARD_MB=1408`). |
| `solv_orphan_containers_reclaimed_total` | Counter | Total de contenedores zombies destruidos por el collector. |
