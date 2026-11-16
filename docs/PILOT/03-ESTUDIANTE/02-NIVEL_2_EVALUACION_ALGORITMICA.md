# Estudiante — Nivel 2: Evaluación algorítmica

> **Ruta:** `/student/exercises/:exerciseId`
> **Propósito:** comprobar que el estudiante entiende los casos de ejemplo, envía una solución y recibe veredictos accionables sin depender de la consola del docente.
> **Criterio del nivel:** EST-2.1, EST-2.2 y EST-2.3 en `APROBADO`.

---

## Resumen del Nivel 2

| ID | Subnivel | Caso | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- |
| **EST-2.1** | Fácil | Resolver y enviar una solución correcta | Veredicto `AC` e identificador de entrega | `PENDIENTE` |
| **EST-2.2** | Medio | Corregir una respuesta incorrecta | Veredicto `WA`, corrección y nuevo envío `AC` | `PENDIENTE` |
| **EST-2.3** | Difícil | Interpretar tiempo excedido y error de ejecución | Veredictos `TLE` y `RE` con mensajes accionables | `PENDIENTE` |

---

## EST-2.1 (Fácil): Resolver y enviar una solución correcta

**Objetivo:** leer los casos de ejemplo, escribir una solución y enviarla al entorno de evaluación.

**Precondiciones:** práctica de evaluación publicada; lenguaje `Python`; estudiante matriculado; EST-1.2 aprobado.

**Datos de prueba:** suma de dos enteros; entrada `2 3`; salida esperada `5`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir la práctica desde `/student` o `/student/courses/:courseId`. | Se muestra la consigna, el lenguaje seleccionado y el bloque `Casos de Ejemplo (N)`. |
| 2 | Leer el caso de ejemplo. | La entrada y la salida esperada son visibles; no se muestra ningún caso privado. |
| 3 | Escribir una solución correcta en el editor. | El editor conserva el código y permite enviarlo. |
| 4 | Hacer clic en `Enviar a Evaluación`. | El botón pasa a `Evaluando...` y luego muestra un resultado. |
| 5 | Revisar el veredicto. | El resultado es `AC` y se muestra el identificador de la entrega. |
| 6 | Abrir `/student/history`. | La entrega aparece con la práctica, la fecha, la nota o el veredicto y un enlace al detalle. |
| 7 | Recargar la práctica. | El código enviado o el estado de entrega se conserva; no se pierde el intento. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del veredicto `AC` con identificador y de la fila en el historial |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el caso privado se muestra, el envío no produce veredicto, o la entrega no aparece en el historial.

---

## EST-2.2 (Medio): Corregir una respuesta incorrecta

**Objetivo:** recibir un veredicto `WA`, identificar el caso fallido y corregir la solución hasta obtener `AC`.

**Precondiciones:** EST-2.1 aprobado; práctica con al menos un caso público y uno privado.

**Datos de prueba:** primera solución que devuelve la resta en lugar de la suma; segunda solución correcta.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Enviar una solución incorrecta. | El veredicto es `WA`; el intento queda registrado. |
| 2 | Abrir el detalle del resultado. | Se identifica el caso fallido sin revelar la entrada o salida de los casos privados. |
| 3 | Corregir el código en el editor. | El editor conserva la nueva versión y no borra el historial anterior. |
| 4 | Volver a enviar. | El nuevo intento produce `AC` y se agrega como intento distinto. |
| 5 | Revisar el historial de intentos. | Figuran al menos dos intentos con veredicto, tiempo y fecha; el orden es cronológico. |
| 6 | Abrir el detalle de la entrega corregida. | Se muestra el código de la última entrega y el veredicto `AC`. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de `WA`, del caso fallido y del segundo intento `AC`; historial con dos intentos |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el caso privado revela su contenido, el segundo envío reemplaza el primero sin historial, o el veredicto no cambia después de corregir.

---

## EST-2.3 (Difícil): Interpretar tiempo excedido y error de ejecución

**Objetivo:** comprobar que los veredictos `TLE` y `RE` se explican con suficiente contexto para que el estudiante pueda corregir sin asistencia.

**Precondiciones:** EST-2.2 aprobado; práctica con límite de tiempo configurado; lenguaje `Python`.

**Datos de prueba:** bucle infinito para `TLE`; acceso a un índice fuera de rango o error de sintaxis para `RE`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Enviar una solución con un bucle infinito. | El veredicto es `TLE`; el intento termina y no deja el editor bloqueado. |
| 2 | Revisar el mensaje del resultado. | El mensaje indica que se excedió el tiempo y muestra el límite configurado. |
| 3 | Corregir el bucle y enviar una solución que provoque un error de ejecución. | El veredicto es `RE` y el mensaje identifica el tipo de error. |
| 4 | Corregir el error y enviar una solución válida. | El veredicto es `AC`; los intentos anteriores se conservan. |
| 5 | Revisar el historial de intentos. | Los tres veredictos aparecen en orden, con tiempo y memoria cuando estén disponibles. |
| 6 | Recargar la página después de cada veredicto. | El estado y el historial se mantienen; no se repite el envío de forma automática. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de `TLE`, de `RE`, del mensaje de cada uno y del historial con los tres intentos |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el estudiante no recibe ningún mensaje, el editor queda bloqueado después de `TLE`, o los intentos se pierden al recargar.
