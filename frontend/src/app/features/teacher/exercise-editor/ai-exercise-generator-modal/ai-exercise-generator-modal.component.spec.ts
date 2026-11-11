import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AiExerciseGeneratorModalComponent } from './ai-exercise-generator-modal.component';
import {
  TeacherAiCopilotService,
  AIGenerateExerciseResponse,
  AIUserOverrideDTO,
  AIAvailabilityStatus
} from '../../services/teacher-ai-copilot.service';
import { ExerciseEditorStore } from '../exercise-editor.store';
import { AiFlowDrafts } from '@shared/services/ai-flow-drafts';

const mockDraft: AIGenerateExerciseResponse = {
  title: 'Suma de primos hasta N',
  description: 'Calcula la suma de todos los primos menores o iguales a N.',
  difficulty: 'medium',
  language: 'python',
  boilerplate: 'def suma_primos(n: int) -> int:\n    pass\n',
  reference_solution: 'def suma_primos(n):\n    return sum(p for p in range(2, n + 1))\n',
  test_cases: [
    {
      order_index: 1,
      input: '10',
      expected_output: '17',
      visibility: 'example',
      weight: 1,
      verified_by_runner: true
    },
    {
      order_index: 2,
      input: '20',
      expected_output: '77',
      visibility: 'hidden',
      weight: 2,
      verified_by_runner: true
    }
  ],
  runner_verified_all: true,
  latency_ms: 1234
};

const mockUserOverride: AIUserOverrideDTO = {
  enabled: true,
  provider: 'openrouter',
  provider_label: 'OpenRouter',
  model: 'meta-llama/llama-3.3-70b-instruct:free',
  base_url: '',
  has_api_key: true,
  api_key_masked: 'sk-or-•••••••1234',
  updated_at: '2026-10-25T12:00:00Z',
  institutional_enabled: true,
  institutional_provider_label: 'Gemini',
  institutional_model: 'gemini-2.5-flash'
};

const mockStatus: AIAvailabilityStatus = {
  enabled: true,
  provider: 'openrouter',
  model: 'meta-llama/llama-3.3-70b-instruct:free',
  using_personal_key: true,
  institutional_enabled: true
};

