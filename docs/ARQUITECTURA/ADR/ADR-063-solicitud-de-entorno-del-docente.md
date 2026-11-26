# ADR 063: Solicitud de Entorno del Docente como Intención y Receta

> **Estado:** Implementado
> **Fecha:** 2026-11-02
> **Ámbito:** Docencia, Catálogo de plantillas, Constructor de entornos
> **Relacionado:** ADR-030 (catálogo y aprobación), ADR-043 (modalidad dual), ADR-046 (constructor de entornos), ADR-048 (copiloto de arquitectura), ADR-054 (catálogo del docente)

---

## Contexto

El docente que no encontraba un entorno en `/teacher/templates` abría el modal **Solicitar Nuevo Entorno** y lo enviaba a `POST /api/v1/templates`. Ese endpoint es el repositorio legacy y ejecuta:

```sql
INSERT INTO lab_templates (name, docker_image, base_ram_mb) VALUES ($1, $2, $3)
```

Solo sobrevivían tres columnas. La justificación, el `stack_config`, la modalidad y la materia se descartaban en el camino. Peor: `tenant_id` y `requested_by` quedaban nulos, de modo que la solicitud aparecía como **"Docente de la institución"**, se mostraba en la cola de todos los tenants (`tenant_id = $1 OR tenant_id IS NULL`) y el frontend la marcaba `is_official = true` porque esa bandera se deriva de `requested_by IS NULL`. La ruta, además, no pasaba por el middleware de auth ni de tenant.

El formulario en sí pedía decisiones de administrador: elegir una **imagen con tag**, un **perfil de carga** que mapeaba a RAM fija (512/1024/2048) y, en un segundo paso, confirmar la imagen como texto editable. Los frameworks y las bases de datos se calculaban en el componente pero **no se renderizaban**: solo se podía elegir el lenguaje. La modalidad se fijaba a `IDE_PERSISTENTE` aunque el modelo ya soporta `AMBOS`.

El resultado fue medible: la base tenía seis plantillas idénticas, todas aprobadas, con el mismo `docker_image` (`solv-local/stack:136f61d22016`) y nombres distintos (`IDE Python + Postgres #391940` … `#434098`), cada una enlazada a una materia distinta. El constructor reutilizaba piezas del catálogo, pero nadie comparaba recetas terminadas, y el nombre único del tenant forzaba un sufijo aleatorio en lugar de la reutilización.

---

## Decisión

### 1. La solicitud es una entidad propia, no una plantilla

Nueva tabla `environment_requests` con `tenant_id`, `requested_by`, título, descripción, el texto original del docente, los usos declarados, la modalidad derivada, el `stack_config`, la foto de piezas resueltas, las piezas faltantes, las materias que la esperan, la huella y el estado de revisión.

La plantilla definitiva se sigue construyendo en el constructor del administrador. La solicitud solo se enlaza a ella (`resolved_template_id`) al aprobar.

### 2. El docente declara uso, no recursos

Desaparece el perfil de carga. El docente marca **"para practicar en clase"** y/o **"para entregar y calificar"**; el backend deriva `IDE_PERSISTENTE`, `JUEZ_EFIMERO` o `AMBOS`, y la RAM recomendada sale del constructor (`RecommendedRamMB`), que ya la calcula según las piezas. El administrador sigue fijando la cuota real de cgroups.

### 3. Una pantalla, tres bloques y un panel de resumen

`template-request-modal` deja de ser un asistente de dos pasos y pasa a ser una pantalla dedicada:

1. **Cuéntanos tu práctica.** Un solo campo obligatorio: qué van a hacer los estudiantes y qué deben lograr. Ese texto es el que decide la administración y el que alimenta al copiloto; no hay un segundo campo que pida lo mismo dos veces. Debajo, las piezas convertidas en etiquetas editables, con un selector que no se cierra al elegir para poder sumar un lenguaje y su complemento en la misma pasada. Las piezas que no están en el catálogo se muestran igual, marcadas como "no la tenemos, la pedimos".
2. **¿Cómo lo van a usar?** Las dos casillas de uso, nombradas por el entorno que producen: entorno de desarrollo para practicar en clase y entorno de evaluación para entregar y calificar.
3. **Dónde se va a usar.** Materia (opcional, con el combobox compartido) y nombre propuesto. La materia no limita: el entorno aprobado queda en el catálogo y puede usarse en otras materias y cursos.

El panel lateral muestra exactamente lo que va a leer el administrador, en tarjetas: nombre, herramientas, uso, materia y qué deben lograr. Los detalles técnicos quedan plegados al final, se arman con la selección del docente aunque la receta todavía esté en camino, y la imagen nunca es editable.

Materia y nombre son opcionales: si el docente no escribe nombre, el backend usa el que sugiere el constructor, y una solicitud sin materia sigue siendo revisable. La materia solo indica a quién le sirve hoy; el entorno aprobado queda en el catálogo para las demás.

