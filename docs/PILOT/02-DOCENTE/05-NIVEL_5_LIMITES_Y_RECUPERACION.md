# Docente — Nivel 5: Límites y recuperación

> **Rutas:** `/teacher/cursos/:id`, `/teacher/revision/:submissionId`, `/teacher/aula`
> **Propósito:** comprobar que el sistema aísla materias y entregas, resuelve la revisión concurrente sin pérdida silenciosa y comunica con claridad cuando un entorno no está disponible.
> **Criterio del nivel:** DOC-5.1, DOC-5.2 y DOC-5.3 en `APROBADO`.

---

## Resumen del Nivel 5

| ID | Subnivel | Caso | Ruta principal | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **DOC-5.1** | Fácil | Aislar materias y entregas entre docentes | `/teacher/cursos/:id` | Sin acceso a materias ni entregas ajenas | `PENDIENTE` |
| **DOC-5.2** | Medio | Revisión concurrente de una entrega | `/teacher/revision/:submissionId` | Estado `En revisión` y conflicto explícito al guardar | `PENDIENTE` |
| **DOC-5.3** | Difícil | Entorno no disponible durante la clase | `/teacher/aula` | Estado `Sin entorno` y escalamiento sin bloqueo de la pantalla | `PENDIENTE` |

---

## DOC-5.1 (Fácil): Aislar materias y entregas entre docentes

**Objetivo:** confirmar que un docente sin asignación no puede leer materias, entregas ni comentarios de otro docente.

**Precondiciones:** dos docentes (`DOC-PILOTO-A` y `DOC-PILOTO-B`); materia de A con una entrega; B sin matrícula ni asignación.

**Datos de prueba:** URL de la materia de A y URL directa de una entrega.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Como `DOC-PILOTO-B`, abrir la URL `/teacher/cursos/:id` de la materia de A. | La vista responde con acceso denegado, página no encontrada o estado vacío sin datos. |
| 2 | Abrir la URL directa `/teacher/revision/:submissionId` de una entrega de A. | No se muestra código, nombre ni comentarios del estudiante. |
| 3 | Intentar cargar la lista de entregas de la materia por la interfaz. | La materia no aparece en el selector de B o la petición se rechaza. |
| 4 | Repetir con la sesión de A. | A sí ve la materia y la entrega; el aislamiento no depende del navegador. |
| 5 | Revisar la respuesta de red de los intentos de B. | El cuerpo del error no incluye datos personales ni rutas internas. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Capturas de los intentos de B y de la vista correcta de A; respuestas HTTP |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** B ve cualquier dato de la materia o de la entrega, o el error expone información interna.

---

## DOC-5.2 (Medio): Revisión concurrente de una entrega

**Objetivo:** comprobar que dos docentes pueden identificar una entrega en revisión y que el segundo guardado no borra la calificación del primero sin advertencia.

**Precondiciones:** dos docentes con acceso a la misma materia; una entrega pendiente; DOC-2.1 aprobado.

**Datos de prueba:** docente A califica con `80`; docente B intenta calificar con `70`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir la misma entrega en dos sesiones, una por docente. | Ambas sesiones muestran la entrega y el estado `En revisión`. |
| 2 | Docente A califica con `80` y guarda. | La calificación se registra y la entrega sale de la cola de pendientes. |
| 3 | Docente B intenta guardar `70` sin recargar. | El sistema muestra un conflicto, una advertencia de revisión concurrente o rechaza el guardado. |
| 4 | Recargar la sesión de B. | B ve la calificación de A y el estado actualizado; no se presenta `70` como si fuera la nota vigente. |
| 5 | Revisar la auditoría de la entrega. | Figuran ambas acciones o solo la última con autor y fecha; no hay pérdida silenciosa. |
| 6 | Repetir el guardado de B después de recargar, con motivo. | Si el sistema permite el cambio, queda registrado como ajuste posterior y notifica al estudiante. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Capturas de ambas sesiones, del conflicto y del registro de auditoría |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el segundo guardado reemplaza la nota sin advertencia, o no queda rastro de quién calificó primero.

---

## DOC-5.3 (Difícil): Entorno no disponible durante la clase

**Objetivo:** verificar que el aula informa con claridad cuando un estudiante no tiene entorno y que el docente puede continuar la clase sin una pantalla bloqueada.

**Precondiciones:** sesión de aula abierta; un estudiante sin entorno iniciado; un segundo estudiante con entorno activo.

**Datos de prueba:** `EST-PILOTO-02` sin entorno; `EST-PILOTO-01` con entorno activo.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/teacher/aula` con la sesión activa. | El tablero carga con ambos estudiantes y sus estados. |
| 2 | Localizar a `EST-PILOTO-02`. | La fila muestra `Sin entorno` y no ofrece una consola vacía como si hubiera un entorno. |
| 3 | Hacer clic en la acción de consola sobre `EST-PILOTO-02`. | Se muestra un mensaje que explica que no hay entorno o que no está disponible; no queda un modal en blanco. |
| 4 | Verificar la fila de `EST-PILOTO-01`. | Su entorno y su consola siguen operativos; el fallo de un estudiante no bloquea al resto. |
| 5 | Actualizar el aula mientras el entorno de `EST-PILOTO-02` sigue ausente. | La pantalla no entra en un bucle de carga ni duplica filas. |
| 6 | Registrar el incidente para el administrador. | El estado queda visible para el docente y el caso se puede escalar sin reiniciar la sesión. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de la fila `Sin entorno`, del mensaje de consola y del aula con el otro estudiante operativo |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** la consola queda en blanco, el aula se bloquea, o el problema de un estudiante impide ver o atender a los demás.
