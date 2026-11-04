import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';

import { StudentAulaComponent } from './student-aula.component';

const AULA = {
  id: '55555555-5555-5555-5555-555555555555',
  subject_id: '44444444-4444-4444-4444-444444444444',
  subject_code: 'SIS-211',
  subject_name: 'Estructuras de Datos',
  exercise_id: '11111111-1111-1111-1111-111111111111',
  exercise_title: 'Recursión básica',
  purpose: 'class',
  status: 'open',
  opens_at: '2026-11-04T14:00:00Z',
  ends_at: '2026-11-04T15:30:00Z',
  closes_at: null,
  frozen_at: null,
  grace_until: null,
  enrolled: 28,
  present: 21
};

const AVANCE = {
  id: 'cccccccc-cccc-cccc-cccc-cccccccccccc',
  session_id: AULA.id,
  exercise_id: AULA.exercise_id,
  file_count: 3,
  created_at: '2026-11-04T14:40:00Z',
  subject_id: AULA.subject_id,
  session_status: 'open',
  grace_until: null,
  purpose: 'class'
};

/** Deja correr las promesas que resuelve el flush antes de mirar el DOM. */
function settle(): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, 0));
}

describe('StudentAulaComponent', () => {
  let httpMock: HttpTestingController;

  beforeEach(() => TestBed.resetTestingModule());

  async function setup(aulas: unknown[], advances: unknown[]): Promise<ComponentFixture<StudentAulaComponent>> {
    TestBed.configureTestingModule({
      imports: [StudentAulaComponent],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])]
    });
    httpMock = TestBed.inject(HttpTestingController);

    const fixture = TestBed.createComponent(StudentAulaComponent);
    fixture.detectChanges();

    httpMock.expectOne('/api/v1/student/sessions/active').flush({ data: aulas });
    await settle();
    fixture.detectChanges();

    httpMock.expectOne('/api/v1/class-advances/pending').flush({ data: advances });
    await settle();
    fixture.detectChanges();

    return fixture;
  }

  afterEach(() => {
    httpMock.verify();
  });

  it('sin aula abierta ofrece la lista de prácticas', async () => {
    const fixture = await setup([], []);

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('No tienes clase abierta en este momento');
    expect(fixture.nativeElement.querySelector('.aula-card')).toBeNull();

    fixture.destroy();
  });

  it('muestra la materia, la práctica y el horario del aula', async () => {
    const fixture = await setup([AULA], []);

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Estructuras de Datos');
    expect(text).toContain('Recursión básica');
    expect(text).toContain('Abierta');
    expect(text).toContain('21');
    expect(text).toContain('28');

    const link = fixture.nativeElement.querySelector('.aula-card a.btn-primary') as HTMLAnchorElement;
    expect(link.getAttribute('href')).toBe(`/student/exercises/${AULA.exercise_id}`);

    fixture.destroy();
  });

  it('junta el avance pendiente con el aula y lo abre para revisar', async () => {
    const fixture = await setup([AULA], [AVANCE]);

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Código nuevo del docente');

    const button = Array.from(
      fixture.nativeElement.querySelectorAll('.advance-item button') as NodeListOf<HTMLButtonElement>
    )[0];
    button.click();
    fixture.detectChanges();

    httpMock.expectOne(`/api/v1/class-advances/${AVANCE.id}/files`).flush({
      data: {
        advance_id: AVANCE.id,
        exercise_id: AVANCE.exercise_id,
        file_count: AVANCE.file_count,
        added: ['main.py'],
        overwritten: []
      }
    });
    await settle();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('advance-confirm-modal')).not.toBeNull();

    fixture.destroy();
  });

  it('muestra cada aula abierta y avisa cuando una no tiene práctica', async () => {
    const otra = {
      ...AULA,
      id: 'otra-sesion',
      subject_code: 'MAT-101',
      subject_name: 'Álgebra',
      exercise_id: null,
      exercise_title: undefined
    };
    const fixture = await setup([AULA, otra], []);

    const cards = fixture.nativeElement.querySelectorAll('.aula-card') as NodeListOf<HTMLElement>;
    expect(cards.length).toBe(2);

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Álgebra');
    expect(text).toContain('Sin práctica asociada');

    fixture.destroy();
  });

  it('ignora los avances de otra aula', async () => {
    const fixture = await setup([AULA], [{ ...AVANCE, session_id: 'otra-sesion' }]);

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Sin avances pendientes');

    fixture.destroy();
  });
});
