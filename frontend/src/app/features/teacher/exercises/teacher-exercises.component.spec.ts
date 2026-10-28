import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TeacherExercisesComponent } from './teacher-exercises.component';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { RouterTestingModule } from '@angular/router/testing';
import { Router } from '@angular/router';
import { of } from 'rxjs';
import { signal } from '@angular/core';
import { TeacherCourseSummary, TeacherLabStats } from '../models/teacher.models';

describe('TeacherExercisesComponent', () => {
  let component: TeacherExercisesComponent;
  let fixture: ComponentFixture<TeacherExercisesComponent>;
  let mockDashboardService: Partial<TeacherDashboardService>;
  let mockCourseService: Partial<TeacherCourseService>;

  const mockCourses: TeacherCourseSummary[] = [
    {
      id: 'course-1',
      code: 'CS101',
      name: 'Estructuras de Datos',
      students_count: 25,
      active_now: 4,
      pending_review: 2,
      at_risk: 1
    },
    {
      id: 'course-2',
      code: 'CS102',
      name: 'Bases de Datos',
      students_count: 30,
      active_now: 6,
      pending_review: 0,
      at_risk: 0
    }
  ];

  const mockLabsCourse1: TeacherLabStats[] = [
    {
      id: 'lab-1',
      title: 'Árboles Binarios AVL',
      status: 'published',
      type: 'ALGORITMO',
      language: 'python',
      memory_limit_mb: 256,
      time_limit_ms: 2000,
      submissions_count: 15,
      students_count: 25,
      auto_graded: 10,
      pending_review: 2,
      at_risk: 1
    },
    {
      id: 'lab-2',
      title: 'Grafos y Dijkstra',
      status: 'draft',
      type: 'ALGORITMO',
      language: 'cpp',
      memory_limit_mb: 128,
      time_limit_ms: 1000,
      submissions_count: 0,
      students_count: 25,
      auto_graded: 0,
      pending_review: 0,
      at_risk: 0
    }
  ];

  const mockLabsCourse2: TeacherLabStats[] = [
    {
      id: 'lab-3',
      title: 'Consultas Complejas SQL',
      status: 'published',
      type: 'DATABASE',
      language: 'sql',
      memory_limit_mb: 512,
      time_limit_ms: 3000,
      submissions_count: 20,
      students_count: 30,
      auto_graded: 20,
      pending_review: 0,
      at_risk: 0
    }
  ];

  beforeEach(async () => {
    mockDashboardService = {
      courses: signal<TeacherCourseSummary[]>(mockCourses),
      loadDashboardData: vi.fn().mockReturnValue(of({ courses: mockCourses, attention: { critical: [], warning: [], standard: [] } }))
    };

    mockCourseService = {
      getCourseLabs: vi.fn().mockImplementation((courseId: string) => {
        if (courseId === 'course-1') return of(mockLabsCourse1);
        if (courseId === 'course-2') return of(mockLabsCourse2);
        return of([]);
      })
    };

    await TestBed.configureTestingModule({
      imports: [TeacherExercisesComponent, RouterTestingModule],
      providers: [
        { provide: TeacherDashboardService, useValue: mockDashboardService },
        { provide: TeacherCourseService, useValue: mockCourseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherExercisesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should aggregate all labs from all courses and calculate KPI metrics correctly', () => {
    expect(component.courses().length).toBe(2);
    expect(component.allExercises().length).toBe(3);

    const stats = component.kpiStats();
    expect(stats.total).toBe(3);
    expect(stats.published).toBe(2);
    expect(stats.drafts).toBe(1);
    expect(stats.totalSubmissions).toBe(35);
  });

  it('should filter exercises by course', () => {
    component.courseFilter.set('course-1');
    expect(component.filteredExercises().length).toBe(2);

    component.courseFilter.set('course-2');
    expect(component.filteredExercises().length).toBe(1);
    expect(component.filteredExercises()[0].title).toBe('Consultas Complejas SQL');
  });

  it('should filter exercises by modality', () => {
    component.modalityFilter.set('DATABASE');
    expect(component.filteredExercises().length).toBe(1);
    expect(component.filteredExercises()[0].id).toBe('lab-3');

    component.modalityFilter.set('ALGORITMO');
    expect(component.filteredExercises().length).toBe(2);
  });

  it('should filter exercises by status', () => {
    component.statusFilter.set('draft');
    expect(component.filteredExercises().length).toBe(1);
    expect(component.filteredExercises()[0].title).toBe('Grafos y Dijkstra');
  });

  it('should filter exercises by search text', () => {
    component.searchTerm.set('AVL');
    expect(component.filteredExercises().length).toBe(1);
    expect(component.filteredExercises()[0].id).toBe('lab-1');
  });

  it('should navigate to create and edit routes correctly', () => {
    const router = TestBed.inject(Router);
    const navSpy = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    component.openCreateModal();
    expect(navSpy).toHaveBeenCalledWith(['/teacher/courses', 'course-1', 'exercises', 'new']);

    component.openEditModal(component.allExercises()[0]);
    expect(navSpy).toHaveBeenCalledWith(['/teacher/courses', 'course-1', 'exercises', 'lab-1', 'edit']);
  });

  it('should open and close fuzzing modal correctly', () => {
    component.openFuzzingModal(component.allExercises()[0]);
    expect(component.isFuzzingOpen()).toBe(true);
    expect(component.exerciseForFuzzing()?.id).toBe('lab-1');

    component.closeFuzzingModal();
    expect(component.isFuzzingOpen()).toBe(false);
  });

  it('should disable creation and show specific empty state when teacher has no courses', () => {
    const coursesSignal = mockDashboardService.courses as ReturnType<typeof signal<TeacherCourseSummary[]>>;
    coursesSignal.set([]);
    fixture.detectChanges();

    expect(component.hasCourses()).toBe(false);

    const router = TestBed.inject(Router);
    const navSpy = vi.spyOn(router, 'navigate');

    component.openCreateModal();
    expect(navSpy).not.toHaveBeenCalled();

    const compiled = fixture.nativeElement as HTMLElement;
    const headerBtn = compiled.querySelector('.exercises-header .btn-primary') as HTMLButtonElement;
    expect(headerBtn.disabled).toBe(true);
    expect(headerBtn.getAttribute('title')).toContain('Debes tener al menos una materia asignada');

    const emptyStateText = compiled.querySelector('.empty-state h3')?.textContent;
    expect(emptyStateText).toBe('Sin materias asignadas');
  });
});
