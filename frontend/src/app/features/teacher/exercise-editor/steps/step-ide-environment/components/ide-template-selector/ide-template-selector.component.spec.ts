import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { IdeTemplateSelectorComponent } from './ide-template-selector.component';
import { ApprovedTemplate } from '../../step-ide-environment.component';
import { AdminTemplatesService, TemplateComponent } from '@features/admin/services/admin-templates.service';

function buildMockComponent(data: {
  id: string;
  slug: string;
  name: string;
  layer: TemplateComponent['layer'];
  sub_category?: string;
  icon_key: string;
}): TemplateComponent {
  return {
    id: data.id,
    slug: data.slug,
    name: data.name,
    layer: data.layer,
    sub_category: data.sub_category || '',
    description: '',
    icon_key: data.icon_key,
    base_image: '',
    compatible_runtimes: [],
    install_mode: 'none',
    package_names: [],
    build_layer_snippet: '',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: [],
    service_engine: data.layer === 'database' ? 'postgres' : '',
    ram_delta_mb: 0,
    is_system: true,
    is_active: true,
    sort_order: 1
  };
}

describe('IdeTemplateSelectorComponent', () => {
  let component: IdeTemplateSelectorComponent;
  let fixture: ComponentFixture<IdeTemplateSelectorComponent>;

  const mockTemplatesService = {
    getTechCatalog: vi.fn().mockReturnValue(of([]))
  };

  beforeEach(async () => {
    mockTemplatesService.getTechCatalog.mockReturnValue(of([]));

    await TestBed.configureTestingModule({
      imports: [IdeTemplateSelectorComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: mockTemplatesService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(IdeTemplateSelectorComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('muestra el stack de tecnologías y la base de datos como chips uniformes', () => {
    const tpl: ApprovedTemplate = {
      id: '8ec592bf-4856-4059-b03f-e3b15efca849',
      name: 'Entorno Full-Stack JS',
      description: 'Node.js, Next.js y PostgreSQL para aplicaciones web',
      docker_image: 'solv-local/stack:660620ce5745',
      base_ram_mb: 256,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved',
      services_config: {
        services: [
          { category: 'database', engine: 'postgres', role: 'primary_database' }
        ]
      }
    };

    const badges = component.getTechBadges(tpl);
    expect(badges.some(b => b.label === 'Node.js' && b.icon === 'node')).toBe(true);
    expect(badges.some(b => b.label === 'Next.js' && b.icon === 'nextjs')).toBe(true);
    expect(badges.some(b => b.label === 'PostgreSQL' && b.icon === 'postgres')).toBe(true);

    expect(component.hasDatabaseService(tpl)).toBe(true);
    expect(component.databaseEngine(tpl)).toBe('PostgreSQL');
  });

  it('resuelve el stack completo con addons (Prisma) y base de datos desde el catálogo de componentes', () => {
    component.techCatalog.set([
      buildMockComponent({ id: 'cat-node', slug: 'node-20', name: 'Node.js 20 LTS', layer: 'runtime', icon_key: 'node' }),
      buildMockComponent({ id: 'cat-next', slug: 'nextjs', name: 'Next.js (React Full-Stack)', layer: 'runtime', sub_category: 'web', icon_key: 'nextjs' }),
      buildMockComponent({ id: 'cat-prisma', slug: 'prisma', name: 'Prisma ORM + Clientes SQL', layer: 'addon', icon_key: 'prisma' }),
      buildMockComponent({ id: 'cat-pg', slug: 'postgres-18', name: 'PostgreSQL 18 (Aula)', layer: 'database', icon_key: 'postgres' }),
    ]);

    const tplWithStack: ApprovedTemplate = {
      id: '8ec592bf-4856-4059-b03f-e3b15efca849',
      name: 'Entorno Full-Stack JS',
      description: 'Node.js, Next.js y PostgreSQL para aplicaciones web',
      docker_image: 'solv-local/stack:660620ce5745',
      base_ram_mb: 256,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved',
      stack_config: {
        runtime_id: 'cat-node',
        web_platform_id: 'cat-next',
        addon_ids: ['cat-prisma'],
        database_id: 'cat-pg'
      }
    };

    const badges = component.getTechBadges(tplWithStack);
    expect(badges.length).toBe(4);
    expect(badges[0]).toEqual({ label: 'Node.js 20 LTS', icon: 'node' });
    expect(badges[1]).toEqual({ label: 'Next.js (React Full-Stack)', icon: 'nextjs' });
    expect(badges[2]).toEqual({ label: 'Prisma ORM + Clientes SQL', icon: 'prisma' });
    expect(badges[3]).toEqual({ label: 'PostgreSQL 18 (Aula)', icon: 'postgres' });
  });

  it('muestra todos los componentes del stack en entornos de Data Science', () => {
    const tpl: ApprovedTemplate = {
      id: 'b0000000-0000-0000-0000-000000000001',
      name: 'Python 3.12 Data Science & IA',
      description: 'Python con Pandas y NumPy para ciencia de datos',
      docker_image: 'gitpod/openvscode-server:1.96.0',
      base_ram_mb: 512,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved'
    };

    const badges = component.getTechBadges(tpl);
    expect(badges.some(b => b.label === 'Python' && b.icon === 'python')).toBe(true);
    expect(badges.some(b => b.label === 'Pandas')).toBe(true);
    expect(badges.some(b => b.label === 'NumPy')).toBe(true);
  });

  it('muestra el chip de runtime para entornos monomotores básicos', () => {
    const tpl: ApprovedTemplate = {
      id: 'b0000000-0000-0000-0000-000000000007',
      name: 'PHP 7.4 Legacy Monolith',
      description: 'Entorno base PHP',
      docker_image: 'php:7.4-apache',
      base_ram_mb: 256,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved'
    };

    const badges = component.getTechBadges(tpl);
    expect(badges.length).toBe(1);
    expect(badges[0].label).toBe('PHP');
    expect(badges[0].icon).toBe('php');
  });

  it('detecta base de datos satélite activa tanto en services[] como en database.enabled', () => {
    const tplModernServices: ApprovedTemplate = {
      id: 'tpl-1',
      name: 'Entorno Full-Stack JS',
      docker_image: 'solv-local/stack:660620ce5745',
      base_ram_mb: 256,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved',
      services_config: {
        services: [
          { category: 'database', engine: 'postgres', role: 'primary_database' }
        ]
      }
    };

    expect(component.hasDatabaseService(tplModernServices)).toBe(true);
    expect(component.databaseEngine(tplModernServices)).toBe('PostgreSQL');

    const tplHistorical: ApprovedTemplate = {
      id: 'tpl-2',
      name: 'SQL Lab',
      docker_image: 'postgres:16',
      base_ram_mb: 256,
      target_environment: 'IDE_PERSISTENTE',
      status: 'approved',
      services_config: {
        database: {
          enabled: true,
          engine: 'MySQL'
        }
      }
    };

    expect(component.hasDatabaseService(tplHistorical)).toBe(true);
    expect(component.databaseEngine(tplHistorical)).toBe('MySQL');
  });

  it('emite templateSelected al seleccionar una opción del combobox', () => {
    let emitted = '';
    component.templateSelected.subscribe(id => (emitted = id));

    component.onTemplateOptionSelected({ id: 'tpl-1', label: 'Node', value: 'tpl-1' });
    expect(emitted).toBe('tpl-1');
  });
});
