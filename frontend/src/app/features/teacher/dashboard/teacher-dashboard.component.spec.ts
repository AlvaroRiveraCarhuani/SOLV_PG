import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { TeacherDashboardComponent, TEACHER_DEFAULT_WIDGETS, TEACHER_DASHBOARD_STORAGE_KEY } from './teacher-dashboard.component';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { DashboardLayoutService } from '@core/services/dashboard-layout.service';
import { CourseColorService } from '@core/services/course-color.service';

describe('TeacherDashboardComponent', () => {
  let component: TeacherDashboardComponent;
  let fixture: ComponentFixture<TeacherDashboardComponent>;
  let layoutService: DashboardLayoutService;

  beforeEach(async () => {
    localStorage.clear();

    const mockDashboardService = {
      courses: () => [
        { id: 'c1', name: 'Algoritmos', code: 'CS101', students_count: 20, active_now: 5, pending_review: 2, at_risk: 0 }
      ],
      attention: () => ({ critical: [], warning: [], standard: [] }),
      periods: () => [],
      isLoading: () => false,
      getAcademicPeriods: () => of([]),
      loadDashboardData: () => of({ courses: [], attention: { critical: [], warning: [], standard: [] } })
    };

    await TestBed.configureTestingModule({
      imports: [TeacherDashboardComponent],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: TeacherDashboardService, useValue: mockDashboardService },
        DashboardLayoutService,
        CourseColorService
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherDashboardComponent);
    component = fixture.componentInstance;
    layoutService = TestBed.inject(DashboardLayoutService);
    fixture.detectChanges();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('should initialize with default widgets and compute reactive stats', () => {
    expect(component).toBeTruthy();
    expect(component.widgets().length).toBe(TEACHER_DEFAULT_WIDGETS.length);
    expect(component.greetingMessage().text).toContain('Todo al día');
  });

  it('should toggle customize mode', () => {
    expect(component.isCustomizing()).toBe(false);
    component.toggleCustomizing();
    expect(component.isCustomizing()).toBe(true);
  });

  it('should update widget colSpan and persist layout', () => {
    component.setWidgetColSpan('courses', 12);
    const target = component.widgets().find(w => w.id === 'courses');
    expect(target?.colSpan).toBe(12);

    const saved = layoutService.loadLayout(TEACHER_DASHBOARD_STORAGE_KEY, TEACHER_DEFAULT_WIDGETS);
    expect(saved.find(w => w.id === 'courses')?.colSpan).toBe(12);
  });

  it('should toggle widget visibility and reset layout', () => {
    component.toggleWidgetVisibility('attention');
    let target = component.widgets().find(w => w.id === 'attention');
    expect(target?.visible).toBe(false);

    component.resetLayout();
    target = component.widgets().find(w => w.id === 'attention');
    expect(target?.visible).toBe(true);
  });
});
