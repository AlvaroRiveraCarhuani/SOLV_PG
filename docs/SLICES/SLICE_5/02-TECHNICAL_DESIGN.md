# Diseño Técnico - SLICE 5: QoS, Auto-Bursting y Profiler Empírico

## 1. Arquitectura del Worker QoS (`QoSOrchestratorWorker`)

```
                          ┌───────────────────────────┐
                          │   QoSOrchestratorWorker   │ (Loop cada 100ms/10s)
                          └─────────────┬─────────────┘
                                        │
        ┌───────────────────────────────┼───────────────────────────────┐
        ▼                               ▼                               ▼
 [1. Monitor de Host]            [2. Auto-Bursting]           [3. Hibernación Dual]
  gopsutil lee RAM               Si RAM > 80% del límite,     Si Heartbeat ok PERO
  ¿Disponible < OOM_GUARD_MB?    invoca ContainerUpdate       CPU < 0.5% & Net < 1KB (15m)
  ==> Bloquea con HTTP 503       RAM +256MB (hasta 2GB)       ==> Destruye contenedor
```

## 2. Cálculo del Porcentaje Delta de CPU

$$\text{CPU\%} = \frac{\Delta \text{ContainerCPU}}{\Delta \text{SystemCPU}} \times \text{OnlineCPUs} \times 100.0$$

## 3. Diagrama del Profiler Empírico `cmd/autotune/main.go`

1. **Fase 1 (Baseline):** Mide la huella del host en reposo (Ubuntu + Docker + Postgres + Traefik = 408MB).
2. **Fase 2 (Estrés Progresivo):** Inyecta contenedores dummy incrementales.
3. **Fase 3 (Circuit-Breaker):** Interrumpe si Swap Delta $\ge 10\text{MB}$ o RAM disponible $\le 1000\text{MB}$.
4. **Fase 4 (Salida):** Genera `autotune_report.json` e imprime `OOM_GUARD_MB=1408`.
