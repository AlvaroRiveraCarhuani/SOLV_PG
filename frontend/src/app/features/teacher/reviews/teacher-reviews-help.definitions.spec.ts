import { describe, it, expect } from 'vitest';
import {
  TeacherReviewsTutorialContext,
  buildTeacherReviewsTour,
  buildTeacherReviewsTutorial
} from './teacher-reviews-help.definitions';

describe('TeacherReviewsHelpDefinitions', () => {
  describe('buildTeacherReviewsTour', () => {
    it('debe construir el tour visual con 5 pasos y selectores reales', () => {
      const tour = buildTeacherReviewsTour();
      expect(tour.id).toBe('teacher-reviews-tour-v1');
      expect(tour.steps.length).toBe(5);
      expect(tour.steps[0].selector).toBe('[data-tour="reviews-header-summary"]');
      expect(tour.steps[1].selector).toBe('[data-tour="reviews-queue-pane"]');
      expect(tour.steps[2].selector).toBe('[data-tour="reviews-work-header"]');
      expect(tour.steps[3].selector).toBe('[data-tour="reviews-table"]');
      expect(tour.steps[4].selector).toBe('[data-tour="reviews-grade-action"]');
    });
  });

  describe('buildTeacherReviewsTutorial', () => {
    it('debe evaluar correctamente la progresión reactiva de los 4 pasos', () => {
      let hasGroups = false;
      let hasSelectedGroup = false;
      let submissionsCount = 0;
      let isFilterActive = false;

      const mockContext: TeacherReviewsTutorialContext = {
        hasGroups: () => hasGroups,
        hasSelectedGroup: () => hasSelectedGroup,
        submissionsCount: () => submissionsCount,
        isFilterActive: () => isFilterActive
      };

      const tutorial = buildTeacherReviewsTutorial(mockContext);
      expect(tutorial.id).toBe('teacher-reviews-tutorial');
      expect(tutorial.steps.length).toBe(4);

      // Paso 1: Seleccionar práctica
      const step1 = tutorial.steps[0];
      expect(step1.check?.isComplete()).toBe(false);
      hasSelectedGroup = true;
      expect(step1.check?.isComplete()).toBe(true);

      // Paso 2: Priorizar por tiempo de espera
      const step2 = tutorial.steps[1];
      expect(step2.check?.isComplete()).toBe(false);
      submissionsCount = 3;
      expect(step2.check?.isComplete()).toBe(true);

      // Paso 3: Identificar estado de revisión
      const step3 = tutorial.steps[2];
      expect(step3.check?.isComplete()).toBe(true);

      // Paso 4: Lanzar la calificación
      const step4 = tutorial.steps[3];
      expect(step4.check?.isComplete()).toBe(true);
      submissionsCount = 0;
      expect(step4.check?.isComplete()).toBe(false);
    });
  });
});
