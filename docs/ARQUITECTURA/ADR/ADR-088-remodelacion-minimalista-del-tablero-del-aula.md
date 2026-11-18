# ADR 088: Remodelación Minimalista del Tablero del Aula

> **Estado:** Aceptado
> **Fecha:** 2026-11-18
> **Ámbito:** Docencia, Interfaz del docente
> **Relacionado:** ADR-087 (identidad del aula como tablero de la clase), ADR-058 (avance voluntario en aula), ADR-083 (perfil de entorno por propósito)

---

## Contexto

ADR-087 fijó qué es esta pantalla: responde «¿la clase va bien o hay que intervenir?» mientras la clase ocurre. La respuesta era correcta y la forma no:

- **Seis pestañas, cinco en cero.** `Todos 1 · Trabajados 0 · Sin empezar 0 · Salieron 0 · Entregaron 0 · Trabajando 1` ocupaba una franja entera, obligaba a leer seis números y repetía el estado que la tabla ya mostraba en su columna.
- **Cuatro botones del mismo peso** —abrir el IDE, emitir un avance, pausar y cerrar— más tres «Actualizar» distintos para refrescar el tablero, los avances y el historial.
- **Cuatro columnas para una fila.** «Entrega —» e «Intentos 0» dejaban dos columnas vacías al lado de un nombre: la lista se leía como un componente roto.
- **Una caja por bloque**, con borde, cabecera tintada con el color de la institución y separadores en cada fila. Con la tabla, los avances y el historial en el mismo peso, nada indicaba qué mirar primero.
- **Un velo de marca** sobre el lienzo que teñía toda la parte superior.

El docente no habita esta pantalla: está en el código, en el entorno de la práctica o frente al curso. Entra a mirar quién se trabó y sale. La densidad que se justifica en un editor —donde el usuario sí pasa la clase— acá solo agrega ruido: **una pantalla que se mira de paso tiene que resolverse en una mirada**.

## Decisión

### 1. Dos acciones y un menú

La cabecera conserva las acciones que se usan con la clase viva —`Abrir mi IDE` (secundaria) y `Emitir avance` (primaria)— y mueve el cierre a un menú `⋯`. En un examen, la pausa ocupa el lugar de la emisión porque es un control en vivo; el cierre también va al menú. El cierre es la acción del final, no la del momento, y no compite por la mirada con las otras dos.

### 2. La conclusión es una sola frase

La cabecera no alterna entre una frase y una fila de chips: cuando hay algo que mirar, la misma línea empieza por la presencia (`1 de 1 dentro`) y sigue con el chip de la excepción (`1 sin entregar`). El docente lee una sola cosa en lugar de un chip suelto que hay que interpretar.

### 2.1. Filtros solo donde hay gente

Las pestañas de estado se generan con los estados que tienen al menos un alumno. Con un aula de una persona quedan `Todos` y `Trabajando`; con un aula de doce aparecen las que correspondan. El estado activo se conserva aunque llegue a cero, para que el filtro elegido no desaparezca mientras se mira. La franja deja de enumerar ausencias.

### 3. Tres columnas: alumno, estado y actividad

`Alumno` reúne nombre y correo, y los intentos pasan a texto secundario que solo aparece cuando son mayores que cero. `Actividad` reúne la hora de entrega y el veredicto. La columna `Intentos` desaparece: era una cifra sin acción y el dato sigue disponible.

### 4. La tabla no es una caja

Sin borde exterior, sin fondo propio, sin cabecera tintada y sin separador por fila. Una sola regla bajo la cabecera, 56 px de alto por fila y un tinte muy leve en el hover. El único color de la fila es el punto de estado. La cabecera de la lista se retira y el contenido queda como protagonista.

### 5. Un solo refresco

El tablero se relee cada cinco segundos por sí solo; los avances y el historial solo se pedían a mano. Los tres botones «Actualizar» se reemplazan por uno de ícono en la barra de la lista, que actualiza las tres fuentes. El historial plegado deja de tener su propio botón y su `output` de refresco.

