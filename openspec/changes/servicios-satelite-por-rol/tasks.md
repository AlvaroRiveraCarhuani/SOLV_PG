# Tareas: Servicios Satélite por Rol, Pin del Entorno y Contratos de Capacidad y Aislamiento

## Pronóstico de Carga de Revisión

| Campo | Valor |
|-------|-------|
| Líneas estimadas modificadas | 2,200–2,800 (en 6 unidades de trabajo autocontenidas) |
| Riesgo de presupuesto de 400 líneas | Alto |
| PRs encadenados recomendados | Sí |
| División sugerida | 6 unidades de trabajo (~350–450 líneas cada una) |
| Estrategia de entrega | ask-on-risk |

### Unidades de Trabajo

| Unidad | Objetivo | Comando de prueba focalizado | Límite de reversión |
|--------|----------|------------------------------|---------------------|
| 1 | Contrato de bindings, roles y validación server-side | `cd backend && go test ./internal/core/domain/... ./internal/core/services/... -run 'Service|Binding|Role' -count=1` | `core/domain/admin_governance.go`, `core/services/admin_governance_templates.go` |
| 2 | Capacidad: `expected_mb` / `reserve_mb`, reserva del nodo y piso vital | `cd backend && go test ./internal/core/services/... -run 'Ram|Capacity|Floor' -count=1` | `admin_governance_templates.go`, `workspace_service.go` |
| 3 | Pin del entorno al publicar y lectura en el arranque | `cd backend && go test ./internal/core/services/... -run 'Snapshot|Publish|Legacy' -count=1` | `evaluation_exercise_crud.go`, migración `environment_snapshot` |
| 4 | Adaptadores, pipeline de N bindings, init y scale-to-zero por conjunto | `cd backend && go test ./internal/core/services/... ./internal/infrastructure/docker/... -run 'Satellite|Binding|ScaleToZero' -count=1` | `workspace_satellite.go`, `workspace_service.go`, `docker_satellite.go` |
| 5 | Aislamiento por motor y scope con tenant | `cd backend && go test ./internal/core/services/... -run 'Isolation|ACL|Role' -count=1` | `workspace_satellite.go`, adaptadores de aislamiento |
| 6 | Studio, copiloto y modal docente | `cd frontend && npx ng test --watch=false --include='**/template-studio*.spec.ts' --include='**/template-request-modal*.spec.ts' && npm run lint:styles` | `templates/studio/`, `teacher/templates/`, `admin/services/` |

---

## Fase 1: Contrato de Bindings y Validación

- [ ] 1.1 RED: Pruebas en `backend/internal/core/domain/admin_governance_test.go` para `EffectiveRole` (rol explícito, `category: cache`, motor Redis), cardinalidad (`multiple_primary_database`), motor duplicado, rol incoherente, alias repetido y motor desconocido.
- [ ] 1.2 GREEN: Implementar `ServiceRole`, `ServiceBinding`, `InitKind`, `EffectiveRole` y `NormalizeAndValidateServices` en `backend/internal/core/domain/admin_governance.go` y `template_component.go`.
- [ ] 1.3 RED: Pruebas en `backend/internal/core/services/admin_governance_service_test.go` para `CreateOfficialTemplate` y `UpdateOfficialTemplate` con dos bindings válidos y con cada rechazo (`422`).
- [ ] 1.4 GREEN: Llamar la validación en alta y edición, agregar los códigos de error estables y mantener `category` / `database_id` como alias de lectura.
- [ ] 1.5 GREEN: Agregar `service_role` a `template_components` con migración aditiva y derivación por motor para filas existentes; exponerlo en el catálogo.
- [ ] 1.6 Verificación: `cd backend && go vet ./... && go test ./internal/core/domain/... ./internal/core/services/... -run 'Service|Binding|Role' -count=1`.

## Fase 2: Capacidad y Reserva del Nodo

- [ ] 2.1 RED: Pruebas de capacidad en `backend/internal/core/domain` y `core/services` para `expected_mb` / `reserve_mb`, reserva una vez por identidad y fórmula `(allocatable − Σ reserve) / per_lab_quota`.
- [ ] 2.2 GREEN: Extender `SatelliteServiceCapability` con `role`, `expected_mb`, `reserve_mb` e `init_kinds`; eliminar la tabla de megabytes hardcodeada de `calculateMinimumVitalFloorMB`.
- [ ] 2.3 GREEN: Ajustar `StartWorkspaceForExercise` para descontar la reserva de satélites y rechazar el alta si no entra, en lugar de confiar en el OOM.
- [ ] 2.4 GREEN: Ajustar `computeRecommendedRamMBForStack` y `computeMinimumFloorMBForStack` para recibir una lista de bindings y no sumar satélites a la cuota del alumno.
- [ ] 2.5 Verificación: `cd backend && go test ./internal/core/services/... -run 'Ram|Capacity|Floor' -count=1` y `cd frontend && npm run lint:ts`.

## Fase 3: Pin del Entorno al Publicar

