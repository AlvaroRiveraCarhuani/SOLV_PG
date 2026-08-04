# ADR-020: Motor de Auditoría Semántica AST Inmutable con Montaje de Solo Lectura (D4)

* **Estado:** Aceptado
* **Fecha:** 2026-08-04

## Contexto y Problema
El análisis estático de código mediante herramientas como Semgrep requiere acceder directamente a los archivos creados por el estudiante en su volumen de trabajo. Permitir que un proceso o contenedor de análisis secundario tenga permisos de escritura sobre dicho volumen expone el código del estudiante a posibles corrupciones accidentales o modificaciones indebidas durante el proceso de calificación.

## Decisión Tomada
1. Adopción de la decisión de arquitectura **D4**: El worker de análisis estático `SemgrepWorker` debe instanciar contenedores efímeros montando el volumen del estudiante de forma estrictamente inmutable en **Solo Lectura (`/src:ro`)**.
2. El contenedor de Semgrep ejecuta el análisis AST, emite la salida JSON por `stdout` y se destruye inmediatamente.
3. El resultado del escaneo se inyecta de forma inalterable en la columna de base de datos `semgrep_audit JSONB` de la tabla `workspaces`.

## Consecuencias
* **Positivas:**
  * Garantía matemática de inmutabilidad del código fuente del estudiante durante las evaluaciones.
  * Persistencia estructurada en PostgreSQL lista para consulta por la interfaz docente.
  * Destrucción limpia de contenedores de escaneo sin consumo residual de recursos.
