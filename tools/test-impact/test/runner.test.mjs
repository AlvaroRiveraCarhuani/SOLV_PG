import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { computeCacheKey } from '../lib/cache.mjs';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const RUNNER = path.join(REPO_ROOT, 'tools/test-impact/run.mjs');
const ICON = 'frontend/public/tech-icons/python.svg';
const SECOND_ICON = 'frontend/public/tech-icons/angular.svg';

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

function reportFile(directory, name) {
  return path.join(directory, name);
}

test('el ejecutor no corre suites que el plan no seleccionó', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-run-'));
  const reportPath = reportFile(directory, 'report.json');
  try {
    const result = runner([
      '--all', '--files', `M:${ICON}`, '--dry-run', '--report', reportPath
    ]);
    assert.equal(result.status, 2, result.stderr);
    const report = JSON.parse(fs.readFileSync(reportPath, 'utf8'));
    assert.equal(report.level, 'L0');
    assert.equal(report.totals.selected, 1);
    assert.equal(report.suites['asset-validation'].selected, true);
    assert.equal(report.suites['backend-unit'].selected, false);
    assert.equal(report.suites['backend-integration'].selected, false);
    assert.equal(report.suites['backend-docker'].selected, false);
    assert.equal(report.suites['frontend-e2e'].selected, false);
    assert.equal(report.suites.build.selected, false);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('--dry-run puede recorrer el plan completo sin abrir infraestructura', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-dry-'));
  const reportPath = reportFile(directory, 'report.json');
  try {
    const result = runner([
      '--all', '--full', '--working-tree', '--dry-run', '--report', reportPath
    ]);
    assert.equal(result.status, 2, result.stderr);
    const report = JSON.parse(fs.readFileSync(reportPath, 'utf8'));
    assert.equal(report.dryRun, true);
    assert.equal(report.totals.selected, 10);
    assert.equal(report.totals.failed, 0);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('un resultado determinista se reutiliza solo con la misma clave', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-cache-'));
  const cacheDirectory = path.join(directory, 'cache');
  const firstReport = reportFile(directory, 'first.json');
  const secondReport = reportFile(directory, 'second.json');
  try {
    const args = [
      '--partial', '--suite', 'asset-validation', '--files', `M:${ICON}`,
      '--cache', '--cache-dir', cacheDirectory
    ];
    const first = runner([...args, '--report', firstReport]);
    assert.equal(first.status, 0, first.stderr);
    const second = runner([...args, '--report', secondReport]);
    assert.equal(second.status, 0, second.stderr);
    const firstData = JSON.parse(fs.readFileSync(firstReport, 'utf8'));
    const secondData = JSON.parse(fs.readFileSync(secondReport, 'utf8'));
    assert.equal(firstData.suites['asset-validation'].cached, false);
    assert.equal(firstData.suites['asset-validation'].executed, true);
    assert.equal(secondData.suites['asset-validation'].cached, true);
    assert.equal(secondData.suites['asset-validation'].executed, false);
    assert.equal(secondData.totals.cached, 1);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('un recurso ausente falla y queda marcado en el informe', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-fail-'));
  const reportPath = reportFile(directory, 'report.json');
  try {
    const result = runner([
      '--partial', '--suite', 'asset-validation',
      '--files', 'M:frontend/public/tech-icons/no-existe.svg',
      '--report', reportPath
    ]);
    assert.equal(result.status, 1);
    const report = JSON.parse(fs.readFileSync(reportPath, 'utf8'));
    assert.equal(report.suites['asset-validation'].status, 1);
    assert.equal(report.totals.failed, 1);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('cambiar el plan invalida la caché anterior', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-cache-key-'));
  const cacheDirectory = path.join(directory, 'cache');
  const firstReport = reportFile(directory, 'first.json');
  const secondReport = reportFile(directory, 'second.json');
  try {
    const first = runner([
      '--partial', '--suite', 'asset-validation', '--files', `M:${ICON}`,
      '--cache', '--cache-dir', cacheDirectory, '--report', firstReport
    ]);
    const second = runner([
      '--partial', '--suite', 'asset-validation', '--files', `M:${ICON},M:${SECOND_ICON}`,
      '--cache', '--cache-dir', cacheDirectory, '--report', secondReport
    ]);
    assert.equal(first.status, 0, first.stderr);
    assert.equal(second.status, 0, second.stderr);
    const secondData = JSON.parse(fs.readFileSync(secondReport, 'utf8'));
    assert.equal(secondData.suites['asset-validation'].cached, false);
    assert.equal(secondData.suites['asset-validation'].executed, true);
    assert.equal(secondData.suites['asset-validation'].files.length, 2);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test('cambiar el contenido de un archivo invalida la clave de caché', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'solv-cache-content-'));
  try {
    const git = (...args) => spawnSync('git', args, { cwd: directory, encoding: 'utf8' });
    assert.equal(git('init', '-q').status, 0);
    git('config', 'user.email', 'test@solv.test');
    git('config', 'user.name', 'SOLV Test');
    const file = path.join(directory, 'sample.txt');
    fs.writeFileSync(file, 'primera versión\n');
    git('add', 'sample.txt');
    assert.equal(git('commit', '-q', '-m', 'base').status, 0);
    const plan = { mode: 'targeted', level: 'L0', suites: { 'asset-validation': { selected: true } } };
    const first = computeCacheKey(directory, plan);
    fs.writeFileSync(file, 'segunda versión\n');
    const second = computeCacheKey(directory, plan);
    assert.notEqual(first, second);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});