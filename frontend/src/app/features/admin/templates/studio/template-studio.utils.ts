import {
  RuntimeCapabilities,
  TargetEnvironment,
  TemplateComponent
} from '../../services/admin-templates.service';

// --- Resolución de propuestas del Copiloto contra el catálogo real del tenant ---

// findCatalogComponentMatch resuelve un slug propuesto por el Copiloto
// (python, node, cpp, angular...) contra un componente del catálogo activo por
// id, slug o icon_key, en ese orden. Devuelve nil si el slug no existe hoy en
// el catálogo: la pieza se marca como no disponible en la propuesta.
export function findCatalogComponentMatch(components: TemplateComponent[], rawSlug: string): TemplateComponent | null {
	const key = (rawSlug || '').trim().toLowerCase();
	if (!key) {
		return null;
	}
	for (const c of components) {
		if ((c.id || '').toLowerCase() === key) {
			return c;
		}
	}
	for (const c of components) {
		if ((c.slug || '').toLowerCase() === key) {
			return c;
		}
	}
	for (const c of components) {
		if ((c.icon_key || '').toLowerCase() === key) {
			return c;
		}
	}
	return null;
}

// findDatabaseOptionMatch resuelve un slug de motor propuesto (sqlite, postgres)
// contra las opciones de base de datos reales del nodo, por service_engine, id o
// slug. Devuelve nil si el motor no está disponible o el slug no existe.
export function findDatabaseOptionMatch(options: StudioDatabaseOption[], rawSlug: string): StudioDatabaseOption | null {
	const key = (rawSlug || '').trim().toLowerCase();
	if (!key || key === 'none' || key === 'null') {
		return null;
	}
	for (const o of options) {
		if ((o.service_engine || '').toLowerCase() === key && (o.service_engine || '').toLowerCase() !== 'none') {
			return o;
		}
	}
	for (const o of options) {
		if ((o.id || '').toLowerCase() === key) {
			return o;
		}
	}
	for (const o of options) {
		if ((o.slug || '').toLowerCase() === key) {
			return o;
		}
	}
	return null;
}

export interface DockerfileDiagnostic {
  severity: 'error' | 'warning';
  message: string;
}

export interface VerificationParameters {
  targetEnvironment: string;
  dockerImage: string;
  dockerfileContent: string;
  setupScript: string;
  entrypoint: string;
  toolsDeclared: string;
  timeoutMS: number;
  ramMB: number;
}

export interface CapacityOption {
  mb: number;
  title: string;
  desc: string;
  isRecommended: boolean;
}

export interface StudioDatabaseOption extends TemplateComponent {
  is_available: boolean;
  unavailable_reason?: string;
  version?: string;
}

export interface StudioActionBlocker {
  reason: string;
  tone: 'neutral' | 'warning' | 'error';
}

export interface StudioResourceBreakdown {
  isJudge: boolean;
  totalMB: number;
  editorBaseMB: number;
  stackBaseMB: number;
  usableMB: number;
}

export interface AiProposalView {
  runtime: TemplateComponent | null;
  runtimeSlug: string;
  web: TemplateComponent | null;
  webSlug: string;
  addons: TemplateComponent[];
  addonSlugs: string[];
  database: StudioDatabaseOption | null;
  databaseSlug: string;
  missing: string[];
  ramMB: number;
  hasSetupScript: boolean;
  rationale: string;
  latencyMs?: number;
}

// --- Modelo de la constelación visual del stack (mapa de herramientas) ---

export type ConstellationRole =
  | 'lenguaje'
  | 'web'
  | 'addon'
  | 'database';

export type ConstellationAvailability =
  | 'available'   // en el catálogo y compatible con el núcleo
  | 'incompatible' // en el catálogo pero incompatible con el núcleo (⚠)
  | 'missing';     // no existe en el catálogo (✗)

export interface ConstellationNode {
  role: ConstellationRole;
  name: string;
  slug: string;
  iconKey: string;
  availability: ConstellationAvailability;
  isCore: boolean;
  ramDeltaMB: number;
}

export interface StackConstellation {
  core: ConstellationNode | null;
  satellites: ConstellationNode[];
  ramMB: number;
  hasSetupScript: boolean;
}

