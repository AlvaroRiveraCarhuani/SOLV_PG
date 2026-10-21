import {
  TemplateComponent,
  TemplateModelItem
} from '@features/admin/models/admin-templates.models';

export function filterModelsForLibrary(params: {
  models: TemplateModelItem[];
  searchQuery: string;
  filterPurpose: string;
  filterCategory: string;
  filterStatus: 'ALL' | 'ACTIVE' | 'INACTIVE';
}): TemplateModelItem[] {
  const { models, searchQuery, filterPurpose, filterCategory, filterStatus } = params;
  const query = (searchQuery || '').trim().toLowerCase();

  return models.filter(m => {
    if (query) {
      const matchesName = (m.name || m.title || '').toLowerCase().includes(query);
      const matchesDesc = (m.description || '').toLowerCase().includes(query);
      const matchesImage = (m.docker_image || '').toLowerCase().includes(query);
      if (!matchesName && !matchesDesc && !matchesImage) return false;
    }

    if (filterPurpose !== 'ALL') {
      if (filterPurpose === 'JUEZ_VIRTUAL') {
        if (
          m.target_environment !== 'JUEZ_VIRTUAL' &&
          m.target_environment !== 'JUEZ_EFIMERO' &&
          m.target_environment !== 'AMBOS'
        ) {
          return false;
        }
      } else if (filterPurpose === 'IDE_PERSISTENTE') {
        if (m.target_environment !== 'IDE_PERSISTENTE' && m.target_environment !== 'AMBOS') {
          return false;
        }
      } else if (m.target_environment !== filterPurpose) {
        return false;
      }
    }

    if (filterCategory !== 'ALL' && m.category_id !== filterCategory) {
      return false;
    }

    if (filterStatus === 'ACTIVE' && m.is_active === false) return false;
    if (filterStatus === 'INACTIVE' && m.is_active !== false) return false;

    return true;
  });
}

export function filterAndSortTechComponentsForLibrary(params: {
  components: TemplateComponent[];
  searchQuery: string;
  profileFilter: 'ALL' | 'BACKEND' | 'WEB' | 'ADDON' | 'DATABASE';
  statusFilter: 'ALL' | 'ACTIVE' | 'INACTIVE';
}): TemplateComponent[] {
  const { components, searchQuery, profileFilter, statusFilter } = params;
  const query = (searchQuery || '').trim().toLowerCase();

  return components
    .filter(c => {
      if (query) {
        const matchesName = (c.name || '').toLowerCase().includes(query);
        const matchesSlug = (c.slug || '').toLowerCase().includes(query);
        const matchesDesc = (c.description || '').toLowerCase().includes(query);
        const matchesImage = (c.base_image || '').toLowerCase().includes(query);
        const matchesTools = (c.tools_provided || []).some(t => t.toLowerCase().includes(query));
        const matchesPkgs = (c.package_names || []).some(p => p.toLowerCase().includes(query));
        if (!matchesName && !matchesSlug && !matchesDesc && !matchesImage && !matchesTools && !matchesPkgs) {
          return false;
        }
      }

      if (profileFilter !== 'ALL') {
        if (profileFilter === 'BACKEND' && (c.layer !== 'runtime' || c.sub_category === 'web')) return false;
        if (profileFilter === 'WEB' && (c.layer !== 'runtime' || c.sub_category !== 'web')) return false;
        if (profileFilter === 'ADDON' && c.layer !== 'addon') return false;
        if (profileFilter === 'DATABASE' && c.layer !== 'database') return false;
      }

      if (statusFilter === 'ACTIVE' && c.is_active === false) return false;
      if (statusFilter === 'INACTIVE' && c.is_active !== false) return false;

      return true;
    })
    .sort((a, b) => {
      const aActive = a.is_active !== false ? 1 : 0;
      const bActive = b.is_active !== false ? 1 : 0;
      if (aActive !== bActive) return bActive - aActive;

      const profileOrder = (item: TemplateComponent): number => {
        if (item.layer === 'runtime') return item.sub_category === 'web' ? 2 : 1;
        if (item.layer === 'addon') return 3;
        return 4;
      };
      const orderA = profileOrder(a);
      const orderB = profileOrder(b);
      if (orderA !== orderB) return orderA - orderB;

      const soA = a.sort_order ?? 0;
      const soB = b.sort_order ?? 0;
      if (soA !== soB) return soA - soB;

      return (a.name || '').localeCompare(b.name || '');
    });
}

export function getComponentProfileLabel(comp: TemplateComponent): string {
  if (comp.layer === 'runtime') {
    return comp.sub_category === 'web' ? 'Interfaz Web' : 'Lenguaje Principal';
  }
  if (comp.layer === 'addon') {
    return 'Complemento / Librería';
  }
  return 'Base de Datos';
}

