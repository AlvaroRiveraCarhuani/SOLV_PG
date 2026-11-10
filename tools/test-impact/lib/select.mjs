import fs from 'node:fs';
import path from 'node:path';
import { allE2eSpecs, e2eSpecMetadata, matchingEntries } from './policy.mjs';
import { uniqueSorted } from './util.mjs';

const CODE_EXTENSIONS = new Set(['.go', '.ts', '.html', '.scss', '.json', '.mjs', '.js', '.sql']);
const FRONTEND_UNIT_EXTENSIONS = new Set(['.ts', '.html', '.scss']);
const LEVEL_RANK = { L0: 0, L1: 1, L2: 2, L3: 3 };
const LEVEL_BY_RANK = ['L0', 'L1', 'L2', 'L3'];
const VISUAL_EXTENSIONS = new Set(['.html', '.scss', '.css']);
// Plantilla Angular con lógica real: directivas de control de flujo, bindings,
// eventos, referencias o interpolación. Un HTML que solo contiene marcado
// decorativo no necesita compilación ni specs por sí mismo.
const ANGULAR_TEMPLATE_SYNTAX =
  /@(?:if|for|switch|defer|let|empty)\b|\{\{|\*ng(?:If|For|Switch|Class|Style)|\[\(?[a-zA-Z][\w.-]*\]|\([a-zA-Z][\w.-]*\)=|#\w+\s*=|\[ngClass\]|\[ngStyle\]/;
const CONFIG_NAMES = new Set([
  'angular.json',
  'package.json',
  'package-lock.json',
  'go.mod',
  'go.sum',
  'Makefile',
  'Dockerfile'
]);

function isCodePath(filePath) {
  const base = path.posix.basename(filePath);
  if (CONFIG_NAMES.has(base)) return true;
  if (base.endsWith('.yml') || base.endsWith('.yaml')) return true;
  if (base.startsWith('Dockerfile')) return true;
  return CODE_EXTENSIONS.has(path.posix.extname(filePath));
}

function isVisualPath(filePath) {
  // index.html es el shell de arranque, no una vista: no se degrada a L0 ni se
  // le aplica la optimización de cambios visuales.
  if (filePath === 'frontend/src/index.html') return false;
  return filePath.startsWith('frontend/src/') && VISUAL_EXTENSIONS.has(path.posix.extname(filePath));
}

function readTemplate(repoRoot, filePath) {
  try {
    return fs.readFileSync(path.resolve(repoRoot, filePath), 'utf8');
  } catch {
    return null;
  }
}

function hasAngularLogic(repoRoot, filePath) {
  if (!filePath.endsWith('.html')) return false;
  const source = readTemplate(repoRoot, filePath);
  if (source === null) return false;
  ANGULAR_TEMPLATE_SYNTAX.lastIndex = 0;
  return ANGULAR_TEMPLATE_SYNTAX.test(source);
}

/**
 * Una plantilla Angular con lógica y sin spec no puede quedar sin validación.
 * La compilación del frontend compila sus bindings y directivas; se evita a
 * propósito seleccionar todas las specs, que no cubren esta plantilla.
 */
function templateNeedsCompilation(repoRoot, filePath, frontendGraph) {
  if (!hasAngularLogic(repoRoot, filePath)) return false;
  return frontendGraph.specsForPath(filePath).specs.length === 0;
}

function isAssetPath(filePath) {
  return filePath.startsWith('frontend/public/');
}

function maxLevel(left, right) {
  return LEVEL_RANK[left] >= LEVEL_RANK[right] ? left : right;
}

function areaLevel(area, change) {
  if (change.visualOnly && area.frontend === true && area.level !== 'L3') {
    // Una plantilla con lógica y sin spec baja a validación de módulo (build),
    // no a L0: compilar la plantilla es una validación real del cambio.
    return change.templateNeedsCompile ? 'L1' : 'L0';
  }
  return area.level || (area.backend ? 'L2' : area.frontend ? 'L1' : 'L0');
}

function specificity(glob) {
  return glob.replace(/[?*[\]{}]/g, '').length;
}

function bestAreas(policy, filePath) {
  const matches = matchingEntries(policy.areas, filePath);
  if (matches.length === 0) return [];
  const scored = matches.map((area) => ({
    area,
    score: Math.max(...area.globs.map(specificity))
  }));
  const best = Math.max(...scored.map((entry) => entry.score));
  // Las reglas empatadas se combinan: nunca se elige una y se descarta otra
  // que pueda aportar suites o flujos E2E. El orden de `policy.areas` no
  // participa en la decisión.
  return scored
    .filter((entry) => entry.score === best)
    .map((entry) => entry.area)
    .sort((left, right) => left.id.localeCompare(right.id));
}

function suite(selected, reason, extra = {}) {
  return { selected, reason, ...extra };
}

function emptySuites() {
  return {
    gates: suite(false, 'sin gates de estilo o semántica'),
    'asset-validation': suite(false, 'sin recursos estáticos', { files: [] }),
    'backend-unit': suite(false, 'sin paquetes afectados', { packages: [], full: false }),
    'backend-integration': suite(false, 'sin cambios de backend'),
    'backend-docker': suite(false, 'sin cambios que ejecuten contenedores'),
    'frontend-unit': suite(false, 'sin specs afectadas', { specs: [], full: false }),
    'frontend-e2e': suite(false, 'sin flujos afectados', { specs: [] }),
    migrations: suite(false, 'sin cambios de esquema'),
    build: suite(false, 'sin cambios compilables', { backend: false, frontend: false }),
    security: suite(false, 'sin áreas críticas')
  };
}

function fullSuites(policy, scope, backendGraph, frontendGraph, reason) {
  const suites = emptySuites();
  const includeBackend = scope === 'all' || scope === 'backend';
  const includeFrontend = scope === 'all' || scope === 'frontend';

  suites.gates = suite(includeFrontend, reason);
  suites['asset-validation'] = suite(
    includeFrontend,
    includeFrontend
      ? 'la ejecución completa valida todos los recursos estáticos'
      : 'alcance de backend',
    { files: [], full: includeFrontend }
  );
  suites['backend-unit'] = includeBackend
    ? suite(true, reason, { packages: [...backendGraph.unitPackages], full: true })
    : suite(false, 'alcance de frontend');
  suites['backend-integration'] = suite(includeBackend, reason);
  suites['backend-docker'] = suite(includeBackend, reason);
  suites['frontend-unit'] = includeFrontend
    ? suite(true, reason, { specs: [...frontendGraph.allSpecs], full: true })
    : suite(false, 'alcance de backend');
  suites['frontend-e2e'] = suite(true, reason, { specs: allE2eSpecs(policy) });
  suites.migrations = suite(includeBackend, reason);
  suites.build = suite(true, reason, { backend: includeBackend, frontend: includeFrontend });
  suites.security = suite(true, reason);
  return suites;
}

function normalizeChange(change) {
  return {
    status: String(change.status || 'M').charAt(0),
    path: change.path,
    oldPath: change.oldPath || null
  };
}

/**
 * Todo plan, incluido el completo, publica la decisión de cada suite y del
 * nivel. La validación independiente usa esa decisión como contrato, no como
 * fuente de verdad: vuelve a calcular las obligaciones desde la política.
 */
function completedPlan(plan) {
  return {
    ...plan,
    decisions: {
      level: {
        id: plan.level,
        title: plan.levelTitle,
        reasons: plan.levelReasons
      },
      suites: Object.fromEntries(
        Object.entries(plan.suites).map(([id, entry]) => [
          id,
          { selected: entry.selected, reason: entry.reason }
        ])
      )
    }
  };
}

export function createPlan({ policy, changes, backendGraph, frontendGraph, options = {} }) {
  const repoRoot = options.repoRoot || path.resolve(frontendGraph.frontendRoot, '..');
  const normalized = changes.map(normalizeChange).filter((change) => Boolean(change.path));
  const event = options.event || 'local';
  const base = options.base || null;
  const head = options.head || 'HEAD';
  const maxChangedFiles = options.maxChangedFiles || policy.limits.maxChangedFiles;
  const policyVersion = policy.version;

  const annotated = normalized.map((change) => {
    const paths = [change.path, change.oldPath].filter(Boolean);
    const visualOnly = paths.length > 0 && paths.every(isVisualPath);
    const areas = paths.flatMap((filePath) => bestAreas(policy, filePath));
    const critical = paths.flatMap((filePath) => matchingEntries(policy.critical, filePath));
    const globals = paths.flatMap((filePath) => matchingEntries(policy.globalTriggers, filePath));
    const neutral = paths.every((filePath) => matchingEntries([policy.neutral], filePath).length > 0);
    const uniqueAreas = [...new Map(areas.map((area) => [area.id, area])).values()];
    const templateNeedsCompile = paths.some((filePath) =>
      templateNeedsCompilation(repoRoot, filePath, frontendGraph)
    );
    return {
      ...change,
      visualOnly,
      areas: uniqueAreas,
      critical,
      globals,
      neutral,
      templateNeedsCompile
    };
  });

  const basePlan = {
    policyVersion,
    mode: 'targeted',
    reason: '',
    base,
    head,
    event,
    changed: annotated,
    unknown: [],
    structural: [],
    uncovered: { backend: [], frontend: [] },
    criticalAreas: [],
    coverageGaps: [],
    level: 'L0',
    levelTitle: policy.levels.L0.title,
    levelReasons: [],
    decisions: {},
    suites: emptySuites(),
    e2e: { required: false, specs: [] }
  };

  if (normalized.length === 0) {
    return completedPlan({
      ...basePlan,
      mode: 'full',
      level: 'L3',
      levelTitle: policy.levels.L3.title,
      levelReasons: [{ source: 'mode', level: 'L3', reason: options.forcedReason || 'diff sin base verificable' }],
      reason: options.forcedReason || 'no hay diff que analizar; no se puede determinar el impacto',
      suites: fullSuites(policy, 'all', backendGraph, frontendGraph, 'diff vacío')
    });
  }

  const forcedFull = options.full === true || (policy.fullEvents || []).includes(event);
  if (forcedFull) {
    return completedPlan({
      ...basePlan,
      mode: 'full',
      level: 'L3',
      levelTitle: policy.levels.L3.title,
      levelReasons: [{ source: 'event', level: 'L3', reason: options.full ? 'ejecución completa solicitada' : `evento ${event}` }],
      reason: options.full ? 'ejecución completa solicitada' : `evento ${event} exige ejecución completa`,
      suites: fullSuites(policy, 'all', backendGraph, frontendGraph, 'ejecución completa')
    });
  }

  if (annotated.every((change) => change.neutral)) {
    return completedPlan({
      ...basePlan,
      mode: 'neutral',
      level: 'L0',
      levelTitle: policy.levels.L0.title,
      levelReasons: [{ source: 'neutral', level: 'L0', reason: 'solo documentación o herramientas sin impacto' }],
      reason: 'solo documentación o herramientas sin impacto en SOLV',
      suites: emptySuites()
    });
  }

  if (normalized.length > maxChangedFiles) {
    return completedPlan({
      ...basePlan,
      mode: 'full',
      level: 'L3',
      levelTitle: policy.levels.L3.title,
      levelReasons: [{ source: 'limit', level: 'L3', reason: `${normalized.length} archivos superan el máximo` }],
      reason: `${normalized.length} archivos cambiados superan el límite de ${maxChangedFiles}`,
      suites: fullSuites(policy, 'all', backendGraph, frontendGraph, 'demasiados archivos')
    });
  }

  const globalAll = annotated.some((change) => change.globals.some((entry) => entry.scope === 'all'));
  if (globalAll) {
    const triggers = uniqueSorted(
      annotated.flatMap((change) => change.globals.map((entry) => entry.id))
    );
    return completedPlan({
      ...basePlan,
      mode: 'full',
      level: 'L3',
      levelTitle: policy.levels.L3.title,
      levelReasons: [{ source: 'global', level: 'L3', reason: `configuración global: ${triggers.join(', ')}` }],
      reason: `configuración global modificada: ${triggers.join(', ')}`,
      suites: fullSuites(policy, 'all', backendGraph, frontendGraph, 'configuración global')
    });
  }

  const unknown = uniqueSorted(
    annotated
      .filter(
        (change) =>
          !change.neutral &&
          change.areas.length === 0 &&
          change.critical.length === 0 &&
          change.globals.length === 0
      )
      .flatMap((change) => [change.path, change.oldPath].filter(Boolean))
  );
  if (unknown.length > 0) {
    return completedPlan({
      ...basePlan,
      mode: 'full',
      level: 'L3',
      levelTitle: policy.levels.L3.title,
      levelReasons: [{ source: 'unknown', level: 'L3', reason: `rutas desconocidas: ${unknown.join(', ')}` }],
      reason: `el selector no puede clasificar con certeza: ${unknown.join(', ')}`,
      unknown,
      suites: fullSuites(policy, 'all', backendGraph, frontendGraph, 'rutas desconocidas')
    });
  }

  const scopeFull = { backend: false, frontend: false };
  const structural = [];
  for (const change of annotated) {
    if (!['D', 'R'].includes(change.status)) continue;
    for (const filePath of [change.path, change.oldPath].filter(Boolean)) {
      if (change.neutral) continue;
      if (filePath.startsWith('frontend/e2e/')) {
        if (!filePath.endsWith('.spec.mjs')) scopeFull.frontend = true;
        structural.push(filePath);
        continue;
      }
      if (!isCodePath(filePath)) continue;
      if (filePath.startsWith('backend/')) scopeFull.backend = true;
      if (filePath.startsWith('frontend/')) scopeFull.frontend = true;
      structural.push(filePath);
    }
  }

  const scopedGlobalBackend = annotated.some((change) =>
    change.globals.some((entry) => entry.scope === 'backend')
  );
  const scopedGlobalFrontend = annotated.some((change) =>
    change.globals.some((entry) => entry.scope === 'frontend')
  );

  if (scopeFull.backend || scopedGlobalBackend) {
    scopeFull.backend = true;
  }
  if (scopeFull.frontend || scopedGlobalFrontend) {
    scopeFull.frontend = true;
  }

  const critical = annotated.flatMap((change) => change.critical);
  const criticalIds = uniqueSorted(critical.map((entry) => entry.id));
  const criticalBackendAll = critical.some((entry) => entry.backendUnit === 'all');
  const criticalFrontendAll = critical.some((entry) => entry.frontendUnit === 'all');
  const criticalIntegration = critical.some((entry) => entry.backendIntegration === true);
  const criticalDocker = critical.some((entry) => entry.backendDocker === true);
  const criticalMigrations = critical.some((entry) => entry.migrations === true);
  const criticalE2eAll = critical.some((entry) => entry.e2eAll === true);
  const criticalE2e = critical.flatMap((entry) => entry.e2e || []);
  basePlan.coverageGaps = uniqueSorted(
    annotated.flatMap((change) => change.critical.flatMap((entry) => entry.coverageGaps || []))
  );

  const areas = annotated.flatMap((change) => change.areas);
  const levelReasons = [];
  let level = 'L0';
  for (const change of annotated) {
    for (const area of change.areas) {
      const candidate = areaLevel(area, change);
      if (LEVEL_RANK[candidate] > LEVEL_RANK[level]) level = candidate;
      levelReasons.push({
        source: 'area',
        id: area.id,
        level: candidate,
        reason: change.visualOnly && candidate === 'L0'
          ? 'cambio visual localizado; no cruza contrato ni backend'
          : area.level === 'L0'
            ? 'recurso estático aislado'
            : `regla de área ${area.id}`
      });
    }
    for (const entry of change.critical) {
      level = maxLevel(level, entry.level || 'L3');
      levelReasons.push({ source: 'critical', id: entry.id, level: entry.level || 'L3', reason: entry.reason });
    }
    for (const entry of change.globals) {
      level = maxLevel(level, entry.level || 'L3');
      levelReasons.push({ source: 'global', id: entry.id, level: entry.level || 'L3', reason: entry.reason });
    }
  }
  if (scopeFull.backend || scopeFull.frontend) {
    level = 'L3';
    levelReasons.push({ source: 'scope', level: 'L3', reason: 'eliminación, renombrado o configuración global' });
  }
  if (levelReasons.length === 0) {
    levelReasons.push({ source: 'localized', level: 'L0', reason: 'sin dependencia demostrada' });
  }

  const globalStyleTouched = annotated.some((change) =>
    [change.path, change.oldPath].filter(Boolean).some(
      (filePath) => filePath.startsWith('frontend/src/styles/') || filePath === 'frontend/src/styles.scss'
    )
  );
  const allVisualOnly = annotated.every((change) => change.visualOnly) && !globalStyleTouched;
  const buildBackend =
    scopeFull.backend ||
    areas.some((area) => area.backend === true && area.build === true) ||
    criticalIds.includes('shared-contracts');
  const templateCompile = annotated.some((change) => change.templateNeedsCompile);
  let buildFrontend =
    scopeFull.frontend ||
    templateCompile ||
    areas.some((area) => area.frontend === true && area.build === true);
  const bootstrapCompile = areas.some((area) => area.id === 'frontend-bootstrap');
  // La optimización visual no puede anular una obligación crítica de
  // compilación ni la compilación de una plantilla sin spec.
  if (
    allVisualOnly &&
    !scopeFull.frontend &&
    criticalIds.length === 0 &&
    !templateCompile &&
    !bootstrapCompile
  ) {
    buildFrontend = false;
  }
  const assetFiles = uniqueSorted(
    annotated
      .filter((change) => change.status !== 'D')
      .flatMap((change) => [change.path, change.oldPath].filter(Boolean))
      .filter(isAssetPath)
  );

  const backendTouched = scopeFull.backend || areas.some((area) => area.backend === true);
  const frontendTouched = scopeFull.frontend || areas.some((area) => area.frontend === true);
  const buildTouched = buildBackend || buildFrontend;
  const integrationTouched =
    areas.some((area) => area.integration === true) || scopeFull.backend || criticalIntegration;
  const areaFrontendAll = areas.some((area) => area.frontendUnitAll === true);
  const areaE2eAll =
    areas.some((area) => area.e2eAll === true) || scopeFull.backend || scopeFull.frontend;
  const areaE2e = areas.flatMap((area) => area.e2e || []);
  const scssTouched = annotated.some((change) =>
    [change.path, change.oldPath].filter(Boolean).some((filePath) => filePath.endsWith('.scss'))
  );
  const gatesTouched = areas.some((area) => area.gates === true) || scssTouched || scopeFull.frontend;
  const e2eSelf = annotated
    .filter((change) => change.areas.some((area) => area.e2eSelf === true))
    .map((change) => {
      const specPath = change.path.split('/').slice(1).join('/');
      return e2eSpecMetadata(policy, specPath) ? specPath : null;
    })
    .filter(Boolean);

  const suites = emptySuites();
  suites.gates = suite(
    gatesTouched,
    gatesTouched
      ? scssTouched
        ? 'hay estilos de componente o globales que validar'
        : 'hay scripts de gates afectados'
      : 'sin estilos ni scripts de gates afectados'
  );
  suites['asset-validation'] = assetFiles.length
    ? suite(true, `${assetFiles.length} recurso(s) estáticos modificados`, { files: assetFiles })
    : suite(false, 'sin recursos estáticos modificados', { files: [] });

  const backendPackages = new Set();
  for (const change of annotated) {
    if (change.neutral) continue;
    for (const filePath of [change.path, change.oldPath].filter(Boolean)) {
      if (!filePath.startsWith('backend/')) continue;
      if (!change.areas.some((area) => area.backend === true) && !scopeFull.backend) continue;
      const pkg = backendGraph.packageForPath(filePath);
      if (!pkg) continue;
      if (filePath.endsWith('_test.go')) {
        backendPackages.add(pkg);
      } else if (['D', 'R'].includes(change.status)) {
        backendPackages.add(pkg);
        for (const dependent of backendGraph.reverseClosure([pkg])) backendPackages.add(dependent);
      } else {
        for (const dependent of backendGraph.reverseClosure([pkg])) backendPackages.add(dependent);
      }
    }
  }

  let unitPackages = [...backendPackages].filter((pkg) => backendGraph.unitPackages.includes(pkg));
  if (scopeFull.backend || criticalBackendAll) {
    unitPackages = [...backendGraph.unitPackages];
  }
  unitPackages = uniqueSorted(unitPackages);
  suites['backend-unit'] = unitPackages.length
    ? suite(
        true,
        criticalBackendAll || scopeFull.backend
          ? 'área crítica o alcance completo del backend'
          : `${unitPackages.length} paquete(s) por dependencias directas e inversas`,
        { packages: unitPackages, full: criticalBackendAll || scopeFull.backend }
      )
    : suite(false, 'no hay paquetes con tests afectados', { packages: [], full: false });

  suites['backend-integration'] = suite(
    integrationTouched,
    integrationTouched
      ? criticalIntegration
        ? 'área crítica de backend'
        : 'hay cambios que llegan a PostgreSQL'
      : 'solo tests unitarios o documentación'
  );

  suites['backend-docker'] = suite(
    criticalDocker || scopeFull.backend,
    criticalDocker || scopeFull.backend
      ? 'se modificó ejecución de contenedores o el alcance es completo'
      : 'sin cambios de ejecución de contenedores'
  );

  let specs = new Set();
  let ownershipUnknown = false;
  if (scopeFull.frontend || criticalFrontendAll || areaFrontendAll) {
    specs = new Set(frontendGraph.allSpecs);
  } else {
    for (const change of annotated) {
      if (change.neutral) continue;
      for (const filePath of [change.path, change.oldPath].filter(Boolean)) {
        if (!filePath.startsWith('frontend/src/')) continue;
        if (!change.areas.some((area) => area.frontend === true)) continue;
        const resolution = frontendGraph.specsForPath(filePath);
        for (const spec of resolution.specs) specs.add(spec);
        const extension = path.posix.extname(filePath);
        if (
          resolution.specs.length === 0 &&
          FRONTEND_UNIT_EXTENSIONS.has(extension) &&
          !filePath.endsWith('.spec.ts')
        ) {
          if (extension === '.html') {
            // Un HTML sin spec nunca selecciona todas las specs por sí solo.
            // Si tiene lógica, el build compila la plantilla y queda registrado
            // como cobertura faltante; si es decorativo, no exige nada.
            if (hasAngularLogic(repoRoot, filePath)) {
              basePlan.uncovered.frontend.push(filePath);
            }
          } else {
            basePlan.uncovered.frontend.push(filePath);
            // Un SCSS huérfano puede afectar a cualquier vista; un .ts sin
            // importadores queda como cobertura faltante sin arrastrar todo.
            if (!resolution.owned && extension === '.scss') ownershipUnknown = true;
          }
        }
      }
    }
  }

  if (ownershipUnknown) {
    specs = new Set(frontendGraph.allSpecs);
  }

  const existingSpecs = new Set(frontendGraph.allSpecs);
  const frontendSpecs = uniqueSorted([...specs].filter((spec) => existingSpecs.has(spec)));
  suites['frontend-unit'] = frontendSpecs.length
    ? suite(
        true,
        scopeFull.frontend || criticalFrontendAll || areaFrontendAll || ownershipUnknown
          ? 'alcance completo del frontend'
          : `${frontendSpecs.length} spec(s) por dependencias de importación`,
        { specs: frontendSpecs, full: scopeFull.frontend || criticalFrontendAll || areaFrontendAll || ownershipUnknown }
      )
    : suite(false, 'no hay specs afectadas', { specs: [], full: false });

  let e2eSpecs;
  if (areaE2eAll || criticalE2eAll) {
    e2eSpecs = allE2eSpecs(policy);
  } else {
    // L1 valida el módulo y no paga E2E por áreas genéricas; los flujos
    // propios de una spec y las reglas críticas siempre se conservan.
    const selected = uniqueSorted([
      ...(LEVEL_RANK[level] >= LEVEL_RANK.L2 ? areaE2e : []),
      ...criticalE2e,
      ...e2eSelf
    ]);
    e2eSpecs = selected.map((spec) => e2eSpecMetadata(policy, spec)).filter(Boolean);
  }
  suites['frontend-e2e'] = e2eSpecs.length
    ? suite(true, `${e2eSpecs.length} flujo(s) por área, contrato o seguridad`, { specs: e2eSpecs })
    : suite(false, 'sin flujos E2E asociados al cambio', { specs: [] });

  suites.migrations = suite(
    criticalMigrations || scopeFull.backend,
    criticalMigrations || scopeFull.backend
      ? 'esquema de base de datos o alcance completo'
      : 'sin cambios de esquema'
  );
  suites.build = suite(
    buildTouched,
    buildTouched
      ? templateCompile && !scopeFull.frontend
        ? 'compilación de plantillas sin spec'
        : `compilación de ${[buildBackend ? 'backend' : null, buildFrontend ? 'frontend' : null].filter(Boolean).join(' y ')}`
      : 'sin compilación necesaria para un cambio visual localizado o estático',
    { backend: buildBackend, frontend: buildFrontend }
  );
  suites.security = suite(
    criticalIds.length > 0 || scopeFull.backend || scopeFull.frontend,
    criticalIds.length > 0
      ? `áreas críticas: ${criticalIds.join(', ')}`
      : scopeFull.backend || scopeFull.frontend
        ? 'alcance completo'
        : 'sin áreas críticas'
  );

  if (backendTouched) {
    for (const change of annotated) {
      if (change.neutral) continue;
      if (!change.path.startsWith('backend/')) continue;
      if (!change.path.endsWith('.go') || change.path.endsWith('_test.go')) continue;
      const pkg = backendGraph.packageForPath(change.path);
      if (!pkg) continue;
      const covered = backendGraph.reverseClosure([pkg]).some((entry) =>
        backendGraph.unitPackages.includes(entry)
      );
      if (!covered && !criticalBackendAll && !scopeFull.backend) {
        basePlan.uncovered.backend.push(change.path);
      }
    }
  }

  basePlan.mode = 'targeted';
  basePlan.level = level;
  basePlan.levelTitle = policy.levels[level].title;
  basePlan.levelReasons = levelReasons;
  basePlan.reason = criticalIds.length
    ? `cambio dirigido con cobertura crítica (${criticalIds.join(', ')})`
    : 'cambio dirigido por impacto';
  basePlan.criticalAreas = criticalIds;
  basePlan.suites = suites;
  basePlan.decisions = {
    level: {
      id: level,
      title: policy.levels[level].title,
      reasons: levelReasons
    },
    suites: Object.fromEntries(
      Object.entries(suites).map(([id, entry]) => [
        id,
        { selected: entry.selected, reason: entry.reason }
      ])
    )
  };
  basePlan.e2e = {
    required: suites['frontend-e2e'].selected,
    specs: suites['frontend-e2e'].specs
  };
  basePlan.uncovered.backend = uniqueSorted(basePlan.uncovered.backend);
  basePlan.uncovered.frontend = uniqueSorted(basePlan.uncovered.frontend);
  if (structural.length > 0) basePlan.structural = uniqueSorted(structural);

  if (scopeFull.backend || scopeFull.frontend) {
    const scope = scopeFull.backend && scopeFull.frontend ? 'all' : scopeFull.backend ? 'backend' : 'frontend';
    return completedPlan({
      ...basePlan,
      mode: 'full',
      reason: `${basePlan.reason}; alcance completo de ${
        scope === 'all' ? 'backend y frontend' : scope
      } por eliminación, renombrado o configuración global`,
      suites: fullSuites(policy, scope, backendGraph, frontendGraph, 'alcance completo')
    });
  }

  return completedPlan(basePlan);
}

export function summarizePlan(plan) {
  const lines = [];
  lines.push(`Nivel: ${plan.level || 'L0'} — ${plan.levelTitle || 'Validación inmediata'}`);
  lines.push(`Modo: ${plan.mode} — ${plan.reason}`);
  for (const [id, entry] of Object.entries(plan.suites)) {
    const marker = entry.selected ? 'x' : '-';
    let detail = '';
    if (entry.packages) detail = ` (${entry.packages.length} paquetes)`;
    if (entry.specs) detail = ` (${entry.specs.length} specs)`;
    if (id === 'build' && (entry.backend || entry.frontend)) {
      detail = ` (${[entry.backend ? 'backend' : null, entry.frontend ? 'frontend' : null].filter(Boolean).join(' + ')})`;
    }
    lines.push(`[${marker}] ${id}${detail}: ${entry.reason}`);
  }
  if (plan.criticalAreas.length) lines.push(`Áreas críticas: ${plan.criticalAreas.join(', ')}`);
  if (plan.levelReasons?.length) {
    lines.push('Justificación del nivel:');
    for (const reason of plan.levelReasons) {
      lines.push(`- ${reason.level} ${reason.source}${reason.id ? `:${reason.id}` : ''} — ${reason.reason}`);
    }
  }
  if (plan.coverageGaps?.length) {
    lines.push('Cobertura faltante declarada:');
    for (const gap of plan.coverageGaps) lines.push(`- ${gap}`);
  }
  if (plan.unknown.length) lines.push(`Rutas desconocidas: ${plan.unknown.join(', ')}`);
  if (plan.uncovered.backend.length)
    lines.push(`Backend sin tests: ${plan.uncovered.backend.join(', ')}`);
  if (plan.uncovered.frontend.length)
    lines.push(`Frontend sin specs: ${plan.uncovered.frontend.join(', ')}`);
  return `${lines.join('\n')}\n`;
}
