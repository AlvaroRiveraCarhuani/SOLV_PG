import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { signal } from '@angular/core';
import { StudentDashboardComponent } from './student-dashboard.component';
import { StudentService, StudentDashboardData, DueAssignment } from '@core/services/student.service';
import { AuthService } from '@core/services/auth.service';
import { StudentAulaService, StudentAula } from '../services/student-aula.service';

const AULA: StudentAula = {
  id: 'sesion-1',
  subject_id: 's1',
  subject_code: 'SIS-211',
  subject_name: 'Estructuras de Datos',
  exercise_id: 'ex-1',
  exercise_title: 'Árboles binarios',
  purpose: 'class',
  status: 'open',
  opens_at: '2026-11-04T08:00:00',
  ends_at: '2026-11-04T10:00:00',
  closes_at: '2026-11-04T10:00:00',
  frozen_at: null,
  grace_until: null,
  enrolled: 28,
  present: 21
};

const FULL_DASHBOARD: StudentDashboardData = {
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
    }
  ],
  recent_submissions: [
    {
      id: 'sub-1',
      exercise_id: 'ex-1',
      exercise_title: 'Árboles binarios',
      subject_code: 'SIS-211',
      verdict: 'AC',
      score: 100,
      execution_time_ms: 120,
      memory_used_mb: 32,
      submitted_at: '2026-10-20T14:05:00'
    }
  ]
};

const DUE: DueAssignment[] = [
  {
    exercise_id: 'ex-9',
    title: 'Listas enlazadas',
    subject_id: 's1',
    subject_name: 'Estructuras de Datos',
    subject_code: 'SIS-211',
    due_date: '2026-11-06T23:59:00',
    type: 'tarea'
  }
];

const EMPTY_DASHBOARD: StudentDashboardData = {
  student_id: 'u1',
  tenant_id: 't1',
  subjects: [],
  recent_submissions: []
};

describe('StudentDashboardComponent', () => {
  let fixture: ComponentFixture<StudentDashboardComponent>;

  async function setup(options: {
    dashboard?: StudentDashboardData;
    aulas?: StudentAula[];
    due?: DueAssignment[];
    dueFailed?: boolean;
    aulaError?: string | null;
  } = {}): Promise<void> {
    TestBed.resetTestingModule();

    const mockAuthService = {
      currentUser: () => ({ id: 'u1', first_name: 'Mateo', email: 'mateo@solv.edu' })
    };

    const mockStudentService = {
      dashboardData: signal(options.dashboard ?? FULL_DASHBOARD),
      dueAssignments: signal(options.due ?? DUE),
      dueLoadFailed: signal(options.dueFailed ?? false),
      loadDashboard: () => Promise.resolve()
    };

    const mockAulaService = {
      aulas: signal(options.aulas ?? [AULA]),
      loading: signal(false),
      error: signal(options.aulaError ?? null),
      load: () => Promise.resolve()
    };

    await TestBed.configureTestingModule({
      imports: [StudentDashboardComponent],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: mockAuthService },
        { provide: StudentService, useValue: mockStudentService },
        { provide: StudentAulaService, useValue: mockAulaService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentDashboardComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  }

  function text(): string {
    return (fixture.nativeElement as HTMLElement).textContent ?? '';
  }

  function links(): HTMLAnchorElement[] {
    return Array.from((fixture.nativeElement as HTMLElement).querySelectorAll('a'));
  }

  it('saluda al alumno y resume la clase abierta', async () => {
    await setup();

    expect(text()).toContain('Mateo');
    expect(text()).toContain('Tienes clase abierta de Estructuras de Datos.');
  });

  it('muestra la práctica, el horario y la presencia del aula', async () => {
    await setup();

    expect(text()).toContain('Árboles binarios');
    expect(text()).toContain('21');
    expect(text()).toContain('de');
    expect(text()).toContain('28');

    const times = (fixture.nativeElement as HTMLElement).querySelectorAll('.aula-meta time');
    expect(times.length).toBe(2);
    expect(times[0].textContent?.trim()).toBe('08:00');

    // Con práctica asociada, la tarjeta lleva directo a la práctica: el aula es
    // el contexto (presencia y avances), no un paso intermedio obligatorio.
    const practiceLink = links().find(a => a.textContent?.includes('Abrir práctica'));
    expect(practiceLink?.getAttribute('href')).toBe('/student/exercises/ex-1');

    const aulaLink = links().find(a => a.textContent?.includes('Ver el aula'));
    expect(aulaLink?.getAttribute('href')).toBe('/student/aula');
  });

  it('sin práctica asociada entra al aula como única acción', async () => {
    await setup({ aulas: [{ ...AULA, exercise_id: null, exercise_title: undefined }] });

    const aulaLink = links().find(a => a.textContent?.includes('Entrar al aula'));
    expect(aulaLink?.getAttribute('href')).toBe('/student/aula');
    expect(text()).not.toContain('Abrir práctica');
  });

  it('avisa cuando hay más de una clase abierta', async () => {
    await setup({ aulas: [AULA, { ...AULA, id: 'sesion-2', subject_name: 'Álgebra' }] });

    expect(text()).toContain('Ver las 2 clases');
  });

  it('sin clase abierta invita a las prácticas y no afirma que está al día si falló la carga', async () => {
    await setup({ aulas: [], dueFailed: true });

    expect(text()).toContain('No tienes clase abierta');
    expect(text()).toContain('No pudimos revisar tus entregas pendientes.');

    const practicas = links().find(a => a.textContent?.includes('Ver mis prácticas'));
    expect(practicas?.getAttribute('href')).toBe('/student/courses');
  });

  it('lista las entregas próximas con su fecha y enlace a la práctica', async () => {
    await setup();

    const row = (fixture.nativeElement as HTMLElement).querySelector('.row-link') as HTMLAnchorElement;
    expect(row.textContent).toContain('Listas enlazadas');
    expect(row.textContent).toContain('SIS-211');
    expect(row.getAttribute('href')).toBe('/student/exercises/ex-9');

    const time = row.querySelector('time');
    expect(time?.getAttribute('datetime')).toBe('2026-11-06T23:59:00');
    expect(time?.textContent?.trim()).toBe('06/11/2026 23:59');
  });

  it('muestra el avance por materia y el total del semestre', async () => {
    await setup();

    const rows = (fixture.nativeElement as HTMLElement).querySelectorAll('.course-row');
    expect(rows.length).toBe(2);
    expect(rows[0].textContent).toContain('3/4');

    const fills = (fixture.nativeElement as HTMLElement).querySelectorAll<HTMLElement>('.progress-fill');
    expect(fills[0].style.width).toBe('75%');
    expect(fills[1].style.width).toBe('0%');
    expect(text()).toContain('50%');
  });

  it('cierra con las últimas entregas y sus estados vacíos', async () => {
    await setup();

    const recent = (fixture.nativeElement as HTMLElement).querySelectorAll('.recent-item');
    expect(recent.length).toBe(1);
    expect(recent[0].textContent).toContain('Árboles binarios');
    expect(recent[0].textContent).toContain('Cumple');

    await setup({ dashboard: EMPTY_DASHBOARD, aulas: [], due: [] });

    expect(text()).toContain('No tienes materias matriculadas.');
    expect(text()).toContain('Aún no registras entregas.');
  });
});
