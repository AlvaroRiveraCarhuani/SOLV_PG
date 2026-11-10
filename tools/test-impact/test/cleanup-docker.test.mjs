import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SCRIPT = path.resolve(HERE, '..', 'cleanup-docker.sh');

function fakeDocker(t) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-cleanup-'));
  const log = path.join(directory, 'calls.log');
  const executable = path.join(directory, 'docker');
  fs.writeFileSync(
    executable,
    `#!/usr/bin/env bash\nprintf '%s\\n' "$*" >> "${log}"\ncase "$1 $2" in\n  "info "*) exit 0 ;;\n  "ps -aq") exit 0 ;;\n  "volume ls") exit 0 ;;\n  "network ls") exit 0 ;;\nesac\nexit 0\n`,
    { mode: 0o755 }
  );
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  return { directory, log };
}

test('rechaza limpiar sin un run ID acotado', () => {
  const result = spawnSync('bash', [SCRIPT], { env: { ...process.env, SOLV_TEST_RUN_ID: '' }, encoding: 'utf8' });
  assert.equal(result.status, 2);
});

test('rechaza un run ID inválido', () => {
  const result = spawnSync('bash', [SCRIPT], { env: { ...process.env, SOLV_TEST_RUN_ID: 'run/ajeno' }, encoding: 'utf8' });
  assert.equal(result.status, 2);
});

test('filtra por etiqueta y no usa prune ni limpieza global', (t) => {
  const fake = fakeDocker(t);
  const result = spawnSync('bash', [SCRIPT], {
    env: { ...process.env, PATH: `${fake.directory}:${process.env.PATH}`, SOLV_TEST_RUN_ID: 'run-123' },
    encoding: 'utf8'
  });
  assert.equal(result.status, 0, result.stderr);
  const calls = fs.readFileSync(fake.log, 'utf8');
  assert.match(calls, /ps -aq --filter label=solv\.test\.run=run-123/);
  assert.doesNotMatch(calls, /prune|system df|system events/);
});