import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { ExerciseEditorStore } from './exercise-editor.store';

describe('ExerciseEditorStore', () => {
  let store: ExerciseEditorStore;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ExerciseEditorStore, provideHttpClient(), provideHttpClientTesting(), provideRouter([])]
    });
    store = TestBed.inject(ExerciseEditorStore);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('bloquea publicar si hay script SQL y la plantilla no incluye base de datos', () => {
    store.metadata.update(m => ({
      ...m,
      title: 'Práctica',
      environment_type: 'IDE_PERSISTENTE',
      modality: 'environment',
      template_id: 'tpl-cli',
      db_init_script: 'SELECT 1;',
      template: { id: 'tpl-cli', name: 'Python CLI', services_config: { services: [] } }
    }));
    store.statement.set({ description: 'Consigna', input_text: '', output_text: '', constraints: '' });

    expect(store.missingPublishRequirements()).toContain(
      'Quitar el script SQL o elegir una plantilla con base de datos.'
    );
  });

  it('no bloquea publicar si la plantilla incluye base de datos', () => {
    store.metadata.update(m => ({
      ...m,
      title: 'Práctica',
      environment_type: 'IDE_PERSISTENTE',
      modality: 'environment',
      template_id: 'tpl-mysql',
      db_init_script: 'SELECT 1;',
      template: {
        id: 'tpl-mysql',
        name: 'MySQL',
        services_config: { services: [{ category: 'database', engine: 'mysql' }] }
      }
    }));
    store.statement.set({ description: 'Consigna', input_text: '', output_text: '', constraints: '' });

    expect(store.missingPublishRequirements()).not.toContain(
      'Quitar el script SQL o elegir una plantilla con base de datos.'
    );
  });

  it('cargar las plantillas de juez no cambia la plantilla del Entorno de Desarrollo', () => {
    store.metadata.update(m => ({
      ...m,
      environment_type: 'IDE_PERSISTENTE',
      modality: 'environment',
      template_id: 'tpl-ide-mysql',
      language: 'python'
    }));

    store.loadRunnerTemplates();
    http.expectOne(request => request.url.startsWith('/api/v1/templates')).flush({
      data: {
        data: [
          {
            id: 'tpl-runner-python',
            name: 'Python Runner Base',
            target_environment: 'JUEZ_EFIMERO',
            status: 'APROBADA',
            docker_image: 'python:3.12'
          }
        ]
      }
    });

    expect(store.runnerTemplates().length).toBe(1);
    expect(store.metadata().template_id).toBe('tpl-ide-mysql');
  });

  it('un entorno con varios ejecutores habilita todos sus lenguajes en la práctica', () => {
    store.loadRunnerTemplates();
    http.expectOne(request => request.url.startsWith('/api/v1/templates')).flush({
      data: {
        data: [
          {
            id: 'tpl-multi',
            name: 'Python y C++',
            target_environment: 'JUEZ_EFIMERO',
            status: 'APROBADA',
            docker_image: 'python:3.12-slim-bookworm',
            base_ram_mb: 256,
            language_runners: [
              { language: 'python', entrypoint: 'python3 solution.py' },
              { language: 'cpp', entrypoint: 'g++ solution.cpp -o solution' }
            ]
          }
        ]
      }
    });

    store.setLanguageAndSyncTemplate('python');

    expect(store.metadata().template_id).toBe('tpl-multi');
    expect(store.metadata().language).toBe('python');
    expect(store.metadata().allowed_languages).toEqual(['python', 'cpp']);
  });

  it('permite alternar lenguajes adicionales admitidos y vincula la plantilla multilingue', () => {
    store.loadRunnerTemplates();
    http.expectOne(request => request.url.startsWith('/api/v1/templates')).flush({
      data: {
        data: [
          {
            id: 'tpl-python-single',
            name: 'Python Runner Base',
            target_environment: 'JUEZ_EFIMERO',
            status: 'approved',
            docker_image: 'python:3.11-slim',
            base_ram_mb: 128,
            language_runners: [{ language: 'python', entrypoint: 'python3 solution.py' }]
          },
          {
            id: 'tpl-multi-all',
            name: 'Runner Multilenguaje Base',
            target_environment: 'JUEZ_EFIMERO',
            status: 'approved',
            docker_image: 'python:3.11-slim',
            base_ram_mb: 256,
            language_runners: [
              { language: 'python', entrypoint: 'python3 solution.py' },
              { language: 'cpp', entrypoint: 'g++ solution.cpp -o solution' },
              { language: 'java', entrypoint: 'javac Main.java' }
            ]
          }
        ]
      }
    });

    store.setLanguageAndSyncTemplate('python');
    expect(store.metadata().allowed_languages).toEqual(['python']);

    // Tildar C++ adicional
    store.toggleAllowedLanguage('cpp');
    expect(store.metadata().allowed_languages).toEqual(['python', 'cpp']);
    expect(store.metadata().template_id).toBe('tpl-multi-all');

    // Intentar destildar el lenguaje principal no tiene efecto
    store.toggleAllowedLanguage('python');
    expect(store.metadata().allowed_languages).toContain('python');

    // Destildar C++ vuelve al estado monolenguaje
    store.toggleAllowedLanguage('cpp');
    expect(store.metadata().allowed_languages).toEqual(['python']);
  });

  it('resuelve una plantilla multilingue para un lenguaje secundario que no tiene plantilla individual', () => {
    store.loadRunnerTemplates();
    http.expectOne(request => request.url.startsWith('/api/v1/templates')).flush({
      data: {
        data: [
          {
            id: 'tpl-multi-modern',
            name: 'Entorno Multilingue Moderno',
            target_environment: 'JUEZ_EFIMERO',
            status: 'approved',
            docker_image: 'solv-runners/modern:1.0',
            base_ram_mb: 256,
            language_runners: [
              { language: 'python', entrypoint: 'python3 solution.py' },
              { language: 'julia', entrypoint: 'julia solution.jl' }
            ]
          }
        ]
      }
    });

    const resolved = store.findTemplateForLanguage('julia');
    expect(resolved?.id).toBe('tpl-multi-modern');

    store.setLanguageAndSyncTemplate('julia');
    expect(store.metadata().template_id).toBe('tpl-multi-modern');
    expect(store.metadata().language).toBe('julia');
  });
});
