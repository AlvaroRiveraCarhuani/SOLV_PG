# Verificación de Pruebas — SLICE 14: Panel Administrador Institucional

## 1. Estado del Slice
- **Estado General:** Planificado (Pendiente de implementación en backend y frontend).
- **Componentes Previos Verificados:** Endpoints básicos de auditoría y métricas (`GET /api/v1/admin/audit-logs`, `GET /api/v1/admin/metrics/health`) verificados en `slice11_5_test.go`.

## 2. Plan de Pruebas a Ejecutar tras Implementación
1. **Modo Mantenimiento (ADR-031):**
   - Validación de respuesta 503 para estudiantes y docentes durante mantenimiento.
   - Verificación de acceso permitido (bypass) para el rol admin.
2. **Acciones de Emergencia (ADR-032):**
   - Validación de rechazo (400 Bad Request) si la frase de confirmación es incorrecta.
   - Ejecución de evicción masiva y verificación de estado detenido en Docker.
3. **Gestión de Estudiantes (ADR-033):**
   - Prueba de reseteo de strikes OOM y desbloqueo de workspace.
4. **Reasignación de Docentes (ADR-036):**
   - Verificación de integridad relacional en PostgreSQL al reasignar materias.
