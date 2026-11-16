# Proposal: Separar el entorno de práctica del entorno de examen

## Contexto y Motivación

Hoy SOLV tiene un solo modelo de entorno. Un `PracticeEnvironment` se identifica por
`(student_id, subject_id, exercise_id)`, se aprovisiona con la misma cuota, la misma
política de hibernación y la misma captura de tecleo, sin importar si el ejercicio es de
práctica (`purpose = "class"`) o de examen (`purpose = "exam"`). La única diferencia
aparece al final: al cerrar el examen el volumen pasa a solo lectura (`frozen_at`).

Esa uniformidad tiene dos consecuencias medibles:

1. **La práctica paga el costo del examen.** El alumno que quiere romper, reinstalar y
   volver a empezar trabaja bajo reglas pensadas para que no se mueva, y su tecleo se
   registra desde la primera sesión, sin que nadie vaya a mirarlo.
2. **El examen no arranca en un estado declarado.** El entorno se crea la primera vez que
   el alumno abre el ejercicio. Si el ejercicio de examen estuvo visible antes de la
   ventana, el espacio donde rinde ya existía, con lo que hubiera dentro. La copia que se
   califica se toma al entregar (`SnapshotPracticeEnvironmentVolume`), así que el
   artefacto es inmutable; lo que no está acotado es el estado de partida.

Separar los dos momentos es lo que permite afirmar, sin depender de la buena fe de nadie:
*"se practica en un espacio flexible y se rinde en un espacio creado para el examen"*.

## Alcance

- **Dominio (`core/domain`)**: perfil de entorno (`EnvironmentProfile`) derivado del
  propósito del ejercicio, con cupo de memoria, política de hibernación, captura de
  tecleo y política de estado de partida. Sin números nuevos: el perfil define **cuál**
  valor aplica; los valores concretos se ajustan en el cambio de dimensionamiento.
- **Servicios (`core/services`)**: resolución del perfil al abrir, creación del espacio de
  examen en la apertura (nunca reutilización de un espacio previo del mismo ejercicio),
  exención de hibernación automática en examen, captura de tecleo acotada al examen y
  aviso accionable cuando la copia entregada excede los límites actuales.
- **Infraestructura (`infrastructure/docker`, `storage/postgres`)**: aplicar el perfil al
  crear el contenedor y persistir el perfil y el intento del entorno.
- **Entrega HTTP y frontend**: exponer el momento en el DTO del entorno y mostrar los
  estados "preparando examen", "examen listo" y "examen cerrado" al alumno.

## Fuera de Alcance

- Dimensionamiento de RAM y CPU (techo de 2 GB vs 4 GB ofrecidos, límite de CPU en
  práctica) y cuota de disco por alumno. Este cambio crea el lugar donde esos números se
  declaran; no los fija.
- El juez de algoritmos: ya ejecuta el código entregado en un contenedor efímero y limpio.
- La copia de entrega: ya existe y ya es el artefacto que se corrige.
- La política de red del examen: se resuelve en `openspec/changes/red-del-examen` (salida
  cerrada por defecto, con excepción declarada por el docente). Este cambio no fija red.
- El congelado por cierre de aula, que sigue como está.
- Migración de entornos existentes: los volúmenes actuales no se borran ni se renombran.

## Capabilities

### New Capabilities

- `entornos-por-momento`: perfil de entorno de práctica y de examen derivado del propósito
  del ejercicio, con estado de partida declarado para el examen y reglas de ciclo de vida
  propias de cada momento.

## Riesgos y Plan de Rollback

| Riesgo | Impacto | Control |
|---|---|---|
| La hibernación automática deja de aplicar en examen y sube el consumo del host durante la ventana | Alto en la semana de exámenes | El perfil de examen se aplica solo a entornos con `purpose = exam` y ventana abierta; el worker de QoS sigue pudiendo hibernar por presión de host y lo audita |
| Crear un espacio nuevo por intento duplica volúmenes y disco | Medio | El intento anterior queda congelado y es candidato a retención; la limpieza se apoya en el barrido existente. No se borra nada en este cambio |
| Un docente cambia `purpose` de un ejercicio con entornos ya creados | Medio | Se bloquea el cambio si existen entornos `running` y se exige confirmación explícita con auditoría |
| El alumno ve un espacio distinto al abrir el examen y cree que perdió su trabajo | Medio | Estado explícito en la UI y mensaje que indica que su práctica sigue intacta en su propio espacio |
| Alumnos que dependen de un espacio ya preparado para el examen | Alto académico | El examen arranca desde la versión base del ejercicio, y esa versión se declara antes de la ventana |

**Rollback:** la migración es aditiva (columnas nuevas con default) y el
comportamiento anterior se recupera con el interruptor `ENV_PROFILE_SPLIT=off`, que hace
que todo entorno use el perfil único actual y que el espacio se reutilice como hoy. No hay
delta destructivo: no se elimina ni se reescribe ningún volumen.

## Impacto Documental

- Requiere ADR nuevo: "Perfiles de entorno por momento (práctica vs examen)" y entrada en
  `docs/ARQUITECTURA/ADR/INDEX.md`.
- Requiere actualizar la trazabilidad de slices y el inventario de arquitectura donde hoy
  se describe un único modelo de entorno.
- `docs/OPERATIONS/capacidad.md` debe reflejar que el perfil de examen no hiberna.

## Cambios Relacionados

- `openspec/changes/red-del-examen`: cierra la política de red del examen, que este cambio
  deja fuera de alcance.
- `openspec/changes/dimensionamiento-practica-ram-cpu`: fija los números de memoria y CPU
  del perfil de práctica.
- `openspec/changes/cuota-de-disco-por-entorno`: cuota de disco por entorno, fuera de los
  perfiles.