export function getAddonEcosystemLabel(comp: TemplateComponent): string {
  if (comp.layer !== 'addon') return '';
  const cr = (comp.compatible_runtimes || []).map(r => r.trim().toLowerCase()).filter(Boolean);
  if (!cr.length || cr.includes('universal') || cr.includes('any')) {
    return 'Universal';
  }
  const tokenSet = new Set(cr);
  const labels: string[] = [];
  if (tokenSet.has('c') || tokenSet.has('cpp') || tokenSet.has('c-gcc13') || tokenSet.has('cpp-gcc13')) {
    labels.push('C / C++');
  }
  if (tokenSet.has('python') || tokenSet.has('python-312')) {
    labels.push('Python');
  }
  if (tokenSet.has('node') || tokenSet.has('web') || tokenSet.has('node-20')) {
    labels.push('Node.js / Web');
  }
  if (tokenSet.has('java') || tokenSet.has('java-21')) {
    labels.push('Java');
  }
  if (tokenSet.has('go') || tokenSet.has('go-122')) {
    labels.push('Go');
  }
  if (tokenSet.has('rust') || tokenSet.has('rust-stable')) {
    labels.push('Rust');
  }
  if (tokenSet.has('php') || tokenSet.has('php-83')) {
    labels.push('PHP');
  }
  if (tokenSet.has('dotnet') || tokenSet.has('dotnet-8')) {
    labels.push('.NET / C#');
  }
  if (labels.length > 0) {
    return labels.join(' · ');
  }
  return cr.map(r => r.toUpperCase()).join(', ');
}

export function matchesRuntimeEcosystem(rt: TemplateComponent, targetToken: string): boolean {
  const low = targetToken.trim().toLowerCase();
  if (!low || low === 'universal' || low === 'any') return false;
  const slug = (rt.slug || '').toLowerCase();
  const icon = (rt.icon_key || '').toLowerCase();
  const name = (rt.name || '').toLowerCase();
  return (
    slug === low ||
    (icon !== 'docker' && icon === low) ||
    name === low ||
    slug.startsWith(low + '-') ||
    name.startsWith(low + ' ')
  );
}

export function isOrphanAddonInCatalog(comp: TemplateComponent, allComponents: TemplateComponent[]): boolean {
  if (comp.layer !== 'addon') return false;
  const cr = comp.compatible_runtimes || [];
  if (!cr.length || cr.includes('universal') || cr.includes('any')) {
    return false;
  }
  const activeRuntimes = allComponents.filter(t => t.layer === 'runtime' && t.is_active !== false);
  return !cr.some(target => activeRuntimes.some(rt => matchesRuntimeEcosystem(rt, target)));
}

export function getCompatibleAddonsCountForRuntime(
  runtimeComp: TemplateComponent,
  allComponents: TemplateComponent[]
): number {
  if (runtimeComp.layer !== 'runtime') return 0;
  return allComponents.filter(
    c =>
      c.layer === 'addon' &&
      c.is_active !== false &&
      (c.compatible_runtimes || []).some(target => matchesRuntimeEcosystem(runtimeComp, target))
  ).length;
}

export function computeComponentMemoryBadge(
  comp: TemplateComponent
): { label: string; detail: string; variant: 'light' | 'standard' | 'heavy' } {
  if (comp.layer === 'addon') {
    if (comp.ram_delta_mb > 0) {
      return { label: `+${comp.ram_delta_mb} MB`, detail: 'Proceso activo', variant: 'standard' };
    }
    return { label: '0 MB RAM', detail: 'En disco', variant: 'light' };
  }

  if (comp.layer === 'runtime') {
    const ram = comp.ram_delta_mb || 0;
    const isCpp = (comp.icon_key || '').toLowerCase() === 'cpp';
    const isJava = (comp.icon_key || '').toLowerCase() === 'java';

    if ((ram > 0 && ram <= 384) || (ram === 0 && isCpp)) {
      return { label: `${ram > 0 ? ram : 128} MB`, detail: 'Perfil liviano', variant: 'light' };
    }
    if (ram >= 1024 || (ram === 0 && isJava)) {
      return { label: `${ram >= 1024 ? ram : 1024} MB`, detail: 'Perfil intensivo', variant: 'heavy' };
    }
    return { label: `${ram > 0 ? ram : 256} MB`, detail: 'Perfil estándar', variant: 'standard' };
  }

  const ram = comp.ram_delta_mb || 256;
  return {
    label: `${ram} MB`,
    detail: 'Base de datos',
    variant: ram >= 512 ? 'heavy' : 'standard'
  };
}