// Devuelve true si el addon encaja con las familias activas del componente núcleo
// (misma lógica de familias que computeCompatibleAddonsForSelection).
export function addonFitsCore(addon: TemplateComponent, core: TemplateComponent): boolean {
  const activeFamilies = new Set<string>(['universal']);
  if (core.slug) activeFamilies.add(core.slug.toLowerCase());
  if (core.id) activeFamilies.add(core.id.toLowerCase());
  if (core.icon_key) activeFamilies.add(core.icon_key.toLowerCase());
  for (const fam of core.compatible_runtimes || []) {
    activeFamilies.add(fam.toLowerCase());
  }
  const compat = addon.compatible_runtimes || [];
  if (compat.length === 0) return true;
  return compat.some((c) => activeFamilies.has(c.toLowerCase()));
}

// Construye la constelación desde una propuesta del Copiloto. El núcleo es el
// lenguaje (o la web si no hay lenguaje); los satélites son web, complementos y
// base de datos. Los addons presentes pero incompatibles con ese núcleo pasan a
// estado 'incompatible'; los slugs sin pieza en el catálogo pasan a 'missing'.
export function buildConstellationFromProposal(
  view: AiProposalView,
  targetEnvironment: TargetEnvironment
): StackConstellation {
  const satellites: ConstellationNode[] = [];

  const nodeFromComponent = (
    comp: TemplateComponent,
    role: ConstellationRole,
    availability: ConstellationAvailability
  ): ConstellationNode => ({
    role,
    name: comp.name,
    slug: comp.slug || comp.id,
    iconKey: comp.icon_key,
    availability,
    isCore: false,
    ramDeltaMB: comp.ram_delta_mb || 0
  });

  // Núcleo: lenguaje principal; si no hay, la web.
  let core: ConstellationNode | null = null;
  if (view.runtime) {
    core = nodeFromComponent(view.runtime, 'lenguaje', 'available');
  } else if (view.web) {
    core = nodeFromComponent(view.web, 'web', 'available');
  }
  if (core) core.isCore = true;

  // Web como satélite cuando el núcleo es el lenguaje; si se propuso pero no
  // existe en el catálogo, se lista como faltante.
  if (view.runtime && view.web) {
    satellites.push(nodeFromComponent(view.web, 'web', 'available'));
  } else if (view.runtime && view.webSlug && targetEnvironment === 'IDE_PERSISTENTE') {
    satellites.push({
      role: 'web',
      name: view.webSlug,
      slug: view.webSlug,
      iconKey: '',
      availability: 'missing',
      isCore: false,
      ramDeltaMB: 0
    });
  }

  // Complementos: disponibles o incompatibles con el núcleo.
  const resolvedSlugs = new Set<string>();
  for (const addon of view.addons) {
    resolvedSlugs.add((addon.slug || '').toLowerCase());
    resolvedSlugs.add((addon.id || '').toLowerCase());
    resolvedSlugs.add((addon.icon_key || '').toLowerCase());
    const availability: ConstellationAvailability =
      core && !addonFitsCore(addon, view.runtime || view.web!)
        ? 'incompatible'
        : 'available';
    satellites.push(nodeFromComponent(addon, 'addon', availability));
  }

  // Slugs de addons sin pieza en el catálogo.
  for (const slug of view.addonSlugs || []) {
    const key = slug.toLowerCase();
    if (resolvedSlugs.has(key)) continue;
    satellites.push({ role: 'addon', name: slug, slug, iconKey: '', availability: 'missing', isCore: false, ramDeltaMB: 0 });
  }

  // Base de datos.
  if (view.database) {
    satellites.push(nodeFromComponent(view.database, 'database', 'available'));
  } else if (view.databaseSlug) {
    satellites.push({
      role: 'database',
      name: view.databaseSlug,
      slug: view.databaseSlug,
      iconKey: '',
      availability: 'missing',
      isCore: false,
      ramDeltaMB: 0
    });
  }

  // Núcleo ausente: se lista como nodo faltante para que la advertencia exista.
  if (!core && (view.runtimeSlug || view.webSlug)) {
    const missingSlug = view.runtimeSlug || view.webSlug;
    satellites.unshift({
      role: view.runtimeSlug ? 'lenguaje' : 'web',
      name: missingSlug,
      slug: missingSlug,
      iconKey: '',
      availability: 'missing',
      isCore: false,
      ramDeltaMB: 0
    });
  }

  return { core, satellites, ramMB: view.ramMB, hasSetupScript: view.hasSetupScript };
}

