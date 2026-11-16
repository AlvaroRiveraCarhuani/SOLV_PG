# Medición previa del nodo piloto

Ningún número de este cambio se fija antes de completar esta planilla. Cada medición
declara qué decisión alimenta.

## 0. Contexto del nodo

| Dato | Comando | Valor |
|---|---|---|
| Núcleos | `nproc` | |
| RAM total | `free -m` | |
| Swap | `swapon --show` | |
| Versión de Docker | `docker version --format '{{.Server.Version}}'` | |
| Entornos abiertos durante la medición | panel de administración | |

Se registra también el `MemAvailable` con la plataforma levantada y sin ningún entorno
abierto: es el piso real que deja el backend, Traefik, PostgreSQL y los satélites.

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
