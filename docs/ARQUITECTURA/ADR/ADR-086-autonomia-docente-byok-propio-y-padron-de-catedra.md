# ADR-086: Autonomía Operativa del Docente — Proveedores BYOK Propios y Padrón de Cátedra

## Estado
Aceptado

## Contexto

Dos fricciones aparecieron al usar la plataforma con docentes reales.

La primera es la credencial de IA. El docente podía asociar su clave personal solo a los cuatro proveedores base y a los que el administrador había registrado. Si tenía créditos propios en Groq, Mistral, Together o Cerebras —todos compatibles con la API de OpenAI— no podía usarlos. La respuesta de que «OpenRouter cubre esos modelos» no aplica: una clave nativa de Mistral no es interoperable con una cuenta de OpenRouter. Además, la tarjeta de un proveedor institucional como Inception mostraba «Sin configurar» aunque el docente ya estuviera operando con ese motor y sin costo, lo que daba a entender que la IA institucional estaba incompleta.

La segunda es el padrón. En la vista de cursos del administrador, la columna de matriculados era un número inerte: no había forma de ver quiénes eran, dar de alta a un alumno rezagado ni dar de baja a quien abandonó. En la vista del docente, el curso no tenía pestaña de estudiantes; la libreta ocultaba la nómina entera cuando la materia todavía no tenía prácticas. La única vía de alta era un buscador individual que exigía que el administrador hubiera cargado antes al alumno, y la carga por planilla estaba reservada al administrador. El resultado operativo era una parálisis en las primeras semanas del semestre, justo cuando la matrícula todavía se está ajustando.

ADR-079 y ADR-080 ya habían resuelto la procedencia del dato: `enrollments.source_system` distingue la nómina oficial de una carga manual. Lo que faltaba era habilitar el trabajo de aula sin romper esa procedencia.

## Decisión

**1. El docente puede registrar sus propios proveedores compatibles con OpenAI, con una política de red distinta a la del administrador.**

El proveedor personal se guarda dentro de `ai_settings.user_overrides[userID].custom_providers`, nunca en el catálogo del tenant. El identificador sigue el formato `custom_<slug>` y no puede colisionar con uno institucional: el catálogo de la institución no se edita desde la credencial personal.

La validación de destino es la diferencia central:

| Criterio | Administrador institucional | Docente (BYOK personal) |
| :--- | :--- | :--- |
| Esquema | `http://` y `https://` | Solo `https://` |
| Destino | LAN privada, servidores on-premise, dominios públicos | Solo direcciones públicas de internet |
| Bloqueos | Loopback, metadata, nombres internos de contenedores | Loopback, RFC 1918, ULA IPv6, link-local/metadata, multicast y CGNAT |
| Alcance del dato | `tenants.config.ai_settings` (toda la institución) | `user_overrides[userID]` (solo ese docente) |

La regla se aplica al guardar, al listar modelos y al probar la conexión: no alcanza con validar el alta, porque un `base_url` enviado luego en la prueba de conexión abriría el mismo camino. El docente configura una cuenta comercial, no la red del servidor.

**2. El catálogo BYOK distingue el origen y señala el motor institucional activo.**

`GET /api/v1/ai/my-settings` devuelve cada proveedor con `origin` (`institutional` o `personal`) y el identificador `institutional_provider` del motor vigente. El panel marca esa tarjeta como **motor activo de la institución** y explica que la clave personal solo hace falta para superar la cuota compartida. El botón destructivo «Quitar mi clave» se muestra únicamente cuando la tarjeta visualizada es la que tiene la clave guardada.

**3. El padrón es una superficie compartida entre administración y docencia.**

`GET/POST/DELETE /api/v1/courses/{id}/students` y `POST /api/v1/courses/{id}/roster` sirven a ambos roles. El backend deriva el permiso del rol (el administrador opera sobre cualquier materia del tenant; el docente solo sobre las que dicta) y deriva el origen de la matrícula del rol que la ejecuta: `institutional_roster` para el administrador, `teacher_manual` para el docente.

La reconciliación sigue siendo una operación del administrador. En la sesión del docente el backend fuerza `source_system = 'teacher_manual'` y `reconcile = false`: su planilla suma altas de cátedra y no puede borrar matrículas de la nómina oficial. Cuando la institución sincroniza la nómina definitiva, la reconciliación acotada por origen conserva esas altas y las reporta como excepciones.

**4. La interfaz muestra a las personas del curso antes que las prácticas.**

El detalle del curso gana una pestaña **Estudiantes** con búsqueda, origen de matrícula, estado del entorno y baja. El contador del encabezado lleva a esa pestaña. La libreta, cuando hay alumnos pero todavía no hay prácticas, muestra la tabla de estudiantes con una columna colapsada que explica que aún no hay prácticas publicadas: la ausencia de prácticas no puede invisibilizar la nómina.

## Consecuencias

- El docente puede empezar a dictar sin esperar la sincronización central y sin inventar archivos CSV de una línea; las altas quedan trazadas con su origen y son auditables.
- Un docente con credenciales propias no puede usar el servidor como sondeo de la red interna, aunque su credencial esté activa.
- El catálogo institucional conserva su autoridad: no se puede sobrescribir ni eliminar desde el panel del docente.
- La vista de cursos deja de mostrar la plantilla base. Un curso es un contenedor académico semestral; las plantillas de infraestructura se eligen en el ejercicio o el laboratorio.
- El alta de materia advierte cuando el período elegido ya cerró y no preselecciona un período vencido: una materia creada ahí nace archivada por ADR-029.

## Alternativas descartadas

- **Permitir cualquier URL al docente y confiar en la buena fe.** Las peticiones de inferencia salen del servidor universitario; una URL libre convierte la credencial personal en un SSRF contra la LAN o el metadata del proveedor cloud.
- **Un endpoint distinto por rol para el padrón.** Habría duplicado permisos, validaciones y origen de matrícula en dos caminos que hacen lo mismo. El rol ya está en la sesión y el servicio puede decidir con él.
- **Dar reconciliación al docente.** Una planilla de cátedra no es la fuente oficial; reconciliar con ella borraría matrículas legítimas por una lista incompleta.

## Referencias

- [ADR-029 — Períodos Académicos y Archivado de Cursos](ADR-029-periodos-academicos-archivado-cursos.md)
- [ADR-079 — Identificador Externo y Reconciliación de Nómina](ADR-079-identificador-externo-y-reconciliacion-de-nomina.md)
- [ADR-080 — Propiedad de los Datos: Capas Propias, Espejadas y Derivadas](ADR-080-propiedad-de-datos-propias-espejadas-derivadas.md)
- [Contratos de API — API Key Personal (BYOK) y Padrón de una Materia](../../API_CONTRACTS.md)
