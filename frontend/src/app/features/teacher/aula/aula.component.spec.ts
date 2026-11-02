import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { signal } from '@angular/core';
import { NEVER, of, throwError } from 'rxjs';

import { AulaComponent } from './aula.component';
import { ClassSessionService } from '../services/class-session.service';
import { ClassAdvanceService } from '../services/class-advance.service';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherLiveService } from '../services/teacher-live.service';
import { TeacherEnvironmentService } from '../services/teacher-environment.service';
import { ClassSession } from '../models/teacher.models';

function sessionWith(partial: Partial<ClassSession>): ClassSession {
  return {
    id: 'session-1',
    subject_id: 'subject-1',
    subject_code: 'CS101',
    subject_name: 'Algoritmos',
    exercise_id: 'exercise-1',
    exercise_title: 'Ordenamiento',
    purpose: 'class',
    status: 'open',
    opens_at: '2026-05-01T10:00:00Z',
    ends_at: '2026-05-01T11:30:00Z',
    closes_at: null,
    frozen_at: null,
    grace_until: null,
    enrolled: 2,
    present: 1,
    ...partial
  };
}

describe('AulaComponent — Abrir mi IDE', () => {
  let fixture: ComponentFixture<AulaComponent>;
  let component: AulaComponent;
  let aula: ReturnType<typeof signal<ClassSession | null>>;
  let openPracticeIDE: ReturnType<typeof vi.fn>;

  function ideButton(): HTMLButtonElement {
    const buttons = Array.from(
      fixture.nativeElement.querySelectorAll('.header-actions button')
    ) as HTMLButtonElement[];
    const button = buttons.find(item => item.textContent?.includes('Abrir mi IDE'));
    if (!button) {
      throw new Error('No se encontró el botón Abrir mi IDE');
    }
    return button;
  }

  beforeEach(async () => {
    aula = signal<ClassSession | null>(sessionWith({}));
    openPracticeIDE = vi.fn(() => NEVER);

    const sessionService = {
      session: aula,
      board: signal(null),
      isLoading: signal(false),
      isOpening: signal(false),
      error: signal<string | null>(null),
      lastClosedAt: signal(null),
      lastClosedSession: signal(null),
      history: signal([]),
      isLoadingHistory: signal(false),
      historyError: signal(null),
      isChangingFreeze: signal(false),
      isExtendingGrace: signal(false),
      lastFreezeReport: signal(null),
      isFreezingExam: signal(false),
      closedBoard: signal(null),
      isLoadingClosedBoard: signal(false),
      loadCurrent: () => of(aula()),
      loadLastClosed: () => of(null),
      loadBoard: () => of(null),
      loadHistory: () => of([]),
      clearError: () => undefined
    };

    const advanceService = {
      lastEmission: signal(null),
      sessionAdvances: signal([]),
      isLoadingAdvances: signal(false),
      clearNotice: () => undefined,
      clearError: () => undefined,
      loadSessionAdvances: () => of([])
    };

    const liveService = {
      liveSessions: signal([]),
      isPollingActive: signal(false),
      pollingIntervalMs: signal(5000),
      pausePolling: () => undefined,
      setPollingInterval: () => undefined,
      loadLiveSessions: () => of([])
    };

    const environmentService = {
      openPracticeIDE,
      describePracticeIDEError: () => 'No se pudo abrir el IDE de la práctica.'
    };

    await TestBed.configureTestingModule({
      imports: [AulaComponent],
      providers: [
        provideRouter([]),
        { provide: ClassSessionService, useValue: sessionService },
        { provide: ClassAdvanceService, useValue: advanceService },
        { provide: TeacherDashboardService, useValue: { courses: signal([]), getCourses: () => of([]) } },
        { provide: TeacherCourseService, useValue: { getExerciseBank: () => of([]) } },
        { provide: TeacherLiveService, useValue: liveService },
        { provide: TeacherEnvironmentService, useValue: environmentService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AulaComponent);
    component = fixture.componentInstance;
  });

  afterEach(() => {
    fixture.destroy();
    vi.restoreAllMocks();
  });

  it('abre el entorno de la práctica de la sesión en una pestaña nueva', () => {
    openPracticeIDE.mockReturnValue(of({ access_url: 'https://ide.example/session' }));
    const tab = { close: vi.fn(), opener: {} as unknown, location: { href: '' } };
    vi.spyOn(window, 'open').mockReturnValue(tab as unknown as Window);

    fixture.detectChanges();
    component.openIde();

    expect(window.open).toHaveBeenCalledWith('', '_blank');
    expect(openPracticeIDE).toHaveBeenCalledWith('subject-1', 'exercise-1');
    expect(tab.location.href).toBe('https://ide.example/session');
    expect(component.isOpeningIde()).toBe(false);
  });

  it('cierra la pestaña y muestra el error cuando el backend falla', () => {
    openPracticeIDE.mockReturnValue(throwError(() => ({ error: { message: 'cupo lleno' } })));
    const tab = { close: vi.fn(), opener: {} as unknown, location: { href: '' } };
    vi.spyOn(window, 'open').mockReturnValue(tab as unknown as Window);

    fixture.detectChanges();
    component.openIde();

    expect(tab.close).toHaveBeenCalled();
    expect(component.ideError()).toBe('No se pudo abrir el IDE de la práctica.');
    expect(component.isOpeningIde()).toBe(false);
  });

  it('deshabilita la acción y explica por qué cuando el aula no tiene práctica', () => {
    aula.set(sessionWith({ exercise_id: null, exercise_title: undefined }));

    fixture.detectChanges();

    const button = ideButton();
    expect(button.disabled).toBe(true);
    expect(button.title).toBe('Abrí el aula con una práctica para usar tu IDE');
  });
});
