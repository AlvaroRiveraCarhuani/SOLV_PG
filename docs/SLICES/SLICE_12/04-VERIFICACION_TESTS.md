# Verificación de Pruebas — SLICE 12: Experiencia Estudiante y Juez en Tiempo Real

## 1. Pruebas de Integración Backend (Live PostgreSQL)

Los tests de integración del incremento backend de este Slice se encuentran implementados en `backend/tests/integration/lote1_test.go` y fueron ejecutados contra una instancia viva de PostgreSQL 18 en Docker.

### Comando de Ejecución
```bash
DATABASE_URL="postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable" go test -v -run TestLote1 ./tests/integration/...
```

### Resultados de Ejecución
```text
=== RUN   TestLote1_GetMeEndpoint
2026/09/02 10:04:11 Running initial database migrations...
2026/09/02 10:04:12 Initial migrations completed successfully.
--- PASS: TestLote1_GetMeEndpoint (0.14s)
=== RUN   TestLote1_GetDueAssignmentsEndpoint
2026/09/02 10:04:12 Running initial database migrations...
2026/09/02 10:04:12 Initial migrations completed successfully.
--- PASS: TestLote1_GetDueAssignmentsEndpoint (0.14s)
=== RUN   TestLote1_PauseWorkspaceEndpoint
2026/09/02 10:04:12 Running initial database migrations...
2026/09/02 10:04:12 Initial migrations completed successfully.
--- PASS: TestLote1_PauseWorkspaceEndpoint (0.12s)
=== RUN   TestLote1_WebSocketHubAndEvaluation
2026/09/02 10:04:12 Running initial database migrations...
2026/09/02 10:04:12 Initial migrations completed successfully.
--- PASS: TestLote1_WebSocketHubAndEvaluation (0.13s)
PASS
ok  	solv-backend/tests/integration	0.599s
```

## 2. Cobertura de Pruebas Verificada

1. **`TestLote1_GetMeEndpoint`**:
   - Petición sin header `X-User-Id` -> `401 Unauthorized`.
   - Petición con usuario válido en DB -> `200 OK` con DTO completo (`id`, `email`, `role`, `full_name`, `tenant_id`).

2. **`TestLote1_GetDueAssignmentsEndpoint`**:
   - Petición sin credenciales de estudiante -> `401 Unauthorized`.
   - Consulta con inscripciones reales -> `200 OK` con entregas filtradas por `due_date > NOW()` y ordenadas por fecha.

3. **`TestLote1_PauseWorkspaceEndpoint`**:
   - Petición de usuario no dueño -> `403 Forbidden`.
   - Petición del estudiante propietario -> `200 OK`, contenedor detenido y estado `hibernated`.

4. **`TestLote1_WebSocketHubAndEvaluation`**:
   - Handshake y autenticación -> Evento `CONNECTION_ESTABLISHED`.
   - Disparo de envío de solución vía HTTP -> Recepción en tiempo real del evento `EVALUATION_COMPLETED` por WebSocket.

## 3. Estado de la Verificación
- **Backend:** 100% verificado contra PostgreSQL real.
- **Frontend:** Pendiente de implementación y validación E2E.
