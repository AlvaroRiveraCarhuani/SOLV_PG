import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { of } from 'rxjs';
import { AdminTemplatesComponent } from './admin-templates.component';
import {
  AdminTemplatesService,
  AdminTemplateItem,
  CreateOfficialTemplateDTO,
  ReviewTemplateDTO,
  TemplateComponent
} from '../services/admin-templates.service';

describe('AdminTemplatesComponent Spec', () => {
  let component: AdminTemplatesComponent;
  let fixture: ComponentFixture<AdminTemplatesComponent>;

  const mockTemplates: AdminTemplateItem[] = [
    {
      id: 'tpl-1',
      name: 'Python 3 Data Science',
      docker_image: 'python:3.12-slim',
      base_ram_mb: 1024,
      status: 'APROBADA',
      target_environment: 'IDE_PERSISTENTE',
      created_at: '2026-09-23T00:00:00Z'
    },
    {
      id: 'tpl-2',
      name: 'Node.js Backend',
      docker_image: 'node:20-slim',
      base_ram_mb: 512,
      status: 'PENDIENTE_AUDITORIA',
      target_environment: 'IDE_PERSISTENTE',
      created_at: '2026-09-23T00:00:00Z'
    },
    {
      id: 'tpl-3',
      name: 'Insecure C++ Lab',
      docker_image: 'gcc:latest',
      base_ram_mb: 256,
      status: 'RECHAZADA',
      rejection_reason: 'Utiliza tag latest no permitido',
      target_environment: 'IDE_PERSISTENTE',
      created_at: '2026-09-23T00:00:00Z'
    },
    {
      id: 'tpl-4',
      name: 'Pausada Lab',
      docker_image: 'golang:1.24',
      base_ram_mb: 512,
      status: 'paused',
      rejection_reason: 'Mantenimiento preventivo',
      target_environment: 'IDE_PERSISTENTE',
      created_at: '2026-09-23T00:00:00Z'
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

  let mockTemplatesService: {
    getTemplates: ReturnType<typeof vi.fn>;
    getTechCatalog: ReturnType<typeof vi.fn>;
    getCategories: ReturnType<typeof vi.fn>;
    getRuntimeCapabilities: ReturnType<typeof vi.fn>;
    reviewTemplate: ReturnType<typeof vi.fn>;
    promoteToModel: ReturnType<typeof vi.fn>;
    duplicateTemplate: ReturnType<typeof vi.fn>;
    getModels: ReturnType<typeof vi.fn>;
    getLocalImages: ReturnType<typeof vi.fn>;
    getDraft: ReturnType<typeof vi.fn>;
    getTemplatesCacheStatus: ReturnType<typeof vi.fn>;
    warmTemplate: ReturnType<typeof vi.fn>;
    warmAllTemplates: ReturnType<typeof vi.fn>;
    triggerAudit: ReturnType<typeof vi.fn>;
    updateOfficialTemplate: ReturnType<typeof vi.fn>;
    updateTemplateQuota: ReturnType<typeof vi.fn>;
  };

  let routerMock: { navigate: ReturnType<typeof vi.fn> };
  let queryParamNueva: string | null = null;

  async function createFixture(): Promise<void> {
    mockTemplatesService = {
      updateOfficialTemplate: vi.fn().mockImplementation((id: string, dto: CreateOfficialTemplateDTO) => of({
        ...mockTemplates.find(t => t.id === id),
        ...dto
      })),
      updateTemplateQuota: vi.fn().mockImplementation((id: string, dto: { base_ram_mb: number; description?: string }) => of({
        ...mockTemplates.find(t => t.id === id),
        base_ram_mb: dto.base_ram_mb,
        description: dto.description
      })),
      getTemplates: vi.fn().mockReturnValue(of([...mockTemplates])),
      getTechCatalog: vi.fn().mockReturnValue(of([...mockTechCatalog])),
      getCategories: vi.fn().mockReturnValue(of([])),
      getRuntimeCapabilities: vi.fn().mockReturnValue(of({
        host_memory: { total_ram_mb: 16384, available_ram_mb: 8192, used_ram_mb: 8192, cpu_cores: 8 },
        satellite_services: [],
        ide_presets: [],
        judge_presets: [],
        max_allowed_ram_mb: 4096
      })),
      reviewTemplate: vi.fn().mockImplementation((id: string, dto: ReviewTemplateDTO) => of({
        ...mockTemplates.find(t => t.id === id),
        status: dto.status,
        base_ram_mb: dto.base_ram_mb || 512
      })),
      promoteToModel: vi.fn().mockReturnValue(of({ id: 'm-1' })),
      duplicateTemplate: vi.fn().mockReturnValue(of({ ...mockTemplates[0], id: 'dup-1', name: '(Copia) ' + mockTemplates[0].name })),
      getModels: vi.fn().mockReturnValue(of([])),
      getLocalImages: vi.fn().mockReturnValue(of({ images: [], usage_map: {} })),
      getDraft: vi.fn().mockReturnValue(of(null)),
      getTemplatesCacheStatus: vi.fn().mockReturnValue(of([
        {
          template_id: 'tpl-1',
          template_name: 'Python 3 Data Science',
          docker_image: 'python:3.12-slim',
          is_cached: true,
          size_bytes: 125000000,
          status: 'cached'
        }
      ])),
      warmTemplate: vi.fn().mockImplementation((id: string) => of({
        template_id: id,
        template_name: 'Template ' + id,
        docker_image: 'img:1.0',
        is_cached: true,
        size_bytes: 250000000,
        status: 'cached'
      })),
      warmAllTemplates: vi.fn().mockReturnValue(of({
        total_checked: 2,
        already_warm: 1,
        warmed_count: 1,
        failed_count: 0
      })),
      triggerAudit: vi.fn().mockImplementation((id: string) => of({
        ...mockTemplates.find(t => t.id === id),
        smoke_test_status: 'pending',
        security_audit_status: 'pending'
      }))
    };

    await TestBed.configureTestingModule({
      imports: [AdminTemplatesComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: mockTemplatesService },
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { queryParamMap: { get: (key: string) => (key === 'nueva' ? queryParamNueva : null) } } }
        },
        { provide: Router, useValue: routerMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminTemplatesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }

  beforeEach(async () => {
    routerMock = { navigate: vi.fn() };
    queryParamNueva = null;
    await createFixture();
  });

  it('el listado renderiza estados v2 con badges correctos en cada pestaña', () => {
    // 1. Pestaña de auditoría / pendientes (en vista de tarjetas)
    component.activeTab.set('pending');
    component.viewMode.set('cards');
    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    const pendingBadge = compiled.querySelector('.badge-status-audit-pending');
    expect(pendingBadge?.textContent).toContain('Auditoría');

    // 2. Pestaña de catálogo (Activa y Pausada en tabla)
    component.activeTab.set('catalog');
    component.viewMode.set('table');
    fixture.detectChanges();
    const approvedBadge = compiled.querySelector('.badge-status-approved');
    const pausedBadge = compiled.querySelector('.badge-status-paused');
    expect(approvedBadge?.textContent).toContain('Activa');
    expect(pausedBadge?.textContent).toContain('Pausada');

    // 3. Contadores reactivos reflejan estados v2
    expect(component.pendingCount()).toBe(1);
    expect(component.activeCount()).toBe(1);
    expect(component.pausedCount()).toBe(1);
    expect(component.rejectedCount()).toBe(1);
  });

  it('la acción "Promover a modelo" solo aparece en filas con estado APROBADA / approved', () => {
    component.activeTab.set('catalog');
    component.viewMode.set('table');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const rows = compiled.querySelectorAll('tbody tr');
    expect(rows.length).toBe(2);

    // Fila 1 (tpl-1: APROBADA): abrir menú de fila y verificar acción de promoción
    component.toggleRowMenu('tpl-1', new MouseEvent('click', { bubbles: true }));
    fixture.detectChanges();
    const firstRowText = rows[0].textContent || '';
    expect(firstRowText).toContain('Python 3 Data Science');
    const firstRowPromoteBtn = rows[0].querySelector('.menu-item--promote');
    expect(firstRowPromoteBtn).not.toBeNull();

    // Fila 2 (tpl-4: paused): su menú NO contiene promoción, contiene Reactivar
    component.toggleRowMenu('tpl-4', new MouseEvent('click', { bubbles: true }));
    fixture.detectChanges();
    const secondRowPromoteBtn = rows[1].querySelector('.menu-item--promote');
    expect(secondRowPromoteBtn).toBeNull();
    const secondRowReactivateBtn = rows[1].querySelector('.menu-item--success');
    expect(secondRowReactivateBtn).not.toBeNull();
  });

  it('suspender pide motivo: motivo < 10 caracteres bloquea la acción y >= 10 la habilita enviándolo en el payload', () => {
    const tplApproved = mockTemplates[0]; // status APROBADA
    component.requestToggleStatus(tplApproved);
    expect(component.templateToToggleStatus()).toEqual(tplApproved);

    // 1. Intentar suspender sin motivo o con menos de 10 caracteres
    component.toggleStatusReason.set('Corta');
    component.confirmToggleStatus(tplApproved);
    expect(mockTemplatesService.reviewTemplate).not.toHaveBeenCalled();

    component.toggleStatusReason.set('123456789'); // 9 caracteres
    component.confirmToggleStatus(tplApproved);
    expect(mockTemplatesService.reviewTemplate).not.toHaveBeenCalled();

    // 2. Con >= 10 caracteres se habilita y el motivo viaja en el payload como rejection_reason
    const validReason = 'Mantenimiento de dependencias críticas';
    component.toggleStatusReason.set(validReason);
    component.confirmToggleStatus(tplApproved);

    expect(mockTemplatesService.reviewTemplate).toHaveBeenCalledWith(tplApproved.id, {
      status: 'paused',
      rejection_reason: validReason
    });
  });

  it('reenviar una rechazada avisa que vuelve a auditoría y no al catálogo', () => {
    const tplRejected = mockTemplates[2];
    mockTemplatesService.reviewTemplate.mockReturnValueOnce(of({
      ...tplRejected,
      status: 'PENDIENTE_AUDITORIA',
      rejection_reason: undefined
    }));

    component.confirmToggleStatus(tplRejected);

    expect(mockTemplatesService.reviewTemplate).toHaveBeenCalledWith(tplRejected.id, {
      status: 'approved',
      rejection_reason: undefined
    });
    expect(component.toast()?.message).toContain('enviada a auditoría');
    expect(component.toast()?.message).toContain('Volverá al catálogo si pasa');
    expect(component.toast()?.message).not.toContain('reactivada en el catálogo');
  });

  it('el KPI de activas en catálogo y promedio RAM excluyen las plantillas pausadas/suspendidas', () => {
    // Activas en catálogo: solo tpl-1 (APROBADA) -> 1
    expect(component.activeCount()).toBe(1);

    // Promedio RAM: solo tpl-1 (1024 MB) -> 1024 MB
    expect(component.averageRam()).toBe(1024);
  });

  it('no redirige al constructor al cargar sin el parámetro nueva', () => {
    expect(routerMock.navigate).not.toHaveBeenCalled();
  });

  it('redirige al constructor de plantillas al llegar con ?nueva=1', async () => {
    queryParamNueva = '1';
    TestBed.resetTestingModule();
    await createFixture();

    expect(routerMock.navigate).toHaveBeenCalledWith(['/admin/plantillas/nueva']);
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('template-create-modal')).toBeNull();
  });

  it('carga los estados de caché de las plantillas y los formatea correctamente', () => {
    const status = component.getCacheStatus('tpl-1');
    expect(status).toBeDefined();
    expect(status?.is_cached).toBe(true);
    expect(component.formatBytes(status?.size_bytes || 0)).toBe('119.2 MB');
  });

  it('permite calentar una plantilla individual y dispara notificación', () => {
    const target = mockTemplates[1]; // tpl-2
    component.warmTemplate(target);

    expect(mockTemplatesService.warmTemplate).toHaveBeenCalledWith('tpl-2');
    expect(component.toast()?.type).toBe('success');
    expect(component.toast()?.message).toContain('Entorno listo en el servidor');
  });

  it('permite ejecutar calentamiento masivo de caché pre-clase', () => {
    component.warmAllTemplates();

    expect(mockTemplatesService.warmAllTemplates).toHaveBeenCalled();
    expect(component.toast()?.type).toBe('success');
    expect(component.toast()?.message).toContain('Entornos preparados');
  });

  it('getPurposeBadge retorna la etiqueta y variante correcta para cada propósito', () => {
    expect(component.getPurposeBadge('IDE_PERSISTENTE')).toEqual({
      label: 'Entorno de Desarrollo',
      shortLabel: 'DESARROLLO',
      variant: 'ide'
    });
    expect(component.getPurposeBadge('JUEZ_EFIMERO')).toEqual({
      label: 'Entorno de Evaluación',
      shortLabel: 'EVALUACIÓN',
      variant: 'judge'
    });
    expect(component.getPurposeBadge('AMBOS')).toEqual({
      label: 'Entorno de Desarrollo y Evaluación',
      shortLabel: 'DESARROLLO Y EVALUACIÓN',
      variant: 'dual'
    });
    // Un valor sin modalidad no se presenta como Desarrollo ni como Evaluación.
    expect(component.getPurposeBadge()).toEqual({
      label: 'Modalidad sin reconocer',
      shortLabel: 'SIN RECONOCER',
      variant: 'unknown'
    });
    expect(component.getPurposeBadge('OTRO_VALOR')).toEqual({
      label: 'Modalidad sin reconocer',
      shortLabel: 'SIN RECONOCER',
      variant: 'unknown'
    });
  });

  it('el filtro de propósito restringe las plantillas mostradas según el target_environment', () => {
    // Agregamos plantillas con distintos propósitos a allTemplates
    const dualTemplate: AdminTemplateItem = {
      id: 'tpl-dual',
      name: 'Entorno Dual Python',
      docker_image: 'python:3.12-slim',
      base_ram_mb: 1024,
      status: 'APROBADA',
      target_environment: 'AMBOS',
      created_at: '2026-09-23T00:00:00Z'
    };
    const judgeTemplate: AdminTemplateItem = {
      id: 'tpl-judge',
      name: 'Evaluador Python',
      docker_image: 'python:3.12-slim',
      base_ram_mb: 256,
      status: 'APROBADA',
      target_environment: 'JUEZ_EFIMERO',
      created_at: '2026-09-23T00:00:00Z'
    };

    component.allTemplates.set([...mockTemplates, dualTemplate, judgeTemplate]);

    // Con filtro 'ALL', el catálogo incluye todas las aprobadas/pausadas (tpl-1, tpl-4, tpl-dual, tpl-judge)
    component.selectedPurpose.set('ALL');
    expect(component.catalogTemplates().length).toBe(4);

    // Con filtro 'AMBOS', solo devuelve la plantilla dual
    component.selectedPurpose.set('AMBOS');
    expect(component.catalogTemplates().length).toBe(1);
    expect(component.catalogTemplates()[0].id).toBe('tpl-dual');

    // Con filtro 'JUEZ_EFIMERO', devuelve el evaluador y la plantilla dual
    component.selectedPurpose.set('JUEZ_EFIMERO');
    expect(component.catalogTemplates().map(t => t.id).sort()).toEqual(['tpl-dual', 'tpl-judge']);

    // Con filtro 'IDE_PERSISTENTE', devuelve tpl-1, tpl-4 y la plantilla dual
    component.selectedPurpose.set('IDE_PERSISTENTE');
    expect(component.catalogTemplates().map(t => t.id).sort()).toEqual(['tpl-1', 'tpl-4', 'tpl-dual']);
  });

  it('no confunde un stack JavaScript local con C al inferir la tecnología principal', () => {
    const fullStackJs: AdminTemplateItem = {
      id: 'tpl-fullstack-js',
      name: 'Entorno Full-Stack JS',
      docker_image: 'solv-local/stack:660620ce5745',
      base_ram_mb: 1024,
      status: 'APROBADA',
      target_environment: 'IDE_PERSISTENTE',
      created_at: '2026-11-05T00:00:00Z'
    };

    const tech = component.getTemplateTech(fullStackJs);
    expect(tech.iconType).toBe('node');
    expect(tech.iconType).not.toBe('c');
  });

  it('las tarjetas muestran todas las tecnologías del stack con su icono del catálogo', () => {
    const fullStackJs: AdminTemplateItem = {
      id: 'tpl-fullstack-js',
      name: 'Entorno Full-Stack JS',
      docker_image: 'solv-local/stack:660620ce5745',
      base_ram_mb: 1024,
      status: 'APROBADA',
      target_environment: 'IDE_PERSISTENTE',
      stack_config: {
        runtime_id: 'comp-node',
        web_platform_id: 'comp-nextjs',
        addon_ids: ['comp-prisma'],
        database_id: 'comp-db-none'
      },
      created_at: '2026-11-05T00:00:00Z'
    };

    const chips = component.getStackTechs(fullStackJs);
    expect(chips.map(chip => chip.name)).toEqual([
      'Node.js 20 LTS',
      'Next.js (React Full-Stack)',
      'Prisma ORM'
    ]);
    expect(chips.map(chip => chip.iconKey)).toEqual(['node', 'nextjs', 'prisma']);
    expect(component.getPrimaryTechIcon(fullStackJs)).toBe('node');
  });

  it('detectTemplateTech prioriza stack_config.runtime_id y web_platform_id sobre el nombre genérico', () => {
    const itemWithConfig: AdminTemplateItem = {
      id: 'tpl-custom',
      name: 'Laboratorio Especializado de Algoritmos',
      docker_image: 'custom/runner:1.0',
      base_ram_mb: 512,
      status: 'APROBADA',
      stack_config: {
        runtime_id: 'rust-178'
      },
      created_at: '2026-09-23T00:00:00Z'
    };

    const tech = component.getTemplateTech(itemWithConfig);
    expect(tech.iconType).toBe('rust');
    expect(tech.name).toBe('Rust');
  });

  it('handleSaveEdit ajusta la cuota sin exigir una verificación de build', () => {
    const target = mockTemplates[0]; // tpl-1 (APROBADA, 1024 MB)
    component.openEditModal(target);
    expect(component.selectedTemplateForEdit()).toEqual(target);

    component.handleSaveEdit({
      id: target.id,
      base_ram_mb: 2048,
      description: 'Descripción actualizada con cuotas ampliadas'
    });

    expect(mockTemplatesService.updateTemplateQuota).toHaveBeenCalledWith(target.id, {
      base_ram_mb: 2048,
      description: 'Descripción actualizada con cuotas ampliadas'
    });
    expect(mockTemplatesService.updateOfficialTemplate).not.toHaveBeenCalled();

    // Modal cerrado y toast disparado
    expect(component.selectedTemplateForEdit()).toBeNull();
    expect(component.toast()?.type).toBe('success');
    expect(component.toast()?.message).toContain('actualizadas');
  });
  it('limita la tarjeta a 3 tecnologías y resume el resto en un chip +N más', () => {
    const extraComponents: TemplateComponent[] = [
      { ...mockTechCatalog[2], id: 'comp-vite', slug: 'vite', name: 'Vite', icon_key: 'vite' },
      { ...mockTechCatalog[2], id: 'comp-redis', slug: 'redis', name: 'Redis', icon_key: 'redis' }
    ];
    const manyTech: AdminTemplateItem = {
      id: 'tpl-many-tech',
      name: 'Laboratorio: Node.js, Next.js y servicios',
      docker_image: 'solv-local/stack:abc123',
      base_ram_mb: 1024,
      status: 'APROBADA',
      target_environment: 'IDE_PERSISTENTE',
      description: 'Stack completo para el laboratorio de desarrollo web',
      stack_config: {
        runtime_id: 'comp-node',
        web_platform_id: 'comp-nextjs',
        addon_ids: ['comp-prisma', 'comp-vite', 'comp-redis']
      },
      created_at: '2026-09-23T00:00:00Z'
    };

    component.allTemplates.set([manyTech]);
    component.techCatalog.set([...mockTechCatalog, ...extraComponents]);
    component.activeTab.set('catalog');
    component.viewMode.set('cards');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const card = compiled.querySelector('.template-card') as HTMLElement;
    expect(card).not.toBeNull();
    expect(card.querySelectorAll('.card-stack .stack-item:not(.stack-more)').length).toBe(3);
    const more = card.querySelector('.card-stack .stack-more') as HTMLElement;
    expect(more.textContent?.replace(/\s+/g, ' ').trim()).toBe('+2 más');
    // El detalle completo vive en el panel lateral, no en un acordeón inline.
    expect(card.querySelector('details')).toBeNull();
  });

  it('muestra el estado de soporte en español con el mensaje real del backend', () => {
    const eolTemplate: AdminTemplateItem = {
      id: 'tpl-go-runner',
      name: 'Go Runner Base',
      docker_image: 'golang:1.21-bookworm',
      base_ram_mb: 512,
      status: 'APROBADA',
      target_environment: 'JUEZ_EFIMERO',
      eol_status: 'eol',
      eol_date: '2024-02-06',
      eol_message: 'Versión sin soporte oficial (finalizó el 2024-02-06)',
      created_at: '2026-09-23T00:00:00Z'
    };
    const warningTemplate: AdminTemplateItem = {
      id: 'tpl-python-warning',
      name: 'Python 3.10 Base',
      docker_image: 'python:3.10-slim',
      base_ram_mb: 512,
      status: 'APROBADA',
      target_environment: 'IDE_PERSISTENTE',
      eol_status: 'warning',
      eol_message: 'La versión deja de recibir parches el 2026-12-31',
      created_at: '2026-09-23T00:00:00Z'
    };

    component.allTemplates.set([eolTemplate, warningTemplate]);
    component.activeTab.set('catalog');
    component.viewMode.set('cards');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const cards = compiled.querySelectorAll('.template-card');
    const eolPill = cards[0].querySelector('.eol-pill') as HTMLElement;
    expect(eolPill.textContent).toContain('Soporte vencido');
    expect(eolPill.textContent).not.toContain('EOL');
    expect(eolPill.getAttribute('title')).toBe('Versión sin soporte oficial (finalizó el 2024-02-06)');

    const warningPill = cards[1].querySelector('.eol-pill') as HTMLElement;
    expect(warningPill.textContent).toContain('Soporte por vencer');
    expect(warningPill.classList.contains('is-warning')).toBe(true);
    expect(warningPill.getAttribute('title')).toBe('La versión deja de recibir parches el 2026-12-31');
  });

  it('abre el panel lateral de detalles técnicos con los datos reales de la plantilla', () => {
    const detailed: AdminTemplateItem = {
      id: 'tpl-detail',
      name: 'Laboratorio: Node.js y Next.js',
      docker_image: 'solv-local/stack:abc123',
      base_ram_mb: 1024,
      status: 'APROBADA',
      target_environment: 'AMBOS',
      category_name: 'Desarrollo Web',
      resource_profile: { min_mb: 512, high_mb: 1024, max_mb: 2048 },
      timeout_ms: 15000,
      published_exercise_count: 7,
      tools_declared: ['git', 'curl'],
      entrypoint: 'node /judge/main.js',
      stack_config: {
        runtime_id: 'comp-node',
        web_platform_id: 'comp-nextjs',
        addon_ids: ['comp-prisma'],
        dockerfile_content: 'FROM node:20-bookworm-slim\nRUN npm ci'
      },
      created_at: '2026-09-23T00:00:00Z'
    };

    component.allTemplates.set([detailed]);
    component.activeTab.set('catalog');
    component.viewMode.set('cards');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const trigger = compiled.querySelector('.card-details-trigger') as HTMLButtonElement;
    trigger.click();
    fixture.detectChanges();

    expect(component.detailsTemplate()?.id).toBe('tpl-detail');
    const panel = compiled.querySelector('drawer .drawer-panel') as HTMLElement;
    expect(panel).not.toBeNull();
    const panelText = panel.textContent || '';
    expect(panelText).toContain('solv-local/stack:abc123');
    expect(panelText).toContain('node /judge/main.js');
    expect(panelText).toContain('git');
    expect(panelText).toContain('curl');
    expect(panelText).toContain('Desarrollo Web');
    expect(panelText).toContain('15000 ms');
    expect(panelText).toContain('FROM node:20-bookworm-slim');
    expect(panelText).toContain('Editar en Constructor');

    // El cuerpo de la tarjeta también abre el panel.
    component.closeDetailsDrawer();
    fixture.detectChanges();
    const body = compiled.querySelector('.card-stack') as HTMLElement;
    body.click();
    fixture.detectChanges();
    expect(component.detailsTemplate()?.id).toBe('tpl-detail');

    // El botón Editar no debe abrir el panel: su clic no burbujea como clic de tarjeta.
    component.closeDetailsDrawer();
    fixture.detectChanges();
    const editButton = compiled.querySelector('.card-footer-actions button[title="Abrir en el Constructor de Entornos"]') as HTMLButtonElement;
    editButton.click();
    fixture.detectChanges();
    expect(component.detailsTemplate()).toBeNull();
    expect(routerMock.navigate).toHaveBeenCalledWith(['/admin/plantillas', 'tpl-detail', 'editar']);
  });

  it('traduce los estados de auditoría del panel lateral a español', () => {
    const audited: AdminTemplateItem = {
      id: 'tpl-audit',
      name: 'Stack auditado',
      docker_image: 'solv-local/stack:audit01',
      base_ram_mb: 512,
      status: 'APROBADA',
      target_environment: 'AMBOS',
      smoke_test_status: 'passed',
      security_audit_status: 'failed',
      created_at: '2026-09-23T00:00:00Z'
    };

    component.allTemplates.set([audited]);
    component.activeTab.set('catalog');
    component.viewMode.set('cards');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const trigger = compiled.querySelector('.card-details-trigger') as HTMLButtonElement;
    trigger.click();
    fixture.detectChanges();

    const panel = compiled.querySelector('drawer .drawer-panel') as HTMLElement;
    const panelText = panel.textContent || '';
    expect(panelText).toContain('Prueba de arranque');
    expect(panelText).toContain('Aprobada');
    expect(panelText).toContain('Escaneo de seguridad');
    expect(panelText).toContain('Fallido');
    expect(panelText).not.toContain('passed');
    expect(panelText).not.toContain('failed');

    // Todos los estados del backend tienen etiqueta en español.
    expect(component.auditStatusLabel('passed', true)).toBe('Aprobada');
    expect(component.auditStatusLabel('passed')).toBe('Aprobado');
    expect(component.auditStatusLabel('warning')).toBe('Con advertencias');
    expect(component.auditStatusLabel('failed', true)).toBe('Fallida');
    expect(component.auditStatusLabel('failed')).toBe('Fallido');
    expect(component.auditStatusLabel('pending')).toBe('En análisis');
    expect(component.auditStatusLabel('pending', false, true)).toBe('Sin analizar');
    expect(component.auditStatusLabel('skipped')).toBe('No aplica');
    expect(component.auditStatusLabel('error')).toBe('Error de escáner');
    expect(component.auditStatusLabel(undefined)).toBe('Sin analizar');
    expect(component.auditStatusTooltip('skipped')).toContain('Sin binarios declarados');
    expect(component.auditStatusTooltip('failed')).toBe('');
  });

  it('muestra la pastilla de CVE críticas en la tarjeta del catálogo', () => {
    const risky: AdminTemplateItem = {
      id: 'tpl-riesgo',
      name: 'Go Runner Base',
      docker_image: 'golang:1.24-alpine',
      base_ram_mb: 512,
      status: 'APROBADA',
      target_environment: 'JUEZ_EFIMERO',
      cve_critical_count: 2,
      cve_high_count: 478,
      eol_status: 'eol',
      eol_date: '2026-02-10',
      eol_message: 'Versión sin soporte oficial (finalizó el 2026-02-10)',
      created_at: '2026-09-23T00:00:00Z'
    };

    component.allTemplates.set([risky]);
    component.activeTab.set('catalog');
    component.viewMode.set('cards');
    fixture.detectChanges();

    const card = (fixture.nativeElement as HTMLElement).querySelector('.template-card') as HTMLElement;
    const cvePill = card.querySelector('.cve-pill') as HTMLElement;
    expect(cvePill).not.toBeNull();
    expect(cvePill.textContent).toContain('2 CVE críticas');
    // La pastilla de soporte sigue al lado, sin repetir información.
    expect(card.querySelector('.eol-pill')?.textContent).toContain('Soporte vencido');
  });

  it('el banner EOL no duplica la acción ni la fecha', () => {
    const eolTemplate: AdminTemplateItem = {
      id: 'tpl-eol',
      name: 'Python 3.8 Base',
      docker_image: 'python:3.8-slim',
      base_ram_mb: 512,
      status: 'APROBADA',
      target_environment: 'IDE_PERSISTENTE',
      eol_status: 'eol',
      eol_date: '2026-02-10',
      eol_message: 'Versión sin soporte oficial (finalizó el 2026-02-10)',
      created_at: '2026-09-23T00:00:00Z'
    };

    component.allTemplates.set([eolTemplate]);
    component.activeTab.set('catalog');
    component.viewMode.set('cards');
    fixture.detectChanges();

    const trigger = (fixture.nativeElement as HTMLElement).querySelector('.card-details-trigger') as HTMLButtonElement;
    trigger.click();
    fixture.detectChanges();

    const banner = (fixture.nativeElement as HTMLElement).querySelector('.details-alert') as HTMLElement;
    expect(banner.textContent).toContain('Soporte vencido');
    // Una sola mención de la fecha, en formato canónico del pipe.
    expect(banner.textContent?.match(/10\/02\/2026/g)?.length).toBe(1);
    expect(banner.textContent).not.toContain('2026-02-10');
    // La acción de edición vive solo en el pie del drawer.
    expect(banner.querySelector('button')).toBeNull();
  });

  it('el drawer re-escanea y refleja el estado en español', () => {
    const audited: AdminTemplateItem = {
      id: 'tpl-audit-action',
      name: 'Stack con críticas',
      docker_image: 'solv-local/stack:aaa111',
      base_ram_mb: 768,
      status: 'APROBADA',
      target_environment: 'IDE_PERSISTENTE',
      smoke_test_status: 'failed',
      smoke_test_output: 'Faltan las siguientes herramientas declaradas en la imagen: pip',
      security_audit_status: 'failed',
      cve_critical_count: 2,
      cve_high_count: 5,
      security_audited_at: '2026-11-20T10:30:00Z',
      created_at: '2026-09-23T00:00:00Z'
    };

    component.allTemplates.set([audited]);
    component.activeTab.set('catalog');
    component.viewMode.set('cards');
    fixture.detectChanges();

    (fixture.nativeElement as HTMLElement).querySelector<HTMLButtonElement>('.card-details-trigger')?.click();
    fixture.detectChanges();

    const panel = (fixture.nativeElement as HTMLElement).querySelector('drawer .drawer-panel') as HTMLElement;
    const panelText = panel.textContent || '';
    expect(panelText).toContain('Fallida');
    expect(panelText).toContain('Último análisis');
    expect(panelText).toContain('20/11/2026');
    expect(panelText).toContain('Salida de la prueba de arranque');
    expect(panelText).toContain('Faltan las siguientes herramientas');

    const rescan = panel.querySelector('.details-audit-actions button') as HTMLButtonElement;
    expect(rescan.textContent).toContain('Re-escanear');
    expect(rescan.disabled).toBe(false);
    rescan.click();
    fixture.detectChanges();

    expect(mockTemplatesService.triggerAudit).toHaveBeenCalledWith('tpl-audit-action');
    expect(component.isRescanning('tpl-audit-action')).toBe(false);
    // La respuesta del backend deja el drawer en «En análisis».
    expect(panel.textContent).toContain('En análisis');
  });

  it('el botón de re-escaneo se deshabilita mientras la auditoría está pendiente', () => {
    const pending: AdminTemplateItem = {
      id: 'tpl-en-cola',
      name: 'Stack en cola',
      docker_image: 'solv-local/stack:bbb222',
      base_ram_mb: 512,
      status: 'APROBADA',
      target_environment: 'IDE_PERSISTENTE',
      smoke_test_status: 'passed',
      security_audit_status: 'pending',
      created_at: '2026-09-23T00:00:00Z'
    };

    component.allTemplates.set([pending]);
    component.activeTab.set('catalog');
    component.viewMode.set('cards');
    fixture.detectChanges();

    (fixture.nativeElement as HTMLElement).querySelector<HTMLButtonElement>('.card-details-trigger')?.click();
    fixture.detectChanges();

    const rescan = (fixture.nativeElement as HTMLElement).querySelector('.details-audit-actions button') as HTMLButtonElement;
    expect(rescan.textContent).toContain('Analizando...');
    expect(rescan.disabled).toBe(true);
  });

  it('refresca cada 5 s solo mientras hay auditorías en curso', () => {
    vi.useFakeTimers();
    try {
      const inProgress: AdminTemplateItem = {
        id: 'tpl-poll',
        name: 'Stack auditando',
        docker_image: 'solv-local/stack:ccc333',
        base_ram_mb: 512,
        status: 'APROBADA',
        target_environment: 'IDE_PERSISTENTE',
        smoke_test_status: 'pending',
        security_audit_status: 'pending',
        created_at: '2026-09-23T00:00:00Z'
      };
      mockTemplatesService.getTemplates.mockReturnValue(of([inProgress]));

      component.allTemplates.set([inProgress]);
      component.triggerAudit(inProgress);
      const callsAfterTrigger = mockTemplatesService.getTemplates.mock.calls.length;

      vi.advanceTimersByTime(5000);
      expect(mockTemplatesService.getTemplates.mock.calls.length).toBe(callsAfterTrigger + 1);

      // Sin auditorías en curso el refresco se detiene solo.
      mockTemplatesService.getTemplates.mockReturnValue(of([{ ...inProgress, smoke_test_status: 'passed', security_audit_status: 'passed' }]));
      vi.advanceTimersByTime(5000);
      const callsAfterSettle = mockTemplatesService.getTemplates.mock.calls.length;
      vi.advanceTimersByTime(15000);
      expect(mockTemplatesService.getTemplates.mock.calls.length).toBe(callsAfterSettle);
    } finally {
      vi.useRealTimers();
    }
  });

  it('oculta la marca interna SYSTEM_SEED_RUNNER en la descripción', () => {
    expect(component.templateDescription({ description: 'SYSTEM_SEED_RUNNER' })).toBeNull();
    expect(component.templateDescription({ description: '   ' })).toBeNull();
    expect(component.templateDescription({})).toBeNull();
    expect(component.templateDescription({ description: 'Stack de ciencia de datos' })).toBe(
      'Stack de ciencia de datos'
    );
  });

  it('pagina el catálogo oficial y las rechazadas en lotes de 9 tarjetas para alinear la grilla de 3 columnas', () => {
    const approved: AdminTemplateItem[] = Array.from({ length: 13 }, (_, index) => ({
      id: `tpl-cat-${index + 1}`,
      name: `Plantilla ${index + 1}`,
      docker_image: `solv-local/stack:${index + 1}`,
      base_ram_mb: 512,
      status: 'APROBADA',
      target_environment: 'IDE_PERSISTENTE',
      created_at: '2026-09-23T00:00:00Z'
    }));
    const rejected: AdminTemplateItem[] = Array.from({ length: 10 }, (_, index) => ({
      id: `tpl-rej-${index + 1}`,
      name: `Rechazada ${index + 1}`,
      docker_image: `solv-local/rej:${index + 1}`,
      base_ram_mb: 512,
      status: 'RECHAZADA',
      target_environment: 'IDE_PERSISTENTE',
      created_at: '2026-09-23T00:00:00Z'
    }));

    component.allTemplates.set([...approved, ...rejected]);

    expect(component.pageSize()).toBe(9);
    expect(component.totalCatalogPages()).toBe(2);
    expect(component.paginatedCatalogTemplates()).toHaveLength(9);

    component.catalogPage.set(2);
    expect(component.paginatedCatalogTemplates()).toHaveLength(4);

    expect(component.rejectedPageSize()).toBe(9);
    expect(component.totalRejectedPages()).toBe(2);
    expect(component.paginatedRejectedTemplates()).toHaveLength(9);

    component.rejectedPage.set(2);
    expect(component.paginatedRejectedTemplates()).toHaveLength(1);
  });

});

