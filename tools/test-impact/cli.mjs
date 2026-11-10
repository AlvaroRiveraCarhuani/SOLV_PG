#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { loadPolicy } from './lib/policy.mjs';
import { buildBackendGraph } from './lib/backend-graph.mjs';
import { buildFrontendGraph } from './lib/frontend-graph.mjs';
import { collectChanges, resolveCommit } from './lib/git.mjs';
import { createPlan, summarizePlan } from './lib/select.mjs';
import {
  PLAN_CONTRACT_VERSION,
  PLAN_SCHEMA,
  planDigest,
  policyDigest,
  validatePlan
} from './lib/plan-integrity.mjs';
import {
  aggregateReports,
  aggregateText,
  summarizeReport,
  validateReport
} from './lib/report.mjs';
import {
  assertReportBinding,
  loadReceiptKey,
  loadReceiptKeys,
  verifyExecutionEvidence,
  verifyReportReceipt
} from './lib/receipt.mjs';

function parseArgs(argv) {
  const args = { _: [] };
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (!token.startsWith('--')) {
      args._.push(token);
      continue;
    }
    const equals = token.indexOf('=');
    if (equals > 0) {
      args[token.slice(2, equals)] = token.slice(equals + 1);
      continue;
    }
    const key = token.slice(2);
    const next = argv[index + 1];
    if (next === undefined || next.startsWith('--')) {
      args[key] = true;
      continue;
    }
    args[key] = next;
    index += 1;
  }
  return args;
}

function defaultBase(repoRoot) {
  if (resolveCommit(repoRoot, 'origin/main')) return 'origin/main';
  if (resolveCommit(repoRoot, 'HEAD~1')) return 'HEAD~1';
  return null;
}

function buildPlan(args) {
  const startedAt = performance.now();
  const repoRoot = path.resolve(args.repo || process.cwd());
  const policy = loadPolicy(repoRoot);
  const base = args.base === undefined ? defaultBase(repoRoot) : args.base;
  const head = args.head || 'HEAD';
  const event = args.event || process.env.GITHUB_EVENT_NAME || 'local';
  const files = typeof args.files === 'string' ? args.files.split(',').filter(Boolean) : null;
  const includeWorkingTree = args['working-tree'] === true || args['working-tree'] === 'true';
  const selection = collectChanges({ repoRoot, base, head, files, includeWorkingTree });

  const options = {
    full: args.full === true || args.full === 'true',
    event,
    base,
    head,
    maxChangedFiles: args['max-files'] ? Number(args['max-files']) : undefined
  };

  const graphStartedAt = performance.now();
  const backendGraph = buildBackendGraph(repoRoot);
  const frontendGraph = buildFrontendGraph(repoRoot);
  const graphMs = performance.now() - graphStartedAt;

  if (selection.error && !options.full && !files) {
    options.full = true;
    options.forcedReason = selection.error;
  }

  const selectionStartedAt = performance.now();
  const plan = createPlan({
    policy,
    changes: selection.changes,
    backendGraph,
    frontendGraph,
    options: { ...options, repoRoot }
  });
  plan.schema = PLAN_SCHEMA;
  plan.contractVersion = PLAN_CONTRACT_VERSION;
  plan.policyDigest = policyDigest(repoRoot);
  plan.source = {
    kind: files ? 'files' : 'git',
    ...(files ? { files } : {}),
    base: base || null,
    baseCommit: base ? resolveCommit(repoRoot, base) : null,
    head,
    headCommit: resolveCommit(repoRoot, head),
    includeWorkingTree,
    event,
    full: options.full === true,
    maxChangedFiles: options.maxChangedFiles || null
  };
  plan.metrics = {
    analysisMs: Math.round(graphStartedAt - startedAt),
    graphMs: Math.round(graphMs),
    selectionMs: Math.round(performance.now() - selectionStartedAt),
    totalMs: Math.round(performance.now() - startedAt)
  };
  plan.generatedAt = new Date().toISOString();
  return { plan, policy, repoRoot };
}

function writeOutput(plan, args) {
  if (args.output) {
    fs.writeFileSync(path.resolve(args.output), `${JSON.stringify(plan, null, 2)}\n`);
  }
  if (args.format === 'text' || args._[0] === 'explain') {
    process.stdout.write(summarizePlan(plan));
    return;
  }
  if (!args.output) process.stdout.write(`${JSON.stringify(plan, null, 2)}\n`);
}

