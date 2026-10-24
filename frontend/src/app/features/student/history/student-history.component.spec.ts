import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import { StudentHistoryComponent } from './student-history.component';
import { StudentService, StudentSubmissionHistoryItem } from '@core/services/student.service';
import { ComboboxComponent } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';

const SUBMISSIONS: StudentSubmissionHistoryItem[] = [
  {
    id: 'sub-1',
    exercise_id: 'e1',
    exercise_title: 'Recorrido de listas',
    subject_id: 's1',
    subject_code: 'SIS-211',
    subject_name: 'Estructuras de Datos',
    code: 'def recorrer(lista):\n    return lista',
    verdict: 'AC',
    score: 100,
    execution_time_ms: 12,
    memory_used_mb: 32,
    submitted_at: '2026-10-20T14:05:00'
  },
  {
    id: 'sub-2',
    exercise_id: 'e2',
    exercise_title: 'Examen de grafos',
    subject_id: 's1',
    subject_code: 'SIS-211',
    subject_name: 'Estructuras de Datos',
    code: 'bfs(grafo)',
    verdict: 'WA',
    score: 40,
    execution_time_ms: 300,
    memory_used_mb: 64,
    submitted_at: '2026-10-21T09:30:00'
  },
  {
    id: 'sub-3',
    exercise_id: 'e3',
    exercise_title: 'Práctica de árboles',
    subject_id: 's2',
    subject_code: 'SIS-312',
    subject_name: 'Sistemas Operativos',
    code: 'class Nodo:\n    pass',
    verdict: 'AST_VIOLATION',
    score: null,
    execution_time_ms: 5,
    memory_used_mb: 16,
    submitted_at: '2026-10-22T18:00:00'
  },
  {
    id: 'sub-4',
    exercise_id: 'e4',
    exercise_title: 'Examen de ordenamiento',
    subject_id: 's2',
    subject_code: 'SIS-312',
    subject_name: 'Sistemas Operativos',
    code: 'ordenar(lista)',
    verdict: 'RE',
    score: 0,
    execution_time_ms: 80,
    memory_used_mb: 24,
    manual_override: true,
    override_reason: 'Revisión manual: la solución cumple el enunciado.',
    submitted_at: '2026-10-23T11:15:00'
  }
];

