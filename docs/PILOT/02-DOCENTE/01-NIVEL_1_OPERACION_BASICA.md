# Docente — Nivel 1: Operación básica

> **Rutas:** `/teacher/dashboard`, `/teacher/aula`, `/teacher/cursos/:courseId/ejercicios/nuevo`
> **Propósito:** comprobar que el docente puede abrir una clase, publicar una práctica de evaluación y publicar una práctica de desarrollo con rúbrica, sin ayuda del equipo técnico.
> **Criterio del nivel:** DOC-1.1, DOC-1.2 y DOC-1.3 en `APROBADO`.

---

## Resumen del Nivel 1

| ID | Subnivel | Caso | Ruta principal | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **DOC-1.1** | Fácil | Abrir un aula y verificar presencia | `/teacher/aula` | Sesión abierta, tablero con estudiantes y cierre disponible | `PENDIENTE` |
| **DOC-1.2** | Medio | Publicar una práctica de evaluación | `/teacher/cursos/:courseId/ejercicios/nuevo` | Práctica publicada con casos públicos y privados | `PENDIENTE` |
| **DOC-1.3** | Difícil | Publicar una práctica de desarrollo con rúbrica | `/teacher/cursos/:courseId/ejercicios/nuevo` | Entorno homologado y rúbrica disponibles para el estudiante | `PENDIENTE` |

---

## DOC-1.1 (Fácil): Abrir un aula y verificar presencia

**Objetivo:** abrir una sesión de clase, comprobar que el estudiante aparece como presente y cerrar la sesión sin dejar estados inconsistentes.

**Precondiciones:** docente y estudiante autenticados; ambos matriculados en `DOC-PILOTO-01`; período activo.

**Datos de prueba:** materia `DOC-PILOTO-01`; duración `1 h`; estudiante `EST-PILOTO-01`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Iniciar sesión como docente y abrir `/teacher/dashboard`. | Se muestra `Hoy`, el período activo y la acción `Abrir aula`. |
| 2 | Hacer clic en `Abrir aula`. | Se abre el formulario de apertura con materia, práctica publicada opcional y duración. |
| 3 | Seleccionar `DOC-PILOTO-01`, dejar la práctica vacía y elegir `1 h`. Confirmar. | La sesión queda abierta y la cabecera muestra materia, hora de fin y acción `Cerrar aula`. |
| 4 | Ir a `/teacher/aula`. | El tablero muestra la sesión, el marcador de presentes y los filtros `Todos`, `Trabados`, `Sin empezar`, `Salieron`, `Entregaron` y `Trabajando`. |
| 5 | En otra ventana, iniciar sesión como `EST-PILOTO-01` y abrir `/student`. | El inicio del estudiante muestra `Aula ahora` con la materia y la acción `Entrar al aula` o `Abrir práctica`. |
| 6 | Esperar entre 60 y 120 segundos sin cerrar la pestaña del estudiante. Actualizar `/teacher/aula`. | El contador de presentes se incrementa y la fila del estudiante deja de figurar como `Sin empezar`. |
| 7 | Hacer clic en `Cerrar aula` y confirmar. | La sesión desaparece del tablero y el estudiante deja de ver `Aula ahora` en el siguiente refresco. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de `/teacher/aula` con presencia y captura de `/student` con `Aula ahora` |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el aula no aparece en el estudiante después de 120 segundos, el contador de presentes no cambia, o `Cerrar aula` deja la sesión visible.

---

## DOC-1.2 (Medio): Publicar una práctica de evaluación

**Objetivo:** recorrer el editor de tres pasos y publicar una práctica de evaluación con casos públicos y privados, solución de referencia y límite de tiempo.

**Precondiciones:** DOC-1.1 aprobado o materia con matrícula confirmada; plantilla de evaluación disponible; docente con permiso de publicación.

