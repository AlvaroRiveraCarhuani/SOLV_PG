# Selector de pruebas por impacto

La política vive en [`policy.json`](./policy.json) y es la única fuente de reglas.
Este directorio solo la interpreta.

```bash
node tools/test-impact/cli.mjs validate
node tools/test-impact/cli.mjs plan --base origin/main --working-tree --format text
node tools/test-impact/cli.mjs explain backend/internal/core/services/auth_service.go
node tools/test-impact/cli.mjs report --file impact-report.json --plan impact-plan.json

# Todas las suites obligatorias del plan.
node tools/test-impact/cli.mjs plan --base origin/main --working-tree --output impact-plan.json
node tools/test-impact/run.mjs --all --plan impact-plan.json --report impact-report.json

# Una suite concreta: la ejecución parcial debe declararse.
node tools/test-impact/run.mjs --partial --suite frontend-unit --plan impact-plan.json \
  --report impact-report-frontend-unit.json

# Job condicional: si la suite no fue seleccionada, no falla y no valida.
node tools/test-impact/run.mjs --partial --suite build --if-selected --plan impact-plan.json \
  --report impact-report-build.json

# Agregado de informes parciales de CI.
node tools/test-impact/cli.mjs aggregate --plan impact-plan.json --dir impact-reports

# Benchmark reproducible de quince escenarios aislados.
node tools/test-impact/bench.mjs
```

- `lib/` — lectura de la política, grafos de Go y Angular, diff de git, motor de
  selección, integridad del plan, contrato de informes, caché y validación de
  recursos.
- `cli.mjs` — plan, explicación, reporte, validación, agregación y salidas para
  GitHub Actions.
- `run.mjs` — ejecuta solo las suites seleccionadas. `--all` significa «todas
  las del plan», no «todas las suites de la política». `--suite` exige
  `--partial`; sin esa marca una suite aislada no se presenta como validación.
- `lib/plan-integrity.mjs` — reconstruye el diff y las obligaciones desde la
  política vigente. Un plan con suites ocultas, inventario incompleto, nivel
  degradado, regla crítica eliminada o digest de otra política se rechaza.
- `--report` escribe un JSON con nivel, identidad del plan, estado y tiempo por
  suite, `runId`, `jobId` y un recibo HMAC. Los estados son `executed`,
  `cached`, `simulated`, `skipped`, `pending` y `failed`; `validated` solo es
  verdadero para una ejecución completa real.
- `cli report` exige `--plan` y verifica la firma del informe y su
  correspondencia con el plan. `aggregate` verifica además ejecución, job y
  duplicados. En CI la clave sale de `SOLV_TEST_RECEIPT_KEY`; en local se crea
  `tools/test-impact/.cache/receipt.key` con permisos restrictivos.
- `SOLV_TEST_RECEIPT_KEY` debe existir como secreto del repositorio. Sin ese
  secreto los jobs de CI fallan de forma explícita.
- Un plan `source.kind: files` es sintético: se ejecuta, pero no valida el diff
  real del árbol de trabajo. Un `--all` sintético termina con código 2 y no
  cuenta como cobertura en la agregación de CI.
- La caché es opt-in y solo cubre suites deterministas. Su clave incluye
  política, runner, comando, toolchain, entorno relevante, HEAD, diff y archivos
  sin seguimiento; la identidad efímera de la ejecución no forma parte de la
  clave. Cada entrada lleva una firma. No se usa para integración,
  Docker, migraciones, E2E, build ni seguridad.
- `bench.mjs` genera los quince planes de referencia con el selector real. Los
  pesos de coste son orientativos y no son mediciones de suites.
- `test-db.sh` — base PostgreSQL descartable para integración y migraciones;
  exige `TEST_DB_HOST`, `TEST_DB_PORT`, `TEST_DB_USER` y `TEST_DB_PASSWORD`, y
  valida el destino antes de crear la base.
- `cleanup-docker.sh` — elimina solo los recursos del `SOLV_TEST_RUN_ID` actual,
  filtrados por la etiqueta `solv.test.run`; falla si quedan recursos. Con
  `--stale` retira residuos etiquetados de runs anteriores; no usa `prune` ni
  limpieza global.
- `test/` — regresiones del selector, integridad, contrato de ejecución,
  Angular, WebSockets, caché, recursos y documentación.

El plan es la interfaz estable para agentes: incluye `level`, `levelReasons`,
`decisions.suites`, `coverageGaps` y `uncovered`. Un agente no debe ejecutar
suites fuera del plan sin dejar registro de la excepción, ni declarar validación
completa con un informe parcial, simulado o sintético.

La explicación funcional, las suites, la cobertura crítica, el benchmark y las
limitaciones están en
[`docs/GOBERNANZA/TESTING_POR_IMPACTO.md`](../../docs/GOBERNANZA/TESTING_POR_IMPACTO.md).
