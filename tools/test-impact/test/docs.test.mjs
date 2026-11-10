import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { loadPolicy } from '../lib/policy.mjs';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(HERE, '..', '..', '..');
const DOC = path.join(REPO_ROOT, 'docs', 'GOBERNANZA', 'TESTING_POR_IMPACTO.md');

test('la documentación existe y declara la versión vigente de la política', () => {
  assert.ok(fs.existsSync(DOC), 'falta docs/GOBERNANZA/TESTING_POR_IMPACTO.md');
  const policy = loadPolicy(REPO_ROOT);
  const doc = fs.readFileSync(DOC, 'utf8');
  assert.match(doc, new RegExp(`Política v${policy.version}`));
  assert.match(doc, /tools\/test-impact\/policy\.json/);
});

test('la documentación menciona todas las suites de la política', () => {
  const policy = loadPolicy(REPO_ROOT);
  const doc = fs.readFileSync(DOC, 'utf8');
  const missing = Object.keys(policy.suites).filter((suite) => !doc.includes(`\`${suite}\``));
  assert.deepEqual(missing, [], 'hay suites sin documentar');
});

test('la documentación enlaza el CI, el runner y el aislamiento de base', () => {
  const doc = fs.readFileSync(DOC, 'utf8');
  for (const fragment of [
    '.github/workflows/ci.yml',
    'tools/test-impact/run.mjs',
    'TEST_DB_DSN',
    'test-db.sh',
    'e2e.yml'
  ]) {
    assert.ok(doc.includes(fragment), `la documentación no menciona ${fragment}`);
  }
});

test('el CI consume el runner de impacto y no repite comandos de prueba', () => {
  const ci = fs.readFileSync(path.join(REPO_ROOT, '.github', 'workflows', 'ci.yml'), 'utf8');
  assert.match(ci, /tools\/test-impact\/run\.mjs/);
  assert.match(ci, /tools\/test-impact\/cli\.mjs/);
  assert.match(ci, /args=\(plan --format text/);
  assert.match(ci, /test-backend-docker/);
  assert.match(ci, /--partial --suite backend-docker/);
  assert.match(ci, /impact-report-backend-docker\.json/);
  assert.match(ci, /cli\.mjs aggregate/);
  assert.doesNotMatch(ci, /\bgo test\b/);
  assert.doesNotMatch(ci, /\bng test\b/);
  assert.doesNotMatch(ci, /npm run test:ci/);
  assert.doesNotMatch(ci, /\bplaywright test\b/);
});

test('el workflow E2E usa el stack aislado y limpia solo su proyecto', () => {
  const e2e = fs.readFileSync(path.join(REPO_ROOT, '.github', 'workflows', 'e2e.yml'), 'utf8');
  assert.match(e2e, /tools\/test-impact\/compose\.e2e\.yml/);
  assert.match(e2e, /-p solv-e2e-ci/);
  assert.match(e2e, /down -v/);
  assert.match(e2e, /tools\/test-impact\/run\.mjs --partial --suite frontend-e2e/);
  assert.match(e2e, /impact-report-frontend-e2e\.json/);
});

test('la documentación no duplica la tabla de globs de la política', () => {
  const doc = fs.readFileSync(DOC, 'utf8');
  for (const fragment of ['backend/internal/**', 'frontend/src/app/**', '**/*.go']) {
    assert.ok(!doc.includes(fragment), `la documentación duplica la regla ${fragment}`);
  }
});
