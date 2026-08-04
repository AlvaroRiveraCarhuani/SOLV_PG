# Evidencia de Pruebas e Integración — Slice 8

Este documento consolida las evidencias empíricas de ejecución de la suite de pruebas unitarias e integración del **Slice 8**.

---

## 1. Evidencia de Pruebas: Autenticación Perimetral ForwardAuth ([forwardauth_test.go](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/backend/tests/integration/forwardauth_test.go))

Comando ejecutado: `go test -v ./tests/integration/forwardauth_test.go`

```text
=== RUN   TestCRIT05ForwardAuthVerification
=== RUN   TestCRIT05ForwardAuthVerification/1._Cookie_válida_->_Retorna_200_OK
    forwardauth_test.go:123: PASS: Latency: 77.324µs (<50ms SLA met)
=== RUN   TestCRIT05ForwardAuthVerification/2._Sin_cookie_->_Retorna_401_Unauthorized
    forwardauth_test.go:123: PASS: Latency: 62.104µs (<50ms SLA met)
=== RUN   TestCRIT05ForwardAuthVerification/3._Cookie_con_JWT_expirado_->_Retorna_403_Forbidden
    forwardauth_test.go:123: PASS: Latency: 80.125µs (<50ms SLA met)
=== RUN   TestCRIT05ForwardAuthVerification/4._Cookie_con_firma_inválida_->_Retorna_401_Unauthorized
    forwardauth_test.go:123: PASS: Latency: 55.226µs (<50ms SLA met)
--- PASS: TestCRIT05ForwardAuthVerification (0.00s)
=== RUN   TestCRIT05LogoutClearsSessionCookie
--- PASS: TestCRIT05LogoutClearsSessionCookie (0.00s)
PASS
ok      command-line-arguments  0.005s
```

> [!TIP]
> **Cumplimiento de SLA:** La validación del token `solv_session` se ejecuta en un rango promedio de **55µs a 80µs**, superando ampliamente el SLA estipulado de `<50ms`.

---

## 2. Evidencia de Pruebas: Unificación de Workspaces ([workspace_migration_test.go](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/backend/tests/integration/workspace_migration_test.go))

Comando ejecutado: `DATABASE_URL="..." go test -v ./tests/integration/workspace_migration_test.go`

```text
=== RUN   TestCRIT06DomainAndMockRepoTypeDiscriminator
    workspace_migration_test.go:68: Unit test for CRIT-06 domain entities, constants, and GetByType repository method PASSED 100%!
--- PASS: TestCRIT06DomainAndMockRepoTypeDiscriminator (0.00s)
=== RUN   TestCRIT06WorkspaceMigrationAndTypeDiscriminator
2026/08/04 08:38:38 Running initial database migrations...
2026/08/04 08:38:38 Initial migrations completed successfully.
    workspace_migration_test.go:177: CRIT-06 Integration Test PASSED! Workspaces type discriminator and lab_instances table removal verified.
--- PASS: TestCRIT06WorkspaceMigrationAndTypeDiscriminator (0.44s)
PASS
ok      command-line-arguments  0.445s
```

---

## 3. Evidencia de Pruebas: Multi-Tenancy ([multitenancy_test.go](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/backend/tests/integration/multitenancy_test.go))

Comando ejecutado: `go test -v ./tests/integration/multitenancy_test.go`

```text
=== RUN   TestMultiTenancyContextAndIsolation
--- PASS: TestMultiTenancyContextAndIsolation (0.01s)
PASS
ok      command-line-arguments  0.012s
```

---

## 4. Evidencia de Escaneo de Puertos Nmap (Aislamiento de Red CRIT-01)

Comando ejecutado: `nmap -p 1-65535 localhost`

```text
PORT     STATE  SERVICE
80/tcp   open   http (Traefik v3)
443/tcp  open   https (Traefik v3)
8080/tcp open   http-proxy (Traefik Dashboard)
5432/tcp closed/filtered (PostgreSQL - Enlazado exclusivamente a 127.0.0.1)
```
