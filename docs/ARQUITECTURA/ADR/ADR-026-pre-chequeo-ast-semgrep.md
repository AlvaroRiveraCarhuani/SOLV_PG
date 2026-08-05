# ADR-026: Pre-chequeo AST con Semgrep previo a la Aprovisionamiento del Contenedor

* **Estado:** Aceptado
* **Fecha:** 2026-08-04
* **Slice:** 9

## Contexto y Problema
El Juez Virtual evalúa código fuente de estudiantes en entornos interactivos o mediante el pipeline algorítmico. Cierto código malicioso (como importaciones peligrosas de librerías del sistema o manipulación de red) se bloquea en la fase de análisis del Árbol Sintáctico Abstracto (AST). 

Tradicionalmente, la verificación del AST se realizaba dentro del propio contenedor o posterior al inicio del sandbox, lo que consumía innecesariamente recursos del host (CPU, RAM, creación de red bridge) para código que de todos modos iba a ser rechazado.

## Decisión Tomada
1. Implementar el análisis estático de código del estudiante usando **Semgrep CLI local** en el servidor host **antes** de instanciar cualquier sandbox o contenedor Docker.
2. `SemgrepWorker.ScanCode()` crea un archivo temporal efímero con el código recibido y ejecuta `semgrep --config rules/{language}/forbidden.yaml --json`.
3. Si el pre-chequeo del AST determina que el código viola las reglas de seguridad configuradas (ej. llamadas bloqueadas al sistema, imports prohibidos), se retorna directamente el veredicto `AST_BLOCKED` al cliente, se serializan los hallazgos en la columna `ast_result` (JSONB) de PostgreSQL y se cancela la creación del contenedor.
4. El aprovisionamiento del contenedor solo ocurre si tanto la validación por expresiones regulares como el pre-chequeo semántico de Semgrep resultan exitosos.

## Consecuencias
* **Positivas:**
  * Ejecución ultra-rápida en el host evitando la latencia y sobrecosto de instanciar un contenedor Docker dedicado para Semgrep.
  * Optimización de recursos del servidor Asus: se evita el overhead de levantar contenedores efímeros y discos temporales para código malicioso.
  * Mitigación de vectores de ataque por denegación de servicio (DoS) por inyección masiva de tareas inválidas al juez.
* **Negativas:**
  * Requiere instalar `semgrep` CLI (`pip3 install semgrep`) como dependencia previa en el servidor host.
  * Requiere mantener los catálogos de reglas YAML actualizados en `backend/internal/infrastructure/semgrep/rules/`.
