## 1. Contexto y Origen de la Necesidad
  
SOLV debe proporcionar mecanismos para evaluar el trabajo de los estudiantes. Sin embargo, la naturaleza de las tareas de 
programación varía drásticamente: desde algoritmos matemáticos estrictos (Estructuras de Datos) hasta arquitecturas de 
software abiertas o interfaces gráficas (Programación Orientada a Objetos, Desarrollo Web).
  
Intentar automatizar la evaluación de proyectos complejos genera alta fricción para el docente al requerir la redacción de 
pruebas unitarias exhaustivas, o resulta en evaluaciones inexactas. Además, el método tradicional de revisión manual 
(descarga de archivos `.zip` o clonación de repositorios) consume un tiempo excesivo en configuración de entornos locales.
  
---
  
## 2. Decisión Arquitectónica
  
Se implementará una **Estrategia Dual de Evaluación**:
  
### Fase A (Autocalificación Algorítmica)
  
Para laboratorios lógicos, se utilizará un **Juez Virtual Híbrido** que combine:
  
- **Análisis Estático (AST)** para validar la estructura del código.  - un motor de **Ejecución Aislada (I/O)** para 
verificar resultados de salida.
  
### Fase B (Revisión Asistida y Congelamiento)
  
Para proyectos complejos, el sistema no autocalificará. En su lugar, implementará una **máquina de estados para los 
Volúmenes de Docker**.
  
Al finalizar el examen (por acción del docente o entrega voluntaria), el volumen del estudiante pasará a estado de **solo 
lectura (`:ro`)**.
  
El docente podrá levantar contenedores efímeros conectados a estos volúmenes congelados para revisar el código en ejecución 
directamente desde su navegador web en cuestión de segundos.
  
---
  
## 3. Justificación y Eficiencia (Primeros Principios)
  
### Realismo del MVP
  
Limitar el Juez Virtual automático a algoritmos reduce drásticamente la complejidad de desarrollo, garantizando un producto 
funcional para la primera versión (**MVP**).
  
### Eliminación de Fricción (Cero Setup)
  
El **"Review Mode"** elimina la necesidad de que el docente descargue archivos o instale dependencias en su máquina local. 
Todo el entorno del estudiante se reproduce instantáneamente en el servidor privado.
  
### Manejo del Factor Humano
  
El control del congelamiento de entornos se delega a **eventos manuales desde el panel del docente**, permitiendo 
flexibilidad ante retrasos en la clase o excepciones individuales, evitando la rigidez de los temporizadores estrictos.
  
---
  
## 4. Consecuencias y Trade-offs
  
- El backend (**FastAPI**) deberá gestionar rigurosamente los permisos de montaje de los volúmenes de Docker (`:rw` vs 
`:ro`) basándose en el estado de la entrega en la base de datos.
  
- Se requiere diseñar una **interfaz gráfica (UI)** clara para que el docente pueda gestionar el ciclo de vida del examen: - 
iniciar examen - congelar a toda la clase - otorgar excepciones individuales

---

## 5. Bloqueo de edición al cerrar un examen (mecanismo de la Fase B)

La Fase B quedó decidida en §2 como "el volumen pasa a `:ro` al terminar el examen", pero faltaba el mecanismo: quién lo activa, cómo se fuerza y qué pasa con las excepciones. Esta sección lo cierra.

### 5.1 Decisión

**El examen se cierra con el volumen del alumno montado en solo lectura.** Para eso se recrea el contenedor del alumno con el mismo montaje de su volumen, cambiando `:rw` por `:ro`, y el estado queda guardado en la base: `workspaces.frozen_at` con la hora del cierre y el estado `frozen`.

La pantalla del alumno deja de ofrecer el editor y muestra el motivo. El bloqueo no vive en la interfaz: vive en el montaje, que es lo único que el alumno no puede deshacer desde el navegador.

### 5.2 Cuándo se congela y cuándo no

| Momento | Qué pasa |
| --- | --- |
| El examen se abre | El entorno queda escribible. El alumno tiene que programar durante el examen; bloquear al empezar vacía la evaluación. |
| El docente cierra el examen | Se congelan los volúmenes de los alumnos del aula, uno por uno. |
| Se vence `ends_at` | Mismo cierre, sin que el docente tenga que pulsar nada. |
| El alumno entrega antes | Se congela solo su volumen. La clase sigue abierta para los demás. |
| El docente reabre a un alumno | Vuelve a `:rw` con un motivo escrito y queda en auditoría. Es la "excepción individual" de §4. |
| Es una clase, no un examen | No se congela nunca. |

