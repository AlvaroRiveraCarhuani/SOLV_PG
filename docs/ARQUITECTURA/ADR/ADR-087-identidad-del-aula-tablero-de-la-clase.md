# ADR 087: Identidad del Aula como Tablero de la Clase, sin Consola ni Telemetría

> **Estado:** Aceptado
> **Fecha:** 2026-11-18
> **Ámbito:** Docencia, Interfaz del docente
> **Relacionado:** ADR-005 (cierre de examen), ADR-011 (hibernación por inactividad), ADR-053 (revisión docente), ADR-058 (avance voluntario en aula), ADR-059 (entorno del docente por práctica)
> **Lenguaje visual:** revisado por ADR-088 (remodelación minimalista del tablero)

---

## Contexto

La pantalla `/teacher/aula` acumulaba cinco bloques alimentados por cuatro fuentes distintas y dos ritmos de refresco:

1. Tablero de la sesión: inscritos, estado de cada alumno, entrega e intentos, cada 5 segundos.
2. Avances de la clase: emisiones y estado por destinatario.
3. Historial: eventos del aula.
4. Telemetría de entornos: contenedores vivos de la materia.
5. Paneles del cierre: examen congelado y ventana de disponibilidad.

El comentario del componente declara una sola pregunta: quién está, quién falta y quién se trabó. Tres de los cinco bloques la responden; los otros dos responden otra cosa, con otra población y otro ritmo.

Tres defectos concretos motivaron la revisión:

- **La consola no cumplía lo que ofrecía.** El botón "Consola del Estudiante" abría una ventana que enviaba una única foto de los logs del contenedor y no transmitía después. El modo tutor ejecutaba el comando y descartaba su salida: la respuesta era "Comando enviado al environment: `<cmd>`". Tampoco mostraba los archivos del alumno, que es lo que el docente necesita ver ante un error. El botón aparecía dos veces: en la fila de la tabla y en la tarjeta de telemetría.
- **La telemetría llegaba vacía.** El endpoint de sesiones vivas devuelve estado, reinicios por memoria y última actividad, pero no memoria usada, CPU, historial ni actividad de tecleo. La tarjeta mostraba `0 MB`, `0%`, una curva que nunca se llenaba y el texto por defecto "En espera", que el docente lee como "el alumno está inactivo" cuando en realidad significa "no hay dato". La plantilla declaraba además estados de tecleo y palabras por minuto que nada produce.
- **Dos listas de población distinta.** La tabla lista a los inscritos en la materia; la telemetría, a quien tenga contenedor encendido. Un inscrito sin entorno no aparecía en la segunda, y un alumno de otro curso con entorno abierto sí.

## Decisión

### 1. Identidad: el tablero de la clase que está ocurriendo

La pantalla responde, en segundos y de pie frente al curso:

> **¿La clase va bien o hay que intervenir?**

Y concentra las acciones que solo existen en ese momento: abrir el IDE de la práctica, emitir un avance, pausar y cerrar. No es una consola de monitoreo, ni un tablero de recursos, ni una hoja de asistencia, ni una vista de revisión.

### 2. La cabecera concluye

Los contadores sueltos de presencia salen del encabezado. En su lugar, una línea con la conclusión: "Todo en orden: N de M dentro, K entregaron" o los chips de la excepción (trabados, sin entorno, sin entregar, sin empezar, salieron). La cabecera deja de enumerar y pasa a responder.

### 3. Una sola tabla, con la excepción en la fila

La columna "Entorno" desaparece junto con su botón. La única excepción de entorno que la pantalla afirma es **"Sin entorno"**, derivada de `environment_id` del tablero y mostrada solo cuando el alumno está presente en el aula (`trabajando`) y no tiene entorno vivo: está ahí y no puede trabajar. Quien todavía no empezó no es una excepción; quien entregó ya no necesita el entorno. Es un bloqueo real, no una métrica inferida. El estado del entorno solo llega al tablero con valores vivos (`running`, `active`, `pending`), de modo que un entorno caído se presenta como "Sin entorno" y no se distingue de uno que nunca arrancó. Se prefiere esa ambigüedad a inventar una causa.

