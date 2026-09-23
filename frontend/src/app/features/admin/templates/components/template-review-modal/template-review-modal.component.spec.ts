import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { TemplateReviewModalComponent } from './template-review-modal.component';
import { AdminTemplatesService, AdminTemplateItem, RuntimeCapabilities } from '../../../services/admin-templates.service';

describe('TemplateReviewModalComponent Spec', () => {
  let component: TemplateReviewModalComponent;
  let fixture: any;

  const mockTemplate: AdminTemplateItem = {
    id: 'tpl-review-1',
    name: 'Python 3.12 Data Science',
    docker_image: 'python:3.12-slim-bookworm',
    base_ram_mb: 512,
    status: 'PENDIENTE_AUDITORIA',
    target_environment: 'IDE_PERSISTENTE',
    created_at: '2026-09-23T00:00:00Z'
  };

  const mockCapabilities: RuntimeCapabilities = {
    host_memory: {
      total_ram_mb: 16384,
      available_ram_mb: 12288,
      used_ram_mb: 4096,
      cpu_cores: 8
    },
    satellite_services: [],
    ide_presets: [
      { mb: 512, label: '512 MB', desc: 'Ligera' },
      { mb: 1024, label: '1 GB', desc: 'Estándar' },
      { mb: 2048, label: '2 GB', desc: 'Intensiva' },
      { mb: 4096, label: '4 GB', desc: 'Datos & IA' }
    ],
    judge_presets: [
      { mb: 128, label: '128 MB', desc: 'Ultra-ligera' },
      { mb: 256, label: '256 MB', desc: 'Recomendada' },
      { mb: 512, label: '512 MB', desc: 'Completa' }
    ],
    max_allowed_ram_mb: 8192
  };

  let mockTemplatesService: {
    getRuntimeCapabilities: any;
  };

  beforeEach(async () => {
    mockTemplatesService = {
      getRuntimeCapabilities: vi.fn().mockReturnValue(of(mockCapabilities))
    };

    await TestBed.configureTestingModule({
      imports: [TemplateReviewModalComponent],
      providers: [
        { provide: AdminTemplatesService, useValue: mockTemplatesService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TemplateReviewModalComponent);
    component = fixture.componentInstance;
    component.template = { ...mockTemplate };
    fixture.detectChanges();
  });

  it('debe inicializar la RAM seleccionada con el base_ram_mb de la plantilla y cargar presets desde /capabilities', () => {
    expect(component.selectedRam()).toBe(512);
    expect(mockTemplatesService.getRuntimeCapabilities).toHaveBeenCalled();
    expect(component.ramPresets().length).toBe(4);
    expect(component.ramPresets()[1].mb).toBe(1024);
  });

  it('cambiar RAM actualiza selectedRam al valor del preset seleccionado', () => {
    component.setRam(2048);
    expect(component.selectedRam()).toBe(2048);

    const compiled = fixture.nativeElement as HTMLElement;
    fixture.detectChanges();
    const activeChip = compiled.querySelector('.chip-btn.active');
    expect(activeChip?.textContent).toContain('2 GB');
  });

  it('aprobar emite el evento approved con el id de la plantilla y la RAM seleccionada', () => {
    let emittedPayload: { id: string; base_ram_mb: number } | undefined;
    component.approved.subscribe((payload) => {
      emittedPayload = payload;
    });

    component.setRam(1024);
    component.confirmApprove();

    expect(emittedPayload).toBeDefined();
    expect(emittedPayload?.id).toBe('tpl-review-1');
    expect(emittedPayload?.base_ram_mb).toBe(1024);
  });

  it('cerrar modal emite el evento closed', () => {
    let closedEmitted = false;
    component.closed.subscribe(() => {
      closedEmitted = true;
    });

    component.closeModal();
    expect(closedEmitted).toBe(true);
  });
});
