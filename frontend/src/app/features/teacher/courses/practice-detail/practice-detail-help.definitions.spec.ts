import { describe, expect, it } from 'vitest';
import {
  buildPracticeDetailTour,
  buildPracticeDetailTutorial,
  PracticeDetailTutorialContext
} from './practice-detail-help.definitions';

describe('practice-detail-help.definitions', () => {
  describe('buildPracticeDetailTour', () => {
    it('construye tour para modalidad IDE con base de datos satélite', () => {
      const tour = buildPracticeDetailTour(true, true);
      expect(tour.steps.length).toBe(6);
      expect(tour.steps.map(s => s.selector)).toContain('[data-tour="practice-databases"]');
      expect(tour.steps.map(s => s.selector)).toContain('[data-tour="practice-consigna"]');
    });

    it('construye tour para modalidad IDE sin base de datos', () => {
      const tour = buildPracticeDetailTour(true, false);
      expect(tour.steps.length).toBe(5);
      expect(tour.steps.map(s => s.selector)).not.toContain('[data-tour="practice-databases"]');
    });

    it('construye tour para modalidad evaluación (juez)', () => {
      const tour = buildPracticeDetailTour(false, false);
      expect(tour.steps.length).toBe(5);
      const evalStep = tour.steps.find(s => s.selector === '[data-tour="practice-eval-criteria"]');
      expect(evalStep?.title).toBe('Casos de Prueba');
    });
  });

  describe('buildPracticeDetailTutorial', () => {
    it('evalúa la progresión de pasos según el contexto reactivo', () => {
      const context: PracticeDetailTutorialContext = {
        isIde: () => true,
        isConsignaExpanded: () => false,
        isEvalExpanded: () => true,
        hasDatabase: () => false,
        isDatabasesExpanded: () => false
      };

      const tutorial = buildPracticeDetailTutorial(context);
      expect(tutorial.steps.length).toBe(3);
      expect(tutorial.steps[0].check?.isComplete()).toBe(true);
      expect(tutorial.steps[1].check?.isComplete()).toBe(true);
      expect(tutorial.steps[2].check).toBeUndefined();
    });
  });
});
