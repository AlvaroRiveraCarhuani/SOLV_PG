import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { buildBackendGraph } from '../lib/backend-graph.mjs';
import { buildFrontendGraph } from '../lib/frontend-graph.mjs';
import { loadPolicy, matchingEntries } from '../lib/policy.mjs';
import { createPlan } from '../lib/select.mjs';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(HERE, '..', '..', '..');

const policy = loadPolicy(REPO_ROOT);
const backendGraph = buildBackendGraph(REPO_ROOT);
const frontendGraph = buildFrontendGraph(REPO_ROOT);
const trackedFiles = execFileSync('git', ['-C', REPO_ROOT, 'ls-files'], { encoding: 'utf8' })
  .split(/\r?\n/)
  .filter(Boolean);

function planFor(changes, options = {}) {
  return createPlan({ policy, changes, backendGraph, frontendGraph, options });
}

function planForFile(filePath, options = {}) {
  return planFor([{ status: 'M', path: filePath }], options);
}

test('la política se valida y declara todas las suites', () => {
  assert.equal(policy.version, 4);
  for (const suite of [
    'gates',
    'asset-validation',
    'backend-unit',
    'backend-integration',
    'backend-docker',
    'frontend-unit',
    'frontend-e2e',
    'migrations',
    'build',
    'security'
  ]) {
    assert.ok(policy.suites[suite], `falta la suite ${suite}`);
  }
});

test('todas las rutas versionadas de código están clasificadas', () => {
  const unclassified = trackedFiles.filter((filePath) => {
    if (!filePath.startsWith('backend/') && !filePath.startsWith('frontend/')) return false;
    if (filePath.startsWith('frontend/public/')) return false;
    if (filePath.startsWith('frontend/e2e/test-results/')) return false;
    const neutral = matchingEntries([policy.neutral], filePath).length > 0;
    const area = matchingEntries(policy.areas, filePath).length > 0;
    const critical = matchingEntries(policy.critical, filePath).length > 0;
    const global = matchingEntries(policy.globalTriggers, filePath).length > 0;
    return !neutral && !area && !critical && !global;
  });
  assert.deepEqual(unclassified, [], 'hay rutas sin regla de impacto');
});

test('todas las áreas y reglas críticas alcanzan al menos una ruta real', () => {
  const unmatched = [...policy.areas, ...policy.critical].filter(
    (entry) => !trackedFiles.some((filePath) => matchingEntries([entry], filePath).length > 0)
  );
  assert.deepEqual(
    unmatched.map((entry) => entry.id),
    [],
    'hay reglas que no alcanzan ninguna ruta y no protegen nada'
  );
});

test('un icono activa solo validación de recurso y no dispara backend, Docker, E2E ni build', () => {
  const plan = planForFile('frontend/public/tech-icons/python.svg');
  assert.equal(plan.mode, 'targeted');
  assert.equal(plan.suites['backend-unit'].selected, false);
  assert.equal(plan.suites['backend-integration'].selected, false);
  assert.equal(plan.suites['backend-docker'].selected, false);
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.suites['frontend-e2e'].selected, false);
  assert.equal(plan.suites['asset-validation'].selected, true);
  assert.equal(plan.suites.build.selected, false);
  assert.equal(plan.level, 'L0');
});

test('un componente Angular ejecuta su spec y las specs que lo consumen', () => {
  const plan = planForFile('frontend/src/app/shared/components/kpi-card/kpi-card.component.ts');
  assert.equal(plan.suites['backend-unit'].selected, false);
  assert.equal(plan.suites['backend-integration'].selected, false);
  assert.equal(plan.suites['frontend-unit'].selected, true);
  assert.ok(
    plan.suites['frontend-unit'].specs.includes(
      'src/app/shared/components/kpi-card/kpi-card.component.spec.ts'
    )
  );
  assert.equal(plan.suites['frontend-e2e'].selected, false);
});

test('un cambio de bootstrap o providers ejecuta todas las specs y todos los flujos E2E', () => {
  for (const filePath of ['frontend/src/app/app.config.ts', 'frontend/src/main.ts']) {
    const plan = planForFile(filePath);
    assert.equal(plan.mode, 'targeted', filePath);
    assert.equal(plan.suites['frontend-unit'].full, true, filePath);
    assert.equal(plan.suites['frontend-unit'].specs.length, frontendGraph.allSpecs.length, filePath);
    assert.equal(plan.suites['frontend-e2e'].specs.length, 24, filePath);
    assert.equal(plan.suites['backend-unit'].selected, false, filePath);
  }
});

