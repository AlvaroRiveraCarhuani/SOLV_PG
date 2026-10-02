import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TeacherEvaluationsComponent } from './teacher-evaluations.component';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { RouterTestingModule } from '@angular/router/testing';
import { of } from 'rxjs';
import { signal } from '@angular/core';
import { TeacherCourseSummary, SubmissionQueueItem, TeacherLabStats } from '../models/teacher.models';

describe('TeacherEvaluationsComponent', () => {
  let component: TeacherEvaluationsComponent;
  let fixture: ComponentFixture<TeacherEvaluationsComponent>;
  let mockDashboardService: Partial<TeacherDashboardService>;
  let mockCourseService: Partial<TeacherCourseService>;

  const mockCourses: TeacherCourseSummary[] = [
    {
      id: 'course-1',
      code: 'CS101',
      name: 'Estructuras de Datos',
      students_count: 20,
      active_now: 5,
      pending_review: 2,
      at_risk: 1
    }
  ];

  const mockSubmissions: SubmissionQueueItem[] = [
    {
      id: 'sub-1',
      exercise_id: 'ex-1',
      exercise_title: 'Árboles Binarios',
      student_id: 'std-1',
      student_name: 'Ana Pérez',
      student_email: 'ana@example.com',
      verdict: 'AC',
      score: 100,
      manual_override: false,
      execution_time_ms: 120,
      memory_used_mb: 4,
      submitted_at: '2026-03-01T10:00:00Z',
      comments_count: 0
    },
    {
      id: 'sub-2',
      exercise_id: 'ex-1',
      exercise_title: 'Árboles Binarios',
      student_id: 'std-2',
      student_name: 'Carlos Ruiz',
      student_email: 'carlos@example.com',
      verdict: 'WA',
      score: undefined,
      manual_override: false,
      execution_time_ms: 200,
      memory_used_mb: 5,
      submitted_at: '2026-03-01T11:00:00Z',
      comments_count: 0
    }
  ];

  const mockLabs: TeacherLabStats[] = [
    {
      id: 'ex-1',
      title: 'Árboles Binarios',
      status: 'published',
      submissions_count: 2,
      students_count: 20,
      auto_graded: 1,
      pending_review: 1,
      at_risk: 1
    }
  ];

  beforeEach(async () => {
    mockDashboardService = {
      courses: signal<TeacherCourseSummary[]>(mockCourses),
      loadDashboardData: vi.fn().mockReturnValue(of({ courses: mockCourses, attention: { critical: [], warning: [], standard: [] } }))
    };

    mockCourseService = {
      getCourseSubmissions: vi.fn().mockReturnValue(of(mockSubmissions)),
      getCourseLabs: vi.fn().mockReturnValue(of(mockLabs))
    };

    await TestBed.configureTestingModule({
      imports: [TeacherEvaluationsComponent, RouterTestingModule],
      providers: [
        { provide: TeacherDashboardService, useValue: mockDashboardService },
        { provide: TeacherCourseService, useValue: mockCourseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherEvaluationsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should initialize with queue tab active and calculate KPI metrics correctly', () => {
    expect(component.activeTab()).toBe('queue');
    expect(component.courses().length).toBe(1);
    expect(component.queueList().length).toBe(2);

    const stats = component.kpiStats();
    expect(stats.total).toBe(2);
    expect(stats.pending).toBe(1); // sub-2 has undefined score
    expect(stats.graded).toBe(1); // sub-1 has score 100
    expect(stats.atRisk).toBe(1); // sub-2 has WA verdict
  });

  it('should filter queue items by search term', () => {
    component.searchTerm.set('Carlos');
    expect(component.filteredQueue().length).toBe(1);
    expect(component.filteredQueue()[0].student_name).toBe('Carlos Ruiz');

    component.searchTerm.set('NoExiste');
    expect(component.filteredQueue().length).toBe(0);
  });

  it('should filter queue items by verdict', () => {
    component.verdictFilter.set('AC');
    expect(component.filteredQueue().length).toBe(1);
    expect(component.filteredQueue()[0].verdict).toBe('AC');

    component.verdictFilter.set('WA');
    expect(component.filteredQueue().length).toBe(1);
    expect(component.filteredQueue()[0].verdict).toBe('WA');
  });

  it('should filter queue items by status (pending vs graded)', () => {
    component.statusFilter.set('pending');
    expect(component.filteredQueue().length).toBe(1);
    expect(component.filteredQueue()[0].student_name).toBe('Carlos Ruiz');

    component.statusFilter.set('graded');
    expect(component.filteredQueue().length).toBe(1);
    expect(component.filteredQueue()[0].student_name).toBe('Ana Pérez');
  });

  it('should switch tabs between queue and grades', () => {
    component.activeTab.set('grades');
    expect(component.activeTab()).toBe('grades');
  });
});
