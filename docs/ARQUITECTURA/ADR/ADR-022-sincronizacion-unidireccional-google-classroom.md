# ADR-022: Integración y Sincronización Unidireccional con Google Classroom (D6)

* **Estado:** Aceptado
* **Fecha:** 2026-08-04

## Contexto y Problema
La plataforma debe integrarse con los flujos de trabajo de los docentes sin imponer la creación manual de listas de estudiantes o tareas duplicadas.

## Decisión Tomada
1. Adoptar la decisión de arquitectura **D6**: Implementar la sincronización **unidireccional** desde Google Classroom mediante OAuth2 Scopes.
2. Importar listas de estudiantes y tareas de asignación sin modificar los registros del sistema de gestión de aprendizaje (LMS) institucional de origen.

## Consecuencias
* **Positivas:**
  * Facilidad de onboarding para docentes y alumnos mediante credenciales institucionales.
  * Garantía de no alteración de la información de origen en Google Classroom.
