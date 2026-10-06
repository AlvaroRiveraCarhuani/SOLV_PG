import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, ActivatedRoute } from '@angular/router';
import { of } from 'rxjs';
import { StudentCourseDetailComponent } from './student-course-detail.component';
import { StudentService } from '@core/services/student.service';

describe('StudentCourseDetailComponent', () => {
  let component: StudentCourseDetailComponent;
  let fixture: ComponentFixture<StudentCourseDetailComponent>;
  let mockStudentService: {
    dashboardData: ReturnType<typeof vi.fn>;
    loadDashboard: ReturnType<typeof vi.fn>;
    getCourseRecommendations: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    mockStudentService = {
      dashboardData: vi.fn().mockReturnValue({
        student_id: 'std-1',
        tenant_id: 'uab',
        subjects: [
          {
            subject: { id: 'course-101', name: 'Programación II', code: 'SIS-211' },
            active_workspace: { id: 'ws-1', status: 'running', type: 'IDE_PERSISTENTE' }
          }
        ],
        recent_submissions: []
      }),
      loadDashboard: vi.fn().mockResolvedValue(undefined),
      getCourseRecommendations: vi.fn().mockResolvedValue({
        has_enough_data: true,
        weak_tags: [],
        recommendations: [],
        message: '¡Vas al día! No hay refuerzos sugeridos.'
      })
    };

    await TestBed.configureTestingModule({
      imports: [StudentCourseDetailComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'courseId' || key === 'id' ? 'course-101' : null)
              }
            },
            params: of({ courseId: 'course-101' })
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentCourseDetailComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create and resolve course information', () => {
    expect(component).toBeTruthy();
    expect(component.effectiveCourseId()).toBe('course-101');
    expect(component.subjectName()).toBe('Programación II');
    expect(component.subjectCode()).toBe('SIS-211');
  });
});
