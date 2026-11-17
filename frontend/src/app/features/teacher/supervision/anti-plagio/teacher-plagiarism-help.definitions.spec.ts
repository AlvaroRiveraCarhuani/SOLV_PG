import { describe, it, expect } from 'vitest';
import {
  TeacherPlagiarismTutorialContext,
  buildTeacherPlagiarismTour,
  buildTeacherPlagiarismTutorial
} from './teacher-plagiarism-help.definitions';

describe('TeacherPlagiarismHelpDefinitions', () => {
  describe('buildTeacherPlagiarismTour', () => {
    it('debe construir el tour visual de auditoría con 6 pasos válidos', () => {
      const tour = buildTeacherPlagiarismTour();
      expect(tour.id).toBe('teacher-plagiarism-tour-v1');
      expect(tour.steps.length).toBe(6);
      expect(tour.steps[0].selector).toBe('[data-tour="plagiarism-summary"]');
      expect(tour.steps[1].selector).toBe('[data-tour="plagiarism-actions"]');
      expect(tour.steps[2].selector).toBe('[data-tour="plagiarism-pairs-list"]');
      expect(tour.steps[3].selector).toBe('[data-tour="plagiarism-files-list"]');
      expect(tour.steps[4].selector).toBe('[data-tour="plagiarism-diff-viewer"]');
      expect(tour.steps[5].selector).toBe('[data-tour="plagiarism-forensic-metrics"]');
    });
  });

  describe('buildTeacherPlagiarismTutorial', () => {
    it('debe comprobar reactivamente los 4 pasos del tutorial', () => {
      let hasReport = false;
      let matchesCount = 0;
      let isMatchSelected = false;
      let hasPairLoaded = false;
      let currentRiskFilter = 'all';
      let isMultiFile = false;

      const mockContext: TeacherPlagiarismTutorialContext = {
        hasReport: () => hasReport,
        matchesCount: () => matchesCount,
        isMatchSelected: () => isMatchSelected,
        hasPairLoaded: () => hasPairLoaded,
        currentRiskFilter: () => currentRiskFilter,
        isMultiFile: () => isMultiFile
      };

      const tutorial = buildTeacherPlagiarismTutorial(mockContext);
      expect(tutorial.id).toBe('teacher-plagiarism-tutorial');
      expect(tutorial.steps.length).toBe(4);

      // Paso 1: Filtrar por criticidad
      const step1 = tutorial.steps[0];
      expect(step1.check?.isComplete()).toBe(false);
      currentRiskFilter = 'critical';
      expect(step1.check?.isComplete()).toBe(true);

      // Paso 2: Seleccionar un par de alumnos
      const step2 = tutorial.steps[1];
      expect(step2.check?.isComplete()).toBe(false);
      isMatchSelected = true;
      expect(step2.check?.isComplete()).toBe(true);

      // Paso 3: Examinar tramos en el comparador
      const step3 = tutorial.steps[2];
      expect(step3.check?.isComplete()).toBe(false);
      hasPairLoaded = true;
      expect(step3.check?.isComplete()).toBe(true);

      // Paso 4: Contrastar con la telemetría forense
      const step4 = tutorial.steps[3];
      expect(step4.check?.isComplete()).toBe(true);
      hasPairLoaded = false;
      expect(step4.check?.isComplete()).toBe(false);
    });
  });
});
