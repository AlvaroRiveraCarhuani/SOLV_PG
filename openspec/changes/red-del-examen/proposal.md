# Proposal: Política de red del examen

## Contexto y Motivación

El entorno del alumno corre sobre `solv-traefik-net` y tiene salida a internet: la necesita
para instalar dependencias durante la práctica. Cuando ese mismo entorno se usa para rendir,
mantiene la salida. El examen, entonces, no está cerrado: el alumno puede consultar
cualquier cosa y descargar una solución.

`DOCKER-USER` ya bloquea el tráfico entre contenedores y hacia el host, pero no la salida a
internet. La política de red del examen quedó como decisión abierta en
`openspec/changes/separar-practica-y-examen/design.md`, y sin resolverla la expresión
"examen cerrado" no describe ninguna propiedad del sistema.

Un detalle técnico manda el diseño: cortar la red por completo no sirve, porque el editor se
sirve por HTTP a través de Traefik. Lo que hay que cortar es la **salida**, no el acceso.

## Alcance

- Política `exam_network` por ejercicio: `none` (por defecto) o `egress` (excepción
  declarada por el docente, auditada y visible para el alumno).
- Red interna sin puerta de enlace para el entorno de examen, con Traefik conectado a ella.
- Verificación previa: el proyecto base del examen MUST compilar y ejecutar sin red; si no,
  la publicación de la práctica de examen se rechaza con el motivo.
- Aviso visible al alumno: "este examen no tiene acceso a internet".

## Fuera de Alcance

- Bloqueo de dominios por lista: se corta la salida completa o no se corta.
- Política de red de la práctica: sigue como está.
- Detección de tráfico sospechoso o proxy de inspección.

## Capabilities

### New Capabilities

- `red-del-examen`: salida a internet apagada por defecto en el examen, con excepción
  declarada por ejercicio y verificación de que el proyecto base funciona sin red.

## Riesgos y Plan de Rollback

| Riesgo | Impacto | Control |
|---|---|---|
| El proyecto base necesita red para instalar dependencias y el alumno no puede trabajar | Alto académico | La publicación de la práctica de examen se rechaza si el proyecto no compila en un entorno sin salida; el docente lo corrige antes de la ventana |
| Un examen con base de datos necesita resolver el satélite | Medio | El satélite vive en una red interna sin puerta de enlace; se adjunta junto con la red del examen |
| La excepción `egress` se usa por costumbre y el examen deja de estar cerrado | Medio | La excepción exige motivo, queda auditada y se muestra en la pantalla del alumno |
| Traefik no alcanza el contenedor en la red nueva | Alto, examen inaccesible | Verificación de conectividad antes de abrir la ventana; vuelta atrás devolviendo el contenedor a la red actual |

**Rollback:** `exam_network` vuelve a `egress` con un cambio de política; el contenedor se
recrea en `solv-traefik-net` como hoy. No hay migración de datos.

## Impacto Documental

- ADR nuevo: "Aislamiento de salida en el entorno de examen", con entrada en el índice.
- `openspec/changes/separar-practica-y-examen/design.md`: cerrar la decisión abierta con
  referencia a este cambio.
- Manual del docente: qué significa un examen sin internet y cómo declarar la excepción.
