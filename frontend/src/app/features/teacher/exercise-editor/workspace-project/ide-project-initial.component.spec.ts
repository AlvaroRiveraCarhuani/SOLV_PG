import { ComponentFixture, TestBed } from '@angular/core/testing';
import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { IdeProjectInitialComponent } from './ide-project-initial.component';
import { ExerciseEditorStore } from '../exercise-editor.store';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherProjectService, WorkspaceProjectViewDTO } from '../../services/teacher-project.service';

function projectView(overrides: Partial<WorkspaceProjectViewDTO> = {}): WorkspaceProjectViewDTO {
  return {
    project: {
      id: 'project-1',
      exercise_id: 'exercise-1',
      current_published_version_id: null,
      created_at: '2026-11-10T10:00:00Z',
      updated_at: '2026-11-10T10:00:00Z'
    },
    draft: {
      id: 'draft-1',
      project_id: 'project-1',
      text_snapshot: '# === FILE: main.py ===\nprint(1)\n',
      content_sha256: 'hash',
      revision: 1,
      updated_at: '2026-11-10T10:00:00Z'
    },
    versions: [],
    publishing_enabled: false,
    ...overrides
  };
}

describe('IdeProjectInitialComponent', () => {
  let fixture: ComponentFixture<IdeProjectInitialComponent>;
  let component: IdeProjectInitialComponent;
  let store: ExerciseEditorStore;
  let getProject: ReturnType<typeof vi.fn>;
  let saveDraft: ReturnType<typeof vi.fn>;
  let importFolder: ReturnType<typeof vi.fn>;
  let importZip: ReturnType<typeof vi.fn>;
  let publish: ReturnType<typeof vi.fn>;
  let generateProject: ReturnType<typeof vi.fn>;
  let retryPreparation: ReturnType<typeof vi.fn>;

  function proposal() {
    return {
      files: [
        { path: 'README.md', content: '# Proyecto\n' },
        { path: 'src/main.py', content: 'print(1)\n' }
      ],
      summary: 'Proyecto mínimo en Python',
      file_count: 2,
      total_bytes: 20
    };
  }

  function generateWithDescription(description = 'Una API mínima en Python'): void {
    component.openAiGenerator();
    component.onAiFieldInput({ target: { value: description } } as unknown as Event, 'description');
    component.generateProposal();
  }

  beforeEach(async () => {
    getProject = vi.fn().mockReturnValue(of(projectView()));
    saveDraft = vi.fn().mockReturnValue(of(projectView().draft));
    importFolder = vi.fn().mockReturnValue(of({ draft: projectView().draft, file_count: 1, total_bytes: 10 }));
    importZip = vi.fn().mockReturnValue(of({ draft: projectView().draft, file_count: 1, total_bytes: 10 }));
    publish = vi.fn().mockReturnValue(of({ id: 'version-1', version_number: 1, text_snapshot: '', content_sha256: '', file_count: 1, total_bytes: 10, published_at: '2026-11-10T10:00:00Z' }));
    generateProject = vi.fn().mockReturnValue(of(proposal()));
    retryPreparation = vi.fn().mockReturnValue(
      of({
        status: 'pending',
        attempts: 0,
        bytes: 0,
        updated_at: '2026-11-10T11:05:00Z',
        can_retry: false
      })
    );

    await TestBed.configureTestingModule({
      imports: [IdeProjectInitialComponent],
      providers: [
        ExerciseEditorStore,
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: { navigate: vi.fn() } },
        { provide: TeacherCourseService, useValue: {} },
        {
          provide: TeacherProjectService,
          useValue: {
            getProject,
            saveDraft,
            importFolder,
            importZip,
            publish,
            generateProject,
            retryPreparation
          }
        }
      ]
    }).compileComponents();

    store = TestBed.inject(ExerciseEditorStore);
    store.exerciseId.set('exercise-1');
    store.updateMetadata({
      template: {
        id: 'tpl-python',
        name: 'Python 3.12',
        stack_config: { runtime_id: 'python-312' }
      }
    });
    fixture = TestBed.createComponent(IdeProjectInitialComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('muestra el borrador, cuenta archivos y oculta la publicación deshabilitada', () => {
    expect(getProject).toHaveBeenCalledWith('exercise-1');
    expect(component.files().map(file => file.path)).toEqual(['main.py']);
    expect(fixture.nativeElement.textContent).toContain('Revisión 1');
    expect(fixture.nativeElement.textContent).toContain('Deshabilitada');
    expect(fixture.nativeElement.textContent).not.toContain('Publicar versión');
  });

  it('advierte cuando el proyecto no coincide con el entorno seleccionado', () => {
    store.updateMetadata({
      template: {
        id: 'tpl-react',
        name: 'React + Vite',
        stack_config: { runtime_id: 'node', web_platform_id: 'react' }
      }
    });

    expect(component.stackWarning()).toContain('Python');
    expect(component.stackWarning()).toContain('React + Vite');
  });

  it('no advierte cuando el proyecto coincide con el entorno seleccionado', () => {
    store.updateMetadata({
      template: {
        id: 'tpl-python',
        name: 'Python 3.12',
        stack_config: { runtime_id: 'python-312' }
      }
    });

    expect(component.stackWarning()).toBeNull();
  });

  it('selecciona un archivo y marca el borrador como modificado al editar', () => {
    component.files.set([
      { path: 'main.py', content: 'print(1)\n' },
      { path: 'src/util.py', content: 'x = 1\n' }
    ]);
    component.selectFile('src/util.py');
    component.onFileContentChange('x = 2\n');

    expect(component.selectedPath()).toBe('src/util.py');
    expect(component.dirty()).toBe(true);
    expect(component.selectedFile()?.content).toBe('x = 2\n');
  });

  it('importa una carpeta y recarga el borrador resultante', () => {
    component.importFolderFiles([{ path: 'src/main.py', content: 'print(2)\n' }]);

    expect(importFolder).toHaveBeenCalled();
    expect(component.importing()).toBe(false);
    expect(component.notice()).toContain('importados');
  });

  it('advierte antes de importar una carpeta incompatible y espera confirmación', () => {
    store.updateMetadata({
      template: {
        id: 'tpl-react',
        name: 'React + Vite',
        stack_config: { runtime_id: 'node', web_platform_id: 'react' }
      }
    });

    component.importFolderFiles([{ path: 'app.py', content: 'print(1)\n' }]);

    expect(importFolder).not.toHaveBeenCalled();
    expect(component.pendingImportWarning()).toContain('Python');

    component.confirmReplaceImport();

    expect(importFolder).toHaveBeenCalled();
  });

  it('rechaza una importación con traversal antes de llamar al backend', () => {
    component.importFolderFiles([{ path: '../secreto.txt', content: 'x\n' }]);

    expect(importFolder).not.toHaveBeenCalled();
    expect(component.errorMessage()).toContain('segmentos ambiguos');
  });

  it('mantiene la edición local y muestra conflicto cuando la revisión es obsoleta', () => {
    saveDraft.mockReturnValue(throwError(() => new HttpErrorResponse({ status: 409 })));
    component.files.set([{ path: 'main.py', content: 'print(2)\n' }]);
    component.saveDraft();

    expect(component.conflict()).toBe(true);
    expect(component.selectedFile()?.content).toBe('print(2)\n');
    expect(component.errorMessage()).toContain('revisión');
  });

  it('abre la confirmación de publicación solo cuando el flag está habilitado', () => {
    component.view.set(projectView({ publishing_enabled: true }));
    fixture.detectChanges();

    expect(component.canPublish()).toBe(true);
    component.openPublish();
    expect(component.publishModal()).toBe(true);
    component.confirmPublish();
    expect(publish).toHaveBeenCalledWith('exercise-1');
  });

  it('no ofrece publicar si el borrador tiene cambios sin guardar', () => {
    getProject.mockReturnValue(of(projectView({ publishing_enabled: true })));
    component.files.set([{ path: 'main.py', content: 'print(3)\n' }]);

    expect(component.canPublish()).toBe(false);
  });

  it('abre el generador y muestra la propuesta sin guardar nada', () => {
    generateWithDescription();

    expect(generateProject).toHaveBeenCalledWith('exercise-1', expect.objectContaining({ description: 'Una API mínima en Python' }));
    expect(component.aiGenerating()).toBe(false);
    expect(component.aiProposal()?.file_count).toBe(2);
    expect(component.aiPreviewFile()?.path).toBe('README.md');
    expect(saveDraft).not.toHaveBeenCalled();
  });

  it('pide confirmación antes de reemplazar cambios sin guardar y aplica la propuesta al confirmar', () => {
    generateWithDescription();
    component.onFileContentChange('print(2)\n');
    expect(component.dirty()).toBe(true);

    component.requestApplyProposal();
    expect(component.pendingProposal()).not.toBeNull();
    expect(component.files().map(file => file.path)).toEqual(['main.py']);

    component.confirmReplaceProposal();
    expect(component.files().map(file => file.path)).toEqual(['README.md', 'src/main.py']);
    expect(component.selectedPath()).toBe('README.md');
    expect(component.notice()).toContain('Propuesta aplicada');
    expect(saveDraft).not.toHaveBeenCalled();
  });

  it('no sobrescribe la propuesta cuando hay un conflicto de revisión', () => {
    generateWithDescription();
    component.conflict.set(true);

    component.requestApplyProposal();

    expect(component.pendingProposal()).toBeNull();
    expect(component.aiError()).toContain('revisión');
    expect(component.files().map(file => file.path)).toEqual(['main.py']);
  });

  it('descarta la propuesta sin tocar el editor', () => {
    generateWithDescription();

    component.discardProposal();

    expect(component.aiProposal()).toBeNull();
    expect(component.files().map(file => file.path)).toEqual(['main.py']);
  });

  it('muestra el error del proveedor y conserva el borrador', () => {
    generateProject.mockReturnValue(throwError(() => new HttpErrorResponse({ status: 504, error: { error: 'ai_timeout', message: 'El motor de IA superó el tiempo límite de espera.' } })));
    generateWithDescription();

    expect(component.aiGenerating()).toBe(false);
    expect(component.aiError()).toContain('tiempo límite');
    expect(component.files().map(file => file.path)).toEqual(['main.py']);
  });

  it('muestra el estado de la preparación de la versión publicada', () => {
    component.view.set(
      projectView({
        project: { ...projectView().project!, current_published_version_id: 'version-1' },
        preparation: {
          status: 'prepared',
          attempts: 1,
          message: 'árbol preparado y comprobado sin red',
          bytes: 2048,
          updated_at: '2026-11-10T11:00:00Z',
          can_retry: false
        }
      })
    );
    fixture.detectChanges();

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Dependencias preparadas');
    expect(text).toContain('árbol preparado');
    expect(text).not.toContain('Reintentar preparación');
  });

  it('reintenta una preparación fallida y vuelve a leer el estado', () => {
    component.view.set(
      projectView({
        project: { ...projectView().project!, current_published_version_id: 'version-1' },
        preparation: {
          status: 'failed',
          attempts: 3,
          message: 'la instalación falló',
          bytes: 0,
          updated_at: '2026-11-10T11:00:00Z',
          can_retry: true
        }
      })
    );
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Reintentar preparación');
    getProject.mockClear();

    component.retryPreparation();

    expect(retryPreparation).toHaveBeenCalledWith('exercise-1');
    expect(getProject).toHaveBeenCalledWith('exercise-1');
    expect(component.notice()).toContain('encolada');
    expect(component.retrying()).toBe(false);
  });

  it('guarda el borrador con el flujo existente después de aceptar la propuesta', () => {
    saveDraft.mockReturnValue(of({ ...projectView().draft, revision: 2 }));
    generateWithDescription();
    component.requestApplyProposal();
    component.saveDraft();

    expect(saveDraft).toHaveBeenCalledWith('exercise-1', expect.stringContaining('README.md'), 1);
    expect(component.revision()).toBe(2);
  });

  it('genera la propuesta con la tecnología del entorno seleccionado', () => {
    generateWithDescription();

    expect(generateProject).toHaveBeenCalledWith(
      'exercise-1',
      expect.objectContaining({
        technology: expect.stringContaining('Python 3.12'),
        template_id: 'tpl-python',
        stack_config: { runtime_id: 'python-312' }
      })
    );
  });

  it('no genera la propuesta sin un entorno seleccionado', () => {
    store.updateMetadata({ template: null });
    component.openAiGenerator();
    component.onAiFieldInput(
      { target: { value: 'Una API mínima' } } as unknown as Event,
      'description'
    );
    component.generateProposal();

    expect(generateProject).not.toHaveBeenCalled();
    expect(component.aiError()).toContain('entorno');
  });

  it('sugiere y aplica un entorno compatible cuando el proyecto no coincide', () => {
    component.files.set([{ path: 'app.py', content: 'print(1)\n' }]);
    store.updateMetadata({
      template: {
        id: 'tpl-react',
        name: 'React + Vite',
        stack_config: { runtime_id: 'node', web_platform_id: 'react' }
      }
    });
    fixture.componentRef.setInput('environments', [
      {
        id: 'tpl-python',
        name: 'Python 3.12',
        docker_image: 'solv-local/python:3.12',
        base_ram_mb: 512,
        target_environment: 'IDE_PERSISTENTE',
        status: 'approved',
        stack_config: { runtime_id: 'python-312' }
      }
    ]);
    fixture.detectChanges();

    expect(component.compatibleEnvironments().map(environment => environment.id)).toEqual([
      'tpl-python'
    ]);

    component.selectEnvironment(component.compatibleEnvironments()[0]);

    expect(store.metadata().template_id).toBe('tpl-python');
    expect(component.stackWarning()).toBeNull();
  });

  it('exige confirmar la incompatibilidad antes de publicar el proyecto', () => {
    component.view.set(projectView({ publishing_enabled: true }));
    store.updateMetadata({
      template: {
        id: 'tpl-react',
        name: 'React + Vite',
        stack_config: { runtime_id: 'node', web_platform_id: 'react' }
      }
    });
    fixture.detectChanges();

    component.openPublish();
    component.confirmPublish();
    expect(publish).not.toHaveBeenCalled();

    component.onStackAcknowledged({ target: { checked: true } } as unknown as Event);
    component.confirmPublish();
    expect(publish).toHaveBeenCalledWith('exercise-1');
  });
});