# Spec: Navegación Docente y Cola Centralizada de Entregas

## Requirements

### REQ-1: Estructura del Menú Lateral (Sidebar)
El layout del docente SHALL mostrar tres secciones semánticas de navegación: `DOCENCIA`, `CONTENIDOS` y `SUPERVISIÓN`.
- La sección `DOCENCIA` MUST incluir los accesos a "Panel de Control" (`/teacher/dashboard`) y "Evaluaciones & Entregas" (`/teacher/evaluaciones`).
- La sección `CONTENIDOS` MUST incluir los accesos a "Banco de Ejercicios" (`/teacher/exercises`) y "Plantillas Docker" (`/teacher/templates`).
- La sección `SUPERVISIÓN` MUST incluir el acceso a "Anti-Plagio AST" (`/teacher/anti-plagio`).

### REQ-2: Cola Centralizada de Entregas
La vista `/teacher/evaluaciones` MUST mostrar un listado unificado de entregas de todos los cursos del docente autenticado.
- Cada elemento de la tabla MUST indicar: Estudiante, Materia/Curso, Laboratorio, Fecha/Hora de Entrega (con pipe `dateText`), Veredicto del Juez, Calificación y Estado de Revisión.
- Los veredictos MUST seguir la semántica oficial: AC (verde), WA (rojo), TLE (ámbar), RE (naranja), AST_BLOCKED (rojo intenso).

### REQ-3: Filtrado Reactivo Multicriterio
La vista MUST permitir filtrar entregas en tiempo real mediante Signals:
- Filtro por Curso (Todos o materia específica).
- Filtro por Veredicto (Todos, AC, WA, TLE, RE, AST_BLOCKED).
- Filtro por Estado (Todas, Pendientes de Revisión, Calificadas).

### REQ-4: Acceso Directo a SpeedGrader
Cada fila de entrega con revisión pendiente o calificada MUST tener un botón de acción "Revisar" que navegue a `/teacher/revision/:submissionId`.

---

## Scenarios

### Scenario 1: Visualización del Sidebar Completo
Given el docente ha iniciado sesión en SOLV
When ingresa a cualquier vista del módulo docente
Then el sidebar muestra 3 secciones (`DOCENCIA`, `CONTENIDOS`, `SUPERVISIÓN`)
And cada enlace contiene su icono correspondiente y estado activo según la ruta actual.

### Scenario 2: Filtrado de Entregas Pendientes en Cola Global
Given el docente accede a `/teacher/evaluaciones`
When selecciona el filtro de veredicto "WA" y estado "Pendientes de revisión"
Then la tabla muestra únicamente las entregas que coinciden con ambos criterios
And el contador de resultados actualiza su valor reactivamente.

### Scenario 3: Navegación hacia SpeedGrader desde la Cola
Given el docente visualiza una entrega pendiente en la tabla
When hace clic en el botón "Revisar"
Then el sistema navega a la ruta `/teacher/revision/:submissionId`
And abre la interfaz de SpeedGrader con el diff y el reproductor de código.
