import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

/**
 * Recibos de ejecución.
 *
 * El digest del plan prueba qué se quiso ejecutar, no que el runner haya
 * ejecutado una suite. El recibo ata el informe a una ejecución concreta,
 * al plan, a la suite y al job que la produjo. En CI la clave debe venir de
 * un secreto del repositorio; en local se guarda en `.cache/receipt.key` y
 * solo protege contra ediciones accidentales o de terceros sin acceso a la
 * cuenta. Un agente con el mismo usuario puede leer esa clave: la garantía
 * local no es criptográfica frente a ese agente.
 */
export const RECEIPT_VERSION = 1;
export const RECEIPT_ALG = 'hmac-sha256';
export const RECEIPT_KEY_ENV = 'SOLV_TEST_RECEIPT_KEY';
export const TRUST_MODE_ENV = 'SOLV_TEST_TRUST_MODE';

const KEY_FILE = path.join('tools', 'test-impact', '.cache', 'receipt.key');

export const SUITE_JOB = {
  gates: 'lint',
  'asset-validation': 'test-assets',
  'backend-unit': 'test-backend',
  'backend-integration': 'test-backend-integration',
  'backend-docker': 'test-backend-docker',
  'frontend-unit': 'test-frontend',
  'frontend-e2e': 'test-frontend-e2e',
  migrations: 'migration-chain',
  build: 'build',
  security: 'security'
};

// Un mismo flujo puede ejecutarse en el pipeline principal o en el workflow
// E2E dedicado. La identidad válida es el job que realmente produjo el
// informe; no se acepta un job ajeno a esa suite.
export const SUITE_JOB_ALIASES = {
  'frontend-e2e': ['test-frontend-e2e', 'e2e']
};

function canonical(value) {
  if (Array.isArray(value)) return value.map(canonical);
  if (value !== null && typeof value === 'object') {
    return Object.fromEntries(
      Object.keys(value)
        .sort()
        .map((key) => [key, canonical(value[key])])
    );
  }
  return value;
}

function sha256(value) {
  return crypto.createHash('sha256').update(value).digest('hex');
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

export function isCI() {
  return process.env.CI === 'true' || process.env.CI === '1';
}

/**
 * En CI cada job usa una clave propia. `SOLV_TEST_TRUST_MODE=local` solo se usa
 * en los tests del motor y en ejecuciones locales explícitas: nunca se acepta
 * una clave genérica para fabricar un informe de un job de CI.
 */
export function trustMode() {
  if (process.env[TRUST_MODE_ENV] === 'local') return 'local';
  return isCI() ? 'ci' : 'local';
}

function keyFromName(name) {
  const value = String(process.env[name] || '').trim();
  return value || null;
}

function validateKey(value, name) {
  if (value.length < 16) {
    throw new Error(`${name} debe tener al menos 16 caracteres`);
  }
  return Buffer.from(value, 'utf8');
}

export function jobReceiptKeyEnv(jobId) {
  const normalized = String(jobId || '')
    .trim()
    .toUpperCase()
    .replace(/[^A-Z0-9]+/g, '_');
  if (!normalized) return null;
  return `${RECEIPT_KEY_ENV}_${normalized}`;
}

function localReceiptKey({ repoRoot, create }) {
  const fromEnv = keyFromName(RECEIPT_KEY_ENV);
  if (fromEnv) return validateKey(fromEnv, RECEIPT_KEY_ENV);
  const file = path.join(repoRoot, KEY_FILE);
  try {
    const value = fs.readFileSync(file, 'utf8').trim();
    if (value.length >= 16) return Buffer.from(value, 'utf8');
  } catch {
    // Se crea solo en ejecución local y con permisos restrictivos.
  }
  if (!create) {
    throw new Error(
      `no hay clave de recibos local; ejecutá el runner una vez o definí ${RECEIPT_KEY_ENV}`
    );
  }
  const value = crypto.randomBytes(32).toString('hex');
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, `${value}\n`, { mode: 0o600 });
  return Buffer.from(value, 'utf8');
}

/**
 * Carga la clave de recibos del job. En CI exige una clave por job; no existe
 * una clave genérica que permita a un job firmar el informe de otro. En local
 * se conserva la clave compartida, pero el contrato se declara local.
 */
