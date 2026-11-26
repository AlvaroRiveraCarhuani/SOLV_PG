import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TeacherTemplatesComponent, PublishedTemplate } from './teacher-templates.component';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { Router } from '@angular/router';
import { signal } from '@angular/core';
import { TeacherCourseSummary } from '../models/teacher.models';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';
import { TemplateRequestModalComponent } from './template-request-modal/template-request-modal.component';
import { EnvironmentRequestDraftService } from '../services/environment-request-draft.service';

describe('TeacherTemplatesComponent', () => {
  let component: TeacherTemplatesComponent;
  let fixture: ComponentFixture<TeacherTemplatesComponent>;
  let httpMock: HttpTestingController;
  let mockDashboardService: Partial<TeacherDashboardService>;

  const mockCourses: TeacherCourseSummary[] = [
    {
      id: 'course-1',
      code: 'CS101',
      name: 'Estructuras de Datos',
      students_count: 20,
      active_now: 5,
      pending_review: 2,
      at_risk: 1
    }
  ];

  const mockTechCatalog: TemplateComponent[] = [
    {
      id: 'comp-node',
      slug: 'node-20',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Node.js 20 LTS',
      description: 'Entorno JavaScript y TypeScript',
      icon_key: 'node',
      base_image: 'node:20-bookworm-slim',
      compatible_runtimes: ['node', 'web'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: 'node solution.js',
      tools_provided: ['node', 'npm'],
      service_engine: '',
      ram_delta_mb: 512,
      is_system: true,
      is_active: true,
      sort_order: 20
    },
    {
      id: 'comp-nextjs',
      slug: 'nextjs',
      layer: 'runtime',
      sub_category: 'web',
      name: 'Next.js (React Full-Stack)',
      description: 'Framework React de producción',
      icon_key: 'nextjs',
      base_image: 'node:20-bookworm-slim',
      compatible_runtimes: ['node', 'web'],
      install_mode: 'npm',
      package_names: ['next'],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: ['node', 'npm'],
      service_engine: '',
      ram_delta_mb: 128,
      is_system: true,
      is_active: true,
      sort_order: 110
    },
    {
      id: 'comp-prisma',
      slug: 'prisma',
      layer: 'addon',
      sub_category: 'orm',
      name: 'Prisma ORM',
      description: 'ORM tipado para Node.js y TypeScript',
      icon_key: 'prisma',
      base_image: '',
      compatible_runtimes: ['node'],
      install_mode: 'npm',
      package_names: ['prisma'],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: ['npx'],
      service_engine: '',
      ram_delta_mb: 64,
      is_system: true,
      is_active: true,
      sort_order: 280
    },
    {
      id: 'comp-db-none',
      slug: 'db-none',
      layer: 'database',
      sub_category: 'none',
      name: 'Sin Base de Datos',
      description: 'Sin motor de datos',
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
    }
  ];

  const mockTemplates: PublishedTemplate[] = [
    {
      id: 'tpl-1',
      name: 'Python Data Science Stack',
      description: 'Entorno completo con NumPy, Pandas y Jupyter.',
      docker_image: 'solv/python-datascience:v1.0.0',
      default_memory_mb: 1024,
      default_cpu_cores: 2,
      category: 'Programación',
      environment_type: 'IDE_PERSISTENTE',
      is_official: true,
      is_active: true,
      stack_config: { runtime_id: 'python-312', addon_ids: ['jupyter'] }
    },
    {
      id: 'tpl-2',
      name: 'PostgreSQL 16 Engine',
      description: 'Instancia SQL para laboratorios de bases de datos.',
      docker_image: 'solv/postgres:16-alpine',
      default_memory_mb: 512,
      default_cpu_cores: 1,
      category: 'Bases de Datos',
      environment_type: 'JUEZ_EFIMERO',
      is_official: false,
      is_active: true
    }
  ];

  beforeEach(async () => {
    mockDashboardService = {
      courses: signal<TeacherCourseSummary[]>(mockCourses)
    };

    await TestBed.configureTestingModule({
      imports: [TeacherTemplatesComponent, HttpClientTestingModule, RouterTestingModule],
      providers: [
        { provide: TeacherDashboardService, useValue: mockDashboardService }
      ]
    }).compileComponents();

    httpMock = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(TeacherTemplatesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();

    const req = httpMock.expectOne('/api/v1/templates');
    expect(req.request.method).toBe('GET');
    req.flush({ data: mockTemplates });

    // La página carga también las solicitudes propias del docente.
    const requestsReq = httpMock.expectOne('/api/v1/teacher/environment-requests');
    expect(requestsReq.request.method).toBe('GET');
    requestsReq.flush({ data: [] });

    // El catálogo de componentes traduce los ids del stack a nombres e iconos.
    const catalogReq = httpMock.expectOne('/api/v1/admin/template-components');
    expect(catalogReq.request.method).toBe('GET');
    catalogReq.flush({ data: mockTechCatalog });
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should load templates on init and calculate KPIs correctly', () => {
    expect(component.templates().length).toBe(2);

    const stats = component.kpiStats();
    expect(stats.total).toBe(2);
    expect(stats.persistent).toBe(1);
    expect(stats.ephemeral).toBe(1);
    expect(stats.official).toBe(1);
  });

  it('should filter templates by category', () => {
    component.categoryFilter.set('Bases de Datos');
    expect(component.filteredTemplates().length).toBe(1);
    expect(component.filteredTemplates()[0].name).toBe('PostgreSQL 16 Engine');
  });

  it('should filter templates by environment type', () => {
    component.envTypeFilter.set('IDE_PERSISTENTE');
    expect(component.filteredTemplates().length).toBe(1);
    expect(component.filteredTemplates()[0].id).toBe('tpl-1');
  });

  it('shows a dual template under both environment filters', () => {
    const dual: PublishedTemplate = {
      id: 'tpl-dual',
      name: 'Dual Python',
      description: 'Sirve para las dos modalidades.',
      docker_image: 'solv/python:3.12',
      default_memory_mb: 1024,
      default_cpu_cores: 2,
      category: 'Programación',
      environment_type: 'AMBOS',
      is_official: true,
      is_active: true
    };
    component.templates.set([...mockTemplates, dual]);

    component.envTypeFilter.set('IDE_PERSISTENTE');
    expect(component.filteredTemplates().map(t => t.id)).toContain('tpl-dual');

    component.envTypeFilter.set('JUEZ_EFIMERO');
    expect(component.filteredTemplates().map(t => t.id)).toContain('tpl-dual');
  });

  it('resolves the historical JUEZ_VIRTUAL alias when filtering', () => {
    const legacy = { ...mockTemplates[1], id: 'tpl-legacy', environment_type: 'JUEZ_VIRTUAL' as unknown as PublishedTemplate['environment_type'] };
    component.templates.set([legacy]);

    component.envTypeFilter.set('JUEZ_EFIMERO');
    expect(component.filteredTemplates().map(t => t.id)).toEqual(['tpl-legacy']);
  });

  it('should filter templates by search query', () => {
    component.searchTerm.set('datascience');
    expect(component.filteredTemplates().length).toBe(1);
    expect(component.filteredTemplates()[0].id).toBe('tpl-1');
  });

  it('should open and close request modal', () => {
    component.openRequestModal();
    expect(component.isRequestModalOpen()).toBe(true);

    component.closeRequestModal();
    expect(component.isRequestModalOpen()).toBe(false);
  });

  it('starts on the catalog tab and can switch to my requests', () => {
    expect(component.activeTab()).toBe('catalog');

    component.setTab('requests');
    expect(component.activeTab()).toBe('requests');

    httpMock.expectOne('/api/v1/teacher/environment-requests').flush({ data: [] });
  });

  it('resolves the full stored stack in the catalog cards, not only the main language', () => {
    const fullStack: PublishedTemplate = {
      ...mockTemplates[0],
      name: 'Entorno Full-Stack JS',
      docker_image: 'solv-local/stack:660620ce5745',
      stack_config: {
        runtime_id: 'comp-node',
        web_platform_id: 'comp-nextjs',
        addon_ids: ['comp-prisma'],
        database_id: 'comp-db-none'
      }
    };

    expect(component.techChipsFor(fullStack).map(chip => chip.name)).toEqual([
      'Node.js 20 LTS',
      'Next.js (React Full-Stack)',
      'Prisma ORM'
    ]);
    expect(component.primaryIconFor(fullStack)).toBe('node');

    component.templates.set([fullStack]);
    component.searchTerm.set('prisma');
    expect(component.filteredTemplates().length).toBe(1);
  });

  it('carries the base environment and its pieces when asking for a variant', () => {
    component.requestVariant(mockTemplates[0]);
    expect(component.variantBaseName()).toBe(mockTemplates[0].name);
    expect(component.requestPresetStack()?.runtime_id).toBe('python-312');
    expect(component.isRequestModalOpen()).toBe(true);
  });

  it('clears the base environment and pieces when opening the form from scratch', () => {
    component.requestVariant(mockTemplates[0]);
    component.closeRequestModal();

    component.openRequestModal();
    expect(component.variantBaseName()).toBe('');
    expect(component.requestPresetStack()).toBeNull();
  });

  it('should navigate to exercise creation route from template', () => {
    const router = TestBed.inject(Router);
    const navSpy = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    component.createLabFromTemplate(mockTemplates[0]);
    expect(navSpy).toHaveBeenCalledWith(
      ['/teacher/cursos', 'course-1', 'ejercicios', 'nuevo'],
      { queryParams: { template_id: 'tpl-1', environment_type: 'IDE_PERSISTENTE' } }
    );
  });

  it('should correctly parse nested paginated data responses', () => {
    component.loadTemplates();
    const req = httpMock.expectOne('/api/v1/templates');
    req.flush({
      data: {
        data: mockTemplates,
        meta: { total: 2, page: 1, limit: 10 }
      }
    });

    expect(component.templates().length).toBe(2);
    expect(component.isLoading()).toBe(false);
  });

  it('should open and close detail modal', () => {
    component.openDetailModal(mockTemplates[0]);
    expect(component.selectedTemplateForDetail()).toEqual(expect.objectContaining({ id: 'tpl-1' }));

    component.closeDetailModal();
    expect(component.selectedTemplateForDetail()).toBeNull();
  });

  it('should trigger navigation and close modal when onUseFromDetail is called', () => {
    const router = TestBed.inject(Router);
    const navSpy = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    component.openDetailModal(mockTemplates[0]);
    component.onUseFromDetail(mockTemplates[0]);

    expect(component.selectedTemplateForDetail()).toBeNull();
    expect(navSpy).toHaveBeenCalledWith(
      ['/teacher/cursos', 'course-1', 'ejercicios', 'nuevo'],
      { queryParams: { template_id: 'tpl-1', environment_type: 'IDE_PERSISTENTE' } }
    );
  });
});

describe('TemplateRequestModalComponent', () => {
  let modal: TemplateRequestModalComponent;
  let fixture: ComponentFixture<TemplateRequestModalComponent>;
  let httpMock: HttpTestingController;
  let draftService: EnvironmentRequestDraftService;

  const catalogItems: TemplateComponent[] = [
    {
      id: 'comp-python',
      slug: 'python-312',
      layer: 'runtime',
      sub_category: 'backend',
      name: 'Python 3.12',
      description: 'Runtime Python',
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
      ram_delta_mb: 512,
      is_system: true,
      is_active: true,
      sort_order: 10
    }
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TemplateRequestModalComponent, HttpClientTestingModule],
      providers: [
        {
          provide: TeacherDashboardService,
          useValue: {
            courses: signal<TeacherCourseSummary[]>([
              {
                id: 'course-1',
                code: 'CS101',
                name: 'Autómatas',
                students_count: 15,
                active_now: 0,
                pending_review: 0,
                at_risk: 0
              }
            ])
          }
        }
      ]
    }).compileComponents();

    draftService = TestBed.inject(EnvironmentRequestDraftService);
    draftService.clear();
    httpMock = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(TemplateRequestModalComponent);
    modal = fixture.componentInstance;
    fixture.detectChanges();

    httpMock.expectOne('/api/v1/admin/template-components').flush({ data: catalogItems });
    fixture.detectChanges();
  });

  afterEach(() => {
    draftService.clear();
    httpMock.verify();
  });

  it('renderiza una sola columna sin el panel lateral redundante', () => {
    const el = fixture.nativeElement as HTMLElement;
    expect(el.querySelector('.request-summary')).toBeNull();
    expect(el.querySelector('[data-tour="template-request-tools"]')).not.toBeNull();
    expect(el.querySelector('[data-tour="template-request-actions"]')).not.toBeNull();
  });

  it('permite enviar con herramientas del catálogo sin exigir nota manual y la exige si falta una pieza', () => {
    expect(modal.canSubmit()).toBe(false);

    modal.addPiece(catalogItems[0]);
    httpMock.expectOne('/api/v1/teacher/environment-requests/preview').flush({
      data: {
        suggested_name: 'Laboratorio: Python 3.12',
        target_environment: 'IDE_PERSISTENTE',
        recommended_ram_mb: 512,
        signature: 'sig-py',
        components: [],
        missing_components: []
      }
    });

    expect(modal.requiresDescription()).toBe(false);
    expect(modal.canSubmit()).toBe(true);

    // Si la IA reporta una pieza faltante, limpia runtime anterior, actualiza el título y exige nota >= 10 caracteres.
    modal.aiPrompt.set('un entorno con Brainfuck');
    modal.interpret();
    httpMock.expectOne('/api/v1/teacher/ai/suggest-environment').flush({
      data: {
        name: 'Entorno con Brainfuck',
        description: 'Práctica con Brainfuck',
        runtime_slug: '',
        base_ram_mb: 512,
        rationale: 'Brainfuck no está en el catálogo.',
        missing_components: [{ name: 'Brainfuck', layer: 'runtime', reason: 'Falta intérprete' }]
      }
    });
    fixture.detectChanges();

    const previewReq = httpMock.expectOne('/api/v1/teacher/environment-requests/preview');
    expect(previewReq.request.body.missing_components).toEqual(['Brainfuck']);
    previewReq.flush({
      data: {
        suggested_name: '',
        target_environment: 'IDE_PERSISTENTE',
        recommended_ram_mb: 0,
        signature: 'sig-bf',
        components: [],
        missing_components: [],
        existing_template: { id: 'tpl-1', name: 'Falso positivo' }
      }
    });
    fixture.detectChanges();

    expect(modal.runtimeId()).toBe('');
    expect(modal.title()).toBe('Entorno con Brainfuck');
    expect(modal.titleTouched()).toBe(false);
    expect(modal.requiresDescription()).toBe(true);
    expect((fixture.nativeElement as HTMLElement).textContent).not.toContain('Ya existe este entorno');

    // Al volver a consultar con otro prompt, el título sugerido se actualiza porque titleTouched sigue en false.
    modal.aiPrompt.set('un entorno con Haskell');
    modal.interpret();
    httpMock.expectOne('/api/v1/teacher/ai/suggest-environment').flush({
      data: {
        name: 'Entorno con Haskell',
        description: 'Programación funcional',
        runtime_slug: '',
        base_ram_mb: 512,
        rationale: 'Haskell no está en el catálogo.',
        missing_components: [{ name: 'Haskell', layer: 'runtime', reason: 'Falta GHC' }]
      }
    });
    fixture.detectChanges();
    httpMock.expectOne('/api/v1/teacher/environment-requests/preview').flush({
      data: {
        suggested_name: '',
        target_environment: 'IDE_PERSISTENTE',
        recommended_ram_mb: 0,
        signature: 'sig-hs',
        components: [],
        missing_components: []
      }
    });

    expect(modal.title()).toBe('Entorno con Haskell');
  });

  it('limpia el borrador completo al pulsar clearDraft', () => {
    modal.aiPrompt.set('entorno con Python');
    modal.onTitleChange('Mi entorno');
    modal.clearDraft();

    expect(modal.aiPrompt()).toBe('');
    expect(modal.title()).toBe('');
    expect(modal.titleTouched()).toBe(false);
    expect(draftService.hasContent()).toBe(false);
  });
});
