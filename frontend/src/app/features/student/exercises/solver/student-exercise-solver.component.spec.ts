import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, ActivatedRoute, Router } from '@angular/router';
import { Location } from '@angular/common';
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
  let exportCalls: string[] = [];
  let environmentLookupCalls: string[][] = [];

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
    exportPracticeEnvironment: (environmentId: string) => {
      exportCalls.push(environmentId);
      return Promise.resolve({
        blob: new Blob(['zip'], { type: 'application/zip' }),
        filename: 'solv-practica.zip'
      });
    },
    getPracticeEnvironmentForExercise: (subjectId: string, exerciseId: string) => {
      environmentLookupCalls.push([subjectId, exerciseId]);
      return Promise.resolve(null as PracticeEnvironment | null);
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
    exportCalls = [];
    environmentLookupCalls = [];
    mockNotices.pending.set([]);

    await TestBed.configureTestingModule({
      imports: [StudentExerciseSolverComponent],
      providers: [
        provideRouter([{ path: '**', children: [] }]),
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
    expect(callout?.textContent).toContain('Su docente emitió un avance para esta práctica');

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

  it('exporta el proyecto del entorno abierto para correrlo fuera de SOLV', async () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    component.environment.set({ id: 'ws-1', status: 'running' } as unknown as PracticeEnvironment);
    fixture.detectChanges();

    const createObjectURL = vi.fn(() => 'blob:mock');
    const revokeObjectURL = vi.fn();
    Object.defineProperty(URL, 'createObjectURL', { value: createObjectURL, configurable: true });
    Object.defineProperty(URL, 'revokeObjectURL', { value: revokeObjectURL, configurable: true });

    expect(component.canExportProject()).toBe(true);
    await component.exportProject();

    expect(exportCalls).toEqual(['ws-1']);
    expect(component.isExportingProject()).toBe(false);
    expect(component.exportError()).toBeNull();
    expect(createObjectURL).toHaveBeenCalledTimes(1);
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock');
  });

  it('habilita exportar cuando ya existe un entorno previo sin abrirlo', async () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    fixture.detectChanges();

    const service = TestBed.inject(StudentService) as unknown as {
      getPracticeEnvironmentForExercise: (
        subjectId: string,
        exerciseId: string
      ) => Promise<PracticeEnvironment | null>;
    };
    const original = service.getPracticeEnvironmentForExercise;
    service.getPracticeEnvironmentForExercise = (subjectId, exerciseId) => {
      environmentLookupCalls.push([subjectId, exerciseId]);
      return Promise.resolve({ id: 'ws-old', status: 'hibernated' } as PracticeEnvironment);
    };
    try {
      await component.loadPreviousEnvironment();

      expect(environmentLookupCalls[environmentLookupCalls.length - 1]).toEqual(['subj-101', 'ex-101']);
      expect(component.canExportProject()).toBe(true);
      expect(component.environment()).toBeNull();
    } finally {
      service.getPracticeEnvironmentForExercise = original;
    }
  });

  it('pide abrir el entorno una vez cuando no hay entorno previo', async () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    fixture.detectChanges();

    await component.loadPreviousEnvironment();

    expect(component.canExportProject()).toBe(false);
    expect(fixture.nativeElement.textContent).toContain(
      'Abre el entorno al menos una vez para exportar tu código'
    );
  });

  it('explica cuando el proyecto excede el tamaño del paquete', async () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    component.environment.set({ id: 'ws-1', status: 'running' } as unknown as PracticeEnvironment);
    fixture.detectChanges();

    const service = TestBed.inject(StudentService) as unknown as {
      exportPracticeEnvironment: (environmentId: string) => Promise<{ blob: Blob; filename: string }>;
    };
    const original = service.exportPracticeEnvironment;
    service.exportPracticeEnvironment = () => Promise.reject({ status: 413 });
    try {
      await component.exportProject();

      expect(component.isExportingProject()).toBe(false);
      expect(component.exportError()).toContain('demasiado grande');
    } finally {
      service.exportPracticeEnvironment = original;
    }
  });

  it('no ofrece exportar mientras se rinde un examen', () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      purpose: 'exam',
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    component.environment.set({ id: 'ws-1', status: 'running', state: 'ready' } as unknown as PracticeEnvironment);
    fixture.detectChanges();

    expect(component.canExportProject()).toBe(false);
    const button = fixture.nativeElement.querySelector('[data-testid="btn-export-project"]') as HTMLButtonElement | null;
    expect(button?.disabled).toBe(true);
  });

  it('muestra los tres estados del examen y recuerda que la práctica sigue intacta', () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      purpose: 'exam',
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });

    component.environment.set({ id: 'ws-1', status: 'pending' } as unknown as PracticeEnvironment);
    fixture.detectChanges();
    expect(component.environmentState()).toBe('preparing');
    expect(fixture.nativeElement.textContent).toContain('Preparando su examen');

    component.environment.set({
      id: 'ws-1',
      status: 'running',
      state: 'ready'
    } as unknown as PracticeEnvironment);
    fixture.detectChanges();
    expect(component.environmentState()).toBe('ready');
    expect(fixture.nativeElement.textContent).toContain('Está rindiendo el examen');
    expect(fixture.nativeElement.textContent).toContain('Su práctica de esta materia sigue intacta');

    component.environment.set({ id: 'ws-1', status: 'frozen' } as unknown as PracticeEnvironment);
    fixture.detectChanges();
    expect(component.environmentState()).toBe('closed');
    expect(fixture.nativeElement.textContent).toContain('Su espacio está en solo lectura');
  });

  it('muestra el aviso de solo lectura cuando el examen ya terminó y el entorno no carga', async () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      purpose: 'exam',
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null);
    const startSpy = vi.spyOn(mockStudentService, 'startEnvironment').mockRejectedValueOnce({
      error: {
        error: { code: 'environment_frozen' },
        message: 'El examen terminó: tu entorno quedó en solo lectura'
      }
    });

    await component.openEnvironment();
    fixture.detectChanges();

    expect(component.environment()).toBeNull();
    expect(component.examClosed()).toBe(true);
    expect(component.environmentState()).toBe('closed');
    expect(fixture.nativeElement.textContent).toContain('Su espacio está en solo lectura');

    startSpy.mockRestore();
    openSpy.mockRestore();
  });

  it('explica la pausa de la clase cuando el laboratorio de práctica quedó en solo lectura', async () => {
    const current = component.exercise();
    component.exercise.set({
      ...current!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null);
    const startSpy = vi.spyOn(mockStudentService, 'startEnvironment').mockRejectedValueOnce({
      error: {
        error: 'environment_frozen_class',
        message: 'El docente pausó la clase: tu entorno quedó en solo lectura'
      }
    });

    await component.openEnvironment();
    fixture.detectChanges();

    expect(component.isExamProject()).toBe(false);
    expect(component.classPaused()).toBe(true);
    expect(component.examClosed()).toBe(false);
    expect(fixture.nativeElement.textContent).toContain('El docente pausó la clase');
    expect(fixture.nativeElement.textContent).not.toContain('El examen terminó');
    expect(fixture.nativeElement.textContent).not.toContain('Su espacio está en solo lectura');

    startSpy.mockRestore();
    openSpy.mockRestore();
  });

  it('deja abrir el laboratorio de una práctica pausada en solo lectura y bloquea la entrega', () => {
    component.exercise.set({
      ...component.exercise()!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE'
    });
    component.environment.set({
      id: 'ws-1',
      status: 'frozen',
      access_url: 'http://localhost:3998'
    } as unknown as PracticeEnvironment);
    fixture.detectChanges();

    expect(component.isExamProject()).toBe(false);
    expect(component.showsClassPausedNotice()).toBe(true);
    expect(component.canSubmitProject()).toBe(false);
    expect(fixture.nativeElement.textContent).toContain('El docente pausó la clase');
    expect(fixture.nativeElement.textContent).not.toContain('El examen terminó');
  });

  it('no muestra el aviso de examen en una práctica', () => {
    component.environment.set({ id: 'ws-1', status: 'running' } as unknown as PracticeEnvironment);
    fixture.detectChanges();

    expect(component.environmentState()).toBe('ready');
    expect(fixture.nativeElement.querySelector('.exam-state')).toBeNull();
  });

  it('pide confirmación y permite reentregar un examen mientras siga abierto', async () => {
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

    // Primer clic: abre confirm-modal
    component.requestSubmitProject();
    expect(projectSubmitCalls.length).toBe(0);
    expect(component.examConfirmModalOpen()).toBe(true);

    // Confirmación en modal: registra la entrega como versión.
    component.onConfirmExamSubmit();
    await new Promise(resolve => setTimeout(resolve, 0));
    fixture.detectChanges();

    expect(projectSubmitCalls).toEqual(['ws-1']);
    expect(projectSubmitPayloads).toEqual([{ exercise_id: 'ex-101' }]);
    expect(component.projectSubmission()?.submission_id).toBe('sub-exam');
    expect(component.examConfirmModalOpen()).toBe(false);

    // El entorno sigue escribible: se puede volver a entregar.
    expect(component.canSubmitProject()).toBe(true);
    expect(fixture.nativeElement.textContent).not.toContain('congelado en solo lectura');

    component.requestSubmitProject();
    component.onConfirmExamSubmit();
    await new Promise(resolve => setTimeout(resolve, 0));
    expect(projectSubmitCalls).toEqual(['ws-1', 'ws-1']);
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

  it('testCode ejecuta en modo test con casos de ejemplo sin registrar entrega', async () => {
    evaluateCalls = [];
    submitCalls = [];

    await component.testCode();

    expect(evaluateCalls.length).toBe(1);
    expect((evaluateCalls[0] as { mode?: string }).mode).toBe('test');
    expect(submitCalls.length).toBe(0);
    expect(component.isTestRun()).toBe(true);
    expect(component.lastSubmissionId()).toBe('');
    expect(component.isExamSubmitted()).toBe(false);
  });

  it('dibuja la estructura de carpetas de la consigna como árbol y no como bloque plano', () => {
    component.exercise.set({
      ...component.exercise()!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE',
      description: [
        'Estructura de carpetas',
        '',
        '```text',
        'proyecto/',
        '├── backend/',
        '│   └── package.json',
        '└── frontend/',
        '    └── vite.config.ts',
        '```'
      ].join('\n')
    });
    fixture.detectChanges();

    const host = fixture.nativeElement as HTMLElement;
    expect(host.querySelectorAll('.tree .tree-row').length).toBe(5);
    expect(host.querySelector('.tree .tree-item.tree-dir .tree-name')?.textContent).toContain('proyecto');
    expect(host.querySelector('.tree .tree-item.tree-file .tree-name')?.textContent).toContain('package.json');
    expect(host.querySelector('.tree .tree-icon')).not.toBeNull();
    expect(host.textContent).not.toContain('├──');
    expect(host.querySelector('.markdown-body pre')).toBeNull();
  });

  it('muestra el aviso de examen sin acceso a internet cuando exam_network es none', () => {
    component.exercise.set({
      ...component.exercise()!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE',
      purpose: 'exam',
      exam_network: 'none'
    });
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent;
    expect(text).toContain('Este examen no tiene acceso a internet');
    expect(text).not.toContain('Este examen tiene acceso a internet por una excepción');
  });

  it('muestra el aviso de excepcion de red cuando exam_network es egress', () => {
    component.exercise.set({
      ...component.exercise()!,
      type: 'project',
      environment_type: 'IDE_PERSISTENTE',
      purpose: 'exam',
      exam_network: 'egress'
    });
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent;
    expect(text).toContain('Este examen tiene acceso a internet por una excepción declarada por el docente');
  });

  it('no muestra el selector de lenguaje mientras carga ni para entornos de desarrollo (IDE persistente)', () => {
    // 1. Mientras carga (exercise es null) no debe mostrar el selector en la topbar
    component.exercise.set(null);
    fixture.detectChanges();
    const host = fixture.nativeElement as HTMLElement;
    expect(host.querySelector('.lang-selector-group')).toBeNull();

    // 2. Si el ejercicio es IDE persistente tampoco debe mostrar el selector
    component.exercise.set({
      ...component.exercise()!,
      id: 'ex-ide',
      type: 'project',
      environment_type: 'IDE_PERSISTENTE',
      language: 'javascript'
    });
    fixture.detectChanges();
    expect(host.querySelector('.lang-selector-group')).toBeNull();
  });

  it('muestra el selector de lenguaje cuando el ejercicio es del Entorno de Evaluación', () => {
    component.exercise.set({
      ...component.exercise()!,
      id: 'ex-judge',
      type: 'algorithm',
      environment_type: 'JUEZ_EFIMERO',
      language: 'python'
    });
    fixture.detectChanges();
    const host = fixture.nativeElement as HTMLElement;
    expect(host.querySelector('.lang-selector-group')).not.toBeNull();
  });

  it('ofrece solo los lenguajes declarados por el entorno de la práctica', () => {
    component.exercise.set({
      ...component.exercise()!,
      id: 'ex-multi',
      type: 'algorithm',
      environment_type: 'JUEZ_EFIMERO',
      language: 'python',
      allowed_languages: ['cpp', 'python', 'java']
    });

    expect(component.languageOptions().map(option => option.value)).toEqual(['python', 'cpp', 'java']);
  });

  it('al cambiar de lenguaje carga su código de partida y no borra una solución escrita', () => {
    component.exercise.set({
      ...component.exercise()!,
      id: 'ex-multi',
      type: 'algorithm',
      environment_type: 'JUEZ_EFIMERO',
      language: 'python',
      allowed_languages: ['python', 'cpp'],
      boilerplates: {
        python: 'a, b = map(int, input().split())\n',
        cpp: '#include <iostream>\nint main() {}\n'
      }
    });
    component.sourceCode.set('a, b = map(int, input().split())\n');

    component.onLanguageChange('cpp');
    expect(component.selectedLanguage()).toBe('cpp');
    expect(component.sourceCode()).toContain('#include <iostream>');

    component.onCodeChange('int main() { return 0; }');
    component.onLanguageChange('python');
    expect(component.sourceCode()).toBe('int main() { return 0; }');
  });

  it('el botón de volver usa Location.back() cuando hay navegación previa en la app', () => {
    const location = TestBed.inject(Location);
    const locationSpy = vi.spyOn(location, 'back');
    const originalHistoryState = window.history.state;

    try {
      history.replaceState({ navigationId: 2 }, '');
      component.goBack();
      expect(locationSpy).toHaveBeenCalled();
    } finally {
      history.replaceState(originalHistoryState, '');
    }
  });

  it('el botón de volver navega a la materia cuando no hay historial interno', () => {
    const router = TestBed.inject(Router);
    const routerSpy = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    const originalHistoryState = window.history.state;

    try {
      history.replaceState({ navigationId: 1 }, '');
      component.goBack();
      expect(routerSpy).toHaveBeenCalledWith(['/student/courses', 'subj-101']);
    } finally {
      history.replaceState(originalHistoryState, '');
    }
  });

  it('renderiza el botón volver con etiqueta limpia y dispara goBack al hacer clic', () => {
    const goBackSpy = vi.spyOn(component, 'goBack');
    const backBtn = fixture.nativeElement.querySelector('.breadcrumb-back') as HTMLButtonElement | null;
    expect(backBtn).not.toBeNull();
    expect(backBtn?.textContent?.trim()).toContain('Volver');

    backBtn?.click();
    expect(goBackSpy).toHaveBeenCalled();
  });

  it('permite descartar el banner de modo examen y persiste en sessionStorage', () => {
    sessionStorage.clear();
    component.exercise.set({
      id: 'ex-exam-1',
      subject_id: 'subj-101',
      title: 'Examen de Algoritmos',
      description: 'Problema de examen',
      type: 'algorithm',
      environment_type: 'JUEZ_EFIMERO',
      difficulty: 'medium',
      purpose: 'exam',
      language: 'python',
      boilerplate: 'print("test")',
      time_limit_ms: 1000,
      memory_limit_mb: 256,
      constraints: [],
      public_test_cases: []
    });
    fixture.detectChanges();

    expect(component.isExamProject()).toBe(true);
    expect(component.examBannerDismissed()).toBe(false);

    const bannerBefore = fixture.nativeElement.querySelector('.exam-amber-banner');
    expect(bannerBefore).not.toBeNull();

    const closeBtn = fixture.nativeElement.querySelector('.banner-close-btn') as HTMLButtonElement | null;
    expect(closeBtn).not.toBeNull();
    closeBtn?.click();
    fixture.detectChanges();

    expect(component.examBannerDismissed()).toBe(true);
    expect(sessionStorage.getItem('solv_exam_banner_dismissed_ex-exam-1')).toBe('true');

    const bannerAfter = fixture.nativeElement.querySelector('.exam-amber-banner');
    expect(bannerAfter).toBeNull();
  });

  describe('modos de visualización y splitters interactivos', () => {
    beforeEach(() => {
      component.exercise.set({
        id: 'ex-101',
        subject_id: 'subj-101',
        title: 'Suma de Enteros',
        description: 'Lee dos números y muestra su suma.',
        type: 'algorithm',
        environment_type: 'JUEZ_EFIMERO',
        difficulty: 'easy',
        purpose: 'class',
        language: 'python',
        boilerplate: 'print("test")',
        time_limit_ms: 1000,
        memory_limit_mb: 128,
        constraints: [],
        public_test_cases: []
      });
      fixture.detectChanges();
    });

    it('inicia en modo dividido con proporción 42% por defecto', () => {
      expect(component.viewMode()).toBe('both');
      expect(component.splitRatio()).toBe(42);
      expect(component.consoleHeight()).toBe(248);

      const splitLayout = fixture.nativeElement.querySelector('.workbench-split-layout');
      expect(splitLayout.classList).toContain('layout-both');
      expect(fixture.nativeElement.querySelector('.workbench-left-pane')).not.toBeNull();
      expect(fixture.nativeElement.querySelector('.workbench-right-pane')).not.toBeNull();
      expect(fixture.nativeElement.querySelector('.layout-splitter')).not.toBeNull();
    });

    it('conmuta a modo consigna ocultando el editor y splitter', () => {
      component.setViewMode('statement');
      fixture.detectChanges();

      expect(component.viewMode()).toBe('statement');
      const splitLayout = fixture.nativeElement.querySelector('.workbench-split-layout');
      expect(splitLayout.classList).toContain('layout-statement');
      expect(fixture.nativeElement.querySelector('.workbench-left-pane')).not.toBeNull();
      expect(fixture.nativeElement.querySelector('.workbench-right-pane')).toBeNull();
      expect(fixture.nativeElement.querySelector('.layout-splitter')).toBeNull();
    });

    it('conmuta a modo editor (modo enfoque) ocultando la consigna y splitter', () => {
      component.setViewMode('editor');
      fixture.detectChanges();

      expect(component.viewMode()).toBe('editor');
      const splitLayout = fixture.nativeElement.querySelector('.workbench-split-layout');
      expect(splitLayout.classList).toContain('layout-editor');
      expect(fixture.nativeElement.querySelector('.workbench-left-pane')).toBeNull();
      expect(fixture.nativeElement.querySelector('.workbench-right-pane')).not.toBeNull();
      expect(fixture.nativeElement.querySelector('.layout-splitter')).toBeNull();
    });

    it('permite cambiar el modo desde los botones de la barra superior', () => {
      const modeButtons = fixture.nativeElement.querySelectorAll('.btn-mode-toggle');
      expect(modeButtons.length).toBe(3);

      // Botón 1: Consigna
      (modeButtons[1] as HTMLButtonElement).click();
      fixture.detectChanges();
      expect(component.viewMode()).toBe('statement');

      // Botón 2: Editor
      (modeButtons[2] as HTMLButtonElement).click();
      fixture.detectChanges();
      expect(component.viewMode()).toBe('editor');

      // Botón 0: Dividido
      (modeButtons[0] as HTMLButtonElement).click();
      fixture.detectChanges();
      expect(component.viewMode()).toBe('both');
    });

    it('inicia el arrastre del divisor de columnas con onSplitterMouseDown', () => {
      const event = new MouseEvent('mousedown', { cancelable: true });
      vi.spyOn(event, 'preventDefault');

      component.onSplitterMouseDown(event);
      expect(event.preventDefault).toHaveBeenCalled();
      expect(component.isDraggingSplitter()).toBe(true);

      // MouseUp finaliza el arrastre
      window.dispatchEvent(new MouseEvent('mouseup'));
      expect(component.isDraggingSplitter()).toBe(false);
    });

    it('inicia el arrastre del divisor de la consola con onConsoleSplitterMouseDown', () => {
      component.consoleOpen.set(false);
      const event = new MouseEvent('mousedown', { cancelable: true });
      vi.spyOn(event, 'preventDefault');

      component.onConsoleSplitterMouseDown(event);
      expect(event.preventDefault).toHaveBeenCalled();
      expect(component.consoleOpen()).toBe(true);
      expect(component.isDraggingConsoleSplitter()).toBe(true);

      window.dispatchEvent(new MouseEvent('mouseup'));
      expect(component.isDraggingConsoleSplitter()).toBe(false);
    });
  });
});
