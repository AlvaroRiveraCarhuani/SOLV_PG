import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { TemplateEditModalComponent } from './template-edit-modal.component';
import { AdminTemplateItem } from '../../../services/admin-templates.service';

const mockTemplate: AdminTemplateItem = {
  id: 'tpl-edit-1',
  name: 'Python 3.12 DS',
  docker_image: 'python:3.12-slim',
  base_ram_mb: 1024,
  status: 'APROBADA',
  created_at: '2026-09-01T00:00:00Z',
  description: 'Entorno para ciencia de datos'
};

describe('TemplateEditModalComponent', () => {
  let component: TemplateEditModalComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TemplateEditModalComponent]
    }).compileComponents();

    const fixture = TestBed.createComponent(TemplateEditModalComponent);
    component = fixture.componentInstance;
    component.template = { ...mockTemplate };
    component.ngOnInit();
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('ngOnInit inicializa selectedRam y description desde el template', () => {
    expect(component.selectedRam()).toBe(1024);
    expect(component.description()).toBe('Entorno para ciencia de datos');
  });

  it('ngOnInit usa 512 MB como fallback cuando base_ram_mb es undefined', async () => {
    const fixture = TestBed.createComponent(TemplateEditModalComponent);
    const c = fixture.componentInstance;
    c.template = { ...mockTemplate, base_ram_mb: undefined as any };
    c.ngOnInit();
    expect(c.selectedRam()).toBe(512);
  });

  it('setRam actualiza selectedRam correctamente', () => {
    component.setRam(2048);
    expect(component.selectedRam()).toBe(2048);
  });

  it('confirmSave emite payload con id, ram y descripción recortada', () => {
    let payload: any;
    component.saved.subscribe(p => { payload = p; });

    component.setRam(2048);
    component.description.set('  Descripción actualizada  ');
    component.confirmSave();

    expect(payload).toEqual({
      id: 'tpl-edit-1',
      base_ram_mb: 2048,
      description: 'Descripción actualizada'
    });
  });

  it('confirmSave no emite doble vez cuando isSubmitting está activo', () => {
    let count = 0;
    component.saved.subscribe(() => { count++; });
    component.isSubmitting.set(true);
    component.confirmSave();
    expect(count).toBe(0);
  });

  it('closeModal emite closed cuando no está enviando', () => {
    let closed = false;
    component.closed.subscribe(() => { closed = true; });
    component.closeModal();
    expect(closed).toBe(true);
  });

  it('closeModal no emite cuando isSubmitting está activo', () => {
    let closed = false;
    component.closed.subscribe(() => { closed = true; });
    component.isSubmitting.set(true);
    component.closeModal();
    expect(closed).toBe(false);
  });

  it('calcula minimumFloorMB dinámicamente según ADR-045 (210 MB base, 260 MB con Postgres, 64 MB para juez)', () => {
    // 1. IDE base sin servicios: 210 MB
    expect(component.minimumFloorMB()).toBe(210);

    // 2. IDE con Postgres: 210 + 50 = 260 MB
    component.template = {
      ...mockTemplate,
      services_config: {
        services: [{ category: 'database', engine: 'postgres', version: '16' }]
      }
    };
    expect(component.minimumFloorMB()).toBe(260);

    // 3. Juez efímero: 64 MB
    component.template = {
      ...mockTemplate,
      target_environment: 'JUEZ_EFIMERO'
    };
    expect(component.minimumFloorMB()).toBe(64);
  });

  it('identifica modo AMBOS (isDual) y deriva la cuota de RAM del juez efímero (128 MB / 256 MB)', () => {
    component.template = {
      ...mockTemplate,
      target_environment: 'AMBOS'
    };
    expect(component.isDual()).toBe(true);

    component.setRam(512);
    expect(component.derivedJudgeRamMB()).toBe(128);

    component.setRam(1024);
    expect(component.derivedJudgeRamMB()).toBe(256);

    component.setRam(2048);
    expect(component.derivedJudgeRamMB()).toBe(256);
  });
});
