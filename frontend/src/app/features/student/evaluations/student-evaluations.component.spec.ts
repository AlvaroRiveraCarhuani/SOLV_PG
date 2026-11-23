import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import { StudentEvaluationsComponent } from './student-evaluations.component';
import { StudentEvaluationItem, StudentService } from '@core/services/student.service';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { FilterPopoverComponent } from '@shared/components/filter-popover/filter-popover.component';

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
    environment_type: 'ide_environment',
    purpose: 'class',
    difficulty: 'hard',
    subject_id: 's1',
    subject_name: 'Estructuras de Datos',
    subject_code: 'SIS-211',
    attempts_count: 3,
    best_score: 95,
    last_verdict: 'AC',
    last_verdict_origin: 'rubric',
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
    return Array.from((fixture.nativeElement as HTMLElement).querySelectorAll<HTMLElement>('.activity-row'));
  }

  function itemFor(title: string): HTMLElement | undefined {
    return items().find((item) => item.textContent?.includes(title));
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
    component.completedExpanded.set(true);
    fixture.detectChanges();
  }

  it('muestra todas las actividades publicadas agrupadas en radar transversal sin tarjetas de totales', async () => {
    await setup();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(fixture.nativeElement.querySelectorAll('kpi-card').length).toBe(0);
    expect(text).toContain('Exámenes y cierres próximos');
    expect(text).toContain('Prácticas abiertas');

    expect(items().length).toBe(5);
    expect(text).toContain('SIS-211');
    expect(text).toContain('Estructuras de Datos');
    expect(component.isLoading()).toBe(false);
  });

  it('filtra por estado académico', async () => {
    await setup();

    component.setStatus('completed');
    fixture.detectChanges();
    expect(items().length).toBe(1);
    expect(items()[0].textContent).toContain('Práctica de árboles');

    component.setStatus('pending');
    fixture.detectChanges();
    expect(items().length).toBe(2);

    component.setStatus('all');
    fixture.detectChanges();
    expect(items().length).toBe(5);
  });

  it('filtra por materia y por tipo de actividad', async () => {
    await setup();

    component.setSubject('s2');
    fixture.detectChanges();
    expect(component.subjectFilter()).toBe('s2');
    expect(items().length).toBe(2);

    component.setSubject('all');
    component.setPurpose('exam');
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

  it('muestra el último resultado con veredicto humanizado (cero AC/WA/TLE)', async () => {
    await setup();

    expect(itemFor('Examen de grafos')?.textContent).toContain('No cumple');
    expect(itemFor('Práctica de árboles')?.textContent).toContain('Calificado por rúbrica');
    expect(itemFor('Examen de ordenamiento')?.textContent).toContain('Tarda demasiado');
    expect(itemFor('Recorrido de listas')?.querySelector('.verdict-text')).toBeNull();
  });

  it('enruta cada actividad a la práctica que le corresponde', async () => {
    await setup();

    const solverLink = itemFor('Recorrido de listas') as HTMLAnchorElement;
    expect(solverLink?.getAttribute('href')).toBe('/student/exercises/e1');
    expect(solverLink?.textContent).toContain('Resolver');

    expect(itemFor('Examen de grafos')?.textContent).toContain('Continuar');
    expect(itemFor('Examen de ordenamiento')?.textContent).toContain('Resolver');

    const labLink = itemFor('Práctica de árboles')?.querySelector<HTMLAnchorElement>('.action-link');
    expect(labLink?.getAttribute('href')).toBe('/student/exercises/e3');
  });

  it('renderiza fechas como time semántico y expone la hora de entrega', async () => {
    await setup();

    const pendingItem = itemFor('Recorrido de listas');
    const times = pendingItem?.querySelectorAll('time');
    expect(times && times.length >= 2).toBe(true);
    expect(times![0]?.getAttribute('datetime')).toBe('2026-11-05T23:59:00');
    expect(times![0]?.textContent?.trim()).toContain('05 nov');
    expect(times![1]?.textContent?.trim()).toContain('23:59');
  });
});