export function loadReceiptKey({ repoRoot, create = false, jobId = 'local' } = {}) {
  if (trustMode() === 'ci') {
    if (!jobId || jobId === 'local') {
      throw new Error('un informe local no puede firmarse como evidencia de CI');
    }
    const name = jobReceiptKeyEnv(jobId);
    const value = name ? keyFromName(name) : null;
    if (!value) {
      throw new Error(
        `falta ${name || RECEIPT_KEY_ENV} para el job ${jobId}; configurá una clave de recibo por job`
      );
    }
    return validateKey(value, name);
  }
  return localReceiptKey({ repoRoot, create });
}

/**
 * Devuelve las claves disponibles para los jobs declarados y enumera las que
 * faltan. `aggregate` usa este resultado para rechazar un informe cuyo job no
 * tiene una clave independiente.
 */
export function loadReceiptKeys({ repoRoot, jobIds = [] } = {}) {
  const keys = {};
  const missing = [];
  for (const jobId of [...new Set(jobIds)]) {
    try {
      keys[jobId] = loadReceiptKey({ repoRoot, jobId });
    } catch {
      missing.push(jobId);
    }
  }
  return { keys, missing };
}

function reportPayload(report) {
  const payload = { ...report };
  delete payload.receipt;
  return canonical(payload);
}

/**
 * Digest del informe sin el recibo ni la referencia a su evidencia. La
 * evidencia lo usa para atarse al contenido exacto del informe sin crear un
 * ciclo de hashes.
 */
export function reportPayloadDigest(report) {
  const payload = { ...report };
  delete payload.receipt;
  delete payload.execution;
  return sha256(JSON.stringify(canonical(payload)));
}

export function executionEvidencePayload(evidence) {
  const payload = { ...evidence };
  delete payload.signature;
  return canonical(payload);
}

export function executionEvidenceDigest(evidence) {
  return sha256(JSON.stringify(executionEvidencePayload(evidence)));
}

export function signExecutionEvidence(evidence, key) {
  if (!Buffer.isBuffer(key) || key.length < 16) {
    throw new Error('clave de evidencia inválida');
  }
  return crypto
    .createHmac('sha256', key)
    .update(JSON.stringify(executionEvidencePayload(evidence)))
    .digest('hex');
}

export function attachExecutionEvidence(report, reportPath, key) {
  const evidence = {
    schema: 1,
    runId: report.runId,
    jobId: report.jobId,
    planDigest: report.plan.digest,
    reportDigest: reportPayloadDigest(report),
    startedAt: report.startedAt,
    finishedAt: report.finishedAt,
    suites: Object.fromEntries(
      Object.entries(report.suites).map(([suiteId, entry]) => [
        suiteId,
        {
          selected: entry.selected,
          state: entry.state,
          status: entry.status,
          executed: entry.executed,
          cached: entry.cached,
          simulated: entry.simulated,
          durationMs: entry.durationMs
        }
      ])
    )
  };
  evidence.signature = signExecutionEvidence(evidence, key);
  const evidencePath = `${reportPath}.evidence.json`;
  fs.mkdirSync(path.dirname(evidencePath), { recursive: true });
  fs.writeFileSync(evidencePath, `${JSON.stringify(evidence, null, 2)}\n`);
  report.execution = {
    schema: 1,
    evidence: path.basename(evidencePath),
    digest: executionEvidenceDigest(evidence)
  };
  return evidence;
}

