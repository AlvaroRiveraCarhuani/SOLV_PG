import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { collectChanges, parseNameStatus } from '../lib/git.mjs';

function git(cwd, args) {
  return execFileSync('git', ['-C', cwd, ...args], { encoding: 'utf8' }).trim();
}

function createRepo() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-impact-git-'));
  git(root, ['init', '-b', 'main']);
  git(root, ['config', 'user.email', 'impact@solv.test']);
  git(root, ['config', 'user.name', 'Impact Selector']);
  fs.mkdirSync(path.join(root, 'backend', 'internal'), { recursive: true });
  fs.mkdirSync(path.join(root, 'docs'), { recursive: true });
  fs.writeFileSync(path.join(root, 'backend', 'internal', 'a.go'), 'package internal\n');
  fs.writeFileSync(path.join(root, 'backend', 'internal', 'b.go'), 'package internal\n');
  fs.writeFileSync(path.join(root, 'docs', 'guia.md'), '# Guía\n');
  git(root, ['add', '-A']);
  git(root, ['commit', '-m', 'base']);
  return root;
}

test('parseNameStatus entiende modificaciones, eliminaciones y renombrados', () => {
  const changes = parseNameStatus('M\tbackend/a.go\nD\tdocs/viejo.md\nR100\tbackend/b.go\tbackend/c.go\n');
  assert.deepEqual(changes, [
    { status: 'M', path: 'backend/a.go' },
    { status: 'D', path: 'docs/viejo.md' },
    { status: 'R', oldPath: 'backend/b.go', path: 'backend/c.go' }
  ]);
});

test('collectChanges detecta el diff real contra la base', () => {
  const root = createRepo();
  const base = git(root, ['rev-parse', 'HEAD']);
  fs.writeFileSync(path.join(root, 'backend', 'internal', 'a.go'), 'package internal\n// cambio\n');
  git(root, ['mv', 'backend/internal/b.go', 'backend/internal/c.go']);
  fs.rmSync(path.join(root, 'docs', 'guia.md'));
  git(root, ['add', '-A']);
  git(root, ['commit', '-m', 'cambio']);

  const result = collectChanges({ repoRoot: root, base, head: 'HEAD' });
  assert.equal(result.error, null);
  const byPath = new Map(result.changes.map((change) => [change.path, change]));
  assert.equal(byPath.get('backend/internal/a.go').status, 'M');
  assert.equal(byPath.get('backend/internal/c.go').status, 'R');
  assert.equal(byPath.get('backend/internal/c.go').oldPath, 'backend/internal/b.go');
  assert.equal(byPath.get('docs/guia.md').status, 'D');
});

test('collectChanges incluye cambios sin commit cuando se pide el working tree', () => {
  const root = createRepo();
  const base = git(root, ['rev-parse', 'HEAD']);
  fs.writeFileSync(path.join(root, 'backend', 'internal', 'a.go'), 'package internal\n// sin commit\n');
  fs.writeFileSync(path.join(root, 'backend', 'internal', 'nuevo.go'), 'package internal\n');

  const result = collectChanges({ repoRoot: root, base, head: 'HEAD', includeWorkingTree: true });
  const paths = result.changes.map((change) => change.path);
  assert.ok(paths.includes('backend/internal/a.go'));
  assert.ok(paths.includes('backend/internal/nuevo.go'));
});

test('collectChanges incluye cambios preparados aunque no estén en HEAD', () => {
  const root = createRepo();
  const base = git(root, ['rev-parse', 'HEAD']);
  fs.writeFileSync(path.join(root, 'backend', 'internal', 'a.go'), 'package internal\n// staged\n');
  fs.rmSync(path.join(root, 'docs', 'guia.md'));
  git(root, ['add', '-A']);

  const result = collectChanges({ repoRoot: root, base, head: 'HEAD', includeWorkingTree: true });
  const byPath = new Map(result.changes.map((change) => [change.path, change]));
  assert.equal(byPath.get('backend/internal/a.go').status, 'M');
  assert.equal(byPath.get('docs/guia.md').status, 'D');
});

test('una base inexistente no se interpreta como diff vacío', () => {
  const root = createRepo();
  const result = collectChanges({ repoRoot: root, base: 'no-existe', head: 'HEAD' });
  assert.match(result.error, /no existe/);
  assert.deepEqual(result.changes, []);
});

test('sin base no se inventa un diff', () => {
  const root = createRepo();
  const result = collectChanges({ repoRoot: root, base: null, head: 'HEAD' });
  assert.match(result.error, /sin base/);
});
