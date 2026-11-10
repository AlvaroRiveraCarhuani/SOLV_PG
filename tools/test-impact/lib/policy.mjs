import fs from 'node:fs';
import path from 'node:path';
import { globToRegExp, matchesAny, walkFiles } from './util.mjs';

export const POLICY_PATH = 'tools/test-impact/policy.json';

const UNIT_VALUES = new Set(['all', false, null]);
const LEVELS = new Set(['L0', 'L1', 'L2', 'L3']);

export function loadPolicy(repoRoot) {
  const file = path.join(repoRoot, POLICY_PATH);
  const policy = JSON.parse(fs.readFileSync(file, 'utf8'));
  validatePolicy(policy, repoRoot);
  return policy;
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(`Política de testing inválida: ${message}`);
  }
}

function validateId(entry, kind) {
  assert(typeof entry.id === 'string' && entry.id.length > 0, `${kind} sin id`);
  assert(Array.isArray(entry.globs) && entry.globs.length > 0, `${entry.id} sin globs`);
  for (const glob of entry.globs) {
    assert(typeof glob === 'string' && glob.length > 0, `${entry.id} tiene un glob vacío`);
    globToRegExp(glob);
  }
}

export function validatePolicy(policy, repoRoot) {
  assert(policy && typeof policy === 'object', 'el archivo no contiene un objeto');
  assert(Number.isInteger(policy.version) && policy.version > 0, 'falta version');
  assert(policy.suites && typeof policy.suites === 'object', 'faltan suites');
  assert(Array.isArray(policy.areas) && policy.areas.length > 0, 'faltan areas');
  assert(Array.isArray(policy.critical), 'faltan critical');
  assert(Array.isArray(policy.globalTriggers), 'faltan globalTriggers');
  assert(policy.levels && typeof policy.levels === 'object', 'faltan levels');
  for (const level of LEVELS) {
    assert(policy.levels[level], `falta levels.${level}`);
    assert(typeof policy.levels[level].title === 'string', `levels.${level} sin title`);
    assert(
      typeof policy.levels[level].description === 'string',
      `levels.${level} sin description`
    );
  }
  assert(policy.e2e && typeof policy.e2e.specs === 'object', 'faltan e2e.specs');
  assert(
    Number.isInteger(policy.limits?.maxChangedFiles) && policy.limits.maxChangedFiles > 0,
    'falta limits.maxChangedFiles'
  );

  const seen = new Set();
  const globOwners = new Map();
  for (const entry of [...policy.globalTriggers, ...policy.critical, ...policy.areas]) {
    validateId(entry, 'entrada');
    assert(!seen.has(entry.id), `id duplicado: ${entry.id}`);
    seen.add(entry.id);
    for (const glob of entry.globs) {
      const owner = globOwners.get(glob);
      assert(
        owner === undefined,
        `glob duplicado entre ${owner} y ${entry.id}: ${glob}; las reglas deben ser específicas y no depender del orden`
      );
      globOwners.set(glob, entry.id);
    }
  }

  for (const trigger of policy.globalTriggers) {
    assert(
      ['all', 'backend', 'frontend'].includes(trigger.scope),
      `${trigger.id} tiene scope inválido`
    );
    assert(LEVELS.has(trigger.level), `${trigger.id}.level inválido`);
    assert(typeof trigger.reason === 'string' && trigger.reason.length > 0, `${trigger.id} sin reason`);
  }

  for (const entry of policy.critical) {
    for (const key of ['backendUnit', 'frontendUnit']) {
      assert(
        entry[key] === undefined || UNIT_VALUES.has(entry[key]),
        `${entry.id}.${key} debe ser "all" o false`
      );
    }
    for (const key of ['backendIntegration', 'backendDocker', 'migrations', 'e2eAll']) {
      assert(
        entry[key] === undefined || typeof entry[key] === 'boolean',
        `${entry.id}.${key} debe ser booleano`
      );
    }
    assert(LEVELS.has(entry.level), `${entry.id}.level inválido`);
    assert(entry.e2e === undefined || Array.isArray(entry.e2e), `${entry.id}.e2e debe ser una lista`);
    assert(typeof entry.reason === 'string' && entry.reason.length > 0, `${entry.id} sin reason`);
    assert(
      entry.coverageGaps === undefined ||
        (Array.isArray(entry.coverageGaps) &&
          entry.coverageGaps.every((gap) => typeof gap === 'string' && gap.trim().length > 0)),
      `${entry.id}.coverageGaps debe ser una lista de textos`
    );
    for (const spec of entry.e2e || []) {
      assert(policy.e2e.specs[spec], `${entry.id} referencia un flujo E2E desconocido: ${spec}`);
    }
  }

  for (const area of policy.areas) {
    for (const key of ['backend', 'frontend', 'integration', 'build', 'frontendUnitAll', 'e2eAll', 'e2eSelf', 'gates']) {
      assert(
        area[key] === undefined || typeof area[key] === 'boolean',
        `${area.id}.${key} debe ser booleano`
      );
    }
    assert(LEVELS.has(area.level), `${area.id}.level inválido`);
    if (area.e2e !== undefined) {
      assert(Array.isArray(area.e2e), `${area.id}.e2e debe ser una lista`);
      for (const spec of area.e2e) {
        assert(policy.e2e.specs[spec], `${area.id} referencia un flujo E2E desconocido: ${spec}`);
      }
    }
  }

  const knownSuites = new Set(Object.keys(policy.suites));
  for (const suite of knownSuites) {
    assert(typeof policy.suites[suite].title === 'string', `suite ${suite} sin title`);
    assert(
      policy.suites[suite].cacheable === undefined ||
        typeof policy.suites[suite].cacheable === 'boolean',
      `suite ${suite}.cacheable debe ser booleano`
    );
  }

  if (repoRoot) {
    const e2eRoot = path.join(repoRoot, 'frontend', 'e2e');
    const discovered = walkFiles(e2eRoot, { skipDirs: ['test-results', 'node_modules'], extensions: ['.mjs'] })
      .filter((file) => file.endsWith('.spec.mjs'))
      .map((file) => path.relative(path.join(repoRoot, 'frontend'), file).split(path.sep).join('/'));
    const declared = Object.keys(policy.e2e.specs);
    const missing = discovered.filter((spec) => !declared.includes(spec));
    const unknown = declared.filter((spec) => !discovered.includes(spec));
    assert(
      missing.length === 0,
      `hay flujos E2E sin clasificar (se omite cobertura): ${missing.join(', ')}`
    );
    assert(unknown.length === 0, `la política declara flujos E2E inexistentes: ${unknown.join(', ')}`);
  }
}

export function matchingEntries(entries, filePath) {
  return entries.filter((entry) => matchesAny(entry.globs, filePath));
}

export function e2eSpecMetadata(policy, spec) {
  const metadata = policy.e2e.specs[spec];
  if (!metadata) return null;
  return {
    spec,
    config: metadata.config || policy.e2e.defaultConfig
  };
}

export function allE2eSpecs(policy) {
  return Object.keys(policy.e2e.specs).map((spec) => e2eSpecMetadata(policy, spec));
}
