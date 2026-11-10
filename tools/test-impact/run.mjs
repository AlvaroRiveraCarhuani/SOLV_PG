#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { loadPolicy } from './lib/policy.mjs';
import { requireTestDatabase } from './lib/test-db.mjs';
import { computeCacheKey, readCache, writeCache } from './lib/cache.mjs';
import { buildBackendGraph } from './lib/backend-graph.mjs';
import { buildFrontendGraph } from './lib/frontend-graph.mjs';
import { planDigest, validatePlan } from './lib/plan-integrity.mjs';
import { REPORT_SCHEMA, summarizeReport } from './lib/report.mjs';
import {
  attachExecutionEvidence,
  loadReceiptKey,
  signReportReceipt
} from './lib/receipt.mjs';
import { validateAssetContent } from './lib/assets.mjs';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_REPO_ROOT = path.resolve(HERE, '..', '..');
const CLI = path.join(HERE, 'cli.mjs');

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

function run(command, commandArgs, { cwd, env = {}, dryRun = false }) {
  process.stdout.write(`\n$ ${[command, ...commandArgs].join(' ')}  (${cwd})\n`);
  if (dryRun) return 0;
  const result = spawnSync(command, commandArgs, {
    cwd,
    env: { ...process.env, ...env },
    stdio: 'inherit'
  });
  if (result.error) {
    process.stderr.write(`No se pudo ejecutar ${command}: ${result.error.message}\n`);
    return 1;
  }
  return result.status ?? 1;
}

function testRunID() {
  const configured = String(process.env.SOLV_TEST_RUN_ID || '').trim();
  if (configured) return configured;
  return `solv-local-${process.pid}-${Date.now()}`;
}

function cleanupDocker(repoRoot, runId, options, phase) {
  const script = path.join(repoRoot, 'tools', 'test-impact', 'cleanup-docker.sh');
  const status = run('bash', [script], {
    cwd: repoRoot,
    ...options,
    env: { ...(options.env || {}), SOLV_TEST_RUN_ID: runId }
  });
  if (status !== 0) {
    process.stderr.write(`Limpieza Docker (${phase}) falló con código ${status}; no se oculta.\n`);
  }
  return status;
}

function requireDocker() {
  const result = spawnSync('docker', ['info'], { stdio: 'ignore' });
  if (result.status !== 0) {
    throw new Error('la suite backend-docker requiere un demonio Docker accesible');
  }
}

function planFromArgs(args, repoRoot) {
  if (args.plan) {
    return JSON.parse(fs.readFileSync(path.resolve(args.plan), 'utf8'));
  }
  const output = path.join(os.tmpdir(), `solv-impact-plan-${process.pid}.json`);
  const cliArgs = [CLI, 'plan', '--format', 'json', '--output', output];
  if (args.repo) cliArgs.push('--repo', args.repo);
  if (args.base) cliArgs.push('--base', args.base);
  if (args.head) cliArgs.push('--head', args.head);
  if (args.event) cliArgs.push('--event', args.event);
  if (args.files) cliArgs.push('--files', args.files);
  if (args['working-tree']) cliArgs.push('--working-tree');
  if (args.full) cliArgs.push('--full');
  const result = spawnSync(process.execPath, cliArgs, { cwd: repoRoot, stdio: 'inherit' });
  if (result.status !== 0) {
    throw new Error('el selector de impacto no pudo generar el plan');
  }
  const plan = JSON.parse(fs.readFileSync(output, 'utf8'));
  fs.rmSync(output, { force: true });
  return plan;
}

function runGates(repoRoot, options) {
  const frontend = path.join(repoRoot, 'frontend');
  let status = run('npm', ['run', 'lint:styles'], { cwd: frontend, ...options });
  if (status !== 0) return status;
  return run(
    'npx',
    [
      'vitest',
      'run',
      'scripts/date-format-gate.test.mjs',
      'scripts/machine-data-gate.test.mjs',
      'scripts/machine-data-catalog.test.mjs',
      'scripts/motion-gate.test.mjs',
      'scripts/terminology-gate.test.mjs'
    ],
    { cwd: frontend, ...options }
  );
}

