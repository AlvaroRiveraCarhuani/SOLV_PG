import { AdminTemplateItem, TemplateComponent } from '../services/admin-templates.service';
import { TechIconKey } from './components/tech-logo/tech-logo.component';
import { environmentBadge, isEnvironmentType } from '@shared/constants/environment-types.constants';

export interface TechMeta {
  id: string;
  name: string;
  badgeClass: string;
  iconType: TechIconKey;
}

export interface StackTechChip {
  id: string;
  name: string;
  iconKey: string;
  role: 'runtime' | 'web' | 'addon' | 'database';
}

/** Forma mínima que comparten el stack del administrador y el del docente. */
export interface ResolvableStackConfig {
  runtime_id?: string;
  web_platform_id?: string;
  addon_ids?: string[];
  database_id?: string;
  services?: { component_id?: string }[];
}

export interface TemplateTechInferenceInput {
  name?: string;
  docker_image?: string;
  description?: string;
  stack_config?: ResolvableStackConfig | null;
}

interface TechRule {
  id: string;
  name: string;
  iconType: TechIconKey;
  tokens?: string[];
  substrings?: string[];
  unless?: string[];
}

/** Compara por token completo: evita que `c` coincida dentro de "stack" o `go` dentro de "django". */
function matchesToken(text: string, token: string): boolean {
  if (!text || !token) return false;
  const escaped = token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return new RegExp(`(^|[^a-z0-9])${escaped}([^a-z0-9]|$)`, 'i').test(text);
}

function metaFor(key: TechIconKey, label: string): TechMeta {
  return { id: key, name: label, badgeClass: `tech-${key}`, iconType: key };
}

/**
 * Reglas únicas de inferencia para ambos catálogos. El orden decide el icono
 * principal y el orden de los chips; las reglas específicas van antes que las
 * genéricas para que "JavaScript" no caiga en Java ni "C/C++" en C.
 */
const TECH_RULES: TechRule[] = [
  {
    id: 'python',
    name: 'Python',
    iconType: 'python',
    tokens: ['python', 'django', 'flask', 'fastapi', 'pandas', 'numpy', 'streamlit', 'jupyter', 'pytorch'],
    substrings: ['data science', 'machine learning', 'inteligencia artificial']
  },
  { id: 'rust', name: 'Rust', iconType: 'rust', tokens: ['rust', 'cargo', 'actix', 'tokio'] },
  { id: 'go', name: 'Go', iconType: 'go', tokens: ['golang', 'go', 'gin', 'fiber'] },
  {
    id: 'java',
    name: 'Java',
    iconType: 'java',
    tokens: ['java', 'temurin', 'openjdk', 'kotlin', 'spring', 'maven', 'gradle']
  },
  { id: 'php', name: 'PHP', iconType: 'php', tokens: ['php', 'laravel'] },
  {
    id: 'dotnet',
    name: '.NET / C#',
    iconType: 'dotnet',
    tokens: ['dotnet', 'c#', 'csharp'],
    substrings: ['.net']
  },
  { id: 'cpp', name: 'C / C++', iconType: 'cpp', tokens: ['c++', 'cpp', 'g++', 'gcc', 'clang', 'cmake'] },
  { id: 'c', name: 'C', iconType: 'c', tokens: ['c'], unless: ['cpp'] },
  { id: 'angular', name: 'Angular', iconType: 'angular', tokens: ['angular'] },
  {
    id: 'node',
    name: 'Node.js',
    iconType: 'node',
    tokens: ['node', 'nodejs', 'javascript', 'js', 'express', 'nest', 'nestjs', 'bun', 'deno', 'ts-node']
  },
  { id: 'nextjs', name: 'Next.js', iconType: 'nextjs', tokens: ['next', 'nextjs', 'next.js'] },
  { id: 'react', name: 'React', iconType: 'react', tokens: ['react'], unless: ['nextjs'] },
  { id: 'vue', name: 'Vue', iconType: 'vue', tokens: ['vue', 'nuxt'] },
  { id: 'vite', name: 'Vite', iconType: 'vite', tokens: ['vite'] },
  { id: 'typescript', name: 'TypeScript', iconType: 'typescript', tokens: ['typescript', 'ts-node'] },
  { id: 'spring', name: 'Spring Boot', iconType: 'spring', tokens: ['spring'] },
  { id: 'maven', name: 'Maven', iconType: 'maven', tokens: ['maven'] },
  { id: 'gradle', name: 'Gradle', iconType: 'gradle', tokens: ['gradle'] },
  { id: 'fastapi', name: 'FastAPI', iconType: 'fastapi', tokens: ['fastapi'] },
  { id: 'django', name: 'Django', iconType: 'django', tokens: ['django'] },
  { id: 'flask', name: 'Flask', iconType: 'flask', tokens: ['flask'] },
  { id: 'pandas', name: 'Pandas', iconType: 'pandas', tokens: ['pandas'] },
  { id: 'numpy', name: 'NumPy', iconType: 'numpy', tokens: ['numpy'] },
  { id: 'streamlit', name: 'Streamlit', iconType: 'streamlit', tokens: ['streamlit'] },
  { id: 'pytorch', name: 'PyTorch', iconType: 'pytorch', tokens: ['pytorch'] },
  { id: 'jupyter', name: 'Jupyter', iconType: 'python', tokens: ['jupyter'] },
  {
    id: 'data-science',
    name: 'Data Science',
    iconType: 'pandas',
    substrings: ['data science', 'machine learning', 'inteligencia artificial'],
    unless: ['pandas', 'numpy']
  },
  { id: 'express', name: 'Express', iconType: 'express', tokens: ['express'] },
  { id: 'nestjs', name: 'NestJS', iconType: 'nestjs', tokens: ['nest', 'nestjs'] },
  { id: 'bun', name: 'Bun', iconType: 'bun', tokens: ['bun'] },
  { id: 'deno', name: 'Deno', iconType: 'deno', tokens: ['deno'] },
  { id: 'prisma', name: 'Prisma', iconType: 'prisma', tokens: ['prisma'] },
  { id: 'gcc', name: 'GCC / Clang', iconType: 'c', tokens: ['gcc', 'clang', 'g++', 'cmake'] },
  { id: 'webassembly', name: 'WebAssembly', iconType: 'rust', tokens: ['wasm', 'webassembly'] },
  { id: 'grpc', name: 'gRPC', iconType: 'go', tokens: ['grpc'] },
  { id: 'laravel', name: 'Laravel', iconType: 'laravel', tokens: ['laravel'] },
  { id: 'postgres', name: 'PostgreSQL', iconType: 'postgres', tokens: ['postgres', 'postgresql', 'psql'] },
  { id: 'mysql', name: 'MySQL', iconType: 'mysql', tokens: ['mysql'] },
  { id: 'mariadb', name: 'MariaDB', iconType: 'mariadb', tokens: ['mariadb'] },
  { id: 'mongodb', name: 'MongoDB', iconType: 'mongodb', tokens: ['mongodb', 'mongo'] },
  { id: 'redis', name: 'Redis', iconType: 'redis', tokens: ['redis'] },
  { id: 'sqlite', name: 'SQLite', iconType: 'sqlite', tokens: ['sqlite'] },
  {
    id: 'db',
    name: 'Bases de Datos',
    iconType: 'db',
    tokens: ['sql'],
    unless: ['postgres', 'mysql', 'mariadb', 'mongodb', 'redis', 'sqlite']
  }
];

