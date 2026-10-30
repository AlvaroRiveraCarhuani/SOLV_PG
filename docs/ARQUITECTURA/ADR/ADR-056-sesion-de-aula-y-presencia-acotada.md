# ADR 056: Sesión de Aula como Envoltorio Opcional y Presencia Acotada

> **Estado:** Aprobado
> **Fecha:** 2026-10-30
> **Ámbito:** Docencia, Core, Workspaces
> **Relacionado:** ADR-055 (read model por pregunta), ADR-019 (modelo de dominio de workspaces), ADR-049 (telemetría persistida), ADR-011 (ciclo de vida y hibernación)

---

## Contexto

La clase universitaria es sincrónica: docente y alumnos coinciden en una hora. El sistema no tenía ninguna entidad que representara ese hecho. Un entorno se abría cuando el alumno entraba a su práctica, sin hora de inicio ni de fin, y el docente veía en su pantalla todos los entornos de todas sus materias, todo el tiempo.

Eso producía tres problemas:

1. **"Activo" no tenía definición.** La única señal era `last_heartbeat_at` menor a 2 minutos, aplicada a todos los entornos del docente sin acotar. Un entorno abierto a las 22:00 para estudiar contaba igual que el de un alumno en clase.
2. **No se podía responder quién falta.** Sin sesión no existía el conjunto contra el cual comparar: ni "20 de 28 dentro", ni "3 sin empezar", ni "5 sin entregar".
3. **La telemetría crecía sin límite.** Con 30 materias y varios entornos por materia, la lista era el inventario de contenedores del docente, no el estado de una clase.

Al mismo tiempo, una parte del trabajo ocurre fuera de clase y el diseño no puede ignorarlo.

---

## Decisión

### 1. La sesión es del docente y es opcional

Nueva entidad `class_sessions`:

| Campo | Tipo | Nota |
| --- | --- | --- |
| `id` | uuid | PK |
| `tenant_id` | uuid | Aislamiento multi-tenant |
| `subject_id` | uuid | Materia de la sesión |
| `exercise_id` | uuid, nullable | Práctica en curso, si la hay |
| `opens_at` | timestamptz | Inicio |
| `ends_at` | timestamptz | Fin previsto: al vencer, el aula se cierra sola |
| `closes_at` | timestamptz, nullable | Fin real; se completa al cerrar o al vencer |
| `status` | varchar(10) | `open` o `closed` |
| `created_by` | uuid | Docente que la abrió |

Índice parcial único sobre `(tenant_id, subject_id) WHERE status = 'open'`: una materia no puede tener dos aulas abiertas a la vez.

**Una práctica no requiere sesión.** La sesión es una envoltura, no un prerrequisito:

| Situación | Superficie |
| --- | --- |
| Hay aula abierta | Aula completa: presentes, faltantes, trabados, hora de cierre, modo proyección |
| Práctica abierta sin aula | Entornos abiertos ahora, sin lista de faltantes |
| Nadie conectado | Nada; no hay un cero ocupando lugar |

### 2. Presencia acotada a la sesión

`workspaces.session_id` nullable, con clave foránea a `class_sessions`.

- **Dentro de una sesión:** entorno con `session_id` de esa sesión y heartbeat menor a 2 minutos.
- **Fuera de sesión:** entorno con heartbeat menor a 2 minutos, acotado a las materias del docente.

Un entorno abierto antes de abrir el aula no se asigna a la sesión: la presencia mide el trabajo de la clase, no el historial del contenedor. Al cerrar el aula, la sesión pasa a `closed`, `closes_at` se completa y la pantalla deja de mostrar el estado en vivo de esa materia.

Mientras no exista la sesión, el panel usa el estado del contenedor (`running`, `active`, `pending`) como única señal de presencia, y lo llama "Ahora". La razón es concreta: el cliente del alumno todavía no emite heartbeat, así que `last_heartbeat_at` solo se escribe al crear o reactivar un entorno y marcaría cero con contenedores encendidos. La presencia por heartbeat entra con la sesión, y trae como dependencia que el cliente emita la señal cada minuto.

**Implementado el 2026-10-30.** El cliente del alumno consulta sus aulas abiertas cada minuto desde el layout y late solo con la pestaña visible; una respuesta 409 (aula cerrada) o 403 (dejó de estar matriculado) corta el latido de esa sesión. El tablero resuelve sesión y alumnos en dos consultas fijas y calcula el estado de cada fila en SQL, no en el cliente. La columna "Ahora" de la tabla de materias sigue contando contenedores y su encabezado lo dice; "Presentes" cuenta personas por latido dentro de la sesión. Pendiente y declarado: modo proyección, agrupación de excepciones por causa, código sin entregar con consentimiento, actualización por SSE y exportación de asistencia.

