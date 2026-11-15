# Vista 9: Catálogo de Tecnologías y Componentes

> **Especificación Oficial de Interfaz, Componentes y Comportamiento**
> **Rol:** Administrador de Institución (Tenant Admin)
> **Ruta:** `/admin/tecnologias`
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System

---

## 1. Propósito

Administrar las piezas con las que se construyen los entornos: lenguajes y runtimes, interfaces web, complementos y bases de datos. Lo que se registra y se activa aquí es lo que el Constructor de plantillas puede ofrecer.

---

## 2. Composición de la pantalla

### Encabezado
- Título «Catálogo de Tecnologías» y subtítulo.
- Resumen: tecnologías registradas y activas.
- `[Registrar Nueva Tecnología]` abre el formulario de alta.

### Pestañas
| Pestaña | Contenido |
| :--- | :--- |
| Tecnologías y Componentes | Gestión completa del catálogo. |
| Mapa del Sistema | Vista de solo lectura que relaciona lenguajes, capas y complementos compatibles. |

En la pestaña de gestión, un alternador cambia entre tarjetas y tabla sin perder los filtros.

### Filtros
- **Búsqueda:** nombre, slug, herramientas y paquete.
- **Perfil:** todos, lenguaje principal, interfaz web, complemento o base de datos. Cada opción declara cuántas tecnologías agrupa.
- **Estado:** todos, solo activos o solo inactivos.
- **Vacíos:** la pantalla distingue un catálogo vacío de una búsqueda sin coincidencias y ofrece la acción correspondiente.

---

## 3. Qué declara cada tarjeta

| Elemento | Significado |
| :--- | :--- |
| Perfil | Lenguaje principal, interfaz web, complemento o base de datos. |
| Memoria | Huella estimada con su detalle; se usa al calcular la capacidad del aula. |
| Ecosistema | Lenguaje al que pertenece un complemento. |
| Huérfano | El complemento no es compatible con ningún lenguaje activo; conviene revisarlo o desactivarlo. |
| En uso | Cuántas plantillas o modelos referencian la tecnología. |
| Imagen y paquetes | Imagen base, gestor de instalación y paquetes declarados. |
| Complementos compatibles | Cuántos complementos activos admite un lenguaje. |
| Origen | Catálogo oficial o creado por el administrador. |

### Acciones
| Acción | Efecto |
| :--- | :--- |
| Interruptor | Activa o desactiva la tecnología en el constructor sin borrar historial. |
| Complemento | En un lenguaje, abre el alta de un complemento con ese runtime preseleccionado. |
| Editar | Abre el formulario con los datos actuales. |
| Eliminar | Solo aparece en tecnologías que no son base del sistema. |

---

## 4. Reglas de baja

El diálogo decide la salida según el uso real:

| Situación | Lo que ofrece la pantalla |
| :--- | :--- |
| Uso mayor que cero y tecnología activa | No permite eliminar; propone **desactivarla** para retirarla del constructor sin alterar los entornos que la usan. |
| Uso mayor que cero y tecnología inactiva | Informa que no se puede eliminar mientras la referencien y no ofrece una acción destructiva. |
| Uso igual a cero | Permite eliminar con confirmación; la acción no se puede deshacer. |
| Tecnología base del sistema | El botón de eliminar no existe. |

---

## 5. Reglas de negocio

- Una tecnología inactiva deja de ofrecerse en el constructor, pero las plantillas que ya la usan conservan su configuración.
- Un complemento sin lenguaje activo compatible se marca como huérfano.
- El mapa del sistema no edita nada: refleja las relaciones vigentes.
- La memoria declarada por una tecnología participa del mínimo requerido en el Constructor.

---

## 6. Contratos

| Método | Endpoint | Propósito |
| :--- | :--- | :--- |
| `GET` | `/api/v1/admin/template-components` | Catálogo, con opción de incluir inactivos. |
| `POST` | `/api/v1/admin/template-components` | Alta de tecnología. |
| `PUT` | `/api/v1/admin/template-components/{id}` | Edición y cambio de estado. |
| `DELETE` | `/api/v1/admin/template-components/{id}` | Baja, rechazada si está en uso o es base del sistema. |

---

## 7. Guías de la pantalla

El recorrido señala resumen, pestañas, filtros, tarjetas y acciones. El tutorial cubre el caso que no se adivina: retirar una tecnología **en uso**. Exige una búsqueda y explica el contador de uso, el origen oficial y el interruptor; no abre diálogos ni cambia el estado de nada. El botón de eliminar solo aparece en tecnologías creadas por el administrador, así que la guía no lo usa como paso obligatorio.

La sección equivalente del manual es «8.2 Catálogo de Tecnologías».
