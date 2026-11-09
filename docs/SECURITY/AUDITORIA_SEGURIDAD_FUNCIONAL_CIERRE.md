# Cierre de Auditoría Independiente — Seguridad Funcional SOLV

**Fecha de emisión:** 2026-11-09  
**Veredicto:** APROBADA  
**Ámbito:** Verificación de solo lectura sobre las correcciones de cuatro hallazgos de seguridad funcional  
**Estado de la fase:** CERRADA  

---

## 1. Resumen Ejecutivo

La fase de seguridad funcional de SOLV queda formalmente cerrada tras la auditoría independiente de solo lectura efectuada sobre el código del backend. Se comprobó la resolución completa y efectiva de los cuatro hallazgos de seguridad identificados previamente, sin encontrar omisiones en rutas alternativas ni regresiones en los mecanismos de autenticación y autorización perimetral.

---

## 2. Resolución de los Cuatro Hallazgos de Seguridad Funcional

### Hallazgo 1: Autorización Administrativa
- **Estado:** RESUELTO
- **Diagnóstico previo:** Rutas administrativas dependientes de cabeceras de cliente o sin restricción explícita por rol.
- **Controles comprobados:**
  - Se reconstruyó el inventario desde `backend/internal/delivery/http/router.go`: exactamente 87 rutas administrativas están protegidas mediante el wrapper `adminOnly` (`tm(adminOnly(next))` y `tm(am(adminOnly(next)))`).
  - Las dos únicas rutas con prefijo `/admin` asignadas a `staffOnly` corresponden a lecturas de catálogo permitidas a docentes: `GET /api/v1/admin/templates` y `GET /api/v1/admin/template-components`. Sus variantes de mutación (`POST`, `PUT`, `PATCH`, `DELETE`) exigen estrictamente `adminOnly`.
  - La ruta `POST /api/v1/jobs/env-test` exige `staffOnly` (docentes y administradores); los estudiantes reciben respuesta 403 Forbidden antes de alcanzar el handler.
  - Se verificaron alias y métodos alternativos (ej. `POST /api/v1/plantillas/{id}/duplicar`, `GET /api/v1/academic-periods`, alias de componentes y `POST /api/v1/tenants/logo`), confirmando protección homogénea.
  - El middleware `SanitizeIdentityHeaders` en `backend/internal/delivery/http/middleware/identity_headers.go` elimina del request cualquier cabecera `X-User-Id`, `X-User-Role`, `X-Tenant-Id`, `X-User-Email` o `X-Forwarded-*` antes de ingresar al router.
  - Orden de middleware: `TenantMiddleware` (`tm`) valida la firma del JWT y la existencia del tenant inyectando la identidad en `r.Context()`; posteriormente `adminOnly` o `staffOnly` evalúan el rol desde el contexto verificado.
- **Evidencia en tests:** `TestAdminOnlyRejectsNonAdminRoles`, `TestAdminOnlyAllowsAdmin`, `TestStaffOnlyAllowsTeacherAndAdmin`, `TestStaffOnlyRejectsStudentAndUnknownRoles`, `TestAdminRoutesRejectStudentsAndTeachersBeforeHandlers` y `TestEnvTestStartRejectsStudentBeforeHandler` en `backend/internal/delivery/http/authorization_test.go`.

### Hallazgo 2: Redacción de Secretos en Auditoría
- **Estado:** RESUELTO
- **Diagnóstico previo:** Almacenamiento potencial de credenciales y claves en texto plano dentro de la columna `metadata` de `audit_logs`.
- **Controles comprobados:**
  - En `backend/internal/delivery/http/middleware/audit_middleware.go`, el cuerpo crudo de la petición se procesa en memoria y se sanitiza con `sanitizeAuditBody` antes de llamar a `pool.Enqueue(logEntry)`. El payload crudo no se persiste en base de datos.
  - Redacción recursiva en `redactAuditValue` sobre mapas y listas anidadas a cualquier profundidad.
  - La función `isSensitiveAuditKey` normaliza mayúsculas, minúsculas, espacios, guiones y guiones bajos (`apikey`, `encryptedapikey`, `token`, `password`, `secret`, `clientsecret`, `authorization`, etc.), reemplazando los valores por `[REDACTED]`.
  - Cuerpos con formato no JSON (formularios, texto plano o binario) son descartados devolviendo `nil` en `Metadata`.
  - Los metadatos no sensibles legítimos se conservan intactos.
  - Las consultas de auditoría (`GET /api/v1/admin/audit-logs` y `GET /api/v1/admin/audit-logs/actors/{actorId}/timeline`) exigen autorización administrativa (`adminOnly`) y resuelven el `tenant_id` desde el contexto autenticado. No existen rutas accesibles a estudiantes o docentes.
- **Evidencia en tests:** `TestAuditMiddlewareRedactsNestedSecrets` y `TestAuditMiddlewareDiscardsNonJSONBody` en `backend/internal/delivery/http/middleware/audit_middleware_redaction_test.go`.

