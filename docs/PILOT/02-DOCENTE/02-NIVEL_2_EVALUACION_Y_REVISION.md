# Docente — Nivel 2: Evaluación y revisión

> **Rutas:** `/teacher/por-calificar`, `/teacher/revision/:submissionId`
> **Propósito:** comprobar que la cola de revisión refleja solo trabajo pendiente, que la rúbrica calcula la nota y que el ajuste manual queda registrado sin romper el historial.
> **Criterio del nivel:** DOC-2.1, DOC-2.2 y DOC-2.3 en `APROBADO`.

---

## Resumen del Nivel 2

| ID | Subnivel | Caso | Ruta principal | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **DOC-2.1** | Fácil | Tomar una entrega de la cola | `/teacher/por-calificar` | La entrega pendiente abre en el SpeedGrader y sale de la cola al calificarse | `PENDIENTE` |
| **DOC-2.2** | Medio | Calificar con rúbrica y comentarios | `/teacher/revision/:submissionId` | Nota ponderada correcta y retroalimentación visible para el estudiante | `PENDIENTE` |
| **DOC-2.3** | Difícil | Ajustar la calificación manualmente | `/teacher/revision/:submissionId` | Ajuste aplicado, motivo registrado y estudiante notificado | `PENDIENTE` |

---

## DOC-2.1 (Fácil): Tomar una entrega de la cola

**Objetivo:** localizar una entrega pendiente, abrirla y confirmar que la cola solo contiene entregas sin corregir.

**Precondiciones:** `EST-PILOTO-01` envió una solución a la práctica de evaluación de DOC-1.2; el docente tiene acceso a la materia.

**Datos de prueba:** entrega de `EST-PILOTO-01`; práctica `DOC-PILOTO Suma de enteros`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/teacher/por-calificar`. | La vista muestra la lista de prácticas con pendientes y el total de entregas en espera. |
| 2 | Verificar la práctica `DOC-PILOTO Suma de enteros` en el panel izquierdo. | La fila indica materia, título y número de pendientes; no muestra veredictos del juez. |
| 3 | Seleccionar la práctica. | El panel derecho lista las entregas de la más antigua a la más nueva, con alumno y fecha. |
| 4 | Hacer clic en `Calificar` sobre la entrega de `EST-PILOTO-01`. | Se abre `/teacher/revision/:submissionId` con el código, los casos y el panel de evaluación. |
| 5 | Verificar que la entrega figure como `En revisión` al volver a la cola en otra sesión de docente. | La bandera `En revisión` aparece bajo la fecha, sin sacar la entrega de la cola. |
| 6 | Calificar la entrega y volver a `/teacher/por-calificar`. | La entrega calificada ya no aparece entre las pendientes. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de la cola antes y después, e identificador de la entrega |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** la cola muestra entregas ya calificadas, el enlace `Calificar` no abre la entrega correcta, o la entrega permanece en la cola después de guardar.

---

## DOC-2.2 (Medio): Calificar con rúbrica y comentarios

**Objetivo:** seleccionar un nivel por criterio, guardar la calificación y comprobar que la nota y los comentarios llegan al estudiante.

**Precondiciones:** práctica de desarrollo de DOC-1.3 con dos criterios de `50%`; entrega de proyecto de `EST-PILOTO-01`.

**Datos de prueba:** criterio `Funcionalidad` en el nivel superior definido; criterio `Calidad` en el nivel intermedio; comentario general `Revisar el manejo de errores en el endpoint de listado`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir la entrega desde `/teacher/por-calificar`. | El SpeedGrader muestra el proyecto, el árbol de archivos y el panel `Evaluación por Rúbrica`. |
| 2 | Seleccionar el nivel superior para `Funcionalidad` y el intermedio para `Calidad`. | El panel muestra la nota calculada y actualiza el total sin recargar la página. |
| 3 | Escribir el comentario general y guardar con `Guardar Calificación`. | La acción termina sin error y el panel conserva los niveles seleccionados. |
| 4 | Verificar el cálculo con los puntajes de los niveles elegidos y pesos iguales. | La nota coincide con el promedio ponderado; no se redondea de forma distinta a la mostrada. |
| 5 | Como `EST-PILOTO-01`, abrir `/student/history/:submissionId`. | El detalle muestra la nota, el bloque `Evaluación por Rúbrica` y el comentario general. |
| 6 | Recargar el SpeedGrader. | Los niveles y el comentario persisten; no se duplica la evaluación. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del panel con la nota calculada y captura del detalle del estudiante con la misma nota |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** la nota calculada no coincide con la suma ponderada, el comentario no llega al estudiante, o al recargar se pierden los niveles.

---

## DOC-2.3 (Difícil): Ajustar la calificación manualmente

**Objetivo:** aplicar un ajuste manual sobre una entrega calificada, registrar el motivo y verificar que el estudiante ve la nota ajustada y que el historial conserva la acción.

**Precondiciones:** DOC-2.2 aprobado; entrega con nota previa; docente con permiso de ajuste.

**Datos de prueba:** puntaje nuevo `85`; motivo `Se reconoce el esfuerzo adicional en la validación de entradas`; notificación al estudiante activada.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir la entrega calificada en el SpeedGrader. | La nota anterior y la evaluación por rúbrica son visibles. |
| 2 | Hacer clic en `Ajuste manual` o `Calificar`, según la modalidad. | Se abre el modal `Modificar Calificación Manualmente`. |
| 3 | Escribir `85` en `Puntaje Asignado (0-100)`, el motivo y marcar `Notificar al estudiante por correo electrónico`. | El formulario acepta los datos y muestra el puntaje dentro del rango. |
| 4 | Hacer clic en `Aplicar Modificación`. | El modal se cierra, la nota visible cambia a `85` y aparece una confirmación de guardado. |
| 5 | Intentar un puntaje `101` en una segunda apertura. | El formulario rechaza el valor o muestra un error de validación; no se guarda. |
| 6 | Como `EST-PILOTO-01`, revisar el detalle de la entrega. | La nota ajustada y el motivo son visibles; la evaluación por rúbrica anterior se conserva como antecedente. |
| 7 | Revisar el historial o la auditoría de la entrega. | El ajuste figura con autor, fecha, puntaje anterior y puntaje nuevo. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del modal, de la nota ajustada en el estudiante y del registro de auditoría |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** se acepta un puntaje fuera de rango, el ajuste no queda auditado, o el estudiante no ve la nota actualizada.
