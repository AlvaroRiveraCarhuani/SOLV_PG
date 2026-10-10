# ADR-043: Mitigaciones de Seguridad para el Acceso al Socket de Docker

## Estado
Aceptado

## Contexto
El backend de SOLV interactúa directamente con el daemon de Docker mediante `/var/run/docker.sock` para la creación y orquestación de contenedores efímeros y workspaces de IDE Persistente. El acceso de escritura al socket Unix de Docker otorga privilegios equivalentes a `root` sobre el sistema operativo host si la aplicación es vulnerada.

## Decisión
Para la fase de piloto en entorno controlado (Single-Node / On-Premise), se aceptan las mitigaciones perimetrales y de aislamiento existentes sin añadir proxies adicionales intermediarios, posponiendo el endurecimiento avanzado (Docker Rootless o Socket Proxy limitado) para el despliegue multi-tenant público.

### Mitigaciones Activas en Piloto
1. **Permisos de Socket Restringidos**: El socket `/var/run/docker.sock` mantiene permisos `0660` asignados estrictamente al grupo de sistema `root:docker`.
2. **Usuario Dedicado de Backend**: El proceso backend corre como un usuario de sistema dedicado sin privilegios sudoers en el host.
3. **Firewall Perimetral**: Reglas de filtrado estricto `iptables` / `DOCKER-USER` que bloquean tráfico hacia interfaces del host.
4. **Endurecimiento de Contenedores de Ejecución**:
   - `network_mode: none` en runners de evaluación.
   - Filesystem root montado como `read_only`.
   - Remoción total de capacidades del kernel: `--cap-drop ALL`.
   - Límites cgroups de memoria (`memory_limit_mb`) y CPU sin swap.

### Ruta de Endurecimiento Futuro (Decisión Posponible)
- **Implementación de Socket Proxy (e.g. `tecnativa/docker-socket-proxy`)** o migración a **Docker Rootless**.
- **Criterio de Activación**:
  1. Expansión a entorno Multi-Tenant Público.
  2. Más de 1 institución o universidad compartiendo el mismo nodo físico en producción.

## Consecuencias
- Mantiene la simplicidad operativa y baja latencia de orquestación durante la etapa inicial del piloto.
- Documenta explícitamente el vector de riesgo y establece un gatillo claro para la transición hacia arquitectura Zero-Trust de socket.
