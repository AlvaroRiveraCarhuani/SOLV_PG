# ADR-016: Motor de Auditoría Semántica AST con SemgrepWorker

* **Estado:** Aceptado (Complementado por ADR-020)
* **Fecha:** 2026-07-25

## Contexto y Problema
La evaluación tradicional basada en comparación de entradas y salidas I/O (`stdin` / `stdout`) resultaba insuficiente para calificar la calidad del código, el cumplimiento de buenas prácticas o el uso de patrones de diseño específicos. Se requería un mecanismo automatizado para analizar el Árbol de Sintaxis Abstracta (AST) de las entregas de los estudiantes.

## Decisión Tomada
1. Construir el servicio en Go `SemgrepWorker` ([semgrep_worker.go](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/backend/internal/core/services/semgrep_worker.go)).
2. El orquestador ejecuta contenedores efímeros basados en `semgrep/semgrep:latest` montando el volumen del alumno.
3. Capturar el análisis JSON e inyectarlo en la columna `semgrep_audit JSONB` en PostgreSQL.

## Consecuencias
* **Positivas:**
  * Capacidad de auditar patrones de diseño y calidad de código AST de forma 100% automatizada.
  * Persistencia estructurada en PostgreSQL (`JSONB`).
