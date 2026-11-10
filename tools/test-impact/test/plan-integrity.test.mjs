import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test, { before, after } from 'node:test';
import { fileURLToPath } from 'node:url';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const CLI = path.join(REPO_ROOT, 'tools/test-impact/cli.mjs');
const RUNNER = path.join(REPO_ROOT, 'tools/test-impact/run.mjs');
const FORM =
  'frontend/src/app/features/admin/courses/components/course-create-modal/course-create-modal.component.ts';
const AUTH = 'backend/internal/core/services/auth_service.go';
const WEBSOCKET = 'backend/internal/delivery/http/websocket_handler.go';

let directory;
let formPlan;
let authPlan;
let websocketPlan;

function generate(file, name) {
  const output = path.join(directory, `${name}.json`);
  const result = spawnSync(
    process.execPath,
    [CLI, 'plan', '--files', `M:${file}`, '--output', output, '--format', 'json'],
    { cwd: REPO_ROOT, encoding: 'utf8' }
  );
  assert.equal(result.status, 0, result.stderr);
  return JSON.parse(fs.readFileSync(output, 'utf8'));
}

function writePlan(name, plan) {
  const file = path.join(directory, `${name}.json`);
  fs.writeFileSync(file, `${JSON.stringify(plan, null, 2)}\n`);
  return file;
}

function run(planFile, extra = []) {
  return spawnSync(
    process.execPath,
    [RUNNER, '--all', '--plan', planFile, '--dry-run', ...extra],
    { cwd: REPO_ROOT, encoding: 'utf8', timeout: 120_000 }
  );
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function assertRejected(name, mutate, pattern) {
  const plan = clone(formPlan);
  mutate(plan);
  const file = writePlan(name, plan);
  const reportPath = path.join(directory, `${name}-report.json`);
  const result = run(file, ['--report', reportPath]);
  assert.notEqual(result.status, 0, `${name} no fue rechazado`);
  assert.match(result.stderr, pattern || /Plan rechazado|Error/);
  const report = JSON.parse(fs.readFileSync(reportPath, 'utf8'));
  assert.equal(report.validated, false);
  assert.equal(report.validationStatus, 'rejected');
  assert.equal(report.totals.executed, 0);
}

before(() => {
  directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-e7-integrity-'));
  formPlan = generate(FORM, 'form');
  authPlan = generate(AUTH, 'auth');
  websocketPlan = generate(WEBSOCKET, 'websocket');
});

after(() => {
  fs.rmSync(directory, { recursive: true, force: true });
});

test('un plan legítimo se ejecuta y declara las suites obligatorias', () => {
  const result = run(writePlan('legit', formPlan), ['--report', path.join(directory, 'legit-report.json')]);
  assert.equal(result.status, 2, result.stderr);
  const report = JSON.parse(fs.readFileSync(path.join(directory, 'legit-report.json'), 'utf8'));
  assert.deepEqual(report.required, ['build', 'frontend-unit']);
  assert.equal(report.partial, false);
  assert.equal(report.dryRun, true);
  assert.equal(report.validated, false);
  assert.equal(report.validationStatus, 'simulated');
});

test('un plan con todas las suites en false se rechaza y no ejecuta nada', () => {
  assertRejected(
    'all-false',
    (plan) => {
      for (const entry of Object.values(plan.suites)) entry.selected = false;
    },
    /decisiones o alcance alterados/
  );
});

test('un plan sin backend-unit se rechaza', () => {
  assertRejected('missing-backend-unit', (plan) => {
    delete plan.suites['backend-unit'];
  }, /faltan \[backend-unit\]/);
});

test('un plan de autenticación sin security se rechaza', () => {
  const plan = clone(authPlan);
  delete plan.suites.security;
  const result = run(writePlan('auth-no-security', plan));
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /faltan \[security\]/);
});

test('un plan de autenticación con la regla crítica eliminada se rechaza', () => {
  const plan = clone(authPlan);
  plan.criticalAreas = [];
  const result = run(writePlan('auth-no-critical', plan));
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /reglas críticas/);
});

test('un plan de WebSocket sin sus cobertura faltante declarada se rechaza', () => {
  const plan = clone(websocketPlan);
  plan.coverageGaps = [];
  const result = run(writePlan('ws-no-gaps', plan));
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /cobertura faltante/);
});

test('una suite desconocida en el plan se rechaza', () => {
  assertRejected('unknown-suite', (plan) => {
    plan.suites['backdoor'] = { selected: true, reason: 'suite inventada' };
  }, /sobran \[backdoor\]/);
});

test('un plan con versión de política antigua se rechaza', () => {
  assertRejected('old-policy', (plan) => {
    plan.policyVersion = 2;
  }, /política 2 incompatible/);
});

test('un plan generado con otra política se rechaza por su digest', () => {
  assertRejected('old-digest', (plan) => {
    plan.policyDigest = '0'.repeat(64);
  }, /política cambió/);
});

test('un plan de otro contrato se rechaza', () => {
  assertRejected('old-contract', (plan) => {
    plan.contractVersion = 'e6.0';
  }, /contrato e6\.0 incompatible/);
});

test('un plan manipulado con menos archivos se rechaza', () => {
  assertRejected('changed-files', (plan) => {
    plan.changed = [];
  }, /archivos modificados/);
});

test('un plan no puede elevar el límite de archivos de la política', () => {
  assertRejected('max-files', (plan) => {
    plan.source.maxChangedFiles = 1000;
  }, /maxChangedFiles/);
});

test('un plan con el nivel degradado se rechaza', () => {
  assertRejected('level', (plan) => {
    plan.level = 'L0';
    plan.levelTitle = 'Validación inmediata';
  }, /nivel L0 no coincide/);
});

test('un plan que oculta el build obligatorio se rechaza', () => {
  assertRejected('build', (plan) => {
    plan.suites.build = {
      selected: false,
      reason: 'sin compilación',
      backend: false,
      frontend: false
    };
  }, /suite build/);
});