### Hallazgo 3: Propiedad de Entornos de Práctica
- **Estado:** RESUELTO
- **Diagnóstico previo:** Ausencia de validación de pertenencia del entorno al estudiante en operaciones de ciclo de vida.
- **Controles comprobados:**
  - En `backend/internal/infrastructure/storage/postgres/practice_environment_repository.go`, `GetByID` filtra siempre por `WHERE id = $1 AND tenant_id = $2`. Entornos de otro tenant devuelven error de no encontrado sin filtrar información.
  - En `backend/internal/delivery/http/practice_environment_handler.go`, la función `authorizeEnvironment` y el helper `canAccessPracticeEnvironment` comprueban la propiedad: un estudiante solo puede operar sobre entornos donde `ws.StudentID == userID`.
  - Las operaciones `TerminatePracticeEnvironment`, `GetSemgrepAudit`, `Heartbeat`, `RestartPracticeEnvironment` y `PausePracticeEnvironment` ejecutan `authorizeEnvironment` antes de cualquier efecto secundario sobre el servicio o Docker.
  - Docentes y administradores pueden gestionar entornos de alumnos dentro de su tenant, alineado con las reglas funcionales de supervisión de clase de SOLV.
- **Evidencia en tests:** `TestPracticeEnvironmentOperationsRejectOtherStudents`, `TestPracticeEnvironmentOwnerKeepsTerminateAuditHeartbeatAndPause` y `TestPracticeEnvironmentHandler_LifecycleOnMissingPracticeEnvironmentReturnsNotFound` en `backend/internal/delivery/http/practice_environment_authorization_test.go`.

### Hallazgo 4: Configuración Pública
- **Estado:** RESUELTO
- **Diagnóstico previo:** Endpoint de configuración pública exponía el mapa completo de `tenants.config`, incluyendo configuraciones internas y claves.
- **Controles comprobados:**
  - En `backend/internal/delivery/http/config_handler.go`, la respuesta se genera a partir del struct explícito `publicTenantConfig`, que admite exclusivamente 9 campos institucionales públicos (`tenant_id`, `slug`, `institution_name`, `logo_url`, `tenant_primary_color`, `base_domain`, `support_email`, `font_sans_family`, `font_mono_family`).
  - No se serializa `tenants.config` ni se exponen campos como `ai_settings`, `encrypted_api_key`, `server_policies` o `user_overrides`.
  - Se eliminó del handler toda lectura de la cabecera `X-Tenant-Id`. La selección del tenant se realiza únicamente vía parámetro de consulta `?slug=` o utilizando el tenant predeterminado del sistema.
- **Evidencia en tests:** `TestPublicConfigOnlyReturnsAllowlistedFields`, `TestPublicConfigIgnoresTenantHeader` y `TestPublicConfigSelectsTenantBySlug` en `backend/internal/delivery/http/identity_security_test.go`.

---

## 3. Pruebas Ejecutadas

Se ejecutaron localmente las pruebas unitarias y de integración en memoria correspondientes a las capas delivery, servicios y middleware:

- Paquete `backend/internal/delivery/http`: Todos los tests de autorización, seguridad de identidad, configuración pública y entornos de práctica pasan en verde.
- Paquete `backend/internal/delivery/http/middleware`: Todos los tests de redacción de auditoría, limitación de tasa y revocación de sesión pasan en verde.
- Paquete `backend/internal/core/services`: Tests de OAuth, aprovisionamiento y terminal pasan en verde.
- Tiempo total acumulado de ejecución: < 3 segundos.

---

## 4. Deuda Operativa Registrada

| Identificador | Descripción | Ámbito | Acción Requerida |
| :--- | :--- | :--- | :--- |
| **COST-09** | Saneamiento de Registros Históricos de Auditoría | Seguridad / Base de Datos | En despliegues existentes, la tabla `audit_logs` puede contener entradas creadas previo a la incorporación del middleware de redacción con secretos en su campo `metadata`. Aunque la API impide su lectura a usuarios no administradores, se debe ejecutar un script de saneamiento o migración de mantenimiento para depurar datos históricos y evaluar rotación de credenciales previas. Registrado en `docs/GOBERNANZA/BACKLOG_COSTURAS.md`. |

---

## 5. Alcance y Limitaciones de Validación

- **Fuera de alcance:** Las pruebas de integración con PostgreSQL físico y Docker Engine (ej. pruebas E2E globales, validación de aislamiento de red por iptables y suites de integración de larga duración) quedaron explícitamente fuera del alcance de esta auditoría funcional de solo lectura.
- **Fase correspondiente:** Dicha verificación de infraestructura real debe completarse durante la fase de certificación operativa previa al despliegue o piloto con usuarios.

---

## 6. Conclusión Formal

La auditoría de seguridad funcional concluye con resultado satisfactorio. Los cuatro hallazgos de seguridad se encuentran cerrados, no se introdujeron regresiones en los flujos principales del backend y los puntos residuales quedan documentados en el backlog de deuda operativa.
