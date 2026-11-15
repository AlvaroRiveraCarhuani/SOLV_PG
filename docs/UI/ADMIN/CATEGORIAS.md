# Vista 8: Categorías Académicas

> **Especificación Oficial de Interfaz, Componentes y Comportamiento**
> **Rol:** Administrador de Institución (Tenant Admin)
> **Ruta:** `/admin/categorias`
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System

---

## 1. Propósito

Mantener la taxonomía que agrupa plantillas de entorno y modelos del catálogo. La categoría no cambia el contenido de una plantilla: cambia cómo se agrupa, se filtra y se presenta en la biblioteca y en el constructor.

---

## 2. Composición de la pantalla

### Encabezado
- Título «Categorías Académicas» y subtítulo de taxonomía.
- Resumen: categorías registradas y activas.
- `[Nueva Categoría]` abre el alta.

### Lista
La lista ocupa el cuerpo de la pantalla y es donde ocurre toda la gestión. Cada categoría muestra su nombre, su descripción y el uso que tiene en el catálogo.

| Control | Efecto |
| :--- | :--- |
| Arrastrar | Reordena la categoría en la lista. |
| Mover arriba / abajo | Alternativa accesible al arrastre. |
| Editar | Abre el formulario con nombre y descripción. |
| Eliminar | Abre la confirmación de borrado. |

El orden de la lista es el orden en que las categorías aparecen en la biblioteca de modelos y en los selectores del constructor.

---

## 3. Alta y edición

El formulario pide nombre y descripción. El nombre es el que ven docentes y estudiantes en los filtros; conviene que sea corto y reconocible. La categoría nueva aparece al final de la lista y desde ahí se ordena.

---

## 4. Borrado

Eliminar una categoría es irreversible. La pantalla bloquea el borrado mientras tenga plantillas o modelos asociados: primero se reasignan esos elementos y después la categoría queda disponible para eliminarse.

El modal de confirmación declara cuántos elementos dependen de la categoría, para que la decisión no se tome a ciegas.

---

## 5. Reglas de negocio

- Una plantilla o modelo pertenece a una sola categoría.
- El orden es una propiedad del catálogo, no de la vista: se guarda y se comparte.
- Una categoría en uso no se elimina; la pantalla ofrece la restricción antes del intento.
- Renombrar no altera las plantillas que la usan.

---

## 6. Contratos

| Método | Endpoint | Propósito |
| :--- | :--- | :--- |
| `GET` | `/api/v1/admin/template-categories` | Lista ordenada de categorías. |
| `POST` | `/api/v1/admin/template-categories` | Alta. |
| `PUT` | `/api/v1/admin/template-categories/{id}` | Edición de nombre y descripción. |
| `DELETE` | `/api/v1/admin/template-categories/{id}` | Baja, rechazada si tiene elementos asociados. |
| `PUT` | `/api/v1/admin/template-categories/reorder` | Guarda el orden de la lista. |
| `GET` | `/api/v1/admin/template-models` | Modelos que usan cada categoría, para el conteo de dependencias. |

---

## 7. Guías de la pantalla

La pantalla ofrece recorrido visual de cuatro pasos: resumen, alta, orden y restricción de borrado. No lleva tutorial porque el flujo es corto y la única restricción real se explica en el último paso y en la entrada contextual del centro de ayuda.

La sección equivalente del manual es «8.3 Categorías Académicas».
