import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect } from 'vitest';
import { signal } from '@angular/core';
import { buildStudioTour, buildStudioTutorial } from './studio-help.definitions';

describe('Guías del Constructor de Plantillas', () => {
  it('el recorrido señala los elementos reales del constructor', () => {
    const tour = buildStudioTour();

    expect(tour.steps).toHaveLength(6);
    expect(tour.steps.map((step) => step.selector)).toEqual([
      '[data-tour="studio-purpose"]',
      '#studio-tpl-name',
      '[data-tour="studio-stack"]',
      '[data-tour="studio-capacity"]',
      '[data-tour="studio-summary"]',
      '[data-tour="studio-publish"]'
    ]);
    tour.steps.forEach((step) => {
      expect(step.title.length).toBeGreaterThan(0);
      expect(step.description.length).toBeGreaterThan(20);
    });
  });

  it('el tutorial exige nombre, tecnología y memoria válida antes de avanzar', () => {
    const name = signal('');
    const hasStack = signal(false);
    const capacityValid = signal(true);

    const tutorial = buildStudioTutorial({
      templateName: () => name(),
      hasSelectedStack: () => hasStack(),
      isCapacityValid: () => capacityValid(),
      publishBlocker: () => null
    });

    const identity = tutorial.steps.find((step) => step.id === 'identity');
    const stack = tutorial.steps.find((step) => step.id === 'stack');
    const capacity = tutorial.steps.find((step) => step.id === 'capacity');

    expect(identity?.check?.isComplete()).toBe(false);
    name.set('Entorno de prueba');
    expect(identity?.check?.isComplete()).toBe(true);

    expect(stack?.check?.isComplete()).toBe(false);
    hasStack.set(true);
    expect(stack?.check?.isComplete()).toBe(true);

    capacityValid.set(false);
    expect(capacity?.check?.isComplete()).toBe(false);
    capacityValid.set(true);
    expect(capacity?.check?.isComplete()).toBe(true);
  });

  it('el último paso explica la publicación sin ejecutarla ni exigirla', () => {
    const tutorial = buildStudioTutorial({
      templateName: () => 'Entorno de prueba',
      hasSelectedStack: () => true,
      isCapacityValid: () => true,
      publishBlocker: () => ({ reason: 'Falta comprobar el funcionamiento.' })
    });

    const publish = tutorial.steps[tutorial.steps.length - 1];

    expect(publish.id).toBe('publish');
    expect(publish.check).toBeUndefined();
    expect(tutorial.manualFragment).toBe('7-constructor-de-plantillas-de-entornos');
  });
});
