import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';

import { UnfreezeStudentModalComponent } from './unfreeze-student-modal.component';
import { ClassSessionService } from '../../services/class-session.service';
import { ClassSessionStudent, FreezeReport } from '../../models/teacher.models';

const SESSION_ID = '55555555-5555-5555-5555-555555555555';

const STUDENT: ClassSessionStudent = {
  student_id: '77777777-7777-7777-7777-777777777777',
  name: 'Ana Pérez',
  email: 'ana@solv.test',
  state: 'entregado',
  present: true,
  joined_at: '2026-10-31T15:00:00Z',
  last_seen_at: '2026-10-31T16:02:00Z',
  workspace_id: '99999999-9999-9999-9999-999999999999',
  workspace_status: 'frozen',
  submitted_at: '2026-10-31T15:58:00Z',
  verdict: 'AC',
  review_status: null,
  attempts: 1
};

describe('UnfreezeStudentModalComponent', () => {
  let fixture: ComponentFixture<UnfreezeStudentModalComponent>;
  let unfreezeStudent: ReturnType<typeof vi.fn>;
  let unfrozen: (report: FreezeReport) => void;
  let closed: () => void;

  function text(): string {
    return (fixture.nativeElement as HTMLElement).textContent ?? '';
  }

  function submitButton(): HTMLButtonElement | undefined {
    return Array.from((fixture.nativeElement as HTMLElement).querySelectorAll('button'))
      .find(btn => btn.textContent?.includes('Reabrir entorno'));
  }

  async function setup(result = of<FreezeReport>({ frozen: 1, skipped: 0, failed: 0 })): Promise<void> {
    TestBed.resetTestingModule();
    unfreezeStudent = vi.fn().mockReturnValue(result);

    await TestBed.configureTestingModule({
      imports: [UnfreezeStudentModalComponent],
      providers: [{ provide: ClassSessionService, useValue: { unfreezeStudent } }]
    }).compileComponents();

    fixture = TestBed.createComponent(UnfreezeStudentModalComponent);
    unfrozen = vi.fn(() => undefined);
    closed = vi.fn(() => undefined);
    fixture.componentInstance.unfrozen.subscribe(unfrozen);
    fixture.componentInstance.closed.subscribe(closed);
    fixture.componentRef.setInput('sessionId', SESSION_ID);
    fixture.componentRef.setInput('student', STUDENT);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  }

  it('nombra al alumno y exige un motivo antes de reabrir', async () => {
    await setup();

    expect(text()).toContain('Ana Pérez');
    expect(text()).toContain('solo lectura');
    expect(text()).toContain('Motivo');
    expect(submitButton()?.disabled).toBe(true);
  });

  it('reabre el entorno con el motivo escrito y avisa al aula', async () => {
    await setup();

    fixture.componentInstance.onReasonChange('Se le cortó la luz a mitad del examen');
    fixture.detectChanges();
    expect(submitButton()?.disabled).toBe(false);

    submitButton()?.dispatchEvent(new Event('click'));
    await fixture.whenStable();

    expect(unfreezeStudent).toHaveBeenCalledWith(
      SESSION_ID,
      STUDENT.student_id,
      'Se le cortó la luz a mitad del examen'
    );
    expect(unfrozen).toHaveBeenCalledWith({ frozen: 1, skipped: 0, failed: 0 });
  });

  it('muestra el error del servidor y deja el modal abierto', async () => {
    await setup(
      throwError(() => ({ error: { error: 'El alumno no participó de ese examen' } }))
    );

    fixture.componentInstance.onReasonChange('Motivo válido');
    fixture.detectChanges();
    submitButton()?.dispatchEvent(new Event('click'));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(text()).toContain('El alumno no participó de ese examen');
    expect(unfrozen).not.toHaveBeenCalled();
  });
});
