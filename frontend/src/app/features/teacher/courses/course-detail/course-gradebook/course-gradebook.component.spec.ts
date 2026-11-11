import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { CourseGradebookComponent } from './course-gradebook.component';
import { TeacherCourseService } from '../../../services/teacher-course.service';
import { CourseGradesMatrix } from '../../../models/teacher.models';

describe('CourseGradebookComponent', () => {
  let fixture: ComponentFixture<CourseGradebookComponent>;
  let component: CourseGradebookComponent;

  const matrix: CourseGradesMatrix = {
    subject_id: 'course-1',
    subject_name: 'Desarrollo Web Fullstack',
    subject_code: 'SIS-420',
    exercises: [
      { id: 'ex-1', title: 'Práctica 1: Endpoints REST' },
      { id: 'ex-2', title: 'Práctica 2: Consultas SQL' }
    ],
    students: [
      {
        student_id: 'stu-1',
        student_name: 'Valeria Flores',
        student_email: 'valeria@example.com',
        grades: { 'ex-1': 80, 'ex-2': null },
        average: 80,
        graded_count: 1,
        pending_count: 1
      },
      {
        student_id: 'stu-2',
        student_name: 'Mateo Silva',
        student_email: 'mateo@example.com',
        grades: { 'ex-1': null, 'ex-2': null },
        average: 0,
        graded_count: 0,
        pending_count: 2
      }
    ]
  };

  async function setup(service: Partial<TeacherCourseService>): Promise<void> {
    await TestBed.configureTestingModule({
      imports: [CourseGradebookComponent],
      providers: [{ provide: TeacherCourseService, useValue: service }]
    }).compileComponents();

    fixture = TestBed.createComponent(CourseGradebookComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('subjectId', 'course-1');
    fixture.detectChanges();
  }

  it('muestra una fila por estudiante y una columna por práctica', async () => {
    await setup({ getCourseGradesMatrix: () => of(matrix) });

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Valeria Flores');
    expect(text).toContain('Mateo Silva');
    expect(text).toContain('Práctica 1: Endpoints REST');
    expect(text).toContain('Práctica 2: Consultas SQL');
    expect(component.exercises().length).toBe(2);
  });

  it('muestra guion en lo no corregido y no fabrica ceros', async () => {
    await setup({ getCourseGradesMatrix: () => of(matrix) });

    const missing = (fixture.nativeElement as HTMLElement).querySelectorAll('.grade-missing');
    // Mateo no tiene ninguna nota (2 celdas + promedio) y Valeria tiene una
    // práctica sin corregir: cuatro celdas vacías, ninguna con 0.
    expect(missing.length).toBe(4);
    expect(Array.from(missing).every((cell) => cell.textContent?.trim() === '—')).toBe(true);

    const values = (fixture.nativeElement as HTMLElement).querySelectorAll('.grade-value');
    expect(Array.from(values).map((cell) => cell.textContent?.trim())).toEqual(['80', '80']);
  });

  it('informa el error de carga y permite reintentar', async () => {
    await setup({ getCourseGradesMatrix: () => throwError(() => new Error('boom')) });

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('No se pudo cargar la libreta de la materia.');
    expect(fixture.nativeElement.querySelector('.btn-outline')).not.toBeNull();
  });
});
