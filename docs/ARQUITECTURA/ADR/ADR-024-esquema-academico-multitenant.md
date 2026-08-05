# ADR-024: Esquema Académico Multi-Tenant Unificado

* **Estado:** Aceptado
* **Fecha:** 2026-08-04
* **Slice:** 9

## Contexto y Problema
La plataforma requería un modelo de datos relacional para gestionar materias (`subjects`), inscripciones de alumnos (`enrollments`), entregas al Juez Virtual (`submissions`) e invitaciones docentes (`teacher_invitations`), manteniendo estricto aislamiento por institución (`tenant_id`).

## Decisión Tomada
1. Crear las tablas `subjects`, `enrollments`, `submissions` y `teacher_invitations` incluyendo `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT`.
2. Vincular la tabla `workspaces` a `subjects` mediante la Foreign Key `fk_workspaces_subject`.
3. Crear los 4 índices de rendimiento: `idx_subjects_tenant`, `idx_submissions_tenant`, `idx_submissions_exercise`, `idx_enrollments_student`.
4. Exponer endpoints HTTP protegidos por `TenantMiddleware`.

## Consecuencias
* **Positivas:**
  * Soporte relacional completo para gestión académica en producción.
  * Consultas SQL altamente performantes con filtrado por tenant.
  * Eliminación de deuda técnica de FKs huérfanas.
