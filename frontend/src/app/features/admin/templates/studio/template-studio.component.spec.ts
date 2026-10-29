import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { TemplateStudioComponent } from './template-studio.component';
import { AdminTemplatesService } from '../../services/admin-templates.service';
import { AdminConfigIaService } from '../../configuracion/admin-config-ia.service';
import { AdminTemplateItem, TemplateComponent } from '@features/admin/models/admin-templates.models';
import { EnvTestJobService } from '../../services/env-test-job.service';
import { ComponentFormModalComponent } from '../components/model-library/components/component-form-modal/component-form-modal.component';
import { classifyAddonComponent } from './template-studio.utils';
import {
  OFFICIAL_ADDON_PRESETS,
  OFFICIAL_WEB_PROFILES
} from '../components/model-library/components/component-form-modal/component-catalog.presets';

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
    id: 'rt-java-21',
    slug: 'java-21',
    layer: 'runtime',
    sub_category: 'backend',
    name: 'Java 21 (Temurin)',
    description: 'Entorno OpenJDK con Maven y Gradle',
    icon_key: 'java',
    base_image: 'eclipse-temurin:21-jdk-bookworm',
    compatible_runtimes: ['java'],
    install_mode: 'none',
    package_names: [],
    build_layer_snippet: '',
    setup_snippet: '',
    judge_entrypoint: 'javac Solution.java && java Solution',
    tools_provided: ['javac', 'java'],
    service_engine: '',
    ram_delta_mb: 1024,
    is_system: true,
    is_active: true,
    sort_order: 60
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
            description: 'Base de datos relacional MySQL auto-orquestada por estudiante',
            env_var: 'DATABASE_URL',
            is_available: true,
            base_ram_mb: 128,
            provisioning_mode: 'managed_container',
            default_image: 'mysql:8.4'
          },
          {
            category: 'database',
            engine: 'mariadb',
            label: 'MariaDB',
            version: '11.4',
            description: 'Alternativa relacional compatible con MySQL, auto-orquestada',
            env_var: 'DATABASE_URL',
            is_available: true,
            base_ram_mb: 128,
            provisioning_mode: 'managed_container',
            default_image: 'mariadb:11.4'
          },
          {
            category: 'database',
            engine: 'mongodb',
            label: 'MongoDB',
            version: '7.0',
            description: 'Base de datos documental auto-orquestada',
            env_var: 'MONGODB_URI',
            is_available: true,
            base_ram_mb: 128,
            provisioning_mode: 'managed_container',
            default_image: 'mongo:7.0'
          },
          {
            category: 'cache',
            engine: 'redis',
            label: 'Redis',
            version: '7.2',
            description: 'Almacén en memoria auto-orquestado para caché y colas',
            env_var: 'REDIS_URL',
            is_available: true,
            base_ram_mb: 64,
            provisioning_mode: 'managed_container',
            default_image: 'redis:7.2-alpine'
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
      getTemplateById: vi.fn().mockReturnValue(of(null)),
      updateOfficialTemplate: vi.fn().mockReturnValue(of({})),
      createOfficialTemplate: vi.fn().mockReturnValue(of({ id: 'tpl-new-1' }))
    };

    const mockEnvTestService = {
      startJob: vi.fn(),
      pollJob: vi.fn(),
      cancelJob: vi.fn().mockReturnValue(of({ status: 'canceled' })),
      isTerminal: vi.fn((status: string) => status === 'success' || status === 'failed' || status === 'canceled')
    };

    const mockAiService = {
      getStatus: vi.fn().mockReturnValue(of({ enabled: true, provider: 'gemini', model: 'gemini-2.5-flash' })),
      suggestTemplate: vi.fn().mockReturnValue(of({
        name: 'Python Fast Stack',
        description: 'Plantilla sugerida por IA',
        runtime_slug: 'python-312',
        web_platform_slug: '',
        addon_slugs: [],
        database_slug: 'sqlite',
        setup_script: 'pip install -r requirements.txt',
        base_ram_mb: 512,
        rationale: 'Configuración generada para Python y SQLite',
        latency_ms: 180
      })),
      suggestComponent: vi.fn(),
      repairTemplate: vi.fn(),
      resolveError: vi.fn().mockReturnValue('Error de IA')
    };

    await TestBed.configureTestingModule({
      imports: [TemplateStudioComponent],
      providers: [
        provideRouter([]),
        { provide: AdminTemplatesService, useValue: mockTemplatesService },
        { provide: EnvTestJobService, useValue: mockEnvTestService },
        { provide: AdminConfigIaService, useValue: mockAiService }
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

  it('should govern Column 3 database engines from the node capabilities and the active catalog, with + Nuevo opening the modal', () => {
    const hostEl = fixture.nativeElement as HTMLElement;
    const columns = hostEl.querySelectorAll('.studio-column');
    expect(columns.length).toBe(3);

    const col1Btn = columns[0].querySelector('.btn-inline-new');
    const col2Btn = columns[1].querySelector('.btn-inline-new');
    const col3Btn = columns[2].querySelector<HTMLButtonElement>('.btn-inline-new');

    expect(col1Btn).toBeTruthy();
    expect(col2Btn).toBeTruthy();
    expect(col3Btn).toBeTruthy();
    expect(col3Btn?.textContent).toContain('Nuevo');

    col3Btn?.click();
    fixture.detectChanges();
    expect(component.showComponentModal()).toBe(true);
    expect(component.modalDefaultLayer()).toBe('database');

    const modalDebugEl = fixture.debugElement.query(By.directive(ComponentFormModalComponent));
    expect(modalDebugEl).toBeTruthy();
    expect((modalDebugEl.componentInstance as ComponentFormModalComponent).catalogKind()).toBe('database');

    component.handleModalClosed();
    fixture.detectChanges();

    const col3Text = columns[2].textContent || '';
    expect(col3Text).toContain('Sin Base de Datos');
    expect(col3Text).toContain('SQLite3 (Nativo) v3.x');
    expect(col3Text).toContain('+0 MB RAM');
    expect(col3Text).toContain('PostgreSQL (Aula) v18');
    expect(col3Text).toContain('~50 MB RAM');
    expect(col3Text).not.toContain('No habilitado en este nodo');

    const dbCards = Array.from(columns[2].querySelectorAll<HTMLButtonElement>('.tech-component-card'));
    expect(dbCards.length).toBe(7);

    // Los motores gestionados se ofrecen con su impacto en RAM y son seleccionables
    expect(col3Text).toContain('MySQL v8.4');
    expect(col3Text).toContain('~128 MB RAM');
    expect(col3Text).toContain('SQLite3 (Nativo) v3.x');
    component.selectDatabase('db-mysql');
    expect(component.selectedDatabaseId()).toBe('db-mysql');
    expect(component.selectedDatabaseOption()?.service_engine).toBe('mysql');

    // Un motor desactivado en el catálogo deja de ofrecerse aunque el nodo lo reporte
    component.techComponents.update(list =>
      list.map(item => (item.id === 'db-mysql' ? { ...item, is_active: false } : item))
    );
    const enginesAfterDeactivation = component.databaseOptions();
    expect(enginesAfterDeactivation.some(d => d.service_engine === 'mysql')).toBe(false);
    expect(enginesAfterDeactivation.some(d => d.service_engine === 'postgres')).toBe(true);
  });

  it('should render catalog cards as compact native toggle buttons and show description in contextual inspector when selected', () => {
    const hostEl = fixture.nativeElement as HTMLElement;
    const firstColumn = hostEl.querySelectorAll('.studio-column')[0] as HTMLElement;

    let cards = Array.from(firstColumn.querySelectorAll<HTMLElement>('.tech-component-card'));
    expect(cards.length).toBeGreaterThan(0);
    cards.forEach(card => expect(card.tagName).toBe('BUTTON'));

    // Sin selección previa, todas informan aria-pressed=false
    expect(cards.every(card => card.getAttribute('aria-pressed') === 'false')).toBe(true);

    // En reposo, la tarjeta es compacta y ofrece la descripción en el title accesible
    const pythonCard = cards.find(card => card.textContent?.includes('Python 3.12'));
    expect(pythonCard).toBeTruthy();
    expect(pythonCard?.getAttribute('title')).toContain('ciencia de datos');

    // El control de edición es hermano de la tarjeta: nunca hay botones anidados
    expect(firstColumn.querySelector('.tech-component-card .comp-edit-btn')).toBeNull();
    expect(firstColumn.querySelector('.tech-card-wrap > .comp-edit-btn')).toBeTruthy();

    pythonCard?.click();
    fixture.detectChanges();

    cards = Array.from(firstColumn.querySelectorAll<HTMLElement>('.tech-component-card'));
    const pressed = cards.filter(card => card.getAttribute('aria-pressed') === 'true');
    expect(pressed).toHaveLength(1);
    expect(pressed[0].textContent).toContain('Python 3.12');

    // Al seleccionar, la franja de inspección contextual muestra la descripción técnica
    const inspectorStrip = firstColumn.querySelector('.tech-inspector-strip');
    expect(inspectorStrip).toBeTruthy();
    expect(inspectorStrip?.querySelector('.inspector-desc')?.textContent).toContain('ciencia de datos');
  });

  it('should open and close the scope info modal via scope-chip-btn in AMBOS mode', () => {
    component.targetEnvironment.set('AMBOS');
    fixture.detectChanges();

    const hostEl = fixture.nativeElement as HTMLElement;
    const scopeBtn = hostEl.querySelector<HTMLButtonElement>('.scope-chip-btn');
    expect(scopeBtn).toBeTruthy();
    expect(scopeBtn?.textContent).toContain('Modo Dual');

    // Inicialmente cerrado
    expect(hostEl.querySelector('.scope-modal-body')).toBeNull();

    scopeBtn?.click();
    fixture.detectChanges();

    // Abierto
    expect(hostEl.querySelector('.scope-modal-body')).toBeTruthy();
    expect(hostEl.querySelector('.scope-modal-body')?.textContent).toContain('stdin');

    // Cerrar
    component.closeScopeInfoModal();
    fixture.detectChanges();
    expect(hostEl.querySelector('.scope-modal-body')).toBeNull();
  });

  it('should offer catalog-registered database engines that the node does not report yet', () => {
    const registeredEngine = {
      id: 'db-redis-72',
      slug: 'redis-72',
      layer: 'database' as const,
      sub_category: 'nosql',
      name: 'Redis 7.2 (Caché)',
      description: 'Caché auto-orquestada',
      icon_key: 'redis',
      base_image: 'redis:7.2-alpine',
      compatible_runtimes: ['universal'],
      install_mode: 'none' as const,
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: [],
      service_engine: 'redis',
      ram_delta_mb: 64,
      is_system: false,
      is_active: true,
      sort_order: 340
    };

    component.techComponents.update(list => [
      ...list.filter(item => item.service_engine !== 'redis'),
      registeredEngine
    ]);
    component.runtimeCapabilities.update(caps =>
      caps
        ? { ...caps, satellite_services: caps.satellite_services.filter(sat => sat.engine !== 'redis') }
        : caps
    );

    const options = component.databaseOptions();
    const redisOption = options.find(d => d.service_engine === 'redis');
    expect(redisOption).toBeTruthy();
    expect(redisOption?.id).toBe('db-redis-72');
    expect(redisOption?.is_available).toBe(true);

    component.selectDatabase('db-redis-72');
    expect(component.selectedDatabaseId()).toBe('db-redis-72');
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
    expect(component.capacityOptions().find(o => o.mb === 512)?.isRecommended).toBe(true);

    component.selectDatabase('db-postgres');
    expect(component.recommendedRamMB()).toBe(768);
    expect(component.baseRamMB()).toBe(768);
    expect(component.capacityOptions().find(o => o.mb === 768)?.isRecommended).toBe(true);
  });

  it('should render the 3-segment memory bar exactly matching editor + stack + usable space', () => {
    component.selectRuntime('rt-python-312');
    component.selectDatabase('db-postgres');
    fixture.detectChanges();

    const hostEl = fixture.nativeElement as HTMLElement;
    const barEl = hostEl.querySelector('.memory-bar');
    expect(barEl).toBeTruthy();

    const barText = barEl?.textContent || '';
    expect(barText).toContain('Editor de código');
    expect(barText).toContain('210 MB');
    expect(barText).toContain('Tecnologías elegidas');
    expect(barText).toContain('Python 3.12 + PostgreSQL');
    expect(barText).toContain('224 MB');
    expect(barText).toContain('Espacio libre para los programas del alumno');
    expect(barText).toContain('334 MB');

    const breakdown = component.resourceBreakdown();
    expect(breakdown.editorBaseMB + breakdown.stackBaseMB + breakdown.usableMB).toBe(component.baseRamMB());
  });

  it('should show empty pending state in section 3 and sidebar when no runtime is selected', () => {
    expect(component.selectedRuntimeId()).toBeNull();
    expect(component.hasSelectedStack()).toBe(false);
    expect(component.capacityOptions()).toEqual([]);
    fixture.detectChanges();

    const hostEl = fixture.nativeElement as HTMLElement;
    const emptyStateEl = hostEl.querySelector('.capacity-empty-state');
    expect(emptyStateEl).toBeTruthy();
    expect(emptyStateEl?.textContent).toContain('Pendiente de selección tecnológica');
    expect(emptyStateEl?.textContent).toContain('Seleccione primero el lenguaje principal en el Paso 2');
    expect(hostEl.querySelector('.capacity-card')).toBeNull();

    // Panel derecho
    expect(hostEl.textContent).toContain('Pendiente de selección');
    expect(hostEl.textContent).toContain('Seleccione tecnologías en el Paso 2');
  });

  it('should calculate 2 academic profile options without fixed language names and without double counting postgresql RAM', () => {
    component.selectRuntime('rt-python-312');
    component.selectDatabase('db-postgres');
    fixture.detectChanges();

    expect(component.hasSelectedStack()).toBe(true);
    const options = component.capacityOptions();
    expect(options.length).toBe(2);

    // Opción 1: Estándar
    expect(options[0].mb).toBe(768);
    expect(options[0].title).toBe('Prácticas y Ejercicios de Clase');
    expect(options[0].isRecommended).toBe(true);
    expect(options[0].title).not.toContain('Python');
    expect(options[0].title).not.toContain('C / C++');
    expect(options[0].title).not.toContain('Java');

    // Opción 2: Holgura
    expect(options[1].mb).toBe(1024);
    expect(options[1].title).toBe('Proyectos Integradores y Trabajos Pesados');
    expect(options[1].isRecommended).toBe(false);

    // Capacidad sin duplicar dbExtra (+50 MB)
    // maxAllowedRamMB = 16384 * 0.75 = 12288
    // baseRamMB = 768 -> nominalConcurrentStudents = Math.floor(12288 / 768) = 16
    expect(component.nominalConcurrentStudents()).toBe(16);
    expect(component.estimateStudentsForPreset(768)).toBe(16);
    // Si hubiera sumado 768 + 50 = 818, 12288 / 818 daría 15
    expect(component.estimateStudentsForPreset(768)).not.toBe(15);

    // Suma exacta en resourceBreakdown
    const breakdown = component.resourceBreakdown();
    expect(breakdown.editorBaseMB).toBe(210);
    expect(breakdown.stackBaseMB).toBe(224);
    expect(breakdown.usableMB).toBe(334);
    expect(breakdown.editorBaseMB + breakdown.stackBaseMB + breakdown.usableMB).toBe(768);
  });

  it('should auto-elevate baseRamMB when switching to a heavier runtime whose floor exceeds previous allocation', () => {
    component.selectRuntime('rt-cpp-13');
    expect(component.baseRamMB()).toBe(384);
    expect(component.minimumFloorMB()).toBe(338);

    // Cambiar a Java cuyo piso vital (710 MB) supera los 384 MB previos
    component.selectRuntime('rt-java-21');
    expect(component.minimumFloorMB()).toBe(710);
    expect(component.recommendedRamMB()).toBe(1024);
    expect(component.baseRamMB()).toBe(1024);
    expect(component.isRamManuallyEdited()).toBe(false);
  });

  it('should render a single capacity card when intensiveRamMB equals recommendedRamMB to avoid duplicate tracking keys', () => {
    component.selectRuntime('rt-java-21');
    component.toggleAddon('ad-maven');
    if (component.recommendedRamMB() === 2048) {
      expect(component.intensiveRamMB()).toBe(2048);
      const options = component.capacityOptions();
      expect(options.length).toBe(1);
      expect(options[0].mb).toBe(2048);
    }
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

  it('should not render redundant + Registrar Nueva Tecnología button in header', () => {
    const hostEl = fixture.nativeElement as HTMLElement;
    const headerActions = hostEl.querySelector('.studio-header .header-actions');
    expect(headerActions).toBeNull();
  });

  it('should provide combobox categoryOptions for category selector', () => {
    const options = component.categoryOptions();
    expect(options.length).toBe(1);
    expect(options[0].id).toBe('cat-1');
    expect(options[0].label).toBe('Desarrollo Web');
  });

  it('should keep Dockerfile editor visible in code mode and protect manual custom edits', () => {
    component.selectRuntime('rt-python-312');
    component.setStudioMode('code');
    fixture.detectChanges();

    const hostEl = fixture.nativeElement as HTMLElement;
    const editor = hostEl.querySelector('#code-dockerfile-editor');
    expect(editor).toBeTruthy();

    // Editar manualmente el Dockerfile
    component.onDockerfileManualChange('FROM python:3.12-slim-bookworm\nRUN pip install numpy\n');
    expect(component.hasCustomDockerfile()).toBe(true);
    expect(component.needsLocalBuild()).toBe(true);
    expect(component.dockerImage()).toContain('solv-local/stack:');

    // Cambiar complementos en visual no debe sobreescribir el Dockerfile manual
    component.toggleAddon('ad-fastapi');
    expect(component.dockerfileContent()).toBe('FROM python:3.12-slim-bookworm\nRUN pip install numpy\n');

    // Restablecer desde visual restaura la sincronización
    component.resetDockerfileFromVisual();
    expect(component.hasCustomDockerfile()).toBe(false);
  });

  it('should diagnose Dockerfile errors (missing FROM, latest/untagged, apt without -y)', () => {
    // 1. Falta FROM
    component.onDockerfileManualChange('RUN echo hello');
    expect(component.hasBlockingDockerfileError()).toBe(true);
    expect(component.dockerfileDiagnostics().some(d => d.severity === 'error' && d.message.includes('FROM'))).toBe(true);

    // 2. Uso de :latest o sin tag
    component.onDockerfileManualChange('FROM python:latest');
    expect(component.hasBlockingDockerfileError()).toBe(true);
    expect(component.dockerfileDiagnostics().some(d => d.severity === 'error' && d.message.includes(':latest'))).toBe(true);

    // 3. Advertencia apt-get install sin -y
    component.onDockerfileManualChange('FROM python:3.12-slim-bookworm\nRUN apt-get install curl');
    expect(component.hasBlockingDockerfileError()).toBe(false);
    expect(component.dockerfileDiagnostics().some(d => d.severity === 'warning' && d.message.includes('-y'))).toBe(true);

    // 4. Dockerfile válido y limpio
    component.onDockerfileManualChange('FROM python:3.12-slim-bookworm\nRUN apt-get install -y curl');
    expect(component.hasBlockingDockerfileError()).toBe(false);
    expect(component.dockerfileDiagnostics()).toEqual([]);
  });

  it('should block verification and publication when Dockerfile has blocking errors', () => {
    const envSvc = TestBed.inject(EnvTestJobService);
    const startJobSpy = vi.spyOn(envSvc, 'startJob');
    const tplSvc = TestBed.inject(AdminTemplatesService);
    const createSpy = vi.spyOn(tplSvc, 'createOfficialTemplate');

    component.onNameInput('Lab Test Error');
    component.setStudioMode('code');
    component.onDockerfileManualChange('RUN echo missing-from');

    component.runVerificationTest();
    expect(startJobSpy).not.toHaveBeenCalled();

    component.publishTemplate();
    expect(createSpy).not.toHaveBeenCalled();
  });

  it('should block publication when image requires verification until env test succeeds', () => {
    const envSvc = TestBed.inject(EnvTestJobService);
    const tplSvc = TestBed.inject(AdminTemplatesService);
    const createSpy = vi.spyOn(tplSvc, 'createOfficialTemplate');

    component.selectRuntime('rt-python-312');
    component.selectWebPlatform('web-angular');
    component.onNameInput('Lab Python + Angular');

    expect(component.needsLocalBuild()).toBe(true);
    expect(component.requiresVerificationBeforePublish()).toBe(true);

    // Intentar publicar sin verificar debe ser bloqueado
    component.publishTemplate();
    expect(createSpy).not.toHaveBeenCalled();

    // El job de comprobación termina en éxito con la configuración actual
    const successJob = {
      id: 'job-1',
      image: 'solv-local/stack:test',
      tools: [],
      status: 'success' as const,
      finished_at: new Date().toISOString(),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      digest_unverified: false,
      progress: {
        percent: 100,
        current_action: 'Completado',
        bytes_done: 0,
        bytes_total: 0,
        layer_current: 0,
        layers_total: 0
      }
    };
    vi.spyOn(envSvc, 'startJob').mockReturnValue(of(successJob));
    vi.spyOn(envSvc, 'pollJob').mockReturnValue(of(successJob));

    component.runVerificationTest();

    expect(component.activeEnvTestJob()?.status).toBe('success');
    expect(component.verifiedFingerprint()).toBe(component.currentVerificationFingerprint());
    expect(component.requiresVerificationBeforePublish()).toBe(false);
  });

  it('should inject multi-stage official Node.js LTS when web platform is added to non-Node runtime', () => {
    component.selectRuntime('rt-python-312');
    component.selectWebPlatform('web-angular');

    expect(component.dockerfileContent()).toContain('COPY --from=node:20-bookworm-slim /usr/local /usr/local');
    expect(component.dockerfileContent()).not.toContain('apt-get install -y --no-install-recommends nodejs npm');
  });

  it('should detect configuration modifications after successful verification and block publication', () => {
    const tplSvc = TestBed.inject(AdminTemplatesService);
    const createSpy = vi.spyOn(tplSvc, 'createOfficialTemplate');
    const envSvc = TestBed.inject(EnvTestJobService);

    component.selectRuntime('rt-python-312');
    component.onNameInput('Lab Python Test');

    const successJob = {
      id: 'job-verified',
      image: 'python:3.12-slim-bookworm',
      tools: ['python3', 'pip'],
      status: 'success' as const,
      finished_at: new Date().toISOString(),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      digest_unverified: false,
      progress: {
        percent: 100,
        current_action: 'Completado',
        bytes_done: 0,
        bytes_total: 0,
        layer_current: 0,
        layers_total: 0
      }
    };

    vi.spyOn(envSvc, 'startJob').mockReturnValue(of(successJob));
    vi.spyOn(envSvc, 'pollJob').mockReturnValue(of(successJob));

    // Ejecutar verificación atómica
    component.runVerificationTest();

    expect(component.verifiedFingerprint()).toBe(component.currentVerificationFingerprint());
    expect(component.isConfigModifiedAfterVerification()).toBe(false);

    // Modificar un parámetro (ej. RAM)
    component.selectRamPreset(768);

    expect(component.isConfigModifiedAfterVerification()).toBe(true);

    // Intentar publicar con configuración desfasada debe ser bloqueado
    component.publishTemplate();
    expect(createSpy).not.toHaveBeenCalled();
  });

  it('should cancel active test job via cancelEnvTest', () => {
    const envSvc = TestBed.inject(EnvTestJobService);
    const cancelSpy = vi.spyOn(envSvc, 'cancelJob').mockReturnValue(of({ status: 'canceled' }));

    component.activeEnvTestJob.set({
      id: 'job-to-cancel',
      image: 'test:1.0',
      tools: [],
      status: 'testing',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      digest_unverified: false,
      progress: {
        percent: 45,
        current_action: 'Descargando capas...',
        bytes_done: 0,
        bytes_total: 0,
        layer_current: 0,
        layers_total: 0
      }
    });
    component.isVerifying.set(true);

    component.cancelEnvTest();

    expect(cancelSpy).toHaveBeenCalledWith('job-to-cancel');
    expect(component.isVerifying()).toBe(false);
  });

  it('should invalidate fingerprint when toolsDeclared or targetEnvironment changes after verification', () => {
    const envSvc = TestBed.inject(EnvTestJobService);
    component.selectRuntime('rt-python-312');

    const successJob = {
      id: 'job-verified-2',
      image: 'python:3.12-slim-bookworm',
      tools: ['python3', 'pip'],
      status: 'success' as const,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      digest_unverified: false,
      progress: {
        percent: 100,
        current_action: 'Completado',
        bytes_done: 0,
        bytes_total: 0,
        layer_current: 0,
        layers_total: 0
      }
    };

    vi.spyOn(envSvc, 'startJob').mockReturnValue(of(successJob));
    vi.spyOn(envSvc, 'pollJob').mockReturnValue(of(successJob));

    component.runVerificationTest();
    expect(component.isConfigModifiedAfterVerification()).toBe(false);

    // 1. Mutar toolsDeclared
    const initialTools = component.toolsDeclared();
    component.toolsDeclared.set(initialTools + ', pytest');
    expect(component.isConfigModifiedAfterVerification()).toBe(true);

    // Restaurar toolsDeclared
    component.toolsDeclared.set(initialTools);
    expect(component.isConfigModifiedAfterVerification()).toBe(false);

    // 2. Cambiar targetEnvironment
    component.setTargetEnvironment('JUEZ_EFIMERO');
    expect(component.isConfigModifiedAfterVerification()).toBe(true);
  });

  it('should preserve activeEnvTestJob if status was success when editing RAM or Dockerfile', () => {
    const envSvc = TestBed.inject(EnvTestJobService);
    component.selectRuntime('rt-python-312');

    const successJob = {
      id: 'job-verified-3',
      image: 'python:3.12-slim-bookworm',
      tools: ['python3', 'pip'],
      status: 'success' as const,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      digest_unverified: false,
      progress: {
        percent: 100,
        current_action: 'Completado',
        bytes_done: 0,
        bytes_total: 0,
        layer_current: 0,
        layers_total: 0
      }
    };

    vi.spyOn(envSvc, 'startJob').mockReturnValue(of(successJob));
    vi.spyOn(envSvc, 'pollJob').mockReturnValue(of(successJob));

    component.runVerificationTest();
    expect(component.activeEnvTestJob()?.status).toBe('success');

    // Modificar Dockerfile manualmente no debe anular el job si era success
    component.onDockerfileManualChange('FROM python:3.12-slim-bookworm\nRUN echo "custom"');
    expect(component.activeEnvTestJob()).not.toBeNull();
    expect(component.activeEnvTestJob()?.status).toBe('success');
    expect(component.isConfigModifiedAfterVerification()).toBe(true);

    // Modificar RAM tampoco debe anular el job si era success
    component.selectRamPreset(1024);
    expect(component.activeEnvTestJob()).not.toBeNull();
    expect(component.activeEnvTestJob()?.status).toBe('success');
  });

  it('should restore dockerfile_content, mode, custom scripts and local build flags in applyTemplateToStudio', () => {
    const customTemplate: AdminTemplateItem = {
      id: 'tpl-custom-audit',
      name: 'Entorno Python Custom',
      docker_image: 'solv-local/stack:1234567890ab',
      base_ram_mb: 768,
      status: 'approved',
      created_at: new Date().toISOString(),
      description: 'Descripción personalizada',
      target_environment: 'IDE_PERSISTENTE',
      setup_script: 'echo "setup custom script"',
      tools_declared: ['python3', 'pip', 'pytest', 'black'],
      entrypoint: '',
      timeout_ms: 6000,
      stack_config: {
        mode: 'code',
        runtime_id: 'rt-python-312',
        dockerfile_content: 'FROM python:3.12-slim-bookworm\nRUN pip install pytest black'
      }
    };

    (component as unknown as { applyTemplateToStudio: (tpl: AdminTemplateItem, dup: boolean) => void })
      .applyTemplateToStudio(customTemplate, false);

    expect(component.studioMode()).toBe('code');
    expect(component.dockerfileContent()).toBe('FROM python:3.12-slim-bookworm\nRUN pip install pytest black');
    expect(component.hasCustomDockerfile()).toBe(true);
    expect(component.needsLocalBuild()).toBe(true);
    expect(component.dockerImage()).toBe('solv-local/stack:1234567890ab');
    expect(component.setupScript()).toBe('echo "setup custom script"');
    expect(component.toolsDeclared()).toContain('pytest');
    expect(component.baseRamMB()).toBe(768);
  });

  it('should re-apply loaded template when catalog finishes after template is fetched', () => {
    const tplSvc = TestBed.inject(AdminTemplatesService);
    const customTemplate: AdminTemplateItem = {
      id: 'tpl-custom-race',
      name: 'Python Post Catalog',
      docker_image: 'solv-local/stack:abcdef123456',
      base_ram_mb: 1024,
      status: 'approved',
      created_at: new Date().toISOString(),
      target_environment: 'IDE_PERSISTENTE',
      setup_script: 'echo "custom script before catalog"',
      tools_declared: ['python3', 'pip', 'flake8'],
      stack_config: {
        mode: 'code',
        runtime_id: 'rt-python-312',
        dockerfile_content: 'FROM python:3.12-slim-bookworm\nRUN pip install flake8'
      }
    };

    vi.spyOn(tplSvc, 'getTemplateById').mockReturnValue(of(customTemplate));
    vi.spyOn(tplSvc, 'getTechCatalog').mockReturnValue(of(MOCK_CATALOG_ITEMS));

    // Simular carga de plantilla primero
    (component as unknown as { loadExistingTemplate: (id: string, dup: boolean) => void })
      .loadExistingTemplate('tpl-custom-race', false);

    // Luego simular resolución tardía del catálogo
    (component as unknown as { loadInitialCatalog: () => void }).loadInitialCatalog();

    // Comprobar que no se perdieron los datos personalizados al resolver el catálogo
    expect(component.studioMode()).toBe('code');
    expect(component.dockerfileContent()).toBe('FROM python:3.12-slim-bookworm\nRUN pip install flake8');
    expect(component.hasCustomDockerfile()).toBe(true);
    expect(component.setupScript()).toBe('echo "custom script before catalog"');
    expect(component.toolsDeclared()).toContain('flake8');
    expect(component.baseRamMB()).toBe(1024);
  });

  it('should support Solo Frontend composition with 1024 MB RAM, web base image, and verification enabled', () => {
    // Solo seleccionar plataforma web (sin runtime backend)
    component.selectWebPlatform('web-angular');

    expect(component.selectedRuntimeComponent()).toBeNull();
    expect(component.selectedWebComponent()?.id).toBe('web-angular');
    expect(component.effectivePrimaryComponent()?.id).toBe('web-angular');
    expect(component.hasSelectedStack()).toBe(true);

    // RAM recomendada: 640 MB proceso activo + 210 MB base editor = 850 MB -> 1024 MB tier
    expect(component.recommendedRamMB()).toBe(1024);
    expect(component.baseRamMB()).toBe(1024);

    // Dockerfile usa la imagen base de la web y su capa de build sin inyección redundante de Node
    expect(component.dockerfileContent()).toContain('FROM node:20-bookworm-slim');
    expect(component.dockerfileContent()).toContain('RUN npm install -g --no-audit --no-fund @angular/cli');
    expect(component.dockerfileContent()).not.toContain('COPY --from=');
    expect(component.dockerfileContent()).not.toContain('apt-get');

    // Nombre sugerido automático
    expect(component.name()).toBe('Laboratorio: Angular');
    expect(component.stackSummaryLabel()).toBe('Angular');

    // Herramientas declaradas coinciden con web (sin git fantasma)
    expect(component.toolsDeclared()).toBe('node, npm, ng');

    // Complementos compatibles habilitados para web/node
    const addons = component.compatibleAddons();
    expect(addons.some(a => a.id === 'ad-typescript')).toBe(true);

    // Verificación habilitada sin error de "seleccione un lenguaje principal"
    const envSvc = TestBed.inject(EnvTestJobService);
    const soloJob = {
      id: 'job-solo-front',
      image: component.dockerImage(),
      tools: component.toolsList(),
      target_environment: 'IDE_PERSISTENTE' as const,
      base_ram_mb: 1024,
      status: 'pending' as const,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      digest_unverified: false,
      progress: {
        percent: 0,
        current_action: 'Iniciando',
        bytes_done: 0,
        bytes_total: 0,
        layer_current: 0,
        layers_total: 0
      }
    };
    const startJobSpy = vi.spyOn(envSvc, 'startJob').mockReturnValue(of(soloJob));
    vi.spyOn(envSvc, 'pollJob').mockReturnValue(of(soloJob));

    component.runVerificationTest();
    expect(startJobSpy).toHaveBeenCalled();
    expect(component.toast()?.type).not.toBe('error');
  });

  it('should support toggling runtime selection on and off', () => {
    // 1. Seleccionar runtime
    component.selectRuntime('rt-python-312');
    expect(component.selectedRuntimeId()).toBe('rt-python-312');
    expect(component.selectedRuntimeComponent()?.id).toBe('rt-python-312');

    // 2. Hacer toggle (clic sobre el mismo runtime lo deselecciona)
    component.selectRuntime('rt-python-312');
    expect(component.selectedRuntimeId()).toBeNull();
    expect(component.selectedRuntimeComponent()).toBeNull();
    expect(component.hasSelectedStack()).toBe(false);

    // 3. Con web seleccionada, deseleccionar runtime transiciona limpiamente a Solo Frontend
    component.selectRuntime('rt-python-312');
    component.selectWebPlatform('web-angular');
    expect(component.effectivePrimaryComponent()?.id).toBe('rt-python-312');

    // Deseleccionar runtime
    component.selectRuntime('rt-python-312');
    expect(component.selectedRuntimeId()).toBeNull();
    expect(component.effectivePrimaryComponent()?.id).toBe('web-angular');
    expect(component.baseRamMB()).toBe(1024);
    expect(component.name()).toBe('Laboratorio: Angular');
  });

  it('should not contain ghost git in toolsDeclared when not declared by components', () => {
    component.selectRuntime('rt-python-312');
    // rt-python-312 declara ['python3', 'pip']
    expect(component.toolsDeclared()).toBe('python3, pip');
    expect(component.toolsList()).toEqual(['python3', 'pip']);
    expect(component.toolsList()).not.toContain('git');
  });

  it('should support hot-editing tech components in studio and recalculate active selection', () => {
    component.selectRuntime('rt-python-312');
    expect(component.name()).toBe('Laboratorio: Python 3.12');
    expect(component.baseRamMB()).toBe(512);

    // Abrir modal de edición
    const currentComp = component.selectedRuntimeComponent()!;
    component.openEditComponentModal(currentComp);
    expect(component.editingComponent()?.id).toBe('rt-python-312');
    expect(component.showComponentModal()).toBe(true);

    // Guardar componente modificado
    const modifiedComp: TemplateComponent = {
      ...currentComp,
      name: 'Python 3.12 Actualizado',
      tools_provided: ['python3', 'pip', 'poetry']
    };

    component.handleModalComponentSaved(modifiedComp);

    // Modal se cierra y editingComponent se resetea
    expect(component.showComponentModal()).toBe(false);
    expect(component.editingComponent()).toBeNull();

    // Catálogo en memoria se actualizó
    const updatedInCatalog = component.techComponents().find(c => c.id === 'rt-python-312');
    expect(updatedInCatalog?.name).toBe('Python 3.12 Actualizado');

    // Selección activa y estudio se recalcularon automáticamente
    expect(component.name()).toBe('Laboratorio: Python 3.12 Actualizado');
    expect(component.toolsDeclared()).toBe('python3, pip, poetry');
  });

  it('should restore Solo Frontend template correctly in applyTemplateToStudio', () => {
    const soloFrontTemplate: AdminTemplateItem = {
      id: 'tpl-solo-frontend',
      name: 'Solo Angular Lab',
      docker_image: 'solv-local/stack:angular123',
      base_ram_mb: 1024,
      status: 'approved',
      created_at: new Date().toISOString(),
      target_environment: 'IDE_PERSISTENTE',
      setup_script: '',
      tools_declared: ['node', 'npm', 'ng'],
      stack_config: {
        mode: 'visual',
        web_platform_id: 'web-angular',
        addon_ids: []
      }
    };

    (component as unknown as { applyTemplateToStudio: (tpl: AdminTemplateItem, dup: boolean) => void })
      .applyTemplateToStudio(soloFrontTemplate, false);

    expect(component.selectedRuntimeId()).toBeNull();
    expect(component.selectedWebPlatformId()).toBe('web-angular');
    expect(component.effectivePrimaryComponent()?.id).toBe('web-angular');
    expect(component.baseRamMB()).toBe(1024);
  });

  it('should use RUN apk add nodejs npm when web platform is added to alpine runtime', () => {
    const alpineRt: TemplateComponent = {
      id: 'rt-alpine-python',
      slug: 'alpine-python',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Python Alpine',
      description: 'Alpine python',
      icon_key: 'python',
      base_image: 'python:3.12-alpine',
      compatible_runtimes: ['python'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: ['python3'],
      service_engine: '',
      ram_delta_mb: 256,
      is_system: true,
      is_active: true,
      sort_order: 12
    };

    component.techComponents.update(list => [...list, alpineRt]);
    component.selectRuntime('rt-alpine-python');
    component.selectWebPlatform('web-angular');

    expect(component.dockerfileContent()).toContain('RUN apk add --no-cache nodejs npm');
    expect(component.dockerfileContent()).toContain('RUN npm install -g --no-audit --no-fund @angular/cli');
  });

  it('should pass techComponents to component-form-modal and preselect web component key in Solo Frontend stack', () => {
    // 1. Initially nothing selected -> empty key
    expect(component.modalDefaultCompatibleRuntime()).toBe('');

    // 2. Select Solo Frontend stack (Angular, no backend runtime)
    component.selectedRuntimeId.set(null);
    component.selectWebPlatform('web-angular');
    expect(component.selectedRuntimeComponent()).toBeNull();
    expect(component.selectedWebComponent()?.id).toBe('web-angular');
    expect(component.effectivePrimaryComponent()?.id).toBe('web-angular');

    // modalDefaultCompatibleRuntime() must return web component's key: (rt.slug || rt.icon_key || rt.name).toLowerCase()
    // For web-angular, slug is 'angular-web'
    expect(component.modalDefaultCompatibleRuntime()).toBe('angular-web');

    // 3. Open modal to register a new addon technology
    component.openNewComponentModal('addon');
    expect(component.showComponentModal()).toBe(true);
    fixture.detectChanges();

    // 4. Assert component-form-modal receives all techComponents (not just backendRuntimes)
    const modalDebugEl = fixture.debugElement.query(By.directive(ComponentFormModalComponent));
    expect(modalDebugEl).toBeTruthy();
    const modalComp = modalDebugEl.componentInstance as ComponentFormModalComponent;

    expect(modalComp.availableRuntimes).toEqual(component.techComponents());
    expect(modalComp.defaultCompatibleRuntime).toBe('angular-web');

    // Verify compatibleRuntimeOptions includes the web component and universal
    const options = modalComp.compatibleRuntimeOptions();
    expect(options.some(opt => opt.id === 'angular-web' && opt.label === 'Angular')).toBe(true);
    expect(options.some(opt => opt.id === 'universal')).toBe(true);

    // 5. When selecting a backend runtime, it returns backend runtime key
    component.selectRuntime('rt-python-312');
    expect(component.modalDefaultCompatibleRuntime()).toBe('python-312');
  });

  it('should set modalDefaultSubCategory and pass it to component-form-modal', () => {
    // Default openNewComponentModal with backend
    component.openNewComponentModal('runtime', 'backend');
    expect(component.modalDefaultLayer()).toBe('runtime');
    expect(component.modalDefaultSubCategory()).toBe('backend');
    expect(component.showComponentModal()).toBe(true);
    fixture.detectChanges();

    let modalDebugEl = fixture.debugElement.query(By.directive(ComponentFormModalComponent));
    expect(modalDebugEl).toBeTruthy();
    let modalComp = modalDebugEl.componentInstance as ComponentFormModalComponent;
    expect(modalComp.defaultSubCategory).toBe('backend');

    component.handleModalClosed();
    fixture.detectChanges();

    // Open from web button
    component.openNewComponentModal('runtime', 'web');
    expect(component.modalDefaultLayer()).toBe('runtime');
    expect(component.modalDefaultSubCategory()).toBe('web');
    expect(component.showComponentModal()).toBe(true);
    fixture.detectChanges();

    modalDebugEl = fixture.debugElement.query(By.directive(ComponentFormModalComponent));
    expect(modalDebugEl).toBeTruthy();
    modalComp = modalDebugEl.componentInstance as ComponentFormModalComponent;
    expect(modalComp.defaultSubCategory).toBe('web');
  });

  it('should permanently render web platforms subsection with empty hint when empty in IDE_PERSISTENTE', () => {
    // Clear web platforms
    component.techComponents.set(
      MOCK_CATALOG_ITEMS.filter(c => !(c.layer === 'runtime' && c.sub_category === 'web'))
    );
    component.targetEnvironment.set('IDE_PERSISTENTE');
    fixture.detectChanges();

    expect(component.webPlatforms().length).toBe(0);

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Interfaz Web / Frontend (Opcional)');
    expect(compiled.textContent).toContain("No hay plataformas web registradas. Puede agregar una con '+ Nueva web'.");
  });

  it('should disable canRunVerification and canPublishTemplate when 0 technologies are selected, even if template name is provided', () => {
    // 0 technologies selected in Step 2, but name is typed in Step 1
    component.selectedRuntimeId.set(null);
    component.selectedWebPlatformId.set(null);
    component.selectedAddonIds.set([]);
    component.dockerImage.set('');
    component.name.set('Plantilla sin tecnologías');
    fixture.detectChanges();

    expect(component.canRunVerification()).toBe(false);
    expect(component.canPublishTemplate()).toBe(false);

    const vBlocker = component.verificationBlocker();
    expect(vBlocker).not.toBeNull();
    expect(vBlocker?.tone).toBe('neutral');
    expect(vBlocker?.reason).toBe(
      'Seleccione una tecnología principal en el Paso 2 para habilitar la comprobación y publicación.'
    );

    const pBlocker = component.publishBlocker();
    expect(pBlocker).toEqual(vBlocker);

    // Verify DOM: both buttons are disabled and the blocker note is rendered
    const buttons = fixture.debugElement.queryAll(By.css('.blueprint-footer button'));
    expect(buttons.length).toBe(2);
    expect(buttons[0].nativeElement.disabled).toBe(true);
    expect(buttons[1].nativeElement.disabled).toBe(true);

    const note = fixture.debugElement.query(By.css('.blueprint-footer .publish-block-note'));
    expect(note).toBeTruthy();
    expect(note.nativeElement.textContent).toContain(
      'Seleccione una tecnología principal en el Paso 2 para habilitar la comprobación y publicación.'
    );
  });

  it('should require judge entrypoint and render judge-visual-config-bar in JUEZ_EFIMERO visual mode', () => {
    component.targetEnvironment.set('JUEZ_EFIMERO');
    component.selectedRuntimeId.set(null);
    component.dockerImage.set('');
    fixture.detectChanges();

    // 1. Without runtime selected, blocker specifies selecting runtime in Column 1
    expect(component.verificationBlocker()?.reason).toBe(
      'Seleccione un Lenguaje Principal en la Columna 1 para configurar el Evaluador Automático.'
    );
    expect(component.verificationBlocker()?.tone).toBe('neutral');
    expect(component.canRunVerification()).toBe(false);

    // 2. Select Python runtime
    component.selectRuntime('rt-python-312');
    fixture.detectChanges();

    // Entrypoint bar should be visible in visual mode
    const visualBar = fixture.debugElement.query(By.css('.judge-visual-config-bar'));
    expect(visualBar).toBeTruthy();
    const entrypointInput = fixture.debugElement.query(By.css('#visual-judge-entrypoint'));
    expect(entrypointInput).toBeTruthy();
    expect(component.entrypoint()).toBe('python3 solution.py');
    expect(component.verificationBlocker()).toBeNull();
    expect(component.canRunVerification()).toBe(true);

    // 3. Clear entrypoint: should trigger warning blocker
    component.entrypoint.set('   ');
    expect(component.verificationBlocker()?.reason).toBe(
      'Indique el comando de compilación y ejecución del Evaluador Automático.'
    );
    expect(component.verificationBlocker()?.tone).toBe('warning');
    expect(component.canRunVerification()).toBe(false);

    // 4. Restore valid entrypoint
    component.entrypoint.set('python3 solution.py');
    expect(component.verificationBlocker()).toBeNull();
    expect(component.canRunVerification()).toBe(true);
  });

  it('should prevent execution in runVerificationTest and publishTemplate when blockers are active', () => {
    component.selectedRuntimeId.set(null);
    component.selectedWebPlatformId.set(null);
    component.dockerImage.set('');
    component.name.set('Plantilla Bloqueada');

    const envService = TestBed.inject(EnvTestJobService);
    const startJobSpy = vi.spyOn(envService, 'startJob');
    const tplService = TestBed.inject(AdminTemplatesService);
    const createSpy = vi.spyOn(tplService, 'createOfficialTemplate');

    component.runVerificationTest();
    expect(startJobSpy).not.toHaveBeenCalled();
    expect(component.toast()?.type).toBe('error');
    expect(component.toast()?.message).toBe(
      'Seleccione una tecnología principal en el Paso 2 para habilitar la comprobación y publicación.'
    );

    component.publishTemplate();
    expect(createSpy).not.toHaveBeenCalled();
    expect(component.toast()?.type).toBe('error');
    expect(component.toast()?.message).toBe(
      'Seleccione una tecnología principal en el Paso 2 para habilitar la comprobación y publicación.'
    );
  });

  it('should handle targetEnvironment AMBOS: strips web/database, populates both setupScript and judge entrypoint, and prefixes name with Entorno Dual', () => {
    component.techComponents.update(list => [
      ...list,
      {
        id: 'addon-custom-snippet',
        slug: 'custom-snippet',
        layer: 'addon',
        sub_category: 'library',
        name: 'Custom Library',
        description: 'Test snippet',
        icon_key: 'python',
        base_image: '',
        compatible_runtimes: ['python'],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: 'echo "export PYTHONUNBUFFERED=1" >> ~/.bashrc',
        judge_entrypoint: '',
        tools_provided: ['custom'],
        service_engine: '',
        ram_delta_mb: 0,
        is_system: false,
        is_active: true,
        sort_order: 99
      }
    ]);

    component.selectRuntime('rt-python-312');
    component.toggleAddon('addon-custom-snippet');
    component.selectWebPlatform('web-angular');
    component.selectDatabase('db-postgres');

    component.setTargetEnvironment('AMBOS');
    expect(component.targetEnvironment()).toBe('AMBOS');
    expect(component.selectedWebPlatformId()).toBeNull();
    expect(component.selectedDatabaseId()).toBe('db-none');
    expect(component.selectedWebComponent()).toBeNull();
    expect(component.selectedDatabaseOption()).toBeNull();
    expect(component.name()).toContain('Entorno Dual: ');
    expect(component.entrypoint()).toBe('python3 solution.py');
    expect(component.setupScript()).toBe('echo "export PYTHONUNBUFFERED=1" >> ~/.bashrc');
  });

  it('should require selectedRuntimeComponent and entrypoint for AMBOS in verificationBlocker', () => {
    component.selectedRuntimeId.set(null);
    component.setTargetEnvironment('AMBOS');
    expect(component.verificationBlocker()?.reason).toContain('Seleccione un Lenguaje Principal');

    component.selectRuntime('rt-python-312');
    component.entrypoint.set('');
    expect(component.verificationBlocker()?.reason).toContain('Indique el comando de compilación');

    component.entrypoint.set('python3 solution.py');
    expect(component.verificationBlocker()).toBeNull();
  });

  it('should sync modalDefaultSubCategory when opening edit modal for backend or web components', () => {
    const backendComp = component.techComponents().find(c => c.id === 'rt-python-312')!;
    component.openEditComponentModal(backendComp);
    expect(component.modalDefaultSubCategory()).toBe('backend');

    const webComp = component.techComponents().find(c => c.id === 'web-angular')!;
    component.openEditComponentModal(webComp);
    expect(component.modalDefaultSubCategory()).toBe('web');
  });

  describe('Column Search and Selected Item Prioritization', () => {
    it('should filter backend runtimes and prioritize selected runtime at the top', () => {
      component.selectRuntime('rt-python-312');

      component.col1SearchQuery.set('python');
      const filtered = component.filteredBackendRuntimes();
      expect(filtered.length).toBeGreaterThan(0);
      expect(filtered[0].id).toBe('rt-python-312');

      component.col1SearchQuery.set('');
      const allRuntimes = component.filteredBackendRuntimes();
      expect(allRuntimes[0].id).toBe('rt-python-312');
    });

    it('should filter web platforms by col1SearchQuery and prioritize selected web platform', () => {
      component.selectWebPlatform('web-angular');

      component.col1SearchQuery.set('angular');
      const filtered = component.filteredWebPlatforms();
      expect(filtered.length).toBeGreaterThan(0);
      expect(filtered[0].id).toBe('web-angular');
    });

    it('should filter compatible addons by name, slug, description, or packages and prioritize selected addons', () => {
      component.selectRuntime('rt-python-312');
      component.toggleAddon('ad-fastapi');

      component.col2SearchQuery.set('fastapi');
      const filtered = component.filteredCompatibleAddons();
      expect(filtered.length).toBeGreaterThan(0);
      expect(filtered[0].id).toBe('ad-fastapi');

      component.col2SearchQuery.set('');
      const allAddons = component.filteredCompatibleAddons();
      expect(allAddons[0].id).toBe('ad-fastapi');
    });

    it('should render selected-count-badge when addons are selected', () => {
      component.selectRuntime('rt-python-312');
      component.toggleAddon('ad-fastapi');
      fixture.detectChanges();

      const badge = fixture.nativeElement.querySelector('.selected-count-badge');
      expect(badge).toBeTruthy();
      expect(badge.textContent).toContain('1 seleccionados');
    });

    it('should reset search queries on runtime/web selection and modal save', () => {
      component.col2SearchQuery.set('prev-search');
      component.selectRuntime('rt-python-312');
      expect(component.col2SearchQuery()).toBe('');

      component.col2SearchQuery.set('prev-web-search');
      component.selectWebPlatform('web-angular');
      expect(component.col2SearchQuery()).toBe('');

      component.col1SearchQuery.set('search-rt');
      component.handleModalComponentSaved(MOCK_CATALOG_ITEMS[0]);
      expect(component.col1SearchQuery()).toBe('');

      component.col2SearchQuery.set('search-addon');
      const addonComp: TemplateComponent = {
        ...MOCK_CATALOG_ITEMS[0],
        id: 'ad-mock',
        layer: 'addon'
      };
      component.handleModalComponentSaved(addonComp);
      expect(component.col2SearchQuery()).toBe('');
    });
  });

  describe('Data-Driven Profiles and Entrypoint Fallbacks', () => {
    it('should infer entrypoint in JUEZ_EFIMERO and AMBOS when runtime has no explicit entrypoint', () => {
      const gccRuntime: TemplateComponent = {
        id: 'rt-gcc-no-entry',
        slug: 'custom-gcc',
        layer: 'runtime',
        sub_category: 'backend',
        name: 'Custom GCC',
        description: '',
        icon_key: 'custom',
        base_image: 'gcc:13-bookworm',
        compatible_runtimes: [],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['g++', 'gcc'],
        service_engine: '',
        ram_delta_mb: 384,
        is_system: false,
        is_active: true,
        sort_order: 1
      };

      component.techComponents.set([...MOCK_CATALOG_ITEMS, gccRuntime]);
      component.setTargetEnvironment('JUEZ_EFIMERO');
      component.selectRuntime('rt-gcc-no-entry');

      expect(component.entrypoint()).toBe('g++ -O2 solution.cpp -o solution && ./solution');

      component.setTargetEnvironment('AMBOS');
      expect(component.entrypoint()).toBe('g++ -O2 solution.cpp -o solution && ./solution');
    });

    it('should infer entrypoint from tools_provided for unfamiliar runtimes without known presets', () => {
      const rubyRuntime: TemplateComponent = {
        id: 'rt-ruby',
        slug: 'ruby-lang',
        layer: 'runtime',
        sub_category: 'backend',
        name: 'Ruby Language',
        description: '',
        icon_key: 'docker',
        base_image: 'custom-ruby:3.2',
        compatible_runtimes: [],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['ruby', 'gem'],
        service_engine: '',
        ram_delta_mb: 512,
        is_system: false,
        is_active: true,
        sort_order: 1
      };

      component.techComponents.set([...MOCK_CATALOG_ITEMS, rubyRuntime]);
      component.setTargetEnvironment('AMBOS');
      component.selectRuntime('rt-ruby');

      expect(component.entrypoint()).toBe('ruby solution');
    });

    it('should calculate data-driven RAM based on ram_delta_mb for light and heavy runtimes', () => {
      const lightRuntime: TemplateComponent = {
        id: 'rt-lua',
        slug: 'lua-vm',
        layer: 'runtime',
        sub_category: 'backend',
        name: 'Lua VM',
        description: '',
        icon_key: 'docker',
        base_image: 'lua:5.4',
        compatible_runtimes: [],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['lua'],
        service_engine: '',
        ram_delta_mb: 384,
        is_system: false,
        is_active: true,
        sort_order: 1
      };

      const heavyRuntime: TemplateComponent = {
        id: 'rt-julia',
        slug: 'julia-vm',
        layer: 'runtime',
        sub_category: 'backend',
        name: 'Julia VM',
        description: '',
        icon_key: 'docker',
        base_image: 'julia:1.10',
        compatible_runtimes: [],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['julia'],
        service_engine: '',
        ram_delta_mb: 1024,
        is_system: false,
        is_active: true,
        sort_order: 2
      };

      component.techComponents.set([...MOCK_CATALOG_ITEMS, lightRuntime, heavyRuntime]);

      // Lua en Juez -> 128 MB
      component.setTargetEnvironment('JUEZ_EFIMERO');
      component.selectRuntime('rt-lua');
      expect(component.recommendedRamMB()).toBe(128);

      // Julia en Juez -> 512 MB
      component.selectRuntime('rt-julia');
      expect(component.recommendedRamMB()).toBe(512);

      // Julia en IDE -> 1024 MB
      component.setTargetEnvironment('IDE_PERSISTENTE');
      expect(component.recommendedRamMB()).toBe(1024);
    });
  });

  describe('Column 2 Grouping, Quick Filter Pills, and Mutual Exclusion', () => {
    it('should initialize col2CategoryFilter with "all" and filter groups accordingly', () => {
      expect(component.col2CategoryFilter()).toBe('all');

      component.col2CategoryFilter.set('framework');
      expect(component.col2CategoryFilter()).toBe('framework');

      component.col2CategoryFilter.set('orm');
      expect(component.col2CategoryFilter()).toBe('orm');

      component.col2CategoryFilter.set('toolkit');
      expect(component.col2CategoryFilter()).toBe('toolkit');
    });

    it('should reset col2CategoryFilter to "all" when selecting runtime or web platform', () => {
      component.col2CategoryFilter.set('framework');
      component.selectRuntime('rt-python-312');
      expect(component.col2CategoryFilter()).toBe('all');

      component.col2CategoryFilter.set('orm');
      component.selectWebPlatform('web-angular');
      expect(component.col2CategoryFilter()).toBe('all');
    });

    it('should enforce mutual exclusion when selecting multiple framework addons', () => {
      const djangoAddon: TemplateComponent = {
        id: 'ad-django',
        slug: 'django',
        layer: 'addon',
        sub_category: 'framework',
        name: 'Django',
        description: 'Django web framework',
        icon_key: 'python',
        base_image: '',
        compatible_runtimes: ['python'],
        install_mode: 'pip',
        package_names: ['django'],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['django-admin'],
        service_engine: '',
        ram_delta_mb: 256,
        is_system: true,
        is_active: true,
        sort_order: 10
      };

      component.techComponents.set([...MOCK_CATALOG_ITEMS, djangoAddon]);
      component.selectRuntime('rt-python-312');

      // Select FastAPI (framework)
      component.toggleAddon('ad-fastapi');
      expect(component.selectedAddonIds()).toContain('ad-fastapi');

      // Select Django (framework) -> FastAPI must be automatically deselected
      component.toggleAddon('ad-django');
      expect(component.selectedAddonIds()).toContain('ad-django');
      expect(component.selectedAddonIds()).not.toContain('ad-fastapi');

      // Select non-framework addon (e.g. ad-numpy) -> Django stays selected
      component.toggleAddon('ad-numpy');
      expect(component.selectedAddonIds()).toContain('ad-django');
      expect(component.selectedAddonIds()).toContain('ad-numpy');
    });

    it('should auto-tune baseRamMB when stack changes if RAM was not manually edited', () => {
      component.selectRuntime('rt-python-312');
      expect(component.isRamManuallyEdited()).toBe(false);
      const initialRam = component.baseRamMB();
      expect(initialRam).toBe(component.recommendedRamMB());

      // Toggle an addon
      component.toggleAddon('ad-fastapi');
      expect(component.isRamManuallyEdited()).toBe(false);
      expect(component.baseRamMB()).toBe(component.recommendedRamMB());

      // If user manually customizes RAM
      component.isRamManuallyEdited.set(true);
      component.baseRamMB.set(1536);

      // Toggle another addon: custom RAM must be preserved
      component.toggleAddon('ad-numpy');
      expect(component.baseRamMB()).toBe(1536);

      // Switching runtime resets isRamManuallyEdited and auto-tunes
      component.selectRuntime('rt-cpp-13');
      expect(component.isRamManuallyEdited()).toBe(false);
      expect(component.baseRamMB()).toBe(component.recommendedRamMB());
    });
  });

  describe('classifyAddonComponent and Official Catalog Presets', () => {
    it('should prioritize spring-maven slug/id as toolkit', () => {
      const springMavenComp: TemplateComponent = {
        id: 'spring-maven',
        slug: 'spring-maven',
        layer: 'addon',
        sub_category: 'framework', // even if sub_category was set to framework
        name: 'Apache Maven (Proyectos Java / Spring)',
        description: 'Gestor de dependencias Maven',
        icon_key: 'java',
        base_image: '',
        compatible_runtimes: ['java'],
        install_mode: 'apt',
        package_names: ['maven'],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['mvn'],
        service_engine: '',
        ram_delta_mb: 128,
        is_system: true,
        is_active: true,
        sort_order: 1
      };

      expect(classifyAddonComponent(springMavenComp)).toBe('toolkit');
    });

    it('should classify by explicit sub_category before text heuristics', () => {
      const springBootFramework: TemplateComponent = {
        id: 'java-springboot',
        slug: 'java-springboot',
        layer: 'addon',
        sub_category: 'framework',
        name: 'Spring Boot Starter',
        description: 'Includes maven build and spring tools',
        icon_key: 'java',
        base_image: '',
        compatible_runtimes: ['java'],
        install_mode: 'custom',
        package_names: ['spring-boot-starter'],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: ['mvn', 'java'],
        service_engine: '',
        ram_delta_mb: 512,
        is_system: true,
        is_active: true,
        sort_order: 2
      };

      // Even though text has 'maven', explicit sub_category: 'framework' must take precedence
      expect(classifyAddonComponent(springBootFramework)).toBe('framework');

      const connectorComp: TemplateComponent = {
        ...springBootFramework,
        sub_category: 'connector',
        name: 'JDBC Driver',
        description: ''
      };
      expect(classifyAddonComponent(connectorComp)).toBe('orm');

      const heavyToolkit: TemplateComponent = {
        ...springBootFramework,
        sub_category: 'heavy',
        name: 'Heavy Data Science Toolkit',
        description: ''
      };
      expect(classifyAddonComponent(heavyToolkit)).toBe('toolkit');
    });

    it('should fall back to text heuristics when sub_category is not classified', () => {
      const fallbackOrm: TemplateComponent = {
        id: 'custom-comp',
        slug: 'custom-comp',
        layer: 'addon',
        sub_category: '',
        name: 'My Custom Prisma Client',
        description: '',
        icon_key: 'custom',
        base_image: '',
        compatible_runtimes: ['node'],
        install_mode: 'none',
        package_names: [],
        build_layer_snippet: '',
        setup_snippet: '',
        judge_entrypoint: '',
        tools_provided: [],
        service_engine: '',
        ram_delta_mb: 128,
        is_system: false,
        is_active: true,
        sort_order: 1
      };
      expect(classifyAddonComponent(fallbackOrm)).toBe('orm');

      const fallbackFw: TemplateComponent = {
        ...fallbackOrm,
        name: 'FastAPI DevServer'
      };
      expect(classifyAddonComponent(fallbackFw)).toBe('framework');

      const fallbackToolkit: TemplateComponent = {
        ...fallbackOrm,
        name: 'PyTest Suite Runner'
      };
      expect(classifyAddonComponent(fallbackToolkit)).toBe('toolkit');
    });

    it('should include nextjs-web in OFFICIAL_WEB_PROFILES', () => {
      const nextjs = OFFICIAL_WEB_PROFILES.find((p) => p.id === 'nextjs-web');
      expect(nextjs).toBeDefined();
      expect(nextjs?.name).toBe('Next.js (React Full-Stack)');
      expect(nextjs?.iconKey).toBe('react');
      expect(nextjs?.baseImage).toBe('node:20-bookworm-slim');
      expect(nextjs?.packageNames).toContain('create-next-app');
      expect(nextjs?.toolsProvided).toContain('npx');
    });

    it('should include all required 24 official university addon presets', () => {
      const requiredIds = [
        'python-sqlalchemy',
        'python-nosql-clients',
        'python-ml',
        'python-testing-ruff',
        'java-springboot',
        'java-hibernate-jdbc',
        'java-gradle',
        'java-junit5',
        'node-express',
        'node-nestjs',
        'node-prisma',
        'node-typeorm',
        'node-vitest-jest',
        'cpp-sql-drivers',
        'cpp-openmp',
        'go-gin-chi',
        'go-gorm',
        'go-golangci-lint',
        'rust-axum',
        'rust-sqlx',
        'rust-clippy',
        'dotnet-aspnet',
        'dotnet-ef',
        'dotnet-xunit'
      ];

      for (const id of requiredIds) {
        const found = OFFICIAL_ADDON_PRESETS.find((p) => p.id === id);
        expect(found).toBeDefined();
        expect(found?.name.length).toBeGreaterThan(0);
      }

      const maven = OFFICIAL_ADDON_PRESETS.find((p) => p.id === 'spring-maven');
      expect(maven).toBeDefined();
      expect(maven?.name).toBe('Apache Maven (Proyectos Java / Spring)');
      expect(maven?.memoryProfile).toBe('library');
    });
  });

  describe('Sugerencia de Stack con Copiloto IA', () => {
    it('invoca suggestTemplate y genera una propuesta sin aplicar nada aún', () => {
      component.aiPrompt.set('Fullstack Python con SQLite');
      component.suggestStackWithAi();

      // La propuesta queda pendiente de revisión: no toca el constructor.
      expect(component.aiProposal()).not.toBeNull();
      expect(component.aiProposalApplied()).toBe(false);
      expect(component.name()).toBe('');
      expect(component.description()).toBe('');
      expect(component.selectedRuntimeId()).toBeNull();
      expect(component.isSuggestingWithAi()).toBe(false);
    });

    it('aplica la propuesta al constructor solo tras applyAiProposal', () => {
      component.aiPrompt.set('Fullstack Python con SQLite');
      component.suggestStackWithAi();
      expect(component.aiProposal()).not.toBeNull();

      component.applyAiProposal();

      expect(component.name()).toBe('Python Fast Stack');
      expect(component.description()).toBe('Plantilla sugerida por IA');
      expect(component.selectedRuntimeId()).toBe('rt-python-312');
      expect(component.setupScript()).toBe('pip install -r requirements.txt');
      expect(component.baseRamMB()).toBe(512);
      expect(component.aiProposalApplied()).toBe(true);
    });

    it('descarta la propuesta sin modificar el constructor', () => {
      component.aiPrompt.set('Fullstack Python con SQLite');
      component.suggestStackWithAi();
      expect(component.aiProposal()).not.toBeNull();

      component.discardAiProposal();

      expect(component.aiProposal()).toBeNull();
      expect(component.selectedRuntimeId()).toBeNull();
      expect(component.name()).toBe('');
    });

    it('cancela la sugerencia en vuelo al invocar cancelAiSuggestion', () => {
      component.aiPrompt.set('Fullstack');
      component.suggestStackWithAi();
      component.cancelAiSuggestion();
      expect(component.isSuggestingWithAi()).toBe(false);
    });

    it('abre el modal del copiloto IA con openAiCopilotModal', () => {
      expect(component.isAiCopilotModalOpen()).toBe(false);
      component.openAiCopilotModal();
      expect(component.isAiCopilotModalOpen()).toBe(true);
    });

    it('abre y cierra el Mapa de Herramientas con openTechGraph', () => {
      expect(component.isTechGraphOpen()).toBe(false);
      component.openTechGraph();
      expect(component.isTechGraphOpen()).toBe(true);
      component.isTechGraphOpen.set(false);
      expect(component.isTechGraphOpen()).toBe(false);
    });

    it('monta tech-graph-modal en el DOM al pulsar el botón Mapa de Herramientas', () => {
      fixture.detectChanges();
      const buttons = fixture.nativeElement.querySelectorAll(
        'button'
      ) as NodeListOf<HTMLButtonElement>;
      const btn = Array.from(buttons).find(b =>
        (b.textContent || '').includes('Mapa de Herramientas')
      );
      expect(btn).toBeTruthy();

      btn!.click();
      fixture.detectChanges();

      expect(fixture.nativeElement.querySelector('tech-graph-modal')).toBeTruthy();
    });

    it('aplica la propuesta y cierra el modal en onApplyAiCopilotModal', () => {
      component.openAiCopilotModal();
      component.aiPrompt.set('Fullstack Python con SQLite');
      component.suggestStackWithAi();
      expect(component.aiProposal()).not.toBeNull();

      component.onApplyAiCopilotModal();

      expect(component.isAiCopilotModalOpen()).toBe(false);
      expect(component.aiProposalApplied()).toBe(true);
      expect(component.selectedRuntimeId()).toBe('rt-python-312');
    });
  });
});



