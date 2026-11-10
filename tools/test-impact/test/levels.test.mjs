import assert from 'node:assert/strict';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';
import { buildBackendGraph } from '../lib/backend-graph.mjs';
import { buildFrontendGraph } from '../lib/frontend-graph.mjs';
import { loadPolicy } from '../lib/policy.mjs';
import { createPlan } from '../lib/select.mjs';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const policy = loadPolicy(REPO_ROOT);
const backendGraph = buildBackendGraph(REPO_ROOT);
const frontendGraph = buildFrontendGraph(REPO_ROOT);

function planFor(filePath, status = 'M') {
  return createPlan({
    policy,
    changes: [{ status, path: filePath }],
    backendGraph,
    frontendGraph,
    options: { event: 'local' }
  });
}

function assertLevel(plan, level) {
  assert.equal(plan.level, level, `${plan.changed[0]?.path}: nivel esperado ${level}`);
  assert.equal(plan.levelTitle, policy.levels[level].title);
  assert.ok(plan.levelReasons.length > 0, 'el plan debe explicar el nivel');
}

test('los quince escenarios de referencia tienen nivel y selección verificables', () => {
  const cases = [
    {
      name: 'iconos',
      path: 'frontend/public/tech-icons/python.svg',
      level: 'L0',
      selected: ['asset-validation'],
      omitted: ['backend-unit', 'backend-integration', 'backend-docker', 'frontend-e2e', 'build']
    },
    {
      name: 'pantalla visual',
      path: 'frontend/src/app/features/admin/students/admin-students.component.html',
      level: 'L0',
      selected: ['frontend-unit'],
      omitted: ['backend-unit', 'backend-integration', 'backend-docker', 'frontend-e2e', 'build']
    },
    {
      name: 'formulario',
      path: 'frontend/src/app/features/admin/courses/components/course-create-modal/course-create-modal.component.ts',
      level: 'L1',
      selected: ['frontend-unit', 'build'],
      omitted: ['backend-unit', 'backend-integration', 'frontend-e2e']
    },
    {
      name: 'servicio frontend',
      path: 'frontend/src/app/features/admin/services/admin-courses.service.ts',
      level: 'L1',
      selected: ['frontend-unit', 'build'],
      omitted: ['backend-unit', 'frontend-e2e']
    },
    {
      name: 'endpoint Go',
      path: 'backend/internal/delivery/http/admin_academic_handler.go',
      level: 'L2',
      selected: ['backend-unit', 'backend-integration', 'frontend-e2e', 'build'],
      omitted: ['backend-docker', 'migrations']
    },
    {
      name: 'autenticación',
      path: 'backend/internal/core/services/auth_service.go',
      level: 'L3',
      selected: ['backend-unit', 'backend-integration', 'frontend-e2e', 'build', 'security'],
      omitted: ['backend-docker', 'migrations']
    },
    {
      name: 'motor del juez',
      path: 'backend/internal/core/services/judge_service.go',
      level: 'L3',
      selected: ['backend-unit', 'backend-integration', 'backend-docker', 'frontend-e2e', 'security'],
      omitted: ['migrations']
    },
    {
      name: 'migración PostgreSQL',
      path: 'backend/migrations/00051_submission_verdict_origin.sql',
      level: 'L3',
      selected: ['backend-unit', 'backend-integration', 'frontend-e2e', 'migrations', 'build', 'security'],
      omitted: []
    },
    {
      name: 'infraestructura Docker',
      path: 'backend/internal/infrastructure/docker/docker_container.go',
      level: 'L3',
      selected: ['backend-unit', 'backend-integration', 'backend-docker', 'frontend-e2e', 'build', 'security'],
      omitted: ['migrations']
    },
    {
      name: 'contrato compartido',
      path: 'frontend/src/app/shared/models/api-response.model.ts',
      level: 'L3',
      selected: ['backend-unit', 'backend-integration', 'frontend-unit', 'frontend-e2e', 'build', 'security'],
      omitted: ['backend-docker', 'migrations']
    },
    {
      name: 'HTML Angular sin spec',
      path: 'frontend/src/app/features/admin/environment-requests/admin-environment-requests.component.html',
      level: 'L1',
      selected: ['build'],
      omitted: ['backend-unit', 'backend-integration', 'frontend-e2e', 'security']
    },
    {
      name: 'autenticación frontend',
      path: 'frontend/src/app/core/guards/auth.guard.ts',
      level: 'L3',
      selected: ['frontend-unit', 'frontend-e2e', 'build', 'security'],
      omitted: ['backend-unit', 'backend-integration', 'backend-docker', 'migrations']
    },
    {
      name: 'WebSocket backend',
      path: 'backend/internal/delivery/http/websocket_handler.go',
      level: 'L3',
      selected: ['backend-unit', 'backend-integration', 'frontend-e2e', 'build', 'security'],
      omitted: ['backend-docker', 'migrations']
    },
    {
      name: 'WebSocket frontend',
      path: 'frontend/src/app/features/teacher/aula/shadow-terminal-modal/shadow-terminal-modal.component.ts',
      level: 'L3',
      selected: ['frontend-unit', 'frontend-e2e', 'build', 'security'],
      omitted: ['backend-unit', 'backend-integration', 'backend-docker', 'migrations']
    },
    {
      name: 'shell de Angular',
      path: 'frontend/src/index.html',
      level: 'L2',
      selected: ['frontend-e2e', 'build'],
      omitted: ['frontend-unit', 'backend-unit', 'security']
    }
  ];

  for (const item of cases) {
    const plan = planFor(item.path);
    assertLevel(plan, item.level);
    for (const suiteId of item.selected) {
      assert.equal(plan.suites[suiteId].selected, true, `${item.name}: ${suiteId} debe ejecutarse`);
    }
    for (const suiteId of item.omitted) {
      assert.equal(plan.suites[suiteId].selected, false, `${item.name}: ${suiteId} no debe ejecutarse`);
    }
    assert.ok(plan.decisions.suites[item.selected[0]].reason.length > 0);
  }
});

