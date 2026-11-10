import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test, { before, after } from 'node:test';
import { planDigest } from '../lib/plan-integrity.mjs';
import { attachExecutionEvidence, signReportReceipt } from '../lib/receipt.mjs';
import {
  CLI,
  FORM,
  ICON,
  REPO_ROOT,
  RUNNER,
  createGitFixture,
  planInFixture
} from './fixture.mjs';

const TEST_KEY = 'solv-test-receipt-key-0123456789';
const TEST_ENV = {
  ...process.env,
  SOLV_TEST_RECEIPT_KEY: TEST_KEY,
  SOLV_TEST_EXECUTION_ID: '',
  SOLV_TEST_TRUST_MODE: 'local'
};

let directory;
let fixtureRepo;
let formPlanPath;
let formPlan;
let iconPlanPath;

function cli(args) {
  return spawnSync(process.execPath, [CLI, '--repo', fixtureRepo, ...args], {
    cwd: REPO_ROOT,
    env: TEST_ENV,
    encoding: 'utf8',
    timeout: 120_000
  });
}

function runner(args) {
  return spawnSync(process.execPath, [RUNNER, '--repo', fixtureRepo, ...args], {
    cwd: REPO_ROOT,
    env: TEST_ENV,
    encoding: 'utf8',
    timeout: 120_000
  });
}

function read(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

function counts(suites) {
  const values = Object.values(suites);
  return {
    selected: values.filter((entry) => entry.selected).length,
    executed: values.filter((entry) => entry.state === 'executed').length,
    cached: values.filter((entry) => entry.state === 'cached').length,
    simulated: values.filter((entry) => entry.state === 'simulated').length,
    skipped: values.filter((entry) => entry.state === 'skipped').length,
    pending: values.filter((entry) => entry.state === 'pending').length,
    failed: values.filter((entry) => entry.state === 'failed').length
  };
}

function fakeReport(plan, states, overrides = {}) {
  const required = Object.entries(plan.suites)
    .filter(([, entry]) => entry.selected)
    .map(([id]) => id)
    .sort();
  const suites = {};
  const order = [];
  for (const [id, entry] of Object.entries(plan.suites)) {
    const state = states[id] || (entry.selected ? 'pending' : 'skipped');
    suites[id] = {
      selected: Boolean(entry.selected),
      state,
      status: ['executed', 'cached'].includes(state) ? 0 : state === 'failed' ? 1 : null,
      executed: state === 'executed',
      cached: state === 'cached',
      simulated: state === 'simulated',
      durationMs: 0,
      reason: entry.reason
    };
    order.push({ id });
  }
  const totals = { required: required.length, ...counts(suites) };
  const report = {
    schema: 4,
    plan: {
      contractVersion: plan.contractVersion,
      policyVersion: plan.policyVersion,
      policyDigest: plan.policyDigest,
      digest: planDigest(plan),
      source: plan.source,
      level: plan.level,
      mode: plan.mode
    },
    runId: 'test-execution',
    jobId: 'local',
    required,
    partial: true,
    dryRun: false,
    synthetic: plan.source.kind === 'files',
    validated: false,
    validationStatus: plan.source.kind === 'files'
      ? 'synthetic'
      : totals.failed > 0
        ? 'failed'
        : 'partial',
    level: plan.level,
    levelTitle: plan.levelTitle,
    suites,
    order,
    totals,
    ...overrides
  };
  return report;
}

function writeReport(file, report, key = TEST_KEY) {
  attachExecutionEvidence(report, file, Buffer.from(key));
  report.receipt = signReportReceipt(report, Buffer.from(key));
  fs.writeFileSync(file, JSON.stringify(report));
  return report;
}

before(() => {
  directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-e7-contract-'));
  fixtureRepo = createGitFixture();
  formPlanPath = planInFixture(fixtureRepo, ['--base', 'HEAD~1', '--head', 'HEAD']);
  iconPlanPath = planInFixture(fixtureRepo, ['--files', `M:${ICON}`]);
  formPlan = read(formPlanPath);
});

after(() => {
  fs.rmSync(directory, { recursive: true, force: true });
  fs.rmSync(fixtureRepo, { recursive: true, force: true });
  fs.rmSync(formPlanPath, { force: true });
  fs.rmSync(iconPlanPath, { force: true });
});

test('--suite sin --partial se rechaza: una suite aislada no es validación completa', () => {
  const result = runner(['--suite', 'build', '--plan', formPlanPath, '--dry-run']);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /exige --partial/);
});

test('--partial ejecuta una suite y enumera la pendiente sin declarar validación', () => {
  const reportPath = path.join(directory, 'partial.json');
  const result = runner([
    '--partial', '--suite', 'build', '--plan', formPlanPath, '--dry-run', '--report', reportPath
  ]);
  assert.equal(result.status, 0, result.stderr);
  const report = read(reportPath);
  assert.equal(report.partial, true);
  assert.equal(report.validated, false);
  assert.equal(report.suites.build.state, 'simulated');
  assert.equal(report.suites['frontend-unit'].state, 'pending');
  assert.equal(report.totals.executed, 0);
  assert.equal(report.totals.simulated, 1);
  assert.equal(report.totals.pending, 1);
});

