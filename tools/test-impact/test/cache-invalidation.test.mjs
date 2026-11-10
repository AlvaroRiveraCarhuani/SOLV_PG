import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { computeCacheKey, readCache, writeCache } from '../lib/cache.mjs';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const RUNNER = path.join(REPO_ROOT, 'tools/test-impact/run.mjs');
const ICON = 'frontend/public/tech-icons/python.svg';

function createRepo() {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-e7-cache-'));
  const git = (...args) => spawnSync('git', args, { cwd: directory, encoding: 'utf8' });
  assert.equal(git('init', '-q').status, 0);
  git('config', 'user.email', 'test@solv.test');
  git('config', 'user.name', 'SOLV Test');
  fs.mkdirSync(path.join(directory, 'tools', 'test-impact', 'lib'), { recursive: true });
  fs.writeFileSync(
    path.join(directory, 'tools', 'test-impact', 'policy.json'),
    JSON.stringify({ version: 3 })
  );
  fs.writeFileSync(path.join(directory, 'tools', 'test-impact', 'run.mjs'), 'export const v = 1;\n');
  fs.writeFileSync(path.join(directory, 'tools', 'test-impact', 'lib', 'a.mjs'), 'export const a = 1;\n');
  fs.writeFileSync(path.join(directory, 'sample.txt'), 'base\n');
  git('add', '.');
  assert.equal(git('commit', '-q', '-m', 'base').status, 0);
  return directory;
}

function runner(args) {
  return spawnSync(process.execPath, [RUNNER, ...args], {
    cwd: REPO_ROOT,
    env: {
      ...process.env,
      SOLV_TEST_TRUST_MODE: 'local',
      SOLV_TEST_RECEIPT_KEY: 'solv-test-receipt-key-0123456789'
    },
    encoding: 'utf8',
    timeout: 120_000
  });
}

