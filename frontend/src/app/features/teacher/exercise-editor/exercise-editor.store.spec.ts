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
});
