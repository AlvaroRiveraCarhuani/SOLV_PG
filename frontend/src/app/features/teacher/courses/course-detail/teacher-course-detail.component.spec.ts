import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TeacherCourseDetailComponent } from './teacher-course-detail.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherDashboardService } from '../../services/teacher-dashboard.service';
import { ActivatedRoute, Router } from '@angular/router';
import { of } from 'rxjs';
import { signal } from '@angular/core';
import { TeacherLabStats, SubmissionQueueItem } from '../../models/teacher.models';

describe('TeacherCourseDetailComponent', () => {
  let component: TeacherCourseDetailComponent;
  let fixture: ComponentFixture<TeacherCourseDetailComponent>;
  let mockCourseService: Partial<TeacherCourseService>;
  let mockDashboardService: Partial<TeacherDashboardService>;
  let mockRouter: { navigate: ReturnType<typeof vi.fn> };

  const labsSignal = signal<TeacherLabStats[]>([]);
  const subsSignal = signal<SubmissionQueueItem[]>([]);

  const mockLabs: TeacherLabStats[] = [
    {
      id: 'lab-1',
      title: 'Práctica 1: Recursión',
      status: 'published',
      submissions_count: 5,
      students_count: 10,
      auto_graded: 4,
      pending_review: 1,
      at_risk: 0
    },
    {
      id: 'lab-2',
      title: 'Práctica 2: Pilas y Colas',
      status: 'published',
      submissions_count: 3,
      students_count: 10,
      auto_graded: 0,
      pending_review: 3,
      at_risk: 1
    }
  ];

  const mockSubmissions: SubmissionQueueItem[] = [
    {
      id: 'sub-1',
      exercise_id: 'lab-1',
      exercise_title: 'Práctica 1: Recursión',
      student_id: 'std-1',
      student_name: 'Lucas Benítez',
      student_email: 'lucas@example.com',
      verdict: 'AC',
      score: 100,
      manual_override: false,
      execution_time_ms: 120,
      memory_used_mb: 16,
      submitted_at: '2026-03-01T10:00:00Z',
      comments_count: 0,
      review_status: 'graded'
    },
    {
      id: 'sub-2',
      exercise_id: 'lab-1',
      exercise_title: 'Práctica 1: Recursión',
      student_id: 'std-2',
      student_name: 'Ana Pérez',
      student_email: 'ana@example.com',
      verdict: 'WA',
      score: 40,
      manual_override: false,
      execution_time_ms: 150,
      memory_used_mb: 20,
      submitted_at: '2026-03-01T10:15:00Z',
      comments_count: 0,
      review_status: 'graded'
    },
    {
      id: 'sub-3',
      exercise_id: 'lab-2',
      exercise_title: 'Práctica 2: Pilas y Colas',
      student_id: 'std-3',
      student_name: 'Martín Gómez',
      student_email: 'martin@example.com',
      verdict: 'PENDING',
      manual_override: false,
      execution_time_ms: 0,
      memory_used_mb: 0,
      submitted_at: '2026-03-01T11:00:00Z',
      comments_count: 0,
      review_status: 'pending'
    }
  ];

  beforeEach(async () => {
    labsSignal.set(mockLabs);
    subsSignal.set(mockSubmissions);

    mockCourseService = {
      labs: labsSignal,
      submissions: subsSignal,
      isLoading: signal(false),
      getCourseLabs: vi.fn().mockReturnValue(of(mockLabs)),
      getCourseSubmissions: vi.fn().mockReturnValue(of(mockSubmissions)),
      getCourseModules: vi.fn().mockReturnValue(of([]))
    };

    const dashboardMock = {
      courses: [
        {
          id: 'course-101',
          code: 'CS101',
          name: 'Algoritmos y Estructuras',
          students_count: 10,
          active_now: 3,
          pending_review: 4,
          at_risk: 1
        }
      ]
    };

    mockDashboardService = {
      loadDashboardData: vi.fn().mockReturnValue(of(dashboardMock))
    };

    mockRouter = {
      navigate: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [TeacherCourseDetailComponent],
      providers: [
        { provide: TeacherCourseService, useValue: mockCourseService },
        { provide: TeacherDashboardService, useValue: mockDashboardService },
        { provide: Router, useValue: mockRouter },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'id' ? 'course-101' : null)
              },
              queryParamMap: {
                get: (_key: string) => null
              }
            }
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherCourseDetailComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe inicializarse con filtros por defecto y todas las entregas', () => {
    expect(component.subjectId()).toBe('course-101');
    expect(component.labFilter()).toBe('all');
    expect(component.verdictFilter()).toBe('all');
    expect(component.reviewStatusFilter()).toBe('all');
    expect(component.filteredSubmissions().length).toBe(3);
  });

  it('debe filtrar entregas por estado de revisión', () => {
    component.reviewStatusFilter.set('graded');
    expect(component.filteredSubmissions().map(s => s.id)).toEqual(['sub-1', 'sub-2']);

    component.reviewStatusFilter.set('pending');
    expect(component.filteredSubmissions().map(s => s.id)).toEqual(['sub-3']);
  });

  it('debe generar labOptions incluyendo opción para todos y cada laboratorio', () => {
    const options = component.labOptions();
    expect(options.length).toBe(3);
    expect(options[0]).toEqual({ id: 'all', label: 'Todos los laboratorios', value: 'all' });
    expect(options[1]).toEqual({ id: 'lab-1', label: 'Práctica 1: Recursión', value: 'lab-1' });
    expect(options[2]).toEqual({ id: 'lab-2', label: 'Práctica 2: Pilas y Colas', value: 'lab-2' });
  });

  it('debe filtrar entregas por laboratorio específico', () => {
    component.labFilter.set('lab-1');
    const filtered = component.filteredSubmissions();
    expect(filtered.length).toBe(2);
    expect(filtered.every(s => s.exercise_id === 'lab-1')).toBe(true);

    component.labFilter.set('lab-2');
    const filteredLab2 = component.filteredSubmissions();
    expect(filteredLab2.length).toBe(1);
    expect(filteredLab2[0].student_name).toBe('Martín Gómez');
    expect(filteredLab2[0].verdict).toBe('PENDING');
  });

  it('debe filtrar entregas combinando laboratorio y veredicto', () => {
    component.labFilter.set('lab-1');
    component.verdictFilter.set('AC');
    const filtered = component.filteredSubmissions();
    expect(filtered.length).toBe(1);
    expect(filtered[0].id).toBe('sub-1');

    component.verdictFilter.set('WA');
    expect(component.filteredSubmissions().length).toBe(1);
    expect(component.filteredSubmissions()[0].id).toBe('sub-2');

    component.verdictFilter.set('PENDING');
    expect(component.filteredSubmissions().length).toBe(0);
  });

  it('debe cambiar de pestaña y aplicar filtro al invocar viewLabSubmissions', () => {
    component.activeTab.set('labs');
    component.viewLabSubmissions('lab-2');

    expect(component.activeTab()).toBe('submissions');
    expect(component.labFilter()).toBe('lab-2');
    expect(component.filteredSubmissions().length).toBe(1);
    expect(component.filteredSubmissions()[0].exercise_id).toBe('lab-2');
  });

  it('debe cerrar el menú de más acciones al hacer click fuera del dropdown', () => {
    component.showMoreActions.set(true);
    fixture.detectChanges();

    const menu = fixture.nativeElement.querySelector('.dropdown-menu') as HTMLElement;
    expect(menu).toBeTruthy();

    const metricasLink = fixture.nativeElement.querySelector('a.btn-analytics') as HTMLElement;
    metricasLink.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    fixture.detectChanges();

    expect(component.showMoreActions()).toBe(false);
  });

  it('debe mantener abierto el menú al hacer click dentro del dropdown', () => {
    component.showMoreActions.set(true);
    fixture.detectChanges();

    const menu = fixture.nativeElement.querySelector('.dropdown-menu') as HTMLElement;
    menu.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    fixture.detectChanges();

    expect(component.showMoreActions()).toBe(true);
  });
});
