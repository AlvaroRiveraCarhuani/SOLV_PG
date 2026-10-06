import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ActivatedRoute } from '@angular/router';
import { of } from 'rxjs';
import { AnalyticsDashboardComponent } from './analytics-dashboard.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { CourseAnalyticsDTO } from '../../models/teacher.models';

describe('AnalyticsDashboardComponent', () => {
  let component: AnalyticsDashboardComponent;
  let fixture: ComponentFixture<AnalyticsDashboardComponent>;
  let courseService: TeacherCourseService;

  const mockAnalytics: CourseAnalyticsDTO = {
    difficulty_distribution: {
      easy: { count: 4, success_rate: 0.9 },
      medium: { count: 6, success_rate: 0.65 },
      hard: { count: 2, success_rate: 0.35 }
    },
    top_tags: [
      { tag: 'sorting', count: 5, success_rate: 0.8 },
      { tag: 'dp', count: 3, success_rate: 0.4 }
    ],
    most_failed_cases: [
      { exercise_title: 'Fibonacci', case_index: 2, fail_count: 15 }
    ],
    avg_resolution_time_by_difficulty: {
      easy: 180,
      medium: 900,
      hard: 3600
    },
    submissions_timeline: [
      { date: '2026-03-01', count: 10 },
      { date: '2026-03-02', count: 25 }
    ]
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AnalyticsDashboardComponent],
      providers: [
        TeacherCourseService,
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'courseId' || key === 'id' ? 'course-123' : null)
              }
            }
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AnalyticsDashboardComponent);
    component = fixture.componentInstance;
    courseService = TestBed.inject(TeacherCourseService);
  });

  it('debe crearse correctamente y cargar metricas', () => {
    vi.spyOn(courseService, 'getCourseAnalytics').mockReturnValue(of(mockAnalytics));
    fixture.detectChanges();

    expect(component).toBeTruthy();
    expect(component.analytics()).toEqual(mockAnalytics);
    expect(component.totalExercisesWithDifficulty()).toBe(12);
    expect(component.totalSubmissionsTimeline()).toBe(35);
  });

  it('debe formatear segundos a texto legible', () => {
    expect(component.formatSeconds(45)).toBe('45s');
    expect(component.formatSeconds(150)).toBe('2m 30s');
    expect(component.formatSeconds(3600)).toBe('1h 0m');
    expect(component.formatSeconds(0)).toBe('0s');
  });

  it('debe formatear porcentajes con un decimal', () => {
    expect(component.formatPercentage(0.754)).toBe('75.4%');
    expect(component.formatPercentage(1.0)).toBe('100.0%');
    expect(component.formatPercentage(0)).toBe('0.0%');
  });
});