Solo aplica a prácticas con `purpose = exam`. Una sesión de clase congelada (`class_sessions.frozen_at`, ADR-058) es otra cosa: bloquea emitir avances nuevos y no toca el volumen de nadie.

### 5.3 Alternativas comparadas

| Criterio | Montaje `:ro` con recreación | Bloqueo del editor | Pausar el contenedor |
| --- | --- | --- | --- |
| Qué impide | Toda escritura al volumen, venga del editor, del terminal o de un proceso | Solo lo que se ve en la pantalla | Toda escritura, mientras siga pausado |
| Dónde se aplica | Kernel, vía montaje | Navegador | cgroup del contenedor |
| Cómo se salta | No se salta desde el navegador | Abriendo la URL del editor o el terminal integrado | Entrando de nuevo al laboratorio: la reanudación despausa |
| Costo | Recrear el contenedor: unos segundos por alumno | Ninguno | Ninguno |
| Qué se pierde | Los cambios sin guardar del editor | Nada | Nada (el proceso sigue congelado con sus búferes) |
| Al reabrir | Otra recreación con `:rw` | Un clic | Un despause |
| Sobrevive a un reinicio del backend | Sí, porque el estado está en la base | No aplica | No: el estado de pausa no es parte de la entrega |
| Qué ve el alumno | Aviso de examen cerrado; si intenta guardar, error de escritura | Un editor que parece normal | Un editor que parece colgado |

### 5.4 Por qué no las otras alternativas

**Bloquear el editor.** Es un aviso, no un control. El editor se sirve por su propia URL y el alumno la conoce; también tiene terminal integrado dentro del mismo contenedor. Un cerrojo que se abre copiando una dirección da una seguridad que no existe, y en una defensa o una revisión eso es peor que no tener cerrojo.

**Pausar el contenedor.** El camino de reanudación del laboratorio (`reactivateWorkspace`) despausa cualquier contenedor pausado antes de mirar otra cosa, así que el primer alumno que vuelva a entrar descongela su propio volumen sin que nadie se lo pida. Además se mezcla con la hibernación, que existe para ahorrar memoria y se reanuda sola a propósito. Un congelamiento de examen no puede compartir ese camino.

**Cortar el acceso por el proxy.** Es más fuerte que esconder el editor, pero tampoco deja el volumen inmutable: los procesos de adentro siguen escribiendo y, al volver a levantar el contenedor, el alumno recupera la escritura. Además deja al docente sin una vía de lectura, que es justo lo que el montaje `:ro` resuelve.

**Cambiar permisos desde dentro.** El contenedor del alumno corre como usuario `1000`, sin privilegios para cambiar atributos del sistema de archivos, y un contenedor recreado los pierde. No es un estado que se pueda sostener.

### 5.5 Consecuencias

- Congelar y descongelar corta el editor unos segundos por alumno. La pantalla avisa antes de cerrar y el que tenía algo sin guardar lo pierde; es el costo de que el cierre sea real.
- La reanudación del laboratorio tiene que respetar `frozen_at`: un laboratorio congelado no se vuelve a abrir en escritura por pedido del alumno, ni por el reconciliador.
- Aparece un estado nuevo, `frozen`. Su semántica es neutra, como `hibernated`: es un estado querido, no un error.
- El docente lee el código con un contenedor efímero montado en `:ro`, el mismo mecanismo que ya usa el juez (ADR-020). No hace falta que el contenedor del alumno siga vivo.
- Con 35 alumnos, el cierre son 35 recreaciones. Se hacen de a una y el cierre del aula reporta cuántas salieron y cuáles fallaron, en lugar de darse por hecho.

### 5.6 Qué hay que tocar para implementarlo

1. Migración idempotente: `workspaces.frozen_at TIMESTAMPTZ` y `frozen` entre los estados admitidos.
2. Docker: `ReadOnly` en `WorkspaceContainerConfig`, que hoy solo existe para el contenedor del juez, y un camino de recreación (detener, quitar, crear, arrancar) que reutilice `StartWorkspaceContainer` para no duplicar la configuración del contenedor.
3. Servicio: congelar y descongelar por alumno, con las reglas de §5.2 y auditoría; el cierre del aula y el vencimiento de `ends_at` los disparan.
4. Reanudación: si `frozen_at` no es nulo, responder que el examen terminó en lugar de recrear en escritura.
5. Pantalla del alumno: aviso de examen cerrado en lugar del editor, con la hora del cierre.
6. Pantalla del docente: acción de cerrar el examen, conteo de congelados y la excepción individual con motivo.