test('un HTML de arranque compila el shell y ejecuta un flujo de humo, no todo el frontend', () => {
  const plan = planForFile('frontend/src/index.html');
  assert.equal(plan.level, 'L2');
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.suites.build.selected, true);
  assert.equal(plan.suites.build.frontend, true);
  assert.deepEqual(
    plan.suites['frontend-e2e'].specs.map((item) => item.spec),
    ['e2e/flows/01-core-lifecycle.spec.mjs']
  );
});

test('la política no repite globs entre reglas', () => {
  const seen = new Map();
  for (const entry of [...policy.globalTriggers, ...policy.critical, ...policy.areas]) {
    for (const glob of entry.globs) {
      assert.equal(seen.has(glob), false, `glob duplicado: ${glob} en ${seen.get(glob)} y ${entry.id}`);
      seen.set(glob, entry.id);
    }
  }
});

test('permutar el orden de las áreas no cambia suites, áreas ni flujos E2E', () => {
  const change = {
    status: 'M',
    path: 'frontend/src/app/app.routes.ts'
  };
  const baseline = planFor([change]);
  assert.ok(baseline.changed[0].areas.some((area) => area.id === 'frontend-routes'));
  assert.ok(baseline.suites['frontend-e2e'].specs.length > 0);

  const permutations = [
    [...policy.areas].reverse(),
    [...policy.areas].sort((left, right) => left.id.localeCompare(right.id)),
    [...policy.areas].sort((left, right) => right.id.localeCompare(left.id))
  ];
  for (const areas of permutations) {
    const plan = createPlan({
      policy: { ...policy, areas },
      changes: [change],
      backendGraph,
      frontendGraph
    });
    assert.deepEqual(
      plan.suites,
      baseline.suites,
      'las suites cambiaron al permutar las áreas'
    );
    assert.deepEqual(
      plan.changed[0].areas.map((area) => area.id).sort(),
      baseline.changed[0].areas.map((area) => area.id).sort(),
      'las áreas cambiaron al permutar las reglas'
    );
    assert.deepEqual(
      plan.suites['frontend-e2e'].specs.map((item) => `${item.config}:${item.spec}`).sort(),
      baseline.suites['frontend-e2e'].specs.map((item) => `${item.config}:${item.spec}`).sort(),
      'los flujos E2E cambiaron al permutar las reglas'
    );
  }
});

test('un cambio de rutas Angular ejecuta todas las specs y todos los flujos E2E', () => {
  const plan = planForFile('frontend/src/app/app.routes.ts');
  assert.equal(plan.mode, 'targeted');
  assert.equal(plan.suites['frontend-unit'].full, true);
  assert.equal(plan.suites['frontend-unit'].specs.length, frontendGraph.allSpecs.length);
  assert.equal(plan.suites['frontend-e2e'].selected, true);
  assert.ok(plan.suites['frontend-e2e'].specs.length > 20);
  assert.equal(plan.suites['backend-unit'].selected, false);
  assert.equal(plan.suites['backend-integration'].selected, false);
});

test('un pipe compartido ejecuta todas las specs que lo importan', () => {
  const plan = planForFile('frontend/src/app/shared/pipes/date-text.pipe.ts');
  assert.ok(plan.suites['frontend-unit'].specs.length > 10);
  assert.ok(
    plan.suites['frontend-unit'].specs.includes('src/app/shared/pipes/date-text.pipe.spec.ts')
  );
});

test('un servicio de Go ejecuta su paquete y sus dependientes, más integración', () => {
  const plan = planForFile('backend/internal/core/services/submission_service.go');
  assert.equal(plan.suites['backend-unit'].selected, true);
  assert.ok(plan.suites['backend-unit'].packages.includes('./internal/core/services'));
  assert.equal(plan.suites['backend-integration'].selected, true);
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.mode, 'targeted');
});

test('un cambio de dominio compartido activa backend y frontend completos', () => {
  const plan = planForFile('backend/internal/core/domain/user.go');
  assert.equal(plan.mode, 'targeted');
  assert.equal(plan.suites['backend-unit'].full, true);
  assert.equal(plan.suites['frontend-unit'].full, true);
  assert.equal(plan.suites['backend-integration'].selected, true);
  assert.ok(plan.criticalAreas.includes('shared-contracts'));
  assert.equal(plan.suites.security.selected, true);
});

test('autenticación de backend activa cobertura crítica sin correr todo el frontend', () => {
  const plan = planForFile('backend/internal/core/services/auth_service.go');
  assert.ok(plan.criticalAreas.includes('auth-backend'));
  assert.equal(plan.suites['backend-unit'].full, true);
  assert.equal(plan.suites['backend-integration'].selected, true);
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.suites.security.selected, true);
  assert.ok(plan.suites['frontend-e2e'].specs.length >= 2);
});