function githubOutputs(plan) {
  const values = {
    contract: plan.contractVersion,
    policy_version: plan.policyVersion,
    policy_digest: plan.policyDigest,
    required_suites: (plan.requiredSuites || []).join(','),
    mode: plan.mode,
    level: plan.level,
    level_title: plan.levelTitle,
    reason: plan.reason,
    gates: plan.suites.gates.selected,
    asset_validation: plan.suites['asset-validation'].selected,
    backend_unit: plan.suites['backend-unit'].selected,
    backend_packages_count: plan.suites['backend-unit'].packages.length,
    backend_integration: plan.suites['backend-integration'].selected,
    backend_docker: plan.suites['backend-docker'].selected,
    frontend_unit: plan.suites['frontend-unit'].selected,
    frontend_specs_count: plan.suites['frontend-unit'].specs.length,
    frontend_e2e: plan.suites['frontend-e2e'].selected,
    e2e_specs_count: plan.suites['frontend-e2e'].specs.length,
    migrations: plan.suites.migrations.selected,
    build: plan.suites.build.selected,
    build_backend: plan.suites.build.backend === true,
    build_frontend: plan.suites.build.frontend === true,
    security: plan.suites.security.selected,
    backend_touched: plan.suites['backend-unit'].selected || plan.suites['backend-integration'].selected || plan.suites['backend-docker'].selected || plan.suites.migrations.selected,
    frontend_touched: plan.suites['frontend-unit'].selected || plan.suites['frontend-e2e'].selected || plan.suites.build.frontend === true,
    critical: plan.criticalAreas.join(','),
    unknown: plan.unknown.join(',')
  };
  return Object.entries(values)
    .map(([key, value]) => `${key}=${typeof value === 'boolean' ? String(value) : value}`)
    .join('\n');
}

function explain(args) {
  const repoRoot = path.resolve(args.repo || process.cwd());
  const policy = loadPolicy(repoRoot);
  const backendGraph = buildBackendGraph(repoRoot);
  const frontendGraph = buildFrontendGraph(repoRoot);
  const targets = args._.slice(1);
  if (targets.length === 0) {
    throw new Error('explain requiere al menos una ruta');
  }
  const plan = createPlan({
    policy,
    changes: targets.map((target) => ({ status: 'M', path: target })),
    backendGraph,
    frontendGraph,
    options: { event: 'local' }
  });
  process.stdout.write(summarizePlan(plan));
  for (const change of plan.changed) {
    process.stdout.write(
      `\n${change.path}\n  áreas: ${change.areas.map((area) => area.id).join(', ') || '(ninguna)'}\n  críticas: ${
        change.critical.map((entry) => entry.id).join(', ') || '(ninguna)'
      }\n  neutral: ${change.neutral}\n`
    );
  }
  if (plan.decisions) {
    process.stdout.write(`\nDecisiones del plan:\n`);
    for (const [suiteId, decision] of Object.entries(plan.decisions.suites || {})) {
      process.stdout.write(`- ${suiteId}: ${decision.selected ? 'sí' : 'no'} — ${decision.reason}\n`);
    }
  }
}

const args = parseArgs(process.argv.slice(2));
const command = args._[0] || 'plan';

