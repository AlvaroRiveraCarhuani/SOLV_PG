# Vista 7: Directorio Institucional de Estudiantes

> **Especificación Oficial de Interfaz, Componentes y Comportamiento**
> **Rol:** Administrador de Institución (Tenant Admin)
> **Ruta:** `/admin/estudiantes`
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System

---

## 1. Propósito

Consultar la nómina de estudiantes, matricularlos en materias, revisar su estado académico y resolver los bloqueos por memoria. Es la pantalla donde se explica y se revierte una penalización de recursos.

---

## 2. Composición de la pantalla

### Encabezado
- Título «Directorio Institucional de Estudiantes».
- Acciones: registrar estudiante (individual o por planilla) y matricular desde una planilla.
- El resumen del directorio declara el estado del conjunto y los filtros que lo afectan.

### Filtros
- **Período académico:** acota la nómina al semestre elegido o muestra todos.
- **Materia:** reduce a los inscritos en una materia.
- **Estado:** cursando, con entornos activos, en riesgo y bloqueados por memoria.
- **Búsqueda:** nombre o correo institucional.
- **Orden y paginación:** las columnas ordenables alternan ascendente, descendente y sin orden; la paginación declara rango y total.

### Tabla y acciones
Cada fila reúne identidad y correo, materias inscritas, estado académico, faltas de memoria, entornos y el menú de acciones. Las acciones disponibles son:

| Acción | Efecto | Confirmación |
| :--- | :--- | :--- |
| Copiar correo | Copia el correo institucional al portapapeles. | No |
| Ver materias | Abre el detalle de materias inscritas del estudiante. | No |
| Desbloquear memoria | Reinicia el contador de faltas y registra el motivo en auditoría. | Sí, con motivo |
| Activar o suspender | Cambia el acceso del estudiante a la plataforma. | Sí |
| Reiniciar entorno | Reinicia el contenedor activo del estudiante. | No |
| Pausar entorno | Suspende el contenedor conservando su estado. | No |

---

## 3. El modelo de faltas por memoria

Cada vez que un entorno se detiene por exceder su cuota de memoria se registra una falta para el estudiante.

| Faltas | Estado | Consecuencia |
| :--- | :--- | :--- |
| 1 o 2 | En riesgo | Sigue trabajando; la pantalla lo marca para seguimiento. |
| 3 o más | Bloqueado por memoria | No puede abrir laboratorios hasta que un administrador lo desbloquee. |

El desbloqueo reinicia el contador del estudiante, exige un motivo y queda en el Registro de Auditoría. No borra las entregas ni el historial académico.

---

## 4. Altas y matrícula

- **Registro individual:** correo institucional y datos de identidad.
- **Registro por planilla:** importa una lista de correos y devuelve cuántos se crearon.
- **Matrícula desde planilla:** inscribe los correos de una materia concreta y permite reconciliar los que ya existían.

Las tres operaciones informan el resultado real: creados, omitidos y errores por correo.

---

## 5. Reglas de negocio

- El correo institucional identifica al estudiante dentro del tenant.
- Un estudiante bloqueado no puede abrir entornos hasta el desbloqueo.
- Suspender la cuenta corta el acceso, pero conserva matrículas, entregas e historial.
- Reiniciar o pausar un entorno no altera las faltas; solo el desbloqueo reinicia el contador.

---

## 6. Contratos

| Método | Endpoint | Propósito |
| :--- | :--- | :--- |
| `GET` | `/api/v1/admin/students` | Nómina con filtros `search`, `subject_id`, `status` y `period_id`. |
| `POST` | `/api/v1/admin/students` | Alta individual. |
| `POST` | `/api/v1/admin/students/batch` | Alta por lote de correos. |
| `PUT` | `/api/v1/admin/students/{id}/status` | Activar o suspender con motivo. |
| `POST` | `/api/v1/admin/students/{id}/reset-oom` | Desbloquear memoria con motivo. |
| `GET` | `/api/v1/admin/students/{id}/courses` | Materias inscritas del estudiante. |
| `POST` | `/api/v1/admin/subjects/{subjectId}/roster` | Matricular una planilla en una materia. |
| `POST` | `/api/v1/environments/{id}/restart` | Reiniciar el entorno activo. |
| `POST` | `/api/v1/environments/{id}/pause` | Pausar el entorno activo. |
| `GET` | `/api/v1/admin/academic-periods` | Períodos del filtro. |

---

## 7. Guías de la pantalla

La entrada contextual del centro de ayuda explica el modelo de faltas. El recorrido y el tutorial del desbloqueo están planificados en la ficha `odd/tasks/admin-help-guides-coverage.md` (T7); el tutorial deberá exigir motivo y confirmación sin ejecutar el desbloqueo.

La sección equivalente del manual es «4.3 Directorio de Estudiantes y Desbloqueo de Memoria».