test('autenticación de frontend activa todas las specs sin correr todo el backend', () => {
  const plan = planForFile('frontend/src/app/core/guards/auth.guard.ts');
  assert.ok(plan.criticalAreas.includes('auth-frontend'));
  assert.equal(plan.suites['frontend-unit'].full, true);
  assert.equal(plan.suites['backend-unit'].selected, false);
  assert.equal(plan.suites['backend-integration'].selected, false);
  assert.equal(plan.suites.security.selected, true);
});

test('aislamiento de tenants activa cobertura crítica de backend', () => {
  const plan = planForFile('backend/internal/delivery/http/middleware/tenant_middleware.go');
  assert.ok(plan.criticalAreas.includes('tenancy'));
  assert.equal(plan.suites['backend-unit'].full, true);
  assert.equal(plan.suites['backend-integration'].selected, true);
  assert.ok(plan.suites['frontend-e2e'].specs.some((item) => item.spec.includes('J03')));
});

test('ejecución de contenedores activa la suite Docker y los flujos de seguridad', () => {
  const plan = planForFile('backend/internal/infrastructure/docker/client.go');
  assert.ok(plan.criticalAreas.includes('container-execution'));
  assert.equal(plan.suites['backend-docker'].selected, true);
  const specs = plan.suites['frontend-e2e'].specs.map((item) => item.spec);
  assert.ok(specs.some((spec) => spec.includes('J04')));
  assert.ok(specs.some((spec) => spec.includes('08-concurrency')));
});

test('una migración activa la cadena de migraciones y todos los flujos E2E', () => {
  const plan = planForFile('backend/migrations/00051_submission_verdict_origin.sql');
  assert.ok(plan.criticalAreas.includes('migrations'));
  assert.equal(plan.suites.migrations.selected, true);
  assert.equal(plan.suites['frontend-e2e'].specs.length, 24);
});

test('documentación sola queda neutral y no compila ni ejecuta pruebas', () => {
  const plan = planForFile('docs/ARQUITECTURA/STORAGE_STRATEGY.md');
  assert.equal(plan.mode, 'neutral');
  assert.equal(plan.suites['backend-unit'].selected, false);
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.suites.build.selected, false);
  assert.equal(plan.suites['frontend-e2e'].selected, false);
});

test('una ruta desconocida activa la estrategia conservadora completa', () => {
  const plan = planForFile('scripts/nueva-herramienta.bin');
  assert.equal(plan.mode, 'full');
  assert.deepEqual(plan.unknown, ['scripts/nueva-herramienta.bin']);
  assert.equal(plan.suites['backend-unit'].full, true);
  assert.equal(plan.suites['backend-integration'].selected, true);
  assert.equal(plan.suites['backend-docker'].selected, true);
  assert.equal(plan.suites['frontend-unit'].full, true);
  assert.equal(plan.suites['frontend-e2e'].specs.length, 24);
  assert.equal(plan.suites.migrations.selected, true);
  assert.equal(plan.suites.security.selected, true);
});

test('un diff vacío no se interpreta como "sin pruebas" y ejecuta todo', () => {
  const plan = planFor([]);
  assert.equal(plan.mode, 'full');
  assert.equal(plan.suites['frontend-e2e'].specs.length, 24);
});

test('superar el límite de archivos activa la ejecución completa', () => {
  const changes = Array.from({ length: policy.limits.maxChangedFiles + 1 }, (_, index) => ({
    status: 'M',
    path: `frontend/src/app/features/admin/generated-${index}.ts`
  }));
  const plan = planFor(changes);
  assert.equal(plan.mode, 'full');
  assert.match(plan.reason, /superan el límite/);
});

test('un test unitario de Go no arrastra integración ni frontend', () => {
  const plan = planForFile('backend/internal/core/services/submission_service_test.go');
  assert.equal(plan.suites['backend-unit'].selected, true);
  assert.ok(plan.suites['backend-unit'].packages.includes('./internal/core/services'));
  assert.equal(plan.suites['backend-integration'].selected, false);
  assert.equal(plan.suites['frontend-unit'].selected, false);
});

test('un test de integración ejecuta la suite de integración', () => {
  const plan = planForFile('backend/tests/integration/subject_section_test.go');
  assert.equal(plan.suites['backend-integration'].selected, true);
  assert.equal(plan.suites['backend-unit'].selected, false);
});

test('una spec de E2E se ejecuta a sí misma y no compila la aplicación', () => {
  const spec = 'frontend/e2e/flows/02-medium/J04-ast-security-barrier.spec.mjs';
  const plan = planForFile(spec);
  assert.equal(plan.suites.build.selected, false);
  assert.deepEqual(
    plan.suites['frontend-e2e'].specs.map((item) => item.spec),
    ['e2e/flows/02-medium/J04-ast-security-barrier.spec.mjs']
  );
});

