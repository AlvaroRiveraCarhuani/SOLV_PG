# Verificación de Pruebas - SLICE 5

## 1. Pruebas de Integración y Estrés (`qos_test.go`)
```bash
cd backend
go test -v ./tests/integration/qos_test.go ./tests/integration/workspace_test.go
```

## 2. Resultados de la Suite de Pruebas
```text
=== RUN   TestHostAdmissionControl15PercentMargin
    qos_test.go:59: PASS: Host Admission Control correctly rejected request when host free RAM < 15%
--- PASS: TestHostAdmissionControl15PercentMargin (0.00s)

=== RUN   TestOOMKilledThreeStrikePenalty
    qos_test.go:98: PASS: 3-Strike OOMKilled penalty correctly enforced 5-minute cooldown period
--- PASS: TestOOMKilledThreeStrikePenalty (1.76s)

=== RUN   TestQoSAutoBurstingAndDualValidation
    qos_test.go:156: PASS: QoS Auto-Bursting worker and Dual-Validation Hibernation cycle verified successfully! Initial RAM: 256 MB
--- PASS: TestQoSAutoBurstingAndDualValidation (3.66s)

=== RUN   TestWorkspaceServiceStartAndIdempotency
    workspace_test.go:196: Slice 4 Integration Test PASSED! Workspace ID: bd430809-b45f-4be0-95a4-472da081ec11, Access URL: http://bd430809-b45f-4be0-95a4-472da081ec11.solv.local
--- PASS: TestWorkspaceServiceStartAndIdempotency (0.75s)

PASS
ok      command-line-arguments  6.182s
```

## 3. Resumen de Cifras Medidas en Hardware Físico Asus
* **RAM Total Host:** 11,852 MB
* **RAM Usada Reposo (OS + Docker + Postgres + Traefik):** 408 MB
* **Freno de Seguridad (Circuit-Breaker):** 1,000 MB
* **OOM_GUARD_MB Calculado:** **1,408 MB**
* **Memoria Asignable Neto:** **10,444 MB** (~10.4 GB)
* **Densidad Máxima en Reposo (256MB/alumno):** Hasta **40 alumnos simultáneos**
* **Densidad Máxima en Pico (2048MB/alumno):** Hasta **5 alumnos simultáneos a máxima carga**
* Veredicto General: **PASS**.