// Construye la constelación desde la selección actual del constructor (Paso 2)
// cuando no hay propuesta del Copiloto: todo lo seleccionado es compatible por
// construcción, así que nunca hay estados 'missing' ni 'incompatible'.
export function buildConstellationFromSelection(
  rt: TemplateComponent | null,
  web: TemplateComponent | null,
  addons: TemplateComponent[],
  database: StudioDatabaseOption | null,
  targetEnvironment: TargetEnvironment
): StackConstellation {
  const satellites: ConstellationNode[] = [];

  const nodeFromComponent = (
    comp: TemplateComponent,
    role: ConstellationRole
  ): ConstellationNode => ({
    role,
    name: comp.name,
    slug: comp.slug || comp.id,
    iconKey: comp.icon_key,
    availability: 'available',
    isCore: false,
    ramDeltaMB: comp.ram_delta_mb || 0
  });

  let core: ConstellationNode | null = null;
  if (rt) {
    core = nodeFromComponent(rt, 'lenguaje');
  } else if (web && targetEnvironment === 'IDE_PERSISTENTE') {
    core = nodeFromComponent(web, 'web');
  }
  if (core) core.isCore = true;

  if (rt && web && targetEnvironment === 'IDE_PERSISTENTE') {
    satellites.push(nodeFromComponent(web, 'web'));
  }
  for (const addon of addons) {
    satellites.push(nodeFromComponent(addon, 'addon'));
  }
  if (database && database.service_engine !== 'none') {
    satellites.push(nodeFromComponent(database, 'database'));
  }

  return { core, satellites, ramMB: 0, hasSetupScript: false };
}

export function analyzeDockerfile(content: string): DockerfileDiagnostic[] {
  const trimmed = (content || '').trim();
  if (!trimmed) return [];

  const diagnostics: DockerfileDiagnostic[] = [];
  const activeLines = trimmed
    .split('\n')
    .map(l => l.trim())
    .filter(l => l.length > 0 && !l.startsWith('#'));

  const fromLines = activeLines.filter(l => l.toUpperCase().startsWith('FROM '));
  if (fromLines.length === 0) {
    diagnostics.push({
      severity: 'error',
      message: 'El Dockerfile debe comenzar con una instrucción FROM válida (ej: FROM python:3.12-slim-bookworm).'
    });
  } else {
    for (const fl of fromLines) {
      const tokens = fl.split(/\s+/).filter(Boolean);
      const img = tokens[1] || '';
      if (!img || img.endsWith(':latest') || img.includes(':latest@') || (!img.includes(':') && !img.includes('@'))) {
        diagnostics.push({
          severity: 'error',
          message: 'El uso de :latest o imágenes sin versión fija está prohibido por reproducibilidad académica.'
        });
        break;
      }
    }
  }

  for (const line of activeLines) {
    if (line.includes('apt-get install')) {
      const hasY = /(?:^|\s)(-y|--assume-yes)\b/.test(line);
      if (!hasY) {
        diagnostics.push({
          severity: 'warning',
          message: 'Se detectó apt-get install sin el parámetro -y; la construcción desatendida podría detenerse esperando confirmación.'
        });
        break;
      }
    }
    if (/\bsudo\b/.test(line)) {
      diagnostics.push({
        severity: 'warning',
        message: 'Se detectó el comando "sudo". Los contenedores de aprovisionamiento se ejecutan con permisos administrativos.'
      });
      break;
    }
  }

  return diagnostics;
}

export function fnv1aHex(input: string): string {
  let h1 = 0x811c9dc5;
  let h2 = 0x811c9dc5;
  for (let i = 0; i < input.length; i++) {
    const ch = input.charCodeAt(i);
    h1 ^= ch;
    h1 = Math.imul(h1, 0x01000193);
    h2 ^= ch ^ (i & 0xff);
    h2 = Math.imul(h2, 0x01000193);
  }
  const p1 = (h1 >>> 0).toString(16).padStart(8, '0');
  const p2 = (h2 >>> 0).toString(16).padStart(8, '0');
  return (p1 + p2).slice(0, 12);
}

