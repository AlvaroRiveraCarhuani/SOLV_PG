import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AdvanceConfirmModalComponent } from './advance-confirm-modal.component';
import {
  AdvanceApplyResult,
  AdvanceFilePreview,
  ClassAdvanceNoticeService,
  PendingClassAdvance
} from '../../../services/class-advance-notice.service';

const ADVANCE_ID = '33333333-3333-3333-3333-333333333333';

const ADVANCE: PendingClassAdvance = {
  id: ADVANCE_ID,
  session_id: '55555555-5555-5555-5555-555555555555',
  exercise_id: '11111111-1111-1111-1111-111111111111',
  file_count: 2,
  created_at: '2026-10-31T15:30:00Z',
  subject_id: '44444444-4444-4444-4444-444444444444',
  session_status: 'open',
  grace_until: null,
  purpose: 'class'
};

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

describe('AdvanceConfirmModalComponent', () => {
  let fixture: ComponentFixture<AdvanceConfirmModalComponent>;
  let preview: ReturnType<typeof vi.fn>;
  let apply: ReturnType<typeof vi.fn>;

  function text(): string {
    return (fixture.nativeElement as HTMLElement).textContent ?? '';
  }

  function button(label: string): HTMLButtonElement | undefined {
    return Array.from((fixture.nativeElement as HTMLElement).querySelectorAll('button'))
      .find(btn => btn.textContent?.includes(label));
  }

  async function setup(): Promise<void> {
    preview = vi.fn().mockResolvedValue(PREVIEW);
    apply = vi.fn().mockResolvedValue(APPLIED);

    const mockNotices = {
      preview,
      apply,
      dismiss: vi.fn().mockResolvedValue(undefined),
      pending: () => []
    };

    TestBed.resetTestingModule();
    await TestBed.configureTestingModule({
      imports: [AdvanceConfirmModalComponent],
      providers: [{ provide: ClassAdvanceNoticeService, useValue: mockNotices }]
    }).compileComponents();

    fixture = TestBed.createComponent(AdvanceConfirmModalComponent);
    fixture.componentRef.setInput('advance', ADVANCE);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  }

  it('muestra qué archivos se reemplazan y cuáles se agregan antes de aplicar', async () => {
    await setup();

    expect(preview).toHaveBeenCalledWith(ADVANCE_ID);
    expect(text()).toContain('Se reemplazan');
    expect(text()).toContain('main.py');
    expect(text()).toContain('Se agregan');
    expect(text()).toContain('utils.py');
    expect(text()).toContain('copia dentro del laboratorio');
    expect(apply).not.toHaveBeenCalled();
  });

  it('aplica el avance y deja constancia del respaldo', async () => {
    await setup();

    button('Aplicar avance')?.dispatchEvent(new Event('click'));
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(apply).toHaveBeenCalledWith(ADVANCE_ID);
    expect(text()).toContain('El avance está en tu laboratorio');
    expect(text()).toContain(`.solv_backups/${ADVANCE_ID}`);
    expect(button('Cerrar')).toBeTruthy();
  });

  it('explica el motivo del rechazo y no deja aplicar', async () => {
    preview = vi.fn();
    apply = vi.fn();

    const mockNotices = {
      preview: vi.fn().mockRejectedValue({ error: { message: 'La ventana para aplicar el avance venció' } }),
      apply,
      dismiss: vi.fn(),
      pending: () => []
    };

    TestBed.resetTestingModule();
    await TestBed.configureTestingModule({
      imports: [AdvanceConfirmModalComponent],
      providers: [{ provide: ClassAdvanceNoticeService, useValue: mockNotices }]
    }).compileComponents();

    fixture = TestBed.createComponent(AdvanceConfirmModalComponent);
    fixture.componentRef.setInput('advance', ADVANCE);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(text()).toContain('La ventana para aplicar el avance venció');
    expect(button('Aplicar avance')?.disabled).toBe(true);
    expect(apply).not.toHaveBeenCalled();
  });
});