test('--if-selected convierte una suite no requerida en no-op honesto', () => {
  const reportPath = path.join(directory, 'if-selected.json');
  const result = runner([
    '--partial', '--suite', 'backend-unit', '--if-selected', '--plan', formPlanPath,
    '--dry-run', '--report', reportPath
  ]);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /no fue seleccionada/);
  const report = read(reportPath);
  assert.equal(report.suites['backend-unit'].state, 'skipped');
  assert.equal(report.validated, false);
  assert.equal(report.partial, true);
});

test('una suite no seleccionada sin --if-selected falla en lugar de fingir éxito', () => {
  const result = runner([
    '--partial', '--suite', 'backend-unit', '--plan', formPlanPath, '--dry-run'
  ]);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /no está seleccionada/);
});

test('--all --dry-run no cuenta ninguna suite como ejecutada', () => {
  const reportPath = path.join(directory, 'all-dry.json');
  const result = runner(['--all', '--plan', formPlanPath, '--dry-run', '--report', reportPath]);
  assert.equal(result.status, 2, result.stderr);
  const report = read(reportPath);
  assert.equal(report.totals.executed, 0);
  assert.equal(report.totals.simulated, 2);
  assert.equal(report.validated, false);
  assert.equal(report.validationStatus, 'simulated');
});

test('--all ejecuta el plan sintético pero no lo declara validación del árbol', () => {
  const reportPath = path.join(directory, 'icon-report.json');
  const result = runner(['--all', '--plan', iconPlanPath, '--report', reportPath]);
  assert.equal(result.status, 2, result.stderr);
  const report = read(reportPath);
  assert.equal(report.totals.required, 1);
  assert.equal(report.suites['asset-validation'].state, 'executed');
  assert.equal(report.synthetic, true);
  assert.equal(report.validated, false);
  assert.equal(report.validationStatus, 'synthetic');
  const printable = cli(['report', '--file', reportPath, '--plan', iconPlanPath]);
  assert.equal(printable.status, 2, printable.stderr);
  assert.match(printable.stdout, /Alcance sintético/);
});

test('el mismo plan distingue simulación de ejecución real', () => {
  const dryPath = path.join(directory, 'same-dry.json');
  const realPath = path.join(directory, 'same-real.json');
  const dry = runner(['--all', '--plan', iconPlanPath, '--dry-run', '--report', dryPath]);
  assert.equal(dry.status, 2, dry.stderr);
  const dryReport = read(dryPath);
  assert.equal(dryReport.suites['asset-validation'].state, 'simulated');
  assert.equal(dryReport.totals.executed, 0);
  assert.equal(dryReport.totals.simulated, 1);
  const real = runner(['--all', '--plan', iconPlanPath, '--report', realPath]);
  assert.equal(real.status, 2, real.stderr);
  const realReport = read(realPath);
  assert.equal(realReport.suites['asset-validation'].state, 'executed');
  assert.equal(realReport.totals.executed, 1);
  assert.equal(realReport.totals.simulated, 0);
  assert.equal(realReport.validated, false, 'el plan sintético no valida el árbol');
});

test('un informe completo con evidencia y HMAC de prueba se valida', () => {
  const reportPath = path.join(directory, 'complete-report.json');
  const report = fakeReport(
    formPlan,
    { build: 'executed', 'frontend-unit': 'executed' },
    { partial: false, validated: true, validationStatus: 'complete' }
  );
  writeReport(reportPath, report);
  const printable = cli(['report', '--file', reportPath, '--plan', formPlanPath]);
  assert.equal(printable.status, 0, printable.stderr);
  assert.match(printable.stdout, /Validación completa/);
});

test('un informe completo sin archivo de evidencia se rechaza', () => {
  const reportPath = path.join(directory, 'complete-without-evidence.json');
  const report = fakeReport(
    formPlan,
    { build: 'executed', 'frontend-unit': 'executed' },
    { partial: false, validated: true, validationStatus: 'complete' }
  );
  report.receipt = signReportReceipt(report, Buffer.from(TEST_KEY));
  fs.writeFileSync(reportPath, JSON.stringify(report));
  const result = cli(['report', '--file', reportPath, '--plan', formPlanPath]);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /evidencia|execution/);
});

test('un plan sintético con un fallo se informa como fallo y no se valida', () => {
  const reportPath = path.join(directory, 'synthetic-failed.json');
  const report = fakeReport(
    read(iconPlanPath),
    { 'asset-validation': 'failed' },
    { partial: false, validated: false, validationStatus: 'failed', synthetic: true }
  );
  writeReport(reportPath, report);
  const result = cli(['report', '--file', reportPath, '--plan', iconPlanPath]);
  assert.equal(result.status, 1, result.stderr + result.stdout);
  assert.match(result.stdout, /FALLÓ|falló/);
});

