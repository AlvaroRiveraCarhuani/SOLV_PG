import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SCRIPT = path.resolve(HERE, '..', 'ci-summary.sh');

function baseEnv(overrides = {}) {
  return {
    PATH: process.env.PATH,
    IMPACT_RESULT: 'success',
    LINT_RESULT: 'success',
    ASSET_RESULT: 'skipped',
    UNIT_RESULT: 'skipped',
    INTEGRATION_RESULT: 'skipped',
    DOCKER_RESULT: 'skipped',
    FRONTEND_RESULT: 'skipped',
    E2E_RESULT: 'skipped',
    MIGRATION_RESULT: 'skipped',
    BUILD_RESULT: 'skipped',
    SECURITY_RESULT: 'skipped',
    UNIT_EXPECTED: 'false',
    INTEGRATION_EXPECTED: 'false',
    DOCKER_EXPECTED: 'false',
    FRONTEND_EXPECTED: 'false',
    E2E_EXPECTED: 'false',
    MIGRATION_EXPECTED: 'false',
    BUILD_EXPECTED: 'false',
    SECURITY_EXPECTED: 'false',
    ASSET_EXPECTED: 'false',
    ...overrides
  };
}

function run(overrides = {}) {
  return spawnSync('bash', [SCRIPT], {
    env: baseEnv(overrides),
    encoding: 'utf8'
  });
}

test('acepta un plan no seleccionador con jobs skipped', () => {
  assert.equal(run().status, 0);
});

test('una advertencia advisory no bloquea el pipeline', () => {
  const result = run({ LINT_ADVISORY: 'true' });
  assert.equal(result.status, 0);
  assert.match(result.stdout, /advisory/);
});

test('falla si un job obligatorio termina en failure', () => {
  const result = run({ UNIT_EXPECTED: 'true', UNIT_RESULT: 'failure' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /unit.*failure/);
});

test('falla si un job obligatorio termina en cancelled', () => {
  const result = run({ DOCKER_EXPECTED: 'true', DOCKER_RESULT: 'cancelled' });
  assert.notEqual(result.status, 0);
});

test('falla si un job obligatorio queda skipped', () => {
  const result = run({ E2E_EXPECTED: 'true', E2E_RESULT: 'skipped' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /e2e.*skipped/);
});

test('falla si asset-validation obligatorio queda skipped', () => {
  const result = run({ ASSET_EXPECTED: 'true', ASSET_RESULT: 'skipped' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /assets.*skipped/);
});

test('falla si un job no seleccionado no queda skipped', () => {
  const result = run({ SECURITY_EXPECTED: 'false', SECURITY_RESULT: 'success' });
  assert.notEqual(result.status, 0);
});

test('falla si el plan no publica todos los booleanos', () => {
  const result = run({ E2E_EXPECTED: '' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /booleano/);
});

test('falla si impact o lint no terminan en success', () => {
  assert.notEqual(run({ IMPACT_RESULT: 'failure' }).status, 0);
  assert.notEqual(run({ LINT_RESULT: 'cancelled' }).status, 0);
});