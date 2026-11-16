# Estudiante — Nivel 3: Entorno de desarrollo

> **Ruta:** `/student/exercises/:exerciseId`
> **Propósito:** comprobar que el estudiante abre su entorno, conserva el trabajo, entrega el proyecto y entiende el carácter definitivo de un examen.
> **Criterio del nivel:** EST-3.1, EST-3.2 y EST-3.3 en `APROBADO`.

---

## Resumen del Nivel 3

| ID | Subnivel | Caso | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- |
| **EST-3.1** | Fácil | Abrir el entorno y conservar un archivo | El entorno abre en otra pestaña y el cambio persiste al reconectar | `PENDIENTE` |
| **EST-3.2** | Medio | Entregar el proyecto | Snapshot registrado y visible en el historial | `PENDIENTE` |
| **EST-3.3** | Difícil | Entregar un examen y quedar en solo lectura | Confirmación explícita, bloqueo de edición y rechazo de una segunda entrega | `PENDIENTE` |

---

## EST-3.1 (Fácil): Abrir el entorno y conservar un archivo

**Objetivo:** abrir el entorno de una práctica de desarrollo, editar un archivo y comprobar que el trabajo persiste al cerrar y volver a abrir.

**Precondiciones:** práctica de desarrollo publicada con plantilla homologada; EST-1.2 aprobado; cupo de entorno disponible.

**Datos de prueba:** archivo `solucion.py`; cambio `# EST-PILOTO persistencia`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir la práctica de desarrollo. | Se muestra la consigna, la rúbrica y la acción `Abrir entorno`. |
| 2 | Hacer clic en `Abrir entorno`. | El entorno se abre en una pestaña nueva; la práctica no se recarga ni pierde el estado. |
| 3 | En el editor del entorno, agregar el comentario `# EST-PILOTO persistencia` a `solucion.py` y guardar. | El archivo queda guardado sin errores de permisos. |
| 4 | Cerrar la pestaña del entorno y volver a la práctica. | La práctica muestra el estado del entorno y permite volver a abrirlo. |
| 5 | Hacer clic en `Abrir entorno` de nuevo. | El entorno se abre con el comentario todavía presente. |
| 6 | Recargar la práctica y repetir la apertura. | El archivo conserva el cambio; no se restaura la plantilla original sin una acción explícita. |
| 7 | Cerrar sesión, volver a iniciar y abrir la práctica de nuevo. | El entorno y el archivo conservan el cambio; no se restaura la plantilla sin una acción explícita. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del archivo con el comentario antes y después de reconectar |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el entorno no abre, el cambio se pierde al reconectar, o se abre el entorno de otro estudiante.

---

## EST-3.2 (Medio): Entregar el proyecto

**Objetivo:** entregar el contenido actual del entorno y verificar que la entrega queda registrada con su identificador y visible para el docente.

**Precondiciones:** EST-3.1 aprobado; práctica de desarrollo con una única entrega o con reentrega permitida; docente con acceso a la materia.

**Datos de prueba:** archivo `solucion.py` con el cambio de EST-3.1.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir la práctica y verificar que el entorno tiene el archivo modificado. | El editor del entorno muestra el contenido actual. |
| 2 | Hacer clic en `Entregar Proyecto`. | La acción se ejecuta sin pedir copiar archivos a mano. |
| 3 | Revisar la confirmación. | Aparece `Entrega registrada` con el identificador de la entrega. |
| 4 | Abrir `Ver entrega enviada` o el detalle desde `/student/history`. | El detalle muestra el proyecto entregado y la fecha. |
| 5 | Como docente, abrir la entrega desde `/teacher/por-calificar`. | El SpeedGrader muestra el mismo proyecto y los archivos entregados. |
| 6 | Volver a la práctica. | El estado indica que la entrega existe; si la reentrega no está permitida, el botón queda deshabilitado. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de `Entrega registrada` con identificador y del proyecto visto por el docente |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** la entrega no incluye los archivos del entorno, el identificador no se muestra, o el docente ve un proyecto distinto.

---

## EST-3.3 (Difícil): Entregar un examen y quedar en solo lectura

**Objetivo:** comprobar que la entrega de un examen exige confirmación, congela el entorno y rechaza una segunda entrega.

**Precondiciones:** práctica con propósito `Examen / Calificado`; sesión de examen abierta; EST-3.2 aprobado.

**Datos de prueba:** `EST-PILOTO-01` con entorno activo; entrega definitiva.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir la práctica de examen y verificar la marca `Examen`. | La consigna y el entorno están disponibles, con la advertencia de entrega definitiva. |
| 2 | Hacer clic en `Entregar Proyecto`. | El botón cambia a `Confirmar entrega` y aparece la advertencia de solo lectura. |
| 3 | Hacer clic en `Cancelar`. | No se registra ninguna entrega y el entorno sigue editable. |
| 4 | Volver a hacer clic en `Entregar Proyecto` y luego en `Confirmar entrega`. | Se registra la entrega y el entorno pasa a solo lectura. |
| 5 | Intentar editar un archivo del entorno. | La edición se rechaza o el editor está en solo lectura; se muestra el motivo. |
| 6 | Intentar una segunda entrega. | El botón está deshabilitado o la API responde `409`; no se crea otra entrega. |
| 7 | Como docente, reabrir al estudiante con un motivo. | Solo ese estudiante recupera la edición; la acción queda auditada. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de la advertencia, de `Entrega registrada`, del entorno bloqueado y del rechazo de la segunda entrega |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** se entrega sin confirmación, el entorno sigue editable, o una segunda entrega reemplaza la primera.
