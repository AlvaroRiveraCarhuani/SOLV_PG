import { describe, it, expect } from 'vitest';
import { signal } from '@angular/core';
import {
  buildTeacherTemplatesTour,
  buildTeacherTemplateRequestTutorial
} from './teacher-templates-help.definitions';

describe('Guías del Catálogo de Entornos y Solicitud (Docente)', () => {
  it('el recorrido visual tiene pasos completos con selectores válidos', () => {
    const tour = buildTeacherTemplatesTour();

    expect(tour.id).toBe('teacher-templates-tour-v1');
    expect(tour.steps.length).toBeGreaterThanOrEqual(5);

    tour.steps.forEach((step) => {
      expect(step.selector.startsWith('[data-tour="')).toBe(true);
      expect(step.title.trim().length).toBeGreaterThan(0);
      expect(step.description.trim().length).toBeGreaterThan(20);
    });
  });

  it('el tutorial acompaña la apertura y preparación de la solicitud validando estado real', () => {
    const isModalOpen = signal(false);
    const description = signal('');
    const chipsCount = signal(0);
    const usesCount = signal(0);

    const tutorial = buildTeacherTemplateRequestTutorial({
      isRequestModalOpen: () => isModalOpen(),
      descriptionValid: () => description().trim().length >= 10,
      hasChips: () => chipsCount() > 0,
      hasUses: () => usesCount() > 0
    });

    const step = (id: string) => tutorial.steps.find((s) => s.id === id);

    // Paso 1: Abrir el modal desde el catálogo
    const stepOpen = step('open-request');
    expect(stepOpen).toBeDefined();
    expect(stepOpen?.check?.isComplete()).toBe(false);
    isModalOpen.set(true);
    expect(stepOpen?.check?.isComplete()).toBe(true);

    // Paso 2: Agregar herramientas o usar IA (herramientas primero)
    const stepTools = step('tools-and-ai');
    expect(stepTools).toBeDefined();
    expect(stepTools?.selector).toBe('[data-tour="template-request-tools"]');
    expect(stepTools?.check?.isComplete()).toBe(false);
    chipsCount.set(2);
    expect(stepTools?.check?.isComplete()).toBe(true);

    // Paso 3: Modalidad de uso
    const stepUses = step('select-uses');
    expect(stepUses).toBeDefined();
    expect(stepUses?.check?.isComplete()).toBe(false);
    usesCount.set(1);
    expect(stepUses?.check?.isComplete()).toBe(true);

    // Paso 4: Nombre y nota
    const stepPractice = step('describe-practice');
    expect(stepPractice).toBeDefined();
    expect(stepPractice?.selector).toBe('[data-tour="template-request-practice"]');
    expect(stepPractice?.check?.isComplete()).toBe(false);
    description.set('Corta');
    expect(stepPractice?.check?.isComplete()).toBe(false);
    description.set('Construcción de una API con FastAPI y Postgres para testing.');
    expect(stepPractice?.check?.isComplete()).toBe(true);

    // Paso 5: Revisión y envío (sin dependencia de panel lateral)
    const stepSummary = step('review-summary');
    expect(stepSummary).toBeDefined();
    expect(stepSummary?.selector).toBe('[data-tour="template-request-actions"]');
    expect(stepSummary?.check).toBeUndefined();
  });
});
