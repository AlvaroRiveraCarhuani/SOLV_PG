import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { TeacherInviteModalComponent } from './teacher-invite-modal.component';
import { TenantService } from '@core/services/tenant.service';

describe('TeacherInviteModalComponent', () => {
  let component: TeacherInviteModalComponent;
  let mockTenantService: any;

  beforeEach(async () => {
    const configSignal = signal({ support_email: 'soporte@uab.edu.bo', base_domain: 'uab.edu.bo' });
    mockTenantService = { config: configSignal };

    await TestBed.configureTestingModule({
      imports: [TeacherInviteModalComponent],
      providers: [
        { provide: TenantService, useValue: mockTenantService }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(TeacherInviteModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('institutionDomain se extrae del support_email del tenant', () => {
    expect(component.institutionDomain()).toBe('uab.edu.bo');
  });

  it('isValidEmail es false con email vacío', () => {
    component.email.set('');
    expect(component.isValidEmail()).toBe(false);
  });

  it('isValidEmail es false con email demasiado corto (≤ 5 chars)', () => {
    component.email.set('a@b.c');
    expect(component.isValidEmail()).toBe(false);
  });

  it('isValidEmail es true con email correctamente formado', () => {
    component.email.set('docente@uab.edu.bo');
    expect(component.isValidEmail()).toBe(true);
  });

  it('onSubmit con email inválido no emite submit ni activa isSubmitting', () => {
    let emitted = false;
    component.submit.subscribe(() => { emitted = true; });
    component.email.set('invalido');
    component.onSubmit();
    expect(emitted).toBe(false);
    expect(component.isSubmitting()).toBe(false);
  });

  it('onSubmit con email válido emite payload correcto y activa isSubmitting', () => {
    let payload: any;
    component.submit.subscribe(p => { payload = p; });

    component.email.set('nuevo.docente@uab.edu.bo');
    component.sendEmail.set(false);
    component.onSubmit();

    expect(payload).toEqual({
      email: 'nuevo.docente@uab.edu.bo',
      role: 'teacher',
      role_type: 'titular',
      send_email: false
    });
    expect(component.isSubmitting()).toBe(true);
  });

  it('onSubmit con isSubmitting activo no vuelve a emitir (doble submit)', () => {
    let count = 0;
    component.submit.subscribe(() => { count++; });
    component.email.set('docente@uab.edu.bo');
    component.onSubmit();
    component.onSubmit(); // segundo intento
    expect(count).toBe(1);
  });

  it('isExternalDomain es false para email del dominio institucional', () => {
    component.email.set('prof@uab.edu.bo');
    expect(component.isExternalDomain()).toBe(false);
  });

  it('isExternalDomain es true para gmail u otros dominios externos', () => {
    component.email.set('prof@gmail.com');
    expect(component.isExternalDomain()).toBe(true);
  });

  it('close emite evento al ser llamado desde el template', () => {
    let closed = false;
    component.close.subscribe(() => { closed = true; });
    component.close.emit();
    expect(closed).toBe(true);
  });
});