function runAssetValidation(repoRoot, plan, options) {
  const entry = plan.suites['asset-validation'];
  if (!entry.selected) {
    process.stdout.write('asset-validation: nada que validar\n');
    return 0;
  }
  let failed = false;
  let files = entry.files || [];
  if (entry.full) {
    const root = path.join(repoRoot, 'frontend', 'public');
    const walk = (directory) => {
      const found = [];
      for (const item of fs.readdirSync(directory, { withFileTypes: true })) {
        const absolute = path.join(directory, item.name);
        if (item.isDirectory()) found.push(...walk(absolute));
        else if (item.isFile()) found.push(path.relative(repoRoot, absolute));
      }
      return found;
    };
    files = fs.existsSync(root) ? walk(root).sort() : [];
  }
  let checked = 0;
  const limits = new Set();
  for (const relative of files) {
    const absolute = path.join(repoRoot, relative);
    try {
      const stat = fs.statSync(absolute);
      if (!stat.isFile() || stat.size === 0) {
        process.stderr.write(`asset-validation: recurso vacío o inválido: ${relative}\n`);
        failed = true;
        continue;
      }
      const validation = validateAssetContent(relative, fs.readFileSync(absolute));
      if (!validation.ok) {
        process.stderr.write(`asset-validation: ${relative}: ${validation.error}\n`);
        failed = true;
        continue;
      }
      if (validation.limit) limits.add(validation.limit);
      checked += 1;
      if (!entry.full) {
        process.stdout.write(`asset-validation: ${relative} (${stat.size} bytes)\n`);
      }
    } catch (error) {
      process.stderr.write(`asset-validation: no se pudo leer ${relative}: ${error.message}\n`);
      failed = true;
    }
  }
  if (entry.full) process.stdout.write(`asset-validation: ${checked} recurso(s) revisados\n`);
  for (const limit of limits) {
    process.stdout.write(`asset-validation: límite documentado — ${limit}\n`);
  }
  return failed ? 1 : 0;
}

function runBackendUnit(repoRoot, plan, options) {
  const entry = plan.suites['backend-unit'];
  if (!entry.selected || entry.packages.length === 0) {
    process.stdout.write('backend-unit: nada que ejecutar\n');
    return 0;
  }
  return run('go', ['test', '-short', '-count=1', ...entry.packages], {
    cwd: path.join(repoRoot, 'backend'),
    ...options
  });
}

function runBackendIntegration(repoRoot, plan, options) {
  const entry = plan.suites['backend-integration'];
  if (!entry.selected) {
    process.stdout.write('backend-integration: nada que ejecutar\n');
    return 0;
  }
  if (options.dryRun) {
    return run('go', ['test', '-short', '-count=1', './tests/integration/...', '-timeout', '420s'], {
      cwd: path.join(repoRoot, 'backend'),
      ...options
    });
  }
  const dsn = requireTestDatabase({ ...process.env, ...(options.env || {}) });
  return run(
    'go',
    ['test', '-short', '-count=1', './tests/integration/...', '-timeout', '420s'],
    {
      cwd: path.join(repoRoot, 'backend'),
      ...options,
      env: {
        ...(options.env || {}),
        TEST_DB_DSN: dsn,
        TEST_DATABASE_URL: dsn,
        DATABASE_URL: dsn,
        SOLV_REQUIRE_INFRA: '1'
      }
    }
  );
}

function runBackendDocker(repoRoot, plan, options) {
  const entry = plan.suites['backend-docker'];
  if (!entry.selected) {
    process.stdout.write('backend-docker: nada que ejecutar\n');
    return 0;
  }
  if (options.dryRun) {
    return run('go', ['test', '-count=1', './tests/integration/...', '-timeout', '1500s'], {
      cwd: path.join(repoRoot, 'backend'),
      ...options
    });
  }
  requireDocker();
  const dsn = requireTestDatabase({ ...process.env, ...(options.env || {}) });
  const runId = testRunID();
  const initialCleanup = cleanupDocker(repoRoot, runId, options, 'inicio');
  if (initialCleanup !== 0) return initialCleanup;
  // Sin -short se ejecutan todos los tests de integración, incluidos los que
  // arrancan contenedores. No se mantiene una lista de nombres: una prueba
  // nueva de Docker quedaría fuera de la suite sin que nadie lo note.
  const status = run(
    'go',
    ['test', '-count=1', './tests/integration/...', '-timeout', '1500s'],
    {
      cwd: path.join(repoRoot, 'backend'),
      ...options,
      env: {
        ...(options.env || {}),
        TEST_DB_DSN: dsn,
        TEST_DATABASE_URL: dsn,
        DATABASE_URL: dsn,
        SOLV_TEST_RUN_ID: runId,
        SOLV_REQUIRE_INFRA: '1',
        SOLV_REQUIRE_DOCKER: '1'
      }
    }
  );
  const finalCleanup = cleanupDocker(repoRoot, runId, options, 'cierre');
  if (finalCleanup !== 0) return finalCleanup;
  return status;
}