describe('StudentHistoryComponent', () => {
  let component: StudentHistoryComponent;
  let fixture: ComponentFixture<StudentHistoryComponent>;

  function rows(): HTMLElement[] {
    return Array.from(
      (fixture.nativeElement as HTMLElement).querySelectorAll<HTMLElement>('.history-table tbody tr')
    );
  }

  function rowFor(title: string): HTMLElement | undefined {
    return rows().find((row) => row.textContent?.includes(title));
  }

  function comboboxes(): ComboboxComponent[] {
    return fixture.debugElement
      .queryAll(By.directive(ComboboxComponent))
      .map((debugElement) => debugElement.componentInstance as ComboboxComponent);
  }

  function searchBar(): SearchBarComponent {
    return fixture.debugElement.query(By.directive(SearchBarComponent)).componentInstance as SearchBarComponent;
  }

  function modal(): ModalShellComponent | null {
    const debugElement = fixture.debugElement.query(By.directive(ModalShellComponent));
    return debugElement ? (debugElement.componentInstance as ModalShellComponent) : null;
  }

  function modalText(): string {
    const element = (fixture.nativeElement as HTMLElement).querySelector('modal-shell');
    return element?.textContent ?? '';
  }

  async function setup(): Promise<void> {
    TestBed.resetTestingModule();

    const mockStudentService = {
      submissionsHistory: () => SUBMISSIONS,
      loadSubmissionsHistory: () => Promise.resolve(SUBMISSIONS)
    };

    await TestBed.configureTestingModule({
      imports: [StudentHistoryComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentHistoryComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  }

  it('renderiza la tabla con todas las entregas y sus datos', async () => {
    await setup();

    expect(component.isLoading()).toBe(false);
    expect(rows().length).toBe(4);

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Recorrido de listas');
    expect(text).toContain('SIS-211');
    expect(text).toContain('Estructuras de Datos');
    expect(text).toContain('100 pts');
    expect(text).toContain('12 ms');
    expect(text).toContain('Revisado por docente');

    // Nota vacía y veredicto en lenguaje humano.
    expect(rowFor('Práctica de árboles')?.textContent).toContain('—');
    expect(rowFor('Recorrido de listas')?.querySelector('verdict-badge')?.textContent).toContain(
      'Solución correcta'
    );
    expect(rowFor('Práctica de árboles')?.querySelector('verdict-badge')?.textContent).toContain(
      'Restricción de código incumplida'
    );

    const time = rowFor('Recorrido de listas')?.querySelector('time');
    expect(time?.getAttribute('datetime')).toBe('2026-10-20T14:05:00');
    expect(time?.textContent?.trim()).toBe('20/10 14:05:00');

    const exerciseLink = rowFor('Recorrido de listas')?.querySelector<HTMLAnchorElement>('a.action-link');
    expect(exerciseLink?.getAttribute('href')).toBe('/student/exercises/e1');
  });

  it('filtra por materia desde el combobox', async () => {
    await setup();

    expect(rows().length).toBe(4);

    const [subjectCombo] = comboboxes();
    subjectCombo.valueChange.emit('s2');
    fixture.detectChanges();

    expect(component.subjectFilter()).toBe('s2');
    expect(rows().length).toBe(2);
    expect(rowFor('Práctica de árboles')).toBeTruthy();
    expect(rowFor('Examen de ordenamiento')).toBeTruthy();

    subjectCombo.valueChange.emit('all');
    fixture.detectChanges();
    expect(rows().length).toBe(4);
  });

  it('filtra por resultado y AST_BLOCKED incluye AST_VIOLATION', async () => {
    await setup();

    const [, verdictCombo] = comboboxes();

    verdictCombo.valueChange.emit('AC');
    fixture.detectChanges();
    expect(rows().length).toBe(1);
    expect(rows()[0].textContent).toContain('Recorrido de listas');

    verdictCombo.valueChange.emit('AST_BLOCKED');
    fixture.detectChanges();
    expect(component.verdictFilter()).toBe('AST_BLOCKED');
    expect(rows().length).toBe(1);
    expect(rows()[0].textContent).toContain('Práctica de árboles');

    verdictCombo.valueChange.emit('all');
    fixture.detectChanges();
    expect(rows().length).toBe(4);
  });

  it('filtra por título de ejercicio desde el buscador', async () => {
    await setup();

    searchBar().queryChange.emit('grafos');
    fixture.detectChanges();
    expect(rows().length).toBe(1);
    expect(rows()[0].textContent).toContain('Examen de grafos');

    searchBar().cleared.emit();
    fixture.detectChanges();
    expect(rows().length).toBe(4);
  });

  it('abre el modal con el código enviado y lo cierra', async () => {
    await setup();

    expect(modal()).toBeNull();

    const openButton = Array.from(
      rowFor('Práctica de árboles')?.querySelectorAll<HTMLButtonElement>('button') ?? []
    ).find((button) => button.textContent?.includes('Ver código enviado'));
    expect(openButton).toBeTruthy();

    openButton?.dispatchEvent(new Event('click'));
    fixture.detectChanges();

    expect(component.selectedSubmission()?.id).toBe('sub-3');
    expect(modal()).not.toBeNull();

    expect(modalText()).toContain('class Nodo:');
    expect(modalText()).toContain('Práctica de árboles');
    expect(modal()?.title()).toBe('Práctica de árboles');

    modal()?.close.emit();
    fixture.detectChanges();

    expect(component.selectedSubmission()).toBeNull();
    expect(modal()).toBeNull();
  });

  it('muestra la observación del docente cuando hubo revisión manual', async () => {
    await setup();

    const openButton = Array.from(
      rowFor('Examen de ordenamiento')?.querySelectorAll<HTMLButtonElement>('button') ?? []
    ).find((button) => button.textContent?.includes('Ver código enviado'));
    openButton?.dispatchEvent(new Event('click'));
    fixture.detectChanges();

    expect(modalText()).toContain('Observación del docente');
    expect(modalText()).toContain('Revisión manual: la solución cumple el enunciado.');
  });
});