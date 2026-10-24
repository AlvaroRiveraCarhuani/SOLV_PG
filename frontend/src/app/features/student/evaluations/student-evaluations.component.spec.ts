import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import { StudentEvaluationsComponent } from './student-evaluations.component';
import { StudentEvaluationItem, StudentService } from '@core/services/student.service';
import { StatusTabsComponent } from '@shared/components/status-tabs/status-tabs.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { ComboboxComponent } from '@shared/components/combobox/combobox.component';

const EVALUATIONS: StudentEvaluationItem[] = [
  {
    exercise_id: 'e1',
    title: 'Recorrido de listas',
    description: 'Recorre una lista enlazada',
    type: 'code',
    environment_type: 'judge',
    purpose: 'class',
    difficulty: 'easy',
    due_date: '2026-11-05T23:59:00',
    subject_id: 's1',
    subject_name: 'Estructuras de Datos',
    subject_code: 'SIS-211',
    attempts_count: 0,
    best_score: null,
    academic_status: 'pending'
  },
  {
    exercise_id: 'e2',
    title: 'Examen de grafos',
    description: 'Recorridos BFS y DFS',
    type: 'code',
    environment_type: 'judge',
    purpose: 'exam',
    difficulty: 'medium',
    subject_id: 's1',
    subject_name: 'Estructuras de Datos',
    subject_code: 'SIS-211',
    attempts_count: 2,
    best_score: 45,
    last_verdict: 'WA',
    last_submitted_at: '2026-10-20T14:05:00',
    academic_status: 'in_progress'
  },
  {
    exercise_id: 'e3',
    title: 'Práctica de árboles',
    description: 'Implementa un árbol binario',
    type: 'ide',
    environment_type: 'ide_workspace',
    purpose: 'class',
    difficulty: 'hard',
    subject_id: 's1',
    subject_name: 'Estructuras de Datos',
    subject_code: 'SIS-211',
    attempts_count: 3,
    best_score: 95,
    last_verdict: 'AC',
    last_submitted_at: '2026-10-18T09:30:00',
    academic_status: 'completed'
  },
  {
    exercise_id: 'e4',
    title: 'Examen de ordenamiento',
    description: 'Ordena sin librerías',
    type: 'code',
    environment_type: 'judge',
    purpose: 'exam',
    due_date: '2026-10-01T23:59:00',
    subject_id: 's2',
    subject_name: 'Sistemas Operativos',
    subject_code: 'SIS-312',
    attempts_count: 1,
    last_verdict: 'TLE',
    last_submitted_at: '2026-10-02T08:10:00',
    academic_status: 'overdue'
  },
  {
    exercise_id: 'e5',
    title: 'Práctica de memoria',
    description: 'Administración de memoria',
    type: 'code',
    environment_type: 'judge',
    purpose: 'class',
    subject_id: 's2',
    subject_name: 'Sistemas Operativos',
    subject_code: 'SIS-312',
    attempts_count: 0,
    academic_status: 'pending'
  }
];

