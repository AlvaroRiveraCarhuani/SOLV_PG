import '@angular/localize/init';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { TechCatalogComponent, TECH_CATALOG_HELP_KEY } from './tech-catalog.component';
import { AdminTemplatesService, TemplateComponent } from '../../services/admin-templates.service';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ToastService } from '@shared/services/toast.service';

describe('TechCatalogComponent', () => {
  let component: TechCatalogComponent;
  let fixture: ComponentFixture<TechCatalogComponent>;
  let compiled: HTMLElement;
  let getTechCatalogSpy: ReturnType<typeof vi.fn>;
  let helpService: ContextHelpService;

  const mockTechItems: TemplateComponent[] = [
    {
      id: 'rt-py',
      slug: 'python-312',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Python 3.12',
      description: 'Python runtime',
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
      sort_order: 10
    },
    {
      id: 'ad-fastapi',
      slug: 'fastapi',
      layer: 'addon',
      sub_category: 'backend',
      name: 'FastAPI',
      description: 'Modern API framework',
      icon_key: 'fastapi',
      base_image: '',
      compatible_runtimes: ['python'],
      install_mode: 'pip',
      package_names: ['fastapi', 'uvicorn'],
      build_layer_snippet: 'RUN pip install fastapi uvicorn',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: [],
      service_engine: '',
      ram_delta_mb: 64,
      is_system: false,
      is_active: true,
      sort_order: 20
    },
    {
      id: 'db-pg',
      slug: 'postgres-16',
      layer: 'database',
      sub_category: 'backend',
      name: 'PostgreSQL 16',
      description: 'Relational database',
      icon_key: 'postgres',
      base_image: 'postgres:16-alpine',
      compatible_runtimes: ['universal'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: [],
      service_engine: 'postgres',
      ram_delta_mb: 256,
      is_system: true,
      is_active: false,
      sort_order: 30
    }
  ];

  beforeEach(async () => {
    getTechCatalogSpy = vi.fn().mockReturnValue(of(mockTechItems));

    await TestBed.configureTestingModule({
      imports: [TechCatalogComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: AdminTemplatesService,
          useValue: {
            getTechCatalog: getTechCatalogSpy,
            createTechCatalogItem: vi.fn().mockReturnValue(of(mockTechItems[0])),
            updateTechCatalogItem: vi.fn().mockImplementation((_id, dto) => of({ ...mockTechItems[0], ...dto })),
            deleteTechCatalogItem: vi.fn().mockReturnValue(of(void 0))
          }
        }
      ]
    }).compileComponents();

    helpService = TestBed.inject(ContextHelpService);
    fixture = TestBed.createComponent(TechCatalogComponent);
    component = fixture.componentInstance;
    compiled = fixture.nativeElement;
    fixture.detectChanges();
  });

  it('debe inicializar el componente y cargar las tecnologías del catálogo', () => {
    expect(component).toBeTruthy();
    expect(getTechCatalogSpy).toHaveBeenCalledWith(undefined, true);
    expect(component.techComponents().length).toBe(3);
    expect(component.totalComponentsCount()).toBe(3);
    expect(component.activeComponentsCount()).toBe(2);
    expect(component.backendRuntimesCount()).toBe(1);
    expect(component.webRuntimesCount()).toBe(0);
    expect(component.addonsCount()).toBe(1);
    expect(component.databasesCount()).toBe(1);
  });

  it('resume el catálogo en una línea y mueve los conteos a los filtros, sin tarjetas KPI', () => {
    expect(compiled.querySelector('kpi-grid')).toBeNull();
    expect(compiled.querySelector('kpi-card')).toBeNull();

    const summary = compiled.querySelector('.page-summary')?.textContent || '';
    expect(summary).toContain('3');
    expect(summary).toContain('tecnologías registradas');
    expect(summary).toContain('activas');

    const profileLabels = component.componentProfileFilterOptions().map(option => option.label);
    expect(profileLabels).toContain('Lenguaje Principal (1)');
    expect(profileLabels).toContain('Complemento / Librería (1)');
    expect(profileLabels).toContain('Base de Datos (1)');
    expect(profileLabels).toContain('Todos los perfiles (3)');
  });

  it('muestra el esqueleto de carga y no el estado vacío mientras consulta', () => {
    component.isLoading.set(true);
    fixture.detectChanges();

    expect(compiled.querySelector('skeleton-loader')).toBeTruthy();
    expect(compiled.querySelector('.empty-results-card')).toBeNull();
  });

  it('pide confirmación antes de eliminar una tecnología sin uso', () => {
    const service = TestBed.inject(AdminTemplatesService);
    const deleteSpy = vi.spyOn(service, 'deleteTechCatalogItem');
    const toast = TestBed.inject(ToastService);
    const toastSpy = vi.spyOn(toast, 'show');
    const addon = mockTechItems[1];

    component.openDeleteConfirm(addon);
    fixture.detectChanges();
    expect(component.pendingDelete()).toEqual(addon);
    expect(component.deleteDialog()?.action).toBe('delete');
    expect(compiled.querySelector('confirm-modal')).toBeTruthy();
    expect(deleteSpy).not.toHaveBeenCalled();

    component.onDeleteConfirmed();

    expect(deleteSpy).toHaveBeenCalledWith('ad-fastapi');
    expect(component.pendingDelete()).toBeNull();
    expect(toastSpy).toHaveBeenCalledWith(expect.stringContaining('eliminada'), 'success');
  });

  it('no elimina una tecnología en uso: ofrece desactivarla', () => {
    const service = TestBed.inject(AdminTemplatesService);
    const deleteSpy = vi.spyOn(service, 'deleteTechCatalogItem');
    const updateSpy = vi.spyOn(service, 'updateTechCatalogItem');
    const inUse: TemplateComponent = { ...mockTechItems[1], usage_count: 2 };

    component.openDeleteConfirm(inUse);
    expect(component.deleteDialog()?.action).toBe('deactivate');

    component.onDeleteConfirmed();

    expect(deleteSpy).not.toHaveBeenCalled();
    expect(updateSpy).toHaveBeenCalledWith('ad-fastapi', expect.objectContaining({ is_active: false }));
  });

  it('no ofrece eliminar tecnologías base del sistema', () => {
    const toast = TestBed.inject(ToastService);
    const toastSpy = vi.spyOn(toast, 'show');

    component.openDeleteConfirm(mockTechItems[0]);

    expect(component.pendingDelete()).toBeNull();
    expect(toastSpy).toHaveBeenCalledWith(expect.stringContaining('no se pueden eliminar'), 'error');
  });

  it('debe registrar y desregistrar la pantalla en el servicio de ayuda', () => {
    const regSpy = vi.spyOn(helpService, 'registerScreen');
    const unregSpy = vi.spyOn(helpService, 'unregisterScreen');

    component.ngOnInit();
    expect(regSpy).toHaveBeenCalledWith(TECH_CATALOG_HELP_KEY);

    component.ngOnDestroy();
    expect(unregSpy).toHaveBeenCalledWith(TECH_CATALOG_HELP_KEY);
  });

  it('debe alternar entre modo tabla y modo tarjetas', () => {
    component.viewMode.set('table');
    fixture.detectChanges();
    expect(compiled.querySelector('.table-card')).toBeTruthy();
    expect(compiled.querySelector('.components-grid')).toBeNull();

    component.viewMode.set('cards');
    fixture.detectChanges();
    expect(compiled.querySelector('.components-grid')).toBeTruthy();
    expect(compiled.querySelector('.table-card')).toBeNull();
  });

  it('debe abrir el modal de edición de tecnología al llamar a openEditComponentModal', () => {
    const comp = mockTechItems[0];
    component.openEditComponentModal(comp);

    expect(component.showComponentModal()).toBe(true);
    expect(component.editingComponent()).toEqual(comp);
    expect(component.defaultComponentLayer()).toBe('runtime');

    component.closeComponentModal();
    expect(component.showComponentModal()).toBe(false);
    expect(component.editingComponent()).toBeNull();
  });

  it('debe actualizar la fila en techComponents al guardar una tecnología en edición', () => {
    const comp = mockTechItems[0];
    component.openEditComponentModal(comp);

    const updated = { ...comp, name: 'Python 3.12 Actualizado' };
    component.handleComponentSaved(updated);

    expect(component.showComponentModal()).toBe(false);
    expect(component.techComponents().find(c => c.id === comp.id)?.name).toBe('Python 3.12 Actualizado');
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
    component.componentProfileFilter.set('ALL');
    component.openNewComponentModal();
    expect(component.showComponentModal()).toBe(true);
    expect(component.defaultComponentLayer()).toBe('runtime');
    expect(component.defaultComponentSubCategory()).toBe('backend');

    component.componentProfileFilter.set('WEB');
    component.openNewComponentModal();
    expect(component.defaultComponentLayer()).toBe('runtime');
    expect(component.defaultComponentSubCategory()).toBe('web');

    component.componentProfileFilter.set('ADDON');
    component.openNewComponentModal();
    expect(component.defaultComponentLayer()).toBe('addon');
    expect(component.defaultComponentSubCategory()).toBe('backend');

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

  it('debe mostrar el Mapa del Sistema cuando la pestaña activa es map', () => {
    component.activeTab.set('map');
    fixture.detectChanges();

    expect(compiled.querySelector('catalog-tree')).toBeTruthy();
  });

  it('debe limpiar los filtros de tecnologías al llamar a resetComponentFilters', () => {
    component.componentSearchQuery.set('fastapi');
    component.componentProfileFilter.set('ADDON');
    component.componentStatusFilter.set('ACTIVE');

    component.resetComponentFilters();

    expect(component.componentSearchQuery()).toBe('');
    expect(component.componentProfileFilter()).toBe('ALL');
    expect(component.componentStatusFilter()).toBe('ALL');
  });
});
