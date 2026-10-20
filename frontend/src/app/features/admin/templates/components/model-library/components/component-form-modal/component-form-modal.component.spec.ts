import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ComponentFormModalComponent } from './component-form-modal.component';
import { AdminTemplatesService } from '../../../../../services/admin-templates.service';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';

describe('ComponentFormModalComponent', () => {
  let fixture: ComponentFixture<ComponentFormModalComponent>;
  let component: ComponentFormModalComponent;
  let tplService: AdminTemplatesService;

  const mockAdminTemplatesService = {
    createTechCatalogItem: vi.fn()
  };

  beforeEach(async () => {
    mockAdminTemplatesService.createTechCatalogItem.mockReset();

    await TestBed.configureTestingModule({
      imports: [ComponentFormModalComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: mockAdminTemplatesService }
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
        id: 'rt-rust',
        slug: 'rust-175',
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
  });

  it('should support package managers (pip, uv, npm, apt, cargo, go, composer, custom)', () => {
    const presets = component.installPresetOptions;
    const ids = presets.map((p: { id: string }) => p.id);
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
        install_mode: 'custom',
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
});
