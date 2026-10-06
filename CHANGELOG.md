# Changelog

Todos los cambios notables en el proyecto SOLV están documentados en este archivo.

## [2.0.0] - 2026-10-06 — Juez Virtual SOLV Enterprise

### Módulo Docente y Edición de Ejercicios
- **Extracción Relacional de Casos:** Normalización del esquema de datos mediante la tabla `exercise_test_cases` con visibilidad de 3 estados (`example`, `public`, `hidden`) y pesos de calificación por caso (ADR-039).
- **Validador Declarativo e InputFormat:** Validador de contratos estructurales (`input_format`) en Go y generador determinista por contrato.
- **Editor Docente de 3 Columnas:** Rediseño completo del editor reactivo en Angular con arquitectura de Signals y navegación por 3 pasos (Identidad, Contrato/Casos y Publicación).
- **Tabla de Casos en Vivo:** Edición interactiva de casos de prueba con validación inmediata contra el contrato y cálculo de ponderaciones.
- **FormatBuilder con Presets:** Creador visual de contratos con 5 presets (Entero, Vector, Matriz, Cadena, Grafo) y editor JSON bidireccional sincronizado.
- **Reglas AST Custom y Checklist:** Restricciones estáticas personalizables por lenguaje (Semgrep), plantillas base y panel de comprobación con bloqueantes previos a la publicación.
- **Deprecación de Wizard Modal:** Sustitución formal del modal antiguo por el flujo completo en pantalla completa (ADR-040).
- **Fuzzing Estructural v2:** Generación masiva automática de casos límite basados en la gramática del contrato `input_format`.
- **Vista Previa del Estudiante:** Modal de simulación a pantalla completa que reproduce exactamente la experiencia del alumno sin ejecutar el runner.

### Adaptatividad Curricular y Analítica
- **Panel de Métricas Docente:** Dashboard analítico en `/teacher/courses/:courseId/analytics` con distribución de dificultad, top 10 tags por efectividad, casos más fallados y líneas de tendencia temporal.
- **Sugerencias de Refuerzo al Estudiante:** Componente opcional en la vista de curso del alumno que sugiere ejercicios de práctica extracurricular en etiquetas con baja tasa de éxito.
- **Mapa Curricular y Módulos:** Modelo de módulos relacionales (`course_modules`), prerrequisitos de desbloqueo con validación topológica acíclica y fail-closed en evaluación, incluyendo exención pedagógica para exámenes.

### Exámenes Blindados y Herramientas Forenses
- **Semilla Determinista por Estudiante:** Generación de casos únicos por alumno en exámenes mediante `seed = hash(exercise_id + student_id)` y snapshot de auditoría en `submissions.generated_cases` (ADR-041).
- **Time-Travel Replay de Escritura:** Captura continua de keystrokes en búfer (`submission_keystroke_events`) y reproductor interactivo en SpeedGrader con detección y resaltado de pegados sospechosos (ADR-042).
- **Benchmark O(N) Forense:** Motor de regresión asintótica para estimar complejidad temporal y espacial empírica ($O(1)$, $O(N)$, $O(N \log N)$, $O(N^2)$) con comparador de discrepancias en SpeedGrader sin impacto en la calificación.
