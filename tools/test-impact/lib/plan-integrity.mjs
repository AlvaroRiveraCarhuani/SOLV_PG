import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { collectChanges, resolveCommit } from './git.mjs';
import { createPlan } from './select.mjs';
import { isGeneratedArtifact } from './artifacts.mjs';

export { isGeneratedArtifact };

// Bumpear estos valores cuando cambie el contrato del plan o la forma de
// seleccionar. Un plan de una versión anterior no se puede ejecutar.
export const PLAN_SCHEMA = 2;
export const PLAN_CONTRACT_VERSION = 'e7.2';

function fail(message) {
  throw new Error(`Plan de testing inválido: ${message}`);
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function stable(value) {
  return JSON.stringify(value);
}

function rawChanges(changes) {
  return (changes || [])
    .map((change) => ({
      status: String(change.status || 'M').charAt(0),
      path: change.path,
      oldPath: change.oldPath || null
    }))
    .sort((left, right) =>
      `${left.path}:${left.oldPath || ''}:${left.status}`.localeCompare(
        `${right.path}:${right.oldPath || ''}:${right.status}`
      )
    );
}

function suitePayload(entry) {
  if (!isObject(entry)) fail('una suite no es un objeto');
  if (typeof entry.selected !== 'boolean') fail(`suite sin selected booleano: ${entry.reason || 'desconocida'}`);
  if (typeof entry.reason !== 'string') fail('suite sin reason textual');
  const payload = { selected: entry.selected };
  for (const field of ['packages', 'specs', 'files', 'full', 'backend', 'frontend']) {
    if (Object.hasOwn(entry, field)) payload[field] = entry[field];
  }
  return payload;
}

export function policyDigest(repoRoot) {
  const file = path.join(repoRoot, 'tools', 'test-impact', 'policy.json');
  return crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
}

function canonical(value) {
  if (Array.isArray(value)) return value.map(canonical);
  if (isObject(value)) {
    return Object.fromEntries(
      Object.keys(value)
        .sort()
        .map((key) => [key, canonical(value[key])])
    );
  }
  return value;
}

/**
 * Identidad estable del plan. Liga un informe a las suites, decisiones y
 * procedencia concretas, no solo a la política. Cambiar el plan o reutilizar el
 * informe de otro diff invalida la agregación de CI.
 */
export function planDigest(plan) {
  const payload = {
    schema: plan.schema,
    contractVersion: plan.contractVersion,
    policyVersion: plan.policyVersion,
    policyDigest: plan.policyDigest,
    source: plan.source,
    mode: plan.mode,
    level: plan.level,
    levelTitle: plan.levelTitle,
    changed: plan.changed,
    criticalAreas: plan.criticalAreas,
    coverageGaps: plan.coverageGaps,
    uncovered: plan.uncovered,
    unknown: plan.unknown,
    suites: plan.suites,
    decisions: plan.decisions
  };
  return crypto.createHash('sha256').update(JSON.stringify(canonical(payload))).digest('hex');
}

function validateSource(source, policy) {
  if (!isObject(source)) fail('falta source con la procedencia del diff');
  if (source.kind !== 'git' && source.kind !== 'files') {
    fail(`source.kind desconocido: ${String(source.kind)}`);
  }
  if (source.kind === 'files') {
    if (!Array.isArray(source.files) || source.files.length === 0) {
      fail('source.files debe contener al menos un archivo sintético');
    }
    for (const file of source.files) {
      if (typeof file !== 'string' || !file.trim()) fail('source.files contiene un archivo vacío');
    }
  }
  if (source.kind === 'git' && source.full !== true) {
    const base = source.baseCommit || source.base;
    if (typeof base !== 'string' || !base.trim()) {
      fail('source.base o source.baseCommit es obligatorio para un plan git no completo');
    }
  }
  if (source.baseCommit !== undefined && source.baseCommit !== null) {
    if (typeof source.baseCommit !== 'string' || !source.baseCommit.trim()) {
      fail('source.baseCommit debe ser un SHA textual');
    }
  }
  if (source.headCommit !== undefined && source.headCommit !== null) {
    if (typeof source.headCommit !== 'string' || !source.headCommit.trim()) {
      fail('source.headCommit debe ser un SHA textual');
    }
  }
  if (typeof source.event !== 'string') fail('source.event debe ser textual');
  if (source.includeWorkingTree !== undefined && typeof source.includeWorkingTree !== 'boolean') {
    fail('source.includeWorkingTree debe ser booleano');
  }
  if (source.maxChangedFiles !== undefined && source.maxChangedFiles !== null) {
    if (!Number.isInteger(source.maxChangedFiles) || source.maxChangedFiles < 1) {
      fail('source.maxChangedFiles debe ser un entero positivo');
    }
    if (source.maxChangedFiles > policy.limits.maxChangedFiles) {
      fail('source.maxChangedFiles no puede superar el límite de la política');
    }
  }
}

/**
 * El plan declara una procedencia, pero el runner debe comprobar que esa
 * procedencia sea la del checkout real. Sin esta comprobación un plan de un
 * commit antiguo puede ejecutar cero suites y presentarse como completo.
 */
function assertSourceMatchesCheckout(repoRoot, plan) {
  const source = plan.source;
  if (source.kind !== 'git') return;

  const currentHead = resolveCommit(repoRoot, 'HEAD');
  if (!source.headCommit || source.headCommit !== currentHead) {
    fail(
      `source.headCommit ${String(source.headCommit)} no corresponde al HEAD actual ${String(currentHead)}`
    );
  }

  const expectedEvent = process.env.GITHUB_EVENT_NAME;
  if (expectedEvent && source.event !== expectedEvent) {
    fail(`source.event ${String(source.event)} no corresponde al evento ${expectedEvent}`);
  }

  const dirty = collectChanges({
    repoRoot,
    base: currentHead,
    head: currentHead,
    includeWorkingTree: true
  });
  if (dirty.error) fail(`no se pudo comprobar el working tree: ${dirty.error}`);
  const pending = dirty.changes.filter(
    (change) => !isGeneratedArtifact(change.path) || change.status !== 'A'
  );
  if (source.includeWorkingTree !== true && pending.length > 0) {
    fail(
      'el working tree tiene cambios no representados en el plan; regenerá el plan con --working-tree'
    );
  }
}

function changesFromSource(repoRoot, source) {
  if (source.kind === 'files') {
    const selection = collectChanges({
      repoRoot,
      files: source.files.map((entry) => entry.trim()),
      includeWorkingTree: false
    });
    if (selection.error) fail(selection.error);
    return { changes: selection.changes, synthetic: true };
  }
  const base = source.baseCommit || source.base;
  const head = source.headCommit || source.head || 'HEAD';
  const selection = collectChanges({
    repoRoot,
    base,
    head,
    includeWorkingTree: source.includeWorkingTree === true
  });
  if (selection.error) {
    // Un plan completo generado sin base verificable no depende del diff. Si
    // la base sí existe, se reconstruye igual para comparar los archivos.
    if (source.full === true) return { changes: [], synthetic: false };
    fail(`no se pudo reconstruir el diff: ${selection.error}`);
  }
  return { changes: selection.changes, synthetic: false };
}

function assertSuiteContract(plan, expected) {
  if (!isObject(plan.suites)) fail('suites no es un objeto');
  const expectedIds = Object.keys(expected.suites).sort();
  const actualIds = Object.keys(plan.suites).sort();
  if (stable(expectedIds) !== stable(actualIds)) {
    const missing = expectedIds.filter((id) => !actualIds.includes(id));
    const unknown = actualIds.filter((id) => !expectedIds.includes(id));
    fail(`inventario de suites incoherente; faltan [${missing.join(', ')}], sobran [${unknown.join(', ')}]`);
  }

  for (const id of expectedIds) {
    const actual = plan.suites[id];
    const wanted = expected.suites[id];
    const actualPayload = suitePayload(actual);
    const wantedPayload = suitePayload(wanted);
    if (stable(actualPayload) !== stable(wantedPayload)) {
      fail(`decisiones o alcance alterados en la suite ${id}`);
    }
  }
}

function assertDecisions(plan, expected) {
  if (!isObject(plan.decisions)) fail('faltan decisions');
  const expectedLevel = expected.decisions?.level;
  if (expectedLevel) {
    if (!isObject(plan.decisions.level) || plan.decisions.level.id !== expected.level) {
      fail('decisions.level no coincide con el nivel calculado');
    }
  }
  const expectedSuites = expected.decisions?.suites || {};
  const actualSuites = plan.decisions.suites || {};
  if (!isObject(actualSuites)) fail('decisions.suites no es un objeto');
  const expectedIds = Object.keys(expectedSuites).sort();
  const actualIds = Object.keys(actualSuites).sort();
  if (stable(expectedIds) !== stable(actualIds)) {
    fail('decisions.suites no cubre el inventario de suites');
  }
  for (const id of expectedIds) {
    if (actualSuites[id]?.selected !== expectedSuites[id].selected) {
      fail(`decisiones.suites.${id} no coincide con la obligación`);
    }
  }
}

export function validatePlan({ repoRoot, plan, policy, backendGraph, frontendGraph }) {
  if (!isObject(plan)) fail('el archivo no contiene un objeto de plan');
  if (plan.schema !== undefined && plan.schema !== PLAN_SCHEMA) {
    fail(`schema ${plan.schema} incompatible; se esperaba ${PLAN_SCHEMA}`);
  }
  if (plan.contractVersion !== PLAN_CONTRACT_VERSION) {
    fail(`contrato ${String(plan.contractVersion)} incompatible; regenerá el plan`);
  }
  if (plan.policyVersion !== policy.version) {
    fail(`política ${String(plan.policyVersion)} incompatible con la vigente ${policy.version}`);
  }
  const digest = policyDigest(repoRoot);
  if (plan.policyDigest !== digest) {
    fail('la política cambió después de generar el plan');
  }

  validateSource(plan.source, policy);
  assertSourceMatchesCheckout(repoRoot, plan);
  const reconstructed = changesFromSource(repoRoot, plan.source);
  const expected = createPlan({
    policy,
    changes: reconstructed.changes,
    backendGraph,
    frontendGraph,
    options: {
      event: plan.source.event,
      base: plan.source.baseCommit || plan.source.base || null,
      head: plan.source.headCommit || plan.source.head || 'HEAD',
      full: plan.source.full === true,
      maxChangedFiles: plan.source.maxChangedFiles
    }
  });

  const expectedChanges = rawChanges(expected.changed);
  const actualChanges = rawChanges(plan.changed);
  if (stable(expectedChanges) !== stable(actualChanges)) {
    fail('los archivos modificados del plan no coinciden con el diff reconstruido');
  }
  if (plan.mode !== expected.mode) fail(`modo ${plan.mode} no coincide con ${expected.mode}`);
  if (plan.level !== expected.level) fail(`nivel ${plan.level} no coincide con ${expected.level}`);
  if (stable(plan.criticalAreas || []) !== stable(expected.criticalAreas || [])) {
    fail('las reglas críticas del plan no coinciden con las obligaciones vigentes');
  }
  if (stable(plan.coverageGaps || []) !== stable(expected.coverageGaps || [])) {
    fail('la cobertura faltante declarada no coincide con las reglas críticas vigentes');
  }
  if (plan.levelTitle !== expected.levelTitle) {
    fail('el título del nivel no coincide con la política vigente');
  }
  if (stable(plan.unknown || []) !== stable(expected.unknown || [])) {
    fail('las rutas desconocidas del plan no coinciden con el diff');
  }
  if (stable(plan.uncovered || {}) !== stable(expected.uncovered || {})) {
    fail('la cobertura faltante del plan no coincide con el diff');
  }
  assertSuiteContract(plan, expected);
  assertDecisions(plan, expected);

  return {
    ...expected,
    schema: PLAN_SCHEMA,
    contractVersion: PLAN_CONTRACT_VERSION,
    policyDigest: digest,
    source: plan.source,
    synthetic: reconstructed.synthetic,
    requiredSuites: Object.entries(expected.suites)
      .filter(([, entry]) => entry.selected)
      .map(([id]) => id)
      .sort()
  };
}