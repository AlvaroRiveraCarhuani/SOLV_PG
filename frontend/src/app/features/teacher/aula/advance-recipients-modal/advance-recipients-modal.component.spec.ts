import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AdvanceRecipientsModalComponent } from './advance-recipients-modal.component';
import { SessionAdvance } from '../../models/teacher.models';

const ADVANCE: SessionAdvance = {
  id: '99999999-9999-9999-9999-999999999999',
  exercise_id: '11111111-1111-1111-1111-111111111111',
  exercise_title: 'Recorridos de árboles',
  scope: 'class',
  file_count: 2,
  created_at: '2026-10-31T15:30:00Z',
  counts: { total: 4, applied: 1, dismissed: 1, pending: 1, failed: 0, expired: 1 },
  recipients: [
    { student_id: 's1', name: 'Ana Pérez', email: 'ana@solv.test', status: 'applied', decided_at: '2026-10-31T15:32:00Z' },
    { student_id: 's2', name: 'Beto Ruiz', email: 'beto@solv.test', status: 'dismissed', decided_at: '2026-10-31T15:33:00Z' },
    { student_id: 's3', name: 'Caro Díaz', email: 'caro@solv.test', status: 'pending', decided_at: null },
    { student_id: 's4', name: 'Dani Soto', email: 'dani@solv.test', status: 'expired', decided_at: null }
  ]
};

describe('AdvanceRecipientsModalComponent', () => {
  let fixture: ComponentFixture<AdvanceRecipientsModalComponent>;
  let closed: () => void;

  function text(): string {
    return (fixture.nativeElement as HTMLElement).textContent ?? '';
  }

  async function setup(): Promise<void> {
    TestBed.resetTestingModule();
    await TestBed.configureTestingModule({
      imports: [AdvanceRecipientsModalComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(AdvanceRecipientsModalComponent);
    closed = vi.fn(() => undefined);
    fixture.componentInstance.closed.subscribe(closed);
    fixture.componentRef.setInput('advance', ADVANCE);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  }

  it('resume el estado de la emisión y lista a cada alumno', async () => {
    await setup();

    expect(text()).toContain('Recorridos de árboles');
    expect(text()).toContain('1 aplicaron');
    expect(text()).toContain('1 descartaron');
    expect(text()).toContain('1 pendientes');
    expect(text()).toContain('1 vencieron');

    expect(text()).toContain('Ana Pérez');
    expect(text()).toContain('Aplicó');
    expect(text()).toContain('Beto Ruiz');
    expect(text()).toContain('Descartó');
    expect(text()).toContain('Caro Díaz');
    expect(text()).toContain('Pendiente');
    expect(text()).toContain('Dani Soto');
    expect(text()).toContain('Venció');
  });

  it('no muestra los estados que no ocurrieron', async () => {
    await setup();

    expect(text()).not.toContain('fallaron');
    expect(text()).not.toContain('Falló');
  });

  it('cierra el modal desde su pie', async () => {
    await setup();

    const closeButton = Array.from((fixture.nativeElement as HTMLElement).querySelectorAll('button'))
      .find(btn => btn.textContent?.includes('Cerrar'));
    closeButton?.dispatchEvent(new Event('click'));
    fixture.detectChanges();

    expect(closed).toHaveBeenCalled();
  });
});
