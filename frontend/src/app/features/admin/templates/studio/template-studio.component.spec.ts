import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { TemplateStudioComponent } from './template-studio.component';
import { AdminTemplatesService } from '../../services/admin-templates.service';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';
import { EnvTestJobService } from '../../services/env-test-job.service';

const MOCK_CATALOG_ITEMS: TemplateComponent[] = [
  {
    id: 'rt-python-312',
    slug: 'python-312',
    layer: 'runtime',
    sub_category: 'backend',
    name: 'Python 3.12',
    description: 'Entorno general, APIs y ciencia de datos',
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
  },
  {
    id: 'rt-cpp-13',
    slug: 'cpp-gcc13',
    layer: 'runtime',
    sub_category: 'backend',
    name: 'C / C++ (GCC 13)',
    description: 'Compiladores gcc/g++ y make para algoritmia',
    icon_key: 'cpp',
    base_image: 'gcc:13.2-bookworm',
    compatible_runtimes: ['cpp'],
    install_mode: 'none',
    package_names: [],
    build_layer_snippet: '',
    setup_snippet: '',
    judge_entrypoint: 'gcc -O2 solution.c -o solution && ./solution',
    tools_provided: ['gcc', 'g++', 'make'],
    service_engine: '',
    ram_delta_mb: 384,
    is_system: true,
    is_active: true,
    sort_order: 50
  },
  {
    id: 'web-angular',
    slug: 'angular-web',
    layer: 'runtime',
    sub_category: 'web',
    name: 'Angular',
    description: 'Interfaz web SPA con Angular CLI y TypeScript',
    icon_key: 'angular',
    base_image: 'node:20-bookworm-slim',
    compatible_runtimes: ['node', 'web'],
    install_mode: 'npm',
    package_names: ['@angular/cli'],
    build_layer_snippet: 'RUN npm install -g --no-audit --no-fund @angular/cli',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: ['node', 'npm', 'ng'],
    service_engine: '',
    ram_delta_mb: 256,
    is_system: true,
    is_active: true,
    sort_order: 110
  },
  {
    id: 'ad-fastapi',
    slug: 'fastapi',
    layer: 'addon',
    sub_category: 'framework',
    name: 'FastAPI + Uvicorn',
    description: 'Creación rápida de APIs REST en Python',
    icon_key: 'fastapi',
    base_image: '',
    compatible_runtimes: ['python'],
    install_mode: 'pip',
    package_names: ['fastapi', 'uvicorn'],
    build_layer_snippet: 'RUN pip install --no-cache-dir fastapi uvicorn',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: ['python3', 'pip'],
    service_engine: '',
    ram_delta_mb: 64,
    is_system: true,
    is_active: true,
    sort_order: 210
  },
  {
    id: 'ad-flask',
    slug: 'flask',
    layer: 'addon',
    sub_category: 'framework',
    name: 'Flask',
    description: 'Micro-framework ligero para aplicaciones web y servicios en Python',
    icon_key: 'flask',
    base_image: '',
    compatible_runtimes: ['python'],
    install_mode: 'pip',
    package_names: ['flask'],
    build_layer_snippet: 'RUN pip install --no-cache-dir flask',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: ['python3', 'pip'],
    service_engine: '',
    ram_delta_mb: 64,
    is_system: true,
    is_active: true,
    sort_order: 230
  },
  {
    id: 'ad-typescript',
    slug: 'typescript-tools',
    layer: 'addon',
    sub_category: 'toolkit',
    name: 'TypeScript + TS-Node',
    description: 'Compilador tsc y ejecución tipada para proyectos web',
    icon_key: 'typescript',
    base_image: '',
    compatible_runtimes: ['node', 'web'],
    install_mode: 'npm',
    package_names: ['typescript', 'ts-node'],
    build_layer_snippet: 'RUN npm install -g --no-audit --no-fund typescript ts-node',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: ['node', 'tsc'],
    service_engine: '',
    ram_delta_mb: 64,
    is_system: true,
    is_active: true,
    sort_order: 250
  },
  {
    id: 'ad-maven',
    slug: 'spring-maven',
    layer: 'addon',
    sub_category: 'framework',
    name: 'Maven + Gestión Java',
    description: 'Construcción de proyectos empresariales en Java',
    icon_key: 'spring',
    base_image: '',
    compatible_runtimes: ['java'],
    install_mode: 'apt',
    package_names: ['maven'],
    build_layer_snippet: 'RUN apt-get update && apt-get install -y --no-install-recommends maven && rm -rf /var/lib/apt/lists/*',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: ['java', 'mvn'],
    service_engine: '',
    ram_delta_mb: 128,
    is_system: true,
    is_active: true,
    sort_order: 260
  },
  {
    id: 'db-none',
    slug: 'db-none',
    layer: 'database',
    sub_category: 'none',
    name: 'Sin Base de Datos',
    description: 'Solo archivos de código en la carpeta del alumno',
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
    sort_order: 300
  },
  {
    id: 'db-mysql',
    slug: 'mysql-84',
    layer: 'database',
    sub_category: 'relational',
    name: 'MySQL 8.4',
    description: 'Base de datos relacional MySQL lista para la práctica',
    icon_key: 'mysql',
    base_image: '',
    compatible_runtimes: ['universal'],
    install_mode: 'none',
    package_names: [],
    build_layer_snippet: '',
    setup_snippet: '',
    judge_entrypoint: '',
    tools_provided: [],
    service_engine: 'mysql',
    ram_delta_mb: 128,
    is_system: true,
    is_active: true,
    sort_order: 330
  }
];

