import { Component, EventEmitter, Input, Output, HostListener, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  AdminCoursesService, 
  AcademicPeriod, 
  DockerTemplateItem, 
  CreateCoursePayload 
} from '../../../services/admin-courses.service';
import { TeacherItem } from '@core/models/admin.model';
import { 
  LucideBookPlus, 
  LucideX, 
  LucideAlertCircle, 
  LucideUser, 
  LucideLayers, 
  LucideCalendar 
} from '@lucide/angular';

@Component({
  selector: 'solv-course-create-modal',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    LucideBookPlus, 
    LucideX, 
    LucideAlertCircle, 
    LucideUser, 
    LucideLayers, 
    LucideCalendar
  ],
  templateUrl: './course-create-modal.component.html',
  styleUrls: ['./course-create-modal.component.scss']
})
export class CourseCreateModalComponent implements OnInit {
  private coursesService = inject(AdminCoursesService);

  @Input() periods: AcademicPeriod[] = [];
  @Input() teachers: TeacherItem[] = [];
  @Input() templates: DockerTemplateItem[] = [];
  @Input() defaultPeriodId: string | null = null;

  @Output() close = new EventEmitter<void>();
  @Output() courseCreated = new EventEmitter<void>();

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    this.close.emit();
  }

  // Campos del formulario
  name = signal<string>('');
  code = signal<string>('');
  selectedPeriodId = signal<string>('');
  selectedTeacherId = signal<string>('');
  selectedTemplateId = signal<string>('');

  isSubmitting = signal<boolean>(false);
  formError = signal<string | null>(null);

  ngOnInit(): void {
    if (this.defaultPeriodId) {
      this.selectedPeriodId.set(this.defaultPeriodId);
    } else if (this.periods.length > 0) {
      const active = this.periods.find(p => p.is_active) || this.periods[0];
      this.selectedPeriodId.set(active.id);
    }

    if (this.templates.length > 0) {
      this.selectedTemplateId.set(this.templates[0].id);
    }
  }

  submitCourse(): void {
    if (!this.name().trim() || !this.code().trim()) {
      this.formError.set('El nombre y el código de la materia son obligatorios.');
      return;
    }

    this.isSubmitting.set(true);
    this.formError.set(null);

    const payload: CreateCoursePayload = {
      name: this.name().trim(),
      code: this.code().trim().toUpperCase(),
      teacher_id: this.selectedTeacherId() ? this.selectedTeacherId() : undefined,
      academic_period_id: this.selectedPeriodId() ? this.selectedPeriodId() : undefined
    };

    this.coursesService.createCourse(payload).subscribe({
      next: () => {
        this.isSubmitting.set(false);
        this.courseCreated.emit();
      },
      error: (err: { error?: { error?: string } }) => {
        this.isSubmitting.set(false);
        const msg = err.error?.error || 'No se pudo dar de alta el curso. Verifique que el código no esté duplicado.';
        this.formError.set(msg);
      }
    });
  }
}
