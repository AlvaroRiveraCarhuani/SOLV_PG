import { ChangeDetectionStrategy, Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TeacherCourseService } from '../../../services/teacher-course.service';
import { CourseGradesMatrix, StudentGradesRow } from '../../../models/teacher.models';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

/**
 * Libreta de la materia: una fila por estudiante y una columna por práctica.
 * La celda nula del contrato (`grades[exercise_id] = null`) se muestra como
 * «—»: sin corregir no es 0, y la libreta no inventa notas.
 */
@Component({
  selector: 'course-gradebook',
  standalone: true,
  imports: [CommonModule, SkeletonLoaderComponent, MachineDataDirective],
  templateUrl: './course-gradebook.component.html',
  styleUrl: './course-gradebook.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class CourseGradebookComponent implements OnInit {
  readonly subjectId = input.required<string>();

  private readonly courseService = inject(TeacherCourseService);

  readonly matrix = signal<CourseGradesMatrix | null>(null);
  readonly isLoading = signal<boolean>(true);
  readonly error = signal<string | null>(null);

  readonly students = computed(() => this.matrix()?.students ?? []);
  readonly exercises = computed(() => this.matrix()?.exercises ?? []);

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.isLoading.set(true);
    this.error.set(null);

    this.courseService.getCourseGradesMatrix(this.subjectId()).subscribe({
      next: (data) => {
        this.matrix.set(data);
        this.isLoading.set(false);
      },
      error: () => {
        this.error.set('No se pudo cargar la libreta de la materia.');
        this.isLoading.set(false);
      }
    });
  }

  /** Nota de una celda; `null` es sin corregir y nunca se convierte en 0. */
  gradeOf(student: StudentGradesRow, exerciseId: string): number | null {
    return student.grades[exerciseId] ?? null;
  }
}
