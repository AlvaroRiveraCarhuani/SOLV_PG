# Vista 6: Cursos y Períodos Académicos

> **Especificación Oficial de Interfaz, Componentes y Comportamiento**
> **Rol:** Administrador de Institución (Tenant Admin)
> **Ruta:** `/admin/cursos`
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System

---

## 1. Propósito

Supervisar las materias de la institución, asignar su docente titular y controlar el estado del curso dentro de un período académico. La pantalla no crea períodos: esos se gestionan en Configuración. Aquí se eligen y se usan como filtro.

---

## 2. Composición de la pantalla

### Encabezado
- Título «Cursos y Periodos Académicos» y subtítulo de supervisión.
- Resumen del período visible: cursos totales, activos, sin docente y archivados. El resumen responde al selector de período.
- Selector de período: «Todos los periodos» o un semestre concreto, con la marca del período vigente.
- `[Nuevo Curso]` abre el alta.

### Pestañas de estado
Cuatro pestañas con contador: Todos, Activos, Sin Docente y Archivados. «Sin Docente» reúne materias activas sin titular y es la bandeja que necesita atención.

### Tabla
| Columna | Contenido |
| :--- | :--- |
| Código y materia | Código oficial como dato técnico y nombre completo. |
| Docente titular | Nombre del titular o la etiqueta «Sin asignar». |
| Plantilla base | Entorno asociado a la materia, o el valor por defecto del sistema. |
| Matriculados | Estudiantes activos e hibernados. |
| Estado | Activo o Archivado. |
| Acción | Reasignar, editar y archivar o desarchivar. |

El encabezado de cada columna ordenable alterna ascendente, descendente y sin orden. La paginación declara el rango visible y el total.

### Vacíos y errores
- Sin materias en el período: estado vacío que explica que no hay coincidencias con los filtros o el período.
- Error de carga: el servicio conserva lo ya cargado y la pantalla informa; `[Actualizar]` no existe en esta vista, el reintento es recargar la pantalla.

---

## 3. Alta de materia

El modal pide nombre, código oficial y sección opcional. El identificador final se compone con la sección: `SIS-311` con sección `1` se muestra como `SIS-311-1` mientras se escribe.

- **Nombre y código:** obligatorios. El código se normaliza a mayúsculas.
- **Período académico:** opcional. Sin período, la materia existe pero no aparece al filtrar por semestre.
- **Docente titular:** opcional. Sin docente, la materia entra en «Sin Docente».
- **Color de identificación:** opcional; se aplica al curso después de crearlo.

El servidor rechaza un código duplicado y el mensaje se muestra dentro del modal.

---

## 4. Acciones por fila

| Acción | Efecto | Confirmación |
| :--- | :--- | :--- |
| Reasignar | Cambia el docente titular de la materia; pide el nuevo titular y un motivo. | No |
| Editar | Corrige nombre y código. | No |
| Archivar | Deja la materia en modo archivado de solo lectura. | Sí |
| Desarchivar | Devuelve la materia al catálogo activo. | Sí |

Archivar no borra contenido ni entregas: los estudiantes conservan su historial y la materia deja de abrir sesiones nuevas.

---

## 5. Reglas de negocio

- Un curso pertenece a un solo período académico.
- El código oficial identifica la materia en todo el sistema y no se duplica.
- El período vigente es el que usan las demás pantallas para inscribir materias.
- Archivar es reversible; eliminar no se ofrece desde esta vista.

---

## 6. Contratos

| Método | Endpoint | Propósito |
| :--- | :--- | :--- |
| `GET` | `/api/v1/subjects` | Materias de la institución con docente, plantilla y matrícula. |
| `POST` | `/api/v1/subjects` | Alta de materia. |
| `PUT` | `/api/v1/subjects/{id}` | Edición de nombre y código. |
| `PUT` | `/api/v1/subjects/{id}/archive` | Archivar o desarchivar. |
| `POST` | `/api/v1/admin/courses/{courseId}/reassign` | Reasignar docente titular con motivo. |
| `GET` | `/api/v1/admin/academic-periods` | Períodos disponibles para el selector. |
| `GET` | `/api/v1/admin/templates` | Plantillas asociadas a la materia. |
| `GET` | `/api/v1/teachers` | Docentes elegibles para el alta y la reasignación. |

---

## 7. Guías de la pantalla

El centro de ayuda ofrece recorrido visual y tutorial guiado. El recorrido señala resumen, período, pestañas, tabla y acciones. El tutorial acompaña el alta hasta el botón de confirmación y termina sin crear la materia, para poder usarlo en una capacitación sin dejar registros; tampoco ejecuta el archivado.

La sección equivalente del manual es «4.2 Cursos y Asignación de Docentes».
