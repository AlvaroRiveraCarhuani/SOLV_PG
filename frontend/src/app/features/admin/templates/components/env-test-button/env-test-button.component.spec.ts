import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect } from 'vitest';
import { 
  ENV_TEST_ERROR_MESSAGES, 
  getEnvTestErrorMessage 
} from './env-test-button.component';

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
