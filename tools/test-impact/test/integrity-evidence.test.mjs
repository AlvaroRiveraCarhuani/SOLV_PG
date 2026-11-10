import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test, { after, before } from 'node:test';
import { isGeneratedArtifact, planDigest } from '../lib/plan-integrity.mjs';
import { collectChanges } from '../lib/git.mjs';
import { attachExecutionEvidence, signReportReceipt } from '../lib/receipt.mjs';
import {
  CLI,
  REPO_ROOT,
  RUNNER,
  createGitFixture,
  planInFixture,
  runnerWithRepo
} from './fixture.mjs';

const TEST_KEY = 'solv-test-receipt-key-0123456789';
const ENV = {
  ...process.env,
  SOLV_TEST_RECEIPT_KEY: TEST_KEY,
  SOLV_TEST_EXECUTION_ID: '',
  SOLV_TEST_TRUST_MODE: 'local'
};

let directory;
let fixtureRepo;
let planPath;
let plan;

function cli(args) {
  return spawnSync(process.execPath, [CLI, '--repo', fixtureRepo, ...args], {
    cwd: REPO_ROOT,
    env: ENV,
    encoding: 'utf8',
    timeout: 120_000
  });
}

function runner(args) {
  return spawnSync(process.execPath, [RUNNER, '--repo', fixtureRepo, ...args], {
    cwd: REPO_ROOT,
    env: ENV,
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

function makeReport(states, overrides = {}) {
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
    validationStatus: plan.source.kind === 'files' ? 'synthetic' : 'partial',
    level: plan.level,
    levelTitle: plan.levelTitle,
    suites,
    order,
    totals,
    ...overrides
  };
  report.receipt = signReportReceipt(report, Buffer.from(TEST_KEY));
  return report;
}

function writeReport(name, report, key = TEST_KEY) {
  const file = path.join(directory, name);
  attachExecutionEvidence(report, file, Buffer.from(key));
  report.receipt = signReportReceipt(report, Buffer.from(key));
  fs.writeFileSync(file, JSON.stringify(report, null, 2));
  return file;
}

before(() => {
  directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-e7-evidence-'));
  fixtureRepo = createGitFixture();
  planPath = planInFixture(fixtureRepo, ['--base', 'HEAD~1', '--head', 'HEAD']);
  plan = read(planPath);
});

after(() => {
  fs.rmSync(directory, { recursive: true, force: true });
  fs.rmSync(fixtureRepo, { recursive: true, force: true });
  fs.rmSync(planPath, { force: true });
});

test('cli report exige un plan explícito', () => {
  const file = writeReport('no-plan.json', makeReport({ build: 'executed' }));
  const result = cli(['report', '--file', file]);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /--plan/);
});

test('un informe sin recibo no se acepta aunque tenga el digest correcto', () => {
  const report = makeReport({ build: 'executed', 'frontend-unit': 'executed' }, {
    partial: false,
    validated: true,
    validationStatus: 'complete'
  });
  const file = writeReport('unsigned.json', report);
  delete report.receipt;
  fs.writeFileSync(file, JSON.stringify(report, null, 2));
  const result = cli(['report', '--file', file, '--plan', planPath]);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /recibo/);
});

test('un recibo firmado con otra clave no se acepta', () => {
  const report = makeReport({ build: 'executed', 'frontend-unit': 'executed' }, {
    partial: false,
    validated: true,
    validationStatus: 'complete'
  });
  const file = writeReport('wrong-key.json', report);
  report.receipt = signReportReceipt(report, Buffer.from('otra-clave-de-prueba-0123456789'));
  fs.writeFileSync(file, JSON.stringify(report, null, 2));
  const result = cli(['report', '--file', file, '--plan', planPath]);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /recibo/);
});

test('aggregate rechaza un informe sin recibo', () => {
  const reports = path.join(directory, 'reports-unsigned');
  fs.mkdirSync(reports, { recursive: true });
  const report = makeReport({ build: 'executed', 'frontend-unit': 'executed' });
  const file = writeReport(path.join('reports-unsigned', 'build.json'), report);
  delete report.receipt;
  fs.writeFileSync(file, JSON.stringify(report));
  const result = cli(['aggregate', '--plan', planPath, '--dir', reports]);
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /recibo/);
});

