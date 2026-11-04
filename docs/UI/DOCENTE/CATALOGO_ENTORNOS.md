# Catálogo de Entornos del Docente

> **Estado:** Propuesto
> **Rol:** Docente
> **Decisión de arquitectura:** [ADR-054](../../ARQUITECTURA/ADR/ADR-054-catalogo-entornos-docente.md)
> **Contexto de área:** [REDISENO_PANEL_DOCENTE.md](REDISENO_PANEL_DOCENTE.md)

---

## 1. Qué resuelve

El docente llega con una idea: "quiero que programen en Java con Spring Boot y una base de datos". Hoy la pantalla le responde con `eclipse-temurin:21-jdk-alpine`, 1024 MB de RAM y una etiqueta "Oficial" que aparece en todas las tarjetas.

La pantalla tiene que responder tres preguntas, en este orden:

1. **¿Esto sirve para mi práctica?** Se reconoce por las herramientas, no por la imagen.
2. **¿Puedo usarlo hoy?** Si no hay plantilla ejecutable, se solicita.
3. **¿Qué implica?** Los requisitos técnicos viven en el detalle, no en la portada.

---

## 2. Dónde vive y qué significa cada sección

El área docente tiene hoy dos entradas que se pisan. No son lo mismo:

| Sección actual | Pregunta que responde | Qué es | Destino |
| --- | --- | --- | --- |
| **Por calificar** | ¿Qué trabajo manual me falta? | Cola de entregas con `review_status` pendiente, agrupada por práctica | Se queda como destino de trabajo |
| **Evaluaciones & Entregas** | Mezcla dos preguntas | Lista global de todas las entregas, con una pestaña "Cola de Entregas" y KPIs repetidos | Se convierte en **Entregas**: el registro histórico |
| **Plantillas de Entorno** | ¿Dónde van a programar? | Catálogo de imágenes Docker con recursos | Se convierte en **Entornos**: el catálogo por herramientas |

Regla: **una cola, un registro**.

- La cola es **Por calificar**. Solo muestra lo que espera corrección. No es un historial.
- El registro es **Entregas**. Muestra todo lo que pasó, con filtros por materia, práctica y estado. No es una tarea.
- La evaluación del juez es un **resultado dentro de una entrega**, no una sección. Por eso "Evaluaciones" no debe competir como destino.

Mientras "Notas" y "Materia" no absorban el historial, "Entregas" sigue existiendo como destino secundario. Cuando lo hagan, desaparece sin dejar un hueco: la cola ya vive aparte.

### Cambios concretos en la navegación

1. "Evaluaciones & Entregas" pasa a llamarse **Entregas**.
2. Esa página deja de tener la pestaña "Cola de Entregas": duplica "Por calificar".
3. Cada fila pendiente en "Entregas" enlaza a "Por calificar" o al panel de revisión, no a un segundo flujo.
4. Los KPIs de esa página se retiran; el conteo vive en la barra de filtros.
5. "Plantillas de Entorno" pasa a **Entornos** (etiqueta pendiente de confirmar con el usuario).

---

## 3. Estructura de la pantalla

```
Entornos                                                    [ Solicitar entorno ]
Elige dónde van a programar tus alumnos.
──────────────────────────────────────────────────────────────────────────────
Buscar por lenguaje, framework o base de datos…
[ Todos ] [ Editor persistente ] [ Juez automático ]       7 entornos
──────────────────────────────────────────────────────────────────────────────
┌───────────────────────────┐ ┌───────────────────────────┐ ┌───────────────────────────┐
│ Editor persistente        │ │ Editor persistente        │ │ Juez automático           │
│ Java 21 + Spring Boot     │ │ Python 3.12 + Data Science│ │ C / C++ Algoritmia        │
│ APIs REST y servicios     │ │ Análisis y visualización  │ │ Prácticas de algoritmia   │
│ [Java 21][Spring Boot]    │ │ [Python 3.12][NumPy]      │ │ [C][GCC 13][Make]         │
│ [Maven][PostgreSQL 16] +1 │ │ [Pandas][Jupyter]         │ │                           │
│ Ver detalle          Usar │ │ Ver detalle          Usar │ │ Ver detalle     Solicitar │
└───────────────────────────┘ └───────────────────────────┘ └───────────────────────────┘
```

Sin fila de KPIs. Sin RAM, CPU ni nombre de imagen en la tarjeta.

---

## 4. Anatomía de la tarjeta