test('un plan sintético no puede declarar complete aunque omita synthetic', () => {
  const reportPath = path.join(directory, 'synthetic-complete.json');
  const report = fakeReport(
    read(iconPlanPath),
    { 'asset-validation': 'executed' },
    { partial: false, validated: true, validationStatus: 'complete', synthetic: false }
  );
  writeReport(reportPath, report);
  const result = cli(['report', '--file', reportPath, '--plan', iconPlanPath]);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /sintético|synthetic|completa/);
});

test('rechaza un informe que declara validated sin resolver las suites', () => {
  const reportPath = path.join(directory, 'contradictory-report.json');
  const report = fakeReport(formPlan, { build: 'executed' }, {
    partial: false,
    validated: true,
    validationStatus: 'complete'
  });
  writeReport(reportPath, report);
  const result = cli(['report', '--file', reportPath, '--plan', formPlanPath]);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /incoherente|pending/);
});

test('aggregate acepta todos los jobs cuando cubren las suites obligatorias', () => {
  const reports = path.join(directory, 'reports-ok');
  fs.mkdirSync(reports, { recursive: true });
  writeReport(
    path.join(reports, 'build.json'),
    fakeReport(formPlan, { build: 'executed' })
  );
  writeReport(
    path.join(reports, 'unit.json'),
    fakeReport(formPlan, { 'frontend-unit': 'cached' })
  );
  const result = cli(['aggregate', '--plan', formPlanPath, '--dir', reports]);
  assert.equal(result.status, 0, result.stderr + result.stdout);
  assert.match(result.stdout, /Validación global completa/);
});

test('aggregate falla si un job obligatorio sigue pendiente', () => {
  const reports = path.join(directory, 'reports-missing');
  fs.mkdirSync(reports, { recursive: true });
  writeReport(
    path.join(reports, 'build.json'),
    fakeReport(formPlan, { build: 'executed' })
  );
  const result = cli(['aggregate', '--plan', formPlanPath, '--dir', reports]);
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /frontend-unit no fue ejecutada/);
});

test('aggregate falla si un job obligatorio falló', () => {
  const reports = path.join(directory, 'reports-failed');
  fs.mkdirSync(reports, { recursive: true });
  writeReport(
    path.join(reports, 'build.json'),
    fakeReport(formPlan, { build: 'executed' })
  );
  writeReport(
    path.join(reports, 'unit.json'),
    fakeReport(formPlan, { 'frontend-unit': 'failed' })
  );
  const result = cli(['aggregate', '--plan', formPlanPath, '--dir', reports]);
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /frontend-unit falló/);
});

test('aggregate no acepta una suite simulada como validación', () => {
  const reports = path.join(directory, 'reports-simulated');
  fs.mkdirSync(reports, { recursive: true });
  writeReport(
    path.join(reports, 'simulated.json'),
    fakeReport(formPlan, { build: 'simulated' }, { dryRun: true })
  );
  writeReport(
    path.join(reports, 'unit.json'),
    fakeReport(formPlan, { 'frontend-unit': 'executed' })
  );
  const result = cli(['aggregate', '--plan', formPlanPath, '--dir', reports]);
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /build solo fue simulada/);
});

test('aggregate no acepta un informe sintético como validación del árbol', () => {
  const reports = path.join(directory, 'reports-synthetic');
  fs.mkdirSync(reports, { recursive: true });
  const synthetic = fakeReport(formPlan, { build: 'executed', 'frontend-unit': 'executed' }, {
    synthetic: true,
    validationStatus: 'synthetic'
  });
  writeReport(path.join(reports, 'synthetic.json'), synthetic);
  const result = cli(['aggregate', '--plan', formPlanPath, '--dir', reports]);
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /sintético|synthetic/);
});

test('aggregate rechaza un informe generado con otro plan', () => {
  const iconPlan = read(iconPlanPath);
  const reports = path.join(directory, 'reports-other-plan');
  fs.mkdirSync(reports, { recursive: true });
  writeReport(
    path.join(reports, 'other.json'),
    fakeReport(iconPlan, { 'asset-validation': 'executed' })
  );
  const result = cli(['aggregate', '--plan', formPlanPath, '--dir', reports]);
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /otro plan|sintético|procedencia/);
});

test('cli report rechaza un informe que incumple el contrato global', () => {
  const report = fakeReport(formPlan, { build: 'executed', 'frontend-unit': 'executed' });
  delete report.totals;
  const file = path.join(directory, 'broken-report.json');
  writeReport(file, report);
  const result = cli(['report', '--file', file, '--plan', formPlanPath]);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /faltan totals/);
});