test('un fixture de E2E selecciona todos los flujos', () => {
  const plan = planForFile('frontend/e2e/fixtures/api.fixture.mjs');
  assert.equal(plan.suites['frontend-e2e'].specs.length, 24);
});

test('los estilos globales ejecutan todo el frontend y la prueba de tipografía', () => {
  const plan = planForFile('frontend/src/styles/_semantic.scss');
  assert.equal(plan.suites['frontend-unit'].full, true);
  assert.deepEqual(
    plan.suites['frontend-e2e'].specs.map((item) => item.spec),
    ['e2e/typography.spec.mjs']
  );
  assert.equal(plan.suites['frontend-e2e'].specs[0].config, 'playwright.config.mjs');
});

test('la configuración global del frontend ejecuta todo el frontend pero no el backend', () => {
  const plan = planForFile('frontend/angular.json');
  assert.equal(plan.mode, 'full');
  assert.equal(plan.suites['frontend-unit'].full, true);
  assert.equal(plan.suites['backend-unit'].selected, false);
  assert.equal(plan.suites['frontend-e2e'].specs.length, 24);
});

test('una dependencia del frontend recompleta y ejecuta todas las specs sin tocar el backend', () => {
  const plan = planForFile('frontend/package.json');
  assert.equal(plan.mode, 'full');
  assert.equal(plan.suites['frontend-unit'].full, true);
  assert.equal(plan.suites['frontend-unit'].specs.length, frontendGraph.allSpecs.length);
  assert.equal(plan.suites['backend-unit'].selected, false);
  assert.equal(plan.suites['build'].selected, true);
});

test('las dependencias del backend ejecutan todo el backend y todos los E2E', () => {
  const plan = planForFile('backend/go.mod');
  assert.equal(plan.mode, 'full');
  assert.equal(plan.suites['backend-unit'].full, true);
  assert.equal(plan.suites['backend-docker'].selected, true);
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.suites['frontend-e2e'].specs.length, 24);
});

test('eliminar un archivo de código activa alcance completo de su lado', () => {
  const plan = planFor([
    { status: 'D', path: 'frontend/src/app/shared/components/obsoleto/obsoleto.component.ts' }
  ]);
  assert.equal(plan.mode, 'full');
  assert.equal(plan.suites['frontend-unit'].full, true);
  assert.equal(plan.suites['backend-unit'].selected, false);
});

test('renombrar un archivo de código activa alcance completo de su lado', () => {
  const plan = planFor([
    {
      status: 'R100',
      oldPath: 'backend/internal/core/services/viejo.go',
      path: 'backend/internal/core/services/nuevo.go'
    }
  ]);
  assert.equal(plan.mode, 'full');
  assert.equal(plan.suites['backend-unit'].full, true);
  assert.equal(plan.suites['backend-integration'].selected, true);
});

test('eliminar un icono no activa pruebas', () => {
  const plan = planFor([
    { status: 'D', path: 'frontend/public/tech-icons/legacy.svg' }
  ]);
  assert.equal(plan.mode, 'targeted');
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.suites['frontend-e2e'].selected, false);
});

test('el workflow de CI y la política se protegen con ejecución completa', () => {
  for (const filePath of ['.github/workflows/ci.yml', 'tools/test-impact/policy.json']) {
    const plan = planForFile(filePath);
    assert.equal(plan.mode, 'full', filePath);
    assert.equal(plan.suites['frontend-e2e'].specs.length, 24, filePath);
  }
});

test('un archivo de frontend sin spec queda reportado como no cubierto', () => {
  const plan = planForFile('frontend/src/app/features/admin/sin-cobertura/sin-cobertura.ts');
  assert.deepEqual(plan.uncovered.frontend, [
    'frontend/src/app/features/admin/sin-cobertura/sin-cobertura.ts'
  ]);
  assert.equal(plan.suites['frontend-unit'].selected, false);
});

test('un HTML decorativo sin dueño no activa todas las specs arbitrariamente', () => {
  const plan = planForFile('frontend/src/app/features/admin/huerfano/huerfano.html');
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.suites.build.selected, false);
  assert.deepEqual(plan.uncovered.frontend, []);
});

test('el evento schedule ejecuta todo aunque el diff sea pequeño', () => {
  const plan = planForFile('frontend/public/tech-icons/python.svg', { event: 'schedule' });
  assert.equal(plan.mode, 'full');
  assert.equal(plan.suites['frontend-e2e'].specs.length, 24);
});
