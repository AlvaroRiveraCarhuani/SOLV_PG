# Docente — Nivel 3: Aula y seguimiento

> **Rutas:** `/teacher/aula`, `/teacher/revision/:submissionId`
> **Propósito:** comprobar que el docente detecta a un estudiante trabado, emite un avance de código y cierra un examen con los entornos en solo lectura.
> **Criterio del nivel:** DOC-3.1, DOC-3.2 y DOC-3.3 en `APROBADO`.

---

## Resumen del Nivel 3

| ID | Subnivel | Caso | Ruta principal | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **DOC-3.1** | Fácil | Atender a un estudiante trabado | `/teacher/aula` | Fila `Trabados`, consola de solo lectura y diagnóstico visible | `PENDIENTE` |
| **DOC-3.2** | Medio | Emitir un avance de código | `/teacher/aula` | Avance aplicado por el estudiante y registrado en el aula | `PENDIENTE` |
| **DOC-3.3** | Difícil | Cerrar un examen y congelar entornos | `/teacher/aula` | Entornos en solo lectura y reapertura individual auditada | `PENDIENTE` |

---

## DOC-3.1 (Fácil): Atender a un estudiante trabado

**Objetivo:** detectar una entrega con error repetido, abrir la consola del estudiante y confirmar que el docente puede diagnosticar sin modificar el entorno.

**Precondiciones:** sesión de aula abierta con una práctica de evaluación; `EST-PILOTO-01` dentro del aula.

**Datos de prueba:** solución con error de ejecución, por ejemplo división por cero; tres envíos consecutivos.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Como `EST-PILOTO-01`, enviar tres veces una solución que falla con error de ejecución. | Cada envío produce un veredicto `RE` y el historial de intentos lo registra. |
| 2 | Como docente, abrir `/teacher/aula`. | La fila del estudiante cambia a `Se trabó` o queda incluida en el filtro `Trabados`. |
| 3 | Hacer clic en el filtro `Trabados`. | Solo se muestran los estudiantes con ese estado; el contador coincide con las filas. |
| 4 | Hacer clic en `Consola` en la fila del estudiante. | Se abre `Consola del Estudiante · Asistencia en Vivo` con la salida del entorno. |
| 5 | Intentar escribir en la consola. | La consola es de solo lectura o rechaza la entrada; no modifica archivos del estudiante. |
| 6 | Cerrar la consola y abrir la entrega desde el enlace de revisión. | El SpeedGrader muestra el código y el error; el docente puede dejar un comentario. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de la fila `Trabados`, de la consola con la salida y del veredicto `RE` |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el estudiante trabado no aparece en el filtro, la consola queda en blanco sin explicación, o la consola permite escribir en el entorno del estudiante.

---

## DOC-3.2 (Medio): Emitir un avance de código

**Objetivo:** emitir un avance durante la clase y comprobar que el estudiante lo recibe, lo confirma y que el aula registra quién lo aplicó.

**Precondiciones:** sesión de aula abierta con una práctica de desarrollo; estudiante con entorno activo; docente con una versión de código válida.

**Datos de prueba:** avance con un archivo `solucion.py` y un cambio visible en el mensaje de bienvenida.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Como docente, abrir `/teacher/aula` con la sesión activa. | La cabecera muestra `Emitir avance` habilitado. |
| 2 | Hacer clic en `Emitir avance` y confirmar el envío. | El avance queda emitido y aparece en `Avances de esta clase`. |
| 3 | Como `EST-PILOTO-01`, abrir la práctica. | Se muestra el aviso `Tu docente emitió un avance de esta práctica` con el número de archivos. |
| 4 | Abrir el avance y leer la advertencia. | El texto indica que no cambia nada hasta confirmar. |
| 5 | Confirmar la aplicación. | Los archivos del entorno se actualizan y el aviso desaparece. |
| 6 | Volver al aula como docente. | El aula registra al estudiante como aplicado, descartado o pendiente; no lo cuenta dos veces. |
| 7 | Repetir con un estudiante que no confirma. | El aula lo mantiene como pendiente y no marca el avance como aplicado. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del avance emitido, del aviso del estudiante y del estado en `Avances de esta clase` |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el avance se aplica sin confirmación, se pierde el código previo del estudiante sin advertencia, o el aula no distingue aplicados de pendientes.

---

## DOC-3.3 (Difícil): Cerrar un examen y congelar entornos

**Objetivo:** cerrar una sesión de examen, verificar que los entornos quedan en solo lectura y reabrir un estudiante con motivo registrado.

**Precondiciones:** práctica con propósito `Examen / Calificado`; sesión de aula abierta con esa práctica; al menos dos estudiantes con entorno activo.

**Datos de prueba:** estudiantes `EST-PILOTO-01` y `EST-PILOTO-02`; motivo de reapertura `Incidente técnico documentado`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir el aula del examen. | La cabecera muestra la marca `Examen` y la acción `Cerrar examen`. |
| 2 | Verificar que ambos estudiantes tienen el entorno abierto. | Las filas muestran entorno activo y la práctica de la sesión. |
| 3 | Hacer clic en `Cerrar examen` y confirmar. | La sesión se cierra y se muestra el panel `Entornos del examen`. |
| 4 | Revisar el panel. | Indica cuántos entornos quedaron en solo lectura y si algún cierre falló. |
| 5 | Como `EST-PILOTO-01`, intentar editar el entorno. | El editor queda en solo lectura o rechaza la edición; se muestra el motivo del cierre. |
| 6 | Como docente, hacer clic en `Reabrir` sobre `EST-PILOTO-01`. | El formulario exige un motivo; sin motivo no se puede confirmar. |
| 7 | Escribir el motivo y confirmar. | Solo ese estudiante recupera la edición; el resto sigue en solo lectura y la acción queda auditada. |
| 8 | Intentar reabrir sin motivo o reabrir a un estudiante que no pertenece al examen. | La acción se rechaza con un mensaje claro; no se modifica el estado de otros. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del panel `Entornos del examen`, del entorno en solo lectura y del registro de reapertura con motivo |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** un entorno sigue editable después del cierre, la reapertura afecta a más de un estudiante, o el motivo no queda registrado.
