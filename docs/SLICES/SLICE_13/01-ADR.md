# SLICE 13 — ADRs y Decisiones de Arquitectura

## Contexto y Alcance
El Slice 13 consolida la capa BaaS para la gestión académica, evaluación docente y retroalimentación del Juez Virtual en SOLV.
Permite a los docentes diseñar laboratorios, monitorear el estado de los alumnos mediante un dashboard reactivo (*Exception-Based Reporting*), corregir entregas con la herramienta *SpeedGrader*, desenmascarar casos de prueba privados, dejar feedback pedagógico anclado a código, ejecutar pruebas efímeras en sandbox y exportar la matriz de calificaciones en CSV.

---

## Decisiones de Arquitectura Aplicadas (ADRs)

### ADR-026: Consolidación de Capacidades Docentes BaaS y Juez Virtual
- **Estado:** Aceptado / Implementado
- **Decisión:** 
  1. Desacoplar la creación de ejercicios en fases (`draft` -> `published`), exigiendo un mínimo de 1 caso de prueba público para evitar ejercicios inaccesibles.
  2. Implementar importación masiva de casos de prueba mediante CSV con soporte RFC 4180 (comas entre comillas, Unicode, CRLF) con reporte exacto de línea en caso de error (422).
  3. Adoptar el patrón *Exception-Based Reporting* en el dashboard docente clasificando incidentes por severidad (`critical` [OOMKilled], `warning` [ASTBlocked], `standard` [PendingReview]).
  4. Métrica predictiva `at_risk`: Estudiante matriculado sin entregas Y sin heartbeat en las últimas 24 horas previas al `due_date`.
  5. Modo de corrección *SpeedGrader*: Desenmascaramiento de casos privados para docentes y punteros de navegación continua (`prev_submission_id` / `next_submission_id`).
  6. Comentarios in-line persistidos en la tabla `submission_comments` anclados a `line_number`.
  7. Runner efímero en memoria sin persistir submissions en base de datos.
  8. Exportación de calificaciones en CSV con compatibilidad Excel (UTF-8 BOM).

### ADR-012: Protección de Casos de Prueba Privados (Anti-Cheat)
- **Estado:** Activo
- **Decisión:** Los casos de prueba con `is_hidden = true` se desenmascaran únicamente para peticiones autenticadas con rol `teacher` o `admin`. El rol `student` recibe `403 Forbidden` en rutas de revisión docente.

---

## Restricciones y Reglas Invariables
1. Cero lógica de frontend en la capa backend (BaaS puro con contratos JSON / CSV estándar).
2. Aislamiento multi-tenant estricto mediante `X-Tenant-Id` y verificación de pertenencia en `subjects.teacher_id`.
3. Validaciones de integridad: `override_reason` requiere al menos 10 caracteres (`422 Unprocessable Entity` si es menor).
