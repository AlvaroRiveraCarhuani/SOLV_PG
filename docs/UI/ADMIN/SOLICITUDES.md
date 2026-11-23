# Vista 10: Solicitudes de Entornos

> **Especificación Oficial de Interfaz, Componentes y Comportamiento**
> **Rol:** Administrador de Institución (Tenant Admin)
> **Ruta:** `/admin/solicitudes-entorno`
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System

---

## 1. Propósito

Responder los pedidos de entorno que los docentes hacen cuando el catálogo no alcanza. Cada solicitud llega armada con las piezas que el sistema reconoce, las que faltan y el uso que tendrá en clase. Aprobar no es cambiar un estado: es enlazar la plantilla que resuelve el pedido.

---

## 2. Composición de la pantalla

### Encabezado
Título «Solicitudes de entornos» y subtítulo que explica el origen de la cola. No tiene acción de alta: las solicitudes nacen en la vista del docente.

### Filtros
Dos botones con contador:

| Filtro | Contenido |
| :--- | :--- |
| En revisión | Solicitudes que esperan respuesta. |
| Respondidas | Aprobadas, con cambio pedido o no viables. |

### Ficha de la solicitud
| Bloque | Contenido |
| :--- | :--- |
| Encabezado | Título, docente solicitante, fecha, modalidad pedida y estado. |
| Piezas reconocidas | Tecnologías del pedido que ya existen en el catálogo. |
| No están en el catálogo | Piezas que faltan registrar; el pedido no se puede resolver completo hasta cubrirlas. |
| Uso declarado | Practicar en clase, entregar y calificar, o ambos. |
| Materias que lo esperan | Alcance real del pedido. |
| Justificación | Qué deben lograr los estudiantes y el texto libre del docente. |
| Respuesta guardada | La nota de una respuesta anterior, si existe. |
| Plantilla resuelta | Nombre de la plantilla enlazada cuando la solicitud fue aprobada. |

El docente se nombra una sola vez, en la línea de metadatos de la ficha, junto al estado, la modalidad y la fecha. Las capas de las piezas se muestran en español: *Lenguaje principal*, *Interfaz web*, *Complemento*, *Base de datos*. La nota adicional del docente solo aparece cuando escribió algo.

Modo de operaciones (uso declarado): el docente declara si el entorno se usará para practicar en clase, para entregar y calificar o para ambas cosas. La modalidad pedida decide qué plantillas puede aceptar la solicitud.

---

## 3. Responder

`[Responder]` despliega el panel de revisión dentro de la ficha, sin salir de la cola.

1. **Plantilla que resuelve la solicitud.** Solo aparecen plantillas aprobadas que cubren la modalidad pedida. Si la lista está vacía, el panel indica que hay que crear el entorno en el Constructor y volver.
2. **Crear en Constructor.** Abre `/admin/plantillas/nueva?requestId=<id>` en la misma pestaña. El Constructor carga la ficha, precarga modalidad, descripción y piezas, y muestra las que faltan en el catálogo. Al publicar con el checkbox activo, la solicitud queda aprobada y vinculada a la plantilla nueva; si el enlace falla, la plantilla se conserva y el aviso pide vincularla a mano.
3. **Respuesta para el docente.** Texto libre que queda guardado en la ficha.
4. **Decisión.**

| Acción | Requisito | Efecto |
| :--- | :--- | :--- |
| Aprobar con esta plantilla | Plantilla elegida | Registra el entorno y marca la solicitud como aprobada. La nota es opcional. |
| Pedir ajuste | Nota de al menos 5 caracteres | La solicitud sigue en revisión con la explicación para el docente. |
| No es viable | Nota de al menos 5 caracteres | Cierra la solicitud como no viable con el motivo guardado. |
| Cancelar | — | Cierra el panel sin guardar. |

Al aprobar, la solicitud pasa a «Respondidas». Pedir ajuste y no viable la mantienen en «En revisión» para que el docente complete o corrija el pedido.

---

## 4. Estados

| Estado | Etiqueta | Tono |
| :--- | :--- | :--- |
| `EN_REVISION` | En revisión | Advertencia |
| `APROBADA` | Aprobada | Éxito |
| `NECESITA_CAMBIO` | Necesita un cambio | Advertencia |
| `NO_VIABLE` | No es viable | Error |

---

## 5. Reglas de negocio

- Una solicitud respondida no vuelve a la cola de revisión.
- Aprobar exige una plantilla compatible con la modalidad pedida; el sistema filtra las incompatibles.
- Si el pedido incluye piezas que no existen en el catálogo, primero se registran y se construye el entorno.
- Toda respuesta queda asociada al administrador que la emitió.
- La solicitud no crea entornos por sí sola: el entorno es la plantilla enlazada.

---

## 6. Contratos

| Método | Endpoint | Propósito |
| :--- | :--- | :--- |
| `GET` | `/api/v1/admin/environment-requests` | Cola de solicitudes, opcionalmente filtrada por estado. |
| `GET` | `/api/v1/admin/environment-requests/{id}` | Ficha completa de una solicitud, con `stack_config` y piezas, para el Drawer y el Constructor. |
| `PATCH` | `/api/v1/admin/environment-requests/{id}` | Responder con `status`, `review_note` y `resolved_template_id` cuando se aprueba. |
| `POST` | `/api/v1/teacher/environment-requests` | Origen del pedido, desde la vista del docente. |
| `GET` | `/api/v1/admin/templates?status=approved` | Plantillas aprobadas que pueden resolver una solicitud. |

---

## 7. Guías de la pantalla

El recorrido explica la cola, la ficha y dónde se responde. El tutorial exige abrir una solicitud y elegir una plantilla compatible, y termina explicando las tres salidas sin aprobar ni rechazar nada.

La sección equivalente del manual es «3.2 Ciclo de Vida y Gobernanza de Plantillas».
