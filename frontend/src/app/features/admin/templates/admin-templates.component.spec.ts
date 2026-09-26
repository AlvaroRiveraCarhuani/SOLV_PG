import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { of } from 'rxjs';
import { AdminTemplatesComponent } from './admin-templates.component';
import { AdminTemplatesService, AdminTemplateItem } from '../services/admin-templates.service';

describe('AdminTemplatesComponent Spec', () => {
  let component: AdminTemplatesComponent;
  let fixture: any;

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

  let mockTemplatesService: {
    getTemplates: any;
    getCategories: any;
    getRuntimeCapabilities: any;
    reviewTemplate: any;
    promoteToModel: any;
    duplicateTemplate: any;
    getModels: any;
    getAvailableSatelliteServices: any;
    getLocalImages: any;
    getDraft: any;
  };

  let routerMock: { navigate: any };
  let queryParamNueva: string | null = null;

  async function createFixture(): Promise<void> {
    mockTemplatesService = {
      getTemplates: vi.fn().mockReturnValue(of([...mockTemplates])),
      getCategories: vi.fn().mockReturnValue(of([])),
      getRuntimeCapabilities: vi.fn().mockReturnValue(of({
        host_memory: { total_ram_mb: 16384, available_ram_mb: 8192, used_ram_mb: 8192, cpu_cores: 8 },
        satellite_services: [],
        ide_presets: [],
        judge_presets: [],
        max_allowed_ram_mb: 4096
      })),
      reviewTemplate: vi.fn().mockImplementation((id: string, dto: any) => of({
        ...mockTemplates.find(t => t.id === id),
        status: dto.status,
        base_ram_mb: dto.base_ram_mb || 512
      })),
      promoteToModel: vi.fn().mockReturnValue(of({ id: 'm-1' })),
      duplicateTemplate: vi.fn().mockReturnValue(of({ ...mockTemplates[0], id: 'dup-1', name: '(Copia) ' + mockTemplates[0].name })),
      getModels: vi.fn().mockReturnValue(of([])),
      getAvailableSatelliteServices: vi.fn().mockReturnValue([]),
      getLocalImages: vi.fn().mockReturnValue(of({ images: [], usage_map: {} })),
      getDraft: vi.fn().mockReturnValue(of(null))
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

    // Fila 1 (tpl-1: APROBADA) debe contener el botón de promoción
    const firstRowText = rows[0].textContent || '';
    expect(firstRowText).toContain('Python 3 Data Science');
    const firstRowPromoteBtn = rows[0].querySelector('.btn-edit-action[title*="Promover"]');
    expect(firstRowPromoteBtn).not.toBeNull();

    // Fila 2 (tpl-4: paused) NO debe contener el botón de promoción
    const secondRowPromoteBtn = rows[1].querySelector('.btn-edit-action[title*="Promover"]');
    expect(secondRowPromoteBtn).toBeNull();
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

  it('el KPI de activas en catálogo y promedio RAM excluyen las plantillas pausadas/suspendidas', () => {
    // Activas en catálogo: solo tpl-1 (APROBADA) -> 1
    expect(component.activeCount()).toBe(1);

    // Promedio RAM: solo tpl-1 (1024 MB) -> 1024 MB
    expect(component.averageRam()).toBe(1024);
  });

  it('no abre el modal de creación al cargar sin el parámetro nueva', () => {
    expect(component.showCreateModal()).toBe(false);
    expect(routerMock.navigate).not.toHaveBeenCalled();
  });

  it('abre el wizard de creación al llegar con ?nueva=1 y luego limpia el parámetro', async () => {
    queryParamNueva = '1';
    TestBed.resetTestingModule();
    await createFixture();

    expect(component.showCreateModal()).toBe(true);
    expect(routerMock.navigate).toHaveBeenCalledWith([], {
      queryParams: { nueva: null },
      queryParamsHandling: 'merge',
      replaceUrl: true
    });

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('template-create-modal')).not.toBeNull();
  });
});
