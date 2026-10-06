# ADR-039: Extracción Relacional de Casos de Prueba, Visibilidad de 3 Estados y Metadatos Pedagógicos

## Estado
Aceptado

## Contexto
Históricamente, los casos de prueba de los ejercicios algorítmicos se persistían como un array JSON dentro de la columna `config JSONB` en la tabla `exercises`. Esto generaba limitaciones para la validación estricta de esquema, consultas eficientes, relaciones de integridad referencial y modelado de visibilidad pedagógica avanzada.

Adicionalmente, el modelo anterior utilizaba un booleano binario (`is_hidden`) que impedía distinguir didácticamente tres estados esenciales para la docencia:
1. `example`: Casos expuestos en el enunciado del problema antes de resolver y en el feedback posterior.
2. `public`: Casos de prueba públicos que no saturan el enunciado, pero cuyos datos se entregan al estudiante tras el envío para facilitar la depuración.
3. `hidden`: Casos de prueba ciegos de evaluación de nota donde solo se informa el veredicto sin exponer entradas ni salidas.

## Decisión
1. **Extracción a Tabla Relacional (`exercise_test_cases`)**:
   - Se crea la tabla relacional `exercise_test_cases` con clave foránea a `exercises(id) ON DELETE CASCADE` y restricción `UNIQUE (exercise_id, order_index)`.
   - Se eliminan los casos de prueba del blob JSONB `exercises.config` mediante las migraciones 00013, 00014 y 00015.
2. **Visibilidad de 3 Estados**:
   - Se establece la columna `visibility VARCHAR(10) NOT NULL DEFAULT 'public' CHECK (visibility IN ('example','public','hidden'))` como única fuente de verdad.
   - Los flags `is_sample` e `is_hidden` quedan deprecados y se derivan de forma dual únicamente para compatibilidad transitoria de la API con clientes antiguos.
3. **Ponderación de Casos (`weight`)**:
   - Cada caso posee un peso numérico `weight DOUBLE PRECISION NOT NULL DEFAULT 1.0 CHECK (weight >= 0)`.
   - El scoring ponderado se calcula como la suma de pesos de casos aprobados sobre el total de pesos.
4. **Metadatos Pedagógicos en `exercises`**:
   - Se incorporan las columnas `difficulty` (`easy`, `medium`, `hard`), `tags TEXT[]`, `purpose` (`class`, `exam`) y `per_student_seed BOOLEAN`.
   - Invariante fail-closed: `per_student_seed = true` requiere obligatoriamente `purpose = 'exam'`, de lo contrario se rechaza con error 422.

## Consecuencias
- Integridad referencial completa en PostgreSQL con CASCADE deletion.
- Aislamiento multi-tenant preservado en todas las operaciones sobre casos mediante joins con la tabla `exercises`.
- Mayor claridad didáctica y trazabilidad en los entornos de examen y evaluación formativa.