test('aggregate rechaza informes duplicados de una suite resuelta', () => {
  const reports = path.join(directory, 'reports-duplicate');
  fs.mkdirSync(reports, { recursive: true });
  const first = makeReport({ build: 'executed', 'frontend-unit': 'executed' });
  const second = makeReport({ build: 'executed', 'frontend-unit': 'executed' });
  writeReport(path.join('reports-duplicate', 'first.json'), first);
  writeReport(path.join('reports-duplicate', 'second.json'), second);
  const result = cli(['aggregate', '--plan', planPath, '--dir', reports]);
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /duplicad/);
});

test('aggregate rechaza informes de ejecuciones distintas', () => {
  const reports = path.join(directory, 'reports-runs');
  fs.mkdirSync(reports, { recursive: true });
  const first = makeReport({ build: 'executed' }, { runId: 'run-a' });
  const second = makeReport({ 'frontend-unit': 'executed' }, { runId: 'run-b' });
  writeReport(path.join('reports-runs', 'build.json'), first);
  writeReport(path.join('reports-runs', 'unit.json'), second);
  const result = cli(['aggregate', '--plan', planPath, '--dir', reports]);
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /ejecuciones distintas/);
});

test('aggregate rechaza un informe de otra ejecución de CI', () => {
  const reports = path.join(directory, 'reports-expected-run');
  fs.mkdirSync(reports, { recursive: true });
  const report = makeReport({ build: 'executed' }, { runId: 'otra-ejecucion', jobId: 'build' });
  writeReport(path.join('reports-expected-run', 'build.json'), report);
  const result = spawnSync(
    process.execPath,
    [CLI, '--repo', fixtureRepo, 'aggregate', '--plan', planPath, '--dir', reports],
    {
      cwd: REPO_ROOT,
      env: { ...ENV, SOLV_TEST_EXECUTION_ID: 'ejecucion-esperada' },
      encoding: 'utf8',
      timeout: 120_000
    }
  );
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /otra ejecución/);
});

test('aggregate rechaza un informe producido por otro job', () => {
  const reports = path.join(directory, 'reports-job');
  fs.mkdirSync(reports, { recursive: true });
  const report = makeReport({ build: 'executed', 'frontend-unit': 'executed' }, {
    jobId: 'test-frontend'
  });
  writeReport(path.join('reports-job', 'build.json'), report);
  const result = cli(['aggregate', '--plan', planPath, '--dir', reports]);
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /job/);
});

test('aggregate rechaza un job que firma con la clave de otro job', () => {
  const reports = path.join(directory, 'reports-ci-keys');
  fs.mkdirSync(reports, { recursive: true });
  const buildKey = 'clave-build-de-prueba-0123456789';
  const frontendKey = 'clave-frontend-de-prueba-0123456789';
  const build = makeReport({ build: 'executed' }, { runId: 'run-ci', jobId: 'build' });
  const frontend = makeReport(
    { 'frontend-unit': 'executed' },
    { runId: 'run-ci', jobId: 'test-frontend' }
  );
  writeReport(path.join('reports-ci-keys', 'build.json'), build, buildKey);
  // El atacante tiene la clave de build y declara el job de frontend.
  writeReport(path.join('reports-ci-keys', 'frontend.json'), frontend, buildKey);
  const result = spawnSync(
    process.execPath,
    [CLI, '--repo', fixtureRepo, 'aggregate', '--plan', planPath, '--dir', reports],
    {
      cwd: REPO_ROOT,
      env: {
        ...ENV,
        CI: 'true',
        SOLV_TEST_TRUST_MODE: '',
        SOLV_TEST_RECEIPT_KEY: 'clave-generica-que-no-debe-usarse-0123456789',
        SOLV_TEST_RECEIPT_KEY_BUILD: buildKey,
        SOLV_TEST_RECEIPT_KEY_TEST_FRONTEND: frontendKey,
        SOLV_TEST_EXECUTION_ID: 'run-ci'
      },
      encoding: 'utf8',
      timeout: 120_000
    }
  );
  assert.notEqual(result.status, 0);
  assert.match(result.stdout, /recibo|job|evidencia/);
});

test('un plan de un commit anterior no se ejecuta contra el HEAD actual', () => {
  const oldPlanPath = path.join(directory, 'old-plan.json');
  const generated = cli([
    'plan',
    '--base',
    'HEAD~1',
    '--head',
    'HEAD~1',
    '--format',
    'json',
    '--output',
    oldPlanPath
  ]);
  assert.equal(generated.status, 0, generated.stderr);
  const result = runner(['--all', '--plan', oldPlanPath, '--dry-run']);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /HEAD actual/);
});