function runFrontendUnit(repoRoot, plan, options) {
  const entry = plan.suites['frontend-unit'];
  if (!entry.selected) {
    process.stdout.write('frontend-unit: nada que ejecutar\n');
    return 0;
  }
  const frontend = path.join(repoRoot, 'frontend');
  const commandArgs = ['test', '--watch=false'];
  if (!entry.full) {
    for (const spec of entry.specs) {
      commandArgs.push(`--include=${spec.startsWith('src/') ? spec.slice(4) : spec}`);
    }
  }
  return run('npx', ['ng', ...commandArgs], { cwd: frontend, ...options });
}

function runFrontendE2e(repoRoot, plan, options) {
  const entry = plan.suites['frontend-e2e'];
  if (!entry.selected) {
    process.stdout.write('frontend-e2e: nada que ejecutar\n');
    return 0;
  }
  if (entry.specs.length === 0) {
    process.stderr.write('frontend-e2e: el plan marcó la suite pero no seleccionó flujos\n');
    return 1;
  }
  const frontend = path.join(repoRoot, 'frontend');
  const groups = new Map();
  for (const item of entry.specs) {
    if (!groups.has(item.config)) groups.set(item.config, []);
    groups.get(item.config).push(item.spec);
  }
  for (const [config, specs] of groups) {
    const status = run('npx', ['playwright', 'test', ...specs, `--config=${config}`], {
      cwd: frontend,
      ...options
    });
    if (status !== 0) return status;
  }
  return 0;
}

function runMigrations(repoRoot, plan, options) {
  const entry = plan.suites.migrations;
  if (!entry.selected) {
    process.stdout.write('migrations: nada que ejecutar\n');
    return 0;
  }
  if (options.dryRun) {
    for (const action of ['up', 'down', 'up']) {
      const status = run('go', ['run', './cmd/migrate', '-dir', './migrations', `-action=${action}`], {
        cwd: path.join(repoRoot, 'backend'),
        ...options
      });
      if (status !== 0) return status;
    }
    return 0;
  }
  const dsn = requireTestDatabase({ ...process.env, ...(options.env || {}) });
  const backend = path.join(repoRoot, 'backend');
  const env = {
    ...(options.env || {}),
    TEST_DB_DSN: dsn,
    TEST_DATABASE_URL: dsn,
    DATABASE_URL: dsn
  };
  for (const action of ['up', 'down', 'up']) {
    const status = run('go', ['run', './cmd/migrate', '-dir', './migrations', `-action=${action}`], {
      cwd: backend,
      ...options,
      env
    });
    if (status !== 0) return status;
  }
  return 0;
}

function runBuild(repoRoot, plan, options) {
  const entry = plan.suites.build;
  if (!entry.selected) {
    process.stdout.write('build: nada que compilar\n');
    return 0;
  }
  const buildBackend = entry.backend === undefined ? entry.frontend === undefined : entry.backend;
  const buildFrontend = entry.frontend === undefined ? entry.backend === undefined : entry.frontend;
  if (buildBackend) {
    const status = run('go', ['build', './...'], {
      cwd: path.join(repoRoot, 'backend'),
      ...options
    });
    if (status !== 0) return status;
  }
  if (buildFrontend) {
    return run('npm', ['run', 'build'], {
      cwd: path.join(repoRoot, 'frontend'),
      ...options,
      env: { ...(options.env || {}), CI: 'true', NG_CLI_ANALYTICS: 'false' }
    });
  }
  return 0;
}

