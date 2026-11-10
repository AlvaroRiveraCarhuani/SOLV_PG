import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { buildBackendGraph } from '../lib/backend-graph.mjs';
import { buildFrontendGraph } from '../lib/frontend-graph.mjs';
import { loadPolicy, matchingEntries } from '../lib/policy.mjs';
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

const BACKEND_FILES = [
  'backend/internal/delivery/http/websocket_handler.go',
  'backend/internal/delivery/http/websocket_hub.go',
  'backend/internal/delivery/http/teacher_ephemeral.go',
  'backend/internal/delivery/http/router.go',
  'backend/internal/delivery/http/submission_handler.go',
  'backend/internal/delivery/http/evaluation_handler.go'
];

test('los handlers WebSocket del backend activan la regla crítica y seguridad', () => {
  for (const filePath of BACKEND_FILES) {
    const plan = planFor(filePath);
    assert.ok(plan.criticalAreas.includes('websocket-backend'), filePath);
    assert.equal(plan.level, 'L3', filePath);
    assert.equal(plan.suites.security.selected, true, filePath);
    assert.equal(plan.suites['backend-unit'].full, true, filePath);
    assert.equal(plan.suites['backend-integration'].selected, true, filePath);
    assert.equal(plan.suites.build.backend, true, filePath);
    const specs = plan.suites['frontend-e2e'].specs.map((item) => item.spec);
    assert.ok(specs.some((spec) => spec.includes('J08')), filePath);
    assert.ok(specs.some((spec) => spec.includes('08-concurrency')), filePath);
    assert.ok(plan.coverageGaps.length >= 4, `${filePath}: faltan brechas declaradas`);
  }
});

test('las brechas restantes del WebSocket backend quedan declaradas', () => {
  const plan = planFor('backend/internal/delivery/http/websocket_handler.go');
  const text = plan.coverageGaps.join(' ');
  assert.match(text, /tenant/);
  assert.match(text, /expiración/);
  assert.match(text, /exec/);
  assert.match(text, /registro de rutas/);
});

test('el cliente WebSocket del terminal activa seguridad y E2E sin ejecutar todo el frontend', () => {
  const plan = planFor(
    'frontend/src/app/features/teacher/aula/shadow-terminal-modal/shadow-terminal-modal.component.ts'
  );
  assert.ok(plan.criticalAreas.includes('websocket-frontend'));
  assert.equal(plan.level, 'L3');
  assert.equal(plan.suites.security.selected, true);
  assert.equal(plan.suites['frontend-unit'].selected, true);
  assert.ok(plan.suites['frontend-unit'].specs.length >= 1);
  assert.equal(plan.suites['frontend-unit'].full, false);
  const specs = plan.suites['frontend-e2e'].specs.map((item) => item.spec);
  assert.ok(specs.some((spec) => spec.includes('J08')));
});

test('un servicio de polling HTTP no se clasifica como cliente WebSocket', () => {
  const plan = planFor(
    'frontend/src/app/features/student/services/class-advance-notice.service.ts'
  );
  assert.equal(plan.criticalAreas.includes('websocket-frontend'), false);
  assert.equal(plan.suites.security.selected, false);
  assert.notEqual(plan.level, 'L3');
  assert.equal(plan.suites['frontend-unit'].selected, true);
  assert.ok(
    plan.suites['frontend-unit'].specs.includes(
      'src/app/features/student/services/class-advance-notice.service.spec.ts'
    )
  );
});

test('el terminal declara las brechas de handshake y reconexión reales', () => {
  const plan = planFor(
    'frontend/src/app/features/teacher/aula/shadow-terminal-modal/shadow-terminal-modal.component.ts'
  );
  const gaps = plan.coverageGaps.join(' ');
  assert.match(gaps, /no adjunta token explícito/);
  assert.match(gaps, /onclose no reintenta/);
  assert.match(gaps, /otro tenant/);
  assert.match(gaps, /liberar la conexión/);
});

test('el handler exige sesión y aplica la allowlist de origen', () => {
  const handler = fs.readFileSync(
    path.join(REPO_ROOT, 'backend/internal/delivery/http/websocket_handler.go'),
    'utf8'
  );
  assert.match(handler, /websocketSession\(r, h\.authService\)/);
  assert.match(handler, /websocketOriginAllowed\(r\)/);
  assert.doesNotMatch(handler, /CheckOrigin:\s*func\(r \*http\.Request\) bool \{\s*return true/);
  assert.doesNotMatch(handler, /userID := "anonymous"/);
  assert.doesNotMatch(handler, /X-User-Id/);
  assert.doesNotMatch(handler, /X-Tenant-Id/);
});

test('las reglas WebSocket alcanzan archivos reales del repositorio', () => {
  for (const rule of policy.critical.filter((entry) => entry.id.startsWith('websocket-'))) {
    const matched = BACKEND_FILES.concat([
      'frontend/src/app/features/teacher/aula/shadow-terminal-modal/shadow-terminal-modal.component.ts'
    ]).filter((filePath) => matchingEntries([rule], filePath).length > 0);
    assert.ok(matched.length > 0, `${rule.id} no alcanza ninguna ruta real`);
    assert.ok(rule.coverageGaps.length > 0, `${rule.id} no declara brechas de cobertura`);
  }
});
