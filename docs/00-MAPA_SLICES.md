# Mapa Global de Slices — Plataforma SOLV

> **Corte de auditoría:** 2026-11-16
> **Alcance:** 17 vertical slices.
> **Regla de lectura:** `Implementado` describe código presente en el árbol; `Verificado` describe evidencia de ejecución. Ninguno de los dos certifica por sí solo la operación del piloto.

---

## 1. Estado de la auditoría

| Dimensión | Resultado al 2026-11-16 |
| :--- | :--- |
| Backend | `go build`, `go vet` y los tests de paquetes pasan. |
| Frontend | Build correcto; 145 archivos de tests, 1120 tests y 127 tests de gates pasan. |
| E2E | 23 de 23 flujos en verde el 2026-11-15, 554 s, sin fallos ni omitidos. |
| Lint de backend | `golangci-lint` reporta 909 hallazgos; el gate no está limpio. |
| Integración, Docker y migraciones | No se re-ejecutaron en esta auditoría; la evidencia disponible es histórica. |
| OpenSpec | 2 cambios sin `verify-report`, ambos sin tareas iniciadas; 4 activos y 5 archivados el 2026-11-16 con reporte. |
| Operación | Sin apagado ordenado del API; ADR-050 fase 3 sin implementar; checklist pre-piloto `NO CERTIFICADO`. |

La lectura correcta es: el producto está implementado y los flujos principales pasan, pero la validación integral y la operación del piloto no están cerradas.

---

## 2. Tabla general de slices

| Slice | Título | ADRs vinculados | Backend | Frontend | Verificación |
| :---: | :--- | :--- | :---: | :---: | :--- |
| **01** | Canal de Comunicación Básico | ADR-001, ADR-002 | Implementado | N/A | Integración histórica |
| **02** | Persistencia e Identidad Básica | ADR-003, ADR-004 | Implementado | N/A | Integración histórica |
| **03** | Evaluación Algorítmica y Base de Datos | ADR-005, ADR-006 | Implementado | N/A | Integración histórica |
| **04** | Entorno Interactivo Web | ADR-007, ADR-008 | Implementado | N/A | Integración histórica |
| **05** | Orquestador de Recursos y QoS | ADR-009, ADR-010 | Implementado | N/A | Integración histórica |
| **06** | Blindaje de Red y Telemetría | ADR-011, ADR-013 | Implementado | N/A | Integración histórica |
| **07** | OpenVSCode Server y Auditoría AST | ADR-014, ADR-015 | Implementado | N/A | Integración histórica |
| **08** | Hardening BaaS y Modelo Unificado | ADR-016, ADR-017, ADR-018, ADR-019 | Implementado | N/A | Integración histórica |
| **09** | Esquema Académico y Seguridad | ADR-020, ADR-021, ADR-024 | Implementado | N/A | Integración histórica |
| **10** | Robustez BaaS y Resiliencia | ADR-022, ADR-023, ADR-025 | Implementado | N/A | Integración histórica |
| **11** | Operabilidad B2B y Migrations Lock | ADR-027, ADR-028 | Implementado | N/A | Integración histórica |
| **12** | Experiencia Estudiante y Evaluación en Tiempo Real | ADR-012, ADR-029, ADR-037 | Implementado | Implementado | Backend 2026-09-02; E2E 2026-11-15 |
| **13** | Experiencia Docente, Cursos y Creación de Prácticas | ADR-026, ADR-029, ADR-030, ADR-037 | Implementado | Implementado | Integración 2026-09; E2E 2026-11-15 |
| **14** | Panel Administrador Institucional y Gobernanza | ADR-024, ADR-027, ADR-029, ADR-030, ADR-031, ADR-032, ADR-033, ADR-036 | Implementado | Implementado | E2E 2026-11-15; documento de verificación desactualizado |
| **15** | Notificaciones Proactivas | ADR-031, ADR-032, ADR-034, ADR-035 | Implementado | Implementado (in-app) | Integración y PostgreSQL real; correo pendiente |
| **16** | Backups y Retención Institucional | ADR-027, ADR-034, ADR-035 | Implementado | Implementado | Integración `slice16_backups_test`; restauración operativa no ejecutada |
| **17** | Telemetría Verdadera del Panel de Salud | ADR-008, ADR-043, ADR-049 | Implementado | Implementado | Integración 2026-10-27; no re-ejecutada |

---

## 3. Estado por rol

| Rol | Slices | Estado funcional | Límite vigente |
| :--- | :--- | :--- | :--- |
| Estudiante | 12 | Inicio, materias, evaluación algorítmica, entorno de desarrollo, historial y examen implementados. | La matriz de piloto está redactada y `PENDIENTE` de ejecución. |
| Docente | 13, 17 | Aula, prácticas, revisión, rúbrica, libreta, avances y telemetría implementados. | La matriz de piloto está redactada y `PENDIENTE` de ejecución. |
| Administrador | 14, 16, 17 | Gobernanza, auditoría, respaldos y panel de salud implementados. | La restauración y el scrape de métricas no están certificados en operación. |
| Transversal | 15 | Notificaciones in-app implementadas. | El envío de correo y las preferencias por usuario siguen pendientes. |