### 4. Huella de la receta y reutilización

`stack_signature` combina imagen base, capas de build, servicios satélite y modalidad. Se usa en tres lugares:

- **Sumarse:** si ya hay una solicitud `EN_REVISION` con la misma huella, la nueva materia se agrega a esa ficha en lugar de abrir otra.
- **Ofrecer lo existente:** si la receta produce una imagen construida y ya existe una plantilla aprobada con esa referencia, el docente ve "ya existe este entorno" antes de enviar.
- **Detección de duplicados:** la misma huella en dos plantillas significa el mismo entorno.

Las piezas se siguen resolviendo contra `template_components`, de modo que una variante (React → Angular) reutiliza las piezas existentes y solo cambia una.

El atajo **"quiero algo así, pero con…"** abre la solicitud con el texto y las piezas de la plantilla de origen ya puestas: el docente quita la que no quiere y agrega la que sí. Para eso `GET /api/v1/templates` devuelve `stack_config`, que antes se quedaba en la base. Si una pieza de la plantilla ya no existe en el catálogo, se ignora sin ruido y el texto sigue nombrando el entorno completo.

Las plantillas anteriores al constructor no guardan ids de piezas: declaran `{"language": "java", "version": "21"}`. Para esas, la pieza de runtime se busca por el slug del catálogo que corresponde a ese lenguaje (`java-21`); sin coincidencia clara no se adivina nada y la solicitud abre sin runtime, con el texto ya puesto.

### 5. La IA ayuda, no es requisito

`POST /api/v1/teacher/ai/suggest-environment` reutiliza `SuggestTemplate` del copiloto con guarda de rol. Devuelve herramientas del catálogo y las que faltan. Si la IA no está configurada o falla, el formulario sigue funcionando a mano: el texto libre y el selector de piezas no dependen de ella.

### 6. El docente ve el estado de lo que pidió

Pestaña **Mis solicitudes** en `/teacher/templates`, con estados en palabras: *En revisión*, *Aprobada*, *Necesita un cambio*, *No es viable*, más la respuesta del administrador. Los estados técnicos (`EN_REVISION`, …) no salen de la API.

### 7. El administrador responde con una ficha, no con un muro de texto

Nueva página `/admin/solicitudes-entorno`: piezas reconocidas, piezas faltantes, uso declarado, materias y justificación. Aprobar exige enlazar la plantilla que resuelve el pedido; pedir ajuste o declarar no viable exige un motivo. Una aprobación sin plantilla no le sirve al docente.

### 8. Contrato

| Método | Ruta | Rol | Uso |
| --- | --- | --- | --- |
| `POST` | `/api/v1/teacher/environment-requests/preview` | docente/admin | Arma la receta, devuelve piezas, faltantes, huella, plantilla existente y solicitud gemela |
| `POST` | `/api/v1/teacher/environment-requests` | docente/admin | Crea la solicitud o suma la materia a una igual |
| `GET` | `/api/v1/teacher/environment-requests` | docente/admin | Solo las solicitudes propias |
| `GET` | `/api/v1/teacher/environment-requests/{id}` | docente/admin | Detalle propio |
| `GET` | `/api/v1/admin/environment-requests` | admin | Cola de revisión, con filtro por estado |
| `GET` | `/api/v1/admin/environment-requests/{id}` | admin | Ficha completa de la cola, acotada al tenant: `stack_config`, piezas, faltantes y materias |
| `PATCH` | `/api/v1/admin/environment-requests/{id}` | admin | `APROBADA` (exige `resolved_template_id`), `NECESITA_CAMBIO`, `NO_VIABLE` |
| `POST` | `/api/v1/teacher/ai/suggest-environment` | docente/admin | Interpreta la descripción y devuelve piezas + faltantes |

Todas pasan por `TenantMiddleware`; las escrituras, además, por auditoría. El endpoint legacy `POST /api/v1/templates` queda como está para no romper llamadas existentes, pero el docente ya no lo usa.

### 9. La aprobación comprueba que la plantilla sirva (2026-11-14)

Aprobar deja de ser un cambio de estado. El servicio valida que la plantilla exista, sea de la misma institución (o global), esté aprobada y cubra la modalidad pedida con `TemplateSupportsEnvironment`; si no, responde `422` y no enlaza nada. La cola del administrador filtra el selector con la misma regla (`environmentCovers`) y, cuando no hay plantillas compatibles, indica que hay que construir el entorno antes de responder.

Si la respuesta queda vacía, el servicio escribe `Tu solicitud quedó resuelta con el entorno «<nombre>».` para que el docente sepa qué plantilla la resolvió. `NO_VIABLE` también exige motivo, como ya pedía el punto 7. La solicitud devuelve `resolved_template_name` en la cola y en «Mis solicitudes», y el docente puede abrir el catálogo filtrado por ese nombre.