### 4. La consola sale de la vista

Se retiran el botón de la tabla, el de la tarjeta y el modal. La supervisión del trabajo de un alumno vive en la entrega: árbol de archivos, código, diff contra la base, comentarios por línea, rúbrica y, en prácticas IDE, el preview efímero de la entrega (ADR-061). Para reproducir un fallo, el docente abre su propio entorno de la práctica (ADR-059), que parte de la misma base publicada.

Entrar al entorno vivo de un alumno no existe en el sistema, y no se agrega aquí: exige consentimiento del alumno, una sesión acotada y registro de quién entró. Sería una función propia.

### 5. La telemetría sale como panel

RAM, CPU y reinicios son operación del sistema, no información docente:

- El QoS ya apaga entornos por inactividad (ADR-011) y la admisión de memoria se resuelve al crear el contenedor (ADR-082).
- Un aviso de entorno caído pertenece al sistema, que lo detecta, no a un tablero que el docente debe mirar.
- La vista dejaba de ser honesta: pintaba ceros donde no había dato.

Con el panel salen el selector de muestreo 3s/5s/10s, el buscador propio, el refresco propio y el alternador matriz/tabla. La pantalla también deja de consultar `/api/v1/teacher/live-sessions`: el tablero ya informa la presencia de entorno de cada alumno.

Tampoco se agrega una señal de "lleva mucho sin avanzar". La actividad medible es tráfico del IDE y cambios en el volumen, que no equivalen a progreso: leer el enunciado, pensar o discutir con un compañero no produce actividad. Un aviso por inactividad marcaría como atrasado a quien está pensando. Si algún día hace falta, la fuente correcta es que el alumno pida ayuda, no una inferencia.

### 6. El historial queda plegado al final

El registro de eventos se conserva íntegro y se consulta cuando hay que reconstruir algo: un reclamo del alumno sobre su entrada o su entrega, una duda al cerrar la clase o una revisión de coordinación. No es operación del momento, así que arranca cerrado, con el encabezado y el conteo siempre visibles. En el aula cerrada se ubica después del formulario de apertura, que es la acción del cierre.

El bloque se dibuja como una tira separada por una regla, sin tarjeta: no compite en peso con las acciones del cierre y se lee como registro, no como una decisión más. Toda la barra abre y cierra, no solo el texto de la derecha; al abrirse aparecen `[Actualizar]` y la lista completa. La lista no tiene altura máxima: con el registro plegado por defecto no hay motivo para recortarlo, y el recorte anterior dejaba una fila partida al principio.

Cada movimiento ocupa una grilla de tres columnas fijas —acción, detalle y hora—, de modo que la hora no se mueve según el largo de la acción. Los movimientos se agrupan por día con un separador (`Hoy`, `Ayer` o la fecha), porque dos jornadas en una lista continua se leen como una sola. El actor aparece solo cuando cambia respecto de la fila anterior: repetirlo en cada línea ocupaba el lugar más visible sin aportar. La hora se muestra una sola vez; el detalle de un cierre de ventana ya no la repite.

Un ajuste de la ventana se lee como «Ajustó la ventana · hasta HH:MM». La acción es una sola para sumar minutos y para fijar una fecha límite porque el dato es la hora final, no cuánto se sumó. Esa decisión también cierra el defecto que dejaba once filas de «Sumó minutos» por un clic que se repitió: la pantalla manda la hora final y el servidor no vuelve a correr la ventana si ya termina ahí.

### 8. Lenguaje visual: estado con color, movimiento del catálogo y acento de marca

La pantalla no usaba ninguno de los tres recursos que el sistema ya tiene, y por eso se leía plana. Se incorporan sin sumar adorno:

