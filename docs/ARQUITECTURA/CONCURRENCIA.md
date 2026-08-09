# Definición Formal de Métrica de Concurrencia y Resiliencia Físico-Arquitectónica

## Métrica de Estrés para la Defensa de Tesis (CRIT-22)

> **Métrica Oficial:**
> *"La plataforma SOLV soporta **40 usuarios conectados concurrentemente**, generando un pico de **~10 a 15 contenedores activos simultáneos** en el hardware físico On-Premise (servidor Asus de 12GB RAM). Esta relación se logra gracias a la hibernación inteligente basada en inactividad y a la naturaleza asíncrona de las sesiones de laboratorio.*
>
> *Ante escenarios de saturación crítica de memoria RAM física o límite de Hardware Allocatable, el sistema de control de admisión aplica una **degradación elegante (HTTP 503 Service Unavailable con cabeceras de cola de reintento)** evidenciada en tiempo real en Grafana y métricas Prometheus (`solv_host_oom_guard_bytes`), protegiendo al host Asus de Kernel Panics y asegurando la estabilidad del sistema operacional."*

---

## Modelo de Concurrencia Activa vs Conectada

```text
+-------------------------------------------------------------------------------+
|              40 ESTUDIANTES CONECTADOS CONCURRENTEMENTE (HTTP / SSO)          |
+-------------------------------------------------------------------------------+
       |                                                                |
       v (Lectura de guías, edición de código en frontend)              v (Inactividad >500ms)
+------------------------------------+                         +-----------------------------+
| 10 - 15 Contenedores Activos (RAM) |                         | Workspaces Hibernados (SSD) |
| (OpenVSCode + Juez en ejecución)   |                         | (RAM liberada a 0MB)        |
+------------------------------------+                         +-----------------------------+
```

## Mecanismos de Protección y Degradación Elegante

1. **Hibernación Dual:** Liberación de memoria física cuando la inactividad supera el umbral configurado (`DefaultInactivityTimeout = 500ms` en tests / 15min en producción).
2. **QoS Host Admission Control:** Inspección de la memoria disponible física (`GetHostMemoryStats`). Si la RAM libre cae por debajo del margen de resguardo OOM Guard (15% RAM / 1408 MB), el backend rechaza nuevas solicitudes con HTTP 503 evitando caídas del sistema.
3. **Observabilidad Físico-Arquitectónica:** Exposición de métricas Prometheus (`solv_active_workspaces_total`, `solv_host_available_memory_bytes`, `solv_host_oom_guard_bytes`, `solv_orphan_containers_reclaimed_total`, `solv_cert_expiry_days`) visualizadas en tableros de Grafana.