Notificar al docente por correo sigue fuera de este ADR: el estado y la respuesta se consultan en la pestaña de solicitudes.

### 10. De la cola al Constructor sin reescribir el pedido (2026-11-23)

La cola ya no obliga a elegir entre una plantilla existente o crear una a mano. El botón **Crear en Constructor** abre `/admin/plantillas/nueva?requestId=<id>` en la misma pestaña y el Constructor pregunta por la ficha con `GET /api/v1/admin/environment-requests/{id}`.

La precarga respeta la nomenclatura del catálogo: toma la modalidad de `target_environment`, la descripción pedagógica del docente y las piezas de `stack_config` (runtime, interfaz web, complementos y servicios), emparejando por id o slug. El título del pedido no se impone como nombre del entorno: el Constructor sigue sugiriendo el nombre a partir de las tecnologías y el administrador lo edita.

Sobre el Paso 01 aparece el banner de la solicitud con docente, materias y las piezas que faltan en el catálogo, cada una con su motivo. El checkbox **Al publicar, aprobar y vincular automáticamente esta solicitud** llega marcado; al crear la plantilla se envía el `PATCH` de aprobación con `resolved_template_id`. Si ese enlace falla, la plantilla creada se conserva y el aviso indica que la solicitud debe vincularse manualmente desde la cola. **Desvincular** descarta el contexto y deja el Constructor vacío.

El detalle sigue acotado al tenant de la sesión: un identificador de otra institución responde `404`, igual que uno inexistente.

---

## Consecuencias

- La solicitud conserva lo que el docente escribió y quién lo escribió. La cola del administrador deja de mostrar "Docente de la institución".
- Ya no se piden imágenes ni RAM al docente: la imagen se deriva de las piezas y la RAM del constructor. El administrador mantiene la última palabra sobre la cuota.
- Pedir práctica y evaluación en la misma solicitud es normal; antes era imposible porque la modalidad estaba fijada.
- Un mismo entorno pedido por varios docentes queda en una ficha con varias materias, en lugar de varias fichas iguales.
- La detección de duplicados de plantillas sigue pendiente en el constructor: la huella ya se calcula, pero la comparación entre plantillas terminadas no está aplicada en el flujo de creación. Las seis gemelas se consolidaron en una (`IDE Python + Postgres`) reasignando sus 5 ejercicios, 6 materias y 4 workspaces.
- La variante desde una plantilla existente llega con las piezas puestas, no solo con el texto. El contrato del catálogo crece con `stack_config`; las plantillas sin modelos ni piezas declaradas siguen abriendo la solicitud vacía, como antes.
- El formulario ya no depende del namespace `/admin/template-components` para leer el catálogo, aunque la ruta sigue siendo la misma; separarla es deuda declarada.

## Alternativas descartadas

| Alternativa | Motivo |
| --- | --- |
| Guardar la solicitud en `lab_templates` con `status='PENDIENTE_AUDITORIA'` | Mezcla intención con artefacto ejecutable y obliga a inventar una imagen antes de saber si el entorno es posible |
| Mantener el perfil de carga y mapearlo a RAM | El docente no puede estimar memoria y el administrador la sobrescribe igual |
| Dos solicitudes separadas para práctica y evaluación | Duplica el trabajo del docente y del administrador cuando la práctica usa ambas |
| Bloquear el envío cuando una pieza no existe | El catálogo incompleto es justamente el motivo de la solicitud |
| Aprobar sin enlazar plantilla | El docente ve "aprobada" y no tiene nada que usar |
| Aplicar la IA como paso obligatorio | Deja el formulario inútil sin proveedor configurado |
| Pedir dos textos (práctica y logro) | Duplica la escritura y el administrador lee lo mismo dos veces |

## Verificación

- `go build ./...` y `go test ./internal/core/services -run 'Environment|Signature'`: derivación de modalidad, normalización de usos, rechazo de descripción corta, aprobación sin plantilla, suma a una solicitud con la misma huella y estabilidad de la huella ante distinto orden de piezas.
- Prueba SQL contra la base local en transacción revertida: inserción, lectura con nombre del docente, búsqueda por huella y búsqueda de plantilla aprobada por imagen.
- `npx ng build --configuration development`, `npx eslint` sobre los archivos tocados, `stylelint` sobre los SCSS nuevos, y los gates `date-format` y `machine-data`.
- `ng test` del catálogo del docente: 11 casos, incluidos los nuevos de pestañas y variante.
- Prellenado verificado en el spec: la variante abre con el nombre de la plantilla en el texto y con su `stack_config` disponible para resolver las piezas contra el catálogo.

## Continuidad del borrador y de la consulta a la IA

