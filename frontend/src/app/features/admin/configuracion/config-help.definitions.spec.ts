import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect } from 'vitest';
import { signal } from '@angular/core';
import {
  buildAdministradoresTour,
  buildAdministradoresTutorial
} from './tabs/administradores-help.definitions';
import { buildIaTour, buildIaTutorial } from './tabs/ia-help.definitions';
import { buildIdentidadTour, buildIdentidadTutorial } from './tabs/identidad-help.definitions';
import { buildPeriodosTour, buildPeriodosTutorial } from './tabs/periodos-help.definitions';
import { buildServidorTour, buildServidorTutorial } from './tabs/servidor-help.definitions';

describe('Guías de la Configuración Institucional', () => {
  it('cada pestaña tiene su recorrido propio y empieza señalándose a sí misma', () => {
    const tours = [
      { tour: buildPeriodosTour(), tab: 'periodos', id: 'config-periods-tour-v1' },
      { tour: buildIdentidadTour(), tab: 'identidad', id: 'config-branding-tour-v1' },
      { tour: buildServidorTour(), tab: 'servidor', id: 'config-server-tour-v1' },
      { tour: buildIaTour(), tab: 'ia', id: 'config-ai-tour-v1' },
      { tour: buildAdministradoresTour(), tab: 'administradores', id: 'config-admins-tour-v1' }
    ];

    expect(new Set(tours.map(({ id }) => id)).size).toBe(tours.length);

    tours.forEach(({ tour, tab, id }) => {
      expect(tour.id).toBe(id);
      expect(tour.steps[0].selector).toBe(`[data-tour="config-tab-${tab}"]`);
      expect(tour.steps.length).toBeGreaterThanOrEqual(4);
      tour.steps.forEach((step) => {
        expect(step.title.length).toBeGreaterThan(0);
        expect(step.description.length).toBeGreaterThan(20);
        if (step.selector.includes('config-tab-')) {
          expect(step.selector).toBe(`[data-tour="config-tab-${tab}"]`);
        }
      });
    });
  });

  it('el tutorial de períodos exige formulario abierto, datos y guardado real', () => {
    const formOpen = signal(false);
    const name = signal('');
    const code = signal('');
    const valid = signal(false);
    const created = signal(false);

    const tutorial = buildPeriodosTutorial({
      formOpen: () => formOpen(),
      name: () => name(),
      code: () => code(),
      isFormValid: () => valid(),
      hasCreated: () => created(),
      activeCount: () => 0
    });

    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('create')?.check?.isComplete()).toBe(false);
    formOpen.set(true);
    expect(step('create')?.check?.isComplete()).toBe(true);

    expect(step('identity')?.check?.isComplete()).toBe(false);
    name.set('Semestre II');
    expect(step('identity')?.check?.isComplete()).toBe(false);
    code.set('2026-2');
    expect(step('identity')?.check?.isComplete()).toBe(true);

    expect(step('dates')?.check?.isComplete()).toBe(false);
    valid.set(true);
    expect(step('dates')?.check?.isComplete()).toBe(true);

    expect(step('submit')?.check?.isComplete()).toBe(false);
    created.set(true);
    expect(step('submit')?.check?.isComplete()).toBe(true);

    expect(tutorial.steps[tutorial.steps.length - 1].id).toBe('lifecycle');
    expect(tutorial.steps[tutorial.steps.length - 1].check).toBeUndefined();
  });

  it('el tutorial de identidad no avanza con un color inválido y exige guardado', () => {
    const colorValid = signal(false);
    const contrastOk = signal(true);
    const saved = signal(false);

    const tutorial = buildIdentidadTutorial({
      institutionName: () => 'Universidad',
      colorValid: () => colorValid(),
      contrastOk: () => contrastOk(),
      hasSaved: () => saved()
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('identity')?.check?.isComplete()).toBe(true);
    expect(step('color')?.check?.isComplete()).toBe(false);
    colorValid.set(true);
    expect(step('color')?.check?.isComplete()).toBe(true);

    expect(step('preview')?.check?.isComplete()).toBe(true);
    expect(step('save')?.check?.isComplete()).toBe(false);
    saved.set(true);
    expect(step('save')?.check?.isComplete()).toBe(true);
  });

  it('el tutorial de servidor solo valida acciones ya completadas', () => {
    const policies = signal(false);
    const maintenance = signal(false);
    const backup = signal(false);
    const strategy = signal(false);

    const tutorial = buildServidorTutorial({
      hasSavedPolicies: () => policies(),
      maintenanceModalOpen: () => maintenance(),
      hasTriggeredBackup: () => backup(),
      hasSavedBackupStrategy: () => strategy()
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('policies')?.check?.isComplete()).toBe(false);
    expect(step('maintenance')?.check?.isComplete()).toBe(false);
    expect(step('backups')?.check?.isComplete()).toBe(false);
    expect(step('strategy')?.check?.isComplete()).toBe(false);

    policies.set(true);
    maintenance.set(true);
    backup.set(true);
    strategy.set(true);

    expect(step('policies')?.check?.isComplete()).toBe(true);
    expect(step('maintenance')?.check?.isComplete()).toBe(true);
    expect(step('backups')?.check?.isComplete()).toBe(true);
    expect(step('strategy')?.check?.isComplete()).toBe(true);
  });

  it('el tutorial de IA sigue el orden interruptor, proveedor, modelo, prueba y guardado', () => {
    const enabled = signal(false);
    const provider = signal('');
    const model = signal('');
    const passed = signal(false);
    const saved = signal(false);

    const tutorial = buildIaTutorial({
      enabled: () => enabled(),
      provider: () => provider(),
      model: () => model(),
      testPassed: () => passed(),
      hasSaved: () => saved()
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('enable')?.check?.isComplete()).toBe(false);
    enabled.set(true);
    expect(step('enable')?.check?.isComplete()).toBe(true);

    expect(step('provider')?.check?.isComplete()).toBe(false);
    provider.set('openai');
    expect(step('provider')?.check?.isComplete()).toBe(true);

    expect(step('credentials')?.check?.isComplete()).toBe(false);
    model.set('gpt-4o-mini');
    expect(step('credentials')?.check?.isComplete()).toBe(true);

    expect(step('test')?.check?.isComplete()).toBe(false);
    passed.set(true);
    expect(step('test')?.check?.isComplete()).toBe(true);

    expect(step('governance')?.check).toBeUndefined();
    expect(step('save')?.check?.isComplete()).toBe(false);
    saved.set(true);
    expect(step('save')?.check?.isComplete()).toBe(true);
    expect(tutorial.manualFragment).toBe('9-configuracion-de-inteligencia-artificial');
  });

  it('el tutorial de administradores valida el correo antes de confirmar el alta', () => {
    const open = signal(false);
    const email = signal('');
    const error = signal<string | null>(null);
    const created = signal(false);

    const tutorial = buildAdministradoresTutorial({
      isCreateModalOpen: () => open(),
      email: () => email(),
      formError: () => error(),
      hasCreated: () => created()
    });
    const step = (id: string) => tutorial.steps.find((entry) => entry.id === id);

    expect(step('create')?.check?.isComplete()).toBe(false);
    open.set(true);
    expect(step('create')?.check?.isComplete()).toBe(true);

    expect(step('email')?.check?.isComplete()).toBe(false);
    email.set('operador@');
    expect(step('email')?.check?.isComplete()).toBe(false);
    email.set('operador@uab.edu.bo');
    expect(step('email')?.check?.isComplete()).toBe(true);
    error.set('Ingresa un correo institucional válido.');
    expect(step('email')?.check?.isComplete()).toBe(false);
    error.set(null);

    expect(step('submit')?.check?.isComplete()).toBe(false);
    created.set(true);
    expect(step('submit')?.check?.isComplete()).toBe(true);

    expect(tutorial.steps[tutorial.steps.length - 1].id).toBe('revoke');
    expect(tutorial.steps[tutorial.steps.length - 1].check).toBeUndefined();
  });
});
