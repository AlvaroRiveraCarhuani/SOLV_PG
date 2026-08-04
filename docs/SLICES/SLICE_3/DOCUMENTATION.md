### ADR: Arquitectura Híbrida del Juez Virtual (SOLV)

**Contexto y Problema** El sistema SOLV requiere un motor de evaluación capaz de procesar tanto lógica algorítmica tradicional (donde importa la salida estándar) como transacciones de bases de datos (donde importa el estado persistente). Diseñar flujos o tablas independientes para cada modalidad generaría deuda técnica y esquemas rígidos difíciles de mantener.

**Decisión Arquitectónica Principal** Se implementa una **Arquitectura Híbrida Polimórfica** orquestada desde un único backend en Go. El núcleo del modelo de datos en PostgreSQL delega la complejidad a una columna de configuración dinámica, permitiendo que el motor de Docker adapte su estrategia de evaluación según la naturaleza del ejercicio.

### 1. Modelo de Datos Polimórfico (PostgreSQL)

Para evitar un diseño de base de datos inflado con columnas nulas, la tabla `exercises` se mantiene delgada y centralizada.

- **Identificador de Tipo:** Columna `type` que define el flujo de evaluación (`algorithm` o `database`).
    
- **Almacenamiento Dinámico (JSONB):** Columna `config` que muta su estructura según el `type`.
    
- **Configuración Algorithm:** Almacena `test_cases` (input/output/hidden), reglas AST y límites de tiempo/memoria.
    
- **Configuración Database:** Almacena el motor específico, `init_script`, `reference_solution`, `validation_query` y el `expected_json` serializado.
    

### 2. Pipeline de Evaluación Algorítmica (I/O Estándar + AST)

Para los lenguajes de programación (C, C++, C#, Java, JavaScript, Python), el flujo prioriza la seguridad y la prevención de fraudes académicos.

- **Fase A (Análisis AST Estático):** Antes de instanciar cualquier contenedor, el backend procesa el Árbol Sintáctico Abstracto del código. Bloquea importaciones peligrosas (ej. `os`, `sys`, `fs`) y funciones prohibidas (previniendo _hardcoding_).
    
- **Fase B (Ejecución Aislada):** Se utiliza el SDK de Docker para levantar contenedores efímeros con red deshabilitada (`network: none`), montaje de archivos de solo lectura y cuotas estrictas de RAM/CPU.
    
- **Evaluación:** El código se inyecta por `stdin` y el Juez compara el `stdout` contra los casos de prueba esperados.
    

### 3. Pipeline de Evaluación de Bases de Datos (Auditoría de Estado)

Para los motores de persistencia (PostgreSQL, MySQL, MongoDB), la salida estándar de la consulta del alumno es irrelevante. Lo que se audita es cómo el script mutó el estado de la información.

- **Entorno Efímero:** Se despliega un contenedor del motor requerido (SQL o NoSQL).
    
- **Paso 1 (Semilla):** Se inyecta el `init_script` para crear tablas/colecciones y poblar datos iniciales.
    
- **Paso 2 (Mutación):** Se ejecuta el script del estudiante (ej. un bloque de `UPDATE`, `DELETE` o agregaciones).
    
- **Paso 3 (Extracción):** Se inyecta el `validation_query` diseñado por el docente para extraer el estado final formateado.
    
- **Paso 4 (Comparación):** El Juez serializa el resultado a JSON y lo compara estrictamente contra el `expected_json`.
    

### 4. Experiencia de Usuario Docente: El "Dry Run" Automático

Para mantener una estética "Modern SaaS" y evitar fricción operativa, el docente nunca debe escribir el `expected_json` a mano al configurar un ejercicio de base de datos.

- **Generación en Segundo Plano:** Al guardar el ejercicio en la interfaz, el backend en Go ejecuta un "Dry Run".
    
- **Flujo Dry Run:** Levanta un contenedor, inyecta el `init_script`, ejecuta la `reference_solution` del propio docente y lanza el `validation_query`.
    
- **Autoguardado:** El sistema captura el estado resultante, lo formatea a JSON y lo persiste automáticamente en la columna `config` de PostgreSQL. Esto garantiza que la solución del profesor funciona antes de habilitar el reto a los alumnos.
    

### 5. Infraestructura de Pruebas de Integración (Test-Driven)

Para garantizar la estabilidad del orquestador, se establece un estándar de pruebas riguroso en el directorio `tests/integration/testdata/`.

- **Pruebas Parametrizadas (Table-driven):** Los tests en Go iteran sobre una matriz completa de tecnologías.
    
- **Cobertura Multilenguaje:** Inclusión de binarios y scripts para C, C++, C#, Java, JS y Python, probando veredictos de `AC` (Accepted), `WA` (Wrong Answer), `TLE` (Time Limit Exceeded) y `AST_VIOLATION`.
    
- **Cobertura Multimotor:** Casos de prueba específicos que validan el flujo de "Dry Run" y evaluación para PostgreSQL, MySQL y MongoDB.