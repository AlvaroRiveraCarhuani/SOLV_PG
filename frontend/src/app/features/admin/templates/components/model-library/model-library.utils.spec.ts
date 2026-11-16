import { TemplateComponent } from '@features/admin/models/admin-templates.models';
import {
  computeComponentMemoryBadge,
  filterAndSortTechComponentsForLibrary,
  isManageableTechComponent
} from './model-library.utils';

function makeComponent(overrides: Partial<TemplateComponent>): TemplateComponent {
  return {
    id: 'comp-1',
    slug: 'comp-slug',
    layer: 'addon',
    sub_category: 'toolkit',
    name: 'Componente Test',
    description: '',
    icon_key: 'docker',
    base_image: '',
    compatible_runtimes: [],
    install_mode: 'none',
    package_names: [],
    build_layer_snippet: '',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: [],
    service_engine: '',
    ram_delta_mb: 0,
    is_system: false,
    is_active: true,
    sort_order: 0,
    ...overrides
  } as TemplateComponent;
}

describe('model-library.utils', () => {
  describe('isManageableTechComponent', () => {
    it('debe rechazar componentes nulos o indefinidos', () => {
      expect(isManageableTechComponent(null)).toBe(false);
      expect(isManageableTechComponent(undefined)).toBe(false);
    });

    it('debe rechazar la pseudo-tecnología db-none', () => {
      const comp = makeComponent({
        slug: 'db-none',
        layer: 'database',
        service_engine: 'none',
        sub_category: 'none'
      });
      expect(isManageableTechComponent(comp)).toBe(false);
    });

    it('debe rechazar cualquier base de datos con service_engine none', () => {
      const comp = makeComponent({
        slug: 'custom-no-db',
        layer: 'database',
        service_engine: 'none'
      });
      expect(isManageableTechComponent(comp)).toBe(false);
    });

    it('debe aceptar tecnologías reales de base de datos, runtimes y complementos', () => {
      const sqlite = makeComponent({ slug: 'sqlite-3', layer: 'database', service_engine: 'sqlite' });
      const pg = makeComponent({ slug: 'postgres-16', layer: 'database', service_engine: 'postgres' });
      const py = makeComponent({ slug: 'python-312', layer: 'runtime', sub_category: 'backend' });
      const fastapi = makeComponent({ slug: 'fastapi', layer: 'addon', sub_category: 'framework' });

      expect(isManageableTechComponent(sqlite)).toBe(true);
      expect(isManageableTechComponent(pg)).toBe(true);
      expect(isManageableTechComponent(py)).toBe(true);
      expect(isManageableTechComponent(fastapi)).toBe(true);
    });
  });

  describe('computeComponentMemoryBadge', () => {
    it('calcula correctamente memoria 0 MB para SQLite en archivo sin inventar 256 MB', () => {
      const sqlite = makeComponent({
        slug: 'sqlite-3',
        layer: 'database',
        service_engine: 'sqlite',
        ram_delta_mb: 0
      });
      const badge = computeComponentMemoryBadge(sqlite);
      expect(badge.label).toBe('0 MB');
      expect(badge.detail).toBe('En archivo (disco)');
      expect(badge.variant).toBe('light');
    });

    it('calcula correctamente memoria 0 MB para db-none si se evalúa aisladamente', () => {
      const dbNone = makeComponent({
        slug: 'db-none',
        layer: 'database',
        service_engine: 'none',
        ram_delta_mb: 0
      });
      const badge = computeComponentMemoryBadge(dbNone);
      expect(badge.label).toBe('0 MB');
      expect(badge.detail).toBe('Sin servicio');
      expect(badge.variant).toBe('light');
    });

    it('calcula memoria residente para bases de datos satélite con RAM asignada', () => {
      const pg = makeComponent({
        slug: 'postgres-16',
        layer: 'database',
        service_engine: 'postgres',
        ram_delta_mb: 384
      });
      const badgePg = computeComponentMemoryBadge(pg);
      expect(badgePg.label).toBe('384 MB');
      expect(badgePg.detail).toBe('Satélite residente');
      expect(badgePg.variant).toBe('standard');

      const heavyDb = makeComponent({
        slug: 'big-db',
        layer: 'database',
        service_engine: 'mysql',
        ram_delta_mb: 512
      });
      const badgeHeavy = computeComponentMemoryBadge(heavyDb);
      expect(badgeHeavy.label).toBe('512 MB');
      expect(badgeHeavy.detail).toBe('Satélite residente');
      expect(badgeHeavy.variant).toBe('heavy');
    });

    it('calcula memoria para complementos con y sin proceso residente', () => {
      const addonDisco = makeComponent({ layer: 'addon', ram_delta_mb: 0 });
      expect(computeComponentMemoryBadge(addonDisco)).toEqual({
        label: '0 MB RAM',
        detail: 'En disco',
        variant: 'light'
      });

      const addonProceso = makeComponent({ layer: 'addon', ram_delta_mb: 128 });
      expect(computeComponentMemoryBadge(addonProceso)).toEqual({
        label: '+128 MB',
        detail: 'Proceso activo',
        variant: 'standard'
      });
    });

    it('calcula memoria para runtimes según su huella de ejecución', () => {
      const cpp = makeComponent({ layer: 'runtime', icon_key: 'cpp', ram_delta_mb: 0 });
      expect(computeComponentMemoryBadge(cpp).label).toBe('128 MB');

      const java = makeComponent({ layer: 'runtime', icon_key: 'java', ram_delta_mb: 0 });
      expect(computeComponentMemoryBadge(java).label).toBe('1024 MB');

      const py = makeComponent({ layer: 'runtime', icon_key: 'python', ram_delta_mb: 512 });
      expect(computeComponentMemoryBadge(py).label).toBe('512 MB');
    });
  });

  describe('filterAndSortTechComponentsForLibrary', () => {
    it('excluye db-none de la lista filtrada de tecnologías', () => {
      const items: TemplateComponent[] = [
        makeComponent({ id: '1', slug: 'python-312', layer: 'runtime', name: 'Python' }),
        makeComponent({ id: '2', slug: 'db-none', layer: 'database', service_engine: 'none', name: 'Sin Base de Datos' }),
        makeComponent({ id: '3', slug: 'sqlite-3', layer: 'database', service_engine: 'sqlite', name: 'SQLite 3' })
      ];

      const result = filterAndSortTechComponentsForLibrary({
        components: items,
        searchQuery: '',
        profileFilter: 'ALL',
        statusFilter: 'ALL'
      });

      expect(result.map(r => r.slug)).toEqual(['python-312', 'sqlite-3']);
      expect(result.some(r => r.slug === 'db-none')).toBe(false);
    });
  });
});