function runSecurity(repoRoot, plan, options) {
  if (!plan.suites.security.selected) {
    process.stdout.write('security: nada que auditar\n');
    return 0;
  }
  const backendTouched =
    plan.suites['backend-unit'].selected ||
    plan.suites['backend-integration'].selected ||
    plan.suites['backend-docker'].selected ||
    plan.suites.migrations.selected;
  if (backendTouched) {
    const vetStatus = run('go', ['vet', './...'], {
      cwd: path.join(repoRoot, 'backend'),
      ...options
    });
    if (vetStatus !== 0) return vetStatus;
  }
  // Pruebas negativas de los contratos WebSocket, terminal docente y
  // revocación de sesiones declarados en la política. Se ejecutan de verdad:
  // si el handler mantiene una brecha, security falla y no se convierte en una
  // advertencia ni en un skip.
  const websocketStatus = run(
    'go',
    [
      'test',
      '-run',
      '^(TestWebSocketSecurity|TestTerminalAuthorization|TestSessionRevocationMiddleware|TestLogout|TestAuthenticationFailsClosed|TestNoRevocationStore)',
      '-count=1',
      './internal/delivery/http/...'
    ],
    { cwd: path.join(repoRoot, 'backend'), ...options }
  );
  if (websocketStatus !== 0) return websocketStatus;
  let status = run('npm', ['audit', '--audit-level=high', '--omit=dev'], {
    cwd: path.join(repoRoot, 'frontend'),
    ...options
  });
  if (status !== 0) return status;
  return run('bash', ['./scripts/audit-secrets.sh'], { cwd: repoRoot, ...options });
}

const RUNNERS = {
  gates: (repoRoot, plan, options) => runGates(repoRoot, options),
  'asset-validation': runAssetValidation,
  'backend-unit': runBackendUnit,
  'backend-integration': runBackendIntegration,
  'backend-docker': runBackendDocker,
  'frontend-unit': runFrontendUnit,
  'frontend-e2e': runFrontendE2e,
  migrations: runMigrations,
  build: runBuild,
  security: runSecurity
};

/**
 * Describe el comando efectivo de la suite. La caché incluye esta cadena para
 * no reutilizar un resultado obtenido con otro comando o alcance.
 */
function describeSuiteCommand(suiteId, entry) {
  switch (suiteId) {
    case 'gates':
      return 'npm run lint:styles && npx vitest run scripts/{date-format,machine-data,machine-data-catalog,motion,terminology}-gate.test.mjs';
    case 'asset-validation':
      return entry.full
        ? 'asset-validation --full frontend/public'
        : `asset-validation ${[...(entry.files || [])].sort().join(' ')}`;
    case 'backend-unit':
      return `go test -short -count=1 ${[...(entry.packages || [])].sort().join(' ')}`;
    case 'frontend-unit':
      return entry.full
        ? 'npx ng test --watch=false (todas las specs)'
        : `npx ng test --watch=false ${[...(entry.specs || [])].sort().join(' ')}`;
    default:
      return `${suiteId}:${JSON.stringify(entry)}`;
  }
}

const args = parseArgs(process.argv.slice(2));
const repoRoot = path.resolve(args.repo || DEFAULT_REPO_ROOT);
const dryRun = args['dry-run'] === true || args['dry-run'] === 'true';
const suiteArg = args.suite || args._[0];
const all = args.all === true || args.all === 'true';
const partial = args.partial === true || args.partial === 'true';
const ifSelected = args['if-selected'] === true || args['if-selected'] === 'true';
const reportRunId = process.env.SOLV_TEST_EXECUTION_ID || testRunID();
const reportJobId = process.env.GITHUB_JOB || 'local';

function formatDuration(milliseconds) {
  return `${((milliseconds || 0) / 1000).toFixed(2)}s`;
}

const STATE_LABEL = {
  executed: 'ejecutada',
  cached: 'caché',
  simulated: 'simulada',
  skipped: 'omitida',
  pending: 'PENDIENTE',
  failed: 'FALLÓ'
};

function reportText(report) {
  const lines = [];
  lines.push(`Nivel ${report.level} — ${report.levelTitle}`);
  lines.push(
    `Plan ${report.plan.contractVersion} · política v${report.plan.policyVersion} · ${report.mode}; total ${formatDuration(report.durationMs)}`
  );
  for (const entry of report.order) {
    const result = report.suites[entry.id];
    lines.push(
      `[${STATE_LABEL[result.state]}] ${entry.id}: ${result.reason} (${formatDuration(result.durationMs)})`
    );
  }
  const totals = report.totals;
  lines.push(
    `Obligatorias ${totals.required}; ejecutadas ${totals.executed}; caché ${totals.cached}; ` +
      `simuladas ${totals.simulated}; omitidas ${totals.skipped}; pendientes ${totals.pending}; fallidas ${totals.failed}.`
  );
  if (report.validated) {
    lines.push('Validación completa: todas las suites obligatorias terminaron correctamente.');
  } else if (report.dryRun) {
    lines.push('Simulación: no se ejecutó ninguna suite; el cambio no está validado.');
  } else if (report.synthetic) {
    lines.push(
      'Alcance sintético: el plan declara archivos de forma explícita y no prueba el diff real; no valida el cambio completo.'
    );
  } else if (report.partial) {
    lines.push('Ejecución parcial: no valida el cambio completo.');
  } else {
    lines.push('Validación incompleta: el cambio no está validado.');
  }
  return `${lines.join('\n')}\n`;
}

