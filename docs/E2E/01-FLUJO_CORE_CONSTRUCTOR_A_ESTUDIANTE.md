# Flujo Core 01: Del Constructor de Plantillas (Admin) a la Evaluación del Estudiante

> **Archivo de Prueba:** `frontend/e2e/flows/01-core-lifecycle.spec.mjs`
> **Modo de Ejecución:** `serial` con corte inmediato (`maxFailures: 1`)
> **Actores:** Administrador (`adminPage`), Docente (`teacherPage`), Estudiante (`studentPage`)

---

## 1. Objetivo del Flujo

Validar de punta a punta contra el stack real (Angular + Go + PostgreSQL + Docker) el circuito principal de la plataforma:
1. El **Administrador** verifica el Catálogo de Tecnologías y construye una nueva **Plantilla Oficial Dual (`AMBOS`: Laboratorio + Evaluador)** con **Python 3.12**, ejecutando la comprobación real en Docker (`Comprobar funcionamiento`) y publicándola en el catálogo activo.
2. El **Administrador** da de alta un **Curso Institucional** en el periodo académico activo, asignando al **Docente** titular y matriculando al **Estudiante**.
3. El **Docente** ingresa a su curso, abre el **Editor de Ejercicios**, configura un problema algorítmico en Python con su solución oficial de referencia y casos de prueba (ejemplo y oculto), verifica el checklist sin bloqueantes y **Publica el Ejercicio**.
4. El **Estudiante** visualiza el curso en su panel, abre el **Mapa Curricular**, verifica que el ejercicio publicado aparece disponible para resolver (`Resolver`), envía su solución al Juez Virtual (`POST /api/v1/evaluations` + `POST /api/v1/submissions`) obteniendo veredicto **`AC`** (Aceptado), y el **Docente** audita la entrega en el **SpeedGrader** (`/teacher/revision/:submissionId`).

---

## 2. Etapas y Criterios de Éxito Paso a Paso

### Etapa 1 — Administrador: Catálogo y Constructor de Entornos (`/admin/modelos-categorias` y `/admin/plantillas/nueva`)
1. Navega a `/admin/modelos-categorias` y verifica que la pestaña **Catálogo de Tecnologías** lista el componente base `Python 3.12` (`python-312`) activo.
2. Navega a `/admin/plantillas/nueva` (**Registrar Nueva Plantilla Oficial**):
   - En **1. Propósito y Datos del Entorno**, selecciona la tarjeta **`Laboratorio + Evaluador`** (`AMBOS`).
   - En **2. Arquitectura y Tecnologías del Entorno**, selecciona **`Python 3.12`** dentro de la Columna 1 (`Lenguaje Principal`).
   - Completa el campo **`Nombre del Entorno *`** con un nombre único identificable para la corrida E2E y selecciona una **Categoría Académica**.
   - Verifica en el panel derecho (**Resumen del Entorno**) que la imagen contenedora (`python:3.12-slim-bookworm`), el comando de ejecución (`python3 solution.py`) y los programas incluidos (`python3`, `pip`) se sintetizan automáticamente.
3. Hace clic en **`Comprobar funcionamiento`**, espera a que el trabajo asíncrono `POST /api/v1/jobs/env-test` termine en estado `success` y muestre **`¡Entorno Verificado!`**.
4. Hace clic en **`Publicar Plantilla Oficial`** (`POST /api/v1/admin/templates`) y comprueba que redirige a `/admin/plantillas?tab=catalog` mostrando la plantilla recién creada en estado **Activa**.

### Etapa 2 — Administrador: Alta de Curso y Matrícula (`/admin/cursos`)
1. Navega a `/admin/cursos` y abre el modal **`Nuevo Curso`**.
2. Completa **Nombre de la Materia**, **Código Oficial**, selecciona el **Periodo Académico** vigente, asigna al **Docente Titular** de prueba y selecciona la **Plantilla Docker Base** recién publicada en la Etapa 1.
3. Confirma con **`Crear Curso`** (`POST /api/v1/subjects`) y verifica que el curso aparece en la tabla institucional como **Activo**.
4. Registra la matrícula del **Estudiante** de prueba en el curso creado (`POST /api/v1/subjects/{id}/enroll`) y verifica su inclusión en la nómina del curso.

### Etapa 3 — Docente: Creación y Publicación del Ejercicio (`/teacher/cursos/:id`)
1. Con la sesión aislada del **Docente** (`teacherPage`), navega a `/teacher/cursos/:courseId` y presiona **`Nuevo Laboratorio`** (`/teacher/courses/:courseId/exercises/new`).
2. **Paso 1 (Identidad y Pedagogía)**:
   - Selecciona modalidad **`Juez Virtual (I/O)`** y propósito **`Clase / Formativo`**.
   - Ingresa el **Título del ejercicio**, selecciona dificultad **`Fácil`**, elige el **Lenguaje de ejecución** (`Python`) vinculado a la plantilla homologada, agrega una etiqueta temática y redacta el **Enunciado del Ejercicio**.
   - Avanza con **`Siguiente`**.
3. **Paso 2 (Contrato, Solución de Referencia y Casos de Prueba)**:
   - Ingresa la **Solución Oficial de Referencia** en Python (`a, b = map(int, input().split()); print(a + b)`).
   - Agrega dos casos de prueba con **`Agregar Caso`**:
     - Caso #1 (Visibilidad `Ejemplo`, entrada `2 3`, salida esperada `5`).
     - Caso #2 (Visibilidad `Oculto`, entrada `10 25`, salida esperada `35`).
   - Avanza con **`Siguiente`**.
4. **Paso 3 (Checklist y Publicación)**:
   - Verifica en el **Checklist de Publicación** que existen `0` Problemas Bloqueantes (`Sin Bloqueantes`).
   - Si existen advertencias pedagógicas opcionales, marca la casilla de confirmación y presiona **`Publicar`** $\rightarrow$ **`Publicar ahora`**.
   - Verifica que el estado cambia a **`Publicado`** (`POST /api/v1/exercises/{id}/publish` con HTTP 200).

### Etapa 4 — Estudiante y Docente: Resolución, Evaluación en Juez y Revisión en SpeedGrader
1. Con la sesión aislada del **Estudiante** (`studentPage`), navega a `/student`, verifica que la materia figura en **Mis Laboratorios Activos** y entra al detalle del curso (`/student/courses/:courseId`).
2. En el **Mapa Curricular**, verifica que el ejercicio recién publicado aparece bajo **Ejercicios Generales** con el botón **`Resolver`** habilitado.
3. El estudiante ejecuta su entrega de código Python contra el ejercicio publicado:
   - Se invoca la evaluación real en contenedor efímero (`POST /api/v1/evaluations`) y se verifica que el motor devuelve `verdict: "AC"` pasando el 100% de los casos de prueba.
   - Se registra la entrega oficial (`POST /api/v1/submissions`) asociada al estudiante y al ejercicio.
4. Con la sesión del **Docente** (`teacherPage`), vuelve a `/teacher/cursos/:courseId`, abre la pestaña **`Cola de Entregas`**, verifica que aparece la entrega del estudiante con veredicto **`AC`** y hace clic en **`SpeedGrader`** (`/teacher/revision/:submissionId`).
5. En el **SpeedGrader**, comprueba que se visualiza el nombre del estudiante, el código fuente entregado línea por línea y la tarjeta de **Veredicto Oficial** en **`AC`** con `100 / 100 pts`.
