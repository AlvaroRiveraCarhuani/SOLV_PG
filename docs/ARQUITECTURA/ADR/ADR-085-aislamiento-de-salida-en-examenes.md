# ADR-085 — Aislamiento de salida a internet en ejercicios de examen

- **Estado:** Aceptado
- **Fecha:** 2026-11-16
- **Ámbito:** Red de los entornos, ciclo de vida del examen
- **Documentos relacionados:** `openspec/changes/red-del-examen`,
  `docs/ARQUITECTURA/ADR/ADR-083-perfil-de-entorno-por-proposito.md`,
  `infra/firewall/docker-user-rules.sh`

## Contexto

El entorno del alumno corría en `solv-traefik-net` con salida a internet: la
necesita para instalar dependencias en la práctica y la conservaba al rendir.
`DOCKER-USER` ya bloqueaba el tráfico entre contenedores y hacia el host, pero
no la salida. Sin política de red, «examen cerrado» no describía ninguna
propiedad del sistema.

Cortar con `NetworkMode: none` no sirve: el editor se sirve por HTTP y el alumno
se quedaría sin poder trabajar. Lo que hay que cortar es la salida, no el acceso.

Una red `--internal` de Docker no tiene puerta de enlace, y el daemon no permite
publicar puertos de un contenedor conectado solo a ella. Ese es el motivo de que
la ruta del editor tenga que pasar por un proxy.

## Decisión

### 1. La política vive en el ejercicio y la excepción se declara

`exercises.exam_network` vale `none` por defecto y `egress` para la excepción.
`exam_network_reason` es obligatorio con `egress` (mínimo 10 caracteres) y solo
se acepta en ejercicios con `purpose = exam`. La validación está en
`domain.ValidateExamNetwork` y corre en el alta y en la edición.

### 2. El examen corre en una red interna y el editor se publica por proxy

| Pieza | Nombre | Propósito |
| :--- | :--- | :--- |
| Red del examen | `solv-exam-net` | Bridge `Internal=true`, sin puerta de enlace |
| Red de satélites del examen | `solv-exam-satellites-net` | Red interna propia para motores gestionados |
| Proxy del editor | `<contenedor>-proxy` (`alpine/socat:1.8.0.0`) | Reenvía el puerto 3000 desde la red interna hacia la red de Traefik |

El contenedor del examen no publica puerto, no recibe `host.docker.internal` y
lleva la etiqueta `solv.exam.no-egress=true`. El proxy vive en las dos redes y
es quien publica `127.0.0.1:<puerto>` y las etiquetas de Traefik. Al detener el
entorno se retira el proxy antes que el contenedor, para no dejar un puerto
publicado apuntando a la nada.

La red normal de satélites (`solv-satellites-net`) sí tiene gateway: conectarla
al examen le devolvería la salida. Por eso los satélites se adjuntan a la red
interna propia, que comparten con el examen y con nadie más.

### 3. La excepción queda auditada y visible

Cada guardado con `egress` deja `EXAM_NETWORK_EGRESS_GRANTED` en `audit_logs`
con el motivo, el docente y el ejercicio. El alumno ve en la práctica si el
examen tiene salida o no, antes de abrir el entorno.

### 4. Defensa en profundidad en el host

`infra/firewall/docker-user-rules.sh` deriva de la etiqueta
`solv.exam.no-egress=true` una regla en `DOCKER-USER` que bloquea la salida del
contenedor aunque una segunda red le diera ruta. La topología es la defensa
principal; la regla cubre el error de configuración.

## Consecuencias

- Un examen no puede consultar ni descargar nada, y el editor sigue disponible.
- La excepción es explícita, con motivo, auditada y visible para el alumno.
- El proxy agrega un contenedor por examen abierto; se retira con el entorno.
- Si el ejercicio no se puede leer al reactivar, se conserva la salida: cortar
  la red de un examen en curso por un error de lectura sería peor que no cortar.
- El proxy necesita una red de Traefik presente (`solv-traefik-net` o `solv_net`);
  sin ella el arranque del examen falla en lugar de quedar inaccesible.
- Al publicar un examen cerrado de Entorno de Desarrollo se corre el proyecto
  base en un volumen temporal sobre la red interna (setup y arranque declarado,
  corte a los 5 minutos). Si necesita red, la publicación se rechaza con el paso
  y las últimas líneas del log; el resultado queda en `exam_offline_status` y se
  ve en el checklist del editor. Un fallo de infraestructura no bloquea: se
  registra como `skipped`.
- Pendiente: la corrida manual en el nodo con un proyecto que funciona sin red y
  otro que necesita una instalación remota.
