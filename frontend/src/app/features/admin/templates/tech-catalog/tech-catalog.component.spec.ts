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
      sort_order: 10,
      usage_count: 5
    },
    {
      id: 'rt-angular',
      slug: 'angular-spa',
      layer: 'runtime',
      sub_category: 'web',
      name: 'Angular 22 SPA',
      description: 'Frontend SPA framework',
      icon_key: 'angular',
      base_image: 'node:20-bookworm',
      compatible_runtimes: ['web', 'node'],
      install_mode: 'npm',
      package_names: ['@angular/core', '@angular/cli'],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: ['ng', 'node', 'npm'],
      service_engine: '',
      ram_delta_mb: 256,
      is_system: true,
      is_active: true,
      sort_order: 15,
      usage_count: 2
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
      sort_order: 20,
      usage_count: 0
    },
    {
      id: 'db-pg',
      slug: 'postgres-16',
      layer: 'database',
      sub_category: 'relational',
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
      sort_order: 30,
      usage_count: 1
    },
    {
      id: 'db-none',
      slug: 'db-none',
      layer: 'database',
      sub_category: 'none',
      name: 'Sin Base de Datos',
      description: 'Solo archivos de código en la carpeta del estudiante',
      icon_key: 'docker',
      base_image: '',
      compatible_runtimes: ['universal'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: [],
      service_engine: 'none',
      ram_delta_mb: 0,
      is_system: true,
      is_active: true,
      sort_order: 300,
      usage_count: 0
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
    // Excluye db-none
    expect(component.techComponents().length).toBe(4);
    expect(component.totalComponentsCount()).toBe(4);
    expect(component.activeComponentsCount()).toBe(3);
    expect(component.backendRuntimesCount()).toBe(1);
    expect(component.webRuntimesCount()).toBe(1);
    expect(component.addonsCount()).toBe(1);
    expect(component.databasesCount()).toBe(1);
  });

  it('excluye la pseudo-tecnología db-none de los conteos y de la tabla', () => {
    expect(component.techComponents().some(c => c.slug === 'db-none')).toBe(false);
    expect(component.filteredTechComponents().some(c => c.slug === 'db-none')).toBe(false);
    expect(compiled.textContent).not.toContain('Sin Base de Datos');
  });

  it('renderiza cabecera institucional y 5 pestañas por perfil en <status-tabs> sin Mapa del Sistema', () => {
    const title = compiled.querySelector('.page-title');
    expect(title?.textContent).toContain('Catálogo de Tecnologías');

    const subtitle = compiled.querySelector('.page-subtitle');
    expect(subtitle?.textContent).toContain('Administre lenguajes, interfaces web, complementos y bases de datos');

    // No debe haber texto redundante "62 tecnologías registradas · 62 activas"
    expect(compiled.querySelector('.page-summary')).toBeNull();

    // Pestañas por perfil
    const tabs = component.profileTabs();
    expect(tabs.length).toBe(5);
    expect(tabs.map(t => t.id)).toEqual(['ALL', 'BACKEND', 'WEB', 'ADDON', 'DATABASE']);
    expect(tabs.find(t => t.id === 'ALL')?.count).toBe(4);
    expect(tabs.find(t => t.id === 'BACKEND')?.count).toBe(1);
    expect(tabs.find(t => t.id === 'WEB')?.count).toBe(1);
    expect(tabs.find(t => t.id === 'ADDON')?.count).toBe(1);
    expect(tabs.find(t => t.id === 'DATABASE')?.count).toBe(1);

    expect(compiled.querySelector('status-tabs')).toBeTruthy();
    // No debe existir catalog-tree en esta vista
    expect(compiled.querySelector('catalog-tree')).toBeNull();
  });

  it('filtra por perfiles usando las pestañas de status-tabs', () => {
    component.onTabChange('BACKEND');
    fixture.detectChanges();
    expect(component.filteredTechComponents().length).toBe(1);
    expect(component.filteredTechComponents()[0].slug).toBe('python-312');

    component.onTabChange('WEB');
    fixture.detectChanges();
    expect(component.filteredTechComponents().length).toBe(1);
    expect(component.filteredTechComponents()[0].slug).toBe('angular-spa');

    component.onTabChange('ADDON');
    fixture.detectChanges();
    expect(component.filteredTechComponents().length).toBe(1);
    expect(component.filteredTechComponents()[0].slug).toBe('fastapi');

    component.onTabChange('DATABASE');
    fixture.detectChanges();
    expect(component.filteredTechComponents().length).toBe(1);
    expect(component.filteredTechComponents()[0].slug).toBe('postgres-16');

    component.onTabChange('ALL');
    fixture.detectChanges();
    expect(component.filteredTechComponents().length).toBe(4);
  });

  it('filtra por disponibilidad y origen mediante filter-popover', () => {
    expect(component.activeFilterCount()).toBe(0);

    // Filtro disponibilidad
    component.setStatusFilter('ACTIVE');
    expect(component.activeFilterCount()).toBe(1);
    expect(component.filteredTechComponents().length).toBe(3);

    component.setStatusFilter('INACTIVE');
    expect(component.filteredTechComponents().length).toBe(1);
    expect(component.filteredTechComponents()[0].slug).toBe('postgres-16');

    // Filtro origen
    component.setStatusFilter('ALL');
    component.setOriginFilter('CUSTOM');
    expect(component.activeFilterCount()).toBe(1);
    expect(component.filteredTechComponents().length).toBe(1);
    expect(component.filteredTechComponents()[0].slug).toBe('fastapi');

    component.setOriginFilter('OFFICIAL');
    expect(component.filteredTechComponents().length).toBe(3);

    // Limpiar filtros
    component.clearFilters();
    expect(component.activeFilterCount()).toBe(0);
    expect(component.filteredTechComponents().length).toBe(4);
  });

  it('arranca en vista Tabla por defecto con 6 columnas, ordenamiento y pagination-bar', () => {
    expect(component.viewMode()).toBe('table');
    expect(compiled.querySelector('.table-card')).toBeTruthy();

    const headers = compiled.querySelectorAll('.data-table th');
    expect(headers.length).toBe(6);
    expect(headers[0].textContent).toContain('TECNOLOGÍA');
    expect(headers[1].textContent).toContain('PERFIL / ECOSISTEMA');
    expect(headers[2].textContent).toContain('IMAGEN / PAQUETES');
    expect(headers[3].textContent).toContain('USO');
    expect(headers[4].textContent).toContain('ACTIVA');
    expect(headers[5].textContent).toContain('ACCIONES');

    expect(compiled.querySelector('pagination-bar')).toBeTruthy();
  });

  it('ordena por columna al hacer clic en las cabeceras sortables', () => {
    // Por defecto ordenado por nombre ascendente
    expect(component.sortField()).toBe('name');
    expect(component.sortDirection()).toBe('asc');

    // Ordenar por uso
    component.toggleSort('usage');
    expect(component.sortField()).toBe('usage');
    expect(component.sortDirection()).toBe('desc');
    expect(component.filteredTechComponents()[0].slug).toBe('python-312'); // 5 usos

    component.toggleSort('usage');
    expect(component.sortDirection()).toBe('asc');
    expect(component.filteredTechComponents()[0].slug).toBe('fastapi'); // 0 usos

    // Ordenar por perfil
    component.toggleSort('profile');
    expect(component.sortField()).toBe('profile');
  });

  it('oculta cuota de RAM cuando es menor o igual a 0', () => {
    const compZeroRam: TemplateComponent = {
      ...mockTechItems[2],
      ram_delta_mb: 0
    };
    component.techComponents.set([compZeroRam]);
    fixture.detectChanges();

    expect(compiled.textContent).not.toContain('0 MB RAM');
    expect(compiled.textContent).not.toContain('+0 MB');
  });

  it('pluraliza la columna USO con concordancia real y raya cuando no tiene uso', () => {
    component.techComponents.set([
      { ...mockTechItems[0], name: 'Uno', usage_count: 1 },
      { ...mockTechItems[1], name: 'Varios', usage_count: 3 },
      { ...mockTechItems[2], name: 'Ninguno', usage_count: 0 }
    ]);
    fixture.detectChanges();

    const rows = Array.from(compiled.querySelectorAll('.data-table tbody tr'));
    const usageText = (row: Element) =>
      row.querySelectorAll('td')[3].textContent?.replace(/\s+/g, ' ').trim();

    // Orden por nombre ascendente: Ninguno, Uno, Varios
    expect(usageText(rows[0])).toBe('—');
    expect(usageText(rows[1])).toBe('1 entorno');
    expect(usageText(rows[2])).toBe('3 entornos');
    expect(compiled.textContent).not.toContain('entorno(s)');
    expect(compiled.textContent).not.toContain('1 entornos');
  });

  it('muestra la RAM sin «+» en lenguajes base y con «+» en capas aditivas', () => {
    const baseRuntime = { ...mockTechItems[0], ram_delta_mb: 512 };
    const webRuntime = { ...mockTechItems[1], ram_delta_mb: 256 };
    const addon = { ...mockTechItems[2], ram_delta_mb: 128 };
    const database = { ...mockTechItems[3], ram_delta_mb: 256 };

    expect(component.getRamLabel(baseRuntime)).toBe('512 MB');
    expect(component.getRamLabel(webRuntime)).toBe('+256 MB');
    expect(component.getRamLabel(addon)).toBe('+128 MB');
    expect(component.getRamLabel(database)).toBe('+256 MB');
    expect(component.getRamLabel({ ...baseRuntime, ram_delta_mb: 0 })).toBe('');

    component.techComponents.set([baseRuntime, webRuntime, addon, database]);
    fixture.detectChanges();

    const ramTags = Array.from(compiled.querySelectorAll('.ram-delta-tag')).map(tag =>
      tag.textContent?.trim()
    );
    expect(ramTags).toContain('512 MB');
    expect(ramTags).toContain('+256 MB');
    expect(ramTags).toContain('+128 MB');
    expect(ramTags).not.toContain('+512 MB');
  });

  it('reserva ranuras fijas de acción para alinear el lápiz en todas las filas', () => {
    component.techComponents.set([mockTechItems[0], mockTechItems[2]]);
    fixture.detectChanges();

    const rows = Array.from(compiled.querySelectorAll('.data-table tbody tr'));
    const rowOf = (name: string) => rows.find(row => row.textContent?.includes(name)) as Element;
    const actionsOf = (row: Element) => row.querySelector('.row-actions');

    const runtimeRow = rowOf('Python 3.12');
    const addonRow = rowOf('FastAPI');

    // El runtime base tiene «+ Comp.»; el complemento ocupa su ranura vacía.
    expect(actionsOf(runtimeRow)?.querySelector('.btn-action-text')).toBeTruthy();
    expect(actionsOf(addonRow)?.querySelector('.btn-action-text')).toBeNull();
    expect(actionsOf(addonRow)?.querySelector('.action-slot-addon')).toBeTruthy();

    for (const row of [runtimeRow, addonRow]) {
      const actions = actionsOf(row);
      expect(actions?.querySelector('.action-edit')).toBeTruthy();
      // Toda fila reserva exactamente tres ranuras (complemento, editar, eliminar).
      expect(actions?.children.length).toBe(3);
    }
  });

  it('muestra badge Personalizada solo si la tecnología no es de sistema', () => {
    // mockTechItems[2] es FastAPI con is_system: false
    component.techComponents.set([mockTechItems[0], mockTechItems[2]]);
    fixture.detectChanges();

    const customBadges = compiled.querySelectorAll('.custom-badge');
    expect(customBadges.length).toBe(1);
    expect(customBadges[0].textContent).toContain('Personalizada');
    // No debe aparecer la etiqueta repetida 'Catálogo Oficial'
    expect(compiled.textContent).not.toContain('Catálogo Oficial');
  });

  it('deshabilita el botón eliminar en tecnologías no oficiales si están en uso', () => {
    const compInUseCustom: TemplateComponent = {
      ...mockTechItems[2],
      usage_count: 3
    };
    component.techComponents.set([compInUseCustom]);
    fixture.detectChanges();

    const deleteBtn = compiled.querySelector<HTMLButtonElement>('.action-delete');
    expect(deleteBtn).toBeTruthy();
    expect(deleteBtn?.disabled).toBe(true);
    expect(deleteBtn?.title).toContain('No se puede eliminar mientras esté en uso');
  });

  it('pide confirmación antes de eliminar una tecnología sin uso', () => {
    const service = TestBed.inject(AdminTemplatesService);
    const deleteSpy = vi.spyOn(service, 'deleteTechCatalogItem');
    const toast = TestBed.inject(ToastService);
    const toastSpy = vi.spyOn(toast, 'show');
    const addon = mockTechItems[2]; // fastapi: usage_count: 0

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

  it('debe alternar entre modo tabla y modo tarjetas compartiendo la lista paginada', () => {
    component.viewMode.set('cards');
    fixture.detectChanges();
    expect(compiled.querySelector('.components-grid')).toBeTruthy();
    expect(compiled.querySelector('.table-card')).toBeNull();

    component.viewMode.set('table');
    fixture.detectChanges();
    expect(compiled.querySelector('.table-card')).toBeTruthy();
    expect(compiled.querySelector('.components-grid')).toBeNull();
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

  it('debe abrir el modal con presets contextuales según selectedProfileTab al llamar a openNewComponentModal()', () => {
    component.selectedProfileTab.set('ALL');
    component.openNewComponentModal();
    expect(component.showComponentModal()).toBe(true);
    expect(component.defaultComponentLayer()).toBe('runtime');
    expect(component.defaultComponentSubCategory()).toBe('backend');

    component.selectedProfileTab.set('WEB');
    component.openNewComponentModal();
    expect(component.defaultComponentLayer()).toBe('runtime');
    expect(component.defaultComponentSubCategory()).toBe('web');

    component.selectedProfileTab.set('ADDON');
    component.openNewComponentModal();
    expect(component.defaultComponentLayer()).toBe('addon');
    expect(component.defaultComponentSubCategory()).toBe('backend');

    component.selectedProfileTab.set('DATABASE');
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

  it('debe limpiar los filtros de tecnologías al llamar a resetComponentFilters', () => {
    component.componentSearchQuery.set('fastapi');
    component.selectedProfileTab.set('ADDON');
    component.statusFilter.set('ACTIVE');
    component.originFilter.set('CUSTOM');

    component.resetComponentFilters();

    expect(component.componentSearchQuery()).toBe('');
    expect(component.selectedProfileTab()).toBe('ALL');
    expect(component.statusFilter()).toBe('ALL');
    expect(component.originFilter()).toBe('ALL');
  });
});
