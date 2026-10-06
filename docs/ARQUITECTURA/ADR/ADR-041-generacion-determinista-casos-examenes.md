# ADR-041: Generación determinista de casos de prueba por estudiante en exámenes

## Estado
Aprobado

## Contexto
En entornos de evaluación presencial o remota bajo modalidad de examen (`purpose = 'exam'`), existe el riesgo de que los estudiantes compartan o copien entradas y salidas de prueba entre sí. Para mitigar la copia sin perder reproducibilidad en reclamos académicos, es necesario que cada alumno reciba datos de prueba numéricos y estructurales distintos derivados determinísticamente de su identidad.

## Decisión
1. **Activación:** Se añade la opción `per_student_seed` (boolean) en la configuración del ejercicio, habilitada únicamente cuando el propósito es examen (`purpose = 'exam'`).
2. **Generación Determinista:** La semilla $S$ se calcula combinando `hash(exercise_id + student_id)`. El motor `FormatValidator.GenerateCase` genera entradas únicas para cada estudiante basadas en dicha semilla.
3. **Snapshot en Submission:** Se agrega la columna `generated_cases JSONB` a la tabla `submissions` para almacenar las entradas y salidas exactas evaluadas en esa entrega específica.
4. **Reproducibilidad en SpeedGrader:** Si el docente re-ejecuta el sandbox de una entrega o la audita, el sistema reutiliza el snapshot de `generated_cases` preservando el veredicto exacto.
5. **Censura en Feedback Estudiante:** Los datos generados permanecen ocultos para el estudiante (feedback censurado con veredicto global únicamente), pero visibles desenmascarados para el docente en SpeedGrader.

## Consecuencias
- **Positivas:** Cada estudiante recibe una variante única del examen manteniendo el mismo nivel de dificultad. Cero posibilidad de copia directa de datos de prueba entre alumnos.
- **Negativas:** La solución de referencia del docente debe ejecutarse para generar las salidas esperadas en tiempo de evaluación si el ejercicio no las provee estáticamente.
