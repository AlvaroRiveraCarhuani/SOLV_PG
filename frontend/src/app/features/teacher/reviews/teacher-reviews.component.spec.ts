import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { TeacherReviewsComponent } from './teacher-reviews.component';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { ReviewQueueGroup, SubmissionQueueItem } from '../models/teacher.models';

describe('TeacherReviewsComponent', () => {
  let fixture: ComponentFixture<TeacherReviewsComponent>;
  let component: TeacherReviewsComponent;
  let mockCourseService: Partial<TeacherCourseService>;

  const groups: ReviewQueueGroup[] = [
    {
      exercise_id: 'ex-1',
      exercise_title: 'Lista enlazada',
      subject_id: 'subj-1',
      subject_name: 'Programación I',
      environment_type: 'IDE_PERSISTENTE',
      pending: 2,
      total: 12,
      oldest_submitted_at: '2026-10-28T10:00:00Z'
    },
    {
      exercise_id: 'ex-2',
      exercise_title: 'Ordenamiento',
      subject_id: 'subj-2',
      subject_name: 'Estructuras de Datos',
      environment_type: 'IDE_PERSISTENTE',
      pending: 1,
      total: 8,
      oldest_submitted_at: '2026-10-29T09:00:00Z'
    }
  ];

  const submissions: SubmissionQueueItem[] = [
    {
      id: 'sub-new',
      exercise_id: 'ex-1',
      exercise_title: 'Lista enlazada',
      student_id: 'stu-1',
      student_name: 'Ana Quispe',
      student_email: 'ana@example.com',
      verdict: 'PENDING',
      manual_override: false,
      execution_time_ms: 0,
      memory_used_mb: 0,
      submitted_at: '2026-10-29T12:00:00Z',
      comments_count: 0,
      review_status: 'pending'
    },
    {
      id: 'sub-old',
      exercise_id: 'ex-1',
      exercise_title: 'Lista enlazada',
      student_id: 'stu-2',
      student_name: 'Luis Mamani',
      student_email: 'luis@example.com',
      verdict: 'PENDING',
      manual_override: false,
      execution_time_ms: 0,
      memory_used_mb: 0,
      submitted_at: '2026-10-28T08:00:00Z',
      comments_count: 0,
      review_status: 'pending'
    },
    {
      id: 'sub-done',
      exercise_id: 'ex-1',
      exercise_title: 'Lista enlazada',
      student_id: 'stu-3',
      student_name: 'Rosa Flores',
      student_email: 'rosa@example.com',
      verdict: 'AC',
      score: 100,
      manual_override: false,
      execution_time_ms: 10,
      memory_used_mb: 4,
      submitted_at: '2026-10-27T08:00:00Z',
      comments_count: 0,
      review_status: 'graded'
    }
  ];

  async function setup(queue: ReviewQueueGroup[], subjectId: string | null = null): Promise<void> {
    mockCourseService = {
      getReviewQueue: vi.fn().mockReturnValue(of(queue)),
      getCourseSubmissions: vi.fn().mockReturnValue(of(submissions))
    };

    await TestBed.configureTestingModule({
      imports: [TeacherReviewsComponent],
      providers: [
        provideRouter([]),
        { provide: TeacherCourseService, useValue: mockCourseService },
        {
          provide: TeacherDashboardService,
          useValue: { courses: signal([]), getCourses: () => of([]) }
        },
        {
          provide: ActivatedRoute,
          useValue: { queryParamMap: of(convertToParamMap(subjectId ? { subject_id: subjectId } : {})) }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherReviewsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }

  it('suma las entregas pendientes de todas las prácticas', async () => {
    await setup(groups);

    expect(component.totalPending()).toBe(3);
    expect(fixture.nativeElement.textContent).toContain('Lista enlazada');
    expect(fixture.nativeElement.textContent).toContain('Programación I');
  });

  it('selecciona la primera práctica y muestra solo las entregas sin corregir, de la más antigua a la más nueva', async () => {
    await setup(groups);

    expect(component.selectedGroup()?.exercise_id).toBe('ex-1');
    const pending = component.submissions();
    expect(pending.map((s) => s.id)).toEqual(['sub-old', 'sub-new']);
    expect(pending.some((s) => s.id === 'sub-done')).toBe(false);
  });

  it('usa la materia de la URL como filtro de la cola y lo deja quitar', async () => {
    await setup(groups, 'subj-1');

    expect(mockCourseService.getReviewQueue).toHaveBeenCalledWith('subj-1');
    const chip = fixture.nativeElement.querySelector('.filter-value');
    expect(chip?.textContent).toContain('Programación I');
    expect(fixture.nativeElement.querySelector('.filter-clear')).not.toBeNull();
  });

  it('muestra la espera en lugar del veredicto', async () => {
    await setup(groups);

    const waitHeader = fixture.nativeElement.querySelector('.col-wait') as HTMLElement | null;
    expect(waitHeader?.textContent).toContain('ESPERA');
    expect(fixture.nativeElement.querySelector('.col-verdict')).toBeNull();
    const waitCell = fixture.nativeElement.querySelector('.submission-wait') as HTMLElement | null;
    expect(waitCell?.textContent).toContain('hace');
  });

  it('muestra el estado vacío cuando no hay nada por calificar', async () => {
    await setup([]);

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('No hay entregas por calificar');
    expect(component.groups().length).toBe(0);
  });
});