function read(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

const plan = { mode: 'targeted', level: 'L1', suites: { build: { selected: true } } };
const context = { command: 'build:frontend', env: { CI: false, NODE_ENV: null, TZ: null } };

test('la clave de caché es determinista con los mismos inputs', () => {
  const directory = createRepo();
  try {
    assert.equal(
      computeCacheKey(directory, plan, context),
      computeCacheKey(directory, plan, context)
    );
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('cambiar la política invalida la caché', () => {
  const directory = createRepo();
  try {
    const before = computeCacheKey(directory, plan, context);
    fs.writeFileSync(
      path.join(directory, 'tools', 'test-impact', 'policy.json'),
      JSON.stringify({ version: 3, changed: true })
    );
    assert.notEqual(before, computeCacheKey(directory, plan, context));
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('cambiar el runner invalida la caché', () => {
  const directory = createRepo();
  try {
    const before = computeCacheKey(directory, plan, context);
    fs.writeFileSync(path.join(directory, 'tools', 'test-impact', 'lib', 'a.mjs'), 'export const a = 2;\n');
    assert.notEqual(before, computeCacheKey(directory, plan, context));
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('cambiar el comando de la suite invalida la caché', () => {
  const directory = createRepo();
  try {
    const before = computeCacheKey(directory, plan, context);
    assert.notEqual(
      before,
      computeCacheKey(directory, plan, { ...context, command: 'build:backend' })
    );
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('cambiar el toolchain o el entorno invalida la caché', () => {
  const directory = createRepo();
  try {
    const before = computeCacheKey(directory, plan, context);
    assert.notEqual(
      before,
      computeCacheKey(directory, plan, {
        ...context,
        toolchain: { node: 'v0.0.0', go: 'go0', npm: '0' }
      })
    );
    assert.notEqual(
      before,
      computeCacheKey(directory, plan, { ...context, env: { CI: true, NODE_ENV: null, TZ: null } })
    );
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('cambiar el contenido de un archivo invalida la caché', () => {
  const directory = createRepo();
  try {
    const before = computeCacheKey(directory, plan, context);
    fs.writeFileSync(path.join(directory, 'sample.txt'), 'otra versión\n');
    assert.notEqual(before, computeCacheKey(directory, plan, context));
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('cambiar el lockfile de dependencias instaladas invalida la caché', () => {
  const directory = createRepo();
  try {
    const lockfile = path.join(directory, 'frontend', 'package-lock.json');
    fs.mkdirSync(path.dirname(lockfile), { recursive: true });
    fs.writeFileSync(lockfile, '{"lockfileVersion":3,"version":1}\n');
    const before = computeCacheKey(directory, plan, context);
    fs.writeFileSync(lockfile, '{"lockfileVersion":3,"version":2}\n');
    assert.notEqual(before, computeCacheKey(directory, plan, context));
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('una fuente ignorada relevante también invalida la caché', () => {
  const directory = createRepo();
  try {
    fs.writeFileSync(path.join(directory, '.gitignore'), 'backend/ignored.json\n');
    fs.mkdirSync(path.join(directory, 'backend'), { recursive: true });
    const source = path.join(directory, 'backend', 'ignored.json');
    fs.writeFileSync(source, '{"version":1}\n');
    const before = computeCacheKey(directory, plan, context);
    fs.writeFileSync(source, '{"version":2}\n');
    assert.notEqual(before, computeCacheKey(directory, plan, context));
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('un fuente con basename de artefacto también invalida la caché', () => {
  const directory = createRepo();
  try {
    const before = computeCacheKey(directory, plan, context);
    const source = path.join(directory, 'backend', 'config', 'impact-report.json');
    fs.mkdirSync(path.dirname(source), { recursive: true });
    fs.writeFileSync(source, '{"version":1}\n');
    const first = computeCacheKey(directory, plan, context);
    fs.writeFileSync(source, '{"version":2}\n');
    const second = computeCacheKey(directory, plan, context);
    assert.notEqual(before, first);
    assert.notEqual(first, second);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('cambiar una variable de entorno relevante invalida la caché', () => {
  const directory = createRepo();
  const previous = process.env.DATABASE_URL;
  try {
    delete process.env.DATABASE_URL;
    const before = computeCacheKey(directory, plan, { command: 'build:frontend' });
    process.env.DATABASE_URL = 'postgres://otro/destino';
    const after = computeCacheKey(directory, plan, { command: 'build:frontend' });
    assert.notEqual(before, after);
  } finally {
    if (previous === undefined) delete process.env.DATABASE_URL;
    else process.env.DATABASE_URL = previous;
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('la identidad de la ejecución no invalida la caché', () => {
  const directory = createRepo();
  const previousExecution = process.env.SOLV_TEST_EXECUTION_ID;
  const previousRun = process.env.SOLV_TEST_RUN_ID;
  try {
    process.env.SOLV_TEST_EXECUTION_ID = 'run-1';
    process.env.SOLV_TEST_RUN_ID = 'docker-run-1';
    const before = computeCacheKey(directory, plan, { command: 'build:frontend' });
    process.env.SOLV_TEST_EXECUTION_ID = 'run-2';
    process.env.SOLV_TEST_RUN_ID = 'docker-run-2';
    const after = computeCacheKey(directory, plan, { command: 'build:frontend' });
    assert.equal(before, after);
  } finally {
    if (previousExecution === undefined) delete process.env.SOLV_TEST_EXECUTION_ID;
    else process.env.SOLV_TEST_EXECUTION_ID = previousExecution;
    if (previousRun === undefined) delete process.env.SOLV_TEST_RUN_ID;
    else process.env.SOLV_TEST_RUN_ID = previousRun;
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('un fallo de git no produce una clave de caché vacía', () => {
  const directory = createRepo();
  const previousPath = process.env.PATH;
  try {
    process.env.PATH = '';
    assert.throws(
      () => computeCacheKey(directory, plan, { command: 'build:frontend' }),
      /git /
    );
  } finally {
    process.env.PATH = previousPath;
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('una entrada de caché alterada no se acepta', () => {
  const directory = createRepo();
  const cacheDirectory = path.join(directory, 'cache');
  const key = 'a'.repeat(64);
  const receiptKey = Buffer.from('clave-de-prueba-de-cache-0123456789');
  try {
    writeCache(directory, key, { suite: 'build', status: 0, durationMs: 1 }, cacheDirectory, receiptKey);
    assert.ok(readCache(directory, key, cacheDirectory, receiptKey));
    const file = path.join(cacheDirectory, `${key}.json`);
    const entry = JSON.parse(fs.readFileSync(file, 'utf8'));
    entry.status = 0;
    entry.durationMs = 999;
    fs.writeFileSync(file, JSON.stringify(entry));
    assert.equal(readCache(directory, key, cacheDirectory, receiptKey), null);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('un fallo no se guarda en caché como éxito', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-e7-cache-fail-'));
  const cacheDirectory = path.join(directory, 'cache');
  try {
    const args = [
      '--partial', '--suite', 'asset-validation',
      '--files', 'M:frontend/public/tech-icons/no-existe.svg',
      '--cache', '--cache-dir', cacheDirectory
    ];
    const firstReport = path.join(directory, 'first.json');
    const secondReport = path.join(directory, 'second.json');
    assert.equal(runner([...args, '--report', firstReport]).status, 1);
    assert.equal(runner([...args, '--report', secondReport]).status, 1);
    const second = read(secondReport);
    assert.equal(second.suites['asset-validation'].executed, true);
    assert.equal(second.suites['asset-validation'].cached, false);
    assert.equal(fs.existsSync(cacheDirectory) ? fs.readdirSync(cacheDirectory).length : 0, 0);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('un dry-run no contamina la caché de una ejecución real', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-e7-cache-dry-'));
  const cacheDirectory = path.join(directory, 'cache');
  try {
    const args = [
      '--partial', '--suite', 'asset-validation', '--files', `M:${ICON}`,
      '--cache', '--cache-dir', cacheDirectory
    ];
    const dryReport = path.join(directory, 'dry.json');
    const realReport = path.join(directory, 'real.json');
    assert.equal(runner([...args, '--dry-run', '--report', dryReport]).status, 0);
    assert.equal(read(dryReport).suites['asset-validation'].state, 'simulated');
    assert.equal(fs.existsSync(cacheDirectory) ? fs.readdirSync(cacheDirectory).length : 0, 0);
    assert.equal(runner([...args, '--report', realReport]).status, 0);
    const real = read(realReport);
    assert.equal(real.suites['asset-validation'].state, 'executed');
    assert.equal(real.suites['asset-validation'].cached, false);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});
