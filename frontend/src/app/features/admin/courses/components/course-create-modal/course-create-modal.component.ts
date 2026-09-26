import { Component, EventEmitter, Input, Output, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  AdminCoursesService,
  AcademicPeriod,
  DockerTemplateItem,
  CreateCoursePayload
} from '../../../services/admin-courses.service';
import { TeacherItem } from '@core/models/admin.model';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import {
  LucideBookPlus,
  LucideAlertCircle
} from '@lucide/angular';

@Component({
  selector: 'course-create-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ComboboxComponent,
    ModalShellComponent,
    FormFieldComponent,
    LucideBookPlus,
    LucideAlertCircle
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

  // Campos del formulario
  name = signal<string>('');
  code = signal<string>('');
  selectedPeriodId = signal<string>('');
  selectedTeacherId = signal<string>('');
  selectedTemplateId = signal<string>('');

  isSubmitting = signal<boolean>(false);
  formError = signal<string | null>(null);

  periodComboboxOptions = computed<ComboboxOption[]>(() => {
    return [
      { id: '', label: 'Sin periodo asignado', value: '' },
      ...this.periods.map(p => ({
        id: p.id,
        label: `${p.name} (${p.code})`,
        value: p.id,
        badge: p.is_active ? 'Activo' : undefined,
        badgeVariant: p.is_active ? ('official' as const) : undefined
      }))
    ];
  });

  selectedPeriodLabel = computed<string>(() => {
    const id = this.selectedPeriodId();
    if (!id) return 'Sin periodo asignado';
    const match = this.periods.find(p => p.id === id);
    return match ? `${match.name} (${match.code})` : 'Sin periodo asignado';
  });

  teacherComboboxOptions = computed<ComboboxOption[]>(() => {
    return [
      { id: '', label: 'Dejar sin asignar temporalmente', value: '' },
      ...this.teachers.map(t => ({
        id: t.id,
        label: t.full_name,
        value: t.id,
        meta: t.email
      }))
    ];
  });

  selectedTeacherLabel = computed<string>(() => {
    const id = this.selectedTeacherId();
    if (!id) return 'Dejar sin asignar temporalmente';
    const match = this.teachers.find(t => t.id === id);
    return match ? match.full_name : 'Dejar sin asignar temporalmente';
  });

  templateComboboxOptions = computed<ComboboxOption[]>(() => {
    return this.templates.map(tpl => ({
      id: tpl.id,
      label: tpl.display_name || tpl.name,
      value: tpl.id,
      meta: tpl.base_ram_mb ? `${tpl.base_ram_mb} MB RAM` : undefined
    }));
  });

  selectedTemplateLabel = computed<string>(() => {
    const id = this.selectedTemplateId();
    if (!id) return '';
    const match = this.templates.find(t => t.id === id);
    return match ? (match.display_name || match.name) : '';
  });

  onPeriodSelected(opt: ComboboxOption): void {
    this.selectedPeriodId.set(opt.value || '');
  }

  onTeacherSelected(opt: ComboboxOption): void {
    this.selectedTeacherId.set(opt.value || '');
  }

  onTemplateSelected(opt: ComboboxOption): void {
    this.selectedTemplateId.set(opt.value || '');
  }

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
      academic_period_id: this.selectedPeriodId() ? this.selectedPeriodId() : undefined,
      template_id: this.selectedTemplateId() ? this.selectedTemplateId() : undefined
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

