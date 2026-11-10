#!/usr/bin/env node
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { loadPolicy } from './lib/policy.mjs';
import { buildBackendGraph } from './lib/backend-graph.mjs';
import { buildFrontendGraph } from './lib/frontend-graph.mjs';
import { createPlan } from './lib/select.mjs';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(HERE, '..', '..');

// Pesos orientativos en segundos, no mediciones. Sirven para comparar el orden
// de magnitud entre escenarios, no para prometer una duración.
const SUITE_COST = {
  gates: 25,
  'asset-validation': 1,
  'backend-unit': 30,
  'backend-integration': 90,
  'backend-docker': 240,
  'frontend-unit': 45,
  'frontend-e2e': 35,
  migrations: 45,
  build: 45,
  security: 20
};
const FULL_FRONTEND_COST = 60;
const FULL_BACKEND_COST = 30;
const FLOW_COST = 25;

const SCENARIOS = [
  {
    name: 'Icono SVG',
    path: 'frontend/public/tech-icons/python.svg',
    change: 'M'
  },
  {
    name: 'HTML decorativo',
    path: 'frontend/src/app/features/teacher/exercise-editor/steps/step-publication/step-publication.component.html',
    change: 'M'
  },
  {
    name: 'HTML Angular con bindings y sin spec',
    path: 'frontend/src/app/features/admin/environment-requests/admin-environment-requests.component.html',
    change: 'M'
  },
  {
    name: 'Formulario',
    path: 'frontend/src/app/features/admin/courses/components/course-create-modal/course-create-modal.component.ts',
    change: 'M'
  },
  {
    name: 'Servicio frontend',
    path: 'frontend/src/app/features/admin/services/admin-courses.service.ts',
    change: 'M'
  },
  {
    name: 'Endpoint Go',
    path: 'backend/internal/delivery/http/admin_academic_handler.go',
    change: 'M'
  },
  {
    name: 'Autenticación backend',
    path: 'backend/internal/core/services/auth_service.go',
    change: 'M'
  },
  {
    name: 'Autenticación frontend',
    path: 'frontend/src/app/core/guards/auth.guard.ts',
    change: 'M'
  },
  {
    name: 'WebSocket backend',
    path: 'backend/internal/delivery/http/websocket_handler.go',
    change: 'M'
  },
  {
    name: 'WebSocket frontend',
    path: 'frontend/src/app/features/teacher/aula/shadow-terminal-modal/shadow-terminal-modal.component.ts',
    change: 'M'
  },
  {
    name: 'Motor del juez',
    path: 'backend/internal/core/services/judge_service.go',
    change: 'M'
  },
  {
    name: 'Migración SQL',
    path: 'backend/migrations/00051_submission_verdict_origin.sql',
    change: 'M'
  },
  {
    name: 'Infraestructura Docker',
    path: 'backend/internal/infrastructure/docker/docker_container.go',
    change: 'M'
  },
  {
    name: 'Contrato compartido',
    path: 'frontend/src/app/shared/models/api-response.model.ts',
    change: 'M'
  },
  {
    name: 'index.html',
    path: 'frontend/src/index.html',
    change: 'M'
  }
];

function estimateCost(plan) {
  let seconds = 0;
  for (const [suiteId, entry] of Object.entries(plan.suites)) {
    if (!entry.selected) continue;
    seconds += SUITE_COST[suiteId] || 0;
    if (suiteId === 'frontend-e2e') seconds += Math.max(0, (entry.specs?.length || 0) - 1) * FLOW_COST;
    if (suiteId === 'frontend-unit' && entry.full) seconds += FULL_FRONTEND_COST;
    if (suiteId === 'backend-unit' && entry.full) seconds += FULL_BACKEND_COST;
  }
  return seconds;
}

function buildRow(scenario, plan, selectionMs) {
  const selected = Object.entries(plan.suites)
    .filter(([, entry]) => entry.selected)
    .map(([id, entry]) => ({
      id,
      reason: entry.reason,
      details:
        id === 'frontend-e2e'
          ? entry.specs.map((item) => item.spec)
          : id === 'frontend-unit'
            ? entry.specs.length
            : id === 'backend-unit'
              ? entry.packages.length
              : id === 'asset-validation'
                ? entry.files
                : undefined
    }));
  const omitted = Object.entries(plan.suites)
    .filter(([, entry]) => !entry.selected)
    .map(([id, entry]) => ({ id, reason: entry.reason }));
  return {
    name: scenario.name,
    path: scenario.path,
    level: plan.level,
    levelTitle: plan.levelTitle,
    mode: plan.mode,
    criticalAreas: plan.criticalAreas,
    selected,
    omitted,
    build: plan.suites.build.selected
      ? {
          backend: plan.suites.build.backend === true,
          frontend: plan.suites.build.frontend === true
        }
      : null,
    uncovered: plan.uncovered,
    coverageGaps: plan.coverageGaps || [],
    estimatedSeconds: estimateCost(plan),
    selectionMs: Math.round(selectionMs)
  };
}