describe('StudentEvaluationsComponent', () => {
  let component: StudentEvaluationsComponent;
  let fixture: ComponentFixture<StudentEvaluationsComponent>;

  function items(): HTMLElement[] {
    return Array.from((fixture.nativeElement as HTMLElement).querySelectorAll<HTMLElement>('.evaluation-item'));
  }

  function itemFor(title: string): HTMLElement | undefined {
    return items().find((item) => item.textContent?.includes(title));
  }

  function statusTabs(): StatusTabsComponent {
    return fixture.debugElement.query(By.directive(StatusTabsComponent)).componentInstance as StatusTabsComponent;
  }

  function comboboxes(): ComboboxComponent[] {
    return fixture.debugElement
      .queryAll(By.directive(ComboboxComponent))
      .map((debugElement) => debugElement.componentInstance as ComboboxComponent);
  }

  function searchBar(): SearchBarComponent {
    return fixture.debugElement.query(By.directive(SearchBarComponent)).componentInstance as SearchBarComponent;
  }

  async function setup(): Promise<void> {
    TestBed.resetTestingModule();

    const mockStudentService = {
      evaluations: () => EVALUATIONS,
      loadEvaluations: () => Promise.resolve(EVALUATIONS)
    };

    await TestBed.configureTestingModule({
      imports: [StudentEvaluationsComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentEvaluationsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }

  it('muestra los cuatro KPIs y todas las actividades publicadas', async () => {
    await setup();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(fixture.nativeElement.querySelectorAll('kpi-card').length).toBe(4);
    expect(text).toContain('Pendientes');
    expect(text).toContain('En progreso');
    expect(text).toContain('Completadas');
    expect(text).toContain('Vencidas');

    expect(items().length).toBe(5);
    expect(text).toContain('SIS-211');
    expect(text).toContain('Estructuras de Datos');

    const labels = items().map((item) => item.querySelector('.academic-status')?.textContent?.trim());
    expect(labels).toEqual(['Pendiente', 'En progreso', 'Completado', 'Vencido', 'Pendiente']);
    expect(component.isLoading()).toBe(false);
  });

  it('filtra por estado académico desde las pestañas', async () => {
    await setup();

    statusTabs().tabChange.emit('completed');
    fixture.detectChanges();
    expect(items().length).toBe(1);
    expect(items()[0].textContent).toContain('Práctica de árboles');

    statusTabs().tabChange.emit('overdue');
    fixture.detectChanges();
    expect(items().length).toBe(1);
    expect(items()[0].textContent).toContain('Examen de ordenamiento');

    statusTabs().tabChange.emit('all');
    fixture.detectChanges();
    expect(items().length).toBe(5);
  });

  it('filtra por materia y por tipo de actividad desde los combobox', async () => {
    await setup();

    const [subjectCombo, purposeCombo] = comboboxes();
    subjectCombo.valueChange.emit('s2');
    fixture.detectChanges();
    expect(component.subjectFilter()).toBe('s2');
    expect(items().length).toBe(2);

    subjectCombo.valueChange.emit('all');
    purposeCombo.valueChange.emit('exam');
    fixture.detectChanges();
    expect(component.purposeFilter()).toBe('exam');
    expect(items().length).toBe(2);
    expect(items().every((item) => item.textContent?.includes('Examen'))).toBe(true);
  });

  it('filtra por título desde el buscador', async () => {
    await setup();

    searchBar().queryChange.emit('grafos');
    fixture.detectChanges();
    expect(items().length).toBe(1);
    expect(items()[0].textContent).toContain('Examen de grafos');

    searchBar().cleared.emit();
    fixture.detectChanges();
    expect(items().length).toBe(5);
  });

  it('muestra el último resultado con verdict-badge en lenguaje humano', async () => {
    await setup();

    expect(itemFor('Examen de grafos')?.querySelector('verdict-badge')?.textContent).toContain('Resultado incorrecto');
    expect(itemFor('Práctica de árboles')?.querySelector('verdict-badge')?.textContent).toContain('Solución correcta');
    expect(itemFor('Examen de ordenamiento')?.querySelector('verdict-badge')?.textContent).toContain('Tiempo límite excedido');
    expect(itemFor('Recorrido de listas')?.querySelector('verdict-badge')).toBeNull();
  });

  it('enruta al solver o al laboratorio según la actividad', async () => {
    await setup();

    const solverLink = itemFor('Recorrido de listas')?.querySelector<HTMLAnchorElement>('.action-link');
    expect(solverLink?.getAttribute('href')).toBe('/student/exercises/e1');
    expect(solverLink?.textContent).toContain('Resolver');

    expect(itemFor('Examen de grafos')?.querySelector('.action-link')?.textContent).toContain('Continuar');
    expect(itemFor('Examen de ordenamiento')?.querySelector('.action-link')?.textContent).toContain('Resolver');

    const labLink = itemFor('Práctica de árboles')?.querySelector<HTMLAnchorElement>('.action-link');
    expect(labLink?.getAttribute('href')).toBe('/student/workspaces/s1');
    expect(labLink?.textContent).toContain('Abrir en laboratorio');
  });

  it('renderiza fechas como time semántico', async () => {
    await setup();

    const pendingDue = itemFor('Recorrido de listas')?.querySelector('time');
    expect(pendingDue?.getAttribute('datetime')).toBe('2026-11-05T23:59:00');
    expect(pendingDue?.textContent?.trim()).toBe('05/11/2026');

    const lastSubmission = itemFor('Examen de grafos')?.querySelector('time.stat-date');
    expect(lastSubmission?.getAttribute('datetime')).toBe('2026-10-20T14:05:00');
    expect(lastSubmission?.textContent?.trim()).toBe('20/10 14:05:00');
  });
});