export function computeVerificationFingerprint(params: VerificationParameters): string {
  const normalizedTools = (params.toolsDeclared || '')
    .split(',')
    .map(t => t.trim())
    .filter(Boolean)
    .join(',');

  const canonical = [
    (params.targetEnvironment || '').trim(),
    (params.dockerImage || '').trim(),
    (params.dockerfileContent || '').trim(),
    (params.setupScript || '').trim(),
    (params.entrypoint || '').trim(),
    normalizedTools,
    params.timeoutMS ?? 0,
    params.ramMB ?? 0
  ].join(':::');

  return fnv1aHex(canonical);
}

export function computeCompatibleAddonsForSelection(
  techComponents: TemplateComponent[],
  rt: TemplateComponent | null,
  web: TemplateComponent | null,
  targetEnvironment: TargetEnvironment
): TemplateComponent[] {
  if (!rt && !web) {
    return [];
  }
  const activeFamilies = new Set<string>(['universal']);

  if (rt) {
    if (rt.slug) activeFamilies.add(rt.slug.toLowerCase());
    if (rt.id) activeFamilies.add(rt.id.toLowerCase());
    if (rt.icon_key) activeFamilies.add(rt.icon_key.toLowerCase());
    for (const fam of rt.compatible_runtimes || []) {
      activeFamilies.add(fam.toLowerCase());
    }
  }

  if (web && targetEnvironment === 'IDE_PERSISTENTE') {
    activeFamilies.add('node');
    activeFamilies.add('web');
    if (web.slug) activeFamilies.add(web.slug.toLowerCase());
    if (web.id) activeFamilies.add(web.id.toLowerCase());
    if (web.icon_key) activeFamilies.add(web.icon_key.toLowerCase());
    for (const fam of web.compatible_runtimes || []) {
      activeFamilies.add(fam.toLowerCase());
    }
  }

  return techComponents.filter(i => {
    if (i.layer !== 'addon') return false;
    const compat = i.compatible_runtimes || [];
    if (compat.length === 0) return true;
    return compat.some(c => activeFamilies.has(c.toLowerCase()));
  });
}

export function buildStudioDatabaseOptions(
  caps: RuntimeCapabilities | null,
  techComponents: TemplateComponent[]
): StudioDatabaseOption[] {
  const satellites = caps?.satellite_services || [];
  const catalogDbs = techComponents.filter(i => i.layer === 'database');

  const noneOption: StudioDatabaseOption = {
    id: 'db-none',
    slug: 'db-none',
    layer: 'database',
    sub_category: 'none',
    name: 'Sin Base de Datos',
    description: 'Solo archivos de código en la carpeta del alumno',
    icon_key: 'docker',
    base_image: '',
    compatible_runtimes: ['universal'],
    install_mode: 'none',
    package_names: [],
    build_layer_snippet: '',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: [],
    service_engine: 'none',
    ram_delta_mb: 0,
    is_system: true,
    is_active: true,
    sort_order: 300,
    is_available: true
  };

  if (satellites.length > 0) {
    const availableSatellites = satellites.filter(sat => sat.is_available);
    const mappedSatellites = availableSatellites.map((sat, idx): StudioDatabaseOption => {
      const engine = (sat.engine || '').toLowerCase();
      const catMatch = catalogDbs.find(c => (c.service_engine || '').toLowerCase() === engine);
      const verSuffix = sat.version ? ` v${sat.version}` : '';
      return {
        id: catMatch?.id || `db-${engine}`,
        slug: catMatch?.slug || `${engine}-service`,
        layer: 'database',
        sub_category: sat.category || 'database',
        name: `${sat.label}${verSuffix}`,
        description: sat.description || catMatch?.description || '',
        icon_key: catMatch?.icon_key || engine,
        base_image: '',
        compatible_runtimes: ['universal'],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: catMatch?.tools_provided || [],
        service_engine: engine,
        ram_delta_mb: sat.base_ram_mb ?? catMatch?.ram_delta_mb ?? 0,
        is_system: true,
        is_active: true,
        sort_order: 310 + idx * 10,
        is_available: true,
        version: sat.version
      };
    });
    return [noneOption, ...mappedSatellites];
  }

  const fallbackList = catalogDbs
    .filter(item => {
      const engine = (item.service_engine || '').toLowerCase();
      return engine === 'none' || engine === 'sqlite' || engine === 'postgres';
    })
    .map((item): StudioDatabaseOption => ({
      ...item,
      is_available: true
    }));

  if (fallbackList.some(d => d.id === 'db-none' || d.service_engine === 'none')) {
    return fallbackList;
  }
  return [noneOption, ...fallbackList];
}