export function verifyExecutionEvidence(evidence, report, key) {
  if (!isObject(evidence) || !isObject(report)) return false;
  if (evidence.schema !== 1) return false;
  if (evidence.runId !== report.runId || evidence.jobId !== report.jobId) return false;
  if (evidence.planDigest !== report.plan?.digest) return false;
  if (evidence.reportDigest !== reportPayloadDigest(report)) return false;
  if (!report.execution || report.execution.digest !== executionEvidenceDigest(evidence)) return false;
  if (!isObject(evidence.suites)) return false;
  for (const suiteId of report.required || []) {
    const reported = report.suites?.[suiteId];
    const recorded = evidence.suites[suiteId];
    if (!reported || !recorded) return false;
    if (recorded.state !== reported.state || recorded.status !== reported.status) return false;
    if (Boolean(recorded.executed) !== Boolean(reported.executed)) return false;
    if (Boolean(recorded.cached) !== Boolean(reported.cached)) return false;
  }
  if (typeof evidence.signature !== 'string' || evidence.signature.length !== 64) return false;
  const expected = signExecutionEvidence(evidence, key);
  const left = Buffer.from(expected, 'hex');
  const right = Buffer.from(evidence.signature, 'hex');
  return left.length === right.length && crypto.timingSafeEqual(left, right);
}

export function signReportReceipt(report, key) {
  if (!Buffer.isBuffer(key) || key.length < 16) {
    throw new Error('clave de recibo inválida');
  }
  const signature = crypto
    .createHmac('sha256', key)
    .update(JSON.stringify(reportPayload(report)))
    .digest('hex');
  return {
    version: RECEIPT_VERSION,
    alg: RECEIPT_ALG,
    runId: report.runId,
    jobId: report.jobId,
    signature
  };
}

export function verifyReportReceipt(report, key) {
  if (!isObject(report.receipt)) return false;
  const receipt = report.receipt;
  if (receipt.version !== RECEIPT_VERSION || receipt.alg !== RECEIPT_ALG) return false;
  if (receipt.runId !== report.runId || receipt.jobId !== report.jobId) return false;
  if (typeof receipt.signature !== 'string' || receipt.signature.length !== 64) return false;
  const expected = crypto
    .createHmac('sha256', key)
    .update(JSON.stringify(reportPayload(report)))
    .digest('hex');
  const left = Buffer.from(expected, 'hex');
  const right = Buffer.from(receipt.signature, 'hex');
  return left.length === right.length && crypto.timingSafeEqual(left, right);
}

function stable(value) {
  return JSON.stringify(canonical(value));
}

/**
 * Un informe solo puede cubrir el plan exacto con el que fue producido.
 * Devuelve la lista de discrepancias; no lanza para poder agregarlas al
 * resultado de `aggregate`.
 */
export function bindingProblems(report, plan, planDigestValue) {
  const problems = [];
  if (report.plan.digest !== planDigestValue) problems.push('el informe se generó con otro plan');
  if (report.plan.policyDigest !== plan.policyDigest) problems.push('la política no coincide');
  if (report.plan.policyVersion !== plan.policyVersion) problems.push('la versión de política no coincide');
  if (report.plan.contractVersion !== plan.contractVersion) {
    problems.push('el contrato de ejecución no coincide');
  }
  if (report.plan.level !== plan.level) problems.push('el nivel del plan no coincide');
  if (report.plan.mode !== plan.mode) problems.push('el modo del plan no coincide');
  if (stable(report.plan.source) !== stable(plan.source)) {
    problems.push('la procedencia del plan no coincide');
  }
  const expectedRequired = [...(plan.requiredSuites || [])].sort();
  const actualRequired = [...(report.required || [])].sort();
  if (stable(expectedRequired) !== stable(actualRequired)) {
    problems.push('las suites obligatorias no coinciden con el plan');
  }
  const expectedSuites = Object.keys(plan.suites || {}).sort();
  const actualSuites = Object.keys(report.suites || {}).sort();
  if (stable(expectedSuites) !== stable(actualSuites)) {
    problems.push('el inventario de suites no coincide con el plan');
  }
  for (const suiteId of expectedSuites) {
    const expected = plan.suites[suiteId];
    const actual = report.suites?.[suiteId];
    if (!actual) continue;
    if (Boolean(actual.selected) !== Boolean(expected.selected)) {
      problems.push(`la selección de ${suiteId} no coincide con el plan`);
    }
  }
  return problems;
}

export function assertReportBinding(report, plan, planDigestValue) {
  const problems = bindingProblems(report, plan, planDigestValue);
  if (problems.length > 0) {
    throw new Error(`Informe de testing inválido: ${problems.join('; ')}`);
  }
}

export function receiptFingerprint(key) {
  return sha256(key);
}
