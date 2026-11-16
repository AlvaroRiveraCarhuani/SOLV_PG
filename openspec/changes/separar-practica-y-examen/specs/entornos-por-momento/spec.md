# entornos-por-momento Specification

## Purpose

Definir dos perfiles de entorno —práctica y examen— derivados del propósito del ejercicio,
de modo que el alumno practique en un espacio flexible y rinda en un espacio creado para el
examen, con estado de partida declarado y artefacto de corrección inmutable. La política de
red del examen no forma parte de este perfil: se resuelve en `red-del-examen`.

## Requirements

### Requirement: Perfil derivado del propósito del ejercicio

El sistema MUST derivar el perfil del entorno desde `exercise.purpose` y MUST NOT aceptar
un perfil enviado por el cliente. El perfil MUST NOT declarar política de red: esa decisión
pertenece a `red-del-examen`. Un propósito ausente MUST tratarse como `class`.
Un ejercicio con `purpose = "exam"` MUST producir un entorno con perfil `exam`; uno con
`purpose = "class"` MUST producir un entorno con perfil `practice`.

#### Scenario: Ejercicio de práctica abre con perfil practice
- GIVEN un ejercicio con `purpose = "class"` y un alumno inscrito
- WHEN el alumno abre el entorno del ejercicio
- THEN el entorno se crea con perfil `practice`
- AND la respuesta incluye `profile = "practice"`.

#### Scenario: Ejercicio de examen abre con perfil exam
- GIVEN un ejercicio con `purpose = "exam"` y un alumno inscrito
- WHEN el alumno abre el entorno del ejercicio
- THEN el entorno se crea con perfil `exam`
- AND la respuesta incluye `profile = "exam"`.

#### Scenario: El perfil no declara red
- GIVEN un entorno de examen con perfil `exam`
- WHEN se inspecciona la configuración aplicada al contenedor
- THEN la política de red no proviene del perfil
- AND se lee de `exam_network` del ejercicio, resuelto en `red-del-examen`.

#### Scenario: Perfil manipulado en la petición
- GIVEN una petición de apertura que incluye `profile = "practice"` para un ejercicio con `purpose = "exam"`
- WHEN el servicio procesa la petición
- THEN el campo del cliente se ignora
- AND el entorno resultante tiene perfil `exam`.

---

### Requirement: Estado de partida declarado del examen

Para un ejercicio con `purpose = "exam"`, el sistema MUST crear el espacio de examen al
abrir la ventana, sobre un volumen nuevo del intento actual, sembrado desde la versión
base del ejercicio. El sistema MUST NOT reutilizar un volumen previo del mismo ejercicio
para calificar. Un intento cerrado MUST NOT admitir escritura.

#### Scenario: Primer intento crea volumen nuevo desde la versión base
- GIVEN un ejercicio de examen con versión base publicada y un alumno sin intentos previos
- WHEN el alumno abre el examen dentro de la ventana
- THEN se crea el intento `1`
- AND el volumen se siembra con los archivos de la versión base
- AND el estado del entorno es `preparando` hasta que el contenedor responde.

#### Scenario: Apertura previa no contamina el examen
- GIVEN un alumno que abrió el ejercicio de examen antes de la ventana y dejó archivos en su espacio
- WHEN la ventana del examen se abre y el alumno entra
- THEN el sistema crea un intento nuevo con un volumen distinto
- AND los archivos de la apertura previa no están presentes en el espacio de examen.

#### Scenario: Intento cerrado no admite escritura
- GIVEN un intento de examen ya entregado y congelado
- WHEN el alumno intenta abrirlo o entregar de nuevo
- THEN el sistema responde que el entorno está congelado

---

### Requirement: Recuperatorio sobre intento nuevo

El sistema MUST abrir un intento nuevo cuando el alumno vuelve a un examen ya cerrado y
existe una convocatoria del mismo ejercicio abierta después de ese cierre. Sin esa
convocatoria el intento MUST seguir cerrado: recargar la página MUST NOT reabrir nada. El
intento nuevo MUST arrancar sobre un volumen distinto del anterior, sembrado desde la
versión base vigente; el volumen del intento cerrado MUST conservarse sin cambios.

#### Scenario: Convocatoria nueva habilita el recuperatorio
- GIVEN un alumno con el intento 1 cerrado y congelado
- AND una sesión de examen del mismo ejercicio abierta después de ese cierre
- WHEN el alumno abre el examen
- THEN el entorno pasa al intento 2
- AND arranca sobre un volumen distinto, sembrado desde la versión base
- AND los archivos del intento 1 no están presentes.

#### Scenario: Sin convocatoria nueva el examen sigue cerrado
- GIVEN un alumno con el intento cerrado y congelado
- AND ninguna sesión de examen abierta después del cierre
- WHEN el alumno abre el examen
- THEN el sistema responde que el entorno está congelado
- AND el número de intento no cambia.

#### Scenario: Una convocatoria futura no habilita rendir hoy
- GIVEN un alumno con el intento cerrado
- AND una sesión de examen del mismo ejercicio que abre más adelante
- WHEN el alumno abre el examen antes de esa hora
- THEN el sistema responde que el entorno está congelado.

