import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { requireTestDatabase, validateTestDatabaseURL } from '../lib/test-db.mjs';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SCRIPT = path.resolve(HERE, '..', 'test-db.sh');

test('no hay DSN predeterminado ni se acepta DATABASE_URL', () => {
  assert.throws(() => requireTestDatabase({ DATABASE_URL: 'postgres://u:p@127.0.0.1:5433/solv_test' }));
  assert.throws(() => validateTestDatabaseURL(''));
});

test('acepta únicamente loopback y puertos de test', () => {
  for (const dsn of [
    'postgres://u:p@127.0.0.1:5433/solv_test',
    'postgres://u:p@localhost:55432/solv_migration_test',
    'postgres://u:p@[::1]:55432/solv_e2e'
  ]) {
    assert.equal(validateTestDatabaseURL(dsn), dsn);
  }
});

test('rechaza el endpoint del despliegue y destinos ambiguos', () => {
  for (const dsn of [
    'postgres://u:p@127.0.0.1:5432/solv_test',
    'postgres://u:p@localhost:5432/solv_db',
    'postgres://u:p@db.example:5433/solv_test',
    'postgres://u:p@127.0.0.1:5434/solv_test',
    'postgres://u:p@127.0.0.1:5433/solv_db',
    'postgres://u:p@127.0.0.1:5433/solv_test?dbname=solv_db',
    'postgres://u:p@127.0.0.1:5433/solv_test?host=db.example',
    'postgres://u:p@127.0.0.1:5433/solv_test?password=otra',
    'postgres://u@127.0.0.1:5433/solv_test'
  ]) {
    assert.throws(() => validateTestDatabaseURL(dsn), dsn);
  }
});

test('test-db.sh valida antes de invocar psql', () => {
  const result = spawnSync('bash', [SCRIPT, '--', 'true'], {
    env: {
      ...process.env,
      TEST_DB_HOST: '127.0.0.1',
      TEST_DB_PORT: '5432',
      TEST_DB_USER: 'solv_user',
      TEST_DB_PASSWORD: 'solv_password'
    },
    encoding: 'utf8'
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /5432/);
});