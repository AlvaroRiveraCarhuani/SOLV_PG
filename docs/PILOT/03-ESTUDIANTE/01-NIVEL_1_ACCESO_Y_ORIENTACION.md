# Estudiante — Nivel 1: Acceso y orientación

> **Rutas:** `/student`, `/student/courses`, `/student/courses/:courseId`, `/student/aula`
> **Propósito:** comprobar que el estudiante entiende qué tiene pendiente, dónde están sus materias y cómo entrar a una clase activa.
> **Criterio del nivel:** EST-1.1, EST-1.2 y EST-1.3 en `APROBADO`.

---

## Resumen del Nivel 1

| ID | Subnivel | Caso | Ruta principal | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **EST-1.1** | Fácil | Primer ingreso y lectura del inicio | `/student` | El estudiante identifica sus materias y entregas sin datos ajenos | `PENDIENTE` |
| **EST-1.2** | Medio | Navegar materias y mapa curricular | `/student/courses` | La materia abre sus prácticas con su avance y mapa curricular | `PENDIENTE` |
| **EST-1.3** | Difícil | Entrar a un aula activa | `/student` | El aula aparece, la presencia se registra y la práctica se abre | `PENDIENTE` |

---

## EST-1.1 (Fácil): Primer ingreso y lectura del inicio

**Objetivo:** iniciar sesión por primera vez y comprobar que el inicio muestra solo la información del estudiante autenticado.

**Precondiciones:** cuenta `EST-PILOTO-01` matriculada en `DOC-PILOTO-01`; al menos una práctica publicada; otra cuenta `EST-PILOTO-02` con datos distintos.

**Datos de prueba:** materia `DOC-PILOTO-01`; práctica `DOC-PILOTO Suma de enteros`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Iniciar sesión como `EST-PILOTO-01` y abrir `/student`. | Se muestra el inicio del estudiante con su nombre o identificador de cuenta. |
| 2 | Revisar los bloques visibles. | Aparecen `Próximas entregas`, `Últimas entregas` y `Tus materias`; `Aula ahora` solo si hay una clase abierta. |
| 3 | Abrir `Tus materias`. | Solo figura `DOC-PILOTO-01`; no aparece la materia de `EST-PILOTO-02`. |
| 4 | Revisar `Próximas entregas`. | La práctica publicada figura con materia, título y fecha; el enlace apunta a `/student/exercises/:exerciseId`. |
| 5 | Revisar `Últimas entregas`. | Si no hay entregas, se muestra un estado vacío; no se inventa una nota ni una entrega. |
| 6 | Hacer clic en `Historial`. | Se abre `/student/history` con el historial del estudiante autenticado. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del inicio con materia y entrega próxima; captura del historial vacío o propio |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el inicio muestra materias o entregas de otra cuenta, o una práctica sin fecha aparece como entrega próxima.

---

## EST-1.2 (Medio): Navegar materias y mapa curricular

**Objetivo:** encontrar una materia, abrir su ficha y revisar las prácticas del curso.

**Precondiciones:** EST-1.1 aprobado; materia con al menos dos prácticas publicadas.

**Datos de prueba:** materia `DOC-PILOTO-01`; búsqueda por código `DOC-PILOTO`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/student/courses`. | Se muestra `Mis materias` con buscador y filtros. |
| 2 | Buscar `DOC-PILOTO`. | La lista se reduce a la materia coincidente; el contador es coherente con las filas. |
| 3 | Hacer clic en `Ver prácticas`. | Se abre `/student/courses/:courseId` con la ficha de la materia. |
| 4 | Revisar el mapa curricular y prácticas. | Figuran las prácticas publicadas con su estado y acceso; las no publicadas no aparecen. |
| 5 | Abrir una práctica desde la ficha. | Se navega a `/student/exercises/:exerciseId` con la práctica correcta. |
| 6 | Volver al listado y limpiar la búsqueda. | Se recuperan todas las materias del estudiante sin recargar la página. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del listado filtrado, de la ficha de materia y de la práctica abierta |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** la búsqueda muestra materias ajenas o la ficha incluye prácticas no publicadas.

---

## EST-1.3 (Difícil): Entrar a un aula activa

**Objetivo:** detectar un aula abierta, entrar a la práctica de la sesión y confirmar que el docente ve la presencia del estudiante.

**Precondiciones:** docente con aula abierta en `DOC-PILOTO-01`; práctica de la sesión publicada; EST-1.1 aprobado.

**Datos de prueba:** sesión de `1 h`; práctica `DOC-PILOTO Suma de enteros`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/student` con la sesión del docente ya abierta. | El bloque `Aula ahora` muestra la materia, la práctica y la acción `Abrir práctica` o `Entrar al aula`. |
| 2 | Hacer clic en `Abrir práctica`. | Se abre la práctica de la sesión, no una práctica distinta. |
| 3 | Volver al inicio y hacer clic en `Entrar al aula`. | Se abre `/student/aula` con la sesión activa y el estado del estudiante. |
| 4 | Mantener la pestaña visible entre 60 y 120 segundos. | El docente ve al estudiante como presente en `/teacher/aula`; no se exige una acción adicional. |
| 5 | Cambiar de pestaña o cerrar la sesión del estudiante. | La presencia deja de actualizarse; el docente lo ve como `Salieron` o ausente después del umbral. |
| 6 | Volver a la pestaña y esperar un ciclo de presencia. | El estudiante vuelve a figurar como presente sin recargar la clase. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de `Aula ahora`, de `/student/aula` y de la fila del estudiante en `/teacher/aula` |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el aula no aparece, la práctica abierta no corresponde a la sesión, o la presencia no se registra después de 120 segundos con la pestaña visible.
