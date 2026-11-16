# Estudiante — Nivel 5: Límites y casos borde

> **Rutas:** `/student`, `/student/history`, `/student/exercises/:exerciseId`
> **Propósito:** comprobar que el estudiante no ve notas inventadas, no puede duplicar una entrega definitiva y no accede a información de otro estudiante.
> **Criterio del nivel:** EST-5.1, EST-5.2 y EST-5.3 en `APROBADO`.

---

## Resumen del Nivel 5

| ID | Subnivel | Caso | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- |
| **EST-5.1** | Fácil | Práctica sin entrega | Sin nota ni entrega ficticia; estado pendiente explícito | `PENDIENTE` |
| **EST-5.2** | Medio | Doble envío de una entrega definitiva | Una sola entrega; segundo intento rechazado con `409` o botón deshabilitado | `PENDIENTE` |
| **EST-5.3** | Difícil | Acceso a una entrega ajena | Denegación sin datos del otro estudiante | `PENDIENTE` |

---

## EST-5.1 (Fácil): Práctica sin entrega

**Objetivo:** verificar que una práctica publicada pero no resuelta no produce una nota, un veredicto ni una entrega ficticia.

**Precondiciones:** práctica publicada con vencimiento futuro; `EST-PILOTO-01` sin envíos; EST-1.1 aprobado.

**Datos de prueba:** práctica `DOC-PILOTO Sin entregar`; cuenta sin historial para esa práctica.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/student`. | La práctica aparece en `Próximas entregas` si tiene fecha próxima; no aparece como entrega realizada. |
| 2 | Abrir la práctica. | Se muestra la consigna y el editor, sin veredicto ni nota. |
| 3 | Volver al inicio sin enviar. | `Últimas entregas` sigue sin esa práctica; no se registra una entrega formal. |
| 4 | Abrir `/student/history`. | La práctica no figura en el historial o figura como pendiente, según el contrato; nunca con una nota inventada. |
| 5 | Como docente, revisar la cola de la práctica. | El estudiante aparece como sin entrega, no como calificado con cero. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del inicio, de la práctica sin veredicto y del historial sin nota |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el sistema muestra una nota, un veredicto o una entrega que el estudiante no produjo.

---

## EST-5.2 (Medio): Doble envío de una entrega definitiva

**Objetivo:** comprobar que una entrega de examen no se duplica por un doble clic, una recarga o una segunda pestaña.

**Precondiciones:** EST-3.3 aprobado; examen con entrega definitiva; entorno activo.

**Datos de prueba:** dos clics rápidos en `Confirmar entrega`; una segunda pestaña con la misma práctica.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir el examen en dos pestañas con la misma sesión. | Ambas muestran la práctica y el entorno, sin errores de sesión. |
| 2 | En la primera pestaña, hacer clic dos veces rápido en `Confirmar entrega`. | Se registra una sola entrega; el botón se deshabilita durante el envío. |
| 3 | Revisar el historial. | Aparece una única entrega para el examen, con un identificador. |
| 4 | En la segunda pestaña, intentar `Entregar Proyecto`. | La acción se rechaza con `409`, el botón está deshabilitado o se muestra que ya existe una entrega. |
| 5 | Recargar ambas pestañas. | El estado `Examen entregado` se mantiene y no se ofrece una nueva entrega. |
| 6 | Revisar la respuesta de red del segundo intento. | No se crea un segundo recurso ni se modifica el primero. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del historial con una sola entrega y de la respuesta `409` o del botón deshabilitado |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** se crean dos entregas, la segunda reemplaza la primera, o el estado de solo lectura se pierde al recargar.

---

## EST-5.3 (Difícil): Acceso a una entrega ajena

**Objetivo:** confirmar que un estudiante no puede leer el historial, el código ni los comentarios de otro estudiante cambiando el identificador de la URL.

**Precondiciones:** dos estudiantes con entregas; `EST-PILOTO-01` conoce el identificador de una entrega de `EST-PILOTO-02` por el entorno de prueba.

**Datos de prueba:** URL `/student/history/:submissionId` de la entrega de `EST-PILOTO-02`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Como `EST-PILOTO-01`, abrir la URL de una entrega propia. | El detalle se muestra correctamente. |
| 2 | Cambiar el identificador por el de una entrega de `EST-PILOTO-02`. | La vista responde con acceso denegado o página no encontrada. |
| 3 | Revisar el cuerpo de la respuesta. | No incluye código, nombre, correo, nota ni comentarios del otro estudiante. |
| 4 | Intentar la misma operación desde la API de historial. | La respuesta se rechaza o devuelve solo las entregas del estudiante autenticado. |
| 5 | Repetir con un identificador inexistente. | La respuesta es un error controlado, sin traza interna ni datos de otro recurso. |
| 6 | Volver al historial propio. | Las entregas de `EST-PILOTO-01` siguen disponibles. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Capturas de la URL ajena y de la respuesta; comparación con el historial propio |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el estudiante ve cualquier dato de la entrega ajena, o el error expone identificadores internos o trazas.