### 3. El trabajo fuera de clase conserva dueño

Un alumno que se traba a las 22:00 no tiene aula y, sin embargo, su problema es del docente. La regla no es "sin sesión no hay aviso", sino "sin sesión no hay interrupción".

- Dentro de la sesión: aparece en el aula como "se trabó", con acción inmediata.
- Fuera de la sesión: se agrega en la materia como resumen del día siguiente ("3 alumnos se trabaron desde tu última visita"), sin notificación ni tarea en la cola.

### 4. El aula tiene hora de fin y se cierra sola

`class_sessions.ends_at` se fija al abrir con la duración que elige el docente (por defecto 90 minutos, entre 15 y 480). `closes_at` sigue siendo el fin real: se completa al cerrar a mano o al vencer.

El vencimiento no depende de un proceso periódico: **cualquier lectura o apertura cierra primero las sesiones vencidas de la institución** (`UPDATE ... WHERE status = 'open' AND ends_at <= NOW()`), sella el `left_at` de sus participantes y sigue. Las superficies que importan —abrir aula, aula actual, tablero, listado del alumno y latido— pasan por ahí.

El motivo es concreto: el índice parcial único sobre `status = 'open'` convierte un aula olvidada en un bloqueo permanente de la materia. Con hora de fin y vencimiento en la misma transacción de lectura, el bloqueo tiene techo y no hay un planificador más que operar ni un retraso de hasta un minuto entre el vencimiento y el estado real.

### 5. El modo proyección pertenece al aula

Ocultar la barra lateral, subir la tipografía y mostrar el avance del grupo con nombres grandes es una función de la clase, no del panel. Vive en la pantalla de aula y depende de que haya una sesión abierta.

---

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Sesión obligatoria por práctica | Impide el trabajo fuera de clase, que es parte real del uso y obligaría a inventar sesiones para corregir |
| Presencia solo por heartbeat, sin sesión | Es el estado actual: no distingue clase de estudio y no permite calcular faltantes |
| Ventana móvil de 30 minutos sin entidad | No hay dueño de la ventana, no se puede cerrar, ni auditar quién la abrió |
| Guardar asistencia en una tabla de registros | Duplica lo que el entorno ya registra y agrega una pantalla de pasar lista |
| Asignar a la sesión los entornos abiertos antes de abrirla | Mezcla el historial del contenedor con el trabajo de la clase y falsea los faltantes |
| Cerrar el aula con un planificador en segundo plano | Agrega un proceso que operar y un desfase de hasta un intervalo; el vencimiento en la lectura deja el estado correcto en el instante en que alguien mira |
| Duración fija de la plataforma | La clase real no dura siempre lo mismo y el docente ya sabe cuándo termina |

---

## Consecuencias

**Positivas**

- "Activo" pasa a tener una sola definición, y la pantalla de aula puede decir quién falta.
- La telemetría deja de crecer con la cantidad de materias: se mira una clase por vez.
- La asistencia queda registrada como efecto del trabajo, exportable, sin un módulo aparte.
- El alumno fuera de clase no pierde dueño.

**Costos y cuidados**

- Migración con `class_sessions` y `workspaces.session_id`; los entornos existentes quedan con `session_id` nulo y no cuentan como presencia de aula.
- Abrir y cerrar aula son operaciones con permiso del docente de la materia; un docente ajeno recibe 403.
- Cerrar una sesión no detiene entornos. Si el docente quiere apagarlos, esa es otra acción, con su propia confirmación.
- El índice parcial depende de `status = 'open'`; el aula se cierra sola a su hora de fin, así que el bloqueo por olvido tiene techo. Si la clase se extiende, el docente la cierra y abre otra: todavía no hay prórroga de una sesión en curso.
- El vencimiento es perezoso: se resuelve en la siguiente interacción con el aula. Una sesión vencida sin tráfico sigue `open` en la base hasta que alguien abre, consulta o late, y eso no cambia lo que ve el docente.
- `workspaces` cambia de forma: cualquier consulta que lo recorra debe decidir si filtra por sesión o no.

---

## Verificación

- Abrir dos aulas en la misma materia devuelve conflicto, no una segunda fila.
- Un aula con `ends_at` vencido deja de aparecer como abierta y no bloquea abrir otra en la misma materia.
- Un latido contra un aula vencida devuelve 409, igual que contra un aula cerrada a mano.
- Un entorno abierto antes de la sesión no aparece como presente, y uno abierto durante la sesión sí.
- Al cerrar el aula, la pantalla deja el modo en vivo y el resumen de la materia conserva el registro.
- Un alumno que se traba sin sesión no genera aviso inmediato y sí aparece en el resumen del día siguiente.
- La cifra de presentes del panel y la del aula coinciden para la misma materia y el mismo instante.
