import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { ContextHelpService } from './context-help.service';
import { CONTEXT_HELP_ENTRIES } from './help-content.constants';

describe('ContextHelpService', () => {
  let service: ContextHelpService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(ContextHelpService);
  });

  it('abre el centro de ayuda con la pantalla registrada y lo cierra', () => {
    service.registerScreen('dashboard-health');
    service.open();

    expect(service.isOpen()).toBe(true);
    expect(service.entry()?.title).toContain('salud');

    service.close();
    expect(service.isOpen()).toBe(false);
  });

  it('mantiene el contexto de la pantalla al cerrar el panel', () => {
    service.registerScreen('ai-config');
    service.open();
    service.close();

    expect(service.entry()?.key).toBe('ai-config');
    expect(service.isOpen()).toBe(false);
  });

  it('ignora claves desconocidas y deja la pantalla sin ayuda', () => {
    service.registerScreen('no-existe');

    expect(service.screenKey()).toBeNull();
    expect(service.hasContent()).toBe(false);
  });

  it('sin explicación ni guías no se ofrece el acceso', () => {
    expect(service.hasContent()).toBe(false);

    service.registerScreen('templates-catalog');
    expect(service.hasContent()).toBe(true);
  });

  it('toggle cierra el panel ya abierto', () => {
    service.registerScreen('students-penalties');
    service.toggle();
    expect(service.isOpen()).toBe(true);

    service.toggle();
    expect(service.isOpen()).toBe(false);
  });

  it('expone sólo las guías registradas por la pantalla actual', () => {
    service.registerScreen('studio-template-builder');
    service.registerGuides('studio-template-builder', {
      tour: { label: 'Recorrido visual', run: () => undefined },
      tutorial: { label: 'Tutorial guiado', run: () => undefined }
    });

    expect(service.guides().tour?.label).toBe('Recorrido visual');
    expect(service.guides().tutorial?.label).toBe('Tutorial guiado');

    service.unregisterGuides('studio-template-builder');
    expect(service.guides().tour).toBeUndefined();
  });

  it('una pestaña añade su ayuda sobre la de su sección y la devuelve al salir', () => {
    service.registerScreen('config-overview');
    expect(service.entry()?.key).toBe('config-overview');

    service.registerScreen('ai-config');
    expect(service.entry()?.key).toBe('ai-config');

    service.unregisterScreen('ai-config');
    expect(service.entry()?.key).toBe('config-overview');
  });

  it('las guías siguen a la pantalla superior de la pila', () => {
    service.registerScreen('studio-template-builder');
    service.registerGuides('studio-template-builder', {
      tour: { label: 'Recorrido visual', run: () => undefined }
    });
    service.registerScreen('dashboard-health');

    expect(service.guides().tour).toBeUndefined();

    service.unregisterScreen('dashboard-health');
    expect(service.guides().tour?.label).toBe('Recorrido visual');
  });

  it('cada entrada declara contenido y, si tiene ancla, es válida', () => {
    const slugify = (text: string): string =>
      text
        .toLowerCase()
        .normalize('NFD')
        .replace(/[\u0300-\u036f]/g, '')
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)+/g, '');

    for (const entry of Object.values(CONTEXT_HELP_ENTRIES)) {
      expect(entry.items.length).toBeGreaterThan(0);
      if (entry.manualFragment !== null) {
        expect(slugify(entry.manualFragment)).toBe(entry.manualFragment);
      }
    }
  });
});