#### Scenario: Una práctica no tiene intentos
- GIVEN un entorno de práctica congelado
- AND una convocatoria nueva del mismo ejercicio
- WHEN se abre el entorno
- THEN el sistema responde que el entorno está congelado.

#### Scenario: Reabrir el mismo intento es otra operación
- GIVEN un examen cerrado con un alumno afectado por una caída del entorno
- WHEN el docente usa la excepción individual con motivo
- THEN el alumno recupera la escritura sobre el mismo intento
- AND el número de intento no cambia.
- AND no se crea un intento nuevo dentro de la misma ventana.

---

### Requirement: Ciclo de vida propio de cada perfil

El perfil `practice` MUST permitir la hibernación por inactividad y MUST NOT capturar
tecleo. El perfil `exam` MUST NOT hibernar por inactividad mientras la ventana esté
abierta y MUST capturar tecleo. La hibernación por presión de memoria del host MUST seguir
disponible en ambos perfiles y MUST quedar auditada.

#### Scenario: Práctica hiberna por inactividad
- GIVEN un entorno con perfil `practice` sin actividad durante el umbral configurado
- WHEN el worker de QoS evalúa el entorno
- THEN el entorno pasa a `hibernated`
- AND al reabrirse conserva sus archivos.

#### Scenario: Examen no hiberna por inactividad
- GIVEN un entorno con perfil `exam` y ventana abierta, sin actividad durante el umbral
- WHEN el worker de QoS evalúa el entorno
- THEN el entorno permanece `running`
- AND no se registra una hibernación por inactividad.

#### Scenario: Presión de host durante un examen
- GIVEN un entorno con perfil `exam` y memoria del host por debajo del mínimo libre
- WHEN el worker de QoS necesita liberar memoria
- THEN el entorno puede hibernarse
- AND la decisión queda en `audit_logs` con el motivo de presión.

---

### Requirement: Captura de tecleo acotada al examen

El servidor MUST aceptar eventos de tecleo solo para entregas de entornos con perfil
`exam`. Un evento enviado para un entorno con perfil `practice` MUST rechazarse sin
persistirse, aunque el cliente lo envíe.

#### Scenario: Práctica no registra tecleo
- GIVEN un entorno con perfil `practice` y una entrega asociada
- WHEN el cliente envía un lote de eventos de tecleo
- THEN el servidor responde rechazo de captura
- AND no se persiste ningún evento.

#### Scenario: Examen registra tecleo
- GIVEN un entorno con perfil `exam` y una entrega del mismo alumno
- WHEN el cliente envía un lote de eventos de tecleo
- THEN el servidor persiste los eventos
- AND la cantidad persistida coincide con la aceptada.

---

### Requirement: Artefacto de corrección inmutable

La corrección MUST usar la copia tomada al entregar, no el volumen vivo del alumno. Si la
copia excede los límites vigentes (4 MB totales, 800 archivos, 1 MB por archivo), el
sistema MUST rechazar la entrega con un mensaje que indique la ruta y el tamaño del
archivo que excede, y MUST NOT registrar una entrega parcial.

#### Scenario: Entrega normal
- GIVEN un entorno con perfil `exam` y un proyecto dentro de los límites
- WHEN el alumno entrega
- THEN se guarda la copia del volumen
- AND el docente revisa esa copia.

#### Scenario: Entrega que excede el límite
- GIVEN un proyecto cuyo archivo mayor supera 1 MB
- WHEN el alumno entrega
- THEN el sistema rechaza la entrega indicando la ruta y el tamaño
- AND no queda una entrega registrada.

---

### Requirement: Estado visible para el alumno

El sistema MUST exponer el estado del entorno de examen como `preparing`, `ready` o
`closed`. La interfaz MUST informar que el espacio de práctica del alumno no se modifica.

#### Scenario: Preparación del examen
- GIVEN un alumno que entra a un examen dentro de la ventana
- WHEN el contenedor todavía no responde
- THEN la interfaz muestra "preparando tu examen"
- AND al quedar listo muestra "examen listo" sin recargar la página.

#### Scenario: Cierre del examen
- GIVEN un examen entregado y congelado
- WHEN el alumno vuelve a la pantalla del ejercicio
- THEN la interfaz muestra "examen cerrado"
- AND no ofrece reabrir el editor.

---

### Requirement: Cambio de propósito con entornos existentes

El sistema MUST bloquear el cambio de `purpose` de un ejercicio que tenga entornos
`running` y MUST exigir confirmación auditada cuando existan entornos hibernados.

#### Scenario: Cambio bloqueado
- GIVEN un ejercicio de práctica con un entorno `running`
- WHEN el docente intenta cambiar su propósito a `exam`
- THEN el sistema rechaza el cambio
- AND explica que hay entornos activos.

---

### Requirement: Interruptor de reversión

El sistema MUST ofrecer `ENV_PROFILE_SPLIT=off` para volver al comportamiento de un solo
perfil y reutilización del espacio, sin migración inversa de datos.

#### Scenario: Reversión
- GIVEN `ENV_PROFILE_SPLIT=off`
- WHEN un alumno abre un ejercicio de examen
- THEN el entorno usa el perfil único actual
- AND el espacio se reutiliza como antes del cambio.