export function isLightRuntimeComponent(rt: TemplateComponent | null): boolean {
  if (!rt) return false;
  if (rt.ram_delta_mb > 0) return rt.ram_delta_mb <= 384;
  if ((rt.icon_key || '').toLowerCase() === 'cpp' || (rt.icon_key || '').toLowerCase() === 'c') return true;
  if ((rt.compatible_runtimes || []).some(c => c.toLowerCase() === 'cpp' || c.toLowerCase() === 'c')) return true;
  return false;
}

export function isHeavyRuntimeComponent(rt: TemplateComponent | null): boolean {
  if (!rt) return false;
  if (rt.ram_delta_mb > 0) return rt.ram_delta_mb >= 1024;
  if ((rt.icon_key || '').toLowerCase() === 'java') return true;
  if ((rt.compatible_runtimes || []).some(c => c.toLowerCase() === 'java' || c.toLowerCase() === 'dotnet')) return true;
  return false;
}

export function isResidentAddonComponent(addon: TemplateComponent): boolean {
  const sub = (addon.sub_category || '').toLowerCase();
  const slug = (addon.slug || '').toLowerCase();
  if (
    sub === 'devserver' ||
    sub === 'heavy' ||
    sub === 'data' ||
    slug.includes('datascience') ||
    slug.includes('data-science')
  ) {
    return true;
  }
  return (addon.ram_delta_mb || 0) >= 256;
}

export function resolveDefaultJudgeEntrypointForRuntime(rt: TemplateComponent | null): string {
  if (!rt) return '';
  const entry = (rt.judge_entrypoint || '').trim();
  if (entry) return entry;

  const baseImg = (rt.base_image || '').toLowerCase();
  const icon = (rt.icon_key || '').toLowerCase();
  const slug = (rt.slug || '').toLowerCase();
  const name = (rt.name || '').toLowerCase();

  const matches = (token: string): boolean =>
    baseImg.includes(token) || icon.includes(token) || slug.includes(token) || name.includes(token);

  if (matches('python')) return 'python3 solution.py';
  if (matches('cpp') || matches('gcc') || matches('clang') || matches('c++')) {
    return 'g++ -O2 solution.cpp -o solution && ./solution';
  }
  if (matches('java') || matches('temurin') || matches('openjdk')) {
    return 'javac Main.java && java Main';
  }
  if (matches('rust')) return 'rustc -O solution.rs -o solution && ./solution';
  if (matches('go') || matches('golang')) return 'go run solution.go';
  if (matches('node')) return 'node solution.js';
  if (matches('php')) return 'php solution.php';

  if (rt.tools_provided && rt.tools_provided.length > 0) {
    const tool = rt.tools_provided[0].trim();
    if (tool) return `${tool} solution`;
  }
  return '';
}

