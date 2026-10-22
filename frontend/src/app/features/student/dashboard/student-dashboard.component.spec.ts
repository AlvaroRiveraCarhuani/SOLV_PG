import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { StudentDashboardComponent, STUDENT_DEFAULT_WIDGETS, STUDENT_DASHBOARD_STORAGE_KEY } from './student-dashboard.component';
import { StudentService, StudentDashboardData } from '@core/services/student.service';
import { AuthService } from '@core/services/auth.service';
import { DashboardLayoutService } from '@core/services/dashboard-layout.service';
import { CourseColorService } from '@core/services/course-color.service';

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

const EMPTY_DASHBOARD: StudentDashboardData = {
  student_id: 'u1',
  tenant_id: 't1',
  subjects: [],
  recent_submissions: []
};

describe('StudentDashboardComponent', () => {
  let component: StudentDashboardComponent;
  let fixture: ComponentFixture<StudentDashboardComponent>;
  let layoutService: DashboardLayoutService;

  async function setup(dashboard: StudentDashboardData): Promise<void> {
    localStorage.clear();
    TestBed.resetTestingModule();

    const mockAuthService = {
      currentUser: () => ({ id: 'u1', first_name: 'Mateo', email: 'mateo@solv.edu' })
    };

    const mockStudentService = {
      dashboardData: () => dashboard,
      dueAssignments: () => [],
      loadDashboard: () => Promise.resolve(),
      startWorkspace: () => Promise.resolve()
    };

    await TestBed.configureTestingModule({
      imports: [StudentDashboardComponent],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: mockAuthService },
        { provide: StudentService, useValue: mockStudentService },
        DashboardLayoutService,
        CourseColorService
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentDashboardComponent);
    component = fixture.componentInstance;
    layoutService = TestBed.inject(DashboardLayoutService);
    fixture.detectChanges();
  }

  afterEach(() => {
    localStorage.clear();
  });

  it('should initialize student greeting, default widgets, and course colors', async () => {
    await setup(FULL_DASHBOARD);

    expect(component).toBeTruthy();
    expect(component.studentFirstName()).toBe('Mateo');
    expect(component.widgets().length).toBe(STUDENT_DEFAULT_WIDGETS.length);
    const color = component.getCourseColor('s1', 'SIS-211');
    expect(color).toBeTruthy();
  });

  it('should toggle customize mode and widget visibility', async () => {
    await setup(FULL_DASHBOARD);

    expect(component.isCustomizing()).toBe(false);
    component.toggleCustomize();
    expect(component.isCustomizing()).toBe(true);

    component.toggleWidgetVisibility('agenda');
    const agenda = component.widgets().find(w => w.id === 'agenda');
    expect(agenda?.visible).toBe(false);
  });

  it('should update colSpan and persist layout in storage', async () => {
    await setup(FULL_DASHBOARD);

    component.setWidgetColSpan('labs', 12);
    const labs = component.widgets().find(w => w.id === 'labs');
    expect(labs?.colSpan).toBe(12);

    const saved = layoutService.loadLayout(STUDENT_DASHBOARD_STORAGE_KEY, STUDENT_DEFAULT_WIDGETS);
    expect(saved.find(w => w.id === 'labs')?.colSpan).toBe(12);
  });

  it('should compute global progress from enrolled subjects and render their percentages', async () => {
    await setup(FULL_DASHBOARD);

    expect(component.globalProgressPercent()).toBe(50);

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('50% Global');
    expect(text).toContain('Estructuras de Datos');
    expect(text).toContain('75%');

    const fills = (fixture.nativeElement as HTMLElement).querySelectorAll<HTMLElement>('.progress-fill');
    expect(fills.length).toBe(2);
    expect(fills[0].style.width).toBe('75%');
    expect(fills[1].style.width).toBe('0%');
  });

  it('should render recent submissions with title, verdict and formatted date', async () => {
    await setup(FULL_DASHBOARD);

    expect(component.recentSubmissions().length).toBe(1);

    const items = (fixture.nativeElement as HTMLElement).querySelectorAll('.recent-item');
    expect(items.length).toBe(1);
    expect(items[0].textContent).toContain('Árboles binarios');
    expect(items[0].textContent).toContain('AC');

    const time = items[0].querySelector('time');
    expect(time?.getAttribute('datetime')).toBe('2026-10-20T14:05:00');
    expect(time?.textContent?.trim()).toBe('20/10 14:05:00');
  });

  it('should show empty states when the student has no subjects or submissions', async () => {
    await setup(EMPTY_DASHBOARD);

    expect(component.globalProgressPercent()).toBe(0);
    expect(component.recentSubmissions()).toEqual([]);

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Sin progreso registrado en materias activas.');
    expect(text).toContain('Aún no registras entregas recientes.');
  });
});
