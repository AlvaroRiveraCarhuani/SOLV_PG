```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: working-tree-2026-11-16 (base 1a6cf8d)
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 6/6 implemented
scenarios: 9/11 with automated evidence; 2 integration scenarios not executed
test_command: cd backend && go test ./internal/core/services -run 'TestBackupService' -count=1
test_exit_code: 0
test_output_hash: sha256:c141d62632e448977e77cf69ebfdb940c305267f6238806faf52e3f74f53748a
frontend_command: cd frontend && npx ng test --watch=false --include='**/admin/configuracion/tabs/admin-config-servidor.component.spec.ts'
frontend_exit_code: 0
frontend_result: 20 tests passed
frontend_output_hash: sha256:5ec539910983a7d87085f8d040691bf6cf4bc21a38e5835133bc39737fa552b1
lint_command: cd frontend && npm run lint:styles
lint_exit_code: 0
lint_output_hash: sha256:3134c732ffaa924c4ce09faa65e42fd9b30b3a85572877c65e626fce0e83ef64
```

## Verification Report

**Change**: fix-respaldo-corrupto
**Mode**: Strict TDD, verificación acotada por impacto
**Fecha**: 2026-11-16

### Completeness

| Métrica | Valor |
|---|---|
| Tareas | 13/13 marcadas |
| Fases | 4/4 |
| Requisitos | 6/6 implementados |
| Escenarios con evidencia automatizada | 9/11 |
| Escenarios de integración no ejecutados | 2/11 |

Las tareas están completas. La integración HTTP con PostgreSQL no se ejecutó porque no hay `TEST_DB_DSN` ni base levantada en esta sesión.

### Build and Tests Execution

**Backend**: pasan las pruebas unitarias del servicio de respaldos.
```text
ok solv-backend/internal/core/services
```

**Frontend**: pasan 20 pruebas del componente de configuración del servidor.

**Gate de estilos**: pasa.

### Spec Compliance Matrix

| Requisito | Evidencia | Resultado |
|---|---|---|
| Contrato `data.is_valid` | `VerifyBackupResult`, `verifyBackup` y pruebas de envelope OK, faltante y mismatch | Cumplido |
| Mensaje de error del envelope | `resolveError` lee `error.message` antes del fallback | Cumplido por código |
| Tres estados distintos | Pruebas de checksum OK, archivo faltante y checksum mismatch | Cumplido |
| Persistencia del resultado | `UpdateExecutionVerifyColumns`, migración `00008`, columnas `last_verify_*` | Implementado; integración no ejecutada |
| Recarga conserva el resultado | Indicador enlazado a `last_verify_ok/at`; prueba de carga con historial | Implementado; integración no ejecutada |
| Tamaño sub-MB | `formatBytes`; pruebas 150 B, 1024 B y 0 B | Cumplido |
| Confirmación honesta al crear | Prueba de que el toast no afirma verificación | Cumplido |
| Write/Close falla y marca `failed` | `TestBackupService_TriggerBackupFailsClosedOnContentWriteError` y `...OnGzipIOErrors` | Cumplido |
| Salida bajo 1024 B marca `failed` | `TestBackupService_TriggerBackupRejectsOutputBelowFloor` | Cumplido |
| Checksum desde relectura | `TestBackupService_TriggerBackupChecksumsReReadBytes` | Cumplido |

### Correctness and Design

- `MinBackupSizeBytes` es 1024 y se aplica antes de persistir el éxito.
- Los errores de escritura, cierre, `Stat` y relectura marcan `failed` y eliminan el archivo parcial cuando corresponde.
- `VerifyBackup` persiste únicamente las columnas de verificación y no cambia `status` ni `is_valid`.
- La creación exitosa calcula el checksum releyendo el archivo terminado.
- La UI separa faltante de mismatch y no promete restaurabilidad.

### Issues Found

**CRITICAL**: ninguno.

**WARNING**:

- W1: No se ejecutó `TestSlice16_Backups` contra PostgreSQL; la persistencia y la recarga de `last_verify_*` se verificaron por código, migración y pruebas de componente, no contra la base real.
- W2: `TriggerBackup` marca `last_verify_ok=true` al crear el respaldo porque el checksum se calcula por relectura. Es una verificación de integridad en creación, pero difumina la semántica de “verificar” como acción posterior; conviene que la integración confirme que la UI y la auditoría lo interpretan igual.
- W3: Si `io.Copy` falla durante `VerifyBackup`, el servicio devuelve error y no persiste un outcome de verificación. El escenario de “archivo ilegible” no está cubierto por una prueba específica; el de archivo faltante sí.
- W4: No se ejecutó restauración real; el cambio la declara fuera de alcance y no debe presentarse como restaurabilidad certificada.

### Verdict

**PASS WITH WARNINGS**

Los seis requisitos están implementados y las pruebas unitarias y de UI pasan. Las dos advertencias de integración impiden declarar verificado el ciclo completo contra PostgreSQL, pero no bloquean el contrato de creación, verificación y presentación.