try {
  if (command === 'validate') {
    const repoRoot = path.resolve(args.repo || process.cwd());
    const policy = loadPolicy(repoRoot);
    process.stdout.write(`Política v${policy.version} válida: ${Object.keys(policy.suites).length} suites, ${policy.areas.length} áreas, ${policy.critical.length} reglas críticas.\n`);
    process.exit(0);
  }

  if (command === 'outputs') {
    if (!args.plan) throw new Error('outputs requiere --plan');
    const repoRoot = path.resolve(args.repo || process.cwd());
    const policy = loadPolicy(repoRoot);
    const raw = JSON.parse(fs.readFileSync(path.resolve(args.plan), 'utf8'));
    const plan = validatePlan({
      repoRoot,
      plan: raw,
      policy,
      backendGraph: buildBackendGraph(repoRoot),
      frontendGraph: buildFrontendGraph(repoRoot)
    });
    process.stdout.write(`${githubOutputs(plan)}\n`);
    process.exit(0);
  }

  if (command === 'aggregate') {
    if (!args.plan) throw new Error('aggregate requiere --plan');
    const directory = path.resolve(args.dir || 'impact-reports');
    const repoRoot = path.resolve(args.repo || process.cwd());
    const policy = loadPolicy(repoRoot);
    const raw = JSON.parse(fs.readFileSync(path.resolve(args.plan), 'utf8'));
    const plan = validatePlan({
      repoRoot,
      plan: raw,
      policy,
      backendGraph: buildBackendGraph(repoRoot),
      frontendGraph: buildFrontendGraph(repoRoot)
    });
    const files = fs.existsSync(directory)
      ? fs
          .readdirSync(directory)
          .filter((file) => file.endsWith('.json') && !file.endsWith('.evidence.json'))
          .sort()
      : [];
    const items = files.map((file) => {
      try {
        const report = JSON.parse(fs.readFileSync(path.join(directory, file), 'utf8'));
        const evidenceFile = path.join(directory, `${file}.evidence.json`);
        let evidence = null;
        if (fs.existsSync(evidenceFile)) {
          evidence = JSON.parse(fs.readFileSync(evidenceFile, 'utf8'));
        }
        return { report, evidence };
      } catch (error) {
        return { broken: `${file}: ${error.message}` };
      }
    });
    const broken = items.filter((item) => item.broken);
    const reports = items.filter((item) => !item.broken).map((item) => item.report);
    const receiptKeys = loadReceiptKeys({
      repoRoot,
      jobIds: reports.map((report) => report.jobId)
    });
    for (const jobId of receiptKeys.missing) {
      process.stdout.write(`ERROR: falta la clave de recibo del job ${jobId}\n`);
    }
    const result = aggregateReports({
      plan,
      planDigest: planDigest(plan),
      receiptKeys: receiptKeys.keys,
      expectedRunId: process.env.SOLV_TEST_EXECUTION_ID || null,
      reports: items.filter((item) => !item.broken)
    });
    for (const report of broken) result.problems.push(`informe ilegible: ${report.broken}`);
    result.ok = result.ok && receiptKeys.missing.length === 0;
    result.ok = result.ok && broken.length === 0;
    if (args.format === 'json') {
      process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
    } else {
      process.stdout.write(aggregateText(result, plan));
    }
    process.exit(result.ok ? 0 : 1);
  }

  if (command === 'explain') {
    explain(args);
    process.exit(0);
  }

  if (command === 'report') {
    const file = args.file || args.report;
    if (!file) throw new Error('report requiere --file <ruta>');
    if (!args.plan) throw new Error('report requiere --plan <ruta>');
    const repoRoot = path.resolve(args.repo || process.cwd());
    const policy = loadPolicy(repoRoot);
    const rawPlan = JSON.parse(fs.readFileSync(path.resolve(args.plan), 'utf8'));
    const plan = validatePlan({
      repoRoot,
      plan: rawPlan,
      policy,
      backendGraph: buildBackendGraph(repoRoot),
      frontendGraph: buildFrontendGraph(repoRoot)
    });
    const report = validateReport(JSON.parse(fs.readFileSync(path.resolve(file), 'utf8')), plan);
    assertReportBinding(report, plan, planDigest(plan));
    const receiptKey = loadReceiptKey({ repoRoot, jobId: report.jobId });
    if (!verifyReportReceipt(report, receiptKey)) {
      throw new Error('el informe no tiene un recibo de ejecución verificable');
    }
    const evidenceFile = path.resolve(path.dirname(path.resolve(file)), report.execution.evidence);
    if (!fs.existsSync(evidenceFile)) {
      throw new Error('el informe no tiene su archivo de evidencia de ejecución');
    }
    const evidence = JSON.parse(fs.readFileSync(evidenceFile, 'utf8'));
    if (!verifyExecutionEvidence(evidence, report, receiptKey)) {
      throw new Error('la evidencia de ejecución no corresponde al informe o a su job');
    }
    if (args.format === 'json') {
      process.stdout.write(`${JSON.stringify(report, null, 2)}\n`);
      process.exit(report.validated ? 0 : report.totals.failed > 0 ? 1 : 2);
    }
    process.stdout.write(`Nivel ${report.level} — ${report.levelTitle}\n`);
    process.stdout.write(
      `Plan ${report.plan.contractVersion} · política v${report.plan.policyVersion}; ` +
        `total ${((report.durationMs || 0) / 1000).toFixed(2)}s\n`
    );
    for (const item of report.order || []) {
      const entry = report.suites[item.id];
      const state = {
        executed: 'ejecutada',
        cached: 'caché',
        simulated: 'simulada',
        skipped: 'omitida',
        pending: 'PENDIENTE',
        failed: 'FALLÓ'
      }[entry.state];
      process.stdout.write(
        `[${state}] ${item.id}: ${entry.reason} (${((entry.durationMs || 0) / 1000).toFixed(2)}s)\n`
      );
    }
    const totals = summarizeReport(report);
    process.stdout.write(
      `Obligatorias ${totals.required}; ejecutadas ${totals.executed}; desde caché ${totals.cached}; ` +
        `simuladas ${totals.simulated}; omitidas ${totals.skipped}; pendientes ${totals.pending}; fallidas ${totals.failed}.\n`
    );
    if (report.validated) {
      process.stdout.write('Validación completa: todas las suites obligatorias terminaron correctamente.\n');
    } else if (report.dryRun) {
      process.stdout.write('Simulación: no se ejecutó ninguna suite; el cambio no está validado.\n');
    } else if (report.synthetic) {
      process.stdout.write(
        'Alcance sintético: no prueba el diff real del árbol de trabajo; el cambio no está validado.\n'
      );
    } else if (report.partial) {
      process.stdout.write('Ejecución parcial: no valida el cambio completo.\n');
    } else {
      process.stdout.write('Validación incompleta: el cambio no está validado.\n');
    }
    process.exit(report.validated ? 0 : report.totals.failed > 0 ? 1 : 2);
  }

  if (command !== 'plan') {
    throw new Error(`comando desconocido: ${command}`);
  }

  const { plan } = buildPlan(args);
  writeOutput(plan, args);
  if (args['fail-on-full'] === true && plan.mode === 'full') {
    process.exitCode = 2;
  }
} catch (error) {
  process.stderr.write(`Error del selector de impacto: ${error.message}\n`);
  if (process.env.SOLV_DEBUG) process.stderr.write(`${error.stack}\n`);
  process.exit(1);
}
