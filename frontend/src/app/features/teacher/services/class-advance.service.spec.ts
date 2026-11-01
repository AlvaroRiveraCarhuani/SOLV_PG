import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';

import { ClassAdvanceService } from './class-advance.service';
import { SessionAdvance } from '../models/teacher.models';

const SESSION_ID = '55555555-5555-5555-5555-555555555555';

function advance(overrides: Partial<SessionAdvance> = {}): SessionAdvance {
  return {
    id: '99999999-9999-9999-9999-999999999999',
    exercise_id: '11111111-1111-1111-1111-111111111111',
    exercise_title: 'Recorridos de árboles',
    scope: 'class',
    file_count: 2,
    created_at: '2026-10-31T15:30:00Z',
    counts: { total: 2, applied: 1, dismissed: 0, pending: 1, failed: 0, expired: 0 },
    recipients: [
      { student_id: 's1', name: 'Ana', email: 'ana@solv.test', status: 'applied', decided_at: '2026-10-31T15:32:00Z' },
      { student_id: 's2', name: 'Beto', email: 'beto@solv.test', status: 'pending', decided_at: null }
    ],
    ...overrides
  };
}

describe('ClassAdvanceService', () => {
  let service: ClassAdvanceService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ClassAdvanceService, provideHttpClient(), provideHttpClientTesting()]
    });
    service = TestBed.inject(ClassAdvanceService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpMock.verify());

  it('lee el estado de cada destinatario de las emisiones del aula', () => {
    let loaded: SessionAdvance[] = [];
    service.loadSessionAdvances(SESSION_ID).subscribe(list => (loaded = list));

    const req = httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/advances`);
    expect(req.request.method).toBe('GET');
    req.flush({ data: [advance()] });

    expect(loaded).toHaveLength(1);
    expect(service.sessionAdvances()[0].counts.applied).toBe(1);
    expect(service.sessionAdvances()[0].recipients[1].status).toBe('pending');
    expect(service.isLoadingAdvances()).toBe(false);
  });

  it('conserva la última lectura cuando el pedido falla', () => {
    service.sessionAdvances.set([advance()]);

    let loaded: SessionAdvance[] = [];
    service.loadSessionAdvances(SESSION_ID).subscribe(list => (loaded = list));

    httpMock
      .expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/advances`)
      .flush({ error: 'boom', message: 'No se pudo leer el estado de los avances' }, { status: 500, statusText: 'Server Error' });

    expect(loaded).toHaveLength(1);
    expect(service.sessionAdvances()).toHaveLength(1);
    expect(service.advancesError()).toBeTruthy();
    expect(service.isLoadingAdvances()).toBe(false);
  });
});
