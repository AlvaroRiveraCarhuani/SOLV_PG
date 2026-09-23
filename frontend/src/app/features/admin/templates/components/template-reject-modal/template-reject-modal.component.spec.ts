import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { TemplateRejectModalComponent } from './template-reject-modal.component';
import { AdminTemplateItem } from '../../../services/admin-templates.service';

describe('TemplateRejectModalComponent Spec', () => {
  let component: TemplateRejectModalComponent;
  let fixture: any;

  const mockTemplate: AdminTemplateItem = {
    id: 'tpl-reject-1',
    name: 'Custom Unverified Image',
    docker_image: 'user/custom-env:latest',
    base_ram_mb: 512,
    status: 'PENDIENTE_AUDITORIA',
    created_at: '2026-09-23T00:00:00Z',
    requested_by_name: 'Prof. Carlos Perez'
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TemplateRejectModalComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(TemplateRejectModalComponent);
    component = fixture.componentInstance;
    component.template = { ...mockTemplate };
    fixture.detectChanges();
  });

  it('motivo menor a 10 caracteres bloquea el envío y desactiva isValid', () => {
    component.rejectionReason.set('Corta');
    expect(component.charCount()).toBe(5);
    expect(component.isValid()).toBe(false);

    let emitted = false;
    component.rejected.subscribe(() => {
      emitted = true;
    });

    component.confirmReject();
    expect(emitted).toBe(false);
  });

  it('motivo con 9 caracteres sigue bloqueando el envío', () => {
    component.rejectionReason.set('123456789');
    expect(component.charCount()).toBe(9);
    expect(component.isValid()).toBe(false);

    let emitted = false;
    component.rejected.subscribe(() => {
      emitted = true;
    });

    component.confirmReject();
    expect(emitted).toBe(false);
  });

  it('motivo mayor o igual a 10 caracteres habilita el envío', () => {
    component.rejectionReason.set('1234567890');
    expect(component.charCount()).toBe(10);
    expect(component.isValid()).toBe(true);
  });

  it('al confirmar con motivo válido el motivo viaja en el payload', () => {
    const reasonText = 'Imagen Docker no oficial sin dependencias validadas';
    component.rejectionReason.set(reasonText);

    let emittedPayload: { id: string; reason: string } | undefined;
    component.rejected.subscribe((payload) => {
      emittedPayload = payload;
    });

    component.confirmReject();

    expect(emittedPayload).toBeDefined();
    expect(emittedPayload?.id).toBe('tpl-reject-1');
    expect(emittedPayload?.reason).toBe(reasonText);
  });

  it('cerrar modal emite evento closed', () => {
    let closedEmitted = false;
    component.closed.subscribe(() => {
      closedEmitted = true;
    });

    component.closeModal();
    expect(closedEmitted).toBe(true);
  });
});