| Zona | Contenido | Regla |
| --- | --- | --- |
| Cabecera | Modalidad en palabras: "Editor persistente" o "Juez automático" | Nunca "IDE" ni "Juez" solos |
| Título | Nombre humano: "Java 21 + Spring Boot" | El nombre de la imagen no aparece acá |
| Resumen | Una línea de para qué sirve | Máximo 90 caracteres |
| Herramientas | Chips por capa, con nombre e icono | Máximo 5; el resto se resume como "+N" |
| Acciones | "Ver detalle" y "Usar" (o "Solicitar") | "Usar" solo si `available` es verdadero |
| Insignia | "Oficial" solo si el flag es verdadero | Sin default; si no hay definición, no se muestra |

La tarjeta es un `article` con título en `h3`. No es un enlace completo: tiene acciones dentro. El foco se mueve con Tab y las acciones se anuncian con su nombre.

### Chips

- Orden: runtime, plataformas web, complementos, base de datos.
- Texto: nombre visible de la pieza, no el slug ni el binario.
- Un chip no es un botón; si se puede filtrar por esa pieza, el filtro vive en la barra, no en la tarjeta.
- Si no hay piezas declaradas: "Sin detalle declarado", sin inventar chips.

---

## 5. Detalle del entorno

Panel lateral de 480 px, sin abandonar el catálogo.

```
Java 21 + Spring Boot                                  [Cerrar]
Entorno OpenJDK 21 con Maven y PostgreSQL para prácticas de backend.
──────────────────────────────────────────────────────────────
Qué incluye
  Java 21                      runtime
  └ Maven                      complemento
  └ Spring Boot                complemento (si está declarado)
  PostgreSQL 16                base de datos
──────────────────────────────────────────────────────────────
Requisitos técnicos                                   [Mostrar]
  Imagen: eclipse-temurin:21-jdk-alpine
  Memoria recomendada: 1 GB
──────────────────────────────────────────────────────────────
                                             [ Usar en laboratorio ]
```

- "Qué incluye" es el árbol principal → complementos → base de datos. Es la vista que el docente reconoce.
- "Requisitos técnicos" está plegado por defecto. Ahí sí aparecen imagen y memoria, porque son consecuencias.
- La compatibilidad se muestra solo si hay una advertencia que cambie la decisión ("Maven no está disponible para esta imagen"). No se muestra el mapa completo de compatibilidad.
- Si `available` es falso, el botón dice "Solicitar entorno" y explica por qué: "Todavía no hay una plantilla ejecutable para este entorno".

---

## 6. Búsqueda y filtros

- El buscador indexa nombre, resumen y **nombre de cada pieza**. "PostgreSQL" tiene que encontrar todos los entornos que lo traen.
- Filtros primarios: modalidad (Todos, Editor persistente, Juez automático).
- Filtros secundarios: categoría, solo si el catálogo crece y la categoría aporta.
- El conteo de resultados se actualiza con el filtro. No hay KPIs fijos.
- Sin resultados: "No hay entornos con PostgreSQL" y un enlace para limpiar filtros, no un estado vacío genérico.

---

## 7. Estados

| Estado | Qué se ve |
| --- | --- |
| Cargando | Skeletons de tarjeta, no un spinner de pantalla completa |
| Catálogo vacío | "Todavía no hay entornos disponibles" + "Solicitar entorno" |
| Sin coincidencias | Filtros activos + "Limpiar filtros" |
| Error | "No se pudo cargar el catálogo" + "Reintentar" |
| No disponible | Tarjeta visible, acción "Solicitar" y motivo en el detalle |

---

## 8. Copia visible

- "Entorno", "herramientas", "qué incluye", "editor persistente", "juez automático".
- No: "imagen Docker", "cgroup", "OOM", "contenedor", "environment", "stack" en la capa 1.
- "Imagen" y "memoria" solo dentro de "Requisitos técnicos".
- Los botones llevan verbo: "Usar en laboratorio", "Solicitar entorno", "Ver detalle".
- La insignia "Oficial" no se usa si no significa que alguien lo revisó.

---

## 9. Qué se retira de la pantalla actual

- La fila de cuatro KPIs.
- Las cifras de RAM y CPU de la tarjeta.
- La fila "Imagen:" de la tarjeta.
- La insignia "Oficial" mientras sea verdadera por defecto.
- El filtro por categoría como único criterio de búsqueda.

---

## 10. Criterios de aceptación

1. Un docente que busca "PostgreSQL" encuentra todos los entornos que lo incluyen, sin conocer la categoría.
2. Ninguna tarjeta muestra el nombre de la imagen ni cifras de recursos.
3. Un entorno sin plantilla ejecutable muestra "Solicitar", no "Usar".
4. "Oficial" aparece solo cuando el dato lo dice; el conteo de oficiales deja de ser el total.
5. La búsqueda y los filtros actualizan el conteo visible.
6. La pantalla funciona a 1024 px sin scroll horizontal; el detalle pasa a pantalla completa por debajo de 900 px.
7. Teclado: se recorren tarjetas y acciones con Tab; el detalle se cierra con Escape.
