import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { provideRouter, Router } from '@angular/router';
import { ModelLibraryComponent } from './model-library.component';
import { AdminTemplatesService, TemplateModelItem, TemplateCategory, AdminTemplateItem, TemplateComponent } from '../../../services/admin-templates.service';

describe('ModelLibraryComponent Spec', () => {
  let component: ModelLibraryComponent;
  let fixture: ComponentFixture<ModelLibraryComponent>;

  const mockModels: TemplateModelItem[] = [
    {
      id: 'm-1',
      name: 'Python 3 Ciencia de Datos',
      title: 'Python 3 Ciencia de Datos',
      description: 'Stack estándar con pandas, numpy y jupyter',
      category_id: 'c-1',
      category_name: 'Ciencia de Datos & IA',
      docker_image: 'python:3.12-slim-bookworm',
      target_environment: 'IDE_PERSISTENTE',
      base_ram_mb: 1024,
      usage_count: 5,
      is_active: true,
      sort_order: 1,
      source_template_id: 't-source-uuid-1234',
      created_at: '2026-09-22T00:00:00Z',
      updated_at: '2026-09-22T00:00:00Z'
    },
    {
      id: 'm-2',
      name: 'Node.js 20 Fullstack',
      title: 'Node.js 20 Fullstack',
      description: 'Entorno web con express y typescript',
      category_id: 'c-2',
      category_name: 'Desarrollo Web & Cloud',
      docker_image: 'node:20-bookworm-slim',
      target_environment: 'IDE_PERSISTENTE',
      base_ram_mb: 768,
      usage_count: 0,
      is_active: false,
      sort_order: 2,
      source_template_id: null,
      created_at: '2026-09-22T00:00:00Z',
      updated_at: '2026-09-22T00:00:00Z'
    }
  ];

  const mockCategories: TemplateCategory[] = [
    {
      id: 'c-1',
      name: 'Ciencia de Datos & IA',
      description: 'Modelos para machine learning y analítica',
      sort_order: 1,
      created_at: '2026-09-22T00:00:00Z',
      updated_at: '2026-09-22T00:00:00Z'
    },
    {
      id: 'c-2',
      name: 'Desarrollo Web & Cloud',
      description: 'Frameworks web modernos',
      sort_order: 2,
      created_at: '2026-09-22T00:00:00Z',
      updated_at: '2026-09-22T00:00:00Z'
    }
  ];

  const mockApprovedTemplates: Partial<AdminTemplateItem>[] = [
    {
      id: 't-1',
      name: 'Plantilla Python Base',
      description: 'Plantilla aprobada de Python para entornos',
      docker_image: 'python:3.12-slim-bookworm',
      target_environment: 'IDE_PERSISTENTE',
      category_id: 'c-1',
      status: 'approved'
    }
  ];

  let getModelsSpy: ReturnType<typeof vi.fn>;
  let getCategoriesSpy: ReturnType<typeof vi.fn>;
  let deactivateModelSpy: ReturnType<typeof vi.fn>;
  let reorderCategoriesSpy: ReturnType<typeof vi.fn>;
  let deleteCategorySpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    getModelsSpy = vi.fn().mockReturnValue(of(mockModels));
    getCategoriesSpy = vi.fn().mockReturnValue(of(mockCategories));
    deactivateModelSpy = vi.fn().mockReturnValue(of(void 0));
    reorderCategoriesSpy = vi.fn().mockReturnValue(of(void 0));
    deleteCategorySpy = vi.fn().mockReturnValue(of(void 0));

    const mockService = {
      getModels: getModelsSpy,
      getCategories: getCategoriesSpy,
      deactivateModel: deactivateModelSpy,
      reactivateModel: vi.fn().mockReturnValue(of(void 0)),
      updateModel: vi.fn().mockReturnValue(of(mockModels[0])),
      createCategory: vi.fn().mockReturnValue(of(mockCategories[0])),
      updateCategory: vi.fn().mockReturnValue(of(mockCategories[0])),
      deleteCategory: deleteCategorySpy,
      reorderCategories: reorderCategoriesSpy,
      getTemplates: vi.fn().mockReturnValue(of(mockApprovedTemplates)),
      getTechCatalog: vi.fn().mockReturnValue(of([])),
      updateTechCatalogItem: vi.fn().mockImplementation((id: string, dto: unknown) => of({ id, ...(dto as object) })),
      deleteTechCatalogItem: vi.fn().mockReturnValue(of(void 0))
    };

    TestBed.configureTestingModule({
      imports: [ModelLibraryComponent],
      providers: [
        provideRouter([]),
        { provide: AdminTemplatesService, useValue: mockService }
      ]
    });

    fixture = TestBed.createComponent(ModelLibraryComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe inicializar el componente y cargar modelos desde el servicio', () => {
    expect(component).toBeTruthy();
    expect(getModelsSpy).toHaveBeenCalledWith(undefined, undefined, true);
    expect(component.models().length).toBe(2);
  });

  it('debe calcular correctamente los KPIs del catálogo institucional', () => {
    expect(component.totalModelsCount()).toBe(2);
    expect(component.activeModelsCount()).toBe(1);
    expect(component.ideModelsCount()).toBe(2);
    expect(component.judgeModelsCount()).toBe(0);
    expect(component.categoriesCount()).toBe(2);
  });

  it('debe permitir alternar entre modo tabla y modo tarjetas', () => {
    expect(component.viewMode()).toBe('table');
    component.viewMode.set('cards');
    expect(component.viewMode()).toBe('cards');
  });

  it('debe mostrar el usage_count real en la grilla de modelos', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const usageBadges = compiled.querySelectorAll('.usage-badge');
    expect(usageBadges.length).toBe(2);

    // Primer modelo tiene 5 plantillas
    expect(usageBadges[0].textContent).toContain('5');
    expect(usageBadges[0].textContent).toContain('plantillas');

    // Segundo modelo tiene 0 plantillas
    expect(usageBadges[1].textContent).toContain('0');
  });

  it('debe calcular la cantidad de modelos asociados a cada categoría', () => {
    expect(component.getCategoryModelCount('c-1')).toBe(1);
    expect(component.getCategoryModelCount('c-2')).toBe(1);
    expect(component.getCategoryModelCount('c-inexistente')).toBe(0);
  });

  it('debe filtrar modelos por propósito cuando se modifica filterPurpose', () => {
    component.filterPurpose.set('IDE_PERSISTENTE');
    expect(component.filteredModels().length).toBe(2);

    component.filterPurpose.set('JUEZ_VIRTUAL');
    expect(component.filteredModels().length).toBe(0);
  });

  it('debe llamar a deactivateModel al alternar estado de un modelo activo', () => {
    component.toggleModelActive(mockModels[0]);
    expect(deactivateModelSpy).toHaveBeenCalledWith('m-1');
  });

  it('debe reordenar categorías y llamar a reorderCategories', () => {
    component.moveCategoryDown(0);
    expect(reorderCategoriesSpy).toHaveBeenCalled();
  });

  it('debe abrir el modal de nuevo modelo y delegar la promoción en model-new-modal', () => {
    component.openNewModelModal();
    expect(component.showNewModelModal()).toBe(true);
    expect(component.approvedTemplates().length).toBe(1);
    expect(component.approvedTemplates()[0].id).toBe('t-1');
  });

  it('no debe exponer opción fantasma de categoría; el filtro conserva "Todas las categorías"', () => {
    expect((component as unknown as Record<string, unknown>)['categoryComboboxOptions']).toBeUndefined();
    const filterOpts = component.categoryFilterComboboxOptions();
    expect(filterOpts[0]).toMatchObject({ id: 'ALL', label: 'Todas las categorías' });
    expect(filterOpts.some(o => (o.label || '').includes('Sin categoría'))).toBe(false);
  });

  it('navega al estudio dedicado de plantillas al pedir creación', () => {
    const router = TestBed.inject(Router);
    const navigateSpy = vi.spyOn(router, 'navigate');
    component.showNewModelModal.set(true);
    component.navigateToTemplatesWizard();
    expect(component.showNewModelModal()).toBe(false);
    expect(navigateSpy).toHaveBeenCalledWith(['/admin/plantillas/nueva']);
  });

  it('debe abrir el modal de registro de tecnología desde la pestaña Catálogo de Tecnologías', () => {
    const text = (fixture.nativeElement as HTMLElement).textContent || '';
    expect(text).toContain('Catálogo de Tecnologías');
    component.activeTab.set('components');
    component.openNewComponentModal('addon');
    expect(component.showComponentModal()).toBe(true);
    expect(component.defaultComponentLayer()).toBe('addon');
  });

  it('debe abrir el modal de edición de tecnología al llamar a openEditComponentModal', () => {
    const comp: TemplateComponent = {
      id: 'tc-1',
      slug: 'python-312',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Python 3.12',
      description: 'Python runtime',
      icon_key: 'python',
      base_image: 'python:3.12-slim-bookworm',
      compatible_runtimes: ['python'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: ['python3'],
      service_engine: '',
      ram_delta_mb: 512,
      is_system: true,
      is_active: true,
      sort_order: 10
    };

    component.openEditComponentModal(comp);

    expect(component.showComponentModal()).toBe(true);
    expect(component.editingComponent()).toEqual(comp);
    expect(component.defaultComponentLayer()).toBe('runtime');

    component.closeComponentModal();
    expect(component.showComponentModal()).toBe(false);
    expect(component.editingComponent()).toBeNull();
  });

  it('debe actualizar la fila en techComponents in-place al guardar una tecnología en modo edición', () => {
    const initialComp: TemplateComponent = {
      id: 'tc-edit',
      slug: 'go-126',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Go 1.26',
      description: 'Go runtime',
      icon_key: 'go',
      base_image: 'golang:1.26',
      compatible_runtimes: ['go'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: ['go'],
      service_engine: '',
      ram_delta_mb: 512,
      is_system: false,
      is_active: true,
      sort_order: 20
    };

    component.techComponents.set([initialComp]);

    component.openEditComponentModal(initialComp);
    expect(component.editingComponent()).toBe(initialComp);

    const updatedComp: TemplateComponent = {
      ...initialComp,
      name: 'Go 1.26 Updated',
      judge_entrypoint: 'go run main.go'
    };

    component.handleComponentSaved(updatedComp);

    expect(component.showComponentModal()).toBe(false);
    expect(component.editingComponent()).toBeNull();
    const currentList = component.techComponents();
    expect(currentList.length).toBe(1);
    expect(currentList[0].name).toBe('Go 1.26 Updated');
    expect(currentList[0].judge_entrypoint).toBe('go run main.go');
  });

  it('debe soportar modelos duales AMBOS en KPIs y filtrado de propósitos', () => {
    const dualModel: TemplateModelItem = {
      id: 'm-dual-1',
      name: 'Python Dual Stack',
      title: 'Python Dual Stack',
      description: 'Entorno para IDE y Juez',
      category_id: 'cat-prog',
      docker_image: 'python:3.12-slim',
      base_ram_mb: 512,
      target_environment: 'AMBOS',
      is_active: true,
      usage_count: 3,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString()
    };

    component.models.set([...component.models(), dualModel]);

    // Debe contarse tanto para IDE como para Juez
    expect(component.ideModelsCount()).toBe(3);
    expect(component.judgeModelsCount()).toBe(1);

    // Filtrar por AMBOS
    component.filterPurpose.set('AMBOS');
    expect(component.filteredModels().length).toBe(1);
    expect(component.filteredModels()[0].id).toBe('m-dual-1');

    // Filtrar por IDE_PERSISTENTE debe incluir también los modelos AMBOS
    component.filterPurpose.set('IDE_PERSISTENTE');
    expect(component.filteredModels().some(m => m.id === 'm-dual-1')).toBe(true);

    // Filtrar por JUEZ_VIRTUAL debe incluir también los modelos AMBOS
    component.filterPurpose.set('JUEZ_VIRTUAL');
    expect(component.filteredModels().some(m => m.id === 'm-dual-1')).toBe(true);
  });

  describe('Gobernanza del Catálogo de Tecnologías', () => {
    const mockTechItems: TemplateComponent[] = [
      {
        id: 'rt-py',
        slug: 'python-312',
        layer: 'runtime',
        sub_category: 'backend',
        name: 'Python 3.12',
        description: 'Runtime de Python',
        icon_key: 'python',
        base_image: 'python:3.12-bookworm',
        compatible_runtimes: ['python'],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: 'python3 solution.py',
        tools_provided: ['python3', 'pip'],
        service_engine: '',
        ram_delta_mb: 512,
        is_system: true,
        is_active: true,
        sort_order: 1
      },
      {
        id: 'rt-node-web',
        slug: 'angular-dev',
        layer: 'runtime',
        sub_category: 'web',
        name: 'Angular 22 Web',
        description: 'Frontend Angular',
        icon_key: 'angular',
        base_image: 'node:20-bookworm-slim',
        compatible_runtimes: ['node'],
        install_mode: 'npm',
        package_names: ['@angular/cli'],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['ng', 'node'],
        service_engine: '',
        ram_delta_mb: 768,
        is_system: false,
        is_active: true,
        sort_order: 2
      },
      {
        id: 'ad-pytest',
        slug: 'pytest-lib',
        layer: 'addon',
        sub_category: 'framework',
        name: 'Pytest',
        description: 'Framework de pruebas para Python',
        icon_key: 'python',
        base_image: '',
        compatible_runtimes: ['python'],
        install_mode: 'pip',
        package_names: ['pytest'],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['pytest'],
        service_engine: '',
        ram_delta_mb: 0,
        is_system: true,
        is_active: true,
        sort_order: 3
      },
      {
        id: 'ad-fastapi',
        slug: 'fastapi-proc',
        layer: 'addon',
        sub_category: 'framework',
        name: 'FastAPI Uvicorn',
        description: 'Servidor activo Python',
        icon_key: 'python',
        base_image: '',
        compatible_runtimes: ['python'],
        install_mode: 'pip',
        package_names: ['fastapi', 'uvicorn'],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: [],
        service_engine: '',
        ram_delta_mb: 128,
        is_system: false,
        is_active: false,
        sort_order: 4
      },
      {
        id: 'ad-orphan',
        slug: 'rust-diesel',
        layer: 'addon',
        sub_category: 'framework',
        name: 'Diesel ORM',
        description: 'ORM para Rust sin runtime activo',
        icon_key: 'rust',
        base_image: '',
        compatible_runtimes: ['rust'],
        install_mode: 'custom',
        package_names: ['diesel'],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: [],
        service_engine: '',
        ram_delta_mb: 0,
        is_system: false,
        is_active: true,
        sort_order: 5
      },
      {
        id: 'db-pg',
        slug: 'postgres-16',
        layer: 'database',
        sub_category: 'service',
        name: 'PostgreSQL 16',
        description: 'Base de datos relacional',
        icon_key: 'postgresql',
        base_image: 'postgres:16-alpine',
        compatible_runtimes: [],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['psql'],
        service_engine: 'postgres',
        ram_delta_mb: 256,
        is_system: true,
        is_active: true,
        sort_order: 6
      }
    ];

    beforeEach(() => {
      component.techComponents.set(mockTechItems);
    });

    it('debe filtrar tecnologías por búsqueda de texto en name, slug y herramientas', () => {
      component.componentSearchQuery.set('pytest');
      expect(component.filteredTechComponents().length).toBe(1);
      expect(component.filteredTechComponents()[0].id).toBe('ad-pytest');

      component.componentSearchQuery.set('psql');
      expect(component.filteredTechComponents().length).toBe(1);
      expect(component.filteredTechComponents()[0].id).toBe('db-pg');
    });

    it('debe filtrar tecnologías por perfil unificado (BACKEND, WEB, ADDON, DATABASE)', () => {
      component.componentProfileFilter.set('BACKEND');
      expect(component.filteredTechComponents().length).toBe(1);
      expect(component.filteredTechComponents()[0].id).toBe('rt-py');

      component.componentProfileFilter.set('WEB');
      expect(component.filteredTechComponents().length).toBe(1);
      expect(component.filteredTechComponents()[0].id).toBe('rt-node-web');

      component.componentProfileFilter.set('ADDON');
      expect(component.filteredTechComponents().length).toBe(3);

      component.componentProfileFilter.set('DATABASE');
      expect(component.filteredTechComponents().length).toBe(1);
      expect(component.filteredTechComponents()[0].id).toBe('db-pg');
    });

    it('debe filtrar tecnologías por estado activo e inactivo y ordenar activos primero', () => {
      component.componentStatusFilter.set('INACTIVE');
      expect(component.filteredTechComponents().length).toBe(1);
      expect(component.filteredTechComponents()[0].id).toBe('ad-fastapi');

      component.componentStatusFilter.set('ACTIVE');
      expect(component.filteredTechComponents().length).toBe(5);
      expect(component.filteredTechComponents().every(c => c.is_active !== false)).toBe(true);
    });

    it('debe detectar complementos huérfanos correctamente con isOrphanAddon', () => {
      const pytest = mockTechItems.find(c => c.id === 'ad-pytest')!;
      const diesel = mockTechItems.find(c => c.id === 'ad-orphan')!;
      const pyRuntime = mockTechItems.find(c => c.id === 'rt-py')!;

      // Pytest tiene runtime Python activo -> NO es huérfano
      expect(component.isOrphanAddon(pytest)).toBe(false);

      // Diesel requiere Rust y no hay runtime de Rust activo -> ES huérfano
      expect(component.isOrphanAddon(diesel)).toBe(true);

      // Si no es addon -> jamás es huérfano
      expect(component.isOrphanAddon(pyRuntime)).toBe(false);
    });

    it('debe calcular la cantidad de complementos compatibles activos para un runtime', () => {
      const pyRuntime = mockTechItems.find(c => c.id === 'rt-py')!;
      // Solo Pytest está activo (FastAPI está inactivo)
      expect(component.getCompatibleAddonsCount(pyRuntime)).toBe(1);
    });

    it('debe computar la etiqueta de memoria getComponentMemoryBadge según el perfil de consumo', () => {
      const pytest = mockTechItems.find(c => c.id === 'ad-pytest')!;
      const fastapi = mockTechItems.find(c => c.id === 'ad-fastapi')!;
      const pyRuntime = mockTechItems.find(c => c.id === 'rt-py')!;

      // Addon con 0 MB -> En disco
      expect(component.getComponentMemoryBadge(pytest)).toEqual({
        label: '0 MB RAM',
        detail: 'En disco',
        variant: 'light'
      });

      // Addon con > 0 MB -> Proceso activo
      expect(component.getComponentMemoryBadge(fastapi)).toEqual({
        label: '+128 MB',
        detail: 'Proceso activo',
        variant: 'standard'
      });

      // Runtime estándar (512 MB)
      expect(component.getComponentMemoryBadge(pyRuntime)).toEqual({
        label: '512 MB',
        detail: 'Perfil estándar',
        variant: 'standard'
      });

      // Runtime estándar con ram_delta_mb === 0 (ej. Python, Node o Angular por defecto)
      const zeroRamPyRuntime: TemplateComponent = {
        ...pyRuntime,
        ram_delta_mb: 0
      };
      expect(component.getComponentMemoryBadge(zeroRamPyRuntime)).toEqual({
        label: '256 MB',
        detail: 'Perfil estándar',
        variant: 'standard'
      });

      // Runtime liviano por tecnología C/C++ con ram_delta_mb === 0
      const cppRuntime: TemplateComponent = {
        ...pyRuntime,
        icon_key: 'cpp',
        ram_delta_mb: 0
      };
      expect(component.getComponentMemoryBadge(cppRuntime)).toEqual({
        label: '128 MB',
        detail: 'Perfil liviano',
        variant: 'light'
      });

      // Runtime intensivo por tecnología Java con ram_delta_mb === 0
      const javaRuntime: TemplateComponent = {
        ...pyRuntime,
        icon_key: 'java',
        ram_delta_mb: 0
      };
      expect(component.getComponentMemoryBadge(javaRuntime)).toEqual({
        label: '1024 MB',
        detail: 'Perfil intensivo',
        variant: 'heavy'
      });
    });

    it('debe enviar el payload completo en toggleComponentActive y actualizar optimistamente', () => {
      const py = mockTechItems[0];
      const service = TestBed.inject(AdminTemplatesService);
      const updateSpy = vi.spyOn(service, 'updateTechCatalogItem');

      component.toggleComponentActive(py);

      expect(updateSpy).toHaveBeenCalledWith(
        'rt-py',
        expect.objectContaining({
          name: 'Python 3.12',
          base_image: 'python:3.12-bookworm',
          is_active: false
        })
      );
      expect(component.techComponents().find(c => c.id === 'rt-py')?.is_active).toBe(false);
    });

    it('debe abrir el modal con presets contextuales según componentProfileFilter al llamar a openNewComponentModal()', () => {
      // Sin filtro ('ALL') -> runtime / backend
      component.componentProfileFilter.set('ALL');
      component.openNewComponentModal();
      expect(component.showComponentModal()).toBe(true);
      expect(component.defaultComponentLayer()).toBe('runtime');
      expect(component.defaultComponentSubCategory()).toBe('backend');

      // Filtro WEB -> runtime / web
      component.componentProfileFilter.set('WEB');
      component.openNewComponentModal();
      expect(component.defaultComponentLayer()).toBe('runtime');
      expect(component.defaultComponentSubCategory()).toBe('web');

      // Filtro ADDON -> addon / backend
      component.componentProfileFilter.set('ADDON');
      component.openNewComponentModal();
      expect(component.defaultComponentLayer()).toBe('addon');
      expect(component.defaultComponentSubCategory()).toBe('backend');

      // Filtro DATABASE -> database / backend
      component.componentProfileFilter.set('DATABASE');
      component.openNewComponentModal();
      expect(component.defaultComponentLayer()).toBe('database');
      expect(component.defaultComponentSubCategory()).toBe('backend');
    });

    it('debe abrir el modal con presets correctos al llamar a openAddAddonForRuntime', () => {
      const py = mockTechItems[0];
      component.openAddAddonForRuntime(py);

      expect(component.showComponentModal()).toBe(true);
      expect(component.defaultComponentLayer()).toBe('addon');
      expect(component.defaultComponentSubCategory()).toBe('backend');
      expect(component.defaultCompatibleRuntime()).toBe('python');
    });
  });
});
