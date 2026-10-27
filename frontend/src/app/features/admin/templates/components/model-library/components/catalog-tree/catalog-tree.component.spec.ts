import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CatalogTreeComponent, CatalogTreeGroup } from './catalog-tree.component';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';

function makeComponent(overrides: Partial<TemplateComponent>): TemplateComponent {
  return {
    id: 'tc-1',
    slug: 'slug',
    layer: 'runtime',
    sub_category: 'backend',
    name: 'Tecnología',
    description: '',
    icon_key: 'python',
    base_image: 'img:latest',
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
  };
}

const CATALOG: TemplateComponent[] = [
  makeComponent({ id: 'rt-py', slug: 'python-312', name: 'Python 3.12', layer: 'runtime', sub_category: 'backend' }),
  makeComponent({ id: 'rt-node', slug: 'node-20', name: 'Node.js 20', layer: 'runtime', sub_category: 'backend', is_active: false }),
  makeComponent({ id: 'rt-web', slug: 'jupyterlab', name: 'JupyterLab', layer: 'runtime', sub_category: 'web' }),
  makeComponent({ id: 'ad-1', slug: 'pytest', name: 'PyTest', layer: 'addon', sub_category: 'toolkit', compatible_runtimes: ['python'] }),
  makeComponent({ id: 'ad-2', slug: 'cpp-tools', name: 'Herramientas C++', layer: 'addon', sub_category: 'toolkit', compatible_runtimes: ['c'] }),
  makeComponent({ id: 'db-1', slug: 'postgres-16', name: 'PostgreSQL 16', layer: 'database', sub_category: 'relational' })
];

describe('CatalogTreeComponent', () => {
  let component: CatalogTreeComponent;
  let fixture: ComponentFixture<CatalogTreeComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CatalogTreeComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(CatalogTreeComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('components', CATALOG);
    fixture.detectChanges();
  });

  it('debe crear el componente', () => {
    expect(component).toBeTruthy();
  });

  it('agrupa las tecnologías por rol con sus conteos de activas', () => {
    const groups = component.groups();
    expect(groups.map((g) => g.id)).toEqual(['languages', 'web', 'addons', 'databases']);

    const byId = (id: string): CatalogTreeGroup => groups.find((g) => g.id === id)!;
    expect(byId('languages').components.map((c) => c.id)).toEqual(['rt-py', 'rt-node']);
    expect(byId('languages').activeCount).toBe(1);
    expect(byId('web').components.map((c) => c.id)).toEqual(['rt-web']);
    // Orden alfabético entre activas: 'Herramientas C++' antes que 'PyTest'.
    expect(byId('addons').components.map((c) => c.id)).toEqual(['ad-2', 'ad-1']);
    expect(byId('databases').components.map((c) => c.id)).toEqual(['db-1']);
  });

  it('suma el total de tecnologías activas e inactivas del sistema', () => {
    expect(component.total()).toBe(6);
    expect(component.activeTotal()).toBe(5);
    expect(component.inactiveTotal()).toBe(1);
  });

  it('ordena las tecnologías activas antes que las inactivas dentro de cada grupo', () => {
    fixture.componentRef.setInput('components', [
      makeComponent({ id: 'a-off', slug: 'a-off', name: 'AAA Inactiva', layer: 'runtime', sub_category: 'backend', is_active: false }),
      makeComponent({ id: 'b-on', slug: 'b-on', name: 'BBB Activa', layer: 'runtime', sub_category: 'backend' })
    ]);
    fixture.detectChanges();

    const languages = component.groups().find((g) => g.id === 'languages')!;
    expect(languages.components.map((c) => c.id)).toEqual(['b-on', 'a-off']);
  });

  it('alterna el colapso de una rama con aria-reflejado en la plantilla', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const firstRow = compiled.querySelector('.branch-row') as HTMLElement;
    expect(firstRow.getAttribute('aria-expanded')).toBe('true');
    expect(compiled.querySelectorAll('.leaf').length).toBeGreaterThan(0);

    component.toggleGroup('languages');
    fixture.detectChanges();

    expect(firstRow.getAttribute('aria-expanded')).toBe('false');
    expect(compiled.querySelector('.tree-branch')!.classList).toContain('is-collapsed');
  });

  it('marca los complementos sin runtime activo compatible como huérfanos', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const flags = compiled.querySelectorAll('.leaf-flag');
    // Solo 'cpp-tools' queda sin runtime activo compatible (python activo, node inactivo).
    expect(flags.length).toBe(1);
    expect(flags[0].textContent).toContain('Sin runtime compatible');
  });

  it('refleja el estado activo/inactivo de cada tecnología en su hoja', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const inactiveLeaf = compiled.querySelector('.leaf.is-inactive') as HTMLElement;
    expect(inactiveLeaf.textContent).toContain('Node.js 20');
    expect(inactiveLeaf.textContent).toContain('Inactivo');
    expect(compiled.querySelectorAll('.leaf-status.status-on').length).toBe(5);
  });

  it('muestra el estado vacío cuando el catálogo no tiene tecnologías', () => {
    fixture.componentRef.setInput('components', []);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.map-empty')).toBeTruthy();
    expect(compiled.querySelector('.tree')).toBeNull();
    expect(component.total()).toBe(0);
  });
});
