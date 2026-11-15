# ADR-080: Propiedad de los Datos — Capas Propias, Espejadas y Derivadas

## Estado
Aprobado

## Contexto

SOLV trata hoy todos sus datos como propios. Funciona mientras la única fuente es su panel, pero una universidad ya tiene un sistema académico que sabe quién está matriculado, en qué materia y en qué periodo. Si SOLV se presenta como dueño de esos datos, compite con ese sistema, y compite mal: nadie va a mantener dos padrones de alumnos.

El mismo problema aparece en el vocabulario. La libreta exporta una columna llamada `Promedio Final` y un archivo llamado `calificaciones_*.csv`. Un docente que ve eso razonablemente cree que está cerrando el acta de la materia. No lo está: está viendo el promedio de las prácticas de laboratorio que él mismo corrigió, sin los exámenes, sin la asistencia y sin la escala de la facultad. Cuando ese número no coincide con la nota oficial, la discusión no es sobre el cálculo, es sobre quién tiene razón.

ADR-079 ya agregó `external_id`, `source_system` y `last_synced_at` para sostener la identidad de las personas espejadas. Faltaba declarar qué significa eso para el resto del sistema.

## Decisión

**Tres capas con dueño explícito.**

| Capa | Datos | Quién manda | Qué hace SOLV |
| :--- | :--- | :--- | :--- |
| **Propios** | Entornos de práctica, entregas, veredictos del juez, comentarios, sesiones de aula, auditoría, plantillas, configuración | SOLV | Todo: crear, editar, borrar |
| **Espejados** | Personas (nombre, correo, identificador), matrícula, materias y secciones, periodos académicos | El sistema académico de la institución | Leer, mostrar, conciliar |
| **Derivados** | Nota de laboratorio por práctica, promedio del curso, presencia, analítica | SOLV los calcula, nadie los firma | Recalcular siempre |

**Reglas que se derivan de la tabla**

1. **Lo espejado no se edita a mano cuando hay origen.** Si el nombre de un alumno está mal, se corrige en el sistema de la universidad. Una edición local se pierde en la siguiente sincronización, y peor: se pierde sin avisar.
2. **Sin origen, el origen es `manual`.** Una institución que no tiene sistema académico carga los datos desde el panel; esa carga es un origen legítimo y sus filas se pueden editar. La capa no depende de que exista un sistema externo, depende de que el dato tenga dueño declarado.
3. **La sincronización va en un solo sentido.** Del sistema académico hacia SOLV. SOLV no escribe matrícula ni notas en el sistema de la institución. Empujar datos convertiría a SOLV en responsable de un padrón que no controla.
4. **Los atributos locales de una entidad espejada siguen siendo propios.** `users.status` (suspensión administrativa por abuso) y `subjects.is_archived` (archivar un curso que ya terminó) son decisiones de la institución sobre el uso de la plataforma; no viajan al origen y una sincronización no las pisa.
5. **La reconciliación de bajas es la consecuencia operativa de la capa espejada.** Si la matrícula la manda el origen, un alumno que ya no figura en el origen no debe seguir matriculado. Es lo que hace `reconcile` en la importación de nómina (ADR-079), acotado por `source_system`.
6. **Un derivado no se exporta con nombre de documento oficial.** La libreta produce una **nota de laboratorio**, no un promedio final.

## Terminología

| Antes | Ahora | Dónde |
| :--- | :--- | :--- |
| `Promedio Final` | `Nota de Laboratorio` | Encabezado del CSV, columna de la libreta, etiqueta del exportador |
| `calificaciones_<materia>_<fecha>.csv` | `notas_laboratorio_<materia>_<fecha>.csv` | Nombre del archivo exportado en CSV y Excel |
| «Exportar Planilla de Calificaciones» | «Exportar Notas de Laboratorio» | Título del modal de exportación |
| Hoja de Excel `Calificaciones` | Hoja `Notas de Laboratorio` | Nombre de la hoja del libro exportado |

Los nombres internos (`average`, `course_average`, `GetCourseGradesMatrix`) **no cambian**: son contrato de API y de base de datos, y renombrarlos no aporta nada visible. Lo que cambia es lo que lee una persona.

## Consecuencias

**A favor**

- SOLV se integra en el ecosistema de la universidad en lugar de competir con él. Es una posición defendible frente a un comité académico: la plataforma aporta laboratorios, no un padrón paralelo.
- La sincronización de ADR-079 deja de ser una función suelta y pasa a tener un lugar en el modelo.
- El docente entiende qué corrige y qué no. La nota de laboratorio es su insumo; la nota de la materia la pone la facultad.
- El renombre no cuesta migración: son etiquetas de presentación y un nombre de archivo.

**En contra**

- Si hay origen, corregir un nombre mal escrito requiere acceso al sistema de la universidad. Es una fricción real y hay que decirla en el alta.
- `source_system` pasa a ser un dato operativo: si alguien carga una planilla con el origen equivocado, la reconciliación posterior borra matrículas que no correspondían. El panel lo pide explícito y la reconciliación está apagada por defecto.
- El archivo exportado deja de llamarse como algunos docentes esperan. Un nombre de archivo distinto es más barato que una discusión sobre actas.

## Lo que este ADR no decide

- No define integraciones concretas: qué sistema académico, con qué protocolo ni con qué frecuencia. Eso es ADR aparte, por institución.
- No cambia el cálculo de la nota de laboratorio ni su escala (ADR-066).
- No convierte la nota de laboratorio en calificación oficial en ningún caso. Si la institución decide usarla, la carga en su sistema.

## Referencias

- ADR-079 — identificador externo y reconciliación de la nómina
- ADR-066 — veredicto del juez y autoría docente de la nota
- ADR-024 — esquema académico multitenant
- ADR-022 — sincronización unidireccional con Google Classroom
