# Vista: Historial de Entregas y Detalle de Entrega del Estudiante

> **Especificación de interfaz y contratos**  
> **Rol:** Estudiante  
> **Rutas:**  
> - Lista de historial: `/student/history`  
> - Detalle dedicado de entrega: `/student/history/:submissionId` (alias `/student/submissions/:submissionId`)  
> **Gobernanza:** SOLV Design System · ADR-005 (juez virtual dual y modalidades de evaluación)

---

## 1. Propósito y Modelo Mental

El módulo de historial permite al estudiante:
1. Revisar cronológicamente todos los envíos que ha realizado a lo largo de sus materias matriculadas.
2. Filtrar entregas por materia, resultado y **tipo de entorno** (`Todos los entornos`, `Entorno de Desarrollo`, `Entorno de Evaluación`).
3. Acceder mediante una acción directa (`[ Ver entrega ]`) a una **vista de detalle dedicada a pantalla completa**, eliminando modales estrechos que impiden la lectura cómoda de proyectos complejos o código con múltiples archivos.
4. Visualizar su código con resaltado de sintaxis profesional utilizando el componente oficial **CodeMirror** (`CodeEditorComponent` en modo solo lectura) tanto para proyectos multi-archivo como para soluciones algorítmicas de archivo único.
5. Consultar la retroalimentación del docente, comentarios inline línea por línea, puntuación ponderada y desglose de criterios de evaluación de rúbrica en modo solo lectura.

### 1.1. Diferenciación de Modalidad (`environment_type`) y Terminología Oficial

En SOLV se aplican estrictamente los términos de dominio:

1. **Entorno de Evaluación (`JUEZ_EFIMERO`):** Ejercicios algorítmicos o de código en un solo archivo.
   - **Tiempo de ejecución y memoria:** Muestra la duración en milisegundos (`XX ms`) y memoria consumida (`XX MB`) medida por el runner del juez en el contenedor efímero.
   - **Visualización de código:** Visor CodeMirror (`CodeEditorComponent`) con numeración de líneas, resaltado según el lenguaje configurado y sección de comentarios docentes.
   - **Panel lateral:** Puntuación obtenida, tiempo, memoria y badge semántico del veredicto.

2. **Entorno de Desarrollo (`IDE_PERSISTENTE`):** Prácticas y proyectos de desarrollo basados en plantillas de múltiples archivos (OpenVSCode Server).
   - **Tiempo de ejecución:** No aplica tiempo en milisegundos del runner algorítmico; la tabla renderiza un guion neutro (`—`).
   - **Etiqueta distintiva:** Muestra el badge `Entorno de Desarrollo` junto al título del ejercicio.
   - **Visualización a 3 paneles (inspirado en SpeedGrader pero en modo solo lectura para estudiante):**
     - **Panel izquierdo (Árbol de archivos):** Jerarquía de carpetas y archivos (`review-file-tree`) con contador total y selector interactivo de archivo activo.
     - **Panel central (Visor CodeMirror):** Contenido del archivo activo en `CodeEditorComponent` con detección automática de lenguaje por extensión (`.ts`, `.py`, `.go`, `.json`, etc.) y globos de comentarios puntuales dejados por el docente (`SubmissionComment`).
     - **Panel derecho (Evaluación y Rúbrica):** Tarjeta con nota final o estado "Sin calificar", comentarios generales de retroalimentación docente y desglose completo de la rúbrica con pesos porcentuales (`weight %`) y niveles asignados sin controles de edición.

---

## 2. Estructura de las Vistas

### 2.1. Tabla de Historial (`/student/history`)

Columna de acciones unificada en un único enlace semántico y filtros triples:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Historial de entregas                                                                            │
│ Revisa tus entregas anteriores, su resultado y el código que enviaste.                           │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [Materia: Todas v]   [Resultado: Todos v]   [Entorno: Todos v]   [Q Buscar por título...       ] │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ EJERCICIO              MATERIA      RESULTADO      NOTA    TIEMPO  FECHA    ACCIONES             │
│ Examen Parcial [PROY]  SIS-420 Web  Cumple         100 pts   —     29/10... [Ver entrega]        │
│ Práctica 3: Merge Sort SIS-211 Alg  Cumple         100 pts  152 ms 28/10... [Ver entrega]        │
│ Práctica 2: Listas     SIS-211 Alg  Sin calificar    —        —    26/10... [Ver entrega]        │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2. Vista Dedicada de Detalle (`/student/history/:submissionId`)

