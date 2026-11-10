import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, ActivatedRoute } from '@angular/router';
import { of } from 'rxjs';
import { StudentExerciseSolverComponent } from './student-exercise-solver.component';
import { StudentService } from '@core/services/student.service';
import { PracticeEnvironment } from '@core/models/environment.model';
import { AuthService } from '@core/services/auth.service';
import { signal } from '@angular/core';
import {
  ClassAdvanceNoticeService,
  PendingClassAdvance
} from '../../services/class-advance-notice.service';

function pendingAdvance(overrides: Partial<PendingClassAdvance> = {}): PendingClassAdvance {
  return {
    id: 'advance-1',
    session_id: 'session-1',
    exercise_id: 'ex-101',
    file_count: 2,
    created_at: '2026-11-01T15:30:00Z',
    subject_id: 'subj-101',
    session_status: 'open',
    purpose: 'class',
    ...overrides
  };
}

describe('StudentExerciseSolverComponent', () => {
  let component: StudentExerciseSolverComponent;
  let fixture: ComponentFixture<StudentExerciseSolverComponent>;
  let evaluateCalls: unknown[] = [];
  let submitCalls: unknown[] = [];
  let projectSubmitCalls: string[] = [];
  let projectSubmitPayloads: unknown[] = [];
  let startEnvironmentCalls: string[][] = [];

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
    submitPracticeEnvironmentProject: (environmentId: string, payload: unknown) => {
      projectSubmitCalls.push(environmentId);
      projectSubmitPayloads.push(payload);
      return Promise.resolve({
        submission_id: 'sub-exam',
        environment_id: environmentId,
        exercise_id: 'ex-101',
        verdict: 'PENDING'
      });
    },
    startEnvironment: (subjectId: string, exerciseId: string) => {
      startEnvironmentCalls.push([subjectId, exerciseId]);
      return Promise.resolve({
        id: 'ws-1',
        status: 'running',
        access_url: 'http://localhost:3000/ide'
      } as unknown as PracticeEnvironment);
    },
    submissionsHistory: signal([]),
    saveKeystrokeEvents: () => Promise.resolve()
  };

  const mockNotices = {
    pending: signal<PendingClassAdvance[]>([]),
    start: () => undefined,
    stop: () => undefined,
    dismiss: () => Promise.resolve(),
    preview: () =>
      Promise.resolve({
        advance_id: 'advance-1',
        exercise_id: 'ex-101',
        file_count: 2,
        added: ['indices.sql'],
        overwritten: ['main.py']
      }),
    apply: () =>
      Promise.resolve({
        advance_id: 'advance-1',
        files_written: 2,
        overwritten: ['main.py'],
        added: ['indices.sql']
      })
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
    projectSubmitCalls = [];
    projectSubmitPayloads = [];
    startEnvironmentCalls = [];
    mockNotices.pending.set([]);

    await TestBed.configureTestingModule({
      imports: [StudentExerciseSolverComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService },
        { provide: AuthService, useValue: mockAuthService },
        { provide: ClassAdvanceNoticeService, useValue: mockNotices },
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

  it('muestra solo el avance de esta práctica y lo abre para confirmar', async () => {
    mockNotices.pending.set([
      pendingAdvance(),
      pendingAdvance({ id: 'advance-2', exercise_id: 'ex-202' })
    ]);
    fixture.detectChanges();

    expect(component.practiceAdvances().map(advance => advance.id)).toEqual(['advance-1']);
    const callout = fixture.nativeElement.querySelector('.advance-callout') as HTMLElement | null;
    expect(callout?.textContent).toContain('Tu docente emitió un avance de esta práctica');

    component.openAdvance(component.practiceAdvances()[0]);
    fixture.detectChanges();

    expect(component.selectedAdvance()?.id).toBe('advance-1');
    expect(fixture.nativeElement.querySelector('advance-confirm-modal')).not.toBeNull();
  });

  it('entrega el volumen del entorno sin pedir archivos en el navegador', async () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    component.environment.set({ id: 'ws-1', status: 'running' } as unknown as PracticeEnvironment);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.project-editor-body')).toBeNull();
    expect(fixture.nativeElement.textContent).not.toContain('Archivos del proyecto');

    component.requestSubmitProject();
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(projectSubmitCalls).toEqual(['ws-1']);
    expect(projectSubmitPayloads).toEqual([{ exercise_id: 'ex-101' }]);
  });

  it('pide confirmación antes de entregar un examen y no deja reentregar', async () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      purpose: 'exam',
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    component.environment.set({ id: 'ws-1', status: 'running' } as unknown as PracticeEnvironment);
    fixture.detectChanges();

    expect(component.isExamProject()).toBe(true);
    expect(component.canSubmitProject()).toBe(true);

    // Primer clic: solo avisa que el entorno queda en solo lectura.
    component.requestSubmitProject();
    expect(projectSubmitCalls.length).toBe(0);
    expect(component.examSubmitPending()).toBe(true);

    // Segundo clic: entrega y el volumen queda cerrado para ese alumno.
    component.requestSubmitProject();
    await new Promise(resolve => setTimeout(resolve, 0));
    fixture.detectChanges();

    expect(projectSubmitCalls).toEqual(['ws-1']);
    expect(projectSubmitPayloads).toEqual([{ exercise_id: 'ex-101' }]);
    expect(component.projectSubmission()?.submission_id).toBe('sub-exam');
    expect(component.examSubmitPending()).toBe(false);
    expect(component.canSubmitProject()).toBe(false);
  });

  it('abre el entorno de la práctica en un solo paso y sin exponer la URL', async () => {
    const fakeTab = {
      opener: null,
      location: { href: '' },
      close: vi.fn()
    } as unknown as Window;
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(fakeTab);

    component.exercise.set({
      ...component.exercise()!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    fixture.detectChanges();

    await component.openEnvironment();
    fixture.detectChanges();

    expect(startEnvironmentCalls).toEqual([['subj-101', 'ex-101']]);
    expect(fakeTab.location.href).toBe('http://localhost:3000/ide');
    expect(component.environment()?.status).toBe('running');
    expect((fixture.nativeElement as HTMLElement).textContent).not.toContain('http://localhost:3000/ide');

    openSpy.mockRestore();
  });

  it('muestra el mensaje real del backend cuando el entorno no se puede sembrar', async () => {
    const fakeTab = {
      opener: null,
      location: { href: '' },
      close: vi.fn()
    } as unknown as Window;
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(fakeTab);
    const startSpy = vi.spyOn(mockStudentService, 'startEnvironment').mockRejectedValueOnce({
      error: { message: 'El entorno ya contiene archivos y no se volverá a sembrar' }
    });

    component.exercise.set({
      ...component.exercise()!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    fixture.detectChanges();

    await component.openEnvironment();

    expect(component.environmentError()).toBe('El entorno ya contiene archivos y no se volverá a sembrar');
    expect(fakeTab.close).toHaveBeenCalled();
    startSpy.mockRestore();
    openSpy.mockRestore();
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
