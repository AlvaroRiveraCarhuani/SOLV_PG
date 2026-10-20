import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { TemplateStudioComponent } from './template-studio.component';
import { AdminTemplatesService } from '../../services/admin-templates.service';
import { AdminTemplateItem, TemplateComponent } from '@features/admin/models/admin-templates.models';
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

  it('should govern Column 3 database options from node RuntimeCapabilities without a + Nuevo button, displaying only provisioned engines (SQLite and PostgreSQL v18)', () => {
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
    expect(col3Text).toContain('Sin Base de Datos');
    expect(col3Text).toContain('SQLite3 (Nativo) v3.x');
    expect(col3Text).toContain('+0 MB RAM');
    expect(col3Text).toContain('PostgreSQL (Aula) v18');
    expect(col3Text).toContain('~50 MB RAM');
    expect(col3Text).not.toContain('No habilitado en este nodo');
    expect(col3Text).not.toContain('MySQL');
    expect(col3Text).not.toContain('MongoDB');
    expect(col3Text).not.toContain('Redis');

    const dbCards = Array.from(columns[2].querySelectorAll<HTMLButtonElement>('.tech-component-card'));
    expect(dbCards.length).toBe(3);

    // Attempting to select an unprovisioned / unavailable engine does not change selectedDatabaseId
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

    // Simular que el test finaliza en éxito
    component.activeEnvTestJob.set({
      id: 'job-1',
      image: 'solv-local/stack:test',
      tools: [],
      status: 'success',
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
    });

    expect(component.requiresVerificationBeforePublish()).toBe(false);
  });

  it('should generate multi-distro nodejs/npm snippet when web platform is added to non-Node runtime', () => {
    component.selectRuntime('rt-python-312');
    component.selectWebPlatform('web-angular');

    expect(component.dockerfileContent()).toContain('(command -v apk >/dev/null 2>&1 && apk add --no-cache nodejs npm) || (apt-get update && apt-get install -y --no-install-recommends nodejs npm && rm -rf /var/lib/apt/lists/*)');
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
});