export function computeRecommendedRamMBForStack(params: {
  targetEnvironment: TargetEnvironment;
  rt: TemplateComponent | null;
  web: TemplateComponent | null;
  addons: TemplateComponent[];
  db: StudioDatabaseOption | null;
  caps: RuntimeCapabilities | null;
}): number {
  const { targetEnvironment, rt, web, addons, db, caps } = params;
  const isJudge = targetEnvironment === 'JUEZ_EFIMERO';

  if (isJudge) {
    if (!rt) return 256;
    const runtimeBaseMB = caps?.runtime_base_mb ?? 32;
    let processFootprintMB = 224;
    if (isHeavyRuntimeComponent(rt)) {
      processFootprintMB = 480;
    } else if (isLightRuntimeComponent(rt)) {
      processFootprintMB = 96;
    }
    for (const a of addons) {
      if (!isResidentAddonComponent(a)) continue;
      if ((a.sub_category || '').toLowerCase() === 'heavy' || a.ram_delta_mb >= 384) {
        processFootprintMB = Math.max(processFootprintMB, 480);
      } else {
        processFootprintMB = Math.max(processFootprintMB, 224);
      }
    }
    const totalJudge = runtimeBaseMB + processFootprintMB;
    return totalJudge <= 128 ? 128 : totalJudge <= 256 ? 256 : 512;
  }

  const editorBaseMB = caps?.editor_base_mb ?? 210;
  const satelliteBaseMB = db?.ram_delta_mb || 0;
  if (!rt && !web) {
    return satelliteBaseMB > 0 ? 768 : 512;
  }

  let activeProcessFootprintMB = 256;
  if (isLightRuntimeComponent(rt)) {
    activeProcessFootprintMB = 128;
  }
  if (isHeavyRuntimeComponent(rt)) {
    activeProcessFootprintMB = web || addons.length > 0 || satelliteBaseMB > 0 ? 1280 : 640;
  }
  if (web) {
    activeProcessFootprintMB = Math.max(activeProcessFootprintMB, 640);
  }

  for (const a of addons) {
    if (!isResidentAddonComponent(a)) continue;
    if ((a.sub_category || '').toLowerCase() === 'heavy' || a.ram_delta_mb >= 384) {
      activeProcessFootprintMB = Math.max(
        activeProcessFootprintMB,
        web || isHeavyRuntimeComponent(rt) ? 1280 : 512
      );
    } else {
      activeProcessFootprintMB = Math.max(activeProcessFootprintMB, 384);
    }
  }

  const rawTotal = editorBaseMB + satelliteBaseMB + activeProcessFootprintMB;
  if (rawTotal <= 384) return 384;
  if (rawTotal <= 512) return 512;
  if (rawTotal <= 768) return 768;
  if (rawTotal <= 1024) return 1024;
  if (rawTotal <= 1536) return 1536;
  if (rawTotal <= 2048) return 2048;
  return 4096;
}

export function computeMinimumFloorMBForStack(params: {
  targetEnvironment: TargetEnvironment;
  rt: TemplateComponent | null;
  web: TemplateComponent | null;
  addons: TemplateComponent[];
  db: StudioDatabaseOption | null;
  caps: RuntimeCapabilities | null;
}): number {
  const { targetEnvironment, rt, web, addons, db, caps } = params;
  const isJudge = targetEnvironment === 'JUEZ_EFIMERO';
  if (isJudge) {
    return isHeavyRuntimeComponent(rt) ? 128 : 64;
  }
  const editorBase = caps?.editor_base_mb ?? 210;
  const satelliteMB = db?.ram_delta_mb || 0;

  let minRuntimeProcessMB = 0;
  if (rt) {
    minRuntimeProcessMB = isHeavyRuntimeComponent(rt) ? 500 : isLightRuntimeComponent(rt) ? 128 : 174;
  }
  if (web || addons.some(a => isResidentAddonComponent(a))) {
    minRuntimeProcessMB = Math.max(minRuntimeProcessMB, 300);
  }
  return editorBase + satelliteMB + minRuntimeProcessMB;
}

export function filterAndPrioritizeRuntimeItems(
  items: TemplateComponent[],
  rawQuery: string,
  selectedId?: string
): TemplateComponent[] {
  const query = (rawQuery || '').trim().toLowerCase();
  const list = items.filter(item => {
    if (!query) return true;
    return (
      item.name.toLowerCase().includes(query) ||
      (item.description && item.description.toLowerCase().includes(query)) ||
      (item.slug && item.slug.toLowerCase().includes(query))
    );
  });
  if (!selectedId) return list;
  return [...list].sort((a, b) => (a.id === selectedId ? -1 : b.id === selectedId ? 1 : 0));
}

export function filterAndPrioritizeAddonItems(
  items: TemplateComponent[],
  rawQuery: string,
  selectedAddonIds: string[]
): TemplateComponent[] {
  const query = (rawQuery || '').trim().toLowerCase();
  const selSet = new Set(selectedAddonIds);
  const list = items.filter(item => {
    if (!query) return true;
    const nameMatch = item.name.toLowerCase().includes(query);
    const descMatch = !!item.description && item.description.toLowerCase().includes(query);
    const slugMatch = !!item.slug && item.slug.toLowerCase().includes(query);
    const pkgsMatch = (item.package_names || []).some(pkg => pkg.toLowerCase().includes(query));
    return nameMatch || descMatch || slugMatch || pkgsMatch;
  });
  return [...list].sort((a, b) => {
    const aSel = selSet.has(a.id) || selSet.has(a.slug);
    const bSel = selSet.has(b.id) || selSet.has(b.slug);
    return aSel && !bSel ? -1 : !aSel && bSel ? 1 : 0;
  });
}

