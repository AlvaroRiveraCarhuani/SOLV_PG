import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { StudentCreateModalComponent } from './student-create-modal.component';
import { TenantService } from '@core/services/tenant.service';

describe('StudentCreateModalComponent', () => {
  let component: StudentCreateModalComponent;
  let mockTenantService: any;

  beforeEach(async () => {
    const configSignal = signal({ support_email: 'soporte@uab.edu.bo', base_domain: 'uab.edu.bo' });
    mockTenantService = { config: configSignal };

    await TestBed.configureTestingModule({
      imports: [StudentCreateModalComponent],
      providers: [
        { provide: TenantService, useValue: mockTenantService }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(StudentCreateModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('institutionDomain se extrae del support_email configurado', () => {
    expect(component.institutionDomain()).toBe('uab.edu.bo');
  });

  it('onSubmit con campos vacíos establece error y no emite', () => {
    let emitted = false;
    component.studentCreated.subscribe(() => { emitted = true; });
    component.onSubmit();
    expect(component.error()).toBe('Todos los campos son obligatorios.');
    expect(emitted).toBe(false);
  });

  it('onSubmit con email sin arroba establece error de formato', () => {
    component.firstName.set('Juan');
    component.lastName.set('Pérez');
    component.email.set('juanperez-invalido');
    component.onSubmit();
    expect(component.error()).toBe('Ingresá un correo electrónico válido.');
  });

  it('onSubmit con datos válidos emite el DTO correcto', () => {
    let payload: any;
    component.studentCreated.subscribe(dto => { payload = dto; });

    component.firstName.set('  Juan  ');
    component.lastName.set('Pérez');
    component.email.set('JUAN.PEREZ@UAB.EDU.BO');
    component.onSubmit();

    expect(payload).toEqual({
      first_name: 'Juan',
      last_name: 'Pérez',
      email: 'juan.perez@uab.edu.bo'
    });
  });

  it('isExternalDomain es false para email del dominio institucional', () => {
    component.email.set('alumno@uab.edu.bo');
    expect(component.isExternalDomain()).toBe(false);
  });

  it('isExternalDomain es true para dominio diferente al institucional', () => {
    component.email.set('alumno@gmail.com');
    expect(component.isExternalDomain()).toBe(true);
  });

  it('isExternalDomain es false cuando no hay arroba', () => {
    component.email.set('emailsinarroba');
    expect(component.isExternalDomain()).toBe(false);
  });

  it('onSubmit limpia el error previo al reenviar con datos válidos', () => {
    component.onSubmit();
    expect(component.error()).not.toBeNull();

    component.firstName.set('Ana');
    component.lastName.set('García');
    component.email.set('ana@uab.edu.bo');
    component.onSubmit();
    expect(component.error()).toBeNull();
  });
});
