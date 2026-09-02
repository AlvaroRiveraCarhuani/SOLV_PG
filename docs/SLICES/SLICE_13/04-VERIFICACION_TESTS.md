# Verificación de Pruebas — SLICE 13: Experiencia Docente, Cursos y Creación de Laboratorios

## 1. Estado del Slice
- **Estado General:** Planificado (Pendiente de implementación de backend extendido y frontend).
- **Cobertura Base Existente:** Endpoints académicos básicos (`POST /api/v1/exercises`, `POST /api/v1/submissions/{id}/override`) verificados en `slice11_5_test.go` y `academic_schema_test.go`.

## 2. Plan de Pruebas a Ejecutar tras Implementación
1. **Creación de Ejercicio:** Validación de que los casos de prueba ocultos no se expongan al rol estudiante.
2. **Solicitud de Plantillas (ADR-030):** Verificación de estado `pending_review` y aislamiento de plantillas no autorizadas.
3. **Filtro por Periodo Académico (ADR-029):** Comprobación de cursos archivados y visibilidad por periodo lectivo.
4. **Calificación Manual de Entregas:** Verificación de actualización en DB y registro en el log de auditoría.
