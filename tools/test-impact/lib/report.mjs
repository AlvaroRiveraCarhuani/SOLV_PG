/**
 * Contrato del informe de ejecución.
 *
 * Un informe describe qué exigía el plan, qué se ejecutó de verdad, qué se
 * simuló, qué vino de caché, qué se omitió por irrelevancia y qué sigue
 * pendiente. `validated` solo puede ser true cuando todas las suites
 * obligatorias terminaron correctamente en una ejecución real y completa.
 */
import {
  SUITE_JOB,
  SUITE_JOB_ALIASES,
  bindingProblems,
  verifyExecutionEvidence,
  verifyReportReceipt
} from './receipt.mjs';

export const REPORT_SCHEMA = 4;

export const REPORT_STATES = [
  'executed',
  'cached',
  'simulated',
  'skipped',
  'pending',
  'failed'
];

const STATE_SET = new Set(REPORT_STATES);
const VALIDATION_STATUS = new Set([
  'complete',
  'partial',
  'failed',
  'simulated',
  'synthetic',
  'incomplete',
  'rejected'
]);

function fail(message) {
  throw new Error(`Informe de testing inválido: ${message}`);
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function isCount(value) {
  return Number.isInteger(value) && value >= 0;
}

export function summarizeReport(report) {
  const entries = Object.values(report.suites || {});
  return {
    required: Array.isArray(report.required) ? report.required.length : 0,
    selected: entries.filter((entry) => entry.selected).length,
    executed: entries.filter((entry) => entry.state === 'executed').length,
    cached: entries.filter((entry) => entry.state === 'cached').length,
    simulated: entries.filter((entry) => entry.state === 'simulated').length,
    skipped: entries.filter((entry) => entry.state === 'skipped').length,
    pending: entries.filter((entry) => entry.state === 'pending').length,
    failed: entries.filter((entry) => entry.state === 'failed').length
  };
}

export function validateReport(report, plan = null) {
  if (!isObject(report)) fail('el archivo no contiene un objeto');
  if (report.schema !== REPORT_SCHEMA) {
    fail(`schema ${String(report.schema)} incompatible; se esperaba ${REPORT_SCHEMA}`);
  }
  if (!isObject(report.plan)) fail('falta plan con la identidad del plan ejecutado');
  if (typeof report.plan.contractVersion !== 'string' || !report.plan.contractVersion) {
    fail('plan.contractVersion ausente');
  }
  if (!Number.isInteger(report.plan.policyVersion)) fail('plan.policyVersion ausente');
  if (typeof report.plan.policyDigest !== 'string' || report.plan.policyDigest.length !== 64) {
    fail('plan.policyDigest ausente o inválido');
  }
  if (typeof report.plan.digest !== 'string' || report.plan.digest.length !== 64) {
    fail('plan.digest ausente o inválido');
  }
  if (!isObject(report.plan.source)) fail('plan.source ausente o inválido');
  if (typeof report.plan.level !== 'string' || !report.plan.level) {
    fail('plan.level ausente');
  }
  if (typeof report.plan.mode !== 'string' || !report.plan.mode) {
    fail('plan.mode ausente');
  }
  if (typeof report.runId !== 'string' || !report.runId) fail('runId ausente');
  if (typeof report.jobId !== 'string' || !report.jobId) fail('jobId ausente');
  for (const flag of ['partial', 'dryRun', 'validated', 'synthetic']) {
    if (typeof report[flag] !== 'boolean') fail(`${flag} debe ser booleano`);
  }
  if (!isObject(report.execution)) fail('falta execution con la evidencia de ejecución');
  if (report.execution.schema !== 1) fail('execution.schema incompatible');
  if (
    typeof report.execution.evidence !== 'string' ||
    !/^[A-Za-z0-9._-]+$/.test(report.execution.evidence)
  ) {
    fail('execution.evidence debe ser un nombre de archivo seguro');
  }
  if (typeof report.execution.digest !== 'string' || !/^[0-9a-f]{64}$/.test(report.execution.digest)) {
    fail('execution.digest debe ser sha256 hexadecimal');
  }
  if (report.synthetic === true) {
    if (report.validated) fail('un plan sintético no puede declarar validación completa');
    const expectedStatus = report.dryRun ? 'simulated' : 'synthetic';
    const failedStatus = report.totals?.failed > 0 ? 'failed' : null;
    if (report.validationStatus !== expectedStatus && report.validationStatus !== failedStatus) {
      fail(`un plan sintético debe declarar validationStatus ${expectedStatus}`);
    }
  }
  if (!VALIDATION_STATUS.has(report.validationStatus)) {
    fail(`validationStatus desconocido: ${String(report.validationStatus)}`);
  }
  if (!Array.isArray(report.required) || !report.required.every((id) => typeof id === 'string')) {
    fail('required debe ser una lista de identificadores');
  }
  if (new Set(report.required).size !== report.required.length) {
    fail('required contiene suites duplicadas');
  }
  if (!isObject(report.suites)) fail('suites no es un objeto');
  if (!isObject(report.totals)) fail('faltan totals');

  for (const [suiteId, entry] of Object.entries(report.suites)) {
    if (!isObject(entry)) fail(`suites.${suiteId} no es un objeto`);
    if (typeof entry.selected !== 'boolean') fail(`suites.${suiteId}.selected no es booleano`);
    if (!STATE_SET.has(entry.state)) fail(`suites.${suiteId} tiene estado desconocido ${entry.state}`);
    if (typeof entry.reason !== 'string') fail(`suites.${suiteId}.reason no es textual`);
    if (entry.status !== null && !Number.isInteger(entry.status)) {
      fail(`suites.${suiteId}.status debe ser entero o null`);
    }
    if (['executed', 'cached'].includes(entry.state) && entry.status !== 0) {
      fail(`suites.${suiteId} en estado ${entry.state} debe tener status 0`);
    }
    if (entry.state === 'failed' && (entry.status === null || entry.status === 0)) {
      fail(`suites.${suiteId} en estado failed debe tener un status distinto de cero`);
    }
    if (['skipped', 'pending', 'simulated'].includes(entry.state) && entry.status !== null) {
      fail(`suites.${suiteId} en estado ${entry.state} no puede tener status`);
    }
    if (entry.selected && entry.state === 'skipped') {
      fail(`suites.${suiteId} está seleccionada pero marcada como omitida`);
    }
    if (!entry.selected && entry.state !== 'skipped') {
      fail(`suites.${suiteId} no está seleccionada pero su estado es ${entry.state}`);
    }
    if (entry.state === 'simulated' && !report.dryRun) {
      fail(`suites.${suiteId} está simulada fuera de un dry-run`);
    }
    if (entry.state === 'cached' && report.dryRun) {
      fail(`suites.${suiteId} no puede venir de caché en un dry-run`);
    }
  }

  for (const suiteId of report.required) {
    const entry = report.suites[suiteId];
    if (!entry) fail(`la suite obligatoria ${suiteId} no aparece en el informe`);
    if (!entry.selected) fail(`la suite obligatoria ${suiteId} figura como no seleccionada`);
  }

  const computed = summarizeReport(report);
  for (const key of Object.keys(computed)) {
    if (!isCount(report.totals[key])) fail(`totals.${key} debe ser un entero no negativo`);
    if (report.totals[key] !== computed[key]) {
      fail(`totals.${key} declara ${report.totals[key]} y el informe contiene ${computed[key]}`);
    }
  }
  if (report.totals.required !== report.required.length) {
    fail('totals.required no coincide con la lista required');
  }
  if (report.partial && report.validationStatus === 'complete') {
    fail('una ejecución parcial no puede declararse completa');
  }
  if (report.dryRun && report.totals.executed > 0) {
    fail('un dry-run no puede contabilizar suites ejecutadas');
  }
  if (plan) {
    const expectedSynthetic = plan.source?.kind === 'files' || plan.synthetic === true;
    if (report.synthetic !== expectedSynthetic) {
      fail(
        `synthetic=${report.synthetic} no coincide con la procedencia ${plan.source?.kind || 'desconocida'}`
      );
    }
    if (expectedSynthetic && (report.validated || report.validationStatus === 'complete')) {
      fail('un plan sintético no puede declarar validación completa');
    }
    const expectedRequired = [...(plan.requiredSuites || [])].sort();
    const actualRequired = [...report.required].sort();
    if (JSON.stringify(expectedRequired) !== JSON.stringify(actualRequired)) {
      fail('las suites obligatorias no coinciden con el plan');
    }
    for (const suiteId of expectedRequired) {
      if (report.suites[suiteId]?.selected !== true) {
        fail(`la suite obligatoria ${suiteId} no está seleccionada en el informe`);
      }
    }
    for (const [suiteId, entry] of Object.entries(report.suites)) {
      if (entry.selected !== Boolean(plan.suites?.[suiteId]?.selected)) {
        fail(`la selección de ${suiteId} no coincide con el plan`);
      }
    }
  }
  const shouldValidate =
    report.synthetic !== true &&
    !report.partial &&
    !report.dryRun &&
    report.required.length > 0 &&
    report.totals.failed === 0 &&
    report.totals.pending === 0 &&
    report.required.every((id) => ['executed', 'cached'].includes(report.suites[id].state));
  if (report.validated !== shouldValidate) {
    fail(
      `validated=${report.validated} es incoherente: partial=${report.partial}, dryRun=${report.dryRun}, ` +
        `failed=${report.totals.failed}, pending=${report.totals.pending}`
    );
  }
  if (report.validated && report.validationStatus !== 'complete') {
    fail('validated=true exige validationStatus=complete');
  }
  if (!report.validated && report.validationStatus === 'complete') {
    fail('validationStatus=complete exige validated=true');
  }
  return report;
}

/**
 * Comprueba que el conjunto de informes parciales de CI cubre todas las suites
 * obligatorias del plan. Una suite simulada, pendiente o ausente no cuenta como
 * validación; un fallo en cualquier informe invalida el agregado.
 */
export function aggregateReports({
  plan,
  reports,
  planDigest,
  receiptKey = null,
  receiptKeys = null,
  expectedRunId = null
}) {
  const required = [...(plan.requiredSuites || [])].sort();
  const covered = new Set();
  const failed = new Set();
  const problems = [];
  const simulated = new Set();
  const seen = new Map();
  const runIds = new Set();

  if (required.length === 0) {
    problems.push('el plan no tiene suites obligatorias; no se puede declarar validación');
  }

  for (const item of reports) {
    const report = item?.report || item;
    const evidence = item?.evidence || null;
    try {
      validateReport(report, plan);
    } catch (error) {
      problems.push(error.message);
      continue;
    }
    if (report.synthetic === true) {
      problems.push('un informe sintético no valida el árbol de trabajo');
      continue;
    }
    const binding = bindingProblems(report, plan, planDigest);
    if (binding.length > 0) {
      problems.push(...binding.map((problem) => `informe no vinculado al plan: ${problem}`));
      continue;
    }
    const key = receiptKeys ? receiptKeys[report.jobId] : receiptKey;
    if (!key || !verifyReportReceipt(report, key)) {
      problems.push('un informe no tiene un recibo de ejecución verificable para su job');
      continue;
    }
    if (!evidence) {
      problems.push('un informe no tiene evidencia de ejecución verificable');
      continue;
    }
    if (!verifyExecutionEvidence(evidence, report, key)) {
      problems.push('la evidencia de ejecución no corresponde al informe o a su job');
      continue;
    }
    if (expectedRunId && report.jobId === 'local') {
      problems.push('un informe local no puede cubrir una ejecución de CI');
      continue;
    }
    if (expectedRunId && report.runId !== expectedRunId) {
      problems.push('un informe pertenece a otra ejecución');
      continue;
    }
    runIds.add(report.runId);
    if (planDigest && report.plan.digest !== planDigest) {
      problems.push('un informe se generó con otro plan');
      continue;
    }
    if (report.plan.policyDigest !== plan.policyDigest) {
      problems.push('un informe se generó con otra política');
      continue;
    }
    if (report.plan.contractVersion !== plan.contractVersion) {
      problems.push('un informe se generó con otro contrato de ejecución');
      continue;
    }
    if (report.plan.policyVersion !== plan.policyVersion) {
      problems.push('un informe declara otra versión de política');
      continue;
    }
    for (const [suiteId, entry] of Object.entries(report.suites)) {
      if (entry.state === 'simulated') simulated.add(suiteId);
      if (!entry.selected) continue;
      const expectedJob = SUITE_JOB[suiteId];
      const expectedJobs = SUITE_JOB_ALIASES[suiteId] || (expectedJob ? [expectedJob] : []);
      const claimsExecution = ['executed', 'cached', 'simulated', 'failed'].includes(entry.state);
      if (claimsExecution) {
        const key = `${report.runId}:${suiteId}`;
        if (seen.has(key)) {
          problems.push(`la suite ${suiteId} tiene informes duplicados en la misma ejecución`);
          continue;
        }
        seen.set(key, true);
      }
      if (
        claimsExecution &&
        expectedJobs.length > 0 &&
        report.jobId !== 'local' &&
        !expectedJobs.includes(report.jobId)
      ) {
        problems.push(
          `la suite ${suiteId} no fue producida por el job ${expectedJobs.join(' o ')}`
        );
        continue;
      }
      if (entry.state === 'failed' || (entry.status !== null && entry.status !== 0)) {
        failed.add(suiteId);
        continue;
      }
      if (entry.state === 'executed' || entry.state === 'cached') {
        covered.add(suiteId);
      }
    }
  }

  if (runIds.size > 1) problems.push('los informes pertenecen a ejecuciones distintas');

  for (const suiteId of failed) {
    if (required.includes(suiteId)) {
      problems.push(`la suite obligatoria ${suiteId} falló`);
    }
  }
  const missing = required.filter((suiteId) => !covered.has(suiteId) && !failed.has(suiteId));
  for (const suiteId of missing) {
    if (simulated.has(suiteId)) {
      problems.push(`la suite obligatoria ${suiteId} solo fue simulada`);
    } else {
      problems.push(`la suite obligatoria ${suiteId} no fue ejecutada por ningún job`);
    }
  }

  return {
    ok: problems.length === 0,
    required,
    covered: [...covered].sort(),
    missing,
    failed: [...failed].sort(),
    problems
  };
}

export function aggregateText(result, plan) {
  const lines = [];
  lines.push(`Plan ${plan.contractVersion} · política v${plan.policyVersion} (${plan.policyDigest.slice(0, 12)})`);
  lines.push(`Suites obligatorias: ${result.required.join(', ') || '(ninguna)'}`);
  lines.push(`Cubiertas: ${result.covered.join(', ') || '(ninguna)'}`);
  if (result.missing.length) lines.push(`Pendientes: ${result.missing.join(', ')}`);
  if (result.failed.length) lines.push(`Fallidas: ${result.failed.join(', ')}`);
  for (const problem of result.problems) lines.push(`ERROR: ${problem}`);
  lines.push(
    result.ok
      ? 'Validación global completa: todas las suites obligatorias del plan terminaron correctamente.'
      : 'Validación global incompleta: el cambio no está validado.'
  );
  return `${lines.join('\n')}\n`;
}
