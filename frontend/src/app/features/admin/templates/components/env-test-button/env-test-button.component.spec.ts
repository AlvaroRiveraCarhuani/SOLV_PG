import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { 
  EnvTestButtonComponent,
  ENV_TEST_ERROR_MESSAGES, 
  getEnvTestErrorMessage 
} from './env-test-button.component';
import { EnvTestJobService } from '../../../services/env-test-job.service';

describe('EnvTestButton Error Mapping Contract', () => {
  const expectedContractErrorCodes = [
    'pull_stalled',
    'pull_timeout',
    'registry_unreachable',
    'test_oom',
    'test_crash',
    'internal'
  ] as const;

  it('debe mapear todos los 6 codigos de error del contrato del backend', () => {
    for (const code of expectedContractErrorCodes) {
      const message = ENV_TEST_ERROR_MESSAGES[code];
      expect(message, `El codigo de error "${code}" debe tener un mensaje asignado en el frontend`).toBeDefined();
      expect(message.length, `El mensaje para "${code}" no debe ser vacio`).toBeGreaterThan(0);
    }
  });

  it('debe mapear pull_stalled a la descripcion de inactividad de red (TE-75)', () => {
    const msg = getEnvTestErrorMessage('pull_stalled');
    expect(msg).toContain('Descarga detenida');
    expect(msg).toContain('60 s');
  });

  it('debe mapear pull_timeout al tiempo maximo excedido (TE-77)', () => {
    const msg = getEnvTestErrorMessage('pull_timeout');
    expect(msg).toContain('Tiempo total de prueba excedido');
    expect(msg).toContain('15 min');
  });

  it('debe mapear registry_unreachable al error de conexion OCI (TE-76)', () => {
    const msg = getEnvTestErrorMessage('registry_unreachable');
    expect(msg).toContain('No se pudo conectar al registro OCI');
  });

  it('debe mapear test_oom a limite de memoria superado (TE-71)', () => {
    const msg = getEnvTestErrorMessage('test_oom');
    expect(msg).toContain('Límite de memoria superado');
    expect(msg).toContain('OOM');
  });

  it('debe mapear test_crash a fallo al ejecutar el contenedor (TE-72)', () => {
    const msg = getEnvTestErrorMessage('test_crash');
    expect(msg).toContain('Fallo al ejecutar el contenedor de prueba');
  });

  it('debe mapear internal al error generico del sistema (TE-70)', () => {
    const msg = getEnvTestErrorMessage('internal');
    expect(msg).toContain('Error interno en la prueba de entorno');
  });

  it('debe degradar a error interno si recibe un codigo no reconocido', () => {
    const fallback = getEnvTestErrorMessage('unrecognized_code_123');
    expect(fallback).toBe(ENV_TEST_ERROR_MESSAGES['internal']);
  });
});

describe('EnvTestButtonComponent - Botón deshabilitado con razón visible', () => {
  let component: EnvTestButtonComponent;
  let fixture: ComponentFixture<EnvTestButtonComponent>;

  const mockEnvTestJobService = {
    startJob: () => of({}),
    pollJob: () => of({}),
    cancelJob: () => of({})
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EnvTestButtonComponent],
      providers: [
        { provide: EnvTestJobService, useValue: mockEnvTestJobService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(EnvTestButtonComponent);
    component = fixture.componentInstance;
  });

  it('debe deshabilitar el botón y mostrar razón visible cuando falta la imagen', () => {
    fixture.componentRef.setInput('image', '');
    fixture.detectChanges();

    expect(component.canTrigger()).toBe(false);
    expect(component.disabledReason()).toContain('imagen');

    const btn = fixture.nativeElement.querySelector('.btn-env-test');
    expect(btn.disabled).toBe(true);

    const helper = fixture.nativeElement.querySelector('.env-helper');
    expect(helper).toBeTruthy();
    expect(helper.textContent).toContain('imagen');
  });

  it('debe deshabilitar el botón cuando la imagen tiene tag :latest y advertirlo en la razón', () => {
    fixture.componentRef.setInput('image', 'python:latest');
    fixture.detectChanges();

    expect(component.canTrigger()).toBe(false);
    expect(component.disabledReason()).toContain('no :latest');

    const helper = fixture.nativeElement.querySelector('.env-helper');
    expect(helper.textContent).toContain('no :latest');
  });

  it('debe deshabilitar el botón y reportar comando de ejecución faltante para juez efímero', () => {
    fixture.componentRef.setInput('image', 'python:3.11-slim');
    fixture.componentRef.setInput('targetEnvironment', 'JUEZ_EFIMERO');
    fixture.componentRef.setInput('entrypoint', '');
    fixture.detectChanges();

    expect(component.canTrigger()).toBe(false);
    expect(component.disabledReason()).toContain('comando de ejecución');

    const helper = fixture.nativeElement.querySelector('.env-helper');
    expect(helper.textContent).toContain('comando de ejecución');
  });

  it('debe habilitar el botón cuando los requerimientos están completos', () => {
    fixture.componentRef.setInput('image', 'python:3.11-slim');
    fixture.componentRef.setInput('targetEnvironment', 'IDE_PERSISTENTE');
    fixture.detectChanges();

    expect(component.canTrigger()).toBe(true);
    const btn = fixture.nativeElement.querySelector('.btn-env-test');
    expect(btn.disabled).toBe(false);
  });
});
