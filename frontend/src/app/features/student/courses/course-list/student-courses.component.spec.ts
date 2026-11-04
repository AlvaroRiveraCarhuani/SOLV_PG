import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import { StudentCoursesComponent } from './student-courses.component';
import { StudentDashboardData, StudentService, DueAssignment } from '@core/services/student.service';
import { CourseColorService } from '@core/services/course-color.service';
import { StatusTabsComponent } from '@shared/components/status-tabs/status-tabs.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';

const DASHBOARD: StudentDashboardData = {
  student_id: 'u1',
  tenant_id: 't1',
  subjects: [
    {
      subject: { id: 's1', code: 'SIS-211', name: 'Estructuras de Datos' },
      total_exercises: 4,
      completed_exercises: 3,
      progress_percent: 75
    },
    {
      subject: { id: 's2', code: 'SIS-312', name: 'Sistemas Operativos' },
      total_exercises: 2,
      completed_exercises: 0,
      progress_percent: 0
    },
    {
      subject: { id: 's3', code: 'MAT-101', name: 'Cálculo I' },
      total_exercises: 5,
      completed_exercises: 5,
      progress_percent: 100
    }
  ],
  recent_submissions: []
};

const DUE_ASSIGNMENTS: DueAssignment[] = [
  {
    exercise_id: 'ex-1',
    title: 'Listas enlazadas',
    subject_id: 's1',
    subject_name: 'Estructuras de Datos',
    subject_code: 'SIS-211',
    due_date: '2026-11-02T23:59:00',
    type: 'code'
  }
];

describe('StudentCoursesComponent', () => {
  let component: StudentCoursesComponent;
  let fixture: ComponentFixture<StudentCoursesComponent>;

  function cards(): HTMLElement[] {
    return Array.from((fixture.nativeElement as HTMLElement).querySelectorAll<HTMLElement>('.course-item'));
  }

  function statusTabs(): StatusTabsComponent {
    return fixture.debugElement.query(By.directive(StatusTabsComponent)).componentInstance as StatusTabsComponent;
  }

  function searchBar(): SearchBarComponent {
    return fixture.debugElement.query(By.directive(SearchBarComponent)).componentInstance as SearchBarComponent;
  }

  async function setup(): Promise<void> {
    localStorage.clear();
    TestBed.resetTestingModule();

    const mockStudentService = {
      dashboardData: () => DASHBOARD,
      dueAssignments: () => DUE_ASSIGNMENTS,
      loading: () => false,
      loadDashboard: () => Promise.resolve()
    };

    await TestBed.configureTestingModule({
      imports: [StudentCoursesComponent],
      providers: [
        provideRouter([]),
        CourseColorService,
        { provide: StudentService, useValue: mockStudentService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentCoursesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }

  afterEach(() => {
    localStorage.clear();
  });

  it('muestra una fila por materia inscrita sin indicadores agregados', async () => {
    await setup();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    // Los totales del semestre viven en el inicio; repetirlos acá no agrega lectura.
    expect(fixture.nativeElement.querySelectorAll('kpi-card').length).toBe(0);
    expect(text).not.toContain('Ejercicios resueltos');

    expect(cards().length).toBe(3);
    expect(text).toContain('SIS-211');
    expect(text).toContain('Estructuras de Datos');
    expect(text).toContain('Sistemas Operativos');
    expect(text).toContain('Cálculo I');
    expect(component.isLoading()).toBe(false);
  });

  it('filtra por las pestañas de estado académico', async () => {
    await setup();

    statusTabs().tabChange.emit('completed');
    fixture.detectChanges();
    expect(component.filterMode()).toBe('completed');
    expect(cards().length).toBe(1);
    expect(cards()[0].textContent).toContain('Cálculo I');

    statusTabs().tabChange.emit('in_progress');
    fixture.detectChanges();
    expect(cards().length).toBe(2);

    statusTabs().tabChange.emit('all');
    fixture.detectChanges();
    expect(cards().length).toBe(3);
  });

  it('filtra por nombre y por código desde el buscador', async () => {
    await setup();

    searchBar().queryChange.emit('sistemas');
    fixture.detectChanges();
    expect(cards().length).toBe(1);
    expect(cards()[0].textContent).toContain('Sistemas Operativos');

    searchBar().queryChange.emit('mat-101');
    fixture.detectChanges();
    expect(cards().length).toBe(1);
    expect(cards()[0].textContent).toContain('Cálculo I');

    searchBar().cleared.emit();
    fixture.detectChanges();
    expect(component.searchQuery()).toBe('');
    expect(cards().length).toBe(3);
  });

  it('enlaza cada materia con sus prácticas y no ofrece un laboratorio por materia', async () => {
    await setup();

    const firstCard = cards().find((card) => card.textContent?.includes('Estructuras de Datos'));
    expect(firstCard).toBeTruthy();

    const action = firstCard?.querySelector<HTMLAnchorElement>('.course-action');
    expect(action?.getAttribute('href')).toBe('/student/courses/s1');
    expect(action?.textContent).toContain('Ver prácticas');
  });

  it('muestra la próxima entrega como fecha semántica y remite el entorno a cada práctica', async () => {
    await setup();

    const dataCard = cards().find((card) => card.textContent?.includes('Estructuras de Datos'));
    expect(dataCard?.textContent).toContain('Listas enlazadas');
    const time = dataCard?.querySelector('time');
    expect(time?.getAttribute('datetime')).toBe('2026-11-02T23:59:00');
    expect(time?.textContent?.trim()).toBe('02 nov');

    // Ninguna materia anuncia un entorno propio: el estado del contenedor
    // pertenece a la práctica y se consulta ahí.
    for (const card of cards()) {
      expect(card.querySelector('status-badge')).toBeNull();
    }
  });

  it('distingue una materia sin entregas próximas', async () => {
    await setup();

    const dataCard = cards().find((card) => card.textContent?.includes('Sistemas Operativos'));
    expect(dataCard?.textContent).toContain('Sin entregas próximas');
    expect(dataCard?.querySelector('time')).toBeNull();
  });
});
