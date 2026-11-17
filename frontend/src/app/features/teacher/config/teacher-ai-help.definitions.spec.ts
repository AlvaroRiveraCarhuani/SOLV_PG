import { describe, it, expect } from 'vitest';
import { signal } from '@angular/core';
import {
  buildTeacherAiTour,
  buildTeacherAiTutorial
} from './teacher-ai-help.definitions';

describe('Guías de Mi Inteligencia Artificial (Docente)', () => {
  it('el recorrido cuenta con pasos coherentes y selectores válidos', () => {
    const tour = buildTeacherAiTour();

    expect(tour.id).toBe('teacher-ai-tour-v1');
    expect(tour.steps.length).toBeGreaterThanOrEqual(6);

    tour.steps.forEach((step) => {
      expect(step.selector.startsWith('[data-tour="')).toBe(true);
      expect(step.title.trim().length).toBeGreaterThan(0);
      expect(step.description.trim().length).toBeGreaterThan(15);
    });
  });

  it('el tutorial sigue la secuencia de configuración y valida cada paso con estado real', () => {
    const enabled = signal(false);
    const provider = signal('');
    const hasKey = signal(false);
    const model = signal('');
    const testPassed = signal(false);
    const hasSaved = signal(false);

    const tutorial = buildTeacherAiTutorial({
      enabled: () => enabled(),
      provider: () => provider(),
      hasKey: () => hasKey(),
      model: () => model(),
      testPassed: () => testPassed(),
      hasSaved: () => hasSaved()
    });

    const step = (id: string) => tutorial.steps.find((s) => s.id === id);

    // Paso 1: Interruptor activo
    const stepEnable = step('enable-byok');
    expect(stepEnable).toBeDefined();
    expect(stepEnable?.check?.isComplete()).toBe(false);
    enabled.set(true);
    expect(stepEnable?.check?.isComplete()).toBe(true);

    // Paso 2: Selección de proveedor
    const stepProvider = step('select-provider');
    expect(stepProvider).toBeDefined();
    expect(stepProvider?.check?.isComplete()).toBe(false);
    provider.set('google');
    expect(stepProvider?.check?.isComplete()).toBe(true);

    // Paso 3: Credenciales y modelo
    const stepCreds = step('credentials-and-model');
    expect(stepCreds).toBeDefined();
    expect(stepCreds?.check?.isComplete()).toBe(false);
    hasKey.set(true);
    expect(stepCreds?.check?.isComplete()).toBe(false); // falta modelo
    model.set('gemini-2.5-flash');
    expect(stepCreds?.check?.isComplete()).toBe(true);

    // Paso 4: Prueba de conexión
    const stepTest = step('test-connection');
    expect(stepTest).toBeDefined();
    expect(stepTest?.check?.isComplete()).toBe(false);
    testPassed.set(true);
    expect(stepTest?.check?.isComplete()).toBe(true);

    // Paso 5: Guardar configuración
    const stepSave = step('save-configuration');
    expect(stepSave).toBeDefined();
    expect(stepSave?.check?.isComplete()).toBe(false);
    hasSaved.set(true);
    expect(stepSave?.check?.isComplete()).toBe(true);
  });
});
