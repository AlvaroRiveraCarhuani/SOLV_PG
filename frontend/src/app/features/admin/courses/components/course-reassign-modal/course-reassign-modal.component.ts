import { Component, EventEmitter, Input, Output, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminCoursesService, AdminCourseItem } from '../../../services/admin-courses.service';
import { TeacherItem } from '@core/models/admin.model';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  LucideArrowRightLeft,
  LucideAlertCircle,
  LucideFileText
} from '@lucide/angular';

@Component({
  selector: 'course-reassign-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MachineDataDirective,
    ComboboxComponent,
    ModalShellComponent,
    FormFieldComponent,
    LucideArrowRightLeft,
    LucideAlertCircle,
    LucideFileText
  ],
  templateUrl: './course-reassign-modal.component.html',
  styleUrls: ['./course-reassign-modal.component.scss']
})
export class CourseReassignModalComponent implements OnInit {
  private coursesService = inject(AdminCoursesService);

  @Input({ required: true }) course!: AdminCourseItem;
  @Input() teachers: TeacherItem[] = [];

  @Output() closed = new EventEmitter<void>();
  @Output() reassignCompleted = new EventEmitter<void>();

  selectedTeacherId = signal<string>('');
  reason = signal<string>('');

  isSubmitting = signal<boolean>(false);
  formError = signal<string | null>(null);

  teacherComboboxOptions = computed<ComboboxOption[]>(() => {
    return this.teachers
      .filter(t => t.id !== this.course?.teacher_id)
      .map(t => ({
        id: t.id,
        label: t.full_name,
        value: t.id,
        meta: t.email
      }));
  });

  selectedTeacherLabel = computed<string>(() => {
    const id = this.selectedTeacherId();
    if (!id) return '';
    const match = this.teachers.find(t => t.id === id);
    return match ? match.full_name : '';
  });

  onTeacherSelected(opt: ComboboxOption): void {
    this.selectedTeacherId.set(opt.value || opt.id || '');
    this.formError.set(null);
  }

  onTeacherCleared(): void {
    this.selectedTeacherId.set('');
  }

  ngOnInit(): void {
    // Inicia vacío para exigir una selección consciente del administrador
    this.selectedTeacherId.set('');
  }

  submitReassign(): void {
    if (!this.selectedTeacherId()) {
      this.formError.set('Debe seleccionar un docente para la materia.');
      return;
    }

    if (this.selectedTeacherId() === this.course.teacher_id) {
      this.formError.set('Debe seleccionar un docente diferente al titular actual.');
      return;
    }

    if (!this.reason().trim()) {
      this.formError.set('Debe ingresar un motivo para registrar en la auditoría institucional.');
      return;
    }

    this.isSubmitting.set(true);
    this.formError.set(null);

    this.coursesService.reassignCourseTeacher(this.course.id, {
      new_teacher_id: this.selectedTeacherId(),
      reason: this.reason().trim()
    }).subscribe({
      next: () => {
        this.isSubmitting.set(false);
        this.reassignCompleted.emit();
      },
      error: (err: { error?: { error?: string } }) => {
        this.isSubmitting.set(false);
        const msg = err.error?.error || 'No se pudo reasignar el docente. Verifique los permisos.';
        this.formError.set(msg);
      }
    });
  }
}