**Datos de prueba:** título `DOC-PILOTO Suma de enteros`; lenguaje `Python`; tiempo máximo `2000 ms`; un caso público y uno privado.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/teacher/cursos/:courseId/ejercicios/nuevo`. | Se muestra el editor con `Configuración`, `Paso 1 de 3` y los pasos `Identidad y Pedagogía`, `Formato y Casos` y `Reglas y Publicación`. |
| 2 | En `Identidad y Pedagogía`, escribir el título, la consigna y seleccionar la modalidad `Entorno de Evaluación`. | La modalidad queda marcada y el paso 2 cambia a `Formato y Casos`. |
| 3 | Seleccionar el propósito `Clase / Formativo` y guardar el borrador. | El borrador se conserva; el editor no pierde los datos al recargar. |
| 4 | En `Formato y Casos`, definir el formato de entrada y salida y el límite de `2000 ms`. | El resumen muestra la modalidad de evaluación y el límite configurado. |
| 5 | Agregar un caso público con entrada y salida esperada, y un caso privado con sus datos. | La tabla `Casos de Prueba` muestra dos filas y distingue la visibilidad de cada una. |
| 6 | Registrar una solución de referencia en el lenguaje seleccionado. | El editor acepta la solución y el checklist de publicación deja de marcar ese requisito. |
| 7 | En `Reglas y Publicación`, revisar el checklist y hacer clic en `Publicar práctica`. | La práctica queda publicada y aparece en la materia con estado publicado. |
| 8 | Abrir la práctica como `EST-PILOTO-01`. | El estudiante ve el caso público y la consigna; el caso privado no se muestra. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del checklist completo, identificador de la práctica y vista del estudiante con un solo caso visible |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** se puede publicar sin casos o sin solución de referencia, el caso privado se filtra al estudiante, o el límite de tiempo no se conserva.

---

## DOC-1.3 (Difícil): Publicar una práctica de desarrollo con rúbrica

**Objetivo:** publicar una práctica de `Entorno de Desarrollo` usando una plantilla homologada y una rúbrica de dos criterios, y comprobar que el estudiante puede abrir el entorno y que el docente ve la rúbrica en la revisión.

**Precondiciones:** plantilla aprobada en el catálogo; DOC-1.2 aprobado; estudiante matriculado.

**Datos de prueba:** título `DOC-PILOTO API de tareas`; plantilla `Python Web + PostgreSQL`; criterios `Funcionalidad 50%` y `Calidad 50%`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/teacher/cursos/:courseId/ejercicios/nuevo`. | Editor en `Paso 1 de 3`. |
| 2 | Escribir el título y la consigna; seleccionar la modalidad `Entorno de Desarrollo`. | El paso 2 cambia a `Entorno y Entrega`. |
| 3 | En `Entorno y Entrega`, elegir el `Entorno homologado` aprobado y definir la fecha de entrega. | El resumen muestra la plantilla y la fecha; no se permite publicar sin plantilla. |
| 4 | Crear dos criterios de rúbrica con peso `50%` cada uno y al menos dos niveles por criterio. | La suma de pesos es `100%` y el editor no muestra error de validación. |
| 5 | En `Reglas y Publicación`, verificar el checklist y publicar. | La práctica queda publicada con la plantilla y la rúbrica asociadas. |
| 6 | Como `EST-PILOTO-01`, abrir la práctica y hacer clic en `Abrir entorno`. | El entorno se abre en una pestaña nueva y muestra el proyecto inicial de la plantilla. |
| 7 | Como docente, abrir la entrega desde `/teacher/por-calificar` o desde la práctica. | El SpeedGrader muestra el bloque `Evaluación por Rúbrica` con los dos criterios y sus niveles. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de la rúbrica configurada, del entorno abierto por el estudiante y del panel de rúbrica en el SpeedGrader |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** el editor permite publicar sin plantilla, la rúbrica no suma 100%, el entorno no abre, o el SpeedGrader no muestra los criterios configurados.
