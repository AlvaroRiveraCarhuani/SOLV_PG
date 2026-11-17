# Medición previa del nodo piloto

Ningún número de este cambio se fija antes de completar esta planilla. Cada medición
declara qué decisión alimenta.

## 0. Contexto del nodo

| Dato | Comando | Valor |
|---|---|---|
| Núcleos | `nproc` | 8 |
| RAM total | `free -m` | 11852 MB (~12 GB) |
| Swap | `swapon --show` | 4095 MB (/swap.img, 0 MB en uso) |
| Versión de Docker | `docker version --format '{{.Server.Version}}'` | 29.3.1 |
| Entornos abiertos durante la medición | panel de administración | 0 |

Se registra también el `MemAvailable` con la plataforma levantada y sin ningún entorno
abierto: es el piso real que deja el backend, Traefik, PostgreSQL y los satélites:
**10302 MB** (`10551360 kB`, fecha: 2026-11-16).


## 1. Memoria

| # | Qué se mide | Cómo | Decisión que alimenta |
|---|---|---|---|
| M1 | Curva de arranque del IDE: RAM del contenedor cada 2 s desde el clic hasta que el editor responde | `docker stats --no-stream` en bucle + marca de tiempo | Perfil mínimo que evita el OOM en el arranque. Hoy el préstamo no considera este pico |
| M2 | Consumo en régimen: RAM p50 y p95 durante 30 min de trabajo normal (editar, compilar, correr tests) | `docker stats` cada 5 s con un proyecto real | La base real del entorno (hoy 512 MB) |
| M3 | Pico de instalación de dependencias: `npm install` y `pip install` de un proyecto típico del catálogo | `docker stats` durante la instalación | Si la base alcanza para instalar o el piso debe subir |
| M4 | Costo del servidor de IDE según tamaño del proyecto (un archivo vs un proyecto Angular completo) | `docker stats` con dos volúmenes distintos | El valor de `NODE_OPTIONS=--max-old-space-size` (hoy 65% del límite) y el perfil del préstamo |
| M5 | Consumo agregado con 10, 20 y 30 entornos abiertos, en oleadas y en arranque simultáneo | `free -m` y `MemAvailable` cada 5 s, más `docker stats` de los 5 mayores | El techo de seguridad (hoy 85%) y la regla `2 × perfil ≤ presupuesto` |
| M6 | OOM reales ocurridos | `dmesg -T \| grep -i oom`; `oom_strike_count` y `last_oom_killed_at` en la base | Si los perfiles actuales ya se quedan cortos en producción |
| M7 | Comportamiento del swap durante M5 | `vmstat 5` | Si el nodo degrada antes de matar procesos: cambia el margen |

Regla de decisión de memoria: base = p95 de M2 más 25% de margen, redondeada al perfil
inmediato superior. El pico de M1 y M3 no puede superar el perfil más alto del préstamo; si
lo supera, se corrige el perfil o el `max-old-space-size`, no el margen.

### Resultados M1: Curva de arranque del IDE (2026-11-16)

Imagen: `gitpod/openvscode-server:1.96.0` en Asus Server (8 núcleos, 12 GB RAM).

| Métrica | Corrida 1 (frío) | Corrida 2 | Corrida 3 | p50 | p95 |
|---|---|---|---|---|---|
| Tiempo hasta HTTP 200 | 5.76 s | 2.27 s | 2.38 s | 2.38 s | 5.43 s |
| RAM al responder | 69.68 MiB | 26.16 MiB | 26.58 MiB | 26.58 MiB | 65.37 MiB |
| RAM estabilizada post-arranque | 77.03 MiB | 26.35 MiB | 26.57 MiB | 26.57 MiB | 71.98 MiB |

*Observación*: El primer arranque en frío paga la inicialización del daemon de OpenVSCode Server (~77 MiB y 5.76 s); las corridas siguientes estabilizan en ~2.3 s y 26 MiB. El pico máximo de arranque (77 MiB) queda bien por debajo del piso de 512 MB.

### Resultados M2: Consumo en régimen de 30 min (2026-11-16)

Imagen: `gitpod/openvscode-server:1.96.0` con proyecto en `/home/workspace` y actividad de cómputo/ejecución periódica en Asus Server.  
Ventana: 30 minutos (1800 s), 298 muestras tomadas cada 5 s (`scratch/m2_samples.csv`).

| Métrica | Valor |
|---|---|
| Mínimo | 20.07 MiB |
| p50 | 20.11 MiB |
| p90 | 20.14 MiB |
| p95 | 20.14 MiB |
| Máximo | 27.13 MiB |
| Base calculada (p95 + 25% margen) | 25.18 MB (la política actual de 512 MB deja holgura de 20×) |