### 6. Los avances son líneas, no tarjetas

Cada emisión ocupa una línea: hora, archivos, cuántos siguen pendientes y `Ver detalle`. Los conteos de «aplicaron» y «descartaron» se consultan en el detalle, donde están los nombres. Una emisión que ya aplicaron todos se lee «todos lo aplicaron», sin tres cifras de colores.

### 7. Lienzo liso y ancho de lectura

Se retira el velo de marca del lienzo y el fondo tintado de la cabecera del tablero: la pantalla usa el lienzo del sistema sin degradado. El acento de la institución queda en el hover de la fila, el foco y el enlace de detalle, donde acompaña una acción en lugar de decorar una superficie. El contenido se limita a 1120 px y se centra: sin el límite, las columnas se separaban hasta perder relación entre sí.

### 8. Aire entre bloques, poco dentro

24 px entre bloques y 8–12 px dentro de cada uno. Las secciones se separan con una regla fina en lugar de con cajas. Con el aula cerrada, los paneles del examen, de la ventana y de apertura dejan de ser tarjetas y se suceden con la misma regla; la jerarquía la da el título, no el borde.

### 9. Un solo verbo para el efecto: solo lectura

El mecanismo se llama congelar en el contrato (`CLASS_FROZEN`, ADR-005) y en la operación de contenedores, pero la pantalla no lo usa como etiqueta. El panel del examen cierra con «Dejar en solo lectura», el conteo dice «en solo lectura», el reporte «no quedaron en solo lectura» y el historial «Dejó la clase en solo lectura» y «Devolvió la escritura a la clase». Es la misma expresión que ya usaban el aviso del examen, la confirmación del cierre y el laboratorio del alumno («Tu espacio está en solo lectura»): un solo verbo para lo que le pasa al entorno, y el nombre técnico queda en el evento y en el código.

### 10. Lo que no cambia

Los avisos de pausa, examen y emisión conservan su tinte: son estados excepcionales y ahí la densidad está justificada. Tampoco cambian el sondeo de cinco segundos, la lista de excepciones de la conclusión, el historial plegado ni la ventana de los avances.

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Mantener las seis pestañas y ocultar solo las vacías con CSS | Sigue ocupando la franja con números que no filtran; el orden y el conteo cambian sin que el docente lo pida. |
| Un desplegable para el filtro de estado | Un control cerrado esconde el dato; con dos o tres estados reales, dos pestañas se leen de un vistazo. |
| Fusionar los avances dentro del historial | El historial dice que se emitió; los avances dicen quién lo aplicó. Son dos preguntas y la segunda tiene acción. |
| Dejar «Intentos» como columna ordenable | Ordena por una cifra que no se usa para intervenir; ordenar por estado o por entrega responde mejor a lo que se busca. |
| Conservar los tres refrescos | Tres botones para una acción mental («refrescar esto») y tres llamadas que el docente no distingue. |
| Pasar la pantalla a una ruta aparte o a un panel lateral | Ya está en su ruta y es el tablero de la clase; el problema era la forma, no el lugar. |

## Consecuencias

- `aula.component.html`, `aula.component.scss` y `aula.component.ts` cambian la cabecera, la conclusión, los filtros, la lista, el refresco y los avances. No cambian endpoints, modelos ni permisos.
- `class-history` pierde el botón «Actualizar» y su `output` de refresco; el registro sigue leyéndose igual.
- El estado de los filtros se deriva del tablero: un estado sin alumnos no se ofrece, salvo que sea el activo.
- El aula cerrada queda con los mismos bloques, sin tarjetas.
- Queda fuera de alcance: el modal de emisión, el detalle por destinatario y la ventana de los avances.

## Verificación

- `npx tsc -p tsconfig.app.json --noEmit`, `npm run lint:styles` (gates de color, tipografía, movimiento, datos máquina y terminología) y `npm run build` en `frontend/`.
- Recorrido de la vista con un aula de clase y un aula de examen: emitir un avance, abrir el detalle por destinatario, desplegar el historial, pausar y reanudar el examen y cerrarlo.