### Detalle del panel administrador (Slice 14)

| Submódulo | Ruta | Estado | Alcance |
| :--- | :--- | :---: | :--- |
| 14.1 Salud y recursos | `/admin/dashboard` | Implementado | Métricas de host, contenedores e incidentes. |
| 14.2 Gestión de docentes | `/admin/docentes` | Implementado | Invitaciones, roles y reasignación de materias. |
| 14.3 Cursos y períodos | `/admin/cursos` | Implementado | Períodos, alta de cursos y docente titular. |
| 14.4 Directorio de estudiantes | `/admin/estudiantes` | Implementado | Búsqueda, inscripciones y reseteo de strikes OOM. |
| 14.5 Plantillas | `/admin/plantillas` | Implementado | Catálogo, aprobación y límites de recursos. |
| 14.6 Configuración y mantenimiento | `/admin/configuracion` | Implementado | Branding, períodos, QoS, respaldos y mantenimiento. |
| 14.7 Auditoría y emergencias | `/admin/auditoria` | Implementado | Audit log, cronología y acciones de emergencia. |

---

## 4. Brechas abiertas

1. **Cierre de OpenSpec.** Quedan dos cambios sin `verify-report` y sin tareas iniciadas: `dashboard-customization-and-course-theming` y `servicios-satelite-por-rol`. Los cinco cambios con tareas completas se archivaron el 2026-11-16; dos de ellos quedaron como superados por ADR-055, ADR-057 y ADR-062.
2. **Pin del entorno.** La fase 3 de ADR-050, que fija la plantilla al publicar y la lee al arrancar el workspace, sigue `Pendiente`.
3. **Apagado ordenado.** `backend/cmd/api/main.go` crea `http.Server` y llama a `ListenAndServe` sin capturar `SIGTERM` ni cerrar servidor, workers y contenedores de forma coordinada.
4. **Lint de backend.** Los 909 hallazgos de `golangci-lint` impiden declarar el gate de calidad en verde, aunque `go vet` y los tests pasen.
5. **Documentación desalineada.** Los documentos de verificación de los Slice 12, 14 y 16 todavía describen frontend pendiente o slice planificado; ADR-012 menciona Monaco mientras el editor usado es CodeMirror; existen dos ADR-043 y dos ADR-060.
6. **Operación del piloto.** El [checklist pre-piloto](SECURITY/pre-pilot-checklist.md) mantiene 11 bloqueos: rama, CI, respaldo, restauración, métricas, rate limiting probado, CORS, capacidad, runbook, cuenta administrativa y consentimiento.
7. **Validación integral.** La integración contra PostgreSQL, Docker y migraciones no se re-ejecutó en esta auditoría. Los 23 flujos E2E son evidencia del 2026-11-15, no de una corrida posterior a estos documentos.

---

## 5. Estado de OpenSpec

| Cambio | Tareas | `verify-report` | Estado |
| :--- | :---: | :---: | :--- |
| `dashboard-customization-and-course-theming` | 0/8 | No | Sin iniciar |
| `feat-teacher-experience` | 14/14 | Sí | Verificado, sin archivar |
| `servicios-satelite-por-rol` | 0/37 | No | Sin iniciar |
| `template-environment-studio` | 18/18 | Sí | Verificado, sin archivar |

Archivados el 2026-11-16 con `verify-report`: `alta-docentes-y-servicio-smtp` (25/25, PASS WITH WARNINGS), `fix-respaldo-corrupto` (13/13, PASS WITH WARNINGS), `teacher-contents-repository` (10/10, PASS WITH WARNINGS), `teacher-anti-plagiarism` (6/6, SUPERSEDED por ADR-062) y `teacher-workspace-navigation-and-queue` (8/8, SUPERSEDED por ADR-055, ADR-057 y ADR-062). Los dos últimos conservan su spec solo como historia y no se fusionaron con `openspec/specs/`.

Los cambios con tareas completas pero sin `verify-report` no se consideran cerrados por este mapa. La verificación formal sigue siendo el requisito de archivo.

---

## 6. Historial

| Fecha | Cambio |
| :--- | :--- |
| 2026-11-16 | Se incorpora el Slice 17, se corrigen los estados de frontend de los Slice 12, 14 y 16, se registra el resultado de la auditoría y se separan implementación, verificación y operación. |
| 2026-11-16 | Se archivan cinco cambios OpenSpec con `verify-report`; se sincronizan cuatro specs vigentes y se corrige el contrato de conteo de la importación masiva de docentes. |