Para proyectos en entorno de desarrollo (`IDE_PERSISTENTE`):

```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ <- Volver al Historial  |  Práctica 1: API REST ...  [Entorno de Desarrollo]           │
│ Materia: SIS-420 Desarrollo Web Fullstack   |   Entregado: 21/10/2026 19:47            │
├───────────────────┬───────────────────────────────────┬────────────────────────────────┤
│ ARCHIVOS (3)      │ src/services/product.service.ts   │ RÚBRICA Y EVALUACIÓN           │
│                   │ Líneas: 42                        │                                │
│ > package.json    ├───────────────────────────────────┤ Calificación: 95 / 100 pts     │
│ v src             │ [CodeMirror Viewer: Readonly]     │ Veredicto: [ Cumple ]          │
│   > models        │                                   │                                │
│     product.ts    │ --------------------------------- │ Comentarios del Docente:       │
│   > services      │ Comentarios del docente en arch:  │ "Excelente arquitectura de     │
│   * product.serv. │ [Docente]: Buen tipado en L10     │ servicios y tipado."           │
│                   │                                   │ Rúbrica:                       │
│                   │                                   │ - Diseño y Tipado (40%)        │
│                   │                                   │   [Excelente - 100 pts]        │
│                   │                                   │ - Calidad de Código (60%)      │
│                   │                                   │   [Completo - 100 pts]         │
└───────────────────┴───────────────────────────────────┴────────────────────────────────┘
```

---

## 3. Bloqueo de Exámenes en el Resolutor (`student-exercise-solver`)

Para ejercicios de tipo examen (`purpose === 'exam'`):
- **Entorno de Evaluación (`JUEZ_EFIMERO`):** Si el examen fue entregado, se inhabilitan y ocultan los botones "Reiniciar" y "Enviar a Evaluación", bloqueando el editor de código y mostrando el banner de examen entregado con enlace a la entrega.
- **Entorno de Desarrollo (`IDE_PERSISTENTE`):**
  - Si el examen fue entregado, el botón **"Abrir entorno"** queda inhabilitado o bloqueado con el estado "Examen entregado" (evitando modificaciones sobre el volumen congelado).
  - El botón **"Entregar Proyecto"** queda inhabilitado con el estado "Examen entregado".
  - Se muestra el banner informativo: *"Has entregado este examen. Tu entorno de desarrollo ha quedado congelado en solo lectura para la evaluación por rúbrica."*
  - Se provee el botón de acceso directo `[ Ver entrega enviada ]` hacia `/student/history/:submissionId`.

---

## 4. Contratos de Datos

### 4.1. Endpoint Backend: `GET /api/v1/student/submissions`

Devuelve la lista histórica con `language` y `environment_type` canónicos:

```json
[
  {
    "id": "sub-uuid",
    "exercise_id": "ex-uuid",
    "exercise_title": "API REST con autenticación",
    "subject_id": "subj-uuid",
    "subject_code": "SIS-420",
    "subject_name": "Desarrollo Web Fullstack",
    "environment_type": "IDE_PERSISTENTE",
    "language": "typescript",
    "verdict": "AC",
    "score": 100,
    "execution_time_ms": 0,
    "memory_used_mb": 0,
    "manual_override": false,
    "override_reason": null,
    "submitted_at": "2026-10-29T16:47:08Z",
    "files_available": true
  }
]
```

### 4.2. Endpoints Complementarios de Detalle

1. `GET /api/v1/student/submissions/{id}`: Devuelve el objeto de la entrega con su código o array `files` de proyecto.
2. `GET /api/v1/student/submissions/{id}/comments`: Devuelve la lista de comentarios puntuales y generales dejados por el docente (`SubmissionComment[]`).
3. `GET /api/v1/exercises/{exerciseId}/rubric`: Devuelve la definición de la rúbrica del ejercicio (`RubricDTO`) para contrastar con las notas recibidas.