- [ ] 3.1 RED: Pruebas para publicar un ejercicio con dos bindings, editar la plantilla y verificar que el arranque usa el snapshot; y para un ejercicio sin snapshot con fallback a la plantilla viva.
- [ ] 3.2 GREEN: Agregar `environment_snapshot JSONB` a `exercises` con migración nullable y el modelo de dominio.
- [ ] 3.3 GREEN: Persistir el snapshot al publicar (imagen, entrypoint, timeout, RAM, bindings, init, huella) y leerlo en `resolveTemplateForWorkspace`.
- [ ] 3.4 GREEN: Registrar el fallback legacy y avisar en la UI de administración cuántos ejercicios publicados usan una plantilla.
- [ ] 3.5 Verificación: `cd backend && go test ./internal/core/services/... -run 'Snapshot|Publish|Legacy' -count=1`.

## Fase 4: Adaptadores y Pipeline de N Bindings

- [ ] 4.1 RED: Pruebas de `ResolveTemplateBindings` (orden primario/auxiliar, dos bindings), arranque con un motor caído y scale-to-zero con dos laboratorios.
- [ ] 4.2 GREEN: Definir `EngineAdapter`, `EngineCapabilities`, `InstanceKey` y el registro compilado en `core/domain` y `infrastructure/docker`.
- [ ] 4.3 GREEN: Implementar los adaptadores de SQLite, PostgreSQL de aula, MySQL, MariaDB, MongoDB y Redis con imagen, techo, readiness, env e init declarados.
- [ ] 4.4 GREEN: Reemplazar `templateSatelliteEngine` / `templateManagedEngine` por la resolución de la lista, inyectar el entorno de todos los bindings y abortar el arranque si uno falla.
- [ ] 4.5 GREEN: Convertir `ReconcileIdleManagedSatellites` a refcount por conjunto de motores y sellar el init por `(exercise_id, service_alias)`.
- [ ] 4.6 Verificación: `cd backend && go test ./internal/core/services/... ./internal/infrastructure/docker/... -run 'Satellite|Binding|ScaleToZero' -count=1`.

## Fase 5: Aislamiento por Motor

- [ ] 5.1 RED: Pruebas de integración con dos alumnos: PostgreSQL (rol/base por scope), MongoDB (auth y base propia), Redis (ACL y prefijo) y MySQL (base/usuario).
- [ ] 5.2 GREEN: Implementar el scope `tenant + alumno + materia + alias` y la derivación de credenciales con separación de dominio.
- [ ] 5.3 GREEN: PostgreSQL con rol por scope, `REVOKE CONNECT ... FROM PUBLIC` y DSN administrativo separado; quitar el superusuario del entorno del laboratorio.
- [ ] 5.4 GREEN: MongoDB con `--auth` y usuario `readWrite` por base; Redis con `--aclfile`, patrón de claves y comandos peligrosos denegados.
- [ ] 5.5 GREEN: Rechazar en la validación un binding cuyo adaptador no pueda aislar.
- [ ] 5.6 Verificación: `cd backend && TEST_DB_DSN=... go test ./tests/integration/... -run 'Isolation|ACL' -count=1` y prueba manual con dos laboratorios.

## Fase 6: Studio, Copiloto y Modal Docente

- [ ] 6.1 RED: Pruebas de componente para la Columna 3 en dos bloques, reemplazo de la base principal, selección de Redis auxiliar, payload con dos bindings y etiqueta de costo del nodo.
- [ ] 6.2 GREEN: Implementar el estado de bindings en `TemplateStudioComponent`, los dos bloques y el resumen lateral.
- [ ] 6.3 GREEN: Enviar `services_config.services` y `stack_config.services`; mantener `database_id` como alias de escritura durante una versión.
- [ ] 6.4 GREEN: Actualizar el modal de solicitud docente y el copiloto para proponer y validar bindings.
- [ ] 6.5 Verificación: `cd frontend && npx ng test --watch=false --include='**/template-studio*.spec.ts' --include='**/template-request-modal*.spec.ts' && npm run lint:styles && CI=true npm run build`.

## Fase 7: Identidad Versionada del Satélite (Diferida)

- [ ] 7.1 RED: Pruebas de clave `(engine, major, profile)` y migración de nombres/volúmenes.
- [ ] 7.2 GREEN: Implementar la clave versionada y el renombrado de volúmenes con verificación previa.
- [ ] 7.3 GREEN: Exponer la versión efectiva del adaptador en capacidades y en la UI.
- [ ] 7.4 Verificación: arranque de dos versiones del mismo motor en un nodo de prueba.

---

## Dependencias y Puertas

- La fase 1 DEBE mantener el tope de un binding por plantilla hasta que la fase 4 esté desplegada. Si se retira antes, la API acepta dos servicios que el pipeline todavía ignora; el tope se elimina en la misma entrega que el aprovisionamiento de N bindings.
- La fase 2 DEBE completarse antes de la fase 4: aprovisionar dos motores con la cuenta de memoria actual empeora la admisión.
- La fase 5 DEBE completarse antes de la fase 6: no se habilita Redis auxiliar sin ACL.
- La fase 3 puede entrar en paralelo con la 1 y la 2; no depende de los adaptadores.
- La fase 7 no bloquea las anteriores.
- Cada fase se entrega con su documentación y con los tests de la unidad en verde.