export interface SynthesizedVisualStackResult {
  suggestedName: string;
  needsBuild: boolean;
  dockerImage: string;
  dockerfileContent: string;
  toolsDeclared: string;
  setupScript: string;
  entrypoint: string;
}

export function synthesizeVisualStackState(params: {
  targetEnvironment: TargetEnvironment;
  rt: TemplateComponent | null;
  web: TemplateComponent | null;
  addons: TemplateComponent[];
  db: StudioDatabaseOption | null;
}): SynthesizedVisualStackResult {
  const { targetEnvironment, rt, web, addons, db } = params;
  const isJudge = targetEnvironment === 'JUEZ_EFIMERO';
  const isDual = targetEnvironment === 'AMBOS';

  const parts: string[] = [];
  if (rt) parts.push(rt.name);
  if (web) parts.push(web.name);
  for (const a of addons) parts.push(a.name.split(' ')[0]);
  if (db) parts.push(db.name.split(' ')[0]);
  const prefix = isDual ? 'Entorno Dual: ' : isJudge ? 'Evaluador: ' : 'Laboratorio: ';
  const suggestedName = prefix + parts.join(' + ');

  const buildLayers: string[] = [];
  let baseImage = '';

  if (rt) {
    baseImage = rt.base_image || '';
    if (rt.build_layer_snippet) buildLayers.push(rt.build_layer_snippet);
    if (web && !isJudge && !isDual) {
      if (!baseImage.toLowerCase().includes('node')) {
        const nodeSourceImage =
          web.base_image && web.base_image.includes('node') ? web.base_image : 'node:20-bookworm-slim';
        if (baseImage.toLowerCase().includes('alpine')) {
          buildLayers.push('RUN apk add --no-cache nodejs npm');
        } else {
          buildLayers.push(`COPY --from=${nodeSourceImage} /usr/local /usr/local`);
        }
      }
      if (web.build_layer_snippet) buildLayers.push(web.build_layer_snippet);
    }
  } else if (web && !isJudge && !isDual) {
    baseImage = web.base_image || 'node:20-bookworm-slim';
    if (web.build_layer_snippet) buildLayers.push(web.build_layer_snippet);
  }

  for (const addon of addons) {
    if (addon.build_layer_snippet) buildLayers.push(addon.build_layer_snippet);
  }

  const needsBuild = buildLayers.length > 0;
  let dockerImage = baseImage;
  let dockerfileContent = baseImage ? `FROM ${baseImage}` : '';
  if (needsBuild) {
    const canonical = `FROM ${baseImage}\n${buildLayers.join('\n')}`;
    const hash = fnv1aHex(canonical);
    dockerImage = `solv-local/stack:${hash}`;
    dockerfileContent = `# Generado automáticamente por el Constructor de Entornos SOLV\nFROM ${baseImage}\n${buildLayers.join('\n')}`;
  }

  const toolsSet = new Set<string>();
  if (rt) {
    for (const t of rt.tools_provided || []) toolsSet.add(t);
  }
  if (web && !isJudge && !isDual) {
    for (const t of web.tools_provided || []) toolsSet.add(t);
  }
  for (const a of addons) {
    for (const t of a.tools_provided || []) toolsSet.add(t);
  }
  if (db && !isJudge && !isDual) {
    for (const t of db.tools_provided || []) toolsSet.add(t);
  }
  const toolsDeclared = Array.from(toolsSet).join(', ');

  let setupScript: string;
  let entrypoint: string;
  if (isDual) {
    const lines: string[] = [];
    if (rt?.setup_snippet) lines.push(rt.setup_snippet);
    for (const a of addons) {
      if (a.setup_snippet) lines.push(a.setup_snippet);
    }
    setupScript = lines.join('\n');
    entrypoint = resolveDefaultJudgeEntrypointForRuntime(rt);
  } else if (!isJudge) {
    const lines: string[] = [];
    if (rt?.setup_snippet) lines.push(rt.setup_snippet);
    if (web?.setup_snippet) lines.push(web.setup_snippet);
    for (const a of addons) {
      if (a.setup_snippet) lines.push(a.setup_snippet);
    }
    setupScript = lines.join('\n');
    entrypoint = '';
  } else {
    setupScript = '';
    entrypoint = resolveDefaultJudgeEntrypointForRuntime(rt);
  }

  return {
    suggestedName,
    needsBuild,
    dockerImage,
    dockerfileContent,
    toolsDeclared,
    setupScript,
    entrypoint
  };
}

