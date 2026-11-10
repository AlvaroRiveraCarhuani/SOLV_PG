import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

export const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
export const CLI = path.join(REPO_ROOT, 'tools/test-impact/cli.mjs');
export const RUNNER = path.join(REPO_ROOT, 'tools/test-impact/run.mjs');
export const FORM =
  'frontend/src/app/features/admin/courses/components/course-create-modal/course-create-modal.component.ts';
export const ICON = 'frontend/public/tech-icons/python.svg';

/**
 * Repositorio temporal con un plan git real. Los tests de integridad no pueden
 * usar el árbol auditado porque tiene cambios preexistentes: el fixture crea un
 * checkout limpio, conserva el motor actual y agrega un cambio de formulario
 * ya commiteado.
 */
export function createGitFixture() {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-e7-git-fixture-'));
  const archive = spawnSync('git', ['archive', 'HEAD'], {
    cwd: REPO_ROOT,
    encoding: 'buffer',
    maxBuffer: 64 * 1024 * 1024
  });
  assert.equal(archive.status, 0, archive.error?.message || archive.stderr?.toString());
  const tar = spawnSync('tar', ['-x', '-C', directory], { input: archive.stdout });
  assert.equal(tar.status, 0, tar.stderr?.toString());
  fs.rmSync(path.join(directory, 'tools', 'test-impact'), { recursive: true, force: true });
  fs.cpSync(path.join(REPO_ROOT, 'tools', 'test-impact'), path.join(directory, 'tools', 'test-impact'), {
    recursive: true
  });
  const git = (...args) => spawnSync('git', args, { cwd: directory, encoding: 'utf8' });
  assert.equal(git('init', '-q').status, 0);
  assert.equal(git('config', 'user.email', 'test@solv.test').status, 0);
  assert.equal(git('config', 'user.name', 'SOLV Test').status, 0);
  assert.equal(git('add', '.').status, 0);
  assert.equal(git('commit', '-q', '-m', 'fixture base').status, 0);
  // El fixture representa un checkout limpio: los archivos ignorados que
  // `git archive` no trajo no deben convertirse en “cambios” del test.
  assert.equal(git('clean', '-fdX').status, 0);

  const form = path.join(directory, FORM);
  fs.mkdirSync(path.dirname(form), { recursive: true });
  fs.writeFileSync(form, '// cambio de formulario del fixture\n');
  assert.equal(git('add', FORM).status, 0);
  assert.equal(git('commit', '-q', '-m', 'fixture form').status, 0);
  return directory;
}

export function planInFixture(directory, args) {
  const output = path.join(os.tmpdir(), `solv-e7-plan-${crypto.randomUUID()}.json`);
  const result = spawnSync(
    process.execPath,
    [CLI, 'plan', '--repo', directory, '--format', 'json', '--output', output, ...args],
    { cwd: REPO_ROOT, encoding: 'utf8', timeout: 120_000 }
  );
  assert.equal(result.status, 0, result.stderr);
  return output;
}

export function cliWithRepo(args, repo = REPO_ROOT, env = process.env) {
  return spawnSync(process.execPath, [CLI, '--repo', repo, ...args], {
    cwd: REPO_ROOT,
    env,
    encoding: 'utf8',
    timeout: 120_000
  });
}

export function runnerWithRepo(args, repo = REPO_ROOT, env = process.env) {
  return spawnSync(process.execPath, [RUNNER, '--repo', repo, ...args], {
    cwd: REPO_ROOT,
    env,
    encoding: 'utf8',
    timeout: 120_000
  });
}