function ruleMeta(rule: TechRule): TechMeta {
  return { id: rule.id, name: rule.name, badgeClass: `tech-${rule.id}`, iconType: rule.iconType };
}

function matchesRule(text: string, rule: TechRule): boolean {
  const tokenMatch = (rule.tokens || []).some(token => matchesToken(text, token));
  const substringMatch = (rule.substrings || []).some(fragment => text.includes(fragment));
  return tokenMatch || substringMatch;
}

function findRuleForReference(reference: string): TechRule | null {
  const value = reference.toLowerCase().trim();
  if (!value || value.includes('db-none') || value === 'none') return null;
  return (
    TECH_RULES.find(rule => {
      if (rule.id === value) return true;
      if (matchesToken(value, rule.id)) return true;
      return (rule.tokens || []).some(token => matchesToken(value, token));
    }) ?? null
  );
}

/**
 * Inferencia compartida por el catálogo del administrador y el del docente.
 * Primero respeta las piezas declaradas en `stack_config`; después completa con
 * el texto de la plantilla. Devuelve todas las tecnologías detectadas en orden
 * de lectura para que las tarjetas no se queden solo con el lenguaje principal.
 */
export function inferTemplateTechnologies(input: TemplateTechInferenceInput): TechMeta[] {
  const text = `${input.name || ''} ${input.docker_image || ''} ${input.description || ''}`.toLowerCase();
  const matched = new Map<string, TechMeta>();

  const add = (rule: TechRule): void => {
    if (matched.has(rule.id)) return;
    if (rule.unless?.some(id => matched.has(id))) return;
    matched.set(rule.id, ruleMeta(rule));
  };

  const stack = input.stack_config;
  if (stack) {
    const references = [
      stack.runtime_id,
      stack.web_platform_id,
      ...(stack.addon_ids || []),
      stack.database_id,
      ...(stack.services || []).map(service => service.component_id)
    ];
    for (const reference of references) {
      if (!reference) continue;
      const rule = findRuleForReference(reference);
      if (rule) add(rule);
    }
  }

  for (const rule of TECH_RULES) {
    if (matchesRule(text, rule)) add(rule);
  }

  // El stack puede declarar una pieza genérica y el texto una más específica.
  if (['postgres', 'mysql', 'mariadb', 'mongodb', 'redis', 'sqlite'].some(id => matched.has(id))) {
    matched.delete('db');
  }
  if (matched.has('nextjs')) matched.delete('react');
  if (matched.has('pandas') || matched.has('numpy')) matched.delete('data-science');

  return Array.from(matched.values());
}

