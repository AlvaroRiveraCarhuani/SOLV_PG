import { AdminTemplateItem } from '../services/admin-templates.service';
import { CURATED_TECH_ICONS, TechIconKey } from './components/tech-logo/tech-logo.component';

export interface TechMeta {
  id: string;
  name: string;
  badgeClass: string;
  iconType: TechIconKey;
}

export function detectTemplateTech(
  item: Pick<AdminTemplateItem, 'name' | 'docker_image' | 'stack_config'>
): TechMeta {
  const runtimeId = (item.stack_config?.runtime_id || '').toLowerCase().trim();
  const webId = (item.stack_config?.web_platform_id || '').toLowerCase().trim();

  const matchCuratedId = (id: string): TechMeta | null => {
    if (!id) return null;
    const direct = CURATED_TECH_ICONS.find(c => c.key.toLowerCase() === id && c.key !== 'docker');
    if (direct) {
      return { id: direct.key, name: direct.label, badgeClass: `tech-${direct.key}`, iconType: direct.key as TechIconKey };
    }
    for (const c of CURATED_TECH_ICONS) {
      if (c.key === 'docker') continue;
      const key = c.key.toLowerCase();
      const re = new RegExp(`(^|[^a-z0-9])${key}([^a-z0-9]|$)`, 'i');
      if (re.test(id)) {
        return { id: c.key, name: c.label, badgeClass: `tech-${c.key}`, iconType: c.key as TechIconKey };
      }
    }
    return null;
  };

  if (runtimeId) {
    const matched = matchCuratedId(runtimeId);
    if (matched) return matched;
  }

  if (webId) {
    const matched = matchCuratedId(webId);
    if (matched) return matched;
  }

  const name = item.name || '';
  const image = item.docker_image || '';
  const text = `${name} ${image}`.toLowerCase();

  if (
    text.includes('python') ||
    text.includes('django') ||
    text.includes('flask') ||
    text.includes('fastapi') ||
    text.includes('pandas') ||
    text.includes('numpy') ||
    text.includes('streamlit') ||
    text.includes(' ds')
  ) {
    return { id: 'python', name: 'Python', badgeClass: 'tech-python', iconType: 'python' };
  }
  if (text.includes('rust') || text.includes('cargo') || text.includes('actix') || text.includes('tokio')) {
    return { id: 'rust', name: 'Rust', badgeClass: 'tech-rust', iconType: 'rust' };
  }
  if (text.includes('typescript') || text.includes('ts-node')) {
    return { id: 'typescript', name: 'TypeScript', badgeClass: 'tech-typescript', iconType: 'typescript' };
  }
  if (text.includes('angular')) {
    return { id: 'angular', name: 'Angular', badgeClass: 'tech-angular', iconType: 'angular' };
  }
  if (text.includes('react') || text.includes('next')) {
    return { id: 'react', name: 'React', badgeClass: 'tech-react', iconType: 'react' };
  }
  if (text.includes('vue') || text.includes('vite')) {
    return { id: 'vue', name: 'Vue', badgeClass: 'tech-vue', iconType: 'vue' };
  }
  if (
    text.includes('node') ||
    text.includes('javascript') ||
    text.includes('express') ||
    text.includes('nest') ||
    text.includes('bun') ||
    text.includes('deno')
  ) {
    return { id: 'node', name: 'Node.js', badgeClass: 'tech-node', iconType: 'node' };
  }
  if (text.includes('php') || text.includes('laravel')) {
    return { id: 'php', name: 'PHP', badgeClass: 'tech-php', iconType: 'php' };
  }
  if (text.includes('dotnet') || text.includes('c#') || text.includes('.net') || text.includes('csharp')) {
    return { id: 'dotnet', name: '.NET / C#', badgeClass: 'tech-dotnet', iconType: 'dotnet' };
  }
  if (
    text.includes('golang') ||
    text.includes('go:') ||
    text.includes('go-') ||
    text.includes('/go') ||
    text.includes('gin') ||
    text.includes('fiber') ||
    /\bgo\b/.test(text)
  ) {
    return { id: 'go', name: 'Go', badgeClass: 'tech-go', iconType: 'go' };
  }
  if (
    text.includes('java') ||
    text.includes('spring') ||
    text.includes('kotlin') ||
    text.includes('maven') ||
    text.includes('gradle')
  ) {
    return { id: 'java', name: 'Java', badgeClass: 'tech-java', iconType: 'java' };
  }
  if (
    text.includes('c++') ||
    text.includes('cpp') ||
    text.includes('clang') ||
    text.includes('gcc') ||
    text.includes('g++') ||
    text.includes('cmake')
  ) {
    return { id: 'cpp', name: 'C / C++', badgeClass: 'tech-cpp', iconType: 'cpp' };
  }
  if (
    text.includes('postgres') ||
    text.includes('mysql') ||
    text.includes('sql') ||
    text.includes('redis') ||
    text.includes('mongo') ||
    text.includes('mariadb')
  ) {
    return { id: 'db', name: 'Bases de Datos', badgeClass: 'tech-db', iconType: 'db' };
  }

  for (const curated of CURATED_TECH_ICONS) {
    if (curated.key === 'docker' || curated.key === 'go') continue;
    if (text.includes(curated.key)) {
      return {
        id: curated.key,
        name: curated.label,
        badgeClass: `tech-${curated.key}`,
        iconType: curated.key as TechIconKey
      };
    }
  }

  return { id: 'docker', name: 'Docker Base', badgeClass: 'tech-docker', iconType: 'docker' };
}

export function detectTech(name: string, image: string): TechMeta {
  return detectTemplateTech({ name, docker_image: image });
}

export function isPausedTemplateStatus(status?: string): boolean {
  return status === 'paused' || status === 'suspended' || status === 'SUSPENDIDA';
}

export function getTemplatePurposeBadge(targetEnv?: string): {
  label: string;
  shortLabel: string;
  variant: 'ide' | 'judge' | 'dual';
} {
  if (targetEnv === 'AMBOS') {
    return { label: 'Dual: Lab + Juez', shortLabel: 'DUAL', variant: 'dual' };
  }
  if (targetEnv === 'JUEZ_EFIMERO' || targetEnv === 'JUEZ_VIRTUAL') {
    return { label: 'Evaluador Juez', shortLabel: 'JUEZ', variant: 'judge' };
  }
  return { label: 'Laboratorio IDE', shortLabel: 'LAB IDE', variant: 'ide' };
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
      const env = t.target_environment || 'IDE_PERSISTENTE';
      if (purpose === 'IDE_PERSISTENTE' && env !== 'IDE_PERSISTENTE') return false;
      if (purpose === 'JUEZ_EFIMERO' && env !== 'JUEZ_EFIMERO' && env !== 'JUEZ_VIRTUAL') return false;
      if (purpose === 'AMBOS' && env !== 'AMBOS') return false;
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
