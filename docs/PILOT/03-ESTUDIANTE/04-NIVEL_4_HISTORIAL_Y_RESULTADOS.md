# Estudiante — Nivel 4: Historial y resultados

> **Rutas:** `/student/history`, `/student/history/:submissionId`
> **Propósito:** comprobar que el estudiante encuentra sus entregas, entiende la retroalimentación y distingue una nota de una entrega sin corregir.
> **Criterio del nivel:** EST-4.1, EST-4.2 y EST-4.3 en `APROBADO`.

---

## Resumen del Nivel 4

| ID | Subnivel | Caso | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- |
| **EST-4.1** | Fácil | Consultar el historial de entregas | Lista propia, filtrable y con acceso al detalle | `PENDIENTE` |
| **EST-4.2** | Medio | Leer la rúbrica y los comentarios | Nota y retroalimentación coincidentes con la revisión docente | `PENDIENTE` |
| **EST-4.3** | Difícil | Revisar una entrega de proyecto | Árbol de archivos y comentarios por línea sin posibilidad de edición | `PENDIENTE` |

---

## EST-4.1 (Fácil): Consultar el historial de entregas

**Objetivo:** localizar una entrega propia, distinguir su estado y abrir el detalle correcto.

**Precondiciones:** EST-2.1 o EST-3.2 aprobado; al menos una entrega registrada; otra cuenta con entregas distintas.

**Datos de prueba:** entrega de `EST-PILOTO-01`; materia `DOC-PILOTO-01`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/student/history`. | Se muestra `Historial de entregas` con el total y las entregas del estudiante. |
| 2 | Revisar las columnas. | Figuran práctica o evaluación, materia, nota o estado, tiempo, fecha y acción. |
| 3 | Filtrar por `Materia`. | La lista se reduce a `DOC-PILOTO-01`; el total coincide con las filas. |
| 4 | Abrir una entrega. | Se navega a `/student/history/:submissionId` con el detalle correcto. |
| 5 | Volver al historial y limpiar el filtro. | Se recuperan todas las entregas propias. |
| 6 | Verificar que no aparece ninguna entrega de `EST-PILOTO-02`. | La lista solo contiene entregas del estudiante autenticado. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del historial filtrado y del detalle abierto; comparación con la otra cuenta |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el historial muestra entregas ajenas, el filtro no coincide con las filas, o el detalle abre otra entrega.

---

## EST-4.2 (Medio): Leer la rúbrica y los comentarios

**Objetivo:** comprobar que la nota y la retroalimentación visibles para el estudiante coinciden con lo guardado por el docente.

**Precondiciones:** DOC-2.2 aprobado; entrega de proyecto calificada con rúbrica y comentario general; EST-4.1 aprobado.

**Datos de prueba:** nota `85`; criterios `Funcionalidad` y `Calidad`; comentario general de DOC-2.2.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir el detalle de la entrega calificada. | Se muestra la nota y el bloque `Evaluación por Rúbrica`. |
| 2 | Revisar cada criterio. | Figuran el nivel obtenido, el peso y el comentario del docente cuando exista. |
| 3 | Verificar la nota total. | Coincide con la calificación guardada por el docente; no se recalcula con otro criterio. |
| 4 | Revisar el comentario general. | El texto es el mismo que escribió el docente y está asociado a la entrega correcta. |
| 5 | Comparar con la vista del docente en el SpeedGrader. | Niveles, nota y comentarios coinciden en ambas vistas. |
| 6 | Recargar el detalle. | La información se mantiene y no aparece una nota anterior como vigente. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del detalle con nota y rúbrica; captura equivalente del SpeedGrader |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** la nota del estudiante difiere de la del docente, falta un criterio, o el comentario aparece en otra entrega.

---

## EST-4.3 (Difícil): Revisar una entrega de proyecto

**Objetivo:** navegar el proyecto entregado, abrir un archivo y leer los comentarios por línea sin modificar la entrega.

**Precondiciones:** EST-3.2 y EST-4.2 aprobados; entrega de proyecto con varios archivos y al menos un comentario por línea.

**Datos de prueba:** proyecto con `app.py`, `requirements.txt` y un comentario del docente en `app.py`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir el detalle de la entrega de proyecto. | Se muestra el árbol de archivos y el contenido del archivo seleccionado. |
| 2 | Seleccionar `app.py` en el árbol. | El visor muestra el archivo correcto y los comentarios asociados. |
| 3 | Leer el contador de comentarios. | El número coincide con los comentarios visibles del archivo. |
| 4 | Seleccionar otro archivo. | El visor cambia de archivo sin mezclar comentarios del anterior. |
| 5 | Intentar editar el código desde el detalle. | El contenido es de solo lectura; no existe una acción de guardado. |
| 6 | Volver al historial y abrir de nuevo la entrega. | El árbol, el archivo y los comentarios se mantienen. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del árbol, del archivo con comentarios y del contador |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el detalle permite editar la entrega, los comentarios se asignan al archivo equivocado, o el árbol no carga.
