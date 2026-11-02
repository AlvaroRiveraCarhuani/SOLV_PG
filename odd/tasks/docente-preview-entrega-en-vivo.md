# Feature: Preview en Vivo de una Entrega IDE desde SpeedGrader

Objetivo: que el docente ejecute la entrega de un alumno en un contenedor desechable y abra su aplicación en el navegador, sin tocar el workspace vivo del alumno ni el de otro docente.

Branch: `feat/review-live-app` (desde `main`) | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm run test:ci`

Estado: **implementada el 2026-11-01, sin commit**. La migración `00044` está aplicada en la base local desde el arranque del 2026-11-02 (`goose` en versión 44); el e2e del flujo no existe. Esta ficha no propone cambios de diseño: cierra lo que ya está en el árbol de trabajo.

Fuente: `proyecto-grado/Observaciones/Reestructuracion Entorno Docente y Vocabulario.md` y `docs/ARQUITECTURA/ADR/ADR-061-preview-efimero-de-entrega-desde-speedgrader.md`.

## Decisiones cerradas

- **Aislamiento por copia.** Cada preview crea un volumen propio (`solv-review-env-<id>`) con el snapshot de la entrega. Nunca monta ni reinicia el workspace del alumno.
- **Receta explícita.** La práctica declara `app_run_command` y `app_port`. Sin comando no se adivina nada: se responde `app_run_command_missing`.
- **Sin botón de detener.** El QoS apaga el preview por inactividad a los 15 minutos, igual que el resto de entornos (ADR-011). La actividad se mide por tráfico de red, no por heartbeat HTTP.
- **Control de admisión.** Si `OOM_GUARD_MB` no permite la memoria pedida (piso de 512 MB), el arranque se rechaza con `oom_guard` en vez de competir con las aulas.
- **Un preview activo por entrega.** Lo impone un índice parcial único; un fallo o un vencimiento libera el índice y permite reintentar.

## Qué existe hoy (verificado en el código)

### Backend

| Pieza | Archivo |
|---|---|
| Migración | `backend/migrations/00044_review_test_environments.sql` |
| Dominio, constantes y errores | `backend/internal/core/domain/review_environment.go` |
| Servicio | `backend/internal/core/services/review_environment_service.go` |
| Contexto de revisión (snapshot, imagen, receta) | `backend/internal/infrastructure/storage/postgres/review_environment_context.go` |
| Repositorio | `backend/internal/infrastructure/storage/postgres/review_environment_repository.go` |
| Handler | `backend/internal/delivery/http/teacher_review_environment.go` |
| Ruta | `POST /api/v1/teacher/submissions/{id}/test-environment` (`router.go:532`) |
| Tests del servicio | `review_environment_service_test.go` (10 casos: comando, modalidad, OOM, creación aislada, reapertura, contenedor muerto, limpieza, reaper) |
| Tests del handler | `teacher_review_environment_test.go` (403 a alumno, 503 sin orquestador) |

Reglas implementadas en `Start`:

1. Serializa por `submission_id` con un lock en memoria y devuelve `review_environment_starting` si otro pedido del mismo docente está en curso.
2. Si hay un preview `running` con URL y el contenedor vive, renueva `expires_at` y devuelve la misma URL.
3. Si el contenedor murió o el registro quedó a medias, libera volumen y contenedor y crea uno limpio.
4. La creación corre con `context.WithoutCancel` y un techo de 3 minutos, para no dejar volúmenes huérfanos si el navegador cancela.
5. Exige modalidad `IDE_PERSISTENTE`, imagen de plantilla, comando de arranque, puerto válido (por defecto 3000) y snapshot multiarchivo parseable.
6. Escribe el snapshot en el volumen, crea la red `solv-traefik-net` sin ICC y arranca el contenedor con etiquetas `solv.managed`, `solv.review_environment` y `solv.review_environment_id`.
7. Con dominio base configurado publica `Host(review-<id>.<base>)` por Traefik; sin dominio reserva un puerto local. No hay dominio hardcodeado.
8. Espera hasta 45 segundos a que la app escuche. Si no escucha, guarda las últimas 40 líneas del contenedor y limpia todo.
9. `ReapExpired` corre desde `QoSOrchestratorWorker` (`SetReviewEnvironmentReaper`, `qos_worker.go:100,226`), renueva el preview con tráfico y apaga el inactivo.

### Frontend

| Pieza | Archivo |
|---|---|
| Configuración de la práctica | `exercise-editor/steps/step-ide-environment/components/ide-preview-config/` (comando y puerto) |
| Estado del editor | `exercise-editor.store.ts` (`app_run_command`, `app_port`) |
| Servicio | `teacher-grading.service.ts` (`startReviewEnvironment`, señales de carga) |
| Modelo | `teacher.models.ts` (`ReviewEnvironmentAccess`, `app_run_command`, `app_port`) |
| Cabecera | `speed-grader-header.component.html:66-81` (acción visible solo en modo proyecto) |
| Componente | `speed-grader.component.ts:112-113,237` (abre pestaña y guarda `expires_at` para el aviso) |
| Test | `speed-grader.component.spec.ts:312` (pide la app y conserva el vencimiento) |

## Specs

- **S1 — Contrato de arranque.** `POST .../test-environment` responde `{data: {url, expires_at}}` o un error tipado. Alumno recibe 403; nodo sin orquestador recibe 503; modalidad juez recibe 409 `review_environment_requires_ide`; práctica sin comando recibe 409 `app_run_command_missing`.
  - Estado: **cumplido** en handler y servicio.
- **S2 — Aislamiento.** El preview no reutiliza el volumen del workspace del alumno y no altera la entrega; dos docentes sobre la misma entrega comparten preview por la unicidad del índice.
  - Estado: **cumplido** por diseño del volumen y el índice `uq_review_test_environments_active`.
- **S3 — Ciclo de vida.** Un preview sin tráfico se apaga a los 15 minutos y libera contenedor y volumen; uno con tráfico se renueva; un fallo de Docker no borra el registro (queda auditable).
  - Estado: **cumplido** por `ReapExpired` y sus tests.
- **S4 — Configuración de la práctica.** El editor IDE permite declarar comando y puerto; sin comando la UI lo advierte y el arranque del preview falla con mensaje accionable.
  - Estado: **cumplido** en `ide-preview-config` y `ide-preview-config.component.ts`.
- **S5 — Cierre pendiente (lo que falta).**
  - Aplicar `00044` en la base local al reiniciar el backend (idempotente). Hecho el 2026-11-02: el arranque reporta `goose: no migrations to run. current version: 44`.
  - E2E: extender J08 o crear un flujo que publique una plantilla IDE con `app_run_command`, entregue desde el alumno, abra el preview y compruebe que la URL responde y que el contenedor se apaga al vencer.
  - Commit en dos unidades revisables (ver T2).
  - Criterio: `curl` a la URL del preview devuelve 200 con el HTML de la app; `docker ps` no muestra `solv-review-env-*` después del TTL.

## Tasks

- [x] **T1 — S5**: aplicar la migración en la base local y confirmar la tabla `review_test_environments` y sus índices. Verificado el 2026-11-02: 16 columnas y los índices `uq_review_test_environments_active`, `idx_review_test_environments_active_expiry` e `idx_review_test_environments_tenant`.
- [ ] **T2 — S5**: commit en dos unidades. (a) Backend + migración + tests + ADR/contratos: `feat(teacher): ejecutar la entrega IDE en un contenedor efímero`. (b) Frontend + specs + guía de UI: `feat(teacher): abrir la aplicación viva de la entrega desde SpeedGrader`. Cada una compila por separado.
- [ ] **T3 — S5**: e2e del flujo con Docker real, incluido el vencimiento por inactividad.
- [ ] **T4 — S5**: revisar que `docs/API_CONTRACTS.md`, `IDE_PERSISTENTE.md`, `REVISION_POR_MODALIDAD.md` y ADR-061 describan el contrato final (sin cambios de código).
- [ ] **T5 — Futuro, fuera de alcance**: satélites (PostgreSQL de aula) en el contenedor de preview. La configuración `AttachSatellitesNet` existe pero no se activa; requiere decisión de red y de credenciales.

## Dudas abiertas

- **D1 — Reparto del commit.** Recomendación: dos unidades como en T2. Todo junto mezcla migración, Docker y Angular en un solo diff de más de 900 líneas.
- **D2 — E2E con Docker.** Recomendación: extender J08 en vez de crear un flujo nuevo, porque ya publica plantilla IDE y entrega. El vencimiento se puede probar con `SetTimeouts` solo en test de servicio; en e2e basta comprobar que el contenedor existe y que la URL responde.
- **D3 — Traefik frente a puerto local.** En el servidor Asus hay dominio base y el preview sale por `review-<id>.<base>`; en local sale por `localhost:<puerto>`. Confirmar que el certificado wildcard cubre `review-*` antes de anunciar la función en producción.
- **D4 — Satélites.** Recomendación: dejarlo fuera hasta que exista una práctica con BD gestionada que lo necesite. Activar la red sin decidir credenciales mezcla datos de aula con el preview.

## Verificación

```bash
cd backend && gofmt -l internal/core/domain internal/core/services internal/delivery/http internal/infrastructure/storage/postgres
cd backend && go build ./... && go vet ./...
cd backend && go test ./internal/core/services -run ReviewEnvironment -v
cd backend && go test ./internal/delivery/http -run ReviewEnvironment -v
cd frontend && npm run test:ci -- --include='**/speed-grader/**'
cd frontend && npm run lint:styles && CI=true npm run build
```

Estado de la última corrida (sesión del 2026-11-01, **no repetida ahora**): `go build`, `go vet`, los paquetes de dominio/servicios/http/postgres y los 14 tests del SpeedGrader en verde; `npm run build` y `lint:styles` en verde. J07 y J08 pasaron contra el stack local con Docker real. Esa evidencia no cubre el árbol actual sin commit: hay que repetirla antes de commitear.

## Log

- **L1 (2026-11-01):** se implementó el preview efímero con snapshot aislado, receta `app_run_command`/`app_port`, TTL por inactividad y control de admisión por `OOM_GUARD_MB`.
- **L2:** la migración se validó con `ROLLBACK` y quedó aplicada en la base local el 2026-11-02 (versión 44); el reinicio del backend no tuvo migraciones pendientes.
- **L3:** el árbol de trabajo contiene la implementación sin commit (8 archivos nuevos de backend, 1 migración, 1 ADR, 23 archivos modificados entre backend, frontend y docs).
- **L4 — hallazgo:** la ruta es `POST /api/v1/teacher/submissions/{id}/test-environment`, no `.../review-environment`; cualquier cliente nuevo debe usar ese path.
- **L5 — hallazgo:** el preview no tiene heartbeat HTTP. La actividad se mide por `last_rx_bytes`/`last_tx_bytes`; un cambio de UI que espere un ping va a romper la renovación.