test('un icono no abre backend ni infraestructura y conserva la validación del recurso', () => {
  const plan = planFor('frontend/public/tech-icons/python.svg');
  assert.equal(plan.level, 'L0');
  assert.equal(plan.suites['asset-validation'].selected, true);
  assert.equal(plan.suites['asset-validation'].files.length, 1);
  assert.equal(plan.suites['backend-unit'].selected, false);
  assert.equal(plan.suites['backend-integration'].selected, false);
  assert.equal(plan.suites['backend-docker'].selected, false);
  assert.equal(plan.suites['frontend-e2e'].selected, false);
  assert.equal(plan.suites.build.selected, false);
});

test('un formulario L1 no paga E2E, pero conserva su spec y build de frontend', () => {
  const plan = planFor(
    'frontend/src/app/features/admin/courses/components/course-create-modal/course-create-modal.component.ts'
  );
  assert.equal(plan.level, 'L1');
  assert.equal(plan.suites['frontend-unit'].selected, true);
  assert.ok(plan.suites['frontend-unit'].specs.length > 0);
  assert.equal(plan.suites['frontend-e2e'].selected, false);
  assert.equal(plan.suites.build.backend, false);
  assert.equal(plan.suites.build.frontend, true);
});

test('un cambio de autenticación no puede degradarse a L0 por ser una sola línea', () => {
  const plan = planFor('backend/internal/core/services/auth_service.go');
  assert.equal(plan.level, 'L3');
  assert.equal(plan.suites.security.selected, true);
  assert.equal(plan.suites['backend-integration'].selected, true);
  assert.equal(plan.suites['backend-unit'].full, true);
});

test('las reglas críticas son aditivas y no ocultan el recurso visual', () => {
  const plan = createPlan({
    policy,
    changes: [
      { status: 'M', path: 'frontend/public/tech-icons/python.svg' },
      { status: 'M', path: 'backend/internal/core/services/auth_service.go' }
    ],
    backendGraph,
    frontendGraph,
    options: { event: 'local' }
  });
  assert.equal(plan.level, 'L3');
  assert.equal(plan.suites['asset-validation'].selected, true);
  assert.equal(plan.suites.security.selected, true);
  assert.ok(plan.levelReasons.some((reason) => reason.id === 'auth-backend'));
});

test('una ruta no clasificada escala a L3 y no finge cobertura', () => {
  const plan = planFor('scripts/nueva-herramienta.bin');
  assert.equal(plan.mode, 'full');
  assert.equal(plan.level, 'L3');
  assert.ok(plan.unknown.length > 0);
});

test('un contrato compartido no se reduce a un test visual', () => {
  const plan = planFor('frontend/src/app/shared/models/api-response.model.ts');
  assert.equal(plan.level, 'L3');
  assert.equal(plan.suites['frontend-unit'].full, true);
  assert.equal(plan.suites['backend-unit'].full, true);
  assert.equal(plan.suites.security.selected, true);
  assert.equal(plan.suites.build.backend, true);
  assert.equal(plan.suites.build.frontend, true);
});

test('la ejecución completa conserva la validación de recursos estáticos', () => {
  const plan = createPlan({
    policy,
    changes: [{ status: 'M', path: 'backend/internal/core/services/auth_service.go' }],
    backendGraph,
    frontendGraph,
    options: { event: 'schedule', full: true }
  });
  assert.equal(plan.level, 'L3');
  assert.equal(plan.suites['asset-validation'].selected, true);
  assert.equal(plan.suites['asset-validation'].full, true);
});