test('la recolección del working tree distingue cambios no representados', () => {
  const repo = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-e7-dirty-'));
  try {
    const git = (...args) => spawnSync('git', args, { cwd: repo, encoding: 'utf8' });
    assert.equal(git('init', '-q').status, 0);
    git('config', 'user.email', 'test@solv.test');
    git('config', 'user.name', 'SOLV Test');
    fs.writeFileSync(path.join(repo, 'sample.txt'), 'base\n');
    git('add', 'sample.txt');
    assert.equal(git('commit', '-q', '-m', 'base').status, 0);
    const head = git('rev-parse', 'HEAD').stdout.trim();
    const clean = collectChanges({ repoRoot: repo, base: head, head, includeWorkingTree: true });
    assert.deepEqual(clean.changes, []);
    fs.writeFileSync(path.join(repo, 'sample.txt'), 'modificado\n');
    const dirty = collectChanges({ repoRoot: repo, base: head, head, includeWorkingTree: true });
    assert.ok(dirty.changes.some((change) => change.path === 'sample.txt'));
    fs.writeFileSync(path.join(repo, 'nuevo.txt'), 'sin seguimiento\n');
    const untracked = collectChanges({ repoRoot: repo, base: head, head, includeWorkingTree: true });
    assert.ok(untracked.changes.some((change) => change.path === 'nuevo.txt'));

    fs.writeFileSync(
      path.join(repo, '.gitignore'),
      'backend/ignored.json\nnode_modules/\n'
    );
    fs.mkdirSync(path.join(repo, 'backend'), { recursive: true });
    fs.writeFileSync(path.join(repo, 'backend', 'ignored.json'), '{"fuente":"ignorada"}\n');
    fs.mkdirSync(path.join(repo, 'node_modules'), { recursive: true });
    fs.writeFileSync(path.join(repo, 'node_modules', 'paquete.json'), '{}\n');
    const ignored = collectChanges({ repoRoot: repo, base: head, head, includeWorkingTree: true });
    assert.ok(ignored.changes.some((change) => change.path === 'backend/ignored.json'));
    assert.ok(!ignored.changes.some((change) => change.path.includes('node_modules')));
  } finally {
    fs.rmSync(repo, { recursive: true, force: true });
  }
});

test('los diagnósticos de E2E no cuentan como cambios del código', () => {
  for (const file of [
    'backend-e2e.log',
    'backend-e2e.pid',
    'frontend-e2e.log',
    'frontend-e2e.pid',
    'frontend/e2e/test-results/.last-run.json',
    'frontend/test-results/.last-run.json',
    'frontend/playwright-report/index.html',
    'impact-reports/build.json',
    'impact-report-build.json',
    'tools/test-impact/.cache/receipt.key'
  ]) {
    assert.equal(isGeneratedArtifact(file), true, file);
  }
  for (const file of [
    'backend/internal/delivery/http/router.go',
    'frontend/src/app/app.ts',
    'backend/config/impact-report.json',
    'frontend/src/app/impact-plan.json'
  ]) {
    assert.equal(isGeneratedArtifact(file), false, file);
  }
});

test('un fuente no rastreado con nombre de artefacto no puede ocultarse', () => {
  const repo = createGitFixture();
  try {
    const source = path.join(repo, 'backend/config/impact-report.json');
    fs.mkdirSync(path.dirname(source), { recursive: true });
    fs.writeFileSync(source, '{"cambio":"real"}\n');
    const hiddenPlan = planInFixture(repo, ['--base', 'HEAD~1', '--head', 'HEAD']);
    try {
      const result = runnerWithRepo(['--all', '--plan', hiddenPlan, '--dry-run'], repo, ENV);
      assert.notEqual(result.status, 0);
      assert.match(result.stderr, /working tree|cambios no representados/);
    } finally {
      fs.rmSync(hiddenPlan, { force: true });
    }
  } finally {
    fs.rmSync(repo, { recursive: true, force: true });
  }
});

test('un plan con cambios no representados se rechaza aunque el HEAD coincida', () => {
  const report = makeReport({ build: 'executed', 'frontend-unit': 'executed' }, {
    partial: false,
    validated: true,
    validationStatus: 'complete'
  });
  report.plan.source = { ...report.plan.source, headCommit: '0'.repeat(40) };
  report.receipt = signReportReceipt(report, Buffer.from(TEST_KEY));
  const file = writeReport('source-mismatch.json', report);
  const result = cli(['report', '--file', file, '--plan', planPath]);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /procedencia|plan/);
});
