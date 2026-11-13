import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { InteractiveTutorialService } from './interactive-tutorial.service';
import { ProductTourService } from './product-tour.service';
import { TutorialDefinition } from './help.types';

describe('InteractiveTutorialService', () => {
  let service: InteractiveTutorialService;
  let tour: { highlight: ReturnType<typeof vi.fn>; destroy: ReturnType<typeof vi.fn> };

  const nameReady = signal(false);

  const definition: TutorialDefinition = {
    id: 'spec-tutorial',
    title: 'Tutorial de prueba',
    manualFragment: '7-constructor-de-plantillas-de-entornos',
    steps: [
      { id: 'intro', title: 'Intro', body: 'Sin comprobación.' },
      {
        id: 'identity',
        title: 'Identidad',
        body: 'Escriba el nombre.',
        selector: '#studio-tpl-name',
        check: {
          isComplete: () => nameReady(),
          pendingHint: 'Falta el nombre.'
        }
      },
      { id: 'summary', title: 'Resumen', body: 'Paso final.' }
    ]
  };

  beforeEach(() => {
    nameReady.set(false);
    tour = { highlight: vi.fn().mockResolvedValue(undefined), destroy: vi.fn() };
    TestBed.configureTestingModule({
      providers: [{ provide: ProductTourService, useValue: tour }]
    });
    service = TestBed.inject(InteractiveTutorialService);
  });

  it('empieza en el primer paso y expone el progreso', () => {
    service.start(definition);

    expect(service.isActive()).toBe(true);
    expect(service.index()).toBe(0);
    expect(service.stepCount()).toBe(3);
    expect(service.currentStep()?.id).toBe('intro');
    expect(service.isFirstStep()).toBe(true);
  });

  it('no avanza un paso cuya comprobación real no se cumple', () => {
    service.start(definition);
    service.next();

    expect(service.index()).toBe(1);
    expect(service.currentCheckPassed()).toBe(false);

    service.next();
    expect(service.index()).toBe(1);

    nameReady.set(true);
    expect(service.currentCheckPassed()).toBe(true);

    service.next();
    expect(service.index()).toBe(2);
    expect(service.isLastStep()).toBe(true);
  });

  it('resalta el elemento del paso actual y limpia al salir', () => {
    service.start(definition);
    service.next();

    expect(tour.highlight).toHaveBeenCalledTimes(1);
    expect(tour.highlight.mock.calls[0][0].selector).toBe('#studio-tpl-name');

    service.exit();
    expect(service.isActive()).toBe(false);
    expect(tour.destroy).toHaveBeenCalled();
  });

  it('finalizar cierra el tutorial y lo registra como visto', () => {
    service.start({ ...definition, steps: [definition.steps[0]] });

    service.next();

    expect(service.isActive()).toBe(false);
    expect(service.hasSeen('spec-tutorial')).toBe(true);
  });

  it('retroceder no baja del primer paso', () => {
    service.start(definition);

    service.previous();

    expect(service.index()).toBe(0);
  });
});
