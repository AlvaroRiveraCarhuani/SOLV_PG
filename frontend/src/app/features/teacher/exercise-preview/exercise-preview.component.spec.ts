import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter, ActivatedRoute } from '@angular/router';
import { of } from 'rxjs';
import { ExercisePreviewComponent } from './exercise-preview.component';
import { ExerciseEditorStore } from '../exercise-editor/exercise-editor.store';
import { TeacherCourseService } from '../services/teacher-course.service';

describe('ExercisePreviewComponent', () => {
  let component: ExercisePreviewComponent;
  let fixture: ComponentFixture<ExercisePreviewComponent>;
  let store: ExerciseEditorStore;
  let mockCourseService: Partial<TeacherCourseService>;

  beforeEach(async () => {
    mockCourseService = {
      getExercise: vi.fn().mockReturnValue(of({
        id: 'ex-101',
        title: 'Verificación de Delimitadores Balanceados',
        difficulty: 'medium',
        language: 'python',
        boilerplate: 'def solve(): pass',
        reference_solution: 'def solve(): return True',
        time_limit_ms: 1000,
        memory_limit_mb: 128,
        description: 'Verificar si los paréntesis están balanceados.',
        config: {
          algorithm: {
            test_cases: [
              { input: '()', expected_output: 'True', visibility: 'example', weight: 1.0 },
              { input: '(', expected_output: 'False', visibility: 'public', weight: 1.0 }
            ]
          }
        }
      })),
      calculateOutputs: vi.fn().mockReturnValue(of({
        outputs: [
          {
            index: 0,
            input: '()',
            expected_output: 'True',
            status: 'success',
            execution_time_ms: 12
          }
        ]
      }))
    };

    await TestBed.configureTestingModule({
      imports: [ExercisePreviewComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: of(new Map([['courseId', 'course-101'], ['id', 'ex-101']]))
          }
        },
        { provide: TeacherCourseService, useValue: mockCourseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ExercisePreviewComponent);
    component = fixture.componentInstance;
    store = fixture.debugElement.injector.get(ExerciseEditorStore);
    fixture.detectChanges();
  });

  it('debe crearse correctamente e inicializar el ejercicio', () => {
    expect(component).toBeTruthy();
    expect(component.courseId()).toBe('course-101');
    expect(component.exerciseId()).toBe('ex-101');
  });

  it('debe permitir cambiar de lenguaje y alternar el tamaño de fuente', () => {
    component.increaseFontSize();
    expect(component.editorFontSize()).toBe(16);

    component.decreaseFontSize();
    expect(component.editorFontSize()).toBe(14);

    component.onLanguageChange('cpp');
    expect(component.selectedLanguage()).toBe('cpp');
  });

  it('debe cargar la solución canónica y ejecutar la simulación', () => {
    store.cases.set([
      { input: '()', expected_output: 'True', visibility: 'example', weight: 1.0 }
    ]);
    store.referenceSolution.set('def solve(): return True');

    component.loadReferenceSolution();
    expect(component.simulationCode()).toBe('def solve(): return True');

    component.runSimulation();
    expect(mockCourseService.calculateOutputs).toHaveBeenCalled();
  });
});