El formulario es largo y la interpretación de la IA puede tardar hasta 90 s. Por eso el estado no pertenece al diálogo:

- **Borrador en la raíz.** `EnvironmentRequestDraftService` guarda texto, piezas, nombre, materia, usos y avisos de la IA. Cerrar el modal o navegar a otra vista no pierde nada; al volver, el formulario se restaura y la receta se vuelve a pedir al backend.
- **Consulta con dueño en la raíz.** La interpretación corre como operación de `AiOperations` (`teacher-suggest-environment`), el mismo registro que usa el estudio del administrador. Cerrar el modal ya no aborta la llamada: el resultado queda listo, se reclama una sola vez al reabrir y el aviso llega por el toast global.
- **Se limpia al enviar.** El borrador y el resultado de la IA se descartan cuando la solicitud sale, para que la próxima apertura empiece vacía. Cancelar o cerrar sin enviar conserva el trabajo.

La contrapartida es conocida: el borrador vive en memoria del navegador. Recargar la página o cerrar sesión lo descarta; si eso llega a importar, el paso siguiente es guardarlo en el servidor y que la consulta siga en segundo plano.

## Catálogo real en el prompt de la IA

La interpretación no adivina: el prompt incluye el catálogo instalado del tenant con sus slugs exactos. Dos correcciones sobre el comportamiento anterior:

- **Motores de base de datos desde la tabla.** Antes el prompt anunciaba nombres genéricos (`sqlite`, `postgres`, `mysql`), que no coincidían con los slugs reales (`sqlite-3`, `postgres-16`, `mysql-84`); la pieza nombrada se descartaba al resolverla. Ahora la lista sale del catálogo, igual que los lenguajes y los complementos, y las listas van ordenadas para que el mismo catálogo produzca el mismo prompt.
- **El entorno objetivo del docente.** La solicitud del docente no enviaba entorno objetivo, así que el servicio asumía juez efímero y las reglas de ese caso vaciaban la base de datos y la interfaz web aunque el docente las nombrara. Ahora ese flujo usa IDE persistente por defecto, y el prompt respeta lo que la persona pide explícitamente cuando la pieza existe en el catálogo.

Del lado del formulario, la resolución de slugs es tolerante: busca por slug, id, nombre, motor de servicio y prefijo, de modo que una variante de nombre no haga desaparecer una herramienta que sí existe.

---

### 11. Simplificación a una columna y corrección del flujo con piezas faltantes (2026-11-26)

El uso real del formulario mostró tres defectos encadenados cuando el docente pedía una tecnología ausente del catálogo (ej. un lenguaje no instalado) y fricción innecesaria al duplicar la vista en dos columnas:

1. **Modal de una sola columna con herramientas primero.** `template-request-modal` retira el panel lateral redundante («Lo que pedirás») y ordena el flujo en una sola columna: (1) selección manual de herramientas con un bloque compacto opcional de asistencia IA, (2) uso declarado (`IDE_PERSISTENTE` / `JUEZ_EFIMERO`), (3) nombre sugerido, materia de referencia y nota para administración, y (4) detalles técnicos plegados al final. Los avisos de plantilla existente, solicitud gemela o piezas faltantes se muestran encima del pie de envío junto con una acción discreta para limpiar el borrador en memoria.
2. **Nota obligatoria solo cuando faltan datos técnicos.** Si el docente elige al menos una herramienta del catálogo sin piezas faltantes, `description` es opcional tanto en Angular como en `EnvironmentRequestService.Create`. Si no elige herramientas del catálogo o incluye piezas en `missing_components`, la nota sigue siendo obligatoria (mínimo 10 caracteres) para que administración tenga contexto.
3. **Sin runtime inventado en el copiloto.** El prompt de `SuggestTemplate` indica dejar `runtime_slug` vacío cuando el lenguaje principal solicitado no está en el catálogo y reportarlo en `missing_components`; el parser acepta `runtime_slug` vacío siempre que lleguen piezas faltantes u otro componente, y el handler docente `SuggestEnvironmentForTeacher` fija `IDE_PERSISTENTE` por defecto cuando no recibe `target_environment`. Al aplicar la sugerencia en el frontend, un `runtime_slug` vacío limpia `runtimeId` previo y la IA deja de fijar `titleTouched = true`, de modo que cambiar de consulta actualiza el título sugerido sin arrastrar valores viejos.
4. **Sin falsos positivos ni colisiones con piezas faltantes.** `EnvironmentRequestPreviewDTO` incorpora `missing_components`: cuando hay piezas fuera de catálogo (o sin resolver), `Preview` y `Create` suprimen `existing_template` y el cálculo de `stack_signature` incluye los nombres normalizados y ordenados de `missing_components`, evitando que dos solicitudes con distintos lenguajes faltantes se fusionen como duplicadas.