function text(rows) {
  const lines = [];
  lines.push('Benchmark de testing por impacto (selector real, cambios aislados)');
  lines.push('');
  for (const [index, row] of rows.entries()) {
    lines.push(
      `${index + 1}. ${row.name} — ${row.level} (${row.levelTitle}); selección ${row.selectionMs} ms; coste estimado ${row.estimatedSeconds}s`
    );
    lines.push(`   archivo: ${row.path}`);
    lines.push(
      `   seleccionadas: ${row.selected.map((entry) => entry.id).join(', ') || '(ninguna)'}`
    );
    for (const entry of row.selected) {
      lines.push(
        `     - ${entry.id}: ${entry.reason}${
          Array.isArray(entry.details)
            ? ` [${entry.details.join(', ')}]`
            : entry.details === undefined
              ? ''
              : ` (${entry.details})`
        }`
      );
    }
    lines.push(
      `   omitidas: ${row.omitted.map((entry) => `${entry.id} (${entry.reason})`).join('; ') || '(ninguna)'}`
    );
    lines.push(
      `   build: ${
        row.build
          ? [row.build.backend ? 'backend' : null, row.build.frontend ? 'frontend' : null]
              .filter(Boolean)
              .join(' + ')
          : 'no requerido'
      }`
    );
    lines.push(
      `   cobertura faltante: ${
        [...row.uncovered.backend, ...row.uncovered.frontend].join(', ') || 'ninguna detectada'
      }`
    );
    if (row.coverageGaps.length) {
      lines.push(`   brechas críticas declaradas: ${row.coverageGaps.join(' | ')}`);
    }
    lines.push('');
  }
  const total = rows.reduce((sum, row) => sum + row.estimatedSeconds, 0);
  lines.push(`Coste estimado agregado de los 15 escenarios: ${total}s.`);
  lines.push('Los pesos son orientativos; no son mediciones de suites reales.');
  return `${lines.join('\n')}\n`;
}

function markdown(rows) {
  const lines = [];
  lines.push('| # | Escenario | Nivel | Suites seleccionadas | Build | Cobertura faltante | Coste estimado |');
  lines.push('| ---: | :--- | :--- | :--- | :--- | :--- | ---: |');
  for (const [index, row] of rows.entries()) {
    const build = row.build
      ? [row.build.backend ? 'backend' : null, row.build.frontend ? 'frontend' : null]
          .filter(Boolean)
          .join(' + ')
      : 'no';
    const uncovered = [...row.uncovered.backend, ...row.uncovered.frontend];
    lines.push(
      `| ${index + 1} | ${row.name} | ${row.level} | ${
        row.selected.map((entry) => `\`${entry.id}\``).join(', ') || '—'
      } | ${build} | ${uncovered.length ? uncovered.map((item) => `\`${item}\``).join('<br>') : '—'} | ${
        row.estimatedSeconds
      }s |`
    );
  }
  return `${lines.join('\n')}\n`;
}

const policy = loadPolicy(REPO_ROOT);
const backendGraph = buildBackendGraph(REPO_ROOT);
const frontendGraph = buildFrontendGraph(REPO_ROOT);
const rows = SCENARIOS.map((scenario) => {
  const selectionStarted = performance.now();
  const plan = createPlan({
    policy,
    changes: [{ status: scenario.change, path: scenario.path }],
    backendGraph,
    frontendGraph,
    options: { event: 'bench', repoRoot: REPO_ROOT }
  });
  return buildRow(scenario, plan, performance.now() - selectionStarted);
});

const format = process.argv.includes('--format')
  ? process.argv[process.argv.indexOf('--format') + 1]
  : 'text';
process.stdout.write(format === 'json' ? `${JSON.stringify(rows, null, 2)}\n` : format === 'md' ? markdown(rows) : text(rows));
