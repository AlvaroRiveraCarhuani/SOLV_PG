import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';

import { ClassSessionService } from './class-session.service';
import { ClassHistoryEvent, ClassSession } from '../models/teacher.models';

const SESSION_ID = '55555555-5555-5555-5555-555555555555';

function session(overrides: Partial<ClassSession> = {}): ClassSession {
  return {
    id: SESSION_ID,
    subject_id: '22222222-2222-2222-2222-222222222222',
    subject_code: 'SIS-301',
    subject_name: 'Estructuras de Datos',
    exercise_id: null,
    status: 'open',
    opens_at: '2026-10-31T15:00:00Z',
    ends_at: '2026-10-31T16:30:00Z',
    closes_at: null,
    frozen_at: null,
    grace_until: null,
    enrolled: 24,
    present: 19,
    ...overrides
  };
}

describe('ClassSessionService', () => {
  let service: ClassSessionService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ClassSessionService, provideHttpClient(), provideHttpClientTesting()]
    });
    service = TestBed.inject(ClassSessionService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpMock.verify());

  it('freezes the class and keeps the returned session', () => {
    service.setFrozen(SESSION_ID, true).subscribe();

    const req = httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/freeze`);
    expect(req.request.method).toBe('PATCH');
    req.flush({ data: session({ frozen_at: '2026-10-31T15:30:00Z' }) });

    expect(service.session()?.frozen_at).toBe('2026-10-31T15:30:00Z');
    expect(service.isChangingFreeze()).toBe(false);
  });

  it('unfreezes through the same action and clears frozen_at', () => {
    service.session.set(session({ frozen_at: '2026-10-31T15:30:00Z' }));

    service.setFrozen(SESSION_ID, false).subscribe();

    httpMock
      .expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/unfreeze`)
      .flush({ data: session({ frozen_at: null }) });

    expect(service.session()?.frozen_at).toBeNull();
  });

  it('keeps the closed session and the automatic window the server set', () => {
    service.closeAula(SESSION_ID).subscribe();

    httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/close`).flush({
      data: session({ status: 'closed', closes_at: '2026-10-31T16:00:00Z', grace_until: '2026-10-31T16:30:00Z' })
    });

    expect(service.session()).toBeNull();
    expect(service.lastClosedSession()?.id).toBe(SESSION_ID);
    expect(service.lastClosedSession()?.grace_until).toBe('2026-10-31T16:30:00Z');
  });

  it('keeps the freeze report that the close of an exam returns', () => {
    service.closeAula(SESSION_ID).subscribe();

    httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/close`).flush({
      data: {
        ...session({ status: 'closed', purpose: 'exam', closes_at: '2026-10-31T16:00:00Z' }),
        freeze_report: { frozen: 28, skipped: 5, failed: 2 }
      }
    });

    expect(service.lastFreezeReport()).toEqual({ frozen: 28, skipped: 5, failed: 2 });
    expect(service.lastClosedSession()?.purpose).toBe('exam');
  });

  it('repeats the freeze of a closed exam and refreshes the report', () => {
    service.freezeExam(SESSION_ID).subscribe();

    const req = httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/exam-freeze`);
    expect(req.request.method).toBe('POST');
    req.flush({ data: { frozen: 2, skipped: 30, failed: 0 } });

    expect(service.lastFreezeReport()).toEqual({ frozen: 2, skipped: 30, failed: 0 });
    expect(service.isFreezingExam()).toBe(false);
  });

  it('reopens one student with a reason and leaves the page report untouched', () => {
    service.lastFreezeReport.set({ frozen: 28, skipped: 5, failed: 2 });

    let report: unknown;
    service.unfreezeStudent(SESSION_ID, 'student-1', 'Se le cortó la luz').subscribe(value => (report = value));

    const req = httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/students/student-1/unfreeze`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ reason: 'Se le cortó la luz' });
    req.flush({ data: { frozen: 1, skipped: 0, failed: 0 } });

    expect(report).toEqual({ frozen: 1, skipped: 0, failed: 0 });
    expect(service.lastFreezeReport()).toEqual({ frozen: 28, skipped: 5, failed: 2 });
  });

  it('loads the board of a closed exam without reopening the live aula', () => {
    const board = {
      session: session({ status: 'closed', purpose: 'exam' }),
      summary: {
        enrolled: 28,
        present: 0,
        not_started: 0,
        working: 0,
        submitted: 28,
        stuck: 0,
        absent: 0,
        without_submission: 0
      },
      students: [
        {
          student_id: 'student-1',
          name: 'Ana Pérez',
          email: 'ana@solv.test',
          state: 'entregado',
          present: false,
          joined_at: null,
          last_seen_at: null,
          workspace_id: 'workspace-1',
          workspace_status: 'frozen',
          submitted_at: null,
          verdict: 'AC',
          review_status: null,
          attempts: 1
        }
      ]
    };

    service.loadClosedBoard(SESSION_ID).subscribe();
    httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/board`).flush({ data: board });

    expect(service.closedBoard()?.students[0].workspace_status).toBe('frozen');
    expect(service.session()).toBeNull();
    expect(service.isLoadingClosedBoard()).toBe(false);
  });

  it('adds free minutes to the current grace window', () => {
    service.lastClosedSession.set(session({ status: 'closed', closes_at: '2026-10-31T16:00:00Z' }));

    service.addGrace(SESSION_ID, 20).subscribe();

    const req = httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/grace`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ add_minutes: 20 });
    req.flush({
      data: session({ status: 'closed', grace_until: '2026-10-31T17:00:00Z' })
    });

    expect(service.lastClosedSession()?.grace_until).toBe('2026-10-31T17:00:00Z');
    expect(service.isExtendingGrace()).toBe(false);
  });

  it('sets a deadline for a take-home task', () => {
    service.lastClosedSession.set(session({ status: 'closed', closes_at: '2026-10-31T16:00:00Z' }));
    const until = '2026-11-02T23:00:00.000Z';

    service.setGraceDeadline(SESSION_ID, until).subscribe();

    const req = httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/grace`);
    expect(req.request.body).toEqual({ until });
    req.flush({ data: session({ status: 'closed', grace_until: until }) });

    expect(service.lastClosedSession()?.grace_until).toBe(until);
  });

  it('loads the class history and keeps the last list when the read fails', () => {
    const event: ClassHistoryEvent = {
      id: 'evento-1',
      action: 'CLASS_FROZEN',
      actor_id: '22222222-2222-2222-2222-222222222222',
      actor_name: 'Docente',
      actor_email: 'docente@uab.edu.bo',
      actor_role: 'teacher',
      created_at: '2026-10-31T15:10:00Z'
    };

    service.loadHistory(SESSION_ID).subscribe();
    httpMock
      .expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/history`)
      .flush({ data: [event] });

    expect(service.history().length).toBe(1);
    expect(service.history()[0].action).toBe('CLASS_FROZEN');
    expect(service.historyError()).toBeNull();

    service.loadHistory(SESSION_ID).subscribe();
    httpMock
      .expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/history`)
      .flush({ error: 'No se pudo leer el historial' }, { status: 500, statusText: 'Server Error' });

    expect(service.history().length).toBe(1);
    expect(service.historyError()).toBe('No se pudo leer el historial');
  });

  it('closes the window in the moment', () => {
    service.lastClosedSession.set(session({ status: 'closed', closes_at: '2026-10-31T16:00:00Z' }));

    service.closeGrace(SESSION_ID).subscribe();

    const req = httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/grace`);
    expect(req.request.body).toEqual({ close: true });
    req.flush({ data: session({ status: 'closed', grace_until: '2026-10-31T16:10:00Z' }) });

    expect(service.isExtendingGrace()).toBe(false);
  });
});
