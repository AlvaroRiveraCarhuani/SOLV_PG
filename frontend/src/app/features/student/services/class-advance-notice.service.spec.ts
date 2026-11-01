import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';

import {
  AdvanceApplyResult,
  AdvanceFilePreview,
  ClassAdvanceNoticeService,
  PendingClassAdvance
} from './class-advance-notice.service';

const ADVANCE_ID = '33333333-3333-3333-3333-333333333333';

function advance(overrides: Partial<PendingClassAdvance> = {}): PendingClassAdvance {
  return {
    id: ADVANCE_ID,
    session_id: '55555555-5555-5555-5555-555555555555',
    exercise_id: '11111111-1111-1111-1111-111111111111',
    file_count: 2,
    created_at: '2026-10-31T15:30:00Z',
    subject_id: '44444444-4444-4444-4444-444444444444',
    session_status: 'open',
    grace_until: null,
    purpose: 'class',
    ...overrides
  };
}

const PREVIEW: AdvanceFilePreview = {
  advance_id: ADVANCE_ID,
  exercise_id: '11111111-1111-1111-1111-111111111111',
  file_count: 2,
  added: ['utils.py'],
  overwritten: ['main.py']
};

const APPLIED: AdvanceApplyResult = {
  advance_id: ADVANCE_ID,
  files_written: 2,
  overwritten: ['main.py'],
  added: ['utils.py'],
  backup_path: `.solv_backups/${ADVANCE_ID}`
};

describe('ClassAdvanceNoticeService', () => {
  let service: ClassAdvanceNoticeService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ClassAdvanceNoticeService, provideHttpClient(), provideHttpClientTesting()]
    });
    service = TestBed.inject(ClassAdvanceNoticeService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    service.stop();
    httpMock.verify();
  });

  it('carga los avisos pendientes del alumno', async () => {
    const loading = service.refresh();

    const req = httpMock.expectOne('/api/v1/class-advances/pending');
    expect(req.request.method).toBe('GET');
    req.flush({ data: [advance()] });
    await loading;

    expect(service.pending()).toEqual([advance()]);
  });

  it('lee qué archivos se agregan y cuáles se reemplazan antes de aplicar', async () => {
    const loading = service.preview(ADVANCE_ID);

    const req = httpMock.expectOne(`/api/v1/class-advances/${ADVANCE_ID}/files`);
    expect(req.request.method).toBe('GET');
    req.flush({ data: PREVIEW });

    await expect(loading).resolves.toEqual(PREVIEW);
  });

  it('aplica el avance y quita el aviso de la lista', async () => {
    service.pending.set([advance()]);

    const applying = service.apply(ADVANCE_ID);

    const req = httpMock.expectOne(`/api/v1/class-advances/${ADVANCE_ID}/apply`);
    expect(req.request.method).toBe('POST');
    req.flush({ data: APPLIED });

    await expect(applying).resolves.toEqual(APPLIED);
    expect(service.pending()).toEqual([]);
  });

  it('descarta el aviso sin tocar el entorno', async () => {
    service.pending.set([advance()]);

    const dismissing = service.dismiss(ADVANCE_ID);

    const req = httpMock.expectOne(`/api/v1/class-advances/${ADVANCE_ID}/dismiss`);
    expect(req.request.method).toBe('POST');
    req.flush({ data: { ok: true } });
    await dismissing;

    expect(service.pending()).toEqual([]);
  });

  it('conserva el motivo que redacta el servidor', () => {
    const err = { error: { message: 'La ventana para aplicar el avance venció' } };
    expect(ClassAdvanceNoticeService.reason(err, 'otro texto')).toBe('La ventana para aplicar el avance venció');
    expect(ClassAdvanceNoticeService.reason({}, 'otro texto')).toBe('otro texto');
  });
});
