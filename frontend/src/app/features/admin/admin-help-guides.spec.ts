import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect } from 'vitest';
import { signal } from '@angular/core';
import {
  buildEnvironmentRequestsTour,
  buildEnvironmentRequestsTutorial
} from './environment-requests/environment-requests-help.definitions';
import { buildTeachersTour, buildTeachersTutorial } from './teachers/teachers-help.definitions';
import {
  buildAuditRegistroTour,
  buildAuditRegistroTutorial
} from './auditoria/tabs/audit-registro-help.definitions';
import {
  buildEmergenciasTour,
  buildEmergenciasTutorial
} from './auditoria/tabs/emergencias-help.definitions';
import { buildCoursesTour, buildCoursesTutorial } from './courses/courses-help.definitions';
import {
  buildTechCatalogTour,
  buildTechCatalogTutorial
} from './templates/tech-catalog/tech-catalog-help.definitions';
import { buildDashboardTour } from './dashboard/dashboard-help.definitions';
import { buildCategoriesTour } from './categories/categories-help.definitions';
import { buildCatalogTour, buildCatalogTutorial } from './templates/catalog-help.definitions';

describe('Guías de Solicitudes de entornos y Docentes', () => {
  it('cada recorrido tiene identificador propio y pasos con texto real', () => {
    const tours = [
      buildEnvironmentRequestsTour(),
      buildTeachersTour(),
      buildAuditRegistroTour(),
      buildEmergenciasTour(),
      buildCoursesTour(),
      buildTechCatalogTour()
    ];

    expect(new Set(tours.map(({ id }) => id)).size).toBe(tours.length);

    tours.forEach((tour) => {
      expect(tour.steps.length).toBeGreaterThanOrEqual(5);
      tour.steps.forEach((step) => {
        expect(step.selector.startsWith('[data-tour="')).toBe(true);
        expect(step.title.length).toBeGreaterThan(0);
        expect(step.description.length).toBeGreaterThan(20);
      });
    });
  });

  it('el tutorial de solicitudes exige ficha abierta y plantilla, sin ejecutar la decisión', () => {
    const reviewOpen = signal(false);
    const hasTemplate = signal(false);

    const tutorial = buildEnvironmentRequestsTutorial({
      reviewOpen: () => reviewOpen(),
      hasTemplate: () => hasTemplate(),
      note: () => ''
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('open')?.check?.isComplete()).toBe(false);
    reviewOpen.set(true);
    expect(step('open')?.check?.isComplete()).toBe(true);

    expect(step('template')?.check?.isComplete()).toBe(false);
    hasTemplate.set(true);
    expect(step('template')?.check?.isComplete()).toBe(true);

    expect(step('note')?.check).toBeUndefined();
    expect(step('decision')?.check).toBeUndefined();
    expect(tutorial.steps[tutorial.steps.length - 1].id).toBe('decision');
    expect(tutorial.manualFragment).toBe('3-2-ciclo-de-vida-y-gobernanza-de-plantillas');
  });

  it('el tutorial de docentes valida el correo del modal y el registro confirmado', () => {
    const modalOpen = signal(false);
    const emailValid = signal(false);
    const registered = signal(false);

    const tutorial = buildTeachersTutorial({
      inviteModalOpen: () => modalOpen(),
      emailValid: () => emailValid(),
      hasRegistered: () => registered()
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('open')?.check?.isComplete()).toBe(false);
    modalOpen.set(true);
    expect(step('open')?.check?.isComplete()).toBe(true);

    expect(step('email')?.check?.isComplete()).toBe(false);
    emailValid.set(true);
    expect(step('email')?.check?.isComplete()).toBe(true);

    expect(step('submit')?.check?.isComplete()).toBe(false);
    registered.set(true);
    expect(step('submit')?.check?.isComplete()).toBe(true);

    expect(tutorial.steps[tutorial.steps.length - 1].id).toBe('lifecycle');
    expect(tutorial.steps[tutorial.steps.length - 1].check).toBeUndefined();
    expect(tutorial.manualFragment).toBe('4-4-invitaciones-a-docentes');
  });

  it('el tutorial de emergencias exige frase exacta y motivo, y no llega a ejecutar', () => {
    const open = signal(false);
    const phrase = signal('');
    const expectedPhrase = signal('TERMINAR TODOS LOS ENTORNOS');
    const reason = signal('');

    const tutorial = buildEmergenciasTutorial({
      confirmationOpen: () => open(),
      phrase: () => phrase(),
      expectedPhrase: () => expectedPhrase(),
      reason: () => reason()
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('open')?.check?.isComplete()).toBe(false);
    open.set(true);
    expect(step('open')?.check?.isComplete()).toBe(true);

    expect(step('phrase')?.check?.isComplete()).toBe(false);
    phrase.set('TERMINAR');
    expect(step('phrase')?.check?.isComplete()).toBe(false);
    phrase.set('TERMINAR TODOS LOS ENTORNOS');
    expect(step('phrase')?.check?.isComplete()).toBe(true);

    open.set(false);
    expect(step('phrase')?.check?.isComplete()).toBe(false);
    open.set(true);

    expect(step('reason')?.check?.isComplete()).toBe(false);
    reason.set('corto');
    expect(step('reason')?.check?.isComplete()).toBe(false);
    reason.set('Falta memoria y hay clases en curso');
    expect(step('reason')?.check?.isComplete()).toBe(true);

    expect(step('confirm')?.check).toBeUndefined();
    expect(tutorial.steps[tutorial.steps.length - 1].id).toBe('confirm');
    expect(tutorial.manualFragment).toBe('10-mantenimiento-y-consola-de-emergencias');
  });

  it('el tutorial de auditoría exige un filtro aplicado y una cronología abierta', () => {
    const filters = signal(false);
    const drawer = signal(false);

    const tutorial = buildAuditRegistroTutorial({
      hasActiveFilters: () => filters(),
      drawerOpen: () => drawer()
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('filter')?.check?.isComplete()).toBe(false);
    filters.set(true);
    expect(step('filter')?.check?.isComplete()).toBe(true);

    expect(step('open')?.check?.isComplete()).toBe(false);
    drawer.set(true);
    expect(step('open')?.check?.isComplete()).toBe(true);

    expect(step('read')?.check).toBeUndefined();
    expect(tutorial.steps[tutorial.steps.length - 1].id).toBe('read');
    expect(tutorial.manualFragment).toBe('5-2-registro-de-auditoria');
  });

  it('los recorridos sin tutorial tienen pasos propios y selectores reales', () => {
    const tours = [buildDashboardTour(), buildCategoriesTour(), buildCatalogTour()];

    expect(new Set(tours.map(({ id }) => id)).size).toBe(tours.length);

    tours.forEach((tour) => {
      expect(tour.steps.length).toBeGreaterThanOrEqual(3);
      tour.steps.forEach((step) => {
        expect(step.selector.startsWith('[data-tour="')).toBe(true);
        expect(step.title.length).toBeGreaterThan(0);
        expect(step.description.length).toBeGreaterThan(20);
      });
    });
  });

  it('el tutorial de cursos exige modal y datos, y no obliga a crear la materia', () => {
    const modalOpen = signal(false);
    const identity = signal(false);

    const tutorial = buildCoursesTutorial({
      createModalOpen: () => modalOpen(),
      identityComplete: () => identity()
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('open')?.check?.isComplete()).toBe(false);
    modalOpen.set(true);
    expect(step('open')?.check?.isComplete()).toBe(true);

    expect(step('identity')?.check?.isComplete()).toBe(false);
    identity.set(true);
    expect(step('identity')?.check?.isComplete()).toBe(true);

    modalOpen.set(false);
    expect(step('identity')?.check?.isComplete()).toBe(false);
    modalOpen.set(true);

    expect(step('period')?.check).toBeUndefined();
    expect(step('teacher')?.check).toBeUndefined();

    expect(step('submit')?.check).toBeUndefined();
    expect(step('lifecycle')?.check).toBeUndefined();

    expect(tutorial.steps[tutorial.steps.length - 1].id).toBe('lifecycle');
    expect(tutorial.steps[tutorial.steps.length - 1].check).toBeUndefined();
    expect(tutorial.manualFragment).toBe('4-2-cursos-y-asignacion-de-docentes');
  });

  it('el tutorial de tecnologías no depende del botón de eliminar del catálogo oficial', () => {
    const query = signal('');

    const tutorial = buildTechCatalogTutorial({
      hasSearchQuery: () => query().trim().length > 0
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('search')?.check?.isComplete()).toBe(false);
    query.set('python');
    expect(step('search')?.check?.isComplete()).toBe(true);

    expect(step('usage')?.check).toBeUndefined();
    expect(step('origin')?.check).toBeUndefined();
    expect(step('exit')?.check).toBeUndefined();

    expect(step('usage')?.selector).toBe('[data-tour="tech-usage"]');
    expect(step('origin')?.selector).toBe('[data-tour="tech-card-actions"]');
    expect(step('exit')?.selector).toBe('[data-tour="tech-toggle"]');

    const selectors = tutorial.steps.map((entry) => entry.selector ?? '').join(' ');
    expect(selectors).not.toContain('tech-delete');
    expect(tutorial.steps[tutorial.steps.length - 1].id).toBe('exit');
    expect(tutorial.manualFragment).toBe('8-2-catalogo-de-tecnologias');
  });

  it('el tutorial de plantillas cambia de guion cuando la bandeja está vacía', () => {
    let pending = 0;
    let tab: 'pending' | 'catalog' | 'rejected' = 'catalog';
    const build = () => buildCatalogTutorial({ activeTab: () => tab, pendingCount: () => pending });

    const empty = build();
    expect(empty.id).toBe('catalog-work-without-pending');
    expect(empty.steps.map((entry) => entry.id)).toEqual(['purpose', 'card', 'catalog']);
    expect(empty.steps.every((entry) => entry.check === undefined)).toBe(true);
    expect(empty.steps.find((entry) => entry.id === 'card')?.selector).toBe('[data-tour="templates-list"]');
    expect(empty.steps.some((entry) => entry.id === 'review')).toBe(false);

    pending = 1;
    const review = build();
    expect(review.id).toBe('catalog-review-first-template');
    expect(review.steps.map((entry) => entry.id)).toContain('review');
    expect(review.steps.find((entry) => entry.id === 'pending')?.check?.isComplete()).toBe(false);

    tab = 'pending';
    expect(review.steps.find((entry) => entry.id === 'pending')?.check?.isComplete()).toBe(true);
  });
});
