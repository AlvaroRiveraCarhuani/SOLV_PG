import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { buildBackendGraph } from '../lib/backend-graph.mjs';
import { buildFrontendGraph } from '../lib/frontend-graph.mjs';
import { loadPolicy } from '../lib/policy.mjs';
import { createPlan } from '../lib/select.mjs';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const policy = loadPolicy(REPO_ROOT);
const backendGraph = buildBackendGraph(REPO_ROOT);
const frontendGraph = buildFrontendGraph(REPO_ROOT);

function planFor(filePath) {
  return createPlan({
    policy,
    changes: [{ status: 'M', path: filePath }],
    backendGraph,
    frontendGraph,
    options: { event: 'local' }
  });
}

function assertTemplateCovered(filePath) {
  const plan = planFor(filePath);
  assert.equal(plan.suites.build.selected, true, `${filePath}: falta compilación`);
  assert.equal(plan.suites.build.frontend, true, `${filePath}: la compilación debe ser de frontend`);
  assert.equal(plan.suites['frontend-unit'].full, false, `${filePath}: no debe ejecutar todas las specs`);
  assert.ok(
    plan.uncovered.frontend.includes(filePath),
    `${filePath}: la falta de spec debe quedar declarada`
  );
  assert.ok(plan.level !== 'L0', `${filePath}: una plantilla funcional sin spec no puede ser L0`);
}

test('una plantilla Angular con lógica y sin spec se valida compilando, no con 128 specs', () => {
  const filePath =
    'frontend/src/app/features/admin/environment-requests/admin-environment-requests.component.html';
  assertTemplateCovered(filePath);
  const plan = planFor(filePath);
  assert.equal(plan.level, 'L1');
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.suites['frontend-e2e'].selected, false);
});

test('las plantillas funcionales sin spec del inventario quedan cubiertas por build', () => {
  for (const filePath of [
    'frontend/src/app/features/dev/motion-gallery/motion-gallery.component.html',
    'frontend/src/app/features/teacher/courses/practice-detail/practice-detail.component.html',
    'frontend/src/app/shared/components/hotkeys-modal/hotkeys-modal.component.html'
  ]) {
    assertTemplateCovered(filePath);
  }
});

test('el HTML del login conserva L3, seguridad y compilación de frontend', () => {
  const filePath = 'frontend/src/app/features/auth/login/login.component.html';
  const plan = planFor(filePath);
  assert.equal(plan.level, 'L3');
  assert.ok(plan.criticalAreas.includes('auth-frontend'));
  assert.equal(plan.suites.security.selected, true);
  assert.equal(plan.suites.build.selected, true);
  assert.equal(plan.suites.build.frontend, true);
  assert.equal(plan.suites['frontend-unit'].full, true);
});

test('un HTML decorativo sin lógica no dispara todas las specs ni build', () => {
  const filePath = 'frontend/src/app/features/admin/legacy/legacy.html';
  const plan = planFor(filePath);
  assert.equal(plan.level, 'L0');
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.equal(plan.suites.build.selected, false);
  assert.deepEqual(plan.uncovered.frontend, []);
});

test('un HTML con spec conserva la spec y no añade compilación', () => {
  const filePath = 'frontend/src/app/features/admin/students/admin-students.component.html';
  const plan = planFor(filePath);
  assert.equal(plan.level, 'L0');
  assert.equal(plan.suites['frontend-unit'].selected, true);
  assert.equal(plan.suites['frontend-unit'].specs.length, 1);
  assert.equal(plan.suites.build.selected, false);
});

test('index.html usa la regla de shell: build y un flujo de humo, sin todo el frontend', () => {
  const plan = planFor('frontend/src/index.html');
  assert.equal(plan.level, 'L2');
  assert.equal(plan.suites.build.selected, true);
  assert.equal(plan.suites.build.frontend, true);
  assert.equal(plan.suites['frontend-unit'].selected, false);
  assert.deepEqual(
    plan.suites['frontend-e2e'].specs.map((item) => item.spec),
    ['e2e/flows/01-core-lifecycle.spec.mjs']
  );
  assert.ok(plan.suites['frontend-e2e'].specs.length < frontendGraph.allSpecs.length);
});

test('el bootstrap Angular usa los archivos reales y exige validación completa', () => {
  for (const filePath of [
    'frontend/src/app/app.ts',
    'frontend/src/app/app.html',
    'frontend/src/app/app.scss'
  ]) {
    const plan = planFor(filePath);
    assert.equal(plan.level, 'L3', filePath);
    assert.equal(plan.suites['frontend-unit'].selected, true, filePath);
    assert.equal(plan.suites['frontend-unit'].full, true, filePath);
    assert.equal(plan.suites.build.selected, true, filePath);
    assert.equal(plan.suites.build.frontend, true, filePath);
    assert.ok(plan.suites['frontend-e2e'].specs.length > 1, filePath);
  }
});

test('una plantilla con spec y dependencia compartida sigue resolviendo por grafo', () => {
  const plan = planFor('frontend/src/app/shared/pipes/date-text.pipe.ts');
  assert.ok(plan.suites['frontend-unit'].specs.length > 10);
  assert.ok(plan.suites['frontend-unit'].specs.includes('src/app/shared/pipes/date-text.pipe.spec.ts'));
});
