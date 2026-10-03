import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { StudentDashboardComponent, STUDENT_DEFAULT_WIDGETS, STUDENT_DASHBOARD_STORAGE_KEY } from './student-dashboard.component';
import { StudentService } from '@core/services/student.service';
import { AuthService } from '@core/services/auth.service';
import { DashboardLayoutService } from '@core/services/dashboard-layout.service';
import { CourseColorService } from '@core/services/course-color.service';

describe('StudentDashboardComponent', () => {
  let component: StudentDashboardComponent;
  let fixture: ComponentFixture<StudentDashboardComponent>;
  let layoutService: DashboardLayoutService;

  beforeEach(async () => {
    localStorage.clear();

    const mockAuthService = {
      currentUser: () => ({ id: 'u1', first_name: 'Mateo', email: 'mateo@solv.edu' })
    };

    const mockStudentService = {
      dashboardData: () => ({
        subjects: [
          {
            subject: { id: 's1', code: 'SIS-211', name: 'Estructuras de Datos' },
            active_workspace: { id: 'w1', status: 'running', memory_limit_mb: 512, type: 'C/C++' }
          }
        ]
      }),
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
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('should initialize student greeting, default widgets, and course colors', () => {
    expect(component).toBeTruthy();
    expect(component.studentFirstName()).toBe('Mateo');
    expect(component.widgets().length).toBe(STUDENT_DEFAULT_WIDGETS.length);
    const color = component.getCourseColor('s1', 'SIS-211');
    expect(color).toBeTruthy();
  });

  it('should toggle customize mode and widget visibility', () => {
    expect(component.isCustomizing()).toBe(false);
    component.toggleCustomize();
    expect(component.isCustomizing()).toBe(true);

    component.toggleWidgetVisibility('agenda');
    const agenda = component.widgets().find(w => w.id === 'agenda');
    expect(agenda?.visible).toBe(false);
  });

  it('should update colSpan and persist layout in storage', () => {
    component.setWidgetColSpan('labs', 12);
    const labs = component.widgets().find(w => w.id === 'labs');
    expect(labs?.colSpan).toBe(12);

    const saved = layoutService.loadLayout(STUDENT_DASHBOARD_STORAGE_KEY, STUDENT_DEFAULT_WIDGETS);
    expect(saved.find(w => w.id === 'labs')?.colSpan).toBe(12);
  });
});