- **Color de estado, no decorativo.** El punto del rótulo "Aula abierta" es verde mientras la clase corre y ámbar cuando está pausada; el chip de la ventana es verde abierta y neutro cerrada; el tablero conserva su punto por estado. El resto sigue en la escala de grises: el color solo aparece cuando comunica algo.
- **Movimiento del catálogo.** `motion-rise` en los avisos de examen, pausa y emisión, `motion-fade` en la tabla y en el despliegue del historial, y transiciones de `--transition-fast` en los estados de hover y en el chip. No se declaran keyframes propios: el gate de movimiento exige las primitivas compartidas y su bloque de movimiento reducido.
- **Acento de marca.** El rótulo superior va en `--tenant-primary`; el lienzo de la pantalla lleva un velo de `--tenant-primary-subtle` que se desvanece a los 220 px; el código de la materia se lee en una píldora con punto de marca; cada bloque abre con un ícono Lucide en color de institución; el encabezado del tablero y el hover de sus filas usan el mismo velo. El color de la institución deja de existir solo en el botón primario. Los íconos son anclas de sección, no adornos: uno por bloque, siempre el mismo.

> El velo del lienzo, la píldora del código y el encabezado tintado del tablero se retiran en ADR-088: el acento de marca queda en el hover de la fila, el foco y el enlace de detalle.

Los tres campos de apertura comparten ancho y altura —rótulo y control, sin textos auxiliares debajo—, y el botón se alinea con los controles. Las aclaraciones que antes colgaban de cada campo se juntaron en la línea que describe el bloque: con alturas distintas, los rótulos y los controles quedaban a distinta altura y el formulario se veía torcido.

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Fusionar la telemetría como columnas de la tabla | Ensancha la tabla y pone números a todos; lo único accionable es la excepción, y esos datos no llegan al cliente. |
| Mantener el panel solo cuando hay algo que mirar | Sigue dejando al docente como vigilante de recursos y obliga a sostener un endpoint que no entrega las métricas que la tarjeta pinta. |
| Arreglar la métrica y conservar el panel | Resuelve la mentira, no el papel: la operación de contenedores no es tarea del docente. |
| Conservar la consola como vista de solo lectura | Sigue sin transmitir y sin mostrar archivos; una acción que no hace nada es peor que su ausencia. |
| Historial en una pestaña o ruta aparte | Es un registro de una clase concreta; sacarlo del aula obliga a navegar y a recordar a qué sesión pertenece. |

### 7. El cierre tiene dos niveles

Con el aula cerrada, la pantalla muestra primero la acción —la ventana de los avances y el formulario de apertura, con su encabezado "Abrir otra clase"— y después el registro. Los títulos de cada bloque son encabezados de sección, no etiquetas sueltas: la vista deja de ser tres cajas del mismo peso.

Dentro de esa jerarquía, el estado de la ventana se lee en un chip (`Abierta` en verde de estado, `Cerrada` en neutro) junto al título, no en una frase que hay que interpretar. El aviso del cierre anterior deja de ser una caja y pasa a ser una anotación con regla al margen: informa un hecho, no compite con la acción de abrir otra clase. Los campos de apertura comparten una sola fila con el botón al final.

## Consecuencias

- `aula.component.html`, `aula.component.ts` y `aula.component.scss` dejan de importar el widget de telemetría y el modal de consola, y pierden el sondeo de sesiones vivas.
- Los componentes `aula-telemetry-widget` y `shadow-terminal-modal` quedan sin uso. No se borran en este cambio: la galería de movimiento documenta efectos que viven en sus hojas de estilo y la política de impacto de tests referencia sus rutas. Su retiro se resuelve junto con ambas.
- Los endpoints de terminal del backend (`/api/v1/teacher/live-sessions/{id}/terminal`, `/exec`, `GET /api/v1/teacher/live-sessions/metrics`) siguen existiendo y no se modifican.
- El historial pasa a ser un bloque plegable con el encabezado visible.
- Queda fuera de alcance: una cola de "necesito ayuda" para el alumno, los avisos de entorno caído y cualquier forma de acceso al entorno vivo de un alumno.

## Verificación

- `npx tsc -p tsconfig.app.json --noEmit`, `npm run build` y `npm run lint:styles` en `frontend/`.
- Revisión de la vista: aula abierta y aula cerrada, con y sin excepciones, y el historial plegado y desplegado.
