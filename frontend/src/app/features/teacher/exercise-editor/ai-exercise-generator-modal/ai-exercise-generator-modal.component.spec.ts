import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AiExerciseGeneratorModalComponent } from './ai-exercise-generator-modal.component';
import {
  TeacherAiCopilotService,
  AIGenerateExerciseResponse
} from '../../services/teacher-ai-copilot.service';
import { ExerciseEditorStore } from '../exercise-editor.store';

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

describe('AiExerciseGeneratorModalComponent', () => {
  let component: AiExerciseGeneratorModalComponent;
  let fixture: ComponentFixture<AiExerciseGeneratorModalComponent>;
  let mockService: {
    generateExercise: ReturnType<typeof vi.fn>;
    resolveError: ReturnType<typeof vi.fn>;
    getStatus: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    mockService = {
      generateExercise: vi.fn().mockReturnValue(of(mockDraft)),
      resolveError: vi.fn().mockReturnValue('error de prueba'),
      getStatus: vi.fn()
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

  it('debe invocar generateExercise con los parámetros seleccionados y mostrar el resultado sellado', () => {
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
    const latency = fixture.nativeElement.querySelector('.preview-summary code.font-mono');
    expect(latency?.textContent).toContain('1234 ms');
    const seal = fixture.nativeElement.querySelector('.seal-banner.seal-ok');
    expect(seal?.textContent).toContain('selladas por el ejecutor del Juez Virtual');
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
});
