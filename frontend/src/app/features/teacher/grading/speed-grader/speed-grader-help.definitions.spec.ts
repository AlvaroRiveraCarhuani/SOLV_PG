import { describe, it, expect } from 'vitest';
import {
  SpeedGraderTutorialContext,
  buildSpeedGraderTour,
  buildSpeedGraderTutorial
} from './speed-grader-help.definitions';

describe('SpeedGraderHelpDefinitions', () => {
  describe('buildSpeedGraderTour', () => {
    it('debe construir el tour para Entorno de Evaluación (judge) con 6 pasos', () => {
      const tour = buildSpeedGraderTour('judge', false);
      expect(tour.id).toBe('speed-grader-tour-judge');
      expect(tour.steps.length).toBe(6);
      expect(tour.steps[0].selector).toBe('[data-tour="review-header-nav"]');
      expect(tour.steps[1].selector).toBe('[data-tour="review-code-viewer"]');
      expect(tour.steps[2].selector).toBe('[data-tour="review-diff-tool"]');
      expect(tour.steps[3].selector).toBe('[data-tour="review-eval-panel"]');
      expect(tour.steps[3].title).toContain('Resultados del Juez');
      expect(tour.steps[4].selector).toBe('[data-tour="review-forensic-tools"]');
      expect(tour.steps[5].selector).toBe('[data-tour="review-grade-action"]');
      expect(tour.steps[5].title).toBe('Convalidar o Ajustar Nota');
    });

    it('debe adaptar los títulos del tour para Entorno de Desarrollo (project) con y sin rúbrica', () => {
      const tourWithRubric = buildSpeedGraderTour('project', true);
      expect(tourWithRubric.id).toBe('speed-grader-tour-project');
      expect(tourWithRubric.steps[3].title).toBe('Evaluación por Rúbrica Ponderada');
      expect(tourWithRubric.steps[5].title).toBe('Guardar Calificación');

      const tourDirect = buildSpeedGraderTour('project', false);
      expect(tourDirect.steps[3].title).toBe('Calificación Directa y Devolución');
      expect(tourDirect.steps[5].title).toBe('Guardar Calificación');
    });
  });

  describe('buildSpeedGraderTutorial', () => {
    it('debe evaluar correctamente la progresión reactiva de los 4 pasos', () => {
      let mode: 'judge' | 'project' = 'project';
      let hasCodeLoaded = false;
      let isDiffActive = false;
      let hasEvaluationInput = false;
      let isGraded = false;

      const mockContext: SpeedGraderTutorialContext = {
        mode: () => mode,
        hasCodeLoaded: () => hasCodeLoaded,
        isDiffActive: () => isDiffActive,
        hasEvaluationInput: () => hasEvaluationInput,
        isGraded: () => isGraded
      };

      const tutorial = buildSpeedGraderTutorial(mockContext);
      expect(tutorial.id).toBe('speed-grader-tutorial');
      expect(tutorial.steps.length).toBe(4);

      // Paso 1: Inspeccionar código
      const step1 = tutorial.steps[0];
      expect(step1.check?.isComplete()).toBe(false);
      hasCodeLoaded = true;
      expect(step1.check?.isComplete()).toBe(true);

      // Paso 2: Contrastar con base (completo si el diff está activo o hay código)
      const step2 = tutorial.steps[1];
      expect(step2.check?.isComplete()).toBe(true);

      // Paso 3: Evaluar desempeño
      const step3 = tutorial.steps[2];
      expect(step3.check?.isComplete()).toBe(false);
      hasEvaluationInput = true;
      expect(step3.check?.isComplete()).toBe(true);

      // Paso 4: Asentar calificación
      const step4 = tutorial.steps[3];
      expect(step4.check?.isComplete()).toBe(false);
      isGraded = true;
      expect(step4.check?.isComplete()).toBe(true);
    });
  });
});
