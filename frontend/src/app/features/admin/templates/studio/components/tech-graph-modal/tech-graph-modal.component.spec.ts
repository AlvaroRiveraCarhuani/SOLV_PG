import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TechGraphModalComponent } from './tech-graph-modal.component';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';

function makeComponent(overrides: Partial<TemplateComponent>): TemplateComponent {
  return {
    id: 'comp-1',
    slug: 'comp',
    layer: 'addon',
    sub_category: 'toolkit',
    name: 'Componente',
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

const CATALOG: TemplateComponent[] = [
  makeComponent({ id: 'rt-python', slug: 'python-312', layer: 'runtime', sub_category: 'backend', name: 'Python 3.12', icon_key: 'python', sort_order: 10 }),
  makeComponent({ id: 'rt-node', slug: 'node-20', layer: 'runtime', sub_category: 'backend', name: 'Node.js 20', icon_key: 'node', compatible_runtimes: ['node'], sort_order: 20, is_active: false }),
  makeComponent({ id: 'web-angular', slug: 'angular', layer: 'runtime', sub_category: 'web', name: 'Angular', icon_key: 'angular', compatible_runtimes: ['node'], sort_order: 30 }),
  makeComponent({ id: 'ad-fastapi', slug: 'fastapi', layer: 'addon', sub_category: 'framework', name: 'FastAPI', icon_key: 'fastapi', compatible_runtimes: ['python'], sort_order: 40 }),
  makeComponent({ id: 'ad-pytest', slug: 'pytest', layer: 'addon', sub_category: 'toolkit', name: 'PyTest', icon_key: 'pytest', compatible_runtimes: ['universal'], sort_order: 50 }),
  makeComponent({ id: 'ad-orphan', slug: 'orphan-kit', layer: 'addon', sub_category: 'toolkit', name: 'Kit Huérfano', icon_key: 'docker', compatible_runtimes: ['rust-stable'], sort_order: 60 }),
  makeComponent({ id: 'db-postgres', slug: 'postgresql', layer: 'database', sub_category: 'database', name: 'PostgreSQL', icon_key: 'postgresql', compatible_runtimes: ['universal'], service_engine: 'postgres', sort_order: 70 }),
  makeComponent({ id: 'db-none', slug: 'db-none', layer: 'database', sub_category: 'none', name: 'Sin Base de Datos', icon_key: 'docker', compatible_runtimes: ['universal'], service_engine: 'none', sort_order: 5 })
];

describe('TechGraphModalComponent', () => {
  let component: TechGraphModalComponent;
  let fixture: ComponentFixture<TechGraphModalComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TechGraphModalComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(TechGraphModalComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('components', CATALOG);
    fixture.detectChanges();
  });

  it('should create the component', () => {
    expect(component).toBeTruthy();
  });

  it('lista todos los lenguajes como chips y selecciona el primero por defecto', () => {
    expect(component.languages().map(n => n.key)).toEqual(['rt-python', 'rt-node']);
    expect(component.graph().selectedLanguage?.key).toBe('rt-python');
  });

  it('dibuja solo la red del lenguaje elegido: núcleo + piezas compatibles', () => {
    const nodes = component.nodes();
    const keys = nodes.map(n => n.key).sort();

    // Núcleo python + angular (todas las webs se muestran) + postgres + fastapi + pytest.
    expect(keys).toEqual(['db-postgres', 'rt-python', 'ad-fastapi', 'ad-pytest', 'web-angular'].sort());
    // El huérfano, el otro lenguaje y la opción «Sin Base de Datos» no aparecen.
    expect(keys).not.toContain('ad-orphan');
    expect(keys).not.toContain('rt-node');
    expect(keys).not.toContain('db-none');

    const core = nodes.filter(n => n.col === 0);
    expect(core.length).toBe(1);
    expect(core[0].key).toBe('rt-python');
    expect(core[0].role).toBe('lenguaje');
  });

  it('muestra las interfaces web en cualquier lenguaje aunque su compat declare node', () => {
    // Angular declara ['node'] pero el Studio permite combinarlo con Python:
    // el mapa no puede ocultarlo.
    component.selectLanguage('rt-python');
    const keys = component.nodes().map(n => n.key);
    expect(keys).toContain('web-angular');
  });

  it('muestra interfaces y bases de datos como opciones combinables sin afirmar compatibilidad', () => {
    const edgeKinds = new Map(component.edges().map(edge => [edge.target, edge.kind]));
    expect(edgeKinds.get('web-angular')).toBe('combinable');
    expect(edgeKinds.get('db-postgres')).toBe('combinable');
    expect(edgeKinds.get('ad-fastapi')).toBe('compatible');
  });

  it('cuenta las piezas combinables de cada lenguaje para los chips', () => {
    const counts = component.graph().chipCounts;
    // Python: angular + postgres + fastapi + pytest = 4
    expect(counts['rt-python']).toBe(4);
    // Node: angular + postgres + pytest = 3 (fastapi es solo de Python)
    expect(counts['rt-node']).toBe(3);
  });



  it('cada arista sale del núcleo hacia una pieza de la red', () => {
    const edges = component.edges();
    expect(edges.length).toBe(4);
    expect(edges.every(e => e.source === 'rt-python')).toBe(true);
    expect(edges.map(e => e.target).sort()).toEqual(['ad-fastapi', 'ad-pytest', 'db-postgres', 'web-angular'].sort());
    expect(edges.filter(e => e.kind === 'compatible').map(e => e.target).sort()).toEqual(['ad-fastapi', 'ad-pytest']);
    expect(edges.filter(e => e.kind === 'combinable').map(e => e.target).sort()).toEqual(['db-postgres', 'web-angular']);
  });

  it('marca con ∞ a los complementos universales y no a los específicos', () => {
    const pytest = component.nodes().find(n => n.key === 'ad-pytest')!;
    const fastapi = component.nodes().find(n => n.key === 'ad-fastapi')!;

    expect(pytest.isUniversal).toBe(true);
    expect(fastapi.isUniversal).toBe(false);
  });

  it('al cambiar de lenguaje reconstruye su red y limpia la selección', () => {
    component.selectNode('ad-fastapi');
    expect(component.selectedKey()).toBe('ad-fastapi');

    component.selectLanguage('rt-node');
    fixture.detectChanges();

    expect(component.selectedKey()).toBeNull();
    expect(component.graph().selectedLanguage?.key).toBe('rt-node');

    const keys = component.nodes().map(n => n.key);
    expect(keys).toContain('rt-node');
    expect(keys).toContain('ad-pytest'); // universal, también vale para Node
    expect(keys).not.toContain('ad-fastapi'); // solo compatible con Python
    expect(keys).not.toContain('rt-python');
  });

  it('cuenta los complementos sin runtime compatible del catálogo', () => {
    expect(component.graph().orphanCount).toBe(1);
  });

  it('refleja el estado activo e inactivo de las piezas visibles', () => {
    component.selectLanguage('rt-node');
    fixture.detectChanges();

    const node = component.nodes().find(n => n.key === 'rt-node')!;
    expect(node.isActive).toBe(false);
    expect(component.graph().inactiveCount).toBe(1);
  });



  it('al pulsar dos veces la misma pieza limpia la selección', () => {
    component.selectNode('ad-pytest');
    expect(component.selectedKey()).toBe('ad-pytest');
    component.selectNode('ad-pytest');
    expect(component.selectedKey()).toBeNull();
  });


  it('limita la lista inicial de complementos y permite consultar el total con búsqueda', () => {
    const manyAddons = Array.from({ length: 20 }, (_, index) =>
      makeComponent({
        id: `ad-extra-${index}`,
        slug: `extra-${index}`,
        name: `Extra ${index}`,
        layer: 'addon',
        compatible_runtimes: ['python']
      })
    );
    fixture.componentRef.setInput('components', [...CATALOG, ...manyAddons]);
    fixture.detectChanges();

    expect(component.graph().visibleAddons).toBe(12);
    expect(component.graph().hasMoreAddons).toBe(true);

    component.setSearchQuery('Extra 18');
    expect(component.graph().totalAddons).toBe(1);
    expect(component.nodes().some(node => node.name === 'Extra 18')).toBe(true);
  });

  it('expone las relaciones como texto accesible', () => {
    expect(component.accessibleRelationships()).toContain('Python 3.12: Compatible con PyTest');
    expect(component.accessibleRelationships()).toContain('Python 3.12: Combinable con Angular');
  });

  it('muestra el estado vacío cuando el catálogo no tiene lenguajes', () => {
    fixture.componentRef.setInput('components', []);
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.querySelector('.graph-void')).toBeTruthy();
    expect(el.querySelector('.catalog-groups')).toBeNull();
    expect(component.languages().length).toBe(0);
  });

  it('emite close al pulsar Cerrar', () => {
    let closed = false;
    component.dismiss.subscribe(() => (closed = true));

    const buttons = fixture.nativeElement.querySelectorAll(
      'button'
    ) as NodeListOf<HTMLButtonElement>;
    const btn = Array.from(buttons).find(b => (b.textContent || '').trim() === 'Cerrar');
    expect(btn).toBeTruthy();
    btn!.click();
    expect(closed).toBe(true);
  });
});
