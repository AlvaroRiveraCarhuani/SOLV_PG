# ADR-040: Deprecación del Asistente Modal de Ejercicios y Migración Guiada al Editor Completo

## Estado
Aceptado

## Contexto
El módulo docente de la plataforma coexistía con dos flujos de autoría y edición de ejercicios:
1. El asistente modal tradicional en ventana emergente (`exercise-editor-modal`), con formularios acoplados y pasos secuenciales rígidos.
2. La nueva interfaz de página completa en 3 columnas (`exercise-editor`), equipada con gestión de estado reactiva basada en Signals, contratos declarativos de entrada (`input_format`), tabla relacional de casos de prueba con ponderación en vivo, verificación sintáctica inmediata y checklist de auditoría previa a la publicación.

Mantener ambas interfaces de forma indefinida genera dispersión en la experiencia de usuario y costo de mantenimiento. No obstante, una eliminación abrupta del modal antiguo interrumpiría flujos de trabajo docentes en curso.

## Decisión
Se establece una estrategia de deprecación planificada y migración progresiva asistida:

1. **Banner de Obsolescencia y Acción de Migración**:
   - Al desplegar el modal antiguo, se presenta un banner informativo visible que alerta sobre el estado de obsolescencia de dicha interfaz.
   - Se provee una acción directa ("Migrar al nuevo editor") que redirige hacia `/teacher/courses/:courseId/exercises/:id/edit` (o `/new`), preservando el contexto y cerrando el modal.
   - Se ofrece la opción de desestimar el banner para la sesión actual en caso de requerir continuidad inmediata en el modal.

2. **Telemetría y Registro de Uso**:
   - Se registra el evento de apertura (`exercise_editor_opened`) identificando la modalidad (`legacy_modal` vs `new_editor`), el identificador del ejercicio y del curso, a fin de cuantificar la tasa de adopción real.

3. **Criterios y Umbrales para el Retiro Definitivo**:
   - El componente modal antiguo será retirado del código fuente cuando se cumpla cualquiera de las siguientes condiciones:
     - El 90% de los ejercicios activos hayan sido editados o creados a través del nuevo editor de 3 columnas.
     - Transcurra un período de 6 meses a partir de la publicación de esta política de deprecación.

4. **Preservación de Operatividad**:
   - Durante el período de transición, el modal antiguo conserva su funcionamiento técnico habitual sin degradación de capacidades previas.

## Consecuencias
- Los docentes cuentan con un período de transición claro y guiado hacia la nueva interfaz sin bloqueos operativos.
- El equipo de ingeniería dispone de métricas de telemetría objetivas para programar la eliminación final del código heredado.
- Se mantiene la compatibilidad de datos hacia atrás mientras se consolida la adopción de las nuevas capacidades pedagógicas.
