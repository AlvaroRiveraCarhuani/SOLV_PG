import { describe, it, expect } from 'vitest';
import { signal } from '@angular/core';
import {
  buildTeacherExercisesTour,
  buildTeacherExercisesTutorial
} from './teacher-exercises-help.definitions';

describe('Guías del Banco de Ejercicios y Duplicación (Docente)', () => {
  it('el recorrido visual tiene 6 pasos completos con selectores válidos', () => {
    const tour = buildTeacherExercisesTour();

    expect(tour.id).toBe('teacher-exercises-tour-v1');
    expect(tour.steps.length).toBe(6);

    tour.steps.forEach((step) => {
      expect(step.selector.startsWith('[data-tour="')).toBe(true);
      expect(step.title.trim().length).toBeGreaterThan(0);
      expect(step.description.trim().length).toBeGreaterThan(20);
    });

    const titles = tour.steps.map((s) => s.title);
    expect(titles).toContain('Resumen y filtros rápidos');
    expect(titles).toContain('Entorno de Evaluación vs Entorno de Desarrollo');
    expect(titles).toContain('Casos límite automáticos');
    expect(titles).toContain('Reutilizar en otra materia');
  });

  it('el tutorial interactivo valida la progresión de la duplicación inter-materia', () => {
    const hasCourses = signal(true);
    const coursesCount = signal(2);
    const totalExercisesCount = signal(0);
    const isModalOpen = signal(false);
    const hasTargetCourse = signal(false);
    const isFinished = signal(false);

    const tutorial = buildTeacherExercisesTutorial({
      hasCourses: () => hasCourses(),
      coursesCount: () => coursesCount(),
      totalExercisesCount: () => totalExercisesCount(),
      isDuplicateModalOpen: () => isModalOpen(),
      hasSelectedTargetCourse: () => hasTargetCourse(),
      isDuplicationFinished: () => isFinished()
    });

    const step = (id: string) => tutorial.steps.find((s) => s.id === id);

    // Paso 1: Seleccionar laboratorio fuente (requiere > 0 ejercicios)
    const stepSource = step('select-source-lab');
    expect(stepSource).toBeDefined();
    expect(stepSource?.check?.isComplete()).toBe(false);
    totalExercisesCount.set(3);
    expect(stepSource?.check?.isComplete()).toBe(true);

    // Paso 2: Abrir modal de duplicación
    const stepOpen = step('open-duplicate-dialog');
    expect(stepOpen).toBeDefined();
    expect(stepOpen?.check?.isComplete()).toBe(false);
    isModalOpen.set(true);
    expect(stepOpen?.check?.isComplete()).toBe(true);

    // Paso 3: Seleccionar materia destino
    const stepCourse = step('select-target-course');
    expect(stepCourse).toBeDefined();
    expect(stepCourse?.check?.isComplete()).toBe(false);
    hasTargetCourse.set(true);
    expect(stepCourse?.check?.isComplete()).toBe(true);

    // Paso 4: Confirmar duplicación (materia seleccionada y modal cerrado tras éxito)
    const stepConfirm = step('confirm-duplication');
    expect(stepConfirm).toBeDefined();
    expect(stepConfirm?.check?.isComplete()).toBe(false);
    isFinished.set(true);
    expect(stepConfirm?.check?.isComplete()).toBe(true);

    // Paso 5: Cierre pedagógico
    const stepCompleted = step('duplication-completed');
    expect(stepCompleted).toBeDefined();
    expect(stepCompleted?.check?.isComplete()).toBe(true);
  });
});