### 5.7 Verificación

- Con el examen cerrado, escribir en el volumen falla desde el editor y desde un contenedor efímero en escritura.
- El alumno que recarga y pide abrir su laboratorio no recupera la escritura.
- Descongelar con motivo devuelve la escritura y deja la entrada en auditoría con actor, alumno y motivo.
- Cerrar un aula de clase no congela ningún volumen.
- El cierre de un aula de examen reporta el conteo real de congelados y de fallos.

### 5.8 Documentos que hay que actualizar cuando se implemente

- `docs/UI/ESTUDIANTE/LABORATORIO_ACTIVO.md`: el aviso de examen cerrado.
- `docs/UI/MAPA_DE_VISTAS.md` y `docs/API_CONTRACTS.md`: el estado `frozen`, el cierre del examen y la excepción.
- `odd/tasks/docente-avance-voluntario.md`: registra la decisión y enlaza la ficha de esta implementación.

### 5.9 Estado de implementación

Implementado el 31/10/2026.

- **Base.** `workspaces.frozen_at` (migración idempotente `00041_workspace_exam_freeze.sql`) y el estado `frozen`, que no necesita restricción nueva porque `status` es texto. El estado es neutro: no cuenta como fallo ni libera el cupo como `orphaned`.
- **Docker.** `WorkspaceContainerConfig.ReadOnly` monta el volumen en `:ro`; `SetWorkspaceMount` reconstruye el contenedor con la configuración de siempre (detener, quitar, crear y arrancar). Reabrir hace lo mismo con `:rw`. El arranque no bloquea: quien va a escribir espera a que el proceso esté en ejecución.
- **Servicio.** `ExamFreezeService` congela a los participantes del aula, salta los que ya estaban congelados y cuenta los fallos sin abortar el resto. Se dispara al cerrar el aula, al vencer `ends_at` y desde un worker al arrancar y cada 15 minutos. Cada movimiento queda en `audit_logs` con `resource_type = class_session`.
- **API.** `POST /api/v1/teacher/sessions/{id}/exam-freeze` devuelve `{ frozen, skipped, failed }`; `POST /api/v1/teacher/sessions/{id}/students/{studentID}/unfreeze` reabre un alumno con motivo obligatorio. El cierre del aula incluye `freeze_report` en la respuesta.
- **Reanudación.** Con `frozen_at` no nulo, abrir o reiniciar el laboratorio responde `409` y no se recrea en escritura, ni por pedido del alumno ni por el reconciliador.
- **Pantalla del alumno.** Con el estado `frozen` se muestra el aviso de examen cerrado con la hora del cierre en lugar del editor, y no se ofrece iniciar el entorno.
- **Pantalla del docente.** La sesión de aula trae `purpose` (`class` o `exam`), así que `/teacher/aula` marca el examen y cambia `[Cerrar aula]` por `[Cerrar examen]`, con el aviso de que el cierre deja los entornos en solo lectura. Ya cerrada, muestra el conteo de volúmenes en solo lectura, el resultado del cierre o del reintento, y la lista de alumnos con entorno para reabrir uno con motivo.

- **Entrega anticipada.** Entregar un examen desde el IDE cierra solo el volumen de ese alumno: el montaje pasa a `:ro`, la fila queda `frozen` y la clase sigue abierta para los demás. Una segunda entrega responde `409`, porque escribir por esa vía sería seguir editando. La pantalla del alumno pide confirmación antes de entregar y avisa que el entorno queda en solo lectura; la auditoría registra el laboratorio como recurso.

Todos los puntos de §5.6 quedaron implementados.

La prueba de punta a punta corre contra Docker y el volumen reales: arranca el laboratorio con el montaje en `:rw`, escribe desde el contenedor como root, entrega el examen por el mismo camino de la aplicación —que dispara el congelamiento— y comprueba que la siguiente escritura falla con `Read-only file system` mientras la lectura sigue disponible. Repite la comprobación con un contenedor efímero montado en `:ro`, verifica que una segunda entrega responde `409` sin tocar el volumen y que reabrir devuelve la escritura. Está apagada por defecto porque necesita el daemon, la imagen del editor y la red de Traefik: `EXAM_FREEZE_E2E=1 go test ./tests/integration -run TestExamFreezeDockerE2E`.

La ficha de trabajo con las tareas y el registro de decisiones es [examen-congelamiento.md](../../../odd/tasks/examen-congelamiento.md).