function writeReport(report) {
  if (args.report) {
    fs.writeFileSync(path.resolve(args.report), `${JSON.stringify(report, null, 2)}\n`);
  }
  if (args.format === 'json') {
    process.stdout.write(`${JSON.stringify(report, null, 2)}\n`);
  } else {
    process.stdout.write(reportText(report));
  }
}

let report = null;

function reject(error) {
  const rejected = {
    schema: REPORT_SCHEMA,
    plan: null,
    partial: !all,
    dryRun,
    synthetic: false,
    validated: false,
    validationStatus: 'rejected',
    required: [],
    suites: {},
    order: [],
    totals: {
      required: 0,
      selected: 0,
      executed: 0,
      cached: 0,
      simulated: 0,
      skipped: 0,
      pending: 0,
      failed: 0
    },
    error: error.message,
    startedAt: new Date().toISOString()
  };
  if (args.report) {
    fs.writeFileSync(path.resolve(args.report), `${JSON.stringify(rejected, null, 2)}\n`);
  }
  process.stderr.write(`Plan rechazado: ${error.message}\n`);
  process.exit(1);
}

function finish(exitCode) {
  report.finishedAt = new Date().toISOString();
  report.durationMs = Date.now() - report.startedAtMs;
  for (const suiteId of report.required) {
    const entry = report.suites[suiteId];
    if (!entry) continue;
    if (entry.state === 'pending' && !entry.selected) entry.state = 'skipped';
  }
  report.totals = summarizeReport(report);
  const everyRequiredResolved = report.required.every((suiteId) =>
    ['executed', 'cached'].includes(report.suites[suiteId].state)
  );
  report.validated =
    report.synthetic !== true &&
    !report.partial &&
    !report.dryRun &&
    report.required.length > 0 &&
    report.totals.failed === 0 &&
    report.totals.pending === 0 &&
    everyRequiredResolved;
  report.validationStatus =
    report.totals.failed > 0
      ? 'failed'
      : report.dryRun
        ? 'simulated'
        : report.synthetic
          ? 'synthetic'
          : report.partial
            ? 'partial'
          : report.totals.pending > 0
            ? 'incomplete'
            : report.validated
              ? 'complete'
              : 'incomplete';
  const receiptKey = loadReceiptKey({ repoRoot, create: true, jobId: reportJobId });
  if (args.report) {
    attachExecutionEvidence(report, path.resolve(args.report), receiptKey);
  }
  report.receipt = signReportReceipt(report, receiptKey);
  writeReport(report);
  // Ni un plan sintético ni un dry-run validan el diff del árbol. El código 2
  // impide que un consumidor que solo mire el exit code los confunda con una
  // validación completa.
  const finalCode =
    exitCode === 0 &&
    all &&
    (report.synthetic === true || report.dryRun === true || report.required.length === 0)
      ? 2
      : exitCode;
  process.exit(finalCode);
}

function executeSuite(suiteId, options, cacheDirectory) {
  const planEntry = plan.suites[suiteId];
  const result = report.suites[suiteId];
  if (!planEntry.selected) return 0;

  if (dryRun) {
    result.state = 'simulated';
    result.simulated = true;
    result.status = null;
    process.stdout.write(`${suiteId}: simulada; no se ejecutó\n`);
    return 0;
  }

  const cacheable = policy.suites[suiteId]?.cacheable === true;
  const cacheKey =
    args.cache && cacheable
      ? computeCacheKey(repoRoot, plan, {
          command: describeSuiteCommand(suiteId, planEntry)
        })
      : null;
  if (cacheKey) {
    const cached = readCache(
      repoRoot,
      cacheKey,
      args['cache-dir'] || cacheDirectory,
      receiptKey
    );
    if (cached && cached.suite === suiteId) {
      result.state = 'cached';
      result.cached = true;
      result.status = 0;
      result.durationMs = cached.durationMs || 0;
      process.stdout.write(`${suiteId}: resultado reutilizado desde caché\n`);
      return 0;
    }
  }

  const runner = RUNNERS[suiteId];
  if (!runner) throw new Error(`suite sin ejecutor: ${suiteId}`);
  const started = Date.now();
  result.executed = true;
  try {
    result.status = runner(repoRoot, plan, options);
  } catch (error) {
    process.stderr.write(`${suiteId}: ${error.message}\n`);
    result.status = 1;
    result.error = error.message;
  }
  result.durationMs = Date.now() - started;
  result.state = result.status === 0 ? 'executed' : 'failed';
  if (result.status === 0 && cacheKey) {
    try {
      writeCache(
        repoRoot,
        cacheKey,
        { suite: suiteId, status: 0, durationMs: result.durationMs },
        args['cache-dir'] || cacheDirectory,
        receiptKey
      );
    } catch (error) {
      process.stderr.write(`Aviso: no se pudo escribir la caché: ${error.message}\n`);
    }
  }
  return result.status;
}

