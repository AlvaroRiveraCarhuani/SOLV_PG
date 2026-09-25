import { Component, EventEmitter, Input, Output, HostListener, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminCoursesService, AdminCourseItem } from '../../../services/admin-courses.service';
import { TeacherItem } from '@core/models/admin.model';
import { ComboboxComponent, ComboboxOption } from '../../../../../shared/components/combobox/combobox.component';
import { 
  LucideArrowRightLeft, 
  LucideX, 
  LucideAlertCircle, 
  LucideUser, 
  LucideFileText 
} from '@lucide/angular';

@Component({
  selector: 'solv-course-reassign-modal',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    ComboboxComponent,
    LucideArrowRightLeft, 
    LucideX, 
    LucideAlertCircle, 
    LucideUser, 
    LucideFileText
  ],
  templateUrl: './course-reassign-modal.component.html',
  styleUrls: ['./course-reassign-modal.component.scss']
})
export class CourseReassignModalComponent implements OnInit {
  private coursesService = inject(AdminCoursesService);

  @Input({ required: true }) course!: AdminCourseItem;
  @Input() teachers: TeacherItem[] = [];

  @Output() close = new EventEmitter<void>();
  @Output() reassignCompleted = new EventEmitter<void>();

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    this.close.emit();
  }

  selectedTeacherId = signal<string>('');
  reason = signal<string>('');

  isSubmitting = signal<boolean>(false);
  formError = signal<string | null>(null);

  teacherComboboxOptions = computed<ComboboxOption[]>(() => {
    return this.teachers.map(t => ({
      id: t.id,
      label: t.full_name,
      value: t.id,
      meta: t.email,
      badge: t.id === this.course.teacher_id ? 'Actual' : undefined,
      badgeVariant: t.id === this.course.teacher_id ? ('official' as const) : undefined
    }));
  });

  selectedTeacherLabel = computed<string>(() => {
    const id = this.selectedTeacherId();
    if (!id) return '';
    const match = this.teachers.find(t => t.id === id);
    return match ? match.full_name : '';
  });

  onTeacherSelected(opt: ComboboxOption): void {
    this.selectedTeacherId.set(opt.value || '');
  }

  ngOnInit(): void {
    // Filtrar para que por defecto no seleccione al docente actual si ya está asignado
    const available = this.teachers.filter(t => t.id !== this.course.teacher_id);
    if (available.length > 0) {
      this.selectedTeacherId.set(available[0].id);
    }
  }

  submitReassign(): void {
    if (!this.selectedTeacherId()) {
      this.formError.set('Debe seleccionar un docente para la materia.');
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
