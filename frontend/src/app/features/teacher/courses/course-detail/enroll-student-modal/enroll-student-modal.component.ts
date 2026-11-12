import {
  Component,
  input,
  output,
  signal,
  inject,
  ChangeDetectionStrategy
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideSearch,
  LucideUserPlus,
  LucideUsers,
  LucideCheck,
  LucideAlertCircle,
  LucideLoader2
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { TeacherCourseService, AvailableStudent } from '../../../services/teacher-course.service';

@Component({
  selector: 'enroll-student-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideSearch,
    LucideUserPlus,
    LucideUsers,
    LucideCheck,
    LucideAlertCircle,
    LucideLoader2,
    ModalShellComponent
  ],
  templateUrl: './enroll-student-modal.component.html',
  styleUrl: './enroll-student-modal.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class EnrollStudentModalComponent {
  private teacherCourseService = inject(TeacherCourseService);

  readonly subjectId = input.required<string>();
  readonly closed = output<void>();
  readonly enrolled = output<AvailableStudent>();

  readonly searchQuery = signal<string>('');
  readonly students = signal<AvailableStudent[]>([]);
  readonly selectedStudent = signal<AvailableStudent | null>(null);
  readonly isSearching = signal<boolean>(false);
  readonly isEnrolling = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);
  readonly hasSearched = signal<boolean>(false);

  private searchTimeout: ReturnType<typeof setTimeout> | null = null;

  onSearchChange(query: string): void {
    this.searchQuery.set(query);
    this.selectedStudent.set(null);
    this.errorMessage.set(null);

    if (this.searchTimeout) {
      clearTimeout(this.searchTimeout);
    }

    const trimmed = query.trim();
    if (trimmed.length < 2) {
      this.students.set([]);
      this.hasSearched.set(false);
      this.isSearching.set(false);
      return;
    }

    this.isSearching.set(true);
    this.searchTimeout = setTimeout(() => {
      this.performSearch(trimmed);
    }, 300);
  }

  performSearch(query: string): void {
    this.teacherCourseService.searchAvailableStudents(this.subjectId(), query).subscribe({
      next: (results) => {
        this.students.set(results);
        this.hasSearched.set(true);
        this.isSearching.set(false);
      },
      error: (err) => {
        this.errorMessage.set(err?.error?.error || 'Error al buscar estudiantes');
        this.isSearching.set(false);
        this.hasSearched.set(true);
      }
    });
  }

  selectStudent(student: AvailableStudent): void {
    this.selectedStudent.set(student);
    this.errorMessage.set(null);
  }

  confirmEnroll(): void {
    const student = this.selectedStudent();
    if (!student || this.isEnrolling()) {
      return;
    }

    this.isEnrolling.set(true);
    this.errorMessage.set(null);

    this.teacherCourseService.enrollStudent(this.subjectId(), student.id).subscribe({
      next: () => {
        this.isEnrolling.set(false);
        this.enrolled.emit(student);
        this.closed.emit();
      },
      error: (err) => {
        this.errorMessage.set(err?.error?.error || 'Error al matricular al estudiante');
        this.isEnrolling.set(false);
      }
    });
  }
}
