```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: working-tree-2026-11-16 (base 1a6cf8d)
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 8/8 implemented
scenarios: 8/14 with automated evidence; 6 partial or static
test_command: cd backend && go test ./internal/infrastructure/mailer ./internal/core/services ./internal/delivery/http -run 'SMTP|Roster|TeacherInvitation|BuildTeacherWelcomeEmail|TestBackupService|TestBulkAddTestCases_EnvironmentGuard' -count=1
test_exit_code: 0
test_output_hash: sha256:c141d62632e448977e77cf69ebfdb940c305267f6238806faf52e3f74f53748a
frontend_command: cd frontend && npx ng test --watch=false --include='**/admin/teachers/**/*.spec.ts' --include='**/admin/configuracion/tabs/admin-config-servidor.component.spec.ts' --include='**/teacher/supervision/anti-plagio/teacher-plagiarism.component.spec.ts' --include='**/teacher/exercises/teacher-exercises.component.spec.ts' --include='**/teacher/templates/teacher-templates.component.spec.ts' --include='**/teacher/reviews/teacher-reviews.component.spec.ts'
frontend_exit_code: 0
frontend_result: 7 files, 78 tests passed
frontend_output_hash: sha256:5ec539910983a7d87085f8d040691bf6cf4bc21a38e5835133bc39737fa552b1
lint_command: cd frontend && npm run lint:styles
lint_exit_code: 0
lint_output_hash: sha256:3134c732ffaa924c4ce09faa65e42fd9b30b3a85572877c65e626fce0e83ef64
```

## Verification Report

**Change**: alta-docentes-y-servicio-smtp
**Mode**: Strict TDD, verificación acotada por impacto
**Fecha**: 2026-11-16

### Completeness

| Métrica | Valor |
|---|---|
| Tareas | 25/25 marcadas |
| Fases | 6/6 |
| Requisitos | 8/8 implementados |
| Escenarios con evidencia automatizada | 8/14 |
| Escenarios con evidencia estática o pendientes | 6/14 |

Las tareas están completas. La implementación no está íntegramente cubierta por pruebas de integración porque el repositorio no tiene PostgreSQL levantado ni `TEST_DB_DSN` configurado en esta sesión.

### Build and Tests Execution

**Backend**: pasa.
```text
ok solv-backend/internal/infrastructure/mailer
ok solv-backend/internal/core/services
ok solv-backend/internal/delivery/http
```

**Frontend**: pasan las 18 pruebas del área de docentes, dentro de una corrida de 78 pruebas de los cinco cambios.

**Gate de estilos**: pasa.
```text
inline-styles-gate: OK
date-format-gate: OK
machine-data-gate: OK
motion-gate: OK
terminology-gate: OK
```

### Defecto encontrado y corregido durante la verificación

El backend de importación masiva devuelve `registered_count`, pero el frontend leía `created_count`. El aviso de éxito mostraba `undefined docentes registrados exitosamente`. La prueba existente no lo detectaba porque el mock usaba la forma incorrecta.

Corrección aplicada:

- `BatchDirectTeacherResponse` usa `registered_count`, `total_processed` y `notified`.
- `AdminTeachersComponent` muestra `res.registered_count`.
- El mock de la prueba usa la respuesta real del backend.
- `DirectTeacherResponse` declara `data` y `notified`, que son los campos que devuelve el handler.

La corrección se verificó con las 18 pruebas de docentes y con el test de handler que ya afirmaba `registered_count`.

### Spec Compliance Matrix

| Requisito | Evidencia | Resultado |
|---|---|---|
| Alta directa individual | Handler `POST /api/v1/teachers/direct`, `UpsertDirectTeacher` y prueba HTTP 201 | Implementado; sin integración PostgreSQL |
| Promoción de estudiante existente | `ON CONFLICT (email) DO UPDATE SET role = 'teacher'` | Implementado; verificado estáticamente |
| Re-registro idempotente | Mismo upsert y actualización de rol | Implementado; verificado estáticamente |
| Carga CSV tolerante | `TestRosterParser_CSVWithSynonymHeaders`, `CSVWithSemicolonsAndDuplicates`, `RegexFallbackNoHeaders` | Cumplido |
| Carga `.xlsx` | `excelize.OpenReader` y lectura de la primera hoja | Implementado; sin prueba automatizada |
| Previsualización antes de persistir | `RosterUploadComponent` muestra total, válidos, duplicados e inválidos; prueba de cambio de pestaña y proceso | Cumplido en UI |
| Confirmación de alta masiva | Handler 201, `BatchUpsertDirectTeachers`, aviso corregido | Cumplido en contrato HTTP y UI |
| Notificación opcional | Interruptor activo por defecto; handler acepta `send_notification`; cola asíncrona | Parcial: el camino activo no tiene prueba de entrega |
| Configuración SMTP estándar | `NewSMTPMailerFromEnv`, STARTTLS y TLS directo | Implementado; sin relay real |
| Fallback local | `TestSMTPMailer_MockMode` y `TestSMTPMailer_AsyncQueue` | Cumplido |
| Plantilla HTML institucional | `TestBuildTeacherWelcomeEmail` valida institución, nombre, enlace y alternativa de texto | Cumplido |
| Cola no bloqueante | `SendAsync` sobre canal bufferizado y workers | Implementado; la prueba solo verifica que no bloquea, no la entrega |
| TLS puerto 465 | Rama `tls.Dialer` en el adaptador | Implementado; sin prueba |
| Alta concurrente de 50 docentes | No hay prueba de carga | No verificado |

### TDD and Assertion Quality

Las pruebas nuevas y existentes ejercitan comportamiento real: estados HTTP, conteos, correos válidos, deduplicación, plantilla y cola. La prueba de cola asíncrona es una prueba de humo: envía cinco mensajes y espera 50 ms, sin afirmar que el worker los consumió. No se encontraron aserciones tautológicas ni bucles vacíos.

### Issues Found

**CRITICAL**: ninguno.

**WARNING**:

- W1: No se ejecutó integración contra PostgreSQL; la promoción, el upsert idempotente y la persistencia masiva se verificaron por código y mocks, no contra la base real.
- W2: No existe `admin-teachers.service.spec.ts`, aunque la tarea 4.3 lo pedía. Las pruebas de componente cubren la llamada al servicio y el conteo, pero no la serialización HTTP aislada.
- W3: La rama `.xlsx` está implementada con `excelize`, pero no tiene prueba automatizada.
- W4: El camino `send_notification=true` se verificó estáticamente; no hay prueba que confirme que el mensaje llega a la cola ni que el trabajador lo despacha.
- W5: No se probó un relay SMTP real ni TLS 465; solo modo mock, MIME y construcción de plantilla.
- W6: El flujo E2E de alta directa, previsualización e inicio de sesión con Google SSO no se ejecutó en esta sesión.
- W7: La implementación vive en `TeacherInvitationService` y `TeacherInvitationHandler`, no en `TeacherService`/`TeacherHandler` como indican las tareas; el mailer se inicializa en `cmd/api/main.go`, no en `cmd/server/main.go`. El contrato HTTP y la funcionalidad son equivalentes, pero la trazabilidad de archivos quedó desalineada.

### Verdict

**PASS WITH WARNINGS**

El contrato funcional de alta directa, importación tolerante, previsualización y notificación está implementado. El defecto de conteo se corrigió y las pruebas acotadas pasan. Las advertencias son de cobertura: integración PostgreSQL, `.xlsx`, entrega real de correo, TLS y E2E no se ejecutaron.