describe('AiExerciseGeneratorModalComponent', () => {
  let component: AiExerciseGeneratorModalComponent;
  let fixture: ComponentFixture<AiExerciseGeneratorModalComponent>;
  let mockService: {
    generateExercise: ReturnType<typeof vi.fn>;
    resolveError: ReturnType<typeof vi.fn>;
    getStatus: ReturnType<typeof vi.fn>;
    getMySettings: ReturnType<typeof vi.fn>;
    updateMySettings: ReturnType<typeof vi.fn>;
    testMyConnection: ReturnType<typeof vi.fn>;
    listMyModels: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    mockService = {
      generateExercise: vi.fn().mockReturnValue(of(mockDraft)),
      resolveError: vi.fn().mockReturnValue('error de prueba'),
      getStatus: vi.fn().mockReturnValue(of(mockStatus)),
      getMySettings: vi.fn().mockReturnValue(of(mockUserOverride)),
      updateMySettings: vi.fn().mockReturnValue(of(mockUserOverride)),
      testMyConnection: vi.fn().mockReturnValue(of({ ok: true, message: 'Conexión OK (120ms)', latency_ms: 120 })),
      listMyModels: vi.fn().mockReturnValue(of(['google/gemini-2.0-flash-exp:free', 'meta-llama/llama-3.3-70b-instruct:free', 'openai/gpt-4o-mini']))
    };

    await TestBed.configureTestingModule({
      imports: [AiExerciseGeneratorModalComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        { provide: TeacherAiCopilotService, useValue: mockService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AiExerciseGeneratorModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe inicializar cargando el estado del motor de IA', () => {
    expect(mockService.getStatus).toHaveBeenCalled();
    expect(component.aiStatus()?.using_personal_key).toBe(true);
  });

  it('delega la credencial personal al panel compartido y refresca el estado al guardar', () => {
    expect(component.showByokPanel()).toBe(false);
    expect(fixture.nativeElement.querySelector('ai-byok-panel')).toBeNull();

    component.toggleByokPanel();
    fixture.detectChanges();
    expect(component.showByokPanel()).toBe(true);
    expect(fixture.nativeElement.querySelector('ai-byok-panel')).toBeTruthy();

    const callsBefore = mockService.getStatus.mock.calls.length;
    component.onByokChanged();
    expect(mockService.getStatus.mock.calls.length).toBe(callsBefore + 1);
  });

  it('debe invocar generateExercise con los parámetros seleccionados y mostrar el resultado en la tarjeta estructurada', () => {
    component.prompt.set('Suma de los primos hasta N con casos extremos');
    component.language.set('python');
    component.difficulty.set('hard');
    component.publicCasesCount.set(3);
    component.hiddenCasesCount.set(4);

    component.generate();

    expect(mockService.generateExercise).toHaveBeenCalledWith({
      prompt: 'Suma de los primos hasta N con casos extremos',
      language: 'python',
      difficulty: 'hard',
      environment_type: 'JUEZ_EFIMERO',
      public_cases_count: 3,
      hidden_cases_count: 4,
      verify_with_runner: true
    });

    const res = component.generatedResult();
    expect(res).toBeTruthy();
    expect(res?.runner_verified_all).toBe(true);
    expect(res?.latency_ms).toBe(1234);
    expect(component.isGenerating()).toBe(false);

    fixture.detectChanges();
    const card = fixture.nativeElement.querySelector('.exercise-preview-card');
    expect(card).toBeTruthy();
    const latency = fixture.nativeElement.querySelector('.chip-latency');
    expect(latency?.textContent).toContain('1234 ms');
    const seal = fixture.nativeElement.querySelector('.seal-banner.seal-ok');
    expect(seal?.textContent).toContain('El motor verificó las salidas esperadas');
  });

  it('debe invocar generateExercise en modo IDE sin lenguaje ni casos de juez', () => {
    component.environmentType = 'IDE_PERSISTENTE';
    component.prompt.set('Práctica de NestJS con base de datos');
    component.difficulty.set('medium');

    component.generate();

    expect(mockService.generateExercise).toHaveBeenCalledWith({
      prompt: 'Práctica de NestJS con base de datos',
      language: undefined,
      difficulty: 'medium',
      environment_type: 'IDE_PERSISTENTE',
      template_id: undefined,
      public_cases_count: 0,
      hidden_cases_count: 0,
      verify_with_runner: false
    });
  });

  it('envía template_id cuando el entorno ya fue preseleccionado por el docente', () => {
    component.environmentType = 'IDE_PERSISTENTE';
    component.templateId = 'tpl-python-postgres';
    component.prompt.set('Práctica de FastAPI con PostgreSQL');
    component.difficulty.set('hard');

    component.generate();

    expect(mockService.generateExercise).toHaveBeenCalledWith({
      prompt: 'Práctica de FastAPI con PostgreSQL',
      language: undefined,
      difficulty: 'hard',
      environment_type: 'IDE_PERSISTENTE',
      template_id: 'tpl-python-postgres',
      public_cases_count: 0,
      hidden_cases_count: 0,
      verify_with_runner: false
    });
  });

  it('conserva la plantilla homologada resuelta por el backend al aplicar un borrador IDE', () => {
    component.environmentType = 'IDE_PERSISTENTE';
    component.generatedResult.set({
      ...mockDraft,
      environment_type: 'IDE_PERSISTENTE',
      template_id: 'ai-suggested-template',
      template_name: 'Plantilla sugerida por IA',
      missing_technology: 'Svelte'
    });
    component.reviewDraft();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('#ai-template-choice')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('no encontró un entorno homologado');
    expect(component.canApplyDraft()).toBe(true);

    const appliedSpy = vi.spyOn(component.applied, 'emit');
    component.applyDraft();

    const appliedDraft = appliedSpy.mock.calls[0]?.[0];
    expect(appliedDraft).toEqual(expect.objectContaining({
      environment_type: 'IDE_PERSISTENTE',
      template_id: 'ai-suggested-template',
      template_name: 'Plantilla sugerida por IA'
    }));

    const store = TestBed.inject(ExerciseEditorStore);
    store.applyAIGeneratedDraft(appliedDraft!);
    expect(store.metadata().template_id).toBe('ai-suggested-template');
  });

  it('permite editar el borrador y volver a revisarlo sin perder cambios', () => {
    component.generatedResult.set({
      ...mockDraft,
      rubric: {
        criteria: [{
          name: 'Correctitud',
          description: 'Resuelve el problema',
          weight: 100,
          levels: [{ name: 'Excelente', description: 'Solución completa', score: 100 }]
        }]
      }
    });
    component.startEditingDraft();
    component.updateDraft({ title: 'Título corregido', description: 'Enunciado editado' });
    component.updateRubricCriterion(0, { weight: 80 });
    component.updateRubricLevel(0, 0, { description: 'Nivel ajustado' });
    component.updateTestCase(0, { expected_output: '18' });
    component.reviewDraft();
    fixture.detectChanges();

    expect(component.draftStep()).toBe('review');
    expect(fixture.nativeElement.textContent).toContain('Enunciado editado');
    expect(component.rubricTotal(component.draftForReview()!.rubric!.criteria)).toBe(80);
    expect(component.allTestCasesVerified(component.draftForReview()!)).toBe(false);

    const appliedSpy = vi.spyOn(component.applied, 'emit');
    component.applyDraft();
    const appliedDraft = appliedSpy.mock.calls[0]?.[0];
    expect(appliedDraft).toEqual(expect.objectContaining({
      title: 'Título corregido',
      description: 'Enunciado editado',
      test_cases: expect.arrayContaining([expect.objectContaining({ expected_output: '18', verified_by_runner: false })])
    }));
  });

  it('conserva las ediciones al cambiar al prompt y confirma antes de descartar', () => {
    component.generatedResult.set(mockDraft);
    component.startEditingDraft();
    component.updateDraft({ title: 'Conservado' });
    component.editPrompt();
    fixture.detectChanges();

    expect(component.draftStep()).toBe('prompt');
    component.reviewDraft();
    expect(component.draftForReview()?.title).toBe('Conservado');

    component.requestDiscardDraft();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.discard-confirm')).toBeTruthy();
    component.cancelDiscardDraft();
    expect(component.draftForReview()?.title).toBe('Conservado');

    component.requestDiscardDraft();
    component.discardDraft();
    expect(component.generatedResult()).toBeNull();
    expect(component.editingDraft()).toBeNull();
  });

  it('debe cancelar la generación en curso al llamar cancelGeneration()', () => {
    component.prompt.set('Ejercicio largo que demora');
    component.generate();
    expect(component.isGenerating()).toBe(false); // mock observable is synchronous

    // Simular que está en curso
    component.generate();
    component.cancelGeneration();

    expect(component.isGenerating()).toBe(false);
    expect(component.errorMessage()).toContain('cancelada por el docente');
  });

  it('debe emitir (applied) al confirmar y que applyAIGeneratedDraft pueble el editor', () => {
    const appliedSpy = vi.spyOn(component.applied, 'emit');
    const closedSpy = vi.spyOn(component.closed, 'emit');

    component.generatedResult.set(mockDraft);
    component.applyDraft();

    expect(appliedSpy).toHaveBeenCalledWith(mockDraft);
    expect(closedSpy).toHaveBeenCalled();

    const store = TestBed.inject(ExerciseEditorStore);
    store.applyAIGeneratedDraft(mockDraft);

    expect(store.metadata().title).toBe('Suma de primos hasta N');
    expect(store.metadata().difficulty).toBe('medium');
    expect(store.metadata().language).toBe('python');
    expect(store.statement().description).toBe(mockDraft.description);
    expect(store.referenceSolution()).toBe(mockDraft.reference_solution);
    expect(store.boilerplate()).toBe(mockDraft.boilerplate);
    expect(store.boilerplates()['python']).toBe(mockDraft.boilerplate);

    const cases = store.cases();
    expect(cases.length).toBe(2);
    expect(cases[0].expected_output).toBe('17');
    expect(cases[0].is_sample).toBe(true);
    expect(cases[0].is_hidden).toBe(false);
    expect(cases[1].is_hidden).toBe(true);
    expect(cases[1].weight).toBe(2);
    expect(store.hasInteracted()).toBe(true);
  });

  it('conserva la consigna y los parámetros al cerrar y reabrir el modal', () => {
    component.prompt.set('Recorrido de árboles binarios con casos ocultos');
    component.difficulty.set('hard');
    component.publicCasesCount.set(4);
    component.hiddenCasesCount.set(6);
    fixture.detectChanges();
    fixture.destroy();

    const reopened = TestBed.createComponent(AiExerciseGeneratorModalComponent);
    reopened.detectChanges();
    const instance = reopened.componentInstance;

    expect(instance.prompt()).toBe('Recorrido de árboles binarios con casos ocultos');
    expect(instance.difficulty()).toBe('hard');
    expect(instance.publicCasesCount()).toBe(4);
    expect(instance.hiddenCasesCount()).toBe(6);
  });

  it('limpia el borrador guardado cuando el resultado se aplica al editor', () => {
    const drafts = TestBed.inject(AiFlowDrafts);
    component.prompt.set('Suma de primos');
    fixture.detectChanges();
    expect(drafts.get('generate-exercise')).not.toBeNull();

    component.generatedResult.set(mockDraft);
    component.applyDraft();

    expect(drafts.get('generate-exercise')).toBeNull();
  });

  it('dispara generate al presionar Ctrl+Enter o Cmd+Enter en el prompt', () => {
    const generateSpy = vi.spyOn(component, 'generate');
    const eventCtrl = new KeyboardEvent('keydown', { key: 'Enter', ctrlKey: true });
    const preventCtrl = vi.spyOn(eventCtrl, 'preventDefault');
    component.onPromptKeydown(eventCtrl);
    expect(preventCtrl).toHaveBeenCalled();
    expect(generateSpy).toHaveBeenCalledTimes(1);

    const eventMeta = new KeyboardEvent('keydown', { key: 'Enter', metaKey: true });
    const preventMeta = vi.spyOn(eventMeta, 'preventDefault');
    component.onPromptKeydown(eventMeta);
    expect(preventMeta).toHaveBeenCalled();
    expect(generateSpy).toHaveBeenCalledTimes(2);

    const regularEnter = new KeyboardEvent('keydown', { key: 'Enter', ctrlKey: false, metaKey: false });
    const preventRegular = vi.spyOn(regularEnter, 'preventDefault');
    component.onPromptKeydown(regularEnter);
    expect(preventRegular).not.toHaveBeenCalled();
    expect(generateSpy).toHaveBeenCalledTimes(2);
  });
});
