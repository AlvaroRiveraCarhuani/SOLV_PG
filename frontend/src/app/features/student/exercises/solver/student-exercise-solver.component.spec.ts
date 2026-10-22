import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, ActivatedRoute } from '@angular/router';
import { of } from 'rxjs';
import { StudentExerciseSolverComponent } from './student-exercise-solver.component';
import { StudentService } from '@core/services/student.service';
import { AuthService } from '@core/services/auth.service';
import { signal } from '@angular/core';

describe('StudentExerciseSolverComponent', () => {
  let component: StudentExerciseSolverComponent;
  let fixture: ComponentFixture<StudentExerciseSolverComponent>;
  let evaluateCalls: unknown[] = [];
  let submitCalls: unknown[] = [];

  const mockStudentService = {
    getPublicExercise: () =>
      Promise.resolve({
        id: 'ex-101',
        subject_id: 'subj-101',
        title: 'Suma de Enteros',
        description: 'Lee dos números y muestra su suma.',
        type: 'algorithm',
        environment_type: 'JUEZ_EFIMERO',
        difficulty: 'easy',
        purpose: 'class',
        language: 'python',
        boilerplate: 'a, b = map(int, input().split())\nprint(a + b)\n',
        time_limit_ms: 1000,
        memory_limit_mb: 128,
        constraints: ['Forbidden import: os'],
        public_test_cases: [{ input: '2 3', expected_output: '5' }]
      }),
    evaluateExercise: (payload: unknown) => {
      evaluateCalls.push(payload);
      return Promise.resolve({
        verdict: 'AC' as const,
        execution_time_ms: 18,
        memory_used_mb: 14,
        message: '¡Solución Aceptada!',
        case_results: [{ index: 0, verdict: 'AC' as const, duration_ms: 18 }]
      });
    },
    submitExercise: (payload: unknown) => {
      submitCalls.push(payload);
      return Promise.resolve({ id: 'sub-909' });
    },
    saveKeystrokeEvents: () => Promise.resolve()
  };

  const mockAuthService = {
    currentUser: signal({
      id: 'student-uuid-1',
      email: 'student@uab.edu.bo',
      full_name: 'Estudiante Test',
      role: 'student',
      tenant_id: '00000000-0000-0000-0000-000000000001'
    })
  };

  beforeEach(async () => {
    evaluateCalls = [];
    submitCalls = [];

    await TestBed.configureTestingModule({
      imports: [StudentExerciseSolverComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService },
        { provide: AuthService, useValue: mockAuthService },
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: of(new Map([['exerciseId', 'ex-101']]))
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentExerciseSolverComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should load public exercise without hidden test cases and submit solution through the judge', async () => {
    expect(component).toBeTruthy();
    expect(component.exercise()?.title).toBe('Suma de Enteros');
    expect(component.exampleCases().length).toBe(1);

    await component.submitSolution();
    expect(evaluateCalls.length).toBe(1);
    expect(submitCalls.length).toBe(1);
    expect(component.lastResult()?.verdict).toBe('AC');
    expect(component.lastSubmissionId()).toBe('sub-909');
  });
});
