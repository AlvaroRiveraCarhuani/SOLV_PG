import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';

import { ClassAdvanceModalComponent } from './class-advance-modal.component';
import { ClassAdvanceService } from '../../services/class-advance.service';
import { ClassSessionStudent } from '../../models/teacher.models';

const SESSION_ID = '55555555-5555-5555-5555-555555555555';
const EXERCISE_ID = '11111111-1111-1111-1111-111111111111';
const PRACTICE_LABEL = 'Recursión básica';

const participants: ClassSessionStudent[] = [
  {
    student_id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    name: 'Ana García',
    email: 'ana@uach.edu.bo',
    state: 'trabajando',
    present: true,
    joined_at: '2026-10-31T15:00:00Z',
    last_seen_at: '2026-10-31T15:10:00Z',
    environment_id: null,
    environment_status: null,
    submitted_at: null,
    verdict: null,
    review_status: null,
    attempts: 0
  },
  {
    student_id: 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
    name: 'Luis Pérez',
    email: 'luis@uach.edu.bo',
    state: 'sin_empezar',
    present: false,
    joined_at: null,
    last_seen_at: null,
    environment_id: null,
    environment_status: null,
    submitted_at: null,
    verdict: null,
    review_status: null,
    attempts: 0
  }
];

function advanceResponse(recipients: string[]) {
  return {
    data: {
      advance: {
        id: 'cccccccc-cccc-cccc-cccc-cccccccccccc',
        session_id: SESSION_ID,
        exercise_id: EXERCISE_ID,
        scope: 'class',
        snapshot_key: `class-advances/tenant/${SESSION_ID}.tar.gz`,
        snapshot_sha256: 'a'.repeat(64),
        snapshot_size_bytes: 1024,
        file_count: 2,
        created_at: '2026-10-31T15:20:00Z'
      },
      recipients
    },
    message: 'Avance emitido'
  };
}

describe('ClassAdvanceModalComponent', () => {
  let httpMock: HttpTestingController;

  function setup(practiceId: string | null = EXERCISE_ID): ClassAdvanceModalComponent {
    TestBed.configureTestingModule({
      imports: [ClassAdvanceModalComponent],
      providers: [ClassAdvanceService, provideHttpClient(), provideHttpClientTesting()]
    });
    httpMock = TestBed.inject(HttpTestingController);

    const fixture = TestBed.createComponent(ClassAdvanceModalComponent);
    fixture.componentRef.setInput('sessionId', SESSION_ID);
    fixture.componentRef.setInput('students', participants);
    fixture.componentRef.setInput('practiceId', practiceId);
    fixture.componentRef.setInput('practiceTitle', practiceId ? PRACTICE_LABEL : null);
    fixture.detectChanges();
    return fixture.componentInstance;
  }

  afterEach(() => httpMock.verify());

  it('emits to the whole class under the practice fixed by the aula', () => {
    const component = setup();
    const emitted: string[][] = [];
    component.emitted.subscribe(emission => emitted.push(emission.recipients));

    expect(component.practiceId()).toBe(EXERCISE_ID);
    expect(component.participants().length).toBe(1);

    component.emit();

    const req = httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/advances`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ scope: 'class' });
    req.flush(advanceResponse(['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa']));

    expect(emitted).toEqual([['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa']]);
  });

  it('requires a selection before emitting to chosen students', () => {
    const component = setup();
    component.setScope('selected');

    expect(component.canEmit()).toBe(false);
    component.emit();
    httpMock.expectNone(`/api/v1/teacher/sessions/${SESSION_ID}/advances`);

    component.toggleStudent('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa');
    expect(component.canEmit()).toBe(true);
    component.emit();

    const req = httpMock.expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/advances`);
    expect(req.request.body).toEqual({
      scope: 'selected',
      student_ids: ['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa']
    });
    req.flush(advanceResponse(['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa']));
  });

  it('refuses to emit when the aula has no practice', () => {
    const component = setup(null);

    expect(component.hasPractice()).toBe(false);
    expect(component.canEmit()).toBe(false);

    component.emit();
    httpMock.expectNone(`/api/v1/teacher/sessions/${SESSION_ID}/advances`);
  });

  it('offers only the students who were in the aula', () => {
    const component = setup();
    expect(component.participants().map(student => student.name)).toEqual(['Ana García']);
  });

  it('keeps the server reason when the emission is rejected', () => {
    const component = setup();
    component.emit();

    httpMock
      .expectOne(`/api/v1/teacher/sessions/${SESSION_ID}/advances`)
      .flush(
        { error: 'el aula necesita una práctica asociada', message: 'Asigna una práctica al aula' },
        { status: 409, statusText: 'Conflict' }
      );

    expect(component.serviceError()).toContain('práctica');
  });
});
