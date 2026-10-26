import { ComponentFixture, TestBed } from '@angular/core/testing';
import { StackConstellationComponent } from './stack-constellation.component';
import {
  buildConstellationFromProposal,
  buildConstellationFromSelection,
  AiProposalView,
  StudioDatabaseOption
} from '../../template-studio.utils';

function runtime(id: string, slug: string, iconKey: string, families: string[]) {
  return {
    id,
    slug,
    icon_key: iconKey,
    layer: 'runtime',
    name: slug === 'python-312' ? 'Python 3.12' : slug,
    compatible_runtimes: families,
    ram_delta_mb: 512,
    base_image: 'x',
    description: '',
    sub_category: 'backend',
    install_mode: 'none',
    package_names: [],
    build_layer_snippet: '',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: [],
    service_engine: '',
    is_system: true,
    is_active: true,
    sort_order: 0
  } as unknown as import('@features/admin/models/admin-templates.models').TemplateComponent;
}

function addon(slug: string, iconKey: string, families: string[]) {
  return {
    id: `ad-${slug}`,
    slug,
    icon_key: iconKey,
    layer: 'addon',
    name: slug,
    compatible_runtimes: families,
    ram_delta_mb: 0,
    base_image: '',
    description: '',
    sub_category: 'toolkit',
    install_mode: 'none',
    package_names: [],
    build_layer_snippet: '',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: [],
    service_engine: '',
    is_system: false,
    is_active: true,
    sort_order: 0
  } as unknown as import('@features/admin/models/admin-templates.models').TemplateComponent;
}

describe('StackConstellationComponent', () => {
  let component: StackConstellationComponent;
  let fixture: ComponentFixture<StackConstellationComponent>;

  beforeEach(() => {
    fixture = TestBed.createComponent(StackConstellationComponent);
    component = fixture.componentInstance;
  });

  it('should create the component', () => {
    expect(component).toBeTruthy();
  });

  it('ordena los satelites poniendo faltantes e incompatibles primero', () => {
    const python = runtime('rt-python-312', 'python-312', 'python', ['python']);
    const fastapi = addon('fastapi', 'fastapi', ['python']);
    const mysqlAddon = addon('mysql-driver', 'db', ['node']);

    const view: AiProposalView = {
      runtime: python,
      runtimeSlug: 'python-312',
      web: null,
      webSlug: '',
      addons: [fastapi, mysqlAddon],
      addonSlugs: ['fastapi', 'mysql-driver', 'libreria-fantasma'],
      database: null,
      databaseSlug: '',
      missing: ['libreria-fantasma'],
      ramMB: 1024,
      hasSetupScript: true,
      rationale: 'Stack para APIs'
    };

    const model = buildConstellationFromProposal(view, 'JUEZ_EFIMERO');
    fixture.componentRef.setInput('model', model);

    expect(component.orderedSatellites().map((n) => n.availability)).toEqual([
      'missing',
      'incompatible',
      'available'
    ]);
    expect(component.hasWarnings()).toBe(true);
  });

  it('marca el runtime como nucleo, web como satelite y base de datos como pieza', () => {
    const python = runtime('rt-python-312', 'python-312', 'python', ['python']);
    const angular = runtime('web-angular', 'angular', 'angular', ['node', 'web']);
    const db: StudioDatabaseOption = {
      ...runtime('db-sqlite', 'sqlite', 'sqlite', []),
      layer: 'database',
      service_engine: 'sqlite',
      is_available: true,
      id: 'db-sqlite',
      slug: 'sqlite'
    } as unknown as StudioDatabaseOption;

    const model = buildConstellationFromSelection(python, angular, [], db, 'IDE_PERSISTENTE');

    expect(model.core?.name).toBe('Python 3.12');
    expect(model.core?.isCore).toBe(true);
    expect(model.satellites.map((s) => s.role)).toEqual(['web', 'database']);
  });

  it('no tiene advertencias cuando la seleccion actual es compatible por construccion', () => {
    const python = runtime('rt-python-312', 'python-312', 'python', ['python']);
    const model = buildConstellationFromSelection(python, null, [], null, 'JUEZ_EFIMERO');

    fixture.componentRef.setInput('model', model);
    expect(component.hasWarnings()).toBe(false);
  });
});