export function detectTemplateTech(
  item: TemplateTechInferenceInput
): TechMeta {
  return inferTemplateTechnologies(item)[0] ?? metaFor('docker', 'Docker Base');
}

/**
 * Resuelve las piezas declaradas en `stack_config` contra el catálogo real de
 * tecnologías, conservando el orden lenguaje → web → complementos → datos.
 * Reutiliza nombre e icono del catálogo en lugar de inferirlos del tag Docker.
 */
export function resolveTemplateStack(
  item: { stack_config?: ResolvableStackConfig | null },
  catalog: TemplateComponent[]
): StackTechChip[] {
  const config = item.stack_config;
  if (!config || catalog.length === 0) return [];

  const byId = new Map(catalog.map(component => [component.id, component]));
  const bySlug = new Map(catalog.map(component => [component.slug, component]));
  const seen = new Set<string>();
  const chips: StackTechChip[] = [];

  const push = (reference?: string): void => {
    if (!reference) return;
    const component = byId.get(reference) || bySlug.get(reference);
    if (!component || seen.has(component.id)) return;
    // "Sin base de datos" es una decisión, no una tecnología del entorno.
    if (component.layer === 'database' && (component.service_engine === 'none' || component.slug === 'db-none')) {
      return;
    }
    seen.add(component.id);
    const role: StackTechChip['role'] =
      component.layer === 'runtime'
        ? component.sub_category === 'web'
          ? 'web'
          : 'runtime'
        : component.layer;
    chips.push({
      id: component.id,
      name: component.name,
      iconKey: component.icon_key || 'docker',
      role
    });
  };

  push(config.runtime_id);
  push(config.web_platform_id);
  for (const addonId of config.addon_ids || []) push(addonId);
  push(config.database_id);
  for (const service of config.services || []) push(service.component_id);

  return chips;
}

export function detectTech(name: string, image: string): TechMeta {
  return detectTemplateTech({ name, docker_image: image });
}

export function isPausedTemplateStatus(status?: string): boolean {
  return status === 'paused' || status === 'suspended' || status === 'SUSPENDIDA';
}

/**
 * Presentación de la modalidad de una plantilla. Un valor desconocido devuelve
 * el estado neutro del catálogo, nunca Desarrollo ni Evaluación.
 */
export function getTemplatePurposeBadge(targetEnv?: string): {
  label: string;
  shortLabel: string;
  variant: 'ide' | 'judge' | 'dual' | 'unknown';
} {
  const badge = environmentBadge(targetEnv);
  return { label: badge.label, shortLabel: badge.short.toUpperCase(), variant: badge.variant };
}

export function formatByteSize(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B';
  const k = 1024;
  const dm = 1;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
}

export function filterAdminTemplatesForView(params: {
  templates: AdminTemplateItem[];
  statusPredicate: (t: AdminTemplateItem) => boolean;
  purpose: 'ALL' | 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO' | 'AMBOS';
  tech: string;
  searchTerm: string;
  includeRequestedBy?: boolean;
  includeRejectionReason?: boolean;
}): AdminTemplateItem[] {
  const {
    templates,
    statusPredicate,
    purpose,
    tech,
    searchTerm,
    includeRequestedBy = false,
    includeRejectionReason = false
  } = params;
  const term = (searchTerm || '').toLowerCase().trim();

  return templates.filter(t => {
    if (!statusPredicate(t)) return false;
    if (purpose !== 'ALL') {
      const env = t.target_environment;
      if (purpose === 'AMBOS' && !isEnvironmentType(env, 'AMBOS')) return false;
      // Una plantilla dual sirve a las dos modalidades y aparece en ambos filtros.
      if (
        purpose === 'IDE_PERSISTENTE' &&
        !isEnvironmentType(env, 'IDE_PERSISTENTE') &&
        !isEnvironmentType(env, 'AMBOS')
      ) {
        return false;
      }
      if (
        purpose === 'JUEZ_EFIMERO' &&
        !isEnvironmentType(env, 'JUEZ_EFIMERO') &&
        !isEnvironmentType(env, 'AMBOS')
      ) {
        return false;
      }
    }
    if (tech !== 'all') {
      const itemTech = detectTemplateTech(t);
      if (itemTech.id !== tech) return false;
    }
    if (!term) return true;
    return (
      t.name.toLowerCase().includes(term) ||
      t.docker_image.toLowerCase().includes(term) ||
      (includeRequestedBy && !!t.requested_by_name && t.requested_by_name.toLowerCase().includes(term)) ||
      (includeRejectionReason && !!t.rejection_reason && t.rejection_reason.toLowerCase().includes(term)) ||
      (!!t.description && t.description.toLowerCase().includes(term))
    );
  });
}
