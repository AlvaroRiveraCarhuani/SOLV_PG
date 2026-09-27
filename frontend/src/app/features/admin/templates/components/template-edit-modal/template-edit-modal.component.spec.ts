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
});
