import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { ComponentFormModalComponent } from './component-form-modal.component';
import { AdminTemplatesService } from '../../../../../services/admin-templates.service';
import { AdminConfigIaService } from '../../../../../configuracion/admin-config-ia.service';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';

describe('ComponentFormModalComponent', () => {
  let fixture: ComponentFixture<ComponentFormModalComponent>;
  let component: ComponentFormModalComponent;
  let tplService: AdminTemplatesService;

  const mockAdminTemplatesService = {
    createTechCatalogItem: vi.fn(),
    updateTechCatalogItem: vi.fn()
  };

  const mockAiService = {
    getStatus: vi.fn().mockReturnValue(of({ enabled: true, provider: 'gemini', model: 'gemini-2.5-flash' })),
    suggestComponent: vi.fn(),
    resolveError: vi.fn().mockReturnValue('Error en el servicio de IA')
  };

  beforeEach(async () => {
    mockAdminTemplatesService.createTechCatalogItem.mockReset();
    mockAdminTemplatesService.updateTechCatalogItem.mockReset();
    mockAiService.getStatus.mockReset();
    mockAiService.getStatus.mockReturnValue(of({ enabled: true, provider: 'gemini', model: 'gemini-2.5-flash' }));
    mockAiService.suggestComponent.mockReset();
    mockAiService.resolveError.mockReset();
    mockAiService.resolveError.mockReturnValue('Error en el servicio de IA');

    await TestBed.configureTestingModule({
      imports: [ComponentFormModalComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: mockAdminTemplatesService },
        { provide: AdminConfigIaService, useValue: mockAiService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ComponentFormModalComponent);
    component = fixture.componentInstance;
    tplService = TestBed.inject(AdminTemplatesService);
  });

  it('should initialize with defaultLayer input and populate available runtimes dynamically', () => {
    component.defaultLayer = 'addon';
    component.defaultCompatibleRuntime = 'python';
    component.availableRuntimes = [
      {
        id: 'rt-python',
        slug: 'python',
        layer: 'runtime',
        sub_category: 'backend',
        name: 'Python',
        description: '',
        icon_key: 'python',
        base_image: 'python:3.12-slim-bookworm',
        compatible_runtimes: ['python'],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: 'python3 solution.py',
        tools_provided: ['python3'],
        service_engine: '',
        ram_delta_mb: 256,
        is_system: true,
        is_active: true,
        sort_order: 10
      },
      {
        id: 'rt-rust',
        slug: 'rust',
        layer: 'runtime',
        sub_category: 'backend',
        name: 'Rust (Cargo)',
        description: '',
        icon_key: 'rust',
        base_image: 'rust:1.75-bookworm',
        compatible_runtimes: ['rust'],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['cargo'],
        service_engine: '',
        ram_delta_mb: 256,
        is_system: true,
        is_active: true,
        sort_order: 70
      },
      {
        id: 'ad-fastapi',
        slug: 'fastapi',
        layer: 'addon',
        sub_category: 'framework',
        name: 'FastAPI',
        description: '',
        icon_key: 'python',
        base_image: '',
        compatible_runtimes: ['python'],
        install_mode: 'pip',
        package_names: ['fastapi'],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: [],
        service_engine: '',
        ram_delta_mb: 0,
        is_system: false,
        is_active: true,
        sort_order: 100
      }
    ];

    component.ngOnInit();
    fixture.detectChanges();

    expect(component.layer()).toBe('addon');
    expect(component.compatibleRuntime()).toBe('python');

    const runtimeOpts = component.compatibleRuntimeOptions();
    expect(runtimeOpts.some(o => o.id === 'python')).toBe(true);
    expect(runtimeOpts.some(o => o.id === 'rust')).toBe(true);
    expect(runtimeOpts.some(o => o.id === 'universal')).toBe(true);
    // Non-runtime component should not be in compatibleRuntimeOptions
    expect(runtimeOpts.some(o => o.id === 'fastapi')).toBe(false);
  });

  it('should support package managers including none, pip, uv, npm, apt, cargo, go, composer, custom', () => {
    const presets = component.installPresetOptions;
    const ids = presets.map((p: { id: string }) => p.id);
    expect(ids).toContain('none');
    expect(ids).toContain('pip');
    expect(ids).toContain('uv');
    expect(ids).toContain('npm');
    expect(ids).toContain('apt');
    expect(ids).toContain('cargo');
    expect(ids).toContain('go');
    expect(ids).toContain('composer');
    expect(ids).toContain('custom');
  });

  it('should synthesize safe custom install command for uv manager and submit payload', () => {
    const savedSpy = vi.fn();
    component.saved.subscribe(savedSpy);

    const createdMock: TemplateComponent = {
      id: 'ad-uv-fastapi',
      slug: 'uv-fastapi',
      layer: 'addon',
      sub_category: 'framework',
      name: 'FastAPI via uv',
      description: '',
      icon_key: 'python',
      base_image: '',
      compatible_runtimes: ['python'],
      install_mode: 'custom',
      package_names: ['fastapi', 'uvicorn'],
      build_layer_snippet: 'RUN uv pip install --system --no-cache fastapi uvicorn',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: ['uvicorn'],
      service_engine: '',
      ram_delta_mb: 64,
      is_system: false,
      is_active: true,
      sort_order: 100
    };
    mockAdminTemplatesService.createTechCatalogItem.mockReturnValue(of(createdMock));

    component.defaultLayer = 'addon';
    component.ngOnInit();

    component.name.set('FastAPI via uv');
    component.compatibleRuntime.set('python');
    component.installPreset.set('uv');
    component.packageNamesInput.set('fastapi, uvicorn');
    component.toolsProvidedInput.set('uvicorn');

    component.onSave();

    expect(tplService.createTechCatalogItem).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'FastAPI via uv',
        layer: 'addon',
        install_mode: 'uv',
        package_names: ['fastapi', 'uvicorn'],
        build_layer_snippet: 'RUN uv pip install --system --no-cache fastapi uvicorn'
      })
    );
    expect(savedSpy).toHaveBeenCalledWith(createdMock);
  });

  it('should block unsafe characters in package names for security', () => {
    component.defaultLayer = 'addon';
    component.ngOnInit();

    component.name.set('Malicious Package');
    component.compatibleRuntime.set('python');
    component.installPreset.set('pip');
    component.packageNamesInput.set('numpy; rm -rf /');

    component.onSave();

    expect(component.errorMessage()).toContain('caracteres inválidos');
    expect(tplService.createTechCatalogItem).not.toHaveBeenCalled();
  });

  it('should save runtime with normalized slug and judge entrypoint for backend', () => {
    const savedSpy = vi.fn();
    component.saved.subscribe(savedSpy);

    const createdRuntime: TemplateComponent = {
      id: 'rt-go-126',
      slug: 'go-1-26',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Go 1.26 Modern',
      description: 'Go compiler and tools',
      icon_key: 'go',
      base_image: 'golang:1.26-bookworm',
      compatible_runtimes: ['go-1-26', 'go'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: 'go run main.go',
      tools_provided: ['go'],
      service_engine: '',
      ram_delta_mb: 512,
      is_system: false,
      is_active: true,
      sort_order: 80
    };
    mockAdminTemplatesService.createTechCatalogItem.mockReturnValue(of(createdRuntime));

    component.defaultLayer = 'runtime';
    component.ngOnInit();

    component.name.set('Go 1.26 Modern');
    component.baseImage.set('golang:1.26-bookworm');
    component.iconKey.set('go');
    component.judgeEntrypoint.set('go run main.go');
    component.toolsProvidedInput.set('go');

    component.onSave();

    expect(tplService.createTechCatalogItem).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Go 1.26 Modern',
        slug: 'go-1-26-modern',
        layer: 'runtime',
        sub_category: 'backend',
        base_image: 'golang:1.26-bookworm',
        judge_entrypoint: 'go run main.go',
        compatible_runtimes: expect.arrayContaining(['go-1-26-modern', 'go'])
      })
    );
    expect(savedSpy).toHaveBeenCalledWith(createdRuntime);
  });

  it('should apply web platform runtime defaults (npm preset and node base image)', () => {
    component.defaultLayer = 'runtime';
    component.ngOnInit();

    expect(component.subCategory()).toBe('backend');
    expect(component.installPreset()).toBe('none');

    // Switch to web platform
    component.onSubCategoryChange('web');

    expect(component.subCategory()).toBe('web');
    expect(component.baseImage()).toBe('node:20-bookworm-slim');
    expect(component.installPreset()).toBe('npm');
  });

  it('should initialize with defaultSubCategory web when opened for web creation', () => {
    component.defaultLayer = 'runtime';
    component.defaultSubCategory = 'web';
    component.ngOnInit();

    expect(component.layer()).toBe('runtime');
    expect(component.subCategory()).toBe('web');
    expect(component.baseImage()).toBe('node:20-bookworm-slim');
    expect(component.installPreset()).toBe('npm');
  });

  it('should initialize with defaultSubCategory backend when opened for backend runtime creation', () => {
    component.defaultLayer = 'runtime';
    component.defaultSubCategory = 'backend';
    component.ngOnInit();

    expect(component.layer()).toBe('runtime');
    expect(component.subCategory()).toBe('backend');
    expect(component.installPreset()).toBe('none');
  });

  it('should preload form values when editingComponent is provided', () => {
    const existingComp: TemplateComponent = {
      id: 'rt-edit-1',
      slug: 'python-312',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Python 3.12 Custom',
      description: 'Pre-configured Python environment',
      icon_key: 'python',
      base_image: 'python:3.12-slim-bookworm',
      compatible_runtimes: ['python-312', 'python'],
      install_mode: 'custom',
      package_names: ['pytest', 'requests'],
      build_layer_snippet: 'RUN pip install --no-cache-dir pytest requests',
      setup_snippet: '',
      judge_entrypoint: 'python3 solution.py',
      tools_provided: ['python3', 'pytest'],
      service_engine: '',
      ram_delta_mb: 512,
      is_system: false,
      is_active: true,
      sort_order: 15
    };

    component.editingComponent = existingComp;
    component.ngOnInit();

    expect(component.name()).toBe('Python 3.12 Custom');
    expect(component.layer()).toBe('runtime');
    expect(component.subCategory()).toBe('backend');
    expect(component.baseImage()).toBe('python:3.12-slim-bookworm');
    expect(component.judgeEntrypoint()).toBe('python3 solution.py');
    expect(component.installPreset()).toBe('custom');
    expect(component.customBuildSnippet()).toBe('RUN pip install --no-cache-dir pytest requests');
    expect(component.packageNamesInput()).toBe('pytest, requests');
    expect(component.toolsProvidedInput()).toBe('python3, pytest');
  });

  it('should call updateTechCatalogItem when saving in edit mode', () => {
    const existingComp: TemplateComponent = {
      id: 'rt-edit-2',
      slug: 'python-312',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Python 3.12 Custom',
      description: 'Pre-configured Python environment',
      icon_key: 'python',
      base_image: 'python:3.12-slim-bookworm',
      compatible_runtimes: ['python-312', 'python'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: 'python3 solution.py',
      tools_provided: ['python3'],
      service_engine: '',
      ram_delta_mb: 512,
      is_system: false,
      is_active: true,
      sort_order: 15
    };

    const updatedComp: TemplateComponent = {
      ...existingComp,
      name: 'Python 3.12 Updated',
      judge_entrypoint: 'pytest -q'
    };

    mockAdminTemplatesService.updateTechCatalogItem.mockReturnValue(of(updatedComp));

    const savedSpy = vi.fn();
    component.saved.subscribe(savedSpy);

    component.editingComponent = existingComp;
    component.ngOnInit();

    component.name.set('Python 3.12 Updated');
    component.judgeEntrypoint.set('pytest -q');

    component.onSave();

    expect(tplService.updateTechCatalogItem).toHaveBeenCalledWith(
      'rt-edit-2',
      expect.objectContaining({
        name: 'Python 3.12 Updated',
        judge_entrypoint: 'pytest -q'
      })
    );
    expect(tplService.createTechCatalogItem).not.toHaveBeenCalled();
    expect(savedSpy).toHaveBeenCalledWith(updatedComp);
  });

  describe('Unified Catalog Profiles and Ecosystem Synchronization', () => {
    it('should expose 3 unified catalogKindOptions (backend, web, addon)', () => {
      const kinds = component.catalogKindOptions.map(o => o.value);
      expect(kinds).toEqual(['backend', 'web', 'addon']);
    });

    it('should compute catalogKind accurately from layer and subCategory', () => {
      component.layer.set('runtime');
      component.subCategory.set('backend');
      expect(component.catalogKind()).toBe('backend');

      component.subCategory.set('web');
      expect(component.catalogKind()).toBe('web');

      component.layer.set('addon');
      expect(component.catalogKind()).toBe('addon');
    });

    it('should lock context and display banner when lockContext=true without editingComponent', () => {
      component.lockContext = true;
      component.defaultLayer = 'runtime';
      component.defaultSubCategory = 'backend';
      component.ngOnInit();
      fixture.detectChanges();

      expect(component.isContextLocked()).toBe(true);

      const banner = fixture.nativeElement.querySelector('.context-lock-banner');
      expect(banner).toBeTruthy();
      expect(fixture.nativeElement.querySelector('#comp-layer')).toBeNull();

      const unlockBtn: HTMLButtonElement | null = fixture.nativeElement.querySelector('.btn-unlock-context');
      expect(unlockBtn).toBeTruthy();
      unlockBtn?.click();
      fixture.detectChanges();

      expect(component.isContextLocked()).toBe(false);
      expect(fixture.nativeElement.querySelector('#comp-layer')).toBeTruthy();
    });

    it('should not lock context when editingComponent is present even if lockContext=true', () => {
      component.lockContext = true;
      component.editingComponent = {
        id: 'ad-test',
        slug: 'test',
        layer: 'addon',
        sub_category: 'framework',
        name: 'Test Addon',
        description: '',
        icon_key: 'docker',
        base_image: '',
        compatible_runtimes: [],
        install_mode: 'pip',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: [],
        service_engine: '',
        ram_delta_mb: 0,
        is_system: false,
        is_active: true,
        sort_order: 1
      };
      component.ngOnInit();
      fixture.detectChanges();

      expect(component.isContextLocked()).toBe(false);
      expect(fixture.nativeElement.querySelector('.context-lock-banner')).toBeNull();
    });

    it('should switch defaults properly when onCatalogKindChange is called', () => {
      component.onCatalogKindChange('web');
      expect(component.layer()).toBe('runtime');
      expect(component.subCategory()).toBe('web');
      expect(component.baseImage()).toBe('node:20-bookworm-slim');
      expect(component.installPreset()).toBe('npm');

      component.onCatalogKindChange('addon');
      expect(component.layer()).toBe('addon');
      expect(component.subCategory()).toBe('framework');
      expect(component.memoryProfile()).toBe('library');

      component.onCatalogKindChange('backend');
      expect(component.layer()).toBe('runtime');
      expect(component.subCategory()).toBe('backend');
      expect(component.installPreset()).toBe('none');
    });

    it('should infer appropriate package preset in onCompatibleRuntimeChange', () => {
      component.layer.set('addon');
      component.availableRuntimes = [
        {
          id: 'rt-node',
          slug: 'node',
          layer: 'runtime',
          sub_category: 'web',
          name: 'Angular / Node.js',
          description: '',
          icon_key: 'angular',
          base_image: 'node:20',
          compatible_runtimes: ['node'],
          install_mode: 'none',
          package_names: [],
          build_layer_snippet: '',
          setup_snippet: '',
          judge_entrypoint: '',
          tools_provided: [],
          service_engine: '',
          ram_delta_mb: 256,
          is_system: true,
          is_active: true,
          sort_order: 1
        }
      ];

      component.onCompatibleRuntimeChange('node');
      expect(component.installPreset()).toBe('npm');

      component.onCompatibleRuntimeChange('python');
      expect(component.installPreset()).toBe('pip');

      component.onCompatibleRuntimeChange('rust');
      expect(component.installPreset()).toBe('cargo');

      component.onCompatibleRuntimeChange('golang');
      expect(component.installPreset()).toBe('go');

      component.onCompatibleRuntimeChange('php');
      expect(component.installPreset()).toBe('composer');

      component.onCompatibleRuntimeChange('cpp');
      expect(component.installPreset()).toBe('apt');
    });

    it('should update uiCopy reactively depending on kind and install preset', () => {
      component.onCatalogKindChange('backend');
      expect(component.uiCopy().modalTitle).toBe('Registrar Lenguaje Principal');
      expect(component.uiCopy().contextBadgeLabel).toContain('Lenguaje Principal');

      component.onCatalogKindChange('web');
      expect(component.uiCopy().modalTitle).toBe('Registrar Plataforma Web / Frontend');
      expect(component.uiCopy().contextBadgeLabel).toContain('Interfaz Web');

      component.onCatalogKindChange('addon');
      expect(component.uiCopy().modalTitle).toBe('Registrar Complemento o Librería');
      expect(component.uiCopy().contextBadgeLabel).toContain('Complemento / Librería');

      component.installPreset.set('npm');
      expect(component.uiCopy().packagesPlaceholder).toContain('@angular');

      component.installPreset.set('apt');
      expect(component.uiCopy().packagesPlaceholder).toContain('valgrind');
    });

    it('should clear residual web baseImage when switching kind to backend', () => {
      component.onCatalogKindChange('web');
      expect(component.baseImage()).toBe('node:20-bookworm-slim');

      component.onCatalogKindChange('backend');
      expect(component.baseImage()).toBe('');
      expect(component.layer()).toBe('runtime');
      expect(component.subCategory()).toBe('backend');
    });

    it('should inherit install_mode from matched available runtime before regex fallback', () => {
      component.layer.set('addon');
      component.availableRuntimes = [
        {
          id: 'rt-custom-haskell',
          slug: 'haskell-cabal',
          layer: 'runtime',
          sub_category: 'backend',
          name: 'Haskell GHC',
          description: '',
          icon_key: 'docker',
          base_image: 'haskell:9.6',
          compatible_runtimes: ['haskell'],
          install_mode: 'apt',
          package_names: [],
          build_layer_snippet: '',
          setup_snippet: '',
          judge_entrypoint: '',
          tools_provided: ['ghc', 'cabal'],
          service_engine: '',
          ram_delta_mb: 512,
          is_system: false,
          is_active: true,
          sort_order: 10
        }
      ];

      component.onCompatibleRuntimeChange('haskell-cabal');
      expect(component.installPreset()).toBe('apt');
    });
  });

  describe('Smart Icon Bar and Progressive Disclosure', () => {
    it('should filter curated icons based on iconSearchQuery', () => {
      expect(component.filteredCuratedIcons().length).toBeGreaterThan(0);

      component.iconSearchQuery.set('python');
      const filtered = component.filteredCuratedIcons();
      expect(filtered.some(i => i.key === 'python')).toBe(true);
      expect(filtered.every(i => i.label.toLowerCase().includes('python') || i.key.includes('python'))).toBe(true);
    });

    it('should auto-detect icons from common technology aliases in name', () => {
      component.onNameChange('Pytest Runner');
      expect(component.iconKey()).toBe('python');
      expect(component.activeIconLabel()).toBe('Python');
      expect(component.iconResolutionSource()).toBe('Detectado automáticamente');

      component.onNameChange('GCC 13 C++ Compiler');
      expect(component.iconKey()).toBe('cpp');
      expect(component.activeIconLabel()).toBe('C++');

      component.onNameChange('Angular Material Components');
      expect(component.iconKey()).toBe('angular');

      component.onNameChange('JUnit 5 Test Framework');
      expect(component.iconKey()).toBe('java');

      component.onNameChange('Tokio Async Runtime');
      expect(component.iconKey()).toBe('rust');
    });

    it('should inherit ecosystem icon for addons when name does not match any known icon', () => {
      component.layer.set('addon');
      component.availableRuntimes = [
        {
          id: 'rt-rust',
          slug: 'rust',
          layer: 'runtime',
          sub_category: 'backend',
          name: 'Rust (Cargo)',
          description: '',
          icon_key: 'rust',
          base_image: 'rust:1.75-bookworm',
          compatible_runtimes: ['rust'],
          install_mode: 'none',
          package_names: [],
          build_layer_snippet: '',
          setup_snippet: '',
          judge_entrypoint: '',
          tools_provided: ['cargo'],
          service_engine: '',
          ram_delta_mb: 256,
          is_system: true,
          is_active: true,
          sort_order: 1
        }
      ];

      component.onNameChange('Unmatched Generic Library');
      component.onCompatibleRuntimeChange('rust');

      expect(component.iconKey()).toBe('rust');
      expect(component.activeIconLabel()).toBe('Rust');
      expect(component.iconResolutionSource()).toBe('Detectado automáticamente');
    });

    it('should toggle icon picker and custom URL input with mutual exclusion', () => {
      expect(component.showIconPickerGrid()).toBe(false);
      expect(component.showCustomIconInput()).toBe(false);

      component.toggleIconPickerGrid();
      expect(component.showIconPickerGrid()).toBe(true);
      expect(component.showCustomIconInput()).toBe(false);

      component.toggleCustomIconInput();
      expect(component.showCustomIconInput()).toBe(true);
      expect(component.showIconPickerGrid()).toBe(false);

      component.toggleCustomIconInput();
      expect(component.showCustomIconInput()).toBe(false);
    });

    it('should handle manual icon selection and close drawers', () => {
      component.showIconPickerGrid.set(true);
      component.selectCuratedIcon('go');

      expect(component.iconKey()).toBe('go');
      expect(component.isIconManuallySelected()).toBe(true);
      expect(component.showIconPickerGrid()).toBe(false);
      expect(component.showCustomIconInput()).toBe(false);
      expect(component.iconResolutionSource()).toBe('Seleccionado manualmente');
    });

    it('should show custom icon source when custom URL is set', () => {
      component.customIconUrl.set('https://example.com/logo.svg');
      expect(component.effectiveIconKey()).toBe('https://example.com/logo.svg');
      expect(component.activeIconLabel()).toBe('Icono personalizado (URL / SVG)');
      expect(component.iconResolutionSource()).toBe('URL personalizada');
    });

    it('should prevent short token collision when matching icons in name', () => {
      component.onNameChange('Django REST Framework');
      expect(component.iconKey()).toBe('django');

      component.onNameChange('MongoDB Driver');
      expect(component.iconKey()).toBe('mongodb');

      component.onNameChange('Go 1.22');
      expect(component.iconKey()).toBe('go');

      component.onNameChange('Compilador Go');
      expect(component.iconKey()).toBe('go');
    });

    it('should not lock manual icon selection when editing a component with default docker icon', () => {
      component.editingComponent = {
        id: 'ad-docker-default',
        slug: 'custom-tool',
        layer: 'addon',
        sub_category: 'framework',
        name: 'Custom Pytest Tool',
        description: '',
        icon_key: 'docker',
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
        is_system: false,
        is_active: true,
        sort_order: 1
      };

      component.ngOnInit();

      expect(component.isIconManuallySelected()).toBe(false);
      expect(component.iconKey()).toBe('python');
    });
  });

  describe('Guided Ecosystem Profiles, Installer Filtering & Space-Separated Tools', () => {
    it('should apply official ecosystem profile for C and C++ separately without manual commands', () => {
      component.defaultLayer = 'runtime';
      component.defaultSubCategory = 'backend';
      component.ngOnInit();

      component.onSelectEcosystemProfile('c-gcc13');
      expect(component.name()).toBe('C (GCC 13)');
      expect(component.iconKey()).toBe('c');
      expect(component.baseImage()).toBe('gcc:13.2-bookworm');
      expect(component.judgeEntrypoint()).toBe('gcc -O2 solution.c -o solution && ./solution');
      expect(component.ramDeltaMB()).toBe(384);

      component.onSelectEcosystemProfile('cpp-gcc13');
      expect(component.name()).toBe('C++ (G++ 13)');
      expect(component.iconKey()).toBe('cpp');
      expect(component.baseImage()).toBe('gcc:13.2-bookworm');
      expect(component.judgeEntrypoint()).toBe('g++ -O2 solution.cpp -o solution && ./solution');
      expect(component.ramDeltaMB()).toBe(384);
    });

    it('should block incompatible package managers (like npm or pip) when compatible runtime is C/C++', () => {
      component.defaultLayer = 'addon';
      component.ngOnInit();

      component.onCompatibleRuntimeChange('cpp');
      const allowedIds = component.filteredInstallPresetOptions().map(o => o.id);
      expect(allowedIds).toContain('apt');
      expect(allowedIds).not.toContain('npm');
      expect(allowedIds).not.toContain('pip');
      expect(allowedIds).not.toContain('composer');
      expect(allowedIds).not.toContain('cargo');
    });

    it('should only allow native ecosystem manager plus apt for Python and Node addons', () => {
      component.defaultLayer = 'addon';
      component.ngOnInit();

      component.onCompatibleRuntimeChange('python');
      const pyAllowed = component.filteredInstallPresetOptions().map(o => o.id);
      expect(pyAllowed).toContain('pip');
      expect(pyAllowed).toContain('uv');
      expect(pyAllowed).toContain('apt');
      expect(pyAllowed).not.toContain('npm');
      expect(pyAllowed).not.toContain('composer');

      component.onCompatibleRuntimeChange('node');
      const nodeAllowed = component.filteredInstallPresetOptions().map(o => o.id);
      expect(nodeAllowed).toContain('npm');
      expect(nodeAllowed).toContain('apt');
      expect(nodeAllowed).not.toContain('pip');
      expect(nodeAllowed).not.toContain('cargo');
    });

    it('should split space-separated toolsProvidedInput and auto-populate tools from apt packages on save', () => {
      mockAdminTemplatesService.createTechCatalogItem.mockImplementation(dto => of({ id: 'new-1', ...dto }));

      component.defaultLayer = 'addon';
      component.ngOnInit();
      component.name.set('Kit Depuracion C++');
      component.onCompatibleRuntimeChange('cpp');
      component.packageNamesInput.set('gdb valgrind cmake');

      component.onSave();

      expect(tplService.createTechCatalogItem).toHaveBeenCalledWith(
        expect.objectContaining({
          install_mode: 'apt',
          package_names: ['gdb', 'valgrind', 'cmake'],
          tools_provided: ['gdb', 'valgrind', 'cmake'],
          ram_delta_mb: 0
        })
      );
    });
  });

  describe('Contextual Catalog Presets, Smart Card Auto-Fill & Column Icon Filtering', () => {
    it('should filter curated icons by column: backend excludes web frameworks and databases', () => {
      component.onCatalogKindChange('backend');
      const keys = component.filteredCuratedIcons().map(i => i.key);
      expect(keys).toContain('c');
      expect(keys).toContain('cpp');
      expect(keys).toContain('python');
      expect(keys).toContain('rust');
      expect(keys).not.toContain('angular');
      expect(keys).not.toContain('react');
      expect(keys).not.toContain('postgres');
      expect(keys).not.toContain('mongodb');
    });

    it('should filter curated icons by column: web only shows web platforms and never C++, Rust or databases', () => {
      component.onCatalogKindChange('web');
      const keys = component.filteredCuratedIcons().map(i => i.key);
      expect(keys).toContain('angular');
      expect(keys).toContain('react');
      expect(keys).toContain('vue');
      expect(keys).toContain('vite');
      expect(keys).not.toContain('cpp');
      expect(keys).not.toContain('rust');
      expect(keys).not.toContain('python');
      expect(keys).not.toContain('postgres');
    });

    it('should filter curated icons by column: addon excludes database engines and respects compatible ecosystem', () => {
      component.onCatalogKindChange('addon');
      component.onCompatibleRuntimeChange('python');
      const pyKeys = component.filteredCuratedIcons().map(i => i.key);
      expect(pyKeys).toContain('fastapi');
      expect(pyKeys).toContain('django');
      expect(pyKeys).toContain('python');
      expect(pyKeys).not.toContain('postgres');
      expect(pyKeys).not.toContain('mysql');
      expect(pyKeys).not.toContain('mongodb');
      expect(pyKeys).not.toContain('redis');
      expect(pyKeys).not.toContain('laravel');
    });

    it('should auto-fill full technology preset when clicking a card in the icon grid during creation', () => {
      component.defaultLayer = 'runtime';
      component.defaultSubCategory = 'backend';
      component.ngOnInit();

      component.selectCuratedIcon('rust');
      expect(component.name()).toBe('Rust');
      expect(component.baseImage()).toBe('rust:1.77-slim-bookworm');
      expect(component.judgeEntrypoint()).toContain('rustc');

      component.onCatalogKindChange('web');
      component.selectCuratedIcon('angular');
      expect(component.name()).toBe('Angular');
      expect(component.baseImage()).toBe('node:20-bookworm-slim');
      expect(component.packageNamesInput()).toContain('@angular/cli');
    });

    it('should expose contextual addon presets for C/C++ and Python and apply them in 1 click', () => {
      component.defaultLayer = 'addon';
      component.ngOnInit();

      component.onCompatibleRuntimeChange('cpp');
      const cppPresets = component.contextualAddonPresets();
      expect(cppPresets.some(p => p.id === 'cpp-debug-valgrind')).toBe(true);
      expect(cppPresets.some(p => p.id === 'cpp-cmake')).toBe(true);

      component.onSelectAddonPreset('cpp-debug-valgrind');
      expect(component.name()).toBe('Depuración y Memoria (GDB + Valgrind)');
      expect(component.installPreset()).toBe('apt');
      expect(component.packageNamesInput()).toBe('gdb, valgrind');
    });

    it('should detect already-installed presets in the catalog, sort uninstalled first, and switch to update mode if an installed preset is selected', () => {
      component.availableRuntimes = [
        {
          id: 'existing-py-id',
          slug: 'python-312',
          layer: 'runtime',
          sub_category: 'backend',
          name: 'Python 3.12',
          description: 'Entorno oficial con pip',
          icon_key: 'python',
          base_image: 'python:3.12-slim-bookworm',
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
        }
      ];
      component.defaultLayer = 'runtime';
      component.defaultSubCategory = 'backend';
      component.ngOnInit();

      expect(component.isPresetInCatalog('python-312', 'Python 3.12')).toBe(true);
      expect(component.isPresetInCatalog('php-83', 'PHP 8.3 CLI')).toBe(false);

      // Selecting an already-installed preset should bind existingCatalogMatchId so saving updates instead of failing with 409
      component.onSelectEcosystemProfile('python-312');
      expect(component.existingCatalogMatch()?.id).toBe('existing-py-id');
    });

    it('should clear preset state cleanly when selecting custom mode (+ Personalizado)', () => {
      component.defaultLayer = 'runtime';
      component.defaultSubCategory = 'backend';
      component.ngOnInit();

      component.onSelectEcosystemProfile('rust-stable');
      expect(component.name()).toBe('Rust');
      expect(component.judgeEntrypoint()).toContain('rustc');

      component.onSelectCustomMode();
      expect(component.isCustomPresetSelected()).toBe(true);
      expect(component.selectedEcosystemProfileId()).toBe('');
      expect(component.name()).toBe('');
      expect(component.baseImage()).toBe('');
      expect(component.judgeEntrypoint()).toBe('');
      expect(component.toolsProvidedInput()).toBe('');
    });

    it('should support open version parameterization (suggested and custom versions) and update baseImage and name dynamically', () => {
      component.defaultLayer = 'runtime';
      component.defaultSubCategory = 'backend';
      component.ngOnInit();

      component.onSelectEcosystemProfile('python-312');
      expect(component.selectedVersion()).toBe('3.12');
      expect(component.baseImage()).toBe('python:3.12-slim-bookworm');
      expect(component.availableVersionsForSelectedProfile().length).toBeGreaterThan(2);

      // Docente/Admin chooses an older version (3.10) or a newer/custom version (3.14)
      component.onVersionChange('3.10');
      expect(component.baseImage()).toBe('python:3.10-slim-bookworm');
      expect(component.name()).toBe('Python 3.10');

      component.onVersionChange('3.14');
      expect(component.baseImage()).toBe('python:3.14-slim-bookworm');
      expect(component.name()).toBe('Python 3.14');
    });

    it('should expose pnpm, yarn, and bun for Node/Web ecosystem and resolve custom web baseImage dynamically from catalog', () => {
      component.availableRuntimes = [
        {
          id: 'rt-node-22',
          slug: 'node-22',
          layer: 'runtime',
          sub_category: 'backend',
          name: 'Node.js 22 LTS',
          description: '',
          icon_key: 'node',
          base_image: 'node:22-bookworm-slim',
          compatible_runtimes: ['node', 'web'],
          install_mode: 'none',
          package_names: [],
          build_layer_snippet: '',
          setup_snippet: '',
          judge_entrypoint: 'node solution.js',
          tools_provided: ['node', 'npm', 'npx'],
          service_engine: '',
          ram_delta_mb: 512,
          is_system: true,
          is_active: true,
          sort_order: 20
        }
      ];
      component.defaultLayer = 'addon';
      component.ngOnInit();

      component.onCompatibleRuntimeChange('node-22');
      const nodeAllowed = component.filteredInstallPresetOptions().map(o => o.id);
      expect(nodeAllowed).toContain('npm');
      expect(nodeAllowed).toContain('pnpm');
      expect(nodeAllowed).toContain('yarn');
      expect(nodeAllowed).toContain('bun');

      // When switching to Web custom mode, it should inherit node:22-bookworm-slim from active catalog runtime rather than hardcoded node:20
      component.onCatalogKindChange('web');
      component.onSelectCustomMode();
      expect(component.baseImage()).toBe('node:22-bookworm-slim');
    });
  });

  describe('Asistente Copiloto IA', () => {
    it('debe inicializar aiEnabled consultando getStatus en ngOnInit', () => {
      component.ngOnInit();
      expect(mockAiService.getStatus).toHaveBeenCalled();
      expect(component.aiEnabled()).toBe(true);
    });

    it('cuando aiEnabled es true y se invoca generateWithAi, puebla los signals del formulario y aiRationale', () => {
      component.ngOnInit();
      component.aiPrompt.set('PyTorch CPU y scikit-learn para Inteligencia Artificial');

      const mockResponse = {
        component: {
          name: 'PyTorch y Scikit-Learn',
          layer: 'addon' as const,
          description: 'Stack de computación científica y ML para Python',
          icon_key: 'python',
          base_image: '',
          install_mode: 'pip',
          package_names: ['torch', 'torchvision', 'scikit-learn'],
          tools_provided: ['python3'],
          judge_entrypoint: '',
          ram_delta_mb: 512
        },
        rationale: 'Se configuró pip para instalar PyTorch modo CPU y scikit-learn con 512MB RAM adicional recomendada.',
        latency_ms: 240
      };

      mockAiService.suggestComponent.mockReturnValue(of(mockResponse));

      component.generateWithAi();

      expect(mockAiService.suggestComponent).toHaveBeenCalledWith({
        prompt: 'PyTorch CPU y scikit-learn para Inteligencia Artificial',
        layer: 'addon',
        target_runtime: undefined
      });

      expect(component.isCustomPresetSelected()).toBe(true);
      expect(component.name()).toBe('PyTorch y Scikit-Learn');
      expect(component.description()).toBe('Stack de computación científica y ML para Python');
      expect(component.iconKey()).toBe('python');
      expect(component.installMode()).toBe('pip');
      expect(component.installPreset()).toBe('pip');
      expect(component.packageNamesInput()).toBe('torch, torchvision, scikit-learn');
      expect(component.toolsProvidedInput()).toBe('python3');
      expect(component.ramDeltaMB()).toBe(512);
      expect(component.aiRationale()).toBe('Se configuró pip para instalar PyTorch modo CPU y scikit-learn con 512MB RAM adicional recomendada.');
      expect(component.aiLatencyMs()).toBe(240);
      expect(component.showManualFineTune()).toBe(false);
      expect(component.isGeneratingAi()).toBe(false);
    });

    it('cancelAiGeneration cancela la suscripción y detiene la generación', () => {
      component.ngOnInit();
      component.aiPrompt.set('FastAPI');
      mockAiService.suggestComponent.mockReturnValue(of({
        component: { name: 'FastAPI', layer: 'addon' as const },
        rationale: '',
        latency_ms: 100
      }));

      component.generateWithAi();
      component.cancelAiGeneration();
      expect(component.isGeneratingAi()).toBe(false);
    });

    it('switchModalTab cambia la pestaña activa manteniendo visibles los campos del formulario', () => {
      component.name.set('Preset Previo');
      expect(component.aiRationale()).toBeNull();

      component.switchModalTab('ai');
      expect(component.activeModalTab()).toBe('ai');
      expect(component.name()).toBe('Preset Previo');
    });

    it('maneja errores del servicio de IA y activa errorMessage', () => {
      component.ngOnInit();
      component.aiPrompt.set('Tensorflow');

      mockAiService.suggestComponent.mockReturnValue(throwError(() => new Error('Fallo conexión')));

      component.generateWithAi();

      expect(component.errorMessage()).toBe('Error en el servicio de IA');
      expect(component.isGeneratingAi()).toBe(false);
    });
  });
});