let policy;
let plan;
let receiptKey = null;

try {
  if (all && partial) throw new Error('--all y --partial son incompatibles');
  if (all && suiteArg) throw new Error('--all no acepta --suite');
  if (!all && !suiteArg) throw new Error('indicá --all o --suite <id> --partial');
  if (!all && !partial) {
    throw new Error(
      `--suite ${suiteArg} exige --partial: una suite aislada no valida el cambio completo`
    );
  }
  if (ifSelected && !partial) throw new Error('--if-selected requiere --partial');
  if (!all && !RUNNERS[suiteArg]) throw new Error(`suite desconocida: ${suiteArg}`);

  policy = loadPolicy(repoRoot);
  const rawPlan = planFromArgs(args, repoRoot);
  plan = validatePlan({
    repoRoot,
    plan: rawPlan,
    policy,
    backendGraph: buildBackendGraph(repoRoot),
    frontendGraph: buildFrontendGraph(repoRoot)
  });
  receiptKey = loadReceiptKey({ repoRoot, create: true, jobId: reportJobId });

  if (!all && !plan.requiredSuites.includes(suiteArg)) {
    if (!ifSelected) {
      throw new Error(
        `la suite ${suiteArg} no está seleccionada por el plan; usá --if-selected si el job es condicional`
      );
    }
  }

  report = {
    schema: REPORT_SCHEMA,
    plan: {
      schema: plan.schema,
      contractVersion: plan.contractVersion,
      policyVersion: plan.policyVersion,
      policyDigest: plan.policyDigest,
      digest: planDigest(plan),
      mode: plan.mode,
      level: plan.level,
      source: plan.source
    },
    required: [...plan.requiredSuites].sort(),
    runId: reportRunId,
    jobId: reportJobId,
    partial: !all,
    dryRun,
    synthetic: plan.synthetic === true,
    validated: false,
    validationStatus: 'incomplete',
    level: plan.level,
    levelTitle: plan.levelTitle,
    mode: plan.mode,
    planReason: plan.reason,
    levelReasons: plan.levelReasons || [],
    startedAt: new Date().toISOString(),
    startedAtMs: Date.now(),
    suites: {},
    order: []
  };

  for (const suiteId of Object.keys(policy.suites)) {
    const entry = plan.suites[suiteId];
    report.suites[suiteId] = {
      selected: Boolean(entry.selected),
      state: entry.selected ? 'pending' : 'skipped',
      status: null,
      executed: false,
      cached: false,
      simulated: false,
      durationMs: 0,
      reason: entry.reason
    };
    if (entry.files) report.suites[suiteId].files = entry.files;
    report.order.push({ id: suiteId });
  }

  const options = { dryRun };
  const cacheDirectory = args['cache-dir'] || null;

  if (all) {
    for (const suiteId of report.order.map((item) => item.id)) {
      if (!report.suites[suiteId].selected) continue;
      const status = executeSuite(suiteId, options, cacheDirectory);
      if (status !== 0) finish(1);
    }
    finish(0);
  }

  if (!plan.requiredSuites.includes(suiteArg)) {
    // Ejecución condicional sin trabajo: no se presenta como validación.
    process.stdout.write(
      `La suite ${suiteArg} no fue seleccionada por el plan; no hay nada que ejecutar.\n`
    );
    finish(0);
  }

  const status = executeSuite(suiteArg, options, cacheDirectory);
  finish(status === 0 ? 0 : 1);
} catch (error) {
  if (process.env.SOLV_DEBUG) process.stderr.write(`${error.stack}\n`);
  if (!plan) {
    reject(error);
  } else {
    process.stderr.write(`Error del ejecutor de pruebas: ${error.message}\n`);
    process.exit(1);
  }
}