describe('TemplateStudioComponent', () => {
  let component: TemplateStudioComponent;
  let fixture: ComponentFixture<TemplateStudioComponent>;

  beforeEach(async () => {
    const mockTemplatesService = {
      getTechCatalog: vi.fn().mockReturnValue(of(MOCK_CATALOG_ITEMS)),
      getCategories: vi.fn().mockReturnValue(of([{ id: 'cat-1', name: 'Desarrollo Web', created_at: '', updated_at: '' }])),
      getModels: vi.fn().mockReturnValue(of([])),
      getTemplates: vi.fn().mockReturnValue(of([])),
      getRuntimeCapabilities: vi.fn().mockReturnValue(of({
        host_memory: { total_ram_mb: 16384, available_ram_mb: 8192, used_ram_mb: 8192, cpu_cores: 8 },
        satellite_services: [
          {
            category: 'database',
            engine: 'sqlite',
            label: 'SQLite3 (Nativo)',
            version: '3.x',
            description: 'Base de datos embebida en archivo local (/home/workspace). Consumo nulo (+0 MB RAM).',
            env_var: 'SQLITE_PATH',
            is_available: true,
            base_ram_mb: 0
          },
          {
            category: 'database',
            engine: 'postgres',
            label: 'PostgreSQL (Aula)',
            version: '18',
            description: 'Perfil sintonizado para laboratorio (~50 MB RAM) con corte de consultas largas (5s).',
            env_var: 'DATABASE_URL',
            is_available: true,
            base_ram_mb: 50
          },
          {
            category: 'database',
            engine: 'mysql',
            label: 'MySQL',
            version: '8.4',
            description: 'Base de datos relacional MySQL para ejercicios de SQL',
            env_var: 'DATABASE_URL',
            is_available: false,
            base_ram_mb: 128
          }
        ],
        ide_presets: [],
        judge_presets: [],
        max_allowed_ram_mb: 12288,
        editor_base_mb: 210,
        runtime_base_mb: 32
      })),
      getDraft: vi.fn().mockReturnValue(of(null)),
      saveDraft: vi.fn().mockReturnValue(of({})),
      createOfficialTemplate: vi.fn().mockReturnValue(of({ id: 'tpl-new-1' }))
    };

    const mockEnvTestService = {
      startJob: vi.fn(),
      pollJob: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [TemplateStudioComponent],
      providers: [
        provideRouter([]),
        { provide: AdminTemplatesService, useValue: mockTemplatesService },
        { provide: EnvTestJobService, useValue: mockEnvTestService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TemplateStudioComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should initialize in a clean null state without preselected runtime, category, name, or docker image', () => {
    expect(component.selectedRuntimeId()).toBeNull();
    expect(component.selectedRuntimeComponent()).toBeNull();
    expect(component.selectedCategoryId()).toBeNull();
    expect(component.name()).toBe('');
    expect(component.dockerImage()).toBe('');
    expect(component.dockerfileContent()).toBe('');
    expect(component.toolsDeclared()).toBe('');
    expect(component.setupScript()).toBe('');
    expect(component.selectedStackComponents()).toEqual([]);

    const text = (fixture.nativeElement as HTMLElement).textContent || '';
    expect(text).toContain('Seleccione primero un lenguaje principal en la Columna 1 para ver sus complementos compatibles.');
    expect(text).not.toContain('Combinaciones rápidas');
    expect(text).not.toContain('Carpetas iniciales listas para el estudiante');
  });

  it('should filter Column 2 addons dynamically based on selected Column 1 runtime and web platform', () => {
    expect(component.compatibleAddons()).toEqual([]);

    component.selectRuntime('rt-python-312');
    component.selectWebPlatform(null);
    const pyAddons = component.compatibleAddons();
    expect(pyAddons.some(a => a.slug === 'fastapi')).toBe(true);
    expect(pyAddons.some(a => a.slug === 'spring-maven')).toBe(false);

    // When selecting Angular in Web Platform, TypeScript tools (node/web) also become compatible
    component.selectWebPlatform('web-angular');
    const fullstackAddons = component.compatibleAddons();
    expect(fullstackAddons.some(a => a.slug === 'typescript-tools')).toBe(true);
  });

  it('should build a deterministic solv-local/stack:<hash> tag when combining Python + Angular + PostgreSQL and sync with Advanced Mode', () => {
    component.selectRuntime('rt-python-312');
    component.selectWebPlatform('web-angular');
    component.toggleAddon('ad-fastapi');
    component.selectDatabase('db-postgres');

    expect(component.name()).toContain('Python 3.12');
    expect(component.needsLocalBuild()).toBe(true);
    expect(component.dockerImage().startsWith('solv-local/stack:')).toBe(true);
    expect(component.toolsList()).toContain('python3');
    expect(component.toolsList()).toContain('ng');
    expect(component.baseRamMB()).toBeGreaterThanOrEqual(1024);
    expect(component.selectedStackComponents().length).toBe(4);

    // Switching to Advanced Mode preserves the generated values
    component.setStudioMode('code');
    expect(component.dockerfileContent()).toContain('# Generado automáticamente por el Constructor de Entornos SOLV');
    expect(component.dockerfileContent()).toContain('FROM python:3.12-slim-bookworm');
    expect(component.dockerfileContent()).toContain('@angular/cli');
  });

  it('should govern Column 3 database options from node RuntimeCapabilities without a + Nuevo button, enabling SQLite (+0 MB RAM) and PostgreSQL v18 (~50 MB RAM) and disabling unprovisioned engines', () => {
    const hostEl = fixture.nativeElement as HTMLElement;
    const columns = hostEl.querySelectorAll('.studio-column');
    expect(columns.length).toBe(3);

    const col1Btn = columns[0].querySelector('.btn-inline-new');
    const col2Btn = columns[1].querySelector('.btn-inline-new');
    const col3Btn = columns[2].querySelector('.btn-inline-new');

    expect(col1Btn).toBeTruthy();
    expect(col2Btn).toBeTruthy();
    expect(col3Btn).toBeNull();

    const col3Text = columns[2].textContent || '';
    expect(col3Text).toContain('SQLite3 (Nativo) v3.x');
    expect(col3Text).toContain('+0 MB RAM');
    expect(col3Text).toContain('PostgreSQL (Aula) v18');
    expect(col3Text).toContain('~50 MB RAM');
    expect(col3Text).toContain('No habilitado en este nodo');

    const dbCards = Array.from(columns[2].querySelectorAll<HTMLButtonElement>('.tech-component-card'));
    const mysqlCard = dbCards.find(btn => (btn.textContent || '').includes('MySQL'));
    expect(mysqlCard).toBeDefined();
    expect(mysqlCard?.disabled).toBe(true);
    expect(mysqlCard?.classList.contains('disabled')).toBe(true);

    // Attempting to select a disabled engine does not change selectedDatabaseId
    component.selectDatabase('db-mysql');
    expect(component.selectedDatabaseId()).toBe('db-none');
  });

  it('should render clean domain terminology in the UI without culinary metaphors', () => {
    const text = (fixture.nativeElement as HTMLElement).textContent || '';
    expect(text).toContain('Constructor de Entornos');
    expect(text).toContain('2. Arquitectura y Tecnologías del Entorno');
    expect(text).toContain('Modo Visual');
    expect(text).toContain('Modo Avanzado (Dockerfile / Scripts)');
    expect(text).toContain('Resumen del Entorno');
  });

  it('should automatically populate Judge entrypoint and strip web/database layers when purpose is JUEZ_EFIMERO', () => {
    component.selectRuntime('rt-cpp-13');
    component.selectWebPlatform('web-angular');
    component.selectDatabase('db-postgres');

    component.setTargetEnvironment('JUEZ_EFIMERO');
    expect(component.entrypoint()).toContain('gcc -O2');
    expect(component.selectedWebPlatformId()).toBeNull();
    expect(component.selectedDatabaseId()).toBe('db-none');
  });

  it('should leave techComponents empty and show an error toast when getTechCatalog fails without injecting hardcoded fallbacks', async () => {
    TestBed.resetTestingModule();
    await TestBed.configureTestingModule({
      imports: [TemplateStudioComponent],
      providers: [
        provideRouter([]),
        {
          provide: AdminTemplatesService,
          useValue: {
            getTechCatalog: vi.fn().mockReturnValue(throwError(() => new Error('HTTP 500'))),
            getCategories: vi.fn().mockReturnValue(of([])),
            getRuntimeCapabilities: vi.fn().mockReturnValue(of(null))
          }
        },
        { provide: EnvTestJobService, useValue: { startJob: vi.fn(), pollJob: vi.fn() } }
      ]
    }).compileComponents();

    const errorFixture = TestBed.createComponent(TemplateStudioComponent);
    const errorComp = errorFixture.componentInstance;
    errorFixture.detectChanges();

    expect(errorComp.techComponents()).toEqual([]);
    expect(errorComp.toast()?.type).toBe('error');
  });

  it('should keep recommended RAM at 512 MB for Python + FastAPI + Flask (disk libraries) and raise to 768 MB when PostgreSQL Aula is selected (ADR-045)', () => {
    component.selectRuntime('rt-python-312');
    component.toggleAddon('ad-fastapi');
    component.toggleAddon('ad-flask');

    expect(component.recommendedRamMB()).toBe(512);
    expect(component.baseRamMB()).toBe(512);
    expect(component.isRecommendedPreset(512)).toBe(true);

    component.selectDatabase('db-postgres');
    expect(component.recommendedRamMB()).toBe(768);
    expect(component.baseRamMB()).toBe(768);
    expect(component.isRecommendedPreset(768)).toBe(true);
  });

  it('should render the segmented memory-bar with IDE Web (210 MB), database segment, and student code segment', () => {
    component.selectRuntime('rt-python-312');
    component.selectDatabase('db-postgres');
    fixture.detectChanges();

    const hostEl = fixture.nativeElement as HTMLElement;
    const barEl = hostEl.querySelector('.memory-bar');
    expect(barEl).toBeTruthy();

    const barText = barEl?.textContent || '';
    expect(barText).toContain('IDE Web');
    expect(barText).toContain('210 MB');
    expect(barText).toContain('PostgreSQL (Aula)');
    expect(barText).toContain('50 MB');
    expect(barText).toContain('Libre para compilación y código del alumno');
    expect(barText).toContain('508 MB');
  });

  it('should activate safety locks when custom RAM is below minimumFloorMB or above maxAllowedRamMB and block publishTemplate', () => {
    const svc = TestBed.inject(AdminTemplatesService);
    const createSpy = vi.spyOn(svc, 'createOfficialTemplate');

    component.selectRuntime('rt-python-312');
    component.selectDatabase('db-postgres');
    // Piso vital: 210 (IDE) + 174 (Python) + 50 (PostgreSQL) = 434 MB
    expect(component.minimumFloorMB()).toBe(434);

    component.onCustomRamChange(256);
    fixture.detectChanges();
    expect(component.isRamBelowFloor()).toBe(true);
    expect(component.isRamExceedingHost()).toBe(false);

    component.publishTemplate();
    expect(createSpy).not.toHaveBeenCalled();

    component.onCustomRamChange(16384);
    fixture.detectChanges();
    expect(component.isRamBelowFloor()).toBe(false);
    expect(component.isRamExceedingHost()).toBe(true);

    component.publishTemplate();
    expect(createSpy).not.toHaveBeenCalled();
  });

  it('should include resource_profile { min_mb, high_mb, max_mb } in buildCreatePayload()', () => {
    component.selectRuntime('rt-python-312');
    component.selectDatabase('db-postgres');

    const payload = component.buildCreatePayload();
    expect(payload.resource_profile).toEqual({
      min_mb: 434,
      high_mb: Math.max(434, Math.floor(768 * 0.85)),
      max_mb: 768
    });
  });
});