export function downloadBrowserTextFile(filename: string, content: string, mimeType: string = 'text/plain'): void {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export function importBrowserTextFile(accept: string, callback: (text: string) => void): void {
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = accept;
  input.onchange = (e: Event) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = event => {
      const text = event.target?.result;
      if (typeof text === 'string') {
        callback(text);
      }
    };
    reader.readAsText(file);
  };
  input.click();
}

export const STUDIO_IDE_RAM_TIERS = [384, 512, 768, 1024, 1536, 2048] as const;
export const STUDIO_JUDGE_RAM_TIERS = [128, 256, 512, 1024] as const;

export function computeIntensiveRamMB(params: {
  isJudge: boolean;
  recommendedRamMB: number;
  maxAllowedRamMB: number;
}): number {
  const tiers = params.isJudge ? STUDIO_JUDGE_RAM_TIERS : STUDIO_IDE_RAM_TIERS;
  const nextTier = tiers.find(t => t > params.recommendedRamMB) ?? params.recommendedRamMB;
  return Math.min(nextTier, params.maxAllowedRamMB);
}

export type AddonAcademicCategory = 'framework' | 'orm' | 'toolkit';

export function classifyAddonComponent(addon: TemplateComponent): AddonAcademicCategory {
  const slug = (addon.slug || '').toLowerCase();
  const id = (addon.id || '').toLowerCase();

  // 1. Excepción explícita para Maven (herramienta de construcción académica)
  if (slug === 'spring-maven' || id === 'spring-maven') {
    return 'toolkit';
  }

  // 2. Prioridad de sub_category declarada
  const sub = (addon.sub_category || '').toLowerCase();
  if (sub === 'orm' || sub === 'connector') {
    return 'orm';
  }
  if (sub === 'framework' || sub === 'devserver') {
    return 'framework';
  }
  if (sub === 'toolkit' || sub === 'heavy') {
    return 'toolkit';
  }

  // 3. Heurística de respaldo por texto
  const name = (addon.name || '').toLowerCase();
  const packages = (addon.package_names || []).map(p => p.toLowerCase());
  const allText = `${slug} ${id} ${name} ${packages.join(' ')}`;

  if (
    allText.includes('sqlalchemy') ||
    allText.includes('prisma') ||
    allText.includes('typeorm') ||
    allText.includes('jdbc') ||
    allText.includes('hibernate') ||
    allText.includes('gorm') ||
    allText.includes('peewee') ||
    allText.includes('tortoise') ||
    allText.includes('sequelize') ||
    allText.includes('drizzle') ||
    allText.includes('sqlx') ||
    allText.includes('connector') ||
    allText.includes('orm') ||
    allText.includes('nosql')
  ) {
    return 'orm';
  }

  if (
    allText.includes('fastapi') ||
    allText.includes('django') ||
    allText.includes('flask') ||
    allText.includes('streamlit') ||
    allText.includes('express') ||
    allText.includes('nestjs') ||
    allText.includes('laravel') ||
    allText.includes('rails') ||
    allText.includes('gin') ||
    allText.includes('echo') ||
    allText.includes('fiber') ||
    allText.includes('springboot') ||
    allText.includes('spring-boot') ||
    allText.includes('aspnet') ||
    allText.includes('axum')
  ) {
    return 'framework';
  }

  return 'toolkit';
}

export function isFrameworkAddon(addon: TemplateComponent): boolean {
  return classifyAddonComponent(addon) === 'framework';
}

export function getComponentRoleBadge(comp: TemplateComponent | StudioDatabaseOption): string {
  if (comp.layer === 'database') {
    return comp.service_engine === 'none' ? 'Sin BD' : 'Servicio';
  }
  if (comp.layer === 'runtime') {
    return comp.sub_category === 'web' ? 'Web' : 'Lenguaje';
  }
  const cat = classifyAddonComponent(comp as TemplateComponent);
  if (cat === 'framework') return 'Framework';
  if (cat === 'orm') return 'ORM';
  return 'Herramienta';
}
