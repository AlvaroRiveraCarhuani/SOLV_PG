import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TemplateStatusModalComponent } from './template-status-modal.component';
import { AdminTemplateItem } from '../../../services/admin-templates.service';

describe('TemplateStatusModalComponent', () => {
  let component: TemplateStatusModalComponent;
  let fixture: ComponentFixture<TemplateStatusModalComponent>;

  const mockApprovedTemplate: AdminTemplateItem = {
    id: 'tpl-1',
    name: 'Python 3.12 Lab',
    docker_image: 'python:3.12-slim',
    description: 'Entorno de python',
    base_ram_mb: 512,
    status: 'approved',
    created_at: '2026-09-01T00:00:00Z'
  };

  const mockPausedTemplate: AdminTemplateItem = {
    ...mockApprovedTemplate,
    id: 'tpl-2',
    name: 'C++ Lab',
    status: 'paused'
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TemplateStatusModalComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(TemplateStatusModalComponent);
    component = fixture.componentInstance;
  });

  it('debe crearse correctamente', () => {
    component.template = mockApprovedTemplate;
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('debe identificar isPausing como true para plantilla aprobada', () => {
    component.template = mockApprovedTemplate;
    fixture.detectChanges();
    expect(component.isPausing()).toBe(true);
    expect(component.isValid()).toBe(false); // requiere >= 10 caracteres
  });

  it('debe validar motivo mínimo de 10 caracteres al suspender', () => {
    component.template = mockApprovedTemplate;
    fixture.detectChanges();

    component.reason.set('corto');
    expect(component.isValid()).toBe(false);

    component.reason.set('Mantenimiento programado de paquetes');
    expect(component.isValid()).toBe(true);
  });

  it('debe identificar isPausing como false para plantilla pausada y no requerir motivo', () => {
    component.template = mockPausedTemplate;
    fixture.detectChanges();
    expect(component.isPausing()).toBe(false);
    expect(component.isValid()).toBe(true);
  });

  it('debe emitir confirmed con el template y motivo', () => {
    component.template = mockApprovedTemplate;
    component.reason.set('Mantenimiento programado de paquetes');
    fixture.detectChanges();

    const confirmedSpy = vi.fn();
    component.confirmed.subscribe(confirmedSpy);

    component.confirm();
    expect(confirmedSpy).toHaveBeenCalledWith({
      template: mockApprovedTemplate,
      reason: 'Mantenimiento programado de paquetes'
    });
  });

  it('debe emitir closed al cancelar o cerrar', () => {
    component.template = mockApprovedTemplate;
    fixture.detectChanges();

    const closedSpy = vi.fn();
    component.closed.subscribe(closedSpy);

    component.closeModal();
    expect(closedSpy).toHaveBeenCalled();
  });
});