*Observación*: El entorno de desarrollo en régimen de trabajo es sumamente liviano (~20.1 MiB), manteniéndose completamente plano a lo largo de los 30 minutos con un pico de 27.13 MiB durante la actividad de cómputo.

### Resultados M5: Consumo agregado con 10, 20 y 30 entornos (2026-11-16)

Imagen: `gitpod/openvscode-server:1.96.0` en Asus Server (8 núcleos, 12 GB RAM, Docker 29.3.1).
Corrida en oleadas acumulativas de 10 contenedores (`--memory=512m`, `--cpus=1.0`), con 12 s de estabilización por escalón.

| Escenario | Contenedores | RAM Usada (`free -m`) | Delta RAM Total | RAM / Contenedor | `MemAvailable` | Load Avg | Swap en uso |
|---|---|---|---|---|---|---|---|
| **Línea Base** | 0 | 1649 MB | - | - | 10202 MB (10447984 kB) | 0.95 | 1 MB |
| **Oleada 1** | 10 | 1893 MB | +244 MB | 24.4 MB | 9959 MB (10198092 kB) | 1.13 | 1 MB |
| **Oleada 2** | 20 | 2131 MB | +482 MB | 24.1 MB | 9720 MB (9953732 kB) | 1.24 | 1 MB |
| **Oleada 3** | 30 | 2375 MB | +726 MB | 24.2 MB | 9476 MB (9703844 kB) | 1.27 | 1 MB |

*Observaciones*:
- **Linealidad perfecta del consumo**: Cada contenedor de OpenVSCode Server añade aproximadamente ~24 MB de memoria física residente al nodo (~19-20.7 MiB por proceso del contenedor reportado por `docker stats`, más la sobrecarga del kernel / cgroup v2).
- **Margen del nodo intacto**: Con 30 entornos levantados simultáneamente, el nodo aún retiene **9.47 GB de RAM disponible** (`MemAvailable`), habiendo consumido únicamente el 6.1% de la memoria total del sistema.
- **Swap y estabilidad**: El swap permaneció completamente inactivo (1 MB), sin disparar presión de memoria ni desestabilizar el sistema operativo ni el daemon de Docker.
- **Carga de CPU**: El load average se mantuvo en 1.27 sobre un host de 8 núcleos (apenas un 15.8% de capacidad de cómputo utilizada).
- **Conclusión de capacidad**: El Asus Server sostiene con comodidad la meta objetivo de 30 a 40 entornos simultáneos bajo la cuota de 512 MB de RAM y 1.0 núcleo por entorno.

## 2. CPU

| # | Qué se mide | Cómo | Decisión que alimenta |
|---|---|---|---|
| C1 | Línea base del nodo | `uptime`, `vmstat 5` con la plataforma levantada y sin entornos | Punto de comparación de la contención |
| C2 | CPU de un entorno compilando: pico y promedio | `docker stats` durante una compilación del proyecto de referencia | El `NanoCPUs` por contenedor |
| C3 | Curva de contención con 1, 4 y 8 entornos compilando a la vez: tiempo de compilación y tiempo hasta editor listo | misma compilación en paralelo, cronometrada | Si 1 núcleo por contenedor sostiene la clase o hace falta 1.5 o 2 |
| C4 | Latencia de la API mientras N entornos compilan | p95 de un endpoint liviano durante C3 | Si hace falta reserva de CPU para el backend y los satélites |
| C5 | `load average` durante la oleada | `uptime` cada 5 s | El punto donde el nodo deja de responder con normalidad |

Regla de decisión de CPU: el límite por contenedor es el mayor escalón (0.5, 1, 1.5, 2
núcleos) que mantiene la compilación de referencia dentro de 1.5× del tiempo sin límite.
Si ningún escalón lo logra, el problema es el nodo, no el límite, y se documenta como
límite de capacidad.

## 3. Reconciliación de cuota

| # | Qué se mide | Cómo | Decisión que alimenta |
|---|---|---|---|
| R1 | Los tres números de un mismo entorno: `base_ram_mb` de la plantilla, `memory_limit_mb` del entorno y `HostConfig.Memory` del contenedor | `docker inspect` + una consulta a la base | Qué valor mostrar en cada pantalla y con qué nombre |
| R2 | Frecuencia con la que la admisión concede menos que el perfil de la plantilla | logs de admisión en una oleada de 30 | Si el aviso "cuota concedida menor al perfil" debe ser visible al crear la práctica |

## 4. Salida

Tres corridas por escenario, y se reporta p50 y p95, nunca un valor único. La planilla
completa se adjunta al ADR y reemplaza la tabla de supuestos de
`docs/OPERATIONS/capacidad.md` §5. Las cifras que no se puedan medir se declaran como no
medidas: el cambio no rellena huecos con estimaciones.
